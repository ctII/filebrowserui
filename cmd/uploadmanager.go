package cmd

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/ctII/filebrowserui/wal"
)

/*

internal documention:

*/

type cancellableQueue struct {
}

// uploadManager manages the paused, running, and failed uploads to the server
// as well as the ones that crashed and must be resumed.
type uploadManager struct {
	wal *wal.WriteAheadLog
	fb  *filebrowserSession

	onBatchItemError func(bid string, path string, err error)
	onGeneralError   func(err error)

	batchChannel chan wal.Batch
	stopChannel  chan error

	queue cancellableQueue
}

// removeIndexFromSlice by modifying the slice and returning the result, modifying the content of the original slice
func removeIndexFromSlice[S any](s []S, index int) []S {
	return slices.Clip(slices.Delete(s, index, index+1))
}

func (um *uploadManager) startItem(ctx context.Context, path string) {

}

func (um *uploadManager) startUploadingBatch(b wal.Batch, finished func(), cancel chan struct{}) {
	defer finished()

	// TODO: maybe get list from distribute batches instead of calling batch list
	unfinishedUploads, err := b.ListUnfinished()
	if err != nil {
		// TODO: throw a better error with the ability to cancel an item out of this batch
		um.onGeneralError(err)
		return
	}

	for _, unfinishedFilePath := range unfinishedUploads {
		select {
		case <-cancel:
			// Did we get cancelled?
		default:
		}

		go func() {
			// TODO: start item in the queue and limit by a user-defined variable
			// maybe some algo that checks the max number of files we should upload by testing
			_ = unfinishedFilePath
		}()
	}

	// TODO: maybe add a whole list to the batch at once
}

type uploadWork struct {
	// batch to be working on
	batch wal.Batch

	// should the uploadBatch worker stop its work?
	cancel chan struct{}
}

func (um *uploadManager) distributeBatches() {
	var (
		batchesMu sync.Mutex
		batches   map[string]uploadWork
	)
	for {
		select {
		case <-um.stopChannel:
			um.stopChannel <- nil
		case batch := <-um.batchChannel:
			work := uploadWork{
				batch:  batch,
				cancel: make(chan struct{}),
			}
			batchesMu.Lock()
			batches[batch.ID()] = work
			batchesMu.Unlock()

			go um.uploadBatch(batch, func() {
				batchesMu.Lock()
				delete(batches, batch.ID())
				batchesMu.Unlock()
			}, work.cancel)
		}
	}
}

// TODO: how can the GUI cancel a batch? or maybe edit the list of files to be uploaded?
// TODO: how does the GUI corrolate the specific batch error to starting an action (cancelling that batch or excluding a file and retrying)?
// TODO: should begin upload return a batch wrapper for cancelling/editting?

// BeginUpload of paths, returning no error if starting that upload was recorded.
func (um *uploadManager) BeginUpload(dir string, paths []string) error {
	batch, err := um.wal.NewBatch(dir)
	if err != nil {
		return fmt.Errorf("could not make new wal batch: %w", err)
	}

	for i := range paths {
		if err = batch.Start(paths[i]); err != nil {
			return fmt.Errorf("could not add path (%v) to the batch (%v): %w", paths[i], batch.ID(), err)
		}
	}

	// TODO tell uploadManager goroutine to start uploading this batch
	um.batchChannel <- batch

	return nil
}

// Start uploadManager goroutine. Returning once on-disk batches are sent to goroutine or an error occurs during disk loading
func (um *uploadManager) Start() error {
	// Get unfinished batches
	batches, err := um.wal.ListBatches()
	if err != nil {
		return fmt.Errorf("could not list wal batches: %w", err)
	}

	// Do a quick cleanup of dangling batches (those without any unfinished uploads)
	for i := range batches {
		paths, err := batches[i].ListUnfinished()
		if err != nil {
			return fmt.Errorf("could not list unfinished batches for (%v): %w", batches[i].ID(), err)
		}

		if len(paths) != 0 {
			continue
		}

		if err = um.wal.RemoveBatch(batches[i]); err != nil {
			return fmt.Errorf("could not remove batch: %w", err)
		}
	}

	// Start worker gorountine
	go um.distributeBatches()

	for i := range batches {
		um.batchChannel <- batches[i]
	}

	return nil
}

// Stop uploadManager goroutine.
func (um *uploadManager) Stop() error {
	return <-um.stopChannel
}

// TODO: add option that filepath.Walks a directoy and starts uploads while it is still walking

func newUploadManager(
	writeAheadLog *wal.WriteAheadLog,
	session *filebrowserSession,
	onBatchItemError func(bid string, path string, err error),
	onGeneralError func(err error),
) (*uploadManager, error) {
	return &uploadManager{
		wal:              writeAheadLog,
		fb:               session,
		onBatchItemError: onBatchItemError,
		onGeneralError:   onGeneralError,
		batchChannel:     make(chan wal.Batch, 10),
		stopChannel:      make(chan error),
	}, nil
}
