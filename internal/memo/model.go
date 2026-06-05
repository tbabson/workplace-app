package memo

import "time"

type Memo struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	SenderID     string     `json:"sender_id"`
	RecipientID  *string    `json:"recipient_id"`  // nil = broadcast
	DepartmentID *string    `json:"department_id"` // nil = all departments
	ReadAt       *time.Time `json:"read_at"`
	CreatedAt    time.Time  `json:"created_at"`

	// Populated on list/get responses
	AcknowledgedAt    *time.Time       `json:"acknowledged_at,omitempty"`  // current viewer's ack time
	AcknowledgedCount int              `json:"acknowledged_count"`
	PendingCount      int              `json:"pending_count"`
	Acknowledgements  []Acknowledgement `json:"acknowledgements,omitempty"` // populated for sender
}

type Acknowledgement struct {
	UserID         string    `json:"user_id"`
	UserName       string    `json:"user_name"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}
