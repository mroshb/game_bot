// Part 2 of quiz_match_handler.go - Category Selection, Questions, Boosters, End Game

package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// ========================================
// CATEGORY SELECTION
// ========================================

func (h *HandlerManager) ShowCategorySelection(userID int64, matchID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	if match.TurnUserID == nil || *match.TurnUserID != user.ID {
		bot.SendMessage(userID, "⚠️ الان نوبت شما نیست!", nil)
		return
	}

	categories := []string{"اطلاعات عمومی", "تاریخ", "فوتبال", "تکنولوژی", "جغرافیا", "بازی های ویدیویی", "مذهبی", "زبان انگلیسی", "ریاضی"}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(categories), func(i, j int) {
		categories[i], categories[j] = categories[j], categories[i]
	})
	selectedCats := categories[:3]

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, cat := range selectedCats {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cat, fmt.Sprintf("btn:qcat_%d_%s", matchID, cat)),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := fmt.Sprintf("🎭 راند %d: نوبت شماست که موضوع رو انتخاب کنی!\n\n⏱ زمان: %d ثانیه\n\nیکی از موضوعات زیر رو انتخاب کن:", match.CurrentRound, models.QuizCategoryTimeSeconds)

	bot.SendMessage(userID, msg, keyboard)

	session := getQuizGameSession(matchID)
	session.mu.Lock()
	if session.CategoryTimer != nil {
		session.CategoryTimer.Stop()
	}
	session.CategoryTimer = time.AfterFunc(time.Duration(models.QuizCategoryTimeSeconds)*time.Second, func() {
		h.HandleCategorySelection(userID, matchID, selectedCats[0], bot)
	})
	session.mu.Unlock()
}

