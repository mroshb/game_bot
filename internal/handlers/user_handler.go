package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/internal/security"
	apperrors "github.com/mroshb/game_bot/pkg/errors"
	"github.com/mroshb/game_bot/pkg/logger"
)

// Bot interface to avoid circular dependency
type BotInterface interface {
	SendMessage(chatID int64, text string, keyboard interface{}) int
	DeleteMessage(chatID int64, messageID int)
	EditMessage(chatID int64, messageID int, text string, keyboard interface{})
	SendPhoto(chatID int64, photoID string, caption string, keyboard interface{}) int
	SendMainMenu(chatID int64, isAdmin bool)
	GetMainMenuKeyboard(isAdmin bool) interface{}
	GetGenderKeyboard() interface{}
	GetAgeSelectionKeyboard() interface{}
	GetProvinceKeyboard() interface{}
	GetPhotoSelectionKeyboard() interface{}
	GetPhotoSkipKeyboard() interface{}
	GetCancelInlineKeyboard() interface{}
	GetEditProfileFieldsKeyboard() interface{}
	GetConfig() interface{}
	GetAPI() interface{}
	AnswerCallbackQuery(queryID string, text string, showAlert bool)
	GetVillageHubKeyboard(hasVillage bool) interface{}
	GetCancelKeyboard() interface{}
}

type UserSession struct {
	State string
	Data  map[string]interface{}
}

const (
	StateRegisterName     = "register_name"
	StateRegisterGender   = "register_gender"
	StateRegisterAge      = "register_age"
	StateRegisterProvince = "register_province"
	StateRegisterCity     = "register_city"
	StateRegisterPhoto    = "register_photo"

	StateEditName     = "edit_name"
	StateEditAge      = "edit_age"
	StateEditProvince = "edit_province"
	StateEditCity     = "edit_city"
	StateEditPhoto    = "edit_photo"
	StateEditBio      = "edit_bio"

	// Search States
	StateSearchGender = "search_gender"
	StateSearchAge    = "search_age"
	StateSearchCity   = "search_city"

	// Purchase States
	StateAwaitingReceipt = "awaiting_receipt"
)

func (h *HandlerManager) HandleRegistration(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID

	switch session.State {
	case StateRegisterName:
		h.handleRegisterName(message, session, bot)

	case StateRegisterProvince:
		// Users should use the inline keyboard. If they type, remind them.
		bot.SendMessage(userID, "📍 لطفا شهرت رو از لیست دکمه‌های شیشه‌ای بالا انتخاب کن!", nil)

	case StateRegisterPhoto:
		h.handleRegisterPhoto(message, session, bot)

	default:
		logger.Warn("Unknown registration state", "state", session.State, "user_id", userID)
	}
}

func (h *HandlerManager) HandleRegistrationCallback(query *tgbotapi.CallbackQuery, session *UserSession, bot BotInterface) {
	userID := query.From.ID
	data := query.Data

	if data == "reg_cancel" {
		// Delete bot message
		if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
			bot.DeleteMessage(userID, lastMsgID)
		}
		// Clear session
		session.State = ""
		session.Data = make(map[string]interface{})
		bot.SendMessage(userID, "❌ ثبت‌نام لغو شد. هر وقت دوست داشتی دوباره /start بزن!", nil)
		return
	}

	switch session.State {
	case StateRegisterGender:
		if strings.HasPrefix(data, "reg_gender_") {
			gender := strings.TrimPrefix(data, "reg_gender_")

			// Delete previous bot message
			if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
				bot.DeleteMessage(userID, lastMsgID)
			}

			session.Data["gender"] = gender
			session.State = StateRegisterName
			msgID := bot.SendMessage(userID, "خوشبختم! توی بازی چی صدات کنیم؟ (یک اسم کوتاه و خفن بنویس)", bot.GetCancelInlineKeyboard())
			session.Data["last_bot_msg_id"] = msgID
		}

	case StateRegisterAge:
		if strings.HasPrefix(data, "reg_age_") {
			ageStr := strings.TrimPrefix(data, "reg_age_")
			age, _ := strconv.Atoi(ageStr)

			// Delete previous bot message
			if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
				bot.DeleteMessage(userID, lastMsgID)
			}

			session.Data["age"] = age

			// Get name to personalize the next message
			name := "?"
			if n, ok := session.Data["name"].(string); ok {
				name = n
			}

			session.State = StateRegisterProvince
			msgID := bot.SendMessage(userID, fmt.Sprintf("%s عزیز، از کدوم شهری؟ 🌍 (اینطوری میتونی همشهریهات رو توی بازی پیدا کنی)", name), bot.GetProvinceKeyboard())
			session.Data["last_bot_msg_id"] = msgID
		}

	case StateRegisterProvince:
		if strings.HasPrefix(data, "reg_province_") {
			province := strings.TrimPrefix(data, "reg_province_")

			// Delete previous bot message
			if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
				bot.DeleteMessage(userID, lastMsgID)
			}

			session.Data["province"] = province
			session.State = StateRegisterPhoto
			msgID := bot.SendMessage(userID, "آخریش! یه عکس برامون بفرست تا بقیه بشناسنت. 📸 (اگر نفرستی، ما یه آواتار جالب برات میذاریم)", bot.GetPhotoSelectionKeyboard())
			session.Data["last_bot_msg_id"] = msgID
		}

	case StateRegisterPhoto:
		if data == "reg_photo_skip" {
			h.completeRegistration(userID, session, bot)
		}
	}
}

