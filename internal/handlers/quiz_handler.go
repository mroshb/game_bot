package handlers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

type QuizSession struct {
	GameSessionID     uint
	Questions         []models.Question
	CurrentQuestion   int
	User1ID           uint
	User2ID           uint
	User1Score        int
	User2Score        int
	QuestionStartTime time.Time
	AnsweredUsers     map[uint]bool // userID -> true
	mu                sync.Mutex    // Protects session-specific data
}

var (
	quizSessions   = make(map[uint]*QuizSession) // matchID -> QuizSession
	quizSessionsMu sync.RWMutex
)

// StartQuiz starts a quiz game
func (h *HandlerManager) StartQuiz(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if user is in an active match
	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil {
		logger.Error("Failed to get active match", "error", err)
		bot.SendMessage(userID, "❌ خطایی رخ داد!", nil)
		return
	}

	if match == nil {
		bot.SendMessage(userID, "⚠️ شما در چت فعالی نیستید!", nil)
		return
	}

	// Check if quiz already in progress
	quizSessionsMu.RLock()
	_, exists := quizSessions[match.ID]
	quizSessionsMu.RUnlock()
	if exists {
		bot.SendMessage(userID, "⚠️ بازی کوییز در حال انجام است!", nil)
		return
	}

	// Get 5 random quiz questions
	questions, err := h.GameRepo.GetQuizQuestions(5)
	if err != nil || len(questions) < 5 {
		logger.Error("Failed to get quiz questions", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت سوالات! لطفاً بعداً تلاش کنید.", nil)
		return
	}

	// Create quiz session
	quizSession := &QuizSession{
		Questions:         questions,
		CurrentQuestion:   0,
		User1ID:           match.User1ID,
		User2ID:           match.User2ID,
		User1Score:        0,
		User2Score:        0,
		QuestionStartTime: time.Now(),
	}
	quizSessionsMu.Lock()
	quizSessions[match.ID] = quizSession
	quizSessionsMu.Unlock()

	// Get other user
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

	// Notify both users
	msg := "🎮 بازی کوییز شروع شد!\n\n5 سوال - 10 ثانیه برای هر سوال\n\nآماده باش!"
	bot.SendMessage(userID, msg, nil)
	if otherUser != nil {
		bot.SendMessage(otherUser.TelegramID, msg, nil)
	}

	// Wait a moment then send first question in a goroutine
	go func() {
		time.Sleep(2 * time.Second)
		h.SendQuizQuestion(match.ID, bot)
	}()
}

// SendQuizQuestion sends the current quiz question to both users
func (h *HandlerManager) SendQuizQuestion(matchID uint, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		return
	}

	session.mu.Lock()
	if session.CurrentQuestion >= len(session.Questions) {
		session.mu.Unlock()
		h.EndQuiz(matchID, bot)
		return
	}

	question := session.Questions[session.CurrentQuestion]
	session.QuestionStartTime = time.Now()
	session.AnsweredUsers = make(map[uint]bool) // Reset for new question
	currentIdx := session.CurrentQuestion       // Capture current index for timer check
	session.mu.Unlock()

	// Parse options from JSON
	var options []string
	if err := json.Unmarshal([]byte(question.Options), &options); err != nil {
		logger.Error("Failed to parse question options", "error", err)
		options = []string{"گزینه 1", "گزینه 2", "گزینه 3", "گزینه 4"}
	}

	// Create inline keyboard with options
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("quiz_%d_0", matchID)),
			tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("quiz_%d_1", matchID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("quiz_%d_2", matchID)),
			tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("quiz_%d_3", matchID)),
		),
	)

	msg := fmt.Sprintf("❓ سوال %d از 5:\n\n%s\n\n⏱ زمان: 10 ثانیه",
		session.CurrentQuestion+1, question.QuestionText)

	// Send to both users
	user1, _ := h.UserRepo.GetUserByID(session.User1ID)
	user2, _ := h.UserRepo.GetUserByID(session.User2ID)

	if user1 != nil {
		msgConfig := tgbotapi.NewMessage(user1.TelegramID, msg)
		msgConfig.ReplyMarkup = keyboard
		if apiInterface := bot.GetAPI(); apiInterface != nil {
			if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
				api.Send(msgConfig)
			}
		}
	}

	if user2 != nil {
		msgConfig := tgbotapi.NewMessage(user2.TelegramID, msg)
		msgConfig.ReplyMarkup = keyboard
		if apiInterface := bot.GetAPI(); apiInterface != nil {
			if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
				api.Send(msgConfig)
			}
		}
	}

	// Set timer for 10 seconds (Auto-skip if people don't answer)
	go func() {
		time.Sleep(10 * time.Second)
		quizSessionsMu.RLock()
		s, exists := quizSessions[matchID]
		quizSessionsMu.RUnlock()
		if exists {
			s.mu.Lock()
			// Only advance if we are still on the SAME question
			if s.CurrentQuestion == currentIdx {
				s.mu.Unlock()
				h.NextQuizQuestion(matchID, bot)
			} else {
				s.mu.Unlock()
			}
		}
	}()
}

