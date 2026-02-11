// Part 3: Boosters, Round/Game End, Timeout - quiz_match_handler_part3.go

package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// ========================================
// BOOSTERS
// ========================================

func (h *HandlerManager) HandleBoosterRemove2(userID int64, matchID uint, questionNum int, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if user == nil {
		return
	}

	match, _ := h.QuizMatchRepo.GetQuizMatch(matchID)
	if match == nil || match.State == models.QuizStateRoundFinished || match.State == models.QuizStateGameFinished {
		return
	}

	session := getQuizGameSession(matchID)
	h.ensureQuizSessionLoaded(session, match)

	session.mu.Lock()

	usedBefore := false
	alreadyAnswered := false
	if match.User1ID == user.ID {
		usedBefore = session.User1UsedRemove2[questionNum]
		alreadyAnswered = session.User1AnsweredQ[questionNum]
	} else {
		usedBefore = session.User2UsedRemove2[questionNum]
		alreadyAnswered = session.User2AnsweredQ[questionNum]
	}

	if usedBefore {
		session.mu.Unlock()
		bot.SendMessage(userID, "⚠️ شما قبلاً از این بوستر استفاده کرده‌اید!", nil)
		return
	}

	if alreadyAnswered {
		session.mu.Unlock()
		bot.SendMessage(userID, "⚠️ شما قبلاً به این سوال پاسخ داده‌اید!", nil)
		return
	}

	if len(session.Questions) < questionNum {
		session.mu.Unlock()
		return
	}
	question := session.Questions[questionNum-1]
	session.mu.Unlock()

	err := h.QuizMatchRepo.UseBooster(user.ID, models.BoosterRemove2Options)
	if err != nil {
		bot.SendMessage(userID, "❌ بوستر کافی ندارید!", nil)
		return
	}

	session.mu.Lock()
	if match.User1ID == user.ID {
		session.User1UsedRemove2[questionNum] = true
	} else {
		session.User2UsedRemove2[questionNum] = true
	}
	session.mu.Unlock()

	var options []string
	json.Unmarshal([]byte(question.Options), &options)

	correctIdx := -1
	for i, opt := range options {
		if opt == question.CorrectAnswer {
			correctIdx = i
			break
		}
	}

	var newOptions []string
	var newIndices []int
	newOptions = append(newOptions, options[correctIdx])
	newIndices = append(newIndices, correctIdx)

	for i := range options {
		if i != correctIdx {
			newOptions = append(newOptions, options[i])
			newIndices = append(newIndices, i)
			break
		}
	}

	msg := fmt.Sprintf("✂️ بوستر فعال شد!\n\n❓ سؤال %d از %d\n\n*%s*\n\n", questionNum, models.QuizQuestionsPerRound, question.QuestionText)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, opt := range newOptions {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(opt, fmt.Sprintf("btn:qans_%d_%d_%d", matchID, questionNum, newIndices[i])),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	newMsgID := bot.SendMessage(userID, msg, keyboard)

	session.mu.Lock()
	if match.User1ID == user.ID {
		session.User1LastQMsgID = newMsgID
	} else {
		session.User2LastQMsgID = newMsgID
	}
	session.mu.Unlock()
	// Keyboard of old message is already removed by global handler in bot.go
}

func (h *HandlerManager) HandleBoosterRetry(userID int64, matchID uint, questionNum int, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if user == nil {
		return
	}

	match, _ := h.QuizMatchRepo.GetQuizMatch(matchID)
	if match == nil || match.State == models.QuizStateRoundFinished || match.State == models.QuizStateGameFinished {
		return
	}

	session := getQuizGameSession(matchID)
	h.ensureQuizSessionLoaded(session, match)

	// Keyboard is already removed by global handler in bot.go, no need to fetch and edit oldMsgID

	// Remove old keyboard immediately
	// Keyboard is already removed by global handler in bot.go

	session.mu.Lock()

	usedBefore := false
	alreadyAnswered := false
	if match.User1ID == user.ID {
		usedBefore = session.User1UsedRetry[questionNum]
		alreadyAnswered = session.User1AnsweredQ[questionNum]
	} else {
		usedBefore = session.User2UsedRetry[questionNum]
		alreadyAnswered = session.User2AnsweredQ[questionNum]
	}

	if usedBefore {
		session.mu.Unlock()
		bot.SendMessage(userID, "⚠️ شما قبلاً از این بوستر استفاده کرده‌اید!", nil)
		return
	}

	if !alreadyAnswered {
		session.mu.Unlock()
		bot.SendMessage(userID, "⚠️ شما هنوز به این سوال پاسخ نداده‌اید!", nil)
		return
	}

	if len(session.Questions) < questionNum {
		session.mu.Unlock()
		return
	}
	session.mu.Unlock()

	err := h.QuizMatchRepo.UseBooster(user.ID, models.BoosterSecondChance)
	if err != nil {
		bot.SendMessage(userID, "❌ بوستر کافی ندارید!", nil)
		return
	}

	session.mu.Lock()
	if match.User1ID == user.ID {
		session.User1UsedRetry[questionNum] = true
		session.User1AnsweredQ[questionNum] = false
		session.User1QuestionStart = time.Time{} // Reset timer
	} else {
		session.User2UsedRetry[questionNum] = true
		session.User2AnsweredQ[questionNum] = false
		session.User2QuestionStart = time.Time{} // Reset timer
	}
	session.mu.Unlock()

	// Delete previous answer from DB so a new one can be recorded
	h.QuizMatchRepo.DeleteUserAnswer(matchID, session.RoundID, user.ID, questionNum)

	// Update lights to show white again
	h.UpdateQuizLights(matchID, user.ID, questionNum, false, bot)

	bot.SendMessage(userID, "🛡 بوستر فعال شد! میتونی دوباره جواب بدی!", nil)

	time.Sleep(1 * time.Second)

	h.SendQuizQuestionToUser(matchID, user.ID, questionNum, bot)
}

// ========================================
// END ROUND
// ========================================

func (h *HandlerManager) EndQuizRound(matchID uint, bot BotInterface) {
	// Set state to round_finished immediately to prevent concurrent executions
	success, _ := h.QuizMatchRepo.UpdateQuizMatchStateAtomic(matchID, []string{
		models.QuizStateWaitingCategory,
		models.QuizStatePlayingQ1,
		models.QuizStatePlayingQ2,
		models.QuizStatePlayingQ3,
		models.QuizStatePlayingQ4,
	}, models.QuizStateRoundFinished)

	if !success {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	session := getQuizGameSession(matchID)

	user1Answers, _ := h.QuizMatchRepo.GetUserAnswers(matchID, session.RoundID, match.User1ID)
	user2Answers, _ := h.QuizMatchRepo.GetUserAnswers(matchID, session.RoundID, match.User2ID)

	user1Correct := 0
	user1Time := 0
	for _, ans := range user1Answers {
		if ans.IsCorrect {
			user1Correct++
		}
		user1Time += ans.TimeTakenMs
	}

	user2Correct := 0
	user2Time := 0
	for _, ans := range user2Answers {
		if ans.IsCorrect {
			user2Correct++
		}
		user2Time += ans.TimeTakenMs
	}

	h.QuizMatchRepo.UpdateRoundStats(session.RoundID, match.User1ID, user1Correct, user1Time)
	h.QuizMatchRepo.UpdateRoundStats(session.RoundID, match.User2ID, user2Correct, user2Time)

	totalUser1Correct := match.User1TotalCorrect + user1Correct
	totalUser2Correct := match.User2TotalCorrect + user2Correct
	totalUser1Time := match.User1TotalTimeMs + int64(user1Time)
	totalUser2Time := match.User2TotalTimeMs + int64(user2Time)

	h.QuizMatchRepo.UpdateQuizMatchScore(matchID, match.User1ID, totalUser1Correct, totalUser1Time)
	h.QuizMatchRepo.UpdateQuizMatchScore(matchID, match.User2ID, totalUser2Correct, totalUser2Time)

	msg1 := fmt.Sprintf("📊 پایان راند %d\n\n", match.CurrentRound)
	msg1 += fmt.Sprintf("✅ پاسخ صحیح شما: %d از %d\n", user1Correct, models.QuizQuestionsPerRound)
	msg1 += fmt.Sprintf("⏱ زمان شما: %.1f ثانیه\n\n", float64(user1Time)/1000.0)
	msg1 += fmt.Sprintf("👤 %s: %d صحیح | %.1fث\n\n", match.User2.FullName, user2Correct, float64(user2Time)/1000.0)
	msg1 += fmt.Sprintf("📈 امتیاز کل: %d - %d", totalUser1Correct, totalUser2Correct)

	msg2 := fmt.Sprintf("📊 پایان راند %d\n\n", match.CurrentRound)
	msg2 += fmt.Sprintf("✅ پاسخ صحیح شما: %d از %d\n", user2Correct, models.QuizQuestionsPerRound)
	msg2 += fmt.Sprintf("⏱ زمان شما: %.1f ثانیه\n\n", float64(user2Time)/1000.0)
	msg2 += fmt.Sprintf("👤 %s: %d صحیح | %.1fث\n\n", match.User1.FullName, user1Correct, float64(user1Time)/1000.0)
	msg2 += fmt.Sprintf("📈 امتیاز کل: %d - %d", totalUser2Correct, totalUser1Correct)

	bot.SendMessage(match.User1.TelegramID, msg1, nil)
	bot.SendMessage(match.User2.TelegramID, msg2, nil)

	time.Sleep(3 * time.Second)

	if match.CurrentRound >= models.QuizTotalRounds {
		h.EndQuizGame(matchID, bot)
	} else {
		h.QuizMatchRepo.AdvanceRound(matchID)
		h.QuizMatchRepo.SwitchTurn(matchID)

		session.mu.Lock()
		session.Questions = nil
		session.User1AnsweredQ = make(map[int]bool)
		session.User2AnsweredQ = make(map[int]bool)
		session.User1UsedRemove2 = make(map[int]bool)
		session.User2UsedRemove2 = make(map[int]bool)
		session.User1UsedRetry = make(map[int]bool)
		session.User2UsedRetry = make(map[int]bool)
		session.mu.Unlock()

		// Refresh match data to get updated turn and state
		match, _ = h.QuizMatchRepo.GetQuizMatch(matchID)

		// Send explicit notification to the turn user
		if match.TurnUserID != nil {
			var turnUserTgID int64
			if *match.TurnUserID == match.User1ID {
				turnUserTgID = match.User1.TelegramID
			} else {
				turnUserTgID = match.User2.TelegramID
			}
			bot.SendMessage(turnUserTgID, "🔔 نوبت شماست! راند جدید آغاز شد.", nil)
		}

		h.ShowQuizGameDetail(match.User1.TelegramID, matchID, bot)
		h.ShowQuizGameDetail(match.User2.TelegramID, matchID, bot)
	}
}

// ========================================
// END GAME
// ========================================

func (h *HandlerManager) EndQuizGame(matchID uint, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	var winnerID uint

	if match.User1TotalCorrect > match.User2TotalCorrect {
		winnerID = match.User1ID
	} else if match.User2TotalCorrect > match.User1TotalCorrect {
		winnerID = match.User2ID
	} else {
		if match.User1TotalTimeMs < match.User2TotalTimeMs {
			winnerID = match.User1ID
		} else if match.User2TotalTimeMs < match.User1TotalTimeMs {
			winnerID = match.User2ID
		}
	}

	if winnerID > 0 {
		loserID := match.User1ID
		if winnerID == match.User1ID {
			loserID = match.User2ID
		}
		h.QuizMatchRepo.FinishQuizMatch(matchID, winnerID)
		h.CoinRepo.AddCoins(winnerID, int64(models.QuizWinRewardCoins), models.TxTypeGameReward, "Quiz game win reward")
		h.UserRepo.AddXP(winnerID, models.QuizWinRewardXP)
		h.UserRepo.AddXP(loserID, models.QuizLoseRewardXP)
		h.UserRepo.AddLeaguePoints(winnerID, 20)

		// Village rewards
		h.VillageSvc.UpdateWarScore(winnerID, 15)
		h.VillageSvc.AddXPForUser(winnerID, 60)
	} else {
		h.QuizMatchRepo.FinishQuizMatch(matchID, 0)
		h.CoinRepo.AddCoins(match.User1ID, int64(models.QuizDrawRewardCoins), models.TxTypeGameReward, "Quiz game draw reward")
		h.CoinRepo.AddCoins(match.User2ID, int64(models.QuizDrawRewardCoins), models.TxTypeGameReward, "Quiz game draw reward")
		h.UserRepo.AddXP(match.User1ID, models.QuizDrawRewardXP)
		h.UserRepo.AddXP(match.User2ID, models.QuizDrawRewardXP)
		h.UserRepo.AddLeaguePoints(match.User1ID, 8)
		h.UserRepo.AddLeaguePoints(match.User2ID, 8)

		// Village rewards for draw
		h.VillageSvc.AddXPForUser(match.User1ID, 25)
		h.VillageSvc.AddXPForUser(match.User2ID, 25)
	}

	msg1 := "🎮 بازی تمام شد!\n\n"
	msg1 += "📊 نتیجه نهایی:\n"
	msg1 += fmt.Sprintf("👤 شما: %d صحیح | ⏱ %.1fث\n", match.User1TotalCorrect, float64(match.User1TotalTimeMs)/1000.0)
	msg1 += fmt.Sprintf("👤 %s: %d صحیح | ⏱ %.1fث\n\n", match.User2.FullName, match.User2TotalCorrect, float64(match.User2TotalTimeMs)/1000.0)

	switch winnerID {
	case match.User1ID:
		msg1 += "🏆 شما برنده شدید!\n\n"
		msg1 += fmt.Sprintf("💰 پاداش: +%d سکه | ⭐ +%d امتیاز تجربه", models.QuizWinRewardCoins, models.QuizWinRewardXP)
	case match.User2ID:
		msg1 += "❌ شما باختید!\n\n"
		msg1 += fmt.Sprintf("⭐ پاداش تلاش: +%d امتیاز تجربه", models.QuizLoseRewardXP)
	default:
		msg1 += "🤝 مساوی!\n\n"
		msg1 += fmt.Sprintf("💰 پاداش: +%d سکه | ⭐ +%d امتیاز تجربه", models.QuizDrawRewardCoins, models.QuizDrawRewardXP)
	}

	msg2 := "🎮 بازی تمام شد!\n\n"
	msg2 += "📊 نتیجه نهایی:\n"
	msg2 += fmt.Sprintf("👤 شما: %d صحیح | ⏱ %.1fث\n", match.User2TotalCorrect, float64(match.User2TotalTimeMs)/1000.0)
	msg2 += fmt.Sprintf("👤 %s: %d صحیح | ⏱ %.1fث\n\n", match.User1.FullName, match.User1TotalCorrect, float64(match.User1TotalTimeMs)/1000.0)

	switch winnerID {
	case match.User2ID:
		msg2 += "🏆 شما برنده شدید!\n\n"
		msg2 += fmt.Sprintf("💰 پاداش: +%d سکه | ⭐ +%d امتیاز تجربه", models.QuizWinRewardCoins, models.QuizWinRewardXP)
	case match.User1ID:
		msg2 += "❌ شما باختید!\n\n"
		msg2 += fmt.Sprintf("⭐ پاداش تلاش: +%d امتیاز تجربه", models.QuizLoseRewardXP)
	default:
		msg2 += "🤝 مساوی!\n\n"
		msg2 += fmt.Sprintf("💰 پاداش: +%d سکه | ⭐ +%d امتیاز تجربه", models.QuizDrawRewardCoins, models.QuizDrawRewardXP)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎮 بازی جدید", "btn:new_quiz_game"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 بازیهای من", "btn:quiz_games"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 منوی اصلی", "btn:main_menu"),
		),
	)

	bot.SendMessage(match.User1.TelegramID, msg1, keyboard)
	bot.SendMessage(match.User2.TelegramID, msg2, keyboard)

	cleanupQuizGameSession(matchID)

	// Set status back to online if no other active games
	h.updateQuizPlayerStatus(match.User1ID)
	h.updateQuizPlayerStatus(match.User2ID)
}

func (h *HandlerManager) updateQuizPlayerStatus(userID uint) {
	// Don't change searching status
	user, err := h.UserRepo.GetUserByID(userID)
	if err != nil || user == nil || user.Status == models.UserStatusSearching || user.Status == models.UserStatusOnline {
		return
	}

	activeQuiz, _ := h.QuizMatchRepo.GetAllActiveQuizMatchesByUser(userID)
	activeTod, _ := h.TodRepo.GetActiveGameForUser(userID)

	if len(activeQuiz) == 0 && activeTod == nil {
		h.UserRepo.UpdateUserStatus(userID, models.UserStatusOnline)
	}
}

// ========================================
// TIMEOUT MANAGEMENT
// ========================================

func (h *HandlerManager) CheckQuizTimeouts(bot BotInterface) {
	matches, err := h.QuizMatchRepo.GetTimeoutMatches()
	if err != nil {
		logger.Error("Failed to get timeout matches", "error", err)
		return
	}

	for _, match := range matches {
		h.HandleQuizTimeout(match.ID, bot)
	}
}

func (h *HandlerManager) HandleQuizTimeout(matchID uint, bot BotInterface) {
	// Atomic check and update to timeout state
	success, _ := h.QuizMatchRepo.UpdateQuizMatchStateAtomic(matchID, []string{
		models.QuizStateWaitingCategory,
		models.QuizStatePlayingQ1,
		models.QuizStatePlayingQ2,
		models.QuizStatePlayingQ3,
		models.QuizStatePlayingQ4,
		models.QuizStateRoundFinished,
	}, models.QuizStateTimeout)

	if !success {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	var inactiveName string
	if match.TurnUserID != nil {
		if *match.TurnUserID == match.User1ID {
			inactiveName = match.User1.FullName
		} else {
			inactiveName = match.User2.FullName
		}
	} else {
		inactiveName = match.User1.FullName
	}

	msg := fmt.Sprintf("⏰ بازی به دلیل عدم فعالیت %s به پایان رسید.\n\nهیچ امتیاز یا سکهای تعلق نگرفت.", inactiveName)

	bot.SendMessage(match.User1.TelegramID, msg, nil)
	bot.SendMessage(match.User2.TelegramID, msg, nil)

	if match.TurnUserID != nil {
		h.UserRepo.AddLeaguePoints(*match.TurnUserID, -10)
		winnerID := match.User1ID
		if *match.TurnUserID == match.User1ID {
			winnerID = match.User2ID
		}
		h.UserRepo.AddLeaguePoints(winnerID, 10)
		h.CoinRepo.AddCoins(winnerID, 20, models.TxTypeGameReward, "پاداش برد به دلیل AFK حریف")
		h.UserRepo.AddXP(winnerID, 15)

		// Village rewards
		h.VillageSvc.UpdateWarScore(winnerID, 8)
		h.VillageSvc.AddXPForUser(winnerID, 35)
	}

	cleanupQuizGameSession(matchID)

	// Set status back to online if no other active games
	h.updateQuizPlayerStatus(match.User1ID)
	h.updateQuizPlayerStatus(match.User2ID)

	logger.Info("Quiz match timed out", "match_id", matchID)
}

func (h *HandlerManager) HandleQuizForceWin(userID int64, matchID uint, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	// Check if 24 hours passed since last activity
	if time.Since(match.LastActivityAt) < 24*time.Hour {
		bot.SendMessage(userID, "⏳ نوبتش نگذشته! فقط در صورتی که حریف بیش از ۲۴ ساعت غیرفعال باشد می‌توانید برد فنی بگیرید.", nil)
		return
	}

	h.HandleQuizTimeout(matchID, bot)
}

// ========================================
// BACKWARD COMPATIBILITY
// ========================================

// StartQuiz - Legacy function redirects to Glass Menu
func (h *HandlerManager) StartQuiz(userID int64, bot BotInterface) {
	h.ShowActiveQuizGames(userID, bot)
}

// HandleQuizCategorySelection - Legacy compatibility
func (h *HandlerManager) HandleQuizCategorySelection(userID int64, matchID uint, category string, bot BotInterface) {
	h.HandleCategorySelection(userID, matchID, category, bot)
}
