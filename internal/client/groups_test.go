package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func TestFetchGroups_Success(t *testing.T) {
	want := models.GroupsResponse{
		UserID: "123",
		Groups: []models.Group{{ID: "customer1", Label: "Customer 1", FileRetentionDays: 7}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/groups" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token" {
			t.Fatalf("unexpected authorization header: %s", auth)
		}
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()

	got, err := FetchGroups(context.Background(), server.URL+"/", "token", server.Client())
	if err != nil {
		t.Fatalf("FetchGroups() error = %v", err)
	}
	if got.UserID != want.UserID || len(got.Groups) != len(want.Groups) || got.Groups[0].ID != want.Groups[0].ID {
		t.Fatalf("FetchGroups() = %+v, want %+v", got, want)
	}
}

func TestFetchGroups_ErrorMissingToken(t *testing.T) {
	_, err := FetchGroups(context.Background(), "https://example.com/", "", http.DefaultClient)
	if err == nil {
		t.Fatal("FetchGroups() error = nil, want error")
	}
}

func TestFetchGroups_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := FetchGroups(context.Background(), server.URL+"/", "token", server.Client())
	if err == nil {
		t.Fatal("FetchGroups() error = nil, want error")
	}
}
