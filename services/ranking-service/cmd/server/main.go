package main

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"activity/services/ranking-service/internal/ranking/application"
	"activity/services/ranking-service/internal/ranking/infrastructure/activityhttp"
	identitygrpc "activity/services/ranking-service/internal/ranking/infrastructure/identitygrpc"
	store "activity/services/ranking-service/internal/ranking/infrastructure/mysql"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	httpapi "activity/services/ranking-service/internal/ranking/interfaces/http"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	rdb "github.com/redis/go-redis/v9"
)

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := platform.Env(key, "")
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	platform.Must(err)
	return v
}
func intEnv(key string, fallback int) int {
	raw := platform.Env(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	platform.Must(err)
	return v
}
func main() {
	db := platform.OpenDB()
	defer db.Close()
	r := platform.Redis()
	defer r.Close()
	clients, err := rankingRedisClients(r)
	platform.Must(err)
	for _, client := range clients {
		if client != r {
			defer client.Close()
		}
	}
	activityURL := platform.Env("ACTIVITY_URL", "http://localhost:8082")
	var boundsByEdition sync.Map
	loadBounds := func(ctx context.Context, edition string) (cache.Bounds, error) {
		if value, ok := boundsByEdition.Load(edition); ok {
			return value.(cache.Bounds), nil
		}
		var b cache.Bounds
		e := platform.Call(ctx, activityURL, "/internal/activity/v1/encoding", map[string]string{"edition_id": edition}, &b)
		if e == nil {
			boundsByEdition.Store(edition, b)
		}
		return b, e
	}
	projection := &cache.Cache{Client: r, Clients: clients, LoadBounds: loadBounds}
	hourlyCache := &cache.HourlyCache{Clients: clients, LoadBounds: loadBounds, Namespace: platform.Env("APP_ENV", "local")}
	hourlyWorker := application.HourlyWorker{Store: store.HourlyStore{DB: db}, Activity: activityhttp.Client{URL: activityURL}, Projection: hourlyCache, Wake: make(chan struct{}, 1)}
	repo := &store.Repository{DB: db, Cache: projection, ProjectHourly: hourlyWorker.Project}
	profiles, err := identitygrpc.New(
		platform.Env("IDENTITY_GRPC_ADDR", "localhost:9090"),
		os.Getenv("INTERNAL_TOKEN"),
		durationEnv("PROFILE_TIMEOUT", 200*time.Millisecond),
		durationEnv("PROFILE_CACHE_TTL", 60*time.Second),
		durationEnv("PROFILE_CACHE_STALE_TTL", 10*time.Minute),
		intEnv("PROFILE_CACHE_MAX", 10000),
	)
	platform.Must(err)
	defer profiles.Close()
	limiter := ratelimit.NewRedis()
	defer limiter.Close()
	// Ranking reads and settlement can use MySQL while Redis is unavailable.
	g := platform.Router(db, nil)
	httpapi.Routes(g, application.Service{Repo: repo, Profiles: profiles}, limiter)
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db)
		go hourlyWorker.Schedule(ctx)
		go hourlyWorker.Run(ctx)
		go hourlyWorker.Retire(ctx)
		for range 4 {
			go hourlyWorker.Deliver(ctx)
		}
		if os.Getenv("RANKING_HOURLY_WORKER_ONLY") == "true" {
			// Additional workers poll the durable state left by the active consumer.
			<-ctx.Done()
			return
		}
		go repo.MaintainCache(ctx)
		go hourlyWorker.Warm(ctx)
		platform.Consume(ctx, "ranking.facts.v1", []string{"ParticipationInitialized.v1", "FormalDrawCommitted.v1", "RankingAwardGranted.v1", hourly.BarrierType}, func(ctx context.Context, ev platform.Event) error {
			if ev.Type == hourly.BarrierType {
				return hourlyWorker.Barrier(ctx, ev)
			}
			return repo.Consume(ctx, ev)
		})
	})
}

func rankingRedisClients(fallback *rdb.Client) ([hourly.Shards]*rdb.Client, error) {
	var clients [hourly.Shards]*rdb.Client
	raw := os.Getenv("RANKING_REDIS_SHARD_ADDRS")
	if raw == "" {
		for i := range clients {
			clients[i] = fallback
		}
		return clients, nil
	}
	addresses := strings.Split(raw, ",")
	if len(addresses) != hourly.Shards {
		return clients, fmt.Errorf("RANKING_REDIS_SHARD_ADDRS requires %d addresses", hourly.Shards)
	}
	for i, address := range addresses {
		addresses[i] = strings.TrimSpace(address)
		if addresses[i] == "" {
			return clients, fmt.Errorf("empty Redis shard address at position %d", i)
		}
	}
	for i, address := range addresses {
		options := *fallback.Options()
		options.Addr = address
		clients[i] = rdb.NewClient(&options)
	}
	return clients, nil
}