func (h *HandlerManager) handleRegisterName(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	name := security.SanitizeString(message.Text)

	if name == "" || len(name) < 2 {
		bot.SendMessage(userID, "❌ نام باید حداقل 2 حرف باشه! دوباره وارد کن:", nil)
		return
	}

	if len([]rune(name)) > 12 {
		bot.SendMessage(userID, "خیلی طولانیه! لطفاً زیر ۱۲ حرف باشه.", nil)
		return
	}

	// Delete previous bot message
	if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
		bot.DeleteMessage(userID, lastMsgID)
	}
	// Delete user message
	bot.DeleteMessage(userID, message.MessageID)

	session.Data["name"] = name
	session.State = StateRegisterAge
	msgID := bot.SendMessage(userID, fmt.Sprintf("%s عزیز، چند سالته؟ (برای پیدا کردن هم‌سن‌هات)", name), bot.GetAgeSelectionKeyboard())
	session.Data["last_bot_msg_id"] = msgID
}

func (h *HandlerManager) handleRegisterPhoto(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	// Clean up previous bot prompt
	if lastMsgID, ok := session.Data["last_bot_msg_id"].(int); ok {
		bot.DeleteMessage(userID, lastMsgID)
	}
	// Delete user message
	bot.DeleteMessage(userID, message.MessageID)

	if text == "⏩ فعلاً رد کن" {
		h.completeRegistration(userID, session, bot)
		return
	}

	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		session.Data["photo"] = photo.FileID
		h.completeRegistration(userID, session, bot)
		return
	}

	bot.SendMessage(userID, "❌ لطفاً عکس بفرست یا از دکمه‌ها استفاده کن!", bot.GetPhotoSelectionKeyboard())
}

