package wal

import (
	"os"
	"testing"

	"go.etcd.io/bbolt"
)

func TestWAL(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Fatal(err)
		}
	}()

	db, err := bbolt.Open(f.Name(), 0o775, nil)
	if err != nil {
		t.Fatal(err)
	}

	wal, err := NewWriteAheadLog(db)
	if err != nil {
		t.Fatal(err)
	}

	batch, err := wal.NewBatch("")
	if err != nil {
		t.Fatal(err)
	}

	batches, err := wal.ListBatches()
	if err != nil {
		t.Fatal(err)
	}

	if len(batches) != 1 {
		t.Fatalf("expected 1 batch instead got: %v", len(batches))
	}

	err = wal.RemoveBatch(batch)
	if err != nil {
		t.Fatal(err)
	}

	batches, err = wal.ListBatches()
	if err != nil {
		t.Fatal(err)
	}

	if len(batches) != 0 {
		t.Fatalf("expected 0 batches instead got: %v", len(batches))
	}
}
