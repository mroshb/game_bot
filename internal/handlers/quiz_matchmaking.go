package handlers

import (
	"fmt"
	"time"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// StartQuizMatchmaking starts the matchmaking process for quiz games
func (h *HandlerManager) StartQuizMatchmaking(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	// Check if user already has an active quiz match
	activeMatch, _ := h.QuizMatchRepo.GetActiveQuizMatchByUser(user.ID)
	if activeMatch != nil {
		bot.SendMessage(userID, "⚠️ شما یک بازی فعال دارید! ابتدا آن را تمام کنید.", nil)
		h.ShowQuizGameDetail(userID, activeMatch.ID, bot)
		return
	}

	// Check if user is already in matchmaking queue
	inQueue, _ := h.MatchRepo.IsUserInQueue(user.ID)
	if inQueue {
		bot.SendMessage(userID, "⏳ شما در حال حاضر در صف matchmaking هستید!\n\nلطفاً صبر کنید تا حریف پیدا شود...", nil)
		return
	}

	// Add user to matchmaking queue
	queue := &models.MatchmakingQueue{
		UserID:          user.ID,
		RequestedGender: models.RequestedGenderAny,
		CoinsPaid:       0,
		GameType:        models.GameTypeQuiz,
	}

	err = h.MatchRepo.AddToQueue(queue)
	if err != nil {
		logger.Error("Failed to add user to queue", "error", err)
		bot.SendMessage(userID, "❌ خطا در شروع matchmaking!", nil)
		return
	}

	// Update user status
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusSearching)

	// Send searching message
	bot.SendMessage(userID, "🔍 در حال جستجوی حریف برای بازی کوئیز...\n\n⏳ لطفاً صبر کنید...", nil)

	// Try to find a match immediately
	go h.tryQuizMatchmaking(user.ID, bot)
}

// tryQuizMatchmaking attempts to find a match for quiz game
func (h *HandlerManager) tryQuizMatchmaking(userID uint, bot BotInterface) {
	// Wait a bit to allow other users to join
	time.Sleep(2 * time.Second)

	// Get user from queue
	_, err := h.MatchRepo.GetQueueEntry(userID)
	if err != nil {
		// User might have cancelled
		return
	}

	// Try to find a match
	filters := &models.MatchFilters{
		Gender:   models.RequestedGenderAny,
		GameType: models.GameTypeQuiz,
	}
	opponent, err := h.MatchRepo.FindMatch(userID, filters)
	if err != nil || opponent == nil {
		// No match found yet, user stays in queue
		user, _ := h.UserRepo.GetUserByID(userID)
		if user != nil {
			bot.SendMessage(user.TelegramID, "⏳ هنوز حریفی پیدا نشد...\n\nشما در صف matchmaking هستید. به محض پیدا شدن حریف، بازی شروع می‌شود!", nil)
		}
		return
	}

	// Match found! Remove both users from queue
	h.MatchRepo.RemoveFromQueue(userID)
	h.MatchRepo.RemoveFromQueue(opponent.ID)

	// Create quiz match
	match, err := h.QuizMatchRepo.CreateQuizMatch(userID, opponent.ID)
	if err != nil {
		logger.Error("Failed to create quiz match", "error", err)
		user, _ := h.UserRepo.GetUserByID(userID)
		if user != nil {
			bot.SendMessage(user.TelegramID, "❌ خطا در ایجاد بازی!", nil)
		}
		bot.SendMessage(opponent.TelegramID, "❌ خطا در ایجاد بازی!", nil)

		// Update statuses back to online
		h.UserRepo.UpdateUserStatus(userID, models.UserStatusOnline)
		h.UserRepo.UpdateUserStatus(opponent.ID, models.UserStatusOnline)
		return
	}

	// Update both users' statuses
	h.UserRepo.UpdateUserStatus(userID, models.UserStatusInMatch)
	h.UserRepo.UpdateUserStatus(opponent.ID, models.UserStatusInMatch)

	// Notify both users
	user, _ := h.UserRepo.GetUserByID(userID)
	if user != nil {
		msg := fmt.Sprintf("🎉 حریف پیدا شد!\n\n🧠 بازی کوئیز با %s شروع شد!\n\n📊 شرایط بازی:\n▫️ %d راند %d سؤاله\n▫️ هر راند یک موضوع انتخابی\n▫️ برنده بر اساس جواب درست و سرعت مشخص میشه!\n\nآماده باش!", opponent.FullName, models.QuizTotalRounds, models.QuizQuestionsPerRound)
		bot.SendMessage(user.TelegramID, msg, nil)

		time.Sleep(2 * time.Second)
		h.ShowQuizGameDetail(user.TelegramID, match.ID, bot)
	}

	msg := fmt.Sprintf("🎉 حریف پیدا شد!\n\n🧠 بازی کوئیز با %s شروع شد!\n\n📊 شرایط بازی:\n▫️ %d راند %d سؤاله\n▫️ هر راند یک موضوع انتخابی\n▫️ برنده بر اساس جواب درست و سرعت مشخص میشه!\n\nآماده باش!", user.FullName, models.QuizTotalRounds, models.QuizQuestionsPerRound)
	bot.SendMessage(opponent.TelegramID, msg, nil)

	time.Sleep(2 * time.Second)
	h.ShowQuizGameDetail(opponent.TelegramID, match.ID, bot)
}

// CancelQuizMatchmaking cancels quiz matchmaking for a user
func (h *HandlerManager) CancelQuizMatchmaking(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	h.MatchRepo.RemoveFromQueue(user.ID)
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)

	bot.SendMessage(userID, "❌ جستجوی حریف لغو شد.", nil)
	h.ShowActiveQuizGames(userID, bot)
}
