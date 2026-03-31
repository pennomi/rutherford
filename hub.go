package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[chan []byte]struct{}),
	}
}

func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 256)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

type watchEvent struct {
	Type   string          `json:"type"`
	Object json.RawMessage `json:"object"`
}

func HandleWebSocket(auth Authenticator, hub *Hub, watcher *Watcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			http.Error(w, "websocket accept failed", http.StatusBadRequest)
			return
		}

		_, tokenBytes, err := conn.Read(r.Context())
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "failed to read auth token")
			return
		}
		err = auth.ValidateToken(string(tokenBytes))
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "invalid auth token")
			return
		}

		ctx := conn.CloseRead(r.Context())

		snapshot, err := watcher.Snapshot(r.Context())
		if err != nil {
			conn.Close(websocket.StatusInternalError, "failed to fetch snapshot")
			return
		}

		for _, msg := range snapshot {
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				conn.Close(websocket.StatusInternalError, "write failed")
				return
			}
		}

		sub := hub.Subscribe()
		defer hub.Unsubscribe(sub)

		for {
			select {
			case <-ctx.Done():
				conn.Close(websocket.StatusNormalClosure, "")
				return
			case msg, ok := <-sub:
				if !ok {
					return
				}
				writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				err := conn.Write(writeCtx, websocket.MessageText, msg)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}
}
