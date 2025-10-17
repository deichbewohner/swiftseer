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
		if _, err := os.Stat(csvPath); err != nil {
			return fmt.Errorf("CSV file not found: %s", csvPath)
		}
		if horizon <= 0 {
			return fmt.Errorf("horizon must be positive")
		}
		if confidence <= 0 || confidence >= 1 {
			return fmt.Errorf("confidence must be between 0 and 1")
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

	var resultsJSON []byte
	var actualReportID int

	if useTUI {
		params := ui.ForecastParams{
			CSVPath:        csvPath,
			Horizon:        horizon,
			Confidence:     confidence,
			Title:          title,
			Output:         output,
			Verbose:        verbose,
			TokenExpiresAt: cfg.TokenExpiresAt,
			Clean:          clean,
		}

		api := workflow.NewClientAdapter(apiClient)
		_ = apiClient.EnsureValidToken(ctx)
		params.TokenExpiresAt = cfg.TokenExpiresAt
		forecastModel := ui.NewForecastModel(ctx, api, params)
		p := tea.NewProgram(forecastModel)

		// restore terminal state on panic
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

	if reportID > 0 {
		actualReportID = reportID

		var cleanPolicy workflow.CleanPolicy
		if clean {
			cleanPolicy = workflow.CleanOnSuccess
		} else {
			cleanPolicy = workflow.CleanNever
		}

		runner, reporter := newWorkflowRunner(apiClient, cleanPolicy, jsonEvents, verbose)

		if !jsonEvents {
			now := time.Now()
			rid := actualReportID
			reporter.OnEvent(workflow.Event{At: now, Stage: workflow.StageStartForecast, Kind: workflow.KindComplete, ReportID: &rid})
		}

		result, err := runner.Resume(ctx, actualReportID)
		if err != nil {
			return fmt.Errorf("forecast resume failed: %w", err)
		}

		resultsJSON = result.Results

		if err := os.WriteFile(output, resultsJSON, 0644); err != nil {
			return fmt.Errorf("failed to write results to file: %w", err)
		}
	} else {
		var cleanPolicy workflow.CleanPolicy
		if clean {
			cleanPolicy = workflow.CleanOnSuccess
		} else {
			cleanPolicy = workflow.CleanNever
		}

		runner, _ := newWorkflowRunner(apiClient, cleanPolicy, jsonEvents, verbose)

		result, err := runner.Run(ctx, workflow.RunParams{
			CSVPath:    csvPath,
			Horizon:    horizon,
			Confidence: confidence,
			Title:      title,
		})
		if err != nil {
			return fmt.Errorf("forecast workflow failed: %w", err)
		}

		if err := os.WriteFile(output, result.Results, 0644); err != nil {
			return fmt.Errorf("failed to write results to file: %w", err)
		}
	}

	return nil
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
