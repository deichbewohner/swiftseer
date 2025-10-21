package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func TestDetectDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		want      string
		wantError bool
	}{
		{
			name:    "comma delimiter",
			content: "date,product,value\n2020-01-01,A,100\n",
			want:    ",",
		},
		{
			name:    "semicolon delimiter",
			content: "date;product;value\n2020-01-01;A;100\n",
			want:    ";",
		},
		{
			name:    "tab delimiter",
			content: "date\tproduct\tvalue\n2020-01-01\tA\t100\n",
			want:    "\t",
		},
		{
			name:      "empty file",
			content:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.csv")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			got, err := detectDelimiter(tmpFile)
			if (err != nil) != tt.wantError {
				t.Errorf("detectDelimiter() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("detectDelimiter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectDateFormat(t *testing.T) {
	tests := []struct {
		name       string
		sampleRows [][]string
		colIdx     int
		want       string
		wantError  bool
	}{
		{
			name: "ISO date format",
			sampleRows: [][]string{
				{"2020-01-01", "100"},
				{"2020-02-01", "200"},
			},
			colIdx: 0,
			want:   "%Y-%m-%d",
		},
		{
			name: "slash date format",
			sampleRows: [][]string{
				{"2020/01/01", "100"},
				{"2020/02/01", "200"},
			},
			colIdx: 0,
			want:   "%Y/%m/%d",
		},
		{
			name: "dot date format",
			sampleRows: [][]string{
				{"01.01.2020", "100"},
				{"01.02.2020", "200"},
			},
			colIdx: 0,
			want:   "%d.%m.%Y",
		},
		{
			name: "invalid date",
			sampleRows: [][]string{
				{"not-a-date", "100"},
			},
			colIdx:    0,
			wantError: true,
		},
		{
			name:       "empty sample",
			sampleRows: [][]string{},
			colIdx:     0,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectDateFormat(tt.sampleRows, tt.colIdx)
			if (err != nil) != tt.wantError {
				t.Errorf("detectDateFormat() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("detectDateFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectValueColumns(t *testing.T) {
	tests := []struct {
		name       string
		header     []string
		sampleRows [][]string
		dateColIdx int
		want       []int
	}{
		{
			name:   "single value column",
			header: []string{"date", "value", "product"},
			sampleRows: [][]string{
				{"2020-01-01", "100", "A"},
				{"2020-02-01", "200", "B"},
			},
			dateColIdx: 0,
			want:       []int{1},
		},
		{
			name:   "multiple value columns",
			header: []string{"date", "units", "revenue", "product"},
			sampleRows: [][]string{
				{"2020-01-01", "100", "1500.50", "A"},
				{"2020-02-01", "200", "3000.75", "B"},
			},
			dateColIdx: 0,
			want:       []int{1, 2},
		},
		{
			name:   "no value columns",
			header: []string{"date", "product"},
			sampleRows: [][]string{
				{"2020-01-01", "A"},
				{"2020-02-01", "B"},
			},
			dateColIdx: 0,
			want:       []int{},
		},
		{
			name:   "value column with empty values",
			header: []string{"date", "value"},
			sampleRows: [][]string{
				{"2020-01-01", "100"},
				{"2020-02-01", ""},
				{"2020-03-01", "300"},
			},
			dateColIdx: 0,
			want:       []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectValueColumns(tt.header, tt.sampleRows, tt.dateColIdx)
			if len(got) != len(tt.want) {
				t.Errorf("detectValueColumns() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("detectValueColumns() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

func TestDetectGroupColumns(t *testing.T) {
	tests := []struct {
		name            string
		header          []string
		sampleRows      [][]string
		dateColIdx      int
		valueColIndices []int
		want            []int
	}{
		{
			name:            "single group column",
			header:          []string{"date", "value", "product"},
			sampleRows:      [][]string{{"2020-01-01", "100", "A"}},
			dateColIdx:      0,
			valueColIndices: []int{1},
			want:            []int{2},
		},
		{
			name:            "multiple group columns",
			header:          []string{"date", "value", "product", "customer"},
			sampleRows:      [][]string{{"2020-01-01", "100", "A", "Customer1"}},
			dateColIdx:      0,
			valueColIndices: []int{1},
			want:            []int{2, 3},
		},
		{
			name:            "no group columns",
			header:          []string{"date", "value"},
			sampleRows:      [][]string{{"2020-01-01", "100"}},
			dateColIdx:      0,
			valueColIndices: []int{1},
			want:            []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectGroupColumns(tt.header, tt.sampleRows, tt.dateColIdx, tt.valueColIndices)
			if len(got) != len(tt.want) {
				t.Errorf("detectGroupColumns() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("detectGroupColumns() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

func TestBuildCheckInRequest(t *testing.T) {
	// Create a realistic CSV file
	csvContent := `date,product,customer,units
2020-11-01,Liquid Soap,REWE Group,77383
2020-12-01,Liquid Soap,REWE Group,75628
2021-01-01,Liquid Soap,REWE Group,72231
`

	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}

	req, err := BuildCheckInRequest("test-uuid-123", csvFile)
	if err != nil {
		t.Fatalf("BuildCheckInRequest() error = %v", err)
	}

	// Validate request structure
	if req.RawDataSource != "test-uuid-123" {
		t.Errorf("RawDataSource = %q, want %q", req.RawDataSource, "test-uuid-123")
	}

	if req.DataDefinition.DateColumns.Name != "date" {
		t.Errorf("DateColumns.Name = %q, want %q", req.DataDefinition.DateColumns.Name, "date")
	}

	if req.DataDefinition.DateColumns.Format != "%Y-%m-%d" {
		t.Errorf(
			"DateColumns.Format = %q, want %q",
			req.DataDefinition.DateColumns.Format,
			"%Y-%m-%d",
		)
	}

	if len(req.DataDefinition.ValueColumns) != 1 {
		t.Errorf("ValueColumns length = %d, want 1", len(req.DataDefinition.ValueColumns))
	} else {
		if req.DataDefinition.ValueColumns[0].Name != "units" {
			t.Errorf("ValueColumns[0].Name = %q, want %q", req.DataDefinition.ValueColumns[0].Name, "units")
		}
	}

	if len(req.DataDefinition.GroupColumns) != 2 {
		t.Errorf("GroupColumns length = %d, want 2", len(req.DataDefinition.GroupColumns))
	} else {
		if req.DataDefinition.GroupColumns[0].Name != "product" {
			t.Errorf("GroupColumns[0].Name = %q, want %q", req.DataDefinition.GroupColumns[0].Name, "product")
		}
		if req.DataDefinition.GroupColumns[1].Name != "customer" {
			t.Errorf("GroupColumns[1].Name = %q, want %q", req.DataDefinition.GroupColumns[1].Name, "customer")
		}
	}

	if req.FileSpecification.Delimiter != "," {
		t.Errorf("FileSpecification.Delimiter = %q, want %q", req.FileSpecification.Delimiter, ",")
	}
}

func TestBuildCheckInRequestWithOverrides_DateAndFormat(t *testing.T) {
	csvContent := `OrderDate,units
2020/11/01,10
2020/12/01,20
`
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}

	req, err := BuildCheckInRequestWithOverrides("uuid-1", csvFile, &models.CheckInOverrides{DateColumn: "OrderDate", DateFormat: "%Y/%m/%d"})
	if err != nil {
		t.Fatalf("BuildCheckInRequestWithOverrides() error = %v", err)
	}
	if req.DataDefinition.DateColumns.Name != "OrderDate" {
		t.Errorf("DateColumns.Name = %q, want %q", req.DataDefinition.DateColumns.Name, "OrderDate")
	}
	if req.DataDefinition.DateColumns.Format != "%Y/%m/%d" {
		t.Errorf("DateColumns.Format = %q, want %q", req.DataDefinition.DateColumns.Format, "%Y/%m/%d")
	}
}

func TestBuildCheckInRequestWithOverrides_DateFormatOnly(t *testing.T) {
	csvContent := `OrderDate,units
01.11.2020,10
01.12.2020,20
`
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}

	req, err := BuildCheckInRequestWithOverrides("uuid-override", csvFile, &models.CheckInOverrides{
		DateFormat: "%d.%m.%Y",
	})
	if err != nil {
		t.Fatalf("BuildCheckInRequestWithOverrides() error = %v", err)
	}

	if req.DataDefinition.DateColumns.Name != "OrderDate" {
		t.Errorf("DateColumns.Name = %q, want %q", req.DataDefinition.DateColumns.Name, "OrderDate")
	}
	if req.DataDefinition.DateColumns.Format != "%d.%m.%Y" {
		t.Errorf("DateColumns.Format = %q, want %q", req.DataDefinition.DateColumns.Format, "%d.%m.%Y")
	}
}

func TestBuildCheckInRequestWithOverrides_ValueAndGroup(t *testing.T) {
	csvContent := `date,product,customer,units,revenue
2020-11-01,Liquid Soap,REWE Group,10,150.5
2020-12-01,Liquid Soap,REWE Group,20,300.0
`
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}

	req, err := BuildCheckInRequestWithOverrides("uuid-2", csvFile, &models.CheckInOverrides{
		ValueColumns: []string{"revenue"},
		GroupColumns: []string{"product"},
	})
	if err != nil {
		t.Fatalf("BuildCheckInRequestWithOverrides() error = %v", err)
	}
	if len(req.DataDefinition.ValueColumns) != 1 || req.DataDefinition.ValueColumns[0].Name != "revenue" {
		t.Errorf("ValueColumns = %+v, want [revenue]", req.DataDefinition.ValueColumns)
	}
	if len(req.DataDefinition.GroupColumns) != 1 || req.DataDefinition.GroupColumns[0].Name != "product" {
		t.Errorf("GroupColumns = %+v, want [product]", req.DataDefinition.GroupColumns)
	}
}

func TestBuildCheckInRequestWithOverrides_InvalidColumn(t *testing.T) {
	csvContent := `date,units
2020-11-01,10
`
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}
	_, err := BuildCheckInRequestWithOverrides("uuid-3", csvFile, &models.CheckInOverrides{DateColumn: "nonexistent"})
	if err == nil {
		t.Fatalf("expected error for invalid override column")
	}
}

func TestAnalyzeCSV(t *testing.T) {
	csvContent := `date,product,units
2020-11-01,A,10
2020-12-01,A,20
`
	tmpDir := t.TempDir()
	csvFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(csvFile, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create test CSV: %v", err)
	}
	r, err := AnalyzeCSV(csvFile)
	if err != nil {
		t.Fatalf("AnalyzeCSV() error = %v", err)
	}
	if r.DateColumn != "date" || r.DateFormat != "%Y-%m-%d" {
		t.Errorf("unexpected date detection: %v %v", r.DateColumn, r.DateFormat)
	}
	if len(r.ValueCols) != 1 || r.ValueCols[0] != "units" {
		t.Errorf("unexpected value columns: %v", r.ValueCols)
	}
	if len(r.GroupCols) != 1 || r.GroupCols[0] != "product" {
		t.Errorf("unexpected group columns: %v", r.GroupCols)
	}
}
