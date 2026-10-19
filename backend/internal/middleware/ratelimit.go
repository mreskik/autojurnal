package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"autojurnal/internal/config"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// Hitungan disimpan di memori server (hilang saat restart, tidak sinkron antar-instance).

// IPLimiter: batas global per IP, dipasang sebelum cek token.
func IPLimiter(cfg config.RateLimitConfig) fiber.Handler {
	return newLimiter(cfg, cfg.IPMax, time.Minute, false, func(c fiber.Ctx) string {
		return "ip:" + c.IP()
	})
}

// UserLimiter: batas per user, dipasang setelah RequireAuth.
// userID mengambil id user yang sedang login dari context.
func UserLimiter(cfg config.RateLimitConfig, userID func(c fiber.Ctx) string) fiber.Handler {
	return newLimiter(cfg, cfg.UserMax, time.Minute, false, func(c fiber.Ctx) string {
		return "user:" + userID(c)
	})
}

// LoginLimiter: batas percobaan login gagal per IP + username.
// Login yang berhasil tidak dihitung.
func LoginLimiter(cfg config.RateLimitConfig) fiber.Handler {
	return newLimiter(cfg, cfg.LoginMax, cfg.LoginWindow, true, func(c fiber.Ctx) string {
		var body struct {
			Username string `json:"username"`
		}
		_ = json.Unmarshal(c.Body(), &body)
		return "login:" + c.IP() + ":" + strings.ToLower(strings.TrimSpace(body.Username))
	})
}

func newLimiter(cfg config.RateLimitConfig, max int, window time.Duration, skipSuccess bool, key func(fiber.Ctx) string) fiber.Handler {
	if !cfg.Enabled {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return limiter.New(limiter.Config{
		Max:                    max,
		Expiration:             window,
		KeyGenerator:           key,
		SkipSuccessfulRequests: skipSuccess,
		LimiterMiddleware:      limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			// header Retry-After sudah di-set oleh limiter
			return response.Error(c, fiber.StatusTooManyRequests, response.CodeTooManyRequests, "too many requests, try again later", nil)
		},
	})
}
