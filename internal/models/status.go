package models

type StatusResponse struct {
	ReportID                int                    `json:"report_id"`
	StatusSummary           StatusSummary          `json:"status_summary"`
	CalculationStartTimeUTC string                 `json:"calculation_start_time_utc"`
	CustomerSpecific        map[string]interface{} `json:"customer_specific,omitempty"`
}

type StatusSummary struct {
	Created    int `json:"Created"`
	Running    int `json:"Running"`
	Computed   int `json:"Computed"`
	Successful int `json:"Successful"`
}

func (s *StatusSummary) IsComplete() bool {
	return s.Computed == s.Created && s.Created > 0
}

func (s *StatusSummary) Progress() float64 {
	if s.Created == 0 {
		return 0.0
	}
	return float64(s.Computed) / float64(s.Created)
}
