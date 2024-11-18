package runner

import (
	"github.com/prongbang/gocron/configuration"
	"github.com/prongbang/gocron/internal/gocron/api"
	"github.com/prongbang/gocron/internal/gocron/builtin"
	"github.com/prongbang/gocron/internal/gocron/database"
	"log"
	"os"
	"strconv"
)

func Start() {
	source := os.Getenv("GOCRON_SOURCE")
	cronBuildIn, _ := strconv.ParseBool(os.Getenv("GOCRON_BUILDIN"))
	cronApi, _ := strconv.ParseBool(os.Getenv("GOCRON_API"))

	if source == configuration.FileSource {
		if err := configuration.Load(); err != nil {
			log.Fatal("[ERROR]", err)
		}
	} else if source == configuration.RemoteSource {
		secure, _ := strconv.ParseBool(os.Getenv("GOCRON_REMOTE_SECURE"))
		if err := configuration.LoadRemote(configuration.RemoteProvider{
			Secure:        secure,
			Provider:      os.Getenv("GOCRON_REMOTE_PROVIDER"),
			Endpoint:      os.Getenv("GOCRON_REMOTE_ENDPOINT"),
			Path:          os.Getenv("GOCRON_REMOTE_PATH"),
			SecretKeyring: os.Getenv("GOCRON_REMOTE_SECRET_KEYRING"),
		}); err != nil {
			log.Fatal("[ERROR]", err)
		}
	}

	cronApiFun := func() {
		dbDriver := database.NewDatabaseDriver()
		apis := api.CreateAPI(dbDriver)
		apis.Register()
	}

	// Start cronApiFun based on the cronApi and cronBuildIn flags
	if cronApi {
		if cronBuildIn {
			go cronApiFun()
		} else {
			cronApiFun()
		}
	}

	// Register builtin if cronBuildIn is enabled
	if cronBuildIn {
		builtin.New().Register()
	}
}
