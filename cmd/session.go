package cmd

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
	"strconv"
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
		return nil, fmt.Errorf("could not create a http.GET(%v) : %w", uri.String(), err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not http.Do request (GET %v): %w", uri.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 http status while getting info: %v", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1e6))
	if err != nil {
		return nil, fmt.Errorf("could not read resp body: %w", err)
	}

	res := Resource{}

	t := time.Now()
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("could not unmarshal request body json: %w", err)
	}
	slog.Debug("time it took to unmarshal json", "time", time.Since(t).String())

	return &res, nil
}

func (sess *filebrowserSession) SHA256(ctx context.Context, filepath string) (string, error) {
	slog.Debug("Getting the SHA256 hash from filebrowser", "path", filepath)

	uri, err := url.Parse(sess.host)
	if err != nil {
		return "", fmt.Errorf("(%v) is not a valid url: %w", sess.host, err)
	}

	filepath = path.Clean(filepath)

	uri = uri.JoinPath("/api/resources/", filepath)

	query := uri.Query()
	query.Add("checksum", "sha256")
	uri.RawQuery = query.Encode()

	httpClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
	if err != nil {
		return "", fmt.Errorf("could not create a http.GET( %v ): %w", uri.String(), err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not http.GET sha256 sum from filebrowser (%v): %w", sess.host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http server returned non-200 status code: %v", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1e5))
	if err != nil {
		return "", fmt.Errorf("could not read resp body: %w", err)
	}

	respJson := struct {
		Checksums struct {
			Sha256 string `json:"sha256"`
		}
	}{}

	if err := json.Unmarshal(body, &respJson); err != nil {
		return "", fmt.Errorf("could not decode json from filebrowser (%v): %w", sess.host, err)
	}

	return respJson.Checksums.Sha256, nil
}

/*
Documentation of upload protocol specifically for filebrowser. We will follow how the webui works, with some help from https://tus.io/protocols/resumable-upload

WebUI gets information from parent as first thing, but this is unlikely to be needed for us as we don't need to refresh all the information about the page, if we do this is done from the browse gui.

Glossary:
	{filepath} : full path to the file from the root / on the filebrowser server
					ex: {filepath} = /root/{dir}/{file}
					ex  {filepath} = /Documents/Notes.txt = /{dir}/{file}

For uploading a new file that doens't exist yet:
1. POST filebrowser.example.com/api/tus/{filepath}:
		This should return a HTTP 201 if we successfully created the file. Note that without a url parameter "override=true", the file is not actually modified, instead the server just returns a HTTP 201 the same as if this *was* creating the file. The file retains the original size and content.
		If we "?override=true" the file is set to a empty zero length file.

2. HEAD filebrowser.example.com/api/tus/{filepath}:
	Headers:
		"Tus-Resumable: 1.0.0"

	This should return a http header "upload-offset=0". The response header specifies "upload-length=-1" which is invalid TUS. https://tus.io/protocols/resumable-upload#upload-length
	If upload-offset is anything other than 0, the file existed already and needs its upload to be resumed starting the byte after that offset

3. PATCH filebrowser.example.com/api/tus/{filepath}:
	Headers:
		"Content-Type application/offset+octet-stream"
		"Content-Length: {Content Bytes Length}"
		"Tus-Resumable: 1.0.0"
		"Upload-Offset: {offset}"

	Body of the request should contain the bytes from {Content Bytes Length}.

	Response should be a header "upload-offset" that is equal to the number of bytes the server successfully received, compare this to our length.
*/

// ErrResumable instructs the caller that the error that was returned is resumable, and likely should be called until it returns no error.
// This happens because of transient errors like io.ErrUnexpectedEOF when the http request unexpectedly is closed before finishing.
type ErrResumable struct {
	err error
}

func (e ErrResumable) Unwrap() error {
	return e.err
}

func (e ErrResumable) Error() string {
	return fmt.Sprintf("filebrowserui-session: resumable error occurred: %v", e.err.Error())
}

