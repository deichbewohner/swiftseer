package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

type CleanPolicy int

const (
    CleanNever CleanPolicy = iota
    CleanOnSuccess
    CleanAlways
)

type Runner struct {
	api            ForecastAPI
	reporter       Reporter
	pollInterval   time.Duration
	clean          CleanPolicy
	tokenExp       *time.Time
	clock          Clock
	checkInBuilder CheckInRequestBuilder
}

const (
    DefaultPollInterval = 10 * time.Second
    UploadTimeout      = 60 * time.Second
    CheckInTimeout     = 10 * time.Minute
    StartTimeout       = 30 * time.Second
    PollRequestTimeout = 30 * time.Second
    DownloadTimeout    = 5 * time.Minute
    DeleteTimeout      = 30 * time.Second
)

type RunParams struct {
	CSVPath    string
	Horizon    int
	Confidence float64
	Title      string
}

type RunResult struct {
	ReportID int
	Status   *models.StatusResponse
	Results  []byte
	Total    time.Duration
}

type Option func(*Runner)

func WithReporter(rep Reporter) Option {
	return func(r *Runner) {
		r.reporter = rep
	}
}

func WithPollInterval(d time.Duration) Option {
	return func(r *Runner) {
		r.pollInterval = d
	}
}

func WithClean(p CleanPolicy) Option {
	return func(r *Runner) {
		r.clean = p
	}
}

func WithTokenExpiry(t time.Time) Option {
	return func(r *Runner) {
		r.tokenExp = &t
	}
}

func WithClock(c Clock) Option {
	return func(r *Runner) {
		r.clock = c
	}
}

func WithCheckInBuilder(b CheckInRequestBuilder) Option {
	return func(r *Runner) {
		r.checkInBuilder = b
	}
}

