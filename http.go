package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"strings"
)

var ch = make(chan []byte)

func Web(db *sql.DB) {
	go func() {
		err := http.ListenAndServe("127.0.1.69:3141", http.HandlerFunc(Http))
		if err != nil {
			ch <- []byte("EXIT")
		}
	}()
	var b []byte
	var r *sql.Rows
	var m []EmailMeta
	var bb bytes.Buffer
	for {
		b = <-ch
		if string(b) == "EXIT" {
			break
		}
		r, _ = db.Query(string(b))
		m, _ = ReadRows(r)
		gob.NewEncoder(&bb).Encode(m)
		ch <- bb.Bytes()
		bb.Reset()
		r.Close()
	}
}

func E(s ...string) []any {
	a := make([]any, len(s))
	for i, v := range s {
		a[i] = html.EscapeString(v)
	}
	return a
}

func Http(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		r.ParseForm()
		var q []string
		if r.Form.Get("to") != "" {
			q = append(q, fmt.Sprintf(` toaddr LIKE '%%%s%%'`, r.Form.Get("to")))
		}
		if r.Form.Get("from") != "" {
			q = append(q, fmt.Sprintf(` fromaddr LIKE '%%%s%%'`, r.Form.Get("from")))
		}
		if r.Form.Get("subject") != "" {
			q = append(q, fmt.Sprintf(` subject LIKE '%%%s%%'`, r.Form.Get("subject")))
		}
		if r.Form.Get("date") != "" {
			q = append(q, fmt.Sprintf(` date LIKE '%%%s%%'`, r.Form.Get("date")))
		}
		ch <- []byte(fmt.Sprintf("SELECT * FROM emails %s ORDER BY date DESC", func() string {
			if len(q) == 0 {
				return ""
			}
			return "WHERE " + strings.Join(q, " AND ")
		}()))
		var metas []EmailMeta
		var b []byte
		b = <-ch
		dec := gob.NewDecoder(strings.NewReader(string(b)))
		if dec.Decode(&metas) != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		fmt.Fprint(w, `<html><head><title>Emails</title>
<style>
section {
	display: flex;
	flex-direction: row;
	cursor: pointer;
	border-bottom: 1px solid #666;
	width: 100%;
}
section > *{
	display: inline-block;
	padding: 5px;
}
section > div:last-child {
	border-right: none!important;
}	
section > div {
	border-right: 1px solid #666;
}	
section:hover {
	background-color: rgba(0,0,0,0.2);
}
.addr {
	font-weight: bold;
}
body {
	margin: 10px;
	display: flex;
	flex-direction: column;
}
marquee {
	width: 100%;
}
.addr > marquee {
	display: none;
}
.addr:hover > marquee {
	display: inline;
}
.addr:hover > span {
	display: none;
}
.addr > span {
	display: inline;
}
div {
	margin: 5px auto 5px auto;
	overflow: hidden;
}
.time {
	width: 13ch;
}
.sub {
	width: calc(100% - 13ch - 20%);
}
.addr {
	width: 15%;
	font-size: 0.8em;
}
</style>
</head><body>`)
		var from *mail.Address
		for _, em := range metas {
			from, _ = mail.ParseAddress(em.From)
			if from.Name == "" {
				if strings.Contains(from.Address, "@") {
					from.Name = strings.Split(from.Address, "@")[0]
				} else {
					from.Name = from.Address
				}
			}

			fmt.Fprintf(w, `<section onclick="location.href='%s'">
	<div class="time">%s</div>
	<div class="addr" title="%s">%s</div>
	<div class="addr" title="%s">%s</div>
	<div class="sub">%s</div>
</section>`, E(em.Id, TimeStr(em.Date), from.Address, from.Name, em.To, func() string {
				a, err := mail.ParseAddressList(em.To)
				if err != nil {
					return em.To
				}
				return strings.Split(a[0].Address+"@", "@")[0]
			}(), em.Subject)...)
		}
		fmt.Fprintf(w, "</body></html>")
		return
	}
	http.FileServer(http.Dir(SAVEPATH)).ServeHTTP(w, r)
}
