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
	kick       chan kickReq
}

type kickReq struct {
	roomID string
	userID string
}

// broadcastMsg is a message that has already been saved.
type broadcastMsg struct {
	msg *Message
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
		kick:       make(chan kickReq, 64),
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
			h.removeLocked(client)
			h.mu.Unlock()

		case k := <-h.kick:
			h.mu.Lock()
			for c := range h.rooms[k.roomID] {
				if c.userID == k.userID {
					h.removeLocked(c)
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

// removeLocked unregisters a client and closes its send channel, which makes its
// WritePump close the connection. Callers must hold h.mu for writing.
func (h *Hub) removeLocked(c *Client) {
	members, ok := h.rooms[c.roomID]
	if !ok || !members[c] {
		return
	}
	delete(members, c)
	close(c.send)
	if len(members) == 0 {
		delete(h.rooms, c.roomID)
	}
}

func (h *Hub) deliverToRoom(b *broadcastMsg) {
	m := b.msg

	payload, _ := json.Marshal(WSMessage{
		Type:      "message",
		Content:   m.Content,
		SenderID:  m.SenderID,
		RoomID:    m.RoomID,
		MessageID: m.ID,
		ReplyToID: m.ReplyToID,
	})

	// This runs on the hub goroutine, so it must never send to the hub's own
	// channels. Clients whose buffers are full are dropped directly.
	h.mu.Lock()
	connectedUsers := make(map[string]bool)
	for c := range h.rooms[m.RoomID] {
		select {
		case c.send <- payload:
			connectedUsers[c.userID] = true
		default:
			h.removeLocked(c)
		}
	}
	h.mu.Unlock()

	go h.notifyOfflineMembers(m.RoomID, m.SenderID, m.Content, connectedUsers)
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

// Broadcast saves a message and fans it out to the room. It is called from each
// client's ReadPump goroutine, so the database write doesn't stall the hub.
func (h *Hub) Broadcast(roomID, senderID, content string, replyToID *string) {
	m := &Message{RoomID: roomID, SenderID: senderID, Content: content, ReplyToID: replyToID}
	if err := h.repo.SaveMessage(context.Background(), m); err != nil {
		log.Printf("save message: %v", err)
		return
	}
	h.broadcast <- &broadcastMsg{msg: m}
}

// Kick disconnects every live connection userID has to roomID, e.g. after they
// are removed from the room.
func (h *Hub) Kick(roomID, userID string) {
	h.kick <- kickReq{roomID: roomID, userID: userID}
}

func (h *Hub) SendTyping(roomID, senderID string) {
	h.typing <- &typingEvent{roomID: roomID, senderID: senderID}
}

// truncate shortens s to at most n characters without splitting a UTF-8 sequence.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
