package cron

import (
	"os"
	"time"

	"github.com/go-co-op/gocron"
)

var Schedulers = map[string]*gocron.Scheduler{}

func New() *gocron.Scheduler {
	tz, _ := time.LoadLocation(os.Getenv("TZ"))
	return gocron.NewScheduler(tz)
}
