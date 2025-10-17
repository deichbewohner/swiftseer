package models

import (
	"encoding/json"
	"testing"
)

func TestUploadResponse_GetFileUUID(t *testing.T) {
	tests := []struct {
		name string
		resp UploadResponse
		want string
	}{
		{
			name: "single file",
			resp: UploadResponse{
				UUID: "session-uuid",
				Files: []UploadFile{
					{
						UUID:     "file-uuid-123",
						Filename: "data.csv",
						Size:     1024,
					},
				},
			},
			want: "file-uuid-123",
		},
		{
			name: "multiple files returns first",
			resp: UploadResponse{
				UUID: "session-uuid",
				Files: []UploadFile{
					{
						UUID:     "first-file-uuid",
						Filename: "data1.csv",
						Size:     1024,
					},
					{
						UUID:     "second-file-uuid",
						Filename: "data2.csv",
						Size:     2048,
					},
				},
			},
			want: "first-file-uuid",
		},
		{
			name: "no files",
			resp: UploadResponse{
				UUID:  "session-uuid",
				Files: []UploadFile{},
			},
			want: "",
		},
		{
			name: "nil files",
			resp: UploadResponse{
				UUID:  "session-uuid",
				Files: nil,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resp.GetFileUUID(); got != tt.want {
				t.Errorf("GetFileUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadResponse_JSON(t *testing.T) {
	// Test unmarshaling from API response
	jsonData := `{
		"uuid": "12345678-1234-5678-1234-567812345678",
		"files": [
			{
				"uuid": "87654321-4321-8765-4321-876543218765",
				"filename": "sales-data-41.csv",
				"size": 145678
			}
		]
	}`

	var resp UploadResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.UUID != "12345678-1234-5678-1234-567812345678" {
		t.Errorf("UUID = %s, want 12345678-1234-5678-1234-567812345678", resp.UUID)
	}

	if len(resp.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(resp.Files))
	}

	file := resp.Files[0]
	if file.UUID != "87654321-4321-8765-4321-876543218765" {
		t.Errorf("File UUID = %s, want 87654321-4321-8765-4321-876543218765", file.UUID)
	}

	if file.Filename != "sales-data-41.csv" {
		t.Errorf("Filename = %s, want sales-data-41.csv", file.Filename)
	}

	if file.Size != 145678 {
		t.Errorf("Size = %d, want 145678", file.Size)
	}
}
