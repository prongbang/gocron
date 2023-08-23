package scheduler

import (
	"fmt"

	"github.com/prongbang/gocron/pkg/cron"
)

type UseCase interface {
	GetAll() []CreateScheduler
	CreateOnServiceStart() []StatusScheduler
	Create(job string, data CreateScheduler) (string, error)
	Delete(key string) error
}

type useCase struct {
	Repo Repository
	Task Task
}

func (u *useCase) GetAll() []CreateScheduler {
	// Get config list all from database
	list := u.Repo.GetConfigAll()

	// Get config and update status running
	data := []CreateScheduler{}
	for k, v := range cron.Schedulers {
		for _, c := range list {
			if c.Job == k {
				// Update status running
				c.Running = v.IsRunning()

				data = append(data, c)
			}
		}
	}

	return data
}

func (u *useCase) CreateOnServiceStart() []StatusScheduler {
	configList := u.Repo.GetConfigAll()

	data := []StatusScheduler{}
	for _, c := range configList {
		// Check job created and running
		j := cron.Schedulers[c.Job]
		if j == nil {
			// Create job scheduler
			job, _ := u.Create(c.Job, c)

			// Get job by key
			sc := cron.Schedulers[job]

			data = append(data, StatusScheduler{Job: job, Running: sc.IsRunning()})
		}
	}

	return data
}

func (u *useCase) Create(job string, data CreateScheduler) (string, error) {
	cr := cron.New()

	// Set job id
	data.Job = job

	// Cron expressions supported: https://crontab.guru/
	// Every minute */1 * * * *
	_, err := cr.Cron(data.Cron).Do(u.Task.ApiRequest, data)
	if err == nil {
		// Add config to database
		err = u.Repo.Add(data.Job, data)
		if err == nil {
			// Start job
			cr.StartAsync()

			// Add job to memory cache
			cron.Schedulers[data.Job] = cr

			return data.Job, nil
		}
	}

	fmt.Println("[ERROR]", err)

	return "", err
}

func (u *useCase) Delete(key string) error {
	if err := u.Repo.Delete(key); err != nil {
		fmt.Println("[ERROR]", err)
		return err
	}

	// Delete scheduler from map by job key
	delete(cron.Schedulers, key)

	return nil
}

func NewUseCase(repo Repository, task Task) UseCase {
	return &useCase{
		Repo: repo,
		Task: task,
	}
}
