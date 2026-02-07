package repositories

import (
	"fmt"
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
	"gorm.io/gorm"
)

type TodRepository struct {
	db *gorm.DB
}

func NewTodRepository(db *gorm.DB) *TodRepository {
	return &TodRepository{db: db}
}

// ========================================
// GAME CRUD OPERATIONS
// ========================================

// CreateGame creates a new ToD game linked to a match
func (r *TodRepository) CreateGame(matchID uint, player1ID, player2ID uint) (*models.TodGame, error) {
	now := time.Now()
	deadline := now.Add(60 * time.Second)

	game := &models.TodGame{
		MatchID:         matchID,
		State:           models.TodStateMatchmaking,
		ActivePlayerID:  player1ID,
		PassivePlayerID: player2ID,
		CurrentRound:    1,
		MaxRounds:       10,
		TurnStartedAt:   &now,
		TurnDeadline:    &deadline,
		AllowItems:      true,
		DifficultyLevel: "normal",
		StartedAt:       &now,
	}

	if err := r.db.Create(game).Error; err != nil {
		logger.Error("Failed to create ToD game", "error", err)
		return nil, err
	}

	// Preload match data
	r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").First(game, game.ID)

	return game, nil
}

// GetGameByID retrieves a game by ID with preloaded relationships
func (r *TodRepository) GetGameByID(gameID uint) (*models.TodGame, error) {
	var game models.TodGame
	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").First(&game, gameID).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// GetGameByMatchID retrieves a game by match ID
func (r *TodRepository) GetGameByMatchID(matchID uint) (*models.TodGame, error) {
	var game models.TodGame
	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").
		Where("match_id = ?", matchID).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// GetActiveGameForUser retrieves active game for a user
func (r *TodRepository) GetActiveGameForUser(userID uint) (*models.TodGame, error) {
	var game models.TodGame
	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").
		Where("(active_player_id = ? OR passive_player_id = ?) AND state NOT IN (?, ?)",
			userID, userID, models.TodStateGameEnd, models.TodStateForfeit).
		First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// UpdateGameState updates the game state
func (r *TodRepository) UpdateGameState(gameID uint, newState string) error {
	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		Update("state", newState).
		Update("updated_at", time.Now()).Error
}

// ========================================
// TURN MANAGEMENT
// ========================================

// CreateTurn creates a new turn
func (r *TodRepository) CreateTurn(gameID uint, playerID, judgeID uint, roundNum int) (*models.TodTurn, error) {
	turn := &models.TodTurn{
		GameID:      gameID,
		RoundNumber: roundNum,
		PlayerID:    playerID,
		JudgeID:     judgeID,
	}

	if err := r.db.Create(turn).Error; err != nil {
		logger.Error("Failed to create turn", "error", err)
		return nil, err
	}

	// Update game's current_turn_id
	r.db.Model(&models.TodGame{}).Where("id = ?", gameID).Update("current_turn_id", turn.ID)

	return turn, nil
}

// GetCurrentTurn retrieves the current turn for a game
func (r *TodRepository) GetCurrentTurn(gameID uint) (*models.TodTurn, error) {
	var game models.TodGame
	if err := r.db.First(&game, gameID).Error; err != nil {
		return nil, err
	}

	if game.CurrentTurnID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var turn models.TodTurn
	err := r.db.Preload("Challenge").First(&turn, game.CurrentTurnID).Error
	return &turn, err
}

// UpdateTurnChoice updates the choice made in a turn
func (r *TodRepository) UpdateTurnChoice(turnID uint, choice string) error {
	now := time.Now()
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Updates(map[string]interface{}{
			"choice":    choice,
			"chosen_at": now,
		}).Error
}

// UpdateTurnChallenge updates the challenge for a turn
func (r *TodRepository) UpdateTurnChallenge(turnID uint, challengeID uint, challengeText string) error {
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Updates(map[string]interface{}{
			"challenge_id":   challengeID,
			"challenge_text": challengeText,
		}).Error
}

// UpdateTurnProof updates the proof submission
func (r *TodRepository) UpdateTurnProof(turnID uint, proofType, proofData string) error {
	now := time.Now()
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Updates(map[string]interface{}{
			"proof_type":         proofType,
			"proof_data":         proofData,
			"proof_submitted_at": now,
		}).Error
}

// UpdateTurnJudgment updates the judgment result
func (r *TodRepository) UpdateTurnJudgment(turnID uint, result, reason string) error {
	now := time.Now()
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Updates(map[string]interface{}{
			"judgment_result": result,
			"judgment_reason": reason,
			"judged_at":       now,
		}).Error
}

// CompleteTurn marks a turn as completed
func (r *TodRepository) CompleteTurn(turnID uint) error {
	now := time.Now()
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Update("completed_at", now).Error
}

// UpdateTurnRewards updates XP and coins awarded
func (r *TodRepository) UpdateTurnRewards(turnID uint, xp, coins int) error {
	return r.db.Model(&models.TodTurn{}).Where("id = ?", turnID).
		Updates(map[string]interface{}{
			"xp_awarded":    xp,
			"coins_awarded": coins,
		}).Error
}

// ========================================
// CHALLENGE SELECTION
// ========================================

// GetRandomChallenge retrieves a random challenge based on filters
func (r *TodRepository) GetRandomChallenge(challengeType, difficulty, category, gender, relation string) (*models.TodChallenge, error) {
	var challenge models.TodChallenge

	// Base query
	query := r.db.Where("type = ? AND is_active = ?", challengeType, true)

	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}

	// Temporarily remove category filter to increase match rate
	// if category != "" {
	// 	query = query.Where("category = ?", category)
	// }

	if gender != "" && gender != "all" {
		query = query.Where("gender_target IN (?)", []string{gender, "all"})
	}

	if relation != "" {
		// query = query.Where("relation_level = ?", relation)
	}

	err := query.Order("RANDOM()").First(&challenge).Error
	if err != nil {
		// Fallback: try without filters
		err = r.db.Where("type = ? AND is_active = ?", challengeType, true).
			Order("RANDOM()").First(&challenge).Error
	}

	return &challenge, err
}

// GetChallengeByID retrieves a specific challenge
func (r *TodRepository) GetChallengeByID(challengeID uint) (*models.TodChallenge, error) {
	var challenge models.TodChallenge
	err := r.db.First(&challenge, challengeID).Error
	return &challenge, err
}

// IncrementChallengeUsage increments the usage counter
func (r *TodRepository) IncrementChallengeUsage(challengeID uint) error {
	return r.db.Model(&models.TodChallenge{}).Where("id = ?", challengeID).
		UpdateColumn("times_used", gorm.Expr("times_used + 1")).Error
}

// UpdateChallengeAcceptanceRate updates the acceptance rate
func (r *TodRepository) UpdateChallengeAcceptanceRate(challengeID uint, wasAccepted bool) error {
	var challenge models.TodChallenge
	if err := r.db.First(&challenge, challengeID).Error; err != nil {
		return err
	}

	totalJudgments := challenge.TimesUsed
	if totalJudgments == 0 {
		totalJudgments = 1
	}

	currentAccepted := challenge.AcceptanceRate * float64(totalJudgments-1)
	if wasAccepted {
		currentAccepted++
	}

	newRate := currentAccepted / float64(totalJudgments)

	return r.db.Model(&models.TodChallenge{}).Where("id = ?", challengeID).
		Update("acceptance_rate", newRate).Error
}

// ========================================
// TIMER & TIMEOUT
// ========================================

// SetTurnDeadline sets the deadline for current turn
func (r *TodRepository) SetTurnDeadline(gameID uint, deadline time.Time) error {
	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		Updates(map[string]interface{}{
			"turn_deadline":    deadline,
			"warning_shown_at": nil, // Reset warning
		}).Error
}

// GetGamesNearingTimeout retrieves games approaching 30s warning
func (r *TodRepository) GetGamesNearingTimeout() ([]*models.TodGame, error) {
	var games []*models.TodGame
	warningTime := time.Now().Add(30 * time.Second)

	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").
		Where("state IN (?, ?, ?) AND turn_deadline IS NOT NULL AND turn_deadline <= ? AND warning_shown_at IS NULL",
			models.TodStateWaitingChoice, models.TodStateWaitingProof, models.TodStateWaitingJudgment,
			warningTime).
		Find(&games).Error

	return games, err
}

// GetTimedOutGames retrieves games that have exceeded deadline
func (r *TodRepository) GetTimedOutGames() ([]*models.TodGame, error) {
	var games []*models.TodGame
	now := time.Now()

	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").
		Where("state IN (?, ?, ?) AND turn_deadline IS NOT NULL AND turn_deadline < ?",
			models.TodStateWaitingChoice, models.TodStateWaitingProof, models.TodStateWaitingJudgment,
			now).
		Find(&games).Error

	return games, err
}

// MarkWarningShown marks that warning has been shown
func (r *TodRepository) MarkWarningShown(gameID uint) error {
	now := time.Now()
	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		Update("warning_shown_at", now).Error
}

// HandleTimeout handles game timeout
func (r *TodRepository) HandleTimeout(gameID uint) error {
	now := time.Now()
	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		Updates(map[string]interface{}{
			"state":      models.TodStateForfeit,
			"ended_at":   now,
			"end_reason": "timeout",
		}).Error
}

// ========================================
// PLAYER STATS
// ========================================

// GetOrCreatePlayerStats retrieves or creates player stats
func (r *TodRepository) GetOrCreatePlayerStats(userID uint) (*models.TodPlayerStats, error) {
	var stats models.TodPlayerStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error

	if err == gorm.ErrRecordNotFound {
		stats = models.TodPlayerStats{
			UserID:       userID,
			JudgeScore:   100.0,
			ShieldsOwned: 1,
			SwapsOwned:   1,
			MirrorsOwned: 1,
		}
		if err := r.db.Create(&stats).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &stats, nil
}

// UpdatePlayerStats updates player statistics
func (r *TodRepository) UpdatePlayerStats(userID uint, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.Model(&models.TodPlayerStats{}).Where("user_id = ?", userID).
		Updates(updates).Error
}

// IncrementGamesPlayed increments games played counter
func (r *TodRepository) IncrementGamesPlayed(userID uint, won bool) error {
	updates := map[string]interface{}{
		"games_played": gorm.Expr("games_played + 1"),
	}
	if won {
		updates["games_won"] = gorm.Expr("games_won + 1")
	} else {
		updates["games_lost"] = gorm.Expr("games_lost + 1")
	}

	return r.UpdatePlayerStats(userID, updates)
}

// IncrementChallengeCompleted increments challenge stats
func (r *TodRepository) IncrementChallengeCompleted(userID uint, choiceType string, wasAccepted bool) error {
	updates := map[string]interface{}{}

	if choiceType == models.TodTypeTruth {
		updates["truths_chosen"] = gorm.Expr("truths_chosen + 1")
	} else if choiceType == models.TodTypeDare {
		updates["dares_chosen"] = gorm.Expr("dares_chosen + 1")
	}

	if wasAccepted {
		updates["challenges_completed"] = gorm.Expr("challenges_completed + 1")
	} else {
		updates["challenges_failed"] = gorm.Expr("challenges_failed + 1")
	}

	return r.UpdatePlayerStats(userID, updates)
}

// UpdateJudgeScore updates judge score
func (r *TodRepository) UpdateJudgeScore(userID uint, newScore float64) error {
	return r.UpdatePlayerStats(userID, map[string]interface{}{
		"judge_score": newScore,
	})
}

// UseItem decrements item count
func (r *TodRepository) UseItem(userID uint, itemType string) error {
	var field string
	switch itemType {
	case models.ItemTypeShield:
		field = "shields_owned"
	case models.ItemTypeSwap:
		field = "swaps_owned"
	case models.ItemTypeMirror:
		field = "mirrors_owned"
	default:
		return fmt.Errorf("invalid item type: %s", itemType)
	}

	// Check if user has item
	var stats models.TodPlayerStats
	if err := r.db.Where("user_id = ?", userID).First(&stats).Error; err != nil {
		return err
	}

	var currentCount int
	switch itemType {
	case models.ItemTypeShield:
		currentCount = stats.ShieldsOwned
	case models.ItemTypeSwap:
		currentCount = stats.SwapsOwned
	case models.ItemTypeMirror:
		currentCount = stats.MirrorsOwned
	}

	if currentCount <= 0 {
		return fmt.Errorf("insufficient items")
	}

	// Decrement
	return r.UpdatePlayerStats(userID, map[string]interface{}{
		field:        gorm.Expr(field + " - 1"),
		"items_used": gorm.Expr("items_used + 1"),
	})
}

// AddItem adds items to inventory
func (r *TodRepository) AddItem(userID uint, itemType string, count int) error {
	var field string
	switch itemType {
	case models.ItemTypeShield:
		field = "shields_owned"
	case models.ItemTypeSwap:
		field = "swaps_owned"
	case models.ItemTypeMirror:
		field = "mirrors_owned"
	default:
		return fmt.Errorf("invalid item type: %s", itemType)
	}

	return r.UpdatePlayerStats(userID, map[string]interface{}{
		field: gorm.Expr(fmt.Sprintf("%s + %d", field, count)),
	})
}

// ========================================
// JUDGMENT & ANTI-ABUSE
// ========================================

// LogJudgment creates a judgment log entry
func (r *TodRepository) LogJudgment(turnID, judgeID, playerID uint, result string) error {
	log := &models.TodJudgmentLog{
		TurnID:   turnID,
		JudgeID:  judgeID,
		PlayerID: playerID,
		Result:   result,
	}

	if err := r.db.Create(log).Error; err != nil {
		logger.Error("Failed to create judgment log", "error", err)
		return err
	}

	// Update judge stats
	updates := map[string]interface{}{
		"judgments_made": gorm.Expr("judgments_made + 1"),
	}
	if result == "accepted" {
		updates["judgments_accepted"] = gorm.Expr("judgments_accepted + 1")
	} else {
		updates["judgments_rejected"] = gorm.Expr("judgments_rejected + 1")
	}

	return r.UpdatePlayerStats(judgeID, updates)
}

// GetRecentJudgments retrieves recent judgments by a judge
func (r *TodRepository) GetRecentJudgments(judgeID uint, limit int) ([]*models.TodJudgmentLog, error) {
	var logs []*models.TodJudgmentLog
	err := r.db.Where("judge_id = ?", judgeID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// DetectUnfairJudgment detects unfair judgment patterns
func (r *TodRepository) DetectUnfairJudgment(judgeID uint) (bool, string, error) {
	logs, err := r.GetRecentJudgments(judgeID, 10)
	if err != nil {
		return false, "", err
	}

	if len(logs) < 5 {
		return false, "", nil // Not enough data
	}

	// Pattern 1: Rejected last 8 out of 10
	rejectedCount := 0
	for _, log := range logs {
		if log.Result == "rejected" {
			rejectedCount++
		}
	}
	if rejectedCount >= 8 {
		return true, "رد متوالی بیش از حد", nil
	}

	// Pattern 2: All rejections in last 5 games
	if len(logs) >= 5 {
		allRejected := true
		for i := 0; i < 5; i++ {
			if logs[i].Result == "accepted" {
				allRejected = false
				break
			}
		}
		if allRejected {
			return true, "رد ۵ مورد متوالی", nil
		}
	}

	return false, "", nil
}

// IncrementUnfairJudgmentCount increments unfair judgment counter
func (r *TodRepository) IncrementUnfairJudgmentCount(judgeID uint) error {
	return r.UpdatePlayerStats(judgeID, map[string]interface{}{
		"unfair_judgment_count": gorm.Expr("unfair_judgment_count + 1"),
	})
}

// CalculateJudgeScore calculates judge score based on recent judgments
func (r *TodRepository) CalculateJudgeScore(judgeID uint) (float64, error) {
	logs, err := r.GetRecentJudgments(judgeID, 20)
	if err != nil {
		return 100.0, err
	}

	if len(logs) < 5 {
		return 100.0, nil // Not enough data
	}

	acceptedCount := 0
	for _, log := range logs {
		if log.Result == "accepted" {
			acceptedCount++
		}
	}

	acceptanceRate := float64(acceptedCount) / float64(len(logs))

	// Score based on acceptance rate
	var score float64
	if acceptanceRate < 0.3 {
		score = 20.0 // Very suspicious
	} else if acceptanceRate < 0.5 {
		score = 50.0 // Suspicious
	} else if acceptanceRate >= 0.7 && acceptanceRate <= 0.9 {
		score = 100.0 // Perfect
	} else if acceptanceRate > 0.9 {
		score = 80.0 // Too lenient
	} else {
		score = 70.0 // Acceptable
	}

	return score, nil
}

// ========================================
// GAME END
// ========================================

// EndGame ends a game and sets winner
func (r *TodRepository) EndGame(gameID uint, winnerID uint, reason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"state":      models.TodStateGameEnd,
		"ended_at":   now,
		"end_reason": reason,
	}

	if winnerID > 0 {
		updates["winner_id"] = winnerID
	}

	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).Updates(updates).Error
}

// GetGameHistory retrieves game history for a user
func (r *TodRepository) GetGameHistory(userID uint, limit int) ([]*models.TodGame, error) {
	var games []*models.TodGame
	err := r.db.Preload("Match").Preload("Match.User1").Preload("Match.User2").
		Where("(active_player_id = ? OR passive_player_id = ?) AND state = ?",
			userID, userID, models.TodStateGameEnd).
		Order("ended_at DESC").
		Limit(limit).
		Find(&games).Error
	return games, err
}

// SwitchTurn switches active and passive players
func (r *TodRepository) SwitchTurn(gameID uint) error {
	var game models.TodGame
	if err := r.db.First(&game, gameID).Error; err != nil {
		return err
	}

	now := time.Now()
	deadline := now.Add(60 * time.Second)

	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		Updates(map[string]interface{}{
			"active_player_id":  game.PassivePlayerID,
			"passive_player_id": game.ActivePlayerID,
			"turn_started_at":   now,
			"turn_deadline":     deadline,
			"warning_shown_at":  nil,
		}).Error
}

// IncrementRound increments the current round
func (r *TodRepository) IncrementRound(gameID uint) error {
	return r.db.Model(&models.TodGame{}).Where("id = ?", gameID).
		UpdateColumn("current_round", gorm.Expr("current_round + 1")).Error
}

// ========================================
// IDEMPOTENCY
// ========================================

// IsActionProcessed checks if action has been processed
func (r *TodRepository) IsActionProcessed(gameID uint, actionID string) bool {
	var count int64
	r.db.Model(&models.TodActionLog{}).
		Where("game_id = ? AND action_id = ?", gameID, actionID).
		Count(&count)
	return count > 0
}

// MarkActionProcessed marks an action as processed
func (r *TodRepository) MarkActionProcessed(gameID uint, userID uint, actionID, action string) error {
	log := &models.TodActionLog{
		GameID:   gameID,
		UserID:   userID,
		ActionID: actionID,
		Action:   action,
	}
	return r.db.Create(log).Error
}

// CleanupOldActions removes old action logs (older than 24 hours)
func (r *TodRepository) CleanupOldActions() error {
	cutoff := time.Now().Add(-24 * time.Hour)
	return r.db.Where("created_at < ?", cutoff).Delete(&models.TodActionLog{}).Error
}
