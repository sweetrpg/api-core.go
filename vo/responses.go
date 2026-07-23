package vo

import "time"

type ErrorVO struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type PingResponseVO struct {
	Date     time.Time `json:"date"`
	Hostname string    `json:"hostname"`
}

type HealthResponseVO struct {
	Database    string   `json:"database"`
	Ping        string   `json:"ping"`
	Collections []string `json:"collections"`
	Messages    []string `json:"messages"`
	Errors      int      `json:"errors"`
}
