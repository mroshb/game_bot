package handlers

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *HandlerManager) ShowVillageMenu(userID int64, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطایی در بارگذاری اطلاعات رخ داد.", nil)
		return
	}

	village, rank, _ := h.VillageSvc.GetUserVillageInfo(user.ID)

	var text string
	if village == nil {
		text = MsgVillageWelcome
	} else {
		text = fmt.Sprintf(MsgVillageInfo,
			village.Name, village.Level, village.XP, rank, village.MemberCount, village.Description)
	}

	bot.SendMessage(userID, text, bot.GetVillageHubKeyboard(village != nil))
}

func (h *HandlerManager) StartVillageCreation(userID int64, session *UserSession, bot BotInterface) {
	session.State = "village_name"
	bot.SendMessage(userID, MsgVillageCreateName, bot.GetCancelKeyboard())
}

func (h *HandlerManager) HandleVillageCreation(message *tgbotapi.Message, session *UserSession, bot BotInterface) {
	userID := message.From.ID
	user, _ := h.UserRepo.GetUserByTelegramID(userID)

	switch session.State {
	case "village_name":
		name := strings.TrimSpace(message.Text)
		if len(name) < 3 || len(name) > 30 {
			bot.SendMessage(userID, "❌ نام دهکده باید بین ۳ تا ۳۰ کاراکتر باشد. دوباره تلاش کنید:", bot.GetCancelKeyboard())
			return
		}

		session.Data["v_name"] = name
		session.State = "village_desc"
		bot.SendMessage(userID, MsgVillageCreateDesc, bot.GetCancelKeyboard())

	case "village_desc":
		desc := strings.TrimSpace(message.Text)
		name := session.Data["v_name"].(string)

		village, err := h.VillageSvc.CreateVillage(name, desc, user.ID)
		if err != nil {
			bot.SendMessage(userID, "❌ خطا: "+err.Error(), nil)
			session.State = ""
			h.ShowVillageMenu(userID, bot)
			return
		}

		bot.SendMessage(userID, fmt.Sprintf(MsgVillageCreateSuccess, village.Name), nil)

		session.State = ""
		h.ShowVillageMenu(userID, bot)
	}
}

func (h *HandlerManager) ShowVillageLeaderboard(userID int64, bot BotInterface) {
	villages, err := h.VillageSvc.GetRanking(10)
	if err != nil {
		bot.SendMessage(userID, "❌ خطایی در دریافت رنکینگ رخ داد.", nil)
		return
	}

	if len(villages) == 0 {
		bot.SendMessage(userID, "📭 هنوز هیچ دهکده‌ای ثبت نشده است.", nil)
		return
	}

	var sb strings.Builder
	sb.WriteString("🏅 **برترین دهکده‌های کشور**\n\n")

	for i, v := range villages {
		sb.WriteString(fmt.Sprintf("%d. %s (سطح %d) - %d امتیاز\n", i+1, v.Name, v.Level, v.Score))
	}

	bot.SendMessage(userID, sb.String(), nil)
}

func (h *HandlerManager) LeaveVillage(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	err := h.VillageSvc.LeaveVillage(user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا: "+err.Error(), nil)
		return
	}

	bot.SendMessage(userID, MsgVillageLeaveSuccess, nil)

	h.ShowVillageMenu(userID, bot)
}

func (h *HandlerManager) SendVillageMessage(userID int64, message *tgbotapi.Message, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	village, _ := h.VillageRepo.GetUserVillage(user.ID)
	if village == nil {
		return
	}

	members, _ := h.VillageRepo.GetVillageMembers(village.ID)

	msgText := fmt.Sprintf("💬 [%s]: %s", user.FullName, message.Text)

	for _, m := range members {
		if m.User.TelegramID != userID {
			bot.SendMessage(m.User.TelegramID, msgText, nil)
		}
	}
}

func (h *HandlerManager) HandleVillageInvite(userID int64, bot BotInterface) {
	user, _ := h.UserRepo.GetUserByTelegramID(userID)
	village, _ := h.VillageRepo.GetUserVillage(user.ID)
	if village == nil {
		bot.SendMessage(userID, "❌ شما عضو هیچ دهکده‌ای نیستید.", nil)
		return
	}

	botUser, _ := (bot.GetAPI().(*tgbotapi.BotAPI)).GetMe()
	inviteLink := fmt.Sprintf("https://t.me/%s?start=vjoin_%d", botUser.UserName, village.ID)

	text := fmt.Sprintf(MsgVillageInviteText, village.Name, inviteLink)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📥 پیوستن به دهکده", fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", inviteLink, "بیا به دهکده ما!")),
		),
	)

	bot.SendMessage(userID, text, keyboard)
}

func (h *HandlerManager) JoinVillageByID(userID int64, villageID uint, bot BotInterface) {
	user, err := h.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		bot.SendMessage(userID, "❌ اول باید ثبت‌نام کنی!", nil)
		return
	}

	err = h.VillageSvc.AddMember(villageID, user.ID)
	if err != nil {
		bot.SendMessage(userID, "❌ خطا: "+err.Error(), nil)
		return
	}

	village, _ := h.VillageRepo.GetVillageByID(villageID)
	bot.SendMessage(userID, fmt.Sprintf(MsgVillageJoinSuccess, village.Name), nil)
	h.ShowVillageMenu(userID, bot)
}
