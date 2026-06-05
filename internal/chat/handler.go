package chat

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
	r.Get("/rooms", h.listRooms)
	r.Post("/rooms", h.createRoom)
	r.Delete("/rooms/{id}", h.deleteRoom)
	r.Post("/dm", h.createDM)
	r.Get("/rooms/{id}/messages", h.getMessages)
	r.Delete("/messages/{msgID}", h.deleteMessage)
	r.Post("/messages/{msgID}/reactions", h.addReaction)
	r.Delete("/messages/{msgID}/reactions/{emoji}", h.removeReaction)
	r.Get("/messages/{msgID}/reactions", h.listReactions)
	r.Post("/rooms/{id}/read", h.markRead)
	r.Post("/rooms/{id}/members", h.addMember)
	r.Delete("/rooms/{id}/members/{userID}", h.removeMember)
	r.Get("/ws/{roomID}", h.wsConnect)
}

func (h *Handler) listRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.svc.ListRooms(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rooms, len(rooms))
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	var dto CreateRoomDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Dept heads always create rooms scoped to their own department
	if middleware.GetRole(r) == "dept_head" {
		deptID := middleware.GetDeptID(r)
		if deptID != "" {
			dto.DepartmentID = &deptID
		}
	}
	room, err := h.svc.CreateRoom(r.Context(), dto, middleware.GetUserID(r), middleware.GetRole(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, room)
}

func (h *Handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteRoom(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), middleware.GetRole(r))
	if err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Room deleted successfully")
}

func (h *Handler) createDM(w http.ResponseWriter, r *http.Request) {
	var dto CreateDMDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	room, err := h.svc.CreateDM(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, room)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.svc.GetMessages(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, msgs, len(msgs))
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMessage(r.Context(), chi.URLParam(r, "msgID"), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Message deleted successfully")
}

func (h *Handler) addReaction(w http.ResponseWriter, r *http.Request) {
	var dto AddReactionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.AddReaction(r.Context(), chi.URLParam(r, "msgID"), middleware.GetUserID(r), dto.Emoji); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Reaction added")
}

func (h *Handler) removeReaction(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveReaction(r.Context(), chi.URLParam(r, "msgID"), middleware.GetUserID(r), chi.URLParam(r, "emoji")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Reaction removed")
}

func (h *Handler) listReactions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListReactions(r.Context(), chi.URLParam(r, "msgID"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkRead(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Marked as read")
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	var dto AddMemberDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.AddMember(r.Context(), chi.URLParam(r, "id"), dto.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Member added to chat room successfully")
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveMember(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "userID")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Member removed from chat room successfully")
}

func (h *Handler) wsConnect(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	userID := middleware.GetUserID(r)

	ok, err := h.svc.IsMember(r.Context(), roomID, userID)
	if err != nil || !ok {
		response.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, roomID, userID)
	h.hub.register <- client

	go client.ReadPump()
	go client.WritePump()
}
