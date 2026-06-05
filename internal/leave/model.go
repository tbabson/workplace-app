package leave

import "time"

type Type string
type Status string

const (
	TypeAnnual    Type = "annual"
	TypeSick      Type = "sick"
	TypeMaternity Type = "maternity"
	TypePaternity Type = "paternity"
	TypeUnpaid    Type = "unpaid"
	TypeOther     Type = "other"

	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
)

type Request struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Type       Type       `json:"type"`
	StartDate  string     `json:"start_date"`
	EndDate    string     `json:"end_date"`
	Days       int        `json:"days"`
	Reason     string     `json:"reason"`
	Status     Status     `json:"status"`
	ReviewedBy *string    `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	ReviewNote *string    `json:"review_note"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Balance struct {
	UserID string `json:"user_id"`
	Type   Type   `json:"type"`
	Year   int    `json:"year"`
	Total  int    `json:"total"`
	Used   int    `json:"used"`
}
