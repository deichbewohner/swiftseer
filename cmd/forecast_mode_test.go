package cmd

import "testing"

func TestShouldUseTUI(t *testing.T) {
	cases := []struct {
		name     string
		reportID int
		noUI     bool
		json     bool
		inTTY    bool
		outTTY   bool
		want     bool
	}{
		{"tui_ok", 0, false, false, true, true, true},
		{"no_ui_flag", 0, true, false, true, true, false},
		{"json_events", 0, false, true, true, true, false},
		{"stdin_not_tty", 0, false, false, false, true, false},
		{"stdout_not_tty", 0, false, false, true, false, false},
		{"resume_report_id", 7, false, false, true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldUseTUI(tc.reportID, tc.noUI, tc.json, tc.inTTY, tc.outTTY)
			if got != tc.want {
				t.Fatalf("shouldUseTUI() = %v, want %v", got, tc.want)
			}
		})
	}
}
