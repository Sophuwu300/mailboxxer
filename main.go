package main

import (
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) > 0 && filepath.Base(os.Args[0]) == "mailbox-parser" {
		Parse()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "parse" {
		Parse()
		return
	}
}
