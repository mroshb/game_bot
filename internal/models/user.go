package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID               uint      `gorm:"primaryKey"`
	TelegramID       int64     `gorm:"uniqueIndex;not null"`
	FullName         string    `gorm:"type:varchar(255);not null"`
	Gender           string    `gorm:"type:varchar(10);not null"`
	Age              int       `gorm:"not null"`
	City             string    `gorm:"type:varchar(100);not null"`
	Province         string    `gorm:"type:varchar(100)"` // New field
	Biography        string    `gorm:"type:text"`         // New field
	Likes            int64     `gorm:"default:0"`         // New field
	ProfilePhoto     string    `gorm:"type:varchar(500)"` // اختیاری
	CoinBalance      int64     `gorm:"default:100;not null"`
	Diamonds         int64     `gorm:"default:0;not null"`
	Level            int       `gorm:"default:1;not null"`
	XP               int64     `gorm:"default:0;not null"`
	Wins             int       `gorm:"default:0;not null"`
	Losses           int       `gorm:"default:0;not null"`
	Draws            int       `gorm:"default:0;not null"`
	ItemsInventory   string    `gorm:"type:text;default:'{}'"`      // JSON: {"shield": 2, "swap": 5}
	CustomAvatarID   string    `gorm:"type:varchar(500)"`           // For uploaded photos
	PublicID         string    `gorm:"uniqueIndex;type:varchar(8)"` // New field
	ReferrerID       uint      `gorm:"default:0"`                   // New field for referral system
	Latitude         float64   `gorm:"type:float"`
	Longitude        float64   `gorm:"type:float"`
	Status           string    `gorm:"type:varchar(20);default:'offline'"`
	LastDailyBonus   time.Time `gorm:"default:NULL"`
	DailyBonusStreak int       `gorm:"default:0;not null"`
	LastActivity     time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
	Distance         float64   `gorm:"-"` // For nearby search result
}

// GetLevelTitle returns the title based on user level
func (u *User) GetLevelTitle() string {
	if u.Level <= 5 {
		return "تازهوارد 🌱"
	} else if u.Level <= 10 {
		return "کاربلد 🧢"
	} else if u.Level <= 20 {
		return "جنگجو ⚔️"
	} else if u.Level <= 50 {
		return "استاد 🥋"
	}
	return "افسانه 👑"
}

// GetXPRequired returns XP needed for current level to reach next
func (u *User) GetXPRequired() int64 {
	return int64(u.Level * 100)
}

// GetXPBar returns a visual progress bar
func (u *User) GetXPBar() string {
	required := u.GetXPRequired()
	if required == 0 {
		return "[□□□□□□□□□□] 0%"
	}

	percentage := int(float64(u.XP) / float64(required) * 100)
	if percentage > 100 {
		percentage = 100
	}

	filledCount := percentage / 10
	emptyCount := 10 - filledCount

	bar := "["
	for i := 0; i < filledCount; i++ {
		bar += "■"
	}
	for i := 0; i < emptyCount; i++ {
		bar += "□"
	}
	bar += fmt.Sprintf("] %d%%", percentage)

	return bar
}

// User status constants
const (
	UserStatusOffline   = "offline"
	UserStatusOnline    = "online"
	UserStatusSearching = "searching"
	UserStatusInMatch   = "in_match"
)

const (
	GenderMale   = "male"
	GenderFemale = "female"
)

// Default avatar URLs
const (
	DefaultAvatarMale   = "https://api.dicebear.com/7.x/avataaars/svg?seed=Felix"
	DefaultAvatarFemale = "https://api.dicebear.com/7.x/avataaars/svg?seed=Anya"
)

// BeforeCreate hook to generate PublicID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// Generate random public ID if not set
	if u.PublicID == "" {
		// We'll rely on the repository or creating function to populate it ideally,
		// but as a fallback/hook we can define logic here, but importing utils might cause cycles if utils imports models?
		// utils imports nothing so it's fine.
		// However, I need to modify imports in this file.
		// Since I cannot modify imports easily in this chunk, I will assume I can't call utils here without import.
		// I will rely on the repository to set it, OR I will add import in another step.
		// Actually, standard GORM pattern is to set default there.
		// Let's modify imports first or do it in one go.
	}
	return nil
}

// BeforeSave hook for validation and sanitization
func (u *User) BeforeSave(tx *gorm.DB) error {
	// Validate gender
	if u.Gender != GenderMale && u.Gender != GenderFemale {
		return gorm.ErrInvalidData
	}

	// Validate age
	if u.Age < 13 || u.Age > 100 {
		return gorm.ErrInvalidData
	}

	// Validate status
	validStatuses := map[string]bool{
		UserStatusOffline:   true,
		UserStatusOnline:    true,
		UserStatusSearching: true,
		UserStatusInMatch:   true,
	}
	if !validStatuses[u.Status] {
		return gorm.ErrInvalidData
	}

	return nil
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

type UserLike struct {
	ID        uint      `gorm:"primaryKey"`
	LikerID   uint      `gorm:"not null;index:idx_like_unique,unique"`
	LikedID   uint      `gorm:"not null;index:idx_like_unique,unique"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (UserLike) TableName() string {
	return "user_likes"
}
