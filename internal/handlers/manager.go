package handlers

import (
	"sync"

	"github.com/mroshb/game_bot/internal/config"
	"github.com/mroshb/game_bot/internal/repositories"
	"github.com/mroshb/game_bot/internal/services"
	"gorm.io/gorm"
)

type HandlerManager struct {
	Config      *config.Config
	DB          *gorm.DB
	UserRepo    *repositories.UserRepository
	CoinRepo    *repositories.CoinRepository
	MatchRepo   *repositories.MatchRepository
	FriendRepo  *repositories.FriendRepository
	GameRepo    *repositories.GameRepository
	RoomRepo    *repositories.RoomRepository
	VillageRepo *repositories.VillageRepository
	VillageSvc  *services.VillageService

	searchingUsers sync.Map // userID -> chan struct{} for cancellation
}

func NewHandlerManager(
	cfg *config.Config,
	db *gorm.DB,
	userRepo *repositories.UserRepository,
	coinRepo *repositories.CoinRepository,
	matchRepo *repositories.MatchRepository,
	friendRepo *repositories.FriendRepository,
	gameRepo *repositories.GameRepository,
	roomRepo *repositories.RoomRepository,
	villageRepo *repositories.VillageRepository,
	villageSvc *services.VillageService,
) *HandlerManager {
	return &HandlerManager{
		Config:      cfg,
		DB:          db,
		UserRepo:    userRepo,
		CoinRepo:    coinRepo,
		MatchRepo:   matchRepo,
		FriendRepo:  friendRepo,
		GameRepo:    gameRepo,
		RoomRepo:    roomRepo,
		VillageRepo: villageRepo,
		VillageSvc:  villageSvc,
	}
}