func (h *HandlerManager) HandleCategorySelection(userID int64, matchID uint, category string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	if match.State != models.QuizStateWaitingCategory {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()
	if session.CategoryTimer != nil {
		session.CategoryTimer.Stop()
	}
	session.mu.Unlock()

	questions, err := h.GameRepo.GetQuestionsByCategory(category, models.QuizQuestionsPerRound)
	if err != nil || len(questions) < models.QuizQuestionsPerRound {
		questions, _ = h.GameRepo.GetQuizQuestions(models.QuizQuestionsPerRound)
	}

	// Store question IDs
	var qIDs []string
	for _, q := range questions {
		qIDs = append(qIDs, fmt.Sprintf("%d", q.ID))
	}
	questionIDsStr := strings.Join(qIDs, ",")

	round, err := h.QuizMatchRepo.CreateQuizRound(matchID, match.CurrentRound, category, user.ID, questionIDsStr)
	if err != nil {
		logger.Error("Failed to create round", "error", err)
		return
	}

	h.QuizMatchRepo.UpdateQuizMatchState(matchID, models.QuizStatePlayingQ1)
	h.QuizMatchRepo.UpdateCurrentQuestion(matchID, 1)

	session.mu.Lock()
	session.RoundID = round.ID
	session.Questions = questions
	// We'll use shared Questions list but start playing only for the current user
	if match.User1ID == user.ID {
		session.User1QuestionStart = time.Now()
	} else {
		session.User2QuestionStart = time.Now()
	}
	session.mu.Unlock()

	opponentID := match.User2ID
	if user.ID == match.User2ID {
		opponentID = match.User1ID
	}
	opponent, _ := h.UserRepo.GetUserByID(opponentID)

	msg := fmt.Sprintf("✅ موضوع انتخاب شد: *%s*\n\nآماده‌ای؟ سؤالات شروع شد!", category)
	bot.SendMessage(user.TelegramID, msg, nil)

	if opponent != nil {
		oppMsg := fmt.Sprintf("🎭 %s موضوع را انتخاب کرد: *%s*\n\nهر وقت آماده بودی بازی رو شروع کن!", user.FullName, category)
		bot.SendMessage(opponent.TelegramID, oppMsg, nil)
		// Send game board to opponent so they can see "Start Round" button
		h.ShowQuizGameDetail(opponent.TelegramID, matchID, bot)
	}

	time.Sleep(1 * time.Second)

	// User who selected category starts immediately
	msgID := bot.SendMessage(user.TelegramID, "⚪️ ⚪️ ⚪️ ⚪️", nil)
	h.QuizMatchRepo.UpdateLightsMessageID(matchID, user.ID, msgID)

	h.SendQuizQuestionToUser(matchID, user.ID, 1, bot)
}

func (h *HandlerManager) SendQuizQuestionToUser(matchID uint, userID uint, questionNum int, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	user, _ := h.UserRepo.GetUserByID(userID)
	if user == nil {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()

	// If session questions are empty (e.g. after restart), reload them from DB
	if len(session.Questions) == 0 {
		round, _ := h.QuizMatchRepo.GetQuizRound(matchID, match.CurrentRound)
		if round != nil && round.QuestionIDs != "" {
			ids := strings.Split(round.QuestionIDs, ",")
			for _, idStr := range ids {
				var id uint
				fmt.Sscanf(idStr, "%d", &id)
				q, _ := h.GameRepo.GetQuestionByID(id)
				if q != nil {
					session.Questions = append(session.Questions, *q)
				}
			}
			session.RoundID = round.ID
		}
	}

	if questionNum > len(session.Questions) {
		session.mu.Unlock()
		return
	}
	question := session.Questions[questionNum-1]

	if userID == match.User1ID {
		session.User1QuestionStart = time.Now()
	} else {
		session.User2QuestionStart = time.Now()
	}
	session.mu.Unlock()

	var options []string
	if err := json.Unmarshal([]byte(question.Options), &options); err != nil {
		logger.Error("Failed to parse options", "error", err)
		return
	}

	h.sendQuestionToUser(user.TelegramID, userID, matchID, questionNum, question, options, bot)

	// Per-user timer
	go func(mID, uID uint, qNum int) {
		time.Sleep(time.Duration(models.QuizQuestionTimeSeconds) * time.Second)
		h.HandleUserQuestionTimeout(mID, uID, qNum, bot)
	}(matchID, userID, questionNum)
}

// ========================================
// QUESTION ARENA
// ========================================

func (h *HandlerManager) HandleQuizPlay(userID int64, matchID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	round, _ := h.QuizMatchRepo.GetQuizRound(matchID, match.CurrentRound)
	if round == nil {
		bot.SendMessage(userID, "❌ این راند هنوز آماده نشده است!", nil)
		return
	}

	// Check how many questions answered
	ans, _ := h.QuizMatchRepo.GetUserAnswers(matchID, round.ID, user.ID)
	nextQ := len(ans) + 1

	if nextQ > models.QuizQuestionsPerRound {
		bot.SendMessage(userID, "⚠️ شما تمام سؤالات این راند رو پاسخ دادید!", nil)
		return
	}

	// Initialize lights if it's the first question
	if nextQ == 1 {
		msgID := bot.SendMessage(userID, "⚪️ ⚪️ ⚪️ ⚪️", nil)
		h.QuizMatchRepo.UpdateLightsMessageID(matchID, user.ID, msgID)
	}

	// Start from the next question
	h.SendQuizQuestionToUser(matchID, user.ID, nextQ, bot)
}

func (h *HandlerManager) sendQuestionToUser(userTgID int64, userID, matchID uint, questionNum int, question models.Question, options []string, bot BotInterface) {
	session := getQuizGameSession(matchID)
	match, _ := h.QuizMatchRepo.GetQuizMatch(matchID)
	if match == nil {
		return
	}

	msg := fmt.Sprintf("❓ سؤال %d از %d\n\n", questionNum, models.QuizQuestionsPerRound)
	msg += fmt.Sprintf("*%s*\n\n", question.QuestionText)
	msg += "⏱ زمان: 10 ثانیه\n\n"

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, opt := range options {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(opt, fmt.Sprintf("btn:qans_%d_%d_%d", matchID, questionNum, i)),
		))
	}

	user1Booster, _ := h.QuizMatchRepo.GetUserBooster(userID, models.BoosterRemove2Options)
	user2Booster, _ := h.QuizMatchRepo.GetUserBooster(userID, models.BoosterSecondChance)

	var boosterRow []tgbotapi.InlineKeyboardButton

	session.mu.Lock()
	usedRemove2 := false
	usedRetry := false
	if match.User1ID == userID {
		usedRemove2 = session.User1UsedRemove2[questionNum]
		usedRetry = session.User1UsedRetry[questionNum]
	} else {
		usedRemove2 = session.User2UsedRemove2[questionNum]
		usedRetry = session.User2UsedRetry[questionNum]
	}
	session.mu.Unlock()

	if !usedRemove2 && user1Booster != nil && user1Booster.Quantity > 0 {
		boosterRow = append(boosterRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("✂️ حذف 2 گزینه (%d)", user1Booster.Quantity),
			fmt.Sprintf("btn:qboost_r2_%d_%d", matchID, questionNum),
		))
	}

	if !usedRetry && user2Booster != nil && user2Booster.Quantity > 0 {
		boosterRow = append(boosterRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🛡 شانس مجدد (%d)", user2Booster.Quantity),
			fmt.Sprintf("btn:qboost_rt_%d_%d", matchID, questionNum),
		))
	}

	if len(boosterRow) > 0 {
		rows = append(rows, boosterRow)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.SendMessage(userTgID, msg, keyboard)
}

