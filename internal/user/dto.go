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
	Email          *string `json:"email"`
	Role           *Role   `json:"role"`
	Position       *string `json:"position"`
	DepartmentID   *string `json:"department_id"`
	DepartmentName *string `json:"department_name"`
	IsActive       *bool   `json:"is_active"`
}

// UpdateMeDTO is what any user may change on their own profile.
type UpdateMeDTO struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Position *string `json:"position"`
}

type ChangePasswordDTO struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
