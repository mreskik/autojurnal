package users

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"

	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(router fiber.Router, db *bun.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	router.Get("/", h.List)
	router.Get("/:id", h.Get)
	router.Post("/", h.Create)
	router.Put("/:id", h.Update)
	router.Delete("/:id", h.Delete)
}

func (h *Handler) List(c fiber.Ctx) error {
	q := ListUserQuery{
		Page:   queryInt(c, "page", 1),
		Limit:  queryInt(c, "limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}

	list, total, err := h.service.List(c.Context(), q)
	if err != nil {
		return handleError(c, err)
	}
	return response.Paginated(c, "success", list, response.Meta{
		Page:      q.Page,
		Limit:     q.Limit,
		Total:     total,
		TotalPage: int(math.Ceil(float64(total) / float64(q.Limit))),
	})
}

func (h *Handler) Get(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	user, err := h.service.Get(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", user)
}

func (h *Handler) Create(c fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	user, err := h.service.Create(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "user created", user)
}

func (h *Handler) Update(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	user, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "user updated", user)
}

func (h *Handler) Delete(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.Delete(c.Context(), id); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "user deleted", nil)
}

var errInvalidID = errors.New("invalid id")

func paramID(c fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errInvalidID
	}
	return id, nil
}

func queryInt(c fiber.Ctx, key string, fallback int) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return v
}

func handleError(c fiber.Ctx, err error) error {
	var verr ValidationErrors
	switch {
	case errors.As(err, &verr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeValidationFailed, "validation failed", verr)
	case errors.Is(err, errInvalidID):
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidParam, err.Error(), nil)
	case errors.Is(err, ErrUserNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeUserNotFound, err.Error(), nil)
	case errors.Is(err, ErrUsernameTaken):
		return response.Error(c, fiber.StatusConflict, response.CodeUsernameTaken, err.Error(), ValidationErrors{"username": err.Error()})
	case errors.Is(err, ErrEmailTaken):
		return response.Error(c, fiber.StatusConflict, response.CodeEmailTaken, err.Error(), ValidationErrors{"email": err.Error()})
	default:
		log.Printf("users: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, response.CodeInternalError, "internal server error", nil)
	}
}
