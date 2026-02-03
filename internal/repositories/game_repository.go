package repositories

import (
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type GameRepository struct {
	db *gorm.DB
}

func NewGameRepository(db *gorm.DB) *GameRepository {
	return &GameRepository{db: db}
}

// GetRandomQuestion retrieves a random question by type and optional category
func (r *GameRepository) GetRandomQuestion(questionType, category string) (*models.Question, error) {
	var question models.Question
	query := r.db.Where("question_type = ?", questionType)

	if category != "" {
		query = query.Where("category = ?", category)
	}

	result := query.Order("RANDOM()").First(&question)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "no questions found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get question")
	}

	return &question, nil
}

// GetQuizQuestions retrieves multiple quiz questions
func (r *GameRepository) GetQuizQuestions(count int) ([]models.Question, error) {
	var questions []models.Question
	result := r.db.Where("question_type = ?", models.QuestionTypeQuiz).
		Order("RANDOM()").
		Limit(count).
		Find(&questions)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get quiz questions")
	}

	return questions, nil
}

// CreateGameSession creates a new game session
func (r *GameRepository) CreateGameSession(roomID uint, gameType string) (*models.GameSession, error) {
	session := &models.GameSession{
		RoomID:     roomID,
		GameType:   gameType,
		Status:     models.GameStatusWaiting,
		TurnUserID: 0,
	}

	if err := r.db.Create(session).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to create game session")
	}

	return session, nil
}

// SetTurnUserID sets the current user turn in a game session
func (r *GameRepository) SetTurnUserID(gameSessionID, userID uint) error {
	result := r.db.Model(&models.GameSession{}).
		Where("id = ?", gameSessionID).
		Update("turn_user_id", userID)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to set turn user id")
	}

	return nil
}

// AddParticipant adds a participant to a game session
func (r *GameRepository) AddParticipant(gameSessionID, userID uint, turnOrder int) error {
	participant := &models.GameParticipant{
		GameSessionID: gameSessionID,
		UserID:        userID,
		Score:         0,
		TurnOrder:     turnOrder,
		Answers:       "[]",
	}

	if err := r.db.Create(participant).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to add participant")
	}

	return nil
}

// UpdateScore updates a participant's score
func (r *GameRepository) UpdateScore(gameSessionID, userID uint, score int) error {
	result := r.db.Model(&models.GameParticipant{}).
		Where("game_session_id = ? AND user_id = ?", gameSessionID, userID).
		Update("score", score)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update score")
	}

	return nil
}

// IncrementScore increments a participant's score
func (r *GameRepository) IncrementScore(gameSessionID, userID uint, points int) error {
	result := r.db.Model(&models.GameParticipant{}).
		Where("game_session_id = ? AND user_id = ?", gameSessionID, userID).
		Update("score", gorm.Expr("score + ?", points))

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to increment score")
	}

	return nil
}

// EndGame ends a game session
func (r *GameRepository) EndGame(gameSessionID uint) error {
	result := r.db.Model(&models.GameSession{}).
		Where("id = ?", gameSessionID).
		Updates(map[string]interface{}{
			"status":   models.GameStatusFinished,
			"ended_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to end game")
	}

	return nil
}

// GetGameSession retrieves a game session by ID
func (r *GameRepository) GetGameSession(gameSessionID uint) (*models.GameSession, error) {
	var session models.GameSession
	result := r.db.Preload("Room.Host").First(&session, gameSessionID)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "game session not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get game session")
	}

	return &session, nil
}

// GetParticipants retrieves all participants of a game session
func (r *GameRepository) GetParticipants(gameSessionID uint) ([]models.GameParticipant, error) {
	var participants []models.GameParticipant
	result := r.db.Where("game_session_id = ?", gameSessionID).
		Preload("User").
		Order("turn_order ASC").
		Find(&participants)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get participants")
	}

	return participants, nil
}

// GetActiveGameSessionByRoomID retrieves an active game session for a room
func (r *GameRepository) GetActiveGameSessionByRoomID(roomID uint) (*models.GameSession, error) {
	var session models.GameSession
	result := r.db.Where("room_id = ? AND status != ?", roomID, models.GameStatusFinished).
		Preload("Room.Host").
		First(&session)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get active game session")
	}

	return &session, nil
}

// GetWinner retrieves the winner of a game session
func (r *GameRepository) GetWinner(gameSessionID uint) (*models.GameParticipant, error) {
	var winner models.GameParticipant
	result := r.db.Where("game_session_id = ?", gameSessionID).
		Preload("User").
		Order("score DESC").
		First(&winner)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "no winner found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get winner")
	}

	return &winner, nil
}

// StartGame starts a game session
func (r *GameRepository) StartGame(gameSessionID uint) error {
	result := r.db.Model(&models.GameSession{}).
		Where("id = ?", gameSessionID).
		Updates(map[string]interface{}{
			"status":     models.GameStatusInProgress,
			"started_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to start game")
	}

	return nil
}

// UpdateCurrentQuestion updates the current question in a game session
func (r *GameRepository) UpdateCurrentQuestion(gameSessionID, questionID uint) error {
	result := r.db.Model(&models.GameSession{}).
		Where("id = ?", gameSessionID).
		Update("current_question_id", questionID)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update current question")
	}

	return nil
}

// UpdateGameStatus updates the status of a game session
func (r *GameRepository) UpdateGameStatus(gameSessionID uint, status string) error {
	result := r.db.Model(&models.GameSession{}).
		Where("id = ?", gameSessionID).
		Update("status", status)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update game status")
	}

	return nil
}

// GetQuestionByID retrieves a question by ID
func (r *GameRepository) GetQuestionByID(id uint) (*models.Question, error) {
	var question models.Question
	result := r.db.First(&question, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &question, nil
}

// GetRecentGames retrieves recent games user participated in
func (r *GameRepository) GetRecentGames(userID uint, limit int) ([]models.GameParticipant, error) {
	var participants []models.GameParticipant
	err := r.db.Preload("GameSession").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&participants).Error
	return participants, err
}
