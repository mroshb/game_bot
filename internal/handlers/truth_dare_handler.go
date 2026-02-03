package handlers

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// StartTruthOrDare starts a Truth or Dare game (1v1)
func (h *HandlerManager) StartTruthOrDare(userID int64, bot BotInterface) {
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

	// Show Truth or Dare buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤔 حقیقت", fmt.Sprintf("truth_%d", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🎯 جرات", fmt.Sprintf("dare_%d", match.ID)),
		),
	)

	msg := "🎮 بازی حقیقت یا جرات!\n\nانتخاب کن:"
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// HandleTruthOrDareChoice handles choice in 1v1 and shows category selection
func (h *HandlerManager) HandleTruthOrDareChoice(userID int64, matchID uint, choice string, bot BotInterface) {
	if _, err := h.UserRepo.GetUserByTelegramID(userID); err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	// Simplified UI: No gender selection, just difficulty/vibe
	var buttons [][]tgbotapi.InlineKeyboardButton

	prefix := "truth"
	if choice == "dare" {
		prefix = "dare"
	}

	buttons = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("😇 فان (Fun)", fmt.Sprintf("match_cat_%d_%s_normal", matchID, prefix)),
			tgbotapi.NewInlineKeyboardButtonData("🔥 هات (Hot)", fmt.Sprintf("match_cat_%d_%s_sexy", matchID, prefix)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🎲 شانسی", fmt.Sprintf("match_cat_%d_%s_random", matchID, prefix)),
		},
	}

	msg := fmt.Sprintf("🎮 بازی %s (۱ به ۱):\n\n📍 سبک سوال رو انتخاب کن:", choice)
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	if api := bot.GetAPI(); api != nil {
		if b, ok := api.(*tgbotapi.BotAPI); ok {
			b.Send(msgConfig)
		}
	}
}

// HandleMatchTruthOrDareCategorySelection handles category selection for 1v1 match
func (h *HandlerManager) HandleMatchTruthOrDareCategorySelection(userID int64, matchID uint, category string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	// Determine question type (Truth vs Dare)
	// format: prefix_matchID_TYPE_CATEGORY (e.g. match_cat_123_truth_normal)
	// Expected key from UI: truth_normal, dare_sexy, etc.
	// But the handler actually passes the full string after prefix?
	// The Bot.go logic splits it. Let's assume 'category' here is just "truth_normal", "dare_sexy" etc. based on how handleCallback works.
	// Actually, looking at Bot.go (not visible here, but usually it strips prefix), let's assume 'category' passed here is like "truth_normal".

	// BUT wait, in my new code I put "match_cat_%d_%s_normal".
	// The caller likely strips "match_cat_%d_".
	// Let's ensure we parse correctly.

	// Resolving specific category based on User Gender
	genderSuffix := "boy"
	if user.Gender == models.GenderFemale {
		genderSuffix = "girl"
	}

	baseCategory := category // e.g. "truth_normal"
	prefix := "truth"
	if strings.HasPrefix(baseCategory, "dare") {
		prefix = "dare"
	}

	// Handle "random"
	if strings.Contains(baseCategory, "random") {
		randVibe := "normal"
		if time.Now().UnixNano()%2 == 0 {
			randVibe = "sexy"
		}
		baseCategory = fmt.Sprintf("%s_%s", prefix, randVibe)
	}

	finalCategory := fmt.Sprintf("%s_%s", baseCategory, genderSuffix) // e.g. truth_normal_boy

	qType := models.QuestionTypeTruth
	if strings.HasPrefix(finalCategory, "dare") {
		qType = models.QuestionTypeDare
	}

	question, err := h.GameRepo.GetRandomQuestion(qType, finalCategory)
	if err != nil {
		// Fallback
		question, err = h.GameRepo.GetRandomQuestion(qType, "")
		if err != nil {
			bot.SendMessage(userID, "❌ سوالی یافت نشد!", nil)
			return
		}
	}

	// Get match to find the other user
	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil || match == nil {
		bot.SendMessage(userID, "❌ چت فعالی پیدا نشد!", nil)
		return
	}

	// Determine other user
	var otherUserID uint
	if match.User1ID == user.ID {
		otherUserID = match.User2ID
	} else {
		otherUserID = match.User1ID
	}

	otherUser, _ := h.UserRepo.GetUserByID(otherUserID)

	// Send question
	// Send question
	genre := "عادی/فان"
	if strings.Contains(finalCategory, "sexy") {
		genre = "هات (+18)"
	}

	msg := fmt.Sprintf("🎮 %s انتخاب کرد: %s\n\n❓ %s\n\n💰 پاداش: %d سکه",
		user.FullName, genre, question.QuestionText, question.Points)

	bot.SendMessage(userID, msg, nil)
	if otherUser != nil {
		bot.SendMessage(otherUser.TelegramID, msg, nil)
	}

	// Award coins
	h.CoinRepo.AddCoins(user.ID, int64(question.Points), models.TxTypeGameReward, "پاداش بازی حقیقت یا جرات")
	if otherUser != nil {
		h.CoinRepo.AddCoins(otherUser.ID, int64(question.Points), models.TxTypeGameReward, "پاداش بازی حقیقت یا جرات")
	}

	// Award Village XP
	h.VillageSvc.AddXPForUser(user.ID, 10)
	if otherUser != nil {
		h.VillageSvc.AddXPForUser(otherUser.ID, 10)
	}
}

