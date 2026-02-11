package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
)

// QuizGameSession holds in-memory state for active quiz games
type QuizGameSession struct {
	MatchID        uint
	RoundID        uint
	QuestionNumber int
	Questions      []models.Question

	User1AnsweredQ map[int]bool
	User2AnsweredQ map[int]bool

	User1QuestionStart time.Time
	User2QuestionStart time.Time

	User1UsedRemove2 map[int]bool
	User2UsedRemove2 map[int]bool
	User1UsedRetry   map[int]bool
	User2UsedRetry   map[int]bool

	User1LastQMsgID int
	User2LastQMsgID int

	CategoryTimer *time.Timer
	QuestionTimer *time.Timer

	mu sync.Mutex
}

var (
	quizGameSessions   = make(map[uint]*QuizGameSession)
	quizGameSessionsMu sync.RWMutex
)

func getQuizGameSession(matchID uint) *QuizGameSession {
	quizGameSessionsMu.RLock()
	session, exists := quizGameSessions[matchID]
	quizGameSessionsMu.RUnlock()

	if !exists {
		quizGameSessionsMu.Lock()
		session = &QuizGameSession{
			MatchID:          matchID,
			User1AnsweredQ:   make(map[int]bool),
			User2AnsweredQ:   make(map[int]bool),
			User1UsedRemove2: make(map[int]bool),
			User2UsedRemove2: make(map[int]bool),
			User1UsedRetry:   make(map[int]bool),
			User2UsedRetry:   make(map[int]bool),
		}
		quizGameSessions[matchID] = session
		quizGameSessionsMu.Unlock()
	}

	return session
}

func cleanupQuizGameSession(matchID uint) {
	quizGameSessionsMu.Lock()
	session, exists := quizGameSessions[matchID]
	if exists {
		if session.CategoryTimer != nil {
			session.CategoryTimer.Stop()
		}
		if session.QuestionTimer != nil {
			session.QuestionTimer.Stop()
		}
		delete(quizGameSessions, matchID)
	}
	quizGameSessionsMu.Unlock()
}

func (h *HandlerManager) ensureQuizSessionLoaded(session *QuizGameSession, match *models.QuizMatch) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if len(session.Questions) > 0 {
		return
	}

	round, _ := h.QuizMatchRepo.GetQuizRound(match.ID, match.CurrentRound)
	if round == nil || round.QuestionIDs == "" {
		return
	}

	idStrings := strings.Split(round.QuestionIDs, ",")
	var ids []uint
	for _, idStr := range idStrings {
		var id uint
		fmt.Sscanf(idStr, "%d", &id)
		if id > 0 {
			ids = append(ids, id)
		}
	}

	questions, err := h.GameRepo.GetQuestionsByIDs(ids)
	if err != nil {
		return
	}

	session.Questions = questions
	session.RoundID = round.ID

	// Sync answer state from DB
	ans1, _ := h.QuizMatchRepo.GetUserAnswers(match.ID, round.ID, match.User1ID)
	for _, a := range ans1 {
		session.User1AnsweredQ[a.QuestionNumber] = true
	}
	ans2, _ := h.QuizMatchRepo.GetUserAnswers(match.ID, round.ID, match.User2ID)
	for _, a := range ans2 {
		session.User2AnsweredQ[a.QuestionNumber] = true
	}
}

// ========================================
// GLASS MENU - Show Active Games
// ========================================

