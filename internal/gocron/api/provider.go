package api

import (
	"github.com/prongbang/gocron/internal/gocron/api/scheduler"
	"github.com/prongbang/gocron/internal/gocron/database"
)

func CreateAPI(dbDriver database.Drivers) API {
	schedulerTask := scheduler.NewTask()
	schedulerRepo := scheduler.NewRepository(dbDriver)
	schedulerUseCase := scheduler.NewUseCase(schedulerRepo, schedulerTask)
	schedulerHandler := scheduler.NewHandler(schedulerUseCase)
	schedulerRouter := scheduler.NewRouter(schedulerHandler)
	apiRouters := NewRouters(schedulerRouter)
	apiAPI := NewAPI(apiRouters)
	return apiAPI
}
