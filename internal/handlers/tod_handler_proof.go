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
			tgbotapi.NewInlineKeyboardButtonData("✅ قبوله", fmt.Sprintf("btn:tod_judge_%d_accepted", gameID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ قبول نیست", fmt.Sprintf("btn:tod_judge_%d_rejected", gameID)),
		),
	)

	bot.SendMessage(passiveUser.TelegramID, judgmentMsg, keyboard)

	// Update active player
	activeMsg := "🎮 مدرک شما ارسال شد!\n\nمنتظر تایید داور باشید... (هر وقت داور تایید کرد خبرت می‌کنیم)"
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
		activeMsg = fmt.Sprintf(`✅ **داور قبول کرد!**

━━━━━━━━━━━━━━
🎉 تبریک! چالش رو با موفقیت انجام دادی!

💰 **پاداش‌ها:**
• +%d سکه 🪙
• +%d تجربه 🏆
• +10 تجربه دهکده 🏰`, coins, xp)
		passiveMsg = "✅ شما چالش را تایید کردید. پاداش به حریف منتقل شد."
	} else {
		activeMsg = `❌ **داور رد کرد!**

━━━━━━━━━━━━━━
😔 متأسفانه چالش پذیرفته نشد.

💸 **جریمه:**
• -5 سکه 🪙
• بدون تجربه

⚖️ اگر فکر می‌کنی ناعادلانه بوده، می‌تونی اعتراض کنی.`
		passiveMsg = "❌ شما چالش را رد کردید."

		activeKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚖️ اعتراض (Appeal)", fmt.Sprintf("btn:tod_appeal_%d", gameID)),
				tgbotapi.NewInlineKeyboardButtonData("➡️ مرحله بعد", fmt.Sprintf("btn:tod_next_round_%d", gameID)),
			),
		)
		passiveKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➡️ مرحله بعد", fmt.Sprintf("btn:tod_next_round_%d", gameID)),
			),
		)
		bot.SendMessage(activeUser.TelegramID, activeMsg, activeKeyboard)
		bot.SendMessage(passiveUser.TelegramID, passiveMsg, passiveKeyboard)
		return
	}

	bot.SendMessage(activeUser.TelegramID, activeMsg, nil)
	bot.SendMessage(passiveUser.TelegramID, passiveMsg, nil)

	time.Sleep(2 * time.Second)
	h.MoveToNextRound(gameID, bot)
}

// HandleTodAppeal handles user appeal against unfair judgment
func (h *HandlerManager) HandleTodAppeal(userID int64, gameID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Verify it's user's turn (the one who did the challenge)
	if game.ActivePlayerID != user.ID {
		return
	}

	// Generate action ID for idempotency
	actionID := uuid.New().String()
	if h.TodRepo.IsActionProcessed(gameID, actionID) {
		return
	}
	h.TodRepo.MarkActionProcessed(gameID, user.ID, actionID, "appeal")

	// Update state
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingAppeal)

	// Notify player
	bot.SendMessage(userID, "⚖️ اعتراض شما ثبت شد!\n\nپرونده شما برای مدیریت ارسال شد. اگر حق با شما باشد، جایزه به شما بازگردانده می‌شود.\n\n⏳ منتظر نتیجه بمانید...", nil)

	// Notify judge
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)
	if passiveUser != nil {
		bot.SendMessage(passiveUser.TelegramID, "⚠️ حریف به داوری شما اعتراض کرد!\n\nپرونده به مدیریت فرستاده شد. اگر داوری شما ناعادلانه تشخیص داده شود، از امتیاز داوری شما کسر خواهد شد.", nil)
	}

	// Send to Admin
	if h.Config.SuperAdminTgID != 0 {
		turn, _ := h.TodRepo.GetCurrentTurn(gameID)
		if turn != nil {
			adminMsg := fmt.Sprintf(`⚖️ **درخواست تجدیدنظر (Appeal)**

🎮 **ID بازی:** %d
👤 **شرکت‌کننده:** %s (ID: %d)
⚖️ **داور:** %s (ID: %d)

🎯 **چالش:**
%s

📸 **مدرک را در پیام بعد مشاهده کنید.**`,
				gameID,
				user.FullName, user.ID,
				passiveUser.FullName, passiveUser.ID,
				turn.ChallengeText,
			)

			// Judgment buttons for admin
			adminKeyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ قبول اعتراض (حق با بازیکن)", fmt.Sprintf("btn:tod_adm_app_%d_%d_accept", gameID, turn.ID)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❌ رد اعتراض (حق با داور)", fmt.Sprintf("btn:tod_adm_app_%d_%d_reject", gameID, turn.ID)),
				),
			)

			bot.SendMessage(h.Config.SuperAdminTgID, adminMsg, adminKeyboard)

			// Forward the proof to admin
			h.forwardProof(h.Config.SuperAdminTgID, turn, bot)
		}
	}

	logger.Info("Appeal filed and sent to admin", "game_id", gameID, "player_id", userID)
}

