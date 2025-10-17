package models

import (
	"encoding/json"
	"testing"
)

func TestStatusSummary_IsComplete(t *testing.T) {
	tests := []struct {
		name    string
		summary StatusSummary
		want    bool
	}{
		{
			name: "all computed",
			summary: StatusSummary{
				Created:  41,
				Computed: 41,
			},
			want: true,
		},
		{
			name: "none computed",
			summary: StatusSummary{
				Created:  41,
				Computed: 0,
			},
			want: false,
		},
		{
			name: "partial progress",
			summary: StatusSummary{
				Created:  41,
				Computed: 20,
			},
			want: false,
		},
		{
			name: "zero created",
			summary: StatusSummary{
				Created:  0,
				Computed: 0,
			},
			want: false,
		},
		{
			name: "computed exceeds created",
			summary: StatusSummary{
				Created:  10,
				Computed: 15,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusSummary_Progress(t *testing.T) {
	tests := []struct {
		name    string
		summary StatusSummary
		want    float64
	}{
		{
			name: "complete",
			summary: StatusSummary{
				Created:  41,
				Computed: 41,
			},
			want: 1.0,
		},
		{
			name: "half complete",
			summary: StatusSummary{
				Created:  100,
				Computed: 50,
			},
			want: 0.5,
		},
		{
			name: "none complete",
			summary: StatusSummary{
				Created:  41,
				Computed: 0,
			},
			want: 0.0,
		},
		{
			name: "zero created",
			summary: StatusSummary{
				Created:  0,
				Computed: 0,
			},
			want: 0.0,
		},
		{
			name: "partial progress",
			summary: StatusSummary{
				Created:  41,
				Computed: 35,
			},
			want: 35.0 / 41.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.Progress(); got != tt.want {
				t.Errorf("Progress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusResponse_JSON(t *testing.T) {
	// Test unmarshaling from API response
	jsonData := `{
		"report_id": 119022,
		"status_summary": {
			"Created": 41,
			"Running": 30,
			"Computed": 11,
			"Successful": 11
		},
		"calculation_start_time_utc": "2025-10-11T10:09:35.663000"
	}`

	var resp StatusResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.ReportID != 119022 {
		t.Errorf("ReportID = %d, want 119022", resp.ReportID)
	}

	if resp.StatusSummary.Created != 41 {
		t.Errorf("Created = %d, want 41", resp.StatusSummary.Created)
	}

	if resp.StatusSummary.Running != 30 {
		t.Errorf("Running = %d, want 30", resp.StatusSummary.Running)
	}

	if resp.StatusSummary.Computed != 11 {
		t.Errorf("Computed = %d, want 11", resp.StatusSummary.Computed)
	}

	if resp.CalculationStartTimeUTC != "2025-10-11T10:09:35.663000" {
		t.Errorf(
			"CalculationStartTimeUTC = %s, want 2025-10-11T10:09:35.663000",
			resp.CalculationStartTimeUTC,
		)
	}
}
