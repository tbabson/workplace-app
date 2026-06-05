package asset

type CreateAssetDTO struct {
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	SerialNumber *string `json:"serial_number"`
	Description  *string `json:"description"`
}

type UpdateAssetDTO struct {
	Name         *string `json:"name"`
	Category     *string `json:"category"`
	SerialNumber *string `json:"serial_number"`
	Description  *string `json:"description"`
	Status       *string `json:"status"`
}

type AssignAssetDTO struct {
	UserID string `json:"user_id"`
}
