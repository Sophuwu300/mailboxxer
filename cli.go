package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/DusanKasan/parsemail"
	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
	"io"
	"strings"
	"time"
)

func TimeStr(t time.Time) string {
	t = t.Local()
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
			return fmt.Sprintf("%d hours ago", int(d.Hours()))
		}
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}(), "th", (func() string {
		if t.Day()%9 < 1 || t.Day()%9 > 3 {
			return "th"
		}
		return []string{"st", "nd", "rd"}[t.Day()%9-1]
	})())
}

func UnBr(buf *bytes.Buffer) error {
	data := buf.Bytes()
	buf.Reset()
	reader := brotli.NewReader(bytes.NewReader(data))
	_, err := buf.ReadFrom(reader)
	return err
}

func Show(b []byte) {
	e, err := parsemail.Parse(bytes.NewReader(b))
	if err != nil {
		fmt.Println(err)
		return
	}
	if e.HTMLBody != "" {
		var b []byte
		if b, err = base64.StdEncoding.DecodeString(e.HTMLBody); err == nil {
			fmt.Println(RenderHTML(string(b)))
		} else {
			fmt.Println(RenderHTML(e.HTMLBody))
		}
	}
	for _, a := range e.Attachments {
		fmt.Println(a.Filename)
		fmt.Println(a.ContentType)
	}
	for _, h := range e.EmbeddedFiles {
		fmt.Println(h.ContentType)
		fmt.Println(h.CID)
	}
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
		case "h1", "h2", "p", "br":
			fmt.Fprint(w, "\033[0m\n")
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
