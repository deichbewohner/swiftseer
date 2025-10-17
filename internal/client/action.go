package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

const statusFinished = "finished"

func isInProgress(s string) bool {
	return s == "created" || s == "pending" || s == "running"
}

func (c *Client) ExecuteAction(
	ctx context.Context,
	coreID string,
	payload *ActionPayload,
	pollInterval time.Duration,
	timeout time.Duration,
) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	actionID, err := c.startAction(ctx, coreID, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to start action: %w", err)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("action timeout after %v", timeout)
			}
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.getActionStatus(ctx, coreID, actionID)
			if err != nil {
				// transient errors during polling are retried
				continue
			}
			if status == statusFinished {
				return c.getActionResult(ctx, coreID, actionID)
			}
			if !isInProgress(status) {
				return nil, fmt.Errorf("action failed with status: %s", status)
			}
		}
	}
}

func (c *Client) startAction(ctx context.Context, coreID string, payload *ActionPayload) (string, error) {
	url := c.buildURL(fmt.Sprintf("/cores/%s/actions", coreID))
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal action payload: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create action request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	var actionResp ActionResponse
	if err := c.doRequest(ctx, req, &actionResp); err != nil {
		return "", err
	}
	if actionResp.ActionID == "" {
		return "", fmt.Errorf("no action ID in response")
	}
	return actionResp.ActionID, nil
}

func (c *Client) getActionStatus(ctx context.Context, coreID, actionID string) (string, error) {
	statusURL := c.buildURL(fmt.Sprintf("/cores/%s/actions/%s/status", coreID, actionID))
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create status request: %w", err)
	}
	var statusResp ActionStatusResponse
	if err := c.doRequest(ctx, req, &statusResp); err != nil {
		return "", err
	}
	s, ok := statusResp.Payload["status"].(string)
	if !ok {
		return "", fmt.Errorf("invalid status response")
	}
	return s, nil
}

func (c *Client) getActionResult(ctx context.Context, coreID, actionID string) (map[string]interface{}, error) {
	resultURL := c.buildURL(fmt.Sprintf("/cores/%s/actions/%s", coreID, actionID))
	req, err := http.NewRequest("GET", resultURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create result request: %w", err)
	}
	var resultResp ActionResponse
	if err := c.doRequest(ctx, req, &resultResp); err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}
	return resultResp.Payload, nil
}
