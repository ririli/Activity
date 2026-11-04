package main

import (
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/application"
	store "activity/services/identity-service/internal/identity/infrastructure/mysql"
	limit "activity/services/identity-service/internal/identity/infrastructure/redis"
	"activity/services/identity-service/internal/identity/infrastructure/security"
	httpapi "activity/services/identity-service/internal/identity/interfaces/http"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"strings"
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
func main() {
	db := platform.OpenDB()
	defer db.Close()
	r := platform.Redis()
	defer r.Close()
	dummy, e := security.HashPassword(platform.Random())
	platform.Must(e)
	s := &application.Service{Repo: store.Repository{DB: db}, Access: signer("access", "activity-api"), Refresh: signer("refresh", "identity-refresh"), Limiter: limit.Limiter{R: r}, HashSlots: make(chan struct{}, 4), DummyHash: dummy}
	g := platform.Router(db, r)
	httpapi.Routes(g, s)
	platform.Run(g, func(ctx context.Context) { platform.Dispatch(ctx, db) })
}
