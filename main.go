package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"sync"

	_ "github.com/carylorrk/go-sqlite3"
)

var (
	workingDirectory string
	stickerDirectory string

	logger *Logger

	stickerDB     *sql.DB
	stickerDBLock sync.Mutex
)

func init() {
	setupWorkingDirectory()
	setupLogger()
	setupStickerDB()
}

func setupWorkingDirectory() {
	workingDirectory = os.Getenv("PONYSTICKER_PATH")
	if workingDirectory == "" {
		fmt.Println("$PONYSTICKER_PATH does not be set.")
		os.Exit(0)
	}

	err := os.MkdirAll(workingDirectory, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	stickerDirectory = path.Join(workingDirectory, "sticker")
	err = os.MkdirAll(stickerDirectory, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func setupStickerDB() {
	dbPath := path.Join(workingDirectory, "sticker.db")

	var err error
	stickerDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
	setupTable()
}

func setupTable() {
	err := stickerDB.Ping()
	if err != nil {
		logger.Println(err)
		os.Exit(-1)
	}

	stickerDBLock.Lock()
	defer stickerDBLock.Unlock()
	tx, err := stickerDB.Begin()
	if err != nil {
		logger.Println(err)
		os.Exit(-1)
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS official(
		packageId INTEGER PRIMARY KEY,
		meta TEXT,
		date INTEGER);
		CREATE INDEX IF NOT EXISTS officialidate ON official(date, packageId);
		CREATE VIRTUAL TABLE IF NOT EXISTS official_fts USING fts4 (
			packageId INTEGER,
			title TEXT,
			author TEXT);

		CREATE TABLE IF NOT EXISTS creator(
		packageId INTEGER PRIMARY KEY,
		meta TEXT,
		date INTEGER);
		CREATE INDEX IF NOT EXISTS creatoridate ON official(date, packageId);
		CREATE VIRTUAL TABLE IF NOT EXISTS creator_fts USING fts4 (
			packageId INTEGER,
			title TEXT,
			author TEXT);

		CREATE TABLE IF NOT EXISTS custom(
		packageId INTEGER PRIMARY KEY,
		meta TEXT,
		date INTEGER);
		CREATE INDEX IF NOT EXISTS customidate ON official(date, packageId);
		CREATE VIRTUAL TABLE IF NOT EXISTS custom_fts USING fts4 (
			packageId INTEGER,
			title TEXT,
			author TEXT);`)

	if err != nil {
		logger.Println(err)
		err = tx.Rollback()
		if err != nil {
			logger.Println(err)
		}
		os.Exit(-1)
	}

	err = tx.Commit()
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
}

func setupLogger() {
	logFilePath := path.Join(workingDirectory, "log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
	writer := io.MultiWriter(logFile, os.Stderr)
	logger = NewLogger(writer, "FreeLiner ", log.LstdFlags)
}

func main() {
	defer stickerDB.Close()
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "update":
		if len(os.Args) < 4 {
			fmt.Println("freeliner-server update <begin> <end>")
			os.Exit(0)
		}

		begin, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("begin must be a number.")
			os.Exit(0)
		}

		end, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("end must be a number.")
			os.Exit(0)
		}

		Update(begin, end)
	case "insert":
		if len(os.Args) < 3 {
			fmt.Println("freeliner-server insert <id>")
			os.Exit(0)
		}

		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("id must be a number.")
			os.Exit(0)
		}

		Insert(id)
	case "run":
		var port int
		if len(os.Args) < 3 {
			port = 50025
		} else {
			var err error
			port, err = strconv.Atoi(os.Args[2])
			if err != nil {
				fmt.Println("port must be a number.")
				os.Exit(0)
			}
		}

		Run(port)
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("freeliner-server <command>\n")
	fmt.Println("commands:")
	fmt.Println("  update <begin> <end>")
	fmt.Println("  run [port]")
	fmt.Println("  insert <id>")
}
