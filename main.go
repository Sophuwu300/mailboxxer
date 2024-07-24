package main

import (
	"database/sql"
	"fmt"
	_ "github.com/glebarez/go-sqlite"
	"os"
	"path/filepath"
)

var DBPATH, INBOX, SAVEPATH string

func getHomeBox() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return filepath.Join(home, ".mailbox")
}

func init() {
	var mailbox string
	if len(os.Args) > 2 && os.Args[1] == "-m" {
		mailbox = os.Args[2]
	} else {
		mailbox = getHomeBox()
	}
	var err error
	if _, err = os.Stat(mailbox); os.IsNotExist(err) {
		os.MkdirAll(mailbox, 0700)
	}
	DBPATH = filepath.Join(mailbox, "mailbox.sqlite")
	INBOX = filepath.Join(mailbox, "inbox", "new")
	if _, err = os.Stat(INBOX); os.IsNotExist(err) {
		os.MkdirAll(INBOX, 0700)
	}
	SAVEPATH = filepath.Join(mailbox, "saved")
	if _, err = os.Stat(SAVEPATH); os.IsNotExist(err) {
		os.MkdirAll(SAVEPATH, 0700)
	}
}

func ReadRows(rows *sql.Rows) ([]EmailMeta, error) {
	var metas []EmailMeta
	var meta EmailMeta
	var err error
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&meta.Id, &meta.Subject, &meta.To, &meta.From, &meta.Date)
		if err != nil {
			return metas, err
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

func main() {
	db, err := sql.Open("sqlite", DBPATH)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS emails (id TEXT PRIMARY KEY, subject TEXT, toaddr TEXT, fromaddr TEXT, date TEXT)")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer func() {
		db.Exec("COMMIT")
		db.Close()
	}()
	var metas []EmailMeta
	err = ScanDir(&metas)
	for _, em := range metas {
		_, err = db.Exec("INSERT INTO emails (id, subject, toaddr, fromaddr, date) VALUES (?, ?, ?, ?, ?)", em.Id, em.Subject, em.To, em.From, em.Date)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	rows, err := db.Query("SELECT * FROM emails ORDER BY date DESC")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	metas, err = ReadRows(rows)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, v := range os.Args[1:] {
		if v == "--cli" {
			CLI(&metas)
			return
		}
	}
}
