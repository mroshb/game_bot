package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// HandleAddFriend handles general friend requests (Paid)
func (h *HandlerManager) HandleAddFriend(userID int64, targetUserID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	// Check if already friends
	areFriends, _ := h.FriendRepo.AreFriends(user.ID, targetUserID)
	if areFriends {
		bot.SendMessage(userID, "✅ شما قبلاً با هم دوست شدید!", nil)
		return
	}

	// Check balance
	hasFunds, _ := h.CoinRepo.HasSufficientBalance(user.ID, h.Config.FriendRequestCost)
	if !hasFunds {
		bot.SendMessage(userID, fmt.Sprintf("❌ سکه کافی نداری! هزینه درخواست دوستی: %d سکه", h.Config.FriendRequestCost), nil)
		return
	}

	// Deduct
	if err := h.CoinRepo.DeductCoins(user.ID, h.Config.FriendRequestCost, models.TxTypeFriendRequest, "هزینه درخواست دوستی"); err != nil {
		bot.SendMessage(userID, "❌ خطا در کسر سکه!", nil)
		return
	}

	// Send request
	if err := h.FriendRepo.SendFriendRequest(user.ID, targetUserID); err != nil {
		logger.Error("Failed to send friend request", "error", err)
		h.CoinRepo.AddCoins(user.ID, h.Config.FriendRequestCost, models.TxTypeRefund, "بازگشت هزینه درخواست ناموفق")
		bot.SendMessage(userID, "⚠️ درخواست قبلاً ارسال شده یا خطایی رخ داد.", nil)
		return
	}

	// Notify Target
	targetUser, _ := h.UserRepo.GetUserByID(targetUserID)
	if targetUser != nil {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ قبول", fmt.Sprintf("friend_accept_%d", user.ID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ رد", fmt.Sprintf("friend_reject_%d", user.ID)),
			),
		)
		bot.SendMessage(targetUser.TelegramID, fmt.Sprintf("👋 %s درخواست دوستی داد!", user.FullName), keyboard)
	}

	bot.SendMessage(userID, fmt.Sprintf("✅ درخواست دوستی ارسال شد (-%d سکه)!", h.Config.FriendRequestCost), nil)
}

// HandleAddFriendFromMatch handles adding a friend specifically from a match context (Free)
func (h *HandlerManager) HandleAddFriendFromMatch(userID int64, matchID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	match, err := h.MatchRepo.GetMatchByID(matchID)
	if err != nil || match == nil {
		bot.SendMessage(userID, "❌ اطلاعات بازی یافت نشد!", nil)
		return
	}

	// Determine other user
	var targetUserID uint
	if match.User1ID == user.ID {
		targetUserID = match.User2ID
	} else if match.User2ID == user.ID {
		targetUserID = match.User1ID
	} else {
		bot.SendMessage(userID, "❌ شما در این بازی نبودید!", nil)
		return
	}

	// Check if already friends
	areFriends, _ := h.FriendRepo.AreFriends(user.ID, targetUserID)
	if areFriends {
		bot.SendMessage(userID, "✅ شما قبلاً با هم دوست شدید!", nil)
		return
	}

	// Normally free if session exists and not Ended/Refunded.
	isFree := false
	if match.Status == models.MatchStatusActive || match.Status == models.MatchStatusTimeout {
		isFree = true
	}

	if !isFree {
		// Paid request
		hasFunds, _ := h.CoinRepo.HasSufficientBalance(user.ID, h.Config.FriendRequestCost)
		if !hasFunds {
			bot.SendMessage(userID, fmt.Sprintf("❌ سکه کافی نداری! هزینه درخواست دوستی: %d سکه", h.Config.FriendRequestCost), nil)
			return
		}
		// Deduct handled later or here? Let's do it here for clarity.
		if err := h.CoinRepo.DeductCoins(user.ID, h.Config.FriendRequestCost, models.TxTypeFriendRequest, "هزینه درخواست دوستی"); err != nil {
			bot.SendMessage(userID, "❌ خطا در کسر سکه!", nil)
			return
		}
	}

	// Send request
	if err := h.FriendRepo.SendFriendRequest(user.ID, targetUserID); err != nil {
		logger.Error("Failed to send friend request", "error", err)
		if !isFree {
			// Refund on error
			h.CoinRepo.AddCoins(user.ID, h.Config.FriendRequestCost, models.TxTypeRefund, "بازگشت هزینه درخواست ناموفق")
		}
		bot.SendMessage(userID, "⚠️ درخواست قبلاً ارسال شده یا خطایی رخ داد.", nil)
		return
	}

	// Notify Target
	targetUser, _ := h.UserRepo.GetUserByID(targetUserID)
	if targetUser != nil {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ قبول", fmt.Sprintf("friend_accept_%d", user.ID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ رد", fmt.Sprintf("friend_reject_%d", user.ID)),
			),
		)
		bot.SendMessage(targetUser.TelegramID, fmt.Sprintf("👋 %s درخواست دوستی داد!", user.FullName), keyboard)
	}

	successMsg := "✅ درخواست دوستی ارسال شد (رایگان)!"
	if !isFree {
		successMsg = fmt.Sprintf("✅ درخواست دوستی ارسال شد (-%d سکه)!", h.Config.FriendRequestCost)
	}
	bot.SendMessage(userID, successMsg, nil)
}

