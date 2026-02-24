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
	Stats() nntppool.ClientStats
	AddProvider(p nntppool.Provider) error
	RemoveProvider(name string) error
	Close() error
}
