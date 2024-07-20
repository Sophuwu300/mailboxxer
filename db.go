package main

import (
	"bytes"
	"encoding/json"
	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"go.etcd.io/bbolt"
	"net"
	"os"
	"os/signal"
)

func DB() error {
	bb, err := storm.Open(DBPATH, storm.BoltOptions(0600, nil))
	if err != nil {
		return err
	}
	var e EmailMeta
	err = bb.Init(&e)
	if err != nil {
		return err
	}
	err = bb.Bolt.Update(func(tx *bbolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("data"))
		return err
	})
	if err != nil {
		return err
	}
	db = bb
	return nil
}

var db *storm.DB

func Listen() {
	err := DB()
	FtlLog(err)
	_ = os.Remove(SOCK)
	var sock *net.UnixListener
	sock, err = net.ListenUnix("unix", &net.UnixAddr{SOCK, "socket"})
	FtlLog(err)

	// handle signals
	go func() {
		chn := make(chan os.Signal, 1)
		signal.Notify(chn, os.Kill, os.Interrupt)
		<-chn
		db.Close()
		sock.Close()
		os.Remove(SOCK)
		FtlLog(err)
		os.Exit(0)
	}()
	var conn *net.UnixConn
	for {
		conn, err = sock.AcceptUnix()
		if err != nil {
			continue
		}
		go Handle(conn)
	}
}

type Req struct {
	CMD  string `json:"CMD"`
	Data []byte `json:"Data"`
}

func sendToSock(req Req) []byte {
	var conn *net.UnixConn
	var err error
	conn, err = net.DialUnix("unix", &net.UnixAddr{
		Name: SOCK,
		Net:  "local",
	}, &net.UnixAddr{Name: SOCK, Net: "socket"})
	FtlLog(err)
	b, err := json.Marshal(req)
	FtlLog(err)
	var n int
	conn.WriteTo(b, &net.UnixAddr{Name: SOCK, Net: "socket"})
	n, err = conn.Write(b)
	FtlLog(err)
	println(n)
	conn.ReadFrom(b)

	return b
}

func Search(req Req) []byte {
	var err error
	var search string
	if err = json.Unmarshal(req.Data, &search); err != nil {
		return []byte("error unmarshalling request")
	}
	println(search)
	search += "*"
	var emails []EmailMeta
	query := db.Select(q.Or(q.Re("Subject", search), q.Re("From", search), q.Re("To", search)))
	_ = query.Find(&emails)
	var b []byte
	b, _ = json.Marshal(emails)
	return b
}

func Handle(conn *net.UnixConn) {
	var b bytes.Buffer
	var buf = make([]byte, 1024)
	var n int
	var err error
	for {
		buf = make([]byte, 1024)
		n, err = conn.Read(buf)
		if err != nil || n == 0 {
			break
		}
		b.Write(buf[:n])
	}
	var req Req
	err = json.Unmarshal(b.Bytes(), &req)
	if err != nil {
		return
	}
	var bb []byte
	switch req.CMD {
	case "SEARCH":
		bb = Search(req)
		break
	case "PUT":
		bb = Put(req)
		break
	}
	n, err = conn.Write(bb)
	if err != nil {
		return
	}
	println(n)
	conn.Close()
}

type PUT struct {
	M EmailMeta `json:"M"`
	D []byte    `json:"D"`
}

func Put(req Req) []byte {
	var put PUT
	if json.Unmarshal(req.Data, &put) != nil {
		return []byte("ERR")
	}
	db.Save(&put.M)
	db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("data")).Put(put.M.Id, put.D)
	})
	return []byte("OK")
}
