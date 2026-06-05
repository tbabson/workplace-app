package attendance

import "time"

type Attendance struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	SignInAt   *time.Time `json:"sign_in_at"`
	SignOutAt  *time.Time `json:"sign_out_at"`
	SignInLat  *float64   `json:"sign_in_lat"`
	SignInLng  *float64   `json:"sign_in_lng"`
	Date       string     `json:"date"`
	AutoClosed bool       `json:"auto_closed"`
	DeviceID   *string    `json:"device_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
