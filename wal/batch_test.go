package wal

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/simplylib/errgroup"
	"go.etcd.io/bbolt"
)

func TestBatch(t *testing.T) {
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

	batch, err := wal.NewBatch("/dev/null")
	if err != nil {
		t.Fatal(err)
	}

	dest, err := batch.Destination()
	if err != nil {
		t.Fatal(err)
	}

	if dest != "/dev/null" {
		t.Fatalf("destination of bucket was (%v) when it should have been (/dev/null)", dest)
	}

	logs, err := batch.ListUnfinished()
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 0 {
		t.Fatalf("expected len(logs)=0, got %v", len(logs))
	}

	err = batch.Start("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}

	err = batch.Start("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}

	err = batch.Finish("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}

	logs, err = batch.ListUnfinished()
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 0 {
		t.Fatalf("expected len(logs)=0, got %v", len(logs))
	}

	err = batch.Finish("/tmp/test")
	if err != nil {
		t.Fatal(err)
	}

	logs, err = batch.ListUnfinished()
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) != 0 {
		t.Fatalf("expected len(logs)=0, got %v", len(logs))
	}
}