// HandleTodAdminAppeal processes the admin decision on an appeal
func (h *HandlerManager) HandleTodAdminAppeal(adminTelegramID int64, gameID uint, turnID uint, result string, bot BotInterface) {
	// Verify admin
	if adminTelegramID != h.Config.SuperAdminTgID {
		return
	}

	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	turn, err := h.TodRepo.GetTurnByID(turnID)
	if err != nil {
		return
	}

	// We only process if game is in appeal state
	if game.State != models.TodStateWaitingAppeal {
		bot.SendMessage(adminTelegramID, "⚠️ این پرونده قبلاً بسته شده یا وضعیت بازی تغییر کرده است.", nil)
		return
	}

	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	if result == "accept" {
		// 1. Refund the -5 penalty
		h.CoinRepo.AddCoins(game.ActivePlayerID, 5, models.TxTypeRefund, "برگشت جریمه (قبول اعتراض توسط ادمین)")

		// 2. Award full rewards
		xpAwarded := 0
		coinsAwarded := 0
		if turn.Challenge != nil {
			xpAwarded = turn.Challenge.XPReward
			coinsAwarded = turn.Challenge.CoinReward
			h.CoinRepo.AddCoins(game.ActivePlayerID, int64(coinsAwarded), models.TxTypeGameReward, "پاداش چالش (قبول اعتراض توسط ادمین)")
			h.UserRepo.AddXP(game.ActivePlayerID, xpAwarded)
			h.VillageSvc.AddXPForUser(game.ActivePlayerID, int64(xpAwarded))
		}

		// 3. Penalize the judge (Passive Player)
		h.TodRepo.IncrementUnfairJudgmentCount(game.PassivePlayerID)
		h.CoinRepo.AddCoins(game.PassivePlayerID, -10, models.TxTypePenalty, "جریمه داوری ناعادلانه (تایید ادمین)")

		// 4. Update Turn Stats
		h.TodRepo.UpdateTurnJudgment(turn.ID, "accepted", "Admin override: "+result)
		h.TodRepo.UpdateTurnRewards(turn.ID, xpAwarded, coinsAwarded)

		// 5. Notify Players
		bot.SendMessage(activeUser.TelegramID, "⚖️ **اعتراض شما پذیرفته شد!**\n\nمدیریت حق را به شما داد. جریمه لغو شد و پاداش چالش به حساب شما واریز شد. 🎉", nil)
		bot.SendMessage(passiveUser.TelegramID, "⚖️ **نتیجه اعتراض:**\n\nمدیریت اعتراض حریف را پذیرفت. داوری شما ناعادلانه تشخیص داده شد و ۱۰ سکه جریمه شدید. لطفاً منصفانه داوری کنید. ⚠️", nil)
		bot.SendMessage(adminTelegramID, "✅ اعتراض پذیرفته شد و جوایز منتقل شد.", nil)
	} else {
		// Reject Appeal
		bot.SendMessage(activeUser.TelegramID, "⚖️ **اعتراض شما رد شد.**\n\nمدیریت داوری حریف را تایید کرد. جریمه پابرجا باقی می‌ماند.", nil)
		bot.SendMessage(passiveUser.TelegramID, "⚖️ **نتیجه اعتراض:**\n\nمدیریت اعتراض حریف را رد کرد و داوری شما را تایید کرد. ✅", nil)
		bot.SendMessage(adminTelegramID, "❌ اعتراض رد شد.", nil)
	}

	// Move to next round
	h.MoveToNextRound(gameID, bot)
}

func (h *HandlerManager) MoveToNextRound(gameID uint, bot BotInterface) {
	// Re-fetch game to get latest state
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Only move if not already finished or in another state
	// If it was in appeal state, we transition it.
	// If it was already moved by someone else, we skip.

	// Switch roles for next turn
	h.TodRepo.SwitchTurn(gameID)
	h.TodRepo.IncrementRound(gameID)
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingChoice)

	// Refresh game object after switch
	game, _ = h.TodRepo.GetGameByID(gameID)

	if game.CurrentRound > game.MaxRounds {
		h.TodRepo.EndGame(gameID, 0, "completed")
		h.UserRepo.UpdateUserStatus(game.Match.User1ID, models.UserStatusOnline)
		h.UserRepo.UpdateUserStatus(game.Match.User2ID, models.UserStatusOnline)

		msg := "🏁 **بازی به پایان رسید!**\n\nخسته نباشید، امیدوارم بهتون خوش گذشته باشه. باز هم می‌تونید بازی کنید! 🔥"
		bot.SendMessage(game.Match.User1.TelegramID, msg, bot.GetMainMenuKeyboard(false))
		bot.SendMessage(game.Match.User2.TelegramID, msg, bot.GetMainMenuKeyboard(false))
		return
	}

	// Notify players about next round
	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	roundMsg := fmt.Sprintf("🎮 **راند %d**", game.CurrentRound)

	// Active player chooses
	activeChoiceMsg := roundMsg + "\n🔔 **نوبت شماست!**\n\nنوع چالش رو انتخاب کن:"
	bot.SendMessage(activeUser.TelegramID, activeChoiceMsg, bot.GetTodChallengeTypeKeyboard(gameID))

	// Passive player waits
	passiveWaitMsg := roundMsg + fmt.Sprintf("\n⏳ منتظر انتخاب حریف (%s)...", activeUser.FullName)
	bot.SendMessage(passiveUser.TelegramID, passiveWaitMsg, nil)
}

// HandleTodNextRound manually triggers next round
func (h *HandlerManager) HandleTodNextRound(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	// Verify user is in game
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if user.ID != game.ActivePlayerID && user.ID != game.PassivePlayerID {
		return
	}

	h.MoveToNextRound(gameID, bot)
}
