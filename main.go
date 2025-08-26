package main

import (
	"os"
	"fmt"
	"git.sophuwu.com/gophuwu/flags"
	"git.sophuwu.com/mailboxxer/db"
	"git.sophuwu.com/mailboxxer/web"
)

func init() {
	newFlag := flags.NewNewFlagWithHandler(func(err error){
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	})
	newFlag("mailbox", "m", "the directory for mailboxxer emails and databases", "$HOME/.mailbox")
	flags.AddHelp("mailbox", `inside the mailbox dir, there should be a postfix directory inbox named "inbox"`)
	newFlag("web", "", "run as web server instead of terminal user interface", false)
	newFlag("listen", "l", "set the ip and port when running in web mode", web.DefaultAddr)
	db.ChkErr(flags.ParseArgs())
}

func main() {
	db.Open()
	defer db.Close()
	isWeb, err := flags.GetBoolFlag("web")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var addr string
	if isWeb {
		addr, err = flags.GetStringFlag("listen")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		web.ServeHttp(addr)
		return
	}
	CLI()
}