func (h *HandlerManager) completeRegistration(userID int64, session *UserSession, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	isNew := false
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok && appErr.Code == apperrors.ErrCodeNotFound {
			isNew = true
			user = &models.User{
				TelegramID:  userID,
				CoinBalance: 100, // Gift bonus
				Status:      models.UserStatusOffline,
				Age:         18,  // Default
				City:        "?", // Legacy
				Biography:   "؟",
			}
		} else {
			logger.Error("Database error during registration check", "error", err, "user_id", userID)
			bot.SendMessage(userID, "❌ خطا در سیستم! لطفاً کمی بعد دوباره تلاش کنید.", nil)
			return
		}
	}

	// Update/Set fields from session
	if name, ok := session.Data["name"].(string); ok {
		user.FullName = name
	}
	if gender, ok := session.Data["gender"].(string); ok {
		user.Gender = gender
	}
	if age, ok := session.Data["age"].(int); ok {
		user.Age = age
	}
	if province, ok := session.Data["province"].(string); ok {
		user.Province = province
	}

	// Set photo if provided, otherwise use default avatar
	if photoID, ok := session.Data["photo"].(string); ok {
		user.ProfilePhoto = photoID
	} else if isNew {
		if user.Gender == models.GenderMale {
			user.ProfilePhoto = models.DefaultAvatarMale
		} else {
			user.ProfilePhoto = models.DefaultAvatarFemale
		}
	}

	// Save to database
	var saveErr error
	if isNew {
		saveErr = h.UserRepo.CreateUser(user)
	} else {
		saveErr = h.UserRepo.UpdateUser(user)
	}

	if saveErr != nil {
		logger.Error("Failed to save user during registration", "error", saveErr, "user_id", userID)
		bot.SendMessage(userID, "❌ خطا در ثبت نام! لطفاً دوباره تلاش کن.", nil)
		return
	}

	// Handle Referral Reward
	if referrerID, ok := session.Data["referrer_id"].(uint); ok && referrerID > 0 {
		user.ReferrerID = referrerID
		h.UserRepo.UpdateUser(user)
		h.CoinRepo.AddCoins(referrerID, 50, models.TxTypeReferralReward, "پاداش دعوت")
		h.CoinRepo.AddCoins(user.ID, 50, models.TxTypeReferralReward, "پاداش ورود با دعوت")
	}

	// Clear session
	session.State = ""
	session.Data = make(map[string]interface{})

	// Success message
	bot.SendMessage(userID, "ثبتنامت تکمیل شد! 🎉 به عنوان هدیه ورود، ۱۰۰ سکه به کیفت اضافه شد. حالا بزن بریم!", nil)
	bot.SendMainMenu(userID, user.TelegramID == h.Config.SuperAdminTgID)
}

func (h *HandlerManager) ShowProfile(userID int64, user *models.User, bot BotInterface) {
	if user == nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	// Calculate percentage for display
	requiredXP := user.GetXPRequired()
	xpPercentage := int(float64(user.XP) / float64(requiredXP) * 100)
	if xpPercentage > 100 {
		xpPercentage = 100
	}

	// Format Inventory Items
	inventoryItems := "خالی"
	if user.ItemsInventory != "" && user.ItemsInventory != "{}" {
		var itemsMap map[string]int
		if err := json.Unmarshal([]byte(user.ItemsInventory), &itemsMap); err == nil {
			var formattedParts []string
			itemLabels := map[string]string{
				"shield": "🛡 سپر",
				"swap":   "🔄 تعویض",
				"5050":   "💣 حذف ۲",
				"freeze": "⏳ انجماد",
			}
			for key, val := range itemsMap {
				if label, ok := itemLabels[key]; ok {
					formattedParts = append(formattedParts, fmt.Sprintf("%s: %d", label, val))
				}
			}
			if len(formattedParts) > 0 {
				inventoryItems = strings.Join(formattedParts, " | ")
			}
		}
	}

	// Member since
	joinDate := user.CreatedAt.Format("2006/01/02")

	// Profile Card Format
	profileText := fmt.Sprintf(`👤 پروفایل کاربری: %s
➖➖➖➖➖➖➖➖
🏅 سطح: [%d] (رتبه: %s)
📈 تجربه: %d/%d XP
%s %d%%

💰 دارایی: [%d] سکه
💎 الماس: [%d]

📊 آمار عملکرد:
🏆 برد: %d | ❌ باخت: %d | 🤝 مساوی: %d
📍 شهر: [%s]
📅 عضویت: [%s]
➖➖➖➖➖➖➖➖
🎒 موجودی آیتمها:
%s`,
		user.FullName,
		user.Level,
		user.GetLevelTitle(),
		user.XP,
		requiredXP,
		user.GetXPBar(),
		xpPercentage,
		user.CoinBalance,
		user.Diamonds,
		user.Wins,
		user.Losses,
		user.Draws,
		user.Province,
		joinDate,
		inventoryItems,
	)

	// Detect if viewing own profile
	isSelf := user.TelegramID == userID

	var keyboard tgbotapi.InlineKeyboardMarkup
	if isSelf {

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnEditProfile, "edit_profile"),
				tgbotapi.NewInlineKeyboardButtonData(BtnLikes, "btn:"+BtnLikes),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnEditLocation, "btn:"+BtnEditLocation),
				tgbotapi.NewInlineKeyboardButtonData(BtnBlocks, "btn:"+BtnBlocks),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnSettings, "btn:"+BtnSettings),
				tgbotapi.NewInlineKeyboardButtonData(BtnGameHistory, "game_history"),
			),
		)
	} else {
		// Viewing someone else's profile
		currentUser, _ := h.UserRepo.GetUserByTelegramID(userID)
		var rows [][]tgbotapi.InlineKeyboardButton

		if currentUser != nil {
			// Like button
			hasLiked, _ := h.UserRepo.HasLiked(currentUser.ID, user.ID)
			if !hasLiked {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❤️ لایک", fmt.Sprintf("like_%d", user.ID)),
				))
			}

			// Friend button
			areFriends, _ := h.FriendRepo.AreFriends(currentUser.ID, user.ID)
			if !areFriends {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("👥 درخواست دوستی", fmt.Sprintf("friend_add_%d", user.ID)),
				))
			}
		}
		keyboard = tgbotapi.NewInlineKeyboardMarkup(rows...)
	}

	photoToUse := user.ProfilePhoto
	if user.CustomAvatarID != "" {
		photoToUse = user.CustomAvatarID
	}

	if photoToUse != "" {
		bot.SendPhoto(userID, photoToUse, profileText, keyboard)
	} else {
		bot.SendMessage(userID, profileText, keyboard)
	}
}

