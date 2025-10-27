package client

import (
	"testing"

	"github.com/deichbewohner/swiftseer/internal/config"
)

func TestBuildURL_WithGroup(t *testing.T) {
	cfg := &config.Config{}
	c := NewClient("https://api.example.com/v1/", "team-a", cfg, nil, nil, nil)

	got := c.buildURL("/userinputs")
	want := "https://api.example.com/v1/groups/team-a/userinputs"
	if got != want {
		t.Fatalf("buildURL() = %q, want %q", got, want)
	}
}
