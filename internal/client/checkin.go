package client

import (
	"context"
	"fmt"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func (c *Client) CheckIn(
	ctx context.Context,
	userInputID, fileID string,
	req *models.CheckInRequest,
) (string, error) {
    checkInConfig := buildCheckInConfig(req)

    checkInConfig["stage"] = "createDataset"
    checkInConfig["fileUuid"] = fileID

    actionPayload := &ActionPayload{
        UserInputID: userInputID,
        Payload:     checkInConfig,
    }

    result, err := c.ExecuteAction(
		ctx,
		"checkin-preprocessing",
		actionPayload,
		2*time.Second,
		10*time.Minute,
	)
	if err != nil {
		return "", fmt.Errorf("check-in action failed: %w", err)
	}

    resultMap, ok := result["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid result structure")
	}

	versionID, ok := resultMap["tsVersion"].(string)
	if !ok || versionID == "" {
		return "", fmt.Errorf("no tsVersion in result")
	}

	return versionID, nil
}

func buildCheckInConfig(req *models.CheckInRequest) map[string]interface{} {
    valueColumns := make([]map[string]interface{}, len(req.DataDefinition.ValueColumns))
	for i, col := range req.DataDefinition.ValueColumns {
		vc := map[string]interface{}{
			"name":     col.Name,
			"dtypeStr": col.DtypeStr,
		}
		if col.Min != nil {
			vc["min"] = *col.Min
		}
		if col.Max != nil {
			vc["max"] = *col.Max
		}
		valueColumns[i] = vc
	}

	groupColumns := make([]map[string]interface{}, len(req.DataDefinition.GroupColumns))
	for i, col := range req.DataDefinition.GroupColumns {
		groupColumns[i] = map[string]interface{}{
			"name":     col.Name,
			"dtypeStr": col.DtypeStr,
		}
	}

	columnDefinition := map[string]interface{}{
		"dateColumns": []map[string]interface{}{
			{
				"name":   req.DataDefinition.DateColumns.Name,
				"format": req.DataDefinition.DateColumns.Format,
			},
		},
		"valueColumns": valueColumns,
		"groupColumns": groupColumns,
	}

    rawDataReviewResults := make(map[string]interface{})
	rawDataReviewResults[req.DataDefinition.DateColumns.Name] = map[string]interface{}{}
	for _, col := range req.DataDefinition.ValueColumns {
		rawDataReviewResults[col.Name] = map[string]interface{}{}
	}
	for _, col := range req.DataDefinition.GroupColumns {
		rawDataReviewResults[col.Name] = map[string]interface{}{}
	}

    timeSeriesDatasetParameter := map[string]interface{}{
		"aggregation": map[string]interface{}{
			"operator": "sum",
			"option":   req.ConfigTsCreation.MissingValueHandler,
		},
		"date": map[string]interface{}{
			"timeGranularity": req.ConfigTsCreation.TimeGranularity,
			"startDate":       nil,
			"endDate":         nil,
		},
		"grouping": map[string]interface{}{
			"dataLevel":     req.ConfigTsCreation.GroupingLevel,
			"saveHierarchy": false,
			"filter":        []interface{}{},
		},
		"values":             []interface{}{},
		"valueColumnsToSave": req.ConfigTsCreation.ValueColumnsToSave,
	}

    return map[string]interface{}{
		"performedTasks": map[string]interface{}{
			"removedCols": []interface{}{},
			"removedRows": []interface{}{},
		},
		"performedTasksLog": []interface{}{},
		"meta": map[string]interface{}{
			"delimiter": req.FileSpecification.Delimiter,
			"decimal":   req.FileSpecification.Decimal,
			"thousands": nil,
			"naValues":  []string{"nan"},
		},
		"columnDefinition":           columnDefinition,
		"rawDataReviewResults":       rawDataReviewResults,
		"timeSeriesDatasetParameter": timeSeriesDatasetParameter,
		"versionDescription":         req.ConfigTsCreation.Description,
	}
}
