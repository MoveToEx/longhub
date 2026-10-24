package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"long/internal/config"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSessionTokenClaims(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previous := config.GetConfig().JWT
	config.GetConfig().JWT = config.JWTConfig{PrivateKey: private, Issuer: "test", Audience: "test", SessionTTL: 1}
	t.Cleanup(func() { config.GetConfig().JWT = previous })

	t.Run("new tokens contain only registered claims", func(t *testing.T) {
		token, err := NewToken(9007199254740993)
		if err != nil {
			t.Fatal(err)
		}
		claims, err := ParseToken(token)
		if err != nil {
			t.Fatal(err)
		}
		if claims.Subject != "9007199254740993" || claims.Issuer != "test" || len(claims.Audience) != 1 || claims.Audience[0] != "test" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
			t.Fatalf("unexpected session claims: %+v", claims)
		}
		if duration := claims.ExpiresAt.Sub(claims.IssuedAt.Time); duration != time.Hour {
			t.Fatalf("session duration %v, want 1 hour", duration)
		}
		payload, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		for field := range fields {
			switch field {
			case "sub", "iss", "aud", "iat", "exp":
			default:
				t.Errorf("unexpected JWT claim %q", field)
			}
		}
	})

	t.Run("existing tokens with permission claims still authenticate", func(t *testing.T) {
		token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
			"sub": "1", "iss": "test", "aud": []string{"test"},
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "Permission": 15,
		}).SignedString(private)
		if err != nil {
			t.Fatal(err)
		}
		claims, err := ParseToken(token)
		if err != nil || claims.Subject != "1" {
			t.Fatalf("existing token rejected: %v", err)
		}
	})
}
