package scheduler

import "time"

const (
	TaskApi    = "api"
	TaskGrpc   = "grpc"
	TaskInflux = "influx"
)

type Scheduler struct {
	Job string `json:"job"`
}

type StatusScheduler struct {
	Job     string `json:"job"`
	Running bool   `json:"running"`
}

type CreateScheduler struct {
	Job     string              `json:"job"`
	Project string              `json:"project"`
	Cron    string              `json:"cron"`
	Task    CreateSchedulerTask `json:"task"`
	Running bool                `json:"running"`
	NextRun *time.Time          `json:"next_run,omitempty"`
}

type CreateSchedulerConfig struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Body   any    `json:"body"`
	Header any    `json:"header"`
}

type CreateSchedulerTask struct {
	Type   string                `json:"type"`
	Config CreateSchedulerConfig `json:"config"`
}
