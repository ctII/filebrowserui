package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type filebrowserSession struct {
	Host       string
	authCookie string
}

func (sess *filebrowserSession) Login(user, pass string) (err error) {
	jsonData, err := json.Marshal(struct{ Username, Password string }{Username: user, Password: pass})

	resp, err := http.NewRequest("GET", sess.Host+"/api/login", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("could not GET login token from %v/api/login: %w", sess.Host, err)
	}
	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			err = errors.Join(err, fmt.Errorf("could not close body of request: %w", err))
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1e6))
	if err != nil {
		return fmt.Errorf("could not read body from login request: %w", err)
	}

	sess.authCookie = string(body)

	return nil
}

func handleLogin(w fyne.Window) (user, pass string, sess *filebrowserSession, err error) {
	done := make(chan struct{}, 1)

	hEntry := widget.NewEntry()
	uEntry := widget.NewEntry()
	pEntry := widget.NewEntry()
	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:   "Host",
				Widget: hEntry,
			},
			{
				Text:   "Username",
				Widget: uEntry,
			},
			{
				Text:   "Password",
				Widget: pEntry,
			},
		},
		OnSubmit: func() {
			user = uEntry.Text
			pass = pEntry.Text

			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	vbox := (container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer()))
	w.SetContent(container.NewGridWithColumns(3, layout.NewSpacer(), vbox, layout.NewSpacer()))

	<-done
	return user, pass, nil, nil
}

func logic(w fyne.Window) (err error) {
	user, pass, _, err := handleLogin(w)
	if err != nil {
		return err
	}

	w.SetContent(widget.NewLabel(fmt.Sprintf("user: %v pass: %v", user, pass)))
	return nil
}

func run() (err error) {
	a := app.New()
	w := a.NewWindow("FilebrowserUI")
	w.Resize(fyne.NewSize(640, 360))

	go logic(w)

	w.ShowAndRun()

	return nil
}

func main() {
	if err := run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
