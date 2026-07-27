package utils

import (
	"errors"
	"long/internal/config"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Permission int64

	jwt.RegisteredClaims
}

func NewToken(userId int64, permission int64) (string, error) {
	payload := Claims{
		Permission: permission,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userId, 10),
			Issuer:    config.GetConfig().JWT.Issuer,
			Audience:  []string{config.GetConfig().JWT.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GetConfig().JWT.SessionTTL) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, payload)

	return t.SignedString(config.GetConfig().JWT.PrivateKey)
}

func ParseToken(s string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		s,
		claims,
		func(t *jwt.Token) (any, error) {
			return config.GetConfig().JWT.PrivateKey.Public(), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithAudience(config.GetConfig().JWT.Audience),
		jwt.WithIssuer(config.GetConfig().JWT.Issuer),
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("Invalid token")
	}

	return claims, nil
}
