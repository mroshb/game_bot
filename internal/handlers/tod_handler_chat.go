package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// HandleTodChat initiates chat mode for a player in a ToD game
func (h *HandlerManager) HandleTodChat(userID int64, gameID uint, bot BotInterface) {
	// First verify the game exists and user is part of it
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		bot.SendMessage(userID, "❌ بازی یافت نشد.", nil)
		return
	}

	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	if game.Match.User1ID != user.ID && game.Match.User2ID != user.ID {
		bot.SendMessage(userID, "❌ شما عضو این بازی نیستید.", nil)
		return
	}

	msg := "💬 **حالت چت فعال شد:**\n\nهر پیامی بفرستی (متن، عکس، ویس و...) مستقیماً برای حریف ارسال میشه.\n\n👇 وقتی کارت تموم شد یا خواستی برگردی به بازی، دکمه زیر رو بزن:"

	// Keyboard to exit chat mode (though technically handled by main menu detection in bot.go,
	// providing a clear button is better).
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 بازگشت به بازی"),
		),
	)

	bot.SendMessage(userID, msg, keyboard)
}

// HandleTodChatMessage forwards a chat message in a ToD game
func (h *HandlerManager) HandleTodChatMessage(message *tgbotapi.Message, user *models.User, bot BotInterface) {
	// Check for exit command
	if message.Text == "🔙 بازگشت به بازی" {
		game, _ := h.TodRepo.GetActiveGameForUser(user.ID)
		if game != nil {
			bot.SendMessage(user.TelegramID, "✅ به محیط بازی برگشتی.", nil)
			h.ResumeTodGame(user.TelegramID, game.ID, bot)
			return
		}
	}

	// Find active game
	game, err := h.TodRepo.GetActiveGameForUser(user.ID)
	if err != nil || game == nil {
		bot.SendMessage(user.TelegramID, "❌ بازی فعالی یافت نشد.", nil)
		return
	}

	// Determine opponent
	var opponentTelegramID int64
	if game.Match.User1ID == user.ID {
		opponentTelegramID = game.Match.User2.TelegramID
	} else {
		opponentTelegramID = game.Match.User1.TelegramID
	}

	// Use the centralized forwardMessage utility
	if err := h.forwardMessage(message, opponentTelegramID, bot, user.FullName); err != nil {
		logger.Error("Failed to forward ToD chat message", "error", err)
		bot.SendMessage(user.TelegramID, "❌ خطا در ارسال پیام!", nil)
	}

	// Internal debug log
	logger.Debug("ToD message forwarded", "from", user.ID, "game", game.ID)
}
