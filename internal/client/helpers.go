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

var (
	dateKeywords = []string{"date", "time", "timestamp", "datetime", "datum"}

	supportedDateFormats = []struct {
		goLayout     string
		pythonFormat string
	}{
		{"2006-01-02", "%Y-%m-%d"},
		{"2006/01/02", "%Y/%m/%d"},
		{"02.01.2006", "%d.%m.%Y"},
		{"01/02/2006", "%m/%d/%Y"},
		{"2006-01-02 15:04:05", "%Y-%m-%d %H:%M:%S"},
	}
)

func pythonToGoLayout(format string) (string, error) {
	for _, candidate := range supportedDateFormats {
		if candidate.pythonFormat == format {
			return candidate.goLayout, nil
		}
	}

	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			b.WriteByte(format[i])
			continue
		}

		i++
		if i >= len(format) {
			return "", fmt.Errorf("incomplete format specifier")
		}

		switch format[i] {
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'd':
			b.WriteString("02")
		case 'H':
			b.WriteString("15")
		case 'I':
			b.WriteString("03")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'f':
			b.WriteString("000000")
		case 'p':
			b.WriteString("PM")
		case 'z':
			b.WriteString("-0700")
		case 'Z':
			b.WriteString("MST")
		case '%':
			b.WriteString("%")
		default:
			return "", fmt.Errorf("unsupported format verb: %%%c", format[i])
		}
	}

	return b.String(), nil
}

func BuildCheckInRequest(fileUUID, csvPath string) (*models.CheckInRequest, error) {
	return BuildCheckInRequestWithOverrides(fileUUID, csvPath, nil)
}

