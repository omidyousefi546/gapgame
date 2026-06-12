package game_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v3"
)

type GameState interface {
	Reset()
	GameType() string
}

type Room struct {
	ID                     int64      `json:"id"`
	Player1                *tele.User `json:"player1"`
	Player2                *tele.User `json:"player2"`
	Player1Name            string     `json:"player1_name"`
	Player2Name            string     `json:"player2_name"`
	Turn                   int64      `json:"turn"`
	MsgID1                 int        `json:"msg_id1"`
	MsgID2                 int        `json:"msg_id2"`
	LastMove               time.Time  `json:"last_move"`
	GameOver               bool       `json:"game_over"`
	PendingGameType        string     `json:"pending_game_type"`
	PendingGameRequestedBy int64      `json:"pending_game_requested_by"`
	State                  GameState  `json:"-"`
}

func (r *Room) NameFor(playerID int64) string {
	if r.Player1 != nil && playerID == r.Player1.ID {
		if r.Player1Name != "" {
			return r.Player1Name
		}
		if r.Player1.FirstName != "" {
			return r.Player1.FirstName
		}
		return "کاربر"
	}
	if r.Player2 != nil && playerID == r.Player2.ID {
		if r.Player2Name != "" {
			return r.Player2Name
		}
		if r.Player2.FirstName != "" {
			return r.Player2.FirstName
		}
		return "کاربر"
	}
	return "کاربر"
}

type RoomManager struct {
	rdb            *redis.Client
	stateFactories map[string]func() GameState
}

func NewRoomManager(rdb *redis.Client) *RoomManager {
	return &RoomManager{
		rdb:            rdb,
		stateFactories: make(map[string]func() GameState),
	}
}

func (rm *RoomManager) RegisterGameState(gameType string, factory func() GameState) {
	rm.stateFactories[gameType] = factory
}

const (
	roomKeyPrefix    = "game:room:"
	playerRoomPrefix = "game:player_room:"
	roomTTL          = 24 * time.Hour
)

func (rm *RoomManager) roomKey(id int64) string {
	return fmt.Sprintf("%s%d", roomKeyPrefix, id)
}

func (rm *RoomManager) playerRoomKey(id int64) string {
	return fmt.Sprintf("%s%d", playerRoomPrefix, id)
}

type serializableRoom struct {
	Room
	GameType  string          `json:"game_type"`
	StateData json.RawMessage `json:"state_data"`
}

func (rm *RoomManager) SaveRoom(room *Room) error {
	ctx := context.Background()
	s := serializableRoom{
		Room: *room,
	}
	if room.State != nil {
		s.GameType = room.State.GameType()
		stateData, err := json.Marshal(room.State)
		if err != nil {
			return err
		}
		s.StateData = stateData
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	pipe := rm.rdb.TxPipeline()
	pipe.Set(ctx, rm.roomKey(room.ID), data, roomTTL)
	if room.Player1 != nil {
		pipe.Set(ctx, rm.playerRoomKey(room.Player1.ID), room.ID, roomTTL)
	}
	if room.Player2 != nil {
		pipe.Set(ctx, rm.playerRoomKey(room.Player2.ID), room.ID, roomTTL)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (rm *RoomManager) CreateRoom(player *tele.User, state GameState, p1Name string) *Room {
	ctx := context.Background()
	id, _ := rm.rdb.Incr(ctx, "game:next_id").Result()

	room := &Room{
		ID:          id,
		Player1:     player,
		Player1Name: p1Name,
		Turn:        player.ID,
		LastMove:    time.Now(),
		State:       state,
	}

	rm.SaveRoom(room)
	return room
}

func (rm *RoomManager) JoinRoom(id int64, player *tele.User, p2Name string) (*Room, bool) {
	room := rm.getRoomByID(id)
	if room == nil {
		return nil, false
	}

	if room.Player2 != nil {
		return nil, false
	}

	room.Player2 = player
	room.Player2Name = p2Name
	rm.SaveRoom(room)

	return room, true
}

func (rm *RoomManager) getRoomByID(id int64) *Room {
	ctx := context.Background()
	data, err := rm.rdb.Get(ctx, rm.roomKey(id)).Result()
	if err != nil {
		return nil
	}

	var s serializableRoom
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil
	}

	room := s.Room
	if factory, ok := rm.stateFactories[s.GameType]; ok {
		state := factory()
		if err := json.Unmarshal(s.StateData, state); err == nil {
			room.State = state
		}
	}

	return &room
}

func (rm *RoomManager) GetRoomByPlayerID(playerID int64) *Room {
	ctx := context.Background()
	roomIDStr, err := rm.rdb.Get(ctx, rm.playerRoomKey(playerID)).Result()
	if err != nil {
		return nil
	}

	id, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		return nil
	}
	return rm.getRoomByID(id)
}

func (rm *RoomManager) RemoveRoomByRoomID(roomID int64) {
	room := rm.getRoomByID(roomID)
	if room == nil {
		return
	}

	ctx := context.Background()
	pipe := rm.rdb.TxPipeline()
	pipe.Del(ctx, rm.roomKey(roomID))
	if room.Player1 != nil {
		pipe.Del(ctx, rm.playerRoomKey(room.Player1.ID))
	}
	if room.Player2 != nil {
		pipe.Del(ctx, rm.playerRoomKey(room.Player2.ID))
	}
	pipe.Exec(ctx)
}

func (rm *RoomManager) RemoveRoomsByPlayerID(playerID int64) {
	room := rm.GetRoomByPlayerID(playerID)
	if room == nil {
		return
	}
	rm.RemoveRoomByRoomID(room.ID)
}

func (r *Room) Reset() {
	r.MsgID1 = 0
	r.MsgID2 = 0
	r.Turn = 0
	r.GameOver = false
	r.PendingGameType = ""
	r.PendingGameRequestedBy = 0

	if r.State != nil {
		r.State.Reset()
	}
}

func (r *Room) StartRoom(b *tele.Bot, keyboard *tele.ReplyMarkup) {
	var text string
	if r.State != nil && r.State.GameType() == "gameDareAndTruth" {
		text = fmt.Sprintf("🎮 برای طرف مقابلت یک مورد را انتخاب کن \nنوبت %v ", r.NameFor(r.Player1.ID))
	} else if r.State != nil && r.State.GameType() == "gameRPS" {
		text = "🔄 راند 1 شروع شد. انتخاب خود را بزنید 👇"
	} else if r.State != nil && r.State.GameType() == "gameWordGuess" {
		text = "نوع بازی را انتخاب کنید 👇"
	} else {
		text = fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v (%v)", r.NameFor(r.Player1.ID), "🔴")
	}

	msg1, _ := b.Send(r.Player1, text, keyboard)
	r.MsgID1 = msg1.ID
	msg2, _ := b.Send(r.Player2, text, keyboard)
	r.MsgID2 = msg2.ID
}
