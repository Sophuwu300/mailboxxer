package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/html"
	"io"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
			return t.Format("Mon 15:04")
		}
		if d.Hours() > 1 {
			return fmt.Sprintf("%d h ago", int(d.Hours()))
		}
		return fmt.Sprintf("%d m ago", int(d.Minutes()))
	}(), "th", (func(day int) string {
		if day/10 == 1 {
			return "th"
		}
		switch day % 10 {
		case 1:
			return "st"
		case 2:
			return "nd"
		case 3:
			return "rd"
		default:
			return "th"
		}
	})(t.Day()))
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

func OpenMail(metas []EmailMeta, page int, h int, s string, w int) error {
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
	var b []byte
	path, err := filepath.Glob(filepath.Join(SAVEPATH, id, "body.*"))
	if len(path) == 0 {
		return fmt.Errorf("no email found")
	}
	b, err = os.ReadFile(path[0])
	if err != nil {
		return err
	}
	s = string(b)
	if filepath.Ext(path[0]) == ".html" {
		s = RenderHTML(s)
	}
	w -= 4
	s = wrap(w, s)

	cmd := exec.Command("less", "-sR")
	cmd.Stdin = strings.NewReader(s)
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
	w, h := GetTSize()
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
			if err := OpenMail(*metas, page, h, s, w); err != nil {
				fmt.Println(err)
			}
		}
	}
}
