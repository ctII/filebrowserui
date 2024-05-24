package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	userFlag = flag.String("user", "test", "username to use")
	passFlag = flag.String("pass", "test", "password to use")
	hostFlag = flag.String("host", "", "full url to the filebrowser instance")
)

// handleError on window with err and call f after user hits "Okay" button.
func handleError(w fyne.Window, err error, f func()) {
	once := sync.Once{}

	w.SetContent(
		container.NewVBox(
			widget.NewLabel(err.Error()),
			widget.NewButton("Copy Error", func() {
				w.Clipboard().SetContent(err.Error())
			}),
			widget.NewButton("Okay", func() {
				once.Do(f)
			}),
		),
	)
}

func login(w fyne.Window) (sess *filebrowserSession, err error) {
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
			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	vbox := (container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer()))
	w.SetContent(container.NewGridWithColumns(3, layout.NewSpacer(), vbox, layout.NewSpacer()))

	<-done

	w.SetContent(container.NewCenter(widget.NewLabel("Logging in")))

	sess, err = loginToFilebrowser(
		cmp.Or(hEntry.Text, *hostFlag),
		cmp.Or(uEntry.Text, *userFlag),
		cmp.Or(pEntry.Text, *passFlag),
	)
	if err != nil {
		return nil, fmt.Errorf("could not loginToFilebrowser: %w", err)
	}

	return sess, nil
}

func browse(w fyne.Window, sess *filebrowserSession) {
	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			switch id {
			case "":
				return []widget.TreeNodeID{"a", "b", "c"}
			case "a":
				return []widget.TreeNodeID{"a1", "a2"}
			}
			return []string{}
		},
		func(id widget.TreeNodeID) bool {
			return id == "" || id == "a"
		},
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabel("Branch template")
			}
			return widget.NewLabel("Leaf template")
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			text := id
			if branch {
				text += " (branch)"
			}
			o.(*widget.Label).SetText(text)
		},
	)
	w.SetContent(tree)
}

func upload(w fyne.Window, sess *filebrowserSession) {

}

func logic(w fyne.Window) {
	sess, err := login(w)
	if err != nil {
		handleError(w, err, func() { go logic(w) })
		return
	}

	browse(w, sess)
}

func run() (err error) {
	flag.Parse()

	a := app.New()
	w := a.NewWindow("FilebrowserUI")
	w.Resize(fyne.NewSize(700, 400))

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
