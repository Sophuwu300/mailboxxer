package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"
)

type FileList map[string][]byte

func getfiles(files *FileList, parts interface{}) {
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
	if mediaType == "text/html" || mediaType == "text/plain" {
		content = []byte(func(s string) string {
			s = strings.ReplaceAll(s, "=\n", "")
			s = strings.ReplaceAll(s, "=3D", "=")
			return s
		}(string(content)))
	}
	name := "body.txt"
	if params["name"] != "" {
		name = params["name"]
	}
	if strings.Contains(mediaType, "html") {
		name = "body.html"
	}
	(*files)[name] = append((*files)[name], content...)
	if head.Get("Content-ID") != "" {
		cid := head.Get("Content-ID")
		cid = strings.TrimPrefix(cid, "<")
		cid = strings.TrimSuffix(cid, ">")
		cid = "cidname: " + cid + " " + name + "\n"
		(*files)["header.txt"] = append((*files)["header.txt"], []byte(cid)...)
	}
}

var cidheader = regexp.MustCompile(`^cidname: [^ ]+ [^ ]+$`) // Content-ID header

func EmlFiles(eml *mail.Message, head []byte) FileList {

	files := make(FileList)
	getfiles(&files, eml)
	for _, v := range cidheader.FindAll(files["header.txt"], -1) {
		v = bytes.TrimSuffix(v, []byte("\n"))
		v = bytes.ReplaceAll(v, []byte("cidname: "), []byte("cid:"))
		n := bytes.Index(v, []byte(" "))
		files["body.html"] = bytes.ReplaceAll(files["body.html"], v[:n], v[n+1:])
	}
	files["header.txt"] = head
	return files
}

func GetFiles(b *bytes.Buffer) (FileList, error) {
	head := bytes.SplitN(b.Bytes(), []byte{10, 10}, 2)[0]
	head = bytes.ReplaceAll(head, []byte{'\t'}, []byte{' '})
	e, err := mail.ReadMessage(b)
	if err != nil {
		return nil, err
	}
	return EmlFiles(e, head), nil
}
