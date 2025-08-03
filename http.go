package main

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

type ReturnQuery struct {
	EM  []EmailMeta
	Err error
}

type Query struct {
	Query  string
	Return chan ReturnQuery
}

func (q *Query) Error(err error) {
	if q.Return != nil {
		q.Return <- ReturnQuery{
			Err: err,
		}
	}
}
func (q *Query) Result(m []EmailMeta, err error) {
	if q.Return != nil {
		q.Return <- ReturnQuery{
			EM:  m,
			Err: err,
		}
	}
}

var QChan = make(chan *Query, 10)

func Web(db *sql.DB) {
	go func() {
		err := http.ListenAndServe("127.0.1.69:3141", http.HandlerFunc(Http))
		if err != nil {
			QChan <- &Query{Query: "EXIT"}
		}
	}()
	var r *sql.Rows
	var q *Query
	var err error
	for {
		q = <-QChan
		if q.Query == "EXIT" {
			q.Error(fmt.Errorf("server stopped"))
			break
		}
		if q.Query == "" {
			q.Error(fmt.Errorf("empty query"))
			continue
		}
		if r, err = db.Query(q.Query); err != nil {
			q.Error(fmt.Errorf("query error: %w", err))
			continue
		}
		q.Result(ReadRows(r))
	}
	close(QChan)
}

func E(s ...string) []any {
	a := make([]any, len(s))
	for i, v := range s {
		a[i] = html.EscapeString(v)
	}
	return a
}

type HtmlEM struct {
	Id       string
	Date     string
	Subject  string
	ToName   string
	ToAddr   string
	FromName string
	FromAddr string
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
		qu := Query{
			Query: fmt.Sprintf("SELECT * FROM emails %s ORDER BY date DESC", func() string {
				if len(q) == 0 {
					return ""
				}
				return "WHERE " + strings.Join(q, " AND ")
			}()),
			Return: make(chan ReturnQuery),
		}
		QChan <- &qu
		ret := <-qu.Return
		close(qu.Return)
		if ret.Err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		if len(ret.EM) == 0 {
			fmt.Fprint(w, `<html><head><title>No Emails</title></head><body><h1>No Emails Found</h1></body></html>`)
			return
		}
		var htmlMetas []HtmlEM
		var err error
		var from *mail.Address
		var to []*mail.Address
		var addrlist []string
		var htmlMeta HtmlEM
		for _, em := range ret.EM {
			htmlMeta = HtmlEM{
				Id:      em.Id,
				Date:    TimeStr(em.Date),
				Subject: em.Subject,
			}
			from, err = mail.ParseAddress(em.From)
			if err != nil {
				htmlMeta.FromAddr = em.From
			} else {
				htmlMeta.FromAddr = from.Address
				htmlMeta.FromName = from.Name
			}
			if htmlMeta.FromName == "" {
				htmlMeta.FromName = htmlMeta.FromAddr
			}
			to, err = mail.ParseAddressList(em.To)
			if err != nil || len(to) == 0 {
				addrlist = strings.Split(em.To, ", ")
				if len(addrlist) == 0 {
					htmlMeta.ToAddr = em.To
				} else {
					htmlMeta.ToAddr = addrlist[0]
				}
			} else {
				htmlMeta.ToAddr = to[0].Address
				htmlMeta.ToName = to[0].Name
			}
			if htmlMeta.ToName == "" {
				htmlMeta.ToName = htmlMeta.ToAddr
			}
			htmlMetas = append(htmlMetas, htmlMeta)
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
		for _, em := range htmlMetas {
			fmt.Fprintf(w, `<section onclick="location.href='%s/html.html'">
	<div class="time">%s</div>
	<div class="addr" title="%s">%s</div>
	<div class="addr" title="%s">%s</div>
	<div class="sub">%s</div>
</section>`, E(em.Id, em.Date, em.FromAddr, em.FromName, em.ToAddr, em.ToName, em.Subject)...)
		}
		fmt.Fprintf(w, "</body></html>")
		return
	}
	if filepath.Base(r.URL.Path) == "html.html" && len(strings.Split(r.URL.Path[1:], "/")) == 2 {
		id := filepath.Base(filepath.Dir(r.URL.Path))
		s, _ := os.ReadFile(filepath.Join(SAVEPATH, id, "header.txt"))
		b, _ := os.ReadFile(filepath.Join(SAVEPATH, id, "body.txt"))
		fmt.Fprintf(w, `<html><head><title>%s</title></head>
<body style="display: flex; flex-direction: row;">
<div style="width: 50%%;display:inline;height:100%%;overflow:scroll;"><pre>%s</pre><br><pre>%s</pre></div>
<iframe style="width: 50%%;display:inline;height:100%%;overflow:scroll;" src="%s"></iframe>
</body></html>`, id, string(s), string(b), "/"+id)
		return
	}
	http.FileServer(http.Dir(SAVEPATH)).ServeHTTP(w, r)
}
