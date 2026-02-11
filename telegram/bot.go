package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/config"
	"github.com/mroshb/game_bot/internal/handlers"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/internal/repositories"
	"github.com/mroshb/game_bot/internal/services"
	"github.com/mroshb/game_bot/pkg/logger"
	"github.com/mroshb/game_bot/pkg/utils"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	config   *config.Config
	db       *gorm.DB
	handlers *handlers.HandlerManager

	// User sessions for conversation state
	sessions map[int64]*handlers.UserSession
	mu       sync.RWMutex

	// Worker pool for parallel processing
	workerChans []chan tgbotapi.Update

	// Shutdown context
	ctx    context.Context
	cancel context.CancelFunc
}

// Session states
const (
	StateNone             = ""
	StateRegisterName     = "register_name"
	StateRegisterGender   = "register_gender"
	StateRegisterAge      = "register_age"
	StateRegisterProvince = "register_province"
	StateRegisterCity     = "register_city"
	StateRegisterPhoto    = "register_photo"

	// Edit Profile States
	StateEditName     = "edit_name"
	StateEditAge      = "edit_age"
	StateEditProvince = "edit_province"
	StateEditCity     = "edit_city"
	StateEditPhoto    = "edit_photo"
	StateEditBio      = "edit_bio"

	StateSearchGender    = "search_gender"
	StateSearchAge       = "search_age"
	StateSearchCity      = "search_city"
	StateInChat          = "in_chat"
	StateRoomName        = "room_name"
	StateRoomMaxPlayers  = "room_max_players"
	StateRoomEntryFee    = "room_entry_fee"
	StateAwaitingReceipt = "awaiting_receipt"
	StateVillageName     = "village_name"
	StateVillageDesc     = "village_desc"
)

func InitBot(cfg *config.Config, db *gorm.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	if cfg.AppEnv == "development" {
		api.Debug = true
	}

	logger.Info("Authorized on account", "username", api.Self.UserName)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	coinRepo := repositories.NewCoinRepository(db)
	matchRepo := repositories.NewMatchRepository(db)
	friendRepo := repositories.NewFriendRepository(db)
	gameRepo := repositories.NewGameRepository(db)
	roomRepo := repositories.NewRoomRepository(db)
	villageRepo := repositories.NewVillageRepository(db)
	quizMatchRepo := repositories.NewQuizMatchRepository(db)
	todRepo := repositories.NewTodRepository(db)
	villageSvc := services.NewVillageService(db, villageRepo, userRepo, coinRepo)

	// Initialize handler manager
	handlerMgr := handlers.NewHandlerManager(cfg, db, userRepo, coinRepo, matchRepo, friendRepo, gameRepo, roomRepo, villageRepo, quizMatchRepo, todRepo, villageSvc)

	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		api:         api,
		config:      cfg,
		db:          db,
		handlers:    handlerMgr,
		sessions:    make(map[int64]*handlers.UserSession),
		workerChans: make([]chan tgbotapi.Update, 10), // 10 workers
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start workers
	for i := 0; i < 10; i++ {
		bot.workerChans[i] = make(chan tgbotapi.Update, 100)
		go bot.startWorker(bot.workerChans[i])
	}

	// Start update listener
	go bot.startUpdateListener()

	// Start background jobs
	go bot.startBackgroundJobs()

	// Start Truth or Dare background jobs
	go bot.StartTodBackgroundJobs()

	return bot, nil
}