// StartGroupTruthDare starts a Truth or Dare game for a room
func (h *HandlerManager) StartGroupTruthDare(userID int64, roomID uint, bot BotInterface) {
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

	// Check if game already active
	activeSession, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	if activeSession != nil {
		bot.SendMessage(userID, "⚠️ یک بازی در حال حاضر فعال است!", nil)
		return
	}

	// Check member count
	members, _ := h.RoomRepo.GetRoomMembers(roomID)
	if len(members) < 2 {
		bot.SendMessage(userID, "👥 حداقل ۲ نفر برای شروع بازی لازم است!", nil)
		return
	}

	// Create session
	session, err := h.GameRepo.CreateGameSession(roomID, models.GameTypeTruthDare)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در ایجاد جلسه بازی!", nil)
		return
	}

	// Add members as participants
	for i, member := range members {
		if err := h.GameRepo.AddParticipant(session.ID, member.ID, i+1); err != nil {
			logger.Error("Failed to add participant", "session_id", session.ID, "user_id", member.ID, "error", err)
			bot.SendMessage(userID, "❌ خطا در افزودن شرکت‌کنندگان!", nil)
			return
		}
	}

	// Start game
	h.GameRepo.StartGame(session.ID)
	h.GameRepo.UpdateGameStatus(session.ID, models.GameStatusWaitingForChoice)
	h.GameRepo.SetTurnUserID(session.ID, members[0].ID)

	// Notify all with turn-specific menus
	msg := fmt.Sprintf("🎮 بازی جرات یا حقیقت شروع شد!\n\n👤 نوبت بازیکن: %s\n\nمنتظر انتخاب بازیکن...", members[0].FullName)
	h.BroadcastGroupGameStatus(session.ID, bot, msg)
}

