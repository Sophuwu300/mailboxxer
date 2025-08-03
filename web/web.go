package web

import (
	_ "embed"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sophuwu.site/mailboxxer/db"
	"strings"
)

//go:embed templates/index.html
var htmlTemplate string

var t *template.Template

func ServeHttp() {
	t = template.Must(template.New("index").Parse(htmlTemplate))
	http.ListenAndServe("127.0.1.69:3141", Http())
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

func ParseInt(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func TempErr(w http.ResponseWriter, code int) {
	msg := "An error occurred"
	switch code {
	case 404:
		msg = "Not found"
	case 500:
		msg = "Internal server error"
	}
	var dat = map[string]any{
		"Error":     msg,
		"ErrorCode": code,
	}
	t.Execute(w, dat)
}

func Http() http.HandlerFunc {
	qu, errr := db.NewQuery(30)
	if errr != nil {
		fmt.Fprintln(os.Stderr, "Error creating query:", errr)
		db.Close()
		os.Exit(1)
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
			where := func() string {
				if len(q) == 0 {
					return ""
				}
				return strings.Join(q, " AND ")
			}()
			var err error
			if where != qu.GetWhere() {
				err = qu.SetWhere(where)
				if err != nil {
					TempErr(w, 500)
					return
				}
			}
			err = qu.SetPage(ParseInt(r.Form.Get("page")) - 1)
			if err != nil {
				TempErr(w, 500)
				return
			}
			if qu.TotalRows() == 0 {
				TempErr(w, 404)
				return
			}

			var htmlMetas []HtmlEM
			var from *mail.Address
			var to []*mail.Address
			var addrlist []string
			var htmlMeta HtmlEM
			for _, em := range qu.Rows() {
				htmlMeta = HtmlEM{
					Id:      em.Id,
					Date:    db.TimeStr(em.Date),
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
			dat := map[string]any{
				"ToAddr":     r.Form.Get("to"),
				"FromAddr":   r.Form.Get("from"),
				"Subject":    r.Form.Get("subject"),
				"Date":       r.Form.Get("date"),
				"Page":       qu.Page(),
				"TotalPages": qu.TotalPages(),
				"HtmlMetas":  htmlMetas,
			}
			t.Execute(w, dat)
			return
		}
		if filepath.Base(r.URL.Path) == "html.html" && len(strings.Split(r.URL.Path[1:], "/")) == 2 {
			id := filepath.Base(filepath.Dir(r.URL.Path))
			s, _ := os.ReadFile(filepath.Join(db.SAVEPATH, id, "header.txt"))
			b, _ := os.ReadFile(filepath.Join(db.SAVEPATH, id, "body.txt"))
			fmt.Fprintf(w, `<html><head><title>%s</title></head>
<body style="display: flex; flex-direction: row;">
<div style="width: 50%%;display:inline;height:100%%;overflow:scroll;"><pre>%s</pre><br><pre>%s</pre></div>
<iframe style="width: 50%%;display:inline;height:100%%;overflow:scroll;" src="%s"></iframe>
</body></html>`, id, string(s), string(b), "/"+id)
			return
		}
		if func(w http.ResponseWriter, r *http.Request) bool {
			var file string
			var id string
			l := strings.SplitN(r.URL.Path[1:], "/", 2)
			if len(l) == 0 {
				return false
			}
			if len(l) >= 1 {
				id = l[0]
			}
			if len(l) >= 2 {
				file = l[1]
			}
			if file != "" && filepath.Ext(file) != ".txt" {
				return false
			}
			de, err := os.ReadDir(filepath.Join(db.SAVEPATH, id))
			if err != nil {
				return false
			}

			darkMode := r.URL.Query().Get("dark")
			if darkMode == "" || darkMode == "0" || darkMode == "false" {
				darkMode = "0"
			} else {
				darkMode = "1"
			}
			dat := map[string]any{
				"DarkMode": darkMode,
				"Style":    `style="overflow: auto;"`,
			}

			var ss string
			var f os.DirEntry
			if file != "" {
				var b []byte
				for _, f = range de {
					if f.Name() == file && f.Type().IsRegular() {
						b, err = os.ReadFile(filepath.Join(db.SAVEPATH, id, file))
						break
					}
				}
				if err != nil || len(b) == 0 {
					return false
				}
				ss = string(b)
				if filepath.Ext(f.Name()) == ".txt" {
					dat["Text"] = ss
				} else {
					return false
				}
			} else {
				for _, f = range de {
					ss += fmt.Sprintf(`<a href="/%s`, filepath.Join(id, f.Name()))
					if filepath.Ext(f.Name()) == ".txt" {
						ss += fmt.Sprintf(`?dark=%s`, darkMode)
					}
					ss += fmt.Sprintf(`">%s</a><br>%c`, html.EscapeString(f.Name()), '\n')
				}
				ss = `<main ` + fmt.Sprint(dat["Style"]) + `>` + ss + `</main>`
				dat["Html"] = template.HTML(ss)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			t.Execute(w, dat)
			return true
		}(w, r) {
			return
		}
		http.FileServer(http.Dir(db.SAVEPATH)).ServeHTTP(w, r)
	}
}
