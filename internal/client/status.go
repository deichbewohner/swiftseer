package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func (c *Client) GetStatus(ctx context.Context, reportID int) (*models.StatusResponse, error) {
	url := c.buildURL(fmt.Sprintf("/reports/%d/status", reportID))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	var statusResp models.StatusResponse
	if err := c.doRequest(ctx, req, &statusResp); err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}

	return &statusResp, nil
}
