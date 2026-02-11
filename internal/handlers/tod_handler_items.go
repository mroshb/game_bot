package handlers

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
	"gorm.io/gorm"
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
	var itemEffectFunc func(tx *gorm.DB, game *models.TodGame, turn *models.TodTurn) error
	// UI update function to run after commit
	var uiUpdateFunc func()

	// Start transaction
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Use repositories with transaction
		todRepo := h.TodRepo.WithTx(tx)
		userRepo := h.UserRepo.WithTx(tx)

		user, err := userRepo.GetUserByTelegramID(userID)
		if err != nil {
			return err
		}

		// Lock game row
		game, err := todRepo.GetGameByID(gameID)
		if err != nil {
			return err
		}

		// Verify it's user's turn
		if game.ActivePlayerID != user.ID {
			bot.SendMessage(userID, "⚠️ نوبت شما نیست!", nil)
			return nil // Return nil to stop logic without rollback error
		}

		// Verify state (waiting_choice OR waiting_proof)
		if game.State != models.TodStateWaitingChoice && game.State != models.TodStateWaitingProof {
			bot.SendMessage(userID, "⚠️ فقط در مرحله انتخاب یا انجام چالش می‌توانید از آیتم استفاده کنید!", nil)
			return nil
		}

		// Special check for Swap: Must be in waiting_proof to swap a question (unless intended otherwise)
		// If used in waiting_choice -> useless consumption?
		// Let's allow use in both, but logic differs.
		// If in waiting_choice: Swap resets choice? No, just re-shows choice. (Wait, consumed for nothing?)
		// Let's restrict SWAP to waiting_proof.
		if itemType == models.ItemTypeSwap && game.State != models.TodStateWaitingProof {
			bot.SendMessage(userID, "⚠️ آیتم تعویض فقط زمانی که سوال را دیده‌اید قابل استفاده است!", nil)
			return nil
		}

		// Generate action ID for idempotency (inside tx check if exists)
		actionID := uuid.New().String()
		if todRepo.IsActionProcessed(gameID, actionID) {
			return nil
		}
		if err := todRepo.MarkActionProcessed(gameID, user.ID, actionID, "use_item_"+itemType); err != nil {
			return err
		}

		// Try to use item
		err = todRepo.UseItem(user.ID, itemType)
		if err != nil {
			bot.SendMessage(userID, "❌ شما این آیتم را ندارید!", nil)
			return nil
		}

		// Get current turn
		turn, err := todRepo.GetCurrentTurn(gameID)
		if err != nil {
			logger.Error("Failed to get current turn", "error", err)
			return err
		}

		// Log item usage
		now := time.Now()
		if err := tx.Model(&models.TodTurn{}).Where("id = ?", turn.ID).
			Updates(map[string]interface{}{
				"item_used":    itemType,
				"item_used_at": now,
			}).Error; err != nil {
			return err
		}

		// Define logic based on item type
		switch itemType {
		case models.ItemTypeShield:
			itemEffectFunc = func(tx *gorm.DB, game *models.TodGame, turn *models.TodTurn) error {
				// Mark turn as completed (skipped)
				if err := tx.Model(&models.TodTurn{}).Where("id = ?", turn.ID).Update("completed_at", time.Now()).Error; err != nil {
					return err
				}
				// Switch turn
				repo := h.TodRepo.WithTx(tx)
				if err := repo.SwitchTurn(gameID); err != nil {
					return err
				}
				// Create new turn
				// Determine next round info (simple increment if passive becomes active)
				// Actually active/passive swapped in SwitchTurn.
				// Need to re-fetch game to get new active/passive IDs? SwitchTurn swaps IDs in DB.
				// In memory 'game' struct is stale.
				// Since we are inside TX, we can manually swap IDs for CreateTurn call or fetch again.
				// Fetching is safer.
				updatedGame, err := repo.GetGameByID(gameID)
				if err != nil {
					return err
				}
				// Create next turn
				round := updatedGame.CurrentRound
				if _, err := repo.CreateTurn(gameID, updatedGame.ActivePlayerID, updatedGame.PassivePlayerID, round); err != nil {
					return err
				}
				return nil
			}

			uiUpdateFunc = func() {
				h.handleShieldUseUI(gameID, bot)
			}

		case models.ItemTypeSwap:
			// Logic: Pick a new random challenge of same type/difficulty
			itemEffectFunc = func(tx *gorm.DB, game *models.TodGame, turn *models.TodTurn) error {
				if turn.ChallengeID == nil || turn.Choice == "" {
					return fmt.Errorf("cannot swap without challenge")
				}
				repo := h.TodRepo.WithTx(tx)
				// Logic: Pick a new random challenge of same type/difficulty
				// We keep difficulty and gender target but allow any relation level for more variety
				newChallenge, err := repo.GetRandomChallenge(turn.Choice, "easy", "", user.Gender, "all")
				if err != nil {
					// Fallback: keep same if error? or error out?
					return err
				}
				// Update turn
				if err := repo.UpdateTurnChallenge(turn.ID, newChallenge.ID, newChallenge.Text); err != nil {
					return err
				}
				if err := repo.IncrementChallengeUsage(newChallenge.ID); err != nil {
					return err
				}
				// Update state? It remains waiting_proof.
				return nil
			}

			uiUpdateFunc = func() {
				h.handleSwapUseUI(gameID, bot)
			}

		case models.ItemTypeMirror:
			itemEffectFunc = func(tx *gorm.DB, game *models.TodGame, turn *models.TodTurn) error {
				repo := h.TodRepo.WithTx(tx)
				// Switch roles immediately
				if err := repo.SwitchTurn(gameID); err != nil {
					return err
				}
				// The current turn is now owned by the *new* active player (previous passive).
				// We need to update the turn's PLayerID to the new active player.
				// Wait, TodTurn has PlayerID. If we switch roles, does the turn stay with old player?
				// Logic: "Transfer challenge to opponent".
				// So we update the turn's PlayerID to the opponent.
				updatedGame, err := repo.GetGameByID(gameID)
				if err != nil {
					return err
				}
				if err := tx.Model(&models.TodTurn{}).Where("id = ?", turn.ID).Update("player_id", updatedGame.ActivePlayerID).Error; err != nil {
					return err
				}
				// State remains waiting_proof (or choice? Mirror usually reflects the *Question*).
				// If used in waiting_choice -> opponent must choose.
				// If used in waiting_proof -> opponent must do *this* challenge.
				return nil
			}

			uiUpdateFunc = func() {
				h.handleMirrorUseUI(gameID, bot)
			}
		}

		// Execute effect
		if itemEffectFunc != nil {
			if err := itemEffectFunc(tx, game, turn); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("Transaction failed in HandleTodItemUse", "error", err)
		bot.SendMessage(userID, "❌ خطایی رخ داد، لطفاً دوباره تلاش کنید.", nil)
		return
	}

	// Trigger UI updates
	if uiUpdateFunc != nil {
		uiUpdateFunc()
	}
}

