package handlers

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// ========================================
// ITEM SYSTEM
// ========================================

// ShowTodItemMenu shows item selection menu
func (h *HandlerManager) ShowTodItemMenu(userID int64, gameID uint, bot BotInterface) {
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

	// Get player stats for inventory
	stats, err := h.TodRepo.GetOrCreatePlayerStats(user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	msg := "🎒 آیتمهای شما:\n\n━━━━━━━━━━━━━━\n"
	msg += fmt.Sprintf("🛡 سپر (%d عدد)\nرد نوبت بدون جریمه\n\n", stats.ShieldsOwned)
	msg += fmt.Sprintf("🔄 تعویض (%d عدد)\nتغییر سوال به سوال دیگر\n\n", stats.SwapsOwned)
	msg += fmt.Sprintf("🪞 آینه (%d عدد)\nانتقال چالش به حریف\n\n", stats.MirrorsOwned)
	msg += "━━━━━━━━━━━━━━"

	var rows [][]tgbotapi.InlineKeyboardButton

	if stats.ShieldsOwned > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🛡 استفاده (%d)", stats.ShieldsOwned), fmt.Sprintf("btn:tod_use_item_%d_shield", gameID)),
		))
	}

	if stats.SwapsOwned > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔄 استفاده (%d)", stats.SwapsOwned), fmt.Sprintf("btn:tod_use_item_%d_swap", gameID)),
		))
	}

	if stats.MirrorsOwned > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🪞 استفاده (%d)", stats.MirrorsOwned), fmt.Sprintf("btn:tod_use_item_%d_mirror", gameID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("btn:tod_back_%d", gameID)),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.SendMessage(userID, msg, keyboard)
}

// HandleTodItemUse handles item usage
func (h *HandlerManager) HandleTodItemUse(userID int64, gameID uint, itemType string, bot BotInterface) {
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

	// Verify state (can only use items during choice phase)
	if game.State != models.TodStateWaitingChoice {
		bot.SendMessage(userID, "⚠️ فقط در مرحله انتخاب می‌توانید از آیتم استفاده کنید!", nil)
		return
	}

	// Generate action ID for idempotency
	actionID := uuid.New().String()
	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		return
	}
	h.TodRepo.MarkActionProcessed(gameID, user.ID, actionID, "use_item_"+itemType)

	// Try to use item
	err = h.TodRepo.UseItem(user.ID, itemType)
	if err != nil {
		bot.SendMessage(userID, "❌ شما این آیتم را ندارید!", nil)
		return
	}

	// Get current turn
	turn, err := h.TodRepo.GetCurrentTurn(gameID)
	if err != nil {
		logger.Error("Failed to get current turn", "error", err)
		return
	}

	// Log item usage
	now := time.Now()
	h.DB.Model(&models.TodTurn{}).Where("id = ?", turn.ID).
		Updates(map[string]interface{}{
			"item_used":    itemType,
			"item_used_at": now,
		})

	// Apply item effect
	switch itemType {
	case models.ItemTypeShield:
		h.handleShieldUse(gameID, bot)
	case models.ItemTypeSwap:
		h.handleSwapUse(gameID, bot)
	case models.ItemTypeMirror:
		h.handleMirrorUse(gameID, bot)
	}
}

// handleShieldUse handles shield item (skip turn without penalty)
func (h *HandlerManager) handleShieldUse(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	msg := "🛡 سپر استفاده شد!\n\nنوبت شما بدون جریمه رد شد"
	bot.SendMessage(activeUser.TelegramID, msg, nil)

	passiveMsg := fmt.Sprintf("🛡 %s از سپر استفاده کرد و نوبت را رد کرد", activeUser.FullName)
	bot.SendMessage(passiveUser.TelegramID, passiveMsg, nil)

	time.Sleep(2 * time.Second)

	// Complete turn
	turn, _ := h.TodRepo.GetCurrentTurn(gameID)
	if turn != nil {
		h.TodRepo.CompleteTurn(turn.ID)
	}

	// Switch turn
	h.TodRepo.SwitchTurn(gameID)

	// Create new turn
	game, _ = h.TodRepo.GetGameByID(gameID)
	h.TodRepo.CreateTurn(gameID, game.ActivePlayerID, game.PassivePlayerID, game.CurrentRound)

	// Show choice screen
	h.ShowTodChoiceScreen(gameID, bot)
}

// handleSwapUse handles swap item (change challenge)
func (h *HandlerManager) handleSwapUse(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)

	msg := "🔄 سوال تعویض شد!\n\nلطفاً دوباره انتخاب کنید:"
	bot.SendMessage(activeUser.TelegramID, msg, nil)

	time.Sleep(1 * time.Second)

	// Show choice screen again
	h.ShowTodChoiceScreen(gameID, bot)
}