// HandleFriendRequestAction handles Accept/Reject
func (h *HandlerManager) HandleFriendRequestAction(userID int64, targetUserID uint, action string, bot BotInterface) {
	// Find the pending request
	// We have targetUserID (requester) and userID (addressee/current user)
	// We need request ID to call Accept/Reject in repo?
	// Repo has AcceptFriendRequest(requestID)
	// So we must find the request first.
	// We need `GetFriendRequest(requesterID, addresseeID)`

	// Let's implement finding logic inline or add to repo.
	// For now, let's assume we fetch pending requests and filter.
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	requests, err := h.FriendRepo.GetPendingRequests(user.ID)
	if err != nil {
		return
	}

	var requestID uint
	for _, req := range requests {
		if req.RequesterID == targetUserID {
			requestID = req.ID
			break
		}
	}

	if requestID == 0 {
		bot.SendMessage(userID, "❌ درخواست معتبری یافت نشد!", nil)
		return
	}

	switch action {
	case "accept":
		if err := h.FriendRepo.AcceptFriendRequest(requestID); err != nil {
			bot.SendMessage(userID, "❌ خطا در قبول درخواست!", nil)
			return
		}
		bot.SendMessage(userID, "✅ درخواست دوستی قبول شد!", nil)

		// Notify Requester
		requester, _ := h.UserRepo.GetUserByID(targetUserID)
		if requester != nil {
			bot.SendMessage(requester.TelegramID, fmt.Sprintf("🥳 %s درخواست دوستی شما را قبول کرد!", user.FullName), nil)
		}

	case "reject":
		h.FriendRepo.RejectFriendRequest(requestID)
		bot.SendMessage(userID, "❌ درخواست دوستی رد شد.", nil)
	}
}

// HandleRemoveFriend handles removing a friend
func (h *HandlerManager) HandleRemoveFriend(userID int64, friendID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	if err := h.FriendRepo.RemoveFriend(user.ID, friendID); err != nil {
		bot.SendMessage(userID, "❌ خطا در حذف دوست!", nil)
		return
	}

	bot.SendMessage(userID, "🗑 دوست با موفقیت حذف شد.", nil)
	// Refresh list
	h.ShowFriendsList(userID, bot)
}

// ShowFriendsList displays friends with management options
func (h *HandlerManager) ShowFriendsList(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	friends, err := h.FriendRepo.GetFriends(user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت لیست!", nil)
		return
	}

	if len(friends) == 0 {
		bot.SendMessage(userID, "😔 شما هنوز دوستی ندارید.", nil)
		return
	}

	msg := "👥 لیست دوستان شما:\n\n"
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	for i, f := range friends {
		status := "🔴 آفلاین"
		switch f.Status {
		case models.UserStatusOnline:
			status = "🟢 آنلاین"
		case models.UserStatusInMatch:
			status = "🟡 در بازی"
		}

		msg += fmt.Sprintf("%d. %s (%s)\n", i+1, f.FullName, status)

		// Add manage button for each friend (Limit to 5-10 to avoid huge msg? Pagination needed for real app but MVP ok)
		if i < 10 {
			row := tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👁 %s", f.FullName), fmt.Sprintf("user_%s", f.PublicID)), // View Profile
				tgbotapi.NewInlineKeyboardButtonData("🗑 حذف", fmt.Sprintf("friend_remove_%d", f.ID)),
			)
			keyboardRows = append(keyboardRows, row)
		}
	}

	if len(friends) > 10 {
		msg += "\n(... و بیشتر)"
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	bot.SendMessage(userID, msg, kb)
}
