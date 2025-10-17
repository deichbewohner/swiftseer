package workflow

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestJSONReporter_Encode(t *testing.T) {
	var buf bytes.Buffer
	rep := NewJSONReporter(&buf)

	at := time.Unix(0, 0).UTC()

    rep.OnEvent(Event{At: at, Stage: StageUpload, Kind: KindStart})

    uploadID, fileID := "u1", "f1"
	dur := 100 * time.Millisecond
	rep.OnEvent(
		Event{
			At:       at,
			Stage:    StageUpload,
			Kind:     KindComplete,
			UploadID: &uploadID,
			FileID:   &fileID,
			Duration: &dur,
		},
	)

    rem := 4 * time.Minute
	reportID := 42
	prog := &Progress{Completed: 1, Total: 2, Running: 1}
	rep.OnEvent(
		Event{
			At:             at,
			Stage:          StagePoll,
			Kind:           KindUpdate,
			ReportID:       &reportID,
			Progress:       prog,
			TokenRemaining: &rem,
		},
	)

    rep.OnEvent(Event{At: at, Stage: StageDownloadResults, Kind: KindError, Err: errSentinel{}})

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 JSON lines, got %d\n%s", len(lines), out)
	}

    want := []string{
		`{"at":"1970-01-01T00:00:00Z","stage":"Upload","kind":"start"}`,
		`{"at":"1970-01-01T00:00:00Z","stage":"Upload","kind":"complete","upload_id":"u1","file_id":"f1","duration_ms":100}`,
		`{"at":"1970-01-01T00:00:00Z","stage":"Forecasting","kind":"update","report_id":42,"progress":{"completed":1,"total":2,"running":1},"token_remaining_s":240}`,
		`{"at":"1970-01-01T00:00:00Z","stage":"Download results","kind":"error","error":"sentinel"}`,
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d mismatch.\n got: %s\nwant: %s", i, lines[i], w)
		}
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }

func TestStdoutReporter_Format(t *testing.T) {
	var buf bytes.Buffer
	rep := NewStdoutReporter(&buf, false) // verbose off to simplify output

	at := time.Unix(0, 0).UTC()

    rep.OnEvent(Event{At: at, Stage: StageUpload, Kind: KindStart})

    uid, fid := "u1", "f1"
	dur := 50 * time.Millisecond
	rep.OnEvent(
		Event{
			At:       at,
			Stage:    StageUpload,
			Kind:     KindComplete,
			UploadID: &uid,
			FileID:   &fid,
			Duration: &dur,
		},
	)

    rep.OnEvent(Event{At: at, Stage: StageCheckIn, Kind: KindStart})
	vid := "v1"
	rep.OnEvent(Event{At: at, Stage: StageCheckIn, Kind: KindComplete, VersionID: &vid})

    rep.OnEvent(Event{At: at, Stage: StagePoll, Kind: KindStart})
	prog := &Progress{Completed: 1, Total: 2, Running: 1}
	rep.OnEvent(Event{At: at, Stage: StagePoll, Kind: KindUpdate, Progress: prog})
	dPoll := 10 * time.Second
	rep.OnEvent(Event{At: at, Stage: StagePoll, Kind: KindComplete, Duration: &dPoll})

    rep.OnEvent(Event{At: at, Stage: StageDownloadResults, Kind: KindStart})
	dDl := 200 * time.Millisecond
	rep.OnEvent(Event{At: at, Stage: StageDownloadResults, Kind: KindComplete, Duration: &dDl})

	got := buf.String()
    var exp strings.Builder
	exp.WriteString("uploaded\n")
	exp.WriteString("checked-in\n")
	exp.WriteString("progress: 1/2 (50%) running=1\n")

	if got != exp.String() {
		t.Errorf(
			"stdout reporter output mismatch.\n--- got ---\n%q\n--- want ---\n%q\n",
			got,
			exp.String(),
		)
	}
}

func TestStdoutReporter_TokenWarning(t *testing.T) {
	var buf bytes.Buffer
	rep := NewStdoutReporter(&buf, true) // verbose on to emit warning

	at := time.Unix(0, 0).UTC()
	rem := 4 * time.Minute

    rep.OnEvent(Event{At: at, Stage: StageUpload, Kind: KindStart, TokenRemaining: &rem})

	got := buf.String()
	want := "warning: token expires in 4m0s\n"
	if got != want {
		t.Errorf("token warning mismatch. got %q want %q", got, want)
	}
}

func TestStdoutReporter_ErrorBranches(t *testing.T) {
	var buf bytes.Buffer
	rep := NewStdoutReporter(&buf, false)
	at := time.Unix(0, 0).UTC()

    rep.OnEvent(Event{At: at, Stage: StageStartForecast, Kind: KindError, Err: errSentinel{}})

    rep.OnEvent(Event{At: at, Stage: StageDownloadResults, Kind: KindError, Err: errSentinel{}})

	out := buf.String()
	if !strings.Contains(out, "error: start forecast failed: sentinel\n") {
		t.Errorf("missing start forecast error line. out=%q", out)
	}
	if !strings.Contains(out, "error: download failed: sentinel\n") {
		t.Errorf("missing download error line. out=%q", out)
	}
}

func TestJSONReporter_OptionalFields(t *testing.T) {
	var buf bytes.Buffer
	rep := NewJSONReporter(&buf)
	at := time.Unix(0, 0).UTC()

    rep.OnEvent(Event{At: at, Stage: StageUpload})

    d := 250 * time.Millisecond
	rep.OnEvent(Event{At: at, Stage: StageDownloadResults, Kind: KindComplete, Duration: &d})

    rep.OnEvent(Event{At: at, Stage: StageStartForecast, Kind: KindError, Err: errSentinel{}})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "\"kind\":\"start\"") {
		t.Errorf("expected kind=start in first line: %s", lines[0])
	}
	if !strings.Contains(lines[1], "\"duration_ms\":250") {
		t.Errorf("missing duration_ms in second line: %s", lines[1])
	}
	if !strings.Contains(lines[2], "\"error\":\"sentinel\"") {
		t.Errorf("missing error in third line: %s", lines[2])
	}
}