// HandleGroupTruthOrDareChoice handles the choice (truth/dare) and shows category selection
func (h *HandlerManager) HandleGroupTruthOrDareChoice(userID int64, sessionID uint, choice string, bot BotInterface) {
	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil {
		return
	}

	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if session.TurnUserID != user.ID {
		bot.SendMessage(userID, "⚠️ نوبت شما نیست!", nil)
		return
	}

	// Simplified UI
	var buttons [][]tgbotapi.InlineKeyboardButton

	prefix := "truth"
	if choice == "dare" {
		prefix = "dare"
	}

	buttons = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("😇 فان (Fun)", fmt.Sprintf("gt_cat_%d_%s_normal", sessionID, prefix)),
			tgbotapi.NewInlineKeyboardButtonData("🔥 هات (Hot)", fmt.Sprintf("gt_cat_%d_%s_sexy", sessionID, prefix)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🎲 شانسی", fmt.Sprintf("gt_cat_%d_%s_random", sessionID, prefix)),
		},
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("gt_status_%d", sessionID)),
	))

	msg := fmt.Sprintf("🎮 نوبت شماست (%s)!\n\n📍 سبک سوال رو انتخاب کن:", user.FullName)
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	if api := bot.GetAPI(); api != nil {
		if b, ok := api.(*tgbotapi.BotAPI); ok {
			b.Send(msgConfig)
		}
	}
}

// HandleGroupTruthOrDareCategorySelection handles category selection and shows the question
func (h *HandlerManager) HandleGroupTruthOrDareCategorySelection(userID int64, sessionID uint, category string, bot BotInterface) {
	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil {
		return
	}

	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	if session.TurnUserID != user.ID {
		bot.SendMessage(userID, "⚠️ نوبت شما نیست!", nil)
		return
	}

	// Resolving specific category based on User Gender for Group Game
	genderSuffix := "boy"
	if user.Gender == models.GenderFemale {
		genderSuffix = "girl"
	}

	baseCategory := category
	prefix := "truth"
	if strings.Contains(baseCategory, "dare") {
		prefix = "dare"
	}

	// Handle "random"
	if strings.Contains(baseCategory, "random") {
		randVibe := "normal"
		if time.Now().UnixNano()%2 == 0 {
			randVibe = "sexy"
		}
		baseCategory = fmt.Sprintf("%s_%s", prefix, randVibe)
	}

	finalCategory := fmt.Sprintf("%s_%s", baseCategory, genderSuffix)

	qType := models.QuestionTypeTruth
	if strings.HasPrefix(finalCategory, "dare") {
		qType = models.QuestionTypeDare
	}

	question, err := h.GameRepo.GetRandomQuestion(qType, finalCategory)
	if err != nil {
		// Fallback to random if category not found
		question, err = h.GameRepo.GetRandomQuestion(qType, "")
		if err != nil {
			bot.SendMessage(userID, "❌ خطایی رخ داد! (سوال یافت نشد)", nil)
			return
		}
	}

	genre := "عادی/فان"
	if strings.Contains(finalCategory, "sexy") {
		genre = "هات (+18)"
	}

	msg := fmt.Sprintf("🎮 %s انتخاب کرد: %s\n\n❓ %s\n\n📸 بعد از انجام، دکمه «انجام دادم» را بزنید.",
		user.FullName, genre, question.QuestionText)

	// Update status to waiting for host
	h.GameRepo.UpdateGameStatus(sessionID, models.GameStatusWaitingForHost)

	// Broadcast with "Wait for Host" status (Host gets Next button)
	h.BroadcastGroupGameWithQuestionStatus(session.ID, bot, msg, category)
}

// BroadcastGroupGameWithQuestionStatus sends status with 'Change Question' button
func (h *HandlerManager) BroadcastGroupGameWithQuestionStatus(sessionID uint, bot BotInterface, message, category string) {
	session, _ := h.GameRepo.GetGameSession(sessionID)
	room, _ := h.RoomRepo.GetRoomByID(session.RoomID)
	members, _ := h.RoomRepo.GetRoomMembers(session.RoomID)

	for _, member := range members {
		var buttons [][]tgbotapi.InlineKeyboardButton
		isTurn := member.ID == session.TurnUserID
		isHost := member.ID == room.HostID

		if isHost {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏭ نفر بعدی (Force)", fmt.Sprintf("gt_next_%d", session.ID)),
			))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🛑 بستن روم", fmt.Sprintf("room_close_%d", room.ID)),
			))
		} else {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👋 ترک اتاق", fmt.Sprintf("room_leave_%d", room.ID)),
			))
		}

		if isTurn {
			// Player can confirm they are done
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ انجام دادم", fmt.Sprintf("gt_next_%d", session.ID)),
			))

			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("♻️ تغییر سوال", fmt.Sprintf("gt_change_%d_%s", session.ID, category)),
			))
		}

		msgConfig := tgbotapi.NewMessage(member.TelegramID, message)
		msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

		if api := bot.GetAPI(); api != nil {
			if b, ok := api.(*tgbotapi.BotAPI); ok {
				b.Send(msgConfig)
			}
		}
	}
}