// BuildCheckInRequestWithOverrides builds the check-in request using optional overrides.
func BuildCheckInRequestWithOverrides(fileUUID, csvPath string, overrides *models.CheckInOverrides) (*models.CheckInRequest, error) {
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

	// Resolve date column and format (apply overrides if present)
	var dateColIdx int
	var dateFormat string
	if overrides != nil && overrides.DateColumn != "" {
		idx := findColumnIndex(header, overrides.DateColumn)
		if idx < 0 {
			return nil, fmt.Errorf("override date column not found: %s", overrides.DateColumn)
		}
		dateColIdx = idx
		if overrides.DateFormat != "" {
			dateFormat = overrides.DateFormat
		} else {
			fmtStr, err := detectDateFormat(sampleRows, dateColIdx)
			if err != nil {
				return nil, fmt.Errorf("failed to detect date format for override column: %w", err)
			}
			dateFormat = fmtStr
		}
	} else {
		var err error
		overrideFormat := ""
		if overrides != nil {
			overrideFormat = overrides.DateFormat
		}
		dateColIdx, dateFormat, err = detectDateColumn(header, sampleRows, overrideFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to detect date column: %w", err)
		}
	}

	// Resolve value columns
	var valueColIndices []int
	if overrides != nil && len(overrides.ValueColumns) > 0 {
		for _, name := range overrides.ValueColumns {
			idx := findColumnIndex(header, name)
			if idx < 0 {
				return nil, fmt.Errorf("override value column not found: %s", name)
			}
			valueColIndices = append(valueColIndices, idx)
		}
	} else {
		valueColIndices = detectValueColumns(header, sampleRows, dateColIdx)
		if len(valueColIndices) == 0 {
			return nil, fmt.Errorf("no numeric value columns found in CSV")
		}
	}

	// Resolve group columns
	var groupColIndices []int
	if overrides != nil && len(overrides.GroupColumns) > 0 {
		for _, name := range overrides.GroupColumns {
			idx := findColumnIndex(header, name)
			if idx < 0 {
				return nil, fmt.Errorf("override group column not found: %s", name)
			}
			// Exclude if it's date or a value column
			if idx == dateColIdx || containsIndex(valueColIndices, idx) {
				continue
			}
			groupColIndices = append(groupColIndices, idx)
		}
	} else {
		groupColIndices = detectGroupColumns(header, sampleRows, dateColIdx, valueColIndices)
	}

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

// Analysis describes detected CSV structure.
type Analysis struct {
	Delimiter  string
	Header     []string
	DateColumn string
	DateFormat string
	ValueCols  []string
	GroupCols  []string
}

// AnalyzeCSV reports detected columns and delimiter without building a request.
func AnalyzeCSV(csvPath string) (*Analysis, error) {
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

	dateColIdx, dateFormat, err := detectDateColumn(header, sampleRows, "")
	if err != nil {
		return nil, fmt.Errorf("failed to detect date column: %w", err)
	}
	valueColIndices := detectValueColumns(header, sampleRows, dateColIdx)
	if len(valueColIndices) == 0 {
		return nil, fmt.Errorf("no numeric value columns found in CSV")
	}
	groupColIndices := detectGroupColumns(header, sampleRows, dateColIdx, valueColIndices)

	var valueNames []string
	for _, i := range valueColIndices {
		valueNames = append(valueNames, header[i])
	}
	var groupNames []string
	for _, i := range groupColIndices {
		groupNames = append(groupNames, header[i])
	}

	return &Analysis{
		Delimiter:  delimiter,
		Header:     header,
		DateColumn: header[dateColIdx],
		DateFormat: dateFormat,
		ValueCols:  valueNames,
		GroupCols:  groupNames,
	}, nil
}

func findColumnIndex(header []string, name string) int {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for i, h := range header {
		if strings.ToLower(strings.TrimSpace(h)) == nameLower {
			return i
		}
	}
	return -1
}

func containsIndex(list []int, idx int) bool {
	for _, i := range list {
		if i == idx {
			return true
		}
	}
	return false
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

func detectDateColumn(header []string, sampleRows [][]string, preferredFormat string) (int, string, error) {
	if preferredFormat != "" {
		layout, err := pythonToGoLayout(preferredFormat)
		if err != nil {
			return 0, "", fmt.Errorf("unsupported date format override: %w", err)
		}

		if idx, ok := findMatchingDateColumn(header, sampleRows, layout, true); ok {
			return idx, preferredFormat, nil
		}
		if idx, ok := findMatchingDateColumn(header, sampleRows, layout, false); ok {
			return idx, preferredFormat, nil
		}

		return 0, "", fmt.Errorf("no column matches provided date format override %q", preferredFormat)
	}

	for i, col := range header {
		if !columnNameLooksLikeDate(col) {
			continue
		}
		format, err := detectDateFormat(sampleRows, i)
		if err == nil {
			return i, format, nil
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

	sampleValue := strings.TrimSpace(sampleRows[0][colIdx])

	for _, format := range supportedDateFormats {
		if _, err := time.Parse(format.goLayout, sampleValue); err == nil {
			return format.pythonFormat, nil
		}
	}

	return "", fmt.Errorf("could not detect date format for value: %s", sampleValue)
}

func findMatchingDateColumn(
	header []string,
	sampleRows [][]string,
	layout string,
	requireKeyword bool,
) (int, bool) {
	for i, col := range header {
		if requireKeyword && !columnNameLooksLikeDate(col) {
			continue
		}
		if columnMatchesFormat(sampleRows, i, layout) {
			return i, true
		}
	}
	return 0, false
}

func columnMatchesFormat(sampleRows [][]string, colIdx int, layout string) bool {
	matched := false
	for _, row := range sampleRows {
		if colIdx >= len(row) {
			return false
		}
		value := strings.TrimSpace(row[colIdx])
		if value == "" {
			continue
		}
		if _, err := time.Parse(layout, value); err != nil {
			return false
		}
		matched = true
	}
	return matched
}

func columnNameLooksLikeDate(name string) bool {
	colLower := strings.ToLower(name)
	for _, keyword := range dateKeywords {
		if strings.Contains(colLower, keyword) {
			return true
		}
	}
	return false
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
