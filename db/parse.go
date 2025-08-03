package db

import (
	"bytes"
	"crypto/sha1"
	_ "encoding/json"
	"fmt"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05 -0700"

func saveEmail(em EmailMeta, files FileList) error {
	path := filepath.Join(SAVEPATH, em.Id)
	var err error
	_ = os.MkdirAll(path, 0700)
	for name, data := range files {
		err = os.WriteFile(filepath.Join(path, name), data, 0600)
		if err != nil {
			return err
		}
	}
	return err
}

func newEntry(f os.DirEntry) error {
	if !f.Type().IsRegular() {
		return fmt.Errorf("unsupported file type in directory")
	}
	meta, files, err := parse(filepath.Join(INBOX, f.Name()))
	if err != nil {
		return fmt.Errorf("error parsing email: %w", err)
	}
	if meta.Date == "" {
		s, _ := f.Info()
		meta.Date = s.ModTime().Format(TimeFormat)
	}
	_, err = db.Exec("INSERT INTO emails (id, subject, toaddr, fromaddr, date) VALUES (?, ?, ?, ?, ?)", meta.Id, meta.Subject, meta.To, meta.From, meta.Date)
	if err != nil {
		return fmt.Errorf("error inserting email into database: %w", err)
	}
	err = saveEmail(meta, files)
	if err != nil {
		return fmt.Errorf("error saving email files: %w", err)
	}
	err = os.Remove(filepath.Join(INBOX, f.Name()))
	if err != nil {
		return fmt.Errorf("error cleaning tmp file: %w", err)
	}
	return nil
}

func parseNewMail() error {
	dir, err := os.ReadDir(INBOX)
	if err != nil {
		return err
	}
	for _, f := range dir {
		err = newEntry(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func parse(path string) (EmailMeta, FileList, error) {
	var email bytes.Buffer
	var meta EmailMeta
	var files FileList
	b, err := os.ReadFile(path)
	if err != nil {
		return meta, files, err
	}
	email.Write(b)
	meta, err = generateMeta(email)
	if err != nil {
		return meta, files, err
	}
	files, err = getFiles(&email)
	if err != nil {
		return meta, files, err
	}
	return meta, files, err
}

func dateHeader(e *mail.Header) string {
	var d time.Time
	var err error
	d, err = e.Date()
	if err != nil {
		return ""
	}
	return d.Format(TimeFormat)
}
func shaHash(b []byte) string {
	h := sha1.New()
	h.Write(b)
	return fmt.Sprintf("%X", h.Sum(nil))
}

func decodeR(s string) string {
	dec := new(mime.WordDecoder)
	decoded, _ := dec.DecodeHeader(s)
	return decoded
}

var decode = func(d mime.WordDecoder) func(s *string) {
	return func(s *string) {
		if ss, ers := d.DecodeHeader(fmt.Sprintf(*s)); ers == nil {
			*s = ss
		}
	}
}(mime.WordDecoder{})

// generateMeta generates the EmailMeta for the EmailData
// This is used to index the email in the database
func generateMeta(email bytes.Buffer) (EmailMeta, error) {
	var em EmailMeta
	em.Id = shaHash(email.Bytes())
	em.Subject = "No Subject"

	e, err := mail.ReadMessage(bytes.NewReader(email.Bytes()))
	if err != nil {
		return em, err
	}

	em.To = func() string {
		if len(e.Header.Get("To")) > 0 {
			return e.Header.Get("To")
		}
		if len(e.Header.Get("X-Original-To")) > 2 {
			return e.Header.Get("X-Original-To")
		}
		if e.Header.Get("Delivered-To") != "" {
			return e.Header.Get("Delivered-To")
		}
		return "Unknown Recipient"
	}()
	decode(&em.To)
	em.From = func() string {
		if len(e.Header.Get("From")) > 0 {
			return e.Header.Get("From")
		}
		if len(e.Header.Get("Return-Path")) > 2 {
			return e.Header.Get("Return-Path")
		}
		if e.Header.Get("Sender") != "" {
			return e.Header.Get("Sender")
		}
		return "Unknown Sender"
	}()
	decode(&em.From)
	if s := e.Header.Get("Subject"); s != "" {
		em.Subject = s
	}
	decode(&em.Subject)
	em.Date = dateHeader(&e.Header)

	return em, nil
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

// EmailMeta contains the fields that will be searchable in the database
type EmailMeta struct {
	From    string `json:"From"`
	To      string `json:"To"`
	Subject string `json:"Subject"`
	Date    string `json:"Date"`
	Id      string `json:"Id" `
}

func stdin() {
	var err error
	var b bytes.Buffer
	b.ReadFrom(os.Stdin)
	fl, e := getFiles(&b)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return
	}
	path := filepath.Dir(DBPATH)
	path = filepath.Join(path, "stdin")
	_ = os.MkdirAll(path, 0700)
	for name, data := range fl {
		err = os.WriteFile(filepath.Join(path, name), data, 0600)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	os.Exit(0)
}
