package user

type CreateUserDTO struct {
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Password       string  `json:"password"`
	Role           Role    `json:"role"`
	Position       *string `json:"position"`
	DepartmentID   *string `json:"department_id"`
	DepartmentName *string `json:"department_name"`
}

type UpdateUserDTO struct {
	Name           *string `json:"name"`
	Role           *Role   `json:"role"`
	Position       *string `json:"position"`
	DepartmentID   *string `json:"department_id"`
	DepartmentName *string `json:"department_name"`
	IsActive       *bool   `json:"is_active"`
}
