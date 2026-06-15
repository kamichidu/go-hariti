package hariti

import (
	"context"

	"github.com/kamichidu/go-hariti/internal/graph"
)

type InstallOptions struct {
	Sync   SyncOptions
	Deploy DeployOptions
}

func (h *Hariti) Install(ctx context.Context, g *graph.Graph, opts InstallOptions) error {
	_, err := h.Sync(ctx, g, opts.Sync)
	if err != nil {
		return err
	}
	_, err = h.Deploy(ctx, g, opts.Deploy)
	return err
}
