package review

import "time"

type Review struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ReviewerID string    `json:"reviewer_id"`
	Period     string    `json:"period"`
	Rating     int       `json:"rating"`
	Goals      string    `json:"goals"`
	Comments   string    `json:"comments"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
