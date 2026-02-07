package models

import (
	"time"
)

// QuizMatch represents a 1v1 quiz game between two users
type QuizMatch struct {
	ID                uint       `gorm:"primaryKey"`
	User1ID           uint       `gorm:"not null;index"`
	User1             User       `gorm:"foreignKey:User1ID;constraint:OnDelete:CASCADE"`
	User2ID           uint       `gorm:"not null;index"`
	User2             User       `gorm:"foreignKey:User2ID;constraint:OnDelete:CASCADE"`
	CurrentRound      int        `gorm:"default:1"`
	CurrentQuestion   int        `gorm:"default:0"`
	State             string     `gorm:"type:varchar(50);default:'waiting_category';index"`
	TurnUserID        *uint      `gorm:"index"`
	User1TotalCorrect int        `gorm:"default:0"`
	User2TotalCorrect int        `gorm:"default:0"`
	User1TotalTimeMs  int64      `gorm:"default:0"`
	User2TotalTimeMs  int64      `gorm:"default:0"`
	User1LightsMsgID  int        `gorm:"default:0"`
	User2LightsMsgID  int        `gorm:"default:0"`
	WinnerID          *uint      `gorm:"index"`
	StartedAt         time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	LastActivityAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP;index"`
	FinishedAt        *time.Time `gorm:"index"`
	TimeoutAt         time.Time  `gorm:"index"`
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`
}

func (QuizMatch) TableName() string {
	return "quiz_matches"
}

// QuizRound represents a single round in a quiz match
type QuizRound struct {
	ID                uint      `gorm:"primaryKey"`
	MatchID           uint      `gorm:"not null;index"`
	Match             QuizMatch `gorm:"foreignKey:MatchID;constraint:OnDelete:CASCADE"`
	RoundNumber       int       `gorm:"not null"`
	Category          string    `gorm:"type:varchar(100);not null"`
	ChosenByUserID    uint      `gorm:"not null"`
	User1CorrectCount int       `gorm:"default:0"`
	User2CorrectCount int       `gorm:"default:0"`
	User1TimeMs       int       `gorm:"default:0"`
	User2TimeMs       int       `gorm:"default:0"`
	QuestionIDs       string    `gorm:"type:text"` // Comma separated list of question IDs
	CreatedAt         time.Time `gorm:"autoCreateTime"`
}

func (QuizRound) TableName() string {
	return "quiz_rounds"
}

// QuizAnswer represents a user's answer to a question
type QuizAnswer struct {
	ID             uint      `gorm:"primaryKey"`
	MatchID        uint      `gorm:"not null;index"`
	Match          QuizMatch `gorm:"foreignKey:MatchID;constraint:OnDelete:CASCADE"`
	RoundID        uint      `gorm:"not null;index"`
	Round          QuizRound `gorm:"foreignKey:RoundID;constraint:OnDelete:CASCADE"`
	UserID         uint      `gorm:"not null;index"`
	User           User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	QuestionID     uint      `gorm:"not null"`
	Question       Question  `gorm:"foreignKey:QuestionID"`
	QuestionNumber int       `gorm:"not null"`
	AnswerIndex    *int
	IsCorrect      bool      `gorm:"default:false"`
	TimeTakenMs    int       `gorm:"not null"`
	BoosterUsed    string    `gorm:"type:varchar(50)"`
	AnsweredAt     time.Time `gorm:"autoCreateTime"`
}

func (QuizAnswer) TableName() string {
	return "quiz_answers"
}

// UserBooster represents boosters owned by a user
type UserBooster struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_user_booster_type"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	BoosterType string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_booster_type"`
	Quantity    int       `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (UserBooster) TableName() string {
	return "user_boosters"
}

// Quiz match states
const (
	QuizStateWaitingCategory  = "waiting_category"
	QuizStateCategorySelected = "category_selected"
	QuizStatePlayingQ1        = "playing_q1"
	QuizStatePlayingQ2        = "playing_q2"
	QuizStatePlayingQ3        = "playing_q3"
	QuizStatePlayingQ4        = "playing_q4"
	QuizStateRoundFinished    = "round_finished"
	QuizStateGameFinished     = "finished"
	QuizStateTimeout          = "timeout"
)

// Booster types
const (
	BoosterRemove2Options = "remove_2_options"
	BoosterSecondChance   = "second_chance"
)

// Booster costs (in coins)
const (
	BoosterRemove2OptionsCost = 40
	BoosterSecondChanceCost   = 40
)

// Game configuration
const (
	QuizTotalRounds         = 4
	QuizQuestionsPerRound   = 4
	QuizQuestionTimeSeconds = 10
	QuizCategoryTimeSeconds = 30
	QuizTimeoutDays         = 3
)

// Rewards
const (
	QuizWinRewardCoins  = 100
	QuizWinRewardXP     = 30
	QuizLoseRewardXP    = 10
	QuizDrawRewardCoins = 50
	QuizDrawRewardXP    = 20
)
