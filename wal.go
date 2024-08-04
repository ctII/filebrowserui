package main

import "go.etcd.io/bbolt"

type WALlist = string

type WAL struct {
	db *bbolt.DB
}

func (wal *WAL) createUploadList(paths []string) WALlist {
	return ""
}
