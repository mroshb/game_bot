package handlers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
)

type QuizSession struct {
	MatchID   uint
	User1ID   uint
	User2ID   uint
	User1TgID int64
	User2TgID int64

	CurrentRound    int // 1, 2, 3
	CurrentQuestion int // 1, 2, 3 (within round)

	RoundQuestions []models.Question
	RoundTopic     string

	// User progress in current round
	User1RoundAnswers []bool
	User2RoundAnswers []bool

	User1TotalCorrect int
	User2TotalCorrect int

	User1TurnStart time.Time
	User2TurnStart time.Time
	User1TotalTime time.Duration
	User2TotalTime time.Duration

	User1LightsMsgID int
	User2LightsMsgID int

	State          string // "choosing_topic", "in_round", "finished"
	ChoosingUserID uint

	TopicTimer    *time.Timer `json:"-"`
	QuestionTimer *time.Timer `json:"-"`

	mu sync.Mutex
}

var (
	quizSessions   = make(map[uint]*QuizSession)
	quizSessionsMu sync.RWMutex
)

// StartQuiz starts a quiz game revamp
func (h *HandlerManager) StartQuiz(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil || match == nil {
		bot.SendMessage(userID, "⚠️ شما در چت فعالی نیستید!", nil)
		return
	}

	quizSessionsMu.RLock()
	_, exists := quizSessions[match.ID]
	quizSessionsMu.RUnlock()
	if exists {
		bot.SendMessage(userID, "⚠️ بازی کوییز در حال انجام است!", nil)
		return
	}

	// Initialize session
	otherUserID := match.User1ID
	if user.ID == match.User1ID {
		otherUserID = match.User2ID
	}
	otherUser, _ := h.UserRepo.GetUserByID(otherUserID)

	session := &QuizSession{
		MatchID:        match.ID,
		User1ID:        match.User1ID,
		User2ID:        match.User2ID,
		User1TgID:      match.User1.TelegramID,
		User2TgID:      match.User2.TelegramID,
		CurrentRound:   0,
		State:          "choosing_topic",
		ChoosingUserID: user.ID, // User who started picks first topic
	}

	// Ensure we have Tg IDs (Match model might not have them directly, fetch if needed)
	if session.User1TgID == 0 || session.User2TgID == 0 {
		u1, _ := h.UserRepo.GetUserByID(session.User1ID)
		u2, _ := h.UserRepo.GetUserByID(session.User2ID)
		if u1 != nil {
			session.User1TgID = u1.TelegramID
		}
		if u2 != nil {
			session.User2TgID = u2.TelegramID
		}
	}

	quizSessionsMu.Lock()
	quizSessions[match.ID] = session
	quizSessionsMu.Unlock()

	msg := "🧠 بازی کوییز (Quiz) شروع شد!\n\n📊 شرایط بازی:\n▫️ ۳ راند ۳ سواله\n▫️ هر راند یک موضوع انتخابی\n▫️ برنده بر اساس جواب درست و سرعت بیشتر مشخص میشه!\n\nآماده باش!"
	bot.SendMessage(user.TelegramID, msg, nil)
	if otherUser != nil {
		bot.SendMessage(otherUser.TelegramID, msg, nil)
	}

	time.Sleep(2 * time.Second)
	h.AskForTopic(session, bot)
}

func (h *HandlerManager) AskForTopic(session *QuizSession, bot BotInterface) {
	session.mu.Lock()
	session.State = "choosing_topic"
	session.CurrentRound++
	chooserID := session.ChoosingUserID
	session.mu.Unlock()

	categories, err := h.GameRepo.GetQuizCategories(3)
	if err != nil || len(categories) == 0 {
		categories = []string{"اطلاعات عمومی", "تاریخ", "ورزش"}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, cat := range categories {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cat, fmt.Sprintf("qcat_%d_%s", session.MatchID, cat)),
		))
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	otherTgID := session.User2TgID
	chooserTgID := session.User1TgID
	if chooserID == session.User2ID {
		otherTgID = session.User1TgID
		chooserTgID = session.User2TgID
	}

	bot.SendMessage(chooserTgID, fmt.Sprintf("🎭 راند %d: نوبت شماست که موضوع رو انتخاب کنی! (۳۰ ثانیه زمان داری)", session.CurrentRound), keyboard)
	bot.SendMessage(otherTgID, fmt.Sprintf("⌛️ راند %d: حریف در حال انتخاب موضوع بازی است...", session.CurrentRound), nil)

	// Set timeout for topic selection
	session.mu.Lock()
	if session.TopicTimer != nil {
		session.TopicTimer.Stop()
	}
	session.TopicTimer = time.AfterFunc(30*time.Second, func() {
		h.HandleQuizTopicTimeout(session, bot)
	})
	session.mu.Unlock()
}