func NewRunner(api ForecastAPI, opts ...Option) *Runner {
	r := &Runner{
		api:            api,
		reporter:       NoopReporter{},
		pollInterval:   DefaultPollInterval,
		clean:          CleanNever,
		clock:          realClock{},
		checkInBuilder: defaultCheckInRequestBuilder{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type RunError struct {
	Stage Stage
	Err   error
}

func (e *RunError) Error() string {
	return fmt.Sprintf("%s: %v", e.Stage, e.Err)
}

func (e *RunError) Unwrap() error {
	return e.Err
}

func (r *Runner) Run(ctx context.Context, params RunParams) (*RunResult, error) {
	start := r.clock.Now()

	var userInputID, fileID, versionID string
	var reportID int
	var err error

    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageUpload, Kind: KindStart})

		uploadCtx, cancel := context.WithTimeout(ctx, UploadTimeout)
		defer cancel()

		uploadResult, err := r.api.UploadCSV(uploadCtx, params.CSVPath)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageUpload, Kind: KindError, Err: err},
			)
			return nil, &RunError{Stage: StageUpload, Err: err}
		}

		userInputID = uploadResult.UserInputID
		fileID = uploadResult.FileID
		duration := r.clock.Since(stageStart)

		r.reporter.OnEvent(Event{
			At:       r.clock.Now(),
			Stage:    StageUpload,
			Kind:     KindComplete,
			UploadID: &userInputID,
			FileID:   &fileID,
			Duration: &duration,
		})
	}

    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageCheckIn, Kind: KindStart})

		checkInReq, err := r.checkInBuilder.Build(fileID, params.CSVPath)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageCheckIn, Kind: KindError, Err: err},
			)
			return nil, &RunError{
				Stage: StageCheckIn,
				Err:   fmt.Errorf("failed to build check-in request: %w", err),
			}
		}

		checkInCtx, cancel := context.WithTimeout(ctx, CheckInTimeout)
		defer cancel()

		versionID, err = r.api.CheckIn(checkInCtx, userInputID, fileID, checkInReq)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageCheckIn, Kind: KindError, Err: err},
			)
			return nil, &RunError{Stage: StageCheckIn, Err: err}
		}

		duration := r.clock.Since(stageStart)
		r.reporter.OnEvent(Event{
			At:        r.clock.Now(),
			Stage:     StageCheckIn,
			Kind:      KindComplete,
			VersionID: &versionID,
			Duration:  &duration,
		})
	}

    if r.clean == CleanOnSuccess || r.clean == CleanAlways {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageDeleteUpload, Kind: KindStart})

		deleteCtx, cancel := context.WithTimeout(ctx, DeleteTimeout)
		defer cancel()

		if err := r.api.DeleteUpload(deleteCtx, userInputID); err != nil {
            // best-effort: report but don't fail
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageDeleteUpload, Kind: KindError, Err: err},
			)
		} else {
			duration := r.clock.Since(stageStart)
			r.reporter.OnEvent(Event{
				At:       r.clock.Now(),
				Stage:    StageDeleteUpload,
				Kind:     KindComplete,
				Duration: &duration,
			})
		}
	}

    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageStartForecast, Kind: KindStart})

		forecastReq := &models.ForecastRequest{
			Version: versionID,
			Config: models.ForecastConfig{
				Title: params.Title,
				Forecasting: models.ForecastingConfig{
					FcHorizon:       params.Horizon,
					ConfidenceLevel: params.Confidence,
				},
			},
		}

		startCtx, cancel := context.WithTimeout(ctx, StartTimeout)
		defer cancel()

		reportID, err = r.api.StartForecast(startCtx, versionID, forecastReq)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageStartForecast, Kind: KindError, Err: err},
			)
			return nil, &RunError{Stage: StageStartForecast, Err: err}
		}

		duration := r.clock.Since(stageStart)
		r.reporter.OnEvent(Event{
			At:       r.clock.Now(),
			Stage:    StageStartForecast,
			Kind:     KindComplete,
			ReportID: &reportID,
			Duration: &duration,
		})
	}

    var status *models.StatusResponse
    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(
			Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindStart, ReportID: &reportID},
		)

		ticker := r.clock.NewTicker(r.pollInterval)
		defer ticker.Stop()

		firstPoll := true

		for {
            if !firstPoll {
				select {
				case <-ctx.Done():
					r.reporter.OnEvent(
						Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: ctx.Err()},
					)
					return nil, &RunError{Stage: StagePoll, Err: ctx.Err()}
				case <-ticker.Chan():
				}
			}
			firstPoll = false

            select {
			case <-ctx.Done():
				r.reporter.OnEvent(
					Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: ctx.Err()},
				)
				return nil, &RunError{Stage: StagePoll, Err: ctx.Err()}
			default:
			}

            pollCtx, cancel := context.WithTimeout(ctx, PollRequestTimeout)
			currentStatus, err := r.api.GetStatus(pollCtx, reportID)
			cancel()

			if err != nil {
                // transient polling error: report and retry
				r.reporter.OnEvent(
					Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: err},
				)
				continue
			}

            progress := Progress{
				Completed: currentStatus.StatusSummary.Computed,
				Total:     currentStatus.StatusSummary.Created,
				Running:   currentStatus.StatusSummary.Running,
			}
			r.reporter.OnEvent(Event{
				At:       r.clock.Now(),
				Stage:    StagePoll,
				Kind:     KindUpdate,
				ReportID: &reportID,
				Progress: &progress,
			})

			if r.tokenExp != nil {
				remaining := r.clock.Until(*r.tokenExp)
				if remaining > 0 {
					r.reporter.OnEvent(Event{
						At:             r.clock.Now(),
						Stage:          StagePoll,
						Kind:           KindUpdate,
						TokenRemaining: &remaining,
					})
				}
			}

			if currentStatus.StatusSummary.IsComplete() {
				status = currentStatus
				duration := r.clock.Since(stageStart)
				r.reporter.OnEvent(Event{
					At:       r.clock.Now(),
					Stage:    StagePoll,
					Kind:     KindComplete,
					Duration: &duration,
				})
				break
			}
		}
	}

	var results []byte
	{
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageDownloadResults, Kind: KindStart})

		opts := &models.ResultsOptions{
			IncludeKBestModels:     1,
			IncludeBacktesting:     false,
			IncludeDiscardedModels: false,
		}

		downloadCtx, cancel := context.WithTimeout(ctx, DownloadTimeout)
		defer cancel()

		results, err = r.api.GetResults(downloadCtx, reportID, opts)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageDownloadResults, Kind: KindError, Err: err},
			)
			return nil, &RunError{Stage: StageDownloadResults, Err: err}
		}

		duration := r.clock.Since(stageStart)
		r.reporter.OnEvent(Event{
			At:       r.clock.Now(),
			Stage:    StageDownloadResults,
			Kind:     KindComplete,
			Duration: &duration,
		})
	}

	if r.clean == CleanOnSuccess || r.clean == CleanAlways {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageDeleteReport, Kind: KindStart})

		deleteCtx, cancel := context.WithTimeout(ctx, DeleteTimeout)
		defer cancel()

		if err := r.api.DeleteReport(deleteCtx, reportID); err != nil {
			// best-effort: report but don't fail
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageDeleteReport, Kind: KindError, Err: err},
			)
		} else {
			duration := r.clock.Since(stageStart)
			r.reporter.OnEvent(Event{
				At:       r.clock.Now(),
				Stage:    StageDeleteReport,
				Kind:     KindComplete,
				Duration: &duration,
			})
		}
	}

	return &RunResult{
		ReportID: reportID,
		Status:   status,
		Results:  results,
		Total:    r.clock.Since(start),
	}, nil
}

