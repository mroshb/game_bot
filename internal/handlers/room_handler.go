package handlers

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"github.com/mroshb/game_bot/pkg/logger"
	"github.com/mroshb/game_bot/pkg/utils"
)

type RoomSession struct {
	State    string
	RoomID   uint
	RoomData map[string]interface{}
}

const (
	StateRoomCreate     = "room_create"
	StateRoomName       = "room_name"
	StateRoomMaxPlayers = "room_max_players"
	StateRoomEntryFee   = "room_entry_fee"
	StateRoomJoinByCode = "room_join_code"
)

// ShowRoomMenu shows the room menu
func (h *HandlerManager) ShowRoomMenu(userID int64, bot BotInterface) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏛 ساخت اتاق عمومی (50 سکه)", "room_create_public"),
			tgbotapi.NewInlineKeyboardButtonData("🔒 ساخت اتاق خصوصی (30 سکه)", "room_create_private"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 لیست اتاقهای عمومی", "room_list_public"),
			tgbotapi.NewInlineKeyboardButtonData("⚡️ ورود سریع (رندوم)", "room_quick_join"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 ورود با کد", "room_join_code"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 اتاقهای من", "room_my_rooms"),
		),
	)

	msg := "🏛 سیستم اتاقها\n\nانتخاب کن:"
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// CreateRoom handles room creation
func (h *HandlerManager) CreateRoom(userID int64, roomType string, session *UserSession, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check coin balance
	cost := int64(50)
	if roomType == models.RoomTypePrivate {
		cost = 30
	}

	hasFunds, _ := h.CoinRepo.HasSufficientBalance(user.ID, cost)
	if !hasFunds {
		bot.SendMessage(userID, fmt.Sprintf("❌ سکه کافی نداری!\n\n💰 موجودی: %d\n💰 نیاز: %d", user.CoinBalance, cost), nil)
		return
	}

	// Deduct coins happens AFTER creation? No, usually before. But if user cancels?
	// Let's deduct now and refund if cancel.
	// Or better: Deduct when room is actually created.
	// Let's strict: Check funds now. Deduct in CompleteRoomCreation.

	// Ask for room name
	bot.SendMessage(userID, "📝 نام اتاق رو وارد کن:", nil)

	// Update session
	session.State = StateRoomName
	// Initialize map if nil
	if session.Data == nil {
		session.Data = make(map[string]interface{})
	}
	session.Data["room_type"] = roomType
}

func (h *HandlerManager) HandleRoomCreation(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	// Handle Cancel
	if text == BtnCancel {
		bot.SendMessage(userID, "❌ عملیات لغو شد.", nil)
		session.State = ""
		session.Data = make(map[string]interface{})
		return
	}

	switch session.State {
	case StateRoomName:
		name := text
		if len(name) < 3 {
			bot.SendMessage(userID, "❌ نام اتاق باید حداقل 3 حرف باشد!", nil)
			return
		}
		session.Data["room_name"] = name

		// Ask for max players
		session.State = StateRoomMaxPlayers

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("2", "btn:2"),
				tgbotapi.NewInlineKeyboardButtonData("3", "btn:3"),
				tgbotapi.NewInlineKeyboardButtonData("4", "btn:4"),
				tgbotapi.NewInlineKeyboardButtonData("5", "btn:5"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("6", "btn:6"),
				tgbotapi.NewInlineKeyboardButtonData("7", "btn:7"),
				tgbotapi.NewInlineKeyboardButtonData("8", "btn:8"),
				tgbotapi.NewInlineKeyboardButtonData("10", "btn:10"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
			),
		)
		bot.SendMessage(userID, "👥 ظرفیت اتاق رو انتخاب کن (2 تا 10 نفر):", keyboard)

	case StateRoomMaxPlayers:
		var maxPlayers int
		_, err := fmt.Sscanf(utils.NormalizePersianNumbers(text), "%d", &maxPlayers)
		if err != nil || maxPlayers < 2 || maxPlayers > 10 {
			bot.SendMessage(userID, "❌ لطفاً یک عدد بین 2 تا 10 وارد کن!", nil)
			return
		}

		session.Data["max_players"] = maxPlayers

		// Ask for entry fee
		session.State = StateRoomEntryFee
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("0", "btn:0"),
				tgbotapi.NewInlineKeyboardButtonData("5", "btn:5"),
				tgbotapi.NewInlineKeyboardButtonData("10", "btn:10"),
				tgbotapi.NewInlineKeyboardButtonData("20", "btn:20"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("50", "btn:50"),
				tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
			),
		)
		bot.SendMessage(userID, "💰 هزینه ورودی اتاق رو مشخص کن (سکه):", keyboard)

	case StateRoomEntryFee:
		var entryFee int64
		_, err := fmt.Sscanf(utils.NormalizePersianNumbers(text), "%d", &entryFee)
		if err != nil || entryFee < 0 {
			bot.SendMessage(userID, "❌ لطفاً یک مبلغ معتبر وارد کن!", nil)
			return
		}

		session.Data["entry_fee"] = entryFee

		// Remove numeric keyboard first
		bot.SendMessage(userID, "⏳ در حال ساخت اتاق...", tgbotapi.NewRemoveKeyboard(true))

		// Complete creation
		roomType := session.Data["room_type"].(string)
		roomName := session.Data["room_name"].(string)
		maxPlayers := session.Data["max_players"].(int)

		roomID := h.CompleteRoomCreation(userID, roomName, roomType, maxPlayers, entryFee, bot)
		if roomID > 0 {
			session.Data["current_room_id"] = roomID
			h.ShowRoomMembers(userID, roomID, bot)

			user, _ := h.UserRepo.GetUserByTelegramID(userID)
			isAdmin := user != nil && user.TelegramID == h.Config.SuperAdminTgID
			bot.SendMessage(userID, "💬 حالا می‌توانید در این اتاق پیام بفرستید یا بازی را شروع کنید.", bot.GetMainMenuKeyboard(isAdmin))
		}

		// Clear session state
		session.State = ""
	}
}

