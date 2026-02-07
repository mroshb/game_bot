package handlers

import (
	"fmt"
	"math/rand"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// ========================================
// MATCHMAKING
// ========================================

// StartTodMatchmaking starts matchmaking for Truth or Dare
func (h *HandlerManager) StartTodMatchmaking(userID int64, bot BotInterface) {
	// This function is now in tod_matchmaking.go
	// Keeping this for backward compatibility
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if already in active game
	activeGame, _ := h.TodRepo.GetActiveGameForUser(user.ID)
	if activeGame != nil {
		bot.SendMessage(userID, "⚠️ شما در یک بازی فعال هستید!", nil)
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
		GameType:        models.GameTypeTod,
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
	bot.SendMessage(userID, "🔍 در حال جستجوی حریف برای بازی جرعت و حقیقت...\n\n⏳ لطفاً صبر کنید...", nil)

	// Try to find a match immediately
	go h.tryTodMatchmaking(user.ID, bot)
}

// tryTodMatchmaking attempts to find a match for ToD game
func (h *HandlerManager) tryTodMatchmaking(userID uint, bot BotInterface) {
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
		GameType: models.GameTypeTod,
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

	// Create match session first (Required for ToD game)
	// We set timeout to 1 hour for game session
	matchSession, err := h.MatchRepo.CreateMatchSession(userID, opponent.ID, 1*time.Hour)
	if err != nil {
		logger.Error("Failed to create match session for ToD", "error", err)
		return
	}

	// Create ToD game
	game, err := h.TodRepo.CreateGame(matchSession.ID, userID, opponent.ID)
	if err != nil {
		logger.Error("Failed to create ToD game", "error", err)
		user, _ := h.UserRepo.GetUserByID(userID)
		if user != nil {
			bot.SendMessage(user.TelegramID, "❌ خطا در ایجاد بازی!", nil)
		}
		bot.SendMessage(opponent.TelegramID, "❌ خطا در ایجاد بازی!", nil)

		// End session and update statuses
		h.MatchRepo.EndMatch(matchSession.ID)
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
		msg := fmt.Sprintf("🎉 حریف پیدا شد!\n\n🔥 بازی جرعت و حقیقت با %s شروع شد!\n\nآماده باش!", opponent.FullName)
		bot.SendMessage(user.TelegramID, msg, nil)
	}

	msg := fmt.Sprintf("🎉 حریف پیدا شد!\n\n🔥 بازی جرعت و حقیقت با %s شروع شد!\n\nآماده باش!", user.FullName)
	bot.SendMessage(opponent.TelegramID, msg, nil)

	time.Sleep(2 * time.Second)

	// Start coin flip
	h.HandleTodCoinFlip(game.ID, bot)
}

// CancelTodMatchmaking cancels ToD matchmaking for a user
func (h *HandlerManager) CancelTodMatchmaking(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	h.MatchRepo.RemoveFromQueue(user.ID)
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)

	bot.SendMessage(userID, "❌ جستجوی حریف لغو شد.", nil)
}

// ========================================
// COIN FLIP
// ========================================

// HandleTodCoinFlip performs coin flip to determine first player
func (h *HandlerManager) HandleTodCoinFlip(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Show coin flip animation
	msg := "🎲 در حال قرعه‌کشی برای تعیین نوبت اول..."
	bot.SendMessage(game.Match.User1.TelegramID, msg, nil)
	bot.SendMessage(game.Match.User2.TelegramID, msg, nil)

	time.Sleep(2 * time.Second)

	// Random selection
	firstPlayer := game.ActivePlayerID
	secondPlayer := game.PassivePlayerID

	if rand.Intn(2) == 1 {
		firstPlayer, secondPlayer = secondPlayer, firstPlayer
		// Update game
		h.DB.Model(&models.TodGame{}).Where("id = ?", gameID).
			Updates(map[string]interface{}{
				"active_player_id":  firstPlayer,
				"passive_player_id": secondPlayer,
			})
	}

	var firstName, secondName string
	if firstPlayer == game.Match.User1ID {
		firstName = game.Match.User1.FullName
		secondName = game.Match.User2.FullName
	} else {
		firstName = game.Match.User2.FullName
		secondName = game.Match.User1.FullName
	}

	resultMsg := fmt.Sprintf("🎲 نتیجه قرعه‌کشی:\n\n🎯 نوبت اول: %s\n⏳ نوبت دوم: %s\n\nبازی شروع شد! 🎮\nراند 1 از %d",
		firstName, secondName, game.MaxRounds)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ ادامه", fmt.Sprintf("btn:tod_start_%d", gameID)),
		),
	)

	bot.SendMessage(game.Match.User1.TelegramID, resultMsg, keyboard)
	bot.SendMessage(game.Match.User2.TelegramID, resultMsg, keyboard)

	// Update state
	h.TodRepo.UpdateGameState(gameID, models.TodStateCoinFlip)
}