func (r *Runner) Resume(ctx context.Context, reportID int) (*RunResult, error) {
	start := r.clock.Now()

    var status *models.StatusResponse
    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(
			Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindStart, ReportID: &reportID},
		)

		ticker := r.clock.NewTicker(r.pollInterval)
		defer ticker.Stop()

		firstPoll := true

		for {
            if !firstPoll {
				select {
				case <-ctx.Done():
					r.reporter.OnEvent(
						Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: ctx.Err()},
					)
					return nil, &RunError{Stage: StagePoll, Err: ctx.Err()}
				case <-ticker.Chan():
				}
			}
			firstPoll = false

            select {
			case <-ctx.Done():
				r.reporter.OnEvent(
					Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: ctx.Err()},
				)
				return nil, &RunError{Stage: StagePoll, Err: ctx.Err()}
			default:
			}

            pollCtx, cancel := context.WithTimeout(ctx, PollRequestTimeout)
			currentStatus, err := r.api.GetStatus(pollCtx, reportID)
			cancel()

			if err != nil {
                // transient polling error: report and retry
				r.reporter.OnEvent(
					Event{At: r.clock.Now(), Stage: StagePoll, Kind: KindError, Err: err},
				)
				continue
			}

            progress := Progress{
				Completed: currentStatus.StatusSummary.Computed,
				Total:     currentStatus.StatusSummary.Created,
				Running:   currentStatus.StatusSummary.Running,
			}
			r.reporter.OnEvent(Event{
				At:       r.clock.Now(),
				Stage:    StagePoll,
				Kind:     KindUpdate,
				ReportID: &reportID,
				Progress: &progress,
			})

			if r.tokenExp != nil {
				remaining := r.clock.Until(*r.tokenExp)
				if remaining > 0 {
					r.reporter.OnEvent(Event{
						At:             r.clock.Now(),
						Stage:          StagePoll,
						Kind:           KindUpdate,
						TokenRemaining: &remaining,
					})
				}
			}

			if currentStatus.StatusSummary.IsComplete() {
				status = currentStatus
				duration := r.clock.Since(stageStart)
				r.reporter.OnEvent(Event{
					At:       r.clock.Now(),
					Stage:    StagePoll,
					Kind:     KindComplete,
					Duration: &duration,
				})
				break
			}
		}
	}

    var results []byte
    {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageDownloadResults, Kind: KindStart})

		opts := &models.ResultsOptions{
			IncludeKBestModels:     1,
			IncludeBacktesting:     false,
			IncludeDiscardedModels: false,
		}

		downloadCtx, cancel := context.WithTimeout(ctx, DownloadTimeout)
		defer cancel()

		var err error
		results, err = r.api.GetResults(downloadCtx, reportID, opts)
		if err != nil {
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageDownloadResults, Kind: KindError, Err: err},
			)
			return nil, &RunError{Stage: StageDownloadResults, Err: err}
		}

		duration := r.clock.Since(stageStart)
		r.reporter.OnEvent(Event{
			At:       r.clock.Now(),
			Stage:    StageDownloadResults,
			Kind:     KindComplete,
			Duration: &duration,
		})
	}

	if r.clean == CleanOnSuccess || r.clean == CleanAlways {
		stageStart := r.clock.Now()
		r.reporter.OnEvent(Event{At: r.clock.Now(), Stage: StageDeleteReport, Kind: KindStart})

		deleteCtx, cancel := context.WithTimeout(ctx, DeleteTimeout)
		defer cancel()

		if err := r.api.DeleteReport(deleteCtx, reportID); err != nil {
			// best-effort: report but don't fail
			r.reporter.OnEvent(
				Event{At: r.clock.Now(), Stage: StageDeleteReport, Kind: KindError, Err: err},
			)
		} else {
			duration := r.clock.Since(stageStart)
			r.reporter.OnEvent(Event{
				At:       r.clock.Now(),
				Stage:    StageDeleteReport,
				Kind:     KindComplete,
				Duration: &duration,
			})
		}
	}

	return &RunResult{
		ReportID: reportID,
		Status:   status,
		Results:  results,
		Total:    r.clock.Since(start),
	}, nil
}
