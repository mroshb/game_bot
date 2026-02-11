package models

import (
	"time"
)

type VillageWar struct {
	ID            uint      `gorm:"primaryKey"`
	Village1ID    uint      `gorm:"not null;index"`
	Village2ID    uint      `gorm:"not null;index"`
	Village1Score int       `gorm:"default:0;not null"`
	Village2Score int       `gorm:"default:0;not null"`
	Status        string    `gorm:"type:varchar(20);default:'active';index"` // active, finished, cancelled
	StartTime     time.Time `gorm:"not null"`
	EndTime       time.Time `gorm:"not null;index"`
	WinnerID      *uint     `gorm:"index"`
	Village1      Village   `gorm:"foreignKey:Village1ID"`
	Village2      Village   `gorm:"foreignKey:Village2ID"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

const (
	WarStatusActive    = "active"
	WarStatusFinished  = "finished"
	WarStatusCancelled = "cancelled"
)

func (VillageWar) TableName() string {
	return "village_wars"
}