// HandleTodStart starts the game after coin flip
func (h *HandlerManager) HandleTodStart(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Create first turn
	_, err = h.TodRepo.CreateTurn(gameID, game.ActivePlayerID, game.PassivePlayerID, 1)
	if err != nil {
		logger.Error("Failed to create turn", "error", err)
		return
	}

	// Update state to waiting for choice
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingChoice)

	// Show choice screen
	h.ShowTodChoiceScreen(gameID, bot)
}

// ========================================
// CHOICE PHASE
// ========================================

// ShowTodChoiceScreen shows the choice screen to both players
func (h *HandlerManager) ShowTodChoiceScreen(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	if activeUser == nil || passiveUser == nil {
		return
	}

	// Calculate remaining time
	remainingSeconds := 60
	if game.TurnDeadline != nil {
		remaining := time.Until(*game.TurnDeadline)
		remainingSeconds = int(remaining.Seconds())
		if remainingSeconds < 0 {
			remainingSeconds = 0
		}
	}

	// Active player view
	activeMsg := fmt.Sprintf("🎮 راند %d/%d\n⏰ نوبت شما! (⏱ %d ثانیه)\n\n━━━━━━━━━━━━━━\nانتخاب کنید:",
		game.CurrentRound, game.MaxRounds, remainingSeconds)

	activeKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 جرئت", fmt.Sprintf("btn:tod_choice_%d_dare", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("🔵 حقیقت", fmt.Sprintf("btn:tod_choice_%d_truth", gameID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎒 آیتمها", fmt.Sprintf("btn:tod_items_%d", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("🏳️ انصراف", fmt.Sprintf("btn:tod_quit_%d", gameID)),
		),
	)

	bot.SendMessage(activeUser.TelegramID, activeMsg, activeKeyboard)

	// Passive player view
	passiveMsg := fmt.Sprintf("🎮 راند %d/%d\n⏳ حریف در حال انتخاب...\n\n━━━━━━━━━━━━━━\n👤 نوبت: %s\n\n⏱ زمان باقی‌مانده: %d ثانیه",
		game.CurrentRound, game.MaxRounds, activeUser.FullName, remainingSeconds)

	passiveKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💤 تلنگر", fmt.Sprintf("btn:tod_nudge_%d", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("💬 کلکل", fmt.Sprintf("btn:tod_chat_%d", gameID)),
		),
	)

	bot.SendMessage(passiveUser.TelegramID, passiveMsg, passiveKeyboard)
}

// HandleTodChoice handles truth or dare choice
func (h *HandlerManager) HandleTodChoice(userID int64, gameID uint, choice string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Verify it's user's turn
	if game.ActivePlayerID != user.ID {
		bot.SendMessage(userID, "⚠️ نوبت شما نیست!", nil)
		return
	}

	// Verify state
	if game.State != models.TodStateWaitingChoice {
		bot.SendMessage(userID, "⚠️ در این مرحله نمی‌توانید انتخاب کنید!", nil)
		return
	}

	// Generate action ID for idempotency
	actionID := uuid.New().String()
	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		return // Already processed
	}
	h.TodRepo.MarkActionProcessed(gameID, user.ID, actionID, "choice_"+choice)

	// Get current turn
	turn, err := h.TodRepo.GetCurrentTurn(gameID)
	if err != nil {
		logger.Error("Failed to get current turn", "error", err)
		return
	}

	// Update turn choice
	h.TodRepo.UpdateTurnChoice(turn.ID, choice)

	// Select challenge
	challenge, err := h.TodRepo.GetRandomChallenge(choice, "easy", "", user.Gender, "stranger")
	if err != nil {
		logger.Error("Failed to get challenge", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت چالش!", nil)
		return
	}

	// Update turn with challenge
	h.TodRepo.UpdateTurnChallenge(turn.ID, challenge.ID, challenge.Text)
	h.TodRepo.IncrementChallengeUsage(challenge.ID)

	// Update state
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingProof)

	// Show challenge
	h.ShowTodChallenge(gameID, challenge, bot)
}

