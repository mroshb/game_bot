package handlers

import (
	"fmt"
	"math/rand"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// ========================================
// MATCHMAKING & FILTERING
// ========================================

// StartTodMatchmaking shows the filter menu for Anonymous mode
func (h *HandlerManager) StartTodMatchmaking(userID int64, bot BotInterface) {
	bot.SendMessage(userID, "🔥 بخش بازی با ناشناس:\n\nلطفاً فیلتر مورد نظرت رو انتخاب کن:", bot.GetTodAnonymousFilterKeyboard())
}

// HandleTodFiltering handles gender selection and starts matchmaking
func (h *HandlerManager) HandleTodFiltering(telegramID int64, filter string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(telegramID)
	if err != nil {
		bot.SendMessage(telegramID, MsgErrorGeneric, nil)
		return
	}

	requestedGender := models.RequestedGenderAny
	cost := 0

	switch filter {
	case "girl":
		requestedGender = models.GenderFemale
		cost = 10
	case "boy":
		requestedGender = models.GenderMale
		cost = 10
	case "random":
		requestedGender = models.RequestedGenderAny
		cost = 0
	case "advanced":
		bot.SendMessage(telegramID, "⚙️ بخش پیشرفته به زودی فعال می‌شود.", nil)
		return
	}

	if user.CoinBalance < int64(cost) {
		bot.SendMessage(telegramID, fmt.Sprintf(MsgInsufficientCoins, user.CoinBalance), nil)
		return
	}

	if cost > 0 {
		if err := h.CoinRepo.DeductCoins(user.ID, int64(cost), "tod_filter", "Truth or Dare gender filter"); err != nil {
			bot.SendMessage(telegramID, MsgErrorGeneric, nil)
			return
		}
	}

	h.StartTodMatchmakingWithFilter(user, requestedGender, bot)
}

// StartTodMatchmakingWithFilter initiates matchmaking with specific filters
func (h *HandlerManager) StartTodMatchmakingWithFilter(user *models.User, gender string, bot BotInterface) {
	inQueue, _ := h.MatchRepo.IsUserInQueue(user.ID)
	if inQueue {
		bot.SendMessage(user.TelegramID, MsgAlreadySearching, nil)
		return
	}

	activeGame, _ := h.TodRepo.GetActiveGameForUser(user.ID)
	if activeGame != nil {
		bot.SendMessage(user.TelegramID, MsgAlreadyInMatch, nil)
		return
	}

	queue := &models.MatchmakingQueue{
		UserID:          user.ID,
		GameType:        models.GameTypeTruthDare,
		RequestedGender: gender,
	}

	if err := h.MatchRepo.AddToQueue(queue); err != nil {
		bot.SendMessage(user.TelegramID, MsgErrorGeneric, nil)
		return
	}

	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusSearching)

	cancelKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ لغو جستجو", "btn:tod_cancel_search"),
		),
	)
	bot.SendMessage(user.TelegramID, "🔍 در حال جستجوی حریف مناسب...\nصبور باشید.", cancelKeyboard)

	go h.tryTodMatchmaking(user.ID, bot)
}

func (h *HandlerManager) tryTodMatchmaking(userID uint, bot BotInterface) {
	time.Sleep(2 * time.Second)

	_, err := h.MatchRepo.GetQueueEntry(userID)
	if err != nil {
		return
	}

	filters := &models.MatchFilters{
		Gender:   models.RequestedGenderAny,
		GameType: models.GameTypeTruthDare,
	}
	opponent, err := h.MatchRepo.FindAndRemoveMatch(userID, filters)
	if err != nil || opponent == nil {
		return
	}

	matchSession, err := h.MatchRepo.CreateMatchSession(userID, opponent.ID, 24*time.Hour)
	if err != nil {
		logger.Error("Failed to create match session", "error", err)
		return
	}

	game, err := h.TodRepo.CreateGame(matchSession.ID, userID, opponent.ID)
	if err != nil {
		h.MatchRepo.EndMatch(matchSession.ID)
		h.UserRepo.UpdateUserStatus(userID, models.UserStatusOnline)
		h.UserRepo.UpdateUserStatus(opponent.ID, models.UserStatusOnline)
		return
	}

	h.UserRepo.UpdateUserStatus(userID, models.UserStatusInMatch)
	h.UserRepo.UpdateUserStatus(opponent.ID, models.UserStatusInMatch)

	user, _ := h.UserRepo.GetUserByID(userID)
	if user != nil {
		bot.SendMessage(user.TelegramID, fmt.Sprintf("🎉 حریف پیدا شد!\n\n🔥 بازی با %s شروع شد!", opponent.FullName), nil)
	}
	bot.SendMessage(opponent.TelegramID, fmt.Sprintf("🎉 حریف پیدا شد!\n\n🔥 بازی با %s شروع شد!", user.FullName), nil)

	time.Sleep(2 * time.Second)
	h.HandleTodCoinFlip(game.ID, bot)
}

