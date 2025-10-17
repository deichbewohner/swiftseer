package models

type CheckInRequest struct {
	RawDataSource     string            `json:"raw_data_source"`
	DataDefinition    DataDefinition    `json:"data_definition"`
	ConfigTsCreation  TsCreationConfig  `json:"config_ts_creation"`
	FileSpecification FileSpecification `json:"file_specification"`
}

type DataDefinition struct {
	DateColumns  DateColumn    `json:"date_columns"`
	ValueColumns []ValueColumn `json:"value_columns"`
	GroupColumns []GroupColumn `json:"group_columns"`
}

type DateColumn struct {
    Name   string `json:"name"`
    Format string `json:"format"`
}

type ValueColumn struct {
    Name     string   `json:"name"`
    Min      *float64 `json:"min,omitempty"`
    Max      *float64 `json:"max,omitempty"`
    DtypeStr string   `json:"dtype_str"`
}

type GroupColumn struct {
    Name     string `json:"name"`
    DtypeStr string `json:"dtype_str"`
}

type TsCreationConfig struct {
    TimeGranularity     string   `json:"time_granularity"`
    ValueColumnsToSave  []string `json:"value_columns_to_save"`
    GroupingLevel       []string `json:"grouping_level"`
    MissingValueHandler string   `json:"missing_value_handler"`
    Description         string   `json:"description,omitempty"`
}

type FileSpecification struct {
    Delimiter string `json:"delimiter"`
    Decimal   string `json:"decimal"`
    Encoding  string `json:"encoding"`
}

type CheckInResponse struct {
	TimeSeries []TimeSeries `json:"time_series"`
	VersionID  string       `json:"version_id"`
}

type TimeSeries struct {
	Name        string            `json:"name"`
	Group       string            `json:"group"`
	Granularity string            `json:"granularity"`
	Grouping    map[string]string `json:"grouping"`
	Values      []TimeSeriesValue `json:"values"`
}

type TimeSeriesValue struct {
	TimeStampUTC string  `json:"time_stamp_utc"`
	Value        float64 `json:"value"`
}
