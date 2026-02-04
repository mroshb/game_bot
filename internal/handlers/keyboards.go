package handlers

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

// AdvancedSearchAgeKeyboard creates inline age selection keyboard (13-100)
func AdvancedSearchAgeKeyboard(step string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for age := 13; age <= 100; age++ {
		// Use a specific prefix to handle callback easily
		// step is "min" or "max"
		btn := tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(age), "search_age_"+step+"_"+strconv.Itoa(age))
		currentRow = append(currentRow, btn)
		if len(currentRow) == 8 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnSkip, "search_age_skip"),
		tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// AdvancedSearchProvinceKeyboard creates inline province selection keyboard with multi-select support
func AdvancedSearchProvinceKeyboard(selected map[string]bool) tgbotapi.InlineKeyboardMarkup {
	provinces := []string{
		"تهران", "کرج", "البرز", "خوزستان", "بوشهر", "اصفهان",
		"خراسان رضوی", "فارس", "آذربایجان شرقی", "مازندران",
		"کرمان", "گیلان", "کهگیلویه و بویراحمد",
		"آذربایجان غربی", "هرمزگان", "مرکزی", "یزد",
		"فرامنطقه ای", "کرمانشاه", "قزوین", "سیستان و بلوچستان",
		"همدان", "ایلام", "گلستان", "لرستان",
		"زنجان", "اردبیل", "قم", "کردستان",
		"سمنان", "چهارمحال و بختیاری", "خراسان شمالی", "خراسان جنوبی",
		"خارج از ایران",
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Actions Row
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ تایید نهایی", "search_province_confirm"),
		tgbotapi.NewInlineKeyboardButtonData("🌐 انتخاب همه", "search_province_all"),
	))

	var currentRow []tgbotapi.InlineKeyboardButton

	for _, p := range provinces {
		label := p
		if selected[p] {
			label = "✅ " + p
		}
		currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(label, "search_province_toggle_"+p))
		if len(currentRow) == 2 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnSkip, "search_province_skip"),
		tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
