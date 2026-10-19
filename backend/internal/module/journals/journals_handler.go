package journals

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/module/master/users"
	"autojurnal/internal/module/runners"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Handler struct {
	service *Service
	// URL publik API untuk AGENT.md (contoh https://api.domain.com/api); kosong = dari request
	publicAPIURL string
}

func NewHandler(db *bun.DB, publicAPIURL string) *Handler {
	return &Handler{
		service:      NewService(NewRepository(db), runners.NewRepository(db)),
		publicAPIURL: publicAPIURL,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/", h.List)
	router.Get("/:id", h.Get)
	router.Post("/", h.Create)
	router.Put("/:id", h.Update)
	router.Delete("/:id", h.Delete)
	router.Post("/:id/regenerate-token", h.RegenerateToken)

	router.Get("/:id/targets", h.ListTargets)
	router.Post("/:id/targets", h.CreateTarget)

	router.Get("/:id/transactions", h.ListTransactions)
	router.Post("/:id/transactions", h.CreateTransaction)
	router.Put("/:id/transactions/:txId", h.UpdateTransaction)
	router.Delete("/:id/transactions/:txId", h.DeleteTransaction)

	router.Get("/:id/trades", h.ListTrades)
	router.Post("/:id/trades", h.CreateTrade)
	router.Get("/:id/trades/:tradeId", h.GetTrade)
	router.Put("/:id/trades/:tradeId", h.UpdateTrade)
	router.Post("/:id/trades/:tradeId/close", h.CloseTrade)
	router.Delete("/:id/trades/:tradeId", h.DeleteTrade)

	router.Get("/:id/runners", h.ListRunners)
	router.Get("/:id/runners/options", h.ListRunnerOptions)
	router.Put("/:id/runners/:runnerId", h.UpdateRunner)
	router.Get("/:id/runners/:runnerId/agent.md", h.DownloadAgentFile)

	h.registerJournalDashboardRoutes(router)
	router.Post("/:id/runners", h.AddRunner)
	router.Delete("/:id/runners/:runnerId", h.RemoveRunner)
}

func (h *Handler) List(c fiber.Ctx) error {
	q := ListJournalQuery{
		Page:   queryInt(c, "page", 1),
		Limit:  queryInt(c, "limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Status: strings.TrimSpace(c.Query("status")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}
	if q.Status != "" && q.Status != StatusActive && q.Status != StatusArchived {
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeValidationFailed, "validation failed",
			users.ValidationErrors{"status": "must be active or archived"})
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
	j, err := h.service.Get(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", j)
}

func (h *Handler) Create(c fiber.Ctx) error {
	var req CreateJournalRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	j, err := h.service.Create(c.Context(), auth.CurrentUser(c).ID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "journal created", j)
}

func (h *Handler) Update(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateJournalRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	j, err := h.service.Update(c.Context(), auth.CurrentUser(c).ID, id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "journal updated", j)
}

func (h *Handler) Delete(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.Delete(c.Context(), auth.CurrentUser(c).ID, id); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "journal deleted", nil)
}

func (h *Handler) RegenerateToken(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	j, err := h.service.RegenerateToken(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "token regenerated", j)
}

func (h *Handler) ListTargets(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	list, err := h.service.ListTargets(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", list)
}

func (h *Handler) CreateTarget(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	var req CreateTargetRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CreateTarget(c.Context(), auth.CurrentUser(c).ID, id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "target created", t)
}

var errInvalidID = errors.New("invalid id")

func paramID(c fiber.Ctx) (uuid.UUID, error) {
	return paramUUID(c, "id")
}

func paramUUID(c fiber.Ctx, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(name))
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
	var (
		verr      users.ValidationErrors
		lockErr   *JournalLockedError
		balErr    *InsufficientBalanceError
		closedErr *TradeClosedError
		openErr   *OpenTradeExistsError
	)
	switch {
	case errors.As(err, &verr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeValidationFailed, "validation failed", verr)
	case errors.Is(err, errInvalidID):
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidParam, err.Error(), nil)
	case errors.Is(err, ErrJournalNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeJournalNotFound, err.Error(), nil)
	case errors.As(err, &lockErr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeJournalLocked, err.Error(), lockErr.Fields)
	case errors.Is(err, ErrTransactionNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeTransactionNotFound, err.Error(), nil)
	case errors.As(err, &balErr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeInsufficientBalance, err.Error(),
			fiber.Map{"balance_after": balErr.BalanceAfter})
	case errors.Is(err, ErrRunnerAlreadyJoined):
		return response.Error(c, fiber.StatusConflict, response.CodeRunnerAlreadyJoined, err.Error(), nil)
	case errors.Is(err, ErrRunnerNotInJournal):
		return response.Error(c, fiber.StatusNotFound, response.CodeRunnerNotInJournal, err.Error(), nil)
	case errors.Is(err, ErrInvalidJournalToken):
		return response.Error(c, fiber.StatusUnauthorized, response.CodeInvalidJournalToken, err.Error(), nil)
	case errors.Is(err, ErrRunnerNotJoined):
		return response.Error(c, fiber.StatusForbidden, response.CodeRunnerNotJoined, err.Error(), nil)
	case errors.Is(err, ErrJournalArchived):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeJournalArchived, err.Error(), nil)
	case errors.As(err, &openErr):
		var existing any
		if openErr.Existing != nil {
			existing = ToTradeResponse(openErr.Existing)
		}
		return response.Error(c, fiber.StatusConflict, response.CodeOpenTradeExists, err.Error(), existing)
	case errors.Is(err, ErrTradeNotOwned):
		return response.Error(c, fiber.StatusForbidden, response.CodeTradeNotOwned, err.Error(), nil)
	case errors.Is(err, ErrTradeNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeTradeNotFound, err.Error(), nil)
	case errors.As(err, &closedErr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeTradeAlreadyClosed, err.Error(), closedErr.Fields)
	case errors.Is(err, ErrTradeAlreadyClosed):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeTradeAlreadyClosed, err.Error(), nil)
	case errors.Is(err, runners.ErrRunnerNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeRunnerNotFound, err.Error(), nil)
	default:
		log.Printf("journals: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, response.CodeInternalError, "internal server error", nil)
	}
}
