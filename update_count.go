package main

func UpdateAllCount() {
	updateRepoCount("official")
	updateRepoCount("creator")
	updateRepoCount("custom")
}

func updateRepoCount(repo string) {
	stickerDBLock.Lock()
	defer stickerDBLock.Unlock()
	var err error
	var count int
	err = stickerDB.QueryRow("SELECT COUNT(*) FROM " + repo).Scan(&count)
	if err != nil {
		logger.Println(err)
		return
	}
	_, err = stickerDB.Exec(`UPDATE meta SET count=? WHERE name=?`, count, repo)
	if err != nil {
		logger.Println(err)
	}
}
