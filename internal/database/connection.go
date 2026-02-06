package database

import (
	"fmt"
	"time"

	"github.com/mroshb/game_bot/internal/config"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	var logLevel gormlogger.LogLevel
	if cfg.AppEnv == "development" {
		logLevel = gormlogger.Info
	} else {
		logLevel = gormlogger.Error
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// High performance settings
		SkipDefaultTransaction: true, // Skip wrapping every operation in a transaction
		PrepareStmt:            true, // Cache prepared statements
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Optimized Connection Pool Settings for 1000+ Concurrent Requests
	// These settings allow the pool to scale up to handle high load while maintaining
	// a healthy number of warm idle connections.
	sqlDB.SetMaxIdleConns(50)                  // Keep 50 idle connections warm
	sqlDB.SetMaxOpenConns(500)                 // Scale up to 500 connections under load
	sqlDB.SetConnMaxLifetime(time.Hour)        // Cycle connections hourly to prevent stale leaks
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections after 10m to free DB resources

	logger.Info("Database connected successfully with high-performance pool settings")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	logger.Info("Running database migrations...")

	// Manually drop phone_hash if it exists (as user requested its removal)
	if db.Migrator().HasColumn(&models.User{}, "phone_hash") {
		logger.Info("Dropping phone_hash column from users table...")
		db.Migrator().DropColumn(&models.User{}, "phone_hash")
	}

	err := db.AutoMigrate(
		&models.User{},
		&models.CoinTransaction{},
		&models.MatchSession{},
		&models.MatchmakingQueue{},
		&models.Friendship{},
		&models.Question{},
		&models.GameSession{},
		&models.GameParticipant{},
		&models.Room{},
		&models.RoomMember{},
		&models.Village{},
		&models.VillageMember{},
		&models.TodGame{},
		&models.TodTurn{},
		&models.TodChallenge{},
		&models.TodPlayerStats{},
		&models.TodJudgmentLog{},
		&models.TodActionLog{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}

func SeedQuestions(db *gorm.DB) error {
	logger.Info("Checking for test questions...")

	// Seed Truth questions if none exist
	var truthCount int64
	db.Model(&models.Question{}).Where("question_type = ?", "truth").Count(&truthCount)
	if truthCount == 0 {
		logger.Info("Seeding Truth questions...")
		questions := []models.Question{
			{QuestionText: "آخرین باری که دروغ گفتی کی بود و چرا؟", QuestionType: "truth", Category: "truth_normal_boy", Difficulty: "easy", Points: 10, Options: "[]"},
			{QuestionText: "بزرگترین آرزوی زندگی‌ات چیست؟", QuestionType: "truth", Category: "truth_normal_girl", Difficulty: "easy", Points: 10, Options: "[]"},
			{QuestionText: "اولین کراش زندگی‌ات چه کسی بود؟", QuestionType: "truth", Category: "truth_sexy_boy", Difficulty: "medium", Points: 15, Options: "[]"},
			{QuestionText: "اگر تنها بودیم، دوست داشتی چی بهم بگی؟", QuestionType: "truth", Category: "truth_sexy_girl", Difficulty: "medium", Points: 15, Options: "[]"},
		}
		db.Create(&questions)
	}

	// Seed Dare questions if none exist
	var dareCount int64
	db.Model(&models.Question{}).Where("question_type = ?", "dare").Count(&dareCount)
	if dareCount == 0 {
		logger.Info("Seeding Dare questions...")
		questions := []models.Question{
			{QuestionText: "یک ویس ضبط کن و بگو 'من خیلی خوشگلم' و بفرست.", QuestionType: "dare", Category: "dare_normal_boy", Difficulty: "easy", Points: 20, Options: "[]"},
			{QuestionText: "یک عکس از محیط اطرافت بگیر و بفرست.", QuestionType: "dare", Category: "dare_normal_girl", Difficulty: "easy", Points: 15, Options: "[]"},
			{QuestionText: "یک عکس سلفی بگیر و برای طرف مقابل بفرست.", QuestionType: "dare", Category: "dare_sexy_boy", Difficulty: "hard", Points: 30, Options: "[]"},
			{QuestionText: "نام اولین شخصی که بهش علاقمند شدی رو بگو.", QuestionType: "dare", Category: "dare_sexy_girl", Difficulty: "hard", Points: 25, Options: "[]"},
		}
		db.Create(&questions)
	}

	// Seed Quiz questions if they don't cover a full session (5)
	var quizCount int64
	db.Model(&models.Question{}).Where("question_type = ?", "quiz").Count(&quizCount)
	if quizCount < 5 {
		logger.Info("Seeding test Quiz questions...")
		questions := []models.Question{
			{
				QuestionText:  "پایتخت فرانسه کجاست؟",
				QuestionType:  "quiz",
				Category:      "جغرافیا",
				Difficulty:    "easy",
				CorrectAnswer: "پاریس",
				Options:       `["پاریس", "لندن", "برلین", "رم"]`,
				Points:        10,
			},
			{
				QuestionText:  "کدام سیاره به سیاره سرخ معروف است؟",
				QuestionType:  "quiz",
				Category:      "علمی",
				Difficulty:    "easy",
				CorrectAnswer: "مریخ",
				Options:       `["زمین", "مریخ", "مشتری", "زهره"]`,
				Points:        10,
			},
			{
				QuestionText:  "بزرگترین اقیانوس جهان کدام است؟",
				QuestionType:  "quiz",
				Category:      "جغرافیا",
				Difficulty:    "easy",
				CorrectAnswer: "آرام",
				Options:       `["اطلس", "هند", "آرام", "منجمد شمالی"]`,
				Points:        10,
			},
			{
				QuestionText:  "مخترع تلفن چه کسی بود؟",
				QuestionType:  "quiz",
				Category:      "تاریخ",
				Difficulty:    "medium",
				CorrectAnswer: "الکساندر گراهام بل",
				Options:       `["توماس ادیسون", "الکساندر گراهام بل", "نیکولا تسلا", "آیزاک نیوتن"]`,
				Points:        15,
			},
			{
				QuestionText:  "واحد پول ژاپن چیست؟",
				QuestionType:  "quiz",
				Category:      "اقتصاد",
				Difficulty:    "easy",
				CorrectAnswer: "ین",
				Options:       `["یوان", "وون", "ین", "رینگیت"]`,
				Points:        10,
			},
		}

		for _, q := range questions {
			var existing models.Question
			if err := db.Where("question_text = ? AND question_type = ?", q.QuestionText, q.QuestionType).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					db.Create(&q)
				}
			}
		}
	}

	return nil
}

func SeedTodChallenges(db *gorm.DB) error {
	logger.Info("Checking for Truth or Dare challenges...")

	var count int64
	db.Model(&models.TodChallenge{}).Count(&count)
	if count > 0 {
		return nil
	}

	logger.Info("Seeding initial Truth or Dare challenges...")
	challenges := []models.TodChallenge{
		// Truth Questions
		{Type: "truth", Text: "آخرین باری که دروغ گفتی کی بود و چرا؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "stranger", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "بدترین خاطره‌ای که از مدرسه داری چیه؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "friend", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "خجالت‌آورترین اتفاقی که برات افتاده چی بوده؟", Difficulty: "easy", Category: "embarrassing", GenderTarget: "all", RelationLevel: "friend", ProofType: "text", XPReward: 18, CoinReward: 12},
		{Type: "truth", Text: "بزرگترین ترست که داری چیه؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "friend", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "بزرگترین رازی که از کسی پنهون کردی چیه؟", Difficulty: "hard", Category: "romantic", GenderTarget: "all", RelationLevel: "close", ProofType: "text", XPReward: 30, CoinReward: 25},

		// Dare Challenges
		{Type: "dare", Text: "یه ویس بفرست و بگو: من سلطان تنبلیام!", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "stranger", ProofType: "voice", XPReward: 20, CoinReward: 15},
		{Type: "dare", Text: "یه سلفی خنده‌دار از خودت بگیر و بفرست", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "friend", ProofType: "image", XPReward: 22, CoinReward: 18},
		{Type: "dare", Text: "یه ویس بفرست و مثل گربه میو کن!", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "stranger", ProofType: "voice", XPReward: 20, CoinReward: 15},
		{Type: "dare", Text: "یه ویدیو کوتاه از خودت برقص و بفرست", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "close", ProofType: "video", XPReward: 30, CoinReward: 25},
		{Type: "dare", Text: "یه عکس سلفی با یه حالت عجیب بگیر", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "friend", ProofType: "image", XPReward: 25, CoinReward: 20},
	}

	return db.Create(&challenges).Error
}
