package handlers

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// SkipKeyboard creates skip/cancel inline keyboard for handlers package usage
func SkipKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnSkip, "btn:"+BtnSkip),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
		),
	)
}
