package journals

import (
	"math"
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/module/master/users"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) ListTransactions(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	q := ListTransactionQuery{
		Page:  queryInt(c, "page", 1),
		Limit: queryInt(c, "limit", 10),
		Type:  strings.TrimSpace(c.Query("type")),
		From:  strings.TrimSpace(c.Query("from")),
		To:    strings.TrimSpace(c.Query("to")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 10
	}
	if q.Type != "" && q.Type != TxDeposit && q.Type != TxWithdraw {
		return handleError(c, users.ValidationErrors{"type": "must be deposit or withdraw"})
	}

	list, total, err := h.service.ListTransactions(c.Context(), auth.CurrentUser(c).ID, id, q)
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

func (h *Handler) CreateTransaction(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	var req CreateTransactionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.CreateTransaction(c.Context(), auth.CurrentUser(c).ID, id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "transaction created", t)
}

func (h *Handler) UpdateTransaction(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	txID, err := paramUUID(c, "txId")
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateTransactionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	t, err := h.service.UpdateTransaction(c.Context(), auth.CurrentUser(c).ID, id, txID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "transaction updated", t)
}

func (h *Handler) DeleteTransaction(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	txID, err := paramUUID(c, "txId")
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.DeleteTransaction(c.Context(), auth.CurrentUser(c).ID, id, txID); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "transaction deleted", nil)
}
