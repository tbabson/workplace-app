package query

import "time"

type Status string

const (
	StatusPending      Status = "pending"
	StatusAcknowledged Status = "acknowledged"
	StatusResolved     Status = "resolved"
)

type Query struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	IssuerID    string     `json:"issuer_id"`
	RecipientID string     `json:"recipient_id"`
	Status      Status     `json:"status"`
	Response    *string    `json:"response"`
	RespondedAt *time.Time `json:"responded_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