func (h *HandlerManager) HandleQuizTopicTimeout(session *QuizSession, bot BotInterface) {
	session.mu.Lock()
	if session.State != "choosing_topic" {
		session.mu.Unlock()
		return
	}
	session.mu.Unlock()

	msg := "⏰ زمان انتخاب موضوع تمام شد و بازی پایان یافت!"
	bot.SendMessage(session.User1TgID, msg, nil)
	bot.SendMessage(session.User2TgID, msg, nil)

	h.CleanupQuizSession(session.MatchID)
}

func (h *HandlerManager) HandleQuizCategorySelection(userID int64, matchID uint, category string, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		return
	}

	session.mu.Lock()
	if session.State != "choosing_topic" || (userID != session.User1TgID && userID != session.User2TgID) {
		session.mu.Unlock()
		return
	}

	// Fetch 3 questions
	questions, err := h.GameRepo.GetQuestionsByCategory(category, 3)
	if err != nil || len(questions) < 3 {
		// Fallback to random questions
		questions, _ = h.GameRepo.GetQuizQuestions(3)
	}

	session.RoundQuestions = questions
	session.RoundTopic = category
	session.CurrentQuestion = 1
	session.User1RoundAnswers = []bool{}
	session.User2RoundAnswers = []bool{}
	session.State = "in_round"
	session.User1TurnStart = time.Now()
	session.User2TurnStart = time.Now()
	if session.TopicTimer != nil {
		session.TopicTimer.Stop()
	}
	session.mu.Unlock()

	msg := fmt.Sprintf("✅ موضوع انتخاب شد: *%s*\n\nسوالات شروع شدند!", category)
	bot.SendMessage(session.User1TgID, msg, nil)
	bot.SendMessage(session.User2TgID, msg, nil)

	time.Sleep(1500 * time.Millisecond)

	// Send progress messages (lights)
	session.User1LightsMsgID = bot.SendMessage(session.User1TgID, "⚪️ ⚪️ ⚪️", nil)
	session.User2LightsMsgID = bot.SendMessage(session.User2TgID, "⚪️ ⚪️ ⚪️", nil)

	h.SendCurrentQuizQuestion(session, bot)
}

func (h *HandlerManager) SendCurrentQuizQuestion(session *QuizSession, bot BotInterface) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.CurrentQuestion > 3 {
		return
	}

	q := session.RoundQuestions[session.CurrentQuestion-1]
	var options []string
	json.Unmarshal([]byte(q.Options), &options)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("qans_%d_%d_0", session.MatchID, session.CurrentQuestion)),
			tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("qans_%d_%d_1", session.MatchID, session.CurrentQuestion)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("qans_%d_%d_2", session.MatchID, session.CurrentQuestion)),
			tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("qans_%d_%d_3", session.MatchID, session.CurrentQuestion)),
		),
	)

	msgText := fmt.Sprintf("❓ سوال %d از ۳:\n\n*%s*", session.CurrentQuestion, q.QuestionText)

	// Track turn start for timing
	now := time.Now()
	if len(session.User1RoundAnswers) < session.CurrentQuestion {
		session.User1TurnStart = now
	}
	if len(session.User2RoundAnswers) < session.CurrentQuestion {
		session.User2TurnStart = now
	}

	bot.SendMessage(session.User1TgID, msgText, keyboard)
	bot.SendMessage(session.User2TgID, msgText, keyboard)

	// Set question timeout
	if session.QuestionTimer != nil {
		session.QuestionTimer.Stop()
	}
	session.QuestionTimer = time.AfterFunc(25*time.Second, func() {
		h.HandleQuizQuestionTimeout(session, bot)
	})
}

