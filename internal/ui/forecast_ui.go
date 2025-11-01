package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/deichbewohner/swiftseer/internal/models"
	"github.com/deichbewohner/swiftseer/internal/version"
	"github.com/deichbewohner/swiftseer/internal/workflow"
	"github.com/guptarohit/asciigraph"
)

var forecastBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240")).
	Padding(0, 1).
	Width(47)

const (
	resultContentWidth    = 46
	resultPlotWidth       = 39
	resultPlotHeight      = 10
	resultFieldLabelWidth = 18
)

type seriesPoint struct {
	Timestamp string
	Value     float64
}

type forecastModelInfo struct {
	Name         string
	Status       string
	Plausibility string
	RankPosition int
	RankScore    float64
}

type forecastResultEntry struct {
	Name        string
	Granularity string
	Actuals     []seriesPoint
	Forecasts   []seriesPoint
	Model       forecastModelInfo
}

func wrapSeriesName(name string, width int) []string {
	if width <= 0 {
		return []string{""}
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return []string{""}
	}
	if utf8.RuneCountInString(trimmed) <= width {
		return []string{"", centerLine(trimmed, width)}
	}

	first, second := balancedLines(trimmed, width)
	var lines []string
	if first != "" {
		lines = append(lines, centerLine(first, width))
	}
	if second != "" {
		lines = append(lines, centerLine(second, width))
	}
	if len(lines) == 0 {
		lines = append(lines, centerLine(truncate(trimmed, width), width))
	}
	if len(lines) == 1 {
		lines = append([]string{""}, lines...)
	}
	return lines
}

func balancedLines(text string, width int) (string, string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "", ""
	}
	if len(words) == 1 {
		word := words[0]
		if utf8.RuneCountInString(word) > width {
			return truncate(word, width), ""
		}
		return word, ""
	}

	bestSplit := -1
	bestDiff := width + 1
	for i := 1; i < len(words); i++ {
		first := strings.Join(words[:i], " ")
		second := strings.Join(words[i:], " ")
		w1 := utf8.RuneCountInString(first)
		w2 := utf8.RuneCountInString(second)
		if w1 > width || w2 > width {
			continue
		}
		diff := w1 - w2
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			bestSplit = i
			if diff == 0 {
				break
			}
		}
	}

	if bestSplit != -1 {
		return strings.Join(words[:bestSplit], " "), strings.Join(words[bestSplit:], " ")
	}

	chosen := []string{words[0]}
	split := 1
	current := words[0]
	for split < len(words) {
		candidate := current + " " + words[split]
		if utf8.RuneCountInString(candidate) > width {
			break
		}
		current = candidate
		chosen = append(chosen, words[split])
		split++
	}

	first := strings.Join(chosen, " ")
	if utf8.RuneCountInString(first) > width {
		first = truncate(first, width)
	}
	if split >= len(words) {
		return first, ""
	}

	second := strings.Join(words[split:], " ")
	if utf8.RuneCountInString(second) > width {
		second = truncate(second, width)
	}
	return first, second
}

func centerLine(text string, width int) string {
	length := utf8.RuneCountInString(text)
	if length >= width {
		return text
	}
	padding := (width - length) / 2
	if padding <= 0 {
		return text
	}
	return strings.Repeat(" ", padding) + text
}

type StageStatus struct {
	Name      string
	Complete  bool
	Duration  time.Duration
	StartTime time.Time
	Error     error
}

type ForecastModel struct {
	ctx     context.Context
	api     workflow.ForecastAPI
	params  workflow.RunParams
	output  string
	verbose bool

	tokenExpiresAt time.Time

	cleanPolicy workflow.CleanPolicy

	currentStage workflow.Stage
	stages       map[workflow.Stage]*StageStatus

	reportID  int
	completed int
	total     int
	running   int

	spinner  spinner.Model
	progress progress.Model

	done     bool
	quitting bool
	err      error
	result   *workflow.RunResult
	results  []forecastResultEntry
	index    int

	job *workflow.Job
}

