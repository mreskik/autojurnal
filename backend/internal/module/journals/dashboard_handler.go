package journals

import (
	"math"
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/module/master/users"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
)

// RegisterDashboardRoutes: dashboard lintas jurnal (/api/dashboard).
func (h *Handler) RegisterDashboardRoutes(router fiber.Router) {
	router.Get("/overview", h.DashboardOverview)
	router.Get("/runners", h.DashboardRunners)
}

// registerJournalDashboardRoutes: dashboard per jurnal (/api/journals/:id/dashboard/*).
func (h *Handler) registerJournalDashboardRoutes(router fiber.Router) {
	router.Get("/:id/dashboard/summary", h.DashboardSummary)
	router.Get("/:id/dashboard/periods", h.DashboardPeriods)
	router.Get("/:id/dashboard/equity", h.DashboardEquity)
	router.Get("/:id/dashboard/calendar", h.DashboardCalendar)
	router.Get("/:id/dashboard/breakdown", h.DashboardBreakdown)
}

func (h *Handler) DashboardSummary(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	res, err := h.service.DashboardSummary(c.Context(), auth.CurrentUser(c).ID, id,
		strings.TrimSpace(c.Query("from")), strings.TrimSpace(c.Query("to")))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}

func (h *Handler) DashboardPeriods(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	page, limit := queryInt(c, "page", 1), queryInt(c, "limit", 30)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	status := strings.TrimSpace(c.Query("status"))
	switch status {
	case "", PeriodAchieved, PeriodMissed, PeriodNoTrade, PeriodInProgress:
	default:
		return handleError(c, users.ValidationErrors{"status": "must be achieved, missed, no_trade or in_progress"})
	}

	res, total, err := h.service.DashboardPeriods(c.Context(), auth.CurrentUser(c).ID, id, status, page, limit)
	if err != nil {
		return handleError(c, err)
	}
	return response.Paginated(c, "success", res, response.Meta{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: int(math.Ceil(float64(total) / float64(limit))),
	})
}

func (h *Handler) DashboardEquity(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	res, err := h.service.DashboardEquity(c.Context(), auth.CurrentUser(c).ID, id,
		strings.TrimSpace(c.Query("interval")), strings.TrimSpace(c.Query("from")), strings.TrimSpace(c.Query("to")))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}

func (h *Handler) DashboardCalendar(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	res, err := h.service.DashboardCalendar(c.Context(), auth.CurrentUser(c).ID, id, strings.TrimSpace(c.Query("month")))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}

func (h *Handler) DashboardBreakdown(c fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return handleError(c, err)
	}
	res, err := h.service.DashboardBreakdown(c.Context(), auth.CurrentUser(c).ID, id,
		strings.TrimSpace(c.Query("by")), strings.TrimSpace(c.Query("from")), strings.TrimSpace(c.Query("to")))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}

func (h *Handler) DashboardOverview(c fiber.Ctx) error {
	res, err := h.service.DashboardOverview(c.Context(), auth.CurrentUser(c).ID, c.Query("include_archived") == "true")
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}

func (h *Handler) DashboardRunners(c fiber.Ctx) error {
	res, err := h.service.RunnerMonitor(c.Context(), auth.CurrentUser(c).ID)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", res)
}
