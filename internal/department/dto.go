package department

type CreateDepartmentDTO struct {
	Name   string  `json:"name"`
	HeadID *string `json:"head_id"`
}

type UpdateDepartmentDTO struct {
	Name   *string `json:"name"`
	HeadID *string `json:"head_id"`
}