type ForecastParams struct {
	CSVPath        string
	Horizon        int
	Confidence     float64
	Title          string
	Output         string
	Verbose        bool
	TokenExpiresAt time.Time
	Clean          bool
	Overrides      *models.CheckInOverrides
}

func NewForecastModel(
	ctx context.Context,
	api workflow.ForecastAPI,
	params ForecastParams,
) *ForecastModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 46

	stages := map[workflow.Stage]*StageStatus{
		workflow.StageUpload:          {Name: "Upload", Complete: false},
		workflow.StageCheckIn:         {Name: "Check-in", Complete: false},
		workflow.StageStartForecast:   {Name: "Start forecast", Complete: false},
		workflow.StagePoll:            {Name: "Forecasting", Complete: false},
		workflow.StageDownloadResults: {Name: "Download results", Complete: false},
	}

	if params.Clean {
		stages[workflow.StageDeleteUpload] = &StageStatus{Name: "Delete upload", Complete: false}
		stages[workflow.StageDeleteReport] = &StageStatus{Name: "Delete report", Complete: false}
	}

	var cleanPolicy workflow.CleanPolicy
	if params.Clean {
		cleanPolicy = workflow.CleanOnSuccess
	} else {
		cleanPolicy = workflow.CleanNever
	}

	return &ForecastModel{
		ctx: ctx,
		api: api,
		params: workflow.RunParams{
			CSVPath:    params.CSVPath,
			Horizon:    params.Horizon,
			Confidence: params.Confidence,
			Title:      params.Title,
			Overrides:  params.Overrides,
		},
		output:         params.Output,
		verbose:        params.Verbose,
		tokenExpiresAt: params.TokenExpiresAt,
		cleanPolicy:    cleanPolicy,
		currentStage:   workflow.StageUpload,
		stages:         stages,
		spinner:        s,
		progress:       p,
	}
}

func NewForecastResultsViewer(ctx context.Context, results []byte) (*ForecastModel, error) {
	m := NewForecastModel(ctx, nil, ForecastParams{})
	if err := m.loadResults(results); err != nil {
		return nil, err
	}
	m.result = &workflow.RunResult{Results: results}
	return m, nil
}

func (m *ForecastModel) Init() tea.Cmd {
	if m.showingResults() {
		return nil
	}
	return tea.Batch(
		m.spinner.Tick,
		m.startWorkflow(),
	)
}

type workflowCompleteMsg struct {
	result *workflow.RunResult
}

type workflowErrorMsg struct {
	err error
}

type jobStartedMsg struct{}

type eventsClosedMsg struct{}

func (m *ForecastModel) startWorkflow() tea.Cmd {
	return func() tea.Msg {
		runner := workflow.NewRunner(m.api,
			workflow.WithPollInterval(5*time.Second),
			workflow.WithClean(m.cleanPolicy),
			workflow.WithTokenExpiry(m.tokenExpiresAt),
		)

		m.job = runner.RunAsync(m.ctx, m.params)
		return jobStartedMsg{}
	}
}

func (m *ForecastModel) subscribeNextEvent() tea.Cmd {
	if m.job == nil {
		return nil
	}
	ch := m.job.Events()
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return eventsClosedMsg{}
		}
		return e
	}
}

func (m *ForecastModel) waitForCompletion() tea.Cmd {
	if m.job == nil {
		return nil
	}
	return func() tea.Msg {
		res, err := m.job.Wait()
		if err != nil {
			return workflowErrorMsg{err: err}
		}
		return workflowCompleteMsg{result: res}
	}
}

