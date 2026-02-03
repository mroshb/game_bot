package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowMatchMenu shows the match menu after finding a match
func (h *HandlerManager) ShowMatchMenu(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		return
	}

	// Check if user is in an active match
	match, err := h.MatchRepo.GetActiveMatch(user.ID)
	if err != nil || match == nil {
		bot.SendMessage(userID, "⚠️ شما در چت فعالی نیستید!", nil)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 چت", fmt.Sprintf("match_chat_%d", match.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎮 بازی کوییز", fmt.Sprintf("match_quiz_%d", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🤔 حقیقت یا جرات", fmt.Sprintf("match_truth_dare_%d", match.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 اضافه به دوستان", fmt.Sprintf("match_add_friend_%d", match.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚪 پایان مسابقه", fmt.Sprintf("match_end_%d", match.ID)),
		),
	)

	msg := "🎮 منوی Match\n\nانتخاب کن:"
	msgConfig := tgbotapi.NewMessage(userID, msg)
	msgConfig.ReplyMarkup = keyboard

	if apiInterface := bot.GetAPI(); apiInterface != nil {
		if api, ok := apiInterface.(*tgbotapi.BotAPI); ok {
			api.Send(msgConfig)
		}
	}
}

// Match menu logic is handled in bot.go for most cases
