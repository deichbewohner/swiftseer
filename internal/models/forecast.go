package models

type ForecastRequest struct {
	Version string         `json:"version"`
	Config  ForecastConfig `json:"config"`
}

type ForecastConfig struct {
	Title           string                `json:"title,omitempty"`
	Forecasting     ForecastingConfig     `json:"forecasting"`
	Preprocessing   PreprocessingConfig   `json:"preprocessing,omitempty"`
	MethodSelection MethodSelectionConfig `json:"method_selection,omitempty"`
}

type ForecastingConfig struct {
	FcHorizon              int      `json:"fc_horizon"`
	LowerBound             *float64 `json:"lower_bound,omitempty"`
	UpperBound             *float64 `json:"upper_bound,omitempty"`
	ConfidenceLevel        float64  `json:"confidence_level"`
	RoundForecastToInteger bool     `json:"round_forecast_to_integer,omitempty"`
	UseEnsemble            bool     `json:"use_ensemble,omitempty"`
}

type PreprocessingConfig struct {
	UseSeasonDetection bool `json:"use_season_detection,omitempty"`
	DetectOutliers     bool `json:"detect_outliers,omitempty"`
	ReplaceOutliers    bool `json:"replace_outliers,omitempty"`
	DetectChangepoints bool `json:"detect_changepoints,omitempty"`
}

type MethodSelectionConfig struct {
	NumberIterations   int    `json:"number_iterations,omitempty"`
	DefaultErrorMetric string `json:"default_error_metric,omitempty"` // "mae", "mape", "mse", etc.
	Refit              bool   `json:"refit,omitempty"`
}

type ForecastResponse struct {
	ReportID   int `json:"report_id"`
	SettingsID int `json:"settings_id"`
}
