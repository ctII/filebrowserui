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

func TestWALRaciness(t *testing.T) {
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

	const (
		nRoutines int = 1e2
		nSubpaths int = 1e3
	)

	eg := errgroup.Group{}
	for i := range nRoutines {
		eg.Go(func() error {
			name := strconv.Itoa(i)

			b, err := wal.NewBatch(name)
			if err != nil {
				return fmt.Errorf("could not create new batch (%v): %w", name, err)
			}

			for i := range nSubpaths {
				if err := b.Start(name); err != nil {
					return fmt.Errorf("could not start subpath (%v) in batch (%v): %w", i, name, err)
				}
				if err := b.Finish(name); err != nil {
					return fmt.Errorf("could not finish subpath (%v) in batch (%v): %w", i, name, err)
				}
			}

			if err := wal.RemoveBatch(b); err != nil {
				return fmt.Errorf("could not remove batch (%v) after finishing tasks: %w", name, err)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}

	batches, err := wal.ListBatches()
	if err != nil {
		t.Fatal(err)
	}

	if len(batches) != 0 {
		t.Fatalf("there are batches when there shouldn't be after goroutines: %v", len(batches))
	}
}
