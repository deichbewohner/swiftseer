package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/deichbewohner/swiftseer/internal/version"
	"github.com/deichbewohner/swiftseer/internal/workflow"
)

var forecastBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240")).
	Padding(0, 1).
	Width(50)

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

func (m *ForecastModel) Init() tea.Cmd {
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
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.job != nil {
				m.job.Cancel()
			}
			m.done = true
			m.quitting = true
			m.err = fmt.Errorf("cancelled by user")
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case workflow.Event:
		e := msg

		switch e.Kind {
		case workflow.KindStart:
			m.currentStage = e.Stage
			if stage, ok := m.stages[e.Stage]; ok && stage.StartTime.IsZero() {
				stage.StartTime = e.At
			}
		case workflow.KindComplete:
			if stage, ok := m.stages[e.Stage]; ok {
				stage.Complete = true
				if e.Duration != nil {
					stage.Duration = *e.Duration
				}
			}
		case workflow.KindError:
			if stage, ok := m.stages[e.Stage]; ok {
				stage.Error = e.Err
			}
		case workflow.KindUpdate:
		default:

			if e.Duration == nil && e.Err == nil && e.Progress == nil {
				m.currentStage = e.Stage
				if stage, ok := m.stages[e.Stage]; ok && stage.StartTime.IsZero() {
					stage.StartTime = e.At
				}
			} else if e.Duration != nil {
				if stage, ok := m.stages[e.Stage]; ok {
					stage.Complete = true
					stage.Duration = *e.Duration
				}
			} else if e.Err != nil {
				if stage, ok := m.stages[e.Stage]; ok {
					stage.Error = e.Err
				}
			}
		}

		if e.Progress != nil {
			m.completed = e.Progress.Completed
			m.total = e.Progress.Total
			m.running = e.Progress.Running
		}

		if e.ReportID != nil {
			m.reportID = *e.ReportID
		}

		return m, m.subscribeNextEvent()

	case eventsClosedMsg:
		if m.result != nil {
			m.done = true
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case workflowCompleteMsg:
		m.result = msg.result
		return m, nil

	case workflowErrorMsg:
		m.done = true
		m.quitting = true
		m.err = msg.err
		return m, tea.Quit

	case tea.QuitMsg:
		if m.job != nil {
			m.job.Cancel()
		}
		return m, tea.Quit
	}

	return m, nil
}

func (m *ForecastModel) View() string {
	var content string

	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	whiteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	content += grayStyle.Render(
		">_",
	) + " " + whiteStyle.Render(
		"future",
	) + " " + grayStyle.Render(
		"("+version.GetVersion()+")",
	) + "\n\n"

	stagesToRender := []workflow.Stage{
		workflow.StageUpload,
		workflow.StageCheckIn,
	}
	if m.cleanPolicy != workflow.CleanNever {
		stagesToRender = append(stagesToRender, workflow.StageDeleteUpload)
	}
	stagesToRender = append(stagesToRender,
		workflow.StageStartForecast,
		workflow.StagePoll,
		workflow.StageDownloadResults,
	)
	if m.cleanPolicy != workflow.CleanNever {
		stagesToRender = append(stagesToRender, workflow.StageDeleteReport)
	}

	for _, stage := range stagesToRender {
		status := m.stages[stage]
		if status == nil {
			continue
		}

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

		content += fmt.Sprintf("  %s %s\n", icon, text)
	}

	if m.currentStage == workflow.StagePoll && m.total > 0 {
		content += "\n"

		percentComplete := float64(m.completed) / float64(m.total)
		content += fmt.Sprintf("Progress: %d/%d (%s)\n",
			m.completed, m.total,
			progressBarStyle.Render(fmt.Sprintf("%.0f%%", percentComplete*100)))

		content += m.progress.ViewAs(percentComplete) + "\n\n"

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
					tokenStr = fmt.Sprintf(" | Token: %s",
						ErrorStyle.Render(timeDisplay))
				} else {
					tokenStr = fmt.Sprintf(" | Token: %s", timeDisplay)
				}
			}

			content += statsStyle.Render(fmt.Sprintf(
				"Running: %d%s%s",
				m.running, etaStr, tokenStr))
			content += "\n"
		}

		content += statsStyle.Render(fmt.Sprintf("Report: %d", m.reportID))
	}

	view := forecastBoxStyle.Render(content)
	if m.quitting {
		return view + "\n"
	}
	return view
}

func (m *ForecastModel) Result() (*workflow.RunResult, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.result, m.output, nil
}
