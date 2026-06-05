package project

type CreateProjectDTO struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Status       Status  `json:"status"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	DepartmentID *string `json:"department_id"`
}

type UpdateProjectDTO struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Status       *Status `json:"status"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	DepartmentID *string `json:"department_id"`
}

type UpdateProgressDTO struct {
	Progress int `json:"progress"`
}

type AssignDTO struct {
	UserID string `json:"user_id"`
}
