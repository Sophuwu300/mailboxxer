package main

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/html"
	"io"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sophuwu.site/mailboxxer/db"
	"strings"
)

func GetTSize() (int, int) {
	w, h, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func displayAddress(a string) string {
	addr, err := mail.ParseAddress(a)
	if err != nil {
		return a
	}
	if addr.Name != "" {
		return addr.Name
	}
	return addr.Address
}

func node(w io.Writer, n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		switch n.Data {
		case "script", "style", "head", "img":
			return
		case "b", "strong":
			fmt.Fprint(w, "\033[1m") // Bold
		case "i":
			fmt.Fprint(w, "\033[2m") // Italic
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			node(w, c)
		}

		// Reset styles after the element
		switch n.Data {
		case "b", "strong", "i", "em":
			fmt.Fprint(w, "\033[0m")
		}

	case html.TextNode:
		fmt.Fprint(w, n.Data, " ")

	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			node(w, c)
		}
	}

}

func RenderHTML(htmlContent string) string {
	doc, err := html.Parse(bytes.NewBufferString(htmlContent))
	if err != nil {
		panic(err)
	}

	var buff bytes.Buffer
	node(&buff, doc)
	return buff.String()
}

func DisplayRows(q *db.Query) {
	fmt.Printf("Page: %d/%d\n", q.Page(), q.TotalPages())
	fmt.Printf(" id | %13.13s | %40.40s | %s\n", "Date", "From", "Subject")
	for i, em := range q.Rows() {
		fmt.Printf("%3d | %13.13s | %40.40s | %s\n", i, db.TimeStr(em.Date), displayAddress(em.From), em.Subject)
	}
}

func wrap(w int, s string) string {
	var line = ""
	var lines, b []string
	for _, n := range strings.Split(s, "\n") {
		b = strings.Split(n, " ")
		for i, v := range b {
			if len(v) > w {
				lines = append(lines, v[:w-1], v[w-1:])
				continue
			}
			if len(line)+len(v) < w {
				line += v + " "
			} else {
				lines = append(lines, strings.TrimSuffix(line, " "))
				line = ""
				continue
			}
			if i == len(b)-1 {
				lines = append(lines, strings.TrimSuffix(line, " "))
				line = ""
			}
		}
	}
	lines = slices.Clip(lines)
	return "  " + strings.Join(lines, "  \n  ")
}

var ErrInvalidNumber = errors.New("invalid number")

func OpenMail(r *db.Query, s string) error {
	n, err := parseInt(s)
	if err != nil {
		return ErrInvalidNumber
	}
	var meta db.EmailMeta
	meta, err = r.Row(n)
	if err != nil {
		return ErrInvalidNumber
	}
	id := meta.Id
	var b []byte
	isHtml := false
	b, err = os.ReadFile(filepath.Join(db.SAVEPATH, id, "body.txt"))
	if errors.Is(err, os.ErrNotExist) {
		b, err = os.ReadFile(filepath.Join(db.SAVEPATH, id, "body.html"))
		isHtml = true
	}
	if err != nil {
		return err
	}
	s = string(b)
	if isHtml {
		s = RenderHTML(s)
	}
	cmd := exec.Command("less", "-sR")
	cmd.Stdin = strings.NewReader(s)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Print("\033[?1049l")
	err = cmd.Run()
	fmt.Print("\033[?1049h")
	return err
}

func getSize(W, H *int) bool {
	w, h := GetTSize()
	if h > 28 {
		h = 28
	} else if h < 5 {
		h = 5
	}
	h -= 4
	b := false
	if H != nil {
		if *H != h {
			b = true
		}
		*H = h
	}
	if W != nil {
		*W = w
	}
	return b
}

func parseInt(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func CLI() {
	var s string
	var w, h int
	getSize(&w, &h)
	r, err := db.NewQuery(h)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating query:", err)
		return
	}

	// alternative screen
	fmt.Print("\033[?1049h")
	// on exit restore terminal
	defer fmt.Print("\033[?1049l")

	for {
		fmt.Print("\033[2J\033[H\r")
		if getSize(&w, &h) {
			if err = r.SetPageSize(h); err != nil {
				break
			}
		}
		DisplayRows(r)
		fmt.Println("n: next page, p: previous  |  q: quit  |  <id>: open")
		fmt.Print("input (n/p/q/id): ")
		s = ""
		fmt.Scanln(&s)
		switch s {
		case "":
			continue
		case "q":
			return
		case "n":
			err = r.Next()
		case "p":
			err = r.Prev()
		default:
			if err = OpenMail(r, s); errors.Is(err, ErrInvalidNumber) {
				continue
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return
		}

	}
}
