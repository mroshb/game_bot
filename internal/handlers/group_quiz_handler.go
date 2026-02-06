// Group Quiz Game (Quiz of King) - Stub handlers for group quiz games

package handlers

import (
	"github.com/mroshb/game_bot/pkg/logger"
)

// StartQuizGame - Group quiz game starter (Quiz of King)
// TODO: Implement group quiz game logic
func (h *HandlerManager) StartQuizGame(userID int64, roomID uint, bot BotInterface) {
	logger.Info("Group quiz game requested but not yet implemented", "user_id", userID, "room_id", roomID)
	bot.SendMessage(userID, "🚧 بازی کوئیز گروهی هنوز در دست توسعه است!", nil)
}

// HandleQuizGameAnswer - Group quiz game answer handler
// TODO: Implement group quiz game answer logic
func (h *HandlerManager) HandleQuizGameAnswer(userID int64, msgID int, sessionID, questionID uint, answerIdx int, bot BotInterface) {
	logger.Info("Group quiz answer received but not yet implemented", "user_id", userID, "session_id", sessionID)
	bot.SendMessage(userID, "🚧 این قابلیت هنوز در دست توسعه است!", nil)
}
