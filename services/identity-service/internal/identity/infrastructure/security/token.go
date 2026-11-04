package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	SessionID string `json:"sid"`
	TokenUse  string `json:"token_use"`
	jwt.RegisteredClaims
}

type TokenSigner struct {
	Issuer     string
	Audience   string
	TokenUse   string
	KeyID      string
	PrivateKey ed25519.PrivateKey
	PublicKeys map[string]ed25519.PublicKey
}

func (s TokenSigner) Issue(userID, sessionID string, now, expires time.Time) (string, error) {
	if len(s.PrivateKey) != ed25519.PrivateKeySize || s.Issuer == "" || s.Audience == "" || s.TokenUse == "" || s.KeyID == "" || userID == "" || sessionID == "" || !expires.After(now) {
		return "", errors.New("invalid token configuration")
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	claims := Claims{SessionID: sessionID, TokenUse: s.TokenUse, RegisteredClaims: jwt.RegisteredClaims{Issuer: s.Issuer, Subject: userID, Audience: jwt.ClaimStrings{s.Audience}, ID: hex.EncodeToString(id), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = s.KeyID
	return token.SignedString(s.PrivateKey)
}

func (s TokenSigner) Verify(raw string) (*Claims, error) {
	if len(raw) == 0 || len(raw) > 4096 || s.Issuer == "" || s.Audience == "" || s.TokenUse == "" {
		return nil, errors.New("invalid token")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		key, found := s.PublicKeys[kid]
		if !ok || !found || len(key) != ed25519.PublicKeySize {
			return nil, errors.New("untrusted signing key")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithIssuer(s.Issuer), jwt.WithAudience(s.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid || claims.Subject == "" || claims.SessionID == "" || claims.ID == "" || claims.IssuedAt == nil || claims.NotBefore == nil || claims.TokenUse != s.TokenUse {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
