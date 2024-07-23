package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/DusanKasan/parsemail"
	"github.com/andybalholm/brotli"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/html"
	"io"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func GetTSize() (int, int) {
	w, h, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func TimeStr(s string) string {
	t, err := time.Parse(TimeFormat, s)
	if err != nil {
		return s
	}
	n := time.Now().Local()
	return strings.ReplaceAll(func() string {
		if t.Year() != n.Year() {
			return t.Format("Jan 02th 2006")
		}
		d := time.Since(t)
		if d.Hours() > 24*6 {
			return t.Format("Jan 02th")
		}
		if d.Hours() > 24 {
			return t.Format("Monday 15:04")
		}
		if d.Hours() > 1 {
			return fmt.Sprintf("%d h ago", int(d.Hours()))
		}
		return fmt.Sprintf("%d m ago", int(d.Minutes()))
	}(), "th", (func() string {
		if t.Day()%9 < 1 || t.Day()%9 > 3 {
			return "th"
		}
		return []string{"st", "nd", "rd"}[t.Day()%9-1]
	})())
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

func UnBr(buf *bytes.Buffer) error {
	data := buf.Bytes()
	buf.Reset()
	reader := brotli.NewReader(bytes.NewReader(data))
	_, err := buf.ReadFrom(reader)
	return err
}

func Show(b *bytes.Buffer) error {
	e, err := parsemail.Parse(b)
	if err != nil {
		return err
	}
	var s string
	var ishtml bool
	if e.HTMLBody != "" {
		s = e.HTMLBody
		ishtml = true
	} else {
		s = e.TextBody
		ishtml = false
	}
	var bb []byte
	if bb, err = base64.StdEncoding.DecodeString(s); err == nil {
		s = string(bb)
	}
	if ishtml {
		s = RenderHTML(s)
	}
	b.Reset()
	b.WriteString(s)
	return nil
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

func (em EmailMeta) Display() string {
	s := fmt.Sprintf("%13.13s | ", TimeStr(em.Date))
	s += fmt.Sprintf("%40.40s | ", displayAddress(em.From))
	s += fmt.Sprintf("%s", em.Subject)
	return s
}

func DisplayRows(metas []EmailMeta, page int, h int) {
	fmt.Println("Page: ", page)
	fmt.Printf("id:\t%13.13s | %40.40s | %s\n", "Date", "From", "Subject")
	page *= h
	for i := page; i < len(metas) && i < page+h; i++ {
		fmt.Printf("%d:\t%s\n", i-page, metas[i].Display())
	}
}

func OpenMail(metas []EmailMeta, page int, h int, s string) error {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	if !strings.HasPrefix(s, fmt.Sprintf("%d", n)) {
		return fmt.Errorf("invalid number")
	}
	n += page * h
	if n >= len(metas) || n < 0 {
		return fmt.Errorf("invalid number")
	}
	id := metas[n].Id
	b, err := os.ReadFile(filepath.Join(SAVEPATH, id+".br"))
	if err != nil {
		return err
	}
	var email bytes.Buffer
	email.Write(b)
	if err = UnBr(&email); err != nil {
		return err
	}
	if err = Show(&email); err != nil {
		return err
	}
	cmd := exec.Command("less")
	cmd.Stdin = &email
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CLI(metas *[]EmailMeta) {
	// use alternate screen
	fmt.Print("\033[?1049h")
	defer fmt.Print("\033[?1049l")
	page := 0
	var s string
	_, h := GetTSize()
	h -= 5
	if h > 20 {
		h = 20
	} else if h < 5 {
		h = 5
	}
	for {
		// clear screen and move cursor to top
		fmt.Print("\033[2J\033[H\r")
		if page < 0 {
			page = 0
		}
		DisplayRows(*metas, page, h)
		fmt.Println("n: next, p: previous, q: quit, (id): open")
		fmt.Print("(n/p/q/id): ")
		fmt.Scanln(&s)
		switch s {
		case "n":
			page++
		case "p":
			page--
		case "q":
			return
		case "":
			continue
		default: // open mail
			if err := OpenMail(*metas, page, h, s); err != nil {
				fmt.Println(err)
			}
		}
	}
}
