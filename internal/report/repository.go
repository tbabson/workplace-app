package report

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// ── Chart/list value types ───────────────────────────────────────────────────

type StatusCount struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type TodayAttendee struct {
	Name       string `json:"name"`
	Position   string `json:"position"`
	Department string `json:"department"`
	SignInAt   string `json:"sign_in_at"`
	SignedOut  bool   `json:"signed_out"`
}

type ProjectProgressItem struct {
	Title    string `json:"title"`
	Progress int    `json:"progress"`
	Status   string `json:"status"`
}

// ── Main dashboard payload ───────────────────────────────────────────────────

type DashboardData struct {
	// Org-wide — super_admin
	TotalUsers        int `json:"total_users"`
	TotalProjects     int `json:"total_projects"`
	ActiveProjects    int `json:"active_projects"`
	CompletedProjects int `json:"completed_projects"`
	PendingLeaves     int `json:"pending_leaves"`
	PendingClaims     int `json:"pending_claims"`
	TotalDepartments  int `json:"total_departments"`
	SignedInToday     int `json:"signed_in_today"`
	TotalAssets       int `json:"total_assets"`
	AssignedAssets    int `json:"assigned_assets"`

	// Dept-level — dept_head
	DeptMembers        int `json:"dept_members"`
	DeptActiveProjects int `json:"dept_active_projects"`
	DeptSignedInToday  int `json:"dept_signed_in_today"`
	DeptPendingLeaves  int `json:"dept_pending_leaves"`

	// Personal — staff / procurement
	MyProjects      int `json:"my_projects"`
	MyPendingTasks  int `json:"my_pending_tasks"`
	MyPendingLeaves int `json:"my_pending_leaves"`
	MyPendingClaims int `json:"my_pending_claims"`

	// Charts (populated based on role)
	ProjectsByStatus         []StatusCount        `json:"projects_by_status"`
	WeeklyAttendance         []DayCount           `json:"weekly_attendance"`
	MonthlyAttendance        []DayCount           `json:"monthly_attendance"`
	PersonalWeeklyAttendance []DayCount           `json:"personal_weekly_attendance"`
	TodayAttendees           []TodayAttendee      `json:"today_attendees"`
	ProjectProgress          []ProjectProgressItem `json:"project_progress"`
}

// ── Report row types ─────────────────────────────────────────────────────────

type AttendanceRow struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	UserName    string     `json:"user_name"`
	Date        string     `json:"date"`
	SignInAt    *time.Time `json:"sign_in_at"`
	SignOutAt   *time.Time `json:"sign_out_at"`
	HoursWorked float64    `json:"hours_worked"`
}

type OvertimeRow struct {
	UserID        string  `json:"user_id"`
	UserName      string  `json:"user_name"`
	Date          string  `json:"date"`
	OvertimeHours float64 `json:"overtime_hours"`
}

// ── Dashboard ────────────────────────────────────────────────────────────────

func (r *Repository) Dashboard(ctx context.Context, userID, role, deptID string) (*DashboardData, error) {
	d := &DashboardData{}
	switch role {
	case "super_admin":
		return r.adminDashboard(ctx, d)
	case "dept_head":
		return r.deptHeadDashboard(ctx, d, deptID)
	default:
		return r.staffDashboard(ctx, d, userID)
	}
}

