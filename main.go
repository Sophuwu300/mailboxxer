package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var DBPATH, SOCK, LOG string

func init() {
	home, err := os.UserHomeDir()
	FtlLog(err)
	if home == "" {
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(home, ".mailbox")); os.IsNotExist(err) {
		os.Mkdir(filepath.Join(home, ".mailbox"), 0755)
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
	fmt.Fprintln(log, "Fatal: ", e)
	log.Close()
	os.Exit(1)
}

func main() {
	if (len(os.Args) > 0 && os.Args[0] == "mailbox-parser") || (len(os.Args) > 1 && os.Args[1] == "parse") {
		meta, filebr := Parse()
		var put PUT
		put.M = meta
		put.D = filebr.Bytes()
		b, err := json.Marshal(put)
		FtlLog(err)
		var r Req
		r.CMD = "PUT"
		r.Data = b
		FtlLog(err)
		b = sendToSock(r)
		fmt.Println(string(b))
	}
	if (len(os.Args) > 0 && os.Args[0] == "mailbox-db") || (len(os.Args) > 1 && os.Args[1] == "db") {
		Listen()
		return
	} else if len(os.Args) > 1 && os.Args[1] == "search" {
		var r Req
		r.CMD = "SEARCH"
		b, _ := json.Marshal(os.Args[2])
		r.Data = b
		fmt.Println(string(sendToSock(r)))

	}

}