// CompleteRoomCreation completes the room creation process and returns the room ID
func (h *HandlerManager) CompleteRoomCreation(userID int64, roomName string, roomType string, maxPlayers int, entryFee int64, bot BotInterface) uint {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return 0
	}

	// Create room
	room := &models.Room{
		RoomName:   roomName,
		HostID:     user.ID,
		RoomType:   roomType,
		MaxPlayers: maxPlayers,
		EntryFee:   entryFee,
		Status:     models.RoomStatusWaiting,
	}

	// Deduct coins now
	cost := int64(50)
	if roomType == models.RoomTypePrivate {
		cost = 30
	}

	if err := h.CoinRepo.DeductCoins(user.ID, cost, models.TxTypeRoomCreation, "هزینه ساخت اتاق"); err != nil {
		logger.Error("Failed to deduct coins", "error", err)
		bot.SendMessage(userID, "❌ خطا در کسر سکه!", nil)
		return 0
	}

	if err := h.RoomRepo.CreateRoom(room); err != nil {
		logger.Error("Failed to create room", "error", err)
		// Refund coins
		h.CoinRepo.AddCoins(user.ID, cost, models.TxTypeRefund, "بازگشت هزینه به دلیل خطا")
		bot.SendMessage(userID, "❌ خطا در ساخت اتاق!", nil)
		return 0
	}

	// Add host as member
	h.RoomRepo.AddMember(room.ID, user.ID)

	// Send success message
	msg := fmt.Sprintf("✅ اتاق با موفقیت ساخته شد!\n\n🏛 نام: %s\n👥 ظرفیت: %d نفر\n💰 ورودی: %d سکه\n📊 وضعیت: در انتظار",
		room.RoomName, room.MaxPlayers, room.EntryFee)

	if roomType == models.RoomTypePrivate {
		msg += fmt.Sprintf("\n🔑 کد دعوت: %s", room.InviteCode)
	}

	bot.SendMessage(userID, msg, nil)

	logger.Info("Room created", "room_id", room.ID, "host_id", user.ID, "type", roomType)
	return room.ID
}

