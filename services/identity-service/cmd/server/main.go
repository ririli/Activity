package main

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"activity/services/identity-service/internal/identity/application"
	revcache "activity/services/identity-service/internal/identity/infrastructure/cache"
	store "activity/services/identity-service/internal/identity/infrastructure/mysql"
	limit "activity/services/identity-service/internal/identity/infrastructure/redis"
	"activity/services/identity-service/internal/identity/infrastructure/security"
	grpcapi "activity/services/identity-service/internal/identity/interfaces/grpc"
	httpapi "activity/services/identity-service/internal/identity/interfaces/http"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func signer(kind, aud string) security.TokenSigner {
	b, e := os.ReadFile(os.Getenv(strings.ToUpper(kind) + "_KEY_FILE"))
	platform.Must(e)
	seed, e := hex.DecodeString(strings.TrimSpace(string(b)))
	platform.Must(e)
	if len(seed) != ed25519.SeedSize {
		panic("expected hex Ed25519 seed")
	}
	key := ed25519.NewKeyFromSeed(seed)
	return security.TokenSigner{Issuer: "activity-identity", Audience: aud, TokenUse: kind, KeyID: "v1", PrivateKey: key, PublicKeys: map[string]ed25519.PublicKey{"v1": key.Public().(ed25519.PublicKey)}}
}
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
	dummy, e := security.HashPassword(platform.Random())
	platform.Must(e)
	var revocations application.RevocationStore
	if platform.Env("REVOCATION_CACHE_ENABLED", "true") != "false" {
		c, err := revcache.NewRevocationCache(r, revcache.Config{
			NegativeTTL: durationEnv("REVOKE_NEGATIVE_TTL", 2*time.Second),
			LifeWindow:  durationEnv("REVOKE_NEGATIVE_LIFE_WINDOW", 2*time.Second),
			CleanWindow: durationEnv("REVOKE_NEGATIVE_CLEAN_WINDOW", time.Second),
			HardMaxMB:   intEnv("REVOKE_NEGATIVE_MAX_MB", 16),
			Padding:     durationEnv("REVOKE_TTL_PADDING", 30*time.Second),
		})
		platform.Must(err)
		revocations = c
	}
	limiter := ratelimit.NewRedis()
	defer limiter.Close()
	s := &application.Service{Repo: store.Repository{DB: db}, Access: signer("access", "activity-api"), Refresh: signer("refresh", "identity-refresh"), Limiter: limit.NewLimiter(limiter.Client, platform.Env("REGISTER_IP_WHITELIST", "")), Revocations: revocations, HashSlots: make(chan struct{}, 4), DummyHash: dummy}

	lis, e := net.Listen("tcp", platform.Env("GRPC_ADDR", ":9090"))
	platform.Must(e)
	var grpcReady atomic.Bool
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(grpcapi.InternalAuth()),
		grpc.ChainStreamInterceptor(grpcapi.InternalAuthStream()),
		grpc.MaxRecvMsgSize(1<<20),
	)
	grpcapi.Register(grpcServer, s)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	go func() {
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			slog.Error("grpc server failed", "error", err)
			grpcReady.Store(false)
		}
	}()
	grpcReady.Store(true)

	g := platform.Router(db, r, func() error {
		if !grpcReady.Load() {
			return errors.New("grpc server not ready")
		}
		return nil
	})
	httpapi.Routes(g, s, limiter)
	platform.Run(g, func(ctx context.Context) { platform.Dispatch(ctx, db) })

	grpcReady.Store(false)
	healthServer.Shutdown()
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		grpcServer.Stop()
	}
}
