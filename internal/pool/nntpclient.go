package pool

import (
	"context"
	"io"

	"github.com/javi11/nntppool/v4"
	"github.com/mnightingale/rapidyenc"
)

// NNTPClient defines the interface for NNTP connection pool operations.
// This wraps *nntppool.Client to enable testing with mocks.
type NNTPClient interface {
	PostYenc(ctx context.Context, headers nntppool.PostHeaders, body io.Reader, meta rapidyenc.Meta) (*nntppool.PostResult, error)
	Stat(ctx context.Context, messageID string) (*nntppool.StatResult, error)
	StatMany(ctx context.Context, messageIDs []string, opts nntppool.StatManyOptions) <-chan nntppool.StatManyResult
	Stats() nntppool.ClientStats
	AddProvider(p nntppool.Provider) error
	RemoveProvider(name string) error
	Close() error
}

// DefaultStatBatchSize is the fallback number of message-IDs per StatMany call
// when no batch size is configured.
const DefaultStatBatchSize = 100

// StatMissing STATs ids in chunks of batchSize (<=0 -> DefaultStatBatchSize)
// and returns the message-IDs NOT confirmed present. Any per-ID error counts
// as missing (preserves the "error => repost" semantics of the previous
// one-by-one checks). Returns ctx.Err() if the sweep was cancelled.
func StatMissing(ctx context.Context, c NNTPClient, ids []string, batchSize int) (map[string]struct{}, error) {
	missing := make(map[string]struct{})
	if len(ids) == 0 {
		return missing, nil
	}
	if batchSize <= 0 {
		batchSize = DefaultStatBatchSize
	}

	for start := 0; start < len(ids); start += batchSize {
		chunk := ids[start:min(start+batchSize, len(ids))]

		reported := make(map[string]struct{}, len(chunk))
		for res := range c.StatMany(ctx, chunk, nntppool.StatManyOptions{}) {
			reported[res.MessageID] = struct{}{}
			if res.Err != nil {
				missing[res.MessageID] = struct{}{}
			}
		}
		if err := ctx.Err(); err != nil {
			return missing, err
		}
		// An interrupted sweep may drop results; ids that never reported must
		// count as missing rather than silently pass as present.
		if len(reported) != len(chunk) {
			for _, id := range chunk {
				if _, ok := reported[id]; !ok {
					missing[id] = struct{}{}
				}
			}
		}
	}

	return missing, nil
}
