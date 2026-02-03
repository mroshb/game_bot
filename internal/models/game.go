package models

import (
	"time"
)

type Question struct {
	ID            uint      `gorm:"primaryKey"`
	QuestionText  string    `gorm:"type:text;not null"`
	QuestionType  string    `gorm:"type:varchar(20);not null;index"`
	Category      string    `gorm:"type:varchar(50);index"`
	Difficulty    string    `gorm:"type:varchar(20);index"`
	CorrectAnswer string    `gorm:"type:text"`
	Options       string    `gorm:"type:jsonb"` // JSON string for PostgreSQL
	Points        int       `gorm:"default:10"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

// Question type constants
const (
	QuestionTypeTruth = "truth"
	QuestionTypeDare  = "dare"
	QuestionTypeQuiz  = "quiz"
)

// Difficulty constants
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

func (Question) TableName() string {
	return "questions"
}

type GameSession struct {
	ID                uint      `gorm:"primaryKey"`
	RoomID            uint      `gorm:"not null;index"`
	Room              Room      `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE"`
	GameType          string    `gorm:"type:varchar(20);not null"`
	CurrentQuestionID *uint     `gorm:"index"`
	CurrentQuestion   *Question `gorm:"foreignKey:CurrentQuestionID"`
	Status            string    `gorm:"type:varchar(20);default:'waiting'"`
	TurnUserID        uint      `gorm:"index"` // User whose turn it is
	StartedAt         *time.Time
	EndedAt           *time.Time
	CreatedAt         time.Time `gorm:"autoCreateTime"`
}

// Game type constants
const (
	GameTypeTruthDare = "truth_dare"
	GameTypeQuiz      = "quiz"
)

// Game status constants
const (
	GameStatusWaiting          = "waiting"
	GameStatusInProgress       = "in_progress"
	GameStatusFinished         = "finished"
	GameStatusWaitingForChoice = "waiting_choice"
	GameStatusWaitingForHost   = "waiting_host"
)

func (GameSession) TableName() string {
	return "game_sessions"
}

type GameParticipant struct {
	ID            uint        `gorm:"primaryKey"`
	GameSessionID uint        `gorm:"not null;index"`
	GameSession   GameSession `gorm:"foreignKey:GameSessionID;constraint:OnDelete:CASCADE"`
	UserID        uint        `gorm:"not null;index"`
	User          User        `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Score         int         `gorm:"default:0"`
	TurnOrder     int         `gorm:"default:0"`  // Order of players in group game
	Answers       string      `gorm:"type:jsonb"` // JSON string
	CreatedAt     time.Time   `gorm:"autoCreateTime"`
}

func (GameParticipant) TableName() string {
	return "game_participants"
}
