package platform

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"strings"
	"sync"
	"time"
)

var publicOnce sync.Once
var accessPublic ed25519.PublicKey
var publicError error

func localAccess(raw string) error {
	publicOnce.Do(func() {
		var b []byte
		b, publicError = os.ReadFile(os.Getenv("ACCESS_PUBLIC_KEY_FILE"))
		if publicError != nil {
			return
		}
		accessPublic, publicError = hex.DecodeString(strings.TrimSpace(string(b)))
		if len(accessPublic) != 32 {
			publicError = errors.New("invalid Access public key")
		}
	})
	if publicError != nil {
		return publicError
	}
	if len(raw) > 4096 {
		return Fail(401, "INVALID_TOKEN")
	}
	claims := jwt.MapClaims{}
	_, e := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Header["kid"] != "v1" {
			return nil, errors.New("unknown kid")
		}
		return accessPublic, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithIssuer("activity-identity"), jwt.WithAudience("activity-api"), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil {
		return Fail(401, "INVALID_TOKEN")
	}
	if claims["token_use"] != "access" || claims["sid"] == nil || claims["jti"] == nil || claims["nbf"] == nil {
		return Fail(401, "INVALID_TOKEN")
	}
	exp, e := claims.GetExpirationTime()
	if e != nil || exp == nil || !exp.After(time.Now()) {
		return Fail(401, "INVALID_TOKEN")
	}
	return nil
}
