package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     uint  `json:"user_id"`
	TelegramID int64 `json:"telegram_id"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for a user
func GenerateJWT(userID uint, telegramID int64, secret string) (string, error) {
	claims := &Claims{
		UserID:     userID,
		TelegramID: telegramID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateSecureCode generates a cryptographically secure random code
func GenerateSecureCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// GenerateRandomToken generates a secure random token
func GenerateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
