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
			return widget.NewLabel("Leaf template")
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(path.Base(strings.ReplaceAll(id, "\n", "\\n")))
		},
	)

	tree.OnSelected = func(id widget.TreeNodeID) {
		res, err := cache.Info(context.Background(), id)
		if err != nil {
			handleError(w, err, func() {})
			return
		}

		fileInfo.SetText(fmt.Sprintf("Name: %v\nModified: %v\nSize: %.2f MB",
			strings.ReplaceAll(res.Name, "\n", "\\n"),
			res.Modified,
			float64(res.Size)/float64(1024)/float64(1024)),
		)
	}

	split := container.NewVSplit(
		tree,
		fileInfo,
	)

	w.SetContent(split)
}