func (sess *filebrowserSession) createTUSFile(ctx context.Context, filepath string, override bool) error {
	slog.Debug("creating tus file on filebrowser", "path", filepath)

	uri, err := url.Parse(sess.host)
	if err != nil {
		return fmt.Errorf("(%v) is not a valid url: %w", sess.host, err)
	}

	uri = uri.JoinPath("/api/tus/", filepath)

	query := uri.Query()
	query.Add("override", strconv.FormatBool(override))
	uri.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", uri.String(), nil)
	if err != nil {
		return fmt.Errorf("could not create a http.POST (%v): %w", uri.String(), err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	resp, err := (&http.Client{Timeout: time.Second * 5}).Do(req)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return ErrResumable{err: err}
		}

		return fmt.Errorf("failed to http.POST (%v): %w", uri.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("non-201 http status code while doing http HEAD request (%v): %v", uri.String(), resp.Status)
	}

	return nil
}

var (
	ErrTUSHeadFileUploadOffsetMissing = errors.New("filebrowserui-session: tus api endpoint response is missing the upload-offset header")
	ErrTUSHeadFileUploadOffsetInvalid = errors.New("filebrowserui-session: tus api endpoint response upload-offset < 0, this is invalid TUS")
)

func (sess *filebrowserSession) headTUSFile(ctx context.Context, filepath string) (offset int64, err error) {
	slog.Debug("getting tus file head", "path", filepath)

	uri, err := url.Parse(sess.host)
	if err != nil {
		return 0, fmt.Errorf("(%v) is not a valid url: %w", sess.host, err)
	}

	uri = uri.JoinPath("/api/tus/", filepath)

	req, err := http.NewRequestWithContext(ctx, "HEAD", uri.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("could not http.HEAD (%v): %w", uri.String(), err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	// Show that we understand the tus protocol
	req.Header.Add("Tus-Resumable", "1.0.0")

	resp, err := (&http.Client{Timeout: time.Second * 5}).Do(req)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, ErrResumable{err: err}
		}

		return 0, fmt.Errorf("failed to http.HEAD (%v): %w", uri.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-200 http status code while doing http HEAD request (%v): %v", uri.String(), resp.Status)
	}

	if resp.Header.Get("upload-offset") == "" {
		return 0, ErrTUSHeadFileUploadOffsetMissing
	}

	parsedInt, err := strconv.ParseInt(resp.Header.Get("upload-offset"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse http header upload-offset as an int64: %w", err)
	}

	if parsedInt < 0 {
		return 0, ErrTUSHeadFileUploadOffsetInvalid
	}

	return parsedInt, nil
}

func (sess *filebrowserSession) uploadTUSReader(ctx context.Context, filepath string, offset, readerLength int64, r io.Reader) (err error) {
	slog.Debug("uploading tus file to filebrowser", "path", filepath)

	uri, err := url.Parse(sess.host)
	if err != nil {
		return fmt.Errorf("(%v) is not a valid url: %w", sess.host, err)
	}

	uri = uri.JoinPath("/api/tus/", filepath)

	req, err := http.NewRequestWithContext(ctx, "PATCH", uri.String(), nil)
	if err != nil {
		return fmt.Errorf("could not http.PATCH (%v): %w", uri.String(), err)
	}

	req.Header.Add("X-Auth", sess.token)
	req.AddCookie(&http.Cookie{Name: "auth", Value: sess.token})

	// Show that we understand the tus protocol
	req.Header.Add("Tus-Resumable", "1.0.0")
	req.Header.Add("Content-Type", "application/offset+octet-stream")
	req.Header.Add("Upload-Offset", strconv.FormatInt(offset, 10))
	req.Header.Add("Content-Length", strconv.FormatInt(readerLength, 10))

	req.Body = io.NopCloser(io.LimitReader(r, readerLength))

	// TODO: detect a stalled upload
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return ErrResumable{err: err}
		}

		return fmt.Errorf("failed to http.PATCH (%v): %w", uri.String(), err)
	}
	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			err = errors.Join(err, fmt.Errorf("could not close body of request: %w", err))
		}
	}()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("non-204 http status code while doing http.PATCH request (%v): %v", uri.String(), resp.Status)
	}

	return nil
}

// upload to directory/filename with the data r.
// internally uses a subset set of TUS that is documented above. it automatically resumes based on what it receives from the server.
func (sess *filebrowserSession) uploadReader(ctx context.Context, dir string, filename string, r io.ReadSeeker, readerLength int64, override bool) error {
	slog.Debug("uploading content to filebrowser", "path", dir)

	filename = path.Clean(filename)
	dir = path.Clean(dir)
	filepath := path.Join(dir, filename)

	// TODO: we actually implicitly overide the file when uploading again over the same path, even when override is false

	// Step one: potentially create the file
	err := sess.createTUSFile(ctx, filepath, override)
	if err != nil {
		return err
	}

	// Step two: check if the file needs to be resumed, if so get the offset to start at
	offset, err := sess.headTUSFile(ctx, filepath)
	if err != nil {
		return err
	}

	if offset == readerLength { // file finished uploading already
		return nil
	}

	if offset > readerLength {
		return errors.New("filebrowserui-session: reader length is smaller than offset, meaning reader is unlikely to be the same file")
	}

	slog.Info("", "offset", offset, "readerlen", readerLength)

	_, err = r.Seek(offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("could not seek to offset returned by TUS http.HEAD: %w", err)
	}

	// Step three: actually upload the bytes, starting at the byte after the offset
	err = sess.uploadTUSReader(ctx, filepath, offset, readerLength, r)
	if err != nil {
		return err
	}

	return nil
}

// TODO: add ability to create a directory
// TODO: add ability to download files

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
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrResumable{err: err}
		}

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
