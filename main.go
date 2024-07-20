package main

import (
	"fmt"
	"os"
	"path/filepath"
)

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

	// Print email details

}

/*#
CREATE TABLE `info` (
    `id` TEXT PRIMARY KEY,
    `from` TEXT,
    `to` TEXT,
    `subject` TEXT,
    `date` TEXT
);
CREATE TABLE `file` (
    `id` TEXT PRIMARY KEY,
    `data` TEXT
);
*/
