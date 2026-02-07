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
// PROOF SUBMISSION
// ========================================

// HandleTodProofSubmission handles proof submission from active player
func (h *HandlerManager) HandleTodProofSubmission(userID int64, gameID uint, message *tgbotapi.Message, bot BotInterface) {
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
	if game.State != models.TodStateWaitingProof {
		bot.SendMessage(userID, "⚠️ در این مرحله نمی‌توانید مدرک ارسال کنید!", nil)
		return
	}

	// Get current turn
	turn, err := h.TodRepo.GetCurrentTurn(gameID)
	if err != nil {
		logger.Error("Failed to get current turn", "error", err)
		return
	}

	if turn.Challenge == nil {
		logger.Error("Turn has no challenge", "turn_id", turn.ID)
		return
	}

	// Validate proof type
	var proofType, proofData string
	requiredType := turn.Challenge.ProofType

	if message.Voice != nil {
		proofType = models.ProofTypeVoice
		proofData = message.Voice.FileID
	} else if len(message.Photo) > 0 {
		proofType = models.ProofTypeImage
		proofData = message.Photo[len(message.Photo)-1].FileID
	} else if message.Video != nil {
		proofType = models.ProofTypeVideo
		proofData = message.Video.FileID
	} else if message.Text != "" {
		proofType = models.ProofTypeText
		proofData = message.Text
	} else {
		bot.SendMessage(userID, "⚠️ نوع مدرک نامعتبر است!", nil)
		return
	}

	// Check if proof type matches requirement
	if requiredType != models.ProofTypeNone && proofType != requiredType {
		proofTypeText := getProofTypeText(requiredType)
		bot.SendMessage(userID, fmt.Sprintf("⚠️ نوع مدرک اشتباه است! باید %s ارسال کنید", proofTypeText), nil)
		return
	}

	// Save proof
	h.TodRepo.UpdateTurnProof(turn.ID, proofType, proofData)

	// Show confirmation
	msg := "✅ مدرک دریافت شد!\n\nآیا مطمئنی که آماده‌ای؟"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ انجام دادم", fmt.Sprintf("btn:tod_confirm_proof_%d", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("🔄 ارسال مجدد", fmt.Sprintf("btn:tod_resubmit_%d", gameID)),
		),
	)

	bot.SendMessage(userID, msg, keyboard)
}

// HandleTodConfirmProof confirms proof and sends to judge
func (h *HandlerManager) HandleTodConfirmProof(userID int64, gameID uint, bot BotInterface) {
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
		return
	}

	// Generate action ID for idempotency
	actionID := uuid.New().String()
	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		return
	}
	h.TodRepo.MarkActionProcessed(gameID, user.ID, actionID, "confirm_proof")

	// Get current turn
	turn, err := h.TodRepo.GetCurrentTurn(gameID)
	if err != nil {
		return
	}

	// Update state
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingJudgment)

	// Show judgment screen
	h.ShowTodJudgmentScreen(gameID, turn, bot)
}

// HandleTodResubmit allows resubmitting proof
func (h *HandlerManager) HandleTodResubmit(userID int64, gameID uint, bot BotInterface) {
	msg := "🔄 لطفاً مدرک جدید خود را ارسال کنید:"
	bot.SendMessage(userID, msg, nil)
}

// ========================================
// JUDGMENT PHASE
// ========================================

