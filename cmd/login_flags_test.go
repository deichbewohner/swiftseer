package cmd

import "testing"

func TestParseLoginFlags(t *testing.T) {
	u, g, e, err := parseLoginFlags([]string{"-u", "alice", "-g", "team-a", "-e", "production"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if u != "alice" || g != "team-a" || e != "production" {
		t.Errorf("unexpected values: u=%s g=%s e=%s", u, g, e)
	}
}
