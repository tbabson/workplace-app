package claim

type CreateClaimDTO struct {
	Title       string  `json:"title"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	ReceiptURL  *string `json:"receipt_url"`
}
