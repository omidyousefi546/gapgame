package bot

import (
	"GapGame/internal/game/dare_and_truth"
	"GapGame/internal/game/dooz4"
	"GapGame/internal/game/dooz_classic"
	"GapGame/internal/game/game_manager"
	"GapGame/internal/game/rps"
	"GapGame/internal/game/word_guess"
	"GapGame/internal/service"
	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/pkg/logger"
	"GapGame/pkg/middleware"

	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type Handler struct {
	bot *tele.Bot

	users *service.UserService

	redis *session.Manager

	db *user.Repository

	rooms *game_manager.RoomManager

	log *zap.Logger
}

var rateLimitDuration = time.Second

func New(b *tele.Bot, userService *service.UserService, sm *session.Manager, repo *user.Repository, rm *game_manager.RoomManager, log *zap.Logger) *Handler {
	rm.RegisterGameState("gameDooz4Gravity", func() game_manager.GameState { return &dooz4.GameDooz4{} })
	rm.RegisterGameState("gameDooz4Normal", func() game_manager.GameState { return &dooz4.GameDooz4Normal{} })
	rm.RegisterGameState("gameDoozClassic", func() game_manager.GameState { return &dooz_classic.GameDoozClassic{} })
	rm.RegisterGameState("gameDareAndTruth", func() game_manager.GameState { return &dare_and_truth.GameDareTruth{} })
	rm.RegisterGameState("gameRPS", func() game_manager.GameState { return &rps.GameRPS{} })
	rm.RegisterGameState("gameWordGuess", func() game_manager.GameState { return &word_guess.GameWordGuess{} })

	return &Handler{
		bot:   b,
		users: userService,
		redis: sm,
		db:    repo,
		rooms: rm,
		log:   log,
	}
}

func (h *Handler) trackLastSeenMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Sender() != nil {
				go h.users.UpdateLastSeen(c.Sender().ID)
			}
			return next(c)
		}
	}
}

