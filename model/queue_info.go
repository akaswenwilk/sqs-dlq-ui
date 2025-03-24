package model

type QueueInfo struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	MessageCount int    `json:"messageCount"`
}