package models

type CheckInOverrides struct {
	DateColumn   string
	DateFormat   string
	ValueColumns []string
	GroupColumns []string
}
