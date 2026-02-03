package services

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/internal/repositories"
	"github.com/mroshb/game_bot/pkg/errors"
)

type VillageService struct {
	repo     *repositories.VillageRepository
	userRepo *repositories.UserRepository
}

func NewVillageService(repo *repositories.VillageRepository, userRepo *repositories.UserRepository) *VillageService {
	return &VillageService{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *VillageService) CreateVillage(name, description string, creatorID uint) (*models.Village, error) {
	// Check if user is already in a village
	existing, err := s.repo.GetUserVillage(creatorID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "شما قبلاً در یک دهکده عضو هستید")
	}

	village := &models.Village{
		Name:        name,
		Description: description,
		CreatorID:   creatorID,
		MemberCount: 1,
	}

	if err := s.repo.CreateVillage(village, creatorID); err != nil {
		return nil, err
	}

	return village, nil
}

func (s *VillageService) AddMember(villageID, userID uint) error {
	village, err := s.repo.GetVillageByID(villageID)
	if err != nil {
		return err
	}

	if village.MemberCount >= 50 {
		return errors.New(errors.ErrCodeValidationFailed, "ظرفیت دهکده تکمیل است (حداکثر ۵۰ نفر)")
	}

	// Check if user is already in a village
	existing, err := s.repo.GetUserVillage(userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New(errors.ErrCodeAlreadyExists, "این کاربر قبلاً در یک دهکده عضو شده است")
	}

	return s.repo.AddMember(villageID, userID, models.VillageRoleMember)
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
	return s.AddXP(village.ID, xp)
}
