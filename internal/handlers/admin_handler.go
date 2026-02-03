package handlers

import (
	"fmt"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// HandleAdminStats shows bot statistics
func (h *HandlerManager) HandleAdminStats(userID int64, user *models.User, bot BotInterface) {
	if user == nil || user.TelegramID != h.Config.SuperAdminTgID {
		bot.SendMessage(userID, "❌ فقط مدیران می‌توانند از این دستور استفاده کنند!", nil)
		return
	}

	// Get total users
	var totalUsers int64
	h.DB.Model(&models.User{}).Count(&totalUsers)

	// Get online users
	var onlineUsers int64
	h.DB.Model(&models.User{}).Where("status = ?", models.UserStatusOnline).Count(&onlineUsers)

	// Get users in match
	var inMatchUsers int64
	h.DB.Model(&models.User{}).Where("status = ?", models.UserStatusInMatch).Count(&inMatchUsers)

	// Get total matches
	var totalMatches int64
	h.DB.Model(&models.MatchSession{}).Count(&totalMatches)

	// Get active matches
	var activeMatches int64
	h.DB.Model(&models.MatchSession{}).Where("status = ?", models.MatchStatusActive).Count(&activeMatches)

	// Get total questions
	var totalQuestions int64
	h.DB.Model(&models.Question{}).Count(&totalQuestions)

	// Get total rooms
	var totalRooms int64
	h.DB.Model(&models.Room{}).Count(&totalRooms)

	statsMsg := fmt.Sprintf(`📊 آمار ربات:

👥 کاربران:
  • کل: %d
  • آنلاین: %d
  • در چت: %d

🎮 Match:
  • کل: %d
  • فعال: %d

❓ سوالات: %d
🏠 اتاق‌ها: %d`,
		totalUsers, onlineUsers, inMatchUsers,
		totalMatches, activeMatches,
		totalQuestions, totalRooms)

	bot.SendMessage(userID, statsMsg, nil)
	logger.Info("Admin viewed stats", "admin_id", userID)
}

// HandleBroadcast sends message to all users
func (h *HandlerManager) HandleBroadcast(userID int64, user *models.User, message string, bot BotInterface) {
	if user == nil || user.TelegramID != h.Config.SuperAdminTgID {
		bot.SendMessage(userID, "❌ فقط مدیران می‌توانند پیام همگانی بفرستند!", nil)
		return
	}

	// Get all users
	var users []models.User
	if err := h.DB.Find(&users).Error; err != nil {
		logger.Error("Failed to get users for broadcast", "error", err)
		bot.SendMessage(userID, "❌ خطا در دریافت لیست کاربران!", nil)
		return
	}

	// Send message to all users
	successCount := 0
	for _, u := range users {
		broadcastMsg := fmt.Sprintf("📢 پیام از مدیریت:\n\n%s", message)
		bot.SendMessage(u.TelegramID, broadcastMsg, nil)
		successCount++
	}

	resultMsg := fmt.Sprintf("✅ پیام به %d کاربر ارسال شد!", successCount)
	bot.SendMessage(userID, resultMsg, nil)
	logger.Info("Admin broadcast message", "admin_id", userID, "recipients", successCount)
}

// HandleBanUser bans a user
func (h *HandlerManager) HandleBanUser(adminID int64, admin *models.User, targetUserID uint, bot BotInterface) {
	if admin == nil || admin.TelegramID != h.Config.SuperAdminTgID {
		bot.SendMessage(adminID, "❌ فقط مدیران می‌توانند کاربر را بن کنند!", nil)
		return
	}

	// Get target user
	targetUser, err := h.UserRepo.GetUserByID(targetUserID)
	if err != nil {
		bot.SendMessage(adminID, "❌ کاربر پیدا نشد!", nil)
		return
	}

	// Don't ban admins
	if targetUser.TelegramID == h.Config.SuperAdminTgID {
		bot.SendMessage(adminID, "❌ نمی‌توانید مدیر را بن کنید!", nil)
		return
	}

	// Update user status to banned (we'll use a custom status)
	if err := h.UserRepo.UpdateUserStatus(targetUserID, "banned"); err != nil {
		logger.Error("Failed to ban user", "error", err)
		bot.SendMessage(adminID, "❌ خطا در بن کردن کاربر!", nil)
		return
	}

	// Notify admin
	msg := fmt.Sprintf("✅ کاربر %s (ID: %d) بن شد!", targetUser.FullName, targetUser.ID)
	bot.SendMessage(adminID, msg, nil)

	// Notify user
	bot.SendMessage(targetUser.TelegramID, "⛔️ شما توسط مدیریت بن شدید!", nil)

	logger.Info("User banned", "admin_id", adminID, "target_id", targetUserID)
}

// HandleUnbanUser unbans a user
func (h *HandlerManager) HandleUnbanUser(adminID int64, admin *models.User, targetUserID uint, bot BotInterface) {
	if admin == nil || admin.TelegramID != h.Config.SuperAdminTgID {
		bot.SendMessage(adminID, "❌ فقط مدیران می‌توانند کاربر را آنبن کنند!", nil)
		return
	}

	// Get target user
	targetUser, err := h.UserRepo.GetUserByID(targetUserID)
	if err != nil {
		bot.SendMessage(adminID, "❌ کاربر پیدا نشد!", nil)
		return
	}

	// Update user status to offline
	if err := h.UserRepo.UpdateUserStatus(targetUserID, models.UserStatusOffline); err != nil {
		logger.Error("Failed to unban user", "error", err)
		bot.SendMessage(adminID, "❌ خطا در آنبن کردن کاربر!", nil)
		return
	}

	// Notify admin
	msg := fmt.Sprintf("✅ کاربر %s (ID: %d) آنبن شد!", targetUser.FullName, targetUser.ID)
	bot.SendMessage(adminID, msg, nil)

	// Notify user
	bot.SendMessage(targetUser.TelegramID, "✅ شما توسط مدیریت آنبن شدید!", nil)

	logger.Info("User unbanned", "admin_id", adminID, "target_id", targetUserID)
}

// HandleListUsers shows list of users with pagination
func (h *HandlerManager) HandleListUsers(adminID int64, admin *models.User, page int, bot BotInterface) {
	if admin == nil || admin.TelegramID != h.Config.SuperAdminTgID {
		bot.SendMessage(adminID, "❌ فقط مدیران می‌توانند لیست کاربران را ببینند!", nil)
		return
	}

	limit := 10
	offset := (page - 1) * limit

	var users []models.User
	if err := h.DB.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		logger.Error("Failed to get users", "error", err)
		bot.SendMessage(adminID, "❌ خطا در دریافت لیست کاربران!", nil)
		return
	}

	if len(users) == 0 {
		bot.SendMessage(adminID, "❌ کاربری یافت نشد!", nil)
		return
	}

	msg := fmt.Sprintf("👥 لیست کاربران (صفحه %d):\n\n", page)
	for i, user := range users {
		status := "آفلاین"
		switch user.Status {
		case models.UserStatusOnline:
			status = "🟢 آنلاین"
		case models.UserStatusSearching:
			status = "🟡 در جستجو"
		case models.UserStatusInMatch:
			status = "🔴 در چت"
		case "banned":
			status = "⛔️ بن شده"
		}

		msg += fmt.Sprintf("%d. %s (ID: %d)\n   💰 %d سکه | %s\n\n",
			offset+i+1, user.FullName, user.ID, user.CoinBalance, status)
	}

	bot.SendMessage(adminID, msg, nil)
}
