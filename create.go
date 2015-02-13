package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/carylorrk/resize"
)

func create(id, begin int) {

	fmt.Print("author: ")
	bio := bufio.NewReader(os.Stdin)
	authorBytes, _, err := bio.ReadLine()
	if err != nil {
		logger.Println(err)
		return
	}

	author := string(authorBytes)

	fmt.Print("title: ")
	titleBytes, _, err := bio.ReadLine()
	if err != nil {
		logger.Println(err)
		return
	}
	title := string(titleBytes)

	dirPath := path.Join(stickerDirectory, fmt.Sprint(id))

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		logger.Println(err)
		return
	}

	count := 0
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		switch ext {
		case ".jpg":
			oldpath := path.Join(dirPath, file.Name())
			newpath := path.Join(dirPath, fmt.Sprint(count, ".jpg"))
			err = os.Rename(oldpath, newpath)
			if err != nil {
				logger.Println(err)
				return
			}
			count += 1
		}
	}

	count = 0
	files, err = ioutil.ReadDir(dirPath)
	if err != nil {
		logger.Println(err)
		return
	}
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		switch ext {
		case ".jpg":
			oldpath := path.Join(dirPath, file.Name())
			newpath := path.Join(dirPath, fmt.Sprint(begin-count, ".jpg"))
			err = os.Rename(oldpath, newpath)
			if err != nil {
				logger.Println(err)
				return
			}
			count += 1
		case ".png":
			oldpath := path.Join(dirPath, file.Name())
			newpath := path.Join(dirPath, fmt.Sprint(begin-count, ".jpg"))
			oldFile, err := os.Open(oldpath)
			if err != nil {
				logger.Println(err)
				return
			}

			pngBuffer, err := pngFileToJpeg(oldFile)
			if err != nil {
				logger.Println(err)
				return
			}

			newFile, err := os.Create(newpath)
			if err != nil {
				logger.Println(err)
				return
			}
			defer newFile.Close()

			_, err = io.Copy(newFile, pngBuffer)
			if err != nil {
				logger.Println(err)
				return
			}

			err = os.Remove(oldpath)
			if err != nil {
				logger.Println(err)
				return
			}
			count += 1
		}
	}

	beginPath := path.Join(dirPath, fmt.Sprint(begin, ".jpg"))
	beginFile, err := os.Open(beginPath)
	if err != nil {
		logger.Println(err)
		return
	}

	beginImg, _, err := image.Decode(beginFile)
	if err != nil {
		logger.Println(err)
		return
	}

	tabOnPath := path.Join(dirPath, "tab_on.jpg")
	tabOnFile, err := os.Create(tabOnPath)
	if err != nil {
		logger.Println(err)
		return
	}

	tabOnImg := resize.Resize(66, 55, beginImg, resize.Lanczos3)

	err = jpeg.Encode(tabOnFile, tabOnImg, nil)
	if err != nil {
		logger.Println(err)
		return
	}

	// Create a new grayscale image
	b := tabOnImg.Bounds()
	tabOffImg := image.NewGray(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y))
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			tabOffImg.Set(x, y, tabOnImg.At(x, y))
		}
	}

	tabOffPath := path.Join(dirPath, "tab_off.jpg")
	tabOffFile, err := os.Create(tabOffPath)
	if err != nil {
		logger.Println(err)
		return
	}

	err = jpeg.Encode(tabOffFile, tabOffImg, nil)
	if err != nil {
		logger.Println(err)
		return
	}

	var meta Meta
	meta.PackageId = int64(id)
	meta.Author = map[string]string{
		"en": author,
	}

	meta.Title = map[string]string{
		"en": title,
	}
	meta.Stickers = make([]int64, 0, count)
	for i := begin; i > begin-count; i -= 1 {
		meta.Stickers = append(meta.Stickers, int64(i))
	}

	metaData, err := json.Marshal(meta)
	if err != nil {
		logger.Println(err)
		return
	}

	rc := bytes.NewReader(metaData)

	metaPath := path.Join(dirPath, "productInfo.meta")
	metaFile, err := os.Create(metaPath)
	if err != nil {
		logger.Println(err)
		return
	}
	defer metaFile.Close()

	_, err = io.Copy(metaFile, rc)
	if err != nil {
		logger.Println(err)
		return
	}

	insert(id)
	fmt.Println("complete")

}
