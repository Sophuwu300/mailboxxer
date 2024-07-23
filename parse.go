package main

import (
	"bytes"
	"crypto/sha1"
	_ "encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05 -0700"

func Brotli(buf *bytes.Buffer) error {
	data := buf.Bytes()
	buf.Reset()
	writer := brotli.NewWriter(buf)
	_, err := writer.Write(data)
	if err != nil {
		return err
	}
	return writer.Close()
}

func SaveEmail(em EmailMeta, email bytes.Buffer) error {
	path := filepath.Join(SAVEPATH, em.Id+".br")
	err := os.WriteFile(path, email.Bytes(), 0600)
	return err
}

func NewEntry(f os.DirEntry) (EmailMeta, error) {
	if !f.Type().IsRegular() {
		return EmailMeta{}, fmt.Errorf("unsupported file type in directory")
	}
	meta, email, err := Parse(filepath.Join(INBOX, f.Name()))
	if err != nil {
		return meta, err
	}
	err = SaveEmail(meta, email)
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

func Parse(path string) (EmailMeta, bytes.Buffer, error) {
	var email bytes.Buffer
	var meta EmailMeta
	b, err := os.ReadFile(path)
	if err != nil {
		return meta, email, err
	}
	email.Write(b)
	meta, err = GenerateMeta(email)
	if err != nil {
		return meta, email, err
	}
	err = Brotli(&email)
	return meta, email, err
}

func dateHeader(e *mail.Header) string {
	var d time.Time
	var err error
	d, err = e.Date()
	if err != nil {
		d = time.Now().Local()
	}
	return d.Format(TimeFormat)
}
func ShaHash(b []byte) string {
	h := sha1.New()
	h.Write(b)
	return fmt.Sprintf("%X", h.Sum(nil))
}

// GenerateMeta generates the EmailMeta for the EmailData
// This is used to index the email in the database
func GenerateMeta(email bytes.Buffer) (EmailMeta, error) {
	var em EmailMeta
	em.Id = ShaHash(email.Bytes())
	em.Subject = "No Subject"

	decode := func(d mime.WordDecoder) func(s *string) {
		return func(s *string) {
			if ss, ers := d.DecodeHeader(fmt.Sprintf(*s)); ers == nil {
				*s = ss
			}
		}
	}(mime.WordDecoder{})

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
