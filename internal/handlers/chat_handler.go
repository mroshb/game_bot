package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

func (h *HandlerManager) HandleChatMessage(message *tgbotapi.Message, user *models.User, bot BotInterface) {
	if user == nil {
		return
	}

	// Get active match
	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil {
		logger.Error("Failed to get active match", "error", err)
		return
	}

	if match == nil {
		bot.SendMessage(message.From.ID, "⚠️ شما در چت فعالی نیستید!", nil)
		return
	}

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
		bot.SendMessage(message.From.ID, "❌ خطا در ارسال پیام!", nil)
		return
	}

	// Check status for cost calculation (Disabled - Unlimited Chat)
	// msgCost := int64(0) (Disabled - Unlimited Chat)
	/*
		if match.Status == models.MatchStatusTimeout {
			msgCost = 2
		}

		if msgCost > 0 {
			hasFunds, _ := h.CoinRepo.HasSufficientBalance(user.ID, msgCost)
			if !hasFunds {
				bot.SendMessage(message.From.ID, fmt.Sprintf("❌ سکه کافی نداری! هزینه هر پیام بعد از پایان زمان: %d سکه", msgCost), nil)
				return
			}

			if err := h.CoinRepo.DeductCoins(user.ID, msgCost, models.TxTypeMessage, "هزینه ارسال پیام (بعد از پایان match)"); err != nil {
				logger.Error("Failed to deduct coins for message", "error", err)
				bot.SendMessage(message.From.ID, "❌ خطا در کسر سکه!", nil)
				return
			}
		}
	*/

	// Forward message to other user
	if err := h.forwardMessage(message, otherUser.TelegramID, bot, ""); err != nil {
		logger.Error("Failed to forward message", "error", err)
		bot.SendMessage(message.From.ID, "❌ خطا در ارسال پیام!", nil)
		return
	}

	logger.Debug("Message forwarded", "from", user.ID, "to", otherUser.ID)
}

func (h *HandlerManager) forwardMessage(message *tgbotapi.Message, targetChatID int64, bot BotInterface, senderName string) error {
	api := bot.GetAPI().(*tgbotapi.BotAPI)

	prefix := ""
	if senderName != "" {
		prefix = fmt.Sprintf("👤 %s:\n━━━━━━━━━━━━━━\n", senderName)
	}

	// Forward text messages
	if message.Text != "" {
		msg := tgbotapi.NewMessage(targetChatID, prefix+message.Text)
		msg.ParseMode = tgbotapi.ModeMarkdown // Support some formatting if possible
		_, err := api.Send(msg)
		return err
	}

	caption := prefix
	if message.Caption != "" {
		caption += message.Caption
	}

	// Forward photos
	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		msg := tgbotapi.NewPhoto(targetChatID, tgbotapi.FileID(photo.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward voice messages
	if message.Voice != nil {
		msg := tgbotapi.NewVoice(targetChatID, tgbotapi.FileID(message.Voice.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward stickers
	if message.Sticker != nil {
		if senderName != "" {
			// Stickers don't have captions, so send intro first
			introMsg := tgbotapi.NewMessage(targetChatID, fmt.Sprintf("👤 %s یک استیکر فرستاد:", senderName))
			api.Send(introMsg)
		}
		msg := tgbotapi.NewSticker(targetChatID, tgbotapi.FileID(message.Sticker.FileID))
		_, err := api.Send(msg)
		return err
	}

	// Forward videos
	if message.Video != nil {
		msg := tgbotapi.NewVideo(targetChatID, tgbotapi.FileID(message.Video.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward documents
	if message.Document != nil {
		msg := tgbotapi.NewDocument(targetChatID, tgbotapi.FileID(message.Document.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward audio
	if message.Audio != nil {
		msg := tgbotapi.NewAudio(targetChatID, tgbotapi.FileID(message.Audio.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward animations
	if message.Animation != nil {
		msg := tgbotapi.NewAnimation(targetChatID, tgbotapi.FileID(message.Animation.FileID))
		msg.Caption = caption
		_, err := api.Send(msg)
		return err
	}

	// Forward video notes (round videos)
	if message.VideoNote != nil {
		if senderName != "" {
			introMsg := tgbotapi.NewMessage(targetChatID, fmt.Sprintf("👤 %s یک ویدیو پیام فرستاد:", senderName))
			api.Send(introMsg)
		}
		msg := tgbotapi.NewVideoNote(targetChatID, message.VideoNote.Length, tgbotapi.FileID(message.VideoNote.FileID))
		_, err := api.Send(msg)
		return err
	}

	return nil
}

func (h *HandlerManager) SendFriendRequest(fromUserID, toUserID uint, bot BotInterface) error {
	// Check sufficient funds
	if h.Config.FriendRequestCost > 0 {
		hasFunds, _ := h.CoinRepo.HasSufficientBalance(fromUserID, h.Config.FriendRequestCost)
		if !hasFunds {
			// Notify user - we can't easily notify via return error here as this might be called from callback
			// But assumption is this is called from handler which handles error?
			// The handler calls this. I should probably return specific error or handle messaging here.
			// Ideally return error and let caller handle.
			return fmt.Errorf("insufficient_funds")
		}
	}

	// Send friend request
	if err := h.FriendRepo.SendFriendRequest(fromUserID, toUserID); err != nil {
		return err
	}

	// Deduct coins
	if h.Config.FriendRequestCost > 0 {
		h.CoinRepo.DeductCoins(fromUserID, h.Config.FriendRequestCost, models.TxTypeFriendRequest, "هزینه درخواست دوستی")
	}

	// Get users
	fromUser, _ := h.UserRepo.GetUserByID(fromUserID)
	toUser, _ := h.UserRepo.GetUserByID(toUserID)

	if fromUser != nil && toUser != nil {
		// Notify sender
		bot.SendMessage(fromUser.TelegramID, "✅ درخواست دوستی ارسال شد!", nil)

		// Notify receiver
		msg := fmt.Sprintf("👥 درخواست دوستی جدید از %s", fromUser.FullName)
		bot.SendMessage(toUser.TelegramID, msg, nil)
	}

	return nil
}
