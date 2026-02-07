package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
)

type QuizMatchRepository struct {
	db *gorm.DB
}

func NewQuizMatchRepository(db *gorm.DB) *QuizMatchRepository {
	return &QuizMatchRepository{db: db}
}

// CreateQuizMatch creates a new quiz match between two users
func (r *QuizMatchRepository) CreateQuizMatch(user1ID, user2ID uint) (*models.QuizMatch, error) {
	timeoutAt := time.Now().Add(time.Duration(models.QuizTimeoutDays) * 24 * time.Hour)

	match := &models.QuizMatch{
		User1ID:         user1ID,
		User2ID:         user2ID,
		CurrentRound:    1,
		CurrentQuestion: 0,
		State:           models.QuizStateWaitingCategory,
		TurnUserID:      &user1ID, // User1 chooses first category
		TimeoutAt:       timeoutAt,
		LastActivityAt:  time.Now(),
	}

	if err := r.db.Create(match).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to create quiz match")
	}

	// Preload users
	if err := r.db.Preload("User1").Preload("User2").First(match, match.ID).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to load match users")
	}

	return match, nil
}

// GetQuizMatch retrieves a quiz match by ID with all relations
func (r *QuizMatchRepository) GetQuizMatch(matchID uint) (*models.QuizMatch, error) {
	var match models.QuizMatch
	result := r.db.Preload("User1").Preload("User2").First(&match, matchID)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrCodeNotFound, "quiz match not found")
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get quiz match")
	}

	return &match, nil
}

// GetActiveQuizMatchByUser retrieves the active quiz match for a user
func (r *QuizMatchRepository) GetActiveQuizMatchByUser(userID uint) (*models.QuizMatch, error) {
	var match models.QuizMatch
	result := r.db.Where("(user1_id = ? OR user2_id = ?) AND state NOT IN (?, ?)",
		userID, userID, models.QuizStateGameFinished, models.QuizStateTimeout).
		Preload("User1").Preload("User2").
		Order("last_activity_at DESC").
		First(&match)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil // No active match
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get active quiz match")
	}

	return &match, nil
}

// GetAllActiveQuizMatchesByUser retrieves all active quiz matches for a user
func (r *QuizMatchRepository) GetAllActiveQuizMatchesByUser(userID uint) ([]models.QuizMatch, error) {
	var matches []models.QuizMatch
	result := r.db.Where("(user1_id = ? OR user2_id = ?) AND state NOT IN (?, ?)",
		userID, userID, models.QuizStateGameFinished, models.QuizStateTimeout).
		Preload("User1").Preload("User2").
		Order("CASE WHEN turn_user_id = " + fmt.Sprint(userID) + " THEN 0 ELSE 1 END, last_activity_at DESC").
		Find(&matches)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get active quiz matches")
	}

	return matches, nil
}

// GetFinishedQuizMatchesByUser retrieves finished quiz matches for a user
func (r *QuizMatchRepository) GetFinishedQuizMatchesByUser(userID uint, limit int) ([]models.QuizMatch, error) {
	var matches []models.QuizMatch
	result := r.db.Where("(user1_id = ? OR user2_id = ?) AND state = ?",
		userID, userID, models.QuizStateGameFinished).
		Preload("User1").Preload("User2").
		Order("finished_at DESC").
		Limit(limit).
		Find(&matches)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get finished quiz matches")
	}

	return matches, nil
}

// UpdateQuizMatchState updates the state of a quiz match
func (r *QuizMatchRepository) UpdateQuizMatchState(matchID uint, state string) error {
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Updates(map[string]interface{}{
			"state":            state,
			"last_activity_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update quiz match state")
	}

	return nil
}

