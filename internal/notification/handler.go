package notification

import (
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/gorilla/websocket"
)

type Handler struct {
	svc      *Service
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewHandler creates the notification handler. checkOrigin decides which browser
// origins may open the notification WebSocket.
func NewHandler(svc *Service, hub *Hub, checkOrigin func(*http.Request) bool) *Handler {
	return &Handler{svc: svc, hub: hub, upgrader: websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin:     checkOrigin,
	}}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix, h.list)
	mux.HandleFunc("GET "+prefix+"/unread-count", h.unreadCount)
	mux.HandleFunc("PUT "+prefix+"/{id}/read", h.markRead)
	mux.HandleFunc("PUT "+prefix+"/read-all", h.markAllRead)
	mux.HandleFunc("DELETE "+prefix+"/clear-all", h.clearAll)
	mux.HandleFunc("GET "+prefix+"/ws", h.wsConnect)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.UnreadCount(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]int{"unread": count})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkRead(r.Context(), r.PathValue("id"), middleware.GetUserID(r)); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Notification marked as read successfully")
}

func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllRead(r.Context(), middleware.GetUserID(r)); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "All notifications marked as read successfully")
}

func (h *Handler) clearAll(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearAll(r.Context(), middleware.GetUserID(r)); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "All notifications cleared successfully")
}

// wsConnect upgrades the connection and streams real-time notifications to the user.
func (h *Handler) wsConnect(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, userID)
	h.hub.register <- client

	go client.ReadPump()
	go client.WritePump()
}
