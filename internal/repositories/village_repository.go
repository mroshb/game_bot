package repositories

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type VillageRepository struct {
	db *gorm.DB
}

func NewVillageRepository(db *gorm.DB) *VillageRepository {
	return &VillageRepository{db: db}
}

func (r *VillageRepository) CreateVillage(village *models.Village, creatorID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(village).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create village")
		}

		member := &models.VillageMember{
			VillageID: village.ID,
			UserID:    creatorID,
			Role:      models.VillageRoleLeader,
		}

		if err := tx.Create(member).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to add creator as leader")
		}

		return nil
	})
}

func (r *VillageRepository) GetVillageByID(id uint) (*models.Village, error) {
	var village models.Village
	if err := r.db.Preload("Creator").First(&village, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "village not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get village")
	}
	return &village, nil
}

func (r *VillageRepository) GetVillageByName(name string) (*models.Village, error) {
	var village models.Village
	if err := r.db.Where("name = ?", name).First(&village).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "village not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get village by name")
	}
	return &village, nil
}

func (r *VillageRepository) GetUserVillage(userID uint) (*models.Village, error) {
	var member models.VillageMember
	if err := r.db.Preload("Village").Where("user_id = ?", userID).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // User not in a village
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get user village")
	}
	return &member.Village, nil
}

func (r *VillageRepository) AddMember(villageID, userID uint, role string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		member := &models.VillageMember{
			VillageID: villageID,
			UserID:    userID,
			Role:      role,
		}
		if err := tx.Create(member).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to add member")
		}

		if err := tx.Model(&models.Village{}).Where("id = ?", villageID).Update("member_count", gorm.Expr("member_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *VillageRepository) RemoveMember(villageID, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("village_id = ? AND user_id = ?", villageID, userID).Delete(&models.VillageMember{}).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to remove member")
		}

		if err := tx.Model(&models.Village{}).Where("id = ?", villageID).Update("member_count", gorm.Expr("member_count - 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *VillageRepository) GetVillageMembers(villageID uint) ([]models.VillageMember, error) {
	var members []models.VillageMember
	if err := r.db.Preload("User").Where("village_id = ?", villageID).Find(&members).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get village members")
	}
	return members, nil
}

func (r *VillageRepository) UpdateVillageXP(villageID uint, xp int64) error {
	return r.db.Model(&models.Village{}).Where("id = ?", villageID).Updates(map[string]interface{}{
		"xp": gorm.Expr("xp + ?", xp),
	}).Error
}

func (r *VillageRepository) GetVillageLeaderboard(limit int) ([]models.Village, error) {
	var villages []models.Village
	if err := r.db.Order("score DESC").Limit(limit).Find(&villages).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get village leaderboard")
	}
	return villages, nil
}

func (r *VillageRepository) GetVillageRank(villageID uint) (int64, error) {
	var village models.Village
	if err := r.db.First(&village, villageID).Error; err != nil {
		return 0, err
	}

	var rank int64
	r.db.Model(&models.Village{}).Where("score > ?", village.Score).Count(&rank)
	return rank + 1, nil
}

func (r *VillageRepository) UpdateVillageStats(villageID uint, level int, xp, score int64) error {
	return r.db.Model(&models.Village{}).Where("id = ?", villageID).Updates(map[string]interface{}{
		"level": level,
		"xp":    xp,
		"score": score,
	}).Error
}
