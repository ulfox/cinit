package models

import "time"

type Service struct {
	T       string   `json:"type"`
	SUID    string   `json:"suid,omitempty"`
	Name    string   `json:"name"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type ServiceAction struct {
	T          string     `json:"action"`
	SUID       string     `json:"-"`
	Name       string     `json:"name"`
	PID        string     `json:"pid,omitempty"`
	Status     string     `json:"status"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	ExitTime   *time.Time `json:"exitTime,omitempty"`
	Log        []byte     `json:"log,omitempty"`
	Error      error      `json:"error,omitempty"`
	ExitStatus string     `json:"exitStatus,omitempty"`
}
