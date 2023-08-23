package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
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

	// Serve
	_ = app.Listen(":8000")
}

func NewAPI(router Routers) API {
	return &api{
		Router: router,
	}
}
