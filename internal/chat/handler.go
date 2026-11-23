package chat

import (
	"encoding/json"
	"errors"
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

// NewHandler creates the chat handler. checkOrigin decides which browser origins
// may open WebSocket connections (the auth cookie is sent automatically, so
// accepting any origin would allow cross-site WebSocket hijacking).
func NewHandler(svc *Service, hub *Hub, checkOrigin func(*http.Request) bool) *Handler {
	return &Handler{svc: svc, hub: hub, upgrader: websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrRoomNotFound):
		response.Error(w, http.StatusNotFound, err.Error())
	default:
		response.ServerError(w, r, err)
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/rooms", h.listRooms)
	mux.HandleFunc("POST "+prefix+"/rooms", h.createRoom)
	mux.HandleFunc("DELETE "+prefix+"/rooms/{id}", h.deleteRoom)
	mux.HandleFunc("POST "+prefix+"/dm", h.createDM)
	mux.HandleFunc("GET "+prefix+"/rooms/{id}/messages", h.getMessages)
	mux.HandleFunc("DELETE "+prefix+"/messages/{msgID}", h.deleteMessage)
	mux.HandleFunc("POST "+prefix+"/messages/{msgID}/reactions", h.addReaction)
	mux.HandleFunc("DELETE "+prefix+"/messages/{msgID}/reactions/{emoji}", h.removeReaction)
	mux.HandleFunc("GET "+prefix+"/messages/{msgID}/reactions", h.listReactions)
	mux.HandleFunc("POST "+prefix+"/rooms/{id}/read", h.markRead)
	mux.HandleFunc("POST "+prefix+"/rooms/{id}/members", h.addMember)
	mux.HandleFunc("DELETE "+prefix+"/rooms/{id}/members/{userID}", h.removeMember)
	mux.HandleFunc("GET "+prefix+"/ws/{roomID}", h.wsConnect)
}

func (h *Handler) listRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.svc.ListRooms(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
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
	if middleware.HasRole(r, middleware.RoleDeptHead) {
		deptID := middleware.GetDeptID(r)
		if deptID != "" {
			dto.DepartmentID = &deptID
		}
	}
	room, err := h.svc.CreateRoom(r.Context(), dto, middleware.GetUserID(r), middleware.GetRole(r), middleware.GetDeptID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, room)
}

func (h *Handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteRoom(r.Context(), r.PathValue("id"), middleware.GetUserID(r), middleware.GetRole(r))
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
	msgs, err := h.svc.GetMessages(r.Context(), r.PathValue("id"), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, msgs, len(msgs))
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMessage(r.Context(), r.PathValue("msgID"), middleware.GetUserID(r)); err != nil {
		response.ServerError(w, r, err)
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
	if err := h.svc.AddReaction(r.Context(), r.PathValue("msgID"), middleware.GetUserID(r), dto.Emoji); err != nil {
		if errors.Is(err, ErrForbidden) {
			writeErr(w, r, err)
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Reaction added")
}

func (h *Handler) removeReaction(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveReaction(r.Context(), r.PathValue("msgID"), middleware.GetUserID(r), r.PathValue("emoji")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Reaction removed")
}

func (h *Handler) listReactions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListReactions(r.Context(), r.PathValue("msgID"), middleware.GetUserID(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkRead(r.Context(), r.PathValue("id"), middleware.GetUserID(r)); err != nil {
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
	if err := h.svc.AddMember(r.Context(), r.PathValue("id"), dto.UserID, middleware.GetUserID(r), middleware.GetRole(r)); err != nil {
		writeErr(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Member added to chat room successfully")
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	roomID, userID := r.PathValue("id"), r.PathValue("userID")
	if err := h.svc.RemoveMember(r.Context(), roomID, userID, middleware.GetUserID(r), middleware.GetRole(r)); err != nil {
		writeErr(w, r, err)
		return
	}
	h.hub.Kick(roomID, userID)
	response.Message(w, http.StatusOK, "Member removed from chat room successfully")
}

func (h *Handler) wsConnect(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	userID := middleware.GetUserID(r)

	ok, err := h.svc.IsMember(r.Context(), roomID, userID)
	if err != nil || !ok {
		response.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, roomID, userID)
	h.hub.register <- client

	go client.ReadPump()
	go client.WritePump()
}
