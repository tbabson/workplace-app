package attendance

type SignInDTO struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	DeviceID  string  `json:"device_id"`
}
