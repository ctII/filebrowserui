package main

import (
	"os"
	"testing"
)

func TestLogin(t *testing.T) {
	sess, err := loginToFilebrowser(os.Getenv("FILEBROWSER_HOST"), os.Getenv("FILEBROWSER_USERNAME"), os.Getenv("FILEBROWSER_PASSWORD"))
	if err != nil {
		t.Fatal(err)
	}

	t.Log(sess)
}
