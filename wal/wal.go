package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

type WriteAheadLog struct {
	db *bbolt.DB
}

func (w *WriteAheadLog) NewBatch(dir string) (Batch, error) {
	var bid []byte
	err := w.db.Update(func(tx *bbolt.Tx) error {
		batchesBucket := tx.Bucket([]byte("batches"))
		if batchesBucket == nil {
			return errors.New("WAL: batches bucket doesn't exist, bug/corruption?")
		}

		// Ignoring error as this can't error unless tx is closed or we are not
		// in a writable transaction
		id, _ := batchesBucket.NextSequence()
		bid = binary.AppendUvarint(nil, id)

		newBucket, err := batchesBucket.CreateBucket(bid)
		if err != nil {
			if errors.Is(err, bbolt.ErrBucketExists) {
				return fmt.Errorf("bucket (%v) exists, even though it shouldn't. this is a bug: %w", id, err)
			}
			return fmt.Errorf("could not create bucket (%v): %w", id, err)
		}

		if err := newBucket.Put([]byte("dest"), []byte(dir)); err != nil {
			return fmt.Errorf("could not add destination metadata to the batch bucket (%v): %w", bid, err)
		}

		return nil
	})
	if err != nil {
		return Batch{}, fmt.Errorf("WAL: could not update bbolt database to create new batch: %w", err)
	}

	return newBatch(bid, w.db), nil
}

func (w *WriteAheadLog) RemoveBatch(b Batch) error {
	err := w.db.Update(func(tx *bbolt.Tx) error {
		batches := tx.Bucket([]byte("batches"))
		if batches == nil {
			return errors.New("WAL: batches bucket doesn't exist, bug/corruption?")
		}

		if err := batches.DeleteBucket(b.id); err != nil {
			if errors.Is(err, bbolt.ErrBucketNotFound) {
				return fmt.Errorf("bucket (%v) not found in database: %w", string(b.id), err)
			}
			return fmt.Errorf("could not delete bucket (%v): %w", string(b.id), err)
		}

		// Reset sequence when we finish all the buckets
		if batches.Stats().BucketN-2 == 0 {
			if err := batches.SetSequence(0); err != nil {
				return fmt.Errorf("could not set sequence number to 0: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not update bbolt db: %w", err)
	}

	return nil
}

// ListBatches all batches currently in the WAL database.
func (w *WriteAheadLog) ListBatches() ([]Batch, error) {
	var batches []Batch
	err := w.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("batches"))
		if bucket == nil {
			return errors.New("WAL: batches bucket doesn't exist while trying to list batches, this likely means database corruption")
		}

		batches = make([]Batch, 0, bucket.Stats().KeyN)

		err := bucket.ForEach(func(k, _ []byte) error {
			if bytes.Equal(k, []byte("dest")) {
				return nil
			}

			// Copy ID into new byte slice, as keys are valid only in the transaction
			batches = append(batches, newBatch(append([]byte{}, k...), w.db))
			return nil
		})
		if err != nil {
			return fmt.Errorf("could not iterate batches bbolt bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not view bbolt database while listing batches: %w", err)
	}

	return batches, nil
}

func NewWriteAheadLog(db *bbolt.DB) (*WriteAheadLog, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		// Add metadata about our database versions for potential migrations
		metadata, err := tx.CreateBucketIfNotExists([]byte("metadata"))
		if err != nil {
			return fmt.Errorf("could not create \"metadata\" bucket in bbolt: %w", err)
		}

		if err := metadata.Put([]byte("version"), []byte("0.0.1")); err != nil {
			return fmt.Errorf("could not add version metadata to bbolt: %w", err)
		}

		// Add bucket to store a list of all our active batches
		_, err = tx.CreateBucketIfNotExists([]byte("batches"))
		if err != nil {
			return fmt.Errorf("could not create \"batches\" bucket in bbolt: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not update bbolt database to contain WriteAheadLog")
	}

	wal := &WriteAheadLog{
		db: db,
	}

	return wal, nil
}
