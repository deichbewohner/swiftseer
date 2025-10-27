package models

type Group struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	FileRetentionDays int    `json:"fileRetentionDays"`
}

type GroupsResponse struct {
	UserID string  `json:"userId"`
	Groups []Group `json:"groups"`
}