// ListPublicRooms lists all public rooms
func (h *HandlerManager) ListPublicRooms(userID int64, bot BotInterface) {
	rooms, err := h.RoomRepo.GetPublicRooms()
	if err != nil {
		logger.Error("Failed to get public rooms", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت لیست اتاقها!", nil)
		return
	}

	if len(rooms) == 0 {
		bot.SendMessage(userID, "📋 هیچ اتاق عمومی فعالی وجود ندارد!", nil)
		return
	}

	msg := "📋 اتاقهای عمومی:\n\n"
	var buttons [][]tgbotapi.InlineKeyboardButton

	for i, room := range rooms {
		memberCount, _ := h.RoomRepo.GetMemberCount(room.ID)
		feeText := "رایگان"
		if room.EntryFee > 0 {
			feeText = fmt.Sprintf("%d سکه", room.EntryFee)
		}
		msg += fmt.Sprintf("%d. %s (%d/%d نفر) - ورودی: %s\n", i+1, room.RoomName, memberCount, room.MaxPlayers, feeText)

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("ورود به %s", room.RoomName),
				fmt.Sprintf("room_join_%d", room.ID),
			),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// JoinRoom handles joining a room
func (h *HandlerManager) JoinRoom(userID int64, roomID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if room exists
	room, err := h.RoomRepo.GetRoomByID(roomID)
	if err != nil {
		bot.SendMessage(userID, "❌ اتاق پیدا نشد!", nil)
		return
	}

	// Check if room is closed
	if room.Status == models.RoomStatusClosed {
		bot.SendMessage(userID, "❌ این اتاق بسته شده!", nil)
		return
	}

	// Check if user is already a member
	isMember, _ := h.RoomRepo.IsMember(roomID, user.ID)
	if !isMember {
		// New member: Check Entry Fee
		if room.EntryFee > 0 {
			hasFunds, _ := h.CoinRepo.HasSufficientBalance(user.ID, room.EntryFee)
			if !hasFunds {
				bot.SendMessage(userID, fmt.Sprintf("❌ سکه کافی برای ورود نداری!\n\n💰 ورودی: %d\n💰 موجودی: %d", room.EntryFee, user.CoinBalance), nil)
				return
			}

			if err := h.CoinRepo.DeductCoins(user.ID, room.EntryFee, models.TxTypeRoomEntry, fmt.Sprintf("ورود به اتاق %s", room.RoomName)); err != nil {
				bot.SendMessage(userID, "❌ خطا در کسر سکه ورودی!", nil)
				return
			}
		}
	}

	// Add member
	if err := h.RoomRepo.AddMember(roomID, user.ID); err != nil {
		// Check if it's "Already in room" error
		if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.ErrCodeAlreadyExists {
			// Transparently proceed as if joined
		} else {
			bot.SendMessage(userID, fmt.Sprintf("❌ خطا در ورود به اتاق: %v", err), nil)
			return
		}
	}

	// Get members
	members, _ := h.RoomRepo.GetRoomMembers(roomID)

	// If room is full, Notify all with special message
	if len(members) >= room.MaxPlayers {
		for _, member := range members {
			h.ShowRoomMembers(member.TelegramID, roomID, bot)
		}
	} else {
		// Just refresh the list for current user
		h.ShowRoomMembers(userID, roomID, bot)
	}

	// If game is already active, show game menu to the new user and add them to participants
	session, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	if session != nil {
		// Add as participant if not already there
		participants, _ := h.GameRepo.GetParticipants(session.ID)
		isParticipant := false
		maxOrder := 0
		for _, p := range participants {
			if p.UserID == user.ID {
				isParticipant = true
			}
			if p.TurnOrder > maxOrder {
				maxOrder = p.TurnOrder
			}
		}

		if !isParticipant {
			h.GameRepo.AddParticipant(session.ID, user.ID, maxOrder+1)
			h.BroadcastGroupGameStatus(session.ID, bot, fmt.Sprintf("👤 %s به بازی ملحق شد!", user.FullName))
		} else {
			// Just refresh status for the user
			h.BroadcastGroupGameStatus(session.ID, bot, "")
		}
	}

	logger.Info("User joined room", "user_id", user.ID, "room_id", roomID)
}

// JoinRoomByCode handles joining a room by invite code
func (h *HandlerManager) JoinRoomByCode(userID int64, inviteCode string, bot BotInterface) {
	// Find room by invite code
	room, err := h.RoomRepo.GetRoomByInviteCode(utils.NormalizePersianNumbers(strings.TrimSpace(inviteCode)))
	if err != nil {
		bot.SendMessage(userID, "❌ کد دعوت نامعتبر است!", nil)
		return
	}

	// Join room
	h.JoinRoom(userID, room.ID, bot)
}

// LeaveRoom handles leaving a room
func (h *HandlerManager) LeaveRoom(userID int64, roomID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if user is host
	isHost, _ := h.RoomRepo.IsHost(roomID, user.ID)
	if isHost {
		// Close room if host leaves
		h.CloseRoom(userID, roomID, bot)
		return
	}

	// Check if game is active and it's this user's turn
	session, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	if session != nil && session.TurnUserID == user.ID {
		// User whose turn it is is leaving
		bot.SendMessage(userID, "⚠️ شما نوبت خود را به دلیل خروج از دست دادید.", nil)
		// Skip turn by using the host's telegram ID
		h.HandleGroupNextTurn(session.Room.Host.TelegramID, session.ID, bot)
	}

	// Remove member
	if err := h.RoomRepo.RemoveMember(roomID, user.ID); err != nil {
		bot.SendMessage(userID, "❌ خطا در ترک اتاق!", nil)
		return
	}

	bot.SendMessage(userID, "👋 از اتاق خارج شدید!", nil)

	// Show main menu
	isAdmin := user.TelegramID == h.Config.SuperAdminTgID
	bot.SendMainMenu(userID, isAdmin)

	// Notify other members and refresh game status if any
	members, _ := h.RoomRepo.GetRoomMembers(roomID)
	for _, member := range members {
		bot.SendMessage(member.TelegramID, fmt.Sprintf("👋 %s از اتاق خارج شد!", user.FullName), nil)
		if session != nil {
			h.BroadcastGroupGameStatus(session.ID, bot, fmt.Sprintf("👤 %s از اتاق خارج شد.", user.FullName))
		}
	}

	logger.Info("User left room", "user_id", user.ID, "room_id", roomID)
}

// CloseRoom closes a room (host only)
func (h *HandlerManager) CloseRoom(userID int64, roomID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if user is host
	isHost, _ := h.RoomRepo.IsHost(roomID, user.ID)
	if !isHost {
		bot.SendMessage(userID, "❌ فقط هاست می‌تواند اتاق را ببندد!", nil)
		return
	}

	// Get members before closing
	members, _ := h.RoomRepo.GetRoomMembers(roomID)

	// Check for active game and end it
	session, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	if session != nil {
		h.GameRepo.EndGame(session.ID)
	}

	// Close room
	if err := h.RoomRepo.CloseRoom(roomID); err != nil {
		logger.Error("Failed to close room", "error", err)
		bot.SendMessage(userID, "❌ خطا در بستن اتاق!", nil)
		return
	}

	// Notify all members and show main menu
	for _, member := range members {
		bot.SendMessage(member.TelegramID, "🚪 اتاق توسط هاست بسته شد!", nil)
		isAdmin := member.TelegramID == h.Config.SuperAdminTgID
		bot.SendMainMenu(member.TelegramID, isAdmin)
	}

	logger.Info("Room closed", "room_id", roomID, "host_id", user.ID)
}

// KickMember kicks a member from room (host only)
func (h *HandlerManager) KickMember(userID int64, roomID uint, targetUserID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if user is host
	isHost, _ := h.RoomRepo.IsHost(roomID, user.ID)
	if !isHost {
		bot.SendMessage(userID, "❌ فقط هاست می‌تواند اعضا را اخراج کند!", nil)
		return
	}

	// Get target user
	targetUser, err := h.UserRepo.GetUserByID(targetUserID)
	if err != nil {
		bot.SendMessage(userID, "❌ کاربر پیدا نشد!", nil)
		return
	}

	// Get room details
	room, err := h.RoomRepo.GetRoomByID(roomID)
	if err != nil {
		return
	}

	// Kick member
	if err := h.RoomRepo.KickMember(roomID, targetUserID); err != nil {
		logger.Error("Failed to kick member", "error", err)
		bot.SendMessage(userID, "❌ خطا در اخراج عضو!", nil)
		return
	}

	// If host was kicked, close the room
	if targetUserID == room.HostID {
		h.CloseRoom(userID, roomID, bot)
		return
	}

	// Notify
	bot.SendMessage(userID, fmt.Sprintf("✅ %s از اتاق اخراج شد!", targetUser.FullName), nil)
	bot.SendMessage(targetUser.TelegramID, "🚫 شما از اتاق اخراج شدید!", nil)

	// Show main menu to kicked user
	isAdmin := targetUser.TelegramID == h.Config.SuperAdminTgID
	bot.SendMainMenu(targetUser.TelegramID, isAdmin)

	// Notify other members
	members, _ := h.RoomRepo.GetRoomMembers(roomID)
	for _, member := range members {
		if member.ID != user.ID && member.ID != targetUserID {
			bot.SendMessage(member.TelegramID, fmt.Sprintf("🚫 %s از اتاق اخراج شد!", targetUser.FullName), nil)
		}
	}

	// Refresh management view for host
	h.ShowManageMembers(userID, roomID, bot)

	logger.Info("Member kicked", "room_id", roomID, "kicked_user_id", targetUserID, "by_user_id", user.ID)
}

// ShowManageMembers shows list of members with kick options for host
func (h *HandlerManager) ShowManageMembers(userID int64, roomID uint, bot BotInterface) {
	members, err := h.RoomRepo.GetRoomMembers(roomID)
	if err != nil {
		return
	}

	room, _ := h.RoomRepo.GetRoomByID(roomID)

	msg := fmt.Sprintf("⚙️ مدیریت اعضای اتاق '%s':\n\nبرای اخراج هر فرد، روی دکمه مربوطه کلیک کنید.", room.RoomName)
	var buttons [][]tgbotapi.InlineKeyboardButton

	for _, member := range members {
		if member.ID == room.HostID {
			continue // Can't kick self
		}

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🚫 اخراج %s", member.FullName), fmt.Sprintf("room_kick_%d_%d", roomID, member.ID)),
		))
	}

	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("room_members_%d", roomID)),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// ShowRoomMembers shows all members of a room
