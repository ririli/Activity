package identitygrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	identityv1 "activity/contracts/rpc/identity/v1"
	"activity/services/ranking-service/internal/ranking/application"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type cacheEntry struct {
	name       string
	freshUntil time.Time
	staleUntil time.Time
}

// Client is the ranking-side adapter for the identity profile RPC. It keeps a
// short in-process name cache and degrades profile failures to placeholders;
// only internal authentication/configuration failures are surfaced.
type Client struct {
	conn     *grpc.ClientConn
	rpc      identityv1.IdentityServiceClient
	timeout  time.Duration
	token    string
	ttl      time.Duration
	staleTTL time.Duration
	max      int
	now      func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry

	fetch func(context.Context, []string) (map[string]string, error)
}

func New(addr, token string, timeout, ttl, staleTTL time.Duration, max int) (*Client, error) {
	if addr == "" {
		return nil, errors.New("IDENTITY_GRPC_ADDR is required")
	}
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	if staleTTL < ttl {
		staleTTL = ttl
	}
	if max <= 0 {
		max = 10000
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1<<20)),
	)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:     conn,
		rpc:      identityv1.NewIdentityServiceClient(conn),
		timeout:  timeout,
		token:    token,
		ttl:      ttl,
		staleTTL: staleTTL,
		max:      max,
		now:      time.Now,
		cache:    map[string]cacheEntry{},
	}
	c.fetch = c.fetchRPC
	return c, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) BatchGet(ctx context.Context, userIDs []string) (map[string]string, error) {
	result := map[string]string{}
	if len(userIDs) == 0 {
		return result, nil
	}
	seen := map[string]struct{}{}
	misses := make([]string, 0, len(userIDs))
	now := c.now()
	c.mu.Lock()
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if entry, ok := c.cache[id]; ok && now.Before(entry.freshUntil) {
			result[id] = entry.name
			continue
		}
		misses = append(misses, id)
	}
	c.mu.Unlock()
	if len(misses) == 0 {
		return result, nil
	}

	names, err := c.fetch(ctx, misses)
	if err != nil {
		c.mu.Lock()
		for _, id := range misses {
			if entry, ok := c.cache[id]; ok && now.Before(entry.staleUntil) {
				result[id] = entry.name
			}
		}
		c.mu.Unlock()
		if status.Code(err) == codes.Unauthenticated || status.Code(err) == codes.PermissionDenied {
			return result, fmt.Errorf("%w: %v", application.ErrProfileAuth, err)
		}
		return result, err
	}

	c.mu.Lock()
	for _, id := range misses {
		name := names[id]
		c.cache[id] = cacheEntry{name: name, freshUntil: now.Add(c.ttl), staleUntil: now.Add(c.staleTTL)}
		result[id] = name
	}
	c.evictLocked(now)
	c.mu.Unlock()
	return result, nil
}

func (c *Client) fetchRPC(ctx context.Context, userIDs []string) (map[string]string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if c.token != "" {
		callCtx = metadata.AppendToOutgoingContext(callCtx, "x-internal-token", c.token)
	}
	resp, err := c.rpc.BatchGetPublicProfiles(callCtx, &identityv1.BatchGetPublicProfilesRequest{UserIds: userIDs})
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(resp.GetProfiles()))
	for _, p := range resp.GetProfiles() {
		names[p.GetUserId()] = p.GetDisplayName()
	}
	return names, nil
}

func (c *Client) evictLocked(now time.Time) {
	if len(c.cache) <= c.max {
		return
	}
	for id, entry := range c.cache {
		if !now.Before(entry.staleUntil) {
			delete(c.cache, id)
		}
	}
	for id := range c.cache {
		if len(c.cache) <= c.max {
			return
		}
		delete(c.cache, id)
	}
}