// HandleGroupNextTurn moves to the next player
func (h *HandlerManager) HandleGroupNextTurn(adminID int64, sessionID uint, bot BotInterface) {
	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil {
		return
	}

	admin, _ := h.UserRepo.GetUserByTelegramID(adminID)

	// Allow Host OR Current Player (if status allows)
	canProceed := false
	if session.Room.HostID == admin.ID {
		canProceed = true
	} else if session.TurnUserID == admin.ID && session.Status == models.GameStatusWaitingForHost {
		canProceed = true
	}

	if !canProceed {
		bot.SendMessage(adminID, "❌ شما دسترسی به این کار را ندارید! (فقط میزبان یا بازیکن در نوبت)", nil)
		return
	}

	// Only reward and progress if we were waiting for the host to confirm completion
	if session.Status == models.GameStatusWaitingForHost {
		// Reward the player who just finished
		h.CoinRepo.AddCoins(session.TurnUserID, 15, models.TxTypeGameReward, "پاداش انجام چالش جرعت یا حقیقت")
		// Notify the player - need TelegramID
		turnUser, _ := h.UserRepo.GetUserByID(session.TurnUserID)
		if turnUser != nil {
			bot.SendMessage(turnUser.TelegramID, "💰 شما 15 سکه پاداش برای انجام چالش دریافت کردید!", nil)
			// Award Village XP for completing group challenge
			h.VillageSvc.AddXPForUser(session.TurnUserID, 15)
		}

	} else if session.Status != models.GameStatusWaitingForChoice && session.Status != models.GameStatusInProgress {
		// If status is something else (like waiting for choice), we might be skipping a turn.
		// But usually we only want to proceed if a choice was made OR if we are explicitly skipping.
		// For now, let's allow it but maybe with a different logic if needed.
	}

	// Get all participants
	allParticipants, _ := h.GameRepo.GetParticipants(sessionID)
	if len(allParticipants) == 0 {
		// Fallback: If participants were never added (e.g. error during start), try to add current room members
		members, _ := h.RoomRepo.GetRoomMembers(session.RoomID)
		if len(members) >= 2 {
			for i, m := range members {
				h.GameRepo.AddParticipant(sessionID, m.ID, i+1)
			}
			allParticipants, _ = h.GameRepo.GetParticipants(sessionID)
		}
	}

	if len(allParticipants) == 0 {
		return
	}

	// Filter participants who are still in the room
	var activeParticipants []models.GameParticipant
	for _, p := range allParticipants {
		isMember, _ := h.RoomRepo.IsMember(session.RoomID, p.UserID)
		if isMember {
			activeParticipants = append(activeParticipants, p)
		}
	}

	if len(activeParticipants) == 0 {
		// No one left? End game
		h.GameRepo.EndGame(sessionID)
		return
	}

	var nextUser *models.User
	for i, p := range activeParticipants {
		if p.UserID == session.TurnUserID {
			nextIdx := (i + 1) % len(activeParticipants)
			nextUser = &activeParticipants[nextIdx].User
			break
		}
	}

	if nextUser == nil {
		nextUser = &activeParticipants[0].User
	}

	h.GameRepo.SetTurnUserID(sessionID, nextUser.ID)
	h.GameRepo.UpdateGameStatus(sessionID, models.GameStatusWaitingForChoice)

	// Notify all with turn-specific menus
	msg := fmt.Sprintf("⏭ میزبان نوبت را به نفر بعدی داد.\n\n👤 نوبت بازیکن: %s", nextUser.FullName)
	h.BroadcastGroupGameStatus(sessionID, bot, msg)
}