// ========================================
// HANDLE ANSWER
// ========================================

func (h *HandlerManager) HandleQuizAnswer(userID int64, matchID uint, questionNum, answerIdx int, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()

	alreadyAnswered := false
	if match.User1ID == user.ID {
		alreadyAnswered = session.User1AnsweredQ[questionNum]
	} else {
		alreadyAnswered = session.User2AnsweredQ[questionNum]
	}

	if alreadyAnswered {
		session.mu.Unlock()
		// Silently ignore if already answered (might be a double click)
		return
	}

	if match.User1ID == user.ID {
		session.User1AnsweredQ[questionNum] = true
	} else {
		session.User2AnsweredQ[questionNum] = true
	}

	var timeTaken time.Duration
	if match.User1ID == user.ID {
		timeTaken = time.Since(session.User1QuestionStart)
	} else {
		timeTaken = time.Since(session.User2QuestionStart)
	}
	timeMs := int(timeTaken.Milliseconds())

	if questionNum > len(session.Questions) {
		session.mu.Unlock()
		return
	}
	question := session.Questions[questionNum-1]
	session.mu.Unlock()

	var options []string
	json.Unmarshal([]byte(question.Options), &options)

	isCorrect := false
	for i, opt := range options {
		if opt == question.CorrectAnswer {
			if i == answerIdx {
				isCorrect = true
			}
			break
		}
	}

	err = h.QuizMatchRepo.RecordAnswer(matchID, session.RoundID, user.ID, question.ID, questionNum, answerIdx, timeMs, isCorrect, "")
	if err != nil {
		logger.Error("Failed to record answer", "error", err)
		return
	}

	if isCorrect {
		bot.SendMessage(userID, fmt.Sprintf("✅ صحیح! ⏱ %.1f ثانیه", float64(timeMs)/1000.0), nil)
	} else {
		bot.SendMessage(userID, fmt.Sprintf("❌ غلط! پاسخ صحیح: %s", question.CorrectAnswer), nil)
	}

	h.UpdateQuizLights(matchID, user.ID, questionNum, isCorrect, bot)

	time.Sleep(1500 * time.Millisecond)

	if questionNum < models.QuizQuestionsPerRound {
		nextQ := questionNum + 1
		h.SendQuizQuestionToUser(matchID, user.ID, nextQ, bot)
	} else {
		// This user finished. Check if both finished.
		session.mu.Lock()
		user1Finished := true
		user2Finished := true
		for i := 1; i <= models.QuizQuestionsPerRound; i++ {
			if !session.User1AnsweredQ[i] {
				user1Finished = false
			}
			if !session.User2AnsweredQ[i] {
				user2Finished = false
			}
		}
		session.mu.Unlock()

		if user1Finished && user2Finished {
			h.EndQuizRound(matchID, bot)
		} else {
			bot.SendMessage(userID, "🏁 شما سؤالات این راند رو تموم کردی!\n\nمنتظر بمون تا حریف هم بازیش رو انجام بده.", nil)
			// Show updated game board
			h.ShowQuizGameDetail(userID, matchID, bot)
		}
	}
}

