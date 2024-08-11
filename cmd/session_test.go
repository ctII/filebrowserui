package cmd

import (
	"testing"
)

func TestLogin(t *testing.T) {
	if err := parseConfig(); err != nil {
		t.Fatal(err)
	}

	sess, err := loginToFilebrowser(config.Host, config.User, config.Pass)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(sess)
}

func TestUpload(t *testing.T) {}
func TestSHA256(t *testing.T) {}
