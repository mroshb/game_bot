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

type RoomQuizSession struct {
	SessionID     uint
	RoomID        uint
	CurrentRound  int
	TotalRounds   int
	QuestionID    uint
	AnsweredUsers map[uint]bool
	mu            sync.Mutex
}

var (
	roomQuizSessions   = make(map[uint]*RoomQuizSession) // sessionID -> RoomQuizSession
	roomQuizSessionsMu sync.RWMutex
)

// StartQuizGame starts a Quiz of King game for a room
func (h *HandlerManager) StartQuizGame(userID int64, roomID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	isHost, _ := h.RoomRepo.IsHost(roomID, user.ID)
	if !isHost {
		bot.SendMessage(userID, "❌ فقط میزبان می‌تواند بازی را شروع کند!", nil)
		return
	}

	activeSession, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	if activeSession != nil {
		bot.SendMessage(userID, "⚠️ یک بازی در حال حاضر فعال است!", nil)
		return
	}

	members, _ := h.RoomRepo.GetRoomMembers(roomID)
	if len(members) < 2 {
		bot.SendMessage(userID, "👥 حداقل ۲ نفر برای شروع بازی لازم است!", nil)
		return
	}

	session, err := h.GameRepo.CreateGameSession(roomID, models.GameTypeQuiz)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در ایجاد جلسه بازی!", nil)
		return
	}

	for i, member := range members {
		h.GameRepo.AddParticipant(session.ID, member.ID, i+1)
	}

	// Initialize RoomQuizSession
	roomQuizSessionsMu.Lock()
	roomQuizSessions[session.ID] = &RoomQuizSession{
		SessionID:     session.ID,
		RoomID:        roomID,
		CurrentRound:  0,
		TotalRounds:   5, // Five rounds of excitement
		AnsweredUsers: make(map[uint]bool),
	}
	roomQuizSessionsMu.Unlock()

	h.GameRepo.StartGame(session.ID)
	h.SendNextQuizRound(session.ID, bot)
}

func (h *HandlerManager) SendNextQuizRound(sessionID uint, bot BotInterface) {
	roomQuizSessionsMu.RLock()
	qSession, exists := roomQuizSessions[sessionID]
	roomQuizSessionsMu.RUnlock()
	if !exists {
		return
	}

	qSession.mu.Lock()
	qSession.CurrentRound++
	qSession.AnsweredUsers = make(map[uint]bool) // Reset for new round
	qSession.mu.Unlock()

	question, err := h.GameRepo.GetRandomQuestion(models.QuestionTypeQuiz, "")
	if err != nil {
		logger.Error("Failed to get quiz question", "error", err)
		return
	}

	qSession.mu.Lock()
	qSession.QuestionID = question.ID
	qSession.mu.Unlock()

	h.GameRepo.UpdateCurrentQuestion(sessionID, question.ID)
	// Update status to in_progress if it was waiting or something
	h.GameRepo.UpdateGameStatus(sessionID, models.GameStatusInProgress)

	var options []string
	json.Unmarshal([]byte(question.Options), &options)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[0], fmt.Sprintf("qok_ans_%d_%d_0", sessionID, question.ID)),
			tgbotapi.NewInlineKeyboardButtonData(options[1], fmt.Sprintf("qok_ans_%d_%d_1", sessionID, question.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(options[2], fmt.Sprintf("qok_ans_%d_%d_2", sessionID, question.ID)),
			tgbotapi.NewInlineKeyboardButtonData(options[3], fmt.Sprintf("qok_ans_%d_%d_3", sessionID, question.ID)),
		),
	)

	msg := fmt.Sprintf("👑 کوییز اف کینگ - مرحله %d از %d\n\n📂 دسته‌بندی: %s\n\n❓ %s\n\n⏱ زمان باقی‌مانده: ۱۵ ثانیه",
		qSession.CurrentRound, qSession.TotalRounds, question.Category, question.QuestionText)

	members, _ := h.RoomRepo.GetRoomMembers(qSession.RoomID)
	for _, member := range members {
		msgConfig := tgbotapi.NewMessage(member.TelegramID, msg)
		msgConfig.ReplyMarkup = keyboard
		if api := bot.GetAPI(); api != nil {
			if b, ok := api.(*tgbotapi.BotAPI); ok {
				b.Send(msgConfig)
			}
		}
	}

	// Set a timer for the round
	go func() {
		time.Sleep(15 * time.Second)
		h.EndQuizRound(sessionID, bot)
	}()
}

