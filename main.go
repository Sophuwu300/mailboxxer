package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var DBPATH, SOCK, LOG string

func init() {
	home, _ := os.UserHomeDir()
	if home == "" {
		os.Exit(1)
	}
	DBPATH = filepath.Join(home, ".mailbox", "mail.storm")
	SOCK = filepath.Join(home, ".mailbox", "mail.sock")
	LOG = filepath.Join(home, ".mailbox", "box.log")
}

func FtlLog(e error) {
	if e == nil {
		return
	}
	log, _ := os.OpenFile(LOG, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	fmt.Fprintln(log, "Fatal: ", e.Error())
	log.Close()
	os.Exit(1)
}

func main() {
	if (len(os.Args) > 0 && filepath.Base(os.Args[0]) == "mailbox-parser") || (len(os.Args) > 1 && os.Args[1] == "parse") {
		meta, filebr := Parse()
		fmt.Println("Email ID: ", meta.Id, filebr.Len())
		fmt.Println("From ", meta.From)
		fmt.Println("To", meta.To)
		fmt.Println("Subject", meta.Subject)
		fmt.Println("Date", TimeStr(meta.Date))
		return
	}
	if (len(os.Args) > 0 && filepath.Base(os.Args[0]) == "mailbox-db") || (len(os.Args) > 1 && os.Args[1] == "db") {

		return
	}

}