func (h *HandlerManager) CancelTodMatchmaking(telegramID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(telegramID)
	if err != nil {
		return
	}
	h.MatchRepo.RemoveFromQueue(user.ID)
	h.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
	bot.SendMessage(telegramID, "❌ جستجو لغو شد.", nil)
}

// ========================================
// GAME FLOW
// ========================================

func (h *HandlerManager) HandleTodCoinFlip(gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	msg := "🎲 در حال تعیین نقش‌ها (سوال‌کننده و پاسخ‌دهنده)..."
	bot.SendMessage(game.Match.User1.TelegramID, msg, nil)
	bot.SendMessage(game.Match.User2.TelegramID, msg, nil)

	time.Sleep(2 * time.Second)

	firstPlayer := game.ActivePlayerID
	secondPlayer := game.PassivePlayerID

	if rand.Intn(2) == 1 {
		firstPlayer, secondPlayer = secondPlayer, firstPlayer
		h.TodRepo.SetGamePlayers(gameID, firstPlayer, secondPlayer)
	}

	// Determine who is starting
	responder := getUserByID(firstPlayer, game.Match)
	questioner := getUserByID(secondPlayer, game.Match)

	bot.SendMessage(responder.TelegramID, fmt.Sprintf("🎲 قرعه‌کشی انجام شد!\n\n👤 نقش شما: **پاسخ‌دهنده** (باید جرات یا حقیقت را انتخاب کنید)\n👤 رقیب: %s (سوال‌کننده)", questioner.FullName), nil)
	bot.SendMessage(questioner.TelegramID, fmt.Sprintf("🎲 قرعه‌کشی انجام شد!\n\n👤 نقش شما: **سوال‌کننده** (منتظر انتخاب حریف بمانید)\n👤 رقیب: %s (پاسخ‌دهنده)", responder.FullName), nil)

	time.Sleep(2 * time.Second)

	// Trigger start
	h.HandleTodStart(int64(game.Match.User1.TelegramID), gameID, bot)
}

func (h *HandlerManager) HandleTodStart(userID int64, gameID uint, bot BotInterface) {
	game, err := h.TodRepo.GetGameByID(gameID)
	if err != nil {
		return
	}

	h.TodRepo.CreateTurn(gameID, game.ActivePlayerID, game.PassivePlayerID, game.CurrentRound)
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingChoice)

	h.ShowTodChoiceScreen(gameID, bot)
}

func (h *HandlerManager) ShowTodChoiceScreen(gameID uint, bot BotInterface) {
	game, _ := h.TodRepo.GetGameByID(gameID)
	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	// Active (Responder) chooses challenge
	activeMsg := fmt.Sprintf("🎮 راند %d\n🔔 **نوبت شماست!**\n\nنوع چالش رو انتخاب کن:", game.CurrentRound)
	bot.SendMessage(activeUser.TelegramID, activeMsg, bot.GetTodChallengeTypeKeyboard(gameID))

	// Passive (Questioner) waits
	passiveMsg := fmt.Sprintf("🎮 راند %d\n⏳ منتظر انتخاب حریف (%s)...", game.CurrentRound, activeUser.FullName)
	bot.SendMessage(passiveUser.TelegramID, passiveMsg, nil)
}

