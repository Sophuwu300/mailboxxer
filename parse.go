package main

import (
	"bytes"
	"crypto/sha1"
	_ "encoding/json"
	"fmt"
	"github.com/andybalholm/brotli"
	_ "github.com/asdine/storm"
	"mime"
	"net/mail"
	"os"
	"time"
)

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

func Parse() (EmailMeta, bytes.Buffer) {
	var email bytes.Buffer
	_, err := email.ReadFrom(os.Stdin)
	FtlLog(err)
	meta, err := GenerateMeta(email)
	FtlLog(err)
	FtlLog(Brotli(&email))
	return meta, email
}

func dateHeader(e *mail.Header) time.Time {
	var d time.Time
	var err error
	d, err = e.Date()
	if err != nil {
		d = time.Now().Local()
	}
	return d
}
func ShaHash(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
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
	From    string    `json:"From" storm:"index"`
	To      string    `json:"To" storm:"index"`
	Subject string    `json:"Subject" storm:"index"`
	Date    time.Time `json:"Date" storm:"index"`
	Id      []byte    `json:"Id" storm:"unique,id"`
}
