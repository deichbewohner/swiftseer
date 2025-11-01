package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
)

var forecastBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240")).
	Padding(0, 1).
	Width(50)

var resultPlotPlaceholder = buildResultPlaceholder()

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

func buildResultPlaceholder() string {
	const (
		contentWidth = 48
		height       = 8
	)
	innerWidth := contentWidth - 2
	pivot := innerWidth / 2
	heights := make([]int, innerWidth)
	for col := 0; col < innerWidth; col++ {
		var h float64
		if col < pivot {
			denom := float64(maxInt(1, pivot-1))
			ratio := float64(col) / denom
			h = 2 + 3*math.Sin(ratio*math.Pi)
		} else {
			denom := float64(maxInt(1, innerWidth-pivot-1))
			ratio := float64(col-pivot) / denom
			h = 3 + 4*ratio
		}
		heights[col] = clampInt(int(math.Round(h)), 0, height-1)
	}
	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", innerWidth) + "┐\n")
	for row := 0; row < height; row++ {
		b.WriteString("│")
		level := height - row - 1
		for col := 0; col < innerWidth; col++ {
			if col == pivot {
				b.WriteRune('│')
				continue
			}
			h := heights[col]
			if h >= level {
				if col < pivot {
					b.WriteRune('█')
				} else {
					b.WriteRune('░')
				}
			} else {
				b.WriteRune(' ')
			}
		}
		b.WriteString("│\n")
	}
	b.WriteString("└" + strings.Repeat("─", innerWidth) + "┘")
	return b.String()
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

func (m *ForecastModel) currentResult() *forecastResultEntry {
	if len(m.results) == 0 {
		return nil
	}
	if m.index < 0 || m.index >= len(m.results) {
		return &m.results[0]
	}
	return &m.results[m.index]
}

func (m *ForecastModel) renderResultView() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	if len(m.results) == 0 {
		b.WriteString("No forecast results available\n")
		return b.String()
	}
	entry := m.results[m.index]
	b.WriteString(fmt.Sprintf("Forecast %d/%d\n", m.index+1, len(m.results)))
	if entry.Name != "" {
		b.WriteString(fmt.Sprintf("Series: %s\n", truncate(entry.Name, 40)))
	}
	b.WriteString("\n")
	b.WriteString(resultPlotPlaceholder)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Model: %s\n", truncate(entry.Model.Name, 40)))
	meta := buildModelMeta(entry)
	if meta != "" {
		b.WriteString(truncate(meta, 48))
		b.WriteString("\n")
	}
	if len(entry.Forecasts) > 0 {
		next := entry.Forecasts[0]
		b.WriteString(fmt.Sprintf("Next: %s  %s\n", formatTimestamp(next.Timestamp), formatValue(next.Value)))
	}
	if len(entry.Actuals) > 0 {
		last := entry.Actuals[len(entry.Actuals)-1]
		b.WriteString(fmt.Sprintf("Last actual: %s  %s\n", formatTimestamp(last.Timestamp), formatValue(last.Value)))
	}
	b.WriteString("\n← Prev    → Next    q Quit\n")
	return b.String()
}

func buildModelMeta(entry forecastResultEntry) string {
	var parts []string
	if entry.Model.Status != "" {
		parts = append(parts, entry.Model.Status)
	}
	if entry.Model.Plausibility != "" {
		parts = append(parts, entry.Model.Plausibility)
	}
	if entry.Model.RankPosition > 0 {
		parts = append(parts, fmt.Sprintf("Rank #%d", entry.Model.RankPosition))
	}
	if entry.Model.RankScore != 0 {
		parts = append(parts, fmt.Sprintf("Score %.2f", entry.Model.RankScore))
	}
	if len(entry.Forecasts) > 0 {
		parts = append(parts, fmt.Sprintf("Horizon %d", len(entry.Forecasts)))
	}
	return strings.Join(parts, "  ")
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
	return fmt.Sprintf("%.2f", v)
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
			entry.Actuals = append(entry.Actuals, seriesPoint{Timestamp: v.Timestamp, Value: v.Value})
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
