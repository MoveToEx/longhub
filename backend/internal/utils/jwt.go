package utils

import (
	"errors"
	"long/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     int64
	Permission int64

	jwt.RegisteredClaims
}

func NewToken(UserId int64, Permission int64) (string, error) {
	payload := Claims{
		UserID:     UserId,
		Permission: Permission,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GetConfig().JWT.SessionTTL) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, payload)

	return t.SignedString(config.GetConfig().JWT.PrivateKey)
}

func ParseToken(Token string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(Token, &Claims{}, func(t *jwt.Token) (any, error) {
		return config.GetConfig().JWT.PublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Invalid token")
}
