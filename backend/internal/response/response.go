package response

import "github.com/gofiber/fiber/v3"

// Format standar: code 0 = sukses, error dimulai dari 1000 (lihat codes.go).
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Meta    any    `json:"meta,omitempty"`
}

type Meta struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

func Success(c fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(Body{Code: CodeOK, Message: message, Data: data})
}

func Paginated(c fiber.Ctx, message string, data any, meta Meta) error {
	return c.Status(fiber.StatusOK).JSON(Body{Code: CodeOK, Message: message, Data: data, Meta: meta})
}

// Error: data berisi detail error (misal error validasi per field), atau nil.
func Error(c fiber.Ctx, status, code int, message string, data any) error {
	return c.Status(status).JSON(Body{Code: code, Message: message, Data: data})
}
