package overtime

import "time"

type CreateOvertimeDTO struct {
	UserID     string     `json:"user_id"`
	ProjectID  *string    `json:"project_id"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	HourlyRate float64    `json:"hourly_rate"`
	Notes      string     `json:"notes"`
}

type UpdateOvertimeDTO struct {
	EndTime    *time.Time `json:"end_time"`
	HourlyRate *float64   `json:"hourly_rate"`
	Notes      *string    `json:"notes"`
}
