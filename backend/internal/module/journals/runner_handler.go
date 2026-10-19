package journals

import (
	"math"
	"strings"

	"autojurnal/internal/module/runners"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	// HeaderJournalToken: header berisi journal token, wajib di semua endpoint jurnal untuk runner.
	HeaderJournalToken = "X-Journal-Token"
	localJournalID     = "runner_journal_id"
)

// RegisterRunnerRoutes: endpoint jurnal & trade untuk agent (group /api/runner, sudah lewat RequireRunner).
// Jurnal ditentukan dari header X-Journal-Token.
func (h *Handler) RegisterRunnerRoutes(router fiber.Router) {
	router.Get("/journal", h.RequireJournal, h.RunnerGetJournal)
	router.Get("/trades", h.RequireJournal, h.RunnerListTrades)
	router.Post("/trades", h.RequireJournal, h.RunnerCreateTrade)
	router.Get("/trades/:tradeId", h.RequireJournal, h.RunnerGetTrade)
	router.Put("/trades/:tradeId", h.RequireJournal, h.RunnerUpdateTrade)
	router.Post("/trades/:tradeId/close", h.RequireJournal, h.RunnerCloseTrade)
}

// RequireJournal: validasi header X-Journal-Token untuk runner yang sedang akses.
// Syarat: token jurnal valid, jurnal milik pemilik runner, dan runner sudah join ke jurnal.
func (h *Handler) RequireJournal(c fiber.Ctx) error {
	j, err := h.service.RunnerJournal(c.Context(), runnerScope(c), strings.TrimSpace(c.Get(HeaderJournalToken)))
	if err != nil {
		return handleError(c, err)
	}
	c.Locals(localJournalID, j.ID)
	return c.Next()
}

func runnerScope(c fiber.Ctx) Scope {
	return RunnerScope(runners.CurrentRunner(c))
}

func runnerJournalID(c fiber.Ctx) uuid.UUID {
	id, _ := c.Locals(localJournalID).(uuid.UUID)
	return id
}

func (h *Handler) RunnerGetJournal(c fiber.Ctx) error {
	j, err := h.service.RunnerJournalDetail(c.Context(), runnerScope(c), runnerJournalID(c))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", j)
}

func (h *Handler) RunnerListTrades(c fiber.Ctx) error {
	q := ListTradeQuery{
		Page:     queryInt(c, "page", 1),
		Limit:    queryInt(c, "limit", 10),
		Status:   strings.TrimSpace(c.Query("status")),
		Position: strings.TrimSpace(c.Query("position")),
		Pair:     strings.TrimSpace(c.Query("pair")),
		Source:   strings.TrimSpace(c.Query("source")),
		From:     strings.TrimSpace(c.Query("from")),
		To:       strings.TrimSpace(c.Query("to")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}
	if errs := normalizeTradeFilter(&q); errs != nil {
		return handleError(c, errs)
	}

	list, total, err := h.service.ListTrades(c.Context(), runnerScope(c), runnerJournalID(c), q)
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

func (h *Handler) RunnerCreateTrade(c fiber.Ctx) error {
	var req CreateTradeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CreateTrade(c.Context(), runnerScope(c), runnerJournalID(c), req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "trade created", t)
}

func (h *Handler) RunnerGetTrade(c fiber.Ctx) error {
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	t, err := h.service.GetTrade(c.Context(), runnerScope(c), runnerJournalID(c), tradeID)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", t)
}

func (h *Handler) RunnerUpdateTrade(c fiber.Ctx) error {
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateTradeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.UpdateTrade(c.Context(), runnerScope(c), runnerJournalID(c), tradeID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "trade updated", t)
}

func (h *Handler) RunnerCloseTrade(c fiber.Ctx) error {
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	var req CloseFields
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CloseTrade(c.Context(), runnerScope(c), runnerJournalID(c), tradeID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "trade closed", t)
}
