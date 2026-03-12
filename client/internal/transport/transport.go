package transport

import (
	"context"
	"statusphere-client/internal/models"
)

type Transport interface {
	Connect(ctx context.Context) error
	Close() error
	Send(snap models.Snapshot) error
	Listen(ctx context.Context, onEvent func(data []byte)) error
}