func (m *ForecastModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jobStartedMsg:
		return m, tea.Batch(m.subscribeNextEvent(), m.waitForCompletion())
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		return m.handleSpinner(msg)
	case workflow.Event:
		return m.handleWorkflowEvent(msg)
	case eventsClosedMsg:
		return m.handleEventsClosed()
	case workflowCompleteMsg:
		return m.handleWorkflowComplete(msg.result)
	case workflowErrorMsg:
		return m.handleWorkflowError(msg.err)
	case tea.QuitMsg:
		if m.job != nil {
			m.job.Cancel()
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m *ForecastModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if !m.done {
			if m.job != nil {
				m.job.Cancel()
			}
			m.err = fmt.Errorf("cancelled by user")
		}
		m.done = true
		m.quitting = true
		return m, tea.Quit
	case tea.KeyLeft:
		if m.canNavigateResults() {
			m.moveSelection(-1)
		}
		return m, nil
	case tea.KeyRight:
		if m.canNavigateResults() {
			m.moveSelection(1)
		}
		return m, nil
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q', 'Q':
				if !m.done && m.job != nil {
					m.job.Cancel()
					m.err = fmt.Errorf("cancelled by user")
				}
				m.done = true
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *ForecastModel) handleSpinner(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *ForecastModel) handleWorkflowEvent(e workflow.Event) (tea.Model, tea.Cmd) {
	m.applyStageEvent(e)
	return m, m.subscribeNextEvent()
}

func (m *ForecastModel) handleEventsClosed() (tea.Model, tea.Cmd) {
	if m.result != nil && m.err == nil {
		return m, nil
	}
	if m.err != nil {
		m.done = true
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *ForecastModel) handleWorkflowComplete(res *workflow.RunResult) (tea.Model, tea.Cmd) {
	m.result = res
	if err := m.loadResults(res.Results); err != nil {
		m.err = fmt.Errorf("failed to read forecast results: %w", err)
		m.done = true
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *ForecastModel) handleWorkflowError(err error) (tea.Model, tea.Cmd) {
	m.done = true
	m.quitting = true
	m.err = err
	return m, tea.Quit
}

func (m *ForecastModel) applyStageEvent(e workflow.Event) {
	switch e.Kind {
	case workflow.KindStart:
		m.markStageStart(e.Stage, e.At)
	case workflow.KindComplete:
		m.markStageComplete(e.Stage, e.Duration)
	case workflow.KindError:
		m.markStageError(e.Stage, e.Err)
	default:
		m.handleDefaultStageEvent(e)
	}
	m.updateStageProgress(e)
	m.updateReportID(e)
}

func (m *ForecastModel) markStageStart(stage workflow.Stage, at time.Time) {
	m.currentStage = stage
	if status, ok := m.stages[stage]; ok && status.StartTime.IsZero() {
		status.StartTime = at
	}
}

func (m *ForecastModel) markStageComplete(stage workflow.Stage, duration *time.Duration) {
	if status, ok := m.stages[stage]; ok {
		status.Complete = true
		if duration != nil {
			status.Duration = *duration
		}
	}
}

func (m *ForecastModel) markStageError(stage workflow.Stage, err error) {
	if status, ok := m.stages[stage]; ok {
		status.Error = err
	}
}

func (m *ForecastModel) handleDefaultStageEvent(e workflow.Event) {
	switch {
	case e.Duration == nil && e.Err == nil && e.Progress == nil:
		m.markStageStart(e.Stage, e.At)
	case e.Duration != nil:
		m.markStageComplete(e.Stage, e.Duration)
	case e.Err != nil:
		m.markStageError(e.Stage, e.Err)
	}
}

func (m *ForecastModel) updateStageProgress(e workflow.Event) {
	if e.Progress == nil {
		return
	}
	m.completed = e.Progress.Completed
	m.total = e.Progress.Total
	m.running = e.Progress.Running
}

func (m *ForecastModel) updateReportID(e workflow.Event) {
	if e.ReportID == nil {
		return
	}
	m.reportID = *e.ReportID
}

func (m *ForecastModel) View() string {
	var content string
	if m.showingResults() {
		content = m.renderResultView()
	} else {
		content += m.renderHeader()
		for _, stage := range m.stagesToRender() {
			status := m.stages[stage]
			if status == nil {
				continue
			}
			content += m.renderStage(stage, status)
		}
		content += m.renderProgress()
	}

	view := forecastBoxStyle.Render(content)
	if m.quitting {
		return view + "\n"
	}
	return view
}

func (m *ForecastModel) renderHeader() string {
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	whiteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	return grayStyle.Render(
		">_",
	) + " " + whiteStyle.Render(
		"swiftseer",
	) + " " + grayStyle.Render(
		"("+version.GetVersion()+")",
	) + "\n\n"
}

func (m *ForecastModel) stagesToRender() []workflow.Stage {
	list := []workflow.Stage{
		workflow.StageUpload,
		workflow.StageCheckIn,
	}
	if m.cleanPolicy != workflow.CleanNever {
		list = append(list, workflow.StageDeleteUpload)
	}
	list = append(list,
		workflow.StageStartForecast,
		workflow.StagePoll,
		workflow.StageDownloadResults,
	)
	if m.cleanPolicy != workflow.CleanNever {
		list = append(list, workflow.StageDeleteReport)
	}
	return list
}

func (m *ForecastModel) renderStage(stage workflow.Stage, status *StageStatus) string {
	var icon string
	var text string
	if status.Error != nil {
		icon = ErrorStyle.Render("✗")
		text = fmt.Sprintf("%s (failed)", status.Name)
	} else if status.Complete {
		icon = SuccessStyle.Render("✓")
		text = status.Name
		if status.Duration > 0 {
			text += fmt.Sprintf("  (%v)", status.Duration.Round(100*time.Millisecond))
		}
	} else if m.currentStage == stage {
		icon = m.spinner.View()
		text = fmt.Sprintf("%s...", status.Name)
		if !status.StartTime.IsZero() {
			elapsed := time.Since(status.StartTime).Round(time.Second)
			text += fmt.Sprintf("  (%v)", elapsed)
		}
	} else {
		icon = InfoStyle.Render("○")
		text = status.Name
	}
	return fmt.Sprintf("  %s %s\n", icon, text)
}

func (m *ForecastModel) renderProgress() string {
	if m.currentStage != workflow.StagePoll || m.total <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	percentComplete := float64(m.completed) / float64(m.total)
	b.WriteString(fmt.Sprintf("Progress: %d/%d (%s)\n",
		m.completed, m.total,
		progressBarStyle.Render(fmt.Sprintf("%.0f%%", percentComplete*100))))
	b.WriteString(m.progress.ViewAs(percentComplete))
	b.WriteString("\n\n")
	if stage, ok := m.stages[workflow.StagePoll]; ok && !stage.StartTime.IsZero() {
		elapsed := time.Since(stage.StartTime)
		var etaStr string
		if percentComplete > 0 && percentComplete < 1 {
			eta := time.Duration(float64(elapsed) / percentComplete * (1 - percentComplete))
			etaStr = fmt.Sprintf(" | ETA: %v", eta.Round(time.Second))
		}
		var tokenStr string
		tokenRemaining := time.Until(m.tokenExpiresAt)
		if tokenRemaining > 0 {
			formattedTime := tokenRemaining.Round(time.Minute)
			timeDisplay := strings.TrimSuffix(formattedTime.String(), "0s")
			if tokenRemaining < 5*time.Minute {
				tokenStr = fmt.Sprintf(" | Token: %s", ErrorStyle.Render(timeDisplay))
			} else {
				tokenStr = fmt.Sprintf(" | Token: %s", timeDisplay)
			}
		}
		b.WriteString(statsStyle.Render(fmt.Sprintf("Running: %d%s%s", m.running, etaStr, tokenStr)))
		b.WriteString("\n")
	}
	b.WriteString(statsStyle.Render(fmt.Sprintf("Report: %d", m.reportID)))
	return b.String()
}

func (m *ForecastModel) showingResults() bool {
	return m.done && m.err == nil && len(m.results) > 0
}

func (m *ForecastModel) canNavigateResults() bool {
	return m.showingResults() && len(m.results) > 1
}

func (m *ForecastModel) moveSelection(delta int) {
	if len(m.results) == 0 {
		return
	}
	count := len(m.results)
	m.index = (m.index + delta) % count
	if m.index < 0 {
		m.index += count
	}
}

func (m *ForecastModel) renderResultView() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	if len(m.results) == 0 {
		b.WriteString("No forecast results available\n")
		return b.String()
	}
	entry := m.results[m.index]
	for _, line := range wrapSeriesName(entry.Name, resultContentWidth) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	plot, _, forecastCount := buildResultPlot(entry)
	b.WriteString(plot)
	b.WriteString("\n\n")
	writeResultField(&b, "Model:", truncate(entry.Model.Name, 40))
	writeResultField(&b, "History points:", fmt.Sprintf("%d", len(filterValues(entry.Actuals))))
	writeResultField(&b, "Forecast points:", fmt.Sprintf("%d", forecastCount))
	if len(entry.Forecasts) > 0 {
		next := entry.Forecasts[0]
		writeResultField(&b, "Next:", fmt.Sprintf("%s  %s", formatTimestamp(next.Timestamp), formatValue(next.Value)))
	}
	if len(entry.Actuals) > 0 {
		last := entry.Actuals[len(entry.Actuals)-1]
		writeResultField(&b, "Last actual:", fmt.Sprintf("%s  %s", formatTimestamp(last.Timestamp), formatValue(last.Value)))
	}
	footer := "← Prev    → Next"
	quit := "    q Quit"
	selection := InfoStyle.Render(fmt.Sprintf("%3d/%3d", m.index+1, len(m.results)))
	footerWidth := len([]rune(footer)) + len([]rune(quit))
	const selectionGap = 12
	padding := resultContentWidth - footerWidth - selectionGap - utf8.RuneCountInString(selection)
	if padding < 0 {
		padding = 0
	}
	b.WriteString("\n")
	b.WriteString(footer)
	b.WriteString(quit)
	b.WriteString(strings.Repeat(" ", selectionGap))
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(selection)
	return b.String()
}

func buildResultPlot(entry forecastResultEntry) (string, int, int) {
	series, historyCount, forecastCount := collectSeries(entry)
	if len(series) == 0 {
		return "No chart data available", historyCount, forecastCount
	}
	width := resultPlotWidth
	if width < 3 {
		width = 3
	}
	resampled := resampleSeries(series, width)
	if len(resampled) == 0 {
		return "No chart data available", historyCount, forecastCount
	}

	if historyCount == 0 {
		graph := asciigraph.Plot(
			resampled,
			asciigraph.Width(len(resampled)),
			asciigraph.Height(resultPlotHeight),
			asciigraph.Offset(0),
			asciigraph.SeriesColors(asciigraph.Red),
		)
		graph = tightenPlotSpacing(graph)
		graph = normalizeYAxisLabels(graph)
		return strings.TrimRight(graph, "\n"), historyCount, forecastCount
	}
	if forecastCount == 0 {
		graph := asciigraph.Plot(
			resampled,
			asciigraph.Width(len(resampled)),
			asciigraph.Height(resultPlotHeight),
			asciigraph.Offset(0),
			asciigraph.SeriesColors(asciigraph.Blue),
		)
		graph = tightenPlotSpacing(graph)
		graph = normalizeYAxisLabels(graph)
		return strings.TrimRight(graph, "\n"), historyCount, forecastCount
	}

	splitCol := calculateSplitColumn(len(series), historyCount, len(resampled))
	data := buildColoredSeries(resampled, splitCol)
	graph := asciigraph.PlotMany(
		data,
		asciigraph.Width(len(resampled)),
		asciigraph.Height(resultPlotHeight),
		asciigraph.Offset(0),
		asciigraph.SeriesColors(asciigraph.Blue, asciigraph.Red),
	)
	graph = tightenPlotSpacing(graph)
	graph = normalizeYAxisLabels(graph)
	return strings.TrimRight(graph, "\n"), historyCount, forecastCount
}

func collectSeries(entry forecastResultEntry) ([]float64, int, int) {
	actuals := filterValues(entry.Actuals)
	forecasts := filterValues(entry.Forecasts)
	forecastCount := len(forecasts)

	historyCount := 0
	if len(actuals) > 0 {
		base := forecastCount / 2
		if len(actuals) >= 3 && base < 3 {
			base = 3
		}
		if base > len(actuals) {
			base = len(actuals)
		}
		if base == 0 && forecastCount > 0 {
			base = 1
		}
		historyCount = base
	}

	series := make([]float64, 0, historyCount+forecastCount)
	if historyCount > 0 {
		series = append(series, actuals[len(actuals)-historyCount:]...)
	}
	if forecastCount > 0 {
		series = append(series, forecasts...)
	}
	if len(series) == 0 && len(actuals) > 0 {
		historyCount = len(actuals)
		series = append(series, actuals...)
	}
	return series, historyCount, forecastCount
}

func resampleSeries(series []float64, width int) []float64 {
	if len(series) == 0 || width <= 0 {
		return nil
	}
	if len(series) == 1 {
		out := make([]float64, width)
		for i := range out {
			out[i] = series[0]
		}
		return out
	}
	if width == 1 {
		return []float64{series[len(series)-1]}
	}
	out := make([]float64, width)
	last := len(series) - 1
	for i := 0; i < width; i++ {
		pos := float64(i) * float64(last) / float64(width-1)
		j := int(math.Floor(pos))
		if j >= last {
			out[i] = series[last]
			continue
		}
		t := pos - float64(j)
		out[i] = series[j]*(1-t) + series[j+1]*t
	}
	return out
}

func calculateSplitColumn(originalLen, historyCount, resampledLen int) int {
	if resampledLen <= 0 {
		return 0
	}
	if historyCount <= 0 {
		return 0
	}
	if resampledLen == 1 || originalLen <= 1 {
		return 0
	}
	splitIndex := historyCount - 1
	if splitIndex < 0 {
		splitIndex = 0
	}
	if splitIndex >= originalLen {
		splitIndex = originalLen - 1
	}
	col := int(math.Round(float64(splitIndex) * float64(resampledLen-1) / float64(originalLen-1)))
	if col < 0 {
		col = 0
	}
	if col >= resampledLen {
		col = resampledLen - 1
	}
	return col
}

func buildColoredSeries(series []float64, splitCol int) [][]float64 {
	total := len(series)
	history := make([]float64, total)
	forecast := make([]float64, total)
	for i := 0; i < total; i++ {
		history[i] = math.NaN()
		forecast[i] = math.NaN()
		if i <= splitCol {
			history[i] = series[i]
		}
		if i >= splitCol {
			forecast[i] = series[i]
		}
	}
	if splitCol >= 0 && splitCol < total {
		history[splitCol] = series[splitCol]
		forecast[splitCol] = series[splitCol]
	}
	return [][]float64{history, forecast}
}

func tightenPlotSpacing(plot string) string {
	lines := strings.Split(plot, "\n")
	for i, line := range lines {
		idx := strings.IndexRune(line, '┤')
		if idx == -1 {
			idx = strings.IndexRune(line, '┼')
		}
		if idx == -1 || idx+1 >= len(line) {
			continue
		}
		if line[idx+1] == ' ' {
			lines[i] = line[:idx+1] + line[idx+2:]
		}
	}
	return strings.Join(lines, "\n")
}

func normalizeYAxisLabels(plot string) string {
	lines := strings.Split(plot, "\n")
	const labelWidth = 5
	for i, line := range lines {
		if line == "" {
			continue
		}
		idx := strings.IndexRune(line, '┤')
		if idx == -1 {
			idx = strings.IndexRune(line, '┼')
		}
		if idx != -1 {
			label := line[:idx]
			formatted := formatAxisLabelValue(label, labelWidth)
			if len(formatted) < labelWidth {
				formatted = strings.Repeat(" ", labelWidth-len(formatted)) + formatted
			} else if len(formatted) > labelWidth {
				formatted = formatted[len(formatted)-labelWidth:]
			}
			lines[i] = formatted + line[idx:]
			continue
		}
		idx = strings.IndexRune(line, '│')
		if idx != -1 {
			lines[i] = strings.Repeat(" ", labelWidth) + line[idx:]
			continue
		}
		lines[i] = strings.Repeat(" ", labelWidth) + line
	}
	return strings.Join(lines, "\n")
}

func formatAxisLabelValue(raw string, width int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return strings.Repeat(" ", width)
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		text := truncate(trimmed, width)
		if len(text) < width {
			text = strings.Repeat(" ", width-len(text)) + text
		}
		return text
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	units := []string{"", "k", "M", "B", "T"}
	unitIdx := 0
	for value >= 1000 && unitIdx < len(units)-1 {
		value /= 1000
		unitIdx++
	}
	unit := units[unitIdx]
	for {
		maxDigits := width - len(sign) - len(unit)
		if maxDigits <= 0 {
			text := truncate(trimmed, width)
			if len(text) < width {
				text = strings.Repeat(" ", width-len(text)) + text
			}
			return text
		}
		maxDecimals := 0
		if maxDigits > 1 {
			maxDecimals = maxDigits - 1
		}
		if maxDecimals > 2 {
			maxDecimals = 2
		}
		retry := false
		for decimals := maxDecimals; decimals >= 0; decimals-- {
			rounded := roundToDecimals(value, decimals)
			if rounded >= 1000 && unitIdx < len(units)-1 {
				value = rounded / 1000
				unitIdx++
				unit = units[unitIdx]
				retry = true
				break
			}
			number := formatNumber(rounded, decimals)
			formatted := number
			if unit != "" && !strings.Contains(formatted, ".") && len(formatted)+2 <= maxDigits {
				formatted += ".0"
			}
			if len(formatted) <= maxDigits {
				result := sign + formatted + unit
				if len(result) < width {
					result = strings.Repeat(" ", width-len(result)) + result
				}
				return result
			}
		}
		if retry {
			continue
		}
		number := fmt.Sprintf("%.0f", math.Round(value))
		if len(number) > maxDigits {
			number = number[len(number)-maxDigits:]
		}
		formatted := number
		if unit != "" && !strings.Contains(formatted, ".") && len(formatted)+2 <= maxDigits {
			formatted += ".0"
		}
		result := sign + formatted + unit
		if len(result) < width {
			result = strings.Repeat(" ", width-len(result)) + result
		} else if len(result) > width {
			result = result[len(result)-width:]
		}
		return result
	}
}

func roundToDecimals(value float64, decimals int) float64 {
	if decimals <= 0 {
		return math.Round(value)
	}
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}

func formatNumber(value float64, decimals int) string {
	if decimals <= 0 {
		return fmt.Sprintf("%.0f", math.Round(value))
	}
	text := strconv.FormatFloat(value, 'f', decimals, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}

func filterValues(points []seriesPoint) []float64 {
	values := make([]float64, 0, len(points))
	for _, p := range points {
		if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) {
			continue
		}
		values = append(values, p.Value)
	}
	return values
}

func writeResultField(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	maxWidth := resultContentWidth - resultFieldLabelWidth
	if maxWidth < 0 {
		maxWidth = 0
	}
	display := truncate(value, maxWidth)
	if label == "" {
		b.WriteString(strings.Repeat(" ", resultFieldLabelWidth))
		b.WriteString(display)
		b.WriteString("\n")
		return
	}
	fmt.Fprintf(b, "%-*s%s\n", resultFieldLabelWidth, label, display)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max == 1 {
		return string(runes[:1])
	}
	return string(runes[:max-1]) + "…"
}

func formatTimestamp(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func formatValue(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "-"
	}
	rounded := math.Round(v)
	if math.Abs(v-rounded) < 0.0001 {
		return addThousands(fmt.Sprintf("%.0f", rounded))
	}
	s := fmt.Sprintf("%.2f", v)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := ""
	if len(parts) > 0 {
		intPart = parts[0]
	}
	formattedInt := addThousands(sign + intPart)
	if len(parts) == 2 {
		return formattedInt + "." + parts[1]
	}
	return formattedInt
}

func addThousands(s string) string {
	if s == "" {
		return s
	}
	sign := ""
	if s[0] == '-' {
		sign = "-"
		s = s[1:]
	}
	if len(s) <= 3 {
		return sign + s
	}
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	var b strings.Builder
	b.WriteString(sign)
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func (m *ForecastModel) loadResults(data []byte) error {
	entries, err := parseForecastResults(data)
	if err != nil {
		return err
	}
	m.results = entries
	if m.index < 0 || m.index >= len(m.results) {
		m.index = 0
	}
	m.done = true
	m.quitting = false
	m.err = nil
	return nil
}

func parseForecastResults(data []byte) ([]forecastResultEntry, error) {
	type rawActual struct {
		Timestamp string  `json:"time_stamp_utc"`
		Value     float64 `json:"value"`
	}
	type rawForecast struct {
		Timestamp string  `json:"time_stamp_utc"`
		Point     float64 `json:"point_forecast_value"`
	}
	type rawRanking struct {
		RankPosition int     `json:"rank_position"`
		Score        float64 `json:"score"`
	}
	type rawModelSelection struct {
		Ranking *rawRanking `json:"ranking"`
	}
	type rawModel struct {
		ModelName            string             `json:"model_name"`
		Status               string             `json:"status"`
		ForecastPlausibility string             `json:"forecast_plausibility"`
		Forecasts            []rawForecast      `json:"forecasts"`
		ModelSelection       *rawModelSelection `json:"model_selection"`
	}
	type rawActuals struct {
		Name        string      `json:"name"`
		Granularity string      `json:"granularity"`
		Values      []rawActual `json:"values"`
	}
	type rawInput struct {
		Actuals rawActuals `json:"actuals"`
	}
	type rawEntry struct {
		Input  rawInput   `json:"input"`
		Models []rawModel `json:"models"`
	}
	var payload []rawEntry
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	entries := make([]forecastResultEntry, 0, len(payload))
	for _, item := range payload {
		if len(item.Models) == 0 {
			continue
		}
		model := item.Models[0]
		entry := forecastResultEntry{
			Name:        item.Input.Actuals.Name,
			Granularity: item.Input.Actuals.Granularity,
			Actuals:     make([]seriesPoint, 0, len(item.Input.Actuals.Values)),
			Forecasts:   make([]seriesPoint, 0, len(model.Forecasts)),
			Model: forecastModelInfo{
				Name:         model.ModelName,
				Status:       model.Status,
				Plausibility: model.ForecastPlausibility,
			},
		}
		if model.ModelSelection != nil && model.ModelSelection.Ranking != nil {
			entry.Model.RankPosition = model.ModelSelection.Ranking.RankPosition
			entry.Model.RankScore = model.ModelSelection.Ranking.Score
		}
		for _, v := range item.Input.Actuals.Values {
			entry.Actuals = append(entry.Actuals, seriesPoint(v))
		}
		for _, f := range model.Forecasts {
			entry.Forecasts = append(entry.Forecasts, seriesPoint{Timestamp: f.Timestamp, Value: f.Point})
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no forecast models in result payload")
	}
	return entries, nil
}

func (m *ForecastModel) Result() (*workflow.RunResult, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.result, m.output, nil
}