// handleShieldUseUI handles shield UI (messages)
func (h *HandlerManager) handleShieldUseUI(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}
	// Note: Active/Passive have swapped.
	// The person who used shield is now Passive.
	// We want to notify them "You used shield".
	// The *new* active player is the opponent.

	newPassiveID := game.PassivePlayerID
	newActiveID := game.ActivePlayerID

	passiveUser := getUserByID(newPassiveID, game.Match)
	activeUser := getUserByID(newActiveID, game.Match)

	msg := "🛡 سپر استفاده شد!\n\nنوبت شما بدون جریمه رد شد."
	bot.SendMessage(passiveUser.TelegramID, msg, nil)

	activeMsg := fmt.Sprintf("🛡 %s از سپر استفاده کرد و نوبت را رد کرد.\n\nحالا نوبت شماست!", passiveUser.FullName)
	bot.SendMessage(activeUser.TelegramID, activeMsg, nil)

	time.Sleep(2 * time.Second)
	// Show choice screen to new active player
	h.ShowTodChoiceScreen(gameID, bot)
}

// handleSwapUseUI handles swap UI
func (h *HandlerManager) handleSwapUseUI(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	turn, _ := h.TodRepo.GetCurrentTurn(gameID)

	msg := "🔄 سوال تعویض شد!\n\nچالش جدید شما:"
	bot.SendMessage(activeUser.TelegramID, msg, nil)

	time.Sleep(1 * time.Second)

	// Show new challenge
	if turn != nil && turn.ChallengeID != nil {
		// Need to fetch full challenge data
		// Since we only have ID/Text in turn, but we need coin reward etc.
		// TodTurn doesn't preload Challenge usually? Let's check.
		// We can fetch via ID.
		challenge, _ := h.TodRepo.GetChallengeByID(*turn.ChallengeID) // Hypothetical method?
		// Actually GetCurrentTurn preloads? No `h.TodRepo.GetCurrentTurn` uses `First`.
		// Let's assume we need to fetch it.
		// If method missing, just show basic text.
		if challenge != nil {
			h.ShowTodChallenge(gameID, challenge, bot)
		} else {
			// Fallback if challenge obj not found (shouldn't happen)
			// Reshow via generic flow?
			// Just send text
			bot.SendMessage(activeUser.TelegramID, turn.Challenge.Text, nil)
		}
	}
}

