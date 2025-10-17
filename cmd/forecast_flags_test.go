package cmd

import "testing"

func TestParseForecastFlags_NewForecast(t *testing.T) {
	args := []string{
		"--horizon",
		"24",
		"--confidence",
		"0.8",
		"-o",
		"out.json",
		"--title",
		"T1",
		"-v",
		"data.csv",
	}
	opts, err := parseForecastFlags(args)
	if err != nil {
		t.Fatalf("parseForecastFlags() err = %v", err)
	}
	if opts.Horizon != 24 || opts.Confidence != 0.8 || opts.Output != "out.json" ||
		opts.Title != "T1" ||
		!opts.Verbose ||
		opts.CSVPath != "data.csv" {
		t.Errorf("parsed opts mismatch: %+v", *opts)
	}
}

func TestParseForecastFlags_ReportID(t *testing.T) {
	args := []string{"--report-id", "42", "--clean"}
	opts, err := parseForecastFlags(args)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if opts.ReportID != 42 || !opts.Clean {
		t.Errorf("unexpected opts: %+v", *opts)
	}
}

func TestParseForecastFlags_ArgErrors(t *testing.T) {
	// Missing csv when report-id is not set
	if _, err := parseForecastFlags([]string{}); err == nil {
		t.Error("expected error for missing csv arg")
	}
	// CSV provided with report-id
	if _, err := parseForecastFlags([]string{"--report-id", "1", "file.csv"}); err == nil {
		t.Error("expected error when csv is provided with --report-id")
	}
	// Invalid horizon
	if _, err := parseForecastFlags([]string{"--horizon", "0", "file.csv"}); err == nil {
		t.Error("expected error for non-positive horizon")
	}
	// Invalid confidence
	if _, err := parseForecastFlags([]string{"--confidence", "1.5", "file.csv"}); err == nil {
		t.Error("expected error for invalid confidence")
	}
}
