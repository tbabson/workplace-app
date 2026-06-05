package chat

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"workplace/internal/notification"
	"workplace/internal/user"
)

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool
	repo       *Repository
	userRepo   *user.Repository
	notifSvc   *notification.Service
	register   chan *Client
	unregister chan *Client
	broadcast  chan *broadcastMsg
	typing     chan *typingEvent
}

type broadcastMsg struct {
	roomID    string
	senderID  string
	content   string
	replyToID *string
}

type typingEvent struct {
	roomID   string
	senderID string
}

func NewHub(repo *Repository, userRepo *user.Repository, notifSvc *notification.Service) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		repo:       repo,
		userRepo:   userRepo,
		notifSvc:   notifSvc,
		register:   make(chan *Client, 128),
		unregister: make(chan *Client, 128),
		broadcast:  make(chan *broadcastMsg, 256),
		typing:     make(chan *typingEvent, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.roomID] == nil {
				h.rooms[client.roomID] = make(map[*Client]bool)
			}
			h.rooms[client.roomID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if members, ok := h.rooms[client.roomID]; ok {
				if members[client] {
					delete(members, client)
					close(client.send)
				}
				if len(members) == 0 {
					delete(h.rooms, client.roomID)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.deliverToRoom(msg)

		case ev := <-h.typing:
			h.fanOutTyping(ev)
		}
	}
}

func (h *Hub) deliverToRoom(msg *broadcastMsg) {
	m := &Message{RoomID: msg.roomID, SenderID: msg.senderID, Content: msg.content, ReplyToID: msg.replyToID}
	if err := h.repo.SaveMessage(context.Background(), m); err != nil {
		log.Printf("save message: %v", err)
		return
	}

	payload, _ := json.Marshal(WSMessage{
		Type:      "message",
		Content:   msg.content,
		SenderID:  msg.senderID,
		RoomID:    msg.roomID,
		MessageID: m.ID,
		ReplyToID: msg.replyToID,
	})

	h.mu.RLock()
	members := h.rooms[msg.roomID]
	connectedUsers := make(map[string]bool)
	for c := range members {
		connectedUsers[c.userID] = true
		select {
		case c.send <- payload:
		default:
			h.unregister <- c
		}
	}
	h.mu.RUnlock()

	go h.notifyOfflineMembers(msg.roomID, msg.senderID, msg.content, connectedUsers)
}

func (h *Hub) fanOutTyping(ev *typingEvent) {
	payload, _ := json.Marshal(WSMessage{
		Type:     "typing",
		SenderID: ev.senderID,
		RoomID:   ev.roomID,
	})

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[ev.roomID] {
		if c.userID == ev.senderID {
			continue
		}
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *Hub) notifyOfflineMembers(roomID, senderID, content string, connectedUsers map[string]bool) {
	ctx := context.Background()

	room, err := h.repo.FindRoomByID(ctx, roomID)
	if err != nil {
		return
	}

	memberIDs, err := h.repo.GetRoomMemberIDs(ctx, roomID)
	if err != nil {
		return
	}

	// Resolve notification title: use sender's name for DMs, room name for group rooms.
	title := "New message in " + room.Name
	if room.Type == RoomDirect {
		if sender, err := h.userRepo.FindByID(ctx, senderID); err == nil {
			title = "New message from " + sender.Name
		}
	}

	refID := roomID
	var inputs []notification.NotifyInput
	for _, uid := range memberIDs {
		if uid == senderID || connectedUsers[uid] {
			continue
		}
		inputs = append(inputs, notification.NotifyInput{
			UserID: uid,
			Type:   notification.TypeChatMessage,
			Title:  title,
			Body:   truncate(content, 100),
			RefID:  &refID,
		})
	}
	h.notifSvc.NotifyMany(ctx, inputs)
}

func (h *Hub) Broadcast(roomID, senderID, content string, replyToID *string) {
	h.broadcast <- &broadcastMsg{roomID: roomID, senderID: senderID, content: content, replyToID: replyToID}
}

func (h *Hub) SendTyping(roomID, senderID string) {
	h.typing <- &typingEvent{roomID: roomID, senderID: senderID}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