func (r *Repository) adminDashboard(ctx context.Context, d *DashboardData) (*DashboardData, error) {
	for _, c := range []struct {
		q    string
		dest *int
	}{
		{`SELECT COUNT(*) FROM users`, &d.TotalUsers},
		{`SELECT COUNT(*) FROM projects`, &d.TotalProjects},
		{`SELECT COUNT(*) FROM projects WHERE status NOT IN ('completed','cancelled')`, &d.ActiveProjects},
		{`SELECT COUNT(*) FROM projects WHERE status='completed'`, &d.CompletedProjects},
		{`SELECT COUNT(*) FROM leave_requests WHERE status='pending'`, &d.PendingLeaves},
		{`SELECT COUNT(*) FROM expense_claims WHERE status='pending'`, &d.PendingClaims},
		{`SELECT COUNT(*) FROM departments`, &d.TotalDepartments},
		{`SELECT COUNT(*) FROM attendance WHERE date=CURRENT_DATE AND sign_in_at IS NOT NULL`, &d.SignedInToday},
		{`SELECT COUNT(*) FROM assets`, &d.TotalAssets},
		{`SELECT COUNT(*) FROM assets WHERE status='assigned'`, &d.AssignedAssets},
	} {
		if err := r.db.QueryRow(ctx, c.q).Scan(c.dest); err != nil {
			return nil, err
		}
	}

	// Projects by status
	rows, err := r.db.Query(ctx, `SELECT status::text, COUNT(*)::int FROM projects GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s StatusCount
		if err := rows.Scan(&s.Name, &s.Value); err != nil {
			rows.Close()
			return nil, err
		}
		d.ProjectsByStatus = append(d.ProjectsByStatus, s)
	}
	rows.Close()

	// Weekly attendance (all staff)
	rows, err = r.db.Query(ctx, `
		SELECT to_char(gs.day::date, 'Dy'), COUNT(a.user_id)::int
		FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') gs(day)
		LEFT JOIN attendance a ON a.date = gs.day::date AND a.sign_in_at IS NOT NULL
		GROUP BY gs.day ORDER BY gs.day`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DayCount
		if err := rows.Scan(&c.Day, &c.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.WeeklyAttendance = append(d.WeeklyAttendance, c)
	}
	rows.Close()

	// Today's attendees
	rows, err = r.db.Query(ctx, `
		SELECT u.name, COALESCE(u.position,''), COALESCE(u.department_name,''),
		       to_char(a.sign_in_at, 'HH12:MI AM'), a.sign_out_at IS NOT NULL
		FROM attendance a JOIN users u ON u.id = a.user_id
		WHERE a.date = CURRENT_DATE AND a.sign_in_at IS NOT NULL
		ORDER BY a.sign_in_at`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t TodayAttendee
		if err := rows.Scan(&t.Name, &t.Position, &t.Department, &t.SignInAt, &t.SignedOut); err != nil {
			rows.Close()
			return nil, err
		}
		d.TodayAttendees = append(d.TodayAttendees, t)
	}
	rows.Close()

	// Monthly attendance trend (last 30 days, grouped by week)
	rows, err = r.db.Query(ctx, `
		SELECT to_char(gs.week, 'Mon DD'), COUNT(a.user_id)::int
		FROM generate_series(CURRENT_DATE - INTERVAL '27 days', CURRENT_DATE, INTERVAL '7 days') gs(week)
		LEFT JOIN attendance a ON a.date >= gs.week::date AND a.date < (gs.week + INTERVAL '7 days')::date
		    AND a.sign_in_at IS NOT NULL
		GROUP BY gs.week ORDER BY gs.week`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DayCount
		if err := rows.Scan(&c.Day, &c.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.MonthlyAttendance = append(d.MonthlyAttendance, c)
	}
	rows.Close()

	// Top active projects by progress
	rows, err = r.db.Query(ctx, `
		SELECT title, progress, status::text FROM projects
		WHERE status NOT IN ('completed','cancelled')
		ORDER BY progress DESC LIMIT 8`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p ProjectProgressItem
		if err := rows.Scan(&p.Title, &p.Progress, &p.Status); err != nil {
			rows.Close()
			return nil, err
		}
		d.ProjectProgress = append(d.ProjectProgress, p)
	}
	rows.Close()

	return d, nil
}

func (r *Repository) deptHeadDashboard(ctx context.Context, d *DashboardData, deptID string) (*DashboardData, error) {
	if deptID == "" {
		return r.adminDashboard(ctx, d)
	}

	for _, c := range []struct {
		q    string
		dest *int
	}{
		{`SELECT COUNT(*) FROM users WHERE department_id=$1`, &d.DeptMembers},
		{`SELECT COUNT(*) FROM projects WHERE department_id=$1 AND status NOT IN ('completed','cancelled')`, &d.DeptActiveProjects},
		{`SELECT COUNT(*) FROM attendance a JOIN users u ON u.id=a.user_id WHERE a.date=CURRENT_DATE AND a.sign_in_at IS NOT NULL AND u.department_id=$1`, &d.DeptSignedInToday},
		{`SELECT COUNT(*) FROM leave_requests lr JOIN users u ON u.id=lr.user_id WHERE lr.status='pending' AND u.department_id=$1`, &d.DeptPendingLeaves},
	} {
		if err := r.db.QueryRow(ctx, c.q, deptID).Scan(c.dest); err != nil {
			return nil, err
		}
	}

	// Dept projects by status
	rows, err := r.db.Query(ctx,
		`SELECT status::text, COUNT(*)::int FROM projects WHERE department_id=$1 GROUP BY status ORDER BY status`, deptID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s StatusCount
		if err := rows.Scan(&s.Name, &s.Value); err != nil {
			rows.Close()
			return nil, err
		}
		d.ProjectsByStatus = append(d.ProjectsByStatus, s)
	}
	rows.Close()

	// Dept weekly attendance
	rows, err = r.db.Query(ctx, `
		SELECT to_char(gs.day::date, 'Dy'), COUNT(a.user_id)::int
		FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') gs(day)
		LEFT JOIN attendance a ON a.date = gs.day::date AND a.sign_in_at IS NOT NULL
		    AND a.user_id IN (SELECT id FROM users WHERE department_id=$1)
		GROUP BY gs.day ORDER BY gs.day`, deptID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DayCount
		if err := rows.Scan(&c.Day, &c.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.WeeklyAttendance = append(d.WeeklyAttendance, c)
	}
	rows.Close()

	// Dept today's attendees
	rows, err = r.db.Query(ctx, `
		SELECT u.name, COALESCE(u.position,''), COALESCE(u.department_name,''),
		       to_char(a.sign_in_at, 'HH12:MI AM'), a.sign_out_at IS NOT NULL
		FROM attendance a JOIN users u ON u.id = a.user_id
		WHERE a.date = CURRENT_DATE AND a.sign_in_at IS NOT NULL AND u.department_id=$1
		ORDER BY a.sign_in_at`, deptID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t TodayAttendee
		if err := rows.Scan(&t.Name, &t.Position, &t.Department, &t.SignInAt, &t.SignedOut); err != nil {
			rows.Close()
			return nil, err
		}
		d.TodayAttendees = append(d.TodayAttendees, t)
	}
	rows.Close()

	// Dept monthly attendance trend
	rows, err = r.db.Query(ctx, `
		SELECT to_char(gs.week, 'Mon DD'), COUNT(a.user_id)::int
		FROM generate_series(CURRENT_DATE - INTERVAL '27 days', CURRENT_DATE, INTERVAL '7 days') gs(week)
		LEFT JOIN attendance a ON a.date >= gs.week::date AND a.date < (gs.week + INTERVAL '7 days')::date
		    AND a.sign_in_at IS NOT NULL
		    AND a.user_id IN (SELECT id FROM users WHERE department_id=$1)
		GROUP BY gs.week ORDER BY gs.week`, deptID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DayCount
		if err := rows.Scan(&c.Day, &c.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.MonthlyAttendance = append(d.MonthlyAttendance, c)
	}
	rows.Close()

	// Dept project progress
	rows, err = r.db.Query(ctx, `
		SELECT title, progress, status::text FROM projects
		WHERE department_id=$1 AND status NOT IN ('completed','cancelled')
		ORDER BY progress DESC LIMIT 8`, deptID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p ProjectProgressItem
		if err := rows.Scan(&p.Title, &p.Progress, &p.Status); err != nil {
			rows.Close()
			return nil, err
		}
		d.ProjectProgress = append(d.ProjectProgress, p)
	}
	rows.Close()

	return d, nil
}

func (r *Repository) staffDashboard(ctx context.Context, d *DashboardData, userID string) (*DashboardData, error) {
	for _, c := range []struct {
		q    string
		dest *int
	}{
		{`SELECT COUNT(DISTINCT project_id) FROM project_assignments WHERE user_id=$1`, &d.MyProjects},
		{`SELECT COUNT(*) FROM project_tasks WHERE assigned_to=$1 AND status!='done'`, &d.MyPendingTasks},
		{`SELECT COUNT(*) FROM leave_requests WHERE user_id=$1 AND status='pending'`, &d.MyPendingLeaves},
		{`SELECT COUNT(*) FROM expense_claims WHERE user_id=$1 AND status='pending'`, &d.MyPendingClaims},
	} {
		if err := r.db.QueryRow(ctx, c.q, userID).Scan(c.dest); err != nil {
			return nil, err
		}
	}

	// Personal attendance last 7 days
	rows, err := r.db.Query(ctx, `
		SELECT to_char(gs.day::date, 'Dy'),
		       CASE WHEN a.sign_in_at IS NOT NULL THEN 1 ELSE 0 END
		FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') gs(day)
		LEFT JOIN attendance a ON a.date = gs.day::date AND a.user_id=$1
		ORDER BY gs.day`, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DayCount
		if err := rows.Scan(&c.Day, &c.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.PersonalWeeklyAttendance = append(d.PersonalWeeklyAttendance, c)
	}
	rows.Close()

	// Assigned project progress
	rows, err = r.db.Query(ctx, `
		SELECT p.title, p.progress, p.status::text
		FROM projects p JOIN project_assignments pa ON pa.project_id = p.id
		WHERE pa.user_id=$1 AND p.status NOT IN ('completed','cancelled')
		ORDER BY p.progress DESC LIMIT 6`, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p ProjectProgressItem
		if err := rows.Scan(&p.Title, &p.Progress, &p.Status); err != nil {
			rows.Close()
			return nil, err
		}
		d.ProjectProgress = append(d.ProjectProgress, p)
	}
	rows.Close()

	return d, nil
}

// ── Reports ──────────────────────────────────────────────────────────────────

func (r *Repository) AttendanceReport(ctx context.Context, from, to string) ([]*AttendanceRow, error) {
	q := `SELECT a.id, a.user_id, u.name, a.date::text, a.sign_in_at, a.sign_out_at,
	             CASE WHEN a.sign_in_at IS NOT NULL AND a.sign_out_at IS NOT NULL
	                  THEN EXTRACT(EPOCH FROM (a.sign_out_at - a.sign_in_at))/3600
	                  ELSE 0 END AS hours_worked
	      FROM attendance a JOIN users u ON u.id = a.user_id
	      WHERE a.date BETWEEN $1 AND $2
	      ORDER BY a.date DESC, u.name`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AttendanceRow
	for rows.Next() {
		row := &AttendanceRow{}
		if err := rows.Scan(&row.ID, &row.UserID, &row.UserName, &row.Date,
			&row.SignInAt, &row.SignOutAt, &row.HoursWorked); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, nil
}

func (r *Repository) OvertimeReport(ctx context.Context, from, to string, _ float64) ([]*OvertimeRow, error) {
	q := `SELECT o.user_id, COALESCE(u.name, '') AS user_name,
	             o.start_time::date::text AS date,
	             COALESCE(
	               o.hours,
	               CASE WHEN o.end_time IS NOT NULL
	                    THEN EXTRACT(EPOCH FROM (o.end_time - o.start_time)) / 3600.0
	               END,
	               0
	             ) AS overtime_hours
	      FROM overtime o
	      LEFT JOIN users u ON u.id = o.user_id
	      WHERE o.start_time::date BETWEEN $1 AND $2
	      ORDER BY o.start_time DESC`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*OvertimeRow
	for rows.Next() {
		row := &OvertimeRow{}
		if err := rows.Scan(&row.UserID, &row.UserName, &row.Date, &row.OvertimeHours); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, nil
}
