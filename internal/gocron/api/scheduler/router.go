package scheduler

import "github.com/gofiber/fiber/v2"

type Router interface {
	Initial(app *fiber.App)
}

type router struct {
	Handle Handler
}

func (r *router) Initial(app *fiber.App) {
	// Run on service started
	r.Handle.Initial()

	// Router
	v1 := app.Group("/v1")
	{
		v1.Get("/scheduler", r.Handle.GetList)
		v1.Post("/scheduler", r.Handle.Create)
		v1.Post("/scheduler/stop", r.Handle.StopByJob)
	}
}

func NewRouter(handle Handler) Router {
	return &router{
		Handle: handle,
	}
}
