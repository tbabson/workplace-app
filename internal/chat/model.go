package chat

import "time"

type RoomType string

const (
	RoomGeneral      RoomType = "general"
	RoomDepartmental RoomType = "departmental"
	RoomProject      RoomType = "project"
	RoomDirect       RoomType = "direct"
)

type Room struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         RoomType  `json:"type"`
	DepartmentID *string   `json:"department_id"`
	ProjectID    *string   `json:"project_id"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	ReplyToID *string   `json:"reply_to_id"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageReaction struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type RoomMember struct {
	RoomID     string     `json:"room_id"`
	UserID     string     `json:"user_id"`
	AddedAt    time.Time  `json:"added_at"`
	LastReadAt *time.Time `json:"last_read_at"`
}
