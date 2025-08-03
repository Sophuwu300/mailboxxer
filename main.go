package main

import (
	"os"
	"sophuwu.site/mailboxxer/db"
	"sophuwu.site/mailboxxer/web"
)

func main() {
	db.Open()
	defer db.Close()
	for _, arg := range os.Args[1:] {
		if arg == "--cli" {
			CLI()
			return
		}
		if arg == "--web" {
			web.ServeHttp()
			return
		}
	}
}
