package runners

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/module/master/users"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const localRunner = "runner"

type Handler struct {
	service *Service
}

func NewHandler(db *bun.DB) *Handler {
	return &Handler{service: NewService(NewRepository(db), users.NewRepository(db))}
}

// RegisterRoutes: manajemen runner oleh user yang login (/api/runners).
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/", h.List)
	router.Get("/:id", h.Get)
	router.Post("/", h.Create)
	router.Put("/:id", h.Update)
	router.Post("/:id/regenerate-token", h.RegenerateToken)
	router.Delete("/:id", h.Delete)
}

// RegisterRunnerRoutes: endpoint yang dipanggil runner pakai token (/api/runner).
func (h *Handler) RegisterRunnerRoutes(router fiber.Router) {
	router.Get("/me", h.Me)
}

// RequireRunner: middleware untuk endpoint runner.
// Header: Authorization: Bearer <token runner>
func (h *Handler) RequireRunner(c fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok {
		return handleError(c, ErrInvalidToken)
	}
	runner, err := h.service.Authenticate(c.Context(), strings.TrimSpace(token))
	if err != nil {
		return handleError(c, err)
	}
	c.Locals(localRunner, runner)
	return c.Next()
}

// CurrentRunner mengambil runner dari token (hanya di route dengan RequireRunner).
func CurrentRunner(c fiber.Ctx) *Runner {
	r, _ := c.Locals(localRunner).(*Runner)
	return r
}

func (h *Handler) List(c fiber.Ctx) error {
	q := ListRunnerQuery{
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

	list, total, err := h.service.List(c.Context(), auth.CurrentUser(c).ID, q)
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
	runner, err := h.service.Get(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", runner)
}

func (h *Handler) Create(c fiber.Ctx) error {
	var req CreateRunnerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	runner, err := h.service.Create(c.Context(), auth.CurrentUser(c).ID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "runner created", runner)
}

func (h *Handler) Update(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateRunnerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	runner, err := h.service.Update(c.Context(), auth.CurrentUser(c).ID, id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "runner updated", runner)
}

func (h *Handler) RegenerateToken(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	runner, err := h.service.RegenerateToken(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "token regenerated", runner)
}

func (h *Handler) Delete(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.Delete(c.Context(), auth.CurrentUser(c).ID, id); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "runner deleted", nil)
}

func (h *Handler) Me(c fiber.Ctx) error {
	return response.Success(c, fiber.StatusOK, "success", ToRunnerSelfResponse(CurrentRunner(c)))
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
	var verr users.ValidationErrors
	switch {
	case errors.As(err, &verr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeValidationFailed, "validation failed", verr)
	case errors.Is(err, errInvalidID):
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidParam, err.Error(), nil)
	case errors.Is(err, ErrRunnerNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeRunnerNotFound, err.Error(), nil)
	case errors.Is(err, ErrInvalidToken):
		return response.Error(c, fiber.StatusUnauthorized, response.CodeInvalidRunnerToken, err.Error(), nil)
	case errors.Is(err, ErrTokenExpired):
		return response.Error(c, fiber.StatusUnauthorized, response.CodeRunnerTokenExpired, err.Error(), nil)
	case errors.Is(err, ErrRunnerInactive):
		return response.Error(c, fiber.StatusForbidden, response.CodeRunnerInactive, err.Error(), nil)
	default:
		log.Printf("runners: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, response.CodeInternalError, "internal server error", nil)
	}
}
