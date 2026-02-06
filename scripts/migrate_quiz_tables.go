package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mroshb/game_bot/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get database connection string
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("🚀 Starting Quiz Match tables migration...")

	// Create tables
	if err := db.AutoMigrate(
		&models.QuizMatch{},
		&models.QuizRound{},
		&models.QuizAnswer{},
		&models.UserBooster{},
	); err != nil {
		log.Fatalf("Failed to migrate tables: %v", err)
	}

	fmt.Println("✅ Tables created successfully!")

	// Create indexes
	fmt.Println("📊 Creating indexes...")

	// QuizMatch indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_matches_user1 ON quiz_matches(user1_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_matches_user2 ON quiz_matches(user2_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_matches_state ON quiz_matches(state)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_matches_last_activity ON quiz_matches(last_activity_at)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_matches_timeout ON quiz_matches(timeout_at) WHERE state NOT IN ('finished', 'timeout')")

	// QuizRound indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_rounds_match ON quiz_rounds(match_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_quiz_rounds_match_round ON quiz_rounds(match_id, round_number)")

	// QuizAnswer indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_answers_match ON quiz_answers(match_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_answers_round ON quiz_answers(round_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_quiz_answers_user ON quiz_answers(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_quiz_answers_unique ON quiz_answers(match_id, round_id, user_id, question_number)")

	fmt.Println("✅ Indexes created successfully!")

	// Set default timeout_at for existing records (if any)
	fmt.Println("⏰ Setting default timeout values...")
	db.Exec("UPDATE quiz_matches SET timeout_at = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE timeout_at IS NULL")

	fmt.Println("✅ Migration completed successfully!")
	fmt.Println("")
	fmt.Println("📋 Summary:")
	fmt.Println("  - quiz_matches table created")
	fmt.Println("  - quiz_rounds table created")
	fmt.Println("  - quiz_answers table created")
	fmt.Println("  - user_boosters table created")
	fmt.Println("  - All indexes created")
	fmt.Println("")
	fmt.Println("🎮 Quiz game system is ready!")
}
