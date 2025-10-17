package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockRoundTripper is a mock implementation of http.RoundTripper for testing
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestAuthClient_Authenticate_Success(t *testing.T) {
	// Create a mock HTTP response
	responseBody := `{
		"access_token": "test-access-token",
		"expires_in": 3600,
		"refresh_expires_in": 1800,
		"refresh_token": "test-refresh-token",
		"token_type": "Bearer"
	}`

	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     make(http.Header),
		},
	}

	// Create HTTP client with mock transport
	httpClient := &http.Client{Transport: mockTransport}

	// Create auth client with injected HTTP client
	authClient := NewAuthClient("https://auth.example.com/token", httpClient)

	// Test authentication
	tokenResp, err := authClient.Authenticate("testuser", "testpass", "")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if tokenResp.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %s, want test-access-token", tokenResp.AccessToken)
	}

	if tokenResp.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %s, want test-refresh-token", tokenResp.RefreshToken)
	}

	if tokenResp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tokenResp.ExpiresIn)
	}
}

func TestAuthClient_Authenticate_InvalidCredentials(t *testing.T) {
	// Create a mock error response
	responseBody := `{
		"error": "invalid_grant",
		"error_description": "Invalid user credentials"
	}`

	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     make(http.Header),
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	authClient := NewAuthClient("https://auth.example.com/token", httpClient)

	// Test authentication with invalid credentials
	_, err := authClient.Authenticate("baduser", "badpass", "")
	if err == nil {
		t.Fatal("Authenticate() expected error, got nil")
	}

	// Verify error message contains status code
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error message should contain status code 401, got: %v", err)
	}
}

func TestAuthClient_RefreshToken_Success(t *testing.T) {
	responseBody := `{
		"access_token": "new-access-token",
		"expires_in": 3600,
		"refresh_expires_in": 1800,
		"refresh_token": "new-refresh-token",
		"token_type": "Bearer"
	}`

	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     make(http.Header),
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	authClient := NewAuthClient("https://auth.example.com/token", httpClient)

	// Test token refresh
	tokenResp, err := authClient.RefreshToken("old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if tokenResp.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %s, want new-access-token", tokenResp.AccessToken)
	}

	if tokenResp.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %s, want new-refresh-token", tokenResp.RefreshToken)
	}
}

func TestAuthClient_DefaultHTTPClient(t *testing.T) {
	// Test that nil httpClient creates a default one
	authClient := NewAuthClient("https://auth.example.com/token", nil)

	if authClient.httpClient == nil {
		t.Error("httpClient should not be nil when passing nil to NewAuthClient")
	}

	// Verify default timeout is set
	if authClient.httpClient.Timeout == 0 {
		t.Error("default httpClient should have a timeout set")
	}
}
