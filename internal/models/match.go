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
	StartedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP"`
	EndedAt   *time.Time `gorm:"index"`
	TimeoutAt time.Time  `gorm:"index"`
	Status    string     `gorm:"type:varchar(20);default:'active'"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
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

type MatchmakingQueue struct {
	ID              uint      `gorm:"primaryKey"`
	UserID          uint      `gorm:"uniqueIndex;not null"`
	User            User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	RequestedGender string    `gorm:"type:varchar(10)"`
	MinAge          *int      `gorm:""`
	MaxAge          *int      `gorm:""`
	City            string    `gorm:"type:varchar(100);index"`
	CoinsPaid       int64     `gorm:"default:5"`
	CreatedAt       time.Time `gorm:"autoCreateTime;index"`
}

// Requested gender constants
const (
	RequestedGenderMale   = "male"
	RequestedGenderFemale = "female"
	RequestedGenderAny    = "any"
)

func (MatchmakingQueue) TableName() string {
	return "matchmaking_queue"
}

// MatchFilters for searching
type MatchFilters struct {
	Gender string
	MinAge *int
	MaxAge *int
	City   string
}
