package api

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/prongbang/gocron/internal/gocron/api/auth"
	"github.com/prongbang/gocron/internal/gocron/api/scheduler"
	"github.com/prongbang/gocron/internal/gocron/database"
)

func CreateAPI(dbDriver database.Drivers) API {
	schedulerRepo := scheduler.NewRepository(dbDriver)
	schedulerTask := scheduler.NewTask(schedulerRepo)
	schedulerUseCase := scheduler.NewUseCase(schedulerRepo, schedulerTask)
	schedulerHandler := scheduler.NewHandler(schedulerUseCase)
	schedulerRouter := scheduler.NewRouter(schedulerHandler)
	apiRouters := NewRouters(schedulerRouter)
	return NewAPI(apiRouters, createAuth(dbDriver))
}

// createAuth returns nil when GOCRON_AUTH is off, which keeps the API open as before.
func createAuth(dbDriver database.Drivers) *auth.Auth {
	if on, _ := strconv.ParseBool(os.Getenv("GOCRON_AUTH")); !on {
		fmt.Println("[WARN] GOCRON_AUTH is off: anyone who can reach port 8000 can manage jobs")
		return nil
	}
	a, err := auth.Setup(dbDriver.BadgerDB(), os.Getenv("GOCRON_ADMIN_USER"), os.Getenv("GOCRON_ADMIN_PASSWORD"))
	if err != nil {
		log.Fatal("[ERROR] ", err)
	}
	return a
}
