// Part 2 of quiz_match_handler.go - Category Selection, Questions, Boosters, End Game

package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
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

	categories := []string{"اطلاعات عمومی", "تاریخ", "ورزش", "سینما", "جغرافیا", "علم"}
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

	round, err := h.QuizMatchRepo.CreateQuizRound(matchID, match.CurrentRound, category, user.ID)
	if err != nil {
		logger.Error("Failed to create round", "error", err)
		return
	}

	h.QuizMatchRepo.UpdateQuizMatchState(matchID, models.QuizStatePlayingQ1)
	h.QuizMatchRepo.UpdateCurrentQuestion(matchID, 1)

	session.mu.Lock()
	session.RoundID = round.ID
	session.Questions = questions
	session.QuestionNumber = 1
	session.User1QuestionStart = time.Now()
	session.User2QuestionStart = time.Now()
	session.mu.Unlock()

	opponentID := match.User2ID
	if user.ID == match.User2ID {
		opponentID = match.User1ID
	}
	opponent, _ := h.UserRepo.GetUserByID(opponentID)

	msg := fmt.Sprintf("✅ موضوع انتخاب شد: *%s*\n\nسؤالات شروع شد!", category)
	bot.SendMessage(user.TelegramID, msg, nil)
	if opponent != nil {
		bot.SendMessage(opponent.TelegramID, msg, nil)
	}

	time.Sleep(1 * time.Second)

	lightsMsg1 := bot.SendMessage(match.User1.TelegramID, "⚪️ ⚪️ ⚪️ ⚪️", nil)
	lightsMsg2 := bot.SendMessage(match.User2.TelegramID, "⚪️ ⚪️ ⚪️ ⚪️", nil)

	h.QuizMatchRepo.UpdateLightsMessageID(matchID, match.User1ID, lightsMsg1)
	h.QuizMatchRepo.UpdateLightsMessageID(matchID, match.User2ID, lightsMsg2)

	h.SendQuizQuestion(matchID, 1, bot)
}

// ========================================
// QUESTION ARENA
// ========================================

func (h *HandlerManager) SendQuizQuestion(matchID uint, questionNum int, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()
	if questionNum > len(session.Questions) {
		session.mu.Unlock()
		return
	}
	question := session.Questions[questionNum-1]
	session.QuestionNumber = questionNum
	session.mu.Unlock()

	var options []string
	if err := json.Unmarshal([]byte(question.Options), &options); err != nil {
		logger.Error("Failed to parse options", "error", err)
		return
	}

	h.sendQuestionToUser(match.User1.TelegramID, match.User1ID, matchID, questionNum, question, options, bot)
	h.sendQuestionToUser(match.User2.TelegramID, match.User2ID, matchID, questionNum, question, options, bot)

	session.mu.Lock()
	if session.QuestionTimer != nil {
		session.QuestionTimer.Stop()
	}
	session.QuestionTimer = time.AfterFunc(time.Duration(models.QuizQuestionTimeSeconds)*time.Second, func() {
		h.HandleQuestionTimeout(matchID, questionNum, bot)
	})
	session.mu.Unlock()
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
		bot.SendMessage(userID, "⚠️ شما قبلاً به این سؤال پاسخ دادهاید!", nil)
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

	session.mu.Lock()
	bothAnswered := session.User1AnsweredQ[questionNum] && session.User2AnsweredQ[questionNum]
	session.mu.Unlock()

	if bothAnswered {
		session.mu.Lock()
		if session.QuestionTimer != nil {
			session.QuestionTimer.Stop()
		}
		session.mu.Unlock()

		time.Sleep(2 * time.Second)

		if questionNum < models.QuizQuestionsPerRound {
			nextQ := questionNum + 1
			nextState := fmt.Sprintf("playing_q%d", nextQ)
			h.QuizMatchRepo.UpdateQuizMatchState(matchID, nextState)
			h.QuizMatchRepo.UpdateCurrentQuestion(matchID, nextQ)

			session.mu.Lock()
			session.User1QuestionStart = time.Now()
			session.User2QuestionStart = time.Now()
			session.mu.Unlock()

			h.SendQuizQuestion(matchID, nextQ, bot)
		} else {
			h.EndQuizRound(matchID, bot)
		}
	}
}

// ========================================
// HANDLE QUESTION TIMEOUT
// ========================================

func (h *HandlerManager) HandleQuestionTimeout(matchID uint, questionNum int, bot BotInterface) {
	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	session := getQuizGameSession(matchID)
	session.mu.Lock()

	if !session.User1AnsweredQ[questionNum] {
		session.User1AnsweredQ[questionNum] = true
		session.mu.Unlock()

		if questionNum <= len(session.Questions) {
			question := session.Questions[questionNum-1]
			h.QuizMatchRepo.RecordAnswer(matchID, session.RoundID, match.User1ID, question.ID, questionNum, -1, models.QuizQuestionTimeSeconds*1000, false, "")
			bot.SendMessage(match.User1.TelegramID, "⏰ زمان تمام شد! پاسخ شما: غلط", nil)
			h.UpdateQuizLights(matchID, match.User1ID, questionNum, false, bot)
		}

		session.mu.Lock()
	}

	if !session.User2AnsweredQ[questionNum] {
		session.User2AnsweredQ[questionNum] = true
		session.mu.Unlock()

		if questionNum <= len(session.Questions) {
			question := session.Questions[questionNum-1]
			h.QuizMatchRepo.RecordAnswer(matchID, session.RoundID, match.User2ID, question.ID, questionNum, -1, models.QuizQuestionTimeSeconds*1000, false, "")
			bot.SendMessage(match.User2.TelegramID, "⏰ زمان تمام شد! پاسخ شما: غلط", nil)
			h.UpdateQuizLights(matchID, match.User2ID, questionNum, false, bot)
		}

		session.mu.Lock()
	}
	session.mu.Unlock()

	time.Sleep(2 * time.Second)

	if questionNum < models.QuizQuestionsPerRound {
		nextQ := questionNum + 1
		nextState := fmt.Sprintf("playing_q%d", nextQ)
		h.QuizMatchRepo.UpdateQuizMatchState(matchID, nextState)
		h.QuizMatchRepo.UpdateCurrentQuestion(matchID, nextQ)

		session.mu.Lock()
		session.User1QuestionStart = time.Now()
		session.User2QuestionStart = time.Now()
		session.mu.Unlock()

		h.SendQuizQuestion(matchID, nextQ, bot)
	} else {
		h.EndQuizRound(matchID, bot)
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