func (h *HandlerManager) HandleDailyBonus(userID int64, queryID string, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		if queryID != "" {
			bot.AnswerCallbackQuery(queryID, "❌ خطا در سیستم!", true)
		} else {
			bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		}
		return
	}

	now := time.Now()
	// Check if already claimed today
	if !user.LastDailyBonus.IsZero() {
		if user.LastDailyBonus.Year() == now.Year() && user.LastDailyBonus.YearDay() == now.YearDay() {
			nextAvailable := user.LastDailyBonus.Add(24 * time.Hour)
			if now.Before(nextAvailable) {
				diff := nextAvailable.Sub(now)
				hours := int(diff.Hours())
				minutes := int(diff.Minutes()) % 60
				msg := fmt.Sprintf("⏳ هنوز وقتش نشده! %d ساعت و %d دقیقه دیگه بیا.", hours, minutes)

				if queryID != "" {
					bot.AnswerCallbackQuery(queryID, msg, true)
				} else {
					bot.SendMessage(userID, msg, nil)
				}
				return
			}
		}
	}

	// Calculate streak
	isStreak := false
	if !user.LastDailyBonus.IsZero() {
		// If last claim was yesterday (YearDay check handles Jan 1st? Not really if year changed, but simple for now)
		// Better: diff < 48 hours
		if now.Sub(user.LastDailyBonus) < 48*time.Hour {
			isStreak = true
		}
	}

	if isStreak {
		user.DailyBonusStreak++
	} else {
		user.DailyBonusStreak = 1
	}

	bonusAmount := int64(40 + (user.DailyBonusStreak * 10)) // Day 1: 50, Day 2: 60...
	if bonusAmount > 200 {
		bonusAmount = 200 // Cap bonus
	}

	user.CoinBalance += bonusAmount
	user.LastDailyBonus = now
	h.UserRepo.UpdateUser(user)

	// Record transaction
	h.CoinRepo.AddCoins(user.ID, bonusAmount, models.TxTypeDailyBonus, fmt.Sprintf("جایزه روزانه (روز %d)", user.DailyBonusStreak))

	bot.SendMessage(userID, fmt.Sprintf("🎁 تبریک! %d سکه امروزت رو گرفتی. فردا بیا تا %d تا بگیری!", bonusAmount, bonusAmount+10), nil)
	if queryID != "" {
		bot.AnswerCallbackQuery(queryID, "✅ جایزه با موفقیت دریافت شد!", false)
	}
}

