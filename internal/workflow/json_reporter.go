package workflow

import (
	"encoding/json"
	"io"
	"time"
)

type JSONReporter struct {
    enc *json.Encoder
}

func NewJSONReporter(w io.Writer) *JSONReporter {
	enc := json.NewEncoder(w)
	return &JSONReporter{enc: enc}
}

type jsonProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
	Running   int `json:"running"`
}

type jsonEvent struct {
	At              string        `json:"at"`
	Stage           string        `json:"stage"`
	Kind            string        `json:"kind,omitempty"`
	UploadID        *string       `json:"upload_id,omitempty"`
	FileID          *string       `json:"file_id,omitempty"`
	VersionID       *string       `json:"version_id,omitempty"`
	ReportID        *int          `json:"report_id,omitempty"`
	Progress        *jsonProgress `json:"progress,omitempty"`
	TokenRemainingS *int64        `json:"token_remaining_s,omitempty"`
	DurationMs      *int64        `json:"duration_ms,omitempty"`
	Error           *string       `json:"error,omitempty"`
}

func (r *JSONReporter) OnEvent(e Event) {
	j := jsonEvent{
		At:    e.At.Format(time.RFC3339Nano),
		Stage: e.Stage.String(),
	}

    if s := e.Kind.String(); s != "" {
        j.Kind = s
    }

	j.UploadID = e.UploadID
	j.FileID = e.FileID
	j.VersionID = e.VersionID
	j.ReportID = e.ReportID

	if e.Progress != nil {
		j.Progress = &jsonProgress{
			Completed: e.Progress.Completed,
			Total:     e.Progress.Total,
			Running:   e.Progress.Running,
		}
	}
    if e.TokenRemaining != nil {
        secs := int64(e.TokenRemaining.Round(time.Second).Seconds())
        if secs < 0 {
            secs = 0
        }
        j.TokenRemainingS = &secs
    }
	if e.Duration != nil {
		ms := e.Duration.Milliseconds()
		j.DurationMs = &ms
	}
	if e.Err != nil {
		s := e.Err.Error()
		j.Error = &s
	}

    _ = r.enc.Encode(j)
}
