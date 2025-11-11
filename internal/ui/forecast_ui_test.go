package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/workflow"
)

func loadSampleResults(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("testdata", "forecast-results.fixture.json")
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
	if contains(view, "┌──────────────────────────────────────────────┐") {
		t.Fatalf("deprecated placeholder should not render: %s", view)
	}
	entries, err := parseForecastResults(data)
	if err != nil {
		t.Fatalf("parseForecastResults error: %v", err)
	}
	expectedHistory := len(filterValues(entries[0].Actuals))
	_, _, expectedForecast := collectSeries(entries[0])
	expectedHistoryLine := fmt.Sprintf("%-*s%d", resultFieldLabelWidth, "History points:", expectedHistory)
	if !contains(view, expectedHistoryLine) {
		t.Fatalf("expected history summary %q in view: %s", expectedHistoryLine, view)
	}
	expectedForecastLine := fmt.Sprintf("%-*s%d", resultFieldLabelWidth, "Forecast points:", expectedForecast)
	if !contains(view, expectedForecastLine) {
		t.Fatalf("expected forecast summary %q in view: %s", expectedForecastLine, view)
	}
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("expected colored plot output in view: %s", view)
	}
	if !contains(view, "┤") {
		t.Fatalf("expected asciigraph axis in view: %s", view)
	}
	selection := fmt.Sprintf("%3d/%3d", 1, len(m.results))
	if !contains(view, selection) {
		t.Fatalf("expected rendered forecast numbering in footer: %s", view)
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

func TestCollectSeriesKeepsHistoryWithForecast(t *testing.T) {
	entry := forecastResultEntry{
		Actuals: []seriesPoint{
			{Value: 10},
			{Value: 12},
		},
		Forecasts: []seriesPoint{
			{Value: 14},
		},
	}
	series, historyCount, forecastCount := collectSeries(entry)
	if historyCount == 0 {
		t.Fatalf("expected at least one history point, got %d", historyCount)
	}
	if forecastCount != 1 {
		t.Fatalf("expected one forecast point, got %d", forecastCount)
	}
	if len(series) != 2 {
		t.Fatalf("expected series length 2, got %d", len(series))
	}
	if series[0] != 12 {
		t.Fatalf("expected retained history value 12, got %.2f", series[0])
	}
}

func TestWrapSeriesNameBalancing(t *testing.T) {
	width := 20
	lines := wrapSeriesName("Alpha Beta Gamma Delta", width)
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %d", len(lines))
	}
	line1 := strings.TrimSpace(lines[0])
	line2 := strings.TrimSpace(lines[1])
	if line1 == "" || line2 == "" {
		t.Fatalf("expected non-empty balanced lines, got %q and %q", line1, line2)
	}
	if diff := intAbs(len([]rune(line1)) - len([]rune(line2))); diff > 1 {
		t.Fatalf("expected lengths within 1, got %d (%q vs %q)", diff, line1, line2)
	}
}

func TestWrapSeriesNameSingleLinePadding(t *testing.T) {
	width := 20
	lines := wrapSeriesName("Short", width)
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %d", len(lines))
	}
	if lines[0] != "" {
		t.Fatalf("expected leading blank line, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "Short" {
		t.Fatalf("expected centered word, got %q", lines[1])
	}
}

func TestFormatAxisLabelValue(t *testing.T) {
	got := formatAxisLabelValue("37000", 5)
	if got != "37.0k" {
		t.Fatalf("expected 37.0k, got %q", got)
	}
	got = formatAxisLabelValue("100", 5)
	if got != "  100" {
		t.Fatalf("expected padded 100, got %q", got)
	}
}

func TestNormalizeYAxisLabels(t *testing.T) {
	raw := "  37000 ┤\n   100 ┤\n"
	normalized := normalizeYAxisLabels(raw)
	for _, line := range strings.Split(normalized, "\n") {
		if line == "" {
			continue
		}
		idx := strings.IndexRune(line, '┤')
		if idx == -1 {
			continue
		}
		label := line[:idx]
		if len([]rune(label)) != 5 {
			t.Fatalf("expected label width 5, got %q", label)
		}
		if strings.TrimSpace(label) == "37000" {
			t.Fatalf("expected scaled label, got %q", label)
		}
	}
}

func intAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
