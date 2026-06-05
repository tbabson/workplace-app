package notification

import (
	"encoding/json"
	"sync"
)

// Hub routes real-time notifications to individual connected users.
// Unlike the chat hub (room-based), this hub is user-based: one slot per user.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client // userID → active WS client
	register   chan *Client
	unregister chan *Client
	push       chan *envelope
}

type envelope struct {
	userID  string
	payload []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		push:       make(chan *envelope, 512),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			// If user already has a connection, close the old one
			if old, ok := h.clients[c.userID]; ok {
				close(old.send)
			}
			h.clients[c.userID] = c
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if h.clients[c.userID] == c {
				delete(h.clients, c.userID)
				close(c.send)
			}
			h.mu.Unlock()

		case env := <-h.push:
			h.mu.RLock()
			c, ok := h.clients[env.userID]
			h.mu.RUnlock()
			if ok {
				select {
				case c.send <- env.payload:
				default:
					h.unregister <- c
				}
			}
		}
	}
}

// Push delivers a notification to a connected user's WebSocket in real-time.
func (h *Hub) Push(n *Notification) {
	payload, _ := json.Marshal(n)
	h.push <- &envelope{userID: n.UserID, payload: payload}
}
