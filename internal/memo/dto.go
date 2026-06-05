package memo

type CreateMemoDTO struct {
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	RecipientID  *string `json:"recipient_id"`
	DepartmentID *string `json:"department_id"`
}
