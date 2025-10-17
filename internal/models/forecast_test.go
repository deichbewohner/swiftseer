package models

import (
	"encoding/json"
	"testing"
)

func TestForecastRequest_JSON(t *testing.T) {
	lowerBound := 0.0
	req := ForecastRequest{
		Version: "68ea2cd6cc05e6ab4b85ebd1",
		Config: ForecastConfig{
			Title: "Test Forecast",
			Forecasting: ForecastingConfig{
				FcHorizon:              12,
				LowerBound:             &lowerBound,
				ConfidenceLevel:        0.75,
				RoundForecastToInteger: true,
				UseEnsemble:            false,
			},
			Preprocessing: PreprocessingConfig{
				UseSeasonDetection: true,
				DetectOutliers:     true,
				ReplaceOutliers:    false,
			},
			MethodSelection: MethodSelectionConfig{
				NumberIterations:   24,
				DefaultErrorMetric: "mae",
				Refit:              false,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded ForecastRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key fields
	if decoded.Version != req.Version {
		t.Errorf("Version = %s, want %s", decoded.Version, req.Version)
	}

	if decoded.Config.Forecasting.FcHorizon != 12 {
		t.Errorf("FcHorizon = %d, want 12", decoded.Config.Forecasting.FcHorizon)
	}

	if decoded.Config.Forecasting.ConfidenceLevel != 0.75 {
		t.Errorf("ConfidenceLevel = %f, want 0.75", decoded.Config.Forecasting.ConfidenceLevel)
	}

	if decoded.Config.Forecasting.LowerBound == nil {
		t.Error("LowerBound is nil, want non-nil")
	} else if *decoded.Config.Forecasting.LowerBound != 0.0 {
		t.Errorf("LowerBound = %f, want 0.0", *decoded.Config.Forecasting.LowerBound)
	}
}

func TestForecastingConfig_OmitEmpty(t *testing.T) {
	tests := []struct {
		name           string
		config         ForecastingConfig
		wantLowerBound bool
		wantUpperBound bool
	}{
		{
			name: "no bounds",
			config: ForecastingConfig{
				FcHorizon:       12,
				ConfidenceLevel: 0.75,
			},
			wantLowerBound: false,
			wantUpperBound: false,
		},
		{
			name: "with lower bound",
			config: ForecastingConfig{
				FcHorizon:       12,
				LowerBound:      ptrFloat64(0),
				ConfidenceLevel: 0.75,
			},
			wantLowerBound: true,
			wantUpperBound: false,
		},
		{
			name: "with upper bound",
			config: ForecastingConfig{
				FcHorizon:       12,
				UpperBound:      ptrFloat64(1000),
				ConfidenceLevel: 0.75,
			},
			wantLowerBound: false,
			wantUpperBound: true,
		},
		{
			name: "with both bounds",
			config: ForecastingConfig{
				FcHorizon:       12,
				LowerBound:      ptrFloat64(0),
				UpperBound:      ptrFloat64(1000),
				ConfidenceLevel: 0.75,
			},
			wantLowerBound: true,
			wantUpperBound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			hasLowerBound := containsKey(string(data), "lower_bound")
			if hasLowerBound != tt.wantLowerBound {
				t.Errorf(
					"JSON contains 'lower_bound' = %v, want %v",
					hasLowerBound,
					tt.wantLowerBound,
				)
			}

			hasUpperBound := containsKey(string(data), "upper_bound")
			if hasUpperBound != tt.wantUpperBound {
				t.Errorf(
					"JSON contains 'upper_bound' = %v, want %v",
					hasUpperBound,
					tt.wantUpperBound,
				)
			}
		})
	}
}

func TestForecastResponse_JSON(t *testing.T) {
	jsonData := `{
		"report_id": 119022,
		"settings_id": 119024
	}`

	var resp ForecastResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.ReportID != 119022 {
		t.Errorf("ReportID = %d, want 119022", resp.ReportID)
	}

	if resp.SettingsID != 119024 {
		t.Errorf("SettingsID = %d, want 119024", resp.SettingsID)
	}
}

func TestMethodSelectionConfig_DefaultValues(t *testing.T) {
	// Test that omitempty works correctly for optional fields
	config := MethodSelectionConfig{
		DefaultErrorMetric: "mae",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Should not include number_iterations or refit if they're zero values
	if containsKey(string(data), "number_iterations") {
		t.Error("JSON should not contain 'number_iterations' when zero")
	}

	if containsKey(string(data), "refit") {
		t.Error("JSON should not contain 'refit' when false")
	}

	// But should include default_error_metric
	if !containsKey(string(data), "default_error_metric") {
		t.Error("JSON should contain 'default_error_metric'")
	}
}
