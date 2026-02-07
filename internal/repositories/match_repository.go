package repositories

import (
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type MatchRepository struct {
	db *gorm.DB
}

func NewMatchRepository(db *gorm.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

// AddToQueue adds a user to the matchmaking queue
func (r *MatchRepository) AddToQueue(queue *models.MatchmakingQueue) error {
	// Check if user is already in queue
	var existing models.MatchmakingQueue
	result := r.db.Where("user_id = ?", queue.UserID).First(&existing)

	if result.Error == nil {
		return errors.New(errors.ErrCodeAlreadyExists, "user already in matchmaking queue")
	}

	if result.Error != gorm.ErrRecordNotFound {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check queue")
	}

	// Add to queue
	if err := r.db.Create(queue).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to add to queue")
	}

	return nil
}

// RemoveFromQueue removes a user from the matchmaking queue
func (r *MatchRepository) RemoveFromQueue(userID uint) error {
	result := r.db.Where("user_id = ?", userID).Delete(&models.MatchmakingQueue{})
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to remove from queue")
	}
	return nil
}

// FindMatch finds a compatible match for the user
func (r *MatchRepository) FindMatch(userID uint, filters *models.MatchFilters) (*models.User, error) {
	// Get the searching user's info
	var searchingUser models.User
	if err := r.db.First(&searchingUser, userID).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get user")
	}

	// Build query for finding match
	query := r.db.Table("matchmaking_queue").
		Select("users.*").
		Joins("JOIN users ON users.id = matchmaking_queue.user_id").
		Where("matchmaking_queue.user_id != ?", userID)

	// Apply GameType filter
	if filters.GameType != "" {
		query = query.Where("matchmaking_queue.game_type = ?", filters.GameType)
	} else {
		// Default to chat if not specified (backward compatibility)
		query = query.Where("matchmaking_queue.game_type = ?", models.GameTypeChat)
	}

	// Apply filters
	if filters.Gender != "" && filters.Gender != models.RequestedGenderAny {
		query = query.Where("users.gender = ?", filters.Gender)
	}

	if filters.MinAge != nil {
		query = query.Where("users.age >= ?", *filters.MinAge)
	}

	if filters.MaxAge != nil {
		query = query.Where("users.age <= ?", *filters.MaxAge)
	}

	if filters.City != "" {
		// Exact city match
		query = query.Where("(users.city = ? OR matchmaking_queue.city = ?)", filters.City, filters.City)
	}

	if len(filters.Provinces) > 0 {
		// User wants one of these provinces
		// AND the target user must be in one of those provinces
		query = query.Where("users.province IN ?", filters.Provinces)
	}

	// Bidirectional check: The match candidate (User B) must also accept searchingUser's (User A) province.
	// User B's preferences are in `matchmaking_queue` table.
	// If User B has specified TargetProvinces, User A's province must be in that list.
	// If User B's TargetProvinces is empty, they accept any province.
	// Note: We use LIKE for simplicity as data is comma-separated.
	// To be robust against similar names, we might want exact match but provinces are usually distinct enough.
	if searchingUser.Province != "" {
		query = query.Where("(matchmaking_queue.target_provinces = '' OR matchmaking_queue.target_provinces IS NULL OR matchmaking_queue.target_provinces LIKE ?)", "%"+searchingUser.Province+"%")
	}

	// Also check if the other user wants to match with this user's gender
	query = query.Where("(matchmaking_queue.requested_gender = ? OR matchmaking_queue.requested_gender = ?)",
		searchingUser.Gender, models.RequestedGenderAny)

	// Order by creation time (FIFO)
	query = query.Order("matchmaking_queue.created_at ASC")

	var matchedUser models.User
	if err := query.First(&matchedUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No match found
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find match")
	}

	return &matchedUser, nil
}

// CreateMatchSession creates a new match session
func (r *MatchRepository) CreateMatchSession(user1ID, user2ID uint, timeoutDuration time.Duration) (*models.MatchSession, error) {
	session := &models.MatchSession{
		User1ID:   user1ID,
		User2ID:   user2ID,
		StartedAt: time.Now(),
		TimeoutAt: time.Now().Add(timeoutDuration),
		Status:    models.MatchStatusActive,
	}

	if err := r.db.Create(session).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to create match session")
	}

	return session, nil
}

// GetActiveMatch retrieves active match for a user
func (r *MatchRepository) GetActiveMatch(userID uint) (*models.MatchSession, error) {
	var session models.MatchSession
	result := r.db.Where("(user1_id = ? OR user2_id = ?) AND status IN (?, ?)",
		userID, userID, models.MatchStatusActive, models.MatchStatusTimeout).
		Preload("User1").
		Preload("User2").
		First(&session)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get active match")
	}

	return &session, nil
}

// EndMatch ends a match session
func (r *MatchRepository) EndMatch(sessionID uint) error {
	now := time.Now()
	result := r.db.Model(&models.MatchSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":   models.MatchStatusEnded,
			"ended_at": now,
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to end match")
	}

	return nil
}

// CheckAndHandleTimeouts checks for timed out matches and handles them
func (r *MatchRepository) CheckAndHandleTimeouts() ([]models.MatchSession, error) {
	var timedOutSessions []models.MatchSession

	result := r.db.Where("status = ? AND timeout_at < ?",
		models.MatchStatusActive, time.Now()).
		Preload("User1").
		Preload("User2").
		Find(&timedOutSessions)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check timeouts")
	}

	// Update status to timeout
	if len(timedOutSessions) > 0 {
		sessionIDs := make([]uint, len(timedOutSessions))
		for i, session := range timedOutSessions {
			sessionIDs[i] = session.ID
		}

		now := time.Now()
		r.db.Model(&models.MatchSession{}).
			Where("id IN ?", sessionIDs).
			Updates(map[string]interface{}{
				"status":   models.MatchStatusTimeout,
				"ended_at": now,
			})
	}

	return timedOutSessions, nil
}

// IsUserInQueue checks if user is in matchmaking queue
func (r *MatchRepository) IsUserInQueue(userID uint) (bool, error) {
	var count int64
	result := r.db.Model(&models.MatchmakingQueue{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check queue")
	}
	return count > 0, nil
}

// GetQueueEntry retrieves user's queue entry
func (r *MatchRepository) GetQueueEntry(userID uint) (*models.MatchmakingQueue, error) {
	var queue models.MatchmakingQueue
	result := r.db.Where("user_id = ?", userID).First(&queue)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "queue entry not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get queue entry")
	}

	return &queue, nil
}