// UpdateQuizMatchStateAtomic updates the state only if current state matches one of the expected ones
func (r *QuizMatchRepository) UpdateQuizMatchStateAtomic(matchID uint, expectedStates []string, newState string) (bool, error) {
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ? AND state IN ?", matchID, expectedStates).
		Updates(map[string]interface{}{
			"state":            newState,
			"last_activity_at": time.Now(),
		})

	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update quiz match state atomically")
	}

	return result.RowsAffected > 0, nil
}

// UpdateLastActivity updates the last activity timestamp
func (r *QuizMatchRepository) UpdateLastActivity(matchID uint) error {
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Update("last_activity_at", time.Now())

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update last activity")
	}

	return nil
}

// UpdateCurrentQuestion updates the current question number
func (r *QuizMatchRepository) UpdateCurrentQuestion(matchID uint, questionNum int) error {
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Updates(map[string]interface{}{
			"current_question": questionNum,
			"last_activity_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update current question")
	}

	return nil
}

// UpdateLightsMessageID updates the lights message ID for a user
func (r *QuizMatchRepository) UpdateLightsMessageID(matchID, userID uint, messageID int) error {
	field := "user1_lights_msg_id"

	var match models.QuizMatch
	if err := r.db.First(&match, matchID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get match")
	}

	if match.User2ID == userID {
		field = "user2_lights_msg_id"
	}

	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Update(field, messageID)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update lights message ID")
	}

	return nil
}

// CreateQuizRound creates a new round in a quiz match
// CreateQuizRound creates a new round in a quiz match
func (r *QuizMatchRepository) CreateQuizRound(matchID uint, roundNum int, category string, chosenBy uint, questionIDs string) (*models.QuizRound, error) {
	// Check if round already exists to prevent duplicates via race conditions
	var existingRound models.QuizRound
	if err := r.db.Where("match_id = ? AND round_number = ?", matchID, roundNum).First(&existingRound).Error; err == nil {
		return &existingRound, nil
	}

	round := &models.QuizRound{
		MatchID:        matchID,
		RoundNumber:    roundNum,
		Category:       category,
		ChosenByUserID: chosenBy,
		QuestionIDs:    questionIDs,
	}

	if err := r.db.Create(round).Error; err != nil {
		// one last check for race condition
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			var rnd models.QuizRound
			if e := r.db.Where("match_id = ? AND round_number = ?", matchID, roundNum).First(&rnd).Error; e == nil {
				return &rnd, nil
			}
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternalError, "failed to create quiz round")
	}

	return round, nil
}

// GetQuizRound retrieves a specific round
func (r *QuizMatchRepository) GetQuizRound(matchID uint, roundNum int) (*models.QuizRound, error) {
	var round models.QuizRound
	result := r.db.Where("match_id = ? AND round_number = ?", matchID, roundNum).First(&round)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get quiz round")
	}

	return &round, nil
}

// GetAllQuizRounds retrieves all rounds for a match
func (r *QuizMatchRepository) GetAllQuizRounds(matchID uint) ([]models.QuizRound, error) {
	var rounds []models.QuizRound
	result := r.db.Where("match_id = ?", matchID).Order("round_number ASC").Find(&rounds)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get quiz rounds")
	}

	return rounds, nil
}

// RecordAnswer records a user's answer to a question
func (r *QuizMatchRepository) RecordAnswer(matchID, roundID, userID, questionID uint, questionNum, answerIdx, timeMs int, isCorrect bool, booster string) error {
	answer := &models.QuizAnswer{
		MatchID:        matchID,
		RoundID:        roundID,
		UserID:         userID,
		QuestionID:     questionID,
		QuestionNumber: questionNum,
		AnswerIndex:    &answerIdx,
		IsCorrect:      isCorrect,
		TimeTakenMs:    timeMs,
		BoosterUsed:    booster,
	}

	// Use transaction to ensure atomicity
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Check if already answered
		var existing models.QuizAnswer
		err := tx.Where("match_id = ? AND round_id = ? AND user_id = ? AND question_number = ?",
			matchID, roundID, userID, questionNum).First(&existing).Error

		if err == nil {
			// Already answered, this is idempotent
			return nil
		}

		if err != gorm.ErrRecordNotFound {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to check existing answer")
		}

		// Create answer
		if err := tx.Create(answer).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to record answer")
		}

		// Update last activity
		if err := tx.Model(&models.QuizMatch{}).Where("id = ?", matchID).
			Update("last_activity_at", time.Now()).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to update last activity")
		}

		return nil
	})
}

