package telegram

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleQuizCallbacks handles all Quiz related callbacks
func (b *Bot) HandleQuizCallbacks(query *tgbotapi.CallbackQuery, data string) bool {
	userID := query.From.ID

	// Normalize data by removing btn: prefix if present for technical matching
	cmd := strings.TrimPrefix(data, "btn:")

	// ========================================
	// NEW QUIZ GAME CALLBACKS
	// ========================================

	// Show active quiz games (Glass Menu)
	if cmd == "quiz_games" {
		b.handlers.ShowActiveQuizGames(userID, b)
		return true
	}

	// New quiz game
	if cmd == "new_quiz_game" {
		b.handlers.StartNewQuizGame(userID, b)
		return true
	}

	// Cancel quiz matchmaking
	if cmd == "cancel_quiz_matchmaking" {
		b.handlers.CancelQuizMatchmaking(userID, b)
		return true
	}

	// Game detail
	if strings.HasPrefix(cmd, "qgame_") {
		var matchID uint
		fmt.Sscanf(cmd, "qgame_%d", &matchID)
		b.handlers.ShowQuizGameDetail(userID, matchID, b)
		return true
	}

	// Start round (category selection)
	if strings.HasPrefix(cmd, "qstart_") {
		var matchID uint
		fmt.Sscanf(cmd, "qstart_%d", &matchID)
		b.handlers.ShowCategorySelection(userID, matchID, b)
		return true
	}

	// Start playing questions (asynchronous)
	if strings.HasPrefix(cmd, "qplay_") {
		var matchID uint
		fmt.Sscanf(cmd, "qplay_%d", &matchID)
		b.handlers.HandleQuizPlay(userID, matchID, b)
		return true
	}

	// Category selection
	if strings.HasPrefix(cmd, "qcat_") {
		parts := strings.SplitN(cmd, "_", 3)
		if len(parts) == 3 {
			var matchID uint
			fmt.Sscanf(parts[1], "%d", &matchID)
			category := parts[2]
			b.handlers.HandleCategorySelection(userID, matchID, category, b)
		} else {
			// Older format or fallback
			var matchID uint
			fmt.Sscanf(cmd, "qcat_%d", &matchID)
			partsAlt := strings.Split(cmd, "_")
			if len(partsAlt) >= 3 {
				category := strings.Join(partsAlt[2:], "_")
				b.handlers.HandleCategorySelection(userID, matchID, category, b)
			}
		}
		return true
	}

	// Answer selection
	if strings.HasPrefix(cmd, "qans_") {
		var matchID uint
		var qIdx, answerIndex int
		fmt.Sscanf(cmd, "qans_%d_%d_%d", &matchID, &qIdx, &answerIndex)
		b.handlers.HandleQuizAnswer(userID, matchID, qIdx, answerIndex, b)
		return true
	}

	// Booster: Remove 2 options
	if strings.HasPrefix(cmd, "qboost_r2_") {
		var matchID uint
		var questionNum int
		fmt.Sscanf(cmd, "qboost_r2_%d_%d", &matchID, &questionNum)
		b.handlers.HandleBoosterRemove2(userID, matchID, questionNum, b)
		return true
	}

	// Booster: Retry
	if strings.HasPrefix(cmd, "qboost_rt_") {
		var matchID uint
		var questionNum int
		fmt.Sscanf(cmd, "qboost_rt_%d_%d", &matchID, &questionNum)
		b.handlers.HandleBoosterRetry(userID, matchID, questionNum, b)
		return true
	}

	// Notify opponent
	if strings.HasPrefix(cmd, "qnotify_") {
		var matchID uint
		fmt.Sscanf(cmd, "qnotify_%d", &matchID)
		b.handlers.NotifyQuizOpponent(userID, matchID, b)
		return true
	}

	// Legacy/Old Quiz Callbacks
	if strings.HasPrefix(cmd, "quiz_") && !strings.HasPrefix(cmd, "quiz_games") {
		var matchID uint
		var answerIdx int
		fmt.Sscanf(cmd, "quiz_%d_%d", &matchID, &answerIdx)
		b.handlers.HandleQuizAnswer(userID, matchID, 0, answerIdx, b)
		return true
	}

	// Quiz of King (Group) Callbacks
	if strings.HasPrefix(cmd, "qok_start_") {
		var roomID uint
		fmt.Sscanf(cmd, "qok_start_%d", &roomID)
		b.handlers.StartQuizGame(userID, roomID, b)
		return true
	}

	if strings.HasPrefix(cmd, "qok_ans_") {
		var sessionID, questionID uint
		var answerIdx int
		fmt.Sscanf(cmd, "qok_ans_%d_%d_%d", &sessionID, &questionID, &answerIdx)
		msgID := 0
		if query.Message != nil {
			msgID = query.Message.MessageID
		}
		b.handlers.HandleQuizGameAnswer(userID, msgID, sessionID, questionID, answerIdx, b)
		return true
	}

	return false
}
