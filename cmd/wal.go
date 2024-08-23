package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"encoding/gob"

	"go.etcd.io/bbolt"
)

/*
Layout of database:

bucket uploads: key: "" value: gob encoded []string that is a list of all currently known uploadsets
bucket {QueueID}: key: "path" value: ""
*/

type QueueID = int

type queues struct {
	IDs []string
}

func newQueues() *queues {
	return &queues{IDs: []string{}}
}

type WAL struct {
	db *bbolt.DB
}

func NewWAL(db *bbolt.DB) (*WAL, error) {
	wal := WAL{db: db}

	err := wal.init()
	if err != nil {
		return nil, err
	}

	return &wal, nil
}

func (wal *WAL) init() error {
	err := wal.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("uploads"))
		if err != nil {
			return fmt.Errorf("could not create wallist bucket: %w", err)
		}

		// does the list key exist? if not lets initalize it with queues
		if bs := bucket.Get([]byte("list")); !bytes.Equal(bs, []byte("")) {
			q := newQueues()

			buf := &bytes.Buffer{}
			err = gob.NewEncoder(buf).Encode(q)
			if err != nil {
				return fmt.Errorf("could not json.Marshal an empty queues struct from WAL: %w", err)
			}

			if err = bucket.Put([]byte("list"), buf.Bytes()); err != nil {
				return fmt.Errorf("could not add queues json to uploads bucket: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not update bbolt database with new upload list: %w", err)
	}

	return nil
}

func randomUUID() ([]byte, error) {
	bs := make([]byte, 64)
	_, err := rand.Read(bs)
	if err != nil {
		return nil, fmt.Errorf("could not get a random byte slice from system: %w", err)
	}

	return []byte(base64.StdEncoding.EncodeToString(bs)), nil
}

func decodeWALListBytes(b []byte, e any) error {
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(e)
	return nil
}

func (wal *WAL) createQueue(paths []string) (QueueID, error) {
	err := wal.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("uploads"))

		randomID, err := randomUUID()
		if err != nil {
			return fmt.Errorf("could not get a randomUUID: %w", err)
		}

		listBytes := bucket.Get([]byte("list"))

		ques := newQueues()
		err = gob.NewDecoder(bytes.NewReader(listBytes)).Decode(ques)
		if err != nil {
			return fmt.Errorf("could not unmarshal gob queue list from uploads bucket: %w", err)
		}

		if slices.Index(ques.IDs, "") != -1 {
			// TODO: make error we can actually check against
			return errors.New("wal: uploads list already contains randomUUID, recommend retrying")
		}

		ques.IDs = append(ques.IDs, string(randomID))

		buf := &bytes.Buffer{}
		err = gob.NewEncoder(buf).Encode(ques)
		if err != nil {

		}

		err = bucket.Put([]byte(""), []byte(""))
		if err != nil {
			return fmt.Errorf("could not add new WALList to queue: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not update bbolt database with new upload list: %w", err)
	}

	return 0, nil
}
