package models

import (
	"time"
)

type CoinTransaction struct {
	ID              uint      `gorm:"primaryKey"`
	UserID          uint      `gorm:"not null;index"`
	User            User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount          int64     `gorm:"not null"`
	TransactionType string    `gorm:"type:varchar(50);not null;index"`
	Description     string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"autoCreateTime;index"`
}

// Transaction type constants
const (
	TxTypeMatchmaking     = "matchmaking"
	TxTypeMatchRefund     = "match_refund"
	TxTypeMessage         = "message"
	TxTypeFriendRequest   = "friend_request"
	TxTypeGameReward      = "game_reward"
	TxTypeRoomEntry       = "room_entry"
	TxTypeRoomCreation    = "room_creation"
	TxTypeRefund          = "refund"
	TxTypeAdminAdjustment = "admin_adjustment"
	TxTypeDailyBonus      = "daily_bonus"
	TxTypeReferralReward  = "referral_reward"
	TxTypeWelcomeBonus    = "welcome_bonus"
	TxTypePenalty         = "penalty"
)

func (CoinTransaction) TableName() string {
	return "coin_transactions"
}
