package models

import (
	"time"

	"github.com/mroshb/game_bot/internal/security"
	"gorm.io/gorm"
)

type Room struct {
	ID         uint      `gorm:"primaryKey"`
	RoomName   string    `gorm:"type:varchar(255);not null"`
	HostID     uint      `gorm:"not null;index"`
	Host       User      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE"`
	RoomType   string    `gorm:"type:varchar(20);not null;index"`
	InviteCode string    `gorm:"type:varchar(10);uniqueIndex"`
	MaxPlayers int       `gorm:"default:10"`
	EntryFee   int64     `gorm:"default:0"`
	Status     string    `gorm:"type:varchar(20);default:'waiting';index"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

// Room type constants
const (
	RoomTypePublic  = "public"
	RoomTypePrivate = "private"
)

// Room status constants
const (
	RoomStatusWaiting    = "waiting"
	RoomStatusInProgress = "in_progress"
	RoomStatusFinished   = "finished"
	RoomStatusClosed     = "closed"
)

// BeforeCreate hook to generate secure invite code for private rooms
func (r *Room) BeforeCreate(tx *gorm.DB) error {
	if r.InviteCode == "" {
		r.InviteCode = security.GenerateSecureCode(8)
	}
	return nil
}

func (Room) TableName() string {
	return "rooms"
}

type RoomMember struct {
	ID       uint      `gorm:"primaryKey"`
	RoomID   uint      `gorm:"not null;index:idx_room_member,unique"`
	Room     Room      `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE"`
	UserID   uint      `gorm:"not null;index:idx_room_member,unique"`
	User     User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	JoinedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	IsKicked bool      `gorm:"default:false"`
}

func (RoomMember) TableName() string {
	return "room_members"
}
