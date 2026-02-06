package telegram

import (
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/logger"
)

// HandleTodCallbacks handles all Truth or Dare related callbacks
func (b *Bot) HandleTodCallbacks(query *tgbotapi.CallbackQuery, data string) bool {
	userID := query.From.ID

	// Truth or Dare game callbacks
	if strings.HasPrefix(data, "btn:tod_") {
		parts := strings.Split(data, "_")

		switch {
		case data == "btn:tod_new_game":
			b.handlers.StartTodMatchmaking(userID, b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case data == "btn:tod_cancel_search":
			// Cancel matchmaking
			b.api.Send(tgbotapi.NewCallback(query.ID, "جستجو لغو شد"))
			b.SendMessage(userID, "❌ جستجو لغو شد", nil)
			return true

		case strings.HasPrefix(data, "btn:tod_start_"):
			// Extract game ID
			gameIDStr := strings.TrimPrefix(data, "btn:tod_start_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.HandleTodStart(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_choice_"):
			// Format: btn:tod_choice_{gameID}_{choice}
			if len(parts) >= 5 {
				gameIDStr := parts[3]
				choice := parts[4]
				gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
				if err != nil {
					logger.Error("Invalid game ID", "data", data)
					return true
				}
				b.handlers.HandleTodChoice(userID, uint(gameID), choice, b)
				b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			}
			return true

		case strings.HasPrefix(data, "btn:tod_confirm_proof_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_confirm_proof_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.HandleTodConfirmProof(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_resubmit_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_resubmit_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.HandleTodResubmit(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_judge_"):
			// Format: btn:tod_judge_{gameID}_{result}
			if len(parts) >= 5 {
				gameIDStr := parts[3]
				result := parts[4]
				gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
				if err != nil {
					logger.Error("Invalid game ID", "data", data)
					return true
				}
				b.handlers.HandleTodJudgment(userID, uint(gameID), result, b)
				b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			}
			return true

		case strings.HasPrefix(data, "btn:tod_items_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_items_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.ShowTodItemMenu(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_use_item_"):
			// Format: btn:tod_use_item_{gameID}_{itemType}
			if len(parts) >= 6 {
				gameIDStr := parts[4]
				itemType := parts[5]
				gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
				if err != nil {
					logger.Error("Invalid game ID", "data", data)
					return true
				}
				b.handlers.HandleTodItemUse(userID, uint(gameID), itemType, b)
				b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			}
			return true

		case strings.HasPrefix(data, "btn:tod_quit_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_quit_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.HandleTodQuit(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_nudge_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_nudge_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.HandleTodNudge(userID, uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, "تلنگر ارسال شد!"))
			return true

		case strings.HasPrefix(data, "btn:tod_back_"):
			gameIDStr := strings.TrimPrefix(data, "btn:tod_back_")
			gameID, err := strconv.ParseUint(gameIDStr, 10, 32)
			if err != nil {
				logger.Error("Invalid game ID", "data", data)
				return true
			}
			b.handlers.ShowTodChoiceScreen(uint(gameID), b)
			b.api.Send(tgbotapi.NewCallback(query.ID, ""))
			return true

		case strings.HasPrefix(data, "btn:tod_chat_"):
			// Limited chat feature (future implementation)
			b.api.Send(tgbotapi.NewCallback(query.ID, "این قابلیت به زودی اضافه می‌شود!"))
			return true
		}
	}

	return false
}

// HandleTodMessages handles text/media messages during ToD game
func (b *Bot) HandleTodMessages(message *tgbotapi.Message) bool {
	userID := message.From.ID

	// Check if user is in active ToD game
	user, err := b.handlers.UserRepo.GetUserByTelegramID(userID)
	if err != nil {
		return false
	}

	activeGame, err := b.handlers.TodRepo.GetActiveGameForUser(user.ID)
	if err != nil || activeGame == nil {
		return false
	}

	// Check if game is in proof submission state
	if activeGame.State == models.TodStateWaitingProof && activeGame.ActivePlayerID == user.ID {
		b.handlers.HandleTodProofSubmission(userID, activeGame.ID, message, b)
		return true
	}

	return false
}

// StartTodBackgroundJobs starts background jobs for ToD game management
func (b *Bot) StartTodBackgroundJobs() {
	// Warning job (runs every 10 seconds)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			games, err := b.handlers.TodRepo.GetGamesNearingTimeout()
			if err != nil {
				continue
			}

			for _, game := range games {
				b.handlers.SendTodWarning(game.ID, b)
				b.handlers.TodRepo.MarkWarningShown(game.ID)
			}
		}
	}()

	// Timeout job (runs every 5 seconds)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			games, err := b.handlers.TodRepo.GetTimedOutGames()
			if err != nil {
				continue
			}

			for _, game := range games {
				b.handlers.HandleTodTimeout(game.ID, b)
			}
		}
	}()

	// Cleanup old action logs (runs every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			b.handlers.TodRepo.CleanupOldActions()
		}
	}()

	logger.Info("Truth or Dare background jobs started")
}
