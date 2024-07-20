package main

import (
	"github.com/asdine/storm"
	"go.etcd.io/bbolt"
)

func DB() *storm.DB { //
	db, err := storm.Open(DBPATH, storm.BoltOptions(0600, nil))
	if err != nil {
		return nil
	}

	// update db and add bucket if not exists
	if err = db.Update(func(tx *bbolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("data"))
		return err
	}); err != nil {

		return nil
	}
	if err = db.Init(&EmailMeta{}) != nil {
		return nil
	}

	return db
}
