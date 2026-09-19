package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prongbang/gocron/web"
)

type API interface {
	Register()
}

type api struct {
	Router Routers
}

func (a *api) Register() {
	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "X-Platform, X-Api-Key, Authorization, Access-Control-Allow-Credentials, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS",
	}))

	// Routers
	a.Router.Initials(app)

	// Web dashboard (SPA): unknown paths fall back to index.html, /v1 stays API-only.
	if ui := web.FS(); ui != nil {
		app.Use(filesystem.New(filesystem.Config{
			Root:         http.FS(ui),
			NotFoundFile: "index.html",
			Next:         func(c *fiber.Ctx) bool { return strings.HasPrefix(c.Path(), "/v1") },
		}))
	} else {
		fmt.Println("[INFO] Web UI not embedded: run `bun run build` in web/ before `go build`")
	}

	// Serve
	_ = app.Listen(":8000")
}

func NewAPI(router Routers) API {
	return &api{
		Router: router,
	}
}
