package core

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Ok(c *fiber.Ctx, data any) error {
	resp := Response{
		Code:    fmt.Sprintf("%d", http.StatusOK),
		Message: http.StatusText(http.StatusOK),
		Data:    data,
	}
	return c.Status(http.StatusOK).JSON(resp)
}

func Created(c *fiber.Ctx, data any) error {
	resp := Response{
		Code:    fmt.Sprintf("%d", http.StatusCreated),
		Message: http.StatusText(http.StatusCreated),
		Data:    data,
	}
	return c.Status(http.StatusOK).JSON(resp)
}

func BadRequest(c *fiber.Ctx, message any) error {
	resp := Response{
		Code:    fmt.Sprintf("%d", http.StatusBadRequest),
		Message: http.StatusText(http.StatusBadRequest),
	}
	if message != nil {
		resp.Message = message.(string)
	}
	return c.Status(http.StatusBadRequest).JSON(resp)
}

func NotFound(c *fiber.Ctx, message any) error {
	resp := Response{
		Code:    fmt.Sprintf("%d", http.StatusNotFound),
		Message: http.StatusText(http.StatusNotFound),
	}
	if message != nil {
		resp.Message = message.(string)
	}
	return c.Status(http.StatusNotFound).JSON(resp)
}
