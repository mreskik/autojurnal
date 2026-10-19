package journals

import (
	"strings"

	"autojurnal/internal/module/auth"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) ListRunners(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	list, err := h.service.ListRunners(c.Context(), auth.CurrentUser(c).ID, id)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", list)
}

func (h *Handler) ListRunnerOptions(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	list, err := h.service.ListRunnerOptions(c.Context(), auth.CurrentUser(c).ID, id, strings.TrimSpace(c.Query("search")))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", list)
}

func (h *Handler) AddRunner(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	var req AddRunnerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	list, err := h.service.AddRunner(c.Context(), auth.CurrentUser(c).ID, id, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, "runner added", list)
}

func (h *Handler) UpdateRunner(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	runnerID, err := paramUUID(c, "runnerId")
	if err != nil {
		return handleError(c, err)
	}
	var req UpdateJoinedRunnerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	jr, err := h.service.UpdateRunner(c.Context(), auth.CurrentUser(c).ID, id, runnerID, req)
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "runner updated", jr)
}

// DownloadAgentFile: unduh AGENT.md untuk runner di jurnal ini (berisi token, hanya pemilik yang login).
func (h *Handler) DownloadAgentFile(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	runnerID, err := paramUUID(c, "runnerId")
	if err != nil {
		return handleError(c, err)
	}
	baseURL := h.publicAPIURL
	if baseURL == "" {
		baseURL = c.BaseURL() + "/api"
	}
	content, err := h.service.AgentFile(c.Context(), auth.CurrentUser(c).ID, id, runnerID, strings.TrimRight(baseURL, "/"))
	if err != nil {
		return handleError(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/markdown; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="AGENT.md"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(content)
}

func (h *Handler) RemoveRunner(c fiber.Ctx) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return handleError(c, err)
	}
	runnerID, err := paramUUID(c, "runnerId")
	if err != nil {
		return handleError(c, err)
	}
	if err := h.service.RemoveRunner(c.Context(), auth.CurrentUser(c).ID, id, runnerID); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "runner removed", nil)
}
