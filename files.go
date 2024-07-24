package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
)

func getfiles(files *map[string][]byte, parts interface{}) {
	var body io.Reader
	var head textproto.MIMEHeader
	switch part := parts.(type) {
	case *mail.Message:
		body = part.Body
		head = textproto.MIMEHeader(part.Header)
	case *multipart.Part:
		body = part
		head = part.Header
	}
	mediaType, params, _ := mime.ParseMediaType(head.Get("Content-Type"))

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		var part *multipart.Part
		var err error
		for {
			if part, err = mr.NextPart(); err != nil {
				break
			}
			getfiles(files, part)
			part.Close()
		}
		return
	}
	content, _ := io.ReadAll(body)
	if head.Get("Content-Transfer-Encoding") == "base64" {
		content, _ = base64.StdEncoding.DecodeString(string(content))
	}
	if len(content) == 0 {
		return
	}
	name := params["name"]
	if strings.HasPrefix(mediaType, "text/") || strings.HasPrefix(mediaType, "message/") {
		name = "body.txt"
		if strings.Contains(mediaType, "html") {
			name = "index.html"
		}
	}
	(*files)[name] = append((*files)[name], content...)
}

func EmlFiles(eml *mail.Message) map[string][]byte {
	s := ""
	for k := range eml.Header {
		s += fmt.Sprintf("%s: %s\n", k, eml.Header.Get(k))
	}
	files := make(map[string][]byte)
	files["header.txt"] = []byte(s)
	getfiles(&files, eml)
	return files
}
