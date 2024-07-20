package main

import (
	"bytes"
	"fmt"
	"github.com/andybalholm/brotli"
	"strings"
	"time"
)

func TimeStr(s int64) string {
	t := time.Unix(s, 0).Local()
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
