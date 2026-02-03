package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set required environment variables
	os.Setenv("BOT_TOKEN", "test_bot_token")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("JWT_SECRET_KEY", "this_is_a_test_secret_key_with_32_chars_minimum")
	os.Setenv("AES_ENCRYPTION_KEY", "12345678901234567890123456789012") // exactly 32 bytes
	defer func() {
		os.Unsetenv("BOT_TOKEN")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("JWT_SECRET_KEY")
		os.Unsetenv("AES_ENCRYPTION_KEY")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.BotToken != "test_bot_token" {
		t.Errorf("BotToken = %q, want %q", cfg.BotToken, "test_bot_token")
	}

	if cfg.DBPassword != "test_password" {
		t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "test_password")
	}

	if len(cfg.AESKey) != 32 {
		t.Errorf("AESKey length = %d, want 32", len(cfg.AESKey))
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name: "Missing BOT_TOKEN",
			envVars: map[string]string{
				"DB_PASSWORD":        "password",
				"JWT_SECRET_KEY":     "this_is_a_test_secret_key_with_32_chars_minimum",
				"AES_ENCRYPTION_KEY": "12345678901234567890123456789012",
			},
		},
		{
			name: "Missing DB_PASSWORD",
			envVars: map[string]string{
				"BOT_TOKEN":          "token",
				"JWT_SECRET_KEY":     "this_is_a_test_secret_key_with_32_chars_minimum",
				"AES_ENCRYPTION_KEY": "12345678901234567890123456789012",
			},
		},
		{
			name: "Missing JWT_SECRET_KEY",
			envVars: map[string]string{
				"BOT_TOKEN":          "token",
				"DB_PASSWORD":        "password",
				"AES_ENCRYPTION_KEY": "12345678901234567890123456789012",
			},
		},
		{
			name: "Missing AES_ENCRYPTION_KEY",
			envVars: map[string]string{
				"BOT_TOKEN":      "token",
				"DB_PASSWORD":    "password",
				"JWT_SECRET_KEY": "this_is_a_test_secret_key_with_32_chars_minimum",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			os.Clearenv()

			// Set only the provided env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			_, err := LoadConfig()
			if err == nil {
				t.Error("LoadConfig() expected error for missing required field, got nil")
			}
		})
	}
}

func TestValidate_JWTSecretTooShort(t *testing.T) {
	cfg := &Config{
		BotToken:   "token",
		DBPassword: "password",
		JWTSecret:  "short", // Less than 32 chars
		AESKey:     "12345678901234567890123456789012",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for short JWT secret, got nil")
	}
}

func TestValidate_AESKeyWrongLength(t *testing.T) {
	tests := []struct {
		name   string
		aesKey string
	}{
		{
			name:   "Too short",
			aesKey: "short",
		},
		{
			name:   "Too long",
			aesKey: "123456789012345678901234567890123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				BotToken:   "token",
				DBPassword: "password",
				JWTSecret:  "this_is_a_test_secret_key_with_32_chars_minimum",
				AESKey:     tt.aesKey,
			}

			err := cfg.Validate()
			if err == nil {
				t.Error("Validate() expected error for wrong AES key length, got nil")
			}
		})
	}
}

func TestValidateProductionSecurity(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		shouldErr bool
	}{
		{
			name: "Valid production config",
			cfg: &Config{
				AppEnv:         "production",
				DBSSLMode:      "require",
				JWTSecret:      "production_secret_key_different_from_default",
				AESKey:         "production_aes_key_32_bytes!",
				SuperAdminTgID: 123456789,
			},
			shouldErr: false,
		},
		{
			name: "Development mode - no validation",
			cfg: &Config{
				AppEnv:    "development",
				DBSSLMode: "disable",
			},
			shouldErr: false,
		},
		{
			name: "Production without SSL",
			cfg: &Config{
				AppEnv:         "production",
				DBSSLMode:      "disable",
				JWTSecret:      "production_secret",
				AESKey:         "production_aes_key_32_bytes!",
				SuperAdminTgID: 123456789,
			},
			shouldErr: true,
		},
		{
			name: "Production with default JWT secret",
			cfg: &Config{
				AppEnv:         "production",
				DBSSLMode:      "require",
				JWTSecret:      "your_jwt_secret_minimum_32_chars_here_change_this",
				AESKey:         "production_aes_key_32_bytes!",
				SuperAdminTgID: 123456789,
			},
			shouldErr: true,
		},
		{
			name: "Production without super admin",
			cfg: &Config{
				AppEnv:         "production",
				DBSSLMode:      "require",
				JWTSecret:      "production_secret_key_different",
				AESKey:         "production_aes_key_32_bytes!",
				SuperAdminTgID: 0,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateProductionSecurity()
			if tt.shouldErr && err == nil {
				t.Error("ValidateProductionSecurity() expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("ValidateProductionSecurity() unexpected error = %v", err)
			}
		})
	}
}

func TestGetDSN(t *testing.T) {
	cfg := &Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBName:     "testdb",
		DBSSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	dsn := cfg.GetDSN()

	if dsn != expected {
		t.Errorf("GetDSN() = %q, want %q", dsn, expected)
	}
}

func TestGetMatchTimeout(t *testing.T) {
	cfg := &Config{
		MatchTimeoutMinutes: 5,
	}

	timeout := cfg.GetMatchTimeout()
	expected := 5 * 60 * 1000000000 // 5 minutes in nanoseconds

	if timeout.Nanoseconds() != int64(expected) {
		t.Errorf("GetMatchTimeout() = %v, want %v", timeout, expected)
	}
}
