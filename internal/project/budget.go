package project

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Expense struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Date        string    `json:"date"`
	LoggedBy    string    `json:"logged_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type BudgetSummary struct {
	Budget   *float64  `json:"budget"`
	Spent    float64   `json:"spent"`
	Expenses []*Expense `json:"expenses"`
}

type AddExpenseDTO struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
}

type SetBudgetDTO struct {
	Budget float64 `json:"budget"`
}

type BudgetRepository struct{ db *pgxpool.Pool }

func NewBudgetRepository(db *pgxpool.Pool) *BudgetRepository { return &BudgetRepository{db: db} }

func (r *BudgetRepository) SetBudget(ctx context.Context, projectID string, budget float64) error {
	_, err := r.db.Exec(ctx, `UPDATE projects SET budget=$1 WHERE id=$2`, budget, projectID)
	return err
}

func (r *BudgetRepository) AddExpense(ctx context.Context, e *Expense) error {
	q := `INSERT INTO project_expenses (project_id, description, amount, "date", logged_by)
	      VALUES ($1,$2,$3,$4::date,$5) RETURNING id, to_char("date", 'YYYY-MM-DD'), created_at`
	return r.db.QueryRow(ctx, q, e.ProjectID, e.Description, e.Amount, e.Date, e.LoggedBy).
		Scan(&e.ID, &e.Date, &e.CreatedAt)
}

func (r *BudgetRepository) DeleteExpense(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_expenses WHERE id=$1`, id)
	return err
}

func (r *BudgetRepository) GetSummary(ctx context.Context, projectID string) (*BudgetSummary, error) {
	var budget *float64
	r.db.QueryRow(ctx, `SELECT budget FROM projects WHERE id=$1`, projectID).Scan(&budget)

	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, description, amount, to_char("date", 'YYYY-MM-DD'), logged_by, created_at
		 FROM project_expenses WHERE project_id=$1 ORDER BY "date" DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []*Expense
	var spent float64
	for rows.Next() {
		e := &Expense{}
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Description, &e.Amount, &e.Date, &e.LoggedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		spent += e.Amount
		expenses = append(expenses, e)
	}
	return &BudgetSummary{Budget: budget, Spent: spent, Expenses: expenses}, nil
}
