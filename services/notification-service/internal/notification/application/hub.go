package application

import (
	"activity/services/notification-service/internal/notification/domain"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrUnavailable  = errors.New("SUBSCRIPTION_UNAVAILABLE")
	ErrCapacity     = errors.New("CONNECTION_CAPACITY")
	ErrSessionLimit = errors.New("SESSION_CONNECTION_LIMIT")
)

type Config struct {
	Shards, MaxConnections, SessionLimit, ShardQueue, ConnectionQueue int
}

func DefaultConfig() Config {
	return Config{Shards: 8, MaxConnections: 10000, SessionLimit: 5, ShardQueue: 1024, ConnectionQueue: 32}
}

func (c Config) Validate() error {
	for _, p := range []struct {
		name  string
		value int
	}{
		{"shards", c.Shards}, {"max connections", c.MaxConnections},
		{"session limit", c.SessionLimit}, {"shard queue", c.ShardQueue},
		{"connection queue", c.ConnectionQueue},
	} {
		if p.value <= 0 {
			return fmt.Errorf("notification %s must be positive", p.name)
		}
	}
	return nil
}

// Frame is copied by value; Data is an immutable string shared by every recipient.
// The transport adapter supplies the encoding, so the Hub knows no SSE protocol.
type Frame struct {
	Data    string
	Expires time.Time
}
type Encoder func(domain.Message) (string, error)

type Connection struct {
	Messages <-chan Frame
	owner    *Hub
	shard    int
	session  string
	messages chan Frame
}
type shard struct {
	mu      sync.Mutex
	clients map[*Connection]struct{}
	queue   chan Frame
}

type Hub struct {
	config    Config
	encode    Encoder
	shards    []*shard
	next      atomic.Uint64
	ready     atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	workers   sync.WaitGroup
	closeOnce sync.Once
	// Receive is serialized to preserve ingress order even with concurrent callers.
	receiveMu sync.Mutex
	// Registration always takes quotaMu before a shard lock. Workers take only
	// their own shard lock and never acquire quotaMu.
	quotaMu           sync.Mutex
	connections       int
	sessionCounts     map[string]int
	invalid           atomic.Uint64
	shardDropped      atomic.Uint64
	connectionDropped atomic.Uint64
	expired           atomic.Uint64
}

func New(parent context.Context, config Config, encode Encoder) (*Hub, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if encode == nil {
		return nil, errors.New("notification encoder is required")
	}
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{config: config, encode: encode, ctx: ctx, cancel: cancel, sessionCounts: make(map[string]int)}
	for i := 0; i < config.Shards; i++ {
		s := &shard{clients: make(map[*Connection]struct{}), queue: make(chan Frame, config.ShardQueue)}
		h.shards = append(h.shards, s)
		h.workers.Add(1)
		go h.run(s)
	}
	// Also clean up after parent cancellation without an explicit Close.
	// This supervisor is not included in the worker wait group.
	go func() { <-ctx.Done(); h.Close() }()
	return h, nil
}

func (h *Hub) Ready() bool           { return h.ready.Load() && h.ctx.Err() == nil }
func (h *Hub) SetReady(v bool)       { h.ready.Store(v) }
func (h *Hub) Done() <-chan struct{} { return h.ctx.Done() }

func (h *Hub) Subscribe(session string) (*Connection, error) {
	h.quotaMu.Lock()
	defer h.quotaMu.Unlock()
	if !h.Ready() {
		return nil, ErrUnavailable
	}
	if h.connections >= h.config.MaxConnections {
		return nil, ErrCapacity
	}
	if h.sessionCounts[session] >= h.config.SessionLimit {
		return nil, ErrSessionLimit
	}
	id := int((h.next.Add(1) - 1) % uint64(len(h.shards)))
	s := h.shards[id]
	s.mu.Lock()
	defer s.mu.Unlock()
	if !h.Ready() {
		return nil, ErrUnavailable
	}
	q := make(chan Frame, h.config.ConnectionQueue)
	c := &Connection{Messages: q, owner: h, shard: id, session: session, messages: q}
	s.clients[c] = struct{}{}
	h.connections++
	h.sessionCounts[session]++
	return c, nil
}

func (h *Hub) Unsubscribe(c *Connection) {
	if c == nil || c.owner != h {
		return
	}
	h.quotaMu.Lock()
	defer h.quotaMu.Unlock()
	s := h.shards[c.shard]
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[c]; !exists {
		return
	}
	delete(s.clients, c)
	h.connections--
	h.sessionCounts[c.session]--
	if h.sessionCounts[c.session] == 0 {
		delete(h.sessionCounts, c.session)
	}
	// Never close a channel while a broadcaster could hold its connection.
	drain(c.messages)
}

func (h *Hub) Receive(raw []byte) {
	h.receiveMu.Lock()
	defer h.receiveMu.Unlock()
	if !h.Ready() {
		return
	}
	m, err := domain.Parse(raw, time.Now())
	if err != nil {
		h.invalid.Add(1)
		return
	}
	data, err := h.encode(m)
	if err != nil {
		h.invalid.Add(1)
		return
	}
	f := Frame{Data: data, Expires: m.Expires}
	for _, s := range h.shards {
		if !h.Ready() {
			return
		}
		select {
		case s.queue <- f:
		default:
			h.shardDropped.Add(1)
		}
	}
}

func (h *Hub) run(s *shard) {
	defer h.workers.Done()
	for {
		select {
		case <-h.ctx.Done():
			return
		case f := <-s.queue:
			if !h.Ready() {
				continue
			}
			s.mu.Lock()
			if !f.Expires.After(time.Now()) {
				h.expired.Add(1)
			} else if h.Ready() {
				var dropped uint64
				for c := range s.clients {
					select {
					case c.messages <- f:
					default:
						dropped++
					}
				}
				h.connectionDropped.Add(dropped)
			}
			s.mu.Unlock()
		}
	}
}

func drain(q chan Frame) {
	for {
		select {
		case <-q:
		default:
			return
		}
	}
}

func (h *Hub) Close() {
	h.closeOnce.Do(func() {
		h.ready.Store(false)
		h.cancel()
		h.workers.Wait()
		// Wait for a concurrent producer before draining the queues.
		h.receiveMu.Lock()
		defer h.receiveMu.Unlock()
		h.quotaMu.Lock()
		defer h.quotaMu.Unlock()
		for _, s := range h.shards {
			s.mu.Lock()
			for c := range s.clients {
				drain(c.messages)
			}
			clear(s.clients)
			drain(s.queue)
			s.mu.Unlock()
		}
		clear(h.sessionCounts)
		h.connections = 0
	})
}

// Stats describe server-side processing, never successful playback.
type Stats struct {
	Connections                                       int
	Invalid, ShardDropped, ConnectionDropped, Expired uint64
	ShardQueueDepths                                  []int
}

func (h *Hub) Stats() Stats {
	h.quotaMu.Lock()
	n := h.connections
	h.quotaMu.Unlock()
	depths := make([]int, len(h.shards))
	for i, s := range h.shards {
		depths[i] = len(s.queue)
	}
	return Stats{Connections: n, Invalid: h.invalid.Load(), ShardDropped: h.shardDropped.Load(), ConnectionDropped: h.connectionDropped.Load(), Expired: h.expired.Load(), ShardQueueDepths: depths}
}
