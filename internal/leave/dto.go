package leave

type CreateRequestDTO struct {
	Type      Type   `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Reason    string `json:"reason"`
}

type ReviewDTO struct {
	Note string `json:"note"`
}

type SetBalanceDTO struct {
	Type  Type `json:"type"`
	Year  int  `json:"year"`
	Total int  `json:"total"`
}