func (h *HandlerManager) ShowLeaderboard(userID int64, bot BotInterface) {
	users, err := h.UserRepo.GetLeaderboard("all", "all", 10)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت جدول برترینها!", nil)
		return
	}

	leaderboardMsg := "<b>🏆 جدول برترینها (۱۰ نفر اول):</b>\n\n"
	for i, u := range users {
		medal := ""
		switch i {
		case 0:
			medal = "🥇 "
		case 1:
			medal = "🥈 "
		case 2:
			medal = "🥉 "
		default:
			medal = fmt.Sprintf("%d. ", i+1)
		}
		leaderboardMsg += fmt.Sprintf("%s %s - 💰 %d سکه\n", medal, u.FullName, u.CoinBalance)
	}

	userRank, _ := h.UserRepo.GetUserRank(uint(userID))
	leaderboardMsg += fmt.Sprintf("\n--------------------\n🏅 رتبه شما: <b>%d</b>", userRank)

	// Filters keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 امروز", "lb_today"),
			tgbotapi.NewInlineKeyboardButtonData("🗓 هفته", "lb_week"),
			tgbotapi.NewInlineKeyboardButtonData("♾️ کل دوران", "lb_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 سلاطین کوییز", "lb_quiz"),
			tgbotapi.NewInlineKeyboardButtonData("🔥 شجاعترینها", "lb_brave"),
		),
	)

	// Note: For EditMessage we would need messageID.
	// Since we don't have it easily here without changing interface,
	// let's stick to SendMessage but the groundwork is there.
	bot.SendMessage(userID, leaderboardMsg, keyboard)
}

// HandleEditProfile handles the edit profile button click
func (h *HandlerManager) HandleEditProfile(userID int64, bot BotInterface) {
	bot.SendMessage(userID, "✏️ کدام قسمت را می‌خواهید ویرایش کنید؟", bot.GetEditProfileFieldsKeyboard())
}

func (h *HandlerManager) HandleEditFieldSelection(userID int64, field string, session *UserSession, bot BotInterface) {
	var msg string

	switch field {
	case "name":
		session.State = StateEditName
		msg = "📝 نام جدید را وارد کنید:"
	case "age":
		session.State = StateEditAge
		msg = "🎂 سن جدید را وارد کنید (عدد):"
	case "province":
		session.State = StateEditProvince
		msg = "📍 استان جدید را انتخاب کنید:"
		bot.SendMessage(userID, msg, ProvinceKeyboard())
		return
	case "photo":
		session.State = StateEditPhoto
		msg = "🖼 عکس پروفایل جدید را بفرستید:"
	case "bio":
		session.State = StateEditBio
		msg = "📝 بیوگرافی جدید خود را بنویسید:"
	}

	bot.SendMessage(userID, msg, tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel)),
	))
}

func (h *HandlerManager) HandleEditProfileInput(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	text := message.Text

	if text == BtnCancel {
		session.State = ""
		session.Data = make(map[string]interface{})
		bot.SendMessage(userID, "❌ ویرایش لغو شد.", nil)
		// Show profile again
		user, _ := h.UserRepo.GetUserByTelegramID(userID)
		h.ShowProfile(userID, user, bot)
		return
	}

	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات کاربر!", nil)
		session.State = ""
		return
	}

	switch session.State {
	case StateEditName:
		name := security.SanitizeString(text)
		if len(name) < 2 {
			bot.SendMessage(userID, "❌ نام کوتاه است!", nil)
			return
		}
		user.FullName = name

	case StateEditAge:
		age, err := strconv.Atoi(text)
		if err != nil || !security.ValidateAge(age) {
			bot.SendMessage(userID, "❌ سن نامعتبر است!", nil)
			return
		}
		user.Age = age

	case StateEditProvince:
		// reuse province validation logic
		provinces := []string{
			"تهران", "خوزستان", "بوشهر", "اصفهان",
			"خراسان رضوی", "فارس", "آذربایجان شرقی", "مازندران",
			"کرمان", "البرز", "گیلان", "کهگیلویه و بویراحمد",
			"آذربایجان غربی", "هرمزگان", "مرکزی", "یزد",
			"فرامنطقه ای", "کرمانشاه", "قزوین", "سیستان و بلوچستان",
			"همدان", "ایلام", "گلستان", "لرستان",
			"زنجان", "اردبیل", "قم", "کردستان",
			"سمنان", "چهارمحال و بختیاری", "خراسان شمالی", "خراسان جنوبی",
			"کرج", "خارج از ایران",
		}
		valid := false
		for _, p := range provinces {
			if text == p {
				valid = true
				break
			}
		}
		if !valid {
			bot.SendMessage(userID, "❌ استان نامعتبر است. از لیست انتخاب کنید.", nil)
			return
		}
		user.Province = text

		user.Province = text

	case StateEditPhoto:
		if len(message.Photo) > 0 {
			photo := message.Photo[len(message.Photo)-1]
			user.ProfilePhoto = photo.FileID
		} else {
			bot.SendMessage(userID, "❌ لطفاً عکس ارسال کنید!", nil)
			return
		}

	case StateEditBio:
		bio := security.SanitizeString(text)
		if len(bio) > 200 {
			bot.SendMessage(userID, "❌ بیوگرافی نباید بیشتر از 200 کاراکتر باشد!", nil)
			return
		}
		user.Biography = bio
	}

	// Save update
	if err := h.UserRepo.UpdateUser(user); err != nil {
		logger.Error("Failed to update user", "error", err)
		bot.SendMessage(userID, "❌ خطا در ذخیره اطلاعات!", nil)
	} else {
		// Micro-interaction: show saving animation
		msgID := bot.SendMessage(userID, "⏳ در حال ذخیره...", nil)
		go func() {
			time.Sleep(1 * time.Second)
			bot.DeleteMessage(userID, msgID)
			bot.SendMessage(userID, "✅ اطلاعاتت با موفقیت آپدیت شد!", tgbotapi.NewRemoveKeyboard(true))
			h.ShowProfile(userID, user, bot)
		}()
	}

	session.State = ""
}

