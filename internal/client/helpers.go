package client

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deichbewohner/swiftseer/internal/models"
)

func BuildCheckInRequest(fileUUID, csvPath string) (*models.CheckInRequest, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV: %w", err)
	}
	defer func() { _ = file.Close() }()

	delimiter, err := detectDelimiter(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect delimiter: %w", err)
	}

	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	var sampleRows [][]string
	for i := 0; i < 10; i++ {
		row, err := reader.Read()
		if err != nil {
			break
		}
		sampleRows = append(sampleRows, row)
	}

	if len(sampleRows) == 0 {
		return nil, fmt.Errorf("CSV file is empty (no data rows)")
	}

	dateColIdx, dateFormat, err := detectDateColumn(header, sampleRows)
	if err != nil {
		return nil, fmt.Errorf("failed to detect date column: %w", err)
	}

	valueColIndices := detectValueColumns(header, sampleRows, dateColIdx)
	if len(valueColIndices) == 0 {
		return nil, fmt.Errorf("no numeric value columns found in CSV")
	}

	groupColIndices := detectGroupColumns(header, sampleRows, dateColIdx, valueColIndices)

	dataDef := models.DataDefinition{
		DateColumns: models.DateColumn{
			Name:   header[dateColIdx],
			Format: dateFormat,
		},
		ValueColumns: make([]models.ValueColumn, 0, len(valueColIndices)),
		GroupColumns: make([]models.GroupColumn, 0, len(groupColIndices)),
	}

	for _, idx := range valueColIndices {
		dataDef.ValueColumns = append(dataDef.ValueColumns, models.ValueColumn{
			Name:     header[idx],
			DtypeStr: "Numeric",
		})
	}

	for _, idx := range groupColIndices {
		dataDef.GroupColumns = append(dataDef.GroupColumns, models.GroupColumn{
			Name:     header[idx],
			DtypeStr: "Character",
		})
	}

	valueColNames := make([]string, len(valueColIndices))
	for i, idx := range valueColIndices {
		valueColNames[i] = header[idx]
	}

	groupColNames := make([]string, len(groupColIndices))
	for i, idx := range groupColIndices {
		groupColNames[i] = header[idx]
	}

	tsConfig := models.TsCreationConfig{
		TimeGranularity:     "monthly", // Default assumption
		ValueColumnsToSave:  valueColNames,
		GroupingLevel:       groupColNames,
		MissingValueHandler: "keepNaN",
	}

	fileSpec := models.FileSpecification{
		Delimiter: delimiter,
		Decimal:   ".",
		Encoding:  "utf-8",
	}

	return &models.CheckInRequest{
		RawDataSource:     fileUUID,
		DataDefinition:    dataDef,
		ConfigTsCreation:  tsConfig,
		FileSpecification: fileSpec,
	}, nil
}

func detectDelimiter(csvPath string) (string, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", fmt.Errorf("empty file")
	}

	firstLine := scanner.Text()

	commaCount := strings.Count(firstLine, ",")
	semicolonCount := strings.Count(firstLine, ";")
	tabCount := strings.Count(firstLine, "\t")

	if commaCount >= semicolonCount && commaCount >= tabCount {
		return ",", nil
	}
	if semicolonCount >= tabCount {
		return ";", nil
	}
	return "\t", nil
}

func detectDateColumn(header []string, sampleRows [][]string) (int, string, error) {
	dateKeywords := []string{"date", "time", "timestamp", "datetime", "datum"}

	for i, col := range header {
		colLower := strings.ToLower(col)
		for _, keyword := range dateKeywords {
			if strings.Contains(colLower, keyword) {
				format, err := detectDateFormat(sampleRows, i)
				if err == nil {
					return i, format, nil
				}
			}
		}
	}

	for i := range header {
		format, err := detectDateFormat(sampleRows, i)
		if err == nil {
			return i, format, nil
		}
	}

	return 0, "", fmt.Errorf("no date column found in CSV")
}

func detectDateFormat(sampleRows [][]string, colIdx int) (string, error) {
	if len(sampleRows) == 0 || colIdx >= len(sampleRows[0]) {
		return "", fmt.Errorf("invalid column index")
	}

	formats := []struct {
		goFormat     string
		pythonFormat string
	}{
		{"2006-01-02", "%Y-%m-%d"},
		{"2006/01/02", "%Y/%m/%d"},
		{"02.01.2006", "%d.%m.%Y"},
		{"01/02/2006", "%m/%d/%Y"},
		{"2006-01-02 15:04:05", "%Y-%m-%d %H:%M:%S"},
	}

	sampleValue := strings.TrimSpace(sampleRows[0][colIdx])

	for _, fmt := range formats {
		if _, err := time.Parse(fmt.goFormat, sampleValue); err == nil {
			return fmt.pythonFormat, nil
		}
	}

	return "", fmt.Errorf("could not detect date format for value: %s", sampleValue)
}

func detectValueColumns(header []string, sampleRows [][]string, dateColIdx int) []int {
	var valueColumns []int

	for i := range header {
		if i == dateColIdx {
			continue
		}

		isNumeric := true
		for _, row := range sampleRows {
			if i >= len(row) {
				isNumeric = false
				break
			}
			value := strings.TrimSpace(row[i])
			if value == "" {
				continue
			}
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				isNumeric = false
				break
			}
		}

		if isNumeric {
			valueColumns = append(valueColumns, i)
		}
	}

	return valueColumns
}

func detectGroupColumns(
	header []string,
	sampleRows [][]string,
	dateColIdx int,
	valueColIndices []int,
) []int {
	var groupColumns []int

	// Create set of value column indices for quick lookup
	valueColSet := make(map[int]bool)
	for _, idx := range valueColIndices {
		valueColSet[idx] = true
	}

	for i := range header {
		if i == dateColIdx || valueColSet[i] {
			continue
		}
		groupColumns = append(groupColumns, i)
	}

	return groupColumns
}
