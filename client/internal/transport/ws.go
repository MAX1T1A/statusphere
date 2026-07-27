package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"statusphere-client/internal/presence"

	"github.com/coder/websocket"
)

const (
	reconnectDelay = 3 * time.Second
	pingInterval   = 20 * time.Second
	readTimeout    = 45 * time.Second
	writeTimeout   = 5 * time.Second
)

type WSTransport struct {
	url      string
	token    string
	deviceID string

	mu         sync.Mutex
	deviceName string
	conn       *websocket.Conn
	cancel     context.CancelFunc
}

func NewWS(serverURL, token, deviceID, roomID string) *WSTransport {
	wsURL := strings.TrimRight(serverURL, "/")
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL += "/ws?room=" + url.QueryEscape(roomID)

	return &WSTransport{
		url:        wsURL,
		token:      token,
		deviceID:   deviceID,
		deviceName: loadName(),
	}
}

func (t *WSTransport) Connect(ctx context.Context) error {
	return t.connect(ctx)
}

func (t *WSTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
	}
	if t.conn != nil {
		err := t.conn.Close(websocket.StatusNormalClosure, "bye")
		t.conn = nil
		return err
	}
	return nil
}

func (t *WSTransport) SetDeviceName(name string) {
	t.mu.Lock()
	t.deviceName = name
	t.mu.Unlock()
	saveName(name)
}

func (t *WSTransport) DeviceName() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.deviceName
}

func (t *WSTransport) drop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	if t.conn != nil {
		t.conn.CloseNow()
		t.conn = nil
	}
}

func (t *WSTransport) Send(snap presence.Snapshot) error {
	t.mu.Lock()
	conn := t.conn
	name := t.deviceName
	t.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	out := snap.Clone()
	out[presence.KeyDeviceID] = t.deviceID
	out[presence.KeyDeviceName] = name

	data, err := json.Marshal(map[string]any(out))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("send error, dropping connection: %v", err)
		t.drop()
		return err
	}
	return nil
}

func (t *WSTransport) SendMessage(to, text string) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(map[string]string{"type": "msg", "to": to, "text": text})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("send message error, dropping connection: %v", err)
		t.drop()
		return err
	}
	return nil
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

		readCtx, readCancel := context.WithTimeout(ctx, readTimeout)
		_, data, err := conn.Read(readCtx)
		readCancel()

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("ws read error: %v", err)
			t.drop()
			t.reconnect(ctx)
			continue
		}

		onEvent(data)
	}
}

func (t *WSTransport) connect(ctx context.Context) error {
	headers := http.Header{
		"X-Room-Token": {t.token},
	}

	conn, _, err := websocket.Dial(ctx, t.url, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return err
	}

	pingCtx, pingCancel := context.WithCancel(ctx)

	t.mu.Lock()
	t.conn = conn
	t.cancel = pingCancel
	t.mu.Unlock()

	go t.pinger(pingCtx, conn)

	log.Println("ws connected")
	return nil
}

func (t *WSTransport) pinger(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Printf("ping failed: %v", err)
				t.drop()
				return
			}
		}
	}
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
