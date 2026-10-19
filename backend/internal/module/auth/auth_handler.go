package auth

import (
	"errors"
	"log"
	"strings"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	localSession = "auth_session"
	localUser    = "auth_user"
)

type Handler struct {
	service *Service
}

func NewHandler(db *bun.DB, sessionTTL time.Duration) *Handler {
	return &Handler{service: NewService(NewRepository(db), users.NewRepository(db), sessionTTL)}
}

// RegisterRoutes: protected = middleware untuk route yang wajib login (RequireAuth + limiter).
func (h *Handler) RegisterRoutes(router fiber.Router, loginLimiter fiber.Handler, protected ...any) {
	router.Post("/login", loginLimiter, h.Login)

	private := router.Group("", protected...)
	private.Post("/logout", h.Logout)
	private.Post("/logout-all", h.LogoutAll)
	private.Get("/me", h.Me)
	private.Get("/sessions", h.ListSessions)
	private.Delete("/sessions/:id", h.RevokeSession)
}

// RequireAuth: middleware untuk route yang wajib login.
// Header: Authorization: Bearer <token>
func (h *Handler) RequireAuth(c fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok {
		return handleError(c, ErrUnauthorized)
	}
	session, user, err := h.service.Authenticate(c.Context(), strings.TrimSpace(token))
	if err != nil {
		return handleError(c, err)
	}
	c.Locals(localSession, session)
	c.Locals(localUser, user)
	return c.Next()
}

// CurrentUser mengambil user yang sedang login (hanya di route dengan RequireAuth).
func CurrentUser(c fiber.Ctx) *users.User {
	u, _ := c.Locals(localUser).(*users.User)
	return u
}

func CurrentSession(c fiber.Ctx) *Session {
	s, _ := c.Locals(localSession).(*Session)
	return s
}

func (h *Handler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body", nil)
	}
	res, err := h.service.Login(c.Context(), req, c.IP(), c.Get(fiber.HeaderUserAgent))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "login success", res)
}

func (h *Handler) Logout(c fiber.Ctx) error {
	if err := h.service.Logout(c.Context(), CurrentSession(c)); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "logout success", nil)
}

func (h *Handler) LogoutAll(c fiber.Ctx) error {
	if err := h.service.LogoutAll(c.Context(), CurrentUser(c).ID); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "all sessions logged out", nil)
}

func (h *Handler) Me(c fiber.Ctx) error {
	return response.Success(c, fiber.StatusOK, "success", users.ToUserResponse(CurrentUser(c)))
}

func (h *Handler) ListSessions(c fiber.Ctx) error {
	list, err := h.service.ListSessions(c.Context(), CurrentSession(c))
	if err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "success", list)
}

func (h *Handler) RevokeSession(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.Error(c, fiber.StatusBadRequest, response.CodeInvalidParam, "invalid id", nil)
	}
	if err := h.service.RevokeSession(c.Context(), CurrentUser(c).ID, id); err != nil {
		return handleError(c, err)
	}
	return response.Success(c, fiber.StatusOK, "session revoked", nil)
}

func handleError(c fiber.Ctx, err error) error {
	var verr users.ValidationErrors
	switch {
	case errors.As(err, &verr):
		return response.Error(c, fiber.StatusUnprocessableEntity, response.CodeValidationFailed, "validation failed", verr)
	case errors.Is(err, ErrInvalidCredentials):
		return response.Error(c, fiber.StatusUnauthorized, response.CodeInvalidCredentials, err.Error(), nil)
	case errors.Is(err, ErrUnauthorized):
		return response.Error(c, fiber.StatusUnauthorized, response.CodeUnauthorized, err.Error(), nil)
	case errors.Is(err, ErrSessionNotFound):
		return response.Error(c, fiber.StatusNotFound, response.CodeSessionNotFound, err.Error(), nil)
	default:
		log.Printf("auth: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, response.CodeInternalError, "internal server error", nil)
	}
}
