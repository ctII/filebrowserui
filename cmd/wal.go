package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"go.etcd.io/bbolt"
)

/*
Layout of database:

bucket uploads: key: "" value: glob encoded []string that is a list of all currently known uploadsets
bucket {QueueID}: key: "path" value: ""
*/

type QueueID = string

type WAL struct {
	db *bbolt.DB
}

func (wal *WAL) init() error {
	err := wal.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("uploads"))
		if err != nil {
			return fmt.Errorf("could not create wallist bucket: %w", err)
		}

		bs := make([]byte, 64)
		_, err = rand.Read(bs)
		if err != nil {
			return fmt.Errorf("could not get a random byte slice from system: %w", err)
		}

		randomID := []byte(base64.StdEncoding.EncodeToString(bs))

		bs = bucket.Get([]byte(""))
		if !bytes.Equal(bs, []byte("")) {

		}

		err = bucket.Put([]byte(""), randomID)
		if err != nil {
			return fmt.Errorf("could not add new WALList to queue: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("could not update bbolt database with new upload list: %w", err)
	}

	return nil
}

func (wal *WAL) createUploadList(paths []string) (QueueID, error) {
	err := wal.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("uploads"))
		if err != nil {
			return fmt.Errorf("could not create wallist bucket: %w", err)
		}

		bs := make([]byte, 64)
		_, err = rand.Read(bs)
		if err != nil {
			return fmt.Errorf("could not get a random byte slice from system: %w", err)
		}

		randomID := []byte(base64.StdEncoding.EncodeToString(bs))

		bs = bucket.Get([]byte(""))
		if !bytes.Equal(bs, []byte("")) {

		}

		err = bucket.Put([]byte(""), randomID)
		if err != nil {
			return fmt.Errorf("could not add new WALList to queue: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("could not update bbolt database with new upload list: %w", err)
	}

	return "", nil
}
