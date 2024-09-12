package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	config = &Config{}
	if err := parseConfigPath("../config-test.json"); err != nil {
		fmt.Printf("could not parse config path ../config-test.json for session testing: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestLogin(t *testing.T) {
	sess, err := loginToFilebrowser(config.Host, config.User, config.Pass)
	if err != nil {
		t.Fatal(err)
	}

	_ = sess
}

func TestUpload(t *testing.T) {
	sess, err := loginToFilebrowser(config.Host, config.User, config.Pass)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("Hello World!")

	err = sess.uploadReader(context.Background(), "/data/", "helloworld.txt", bytes.NewReader(payload), int64(len(payload)), false)
	if err != nil {
		t.Fatalf("error while uploading payload (%v): %v", string(payload), err)
	}

	_, err = sess.Info(context.Background(), "/data/helloworld.txt")
	if err != nil {
		t.Fatalf("error while getting info about uploaded file: %v", err)
	}

	// while we use a sha256 sum to test, this isn't a 100% sure comparision
	payloadSum := sha256.New().Sum(payload)

	hashString, err := sess.SHA256(context.Background(), "/data/helloworld.txt")
	if err != nil {
		t.Fatalf("error while grabbing hash from filebrowser for (helloworld.txt): %v", err)
	}

	if bytes.Equal(payloadSum, []byte(hashString)) {
		t.Fatalf("payloadSum != filebrowser hash. payloadSum(%v) filebrowserHash(%v)", payloadSum, []byte(hashString))
	}
}

// TODO: this is implicitly tested by TestUpload, but it should be tested on its own
func TestSHA256(t *testing.T) {}
