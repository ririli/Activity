package main

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/application"
	identitygrpc "activity/services/ranking-service/internal/ranking/infrastructure/identitygrpc"
	store "activity/services/ranking-service/internal/ranking/infrastructure/mysql"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	httpapi "activity/services/ranking-service/internal/ranking/interfaces/http"
	"context"
	"os"
	"strconv"
	"time"
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
	projection := &cache.Cache{Client: r, LoadBounds: func(ctx context.Context, edition string) (cache.Bounds, error) {
		var b cache.Bounds
		e := platform.Call(ctx, platform.Env("ACTIVITY_URL", "http://localhost:8082"), "/internal/activity/v1/encoding", map[string]string{"edition_id": edition}, &b)
		return b, e
	}}
	repo := &store.Repository{DB: db, Redis: r, Cache: projection}
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
	g := platform.Router(db, r)
	httpapi.Routes(g, application.Service{Repo: repo, Profiles: profiles})
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db)
		platform.Consume(ctx, "ranking.facts.v1", []string{"ParticipationInitialized.v1", "FormalDrawCommitted.v1", "RankingAwardGranted.v1"}, repo.Consume)
	})
}
