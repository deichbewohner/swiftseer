package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/deichbewohner/swiftseer/internal/config"
)

type Client struct {
	baseURL       string
	group         string
	httpClient    *http.Client
	authClient    *AuthClient
	config        *config.Config
	configManager *config.Manager
	mu            sync.RWMutex // Protects token refresh operations
}

func NewClient(
	baseURL, group string,
	cfg *config.Config,
	cfgManager *config.Manager,
	authClient *AuthClient,
	httpClient *http.Client,
) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Client{
		baseURL:       baseURL,
		group:         group,
		httpClient:    httpClient,
		authClient:    authClient,
		config:        cfg,
		configManager: cfgManager,
	}
}

func (c *Client) EnsureValidToken(ctx context.Context) error {
	c.mu.RLock()
	needsRefresh := c.config.NeedsTokenRefresh()
	c.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.config.NeedsTokenRefresh() {
        return nil
    }

    if !c.config.IsRefreshTokenValid() {
        return fmt.Errorf("refresh token expired, please login again")
    }

    tokenResp, err := c.authClient.RefreshToken(c.config.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

    c.config.UpdateToken(
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		tokenResp.ExpiresIn,
		tokenResp.RefreshExpiresIn,
	)

    if err := c.configManager.Save(c.config); err != nil {
        return fmt.Errorf("failed to save refreshed token: %w", err)
    }

	return nil
}

func (c *Client) doRequest(ctx context.Context, req *http.Request, result interface{}) error {
    if err := c.EnsureValidToken(ctx); err != nil {
        return err
    }
    c.mu.RLock()
    token := c.config.AccessToken
    c.mu.RUnlock()

    req.Header.Set("Authorization", "Bearer "+token)
    req = req.WithContext(ctx)
    resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

    body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Method:     req.Method,
			URL:        req.URL.String(),
		}
	}

    if result != nil {
        if err := json.Unmarshal(body, result); err != nil {
            return fmt.Errorf("failed to parse response: %w", err)
        }
    }

	return nil
}

func (c *Client) doRequestRaw(ctx context.Context, req *http.Request) ([]byte, error) {
    if err := c.EnsureValidToken(ctx); err != nil {
        return nil, err
    }
    c.mu.RLock()
    token := c.config.AccessToken
    c.mu.RUnlock()

    req.Header.Set("Authorization", "Bearer "+token)
    req = req.WithContext(ctx)
    resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

    body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Method:     req.Method,
			URL:        req.URL.String(),
		}
	}

	return body, nil
}

func (c *Client) buildURL(path string) string {
	return c.baseURL + "groups/" + c.group + path
}

type APIError struct {
	StatusCode int
	Message    string
	Method     string
	URL        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d %s %s): %s", e.StatusCode, e.Method, e.URL, e.Message)
}
