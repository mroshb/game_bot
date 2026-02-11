package services

import (
	"fmt"
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/internal/repositories"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VillageService struct {
	db       *gorm.DB
	repo     *repositories.VillageRepository
	userRepo *repositories.UserRepository
	coinRepo *repositories.CoinRepository
}

func NewVillageService(db *gorm.DB, repo *repositories.VillageRepository, userRepo *repositories.UserRepository, coinRepo *repositories.CoinRepository) *VillageService {
	return &VillageService{
		db:       db,
		repo:     repo,
		userRepo: userRepo,
		coinRepo: coinRepo,
	}
}

// WithTx returns a new instance of VillageService with the transaction/session
func (s *VillageService) WithTx(tx *gorm.DB) *VillageService {
	return &VillageService{
		db:       tx,
		repo:     s.repo.WithTx(tx),
		userRepo: s.userRepo.WithTx(tx),
		coinRepo: s.coinRepo.WithTx(tx),
	}
}

func (s *VillageService) DonateToTreasury(userID uint, amount int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		coinRepo := s.coinRepo.WithTx(tx)

		village, err := repo.GetUserVillage(userID)
		if err != nil || village == nil {
			return errors.New(errors.ErrCodeNotFound, "شما عضو هیچ دهکده‌ای نیستید")
		}

		if err := coinRepo.DeductCoins(userID, amount, models.TxTypePenalty, "اهدای سکه به خزانه دهکده"); err != nil {
			return err
		}

		return repo.AddToTreasury(village.ID, amount)
	})
}

func (s *VillageService) UpgradeBuff(userID uint, buffType string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		village, err := repo.GetUserVillage(userID)
		if err != nil || village == nil {
			return errors.New(errors.ErrCodeNotFound, "دهکده پیدا نشد")
		}

		// Only leader/elder can upgrade
		member, _ := repo.GetVillageMembers(village.ID)
		var canUpgrade bool
		var currentLevel int
		for _, m := range member {
			if m.UserID == userID && (m.Role == models.VillageRoleLeader || m.Role == models.VillageRoleElder) {
				canUpgrade = true
				break
			}
		}
		if !canUpgrade {
			return errors.New(errors.ErrCodeValidationFailed, "فقط لیدر یا بزرگان می‌توانند ارتقا دهند")
		}

		// Lock village row
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&village, village.ID).Error; err != nil {
			return err
		}

		switch buffType {
		case "xp":
			currentLevel = village.BuffXPLevel
		case "coin":
			currentLevel = village.BuffCoinLevel
		case "shield":
			currentLevel = village.BuffShieldLevel
		default:
			return errors.New(errors.ErrCodeValidationFailed, "نوع باف نامعتبر")
		}

		if currentLevel >= 5 {
			return errors.New(errors.ErrCodeValidationFailed, "این باف در بالاترین سطح خود قرار دارد")
		}

		cost := int64((currentLevel + 1) * 5000)
		if village.Treasury < cost {
			return errors.New(errors.ErrCodeInsufficientFunds, fmt.Sprintf("شارژ خزانه کافی نیست. نیاز به %d سکه", cost))
		}

		return repo.UpgradeBuff(village.ID, buffType, currentLevel+1, cost)
	})
}

func (s *VillageService) StartWar(userID uint) (*models.VillageWar, error) {
	var war *models.VillageWar
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		village, err := repo.GetUserVillage(userID)
		if err != nil || village == nil {
			return errors.New(errors.ErrCodeNotFound, "دهکده پیدا نشد")
		}

		// Lock my village
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(village, village.ID).Error; err != nil {
			return err
		}

		// Only leader can start war
		members, _ := repo.GetVillageMembers(village.ID)
		var isLeader bool
		for _, m := range members {
			if m.UserID == userID && m.Role == models.VillageRoleLeader {
				isLeader = true
				break
			}
		}
		if !isLeader {
			return errors.New(errors.ErrCodeValidationFailed, "فقط لیدر می‌تواند جنگ شروع کند")
		}

		// Check if already in war
		activeWar, _ := repo.GetActiveWar(village.ID)
		if activeWar != nil {
			return errors.New(errors.ErrCodeAlreadyExists, "دهکده شما هم‌اکنون در حال جنگ است")
		}

		// Find target village (similar score, not in war)
		opponents, _ := repo.GetVillageLeaderboard(50)
		var target *models.Village
		for _, opp := range opponents {
			if opp.ID == village.ID {
				continue
			}

			// We need a way to lock the target atomically or check if it's available
			// For simplicity within current architecture:
			oppWar, _ := repo.GetActiveWar(opp.ID)
			if oppWar == nil && opp.MemberCount >= 1 {
				// FOUND A CANDIDATE - TRY TO LOCK IT
				var lockedOpp models.Village
				err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
					Where("id = ?", opp.ID).First(&lockedOpp).Error
				if err == nil {
					target = &lockedOpp
					break
				}
			}
		}

		if target == nil {
			return errors.New(errors.ErrCodeNotFound, "حریف مناسبی پیدا نشد")
		}

		war = &models.VillageWar{
			Village1ID: village.ID,
			Village2ID: target.ID,
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(24 * time.Hour),
			Status:     models.WarStatusActive,
		}

		if err := repo.CreateWar(war); err != nil {
			return err
		}

		// Set preloaded villages for UI
		war.Village1 = *village
		war.Village2 = *target

		return nil
	})

	return war, err
}

