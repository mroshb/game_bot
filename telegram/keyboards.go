package telegram

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MainMenuKeyboard creates the main menu keyboard
func MainMenuKeyboard(_ bool) tgbotapi.ReplyKeyboardMarkup {
	var rows [][]tgbotapi.KeyboardButton

	// Row 1 - My Village
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(BtnVillageHub),
	))

	// Row 2 - Play! - Chat Now!
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(BtnPlayGame),
		tgbotapi.NewKeyboardButton(BtnChatNow),
	))

	// Row 3 - Coins - Leaderboard - Help
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(BtnCoins),
		tgbotapi.NewKeyboardButton(BtnLeaderboard),
		tgbotapi.NewKeyboardButton(BtnHelp),
	))

	// Row 4 - Profile - Friends
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(BtnProfile),
		tgbotapi.NewKeyboardButton(BtnFriends),
	))

	// Row 5 - Referral
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(BtnReferral),
	))

	return tgbotapi.NewReplyKeyboard(rows...)
}

// GenderKeyboard creates inline gender selection keyboard for registration
func GenderKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🙍‍♂️ پسر", "reg_gender_male"),
			tgbotapi.NewInlineKeyboardButtonData("🙍‍♀️ دختر", "reg_gender_female"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "reg_cancel"),
		),
	)
}

// AgeSelectionKeyboard creates inline age selection keyboard (13-100)
func AgeSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var currentRow []tgbotapi.InlineKeyboardButton

	for age := 13; age <= 100; age++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(uintptrToString(age), "reg_age_"+uintptrToString(age))
		currentRow = append(currentRow, btn)
		if len(currentRow) == 5 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "reg_cancel"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func uintptrToString(i int) string {
	return strconv.Itoa(i)
}

// ProvinceInlineKeyboard creates inline province selection keyboard
func ProvinceInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
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
	var currentRow []tgbotapi.InlineKeyboardButton

	for _, p := range provinces {
		currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(p, "reg_province_"+p))
		if len(currentRow) == 2 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ProvinceKeyboard creates an inline keyboard with Iranian provinces
func ProvinceKeyboard() tgbotapi.InlineKeyboardMarkup {
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
	var currentRow []tgbotapi.InlineKeyboardButton

	for _, p := range provinces {
		currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(p, "reg_province_"+p))
		if len(currentRow) == 2 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "reg_cancel"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PhotoSelectionKeyboard creates inline skip keyboard for photo step
func PhotoSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏩ فعلاً رد کن", "reg_photo_skip"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "reg_cancel"),
		),
	)
}

// CancelInlineKeyboard creates a simple cancel inline keyboard
func CancelInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "reg_cancel"),
		),
	)
}

// PhotoSkipKeyboard creates inline skip keyboard for photo step
func PhotoSkipKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏩ فعلاً رد کن", "reg_photo_skip"),
		),
	)
}

// SearchGenderFilterKeyboard creates search gender filter inline keyboard
func SearchGenderFilterKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnMale, "btn:"+BtnMale),
			tgbotapi.NewInlineKeyboardButtonData(BtnFemale, "btn:"+BtnFemale),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnAny, "btn:"+BtnAny),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
		),
	)
}

// CancelKeyboard creates a simple cancel inline keyboard
func CancelKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
		),
	)
}

// EndChatKeyboard creates end chat inline keyboard
func EndChatKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnEndChat, "btn:"+BtnEndChat),
		),
	)
}

// ConfirmKeyboard creates confirm/cancel keyboard
func ConfirmKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnAccept, "confirm_yes"),
			tgbotapi.NewInlineKeyboardButtonData(BtnReject, "confirm_no"),
		),
	)
}

// SkipKeyboard creates skip/cancel inline keyboard
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

// RemoveKeyboard removes the keyboard
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

// PlayModeKeyboard creates the Game Center inline keyboard
func PlayModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnQuickMatch, "btn:"+BtnQuickMatch),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnQuiz, "btn:"+BtnQuiz),
			tgbotapi.NewInlineKeyboardButtonData(BtnTruthDare, "btn:"+BtnTruthDare),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// GameMatchModeKeyboard creates inline keyboard for specific game match modes
func GameMatchModeKeyboard(isQuiz bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnOneVsOneRandom, "btn:"+BtnOneVsOneRandom),
		tgbotapi.NewInlineKeyboardButtonData(BtnPlayWithFriends, "btn:"+BtnPlayWithFriends),
	))
	if isQuiz {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBetting, "btn:"+BtnBetting),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// TruthDareRoomKeyboard creates inline keyboard for truth or dare room selection
func TruthDareRoomKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCreateRoom, "btn:"+BtnCreateRoom),
			tgbotapi.NewInlineKeyboardButtonData(BtnSearchRoom, "btn:"+BtnSearchRoom),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnRandomMatch, "btn:"+BtnRandomMatch),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// LeaderboardKeyboard creates inline keyboard for leaderboard filters
func LeaderboardKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnTodayTop, "btn:"+BtnTodayTop),
			tgbotapi.NewInlineKeyboardButtonData(BtnWeekTop, "btn:"+BtnWeekTop),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnAllTimeTop, "btn:"+BtnAllTimeTop),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnQuizKings, "btn:"+BtnQuizKings),
			tgbotapi.NewInlineKeyboardButtonData(BtnBraveOnes, "btn:"+BtnBraveOnes),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnMyLeague, "btn:"+BtnMyLeague),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// SocialHubKeyboard creates inline keyboard for social management
func SocialHubKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterProvince, "btn:"+BtnFilterProvince),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterAge, "btn:"+BtnFilterAge),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNew, "btn:"+BtnFilterNew),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNoChat, "btn:"+BtnFilterNoChat),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnInviteLink, "btn:"+BtnInviteLink),
			tgbotapi.NewInlineKeyboardButtonData(BtnFriendList, "btn:"+BtnFriendList),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFriendRequests, "btn:"+BtnFriendRequests),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// SettingsHelpKeyboard creates inline keyboard for settings and help
func SettingsHelpKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnNotifications, "btn:"+BtnNotifications),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnTutorials, "btn:"+BtnTutorials),
			tgbotapi.NewInlineKeyboardButtonData(BtnSupport, "btn:"+BtnSupport),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnRules, "btn:"+BtnRules),
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// SearchModeKeyboard creates inline keyboard for selecting search mode
func SearchModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		// Row 1 - Random Search
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnRandomMatch, "btn:"+BtnRandomMatch),
		),
		// Row 2 - Girl - Boy
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFemale, "btn:"+BtnFemale),
			tgbotapi.NewInlineKeyboardButtonData(BtnMale, "btn:"+BtnMale),
		),
		// Row 3 - Age - Province - Near Me
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterAge, "btn:"+BtnFilterAge),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterProvince, "btn:"+BtnFilterProvince),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNearMe, "btn:"+BtnFilterNearMe),
		),
		// Row 4 - New Users - No Chat
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNew, "btn:"+BtnFilterNew),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNoChat, "btn:"+BtnFilterNoChat),
		),
		// Row 5 - Advanced Search
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterAdvanced, "btn:"+BtnFilterAdvanced),
		),
	)
}

// UserListFilterKeyboard creates inline keyboard for user list filters
func UserListFilterKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterRecent, "btn:"+BtnFilterRecent),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterProvince, "btn:"+BtnFilterProvince),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterAge, "btn:"+BtnFilterAge),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNew, "btn:"+BtnFilterNew),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterNoChat, "btn:"+BtnFilterNoChat),
			tgbotapi.NewInlineKeyboardButtonData(BtnFilterAdvanced, "btn:"+BtnFilterAdvanced),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
		),
	)
}

// RequestContactKeyboard creates keyboard for requesting contact
func RequestContactKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 ارسال شماره تماس"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnCancel),
		),
	)
}

// ProfileActionsKeyboard creates inline keyboard for profile actions
func ProfileActionsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ ویرایش پروفایل", "edit_profile"),
		),
	)
}

// EditProfileFieldsKeyboard creates inline keyboard for selecting field to edit
func EditProfileFieldsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 نام", "edit_field_name"),
			tgbotapi.NewInlineKeyboardButtonData("🎂 سن", "edit_field_age"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📍 استان", "edit_field_province"),
			tgbotapi.NewInlineKeyboardButtonData("🖼 عکس", "edit_field_photo"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 بیوگرافی", "edit_field_bio"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", "edit_profile_back"),
		),
	)
}

// VillageHubKeyboard creates inline keyboard for Village management
func VillageHubKeyboard(hasVillage bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if !hasVillage {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnCreateVillage, "btn:"+BtnCreateVillage),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnVillageChat, "btn:"+BtnVillageChat),
			tgbotapi.NewInlineKeyboardButtonData(BtnVillageGame, "btn:"+BtnVillageGame),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnInviteToVillage, "btn:"+BtnInviteToVillage),
			tgbotapi.NewInlineKeyboardButtonData(BtnLeaveVillage, "btn:"+BtnLeaveVillage),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnVillageLeaderboard, "btn:"+BtnVillageLeaderboard),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "btn:"+BtnBack),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