// HandleLike handles user like action
func (h *HandlerManager) HandleLike(likerTgID int64, likedUserID uint, bot BotInterface) {
	liker, err := h.UserRepo.GetUserByTelegramID(likerTgID)
	if err != nil {
		bot.SendMessage(likerTgID, "❌ خطا در دریافت اطلاعات شما!", nil)
		return
	}

	if liker.ID == likedUserID {
		bot.SendMessage(likerTgID, "😊 نمی‌تونی خودت رو لایک کنی!", nil)
		return
	}

	alreadyLiked, _ := h.UserRepo.HasLiked(liker.ID, likedUserID)
	if alreadyLiked {
		bot.SendMessage(likerTgID, "❤️ شما قبلاً این کاربر را لایک کرده‌اید.", nil)
		return
	}

	err = h.UserRepo.AddLike(liker.ID, likedUserID)
	if err != nil {
		bot.SendMessage(likerTgID, "❌ خطا در ثبت لایک!", nil)
		return
	}

	// Get updated target user
	targetUser, _ := h.UserRepo.GetUserByID(likedUserID)

	// Notify target if they are active? (Optional but nice)
	bot.SendMessage(likerTgID, fmt.Sprintf("❤️ شما %s را لایک کردید!", targetUser.FullName), nil)

	// Also notify the liked user
	bot.SendMessage(targetUser.TelegramID, fmt.Sprintf("🎉 %s شما را لایک کرد!", liker.FullName), nil)
}

func (h *HandlerManager) SearchUserByPublicID(searcherID int64, publicID string, bot BotInterface) {
	// Get target user
	targetUser, err := h.UserRepo.GetUserByPublicID(publicID)
	if err != nil {
		bot.SendMessage(searcherID, "❌ کاربر با این ID پیدا نشد!", nil)
		return
	}

	h.ShowProfile(searcherID, targetUser, bot)
}

func (h *HandlerManager) ShowCoins(userID int64, user *models.User, bot BotInterface) {
	if user == nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	// Get transaction history
	transactions, err := h.CoinRepo.GetTransactionHistory(user.ID, 10)
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err)
		transactions = []models.CoinTransaction{}
	}

	message := fmt.Sprintf("💰 موجودی شما: %d سکه\n\n", user.CoinBalance)

	if len(transactions) > 0 {
		message += "📊 آخرین تراکنش‌ها:\n\n"
		for _, tx := range transactions {
			sign := "+"
			if tx.Amount < 0 {
				sign = ""
			}
			message += fmt.Sprintf("%s%d سکه - %s\n", sign, tx.Amount, tx.Description)
		}
	} else {
		message += "📊 هنوز تراکنشی نداری!"
	}

	bot.SendMessage(userID, message, tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBuyCoins, "buy_coins"),
		),
	))
}

