package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"
)

type filebrowserSession struct {
	host  string
	token string
}

type Resource struct {
	// These fields exist only for Directories
	// TODO: maybe make these fields pointers, or move them to another struct
	Items []struct {
		Path      string    `json:"path"`
		Name      string    `json:"name"`
		Size      int       `json:"size"`
		Extension string    `json:"extension"`
		Modified  time.Time `json:"modified"`
		Mode      int64     `json:"mode"`
		IsDir     bool      `json:"isDir"`
		IsSymlink bool      `json:"isSymlink"`
		Type      string    `json:"type"`
	} `json:"items"`
	NumDirs  int `json:"numDirs"`
	NumFiles int `json:"numFiles"`
	Sorting  struct {
		By  string `json:"by"`
		Asc bool   `json:"asc"`
	} `json:"sorting"`

	// Every resource will have these aspects
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int       `json:"size"`
	Extension string    `json:"extension"`
	Modified  time.Time `json:"modified"`
	Mode      int64     `json:"mode"`
	IsDir     bool      `json:"isDir"`
	IsSymlink bool      `json:"isSymlink"`
	Type      string    `json:"type"`
}

func (sess *filebrowserSession) Info(ctx context.Context, filepath string) (*Resource, error) {
	slog.Debug("grabbing filebrowser resource", "path", filepath)

	uri, err := url.Parse(sess.host)
	if err != nil {
		return nil, fmt.Errorf("(%v) is not a valid url: %w", sess.host, err)
	}

	filepath = path.Clean(filepath)

	uri = uri.JoinPath("/api/resources/", filepath)

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http.GET( %v/%v ): %w", sess.host, filepath, err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not http.Do request (GET %v): %w", uri.String(), err)
	}
	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			if err != nil {
				err = fmt.Errorf("could not close response body (%w) after another error: (%w)", err2, err)
			}
			err = err2
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1e6))
	if err != nil {
		return nil, fmt.Errorf("could not read resp body error: %w", err)
	}

	res := Resource{}

	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("could not unmarshal request body json: %w", err)
	}

	// TODO: return a resource
	return &res, nil
}

// upload to directory/filename with the data r.
// internally uses a subset set of TUS (https://tus.io/protocols/resumable-upload)
//func (sess *filebrowserSession) upload(ctx context.Context, directory string, filename string, r io.Reader) (err error) {
//	return nil
//}

func loginToFilebrowser(host, user, pass string) (sess *filebrowserSession, err error) {
	slog.Debug("logging into filebrowser", "host", host, "user", user)

	sess = &filebrowserSession{host: host}
	jsonData, err := json.Marshal(struct {
		Username string
		Password string
	}{
		Username: user,
		Password: pass,
	})
	if err != nil {
		return nil, fmt.Errorf("could not marshal a request for filebrowser for login: %w", err)
	}

	httpClient := http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := httpClient.Post(sess.host+"/api/login", "", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not POST login token from %v/api/login: %w", sess.host, err)
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			err = errors.Join(err, fmt.Errorf("could not close body of request: %w", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 http status code logging in: %v", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1e6))
	if err != nil {
		return nil, fmt.Errorf("could not read body from login request: %w", err)
	}

	sess.token = string(body)

	return sess, nil
}
