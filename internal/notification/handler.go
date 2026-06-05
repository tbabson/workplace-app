package notification

import (
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	svc *Service
	hub *Hub
}

func NewHandler(svc *Service, hub *Hub) *Handler {
	return &Handler{svc: svc, hub: hub}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/unread-count", h.unreadCount)
	r.Put("/{id}/read", h.markRead)
	r.Put("/read-all", h.markAllRead)
	r.Delete("/clear-all", h.clearAll)
	r.Get("/ws", h.wsConnect)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.UnreadCount(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]int{"unread": count})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkRead(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Notification marked as read successfully")
}

func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllRead(r.Context(), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "All notifications marked as read successfully")
}

func (h *Handler) clearAll(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearAll(r.Context(), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "All notifications cleared successfully")
}

// wsConnect upgrades the connection and streams real-time notifications to the user.
func (h *Handler) wsConnect(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, userID)
	h.hub.register <- client

	go client.ReadPump()
	go client.WritePump()
}