// HandleQuizAnswer handles a user's answer to a quiz question
func (h *HandlerManager) HandleQuizAnswer(userID int64, messageID int, matchID uint, answerIndex int, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		bot.SendMessage(userID, "❌ بازی کوییز یافت نشد!", nil)
		return
	}

	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if user already answered this question
	if session.AnsweredUsers[user.ID] {
		return // Ignore multiple clicks
	}

	// Check if answer is within time limit (with small buffer)
	if time.Since(session.QuestionStartTime) > 11*time.Second {
		bot.SendMessage(userID, "⏰ زمان تموم شد!", nil)
		return
	}

	session.AnsweredUsers[user.ID] = true
	question := session.Questions[session.CurrentQuestion]

	// Parse options and check answer
	var options []string
	json.Unmarshal([]byte(question.Options), &options)

	if answerIndex >= 0 && answerIndex < len(options) {
		bot.EditMessage(userID, messageID, fmt.Sprintf("✅ شما پاسخ دادید: %s", options[answerIndex]), nil)
	}

	isCorrect := (answerIndex >= 0 && answerIndex < len(options) && options[answerIndex] == question.CorrectAnswer)

	if isCorrect {
		if user.ID == session.User1ID {
			session.User1Score += question.Points
		} else {
			session.User2Score += question.Points
		}
		bot.SendMessage(userID, "✅ جواب درست! +"+fmt.Sprint(question.Points)+" امتیاز", nil)
	} else {
		bot.SendMessage(userID, "❌ جواب اشتباه!", nil)
	}

	// If both users answered, move to next question IMMEDIATELY
	if len(session.AnsweredUsers) >= 2 {
		go h.NextQuizQuestion(matchID, bot)
	}
}

// NextQuizQuestion moves to the next question
func (h *HandlerManager) NextQuizQuestion(matchID uint, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		return
	}

	session.CurrentQuestion++

	if session.CurrentQuestion >= len(session.Questions) {
		h.EndQuiz(matchID, bot)
	} else {
		h.SendQuizQuestion(matchID, bot)
	}
}

// EndQuiz ends the quiz and announces the winner
func (h *HandlerManager) EndQuiz(matchID uint, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		return
	}

	user1, _ := h.UserRepo.GetUserByID(session.User1ID)
	user2, _ := h.UserRepo.GetUserByID(session.User2ID)

	var winnerMsg string
	rewardCoins := h.Config.WinRewardCoins

	if session.User1Score > session.User2Score {
		winnerMsg = fmt.Sprintf("🏆 برنده: %s\n\n📊 امتیاز:\n%s: %d\n%s: %d\n\n💰 پاداش برنده: %d سکه",
			user1.FullName, user1.FullName, session.User1Score, user2.FullName, session.User2Score, rewardCoins)
		h.CoinRepo.AddCoins(session.User1ID, rewardCoins, models.TxTypeGameReward, "پاداش برد در کوییز")
	} else if session.User2Score > session.User1Score {
		winnerMsg = fmt.Sprintf("🏆 برنده: %s\n\n📊 امتیاز:\n%s: %d\n%s: %d\n\n💰 پاداش برنده: %d سکه",
			user2.FullName, user1.FullName, session.User1Score, user2.FullName, session.User2Score, rewardCoins)
		h.CoinRepo.AddCoins(session.User2ID, rewardCoins, models.TxTypeGameReward, "پاداش برد در کوییز")
	} else {
		winnerMsg = fmt.Sprintf("🤝 مساوی!\n\n📊 امتیاز:\n%s: %d\n%s: %d\n\n💰 هر دو نفر %d سکه دریافت کردید!",
			user1.FullName, session.User1Score, user2.FullName, session.User2Score, rewardCoins/2)
		h.CoinRepo.AddCoins(session.User1ID, rewardCoins/2, models.TxTypeGameReward, "پاداش کوییز")
		h.CoinRepo.AddCoins(session.User2ID, rewardCoins/2, models.TxTypeGameReward, "پاداش کوییز")
	}

	// Award Village XP
	if session.User1Score > session.User2Score {
		h.VillageSvc.AddXPForUser(session.User1ID, 20)
		h.VillageSvc.AddXPForUser(session.User2ID, 5)
	} else if session.User2Score > session.User1Score {
		h.VillageSvc.AddXPForUser(session.User2ID, 20)
		h.VillageSvc.AddXPForUser(session.User1ID, 5)
	} else {
		h.VillageSvc.AddXPForUser(session.User1ID, 10)
		h.VillageSvc.AddXPForUser(session.User2ID, 10)
	}

	// Send results to both users
	if user1 != nil {
		bot.SendMessage(user1.TelegramID, "🎮 بازی کوییز تموم شد!\n\n"+winnerMsg, nil)
	}
	if user2 != nil {
		bot.SendMessage(user2.TelegramID, "🎮 بازی کوییز تموم شد!\n\n"+winnerMsg, nil)
	}

	// Clean up session
	quizSessionsMu.Lock()
	delete(quizSessions, matchID)
	quizSessionsMu.Unlock()

	logger.Info("Quiz ended", "match_id", matchID, "user1_score", session.User1Score, "user2_score", session.User2Score)
}