// ShowTodJudgmentScreen shows judgment screen to judge
func (h *HandlerManager) ShowTodJudgmentScreen(gameID uint, turn *models.TodTurn, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	if activeUser == nil || passiveUser == nil {
		return
	}

	// Send proof to judge
	judgeMsg := fmt.Sprintf("⚖️ داوری با توئه!\n\n━━━━━━━━━━━━━━\n🎯 چالش بود:\n%s\n\n📸 مدرک ارسالی:", turn.ChallengeText)

	bot.SendMessage(passiveUser.TelegramID, judgeMsg, nil)

	// Forward proof
	h.forwardProof(passiveUser.TelegramID, turn, bot)

	// Show judgment buttons
	judgmentMsg := "━━━━━━━━━━━━━━\nآیا حریف چالش را انجام داده؟\n\n⚠️ توجه: رد ناعادلانه باعث کاهش اعتبار داوری شما می‌شود!"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ قبوله", fmt.Sprintf("btn:tod_judge_%d_accept", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ قبول نیست", fmt.Sprintf("btn:tod_judge_%d_reject", gameID)),
		),
	)

	bot.SendMessage(passiveUser.TelegramID, judgmentMsg, keyboard)

	// Update active player
	activeMsg := "🎮 در انتظار داوری حریف...\n\n⏱ زمان باقی‌مانده: 60 ثانیه\n\nمنتظر تصمیم داور..."
	bot.SendMessage(activeUser.TelegramID, activeMsg, nil)
}

// forwardProof forwards the proof to judge
func (h *HandlerManager) forwardProof(judgeID int64, turn *models.TodTurn, bot BotInterface) {
	api := bot.GetAPI()
	if api == nil {
		return
	}

	botAPI, ok := api.(*tgbotapi.BotAPI)
	if !ok {
		return
	}

	switch turn.ProofType {
	case models.ProofTypeVoice:
		voice := tgbotapi.NewVoice(judgeID, tgbotapi.FileID(turn.ProofData))
		botAPI.Send(voice)
	case models.ProofTypeImage:
		photo := tgbotapi.NewPhoto(judgeID, tgbotapi.FileID(turn.ProofData))
		botAPI.Send(photo)
	case models.ProofTypeVideo:
		video := tgbotapi.NewVideo(judgeID, tgbotapi.FileID(turn.ProofData))
		botAPI.Send(video)
	case models.ProofTypeText:
		msg := tgbotapi.NewMessage(judgeID, fmt.Sprintf("📝 پاسخ: %s", turn.ProofData))
		botAPI.Send(msg)
	}
}