func (h *Handler) RegisterHandlers() {

	// Middleware

	h.bot.Use(middleware.Recovery(logger.New("panic")))

	h.bot.Use(middleware.Logging(logger.New("request")))

	h.bot.Use(middleware.RateLimit(logger.New("ratelimit"), &rateLimitDuration))

	h.bot.Use(h.trackLastSeenMiddleware())

	// Commands

	h.bot.Handle("/start", h.StartHandler, middleware.WithHandlerName("startHandler"))

	//// Help
	h.bot.Handle("/help", h.HelpHandler)

	h.bot.Handle("/help_chat", h.HelpChatHandler)
	h.bot.Handle("/help_credit", h.HelpCreditHandler)
	h.bot.Handle("/help_gps", h.HelpGpsHandler)
	h.bot.Handle("/help_profile", h.HelpProfileHandler)
	h.bot.Handle("/help_sendchat", h.HelpSendchatHandler)
	h.bot.Handle("/help_direct", h.HelpDirectHandler)
	h.bot.Handle("/help_shortcuts", h.HelpShortcutsHandler)
	h.bot.Handle("/help_onw", h.HelpOnwHandler)
	h.bot.Handle("/help_chw", h.HelpChwHandler)
	h.bot.Handle("/help_contacts", h.HelpContactsHandler)
	h.bot.Handle("/help_search", h.HelpSearchHandler)
	h.bot.Handle("/help_deleteMessage", h.HelpDeleteMessageHandler)
	h.bot.Handle("/ghavanin", h.GhavaninHandler)

	h.bot.Handle("/deleteAllContacts", h.DeleteAllContactsHandler)

	//  Main menu buttons

	h.bot.Handle(&btnStart, h.StartHandler)

	h.bot.Handle(&btnConnect, h.ConnectHandler)

	h.bot.Handle(&btnGame, h.showGamesHandler)

	h.bot.Handle(&btnSearch, h.SearchHandler)

	h.bot.Handle(&btnProfile, h.ProfileHandler)

	h.bot.Handle(&btnCoins, h.CoinsHandler)

	h.bot.Handle(&btnInvite, h.InviteHandler)

	h.bot.Handle(&btnChatGame, h.ChatGameHandler)

	h.bot.Handle(&btnHelp, h.HelpHandler)

	//  Gender selection

	h.bot.Handle(&btnMale, h.HandleGenderCallback)

	h.bot.Handle(&btnFemale, h.HandleGenderCallback)

	//  Gender Edit

	h.bot.Handle(&btnMaleEdit, h.SetGenderHandler)

	h.bot.Handle(&btnFemaleEdit, h.SetGenderHandler)

	//  Optional Completion

	h.bot.Handle(&btnStart_optional, h.StartOptionalHandler)

	h.bot.Handle(&btnSkip_optional, h.SkipOptionalHandler)

	//  Search flow

	h.bot.Handle("\fstype", h.SearchTypeHandler)
	h.bot.Handle("\fspage", h.SearchPageHandler)
	h.bot.Handle("\fsprov", h.SearchProvinceHandler)

	// // Gender filter in search

	h.bot.Handle("\fsgender", h.SearchGenderHandler)

	// // My Profile Actions
	h.bot.Handle(&btnViewGPS, h.ViewGPSHandler)
	h.bot.Handle(&btnNoGPS, h.NoGPSHandler)
	h.bot.Handle(&btnSilent, h.SilentHandler)
	h.bot.Handle(&btnEditProfile, h.EditProfileHandler)

	// Silent
	h.bot.Handle(&btnSilentForever, h.SilentForeverHandler)
	h.bot.Handle(&btnSilentHour, h.SilentHourHandler)
	h.bot.Handle(&btnSilent20, h.Silent20Handler)
	h.bot.Handle(&btnSilentOff, h.SilentOffHandler)

	// likes
	h.bot.Handle(&btnMyLikes, h.MyLikesHandler)
	h.bot.Handle(&btnLikesPage, h.LikesPageHandler)
	h.bot.Handle(&btnToggleLikes, h.ToggleLikesHandler)

	//contact
	h.bot.Handle(&btnContacts, h.MyContactsHandler)
	h.bot.Handle(&btnAddContact, h.AddContactHandler)
	h.bot.Handle(&btnRemoveContact, h.RemoveContactHandler)
	h.bot.Handle("\fcontactsPage", h.ContactsPageHandler)

	h.bot.Handle(&btnBlocksList, h.BlocksHandler) // لیست بلاک‌شده‌ها
	h.bot.Handle(&btnBlockAck, h.BlockAckHandler)
	h.bot.Handle(&btnUnblock, h.UnblockHandler)
	h.bot.Handle(&btnBlocksPage, h.BlocksPageHandler)

	h.bot.Handle("/deleteAllBlocks", h.DeleteAllBlocksHandler)
	// // Other Profile Actions

	h.bot.Handle(&btnLike, h.LikeHandler)
	h.bot.Handle(&btnDM, h.DMHandler)
	h.bot.Handle(&btnChatRequest, h.ChatRequestHandler)
	h.bot.Handle("\fbtnAcceptChat", h.AcceptChatHandler)
	h.bot.Handle("\fbtnRejectChat", h.RejectChatHandler)
	h.bot.Handle(&btnAddContact, h.AddContactHandler)
	h.bot.Handle(&btnBlocks, h.BlockHandler)
	h.bot.Handle(&btnReport, h.ReportHandler)
	h.bot.Handle(&btnNotifyOnline, h.NotifyOnlineHandler)

	// // Edit profile

	h.bot.Handle(&btnEditName, h.EditNameHandler)

	h.bot.Handle(&btnEditGender, h.EditGenderHandler)

	h.bot.Handle(&btnEditAge, h.EditAgeHandler)

	h.bot.Handle(&btnEditCity, h.EditCityHandler)

	h.bot.Handle(&btnEditProvince, h.EditProvinceHandler)

	h.bot.Handle(&btnEditPhoto, h.EditPhotoHandler)

	h.bot.Handle(&btnEditGPS, h.EditGPSHandler)

	h.bot.Handle(&btnCancelEdit, h.CancelHandler)

	// back to Edit profile
	h.bot.Handle(&btnBackToEditProfile, h.BackToEditProfileHandler)

	// // Coins

	h.bot.Handle(&btnBuyCoins, h.BuyCoinsHandler)

	h.bot.Handle(&btnInviteFriends, h.InviteHandler)

	// // Chat
	h.bot.Handle("\fctype", h.ConnectTypeHandler)
	h.bot.Handle("\fntype", h.NearbyTypeHandler)
	h.bot.Handle(&btnViewChatProfile, h.ViewChatProfileHandler)
	h.bot.Handle(&btnEndChat, h.EndChatHandler)
	h.bot.Handle(&btnConfirmEndChat, h.ConfirmEndChatHandler)
	h.bot.Handle(&btnCancelEndChat, h.CancelEndChatHandler)
	h.bot.Handle(&btnCancelQueue, h.CancelQueueHandler)

	h.bot.Handle("\fcgame_req", h.ChatGameRequestCallback)
	h.bot.Handle("\fcgame_acc", h.ChatGameAcceptCallback)
	h.bot.Handle("\fcgame_rej", h.ChatGameRejectCallback)

	// Game

	h.bot.Handle(&btnNewGame, h.showGamesHandler)
	h.bot.Handle("\fgame_select", h.selectGameHandler)
	h.bot.Handle(&btnFinishGame, h.finishGameHandler)
	h.bot.Handle(&btnDeclineFinishGame, h.declineFinishGame)
	h.bot.Handle(&btnConfirmFinishGame, h.confirmFinishGameHandler)
	h.bot.Handle(&btnRepeatGame, h.repeatGameHandler)
	h.bot.Handle(&btnCancelGame, h.cancelGameHandler)
	h.bot.Handle("\fgame_dooz4_gravity", h.moveDooz4GravityHandler)
	h.bot.Handle("\fgame_dooz4_normal", h.moveDooz4NormalHandler)
	h.bot.Handle("\fgame_dooz_classic", h.moveDoozClassicHandler)
	h.bot.Handle("\fgame_dare_and_truth", h.moveDareAndTruthHandler)
	h.bot.Handle("\fmove_rps", h.moveRPSHandler)
	h.bot.Handle("\fword_type", h.wordTypeHandler)
	h.bot.Handle("\fword_guess", h.wordGuessMoveHandler)

	// // Message types

	h.bot.Handle(tele.OnText, h.TextHandler)
	h.bot.Handle(tele.OnPhoto, h.MediaHandler)
	h.bot.Handle(tele.OnVideo, h.MediaHandler)
	h.bot.Handle(tele.OnAnimation, h.MediaHandler)
	h.bot.Handle(tele.OnAudio, h.MediaHandler)
	h.bot.Handle(tele.OnVoice, h.MediaHandler)
	h.bot.Handle(tele.OnDocument, h.MediaHandler)
	h.bot.Handle(tele.OnSticker, h.MediaHandler)
	h.bot.Handle(tele.OnVideoNote, h.MediaHandler)
	h.bot.Handle(tele.OnLocation, h.LocationHandler)

}

func (h *Handler) Start() {
	h.bot.Start()
}

func (h *Handler) Stop() {

	h.bot.Stop()

}
