package user

import "time"

type Role string

const (
	RoleSuperAdmin  Role = "super_admin"
	RoleDeptHead    Role = "dept_head"
	RoleStaff       Role = "staff"
	RoleProcurement Role = "procurement"
)

type User struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	Role           Role      `json:"role"`
	Position       *string   `json:"position"`
	DepartmentID   *string   `json:"department_id"`
	DepartmentName *string   `json:"department_name"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
