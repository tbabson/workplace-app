package department

type CreateDepartmentDTO struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
	HeadID   *string `json:"head_id"`
}

// UpdateDepartmentDTO: send "parent_id": "" to make a sub-department top-level
// again, and "head_id": "" to remove the head.
type UpdateDepartmentDTO struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id"`
	HeadID   *string `json:"head_id"`
}
