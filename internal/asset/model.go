package asset

import "time"

type Status string

const (
	StatusAvailable Status = "available"
	StatusAssigned  Status = "assigned"
	StatusMaintenance Status = "maintenance"
	StatusRetired   Status = "retired"
)

type Asset struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	SerialNumber *string    `json:"serial_number"`
	Description  *string    `json:"description"`
	Status       Status     `json:"status"`
	AssignedTo   *string    `json:"assigned_to"`
	AssignedAt   *time.Time `json:"assigned_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
