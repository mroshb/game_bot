package repositories

import (
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"github.com/mroshb/game_bot/pkg/utils"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(user *models.User) error {
	if user.PublicID == "" {
		user.PublicID = utils.GenerateRandomID(8)
	}

	result := r.db.Create(user)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to create user")
	}
	return nil
}

// GetUserByPublicID retrieves a user by Public ID
func (r *UserRepository) GetUserByPublicID(publicID string) (*models.User, error) {
	var user models.User
	result := r.db.Where("public_id = ?", publicID).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "user not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user")
	}

	return &user, nil
}

// GetUserByTelegramID retrieves a user by Telegram ID
func (r *UserRepository) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	result := r.db.Where("telegram_id = ?", telegramID).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "user not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user")
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	result := r.db.First(&user, id)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "user not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user")
	}

	return &user, nil
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(user *models.User) error {
	result := r.db.Save(user)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update user")
	}
	return nil
}

// UpdateUserStatus updates user status
func (r *UserRepository) UpdateUserStatus(userID uint, status string) error {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("status", status)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update status")
	}
	return nil
}

// UpdateLastActivity updates user's last activity timestamp
func (r *UserRepository) UpdateLastActivity(userID uint) error {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_activity", gorm.Expr("CURRENT_TIMESTAMP"))
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update last activity")
	}
	return nil
}

// UserExists checks if a user exists by Telegram ID
func (r *UserRepository) UserExists(telegramID int64) (bool, error) {
	var count int64
	result := r.db.Model(&models.User{}).Where("telegram_id = ?", telegramID).Count(&count)
	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check user existence")
	}
	return count > 0, nil
}

// GetBalance retrieves user's coin balance
func (r *UserRepository) GetBalance(userID uint) (int64, error) {
	var user models.User
	result := r.db.Select("coin_balance").First(&user, userID)

	if result.Error == gorm.ErrRecordNotFound {
		return 0, errors.New(errors.ErrCodeNotFound, "user not found")
	}
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get balance")
	}

	return user.CoinBalance, nil
}

// UpdateBalance updates user's coin balance (use with caution, prefer coin repository)
func (r *UserRepository) UpdateBalance(userID uint, newBalance int64) error {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Update("coin_balance", newBalance)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update balance")
	}
	return nil
}

// GetUserStats retrieves user statistics
func (r *UserRepository) GetUserStats(userID uint) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Get total matches
	var matchCount int64
	r.db.Model(&models.MatchSession{}).
		Where("(user1_id = ? OR user2_id = ?) AND status != ?", userID, userID, models.MatchStatusActive).
		Count(&matchCount)
	stats["matches"] = matchCount

	// Get wins (from game participants)
	var winCount int64
	r.db.Table("game_participants").
		Joins("JOIN game_sessions ON game_sessions.id = game_participants.game_session_id").
		Where("game_participants.user_id = ? AND game_sessions.status = ?", userID, models.GameStatusFinished).
		Where("game_participants.score = (SELECT MAX(score) FROM game_participants WHERE game_session_id = game_sessions.id)").
		Count(&winCount)
	stats["wins"] = winCount

	// Get losses
	stats["losses"] = matchCount - winCount

	// Get friend count
	var friendCount int64
	r.db.Model(&models.Friendship{}).
		Where("(requester_id = ? OR addressee_id = ?) AND status = ?", userID, userID, models.FriendshipStatusAccepted).
		Count(&friendCount)
	stats["friends"] = friendCount

	// Get XP and Level from User model
	var user models.User
	if err := r.db.Select("xp, level").First(&user, userID).Error; err == nil {
		stats["xp"] = user.XP
		stats["level"] = int64(user.Level)
	}

	return stats, nil
}

// GetLeaderboard returns top users based on criteria
func (r *UserRepository) GetLeaderboard(category string, period string, limit int) ([]models.User, error) {
	var users []models.User
	query := r.db.Model(&models.User{})

	switch category {
	case "quiz":
		// This should ideally join with game_participants and filter by game_type='quiz'
		// For simplicity now, let's use XP or special score if we had one.
		query = query.Order("xp DESC")
	case "truth_dare":
		query = query.Order("xp DESC")
	default:
		query = query.Order("coin_balance DESC")
	}

	// Period logic would require more complex queries on transactions/matches
	// For now we just return global top users

	err := query.Limit(limit).Find(&users).Error
	return users, err
}

// GetUserRank returns user's rank based on coin balance
func (r *UserRepository) GetUserRank(userID uint) (int64, error) {
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return 0, err
	}

	var rank int64
	r.db.Model(&models.User{}).Where("coin_balance > ?", user.CoinBalance).Count(&rank)
	return rank + 1, nil
}

