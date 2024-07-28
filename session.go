package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type fileInfo struct {
	Path  string
	Name  string
	IsDir bool
	Size  int
}

type filebrowserSession struct {
	Host      string
	authToken string
}

func (sess *filebrowserSession) list(ctx context.Context, directory string) (files []fileInfo, err error) {
	uri, err := url.Parse(sess.Host)
	if err != nil {
		return nil, fmt.Errorf("(%v) is not a valid url: %w", sess.Host, err)
	}

	uri = uri.JoinPath("/api/resources", directory)

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a http.GET( %v/%v ): %w", sess.Host, directory, err)
	}

	req.Header.Add("X-Auth", sess.authToken)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.authToken})

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

	listing := struct {
		Items []struct {
			Path  string `json:"path"`
			Name  string `json:"name"`
			IsDir bool   `json:"isDir"`
			Size  int    `json:"size"`
		}
	}{}
	{
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1e6))
		if err != nil {
			return nil, fmt.Errorf("could not read resp body error: %w", err)
		}

		if err := json.Unmarshal(body, &listing); err != nil {
			return nil, fmt.Errorf("could not unmarshal request body json: %w", err)
		}
	}

	for i := range listing.Items {
		files = append(files, fileInfo(listing.Items[i]))
	}

	return files, nil
}

// upload to directory/filename with the data r.
// internally uses a subset set of TUS (https://tus.io/protocols/resumable-upload)
func (sess *filebrowserSession) upload(ctx context.Context, directory string, filename string, r io.Reader) (err error) {
	return nil
}

func loginToFilebrowser(host, user, pass string) (sess *filebrowserSession, err error) {
	sess = &filebrowserSession{Host: host}
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
	resp, err := httpClient.Post(sess.Host+"/api/login", "", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("could not POST login token from %v/api/login: %w", sess.Host, err)
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

	sess.authToken = string(body)

	return sess, nil
}
