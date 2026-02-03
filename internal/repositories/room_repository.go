package repositories

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// CreateRoom creates a new room
func (r *RoomRepository) CreateRoom(room *models.Room) error {
	if err := r.db.Create(room).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create room")
	}
	return nil
}

// GetRoomByID retrieves a room by ID
func (r *RoomRepository) GetRoomByID(roomID uint) (*models.Room, error) {
	var room models.Room
	result := r.db.Preload("Host").First(&room, roomID)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "room not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get room")
	}

	return &room, nil
}

// GetPublicRooms retrieves all public rooms that are waiting or in progress and not full
func (r *RoomRepository) GetPublicRooms() ([]models.Room, error) {
	var rooms []models.Room

	// Query to find rooms that are not full
	// We join with a subquery that counts active members (is_kicked = false)
	result := r.db.Table("rooms").
		Select("rooms.*").
		Joins("LEFT JOIN (SELECT room_id, COUNT(*) as member_count FROM room_members WHERE is_kicked = false GROUP BY room_id) as counts ON counts.room_id = rooms.id").
		Where("rooms.room_type = ? AND rooms.status IN ? AND (counts.member_count < rooms.max_players OR counts.member_count IS NULL)",
			models.RoomTypePublic,
			[]string{models.RoomStatusWaiting, models.RoomStatusInProgress}).
		Preload("Host").
		Order("rooms.created_at DESC").
		Find(&rooms)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get public rooms")
	}

	return rooms, nil
}

// GetRoomByInviteCode retrieves a room by invite code
func (r *RoomRepository) GetRoomByInviteCode(inviteCode string) (*models.Room, error) {
	var room models.Room
	result := r.db.Where("invite_code = ?", inviteCode).
		Preload("Host").
		First(&room)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "room not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get room")
	}

	return &room, nil
}

// AddMember adds a member to a room
func (r *RoomRepository) AddMember(roomID, userID uint) error {
	// Check if room is full
	var memberCount int64
	r.db.Model(&models.RoomMember{}).Where("room_id = ? AND is_kicked = ?", roomID, false).Count(&memberCount)

	var room models.Room
	if err := r.db.First(&room, roomID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get room")
	}

	if int(memberCount) >= room.MaxPlayers {
		return errors.New(errors.ErrCodeValidationFailed, "room is full")
	}

	// Check if user is already a member
	var existing models.RoomMember
	result := r.db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&existing)
	if result.Error == nil {
		return errors.New(errors.ErrCodeAlreadyExists, "user already in room")
	}

	// Add member
	member := &models.RoomMember{
		RoomID: roomID,
		UserID: userID,
	}

	if err := r.db.Create(member).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to add member")
	}

	return nil
}

// RemoveMember removes a member from a room
func (r *RoomRepository) RemoveMember(roomID, userID uint) error {
	result := r.db.Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&models.RoomMember{})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to remove member")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "member not found")
	}

	return nil
}

// GetRoomMembers retrieves all members of a room
func (r *RoomRepository) GetRoomMembers(roomID uint) ([]models.User, error) {
	var members []models.User
	result := r.db.Table("users").
		Select("users.*").
		Joins("JOIN room_members ON room_members.user_id = users.id").
		Where("room_members.room_id = ? AND room_members.is_kicked = ?", roomID, false).
		Find(&members)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get room members")
	}

	return members, nil
}

// GetMemberCount returns the number of members in a room
func (r *RoomRepository) GetMemberCount(roomID uint) (int, error) {
	var count int64
	result := r.db.Model(&models.RoomMember{}).
		Where("room_id = ? AND is_kicked = ?", roomID, false).
		Count(&count)

	if result.Error != nil {
		return 0, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to count members")
	}

	return int(count), nil
}

// IsHost checks if a user is the host of a room
func (r *RoomRepository) IsHost(roomID, userID uint) (bool, error) {
	var room models.Room
	result := r.db.First(&room, roomID)

	if result.Error == gorm.ErrRecordNotFound {
		return false, errors.New(errors.ErrCodeNotFound, "room not found")
	}
	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get room")
	}

	return room.HostID == userID, nil
}

// IsMember checks if a user is a member of a room
func (r *RoomRepository) IsMember(roomID, userID uint) (bool, error) {
	var count int64
	result := r.db.Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ? AND is_kicked = ?", roomID, userID, false).
		Count(&count)

	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check membership")
	}

	return count > 0, nil
}

// CloseRoom closes a room
func (r *RoomRepository) CloseRoom(roomID uint) error {
	result := r.db.Model(&models.Room{}).
		Where("id = ?", roomID).
		Update("status", models.RoomStatusClosed)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to close room")
	}

	return nil
}

// UpdateRoomStatus updates room status
func (r *RoomRepository) UpdateRoomStatus(roomID uint, status string) error {
	result := r.db.Model(&models.Room{}).
		Where("id = ?", roomID).
		Update("status", status)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update room status")
	}

	return nil
}

// KickMember kicks a member from a room
func (r *RoomRepository) KickMember(roomID, userID uint) error {
	result := r.db.Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("is_kicked", true)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to kick member")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "member not found")
	}

	return nil
}

// GetUserRooms retrieves all rooms a user is a member of
func (r *RoomRepository) GetUserRooms(userID uint) ([]models.Room, error) {
	var rooms []models.Room
	result := r.db.Table("rooms").
		Select("rooms.*").
		Joins("JOIN room_members ON room_members.room_id = rooms.id").
		Where("room_members.user_id = ? AND room_members.is_kicked = ? AND rooms.status != ?",
			userID, false, models.RoomStatusClosed).
		Preload("Host").
		Find(&rooms)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user rooms")
	}

	return rooms, nil
}
