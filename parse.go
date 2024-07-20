package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/andybalholm/brotli"
	"net/mail"
	"os"
	"time"
)

func Br(buf *bytes.Buffer) error {
	data := buf.Bytes()
	buf.Reset()
	writer := brotli.NewWriter(buf)
	_, err := writer.Write(data)
	if err != nil {
		return err
	}
	return writer.Close()
}

func Parse() {
	var email bytes.Buffer
	if _, err := email.ReadFrom(os.Stdin); err != nil {
		os.Exit(1)
	}
	meta, err := GenerateMeta(email)
	if err != nil {
		os.Exit(1)
	}
	_ = meta
	if Br(&email) != nil {
		os.Exit(1)
	}
	fmt.Printf("%v\n", meta)
}

// firstHeader returns the first email address in the list of headers
func firstHeader(e *mail.Header, s ...string) string {
	var addr []*mail.Address
	var err error
	for _, v := range s {
		if addr, err = e.AddressList(v); err == nil && len(addr) > 0 {
			return addr[0].String()
		}
	}
	return "Unknown"
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

// GenerateMeta generates the EmailMeta for the EmailData
// This is used to index the email in the database
func GenerateMeta(email bytes.Buffer) (EmailMeta, error) {
	var em EmailMeta
	em.Id = fmt.Sprintf("%X", sha1.Sum(email.Bytes()))
	em.Subject = "No Subject"

	// e, err := parsemail.Parse(&email)
	e, err := mail.ReadMessage(&email)
	if err != nil {
		return em, err
	}
	em.To = firstHeader(&e.Header, "To", "X-Original-To", "Delivered-To")
	em.From = firstHeader(&e.Header, "From", "Reply-To", "Return-Path", "Sender")
	if s := e.Header.Get("Subject"); s != "" {
		em.Subject = s
	}
	em.Date = dateHeader(&e.Header)
	return em, nil
}

// EmailMeta contains the fields that will be searchable in the database
type EmailMeta struct {
	From    string `storm:"index" json:"from"`
	To      string `storm:"index" json:"to"`
	Subject string `storm:"index" json:"subject"`
	Date    int64  `storm:"index" json:"date"`
	Id      string `storm:"id" json:"id"`
}
