package game_manager

import (
	"fmt"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

type GameState interface {
	Reset()
	GameType() string
}

type Room struct {
	ID       int64 // id room
	Player1  *tele.User
	Player2  *tele.User
	Turn     int64
	MsgID1   int
	MsgID2   int
	LastMove time.Time
	State    GameState
}

type RoomManager struct {
	Rooms      map[int64]*Room // rooms[id room] = room
	PlayerRoom map[int64]int64 // playerroom[player id] = id room
	Mu         sync.RWMutex
	nextID     int64
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms:      make(map[int64]*Room),
		PlayerRoom: make(map[int64]int64),
		nextID:     1,
	}
}

func (r *Room) Reset() {
	r.MsgID1 = 0
	r.MsgID2 = 0
	r.Turn = 0

	if r.State != nil {
		r.State.Reset()
	}
}

func (r *Room) StartRoom(b *tele.Bot, keyboard *tele.ReplyMarkup) {

	text := fmt.Sprintf("🎮 برای طرف مقابلت یک مورد را انتخاب کن \nنوبت %v ", r.Player1.FirstName)

	msg1, _ := b.Send(r.Player1, text, keyboard)
	r.MsgID1 = msg1.ID
	msg2, _ := b.Send(r.Player2, text, keyboard)
	r.MsgID2 = msg2.ID
}

func (rm *RoomManager) CreateRoom(player *tele.User, state GameState) *Room {

	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	id := rm.nextID
	rm.nextID++

	room := &Room{
		ID:       id,
		Player1:  player,
		Turn:     player.ID,
		LastMove: time.Now(),
		State:    state,
	}

	rm.Rooms[id] = room
	rm.PlayerRoom[player.ID] = id

	return room
}

func (rm *RoomManager) JoinRoom(id int64, player *tele.User) (*Room, bool) {

	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	room, ok := rm.Rooms[id]
	if !ok {
		return nil, false
	}

	if room.Player2 != nil {
		return nil, false
	}

	room.Player2 = player
	rm.PlayerRoom[player.ID] = id

	return room, true
}

func (rm *RoomManager) GetRoomByPlayerID(playerID int64) *Room {

	rm.Mu.RLock()
	defer rm.Mu.RUnlock()

	roomID, ok := rm.PlayerRoom[playerID]
	if !ok {
		return nil
	}

	return rm.Rooms[roomID]
}

func (rm *RoomManager) RemoveRoomByRoomID(roomID int64) {

	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	room := rm.Rooms[roomID]

	delete(rm.PlayerRoom, room.Player1.ID)

	if room.Player2 != nil {
		delete(rm.PlayerRoom, room.Player2.ID)
	}

	delete(rm.Rooms, roomID)
}

func (rm *RoomManager) RemoveRoomsByPlayerID(playerID int64) {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	roomID, ok := rm.PlayerRoom[playerID]
	if !ok {
		return // هیچ اتاقی وجود ندارد
	}

	room, ok := rm.Rooms[roomID]
	if !ok {
		delete(rm.PlayerRoom, playerID)
		return
	}

	delete(rm.PlayerRoom, room.Player1.ID)

	if room.Player2 != nil {
		delete(rm.PlayerRoom, room.Player2.ID)
	}

	delete(rm.Rooms, roomID)
}
