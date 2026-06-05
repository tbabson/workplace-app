package chat

type CreateRoomDTO struct {
	Name         string   `json:"name"`
	Type         RoomType `json:"type"`
	DepartmentID *string  `json:"department_id"`
	ProjectID    *string  `json:"project_id"`
}

type AddMemberDTO struct {
	UserID string `json:"user_id"`
}

type CreateDMDTO struct {
	UserID string `json:"user_id"`
}

type AddReactionDTO struct {
	Emoji string `json:"emoji"`
}

// WSMessage is sent over the WebSocket connection.
// type field values: "message", "typing", "read", "reaction_add", "reaction_remove", "message_delete"
type WSMessage struct {
	Type      string  `json:"type"`
	Content   string  `json:"content,omitempty"`
	SenderID  string  `json:"sender_id,omitempty"`
	RoomID    string  `json:"room_id,omitempty"`
	MessageID string  `json:"message_id,omitempty"`
	ReplyToID *string `json:"reply_to_id,omitempty"`
	Emoji     string  `json:"emoji,omitempty"`
}
