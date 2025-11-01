package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/ui"
)

func ResultsCmd(args []string) error {
	fs := flag.NewFlagSet("results", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Show forecast results in TUI

Usage:
  swiftseer results [flags]

Flags:
  --file, -f   Path to forecast results JSON (default: forecast-results.json)
`)
	}
	var filePath string
	fs.StringVar(&filePath, "file", "forecast-results.json", "Path to forecast results JSON")
	fs.StringVar(&filePath, "f", "forecast-results.json", "Path to forecast results JSON (short form)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read results file: %w", err)
	}
	model, err := ui.NewForecastResultsViewer(context.Background(), data)
	if err != nil {
		return fmt.Errorf("failed to load forecast results: %w", err)
	}
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return fmt.Errorf("results UI error: %w", err)
	}
	return nil
}
