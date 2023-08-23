package scheduler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/prongbang/gocron/internal/pkg/common"
	"github.com/prongbang/gocron/pkg/core"
	"github.com/prongbang/gocron/pkg/cron"
)

type Handler interface {
	Initial()
	GetList(c *fiber.Ctx) error
	Create(c *fiber.Ctx) error
	StopByJob(c *fiber.Ctx) error
}

type handler struct {
	Uc UseCase
}

func (h *handler) Initial() {
	list := h.Uc.CreateOnServiceStart()
	for _, s := range list {
		fmt.Println("[INFO] Create job:", s)
	}
}

func (h *handler) GetList(c *fiber.Ctx) error {
	data := []fiber.Map{}
	for k, v := range cron.Schedulers {
		data = append(data, fiber.Map{"job": k, "running": v.IsRunning()})
	}

	list := h.Uc.GetAll()

	return core.Ok(c, list)
}

func (h *handler) Create(c *fiber.Ctx) error {
	s := CreateScheduler{}
	if err := c.BodyParser(&s); err != nil {
		fmt.Println("[ERROR]", err)
		return fiber.ErrBadRequest
	}

	// Create job scheduler
	job, err := h.Uc.Create(common.Uuid(), s)
	if err != nil {
		fmt.Println("[ERROR]", err.Error())
		return core.BadRequest(c, err.Error())
	}

	return core.Created(c, Scheduler{Job: job})
}

func (h *handler) StopByJob(c *fiber.Ctx) error {
	s := Scheduler{}
	if err := c.BodyParser(&s); err != nil {
		fmt.Println("[ERROR]", err)
		return core.BadRequest(c, err.Error())
	}

	// Validate
	if len(s.Job) == 0 {
		return core.BadRequest(c, "Required job id")
	}

	// Find scheduler by job
	cr := cron.Schedulers[s.Job]
	if cr == nil {
		return core.NotFound(c, "No jobs found")
	}

	// Delete & stop job by id
	err := h.Uc.Delete(s.Job)
	if err != nil {
		fmt.Println("[ERROR]", err.Error())
		return core.BadRequest(c, err.Error())
	}

	// Stop job
	cr.Stop()

	return core.Ok(c, Scheduler{Job: s.Job})
}

func NewHandler(uc UseCase) Handler {
	return &handler{
		Uc: uc,
	}
}
