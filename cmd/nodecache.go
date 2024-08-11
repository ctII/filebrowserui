package cmd

import (
	"context"
	"log/slog"

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

	slog.Debug("cache miss, calling filebrowser", "path", path)
	res, err := nc.sess.Info(ctx, path)
	if err != nil {
		return nil, err
	}

	nc.cache.Store(path, Node{Resource: res})
	slog.Debug("caching resource info", "path", path)

	return res, nil
}
