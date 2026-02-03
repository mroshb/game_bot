package handlers

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

func (h *HandlerManager) StartMatchmaking(userID int64, requestedGender string, session *UserSession, bot BotInterface) {
	// Get user
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if already in queue
	queueEntry, err := h.MatchRepo.GetQueueEntry(user.ID)
	if err == nil && queueEntry != nil {
		// User is already in queue. Check if a goroutine is already running.
		if _, active := h.searchingUsers.Load(user.ID); active {
			bot.SendMessage(userID, "🔍 همین الان در حال جستجو هستیم...", nil)
			return
		}
		// Goroutine not running (e.g. after restart), start it.
		bot.SendMessage(userID, "🔍 در حال ادامه جستجو برای شما...", nil)
		h.searchingUsers.Store(user.ID, true)
		go h.findMatch(user.ID, queueEntry, bot)
		return
	}

	// Check if already in a match
	activeMatch, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil {
		logger.Error("Failed to check active match", "error", err)
	}

	if activeMatch != nil {
		bot.SendMessage(userID, "⚠️ شما الان در یک چت فعال هستید!", ChatKeyboard())
		return
	}

	// Check sufficient coins
	hasFunds, err := h.CoinRepo.HasSufficientBalance(user.ID, h.Config.MatchCostCoins)
	if err != nil || !hasFunds {
		msg := fmt.Sprintf("❌ سکه کافی نداری!\n\n💰 موجودی: %d\n💰 نیاز: %d", user.CoinBalance, h.Config.MatchCostCoins)
		bot.SendMessage(userID, msg, nil)
		return
	}

	// Deduct coins
	if err := h.CoinRepo.DeductCoins(user.ID, h.Config.MatchCostCoins, models.TxTypeMatchmaking, "هزینه جستجوی match"); err != nil {
		logger.Error("Failed to deduct coins", "error", err)
		bot.SendMessage(userID, "❌ خطا در کسر سکه!", nil)
		return
	}

	// Add to queue
	queue := &models.MatchmakingQueue{
		UserID:          user.ID,
		RequestedGender: requestedGender,
		CoinsPaid:       h.Config.MatchCostCoins,
	}

	// Apply filters from session
	if minAge, ok := session.Data["min_age"].(int); ok {
		queue.MinAge = &minAge
	}
	if maxAge, ok := session.Data["max_age"].(int); ok {
		queue.MaxAge = &maxAge
	}
	if city, ok := session.Data["search_city"].(string); ok && city != "" {
		queue.City = city
	}

	if err := h.MatchRepo.AddToQueue(queue); err != nil {
		logger.Error("Failed to add to queue", "error", err)
		// Refund coins
		h.CoinRepo.AddCoins(user.ID, h.Config.MatchCostCoins, models.TxTypeMatchRefund, "بازگشت هزینه به دلیل خطا")
		bot.SendMessage(userID, "❌ خطا در افزودن به صف جستجو!", nil)
		return
	}

	// Update user status
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusSearching)

	// Send searching message
	msg := fmt.Sprintf("🔍 جستجو شروع شد!\n\n💰 هزینه: %d سکه\n\nداریم دنبال یک نفر مناسب برات می‌گردیم...", h.Config.MatchCostCoins)

	// Create cancel keyboard explicitly here since we can't import telegram package
	cancelKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
		),
	)

	bot.SendMessage(userID, msg, cancelKeyboard)

	// Start matching process in background
	h.searchingUsers.Store(user.ID, true)
	go h.findMatch(user.ID, queue, bot)
}

func (h *HandlerManager) HandleSearchGenderSelection(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	// Handle Cancel
	if text == BtnCancel {
		user, _ := h.UserRepo.GetUserByTelegramID(userID)
		isAdmin := false
		if user != nil {
			isAdmin = user.TelegramID == h.Config.SuperAdminTgID
		}
		bot.SendMessage(userID, "❌ جستجو لغو شد.", bot.GetMainMenuKeyboard(isAdmin))
		session.State = "" // Clear state
		return
	}

	var requestedGender string
	switch text {
	case BtnMale:
		requestedGender = models.GenderMale
	case BtnFemale:
		requestedGender = models.GenderFemale
	case BtnAny:
		requestedGender = models.RequestedGenderAny
	default:
		bot.SendMessage(userID, "❌ لطفاً یکی از گزینه‌های لیست رو انتخاب کن!", nil)
		return
	}

	session.Data["search_gender"] = requestedGender
	session.State = StateSearchAge

	bot.SendMessage(userID, "🎂 محدوده سنی مورد نظرت رو وارد کن (مثال: 20-30) یا بزن رد شو:", SkipKeyboard())
}

