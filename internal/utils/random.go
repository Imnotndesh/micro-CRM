package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateRandomState generates a secure random string suitable for use as an OIDC state parameter
func GenerateOIDCState() (string, error) {
	// 32 bytes = 256 bits of entropy
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// URL-safe base64 encoding
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
