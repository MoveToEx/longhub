package utils

import (
	"errors"
	"long/internal/config"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func NewToken(userID int64) (string, error) {
	now := time.Now()
	payload := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    config.GetConfig().JWT.Issuer,
		Audience:  []string{config.GetConfig().JWT.Audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(config.GetConfig().JWT.SessionTTL) * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, payload)

	return t.SignedString(config.GetConfig().JWT.PrivateKey)
}

func ParseToken(s string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}

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