// DeleteUserAnswer deletes a specific answer (used for Retry booster)
func (r *QuizMatchRepository) DeleteUserAnswer(matchID, roundID, userID uint, questionNum int) error {
	result := r.db.Where("match_id = ? AND round_id = ? AND user_id = ? AND question_number = ?",
		matchID, roundID, userID, questionNum).Delete(&models.QuizAnswer{})
	return result.Error
}

// GetUserAnswers retrieves all answers for a user in a round
func (r *QuizMatchRepository) GetUserAnswers(matchID, roundID, userID uint) ([]models.QuizAnswer, error) {
	var answers []models.QuizAnswer
	result := r.db.Where("match_id = ? AND round_id = ? AND user_id = ?", matchID, roundID, userID).
		Order("question_number ASC").
		Find(&answers)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user answers")
	}

	return answers, nil
}

// HasAnswered checks if a user has answered a specific question
func (r *QuizMatchRepository) HasAnswered(matchID, roundID, userID uint, questionNum int) (bool, error) {
	var count int64
	result := r.db.Model(&models.QuizAnswer{}).
		Where("match_id = ? AND round_id = ? AND user_id = ? AND question_number = ?",
			matchID, roundID, userID, questionNum).
		Count(&count)

	if result.Error != nil {
		return false, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to check if answered")
	}

	return count > 0, nil
}

// UpdateQuizMatchScore updates the total score and time for a user
func (r *QuizMatchRepository) UpdateQuizMatchScore(matchID, userID uint, correctCount int, timeMs int64) error {
	var match models.QuizMatch
	if err := r.db.First(&match, matchID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get match")
	}

	updates := make(map[string]interface{})
	if match.User1ID == userID {
		updates["user1_total_correct"] = correctCount
		updates["user1_total_time_ms"] = timeMs
	} else {
		updates["user2_total_correct"] = correctCount
		updates["user2_total_time_ms"] = timeMs
	}
	updates["last_activity_at"] = time.Now()

	result := r.db.Model(&models.QuizMatch{}).Where("id = ?", matchID).Updates(updates)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update quiz match score")
	}

	return nil
}

// UpdateRoundStats updates the round statistics
func (r *QuizMatchRepository) UpdateRoundStats(roundID, userID uint, correctCount, timeMs int) error {
	var round models.QuizRound
	if err := r.db.First(&round, roundID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get round")
	}

	var match models.QuizMatch
	if err := r.db.First(&match, round.MatchID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get match")
	}

	updates := make(map[string]interface{})
	if match.User1ID == userID {
		updates["user1_correct_count"] = correctCount
		updates["user1_time_ms"] = timeMs
	} else {
		updates["user2_correct_count"] = correctCount
		updates["user2_time_ms"] = timeMs
	}

	result := r.db.Model(&models.QuizRound{}).Where("id = ?", roundID).Updates(updates)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to update round stats")
	}

	return nil
}

// FinishQuizMatch marks a match as finished and sets the winner
func (r *QuizMatchRepository) FinishQuizMatch(matchID, winnerID uint) error {
	now := time.Now()
	updates := map[string]interface{}{
		"state":       models.QuizStateGameFinished,
		"finished_at": now,
	}

	if winnerID > 0 {
		updates["winner_id"] = winnerID
	}

	result := r.db.Model(&models.QuizMatch{}).Where("id = ?", matchID).Updates(updates)

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to finish quiz match")
	}

	return nil
}

