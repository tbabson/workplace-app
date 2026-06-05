package project

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusOnHold    Status = "on_hold"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

type Project struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       Status    `json:"status"`
	Progress     int       `json:"progress"`
	StartDate    *string   `json:"start_date"`
	EndDate      *string   `json:"end_date"`
	DepartmentID *string   `json:"department_id"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Assignment struct {
	ProjectID  string    `json:"project_id"`
	UserID     string    `json:"user_id"`
	AssignedAt time.Time `json:"assigned_at"`
}
