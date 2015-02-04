package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"
)

func Insert(id int) {
	logger.Println("insert", id)
	setupTable()
	insert(id)
}

func insert(id int) {
	metaPath := path.Join(stickerDirectory, fmt.Sprint(id), "productInfo.meta")
	metaFile, err := os.Open(metaPath)
	defer metaFile.Close()
	if err != nil {
		logger.Println(err)
		return
	}

	metaData, err := ioutil.ReadAll(metaFile)
	if err != nil {
		logger.Println(err)
		return
	}

	metaText := string(metaData)
	var meta Meta
	err = json.Unmarshal(metaData, &meta)
	if err != nil {
		logger.Println(err)
		return
	}

	var authorBuffer bytes.Buffer
	for _, author := range meta.Author {
		_, err = authorBuffer.WriteString(author)
		if err != nil {
			logger.Println(err)
			return
		}
	}

	stickerDBLock.Lock()
	defer stickerDBLock.Unlock()

	var titleBuffer bytes.Buffer
	for _, title := range meta.Title {
		_, err = titleBuffer.WriteString(title)
		if err != nil {
			logger.Println(err)
			return
		}
	}

	authorText := transformQueryText(authorBuffer.String())
	titleText := transformQueryText(titleBuffer.String())

	tx, err := stickerDB.Begin()
	if err != nil {
		logger.Println(err)
		return
	}

	repo := checkRepo(id)
	if repo == "" {
		logger.Println("cannot check repo", id)
		return
	}

	date := time.Now().Unix()
	_, err = tx.Exec(`INSERT INTO `+repo+` (packageId, meta, date)
					  VALUES (?, ?, ?);
					  INSERT INTO `+repo+"_fts"+` (packageId, title, author)
					  VALUES (?, ?, ?);`,
		id, metaText, date,
		id, titleText, authorText)

	if err != nil {
		logger.Println(err)
		err = tx.Rollback()
		if err != nil {
			logger.Println(err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		logger.Println(err)
		return
	}
}

func checkRepo(id int) string {
	switch {
	case id < 0:
		return "custom"
	case id >= 0 && id < 999999:
		return "official"
	case id >= 1000000:
		return "creator"
	}
	return ""
}
