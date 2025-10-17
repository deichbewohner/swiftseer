package client

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) DeleteUpload(ctx context.Context, userInputID string) error {
	url := c.buildURL(fmt.Sprintf("/userinputs/%s", userInputID))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	if err := c.doRequest(ctx, req, nil); err != nil {
		return fmt.Errorf("delete upload request failed: %w", err)
	}

	return nil
}

func (c *Client) DeleteReport(ctx context.Context, reportID int) error {
	url := c.buildURL(fmt.Sprintf("/reports/%d", reportID))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	if err := c.doRequest(ctx, req, nil); err != nil {
		return fmt.Errorf("delete report request failed: %w", err)
	}

	return nil
}