func (s *VillageService) UpdateWarScore(userID uint, points int) error {
	village, err := s.repo.GetUserVillage(userID)
	if err != nil || village == nil {
		return nil
	}

	war, _ := s.repo.GetActiveWar(village.ID)
	if war == nil {
		return nil
	}

	// Calculate points with enemy shield reduction
	enemyID := war.Village2ID
	if village.ID == war.Village2ID {
		enemyID = war.Village1ID
	}

	enemy, _ := s.repo.GetVillageByID(enemyID)
	if enemy != nil && enemy.BuffShieldLevel > 0 {
		// Reduce points by 10% per shield level
		reduction := float64(points) * float64(enemy.BuffShieldLevel) * 0.10
		points -= int(reduction)
		if points < 1 {
			points = 1 // Minimum 1 point
		}
	}

	return s.repo.UpdateWarScore(war.ID, village.ID, points)
}

func (s *VillageService) ProcessExpiredWars() error {
	wars, err := s.repo.GetExpiredWars()
	if err != nil {
		return err
	}

	for _, war := range wars {
		var winnerID uint
		if war.Village1Score > war.Village2Score {
			winnerID = war.Village1ID
		} else if war.Village2Score > war.Village1Score {
			winnerID = war.Village2ID
		}

		if winnerID > 0 {
			// Reward winner
			s.repo.AddToTreasury(winnerID, 5000)
			s.AddXP(winnerID, 1000)
		}

		s.repo.FinishWar(war.ID, winnerID)
	}
	return nil
}

func (s *VillageService) CreateVillage(name, description string, creatorID uint) (*models.Village, error) {
	var village *models.Village
	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		// Check if user is already in a village
		existing, err := repo.GetUserVillage(creatorID)
		if err != nil {
			return err
		}
		if existing != nil {
			return errors.New(errors.ErrCodeAlreadyExists, "شما قبلاً در یک دهکده عضو هستید")
		}

		village = &models.Village{
			Name:        name,
			Description: description,
			CreatorID:   creatorID,
			MemberCount: 1,
		}

		if err := repo.CreateVillage(village, creatorID); err != nil {
			return err
		}
		return nil
	})

	return village, err
}

func (s *VillageService) AddMember(villageID, userID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		// Lock village row to prevent concurrent joins exceeding limit
		var village models.Village
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&village, villageID).Error; err != nil {
			return err
		}

		if village.MemberCount >= 50 {
			return errors.New(errors.ErrCodeValidationFailed, "ظرفیت دهکده تکمیل است (حداکثر ۵۰ نفر)")
		}

		// Check if user is already in a village
		existing, err := repo.GetUserVillage(userID)
		if err != nil {
			return err
		}
		if existing != nil {
			return errors.New(errors.ErrCodeAlreadyExists, "این کاربر قبلاً در یک دهکده عضو شده است")
		}

		return repo.AddMember(villageID, userID, models.VillageRoleMember)
	})
}

func (s *VillageService) LeaveVillage(userID uint) error {
	village, err := s.repo.GetUserVillage(userID)
	if err != nil {
		return err
	}
	if village == nil {
		return errors.New(errors.ErrCodeNotFound, "شما در هیچ دهکده‌ای عضو نیستید")
	}

	// If leader leaves, what happens? For simplicity, we might prevent or transfer leader.
	// For now, let's just allow leaving if not leader, or handle leader transfer logic later.
	members, _ := s.repo.GetVillageMembers(village.ID)
	var isLeader bool
	for _, m := range members {
		if m.UserID == userID && m.Role == models.VillageRoleLeader {
			isLeader = true
			break
		}
	}

	if isLeader && len(members) > 1 {
		return errors.New(errors.ErrCodeValidationFailed, "لیدر دهکده قبل از خروج باید لیدری را واگذار کند")
	}

	return s.repo.RemoveMember(village.ID, userID)
}

func (s *VillageService) AddXP(villageID uint, xp int64) error {
	// 1. First increment XP safely in DB
	if err := s.repo.UpdateVillageXP(villageID, xp); err != nil {
		return err
	}

	// 2. Read new village state to calculate level and score
	village, err := s.repo.GetVillageByID(villageID)
	if err != nil {
		return err
	}

	// 3. Calculate new level based on total XP
	newLevel := village.Level
	for village.XP >= s.GetXPRequiredForLevel(newLevel) {
		newLevel++
	}

	// 4. Update the final stats (Level and Score) based on total XP
	// Score calculation: score := xp/10 + int64(level*100)
	return s.UpdateVillageStats(villageID, newLevel, village.XP)
}

func (s *VillageService) GetXPRequiredForLevel(level int) int64 {
	return int64(level * 500)
}

func (s *VillageService) UpdateVillageStats(villageID uint, level int, xp int64) error {
	score := xp/10 + int64(level*100)
	return s.repo.UpdateVillageStats(villageID, level, xp, score)
}

func (s *VillageService) GetRanking(limit int) ([]models.Village, error) {
	return s.repo.GetVillageLeaderboard(limit)
}

func (s *VillageService) GetUserVillageInfo(userID uint) (*models.Village, int64, error) {
	village, err := s.repo.GetUserVillage(userID)
	if err != nil {
		return nil, 0, err
	}
	if village == nil {
		return nil, 0, nil
	}

	rank, err := s.repo.GetVillageRank(village.ID)
	return village, rank, err
}

func (s *VillageService) AddXPForUser(userID uint, xp int64) error {
	village, err := s.repo.GetUserVillage(userID)
	if err != nil || village == nil {
		return err
	}

	// Apply XP Buff (e.g. 5% per level)
	if village.BuffXPLevel > 0 {
		bonus := (float64(xp) * float64(village.BuffXPLevel) * 0.05)
		xp += int64(bonus)
	}

	return s.AddXP(village.ID, xp)
}
