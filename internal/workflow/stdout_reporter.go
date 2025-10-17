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
		s.handleUpload(e)
	case StageCheckIn:
		s.handleCheckIn(e)
	case StageDeleteUpload:
		s.handleDeleteUpload(e)
	case StageStartForecast:
		s.handleStartForecast(e)
	case StagePoll:
		s.handlePoll(e)
	case StageDownloadResults:
		s.handleDownloadResults(e)
	case StageDeleteReport:
		s.handleDeleteReport(e)
	}

	if e.TokenRemaining != nil && *e.TokenRemaining > 0 && *e.TokenRemaining < 5*time.Minute {
		if s.verbose {
			_, _ = fmt.Fprintf(s.w, "warning: token expires in %v\n", e.TokenRemaining.Round(time.Minute))
		}
	}
}

func (s *StdoutReporter) handleUpload(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "error: upload failed: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete || e.UploadID != nil {
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
}

func (s *StdoutReporter) handleCheckIn(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "error: check-in failed: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete || e.VersionID != nil {
		if s.verbose {
			_, _ = fmt.Fprintf(s.w, "checked-in: version_id=%s\n", *e.VersionID)
		} else {
			_, _ = fmt.Fprintf(s.w, "checked-in\n")
		}
	}
}

func (s *StdoutReporter) handleDeleteUpload(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "warning: failed to delete upload: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete && s.verbose {
		_, _ = fmt.Fprintf(s.w, "upload deleted\n")
	}
}

func (s *StdoutReporter) handleStartForecast(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "error: start forecast failed: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete || e.ReportID != nil {
		_, _ = fmt.Fprintf(s.w, "forecast started: report_id=%d\n", *e.ReportID)
	}
}

func (s *StdoutReporter) handlePoll(e Event) {
	if e.Kind == KindError || e.Err != nil {
		if s.verbose {
			_, _ = fmt.Fprintf(s.w, "warning: status poll error (retrying): %v\n", e.Err)
		}
		return
	}
	if e.Kind == KindUpdate || e.Progress != nil {
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
		return
	}
	if e.Kind == KindComplete && s.verbose {
		_, _ = fmt.Fprintf(s.w, "forecasting complete\n")
	}
}

func (s *StdoutReporter) handleDownloadResults(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "error: download failed: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete && s.verbose {
		_, _ = fmt.Fprintf(s.w, "results downloaded\n")
	}
}

func (s *StdoutReporter) handleDeleteReport(e Event) {
	if e.Kind == KindError || e.Err != nil {
		_, _ = fmt.Fprintf(s.w, "warning: failed to delete report: %v\n", e.Err)
		return
	}
	if e.Kind == KindComplete && s.verbose {
		_, _ = fmt.Fprintf(s.w, "report deleted\n")
	}
}