func (h *HandlerManager) ShowRoomMembers(userID int64, roomID uint, bot BotInterface) {
	members, err := h.RoomRepo.GetRoomMembers(roomID)
	if err != nil {
		logger.Error("Failed to get room members", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت لیست اعضا!", nil)
		return
	}

	room, _ := h.RoomRepo.GetRoomByID(roomID)

	msg := fmt.Sprintf("👥 اعضای اتاق '%s':\n\n", room.RoomName)
	for i, member := range members {
		status := "⚫️"
		if member.Status == models.UserStatusOnline {
			status = "🟢"
		}

		role := ""
		if member.ID == room.HostID {
			role = " (هاست)"
		}

		msg += fmt.Sprintf("%d. %s %s %s\n", i+1, status, member.FullName, role)
	}

	// Add buttons
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	var buttons [][]tgbotapi.InlineKeyboardButton

	isFull := len(members) >= room.MaxPlayers

	// Check for active game
	activeSession, _ := h.GameRepo.GetActiveGameSessionByRoomID(roomID)
	hasActiveGame := activeSession != nil && activeSession.Status != models.GameStatusFinished

	if room.HostID == user.ID {
		if hasActiveGame {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎮 بازگشت به بازی", fmt.Sprintf("gt_status_%d", activeSession.ID)),
			))
		} else if isFull {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔥 شروع بازی (جرعت یا حقیقت)", fmt.Sprintf("gt_start_%d", room.ID)),
			))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👑 شروع بازی (کوئیز اف کینگ)", fmt.Sprintf("qok_start_%d", room.ID)),
			))
		} else {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎮 شروع جرعت حقیقت", fmt.Sprintf("gt_start_%d", room.ID)),
				tgbotapi.NewInlineKeyboardButtonData("👑 شروع کوئیز", fmt.Sprintf("qok_start_%d", room.ID)),
			))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ دعوت از دوستان", fmt.Sprintf("gt_invite_%d", room.ID)),
			))
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚙️ مدیریت اعضا", fmt.Sprintf("room_manage_%d", room.ID)),
			))
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚪 بستن اتاق", fmt.Sprintf("room_close_%d", room.ID)),
		))
	} else {
		if hasActiveGame {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎮 بازگشت به بازی", fmt.Sprintf("gt_status_%d", activeSession.ID)),
			))
		} else if isFull {
			msg += "\n\n⏳ منتظر شروع بازی توسط هاست باشید..."
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👋 ترک اتاق", fmt.Sprintf("room_leave_%d", room.ID)),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// SendRoomMessage sends a message to all room members
// Returns true if processed
func (h *HandlerManager) SendRoomMessage(userID int64, roomID uint, message *tgbotapi.Message, bot BotInterface) bool {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return false
	}
	// Verify room and membership
	room, err := h.RoomRepo.GetRoomByID(roomID)
	if err != nil || room.Status == models.RoomStatusClosed {
		return false
	}
	isMember, _ := h.RoomRepo.IsMember(roomID, user.ID)
	if !isMember {
		return false
	}

	// Get all members
	members, err := h.RoomRepo.GetRoomMembers(roomID)
	if err != nil {
		logger.Error("Failed to get room members", "error", err)
		return false
	}

	// Send message to all members
	for _, member := range members {
		if member.ID == user.ID {
			continue // Skip sender
		}

		// Forward the content with sender name integrated
		h.forwardMessage(message, member.TelegramID, bot, user.FullName)
	}
	return true
}