func (h *HandlerManager) ShowActiveQuizGames(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	activeMatches, err := h.QuizMatchRepo.GetAllActiveQuizMatchesByUser(user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت بازیها!", nil)
		return
	}

	finishedMatches, _ := h.QuizMatchRepo.GetFinishedQuizMatchesByUser(user.ID, 5)

	if len(activeMatches) == 0 && len(finishedMatches) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ 🎮 بازی جدید", "btn:new_quiz_game"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:main_menu"),
			),
		)
		bot.SendMessage(userID, "📋 شما هیچ بازی فعالی ندارید!\n\nبرای شروع بازی جدید روی دکمه زیر کلیک کنید:", keyboard)
		return
	}

	msg := "📋 بازیهای کوئیز شما:\n\n"

	if len(activeMatches) > 0 {
		msg += "🔥 بازیهای فعال:\n"
		for _, match := range activeMatches {
			opponentName := match.User2.FullName
			if user.ID == match.User2ID {
				opponentName = match.User1.FullName
			}

			isMyTurn := match.TurnUserID != nil && *match.TurnUserID == user.ID
			turnIcon := "⏳"
			if isMyTurn {
				turnIcon = "⚔️"
			}

			myScore := match.User1TotalCorrect
			oppScore := match.User2TotalCorrect
			if user.ID == match.User2ID {
				myScore = match.User2TotalCorrect
				oppScore = match.User1TotalCorrect
			}

			status := "نوبت حریفه"
			if isMyTurn {
				status = "نوبت توست"
			}

			msg += fmt.Sprintf("%s با %s — %s\n", turnIcon, opponentName, status)
			msg += fmt.Sprintf("   راند %d از %d | امتیاز: %d-%d\n\n", match.CurrentRound, models.QuizTotalRounds, myScore, oppScore)
		}
	}

	if len(finishedMatches) > 0 {
		msg += "\n✅ بازیهای اخیر:\n"
		for _, match := range finishedMatches {
			opponentName := match.User2.FullName
			if user.ID == match.User2ID {
				opponentName = match.User1.FullName
			}

			result := "🤝 مساوی"
			if match.WinnerID != nil {
				if *match.WinnerID == user.ID {
					result = "🏆 برد"
				} else {
					result = "❌ باخت"
				}
			}

			myScore := match.User1TotalCorrect
			oppScore := match.User2TotalCorrect
			if user.ID == match.User2ID {
				myScore = match.User2TotalCorrect
				oppScore = match.User1TotalCorrect
			}

			msg += fmt.Sprintf("%s با %s | امتیاز: %d-%d\n", result, opponentName, myScore, oppScore)
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	for _, match := range activeMatches {
		opponentName := match.User2.FullName
		if user.ID == match.User2ID {
			opponentName = match.User1.FullName
		}

		isMyTurn := match.TurnUserID != nil && *match.TurnUserID == user.ID
		buttonText := fmt.Sprintf("⏳ %s", opponentName)
		if isMyTurn {
			buttonText = fmt.Sprintf("⚔️ %s (نوبت تو)", opponentName)
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("btn:qgame_%d", match.ID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ 🎮 بازی جدید", "btn:new_quiz_game"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:main_menu"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.SendMessage(userID, msg, keyboard)
}

// ========================================
// GAME DETAIL SCREEN
// ========================================

func (h *HandlerManager) ShowQuizGameDetail(userID int64, matchID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		bot.SendMessage(userID, "❌ بازی پیدا نشد!", nil)
		return
	}

	if match.User1ID != user.ID && match.User2ID != user.ID {
		bot.SendMessage(userID, "❌ شما در این بازی نیستید!", nil)
		return
	}

	opponentName := match.User2.FullName
	myScore := match.User1TotalCorrect
	oppScore := match.User2TotalCorrect
	myTime := float64(match.User1TotalTimeMs) / 1000.0
	oppTime := float64(match.User2TotalTimeMs) / 1000.0

	if user.ID == match.User2ID {
		opponentName = match.User1.FullName
		myScore = match.User2TotalCorrect
		oppScore = match.User1TotalCorrect
		myTime = float64(match.User2TotalTimeMs) / 1000.0
		oppTime = float64(match.User1TotalTimeMs) / 1000.0
	}

	isMyTurn := match.TurnUserID != nil && *match.TurnUserID == user.ID

	msg := fmt.Sprintf("⚔️ بازی با %s\n\n", opponentName)
	msg += "📊 وضعیت بازی:\n"
	msg += fmt.Sprintf("👤 شما: %d درست | ⏱ %.1fث\n", myScore, myTime)
	msg += fmt.Sprintf("👤 %s: %d درست | ⏱ %.1fث\n\n", opponentName, oppScore, oppTime)
	msg += fmt.Sprintf("📍 راند فعلی: %d از %d\n\n", match.CurrentRound, models.QuizTotalRounds)

	rounds, _ := h.QuizMatchRepo.GetAllQuizRounds(matchID)

	msg += "┌─────────────────────────────┐\n"
	for i := 1; i <= models.QuizTotalRounds; i++ {
		var round *models.QuizRound
		for _, r := range rounds {
			if r.RoundNumber == i {
				round = &r
				break
			}
		}

		if round != nil {
			msg += fmt.Sprintf("│ راند %d - %s\n", i, round.Category)

			user1Answers, _ := h.QuizMatchRepo.GetUserAnswers(matchID, round.ID, match.User1ID)
			user2Answers, _ := h.QuizMatchRepo.GetUserAnswers(matchID, round.ID, match.User2ID)

			myAnswers := user1Answers
			oppAnswers := user2Answers
			if user.ID == match.User2ID {
				myAnswers = user2Answers
				oppAnswers = user1Answers
			}

			msg += "│ شما:  "
			for j := 1; j <= models.QuizQuestionsPerRound; j++ {
				found := false
				for _, ans := range myAnswers {
					if ans.QuestionNumber == j {
						if ans.IsCorrect {
							msg += "🟢 "
						} else {
							msg += "🔴 "
						}
						found = true
						break
					}
				}
				if !found {
					msg += "⚪️ "
				}
			}
			msg += "\n"

			msg += fmt.Sprintf("│ %s:  ", opponentName)
			for j := 1; j <= models.QuizQuestionsPerRound; j++ {
				found := false
				for _, ans := range oppAnswers {
					if ans.QuestionNumber == j {
						if ans.IsCorrect {
							msg += "🟢 "
						} else {
							msg += "🔴 "
						}
						found = true
						break
					}
				}
				if !found {
					msg += "⚪️ "
				}
			}
			msg += "\n"
		} else {
			msg += fmt.Sprintf("│ راند %d - ؟\n", i)
			msg += "│ شما:  ⚪️ ⚪️ ⚪️ ⚪️\n"
			msg += fmt.Sprintf("│ %s:  ⚪️ ⚪️ ⚪️ ⚪️\n", opponentName)
		}
		msg += "├─────────────────────────────┤\n"
	}
	msg = strings.TrimSuffix(msg, "├─────────────────────────────┤\n")
	msg += "└─────────────────────────────┘\n"

	var keyboard tgbotapi.InlineKeyboardMarkup

	// Check how many questions this user has answered in the current round
	currentRound, _ := h.QuizMatchRepo.GetQuizRound(matchID, match.CurrentRound)
	questionsAnswered := 0
	if currentRound != nil {
		ans, _ := h.QuizMatchRepo.GetUserAnswers(matchID, currentRound.ID, user.ID)
		questionsAnswered = len(ans)
	}

	if isMyTurn && match.State == models.QuizStateWaitingCategory {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏁 انتخاب موضوع", fmt.Sprintf("btn:qstart_%d", matchID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
			),
		)
	} else if currentRound != nil && questionsAnswered >= models.QuizQuestionsPerRound {
		// User finished this round, but round hasn't advanced (waiting for opponent)
		msg += "\n⏳ منتظر حریف باشید تا راند تمام شود..."
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 بروزرسانی", fmt.Sprintf("btn:qgame_%d", matchID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
			),
		)
	} else if currentRound != nil && questionsAnswered < models.QuizQuestionsPerRound {
		btnText := "🏁 شروع بازی"
		if questionsAnswered > 0 {
			btnText = " ادامه بازی 🔄"
		}
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("btn:qplay_%d", matchID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
			),
		)
	} else if !isMyTurn && match.State == models.QuizStateWaitingCategory {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔔 یادآوری", fmt.Sprintf("btn:qnotify_%d", matchID)),
				tgbotapi.NewInlineKeyboardButtonData("⚖️ برد فنی", fmt.Sprintf("btn:qforce_win_%d", matchID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
			),
		)
	} else {
		// Even if not category state, if it's not my turn, I should be able to force win if timeout
		if !isMyTurn {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⚖️ برد فنی", fmt.Sprintf("btn:qforce_win_%d", matchID)),
					tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
				),
			)
		} else {
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "btn:quiz_games"),
				),
			)
		}
	}

	bot.SendMessage(userID, msg, keyboard)
}

// Notify opponent
func (h *HandlerManager) NotifyQuizOpponent(userID int64, matchID uint, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if user == nil {
		return
	}

	match, err := h.QuizMatchRepo.GetQuizMatch(matchID)
	if err != nil {
		return
	}

	opponentID := match.User2ID
	if user.ID == match.User2ID {
		opponentID = match.User1ID
	}

	opponent, _ := h.UserRepo.GetUserByID(opponentID)
	if opponent == nil {
		return
	}

	msg := fmt.Sprintf("🔔 یادآوری: %s منتظر شماست!\n\nنوبت شماست که بازی را ادامه دهید.", user.FullName)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎮 ادامه بازی", fmt.Sprintf("btn:qgame_%d", matchID)),
		),
	)
	bot.SendMessage(opponent.TelegramID, msg, keyboard)
	bot.SendMessage(userID, "✅ یادآوری ارسال شد!", nil)
}

// ========================================
// START NEW QUIZ GAME
// ========================================

func (h *HandlerManager) StartNewQuizGame(userID int64, bot BotInterface) {
	// Delegate to matchmaking function
	h.StartQuizMatchmaking(userID, bot)
}