// ========================================
// HANDLE QUESTION TIMEOUT
// ========================================

func (h *HandlerManager) HandleUserQuestionTimeout(matchID, userID uint, questionNum int, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	user, _ := h.UserRepo.GetUserByID(userID)
	if user == nil {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()

	alreadyAnswered := false
	if match.User1ID == userID {
		alreadyAnswered = session.User1AnsweredQ[questionNum]
	} else {
		alreadyAnswered = session.User2AnsweredQ[questionNum]
	}

	if alreadyAnswered {
		session.mu.Unlock()
		return
	}

	if match.User1ID == userID {
		session.User1AnsweredQ[questionNum] = true
	} else {
		session.User2AnsweredQ[questionNum] = true
	}

	if questionNum > len(session.Questions) {
		session.mu.Unlock()
		return
	}
	question := session.Questions[questionNum-1]
	session.mu.Unlock()

	// Record wrong answer for timeout
	h.QuizMatchRepo.RecordAnswer(matchID, session.RoundID, userID, question.ID, questionNum, -1, models.QuizQuestionTimeSeconds*1000, false, "")
	bot.SendMessage(user.TelegramID, "⏰ زمان تمام شد! پاسخ شما: غلط", nil)
	h.UpdateQuizLights(matchID, userID, questionNum, false, bot)

	time.Sleep(1500 * time.Millisecond)

	if questionNum < models.QuizQuestionsPerRound {
		nextQ := questionNum + 1
		h.SendQuizQuestionToUser(matchID, userID, nextQ, bot)
	} else {
		// This user finished. Check if both finished.
		session.mu.Lock()
		user1Finished := true
		user2Finished := true
		for i := 1; i <= models.QuizQuestionsPerRound; i++ {
			if !session.User1AnsweredQ[i] {
				user1Finished = false
			}
			if !session.User2AnsweredQ[i] {
				user2Finished = false
			}
		}
		session.mu.Unlock()

		if user1Finished && user2Finished {
			h.EndQuizRound(matchID, bot)
		} else {
			bot.SendMessage(user.TelegramID, "🏁 شما سؤالات این راند رو تموم کردی!\n\nمنتظر بمون تا حریف هم بازیش رو انجام بده.", nil)
			h.ShowQuizGameDetail(user.TelegramID, matchID, bot)
		}
	}
}

// ========================================
// UPDATE LIGHTS
// ========================================

func (h *HandlerManager) UpdateQuizLights(matchID, userID uint, questionNum int, isCorrect bool, bot BotInterface) {
	match, _ := h.QuizMatchRepo.GetQuizMatch(matchID)
	if match == nil {
		return
	}

	var messageID int
	var chatID int64

	if match.User1ID == userID {
		messageID = match.User1LightsMsgID
		chatID = match.User1.TelegramID
	} else {
		messageID = match.User2LightsMsgID
		chatID = match.User2.TelegramID
	}

	if messageID == 0 {
		return
	}

	session := getQuizGameSession(matchID)
	answers, _ := h.QuizMatchRepo.GetUserAnswers(matchID, session.RoundID, userID)

	lights := ""
	for i := 1; i <= models.QuizQuestionsPerRound; i++ {
		found := false
		for _, ans := range answers {
			if ans.QuestionNumber == i {
				if ans.IsCorrect {
					lights += "🟢 "
				} else {
					lights += "🔴 "
				}
				found = true
				break
			}
		}
		if !found {
			lights += "⚪️ "
		}
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, lights)
	bot.GetAPI().(*tgbotapi.BotAPI).Send(edit)
}

// Continued in quiz_match_handler_part3.go...
