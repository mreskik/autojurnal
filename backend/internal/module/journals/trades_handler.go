package journals

import (
	"math"
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) ListTrades(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
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

	list, total, err := h.service.ListTrades(c.Context(), UserScope(auth.CurrentUser(c).ID), id, q)
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

func (h *Handler) GetTrade(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	t, err := h.service.GetTrade(c.Context(), UserScope(auth.CurrentUser(c).ID), id, tradeID)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", t)
}

func (h *Handler) CreateTrade(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	var req CreateTradeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CreateTrade(c.Context(), UserScope(auth.CurrentUser(c).ID), id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "trade created", t)
}

func (h *Handler) UpdateTrade(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateTradeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.UpdateTrade(c.Context(), UserScope(auth.CurrentUser(c).ID), id, tradeID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "trade updated", t)
}

func (h *Handler) CloseTrade(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	var req CloseFields
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CloseTrade(c.Context(), UserScope(auth.CurrentUser(c).ID), id, tradeID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "trade closed", t)
}

func (h *Handler) DeleteTrade(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	tradeID, err := paramUUID(c, "tradeId")
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.DeleteTrade(c.Context(), UserScope(auth.CurrentUser(c).ID), id, tradeID); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "trade deleted", nil)
}
