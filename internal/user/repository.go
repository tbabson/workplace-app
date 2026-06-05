package user

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u *User) error {
	q := `INSERT INTO users (name, email, password_hash, role, position, department_id, department_name)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)
	      RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		u.Name, u.Email, u.PasswordHash, u.Role, u.Position, u.DepartmentID, u.DepartmentName,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) {
	q := `SELECT id, name, email, password_hash, role, position, department_id, department_name, is_active, created_at, updated_at
	      FROM users WHERE id = $1`
	u := &User{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role,
		&u.Position, &u.DepartmentID, &u.DepartmentName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	q := `SELECT id, name, email, password_hash, role, position, department_id, department_name, is_active, created_at, updated_at
	      FROM users WHERE email = $1`
	u := &User{}
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role,
		&u.Position, &u.DepartmentID, &u.DepartmentName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context) ([]*User, error) {
	q := `SELECT id, name, email, role, position, department_id, department_name, is_active, created_at, updated_at
	      FROM users ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role,
			&u.Position, &u.DepartmentID, &u.DepartmentName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) ListByDepartment(ctx context.Context, deptID string) ([]*User, error) {
	q := `SELECT id, name, email, role, position, department_id, department_name, is_active, created_at, updated_at
	      FROM users WHERE department_id = $1 ORDER BY name`
	rows, err := r.db.Query(ctx, q, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role,
			&u.Position, &u.DepartmentID, &u.DepartmentName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) ListByRole(ctx context.Context, role string) ([]*User, error) {
	q := `SELECT id, name, email, role, position, department_id, department_name, is_active, created_at, updated_at
	      FROM users WHERE role = $1 ORDER BY name`
	rows, err := r.db.Query(ctx, q, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role,
			&u.Position, &u.DepartmentID, &u.DepartmentName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateUserDTO) (*User, error) {
	u, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Name != nil {
		u.Name = *dto.Name
	}
	if dto.Role != nil {
		u.Role = *dto.Role
	}
	if dto.Position != nil {
		u.Position = dto.Position
	}
	if dto.DepartmentID != nil {
		u.DepartmentID = dto.DepartmentID
	}
	if dto.DepartmentName != nil {
		u.DepartmentName = dto.DepartmentName
	}
	if dto.IsActive != nil {
		u.IsActive = *dto.IsActive
	}
	u.UpdatedAt = time.Now()

	q := `UPDATE users SET name=$1, role=$2, position=$3, department_id=$4, department_name=$5, is_active=$6, updated_at=$7
	      WHERE id=$8 RETURNING updated_at`
	return u, r.db.QueryRow(ctx, q,
		u.Name, u.Role, u.Position, u.DepartmentID, u.DepartmentName, u.IsActive, u.UpdatedAt, id,
	).Scan(&u.UpdatedAt)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