func (h *HandlerManager) HandleQuizQuestionTimeout(session *QuizSession, bot BotInterface) {
	session.mu.Lock()
	if session.State != "in_round" {
		session.mu.Unlock()
		return
	}

	currQ := session.CurrentQuestion
	u1Answered := len(session.User1RoundAnswers) >= currQ
	u2Answered := len(session.User2RoundAnswers) >= currQ

	if u1Answered && u2Answered {
		session.mu.Unlock()
		return
	}

	// Auto-fill wrong answers for those who didn't answer
	if !u1Answered {
		session.User1RoundAnswers = append(session.User1RoundAnswers, false)
		bot.SendMessage(session.User1TgID, "⏰ زمان پاسخگویی به این سوال تموم شد!", nil)
	}
	if !u2Answered {
		session.User2RoundAnswers = append(session.User2RoundAnswers, false)
		bot.SendMessage(session.User2TgID, "⏰ زمان پاسخگویی به این سوال تموم شد!", nil)
	}

	session.CurrentQuestion++
	shouldFinish := session.CurrentQuestion > 3
	session.mu.Unlock()

	if shouldFinish {
		h.FinishRound(session, bot)
	} else {
		h.SendCurrentQuizQuestion(session, bot)
	}
}

func (h *HandlerManager) HandleQuizAnswer(tgUserID int64, matchID uint, qIdx int, answerIdx int, bot BotInterface) {
	quizSessionsMu.RLock()
	session, exists := quizSessions[matchID]
	quizSessionsMu.RUnlock()
	if !exists {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.State != "in_round" {
		return
	}

	isUser1 := tgUserID == session.User1TgID
	answers := &session.User1RoundAnswers
	totalCorrect := &session.User1TotalCorrect
	totalTime := &session.User1TotalTime
	turnStart := session.User1TurnStart
	lightsMsgID := session.User1LightsMsgID
	if !isUser1 {
		answers = &session.User2RoundAnswers
		totalCorrect = &session.User2TotalCorrect
		totalTime = &session.User2TotalTime
		turnStart = session.User2TurnStart
		lightsMsgID = session.User2LightsMsgID
	}

	// Check if already answered this question or if it's an old/future question button
	if qIdx != session.CurrentQuestion || len(*answers) >= session.CurrentQuestion {
		return
	}

	// Calculate time taken
	duration := time.Since(turnStart)
	*totalTime += duration

	// Check correctness
	q := session.RoundQuestions[session.CurrentQuestion-1]
	var options []string
	json.Unmarshal([]byte(q.Options), &options)

	isCorrect := false
	if answerIdx >= 0 && answerIdx < len(options) && options[answerIdx] == q.CorrectAnswer {
		isCorrect = true
		*totalCorrect++
	}
	*answers = append(*answers, isCorrect)

	// Update lights
	lights := ""
	for i := 0; i < 3; i++ {
		if i < len(*answers) {
			if (*answers)[i] {
				lights += "🟢 "
			} else {
				lights += "🔴 "
			}
		} else {
			lights += "⚪️ "
		}
	}
	bot.EditMessage(tgUserID, lightsMsgID, lights, nil)

	// Check if both users finished current question
	if len(session.User1RoundAnswers) == session.CurrentQuestion && len(session.User2RoundAnswers) == session.CurrentQuestion {
		if session.QuestionTimer != nil {
			session.QuestionTimer.Stop()
		}
		session.CurrentQuestion++
		if session.CurrentQuestion > 3 {
			go h.FinishRound(session, bot)
		} else {
			go h.SendCurrentQuizQuestion(session, bot)
		}
	}
}

func (h *HandlerManager) FinishRound(session *QuizSession, bot BotInterface) {
	session.mu.Lock()

	u1Correct := 0
	for _, a := range session.User1RoundAnswers {
		if a {
			u1Correct++
		}
	}
	u2Correct := 0
	for _, a := range session.User2RoundAnswers {
		if a {
			u2Correct++
		}
	}

	msg := fmt.Sprintf("🏁 پایان راند %d (%s):\n\n👥 نتایج این راند:\n👤 شما: %d درست\n👤 حریف: %d درست",
		session.CurrentRound, session.RoundTopic, u1Correct, u2Correct)
	bot.SendMessage(session.User1TgID, msg, nil)

	msg2 := fmt.Sprintf("🏁 پایان راند %d (%s):\n\n👥 نتایج این راند:\n👤 شما: %d درست\n👤 حریف: %d درست",
		session.CurrentRound, session.RoundTopic, u2Correct, u1Correct)
	bot.SendMessage(session.User2TgID, msg2, nil)

	if session.CurrentRound >= 3 {
		session.mu.Unlock()
		h.EndQuizRevamp(session, bot)
		return
	}

	// Switch chooser
	if session.ChoosingUserID == session.User1ID {
		session.ChoosingUserID = session.User2ID
	} else {
		session.ChoosingUserID = session.User1ID
	}
	session.mu.Unlock()

	time.Sleep(3 * time.Second)
	h.AskForTopic(session, bot)
}

func (h *HandlerManager) EndQuizRevamp(session *QuizSession, bot BotInterface) {
	session.mu.Lock()
	defer session.mu.Unlock()

	u1Score := session.User1TotalCorrect
	u2Score := session.User2TotalCorrect
	u1Time := session.User1TotalTime
	u2Time := session.User2TotalTime

	winnerID := uint(0)
	var resultMsg1, resultMsg2 string

	if u1Score > u2Score {
		winnerID = session.User1ID
	} else if u2Score > u1Score {
		winnerID = session.User2ID
	} else {
		// Tie in correct answers, check speed
		if u1Time < u2Time {
			winnerID = session.User1ID
		} else if u2Time < u1Time {
			winnerID = session.User2ID
		}
	}

	summary := fmt.Sprintf("\n\n📊 آمار نهایی:\n👤 شما: %d درست (%s ثانیه)\n👤 حریف: %d درست (%s ثانیه)",
		u1Score, fmt.Sprintf("%.1f", u1Time.Seconds()), u2Score, fmt.Sprintf("%.1f", u2Time.Seconds()))

	summary2 := fmt.Sprintf("\n\n📊 آمار نهایی:\n👤 شما: %d درست (%s ثانیه)\n👤 حریف: %d درست (%s ثانیه)",
		u2Score, fmt.Sprintf("%.1f", u2Time.Seconds()), u1Score, fmt.Sprintf("%.1f", u1Time.Seconds()))

	rewardCoins := h.Config.WinRewardCoins
	switch winnerID {
	case session.User1ID:
		resultMsg1 = "🏆 تبریک! شما برنده شدی!" + summary
		resultMsg2 = "❌ متأسفانه باختی!" + summary2
		h.CoinRepo.AddCoins(session.User1ID, rewardCoins, models.TxTypeGameReward, "برد در کوییز")
		h.VillageSvc.AddXPForUser(session.User1ID, 30)
		h.VillageSvc.AddXPForUser(session.User2ID, 10)
	case session.User2ID:
		resultMsg1 = "❌ متأسفانه باختی!" + summary
		resultMsg2 = "🏆 تبریک! شما برنده شدی!" + summary2
		h.CoinRepo.AddCoins(session.User2ID, rewardCoins, models.TxTypeGameReward, "برد در کوییز")
		h.VillageSvc.AddXPForUser(session.User2ID, 30)
		h.VillageSvc.AddXPForUser(session.User1ID, 10)
	default:
		resultMsg1 = "🤝 بازی مساوی شد!" + summary
		resultMsg2 = "🤝 بازی مساوی شد!" + summary2
		h.CoinRepo.AddCoins(session.User1ID, rewardCoins/2, models.TxTypeGameReward, "مساوی در کوییز")
		h.CoinRepo.AddCoins(session.User2ID, rewardCoins/2, models.TxTypeGameReward, "مساوی در کوییز")
		h.VillageSvc.AddXPForUser(session.User1ID, 20)
		h.VillageSvc.AddXPForUser(session.User2ID, 20)
	}

	bot.SendMessage(session.User1TgID, "🎮 بازی تمام شد!\n"+resultMsg1, nil)
	bot.SendMessage(session.User2TgID, "🎮 بازی تمام شد!\n"+resultMsg2, nil)

	quizSessionsMu.Lock()
	delete(quizSessions, session.MatchID)
	quizSessionsMu.Unlock()
}

// CleanupQuizSession removes a quiz session if it exists (e.g. when match ends)
func (h *HandlerManager) CleanupQuizSession(matchID uint) {
	quizSessionsMu.Lock()
	delete(quizSessions, matchID)
	quizSessionsMu.Unlock()
}
