package query

type CreateQueryDTO struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	RecipientID string `json:"recipient_id"`
}

type RespondDTO struct {
	Response string `json:"response"`
	Status   Status `json:"status"`
}