// ShowTodChallenge shows the challenge to both players
func (h *HandlerManager) ShowTodChallenge(gameID uint, challenge *models.TodChallenge, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	if activeUser == nil || passiveUser == nil {
		return
	}

	choiceType := "حقیقت"
	if challenge.Type == models.TodTypeDare {
		choiceType = "جرئت"
	}

	proofTypeText := getProofTypeText(challenge.ProofType)

	// Active player view
	activeMsg := fmt.Sprintf("🎯 چالش %s شما:\n\n━━━━━━━━━━━━━━\n%s\n\n━━━━━━━━━━━━━━\n📸 مدرک مورد نیاز: %s\n💰 پاداش: %d سکه + %d XP\n\n⏱ زمان: 60 ثانیه\n\n👇 مدرک خود را ارسال کنید:",
		choiceType, challenge.Text, proofTypeText, challenge.CoinReward, challenge.XPReward)

	bot.SendMessage(activeUser.TelegramID, activeMsg, nil)

	// Passive player view
	passiveMsg := fmt.Sprintf("🎮 راند %d/%d\n⏳ حریف در حال انجام چالش...\n\n━━━━━━━━━━━━━━\n🎯 چالش: %s\n\n⏱ زمان باقی‌مانده: 60 ثانیه\n\nمنتظر ارسال مدرک...",
		game.CurrentRound, game.MaxRounds, choiceType)

	passiveKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💤 تلنگر", fmt.Sprintf("btn:tod_nudge_%d", gameID)),
		),
	)

	bot.SendMessage(passiveUser.TelegramID, passiveMsg, passiveKeyboard)
}

// Helper functions
func getUserByID(userID uint, match models.Match) *models.User {
	if match.User1ID == userID {
		return &match.User1
	} else if match.User2ID == userID {
		return &match.User2
	}
	return nil
}

func getProofTypeText(proofType string) string {
	switch proofType {
	case models.ProofTypeText:
		return "متن"
	case models.ProofTypeVoice:
		return "ویس"
	case models.ProofTypeImage:
		return "عکس"
	case models.ProofTypeVideo:
		return "ویدیو"
	default:
		return "ندارد"
	}
}

// StartTodGameWithMatch starts a ToD game with an existing match
func (h *HandlerManager) StartTodGameWithMatch(userID int64, matchID uint, bot BotInterface) {
	match, err := h.MatchRepo.GetMatchByID(matchID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات مچ!", nil)
		return
	}

	// Check if ToD game already exists
	existingGame, _ := h.TodRepo.GetGameByMatchID(matchID)
	if existingGame != nil {
		// Resume existing game
		h.ResumeTodGame(userID, existingGame.ID, bot)
		return
	}

	// Create new ToD game
	game, err := h.TodRepo.CreateGame(matchID, match.User1ID, match.User2ID)
	if err != nil {
		logger.Error("Failed to create ToD game", "error", err)
		bot.SendMessage(userID, "❌ خطا در ایجاد بازی!", nil)
		return
	}

	// Show match found message
	user1 := match.User1
	user2 := match.User2

	stats1, _ := h.TodRepo.GetOrCreatePlayerStats(user1.ID)
	stats2, _ := h.TodRepo.GetOrCreatePlayerStats(user2.ID)

	msg1 := fmt.Sprintf("✅ حریف پیدا شد!\n\n👤 حریف: %s\n⭐ سطح: %d\n🎖 امتیاز داوری: %.0f/100",
		user2.FullName, user2.Level, stats2.JudgeScore)
	msg2 := fmt.Sprintf("✅ حریف پیدا شد!\n\n👤 حریف: %s\n⭐ سطح: %d\n🎖 امتیاز داوری: %.0f/100",
		user1.FullName, user1.Level, stats1.JudgeScore)

	bot.SendMessage(user1.TelegramID, msg1, nil)
	bot.SendMessage(user2.TelegramID, msg2, nil)

	time.Sleep(2 * time.Second)

	// Coin flip
	h.HandleTodCoinFlip(game.ID, bot)
}
