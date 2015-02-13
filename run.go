package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	IMAGE_MIME_TYPE                = "image/jpeg"
	IMAGE_EXTENSION                = ".jpg"
	LINE_STORE_URL                 = "https://store.line.me"
	OFFICIAL_STICKER_SEARCH_FORMAT = LINE_STORE_URL + "/stickershop/search/en?page=%d&q=%s"
	CREATOR_STICKER_SEARCH_FORMAT  = LINE_STORE_URL + "/stickershop/search/creators/en?page=%d&q=%s"
	STICKER_LINK_FORMAT            = "/stickershop/product/%d"
)

func Run(port int) {
	logger.Println("run at port", port)
	http.HandleFunc("/", testHandler)
	http.HandleFunc("/meta", metaHandler)
	http.HandleFunc("/sticker", stickerHandler)
	http.HandleFunc("/pkg-list", pkgHandler)
	http.HandleFunc("/pkg-count", pkgCountHandler)
	logger.Fatal(http.ListenAndServe(":"+fmt.Sprint(port), nil))
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintln(w, "test at", time.Now(), "\n")
	fmt.Fprintln(w, "APIs:")
	fmt.Fprintln(w, "meta?repo=<REPO>&pkg=<INT>")
	fmt.Fprintln(w, "sticker?pkg=<INT>&sticker=<INT>[&base64=<0|1>]")
	fmt.Fprintln(w, "pkg-list?repo=<REPO>&page=<INT>&size=<INT>&order=<packageId|date>[&q=<STRING>]")
	fmt.Fprintln(w, "pkg-count?repo=<REPO>[&q=<STRING>]")
	fmt.Fprintln(w, "<REPO>=<official|creator|custom>")
}

func metaHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	packageId, err := strconv.Atoi(r.FormValue("pkg"))
	if err != nil {
		http.Error(w, "parameter pkg must be an integer", http.StatusBadRequest)
		return
	}

	repo := r.FormValue("repo")
	if repo != "official" && repo != "creator" && repo != "custom" {
		http.Error(w, "parameter repo format error", http.StatusBadRequest)
		return
	}

	stickerDBLock.Lock()
	defer stickerDBLock.Unlock()

	var meta string
	err = stickerDB.QueryRow("SELECT meta FROM "+repo+" WHERE packageId=?", packageId).Scan(&meta)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			http.Error(w, "no such package", http.StatusNotFound)
		} else {
			logger.Println(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	fmt.Fprint(w, meta)
	w.Header().Set("Content-Type", "application/json")
}

func stickerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	packageId := r.FormValue("pkg")
	if _, err := strconv.Atoi(packageId); err != nil {
		http.Error(w, "parameter pkg must be an integer", http.StatusBadRequest)
		return
	}

	sticker := r.FormValue("sticker")
	if sticker != "tab_on" && sticker != "tab_off" {
		if _, err := strconv.Atoi(strings.Replace(sticker, "_key", "", -1)); err != nil {
			http.Error(w, "parameter sticker format error", http.StatusBadRequest)
			return
		}
	}

	isBase64 := r.FormValue("base64")

	filePath := path.Join(stickerDirectory, packageId, sticker+IMAGE_EXTENSION)
	file, err := os.Open(filePath)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			http.Error(w, "no such file", http.StatusNotFound)
		} else {
			logger.Println(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	if isBase64 == "1" {
		buf := bytes.NewBuffer(nil)
		_, err = io.Copy(buf, file)
		if err != nil {
			logger.Panicln(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		str := base64.StdEncoding.EncodeToString(buf.Bytes())
		_, err := fmt.Fprint(w, str)
		if err != nil {
			logger.Panicln(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		_, err = io.Copy(w, file)
		if err != nil {
			logger.Panicln(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", IMAGE_MIME_TYPE)
}

func pkgCountHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	repo := r.FormValue("repo")
	if repo != "official" && repo != "creator" && repo != "custom" {
		http.Error(w, "parameter repo format error", http.StatusBadRequest)
		return
	}
	query := r.FormValue("q")

	var err error
	var count int
	if query == "" {
		err = stickerDB.QueryRow("SELECT COUNT(*) FROM " + repo).Scan(&count)
	} else {
		repo_fts := repo + "_fts"
		err = stickerDB.QueryRow("SELECT COUNT(meta) "+
			"FROM "+repo+","+repo_fts+
			" WHERE "+repo_fts+" MATCH ? "+
			"AND "+repo+".packageId="+repo_fts+".packageId",
			transformQueryText(query)).Scan(&count)
	}

	if err != nil {
		logger.Println(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = fmt.Fprint(w, count)
	if err != nil {
		logger.Println(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func pkgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		http.Error(w, "parameter page must be an integer", http.StatusBadRequest)
		return
	}

	size, err := strconv.Atoi(r.FormValue("size"))
	if err != nil {
		http.Error(w, "parameter size must be an integer", http.StatusBadRequest)
		return
	}

	repo := r.FormValue("repo")
	if repo != "official" && repo != "creator" && repo != "custom" {
		http.Error(w, "parameter repo format error", http.StatusBadRequest)
		return
	}

	order := r.FormValue("order")
	if order != "packageId" && order != "date" {
		http.Error(w, "parameter order format error", http.StatusBadRequest)
		return
	}

	query := r.FormValue("q")
	if query == "" {
		err = listPackage(w, repo, page, size, order)
	} else {
		err = queryPackage(w, repo, page, size, order, query)
	}

	if err != nil {
		logger.Println(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func queryPackage(w http.ResponseWriter, repo string, page, size int, order, query string) error {
	tx, err := stickerDB.Begin()
	if err != nil {
		return err
	}

	repo_fts := repo + "_fts"
	pkgRows, err := tx.Query("SELECT meta FROM "+repo+","+repo_fts+
		" WHERE "+repo_fts+" MATCH ? AND "+repo+".packageId="+repo_fts+".packageId "+
		"ORDER BY "+repo+"."+order+" "+
		"LIMIT ? OFFSET ?",
		transformQueryText(query), size, (page-1)*size)

	if err != nil {
		return err
	}

	metalist := make([]string, 0, size)
	for pkgRows.Next() {
		var meta string
		err = pkgRows.Scan(&meta)
		if err != nil {
			return err
		}
		metalist = append(metalist, meta)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	res := "[" + strings.Join(metalist, ",") + "]"
	fmt.Fprint(w, res)
	w.Header().Set("Content-Type", "application/json")
	return nil
}

func listPackage(w http.ResponseWriter, repo string, page, size int, order string) error {
	tx, err := stickerDB.Begin()
	if err != nil {
		return err
	}

	pkgRows, err := tx.Query(`SELECT meta FROM `+repo+
		` ORDER BY `+order+` LIMIT ?  OFFSET ?`,
		size, (page-1)*size)
	if err != nil {
		return err
	}

	metalist := make([]string, 0, size)
	for pkgRows.Next() {
		var meta string
		err = pkgRows.Scan(&meta)
		if err != nil {
			return err
		}
		metalist = append(metalist, meta)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	res := "[" + strings.Join(metalist, ",") + "]"
	fmt.Fprint(w, res)
	w.Header().Set("Content-Type", "application/json")
	return nil
}