func (h *HandlerManager) HandleBuyCoins(userID int64, messageID int, bot BotInterface) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnIHavePaid, "paid_coins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت به پروفایل", "edit_profile_back"),
		),
	)

	if messageID != 0 {
		bot.EditMessage(userID, messageID, MsgCoinPurchasePlans, keyboard)
	} else {
		bot.SendMessage(userID, MsgCoinPurchasePlans, keyboard)
	}
}

func (h *HandlerManager) HandlePaid(userID int64, session *UserSession, bot BotInterface) {
	session.State = StateAwaitingReceipt
	bot.SendMessage(userID, MsgRequestReceipt, nil)
}

func (h *HandlerManager) HandlePurchaseReceipt(userID int64, message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	if message.Photo == nil {
		bot.SendMessage(userID, "❌ خطا! لطفاً رسید خود را به صورت عکس ارسال کنید.", nil)
		return
	}

	// Notify admin
	adminMsg := fmt.Sprintf("💰 رسید پرداخت جدید دریافت شد!\n\n👤 کاربر: %s (%d)\n🆔 آیدی عمومی: %s",
		message.From.FirstName, userID, session.Data["public_id"])

	bot.SendPhoto(h.Config.SuperAdminTgID, message.Photo[len(message.Photo)-1].FileID, adminMsg, nil)

	session.State = ""
	bot.SendMessage(userID, MsgPurchasePending, nil)
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
		currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(p, "btn:"+p))
		if len(currentRow) == 2 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
			currentRow = []tgbotapi.InlineKeyboardButton{}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(currentRow...))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "btn:"+BtnCancel),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ShowFriendRequests displays pending friend requests
func (h *HandlerManager) ShowFriendRequests(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	requests, err := h.FriendRepo.GetPendingRequests(user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت درخواستها!", nil)
		return
	}

	if len(requests) == 0 {
		bot.SendMessage(userID, "📥 هنوز هیچ درخواست دوستی جدیدی نداری.", nil)
		return
	}

	for _, req := range requests {
		requester, err := h.UserRepo.GetUserByID(req.RequesterID)
		if err != nil {
			continue
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ قبول", fmt.Sprintf("friend_accept_%d", requester.ID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ رد", fmt.Sprintf("friend_reject_%d", requester.ID)),
			),
		)
		bot.SendMessage(userID, fmt.Sprintf("👋 %s بهت درخواست دوستی داده!", requester.FullName), keyboard)
	}
}
func (h *HandlerManager) ListNearbyUsers(userID int64, lat, lon float64, bot BotInterface) {
	// Update user location
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}
	h.UserRepo.UpdateLocation(user.ID, lat, lon)

	// Get nearby users
	users, err := h.UserRepo.FindNearbyUsers(user.ID, lat, lon, 20)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در یافتن کاربران نزدیک!", nil)
		return
	}

	if len(users) == 0 {
		bot.SendMessage(userID, "📍 متأسفانه کاربری در نزدیکی شما پیدا نشد.", nil)
		return
	}

	message := "📍 کاربران نزدیک شما:\n\n"
	for _, u := range users {
		dist := u.Distance
		message += fmt.Sprintf("👤 %s (%d ساله) - 📏 %.1f کیلومتر\n/user_%s\n\n", u.FullName, u.Age, dist, u.PublicID)
	}

	bot.SendMessage(userID, message, nil)
}

func (h *HandlerManager) HandleFilterRecent(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	users, err := h.UserRepo.FindRecentChatUsers(user.ID, 10)
	h.sendUserList(userID, "🕒 چت های اخیر شما:", users, err, bot)
}

func (h *HandlerManager) HandleFilterProvince(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	users, err := h.UserRepo.FindUsersByProvince(user.ID, user.Province, 10)
	h.sendUserList(userID, "📍 کاربران هم استانی شما:", users, err, bot)
}

func (h *HandlerManager) HandleFilterAge(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	users, err := h.UserRepo.FindUsersByAge(user.ID, user.Age, 10)
	h.sendUserList(userID, "🎂 کاربران هم سن شما:", users, err, bot)
}

func (h *HandlerManager) HandleFilterNew(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	users, err := h.UserRepo.FindNewUsers(user.ID, 10)
	h.sendUserList(userID, "👶 کاربران جدید:", users, err, bot)
}

