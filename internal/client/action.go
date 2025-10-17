package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ActionPayload struct {
	UserInputID string                 `json:"userInputId,omitempty"`
	FileIDs     []string               `json:"fileIds,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
}

type ActionResponse struct {
	ActionID string                 `json:"actionId"`
	Payload  map[string]interface{} `json:"payload"`
}

type ActionStatusResponse struct {
	ActionID string                 `json:"actionId"`
	Payload  map[string]interface{} `json:"payload"`
}

func (c *Client) ExecuteAction(
	ctx context.Context,
	coreID string,
	payload *ActionPayload,
	pollInterval time.Duration,
	timeout time.Duration,
) (map[string]interface{}, error) {
	url := c.buildURL(fmt.Sprintf("/cores/%s/actions", coreID))
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create action request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var actionResp ActionResponse
	if err := c.doRequest(ctx, req, &actionResp); err != nil {
		return nil, fmt.Errorf("failed to start action: %w", err)
	}

	actionID := actionResp.ActionID
	if actionID == "" {
		return nil, fmt.Errorf("no action ID in response")
	}

	statusURL := c.buildURL(fmt.Sprintf("/cores/%s/actions/%s/status", coreID, actionID))
	resultURL := c.buildURL(fmt.Sprintf("/cores/%s/actions/%s", coreID, actionID))

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("action timeout after %v", timeout)
			}

			// Check status
			statusReq, err := http.NewRequest("GET", statusURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create status request: %w", err)
			}

			var statusResp ActionStatusResponse
			if err := c.doRequest(ctx, statusReq, &statusResp); err != nil {
				// Retry on transient errors
				continue
			}

			statusPayload := statusResp.Payload
			status, ok := statusPayload["status"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid status response")
			}

			// Check if finished
			if status == "finished" {
				// Get final result
				resultReq, err := http.NewRequest("GET", resultURL, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to create result request: %w", err)
				}

				var resultResp ActionResponse
				if err := c.doRequest(ctx, resultReq, &resultResp); err != nil {
					return nil, fmt.Errorf("failed to get result: %w", err)
				}

				return resultResp.Payload, nil
			}

			// Check if failed
			if status != "created" && status != "pending" && status != "running" {
				return nil, fmt.Errorf("action failed with status: %s", status)
			}

			// Continue polling
		}
	}
}
