package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomBase64String(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	encoded := base64.URLEncoding.EncodeToString(b)

	return encoded, nil
}