func (h *HandlerManager) HandleSearchAgeSelection(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	if text == BtnCancel {
		user, _ := h.UserRepo.GetUserByTelegramID(userID)
		isAdmin := user != nil && user.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(userID, "❌ جستجو لغو شد.", bot.GetMainMenuKeyboard(isAdmin))
		session.State = ""
		return
	}

	if text != BtnSkip {
		var minAge, maxAge int
		_, err := fmt.Sscanf(text, "%d-%d", &minAge, &maxAge)
		if err != nil {
			// Try single number
			// If single number, maybe exact match or +/- range? Let's assume range format is required for simplicity or lenient parsing.
			// Let's enforce range or simple age.
			bot.SendMessage(userID, "❌ فرمت نامعتبر! لطفاً به صورت 20-30 وارد کن یا رد شو بزن.", nil)
			return
		}

		if minAge < 13 || maxAge > 100 || minAge > maxAge {
			bot.SendMessage(userID, "❌ محدوده سنی نامعتبر است!", nil)
			return
		}

		session.Data["min_age"] = minAge
		session.Data["max_age"] = maxAge
	}

	session.State = StateSearchCity
	bot.SendMessage(userID, "🏙 شهر مورد نظرت رو بنویس یا از لیست انتخاب کن (یا بزن رد شو):", ProvinceKeyboard())
}

func (h *HandlerManager) HandleSearchCitySelection(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	if text == BtnCancel {
		user, _ := h.UserRepo.GetUserByTelegramID(userID)
		isAdmin := user != nil && user.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(userID, "❌ جستجو لغو شد.", bot.GetMainMenuKeyboard(isAdmin))
		session.State = ""
		return
	}

	if text != BtnSkip {
		// Basic validation, maybe check against list of provinces but free text search is also fine for flexibility?
		// User list uses exact match on City field.
		// Let's validate against our known list if possible, or just accept.
		// Since we have CitySelectionKeyboard, user likely picks from it.
		// Let's accept whatever for flexibility but encourage list.
		session.Data["search_city"] = text
	}

	gender := session.Data["search_gender"].(string)
	session.State = "" // Clear state as we are now in "Searching" mode (handled by UserStatus)
	h.StartMatchmaking(userID, gender, session, bot)
}

func (h *HandlerManager) findMatch(userID uint, queue *models.MatchmakingQueue, bot BotInterface) {
	// Get user's telegram ID
	user, err := h.UserRepo.GetUserByID(userID)
	if err != nil {
		logger.Error("Failed to get user", "error", err)
		return
	}

	// Try to find match for up to timeout duration
	timeout := time.After(h.Config.GetMatchTimeout())
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer h.searchingUsers.Delete(userID)

	for {
		select {
		case <-timeout:
			// Match logic should check queue one last time or just timeout
			// But first check if still in queue (maybe cancelled)
			if inQueue, err := h.MatchRepo.IsUserInQueue(userID); err != nil || !inQueue {
				return
			}

			// Timeout - refund half coins and remove from queue
			h.handleQueueTimeout(userID, user.TelegramID, bot)
			return

		case <-ticker.C:
			// Check if user is still in queue
			// This prevents "zombie" searches if user cancelled
			inQueue, err := h.MatchRepo.IsUserInQueue(userID)
			if err != nil {
				logger.Error("Error checking queue status", "error", err)
				continue
			}
			if !inQueue {
				return // Stop searching if user removed from queue
			}

			// Try to find a match
			matchedUser, err := h.MatchRepo.FindMatch(userID, &models.MatchFilters{
				Gender: queue.RequestedGender,
				MinAge: queue.MinAge,
				MaxAge: queue.MaxAge,
				City:   queue.City,
			})

			if err != nil {
				logger.Error("Error finding match", "error", err)
				continue
			}

			if matchedUser != nil {
				// Found a match!
				h.createMatchSession(userID, matchedUser.ID, user.TelegramID, matchedUser.TelegramID, bot)
				return
			}
		}
	}
}

func (h *HandlerManager) createMatchSession(user1ID, user2ID uint, tg1ID, tg2ID int64, bot BotInterface) {
	// Remove both from queue
	h.MatchRepo.RemoveFromQueue(user1ID)
	h.MatchRepo.RemoveFromQueue(user2ID)

	// Create match session
	session, err := h.MatchRepo.CreateMatchSession(user1ID, user2ID, h.Config.GetMatchTimeout())
	if err != nil {
		logger.Error("Failed to create match session", "error", err)

		// Refund both users
		h.refundMatchCost(user1ID, tg1ID, bot)
		h.refundMatchCost(user2ID, tg2ID, bot)
		return
	}

	// Update both users' status
	h.UserRepo.UpdateUserStatus(user1ID, models.UserStatusInMatch)
	h.UserRepo.UpdateUserStatus(user2ID, models.UserStatusInMatch)

	// Notify both users
	msg := fmt.Sprintf("✅ پیدا شد!\n\nیک نفر پیدا کردیم! می‌تونی شروع به چت کنی.\n\n⏰ مدت زمان: %d دقیقه", h.Config.MatchTimeoutMinutes)
	keyboard := ChatKeyboard()

	bot.SendMessage(tg1ID, msg, keyboard)
	bot.SendMessage(tg2ID, msg, keyboard)

	logger.Info("Match created", "session_id", session.ID, "user1", user1ID, "user2", user2ID)
}

func ChatKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnQuiz, "btn:"+BtnQuiz),
			tgbotapi.NewInlineKeyboardButtonData(BtnTruthDare, "btn:"+BtnTruthDare),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnEndChat, "btn:"+BtnEndChat),
		),
	)
}

func (h *HandlerManager) handleQueueTimeout(userID uint, telegramID int64, bot BotInterface) {
	// Remove from queue
	queueEntry, err := h.MatchRepo.GetQueueEntry(userID)
	if err != nil {
		logger.Error("Failed to get queue entry", "error", err)
		return
	}

	h.MatchRepo.RemoveFromQueue(userID)

	// Refund half coins
	refundAmount := queueEntry.CoinsPaid / 2
	if err := h.CoinRepo.AddCoins(userID, refundAmount, models.TxTypeMatchRefund, "بازگشت نصف هزینه به دلیل timeout"); err != nil {
		logger.Error("Failed to refund coins", "error", err)
	}

	// Update status
	h.UserRepo.UpdateUserStatus(userID, models.UserStatusOnline)

	// Notify user
	msg := fmt.Sprintf("⏰ زمان تموم شد!\n\n💰 بازگشت: %d سکه (نصف هزینه)\n\nمتأسفانه کسی پیدا نشد.", refundAmount)

	user, _ := h.UserRepo.GetUserByID(userID)
	isAdmin := false
	if user != nil {
		isAdmin = user.TelegramID == h.Config.SuperAdminTgID
	}
	bot.SendMessage(telegramID, msg, bot.GetMainMenuKeyboard(isAdmin))

	logger.Info("Match timeout", "user_id", userID, "refund", refundAmount)
}

// HandleMatchTimeout handles notification for match session timeout (Active -> Timeout)
func (h *HandlerManager) HandleMatchTimeout(userID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByID(userID)
	if err != nil {
		return
	}

	bot.SendMessage(user.TelegramID, "⏰ زمان چت رایگان تمام شد!\n\n💬 می‌توانید به چت ادامه دهید (هزینه: 2 سکه هر پیام).", nil)
}

func (h *HandlerManager) refundMatchCost(userID uint, telegramID int64, bot BotInterface) {
	if err := h.CoinRepo.AddCoins(userID, h.Config.MatchCostCoins, models.TxTypeMatchRefund, "بازگشت هزینه به دلیل خطا"); err != nil {
		logger.Error("Failed to refund coins", "error", err)
	}

	h.UserRepo.UpdateUserStatus(userID, models.UserStatusOnline)

	user, _ := h.UserRepo.GetUserByID(userID)
	isAdmin := false
	if user != nil {
		isAdmin = user.TelegramID == h.Config.SuperAdminTgID
	}
	bot.SendMessage(telegramID, "❌ خطا در ایجاد match! سکه‌هات برگشت داده شد.", bot.GetMainMenuKeyboard(isAdmin))
}

func (h *HandlerManager) EndChat(userID int64, bot BotInterface) {
	// Get user
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	// Get active match
	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil {
		logger.Error("Failed to get active match", "error", err)
		isAdmin := user.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(userID, "❌ خطایی رخ داد!", bot.GetMainMenuKeyboard(isAdmin))
		return
	}

	if match == nil {
		isAdmin := user.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(userID, "⚠️ شما در چت فعالی نیستید!", bot.GetMainMenuKeyboard(isAdmin))
		return
	}

	// Get other user's telegram ID
	var otherUserID uint
	if match.User1ID == user.ID {
		otherUserID = match.User2ID
	} else {
		otherUserID = match.User1ID
	}

	otherUser, err := h.UserRepo.GetUserByID(otherUserID)
	if err != nil {
		logger.Error("Failed to get other user", "error", err)
	}

	// End match
	if err := h.MatchRepo.EndMatch(match.ID); err != nil {
		logger.Error("Failed to end match", "error", err)
		isAdmin := user.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(userID, "❌ خطا در پایان دادن چت!", bot.GetMainMenuKeyboard(isAdmin))
		return
	}

	// Update both users' status
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
	if otherUser != nil {
		h.UserRepo.UpdateUserStatus(otherUser.ID, models.UserStatusOnline)
	}

	// Award Village XP for finishing a chat
	h.VillageSvc.AddXPForUser(user.ID, 10)
	if otherUser != nil {
		h.VillageSvc.AddXPForUser(otherUser.ID, 10)
	}

	// Notify both users
	// Notify both users
	isAdmin := user.TelegramID == h.Config.SuperAdminTgID
	bot.SendMessage(userID, "👋 چت تموم شد!\n\nامیدواریم لذت برده باشی.", bot.GetMainMenuKeyboard(isAdmin))

	if otherUser != nil {
		otherIsAdmin := otherUser.TelegramID == h.Config.SuperAdminTgID
		bot.SendMessage(otherUser.TelegramID, "👋 طرف مقابل چت رو ترک کرد.", bot.GetMainMenuKeyboard(otherIsAdmin))
	}

	logger.Info("Match ended", "match_id", match.ID, "ended_by", user.ID)
}
