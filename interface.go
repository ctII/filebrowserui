package main

import (
	"context"
	"log/slog"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/simplylib/genericsync"
)

type Node struct {
	*Resource
}

// NodeCache is a non-optimal cache that does cache things
// but sometimes multiple requests for the same object will result in
// multiple filebrowser request, but eventually will keep them in cache
type NodeCache struct {
	sess *filebrowserSession

	// Cache of map[ID string]Node
	cache genericsync.Map[string, Node]
}

func NewNodeCache(sess *filebrowserSession) *NodeCache {
	return &NodeCache{sess: sess}
}

func (nc *NodeCache) Info(ctx context.Context, path string) (*Resource, error) {
	node, ok := nc.cache.Load(path)
	if ok {
		return node.Resource, nil
	}

	res, err := nc.sess.Info(ctx, path)
	if err != nil {
		return nil, err
	}

	nc.cache.Store(path, Node{Resource: res})
	slog.Debug("caching resource info", "path", path)

	return res, nil
}

func browse(w fyne.Window, sess *filebrowserSession) {
	cache := NewNodeCache(sess)

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
			o.(*widget.Label).SetText(path.Base(id))
		},
	)
	w.SetContent(tree)
}
