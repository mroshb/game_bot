package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Telegram
	BotToken string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Security
	JWTSecret      string
	AESKey         string
	SuperAdminTgID int64

	// Application
	AppEnv        string
	AppPort       string
	LogLevel      string
	UploadMaxSize int64

	// Rate Limiting
	RateLimitPerUser int
	RateLimitPerIP   int

	// Matchmaking
	MatchTimeoutMinutes int
	MatchCostCoins      int64
	FriendRequestCost   int64
	MessageCost         int64

	// Game
	DefaultCoins   int64
	WinRewardCoins int64
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		BotToken:   getEnv("BOT_TOKEN", ""),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "gamebot"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "gamebot_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret: getEnv("JWT_SECRET_KEY", ""),
		AESKey:    getEnv("AES_ENCRYPTION_KEY", ""),

		AppEnv:        getEnv("APP_ENV", "development"),
		AppPort:       getEnv("APP_PORT", "8080"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		UploadMaxSize: getEnvInt64("UPLOAD_MAX_SIZE", 5242880),

		RateLimitPerUser: getEnvInt("RATE_LIMIT_PER_USER", 20),
		RateLimitPerIP:   getEnvInt("RATE_LIMIT_PER_IP", 100),

		MatchTimeoutMinutes: getEnvInt("MATCH_TIMEOUT_MINUTES", 5),
		MatchCostCoins:      getEnvInt64("MATCH_COST_COINS", 5),
		FriendRequestCost:   getEnvInt64("FRIEND_REQUEST_COST", 20),
		MessageCost:         getEnvInt64("MESSAGE_COST", 1),

		DefaultCoins:   getEnvInt64("DEFAULT_COINS", 100),
		WinRewardCoins: getEnvInt64("WIN_REWARD_COINS", 50),
	}

	// Parse super admin telegram ID
	superAdminStr := getEnv("SUPER_ADMIN_TELEGRAM_ID", "")
	if superAdminStr != "" {
		id, err := strconv.ParseInt(superAdminStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SUPER_ADMIN_TELEGRAM_ID: %w", err)
		}
		cfg.SuperAdminTgID = id
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}
	if c.DBPassword == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must be at least 32 characters")
	}
	if c.AESKey == "" {
		return fmt.Errorf("AES_ENCRYPTION_KEY is required")
	}
	if len(c.AESKey) != 32 {
		return fmt.Errorf("AES_ENCRYPTION_KEY must be exactly 32 bytes")
	}
	return nil
}

func (c *Config) ValidateProductionSecurity() error {
	if c.AppEnv != "production" {
		return nil
	}

	if c.DBSSLMode != "require" {
		return fmt.Errorf("DB_SSLMODE must be 'require' in production")
	}
	if c.JWTSecret == "your_jwt_secret_minimum_32_chars_here_change_this" {
		return fmt.Errorf("JWT_SECRET_KEY must be changed from default in production")
	}
	if c.AESKey == "your_aes_key_must_be_32_bytes!!" {
		return fmt.Errorf("AES_ENCRYPTION_KEY must be changed from default in production")
	}
	if c.SuperAdminTgID == 0 {
		return fmt.Errorf("SUPER_ADMIN_TELEGRAM_ID must be set in production")
	}

	return nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func (c *Config) GetMatchTimeout() time.Duration {
	return time.Duration(c.MatchTimeoutMinutes) * time.Minute
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}