// handleMirrorUse handles mirror item (transfer challenge to opponent)
func (h *HandlerManager) handleMirrorUse(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	msg := "🪞 آینه استفاده شد!\n\nچالش به حریف منتقل شد!"
	bot.SendMessage(activeUser.TelegramID, msg, nil)

	passiveMsg := fmt.Sprintf("🪞 %s از آینه استفاده کرد!\n\nحالا نوبت شماست!", activeUser.FullName)
	bot.SendMessage(passiveUser.TelegramID, passiveMsg, nil)

	time.Sleep(2 * time.Second)

	// Switch roles
	h.TodRepo.SwitchTurn(gameID)

	// Show choice screen to new active player
	h.ShowTodChoiceScreen(gameID, bot)
}

// ========================================
// GAME END
// ========================================

// EndTodGame ends the game and shows final results
func (h *HandlerManager) EndTodGame(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Calculate scores (based on completed challenges)
	var player1Score, player2Score int

	// Get all turns for this game
	var turns []models.TodTurn
	h.DB.Where("game_id = ?", gameID).Find(&turns)

	for _, turn := range turns {
		if turn.JudgmentResult == "accepted" {
			if turn.PlayerID == game.Match.User1ID {
				player1Score++
			} else {
				player2Score++
			}
		}
	}

	// Determine winner
	var winnerID uint
	if player1Score > player2Score {
		winnerID = game.Match.User1ID
	} else if player2Score > player1Score {
		winnerID = game.Match.User2ID
	}
	// If equal, no winner (draw)

	// End game
	h.TodRepo.EndGame(gameID, winnerID, "completed")

	// Update player stats
	if winnerID > 0 {
		h.TodRepo.IncrementGamesPlayed(winnerID, true)
		var loserID uint
		if winnerID == game.Match.User1ID {
			loserID = game.Match.User2ID
		} else {
			loserID = game.Match.User1ID
		}
		h.TodRepo.IncrementGamesPlayed(loserID, false)

		// Award winner
		h.CoinRepo.AddCoins(winnerID, 50, models.TxTypeGameReward, "پاداش برد بازی جرعت و حقیقت")
	} else {
		// Draw
		h.TodRepo.IncrementGamesPlayed(game.Match.User1ID, false)
		h.TodRepo.IncrementGamesPlayed(game.Match.User2ID, false)
		h.CoinRepo.AddCoins(game.Match.User1ID, 20, models.TxTypeGameReward, "پاداش مساوی")
		h.CoinRepo.AddCoins(game.Match.User2ID, 20, models.TxTypeGameReward, "پاداش مساوی")
	}

	// Show results
	h.ShowTodGameResults(game, player1Score, player2Score, winnerID, bot)
}

// ShowTodGameResults shows final game results
func (h *HandlerManager) ShowTodGameResults(game *models.TodGame, score1, score2 int, winnerID uint, bot BotInterface) {
	user1 := game.Match.User1
	user2 := game.Match.User2

	// Message for user 1
	msg1 := "🎮 بازی تمام شد!\n\n━━━━━━━━━━━━━━\n📊 نتیجه نهایی:\n\n"
	msg1 += fmt.Sprintf("👤 شما: %d امتیاز\n", score1)
	msg1 += fmt.Sprintf("👤 %s: %d امتیاز\n\n", user2.FullName, score2)

	if winnerID == user1.ID {
		msg1 += "🏆 شما برنده شدید! 🎉\n\n💰 پاداش: +50 سکه"
	} else if winnerID == user2.ID {
		msg1 += "❌ شما باختید!"
	} else {
		msg1 += "🤝 مساوی!\n\n💰 پاداش: +20 سکه"
	}

	// Message for user 2
	msg2 := "🎮 بازی تمام شد!\n\n━━━━━━━━━━━━━━\n📊 نتیجه نهایی:\n\n"
	msg2 += fmt.Sprintf("👤 شما: %d امتیاز\n", score2)
	msg2 += fmt.Sprintf("👤 %s: %d امتیاز\n\n", user1.FullName, score1)

	if winnerID == user2.ID {
		msg2 += "🏆 شما برنده شدید! 🎉\n\n💰 پاداش: +50 سکه"
	} else if winnerID == user1.ID {
		msg2 += "❌ شما باختید!"
	} else {
		msg2 += "🤝 مساوی!\n\n💰 پاداش: +20 سکه"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 بازی مجدد", "btn:tod_new_game"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 منوی اصلی", "btn:main_menu"),
		),
	)

	bot.SendMessage(user1.TelegramID, msg1, keyboard)
	bot.SendMessage(user2.TelegramID, msg2, keyboard)
}

// ========================================
// TIMEOUT & AFK HANDLING
// ========================================

