package asset

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cols = `id, name, category, serial_number, description, status, assigned_to, assigned_at, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (*Asset, error) {
	a := &Asset{}
	return a, row.Scan(&a.ID, &a.Name, &a.Category, &a.SerialNumber, &a.Description,
		&a.Status, &a.AssignedTo, &a.AssignedAt, &a.CreatedAt, &a.UpdatedAt)
}

func (r *Repository) Create(ctx context.Context, a *Asset) error {
	q := `INSERT INTO assets (name, category, serial_number, description)
	      VALUES ($1,$2,$3,$4) RETURNING ` + cols
	got, err := scan(r.db.QueryRow(ctx, q, a.Name, a.Category, a.SerialNumber, a.Description))
	if err != nil {
		return err
	}
	*a = *got
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Asset, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+cols+` FROM assets WHERE id=$1`, id))
}

func (r *Repository) List(ctx context.Context, userID string) ([]*Asset, error) {
	var q string
	var args []any
	if userID != "" {
		q = `SELECT ` + cols + ` FROM assets WHERE assigned_to=$1 ORDER BY created_at DESC`
		args = []any{userID}
	} else {
		q = `SELECT ` + cols + ` FROM assets ORDER BY created_at DESC`
	}
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Asset
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateAssetDTO) (*Asset, error) {
	a, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dto.Name != nil {
		a.Name = *dto.Name
	}
	if dto.Category != nil {
		a.Category = *dto.Category
	}
	if dto.SerialNumber != nil {
		a.SerialNumber = dto.SerialNumber
	}
	if dto.Description != nil {
		a.Description = dto.Description
	}
	if dto.Status != nil {
		a.Status = Status(*dto.Status)
	}
	a.UpdatedAt = time.Now()
	_, err = r.db.Exec(ctx,
		`UPDATE assets SET name=$1, category=$2, serial_number=$3, description=$4, status=$5, updated_at=$6 WHERE id=$7`,
		a.Name, a.Category, a.SerialNumber, a.Description, a.Status, a.UpdatedAt, id)
	return a, err
}

func (r *Repository) Assign(ctx context.Context, id, userID string) (*Asset, error) {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE assets SET assigned_to=$1, assigned_at=$2, status=$3, updated_at=$2 WHERE id=$4`,
		userID, now, StatusAssigned, id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Unassign(ctx context.Context, id string) (*Asset, error) {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE assets SET assigned_to=NULL, assigned_at=NULL, status=$1, updated_at=$2 WHERE id=$3`,
		StatusAvailable, now, id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id=$1`, id)
	return err
}
