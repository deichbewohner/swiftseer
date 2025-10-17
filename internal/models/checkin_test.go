package models

import (
	"encoding/json"
	"testing"
)

func TestCheckInRequest_JSON(t *testing.T) {
	minVal := 0.0
	req := CheckInRequest{
		RawDataSource: "file-uuid-123",
		DataDefinition: DataDefinition{
			DateColumns: DateColumn{
				Name:   "date",
				Format: "%Y-%m-%d",
			},
			ValueColumns: []ValueColumn{
				{
					Name:     "units",
					Min:      &minVal,
					DtypeStr: "Numeric",
				},
			},
			GroupColumns: []GroupColumn{
				{Name: "product", DtypeStr: "Character"},
				{Name: "customer", DtypeStr: "Character"},
			},
		},
		ConfigTsCreation: TsCreationConfig{
			TimeGranularity:     "monthly",
			ValueColumnsToSave:  []string{"units"},
			GroupingLevel:       []string{"product", "customer"},
			MissingValueHandler: "keepNaN",
			Description:         "Test data",
		},
		FileSpecification: FileSpecification{
			Delimiter: ",",
			Decimal:   ".",
			Encoding:  "utf-8",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded CheckInRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key fields
	if decoded.RawDataSource != req.RawDataSource {
		t.Errorf("RawDataSource = %s, want %s", decoded.RawDataSource, req.RawDataSource)
	}

	if decoded.DataDefinition.DateColumns.Name != "date" {
		t.Errorf("DateColumn.Name = %s, want date", decoded.DataDefinition.DateColumns.Name)
	}

	if len(decoded.DataDefinition.ValueColumns) != 1 {
		t.Fatalf("len(ValueColumns) = %d, want 1", len(decoded.DataDefinition.ValueColumns))
	}

	vc := decoded.DataDefinition.ValueColumns[0]
	if vc.Name != "units" {
		t.Errorf("ValueColumn.Name = %s, want units", vc.Name)
	}

	if vc.Min == nil {
		t.Error("ValueColumn.Min is nil, want non-nil")
	} else if *vc.Min != 0.0 {
		t.Errorf("ValueColumn.Min = %f, want 0.0", *vc.Min)
	}

	if len(decoded.DataDefinition.GroupColumns) != 2 {
		t.Errorf("len(GroupColumns) = %d, want 2", len(decoded.DataDefinition.GroupColumns))
	}
}

func TestValueColumn_OmitEmpty(t *testing.T) {
	tests := []struct {
		name    string
		vc      ValueColumn
		wantMin bool
		wantMax bool
	}{
		{
			name: "no bounds",
			vc: ValueColumn{
				Name:     "value",
				DtypeStr: "Numeric",
			},
			wantMin: false,
			wantMax: false,
		},
		{
			name: "with min only",
			vc: ValueColumn{
				Name:     "value",
				Min:      ptrFloat64(0),
				DtypeStr: "Numeric",
			},
			wantMin: true,
			wantMax: false,
		},
		{
			name: "with max only",
			vc: ValueColumn{
				Name:     "value",
				Max:      ptrFloat64(100),
				DtypeStr: "Numeric",
			},
			wantMin: false,
			wantMax: true,
		},
		{
			name: "with both bounds",
			vc: ValueColumn{
				Name:     "value",
				Min:      ptrFloat64(0),
				Max:      ptrFloat64(100),
				DtypeStr: "Numeric",
			},
			wantMin: true,
			wantMax: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.vc)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Check if "min" is in JSON
			hasMin := containsKey(string(data), "min")
			if hasMin != tt.wantMin {
				t.Errorf("JSON contains 'min' = %v, want %v", hasMin, tt.wantMin)
			}

			// Check if "max" is in JSON
			hasMax := containsKey(string(data), "max")
			if hasMax != tt.wantMax {
				t.Errorf("JSON contains 'max' = %v, want %v", hasMax, tt.wantMax)
			}
		})
	}
}

func TestCheckInResponse_JSON(t *testing.T) {
	jsonData := `{
		"time_series": [
			{
				"name": "units-Product A-Customer X",
				"group": "units",
				"granularity": "monthly",
				"grouping": {
					"product": "Product A",
					"customer": "Customer X"
				},
				"values": [
					{
						"time_stamp_utc": "2020-11-01T00:00:00",
						"value": 145820
					}
				]
			}
		],
		"version_id": "68ea2cd6cc05e6ab4b85ebd1"
	}`

	var resp CheckInResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.VersionID != "68ea2cd6cc05e6ab4b85ebd1" {
		t.Errorf("VersionID = %s, want 68ea2cd6cc05e6ab4b85ebd1", resp.VersionID)
	}

	if len(resp.TimeSeries) != 1 {
		t.Fatalf("len(TimeSeries) = %d, want 1", len(resp.TimeSeries))
	}

	ts := resp.TimeSeries[0]
	if ts.Name != "units-Product A-Customer X" {
		t.Errorf("TimeSeries.Name = %s, want units-Product A-Customer X", ts.Name)
	}

	if ts.Grouping["product"] != "Product A" {
		t.Errorf("Grouping[product] = %s, want Product A", ts.Grouping["product"])
	}

	if len(ts.Values) != 1 {
		t.Fatalf("len(Values) = %d, want 1", len(ts.Values))
	}

	if ts.Values[0].Value != 145820 {
		t.Errorf("Value = %f, want 145820", ts.Values[0].Value)
	}
}

// Helper functions
func ptrFloat64(f float64) *float64 {
	return &f
}

func containsKey(s, key string) bool {
	searchFor := `"` + key + `"`
	for i := 0; i < len(s)-len(searchFor); i++ {
		if s[i:i+len(searchFor)] == searchFor {
			return true
		}
	}
	return false
}
