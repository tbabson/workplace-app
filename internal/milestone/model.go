package milestone

import "time"

type Milestone struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Title      string     `json:"title"`
	DueDate    string     `json:"due_date"`
	AchievedAt *time.Time `json:"achieved_at"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}
