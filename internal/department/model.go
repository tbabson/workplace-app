package department

import "time"

type Department struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	HeadID    *string   `json:"head_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
