package workflow

import (
	"context"

	"github.com/deichbewohner/swiftseer/internal/client"
	"github.com/deichbewohner/swiftseer/internal/models"
)

type ClientAdapter struct {
    client *client.Client
}

func NewClientAdapter(c *client.Client) *ClientAdapter {
    return &ClientAdapter{client: c}
}

func (a *ClientAdapter) UploadCSV(ctx context.Context, path string) (*UploadResult, error) {
	result, err := a.client.UploadCSV(ctx, path)
	if err != nil {
		return nil, err
	}
	return &UploadResult{
		UserInputID: result.UserInputID,
		FileID:      result.FileID,
	}, nil
}

func (a *ClientAdapter) CheckIn(
	ctx context.Context,
	userInputID, fileID string,
	req *models.CheckInRequest,
) (string, error) {
	return a.client.CheckIn(ctx, userInputID, fileID, req)
}

func (a *ClientAdapter) StartForecast(
	ctx context.Context,
	versionID string,
	req *models.ForecastRequest,
) (int, error) {
	return a.client.StartForecast(ctx, versionID, req)
}

func (a *ClientAdapter) GetStatus(
	ctx context.Context,
	reportID int,
) (*models.StatusResponse, error) {
	return a.client.GetStatus(ctx, reportID)
}

func (a *ClientAdapter) GetResults(
	ctx context.Context,
	reportID int,
	opts *models.ResultsOptions,
) ([]byte, error) {
	return a.client.GetResults(ctx, reportID, opts)
}

func (a *ClientAdapter) DeleteUpload(ctx context.Context, userInputID string) error {
	return a.client.DeleteUpload(ctx, userInputID)
}

func (a *ClientAdapter) DeleteReport(ctx context.Context, reportID int) error {
	return a.client.DeleteReport(ctx, reportID)
}