// TimeoutQuizMatch marks a match as timed out
func (r *QuizMatchRepository) TimeoutQuizMatch(matchID uint) error {
	now := time.Now()
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Updates(map[string]interface{}{
			"state":       models.QuizStateTimeout,
			"finished_at": now,
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to timeout quiz match")
	}

	return nil
}

// GetTimeoutMatches retrieves matches that have exceeded the timeout period
func (r *QuizMatchRepository) GetTimeoutMatches() ([]models.QuizMatch, error) {
	var matches []models.QuizMatch
	result := r.db.Where("timeout_at < ? AND state NOT IN (?, ?)",
		time.Now(), models.QuizStateGameFinished, models.QuizStateTimeout).
		Preload("User1").Preload("User2").
		Find(&matches)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get timeout matches")
	}

	return matches, nil
}

// GetUserBooster retrieves a user's booster
func (r *QuizMatchRepository) GetUserBooster(userID uint, boosterType string) (*models.UserBooster, error) {
	var booster models.UserBooster
	result := r.db.Where("user_id = ? AND booster_type = ?", userID, boosterType).First(&booster)

	if result.Error == gorm.ErrRecordNotFound {
		return &models.UserBooster{
			UserID:      userID,
			BoosterType: boosterType,
			Quantity:    0,
		}, nil
	}
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get user booster")
	}

	return &booster, nil
}

// UseBooster decrements a user's booster quantity
func (r *QuizMatchRepository) UseBooster(userID uint, boosterType string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var booster models.UserBooster
		err := tx.Where("user_id = ? AND booster_type = ?", userID, boosterType).
			First(&booster).Error

		if err == gorm.ErrRecordNotFound {
			return errors.New(errors.ErrCodeNotFound, "booster not found")
		}
		if err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get booster")
		}

		if booster.Quantity <= 0 {
			return errors.New(errors.ErrCodeNotFound, "insufficient booster quantity")
		}

		result := tx.Model(&booster).Update("quantity", gorm.Expr("quantity - 1"))
		if result.Error != nil {
			return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to use booster")
		}

		return nil
	})
}

// AddBooster adds boosters to a user's inventory
func (r *QuizMatchRepository) AddBooster(userID uint, boosterType string, quantity int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var booster models.UserBooster
		err := tx.Where("user_id = ? AND booster_type = ?", userID, boosterType).
			First(&booster).Error

		if err == gorm.ErrRecordNotFound {
			// Create new booster entry
			booster = models.UserBooster{
				UserID:      userID,
				BoosterType: boosterType,
				Quantity:    quantity,
			}
			if err := tx.Create(&booster).Error; err != nil {
				return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create booster")
			}
			return nil
		}

		if err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get booster")
		}

		// Update existing booster
		result := tx.Model(&booster).Update("quantity", gorm.Expr("quantity + ?", quantity))
		if result.Error != nil {
			return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to add booster")
		}

		return nil
	})
}

// SwitchTurn switches the turn to the other user
func (r *QuizMatchRepository) SwitchTurn(matchID uint) error {
	var match models.QuizMatch
	if err := r.db.First(&match, matchID).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get match")
	}

	var newTurnUserID uint
	if match.TurnUserID != nil && *match.TurnUserID == match.User1ID {
		newTurnUserID = match.User2ID
	} else {
		newTurnUserID = match.User1ID
	}

	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Updates(map[string]interface{}{
			"turn_user_id":     newTurnUserID,
			"last_activity_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to switch turn")
	}

	return nil
}

// AdvanceRound advances to the next round
func (r *QuizMatchRepository) AdvanceRound(matchID uint) error {
	result := r.db.Model(&models.QuizMatch{}).
		Where("id = ?", matchID).
		Updates(map[string]interface{}{
			"current_round":    gorm.Expr("current_round + 1"),
			"current_question": 0,
			"state":            models.QuizStateWaitingCategory,
			"last_activity_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to advance round")
	}

	return nil
}
