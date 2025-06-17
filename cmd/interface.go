package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func browse(w fyne.Window, sess *filebrowserSession) {
	cache := NewNodeCache(sess)

	fileInfo := widget.NewLabel("")

	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			res, err := cache.Info(context.Background(), id)
			if err != nil {
				handleError(w, err, func() {})
				return []string{}
			}

			if !res.IsDir {
				slog.Error("Logic error, we have a file in the path instead of a directory for a branch")
			}

			leaves := make([]string, 0, len(res.Items))

			for i := range res.Items {
				leaves = append(leaves, res.Items[i].Path)
			}

			return leaves
		},
		func(id widget.TreeNodeID) bool {
			res, err := cache.Info(context.Background(), id)
			if err != nil {
				handleError(w, err, func() {})
				return false
			}

			return res.IsDir
		},
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabel("Branch template")
			}

			return NewNodeWidget()
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			text := path.Base(strings.ReplaceAll(id, "\n", "\\n"))

			if branch {
				o.(*widget.Label).SetText(text)
				return
			}

			o.(*nodeWidget).SetLabel(text)
			o.(*nodeWidget).SetButtonFunc(func() {
				sum, err := sess.SHA256(context.Background(), id)
				if err != nil {
					ShowDismissablePopup(w, err.Error())
					return
				}

				ShowDismissablePopup(w, sum)
			})
		},
	)

	tree.OnSelected = func(id widget.TreeNodeID) {
		res, err := cache.Info(context.Background(), id)
		if err != nil {
			handleError(w, err, func() {})
			return
		}

		if res.IsDir {
			fileInfo.SetText(fmt.Sprintf("Name: %v\nModified: %v",
				strings.ReplaceAll(res.Name, "\n", "\\n"),
				res.Modified,
			))
		} else {
			// TODO: dynamically change size unit
			fileInfo.SetText(fmt.Sprintf("Name: %v\nModified: %v\nSize: %.2f MB",
				strings.ReplaceAll(res.Name, "\n", "\\n"),
				res.Modified,
				float64(res.Size)/float64(1024)/float64(1024)),
			)
		}
	}

	priorityLayout := container.New(&priorityVLayout{}, tree, fileInfo)

	border := container.NewBorder(widget.NewButton("Upload", func() {}), nil, nil, nil, priorityLayout)

	fyne.DoAndWait(func() { w.SetContent(border) })
}

// handleError on window with err and call f after user hits "Okay" button.
func handleError(w fyne.Window, err error, okay func()) {
	once := sync.Once{}

	w.SetContent(
		container.NewVBox(
			widget.NewLabel(err.Error()),
			widget.NewButton("Copy Error", func() {
				w.Clipboard().SetContent(err.Error())
			}),
			widget.NewButton("Okay", func() {
				once.Do(okay)
			}),
		),
	)
}

func login(w fyne.Window) (sess *filebrowserSession, err error) {
	done := make(chan struct{})

	hEntry := widget.NewEntry()
	hEntry.Text = config.Host
	uEntry := widget.NewEntry()
	uEntry.Text = config.User
	pEntry := widget.NewPasswordEntry()
	pEntry.Text = config.Pass

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
		OnSubmit: sync.OnceFunc(func() { close(done) }),
	}

	vbox := (container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer()))
	fyne.DoAndWait(func() { w.SetContent(container.NewGridWithColumns(3, layout.NewSpacer(), vbox, layout.NewSpacer())) })

	<-done

	fyne.DoAndWait(func() { w.SetContent(container.NewCenter(widget.NewLabel("Logging in"))) })

	if hEntry.Text != config.Host || uEntry.Text != config.User || pEntry.Text != config.Pass {
		config.Host = hEntry.Text
		config.User = uEntry.Text
		config.Pass = pEntry.Text
		config.changed = true
	}

	sess, err = loginToFilebrowser(config.Host, config.User, config.Pass)
	if err != nil {
		return nil, fmt.Errorf("could not login to (%v): %w", config.Host, err)
	}

	return sess, nil
}