// HandleTodJudgment handles judgment from judge
func (h *HandlerManager) HandleTodJudgment(userID int64, gameID uint, result string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Verify user is judge
	if game.PassivePlayerID != user.ID {
		bot.SendMessage(userID, "⚠️ شما داور نیستید!", nil)
		return
	}

	// Verify state
	if game.State != models.TodStateWaitingJudgment {
		bot.SendMessage(userID, "⚠️ در این مرحله نمی‌توانید داوری کنید!", nil)
		return
	}

	// Generate action ID for idempotency
	actionID := uuid.New().String()
	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		return
	}
	h.TodRepo.MarkActionProcessed(gameID, user.ID, actionID, "judgment_"+result)

	// Get current turn
	turn, err := h.TodRepo.GetCurrentTurn(gameID)
	if err != nil {
		return
	}

	// Update judgment
	h.TodRepo.UpdateTurnJudgment(turn.ID, result, "")

	// Log judgment
	h.TodRepo.LogJudgment(turn.ID, game.PassivePlayerID, game.ActivePlayerID, result)

	// Check for unfair judgment
	isUnfair, reason, _ := h.TodRepo.DetectUnfairJudgment(user.ID)
	if isUnfair {
		h.TodRepo.IncrementUnfairJudgmentCount(user.ID)

		// Calculate and update judge score
		newScore, _ := h.TodRepo.CalculateJudgeScore(user.ID)
		h.TodRepo.UpdateJudgeScore(user.ID, newScore)

		// Get stats to check unfair count
		stats, _ := h.TodRepo.GetOrCreatePlayerStats(user.ID)

		if stats.UnfairJudgmentCount >= 5 {
			// Ban message
			banMsg := "🚫 محدودیت داوری!\n\nبه دلیل داوری ناعادلانه مکرر، شما برای 24 ساعت از بازی جرعت و حقیقت محروم شدید"
			bot.SendMessage(userID, banMsg, nil)
			// TODO: Implement actual ban logic
		} else if stats.UnfairJudgmentCount >= 3 {
			// Warning
			warningMsg := fmt.Sprintf("⚠️ هشدار!\n\nاعتبار داوری شما کاهش یافته است.\n\nامتیاز فعلی: %.0f/100\n\nدلیل: %s\n\nلطفاً منصفانه داوری کنید", newScore, reason)
			bot.SendMessage(userID, warningMsg, nil)
		}
	}

	// Award or penalize
	var xpAwarded, coinsAwarded int
	if result == "accepted" {
		// Award player
		if turn.Challenge != nil {
			xpAwarded = turn.Challenge.XPReward
			coinsAwarded = turn.Challenge.CoinReward

			h.CoinRepo.AddCoins(game.ActivePlayerID, int64(coinsAwarded), models.TxTypeGameReward, "پاداش بازی جرعت و حقیقت")
			h.VillageSvc.AddXPForUser(game.ActivePlayerID, int64(xpAwarded))

			// Update challenge acceptance rate
			h.TodRepo.UpdateChallengeAcceptanceRate(turn.Challenge.ID, true)
		}

		// Update stats
		h.TodRepo.IncrementChallengeCompleted(game.ActivePlayerID, turn.Choice, true)
	} else {
		// Penalize player
		coinsAwarded = -5
		h.CoinRepo.AddCoins(game.ActivePlayerID, -5, models.TxTypePenalty, "جریمه رد شدن چالش")

		// Update stats
		h.TodRepo.IncrementChallengeCompleted(game.ActivePlayerID, turn.Choice, false)

		// Update challenge acceptance rate
		if turn.Challenge != nil {
			h.TodRepo.UpdateChallengeAcceptanceRate(turn.Challenge.ID, false)
		}
	}

	// Update turn rewards
	h.TodRepo.UpdateTurnRewards(turn.ID, xpAwarded, coinsAwarded)

	// Complete turn
	h.TodRepo.CompleteTurn(turn.ID)

	// Show results
	h.ShowTodRoundResult(gameID, result, xpAwarded, coinsAwarded, bot)
}

// ShowTodRoundResult shows round result to both players
func (h *HandlerManager) ShowTodRoundResult(gameID uint, result string, xp, coins int, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	if activeUser == nil || passiveUser == nil {
		return
	}

	var activeMsg, passiveMsg string

	if result == "accepted" {
		activeMsg = fmt.Sprintf("✅ داور قبول کرد!\n\n━━━━━━━━━━━━━━\n🎉 تبریک! چالش را با موفقیت انجام دادی!\n\n💰 پاداش‌ها:\n• +%d سکه\n• +%d XP\n• +10 XP دهکده", coins, xp)
		passiveMsg = "✅ شما چالش را تایید کردید"
	} else {
		activeMsg = "❌ داور رد کرد!\n\n━━━━━━━━━━━━━━\n😔 متأسفانه چالش پذیرفته نشد\n\n💸 جریمه:\n• -5 سکه\n• بدون XP"
		passiveMsg = "❌ شما چالش را رد کردید"
	}

	bot.SendMessage(activeUser.TelegramID, activeMsg, nil)
	bot.SendMessage(passiveUser.TelegramID, passiveMsg, nil)

	time.Sleep(3 * time.Second)

	// Check if game should end
	if game.CurrentRound >= game.MaxRounds {
		h.EndTodGame(gameID, bot)
	} else {
		// Next round
		h.TodRepo.IncrementRound(gameID)
		h.TodRepo.SwitchTurn(gameID)
		h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingChoice)

		// Create new turn
		game, _ = h.TodRepo.GetGameByID(gameID)
		h.TodRepo.CreateTurn(gameID, game.ActivePlayerID, game.PassivePlayerID, game.CurrentRound)

		// Show choice screen
		h.ShowTodChoiceScreen(gameID, bot)
	}
}