func (h *HandlerManager) HandleFilterNoChat(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	users, err := h.UserRepo.FindUsersWithNoChats(user.ID, 10)
	h.sendUserList(userID, "😶 کاربران بدون چت:", users, err, bot)
}

func (h *HandlerManager) ShowInventory(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	inventoryText := "🎒 کوله‌پشتی شما فعلاً خالی است!\n\nبا شرکت در بازی‌ها و چالش‌ها، آیتم‌های مختلفی به دست بیار که توی بازی بهت کمک می‌کنن."
	if user.ItemsInventory != "" && user.ItemsInventory != "{}" {
		// Reuse formatting logic or similar
		inventoryText = "🎒 موجودی آیتم‌های شما:\n\n"

		// Map items to descriptions
		itemDescs := map[string]string{
			"shield": "🛡 سپر فرار: برای رد کردن چالش‌های سخت بدون کسر امتیاز.",
			"swap":   "🔄 کارت تعویض: تعویض سوال یا چالش فعلی.",
			"5050":   "💣 حذف دو گزینه: مخصوص کوییز برای حذف گزینه‌های غلط.",
			"freeze": "⏳ زمان اضافه: ۱۰ ثانیه وقت بیشتر برای پاسخ‌دهی.",
		}

		items := strings.ReplaceAll(user.ItemsInventory, "{", "")
		items = strings.ReplaceAll(items, "}", "")
		items = strings.ReplaceAll(items, "\"", "")
		parts := strings.Split(items, ",")

		found := false
		for _, p := range parts {
			kv := strings.Split(p, ":")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				if desc, ok := itemDescs[key]; ok {
					inventoryText += fmt.Sprintf("%s\nتعداد: %s\n\n", desc, val)
					found = true
				}
			}
		}
		if !found {
			inventoryText = "🎒 کوله‌پشتی شما فعلاً خالی است!"
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛍 خرید آیتم‌های بیشتر", "shop_items"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت به پروفایل", "edit_profile_back"),
		),
	)

	bot.SendMessage(userID, inventoryText, keyboard)
}

func (h *HandlerManager) ShowGameHistory(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت اطلاعات!", nil)
		return
	}

	recentGames, err := h.GameRepo.GetRecentGames(user.ID, 10)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت تاریخچه!", nil)
		return
	}

	historyMsg := "📜 آخرین نبردها:\n\n"
	if len(recentGames) == 0 {
		historyMsg += "هنوز بازی ثبت شده‌ای نداری! برو و اولین بازیت رو شروع کن. 🚀"
	} else {
		for i, g := range recentGames {
			statusIcon := "🤝"
			resultText := "مساوی"

			// Simple logic: we need to know what the max score was or who won
			// For now, let's assume we store result in GameSession or we compute it
			// Since we don't have result field in GameParticipant yet, we check score?
			// This is a placeholder for real logic
			if g.Score > 0 {
				statusIcon = "✅"
				resultText = "برد"
			} else if g.Score < 0 {
				statusIcon = "❌"
				resultText = "باخت"
			}

			gameType := "بازی"
			switch g.GameSession.GameType {
			case models.GameTypeQuiz:
				gameType = "کوییز"
			case models.GameTypeTruthDare:
				gameType = "جرعت حقیقت"
			}

			historyMsg += fmt.Sprintf("%d. %s %s (%s) | %d سکه\n", i+1, statusIcon, resultText, gameType, g.Score)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت به پروفایل", "edit_profile_back"),
		),
	)

	bot.SendMessage(userID, historyMsg, keyboard)
}

func (h *HandlerManager) sendUserList(userID int64, title string, users []models.User, err error, bot BotInterface) {
	if err != nil {
		bot.SendMessage(userID, "❌ خطا در دریافت لیست!", nil)
		return
	}

	if len(users) == 0 {
		bot.SendMessage(userID, title+"\n\nهیچ موردی پیدا نشد.", nil)
		return
	}

	message := title + "\n\n"
	for i, u := range users {
		message += fmt.Sprintf("%d. %s (%d ساله) - %s\n/user_%s\n\n", i+1, u.FullName, u.Age, u.City, u.PublicID)
	}

	bot.SendMessage(userID, message, nil)
}
