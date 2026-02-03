package repositories

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type FriendRepository struct {
	db *gorm.DB
}

func NewFriendRepository(db *gorm.DB) *FriendRepository {
	return &FriendRepository{db: db}
}

// SendFriendRequest creates a new friend request
func (r *FriendRepository) SendFriendRequest(requesterID, addresseeID uint) error {
	// Check if already friends or request exists
	var existing models.Friendship
	result := r.db.Where(
		"(requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)",
		requesterID, addresseeID, addresseeID, requesterID,
	).First(&existing)

	if result.Error == nil {
		if existing.Status == models.FriendshipStatusAccepted {
			return errors.New(errors.ErrCodeAlreadyExists, "already friends")
		}
		return errors.New(errors.ErrCodeAlreadyExists, "friend request already exists")
	}

	if result.Error != gorm.ErrRecordNotFound {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check existing friendship")
	}

	// Create new friend request
	friendship := &models.Friendship{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      models.FriendshipStatusPending,
	}

	if err := r.db.Create(friendship).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create friend request")
	}

	return nil
}

// AcceptFriendRequest accepts a friend request
func (r *FriendRepository) AcceptFriendRequest(requestID uint) error {
	result := r.db.Model(&models.Friendship{}).
		Where("id = ? AND status = ?", requestID, models.FriendshipStatusPending).
		Update("status", models.FriendshipStatusAccepted)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to accept friend request")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "friend request not found or already processed")
	}

	return nil
}

// RejectFriendRequest rejects a friend request
func (r *FriendRepository) RejectFriendRequest(requestID uint) error {
	result := r.db.Model(&models.Friendship{}).
		Where("id = ? AND status = ?", requestID, models.FriendshipStatusPending).
		Update("status", models.FriendshipStatusRejected)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to reject friend request")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "friend request not found or already processed")
	}

	return nil
}

// GetFriends retrieves list of user's friends
func (r *FriendRepository) GetFriends(userID uint) ([]models.User, error) {
	var friends []models.User

	// Get friendships where user is either requester or addressee and status is accepted
	err := r.db.Table("users").
		Select("users.*").
		Joins("JOIN friendships ON (friendships.requester_id = users.id OR friendships.addressee_id = users.id)").
		Where("(friendships.requester_id = ? OR friendships.addressee_id = ?) AND friendships.status = ? AND users.id != ?",
			userID, userID, models.FriendshipStatusAccepted, userID).
		Find(&friends).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get friends")
	}

	return friends, nil
}

// GetPendingRequests retrieves pending friend requests for a user
func (r *FriendRepository) GetPendingRequests(userID uint) ([]models.Friendship, error) {
	var requests []models.Friendship

	err := r.db.Where("addressee_id = ? AND status = ?", userID, models.FriendshipStatusPending).
		Preload("Requester").
		Find(&requests).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to get pending requests")
	}

	return requests, nil
}

// RemoveFriend removes a friendship
func (r *FriendRepository) RemoveFriend(user1ID, user2ID uint) error {
	result := r.db.Where(
		"((requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)) AND status = ?",
		user1ID, user2ID, user2ID, user1ID, models.FriendshipStatusAccepted,
	).Delete(&models.Friendship{})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to remove friend")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "friendship not found")
	}

	return nil
}

// AreFriends checks if two users are friends
func (r *FriendRepository) AreFriends(user1ID, user2ID uint) (bool, error) {
	var count int64
	result := r.db.Model(&models.Friendship{}).
		Where(
			"((requester_id = ? AND addressee_id = ?) OR (requester_id = ? AND addressee_id = ?)) AND status = ?",
			user1ID, user2ID, user2ID, user1ID, models.FriendshipStatusAccepted,
		).Count(&count)

	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check friendship")
	}

	return count > 0, nil
}
