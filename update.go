package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LINE_URL   string = "http://dl.stickershop.line.naver.jp/products/0/0/1"
	ZIP_FORMAT string = LINE_URL + "/%s/android/stickers.zip"
	//ZIP_FORMAT       string = "http://line.polppolservice.com/getpng/stickers/sticker%s.zip"
	META_FORMAT      string = LINE_URL + "/%s/android/productInfo.meta"
	TAB_ON_FORMAT    string = LINE_URL + "/%s/android/tab_on.png"
	TAB_OFF_FORMAT   string = LINE_URL + "/%s/android/tab_off.png"
	STICKER_FORMAT   string = LINE_URL + "/%s/android/stickers/%s.png"
	THUMBNAIL_FORMAT string = LINE_URL + "/%s/android/stickers/%s_key.png"
)

var (
	ids  chan int
	wait sync.WaitGroup
)

func Update(begin, end int) {
	logger.Println("update", begin, end)
	setupTable()

	//setup worker
	ids = make(chan int)
	for i := 0; i < 10; i++ {
		go worker()
	}

	//assign job to worker
	for id := begin; id < end; id++ {
		wait.Add(1)
		ids <- id
	}
	wait.Wait()
}

func worker() {
	for id := range ids {
		func(id int) {
			defer wait.Done()

			var packageId int
			repo := checkRepo(id)
			// check exist in database
			stickerDBLock.Lock()
			err := stickerDB.QueryRow("SELECT packageId FROM "+repo+" WHERE packageId=?", id).Scan(&packageId)
			stickerDBLock.Unlock()

			switch {
			case err == sql.ErrNoRows:
				downlodAndInsert(id)
			case err != nil:
				logger.Println("query package", id, "err:", err)
			default:
				fmt.Println("skip", id)
			}
		}(id)
	}
}

func download(id int) (*http.Response, error) {
	res, err := http.Get(fmt.Sprintf(ZIP_FORMAT, strconv.Itoa(id)))
	if err != nil {
		var retry int
		for retry < 5 && err != nil {
			retry++
			time.Sleep(1 * time.Second)
			res, err = http.Get(fmt.Sprintf(ZIP_FORMAT, strconv.Itoa(id)))
		}
		if err != nil {
			logger.Println("http.Get err:", err)
			return nil, err
		}
	}
	return res, err
}

type OriginalMeta struct {
	PackageId int64              `json:"packageId"`
	Title     map[string]string  `json:"title"`
	Author    map[string]string  `json:"author"`
	Stickers  []map[string]int64 `json:"stickers"`
}

type Meta struct {
	PackageId int64             `json:"packageId"`
	Title     map[string]string `json:"title"`
	Author    map[string]string `json:"author"`
	Stickers  []int64           `json:"stickers"`
}

func unzip(id int, res *http.Response) error {
	tempZipFile, err := ioutil.TempFile("", "freeliner-sticker")
	if err != nil {
		return err
	}

	_, err = io.Copy(tempZipFile, res.Body)
	if err != nil {
		return err
	}
	defer tempZipFile.Close()

	zipReader, err := zip.OpenReader(tempZipFile.Name())
	if err != nil {
		return err
	}
	defer zipReader.Close()

	packageDirectory := path.Join(stickerDirectory, fmt.Sprint(id))
	err = os.MkdirAll(packageDirectory, os.ModePerm)
	if err != nil {
		return err
	}

	r, err := regexp.Compile(".*\\.png")
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		content, err := f.Open()
		if err != nil {
			return err
		}
		defer content.Close()

		var rc io.Reader
		var filePath string
		if r.MatchString(f.Name) {
			rc, err = pngFileToJpeg(content)
			filePath = path.Join(packageDirectory, strings.Replace(f.Name, "png", "jpg", -1))
		} else {
			rc, err = changeMeta(content)
			filePath = path.Join(packageDirectory, f.Name)
		}
		if err != nil {
			return err
		}

		file, err := os.Create(filePath)
		defer file.Close()
		if err != nil {
			return err
		}
		_, err = io.Copy(file, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

func changeMeta(origin io.Reader) (io.Reader, error) {
	originMetaData, err := ioutil.ReadAll(origin)
	if err != nil {
		return nil, err
	}

	var originMeta OriginalMeta
	err = json.Unmarshal(originMetaData, &originMeta)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var meta Meta
	meta.PackageId = originMeta.PackageId
	meta.Title = originMeta.Title
	meta.Author = originMeta.Author
	meta.Stickers = make([]int64, 0, len(originMeta.Stickers))
	for _, sticker := range originMeta.Stickers {
		meta.Stickers = append(meta.Stickers, sticker["id"])
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(metaData), nil
}

func downlodAndInsert(id int) {
	res, err := download(id)
	if err != nil {
		logger.Println(err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		fmt.Println(id, " does not exist")
		return
	}

	fmt.Println("process package", id)
	err = unzip(id, res)
	if err != nil {
		logger.Println(err)
		return
	}

	insert(id)
}

func pngFileToJpeg(pngFile io.Reader) (io.Reader, error) {
	img, err := png.Decode(pngFile)
	if err != nil {
		return nil, err
	}

	b := img.Bounds()
	imgRGBA := image.NewRGBA(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a != 0 {
				var sR, sG, sB, sA float64
				sR = float64(r) / float64(a)
				sG = float64(g) / float64(a)
				sB = float64(b) / float64(a)
				sA = float64(a) / float64(65535)

				var tR, tG, tB uint8
				tR = whiteBackground(sR, sA)
				tG = whiteBackground(sG, sA)
				tB = whiteBackground(sB, sA)
				imgRGBA.Set(x, y, color.RGBA{tR, tG, tB, 255})
			} else {
				imgRGBA.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	var buffer bytes.Buffer
	err = jpeg.Encode(&buffer, imgRGBA, nil)
	return &buffer, err
}

func whiteBackground(sC, sA float64) uint8 {
	tC := (sA * sC) + ((1 - sA) * 1)
	return uint8(tC * 255)
}
