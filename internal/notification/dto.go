package notification

// NotifyInput carries everything needed to create and deliver one notification.
type NotifyInput struct {
	UserID    string
	Email     string // recipient email — empty = skip email
	Name      string // recipient name — used in email greeting
	Type      Type
	Title     string
	Body      string
	RefID     *string // optional reference entity ID
	SendEmail bool
}
