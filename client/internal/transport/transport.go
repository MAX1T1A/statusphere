package transport

import (
	"context"

	"statusphere-client/internal/presence"
)

type Transport interface {
	Connect(ctx context.Context) error
	Close() error
	Send(snap presence.Snapshot) error
	Listen(ctx context.Context, onEvent func(data []byte)) error
}
