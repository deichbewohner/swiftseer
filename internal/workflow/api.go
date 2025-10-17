package workflow

import (
	"context"

	"github.com/deichbewohner/swiftseer/internal/models"
)

type ForecastAPI interface {
	UploadCSV(ctx context.Context, path string) (*UploadResult, error)
	CheckIn(
		ctx context.Context,
		userInputID, fileID string,
		req *models.CheckInRequest,
	) (string, error)
	StartForecast(ctx context.Context, versionID string, req *models.ForecastRequest) (int, error)
	GetStatus(ctx context.Context, reportID int) (*models.StatusResponse, error)
	GetResults(ctx context.Context, reportID int, opts *models.ResultsOptions) ([]byte, error)
	DeleteUpload(ctx context.Context, userInputID string) error
	DeleteReport(ctx context.Context, reportID int) error
}

type UploadResult struct {
	UserInputID string
	FileID      string
}
