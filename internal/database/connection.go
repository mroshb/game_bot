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
		logLevel = gormlogger.Warn
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
	logger.Info("Starting database migrations...")

	// Save original log level and set to Silent during migration to avoid wall of text
	// This makes the startup much cleaner and prevents log spam.
	originalLogger := db.Config.Logger
	db.Config.Logger = gormlogger.Default.LogMode(gormlogger.Silent)
	defer func() { db.Config.Logger = originalLogger }()

	// Manually drop phone_hash if it exists (as user requested its removal)
	if db.Migrator().HasColumn(&models.User{}, "phone_hash") {
		logger.Info("Maintenance: Dropping phone_hash column from users table...")
		if err := db.Migrator().DropColumn(&models.User{}, "phone_hash"); err != nil {
			logger.Warn("Failed to drop phone_hash column", "error", err)
		}
	}

	// Manually ensure matchmaking_queue has game_type column with proper default
	// This prevents GORM from potentially looping on this field if metadata detection glitches
	if !db.Migrator().HasColumn(&models.MatchmakingQueue{}, "game_type") {
		logger.Info("Maintenance: Adding game_type column to matchmaking_queue...")
		if err := db.Exec(`ALTER TABLE matchmaking_queue ADD COLUMN IF NOT EXISTS game_type varchar(20) DEFAULT 'chat'`).Error; err != nil {
			logger.Warn("Failed to manually add game_type column", "error", err)
		}
		db.Exec(`CREATE INDEX IF NOT EXISTS idx_matchmaking_queue_game_type ON matchmaking_queue (game_type)`)
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
		&models.QuizMatch{},
		&models.QuizRound{},
		&models.QuizAnswer{},
		&models.UserBooster{},
		&models.VillageWar{},
		&models.UserLike{},
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

	challenges := []models.TodChallenge{
		// --- TRUTH: FUNNY (EASY) ---
		{Type: "truth", Text: "آخرین باری که ضایع شدی کی بود؟ تعریف کن.", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "اگه مجبور باشی اسم خودت رو عوض کنی، چی میذاری؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "توی حموم آواز میخونی؟ اگه آره، چه آهنگی؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "چه غذایی رو حاضری تا آخر عمرت هر روز بخوری؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "مسخره‌ترین خوابی که دیدی چی بوده؟", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},

		// --- TRUTH: EMBARRASSING (MEDIUM) ---
		{Type: "truth", Text: "بدترین نمره‌ای که توی مدرسه گرفتی چند بوده؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "آخرین باری که گریه کردی کی بود و چرا؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "تا حالا سعی کردی کسی رو تحت تاثیر قرار بدی و خرابکاری کنی؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "اگه بتونی یک ویژگی ظاهریت رو عوض کنی، اون چیه؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "تا حالا جلوی آینه با خودت حرف زدی؟ چی گفتی؟", Difficulty: "medium", Category: "embarrassing", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},

		// --- TRUTH: DEEP/PERSONAL (HARD) ---
		{Type: "truth", Text: "بزرگترین حسرت زندگیت چیه؟", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "truth", Text: "اگه بدونی فردا دنیا تموم میشه، امروز چیکار میکنی؟", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "truth", Text: "یک رازی که تا حالا به هیچکس نگفتی رو بگو.", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 35, CoinReward: 30},
		{Type: "truth", Text: "اگه بتونی به گذشته برگردی، چه چیزی رو تغییر میدی؟", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "truth", Text: "چه چیزی بیشتر از همه تو رو میترسونه؟", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "truth", Text: "آیا تا حالا به کسی خیانت کردی؟ (عاطفی یا ...) تعریف کن.", Difficulty: "hard", Category: "deep", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 40, CoinReward: 35},
		{Type: "truth", Text: "بدترین فانتزی‌ای که در مورد کسی داشتی چی بوده؟", Difficulty: "hard", Category: "hot", GenderTarget: "all", RelationLevel: "close", ProofType: "text", XPReward: 45, CoinReward: 40},

		// --- TRUTH: FLIRTY/ROMANTIC (VARIES) ---
		{Type: "truth", Text: "جذاب‌ترین ویژگی جنس مخالف از نظرت چیه؟", Difficulty: "medium", Category: "romantic", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 25, CoinReward: 20},
		{Type: "truth", Text: "اولین کراشت کی بود؟ (اسم نبر، توصیف کن)", Difficulty: "medium", Category: "romantic", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 25, CoinReward: 20},
		{Type: "truth", Text: "به نظرت عشق در نگاه اول وجود داره؟", Difficulty: "easy", Category: "romantic", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 15, CoinReward: 10},
		{Type: "truth", Text: "بدترین دیت (قرار) عاشقانه‌ای که رفتی چطور بود؟", Difficulty: "hard", Category: "romantic", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "truth", Text: "اگه بخوای مخ کسی رو بزنی، جمله‌ی اولت چیه؟", Difficulty: "medium", Category: "romantic", GenderTarget: "all", RelationLevel: "all", ProofType: "text", XPReward: 25, CoinReward: 20},

		// --- DARE: FUNNY (EASY) ---
		{Type: "dare", Text: "یه ویس ۱ دقیقه بفرست و فقط صدای حیوانات دربیار!", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "voice", XPReward: 20, CoinReward: 15},
		{Type: "dare", Text: "یه سلفی با قیافه کج و کوله (شکلک) بگیر بفرست.", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "friend", ProofType: "image", XPReward: 22, CoinReward: 18},
		{Type: "dare", Text: "یه جوک خیلی بی‌مزه تعریف کن (ویس).", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "voice", XPReward: 20, CoinReward: 15},
		{Type: "dare", Text: "با دست مخالف اسمت رو روی کاغذ بنویس و عکس بفرست.", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 22, CoinReward: 18},
		{Type: "dare", Text: "یه شعر کودکانه (مثل یه توپ دارم قلقلیه) رو با صدای اپرا بخون!", Difficulty: "easy", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "voice", XPReward: 25, CoinReward: 20},

		// --- DARE: PHYSICAL/ACTIVE (MEDIUM) ---
		{Type: "dare", Text: "۱۰ تا شنا برو و ویدیو بگیر.", Difficulty: "medium", Category: "physical", GenderTarget: "all", RelationLevel: "all", ProofType: "video", XPReward: 30, CoinReward: 25},
		{Type: "dare", Text: "یه لیوان آب رو یک نفس سر بکش (ویدیو).", Difficulty: "medium", Category: "physical", GenderTarget: "all", RelationLevel: "all", ProofType: "video", XPReward: 30, CoinReward: 25},
		{Type: "dare", Text: "یه دقیقه پلانک نگه دار (ویدیو).", Difficulty: "medium", Category: "physical", GenderTarget: "all", RelationLevel: "all", ProofType: "video", XPReward: 35, CoinReward: 30},
		{Type: "dare", Text: "با چشم بسته یه نقاشی بکش و عکسش رو بفرست.", Difficulty: "medium", Category: "physical", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 25, CoinReward: 20},
		{Type: "dare", Text: "یه قاشق سس تند (یا چیز بد مزه) بخور!", Difficulty: "medium", Category: "physical", GenderTarget: "all", RelationLevel: "all", ProofType: "video", XPReward: 35, CoinReward: 30},

		// --- DARE: SOCIAL/BRAVE (HARD) ---
		{Type: "dare", Text: "به آخرین نفری که بهت پیام داده زنگ بزن و فقط میو کن! (اسکرین شات)", Difficulty: "hard", Category: "social", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 40, CoinReward: 35},
		{Type: "dare", Text: "یه استوری بذار و بنویس 'من عاشق شلغمم' (اسکرین شات).", Difficulty: "hard", Category: "social", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 45, CoinReward: 40},
		{Type: "dare", Text: "به دوست صمیمیت پیام بده و الکی بگو 'من دارم ازدواج میکنم' (اسکرین شات).", Difficulty: "hard", Category: "social", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 50, CoinReward: 45},
		{Type: "dare", Text: "عکس پروفایلت رو به عکس یه سیب‌زمینی تغییر بده (اسکرین).", Difficulty: "hard", Category: "social", GenderTarget: "all", RelationLevel: "all", ProofType: "image", XPReward: 40, CoinReward: 35},
		{Type: "dare", Text: "یه آواز بلند بخون و ویس بگیر (بدون خجالت!).", Difficulty: "hard", Category: "social", GenderTarget: "all", RelationLevel: "all", ProofType: "voice", XPReward: 35, CoinReward: 30},
		{Type: "dare", Text: "یه ویدیو ۱۰ ثانیه‌ای بفرست که داری مثل میمون بالا پایین میپری!", Difficulty: "hard", Category: "funny", GenderTarget: "all", RelationLevel: "all", ProofType: "video", XPReward: 50, CoinReward: 40},
		{Type: "dare", Text: "یه اسکرین‌شات از هیستوری مرورگرت (۳ تای آخر) بفرست.", Difficulty: "hard", Category: "embarrassing", GenderTarget: "all", RelationLevel: "close", ProofType: "image", XPReward: 45, CoinReward: 35},

		// --- GENDER SPECIFIC ---
		{Type: "truth", Text: "اگه میتونستی یه روز جای یه مرد باشی، اولین کاری که میکردی چی بود؟", Difficulty: "medium", Category: "funny", GenderTarget: "female", RelationLevel: "all", ProofType: "text", XPReward: 20, CoinReward: 15},
		{Type: "truth", Text: "بزرگترین دغدغه‌ی دختر بودن از نظرت چیه؟", Difficulty: "hard", Category: "deep", GenderTarget: "female", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
		{Type: "dare", Text: "یه ویس بفرست و با صدای نازک (دخترونه) یه جمله قصار بگو!", Difficulty: "medium", Category: "funny", GenderTarget: "male", RelationLevel: "all", ProofType: "voice", XPReward: 25, CoinReward: 20},
		{Type: "dare", Text: "ادای یه خانم که داره آرایش میکنه رو دربیار و ویدیو بگیر.", Difficulty: "hard", Category: "funny", GenderTarget: "male", RelationLevel: "all", ProofType: "video", XPReward: 40, CoinReward: 30},
		{Type: "truth", Text: "سخت‌ترین قسمت مرد بودن چیه؟", Difficulty: "hard", Category: "deep", GenderTarget: "male", RelationLevel: "all", ProofType: "text", XPReward: 30, CoinReward: 25},
	}

	for _, c := range challenges {
		var existing models.TodChallenge
		// Check by text and type to avoid duplicates
		if err := db.Where("text = ? AND type = ?", c.Text, c.Type).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&c).Error; err != nil {
					logger.Error("Failed to seed ToD challenge", "text", c.Text, "error", err)
				}
			}
		}
	}

	return nil
}