// GetUserRooms shows user's active rooms
func (h *HandlerManager) GetUserRooms(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	rooms, err := h.RoomRepo.GetUserRooms(user.ID)
	if err != nil {
		logger.Error("Failed to get user rooms", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت اتاقها!", nil)
		return
	}

	if len(rooms) == 0 {
		bot.SendMessage(userID, "🏠 شما در هیچ اتاقی عضو نیستید!", nil)
		return
	}

	msg := "🏠 اتاقهای من:\n\n"
	for i, room := range rooms {
		memberCount, _ := h.RoomRepo.GetMemberCount(room.ID)
		role := ""
		if room.HostID == user.ID {
			role = " (هاست)"
		}
		msg += fmt.Sprintf("%d. %s (%d/%d نفر)%s\n", i+1, room.RoomName, memberCount, room.MaxPlayers, role)
	}

	bot.SendMessage(userID, msg, nil)
}

// QuickJoinRoom joins a random public room
func (h *HandlerManager) QuickJoinRoom(userID int64, bot BotInterface) {
	rooms, err := h.RoomRepo.GetPublicRooms()
	if err != nil || len(rooms) == 0 {
		bot.SendMessage(userID, "📋 هیچ اتاق عمومی فعالی برای ورود سریع پیدا نشد!", nil)
		return
	}

	// Try to find a room with space
	for _, room := range rooms {
		memberCount, _ := h.RoomRepo.GetMemberCount(room.ID)
		if memberCount < room.MaxPlayers {
			h.JoinRoom(userID, room.ID, bot)
			return
		}
	}

	bot.SendMessage(userID, "📋 متاسفانه تمامی اتاقها در حال حاضر پر هستند!", nil)
}

// InviteFriendToRoom shows friend list for invitation
func (h *HandlerManager) InviteFriendToRoom(userID int64, roomID uint, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	friends, err := h.FriendRepo.GetFriends(user.ID)
	if err != nil || len(friends) == 0 {
		bot.SendMessage(userID, "👥 شما هنوز دوستی ندارید که دعوت کنید!", nil)
		return
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, friend := range friends {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(friend.FullName, fmt.Sprintf("gt_send_inv_%d_%d", roomID, friend.ID)),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msgConfig := tgbotapi.NewMessage(userID, "👥 دوستت رو انتخاب کن تا دعوت‌نامه براش ارسال بشه:")
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// SendRoomInvitation sends an invitation to a friend
func (h *HandlerManager) SendRoomInvitation(hostID int64, roomID uint, friendID uint, bot BotInterface) {
	host, _ := h.UserRepo.GetUserByTelegramID(hostID)
	room, _ := h.RoomRepo.GetRoomByID(roomID)
	friend, _ := h.UserRepo.GetUserByID(friendID)

	if friend == nil {
		return
	}

	msg := fmt.Sprintf("📩 دعوت‌نامه بازی!\n\n👤 %s شما رو به اتاق '%s' دعوت کرده.", host.FullName, room.RoomName)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ قبول و ورود", fmt.Sprintf("gt_accept_inv_%d", roomID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ رد دعوت", fmt.Sprintf("gt_reject_inv_%d", roomID)),
		),
	)

	msgConfig := tgbotapi.NewMessage(friend.TelegramID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}

	bot.SendMessage(hostID, fmt.Sprintf("✅ دعوت‌نامه برای %s ارسال شد.", friend.FullName), nil)
}
