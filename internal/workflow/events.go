package workflow

import (
	"time"
)

type Stage int

const (
	StageUpload Stage = iota
	StageCheckIn
	StageDeleteUpload
	StageStartForecast
	StagePoll
	StageDownloadResults
	StageDeleteReport
)

func (s Stage) String() string {
	switch s {
	case StageUpload:
		return "Upload"
	case StageCheckIn:
		return "Check-in"
	case StageDeleteUpload:
		return "Delete upload"
	case StageStartForecast:
		return "Start forecast"
	case StagePoll:
		return "Forecasting"
	case StageDownloadResults:
		return "Download results"
	case StageDeleteReport:
		return "Delete report"
	default:
		return "Unknown"
	}
}

type EventKind int

const (
	KindStart EventKind = iota
	KindUpdate
	KindComplete
	KindError
)

func (k EventKind) String() string {
	switch k {
	case KindStart:
		return "start"
	case KindUpdate:
		return "update"
	case KindComplete:
		return "complete"
	case KindError:
		return "error"
	default:
		return ""
	}
}

type Progress struct {
	Completed int
	Total     int
	Running   int
}

type Event struct {
	At             time.Time
	Stage          Stage
	Kind           EventKind
	UploadID       *string
	FileID         *string
	VersionID      *string
	ReportID       *int
	Progress       *Progress
	TokenRemaining *time.Duration
	Duration       *time.Duration
	Err            error
}

type Reporter interface {
	OnEvent(Event)
}

type NoopReporter struct{}

func (NoopReporter) OnEvent(Event) {}