// HandleGroupEndGame ends the group game and closes the room
func (h *HandlerManager) HandleGroupEndGame(adminID int64, sessionID uint, bot BotInterface) {
	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil {
		return
	}

	admin, _ := h.UserRepo.GetUserByTelegramID(adminID)
	if session.Room.HostID != admin.ID {
		bot.SendMessage(adminID, "❌ فقط میزبان می‌تواند بازی را تمام کند!", nil)
		return
	}

	// End the game session
	h.GameRepo.EndGame(sessionID)

	// Close the room completely
	h.CloseRoom(admin.TelegramID, session.RoomID, bot)
}

// BroadcastGroupGameStatus sends the current game status and appropriate keyboards to all members
func (h *HandlerManager) BroadcastGroupGameStatus(sessionID uint, bot BotInterface, message string) {
	session, err := h.GameRepo.GetGameSession(sessionID)
	if err != nil {
		return
	}

	room, _ := h.RoomRepo.GetRoomByID(session.RoomID)
	members, _ := h.RoomRepo.GetRoomMembers(session.RoomID)

	turnUser, _ := h.UserRepo.GetUserByID(session.TurnUserID)
	turnName := "نامشخص"
	if turnUser != nil {
		turnName = turnUser.FullName
	}

	isWaitingForHost := session.Status == models.GameStatusWaitingForHost

	for _, member := range members {
		var buttons [][]tgbotapi.InlineKeyboardButton

		isTurn := member.ID == session.TurnUserID
		isHost := member.ID == room.HostID

		// Create a personalized message
		statusPrefix := "🎮 بازی جرعت یا حقیقت\n"
		if isWaitingForHost {
			statusPrefix += "⌛️ در انتظار تایید میزبان..."
		} else {
			statusPrefix += fmt.Sprintf("👤 نوبت: %s", turnName)
		}

		fullMessage := fmt.Sprintf("%s\n━━━━━━━━━━━━━━\n%s", statusPrefix, message)

		if isHost {
			if isWaitingForHost {
				// Only host needs the "Next" button
				buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⏭ نفر بعدی", fmt.Sprintf("gt_next_%d", session.ID)),
				))
			} else if isTurn {
				// Host's turn to choose
				buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🤔 حقیقت", fmt.Sprintf("gt_truth_%d", session.ID)),
					tgbotapi.NewInlineKeyboardButtonData("🎯 جرات", fmt.Sprintf("gt_dare_%d", session.ID)),
				))
			}
			// Host always has Close Room, but keep it minimal
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🛑 بستن روم", fmt.Sprintf("room_close_%d", room.ID)),
			))
		} else if isTurn && !isWaitingForHost {
			// Player's turn to choose
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🤔 حقیقت", fmt.Sprintf("gt_truth_%d", session.ID)),
				tgbotapi.NewInlineKeyboardButtonData("🎯 جرات", fmt.Sprintf("gt_dare_%d", session.ID)),
			))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👋 ترک اتاق", fmt.Sprintf("room_leave_%d", room.ID)),
			))
		} else {
			// For others, keep a clean chat. Only provide Leave button.
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👋 ترک اتاق", fmt.Sprintf("room_leave_%d", room.ID)),
			))
		}

		msgConfig := tgbotapi.NewMessage(member.TelegramID, fullMessage)
		if len(buttons) > 0 {
			msgConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		}

		if api := bot.GetAPI(); api != nil {
			if b, ok := api.(*tgbotapi.BotAPI); ok {
				b.Send(msgConfig)
			}
		}
	}
}