// FindRecentChatUsers returns users the current user has chatted with recently
func (r *UserRepository) FindRecentChatUsers(userID uint, limit int) ([]models.User, error) {
	var users []models.User

	// Complex query to find unique users from match_sessions
	err := r.db.Raw(`
		SELECT u.* 
		FROM users u
		JOIN (
			SELECT 
				CASE WHEN user1_id = ? THEN user2_id ELSE user1_id END as other_user_id,
				MAX(ended_at) as max_ended_at
			FROM match_sessions
			WHERE (user1_id = ? OR user2_id = ?)
			AND ended_at IS NOT NULL
			GROUP BY other_user_id
		) last_chats ON u.id = last_chats.other_user_id
		ORDER BY last_chats.max_ended_at DESC
		LIMIT ?
	`, userID, userID, userID, limit).Scan(&users).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find recent chat users")
	}

	return users, nil
}

// FindUsersByProvince returns users in the same province
func (r *UserRepository) FindUsersByProvince(userID uint, province string, limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("province = ? AND id != ?", province, userID).
		Order("last_activity DESC").
		Limit(limit).
		Find(&users).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find users by province")
	}
	return users, nil
}

// FindUsersByAge returns users with similar age (+/- 2 years)
func (r *UserRepository) FindUsersByAge(userID uint, age int, limit int) ([]models.User, error) {
	var users []models.User
	minAge := age - 2
	maxAge := age + 2

	err := r.db.Where("age BETWEEN ? AND ? AND id != ?", minAge, maxAge, userID).
		Order("last_activity DESC").
		Limit(limit).
		Find(&users).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find users by age")
	}
	return users, nil
}

// FindNewUsers returns most recently registered users
func (r *UserRepository) FindNewUsers(userID uint, limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("id != ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find new users")
	}
	return users, nil
}

// FindUsersWithNoChats returns users who have very few matches (implied "no chats" or low activity)
// Since we don't store match count directly on user, we do a LEFT JOIN check or just random active users for now if strict 0 is hard.
// Strict 0 check:
func (r *UserRepository) FindUsersWithNoChats(userID uint, limit int) ([]models.User, error) {
	var users []models.User

	err := r.db.Raw(`
		SELECT u.* 
		FROM users u
		LEFT JOIN match_sessions ms ON (u.id = ms.user1_id OR u.id = ms.user2_id)
		WHERE u.id != ?
		GROUP BY u.id
		HAVING COUNT(ms.id) = 0
		ORDER BY u.created_at DESC
		LIMIT ?
	`, userID, limit).Scan(&users).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find users with no chats")
	}
	return users, nil
}

// UpdateLocation updates user's latitude and longitude
func (r *UserRepository) UpdateLocation(userID uint, lat, lon float64) error {
	result := r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"latitude":  lat,
		"longitude": lon,
	})
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update location")
	}
	return nil
}

// FindNearbyUsers returns users sorted by distance
func (r *UserRepository) FindNearbyUsers(userID uint, lat, lon float64, limit int) ([]models.User, error) {
	// Define a struct to capture the computed distance
	type UserWithDistance struct {
		models.User
		Dist float64 `gorm:"column:distance"`
	}
	var results []UserWithDistance

	// Haversine formula
	// 6371 is Earth radius in km
	query := `
		SELECT *, (
			6371 * acos(
				cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + 
				sin(radians(?)) * sin(radians(latitude))
			)
		) AS distance 
		FROM users 
		WHERE id != ? 
		AND latitude != 0 AND longitude != 0
		ORDER BY distance ASC 
		LIMIT ?
	`

	err := r.db.Raw(query, lat, lon, lat, userID, limit).Scan(&results).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to find nearby users")
	}

	// Map back to models.User
	users := make([]models.User, len(results))
	for i, r := range results {
		users[i] = r.User
		users[i].Distance = r.Dist
	}

	return users, nil
}

// HasLiked checks if a user has already liked another user
func (r *UserRepository) HasLiked(likerID, likedID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserLike{}).
		Where("liker_id = ? AND liked_id = ?", likerID, likedID).
		Count(&count).Error
	return count > 0, err
}

// AddLike adds a like from one user to another
func (r *UserRepository) AddLike(likerID, likedID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create like record
		like := models.UserLike{
			LikerID: likerID,
			LikedID: likedID,
		}
		if err := tx.Create(&like).Error; err != nil {
			return err
		}

		// 2. Increment like count on target user
		if err := tx.Model(&models.User{}).Where("id = ?", likedID).Update("likes", gorm.Expr("likes + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})
}

// MarkInactiveUsersOffline marks users as offline if they haven't been active
func (r *UserRepository) MarkInactiveUsersOffline(timeout time.Duration) (int64, error) {
	result := r.db.Model(&models.User{}).
		Where("last_activity < ? AND status != ?", time.Now().Add(-timeout), models.UserStatusOffline).
		Update("status", models.UserStatusOffline)
	return result.RowsAffected, result.Error
}
