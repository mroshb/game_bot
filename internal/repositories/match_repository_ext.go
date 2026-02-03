package repositories

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

// Add GetMatchByID to MatchRepository
func (r *MatchRepository) GetMatchByID(id uint) (*models.MatchSession, error) {
	var session models.MatchSession
	result := r.db.Preload("User1").Preload("User2").First(&session, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "match not found")
		}
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get match from db")
	}
	return &session, nil
}
