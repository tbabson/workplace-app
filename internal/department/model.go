package department

import "time"

type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ParentID is set for a sub-department (e.g. HR under Admin).
	ParentID  *string   `json:"parent_id"`
	HeadID    *string   `json:"head_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
