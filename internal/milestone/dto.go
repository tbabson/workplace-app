package milestone

type CreateMilestoneDTO struct {
	Title   string `json:"title"`
	DueDate string `json:"due_date"`
}

type UpdateMilestoneDTO struct {
	Title   *string `json:"title"`
	DueDate *string `json:"due_date"`
}
