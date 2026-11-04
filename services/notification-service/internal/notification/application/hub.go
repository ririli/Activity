package application

import (
	"activity/pkg/platform"
	"activity/services/notification-service/internal/notification/domain"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Connection struct {
	Session  string
	Messages chan domain.Message
}
type Hub struct {
	mu      sync.Mutex
	clients map[*Connection]bool
	ready   atomic.Bool
	ctx     context.Context
}

func New(ctx context.Context) *Hub { return &Hub{clients: map[*Connection]bool{}, ctx: ctx} }
func (h *Hub) Ready() bool         { return h.ready.Load() && h.ctx.Err() == nil }
func (h *Hub) SetReady(v bool)     { h.ready.Store(v) }
func (h *Hub) Subscribe(session string) (*Connection, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.Ready() {
		return nil, platform.Fail(503, "SUBSCRIPTION_UNAVAILABLE")
	}
	if len(h.clients) >= 1000 {
		return nil, platform.Fail(503, "CONNECTION_CAPACITY")
	}
	n := 0
	for c := range h.clients {
		if c.Session == session {
			n++
		}
	}
	if n >= 5 {
		return nil, platform.Fail(429, "SESSION_CONNECTION_LIMIT")
	}
	c := &Connection{session, make(chan domain.Message, 32)}
	h.clients[c] = true
	return c, nil
}
func (h *Hub) Unsubscribe(c *Connection) { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock() }
func (h *Hub) Receive(raw []byte) {
	m, e := domain.Parse(raw, time.Now())
	if e != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.Messages <- m:
		default:
		}
	}
}
