package review

type CreateReviewDTO struct {
	UserID   string `json:"user_id"`
	Period   string `json:"period"`
	Rating   int    `json:"rating"`
	Goals    string `json:"goals"`
	Comments string `json:"comments"`
}

type UpdateReviewDTO struct {
	Period   *string `json:"period"`
	Rating   *int    `json:"rating"`
	Goals    *string `json:"goals"`
	Comments *string `json:"comments"`
}
