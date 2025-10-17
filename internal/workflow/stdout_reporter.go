package workflow

import (
	"fmt"
	"io"
	"time"
)

type StdoutReporter struct {
    w       io.Writer
    verbose bool

    lastCompleted int
    lastTotal     int
}

func NewStdoutReporter(w io.Writer, verbose bool) *StdoutReporter {
	return &StdoutReporter{
		w:       w,
		verbose: verbose,
	}
}

func (s *StdoutReporter) OnEvent(e Event) {
	switch e.Stage {
	case StageUpload:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "error: upload failed: %v\n", e.Err)
		} else if e.Kind == KindComplete || e.UploadID != nil {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "uploaded: upload_id=%s", *e.UploadID)
				if e.FileID != nil {
					_, _ = fmt.Fprintf(s.w, " file_id=%s", *e.FileID)
				}
				_, _ = fmt.Fprintf(s.w, "\n")
			} else {
				_, _ = fmt.Fprintf(s.w, "uploaded\n")
			}
		}

	case StageCheckIn:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "error: check-in failed: %v\n", e.Err)
		} else if e.Kind == KindComplete || e.VersionID != nil {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "checked-in: version_id=%s\n", *e.VersionID)
			} else {
				_, _ = fmt.Fprintf(s.w, "checked-in\n")
			}
		}

	case StageDeleteUpload:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "warning: failed to delete upload: %v\n", e.Err)
		} else if e.Kind == KindComplete {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "upload deleted\n")
			}
		}

	case StageStartForecast:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "error: start forecast failed: %v\n", e.Err)
		} else if e.Kind == KindComplete || e.ReportID != nil {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "forecast started: report_id=%d\n", *e.ReportID)
			} else {
				_, _ = fmt.Fprintf(s.w, "forecast started: report_id=%d\n", *e.ReportID)
			}
		}

	case StagePoll:
		if e.Kind == KindError || e.Err != nil {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "warning: status poll error (retrying): %v\n", e.Err)
			}
		} else if e.Kind == KindUpdate || e.Progress != nil {
            if e.Progress.Completed != s.lastCompleted || e.Progress.Total != s.lastTotal {
				s.lastCompleted = e.Progress.Completed
				s.lastTotal = e.Progress.Total

				percent := float64(0)
				if e.Progress.Total > 0 {
					percent = float64(e.Progress.Completed) / float64(e.Progress.Total) * 100
				}

				_, _ = fmt.Fprintf(s.w, "progress: %d/%d (%.0f%%) running=%d\n",
					e.Progress.Completed,
					e.Progress.Total,
					percent,
					e.Progress.Running)
			}
		} else if e.Kind == KindComplete {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "forecasting complete\n")
			}
		}

	case StageDownloadResults:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "error: download failed: %v\n", e.Err)
		} else if e.Kind == KindComplete {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "results downloaded\n")
			}
		}

	case StageDeleteReport:
		if e.Kind == KindError || e.Err != nil {
			_, _ = fmt.Fprintf(s.w, "warning: failed to delete report: %v\n", e.Err)
		} else if e.Kind == KindComplete {
			if s.verbose {
				_, _ = fmt.Fprintf(s.w, "report deleted\n")
			}
		}
	}

	if e.TokenRemaining != nil && *e.TokenRemaining > 0 && *e.TokenRemaining < 5*time.Minute {
		if s.verbose {
			_, _ = fmt.Fprintf(s.w, "warning: token expires in %v\n", e.TokenRemaining.Round(time.Minute))
		}
	}
}
