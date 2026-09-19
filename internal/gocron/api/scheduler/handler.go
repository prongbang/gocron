package scheduler

import (
	"fmt"
	"strings"

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
	GetHistory(c *fiber.Ctx) error
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

func (h *handler) GetHistory(c *fiber.Ctx) error {
	q := HistoryQuery{
		Job:     c.Query("job"),
		Project: c.Query("project"),
		Status:  c.Query("status"),
		Q:       strings.TrimSpace(c.Query("q")),
		Page:    c.QueryInt("page", 1),
		Limit:   c.QueryInt("limit", 20),
	}
	switch {
	case q.Page < 1:
		return core.BadRequest(c, "page must be >= 1")
	case q.Limit < 1 || q.Limit > 100:
		return core.BadRequest(c, "limit must be between 1 and 100")
	case q.Status != "" && q.Status != "ok" && q.Status != "failed":
		return core.BadRequest(c, "status must be ok or failed")
	}
	return core.Ok(c, h.Uc.GetHistory(q))
}

func NewHandler(uc UseCase) Handler {
	return &handler{
		Uc: uc,
	}
}
