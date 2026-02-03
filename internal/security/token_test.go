package security

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name       string
		userID     uint
		telegramID int64
	}{
		{
			name:       "Regular user",
			userID:     1,
			telegramID: 123456789,
		},
		{
			name:       "Admin user",
			userID:     2,
			telegramID: 987654321,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.userID, tt.telegramID, "test_secret_key_minimum_32_chars")
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			if token == "" {
				t.Error("GenerateToken() returned empty token")
			}

			// Validate the token
			claims, err := ValidateJWT(token, "test_secret_key_minimum_32_chars")
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			if claims.UserID != tt.userID {
				t.Errorf("UserID = %d, want %d", claims.UserID, tt.userID)
			}

			if claims.TelegramID != tt.telegramID {
				t.Errorf("TelegramID = %d, want %d", claims.TelegramID, tt.telegramID)
			}
		})
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "Empty token",
			token: "",
		},
		{
			name:  "Invalid format",
			token: "invalid.token.here",
		},
		{
			name:  "Random string",
			token: "randomstring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateJWT(tt.token, "test_secret_key_minimum_32_chars")
			if err == nil {
				t.Error("ValidateToken() expected error for invalid token, got nil")
			}
		})
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// This test would require mocking time or using a very short expiration
	// For now, we'll skip it as it's complex to test properly
	t.Skip("Expiration testing requires time mocking")
}

func TestTokenRoundTrip(t *testing.T) {
	// Generate token
	userID := uint(42)
	telegramID := int64(123456789)

	token, err := GenerateJWT(userID, telegramID, "test_secret_key_minimum_32_chars")
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	// Validate token
	claims, err := ValidateJWT(token, "test_secret_key_minimum_32_chars")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	// Verify all claims
	if claims.UserID != userID {
		t.Errorf("UserID = %d, want %d", claims.UserID, userID)
	}

	if claims.TelegramID != telegramID {
		t.Errorf("TelegramID = %d, want %d", claims.TelegramID, telegramID)
	}

	// Verify expiration is in the future
	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("Token already expired")
	}

	// Verify expiration is within 24 hours
	expectedExpiry := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt.Time.After(expectedExpiry.Add(time.Minute)) {
		t.Error("Token expiration is too far in the future")
	}
}
