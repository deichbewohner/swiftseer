package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func FetchGroups(ctx context.Context, baseURL, accessToken string, httpClient *http.Client) (*models.GroupsResponse, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required to fetch groups")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"groups", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch groups: HTTP %d", resp.StatusCode)
	}

	var groups models.GroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups response: %w", err)
	}

	return &groups, nil
}
