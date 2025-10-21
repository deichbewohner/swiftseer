package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deichbewohner/swiftseer/internal/client"
	"github.com/deichbewohner/swiftseer/internal/config"
	"github.com/deichbewohner/swiftseer/internal/models"
	"github.com/deichbewohner/swiftseer/internal/ui"
	"github.com/deichbewohner/swiftseer/internal/workflow"
	"golang.org/x/term"
)

func ForecastCmd(args []string) error {
	opts, err := parseForecastFlags(args)
	if err != nil {
		return err
	}
	horizon, confidence, output, title := opts.Horizon, opts.Confidence, opts.Output, opts.Title
	verbose, reportID, clean, noUI, jsonEvents := opts.Verbose, opts.ReportID, opts.Clean, opts.NoUI, opts.JSONEvents
	csvPath := opts.CSVPath

	if reportID == 0 {
		if err := validateNewRunInputs(csvPath, horizon, confidence); err != nil {
			return err
		}
	}

	cfgManager, err := config.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.RefreshToken == "" || !cfg.IsRefreshTokenValid() {
		return fmt.Errorf(
			"not authenticated or refresh token expired. Please run:\n  swiftseer login",
		)
	}

	env, err := config.GetEnvironment(cfg.Environment)
	if err != nil {
		return fmt.Errorf("invalid environment: %w", err)
	}

	authClient := client.NewAuthClient(env.GetAuthTokenURL(), nil)
	apiClient := client.NewClient(env.APIURL, cfg.Group, cfg, cfgManager, authClient, nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	useTUI := shouldUseTUI(
		reportID,
		noUI,
		jsonEvents,
		term.IsTerminal(int(os.Stdin.Fd())),
		term.IsTerminal(int(os.Stdout.Fd())),
	)

	overrides := buildOverrides(opts)
	if useTUI {
		return runTUIForecast(ctx, cfg, apiClient, csvPath, horizon, confidence, title, output, verbose, clean, overrides)
	}

	if reportID > 0 {
		return resumeForecast(ctx, apiClient, reportID, clean, jsonEvents, verbose, output)
	}

	return runHeadlessForecast(ctx, apiClient, csvPath, horizon, confidence, title, clean, jsonEvents, verbose, output, overrides)
}

func newWorkflowRunner(
	apiClient *client.Client,
	cleanPolicy workflow.CleanPolicy,
	jsonEvents bool,
	verbose bool,
) (*workflow.Runner, workflow.Reporter) {
	api := workflow.NewClientAdapter(apiClient)
	var reporter workflow.Reporter
	if jsonEvents {
		reporter = workflow.NewJSONReporter(os.Stdout)
	} else {
		reporter = workflow.NewStdoutReporter(os.Stdout, verbose)
	}
	return workflow.NewRunner(
		api,
		workflow.WithReporter(reporter),
		workflow.WithPollInterval(workflow.DefaultPollInterval),
		workflow.WithClean(cleanPolicy),
	), reporter
}

func shouldUseTUI(reportID int, noUI, jsonEvents, stdinTTY, stdoutTTY bool) bool {
	if reportID != 0 {
		return false
	}
	if noUI {
		return false
	}
	if jsonEvents {
		return false
	}
	if !stdinTTY || !stdoutTTY {
		return false
	}
	return true
}

func validateNewRunInputs(csvPath string, horizon int, confidence float64) error {
	if _, err := os.Stat(csvPath); err != nil {
		return fmt.Errorf("CSV file not found: %s", csvPath)
	}
	if horizon <= 0 {
		return fmt.Errorf("horizon must be positive")
	}
	if confidence <= 0 || confidence >= 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

func cleanPolicyOf(clean bool) workflow.CleanPolicy {
	if clean {
		return workflow.CleanOnSuccess
	}
	return workflow.CleanNever
}

func runTUIForecast(
	ctx context.Context,
	cfg *config.Config,
	apiClient *client.Client,
	csvPath string,
	horizon int,
	confidence float64,
	title string,
	output string,
	verbose bool,
	clean bool,
	overrides *models.CheckInOverrides,
) error {
	params := ui.ForecastParams{
		CSVPath:        csvPath,
		Horizon:        horizon,
		Confidence:     confidence,
		Title:          title,
		Output:         output,
		Verbose:        verbose,
		TokenExpiresAt: cfg.TokenExpiresAt,
		Clean:          clean,
		Overrides:      overrides,
	}
	api := workflow.NewClientAdapter(apiClient)
	_ = apiClient.EnsureValidToken(ctx)
	params.TokenExpiresAt = cfg.TokenExpiresAt
	forecastModel := ui.NewForecastModel(ctx, api, params)
	p := tea.NewProgram(forecastModel)

	var terminalState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		terminalState, _ = term.GetState(int(os.Stdin.Fd()))
	}
	defer func() {
		if r := recover(); r != nil {
			if terminalState != nil {
				_ = term.Restore(int(os.Stdin.Fd()), terminalState)
			}
			panic(r)
		}
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("forecast UI error: %w", err)
	}
	result := finalModel.(*ui.ForecastModel)
	workflowResult, outputPath, err := result.Result()
	if err != nil {
		return fmt.Errorf("forecast failed: %w", err)
	}
	if err := os.WriteFile(outputPath, workflowResult.Results, 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	return nil
}

func resumeForecast(
	ctx context.Context,
	apiClient *client.Client,
	reportID int,
	clean bool,
	jsonEvents bool,
	verbose bool,
	output string,
) error {
	cleanPolicy := cleanPolicyOf(clean)
	runner, reporter := newWorkflowRunner(apiClient, cleanPolicy, jsonEvents, verbose)
	if !jsonEvents {
		now := time.Now()
		rid := reportID
		reporter.OnEvent(workflow.Event{At: now, Stage: workflow.StageStartForecast, Kind: workflow.KindComplete, ReportID: &rid})
	}
	result, err := runner.Resume(ctx, reportID)
	if err != nil {
		return fmt.Errorf("forecast resume failed: %w", err)
	}
	if err := os.WriteFile(output, result.Results, 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	return nil
}

func runHeadlessForecast(
	ctx context.Context,
	apiClient *client.Client,
	csvPath string,
	horizon int,
	confidence float64,
	title string,
	clean bool,
	jsonEvents bool,
	verbose bool,
	output string,
	overrides *models.CheckInOverrides,
) error {
	cleanPolicy := cleanPolicyOf(clean)
	runner, _ := newWorkflowRunner(apiClient, cleanPolicy, jsonEvents, verbose)
	result, err := runner.Run(ctx, workflow.RunParams{CSVPath: csvPath, Horizon: horizon, Confidence: confidence, Title: title, Overrides: overrides})
	if err != nil {
		return fmt.Errorf("forecast workflow failed: %w", err)
	}
	if err := os.WriteFile(output, result.Results, 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %w", err)
	}
	return nil
}

func buildOverrides(opts *forecastOptions) *models.CheckInOverrides {
	if opts == nil {
		return nil
	}
	has := false
	o := &models.CheckInOverrides{}
	if opts.DateColumn != "" {
		o.DateColumn = opts.DateColumn
		has = true
	}
	if opts.DateFormat != "" {
		o.DateFormat = opts.DateFormat
		has = true
	}
	if len(opts.ValueCols) > 0 {
		o.ValueColumns = opts.ValueCols
		has = true
	}
	if len(opts.GroupCols) > 0 {
		o.GroupColumns = opts.GroupCols
		has = true
	}
	if !has {
		return nil
	}
	return o
}