func (h *HandlerManager) HandleTodChoice(telegramID int64, gameID uint, choice string, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(telegramID)
	game, _ := h.TodRepo.GetGameByID(gameID)

	if game.ActivePlayerID != user.ID {
		return
	}

	challengeType := models.TodTypeTruth
	if choice == "dare" || choice == "dare18" {
		challengeType = models.TodTypeDare
	}
	difficulty := "easy"
	if choice == "truth18" || choice == "dare18" {
		difficulty = "hard"
	}

	// Use relation level "all" by default, or "stranger" if we want to be strict.
	// Given our seeding uses "all" for many, and "friend" for some, "all" is a better fallback.
	challenge, err := h.TodRepo.GetRandomChallenge(challengeType, difficulty, "", user.Gender, "all")
	if err != nil {
		bot.SendMessage(telegramID, "❌ خطا در دریافت چالش! متاسفانه چالشی با این مشخصات پیدا نشد.", nil)
		return
	}

	turn, _ := h.TodRepo.GetCurrentTurn(gameID)
	h.TodRepo.UpdateTurnChoice(turn.ID, choice)
	h.TodRepo.UpdateTurnChallenge(turn.ID, challenge.ID, challenge.Text)
	h.TodRepo.UpdateGameState(gameID, models.TodStateWaitingProof)

	h.ShowTodChallenge(gameID, challenge, bot)
}

func (h *HandlerManager) ShowTodChallenge(gameID uint, challenge *models.TodChallenge, bot BotInterface) {
	game, _ := h.TodRepo.GetGameByID(gameID)
	activeUser := getUserByID(game.ActivePlayerID, game.Match)
	passiveUser := getUserByID(game.PassivePlayerID, game.Match)

	// Format difficulty for display
	difficultyEmoji := "🟢"
	difficultyText := "آسان"
	if challenge.Difficulty == "medium" {
		difficultyEmoji = "🟡"
		difficultyText = "متوسط"
	} else if challenge.Difficulty == "hard" {
		difficultyEmoji = "🔴"
		difficultyText = "سخت"
	}

	challengeMsg := fmt.Sprintf(`🎯 **چالش جدید:**

%s

%s **درجه سختی:** %s
💰 **پاداش:** %d سکه`,
		challenge.Text,
		difficultyEmoji, difficultyText,
		challenge.CoinReward,
	)

	if challenge.ProofHint != "" {
		challengeMsg += fmt.Sprintf("\n\n💡 **راهنمایی مدرک:** %s", challenge.ProofHint)
	}

	// Responder view
	bot.SendMessage(activeUser.TelegramID, challengeMsg, bot.GetTodResponderInteractionKeyboard(gameID))

	// Questioner view (Passive player)
	bot.SendMessage(passiveUser.TelegramID, challengeMsg, bot.GetTodQuestionerInteractionKeyboard(gameID))
}

func getProofTypeText(proofType string) string {
	switch proofType {
	case models.ProofTypeText:
		return "متن"
	case models.ProofTypeVoice:
		return "ویس"
	case models.ProofTypeImage:
		return "عکس"
	case models.ProofTypeVideo:
		return "ویدیو"
	default:
		return "ندارد"
	}
}

// ========================================
// HELPERS
// ========================================

func getUserByID(userID uint, match models.Match) *models.User {
	if match.User1ID == userID {
		return &match.User1
	}
	return &match.User2
}

func (h *HandlerManager) StartTodGameWithMatch(userID int64, matchID uint, bot BotInterface) {
	match, err := h.MatchRepo.GetMatchByID(matchID)
	if err != nil {
		logger.Error("Failed to get match for ToD", "match_id", matchID, "error", err)
		return
	}

	game, err := h.TodRepo.CreateGame(match.ID, match.User1ID, match.User2ID)
	if err != nil {
		logger.Error("Failed to create ToD game", "match_id", matchID, "error", err)
		return
	}

	h.UserRepo.UpdateUserStatus(match.User1ID, models.UserStatusInMatch)
	h.UserRepo.UpdateUserStatus(match.User2ID, models.UserStatusInMatch)

	bot.SendMessage(match.User1.TelegramID, fmt.Sprintf("🎉 بازی با %s شروع شد!", match.User2.FullName), nil)
	bot.SendMessage(match.User2.TelegramID, fmt.Sprintf("🎉 بازی با %s شروع شد!", match.User1.FullName), nil)

	time.Sleep(2 * time.Second)
	h.HandleTodCoinFlip(game.ID, bot)
}

// StartTodFriends initiates the friends/group flow
func (h *HandlerManager) StartTodFriends(telegramID int64, bot BotInterface) {
	bot.SendMessage(telegramID, "👥 بخش بازی با دوستان به زودی فعال می‌شود.", nil)
}

// StartTodQuestionRegistration initiates the submission flow
func (h *HandlerManager) StartTodQuestionRegistration(telegramID int64, bot BotInterface) {
	bot.SendMessage(telegramID, "✍️ بخش ثبت سوال به زودی فعال می‌شود.", nil)
}
