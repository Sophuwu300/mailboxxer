package main

import (
	"bytes"
	"encoding/json"
	"github.com/asdine/storm"
	"go.etcd.io/bbolt"
	"net"
)

func DB() error {
	Db, err := storm.Open(DBPATH, storm.BoltOptions(0600, nil))
	if err != nil {
		return err
	}

	// update db and add bucket if not exists
	if err = db.Update(func(tx *bbolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("data"))
		return err
	}); err != nil {

		return err
	}
	if err = db.Init(&EmailMeta{}); err != nil {
		return err
	}
	db = Db
	return nil
}

var db *storm.DB = nil

func Listen() {
	sock, err := net.ListenUnix("unix", &net.UnixAddr{SOCK, "unix"})
	if err != nil {
		FtlLog(err)
	}
	defer sock.Close()
	if err = DB(); err != nil {
		FtlLog(err)
	}
	defer db.Close()
	for {
		conn, err := sock.Accept()
		if err != nil {
			db.Close()
			sock.Close()
			FtlLog(err)
		}
		go Handle(conn)
	}
}

type Req struct {
	CMD  string `json:"cmd"`
	Data []byte `json:"data"`
}

func Search(conn net.Conn, req *Req) {
	var err error
	var meta EmailMeta
	if json.Unmarshal(req.Data, &meta) != nil {
		return
	}
	var emails []EmailMeta
	if err = db.Find("Id", meta.Id, &emails); err != nil {
		return
	}
	var b []byte
	b, err = json.Marshal(emails)
	conn.Write(b)
	conn.Close()
}

func Handle(conn net.Conn) {
	var b bytes.Buffer
	var req Req
	_, err := b.ReadFrom(conn)
	if err != nil {
		return
	}
	if json.Unmarshal(b.Bytes(), &req) != nil {
		return
	}
	switch req.CMD {
	case "SEARCH":
		Search(conn, &req)
		return

	}
}
