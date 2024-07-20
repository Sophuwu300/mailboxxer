package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"github.com/DusanKasan/parsemail"
	"github.com/andybalholm/brotli"
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
	if _, err := email.ReadFrom(os.Stdin); err != nil {
		os.Exit(1)
	}
	meta, err := GenerateMeta(email)
	if err != nil {
		os.Exit(1)
	}
	_ = meta
	err = Brotli(&email)
	if err != nil {
		os.Exit(1)
	}
	return meta, email
}

func dateHeader(e *mail.Header) int64 {
	var d time.Time
	var err error
	d, err = e.Date()
	if err != nil {
		d = time.Now().Local()
	}
	return d.UTC().Unix()
}

func ShaHash(b []byte) string {
	h := sha1.New()
	h.Write(b)
	return base32.StdEncoding.EncodeToString(h.Sum(nil))
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

	// e, err := mail.ReadMessage(&email)
	e, err := parsemail.Parse(&email)
	if err != nil {
		fmt.Println("Error parsing email: ", err)
		return em, err
	}

	em.To = func() string {
		if len(e.To) > 0 {
			return e.To[0].String()
		}
		if e.Header.Get("X-Original-To") != "" {
			return e.Header.Get("X-Original-To")
		}
		if e.Header.Get("Delivered-To") != "" {
			return e.Header.Get("Delivered-To")
		}
		return "Unknown Recipient"
	}()
	decode(&em.To)
	em.From = func() string {
		if len(e.From) > 0 {
			return e.From[0].String()
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
	From    string
	To      string
	Subject string
	Date    int64
	Id      string
}
