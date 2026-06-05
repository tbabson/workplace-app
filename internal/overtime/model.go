package overtime

import "time"

type Overtime struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	UserName    string     `json:"user_name"`
	ProjectID   *string    `json:"project_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Hours       *float64   `json:"hours"`
	HourlyRate  float64    `json:"hourly_rate"`
	TotalAmount *float64   `json:"total_amount"`
	IsPaid      bool       `json:"is_paid"`
	Notes       string     `json:"notes"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
