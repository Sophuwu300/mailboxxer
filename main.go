package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/DusanKasan/parsemail"
	"github.com/andybalholm/brotli"
	"os"
	"time"
)

func br(buf *bytes.Buffer) error {
	data := buf.Bytes()
	buf.Reset()
	// Create a new buffer to hold the compressed data
	writer := brotli.NewWriter(buf)
	_, err := writer.Write(data)
	if err != nil {
		return err
	}
	// Close the writer to flush any remaining data
	return writer.Close()
}

func main() {
	// if len(os.Args)>0 && filepath.Base(os.Args[0])== "mailbox-parse"{
	var email bytes.Buffer
	if _, err := email.ReadFrom(os.Stdin); err != nil {
		os.Exit(1)
	}
	meta, err := GenerateMeta(email)
	if err != nil {
		os.Exit(1)
	}
	js, _ := json.Marshal(meta)
	fmt.Println(string(js))
	_ = br(&email)

	// }
}

// GenerateMeta generates the EmailMeta for the EmailData
// This is used to index the email in the database
func GenerateMeta(email bytes.Buffer) (EmailMeta, error) {
	var em EmailMeta
	em.Id = fmt.Sprintf("%X", sha1.Sum(email.Bytes()))

	e, err := parsemail.Parse(&email)

	if err != nil {
		return em, err
	}
	if e.Header["X-Original-To"] != nil {
		em.To = e.Header["X-Original-To"][0]
	} else if len(e.To) > 0 {
		em.To = e.To[0].String()
	} else {
		em.To = "Unknown"
	}
	if len(e.From) > 0 {
		em.From = e.From[0].String()
	} else if e.Header["Return-Path"] != nil {
		em.From = e.Header["Return-Path"][0]
	} else {
		em.From = "Unknown"
	}
	if len(e.Subject) > 0 {
		em.Subject = e.Subject
	} else {
		em.Subject = "No Subject"
	}
	if !e.Date.IsZero() {
		em.Date = e.Date.Local().Unix()
	} else {
		em.Date = time.Now().Local().Unix()
	}

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
