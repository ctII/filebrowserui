package main

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
		)
	}

	split := container.NewVSplit(tree, fileInfo)
	split.SetOffset(1.0)

	border := container.NewBorder(widget.NewButton("Upload", func() {}), nil, nil, nil, split)

	w.SetContent(border)
}
