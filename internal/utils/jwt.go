package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret is the secret key used for signing and verifying JWT tokens.
// It should be loaded securely from environment variables.
var jwtSecret []byte

// SetJWTSecret sets the JWT secret key from environment variables.
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateJWT generates a new JWT token for a given user ID.
func GenerateJWT(userID int) (string, error) {
	if len(jwtSecret) == 0 {
		return "", fmt.Errorf("JWT secret not set")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// ParseJWT parses and validates a JWT token string.
// It returns the user ID if the token is valid, otherwise an error.
func ParseJWT(tokenString string) (int, error) {
	if len(jwtSecret) == 0 {
		return 0, fmt.Errorf("JWT secret not set")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	userIDFloat, ok := claims["user_id"].(float64) // JWT numbers are float64 by default
	if !ok {
		return 0, fmt.Errorf("user ID not found in token claims")
	}

	return int(userIDFloat), nil
}
