package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"statusphere-client/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	reconnectDelay = 3 * time.Second
	pingInterval   = 20 * time.Second
)

type WSTransport struct {
	url      string
	token    string
	deviceID string

	mu   sync.Mutex
	conn *websocket.Conn
}

func NewWS(serverURL, token string) *WSTransport {
	url := strings.TrimRight(serverURL, "/")
	url = strings.Replace(url, "https://", "wss://", 1)
	url = strings.Replace(url, "http://", "ws://", 1)
	url += "/ws"

	return &WSTransport{
		url:      url,
		token:    token,
		deviceID: ID(),
	}
}

func (t *WSTransport) Connect(ctx context.Context) error {
	return t.connect(ctx)
}

func (t *WSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		err := t.conn.Close(websocket.StatusNormalClosure, "bye")
		t.conn = nil
		return err
	}
	return nil
}

func (t *WSTransport) Send(snap models.Snapshot) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	return conn.Write(context.Background(), websocket.MessageText, data)
}

func (t *WSTransport) Listen(ctx context.Context, onEvent func(data []byte)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		t.mu.Lock()
		conn := t.conn
		t.mu.Unlock()

		if conn == nil {
			t.reconnect(ctx)
			continue
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("ws read error: %v", err)
			t.mu.Lock()
			t.conn = nil
			t.mu.Unlock()
			t.reconnect(ctx)
			continue
		}

		onEvent(data)
	}
}

func (t *WSTransport) connect(ctx context.Context) error {
	headers := http.Header{
		"X-Room-Token": {t.token},
		"X-Device-Id":  {t.deviceID},
	}

	conn, _, err := websocket.Dial(ctx, t.url, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.conn = conn
	t.mu.Unlock()

	log.Println("ws connected")
	return nil
}

func (t *WSTransport) reconnect(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		log.Printf("reconnecting in %s...", reconnectDelay)

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}

		if err := t.connect(ctx); err != nil {
			log.Printf("reconnect failed: %v", err)
			continue
		}
		return
	}
}
