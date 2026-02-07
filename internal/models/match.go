package models

import (
	"time"
)

type MatchSession struct {
	ID        uint       `gorm:"primaryKey"`
	User1ID   uint       `gorm:"not null;index"`
	User1     User       `gorm:"foreignKey:User1ID;constraint:OnDelete:CASCADE"`
	User2ID   uint       `gorm:"not null;index"`
	User2     User       `gorm:"foreignKey:User2ID;constraint:OnDelete:CASCADE"`
	StartedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP;index"`
	EndedAt   *time.Time `gorm:"index"`
	TimeoutAt time.Time  `gorm:"index"`
	Status    string     `gorm:"type:varchar(20);default:'active';index"`
	CreatedAt time.Time  `gorm:"autoCreateTime;index"`
}

// Match status constants
const (
	MatchStatusActive  = "active"
	MatchStatusEnded   = "ended"
	MatchStatusTimeout = "timeout"
)

func (MatchSession) TableName() string {
	return "match_sessions"
}

// Match is an alias for MatchSession for compatibility
type Match = MatchSession

type MatchmakingQueue struct {
	ID              uint      `gorm:"primaryKey"`
	UserID          uint      `gorm:"uniqueIndex;not null"`
	User            User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RequestedGender string    `gorm:"type:varchar(10);index"`
	MinAge          *int      `gorm:"index"`
	MaxAge          *int      `gorm:"index"`
	City            string    `gorm:"type:varchar(100);index"`
	TargetProvinces string    `gorm:"type:text"`                             // Comma separated list of provinces
	GameType        string    `gorm:"type:varchar(20);default:'chat';index"` // chat, quiz, tod
	CoinsPaid       int64     `gorm:"default:5;index"`
	CreatedAt       time.Time `gorm:"autoCreateTime;index"`
}

// Requested gender constants
const (
	RequestedGenderMale   = "male"
	RequestedGenderFemale = "female"
	RequestedGenderAny    = "any"
)

// Game type constants for matchmaking
const (
	GameTypeChat = "chat"
	GameTypeTod  = "tod"
	// GameTypeQuiz is defined in game.go
)

func (MatchmakingQueue) TableName() string {
	return "matchmaking_queue"
}

// MatchFilters for searching
type MatchFilters struct {
	Gender    string
	MinAge    *int
	MaxAge    *int
	City      string
	Provinces []string
	GameType  string
}