// HandleTodTimeout handles game timeout
func (h *HandlerManager) HandleTodTimeout(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Determine who timed out
	timedOutPlayerID := game.ActivePlayerID
	winnerID := game.PassivePlayerID

	// End game
	h.TodRepo.HandleTimeout(gameID)

	// Update stats
	h.TodRepo.IncrementGamesPlayed(winnerID, true)
	h.TodRepo.IncrementGamesPlayed(timedOutPlayerID, false)

	// Penalize timed out player
	h.CoinRepo.AddCoins(timedOutPlayerID, -20, models.TxTypePenalty, "جریمه تایم‌اوت")
	h.TodRepo.UpdatePlayerStats(timedOutPlayerID, map[string]interface{}{
		"timeout_count": h.DB.Raw("timeout_count + 1"),
	})

	// Reward winner
	h.CoinRepo.AddCoins(winnerID, 30, models.TxTypeGameReward, "پاداش برد به دلیل AFK حریف")

	// Get users
	timedOutUser := getUserByID(timedOutPlayerID, game.Match)
	winnerUser := getUserByID(winnerID, game.Match)

	// Send messages
	timeoutMsg := "⏱ زمان تمام شد!\n\n━━━━━━━━━━━━━━\n🏳️ شما به دلیل عدم پاسخ‌گویی باخت فنی شدید\n\n💸 جریمه:\n• -20 سکه\n• -10 XP\n\n━━━━━━━━━━━━━━\n⚠️ توجه: تایم‌اوت مکرر می‌تواند منجر به محدودیت حساب شود"
	winnerMsg := "🏆 برنده شدید!\n\n━━━━━━━━━━━━━━\nحریف به دلیل AFK باخت فنی شد\n\n💰 پاداش برد:\n• +30 سکه\n• +20 XP"

	bot.SendMessage(timedOutUser.TelegramID, timeoutMsg, nil)
	bot.SendMessage(winnerUser.TelegramID, winnerMsg, nil)

	logger.Info("ToD game timed out", "game_id", gameID, "timed_out_player", timedOutPlayerID)
}

// SendTodWarning sends 30s warning
func (h *HandlerManager) SendTodWarning(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	if activeUser == nil {
		return
	}

	msg := "⚠️ هشدار!\n\n⏰ فقط 30 ثانیه باقی مانده!\n\nسریع انتخاب کن وگرنه باخت فنی می‌شود!"
	bot.SendMessage(activeUser.TelegramID, msg, nil)
}

// ========================================
// UTILITY FUNCTIONS
// ========================================

// HandleTodQuit handles player quitting
func (h *HandlerManager) HandleTodQuit(userID int64, gameID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Determine winner (opponent)
	var winnerID uint
	if game.ActivePlayerID == user.ID {
		winnerID = game.PassivePlayerID
	} else {
		winnerID = game.ActivePlayerID
	}

	// End game
	h.TodRepo.EndGame(gameID, winnerID, "quit")

	// Update stats
	h.TodRepo.IncrementGamesPlayed(winnerID, true)
	h.TodRepo.IncrementGamesPlayed(user.ID, false)

	// Penalize quitter
	h.CoinRepo.AddCoins(user.ID, -10, models.TxTypePenalty, "جریمه انصراف از بازی")

	// Reward winner
	h.CoinRepo.AddCoins(winnerID, 20, models.TxTypeGameReward, "پاداش برد به دلیل انصراف حریف")

	// Send messages
	bot.SendMessage(userID, "🏳️ شما از بازی انصراف دادید\n\n💸 جریمه: -10 سکه", nil)

	winnerUser := getUserByID(winnerID, game.Match)
	if winnerUser != nil {
		bot.SendMessage(winnerUser.TelegramID, "🏆 حریف از بازی انصراف داد!\n\n💰 پاداش: +20 سکه", nil)
	}
}

// HandleTodNudge handles nudge action
func (h *HandlerManager) HandleTodNudge(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	if activeUser == nil {
		return
	}

	msg := "⚡ حریف داره منتظرته! زود باش!"
	bot.SendMessage(activeUser.TelegramID, msg, nil)

	bot.SendMessage(userID, "✅ تلنگر ارسال شد", nil)
}

// ResumeTodGame resumes an existing game
func (h *HandlerManager) ResumeTodGame(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("بازی شما در حال ادامه است...\n\nراند %d از %d", game.CurrentRound, game.MaxRounds)
	bot.SendMessage(userID, msg, nil)

	time.Sleep(1 * time.Second)

	// Show appropriate screen based on state
	switch game.State {
	case models.TodStateWaitingChoice:
		h.ShowTodChoiceScreen(gameID, bot)
	case models.TodStateWaitingProof:
		turn, _ := h.TodRepo.GetCurrentTurn(gameID)
		if turn != nil && turn.Challenge != nil {
			h.ShowTodChallenge(gameID, turn.Challenge, bot)
		}
	case models.TodStateWaitingJudgment:
		turn, _ := h.TodRepo.GetCurrentTurn(gameID)
		if turn != nil {
			h.ShowTodJudgmentScreen(gameID, turn, bot)
		}
	default:
		bot.SendMessage(userID, "⚠️ وضعیت بازی نامشخص است", nil)
	}
}
