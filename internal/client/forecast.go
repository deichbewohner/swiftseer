package client

import (
	"context"
	"fmt"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func (c *Client) StartForecast(
	ctx context.Context,
	versionID string,
	req *models.ForecastRequest,
) (int, error) {
	forecastPayload := buildForecastPayload(versionID, req)

	actionPayload := &ActionPayload{
		Payload: forecastPayload,
	}

	result, err := c.ExecuteAction(
		ctx,
		"forecast-batch",
		actionPayload,
		2*time.Second,
		5*time.Minute,
	)
	if err != nil {
		return 0, fmt.Errorf("forecast action failed: %w", err)
	}

	reportIDFloat, ok := result["report_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("no report_id in result")
	}

	return int(reportIDFloat), nil
}

func buildForecastPayload(versionID string, req *models.ForecastRequest) map[string]interface{} {
	payload := map[string]interface{}{
		"actuals_version": versionID,
		"report_note":     req.Config.Title,
		"forecasting": map[string]interface{}{
			"n_ahead":                   req.Config.Forecasting.FcHorizon,
			"confidence_level":          req.Config.Forecasting.ConfidenceLevel,
			"round_forecast_to_integer": req.Config.Forecasting.RoundForecastToInteger,
			"use_ensemble":              req.Config.Forecasting.UseEnsemble,
		},
		"preprocessing":  make(map[string]interface{}),
		"backtesting":    make(map[string]interface{}),
		"covs_versions":  []interface{}{},
		"actuals_filter": make(map[string]interface{}),
	}

	forecastingMap := payload["forecasting"].(map[string]interface{})
	if req.Config.Forecasting.LowerBound != nil {
		forecastingMap["lower_bound"] = *req.Config.Forecasting.LowerBound
	}
	if req.Config.Forecasting.UpperBound != nil {
		forecastingMap["upper_bound"] = *req.Config.Forecasting.UpperBound
	}

	if req.Config.Preprocessing.UseSeasonDetection {
		payload["preprocessing"].(map[string]interface{})["use_season_detection"] = true
	}
	if req.Config.Preprocessing.DetectOutliers {
		payload["preprocessing"].(map[string]interface{})["detect_outliers"] = true
	}
	if req.Config.Preprocessing.ReplaceOutliers {
		payload["preprocessing"].(map[string]interface{})["replace_outliers"] = true
	}
	if req.Config.Preprocessing.DetectChangepoints {
		payload["preprocessing"].(map[string]interface{})["detect_changepoints"] = true
	}

	if req.Config.MethodSelection.NumberIterations > 0 {
		payload["backtesting"].(map[string]interface{})["number_iterations"] = req.Config.MethodSelection.NumberIterations
	}
	if req.Config.MethodSelection.DefaultErrorMetric != "" {
		payload["backtesting"].(map[string]interface{})["default_error_metric"] = req.Config.MethodSelection.DefaultErrorMetric
	}
	if req.Config.MethodSelection.Refit {
		payload["backtesting"].(map[string]interface{})["refit"] = true
	}

	return payload
}
