package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

type fakeAPI struct {
	uploadRes    *UploadResult
	checkInErr   error
	versionID    string
	reportID     int
	statuses     []*models.StatusResponse
	resultsBody  []byte
	delUploadErr error
	delReportErr error
}

func (f *fakeAPI) UploadCSV(ctx context.Context, path string) (*UploadResult, error) {
	return f.uploadRes, nil
}

func (f *fakeAPI) CheckIn(
	ctx context.Context,
	userInputID, fileID string,
	req *models.CheckInRequest,
) (string, error) {
	return f.versionID, f.checkInErr
}

func (f *fakeAPI) StartForecast(
	ctx context.Context,
	versionID string,
	req *models.ForecastRequest,
) (int, error) {
	return f.reportID, nil
}

func (f *fakeAPI) GetStatus(ctx context.Context, reportID int) (*models.StatusResponse, error) {
	if len(f.statuses) == 0 {
		return &models.StatusResponse{StatusSummary: models.StatusSummary{}}, nil
	}
	s := f.statuses[0]
	f.statuses = f.statuses[1:]
	return s, nil
}

func (f *fakeAPI) GetResults(
	ctx context.Context,
	reportID int,
	opts *models.ResultsOptions,
) ([]byte, error) {
	return f.resultsBody, nil
}

func (f *fakeAPI) DeleteUpload(
	ctx context.Context,
	userInputID string,
) error {
	return f.delUploadErr
}

func (f *fakeAPI) DeleteReport(
	ctx context.Context,
	reportID int,
) error {
	return f.delReportErr
}

type stubBuilder struct{}

func (stubBuilder) Build(fileUUID, csvPath string) (*models.CheckInRequest, error) {
	return &models.CheckInRequest{RawDataSource: fileUUID}, nil
}

type fastClock struct{ now time.Time }

func (c fastClock) Now() time.Time                   { return c.now }
func (c fastClock) Since(t time.Time) time.Duration  { return c.now.Sub(t) }
func (c fastClock) Until(t time.Time) time.Duration  { return t.Sub(c.now) }
func (c fastClock) NewTicker(d time.Duration) Ticker { return fastTicker{} }

type fastTicker struct{}

func (fastTicker) Chan() <-chan time.Time { ch := make(chan time.Time); close(ch); return ch }
func (fastTicker) Stop()                  {}

type captureReporter struct{ events []Event }

func (c *captureReporter) OnEvent(e Event) { c.events = append(c.events, e) }

func TestRunner_Run_Success(t *testing.T) {
	api := &fakeAPI{
		uploadRes: &UploadResult{UserInputID: "u1", FileID: "f1"},
		versionID: "v1",
		reportID:  42,
		statuses: []*models.StatusResponse{
			{
				StatusSummary: models.StatusSummary{
					Created:    10,
					Computed:   10,
					Running:    0,
					Successful: 10,
				},
			},
		},
		resultsBody: []byte(`{"ok":true}`),
	}

	rep := &captureReporter{}
	r := NewRunner(api,
		WithReporter(rep),
		WithPollInterval(1*time.Millisecond),
		WithCheckInBuilder(stubBuilder{}),
		WithClock(fastClock{now: time.Unix(0, 0)}),
	)

	ctx := context.Background()
	res, err := r.Run(
		ctx,
		RunParams{CSVPath: "file.csv", Horizon: 12, Confidence: 0.75, Title: "t"},
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if res.ReportID != 42 {
		t.Fatalf("ReportID = %d, want 42", res.ReportID)
	}
	if len(res.Results) == 0 {
		t.Fatal("Results should not be empty")
	}

	// We expect to see at least these stages represented in events
	wantStages := []Stage{
		StageUpload,
		StageCheckIn,
		StageStartForecast,
		StagePoll,
		StageDownloadResults,
	}
	for _, s := range wantStages {
		found := false
		for _, e := range rep.events {
			if e.Stage == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected event for stage %v", s)
		}
	}
}

func TestRunner_Resume_Success(t *testing.T) {
	api := &fakeAPI{
		reportID: 7,
		statuses: []*models.StatusResponse{
			{StatusSummary: models.StatusSummary{Created: 1, Computed: 1, Successful: 1}},
		},
		resultsBody: []byte(`[]`),
	}
	rep := &captureReporter{}
	r := NewRunner(
		api,
		WithReporter(rep),
		WithPollInterval(1*time.Millisecond),
		WithClock(fastClock{now: time.Unix(0, 0)}),
	)

	res, err := r.Resume(context.Background(), 7)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if res.ReportID != 7 {
		t.Fatalf("ReportID = %d, want 7", res.ReportID)
	}
}
