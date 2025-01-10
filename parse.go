package main

import (
	"bytes"
	"crypto/sha1"
	_ "encoding/json"
	"fmt"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05 -0700"

func SaveEmail(em EmailMeta, files FileList) error {
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

func NewEntry(f os.DirEntry) (EmailMeta, error) {
	if !f.Type().IsRegular() {
		return EmailMeta{}, fmt.Errorf("unsupported file type in directory")
	}
	meta, files, err := Parse(filepath.Join(INBOX, f.Name()))
	if err != nil {
		return meta, err
	}
	if meta.Date == "" {
		s, _ := f.Info()
		meta.Date = s.ModTime().Format(TimeFormat)
	}
	err = SaveEmail(meta, files)
	if err != nil {
		return meta, err
	}
	err = os.Remove(filepath.Join(INBOX, f.Name()))
	return meta, err
}

func ScanDir(newEmails *[]EmailMeta) error {
	dir, err := os.ReadDir(INBOX)
	if err != nil {
		return err
	}
	var meta EmailMeta
	for _, f := range dir {
		meta, err = NewEntry(f)
		if err != nil {
			return err
		}
		*newEmails = append(*newEmails, meta)
	}
	return nil
}

func Parse(path string) (EmailMeta, FileList, error) {
	var email bytes.Buffer
	var meta EmailMeta
	var files FileList
	b, err := os.ReadFile(path)
	if err != nil {
		return meta, files, err
	}
	email.Write(b)
	meta, err = GenerateMeta(email)
	if err != nil {
		return meta, files, err
	}
	files, err = GetFiles(&email)
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
func ShaHash(b []byte) string {
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

// GenerateMeta generates the EmailMeta for the EmailData
// This is used to index the email in the database
func GenerateMeta(email bytes.Buffer) (EmailMeta, error) {
	var em EmailMeta
	em.Id = ShaHash(email.Bytes())
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
	fl, e := GetFiles(&b)
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