func (b *Bot) startUpdateListener() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	for {
		// Check if we should stop
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		logger.Info("Starting update listener...")
		updates := b.api.GetUpdatesChan(u)

		for update := range updates {
			// Find userID for hashing
			var userID int64
			if update.Message != nil {
				userID = update.Message.From.ID
			} else if update.CallbackQuery != nil {
				userID = update.CallbackQuery.From.ID
			}

			if userID != 0 {
				// Hashed dispatch to workers to ensure per-user ordered processing
				workerIdx := userID % int64(len(b.workerChans))
				if workerIdx < 0 {
					workerIdx = -workerIdx
				}
				b.workerChans[workerIdx] <- update
			} else {
				// Non-user related update (if any), process normally
				go b.handleUpdate(update)
			}
		}

		// Check if we stopped intentionally
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		logger.Warn("Update channel closed. Restarting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func (b *Bot) startBackgroundJobs() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Use a silent database session for background jobs to reduce log noise
	silentDB := b.db.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	matchRepo := b.handlers.MatchRepo.WithTx(silentDB)
	userRepo := b.handlers.UserRepo.WithTx(silentDB)

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			// Handle timeouts
			timedOutSessions, err := matchRepo.CheckAndHandleTimeouts()
			if err != nil {
				logger.Error("Failed to check timeouts", "error", err)
			} else {
				// Notify users about timeout
				for _, session := range timedOutSessions {
					b.handlers.HandleMatchTimeout(session.User1ID, b)
					b.handlers.HandleMatchTimeout(session.User2ID, b)
				}
			}

			// Check Quiz Match Timeouts (3 days)
			b.handlers.CheckQuizTimeouts(b)

			// Mark inactive users offline (e.g. 10 minutes)
			if count, err := userRepo.MarkInactiveUsersOffline(10 * time.Minute); err == nil && count > 0 {
				logger.Debug("Marked inactive users offline", "count", count)
			}

			// Process Village Wars
			if err := b.handlers.VillageSvc.WithTx(silentDB).ProcessExpiredWars(); err != nil {
				logger.Error("Failed to process village wars", "error", err)
			}

			// Weekly League Reset (Every Saturday at midnight)
			now := time.Now()
			if now.Weekday() == time.Saturday && now.Hour() == 0 && now.Minute() == 0 {
				if err := userRepo.ResetWeeklyLeagues(); err != nil {
					logger.Error("Failed to reset weekly leagues", "error", err)
				} else {
					logger.Info("Weekly leagues reset successfully")
				}
			}
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in handleUpdate", "error", r)
		}
	}()

	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID

	logger.Debug("Received message",
		"user_id", userID,
		"text", message.Text,
		"has_photo", message.Photo != nil,
		"has_location", message.Location != nil,
		"has_venue", message.Venue != nil,
	)

	// Get or create session
	session := b.getSession(userID)

	// Check if user is registered
	user, err := b.handlers.UserRepo.GetUserByTelegramID(userID)
	isRegistered := err == nil && user != nil

	// Update activity and status to online if registered
	if isRegistered {
		b.handlers.UserRepo.UpdateLastActivity(user.ID)
		if user.Status == models.UserStatusOffline {
			b.handlers.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
		}

		// Truth or Dare message handling (for proof submission)
		if b.HandleTodMessages(message) {
			return
		}
	}

	// Custom commands like /user_ID
	if strings.HasPrefix(message.Text, "/user_") {
		publicID := strings.TrimPrefix(message.Text, "/user_")
		b.handlers.SearchUserByPublicID(userID, publicID, b)
		return
	}

	// Handle commands
	if message.IsCommand() {
		b.handleCommand(message, isRegistered)
		return
	}

	// Handle global cancel for registered users
	if isRegistered && (normalizeButton(message.Text) == normalizeButton(BtnCancel) || message.Text == "/cancel") {
		// Clean up any potential DB state
		b.handlers.MatchRepo.RemoveFromQueue(uint(user.ID))
		// Update status
		b.handlers.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
		// Update activity
		b.handlers.UserRepo.UpdateLastActivity(user.ID)

		b.clearSession(userID)
		b.sendMessage(userID, MsgCancel, MainMenuKeyboard(false))
		return
	}

	// Handle registration flow (highest priority state)
	if strings.HasPrefix(session.State, "register_") {
		b.handleRegistrationFlow(message, session)
		return
	}

	// Handle edit profile flow
	if strings.HasPrefix(session.State, "edit_") {
		b.handleEditProfileFlow(message, session)
		return
	}

	// Handle Purchase Receipt
	if session.State == handlers.StateAwaitingReceipt {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandlePurchaseReceipt(userID, message, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Handle button presses (allows switching context)
	if message.Text != "" {
		if b.handleButtonPress(message, user, isRegistered) {
			return
		}
	}

	// Handle Location
	if isRegistered && (message.Location != nil || message.Venue != nil) {
		var lat, lon float64
		if message.Location != nil {
			lat = message.Location.Latitude
			lon = message.Location.Longitude
		} else {
			lat = message.Venue.Location.Latitude
			lon = message.Venue.Location.Longitude
		}

		logger.Debug("Processing location", "user_id", userID, "lat", lat, "lon", lon)
		b.handlers.ListNearbyUsers(userID, lat, lon, b)
		return
	}

	// Handle Location Text (for clients that don't support RequestLocation)
	if message.Text == "📍 ارسال لوکیشن" {
		b.sendMessage(userID, "⚠️ لطفاً لوکیشن خود را از منوی پیوست (آیکون گیره) ارسال کنید.", nil)
		return
	}

	// Handle gender selection for search
	if session.State == handlers.StateSearchGender {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleSearchGenderSelection(message, handlerSession, b)
		// Update local session
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Handle age selection
	if session.State == handlers.StateSearchAge {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleSearchAgeSelection(message, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Handle city selection
	if session.State == handlers.StateSearchCity {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleSearchCitySelection(message, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Handle room creation flow
	if session.State == StateRoomName || session.State == StateRoomMaxPlayers || session.State == StateRoomEntryFee {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleRoomCreation(message, handlerSession, b)
		// Update local session
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Handle village creation flow
	if session.State == StateVillageName || session.State == StateVillageDesc {
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleVillageCreation(message, handlerSession, b)
		// Update local session
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Check for active match (Explicit or Implicit)
	if isRegistered {
		activeMatch, _ := b.handlers.MatchRepo.GetActiveMatch(user.ID)
		if activeMatch != nil {
			// Update session state
			session.State = StateInChat

			// Handle End Chat explicitly here
			if normalizeButton(message.Text) == normalizeButton(BtnEndChat) {
				b.handlers.EndChat(userID, b)
				session.State = StateNone
				return
			}

			// Forward message
			b.handleChatMessage(message, user)
			return
		} else {
			// If session thought we were in chat, but we are not
			if session.State == StateInChat {
				session.State = StateNone
			}
		}

		// Check if user is in a room and send message
		activeRooms, _ := b.handlers.RoomRepo.GetUserRooms(user.ID)
		if len(activeRooms) > 0 {
			roomID, ok := session.Data["current_room_id"].(uint)
			if !ok || roomID == 0 {
				// If session is lost (restart), try to find the "best" room to route to
				if len(activeRooms) == 1 {
					roomID = activeRooms[0].ID
					session.Data["current_room_id"] = roomID
					ok = true
				} else {
					// Multiple rooms: Check if user is in an active game in one of them
					for _, r := range activeRooms {
						game, _ := b.handlers.GameRepo.GetActiveGameSessionByRoomID(r.ID)
						if game != nil {
							roomID = r.ID
							session.Data["current_room_id"] = roomID
							ok = true
							break
						}
					}
				}
			}

			if ok && roomID > 0 {
				if b.handlers.SendRoomMessage(userID, roomID, message, b) {
					return
				}
			}
		}

		// Check if user is in a village and route to village chat
		// Since we don't have a specific "Village Chat State", we can use a flag or just check if they are in Village Hub
		// For now, let's assume if they sent a message and are not in a room/match, it could be a village chat if they have it enabled.
		// Actually, let's keep it simple: if session.State is "village_chat", route there.
		if session.State == "village_chat" {
			b.handlers.SendVillageMessage(userID, message, b)
			return
		}
	}

	// Default response
	if !isRegistered {
		// No button, just text
		b.sendMessage(userID, "👋 سلام! برای شروع ثبت نام لطفاً دستور /start را بزنید.", nil)
	} else {
		// If registered but unknown input -> Main Menu (respecting activity)
		b.SendMainMenu(userID, user.TelegramID == b.config.SuperAdminTgID)
	}
}

func (b *Bot) handleCommand(message *tgbotapi.Message, isRegistered bool) {
	userID := message.From.ID

	switch message.Command() {
	case "start":
		// Always clear session on start to prevent stuck states
		b.clearSession(userID)

		args := message.CommandArguments()
		if args != "" && strings.HasPrefix(args, "ref_") {
			var refID int64
			fmt.Sscanf(utils.NormalizePersianNumbers(args), "ref_%d", &refID)
			if refID != 0 && refID != userID {
				b.getSession(userID).Data["referrer_id"] = uint(refID)
			}
		} else if args != "" && strings.HasPrefix(args, "vjoin_") {
			var villageID uint
			fmt.Sscanf(utils.NormalizePersianNumbers(args), "vjoin_%d", &villageID)
			if villageID != 0 {
				if isRegistered {
					b.handlers.JoinVillageByID(userID, villageID, b)
				} else {
					// Save pending action for after registration
					session := b.getSession(userID)
					session.Data["pending_vjoin"] = villageID

					// Start registration flow
					session.State = handlers.StateRegisterGender
					msgID := b.sendMessage(userID, MsgWelcome, GenderKeyboard())
					session.Data["last_bot_msg_id"] = msgID
				}
				return
			}
		}

		if isRegistered {
			// Check if user is in an active match
			user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
			activeMatch, _ := b.handlers.MatchRepo.GetActiveMatch(user.ID)

			if activeMatch != nil {
				// Resend chat interface
				b.sendMessage(userID, "⚠️ شما در چت فعال هستید! برای خروج پایان چت را بزنید.", handlers.ChatKeyboard())

				// Ensure session is set to InChat
				session := b.getSession(userID)
				session.State = StateInChat
				return
			}

			b.sendMessage(userID, MsgWelcomeBack, MainMenuKeyboard(false))
		} else {
			// Step 1: Start and Gender (Inline)
			session := b.getSession(userID)
			session.State = handlers.StateRegisterGender
			msgID := b.sendMessage(userID, MsgWelcome, GenderKeyboard())
			session.Data["last_bot_msg_id"] = msgID
		}

	case "help":
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
		isAdmin := user != nil && user.TelegramID == b.config.SuperAdminTgID
		b.sendMessage(userID, MsgHelp, MainMenuKeyboard(isAdmin))

	case "cancel":
		b.clearSession(userID)
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)

		if user != nil {
			// Clean up DB state
			b.handlers.MatchRepo.RemoveFromQueue(user.ID)
			b.handlers.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
		}

		isAdmin := user != nil && user.TelegramID == b.config.SuperAdminTgID
		b.sendMessage(userID, MsgCancel, MainMenuKeyboard(isAdmin))

	case "join":
		args := message.CommandArguments()
		if args != "" {
			b.handlers.JoinRoomByCode(userID, args, b)
		} else {
			b.sendMessage(userID, "🔑 لطفاً کد دعوت را هم بفرستید. مثال: /join CODE123", nil)
		}
	case "room":
		b.handlers.ShowRoomMenu(userID, b)

	default:
		command := message.Command()
		if strings.HasPrefix(command, "user_") {
			publicID := strings.TrimPrefix(command, "user_")
			if publicID != "" {
				b.handlers.SearchUserByPublicID(userID, publicID, b)
			}
		}
	}
}

func normalizeButton(s string) string {
	return strings.ReplaceAll(s, "\u200c", "")
}

func (b *Bot) handleButtonPress(message *tgbotapi.Message, user *models.User, isRegistered bool) bool {
	userID := message.From.ID
	text := message.Text

	if !isRegistered {
		b.sendMessage(userID, "👋 سلام! برای شروع ثبت نام لطفاً دستور /start را بزنید.", nil)
		return true
	}

	btn := normalizeButton(text)

	// Activity guard: if user is in an active state, they can only use their "End" button
	// This prevents them from opening the main menu or other features while searching or in a match
	isEndAction := btn == normalizeButton(BtnEndChat) || btn == normalizeButton(BtnCancel) || btn == normalizeButton(BtnTodQuit)
	isChatGameTrigger := (btn == normalizeButton(BtnTruthDare) || btn == normalizeButton(BtnQuiz)) && user.Status == models.UserStatusInMatch

	if isRegistered && user != nil && !isEndAction && !isChatGameTrigger {
		if user.Status == models.UserStatusSearching || user.Status == models.UserStatusInMatch {
			b.sendMessage(userID, "⚠️ شما در حال حاضر مشغول یک فعالیت (چت یا بازی) هستید. برای دسترسی به منوی اصلی باید فعالیت فعلی را تمام کنید یا گزینه خروج/انصراف را بزنید.", ActivityRestrictionKeyboard(user.Status))
			return true
		}
	}

	// Helper to clear state for menu buttons
	clearState := func() {
		b.getSession(userID).State = ""
	}

	switch btn {
	case normalizeButton(BtnPlayGame):
		clearState()
		b.sendMessage(userID, "چه مدلی میخوای بازی کنی؟ انتخاب کن و وارد میدون شو!", PlayModeKeyboard())

	case normalizeButton(BtnTodAnonymous):
		clearState()
		b.sendMessage(userID, "🔥 بخش بازی با ناشناس:\n\nلطفاً فیلتر مورد نظرت رو انتخاب کن:", TodAnonymousFilterKeyboard())

	case normalizeButton(BtnTodFriends):
		clearState()
		b.handlers.StartTodFriends(userID, b)

	case normalizeButton(BtnTodRegister):
		clearState()
		b.handlers.StartTodQuestionRegistration(userID, b)

	case normalizeButton(BtnTodHelp):
		clearState()
		b.sendMessage(userID, MsgHelp, nil) // Or a specific ToD help message

	case normalizeButton(BtnChatNow):
		clearState()
		b.sendMessage(userID, MsgSelectSearchMode, SearchModeKeyboard())

	case normalizeButton(BtnQuickMatch):
		clearState()
		// Start quick match search flow
		b.startSearchFlow(userID)

	case normalizeButton(BtnQuiz):
		clearState()
		// Show active quiz games menu (Glass Menu)
		b.handlers.ShowActiveQuizGames(userID, b)

	case normalizeButton(BtnTruthDare):
		clearState()
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
		match, _ := b.handlers.MatchRepo.GetActiveMatch(user.ID)

		if match != nil {
			// Start ToD game with existing match
			b.handlers.StartTodGameWithMatch(userID, match.ID, b)
		} else {
			// Show room selection or random matchmaking specifically for ToD
			b.sendMessage(userID, "🔥 بخش جرعت یا حقیقت:\n\nمی‌خوای با کی بازی کنی؟", TruthDareRoomKeyboard())
		}

	case normalizeButton(BtnCreateRoom):
		clearState()
		b.handlers.ShowRoomMenu(userID, b)

	case normalizeButton(BtnSearchRoom):
		clearState()
		b.handlers.ListPublicRooms(userID, b)

	case normalizeButton(BtnRandomMatch):
		clearState()
		// If we know they came from ToD menu, maybe we should start ToD matchmaking?
		// For now, let's stick to the callback "btn:tod_new_game" for specific ToD matchmaking.
		b.handlers.StartMatchmaking(userID, models.RequestedGenderAny, b.getSession(userID), b)

	case normalizeButton(BtnOneVsOneRandom):
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
		if user == nil {
			return true
		}
		match, _ := b.handlers.MatchRepo.GetActiveMatch(user.ID)

		if match != nil {
			gameType, _ := b.getSession(userID).Data["game_type"].(string)
			if gameType == "quiz" {
				b.handlers.StartQuiz(userID, b)
			} else {
				b.handlers.StartTodGameWithMatch(userID, match.ID, b)
			}
		} else {
			// Start matchmaking for 1v1
			b.handlers.StartMatchmaking(userID, models.RequestedGenderAny, b.getSession(userID), b)
		}

	case normalizeButton(BtnPlayWithFriends):
		// Generate share link
		botUser, _ := b.api.GetMe()
		shareLink := fmt.Sprintf("https://t.me/%s?start=join_group", botUser.UserName)
		msgText := "بیا با هم بازی کنیم! بزن روی لینک زیر:\n" + shareLink

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("بیا با هم بازی کنیم! ⚔️", fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", shareLink, "بیا با هم بازی کنیم!")),
			),
		)
		b.sendMessage(userID, msgText, keyboard)

	case normalizeButton(BtnBetting):
		b.sendMessage(userID, "💰 مبلغ شرط‌بندی رو انتخاب کن (به زودی...)", nil)

	case normalizeButton(BtnProfile):
		clearState()
		b.handlers.ShowProfile(userID, user, b)

	case normalizeButton(BtnCoinShop):
		b.sendMessage(userID, "🛍 فروشگاه سکه بزودی راه اندازی می شود.", nil)

	case normalizeButton(BtnDailyBonus), "✅ " + normalizeButton(BtnDailyBonus):
		b.handlers.HandleDailyBonus(userID, "", b)

	case normalizeButton(BtnEditProfile):
		b.handlers.HandleEditProfile(userID, b)

	case normalizeButton(BtnLikes):
		b.sendMessage(userID, "❤️ لایک‌های شما (به زودی...)", nil)

	case normalizeButton(BtnEditLocation):
		b.handlers.HandleFilterProvince(userID, b) // Reuse province selection or dedicated edit loc

	case normalizeButton(BtnBlocks):
		b.sendMessage(userID, "🚫 لیست بلاک شده‌های شما (به زودی...)", nil)

	case normalizeButton(BtnSettings):
		clearState()
		b.sendMessage(userID, "⚙️ تنظیمات پروفایل و کاربری:", SettingsHelpKeyboard())

	case normalizeButton(BtnLeaderboard):
		clearState()
		b.handlers.ShowLeaderboard(userID, b) // Will implement in user_handler

	case normalizeButton(BtnFriends):
		clearState()
		b.sendMessage(userID, "دوستات رو بیار، با هم بازی کنید و سکه بگیرید!", SocialHubKeyboard())

	case normalizeButton(BtnVillageHub):
		clearState()
		b.handlers.ShowVillageMenu(userID, b)

	case normalizeButton(BtnCreateVillage):
		clearState()
		b.handlers.StartVillageCreation(userID, b.getSession(userID), b)

	case normalizeButton(BtnLeaveVillage):
		clearState()
		b.handlers.LeaveVillage(userID, b)

	case normalizeButton(BtnVillageLeaderboard):
		clearState()
		b.handlers.ShowVillageLeaderboard(userID, b)

	case normalizeButton(BtnVillageChat):
		session := b.getSession(userID)
		session.State = "village_chat"
		b.sendMessage(userID, "📝 پیام‌های شما در دهکده ارسال می‌شود. برای خروج /cancel بزنید یا از دکمه‌های منو استفاده کنید.", nil)

	case normalizeButton(BtnVillageGame):
		b.sendMessage(userID, "🎮 بازی‌های دهکده به زودی فعال می‌شوند! (در حال توسعه)", nil)

	case normalizeButton(BtnVillageTreasury):
		clearState()
		b.handlers.ShowVillageTreasury(userID, b)

	case normalizeButton(BtnVillageBuffs):
		clearState()
		b.handlers.ShowVillageBuffs(userID, b)

	case normalizeButton(BtnVillageWar):
		clearState()
		b.handlers.ShowVillageWar(userID, b)

	case normalizeButton(BtnInviteToVillage):
		clearState()
		b.handlers.HandleVillageInvite(userID, b)

	case normalizeButton(BtnInviteLink):
		botUser, _ := b.api.GetMe()
		inviteLink := fmt.Sprintf("https://t.me/%s?start=ref_%d", botUser.UserName, userID)
		b.sendMessage(userID, "🔗 لینک دعوت اختصاصی شما:\n\n"+inviteLink, nil)

	case normalizeButton(BtnFriendList):
		b.handlers.ShowFriendsList(userID, b)

	case normalizeButton(BtnFriendRequests):
		b.handlers.ShowFriendRequests(userID, b)

	case normalizeButton(BtnFilterProvince):
		b.handlers.HandleFilterProvince(userID, b)

	case normalizeButton(BtnFilterAge):
		b.handlers.HandleFilterAge(userID, b)

	case normalizeButton(BtnFilterNew):
		b.handlers.HandleFilterNew(userID, b)

	case normalizeButton(BtnFilterNoChat):
		b.handlers.HandleFilterNoChat(userID, b)

	case normalizeButton(BtnFilterRecent):
		b.handlers.HandleFilterRecent(userID, b)

	case normalizeButton(BtnFilterAdvanced):
		clearState()
		b.startSearchFlow(userID)

	case normalizeButton(BtnFemale), normalizeButton(BtnMale), normalizeButton(BtnAny):
		if b.getSession(userID).State == handlers.StateSearchGender {
			return false // Allow state handler to process the selection
		}
		gender := models.RequestedGenderAny
		if btn == normalizeButton(BtnFemale) {
			gender = models.GenderFemale
		} else if btn == normalizeButton(BtnMale) {
			gender = models.GenderMale
		}
		b.handlers.StartMatchmaking(userID, gender, b.getSession(userID), b)

	case normalizeButton(BtnSkip):
		if b.getSession(userID).State == handlers.StateSearchAge || b.getSession(userID).State == handlers.StateSearchCity {
			return false // Allow state handler to process the skip
		}

	case normalizeButton(BtnFilterNearMe):
		b.sendMessage(userID, "📍 برای مشاهده کاربران نزدیک، لطفاً لوکیشن خود را از منوی پیوست (آیکون گیره 📎) ارسال کنید.", nil)

	case normalizeButton(BtnHelp):
		clearState()
		b.sendMessage(userID, "⚙️ تنظیمات و راهنمای بازی:", SettingsHelpKeyboard())

	case normalizeButton(BtnReferral):
		b.handlers.ShowReferralStats(userID, b)

	case normalizeButton(BtnCoins):
		clearState()
		b.sendMessage(userID, "💰 مدیریت سکه‌ها:\n\nلطفاً یکی از گزینه‌های زیر را انتخاب کنید:", CoinsMenuKeyboard())

	case normalizeButton(BtnIncreaseCoins):
		clearState()
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnIHavePaid, "btn:"+BtnIHavePaid),
			),
		)
		b.sendMessage(userID, MsgCoinPurchasePlans, keyboard)

	case normalizeButton(BtnShowBalance):
		clearState()
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
		b.handlers.ShowCoins(userID, user, b)

	case normalizeButton(BtnNotifications):
		b.sendMessage(userID, "🔔 اعلان‌ها فعال/غیرفعال شد (به زودی...)", nil)

	case normalizeButton(BtnTutorials):
		b.sendMessage(userID, MsgHelp, nil)

	case normalizeButton(BtnSupport):
		b.sendMessage(userID, "💬 پشتیبانی: @support_id", nil)

	case normalizeButton(BtnRules):
		b.sendMessage(userID, "⚖️ قوانین و مقررات:\n\n1. ادب را رعایت کنید.\n2. تقلب ممنوع است.", nil)

	case normalizeButton(BtnBack):
		clearState()
		b.SendMainMenu(userID, false)

	case normalizeButton(BtnEndChat):
		clearState()
		b.handlers.EndChat(userID, b)

	case normalizeButton(BtnTodQuit):
		clearState()
		b.handlers.HandleTodQuitSimple(userID, b)

	case normalizeButton(BtnCancel):
		if user != nil {
			b.handlers.MatchRepo.RemoveFromQueue(user.ID)
			b.handlers.UserRepo.UpdateUserStatus(user.ID, models.UserStatusOnline)
		}
		b.clearSession(userID)
		b.sendMessage(userID, MsgCancel, MainMenuKeyboard(false))

	default:
		return false
	}
	return true
}

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Handle inline keyboard callbacks
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	// Remove inline keyboard to keep chat clean
	if query.Message != nil {
		edit := tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
		b.api.Request(edit)
	}

	// Process callback data
	logger.Debug("Callback query", "data", query.Data, "user_id", query.From.ID)

	// Delegate to handlers based on prefix
	data := query.Data
	userID := query.From.ID

	// Truth or Dare game callbacks
	if b.HandleTodCallbacks(query, data) {
		return
	}

	// Quiz game callbacks
	if b.HandleQuizCallbacks(query, data) {
		return
	}

	// Registration callbacks
	if strings.HasPrefix(data, "reg_") {
		session := b.getSession(userID)
		// We'll wrap the callback in a fake message for the handler if needed,
		// but better to handle it directly in handleRegistrationFlow by checking message or callback query.
		// For now, let's pass it to a new handler or modify handleRegistrationFlow.
		b.handleRegistrationCallback(query, session)
		return
	}

	// Leaderboard callback
	if strings.HasPrefix(data, "lb_") {
		// In a real app we would pass the filter to ShowLeaderboard
		// For now we just refresh it
		b.handlers.ShowLeaderboard(userID, b)
		return
	}

	// Advanced Search Callbacks
	if strings.HasPrefix(data, "search_age_") {
		session := b.getSession(userID)
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		msgID := 0
		if query.Message != nil {
			msgID = query.Message.MessageID
		}
		b.handlers.HandleAdvancedSearchAge(userID, data, msgID, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	if strings.HasPrefix(data, "search_province_") {
		session := b.getSession(userID)
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		msgID := 0
		if query.Message != nil {
			msgID = query.Message.MessageID
		}
		b.handlers.HandleAdvancedSearchProvince(userID, data, msgID, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	if len(data) > 11 && data[:11] == "room_create" {
		roomType := models.RoomTypePublic
		if data == "room_create_private" {
			roomType = models.RoomTypePrivate
		}

		session := b.getSession(userID)
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}

		b.handlers.CreateRoom(userID, roomType, handlerSession, b)

		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	if len(data) > 10 && data[:10] == "room_join_" {
		var roomID uint
		fmt.Sscanf(data, "room_join_%d", &roomID)
		b.handlers.JoinRoom(userID, roomID, b)

		// Set current room in session
		session := b.getSession(userID)
		session.Data["current_room_id"] = roomID
		return
	}

	if data == "room_list_public" {
		b.handlers.ListPublicRooms(userID, b)
		return
	}
	if data == "room_my_rooms" {
		b.handlers.GetUserRooms(userID, b)
		return
	}
	if data == "room_quick_join" {
		b.handlers.QuickJoinRoom(userID, b)
		return
	}
	if data == "room_join_code" {
		b.sendMessage(userID, "🔑 برای ورود به اتاق خصوصی، کد دعوت را با دستور /join ارسال کنید.\nمثال: /join CODE123", nil)
		return
	}

	if len(data) > 11 && data[:11] == "room_leave_" {
		var roomID uint
		fmt.Sscanf(data, "room_leave_%d", &roomID)
		b.handlers.LeaveRoom(userID, roomID, b)

		session := b.getSession(userID)
		delete(session.Data, "current_room_id")
		return
	}
	if len(data) > 12 && data[:12] == "room_manage_" {
		var roomID uint
		fmt.Sscanf(data, "room_manage_%d", &roomID)
		b.handlers.ShowManageMembers(userID, roomID, b)
		return
	}
	if len(data) > 13 && data[:13] == "room_members_" {
		var roomID uint
		fmt.Sscanf(data, "room_members_%d", &roomID)
		b.handlers.ShowRoomMembers(userID, roomID, b)
		return
	}
	if len(data) > 10 && data[:10] == "room_kick_" {
		var roomID, targetUserID uint
		fmt.Sscanf(data, "room_kick_%d_%d", &roomID, &targetUserID)
		b.handlers.KickMember(userID, roomID, targetUserID, b)
		return
	}
	if len(data) > 11 && data[:11] == "room_close_" {
		var roomID uint
		fmt.Sscanf(data, "room_close_%d", &roomID)
		b.handlers.CloseRoom(userID, roomID, b)

		session := b.getSession(userID)
		delete(session.Data, "current_room_id")
		return
	}

	if len(data) > 6 && data[:6] == "match_" {
		if data[:11] == "match_chat_" {
			b.sendMessage(userID, "✅ چت شروع شد! پیام بده.", nil)
		} else if data[:11] == "match_quiz_" {
			b.handlers.StartQuiz(userID, b)
		} else if data[:17] == "match_truth_dare_" {
			b.handlers.StartTruthOrDare(userID, b)
		} else if data[:17] == "match_add_friend_" {
			var matchID uint
			fmt.Sscanf(data, "match_add_friend_%d", &matchID)
			b.handlers.HandleAddFriendFromMatch(userID, matchID, b)
		} else if data[:10] == "match_end_" {
			b.handlers.EndChat(userID, b)
		}
		return
	}

	if strings.HasPrefix(data, "truth_") {
		var matchID uint
		fmt.Sscanf(data, "truth_%d", &matchID)
		b.handlers.HandleTruthOrDareChoice(userID, matchID, "truth", b)
		return
	}

	if strings.HasPrefix(data, "dare_") {
		var matchID uint
		fmt.Sscanf(data, "dare_%d", &matchID)
		b.handlers.HandleTruthOrDareChoice(userID, matchID, "dare", b)
		return
	}

	// ========================================
	// NEW QUIZ GAME CALLBACKS
	// ========================================

	if len(data) > 9 && data[:9] == "gt_start_" {
		var roomID uint
		fmt.Sscanf(data, "gt_start_%d", &roomID)
		b.handlers.StartGroupTruthDare(userID, roomID, b)

		// Set current room
		b.getSession(userID).Data["current_room_id"] = roomID
		return
	}
	if len(data) > 9 && data[:9] == "gt_truth_" {
		var sessionID uint
		fmt.Sscanf(data, "gt_truth_%d", &sessionID)
		b.handlers.HandleGroupTruthOrDareChoice(userID, sessionID, "truth", b)

		// Set current room
		if sess, err := b.handlers.GameRepo.GetGameSession(sessionID); err == nil {
			b.getSession(userID).Data["current_room_id"] = sess.RoomID
		}
		return
	}
	if len(data) > 8 && data[:8] == "gt_dare_" {
		var sessionID uint
		fmt.Sscanf(data, "gt_dare_%d", &sessionID)
		b.handlers.HandleGroupTruthOrDareChoice(userID, sessionID, "dare", b)

		// Set current room
		if sess, err := b.handlers.GameRepo.GetGameSession(sessionID); err == nil {
			b.getSession(userID).Data["current_room_id"] = sess.RoomID
		}
		return
	}
	if len(data) > 8 && data[:8] == "gt_next_" {
		var sessionID uint
		fmt.Sscanf(data, "gt_next_%d", &sessionID)
		b.handlers.HandleGroupNextTurn(userID, sessionID, b)

		// Set current room
		if sess, err := b.handlers.GameRepo.GetGameSession(sessionID); err == nil {
			b.getSession(userID).Data["current_room_id"] = sess.RoomID
		}
		return
	}
	if len(data) > 10 && data[:10] == "gt_status_" {
		var sessionID uint
		fmt.Sscanf(data, "gt_status_%d", &sessionID)
		b.handlers.BroadcastGroupGameStatus(sessionID, b, "🎮 وضعیت فعلی بازی:")

		// Set current room
		if sess, err := b.handlers.GameRepo.GetGameSession(sessionID); err == nil {
			b.getSession(userID).Data["current_room_id"] = sess.RoomID
		}
		return
	}
	if len(data) > 7 && data[:7] == "gt_end_" {
		var sessionID uint
		fmt.Sscanf(data, "gt_end_%d", &sessionID)
		b.handlers.HandleGroupEndGame(userID, sessionID, b)
		return
	}

	// Group invitation callbacks
	if len(data) > 10 && data[:10] == "gt_invite_" {
		var roomID uint
		fmt.Sscanf(data, "gt_invite_%d", &roomID)
		b.handlers.InviteFriendToRoom(userID, roomID, b)
		return
	}

	if len(data) > 12 && data[:12] == "gt_send_inv_" {
		var roomID, friendID uint
		fmt.Sscanf(data, "gt_send_inv_%d_%d", &roomID, &friendID)
		b.handlers.SendRoomInvitation(userID, roomID, friendID, b)
		return
	}

	if len(data) > 14 && data[:14] == "gt_accept_inv_" {
		var roomID uint
		fmt.Sscanf(data, "gt_accept_inv_%d", &roomID)
		b.handlers.JoinRoom(userID, roomID, b)

		session := b.getSession(userID)
		session.Data["current_room_id"] = roomID
		return
	}

	// Village treasury, buffs, and war
	if strings.HasPrefix(data, "vdonate_") {
		var amount int64
		fmt.Sscanf(data, "vdonate_%d", &amount)
		b.handlers.HandleVillageDonate(userID, amount, b)
		return
	}
	if strings.HasPrefix(data, "vupgrade_") {
		buffType := strings.TrimPrefix(data, "vupgrade_")
		b.handlers.HandleVillageUpgrade(userID, buffType, b)
		return
	}
	if data == "vwar_start" {
		b.handlers.HandleVillageWarStart(userID, b)
		return
	}

	if len(data) > 14 && data[:14] == "gt_reject_inv_" {
		b.sendMessage(userID, "❌ دعوت بازی رد شد.", nil)
		return
	}

	if strings.HasPrefix(data, "tod_chat_") {
		var gameID uint
		fmt.Sscanf(data, "tod_chat_%d", &gameID)
		b.handlers.HandleTodChat(userID, gameID, b)
		return
	}

	if len(data) > 10 && data[:10] == "match_cat_" {
		var matchID uint
		var category string
		parts := strings.Split(data, "_")
		if len(parts) >= 5 {
			fmt.Sscanf(parts[2], "%d", &matchID)
			category = strings.Join(parts[3:], "_")
			b.handlers.HandleMatchTruthOrDareCategorySelection(userID, matchID, category, b)
		}
		return
	}

	if len(data) > 7 && data[:7] == "gt_cat_" {
		var sessionID uint
		var category string
		// Extract sessionID and category: gt_cat_{sessionID}_{category}
		// Category could contain underscores, so we need to be careful.
		parts := strings.Split(data, "_")
		if len(parts) >= 5 {
			fmt.Sscanf(parts[2], "%d", &sessionID)
			category = strings.Join(parts[3:], "_")
			b.handlers.HandleGroupTruthOrDareCategorySelection(userID, sessionID, category, b)
		}
		return
	}

	if len(data) > 10 && data[:10] == "gt_change_" {
		var sessionID uint
		var category string
		// gt_change_{sessionID}_{category}
		parts := strings.Split(data, "_")
		if len(parts) >= 5 {
			fmt.Sscanf(parts[2], "%d", &sessionID)
			category = strings.Join(parts[3:], "_")
			b.handlers.HandleGroupTruthOrDareCategorySelection(userID, sessionID, category, b)
		}
		return
	}

	if data == "buy_coins" || data == "shop" {
		msgID := 0
		if query.Message != nil {
			msgID = query.Message.MessageID
		}
		b.handlers.HandleBuyCoins(userID, msgID, b)
		return
	}
	if data == "daily_bonus" {
		b.handlers.HandleDailyBonus(userID, query.ID, b)
		return
	}
	if data == "paid_coins" {
		session := b.getSession(userID)
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandlePaid(userID, handlerSession, b)
		session.State = handlerSession.State
		session.Data = handlerSession.Data
		return
	}

	// Edit Profile Callbacks
	if data == "edit_profile" {
		b.handlers.HandleEditProfile(userID, b)
		return
	}
	if strings.HasPrefix(data, "edit_field_") {
		field := strings.TrimPrefix(data, "edit_field_")
		session := b.getSession(userID)
		handlerSession := &handlers.UserSession{
			State: session.State,
			Data:  session.Data,
		}
		b.handlers.HandleEditFieldSelection(userID, field, handlerSession, b)
		session.State = handlerSession.State
		// Data might be updated too
		session.Data = handlerSession.Data
		return
	}
	if data == "edit_profile_back" {
		user, _ := b.handlers.UserRepo.GetUserByTelegramID(userID)
		b.handlers.ShowProfile(userID, user, b)
		return
	}

	// Friend callbacks
	if data == "inventory" {
		b.handlers.ShowInventory(userID, b)
		return
	}
	if data == "game_history" {
		b.handlers.ShowGameHistory(userID, b)
		return
	}
	if data == "shop_items" {
		b.sendMessage(userID, "🛍 فروشگاه آیتم‌ها بزودی باز می‌شود!", nil)
		return
	}

	// Friend callbacks
	if len(data) > 11 && data[:11] == "friend_add_" {
		var targetUserID uint
		fmt.Sscanf(data, "friend_add_%d", &targetUserID)
		b.handlers.HandleAddFriend(userID, targetUserID, b)
		return
	}
	if len(data) > 13 && data[:14] == "friend_accept_" {
		var friendID uint
		fmt.Sscanf(data, "friend_accept_%d", &friendID)
		b.handlers.HandleFriendRequestAction(userID, friendID, "accept", b)
		return
	}
	if len(data) > 13 && data[:14] == "friend_reject_" {
		var friendID uint
		fmt.Sscanf(data, "friend_reject_%d", &friendID)
		b.handlers.HandleFriendRequestAction(userID, friendID, "reject", b)
		return
	}
	if len(data) > 13 && data[:14] == "friend_remove_" {
		var friendID uint
		fmt.Sscanf(data, "friend_remove_%d", &friendID)
		b.handlers.HandleRemoveFriend(userID, friendID, b)
		return
	}
	if len(data) > 5 && data[:5] == "like_" {
		var likedUserID uint
		fmt.Sscanf(data, "like_%d", &likedUserID)
		b.handlers.HandleLike(userID, likedUserID, b)
		return
	}

	// Fallback: Button Label Emulation
	if strings.HasPrefix(data, "btn:") {
		btnText := strings.TrimPrefix(data, "btn:")
		// Simulate a message to trigger full message handling (including states)
		fakeMsg := &tgbotapi.Message{
			From: query.From,
			Chat: query.Message.Chat,
			Text: btnText,
		}
		b.handleMessage(fakeMsg)
		return
	}
}

func (b *Bot) handleChatMessage(message *tgbotapi.Message, user *models.User) {
	// Check for active ToD game first
	if user.Status == models.UserStatusInMatch {
		// Check if it's a ToD match
		todGame, _ := b.handlers.TodRepo.GetActiveGameForUser(user.ID)
		if todGame != nil {
			b.handlers.HandleTodChatMessage(message, user, b)
			return
		}
	}

	// Default chat handler
	b.handlers.HandleChatMessage(message, user, b)
}

func (b *Bot) handleRegistrationFlow(message *tgbotapi.Message, session *handlers.UserSession) {

	handlerSession := &handlers.UserSession{
		State: session.State,
		Data:  session.Data,
	}
	b.handlers.HandleRegistration(message, handlerSession, b)
	session.State = handlerSession.State
	session.Data = handlerSession.Data
}

func (b *Bot) handleRegistrationCallback(query *tgbotapi.CallbackQuery, session *handlers.UserSession) {
	handlerSession := &handlers.UserSession{
		State: session.State,
		Data:  session.Data,
	}
	b.handlers.HandleRegistrationCallback(query, handlerSession, b)
	session.State = handlerSession.State
	session.Data = handlerSession.Data
}

func (b *Bot) handleEditProfileFlow(message *tgbotapi.Message, session *handlers.UserSession) {
	handlerSession := &handlers.UserSession{
		State: session.State,
		Data:  session.Data,
	}
	b.handlers.HandleEditProfileInput(message, handlerSession, b)
	session.State = handlerSession.State
	session.Data = handlerSession.Data
}

func (b *Bot) startSearchFlow(userID int64) {
	user, err := b.handlers.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return
	}

	inQueue, _ := b.handlers.MatchRepo.IsUserInQueue(user.ID)
	if inQueue {
		// Instead of blocking, trigger the resume logic in StartMatchmaking
		b.handlers.StartMatchmaking(userID, "", b.getSession(userID), b)
		return
	}

	activeMatch, _ := b.handlers.MatchRepo.GetActiveMatch(user.ID)
	if activeMatch != nil {
		b.sendMessage(userID, MsgAlreadyInMatch, handlers.ChatKeyboard())
		return
	}

	session := b.getSession(userID)
	session.State = StateSearchGender
	b.sendMessage(userID, MsgSelectGender, SearchGenderFilterKeyboard())
}

func (b *Bot) getSession(userID int64) *handlers.UserSession {
	b.mu.Lock()
	defer b.mu.Unlock()

	if session, exists := b.sessions[userID]; exists {
		return session
	}

	session := &handlers.UserSession{
		State: StateNone,
		Data:  make(map[string]interface{}),
	}
	b.sessions[userID] = session
	return session
}

func (b *Bot) clearSession(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.sessions[userID] = &handlers.UserSession{
		State: StateNone,
		Data:  make(map[string]interface{}),
	}
}

func (b *Bot) sendMessage(chatID int64, text string, keyboard interface{}) int {
	// Add RTL mark for Persian support
	rtlText := "\u200f" + text
	msg := tgbotapi.NewMessage(chatID, rtlText)
	msg.ParseMode = tgbotapi.ModeHTML

	switch kb := keyboard.(type) {
	case tgbotapi.ReplyKeyboardMarkup:
		msg.ReplyMarkup = kb
	case tgbotapi.InlineKeyboardMarkup:
		msg.ReplyMarkup = kb
	case tgbotapi.ReplyKeyboardRemove:
		msg.ReplyMarkup = kb
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		sentMsg, err := b.api.Send(msg)
		if err != nil {
			logger.Error("Failed to send message", "error", err, "chat_id", chatID, "attempt", i+1)

			// If it's a network error, wait and retry
			if strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "network is unreachable") {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return 0 // Non-network error, don't retry
		}
		return sentMsg.MessageID // Success
	}
	return 0 // All retries failed
}

func (b *Bot) SendMessage(chatID int64, text string, keyboard interface{}) int {
	return b.sendMessage(chatID, text, keyboard)
}

func (b *Bot) DeleteMessage(chatID int64, messageID int) {
	if messageID == 0 {
		return
	}
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := b.api.Request(deleteMsg); err != nil {
		logger.Error("Failed to delete message", "chat_id", chatID, "msg_id", messageID, "error", err)
	}
}

func (b *Bot) EditMessage(chatID int64, messageID int, text string, keyboard interface{}) {
	// Add RTL mark
	rtlText := "\u200f" + text
	msg := tgbotapi.NewEditMessageText(chatID, messageID, rtlText)
	msg.ParseMode = tgbotapi.ModeHTML

	if keyboard != nil {
		if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = &kb
		}
	}

	if _, err := b.api.Send(msg); err != nil {
		logger.Error("Failed to edit message", "error", err, "chat_id", chatID, "message_id", messageID)
	}
}

func (b *Bot) SendMainMenu(chatID int64, isAdmin bool) {
	user, _ := b.handlers.UserRepo.GetUserByTelegramID(chatID)
	if user != nil && (user.Status == models.UserStatusSearching || user.Status == models.UserStatusInMatch) {
		b.sendMessage(chatID, "⚠️ شما در حال حاضر مشغول هستید. برای دسترسی به منوی اصلی باید فعالیت فعلی را تمام کنید یا گزینه انصراف/پایان را بزنید.", ActivityRestrictionKeyboard(user.Status))
		return
	}
	b.sendMessage(chatID, MsgMainMenu, MainMenuKeyboard(isAdmin))
}

func (b *Bot) GetMainMenuKeyboard(isAdmin bool) interface{} {
	return MainMenuKeyboard(isAdmin)
}

func (b *Bot) GetGenderKeyboard() interface{} {
	return GenderKeyboard()
}

func (b *Bot) GetAgeSelectionKeyboard() interface{} {
	return AgeSelectionKeyboard()
}

func (b *Bot) GetProvinceKeyboard() interface{} {
	return ProvinceKeyboard()
}

func (b *Bot) GetPhotoSelectionKeyboard() interface{} {
	return PhotoSelectionKeyboard()
}

func (b *Bot) GetPhotoSkipKeyboard() interface{} {
	return PhotoSkipKeyboard()
}

func (b *Bot) GetCancelInlineKeyboard() interface{} {
	return CancelInlineKeyboard()
}

func (b *Bot) GetEditProfileFieldsKeyboard() interface{} {
	return EditProfileFieldsKeyboard()
}

func (b *Bot) GetTodAnonymousFilterKeyboard() interface{} {
	return TodAnonymousFilterKeyboard()
}

func (b *Bot) GetTodChallengeTypeKeyboard(gameID uint) interface{} {
	return TodChallengeTypeKeyboard(gameID)
}

func (b *Bot) GetTodResponderInteractionKeyboard(gameID uint) interface{} {
	return TodResponderInteractionKeyboard(gameID)
}

func (b *Bot) GetTodQuestionerInteractionKeyboard(gameID uint) interface{} {
	return TodQuestionerInteractionKeyboard(gameID)
}

func (b *Bot) GetTodGroupLobbyKeyboard(sessionID uint, isHost bool) interface{} {
	return TodGroupLobbyKeyboard(sessionID, isHost)
}

func (b *Bot) GetAPI() interface{} {
	return b.api
}

func (b *Bot) GetConfig() interface{} {
	return b.config
}

func (b *Bot) GetVillageHubKeyboard(hasVillage bool) interface{} {
	return VillageHubKeyboard(hasVillage)
}

func (b *Bot) GetVillageTreasuryKeyboard() interface{} {
	return VillageTreasuryKeyboard()
}

func (b *Bot) GetVillageBuffsKeyboard(isLeader bool) interface{} {
	return VillageBuffsKeyboard(isLeader)
}

func (b *Bot) GetVillageWarKeyboard(isLeader bool, hasActiveWar bool) interface{} {
	return VillageWarKeyboard(isLeader, hasActiveWar)
}

func (b *Bot) Stop() {
	b.cancel()
	b.api.StopReceivingUpdates()
	logger.Info("Bot stopped receiving updates")
}
func (b *Bot) EditMessageReplyMarkup(chatID int64, messageID int, keyboard interface{}) {
	var kb tgbotapi.InlineKeyboardMarkup
	if keyboard != nil {
		if k, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			kb = k
		}
	} else {
		// Initialize empty slice to ensure it marshals to [] instead of null
		kb.InlineKeyboard = make([][]tgbotapi.InlineKeyboardButton, 0)
	}
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, kb)
	b.api.Request(edit)
}

func (b *Bot) SendPhoto(chatID int64, photoID string, caption string, keyboard interface{}) int {
	return b.sendPhoto(chatID, photoID, caption, keyboard)
}

func (b *Bot) AnswerCallbackQuery(queryID string, text string, showAlert bool) {
	callback := tgbotapi.NewCallback(queryID, text)
	callback.ShowAlert = showAlert
	if _, err := b.api.Request(callback); err != nil {
		logger.Error("Failed to answer callback query", "error", err, "query_id", queryID)
	}
}

func (b *Bot) sendPhoto(chatID int64, photoID string, caption string, keyboard interface{}) int {
	// Add RTL mark
	rtlCaption := "\u200f" + caption
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(photoID))
	photo.Caption = rtlCaption
	photo.ParseMode = tgbotapi.ModeHTML

	switch kb := keyboard.(type) {
	case tgbotapi.ReplyKeyboardMarkup:
		photo.ReplyMarkup = kb
	case tgbotapi.InlineKeyboardMarkup:
		photo.ReplyMarkup = kb
	case tgbotapi.ReplyKeyboardRemove:
		photo.ReplyMarkup = kb
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		sentMsg, err := b.api.Send(photo)
		if err != nil {
			logger.Error("Failed to send photo", "error", err, "chat_id", chatID, "attempt", i+1)
			if strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "timeout") {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return 0
		}
		return sentMsg.MessageID
	}
	return 0
}

func (b *Bot) startWorker(ch chan tgbotapi.Update) {
	for update := range ch {
		b.handleUpdate(update)
	}
}

func (b *Bot) GetCancelKeyboard() interface{} {
	return CancelKeyboard()
}
