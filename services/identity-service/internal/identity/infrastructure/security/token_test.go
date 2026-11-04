package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestTokenSigner(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s := TokenSigner{Issuer: "activity", Audience: "activity-api", TokenUse: "access", KeyID: "one", PrivateKey: priv, PublicKeys: map[string]ed25519.PublicKey{"one": pub}}
	now := time.Now()
	token, err := s.Issue("123", "session", now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.Verify(token)
	if err != nil || claims.Subject != "123" || claims.SessionID != "session" {
		t.Fatalf("claims %+v error %v", claims, err)
	}
	s.TokenUse = "refresh"
	if _, err := s.Verify(token); err == nil {
		t.Fatal("accepted wrong token use")
	}
}
