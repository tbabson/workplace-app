package notification

import "time"

type Type string

const (
	TypeChatMessage       Type = "chat_message"
	TypeProjectAssigned   Type = "project_assigned"
	TypeProjectUnassigned Type = "project_unassigned"
	TypeProjectProgress   Type = "project_progress"
	TypeProjectStatus     Type = "project_status"
	TypeOvertimeCreated   Type = "overtime_created"
	TypeOvertimePaid      Type = "overtime_paid"
	TypeMemoReceived      Type = "memo_received"
	TypeQueryIssued       Type = "query_issued"
	TypeQueryResponded    Type = "query_responded"
	TypeAttendanceSignIn  Type = "attendance_sign_in"
	TypeAttendanceSignOut Type = "attendance_sign_out"
	TypeLeaveApproved     Type = "leave_approved"
	TypeLeaveRejected     Type = "leave_rejected"
	TypeClaimApproved     Type = "claim_approved"
	TypeClaimRejected     Type = "claim_rejected"
	TypeClaimPaid         Type = "claim_paid"
	TypeAssetAssigned     Type = "asset_assigned"
	TypeTaskAssigned      Type = "task_assigned"
	TypeTaskUpdated       Type = "task_updated"
	TypeProjectComment    Type = "project_comment"
	TypeBudgetUpdated      Type = "budget_updated"
	TypeProjectCreated     Type = "project_created"
	TypeProjectDueReminder Type = "project_due_reminder"
	TypeTaskDueReminder    Type = "task_due_reminder"
	TypeSignOutReminder    Type = "sign_out_reminder"
	TypeAttendanceAutoClosed Type = "attendance_auto_closed"
)

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      Type      `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	RefID     *string   `json:"ref_id"`   // ID of the related entity (project, room, memo, etc.)
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