func (h *HandlerManager) HandleQuizGameAnswer(userID int64, messageID int, sessionID uint, questionID uint, answerIdx int, bot BotInterface) {
	roomQuizSessionsMu.RLock()
	qSession, exists := roomQuizSessions[sessionID]
	roomQuizSessionsMu.RUnlock()
	if !exists {
		return
	}

	user, _ := h.UserRepo.GetUserByTelegramID(userID)

	qSession.mu.Lock()
	if qSession.AnsweredUsers[user.ID] || qSession.QuestionID != questionID {
		qSession.mu.Unlock()
		return
	}
	qSession.AnsweredUsers[user.ID] = true
	qSession.mu.Unlock()

	q, err := h.GameRepo.GetQuestionByID(questionID)
	if err != nil {
		return
	}

	var options []string
	json.Unmarshal([]byte(q.Options), &options)

	if answerIdx < 0 || answerIdx >= len(options) {
		return
	}

	// Remove keyboard after answer
	bot.EditMessage(userID, messageID, fmt.Sprintf("✅ شما پاسخ دادید: %s", options[answerIdx]), nil)

	if options[answerIdx] == q.CorrectAnswer {
		h.GameRepo.IncrementScore(sessionID, user.ID, 10)
		bot.SendMessage(userID, "✅ آفرین! پاسخ درست بود. (+۱۰ امتیاز)", nil)
	} else {
		bot.SendMessage(userID, fmt.Sprintf("❌ اشتباه بود! پاسخ درست: %s", q.CorrectAnswer), nil)
	}
}

func (h *HandlerManager) EndQuizRound(sessionID uint, bot BotInterface) {
	roomQuizSessionsMu.RLock()
	qSession, exists := roomQuizSessions[sessionID]
	roomQuizSessionsMu.RUnlock()
	if !exists {
		return
	}

	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil || session.Status == models.GameStatusFinished {
		return
	}

	if qSession.CurrentRound >= qSession.TotalRounds {
		h.EndQuizGame(sessionID, bot)
	} else {
		msg := "⌛️ زمان تمام شد! آماده برای مرحله بعد..."
		members, _ := h.RoomRepo.GetRoomMembers(qSession.RoomID)
		for _, member := range members {
			bot.SendMessage(member.TelegramID, msg, nil)
		}
		time.Sleep(3 * time.Second)
		h.SendNextQuizRound(sessionID, bot)
	}
}

func (h *HandlerManager) EndQuizGame(sessionID uint, bot BotInterface) {
	session, _ := h.GameRepo.GetGameSession(sessionID)
	participants, _ := h.GameRepo.GetParticipants(sessionID)

	msg := "🏁 بازی به پایان رسید!\n\n📊 رده‌بندی نهایی:\n"
	var winner *models.GameParticipant

	for _, p := range participants {
		msg += fmt.Sprintf("👤 %s: %d امتیاز\n", p.User.FullName, p.Score)
		if winner == nil || p.Score > winner.Score {
			winner = &p
		}
	}

	if winner != nil && winner.Score > 0 {
		reward := int64(winner.Score / 2)
		if reward < 10 {
			reward = 10
		}
		h.CoinRepo.AddCoins(winner.UserID, reward, models.TxTypeGameReward, "پاداش برد در کوئیز اف کینگ")
		msg += fmt.Sprintf("\n🏆 قهرمان: %s (%d سکه جایزه)", winner.User.FullName, reward)

		// Award Village XP to winner
		h.VillageSvc.AddXPForUser(winner.UserID, 20)
	}

	// Award participation XP to everyone in the village
	for _, p := range participants {
		h.VillageSvc.AddXPForUser(p.UserID, 5)
	}

	h.GameRepo.EndGame(sessionID)

	roomQuizSessionsMu.Lock()
	delete(roomQuizSessions, sessionID)
	roomQuizSessionsMu.Unlock()

	members, _ := h.RoomRepo.GetRoomMembers(session.RoomID)
	for _, member := range members {
		bot.SendMessage(member.TelegramID, msg, nil)
		h.ShowRoomMembers(member.TelegramID, session.RoomID, bot)
	}
}
