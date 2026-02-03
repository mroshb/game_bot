package models

import (
	"time"
)

type Village struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	CreatorID   uint      `gorm:"not null"`
	Creator     User      `gorm:"foreignKey:CreatorID"`
	Description string    `gorm:"type:text"`
	XP          int64     `gorm:"default:0;not null"`
	Level       int       `gorm:"default:1;not null"`
	Score       int64     `gorm:"default:0;not null"` // For Ranking
	MemberCount int       `gorm:"default:1;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

type VillageMember struct {
	ID        uint      `gorm:"primaryKey"`
	VillageID uint      `gorm:"not null;index:idx_village_member"`
	UserID    uint      `gorm:"not null;index:idx_village_member"`
	Role      string    `gorm:"type:varchar(20);default:'member'"` // leader, elder, member
	JoinedAt  time.Time `gorm:"autoCreateTime"`
	Village   Village   `gorm:"foreignKey:VillageID"`
	User      User      `gorm:"foreignKey:UserID"`
}

const (
	VillageRoleLeader = "leader"
	VillageRoleElder  = "elder"
	VillageRoleMember = "member"
)

func (Village) TableName() string {
	return "villages"
}

func (VillageMember) TableName() string {
	return "village_members"
}
