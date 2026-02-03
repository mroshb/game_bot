package models

import (
	"time"
)

type Friendship struct {
	ID          uint      `gorm:"primaryKey"`
	RequesterID uint      `gorm:"not null;index:idx_friendship,unique"`
	Requester   User      `gorm:"foreignKey:RequesterID;constraint:OnDelete:CASCADE"`
	AddresseeID uint      `gorm:"not null;index:idx_friendship,unique"`
	Addressee   User      `gorm:"foreignKey:AddresseeID;constraint:OnDelete:CASCADE"`
	Status      string    `gorm:"type:varchar(20);default:'pending'"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// Friendship status constants
const (
	FriendshipStatusPending  = "pending"
	FriendshipStatusAccepted = "accepted"
	FriendshipStatusRejected = "rejected"
)

func (Friendship) TableName() string {
	return "friendships"
}
