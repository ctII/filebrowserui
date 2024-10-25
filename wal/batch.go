package wal

import (
	"bytes"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

type Batch struct {
	id []byte
	db *bbolt.DB
}

func newBatch(bid []byte, db *bbolt.DB) Batch {
	return Batch{
		id: bid,
		db: db,
	}
}

func (b *Batch) Start(name string) (exists bool, err error) {
	err = b.db.Update(func(tx *bbolt.Tx) error {
		batchesBucket := tx.Bucket([]byte("batches"))
		if batchesBucket == nil {
			return errors.New("batches bucket doesn't exist, when it should at this point in the program flow. Likely database corruption or a bug")
		}

		bucket := batchesBucket.Bucket(b.id)
		if bucket == nil {
			return fmt.Errorf("WAL: bucket of Batch(%v) doesn't exist, either some corruption or more likely this was called after bucket was deleted", b.id)
		}

		// Does this path already exist?
		if bs := bucket.Get([]byte(name)); bs != nil && len(bs) == 0 {
			exists = true
			return nil
		}

		if err := bucket.Put([]byte(name), []byte{}); err != nil {
			return fmt.Errorf("could not add key (%v) to the batches bucket: %w", name, err)
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("could not update bbolt database: %w", err)
	}

	return exists, nil
}

func (b *Batch) Finish(name string) error {
	err := b.db.Update(func(tx *bbolt.Tx) error {
		batchesBucket := tx.Bucket([]byte("batches"))
		if batchesBucket == nil {
			return errors.New("batches bucket doesn't exist, when it should at this point in the program flow. Likely database corruption or a bug")
		}

		bucket := batchesBucket.Bucket(b.id)
		if bucket == nil {
			return fmt.Errorf("WAL: bucket of Batch(%v) doesn't exist, either some corruption or more likely this was called after bucket was deleted", b.id)
		}

		if err := bucket.Delete([]byte(name)); err != nil {
			return fmt.Errorf("could not delete bucket (%v) key while Finishing a batch item: %w", b.id, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not update bbolt database: %w", err)
	}

	return nil
}

// ListUnfinished strings in the batch, this will give a snapshot of the currently unfinished keys.
func (b *Batch) ListUnfinished() ([]string, error) {
	var list []string

	err := b.db.View(func(tx *bbolt.Tx) error {
		batchesBucket := tx.Bucket([]byte("batches"))
		if batchesBucket == nil {
			return errors.New("batches bucket doesn't exist, when it should at this point in the program flow. Likely database corruption or a bug")
		}

		bucket := batchesBucket.Bucket(b.id)
		if bucket == nil {
			return fmt.Errorf("WAL: bucket of Batch(%v) doesn't exist, either some corruption or more likely this was called after bucket was deleted", b.id)
		}

		list = make([]string, 0, bucket.Stats().KeyN)

		err := bucket.ForEach(func(k, _ []byte) error {
			if bytes.Equal(k, []byte("dest")) {
				return nil
			}

			list = append(list, string(k))
			return nil
		})
		if err != nil {
			return fmt.Errorf("could not iterate over bucket (%v) keys: %w", string(b.id), err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not view bbolt db to list unfinished items in bucket (%v): %w", string(b.id), err)
	}

	return list, nil
}

func (b *Batch) Destination() (string, error) {
	var destination string

	err := b.db.View(func(tx *bbolt.Tx) error {
		batchesBucket := tx.Bucket([]byte("batches"))
		if batchesBucket == nil {
			return errors.New("batches bucket doesn't exist, when it should at this point in the program flow. Likely database corruption or a bug")
		}

		bucket := batchesBucket.Bucket(b.id)
		if bucket == nil {
			return fmt.Errorf("WAL: bucket of Batch(%v) doesn't exist, either some corruption or more likely this was called after bucket was deleted", b.id)
		}

		dest := bucket.Get([]byte("dest"))
		if dest == nil {
			return errors.New("dest field is nil on a bucket that exists, bug/corruption in library?")
		}

		destination = string(dest)

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("could not get destination from bboltdb: %w", err)
	}

	return destination, nil
}
