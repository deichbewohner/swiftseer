package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/workflow"
)

func loadSampleResults(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "forecast-results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample results: %v", err)
	}
	return data
}

func TestParseForecastResults(t *testing.T) {
	data := loadSampleResults(t)
	entries, err := parseForecastResults(data)
	if err != nil {
		t.Fatalf("parseForecastResults error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected entries, got %d", len(entries))
	}
	first := entries[0]
	if first.Model.Name == "" {
		t.Fatalf("expected model name")
	}
	if len(first.Forecasts) == 0 {
		t.Fatalf("expected forecast horizon")
	}
}

func TestForecastModelRendersResultView(t *testing.T) {
	data := loadSampleResults(t)
	m := NewForecastModel(context.Background(), nil, ForecastParams{})
	res := &workflow.RunResult{Results: data}
	m.handleWorkflowComplete(res)
	if !m.showingResults() {
		t.Fatalf("expected model to be in results mode")
	}
	view := m.renderResultView()
	if !contains(view, "┌──────────────────────────────────────────────┐") {
		t.Fatalf("expected placeholder plot in view: %s", view)
	}
	box := m.View()
	if !contains(box, "Forecast 1/") {
		t.Fatalf("expected rendered forecast numbering in box: %s", box)
	}
}

func TestForecastModelNavigation(t *testing.T) {
	data := loadSampleResults(t)
	m := NewForecastModel(context.Background(), nil, ForecastParams{})
	res := &workflow.RunResult{Results: data}
	m.handleWorkflowComplete(res)
	if len(m.results) < 2 {
		t.Skip("sample data does not contain multiple forecasts")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.index != 1 {
		t.Fatalf("expected index to be 1, got %d", m.index)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.index != 0 {
		t.Fatalf("expected index to return to 0, got %d", m.index)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.index != len(m.results)-1 {
		t.Fatalf("expected wrap to last index, got %d", m.index)
	}
}

func TestNewForecastResultsViewer(t *testing.T) {
	data := loadSampleResults(t)
	m, err := NewForecastResultsViewer(context.Background(), data)
	if err != nil {
		t.Fatalf("NewForecastResultsViewer error: %v", err)
	}
	if !m.showingResults() {
		t.Fatalf("expected results viewer to be in results mode")
	}
	if len(m.results) == 0 {
		t.Fatalf("expected parsed results")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
