package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/mroshb/game_bot/internal/config"
	"github.com/mroshb/game_bot/internal/database"
	"github.com/mroshb/game_bot/pkg/logger"
	"github.com/mroshb/game_bot/telegram"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	// Initialize logger
	logger.Init()
	defer logger.Sync()

	logger.Info("Starting Telegram Game Bot...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", err)
	}

	// Validate production security settings
	if cfg.AppEnv == "production" {
		if err := cfg.ValidateProductionSecurity(); err != nil {
			logger.Fatal("Production security validation failed", err)
		}
		logger.Info("Production security validation passed")
	}

	// Connect to database with TLS
	db, err := database.Connect(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", err)
	}

	// Run GORM auto-migration
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to run migrations", err)
	}

	// Seed test questions
	if err := database.SeedQuestions(db); err != nil {
		logger.Warn("Failed to seed questions", "error", err)
	}

	// Initialize and start Telegram bot
	bot, err := telegram.InitBot(cfg, db)
	if err != nil {
		logger.Fatal("Failed to initialize bot", err)
	}

	logger.Info("Bot started successfully", "env", cfg.AppEnv)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")
	bot.Stop()
	logger.Info("Bot stopped")
}
