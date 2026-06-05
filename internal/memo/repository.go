package memo

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

const cols = `id, title, content, sender_id, recipient_id, department_id, read_at, created_at`

func scanMemo(row interface{ Scan(...interface{}) error }) (*Memo, error) {
	m := &Memo{}
	return m, row.Scan(
		&m.ID, &m.Title, &m.Content, &m.SenderID,
		&m.RecipientID, &m.DepartmentID, &m.ReadAt, &m.CreatedAt,
	)
}

func (r *Repository) Create(ctx context.Context, m *Memo) error {
	q := `INSERT INTO memos (title, content, sender_id, recipient_id, department_id)
	      VALUES ($1, $2, $3, $4, $5)
	      RETURNING id, read_at, created_at`
	return r.db.QueryRow(ctx, q, m.Title, m.Content, m.SenderID, m.RecipientID, m.DepartmentID).
		Scan(&m.ID, &m.ReadAt, &m.CreatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Memo, error) {
	return scanMemo(r.db.QueryRow(ctx, `SELECT `+cols+` FROM memos WHERE id = $1`, id))
}

// ListForUser returns memos visible to a user: addressed to them, their dept, or broadcast.
func (r *Repository) ListForUser(ctx context.Context, userID, deptID string) ([]*Memo, error) {
	q := `SELECT ` + cols + ` FROM memos
	      WHERE recipient_id = $1
	         OR (recipient_id IS NULL AND (department_id = $2 OR department_id IS NULL))
	      ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q, userID, deptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Memo
	for rows.Next() {
		m, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (r *Repository) ListAll(ctx context.Context) ([]*Memo, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM memos ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Memo
	for rows.Next() {
		m, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (r *Repository) MarkRead(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE memos SET read_at=$1 WHERE id=$2 AND read_at IS NULL`,
		time.Now(), id,
	)
	return err
}

func (r *Repository) Acknowledge(ctx context.Context, memoID, userID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO memo_acknowledgements (memo_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		memoID, userID,
	)
	return err
}

func (r *Repository) IsAcknowledged(ctx context.Context, memoID, userID string) (*time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx,
		`SELECT acknowledged_at FROM memo_acknowledgements WHERE memo_id=$1 AND user_id=$2`,
		memoID, userID,
	).Scan(&t)
	if err != nil {
		return nil, nil // not acknowledged yet
	}
	return &t, nil
}

func (r *Repository) ListAcknowledgements(ctx context.Context, memoID string) ([]Acknowledgement, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ma.user_id, COALESCE(u.name,''), ma.acknowledged_at
		 FROM memo_acknowledgements ma
		 LEFT JOIN users u ON u.id = ma.user_id
		 WHERE ma.memo_id = $1
		 ORDER BY ma.acknowledged_at`,
		memoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Acknowledgement
	for rows.Next() {
		var a Acknowledgement
		if err := rows.Scan(&a.UserID, &a.UserName, &a.AcknowledgedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *Repository) CountAcknowledgements(ctx context.Context, memoID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM memo_acknowledgements WHERE memo_id = $1`, memoID,
	).Scan(&count)
	return count, err
}

// CountRecipients returns the total number of users who should receive this memo.
func (r *Repository) CountRecipients(ctx context.Context, m *Memo) (int, error) {
	var q string
	var args []interface{}
	if m.RecipientID != nil {
		q = `SELECT COUNT(*) FROM users WHERE id = $1`
		args = []interface{}{*m.RecipientID}
	} else if m.DepartmentID != nil {
		q = `SELECT COUNT(*) FROM users WHERE department_id = $1 AND id != $2`
		args = []interface{}{*m.DepartmentID, m.SenderID}
	} else {
		q = `SELECT COUNT(*) FROM users WHERE id != $1`
		args = []interface{}{m.SenderID}
	}
	var count int
	err := r.db.QueryRow(ctx, q, args...).Scan(&count)
	return count, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM memos WHERE id = $1`, id)
	return err
}