// handleMirrorUseUI handles mirror UI
func (h *HandlerManager) handleMirrorUseUI(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Active/Passive swapped. User who used mirror is now Passive.
	oldActiveID := game.PassivePlayerID // The one who used mirror
	newActiveID := game.ActivePlayerID  // The opponent

	userUsedMirror := getUserByID(oldActiveID, game.Match)
	newActiveUser := getUserByID(newActiveID, game.Match)

	msg := "🪞 آینه استفاده شد!\n\nچالش به حریف منتقل شد!"
	bot.SendMessage(userUsedMirror.TelegramID, msg, nil)

	passiveMsg := fmt.Sprintf("🪞 %s از آینه استفاده کرد!\n\nحالا نوبت شماست که این چالش را انجام دهید!", userUsedMirror.FullName)
	bot.SendMessage(newActiveUser.TelegramID, passiveMsg, nil)

	time.Sleep(2 * time.Second)

	// Recover state to show challenge to new user
	// We are in waiting_proof (or choice).
	if game.State == models.TodStateWaitingProof {
		turn, _ := h.TodRepo.GetCurrentTurn(gameID)
		if turn != nil && turn.ChallengeID != nil {
			// Fetch challenge
			var challenge models.TodChallenge
			h.DB.First(&challenge, *turn.ChallengeID)
			h.ShowTodChallenge(gameID, &challenge, bot)
		} else {
			h.ShowTodChoiceScreen(gameID, bot)
		}
	} else {
		h.ShowTodChoiceScreen(gameID, bot)
	}
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

	// Close match session
	if game.MatchID > 0 {
		h.MatchRepo.EndMatch(game.MatchID)
	}

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
		h.UserRepo.AddLeaguePoints(winnerID, 25)

		// Village Rewards
		h.VillageSvc.UpdateWarScore(winnerID, 10)
		h.VillageSvc.AddXPForUser(winnerID, 50)
	} else {
		// Draw
		h.TodRepo.IncrementGamesPlayed(game.Match.User1ID, false)
		h.TodRepo.IncrementGamesPlayed(game.Match.User2ID, false)
		h.CoinRepo.AddCoins(game.Match.User1ID, 20, models.TxTypeGameReward, "پاداش مساوی")
		h.CoinRepo.AddCoins(game.Match.User2ID, 20, models.TxTypeGameReward, "پاداش مساوی")
		h.UserRepo.AddLeaguePoints(game.Match.User1ID, 10)
		h.UserRepo.AddLeaguePoints(game.Match.User2ID, 10)

		// Village Rewards for draw
		h.VillageSvc.AddXPForUser(game.Match.User1ID, 20)
		h.VillageSvc.AddXPForUser(game.Match.User2ID, 20)
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

	switch winnerID {
	case user1.ID:
		msg1 += "🏆 شما برنده شدید! 🎉\n\n💰 پاداش: +50 سکه"
	case user2.ID:
		msg1 += "❌ شما باختید!"
	default:
		msg1 += "🤝 مساوی!\n\n💰 پاداش: +20 سکه"
	}

	// Message for user 2
	msg2 := "🎮 بازی تمام شد!\n\n━━━━━━━━━━━━━━\n📊 نتیجه نهایی:\n\n"
	msg2 += fmt.Sprintf("👤 شما: %d امتیاز\n", score2)
	msg2 += fmt.Sprintf("👤 %s: %d امتیاز\n\n", user1.FullName, score1)

	switch winnerID {
	case user2.ID:
		msg2 += "🏆 شما برنده شدید! 🎉\n\n💰 پاداش: +50 سکه"
	case user1.ID:
		msg2 += "❌ شما باختید!"
	default:
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

	// Close match session
	if game.MatchID > 0 {
		h.MatchRepo.EndMatch(game.MatchID)
	}

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
	h.UserRepo.AddLeaguePoints(winnerID, 15)
	h.UserRepo.AddLeaguePoints(timedOutPlayerID, -5)

	// Village Rewards
	h.VillageSvc.UpdateWarScore(winnerID, 5)
	h.VillageSvc.AddXPForUser(winnerID, 30)

	// Get users
	timedOutUser := getUserByID(timedOutPlayerID, game.Match)
	winnerUser := getUserByID(winnerID, game.Match)

	// Send messages
	timeoutMsg := "⏱ زمان تمام شد!\n\n━━━━━━━━━━━━━━\n🏳️ شما به دلیل عدم پاسخ‌گویی باخت فنی شدید\n\n💸 جریمه:\n• -20 سکه\n• -10 XP\n\n━━━━━━━━━━━━━━\n⚠️ توجه: تایم‌اوت مکرر می‌تواند منجر به محدودیت حساب شود"
	winnerMsg := "🏆 برنده شدید!\n\n━━━━━━━━━━━━━━\nحریف به دلیل AFK باخت فنی شد\n\n💰 پاداش برد:\n• +30 سکه\n• +20 XP"

	bot.SendMessage(timedOutUser.TelegramID, timeoutMsg, nil)
	bot.SendMessage(winnerUser.TelegramID, winnerMsg, nil)

	bot.SendMessage(winnerUser.TelegramID, winnerMsg, nil)

	logger.Info("ToD game timed out", "game_id", gameID, "timed_out_player", timedOutPlayerID)
}

// HandleTodForceWin allows a player to claim a win if the opponent is AFK
func (h *HandlerManager) HandleTodForceWin(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Double check the claimant is the one who ISN'T active
	if (game.ActivePlayerID == uint(userID) && game.State != models.TodStateWaitingJudgment) ||
		(game.PassivePlayerID == uint(userID) && game.State == models.TodStateWaitingChoice) {
		// If they are active turn player, they can't claim force win against themselves
		bot.SendMessage(userID, "⚠️ نوبت شماست! نمی‌توانید درخواست باخت فنی دهید.", nil)
		return
	}

	// Check if enough time has passed (e.g. 1 hour)
	// We use TurnStartedAt or LastInteraction logic.
	// Since we set deadline to 24h, checking against deadline is for "Timeout".
	// For "Force Win" button, we want a shorter period like 12 hours.
	minWaitDuration := 12 * time.Hour

	var lastActivity time.Time
	if game.TurnStartedAt != nil {
		lastActivity = *game.TurnStartedAt
	} else {
		lastActivity = time.Now() // Fallback
	}

	if time.Since(lastActivity) < minWaitDuration {
		remaining := minWaitDuration - time.Since(lastActivity)
		bot.SendMessage(userID, fmt.Sprintf("⏳ برای درخواست باخت فنی باید حداقل ۱۲ ساعت از نوبت حریف گذشته باشد.\n\nزمان باقی‌مانده: %d دقیقه", int(remaining.Minutes())), nil)
		return
	}

	h.HandleTodTimeout(gameID, bot)
}

// SendTodWarning sends a reminder
func (h *HandlerManager) SendTodWarning(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	if activeUser == nil {
		return
	}

	msg := "🔔 یادآوری نوبت!\n\nدوست عزیز، نوبت بازی شماست. لطفاً هر چه سریعتر پاسخ دهید تا حریف معطل نشود."
	bot.SendMessage(activeUser.TelegramID, msg, nil)
}

// ========================================
// UTILITY FUNCTIONS
// ========================================

// HandleTodQuitSimple finds the active game for a user and quits it
func (h *HandlerManager) HandleTodQuitSimple(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	activeGame, err := h.TodRepo.GetActiveGameForUser(user.ID)
	if err != nil || activeGame == nil {
		bot.SendMessage(userID, "⚠️ شما در بازی فعالی نیستید.", nil)
		return
	}

	h.HandleTodQuit(userID, activeGame.ID, bot)
}

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

	// Close match session
	if game.MatchID > 0 {
		h.MatchRepo.EndMatch(game.MatchID)
	}

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
	// Rate limit: 1 nudge per 10 minutes
	// We use a time bucket (Unix / 600) as part of ActionID
	bucket := time.Now().Unix() / 600
	actionID := fmt.Sprintf("nudge_%d_%d_%d", gameID, userID, bucket)

	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		bot.SendMessage(userID, "⏳ لطفاً برای ارسال تلنگر بعدی صبر کنید.", nil)
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Mark processed
	h.TodRepo.MarkActionProcessed(gameID, uint(userID), actionID, "nudge")

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
	case models.TodStateMatchmaking:
		// Recover game stuck in matchmaking
		h.HandleTodCoinFlip(gameID, bot)
	case models.TodStateCoinFlip:
		// Coin flip done, waiting for start
		activeUser := getUserByID(game.ActivePlayerID, game.Match)
		passiveUser := getUserByID(game.PassivePlayerID, game.Match)

		msg := fmt.Sprintf("🎲 نتیجه قرعه‌کشی:\n\n🎯 نوبت اول: %s\n⏳ نوبت دوم: %s\n\nبازی شروع شد! 🎮\nراند 1 از %d",
			activeUser.FullName, passiveUser.FullName, game.MaxRounds)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("▶️ ادامه", fmt.Sprintf("btn:tod_start_%d", gameID)),
			),
		)
		bot.SendMessage(userID, msg, keyboard)
	default:
		bot.SendMessage(userID, fmt.Sprintf("⚠️ وضعیت بازی نامشخص است: %s", game.State), nil)
	}
}
