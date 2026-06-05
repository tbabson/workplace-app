package overtime

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

func scanOne(row interface {
	Scan(...interface{}) error
}) (*Overtime, error) {
	o := &Overtime{}
	err := row.Scan(
		&o.ID, &o.UserID, &o.ProjectID, &o.StartTime, &o.EndTime,
		&o.Hours, &o.HourlyRate, &o.IsPaid, &o.Notes, &o.CreatedBy,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if o.Hours != nil {
		total := *o.Hours * o.HourlyRate
		o.TotalAmount = &total
	}
	return o, nil
}

func scanWithName(row interface {
	Scan(...interface{}) error
}) (*Overtime, error) {
	o := &Overtime{}
	err := row.Scan(
		&o.ID, &o.UserID, &o.UserName, &o.ProjectID, &o.StartTime, &o.EndTime,
		&o.Hours, &o.HourlyRate, &o.IsPaid, &o.Notes, &o.CreatedBy,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if o.Hours != nil {
		total := *o.Hours * o.HourlyRate
		o.TotalAmount = &total
	}
	return o, nil
}

const insertCols = `id, user_id, project_id, start_time, end_time, hours, hourly_rate, is_paid, notes, created_by, created_at, updated_at`
const listCols = `o.id, o.user_id, COALESCE(u.name,'') AS user_name, o.project_id, o.start_time, o.end_time, o.hours, o.hourly_rate, o.is_paid, o.notes, o.created_by, o.created_at, o.updated_at`

func (r *Repository) Create(ctx context.Context, o *Overtime) (*Overtime, error) {
	q := `INSERT INTO overtime (user_id, project_id, start_time, end_time, hours, hourly_rate, notes, created_by)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	      RETURNING ` + insertCols
	return scanOne(r.db.QueryRow(ctx, q,
		o.UserID, o.ProjectID, o.StartTime, o.EndTime, o.Hours, o.HourlyRate, o.Notes, o.CreatedBy,
	))
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Overtime, error) {
	return scanWithName(r.db.QueryRow(ctx,
		`SELECT `+listCols+` FROM overtime o LEFT JOIN users u ON u.id = o.user_id WHERE o.id = $1`, id,
	))
}

func (r *Repository) List(ctx context.Context, userID string) ([]*Overtime, error) {
	base := `SELECT ` + listCols + ` FROM overtime o LEFT JOIN users u ON u.id = o.user_id`
	var q string
	var args []interface{}
	if userID != "" {
		q = base + ` WHERE o.user_id = $1 ORDER BY o.created_at DESC`
		args = append(args, userID)
	} else {
		q = base + ` ORDER BY o.created_at DESC`
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Overtime
	for rows.Next() {
		o, err := scanWithName(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (r *Repository) Update(ctx context.Context, id string, dto UpdateOvertimeDTO) (*Overtime, error) {
	o, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.EndTime != nil {
		o.EndTime = dto.EndTime
		diff := dto.EndTime.Sub(o.StartTime).Hours()
		o.Hours = &diff
	}
	if dto.HourlyRate != nil {
		o.HourlyRate = *dto.HourlyRate
	}
	if dto.Notes != nil {
		o.Notes = *dto.Notes
	}
	o.UpdatedAt = time.Now()

	q := `UPDATE overtime SET end_time=$1, hours=$2, hourly_rate=$3, notes=$4, updated_at=$5 WHERE id=$6`
	_, err = r.db.Exec(ctx, q, o.EndTime, o.Hours, o.HourlyRate, o.Notes, o.UpdatedAt, id)
	if err != nil {
		return nil, err
	}
	if o.Hours != nil {
		total := *o.Hours * o.HourlyRate
		o.TotalAmount = &total
	}
	return o, nil
}

func (r *Repository) MarkPaid(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE overtime SET is_paid=true, updated_at=$1 WHERE id=$2`, time.Now(), id,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM overtime WHERE id = $1`, id)
	return err
}
