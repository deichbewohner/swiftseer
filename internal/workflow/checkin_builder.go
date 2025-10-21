package workflow

import (
	"github.com/deichbewohner/swiftseer/internal/client"
	"github.com/deichbewohner/swiftseer/internal/models"
)

type CheckInRequestBuilder interface {
	Build(fileUUID, csvPath string, overrides *models.CheckInOverrides) (*models.CheckInRequest, error)
}

type defaultCheckInRequestBuilder struct{}

func (defaultCheckInRequestBuilder) Build(
	fileUUID, csvPath string, overrides *models.CheckInOverrides,
) (*models.CheckInRequest, error) {
	return client.BuildCheckInRequestWithOverrides(fileUUID, csvPath, overrides)
}
