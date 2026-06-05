package task

type CreateTaskDTO struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
	AssignedTo  *string `json:"assigned_to"`
	DueDate     *string `json:"due_date"`
}

type UpdateTaskDTO struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *Status `json:"status"`
	AssignedTo  *string `json:"assigned_to"`
	DueDate     *string `json:"due_date"`
}
