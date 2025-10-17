package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

type UploadResult struct {
    UserInputID string
    FileID      string
}

func (c *Client) UploadCSV(ctx context.Context, filePath string) (*UploadResult, error) {
    file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

    if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

    if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

    url := c.buildURL("/userinputs")
    req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

    uploadCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()

    var uploadResp models.UploadResponse
    if err := c.doRequest(uploadCtx, req, &uploadResp); err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}

    userInputID := uploadResp.UUID
    fileID := uploadResp.GetFileUUID()

	if userInputID == "" {
		return nil, fmt.Errorf("no user input ID in response")
	}
	if fileID == "" {
		return nil, fmt.Errorf("no file UUID in response")
	}

	return &UploadResult{
		UserInputID: userInputID,
		FileID:      fileID,
	}, nil
}
