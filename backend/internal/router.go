package internal

import (
	"context"
	"time"

	"autojurnal/internal/config"
	"autojurnal/internal/middleware"
	"autojurnal/internal/module/auth"
	"autojurnal/internal/module/journals"
	"autojurnal/internal/module/master/users"
	"autojurnal/internal/module/runners"
	"autojurnal/internal/response"

	"github.com/gofiber/fiber/v3"
	"github.com/uptrace/bun"
)

// SetupRouter mendaftarkan semua route di bawah prefix /api.
func SetupRouter(app *fiber.App, db *bun.DB, cfg *config.Config) {
	// batas global per IP, jalan sebelum cek token
	api := app.Group("/api", middleware.IPLimiter(cfg.RateLimit))

	api.Get("/health", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return response.Error(c, fiber.StatusServiceUnavailable, response.CodeDatabaseUnavailable, "database unavailable", nil)
		}
		return response.Success(c, fiber.StatusOK, "ok", nil)
	})

	authHandler := auth.NewHandler(db, cfg.SessionTTL)

	// middleware untuk route yang wajib login: cek token, lalu batas per user
	protected := []any{
		authHandler.RequireAuth,
		middleware.UserLimiter(cfg.RateLimit, func(c fiber.Ctx) string {
			return auth.CurrentUser(c).ID.String()
		}),
	}

	authHandler.RegisterRoutes(api.Group("/auth"), middleware.LoginLimiter(cfg.RateLimit), protected...)
	users.RegisterRoutes(api.Group("/users", protected...), db)

	journalHandler := journals.NewHandler(db, cfg.PublicAPIURL)
	journalHandler.RegisterRoutes(api.Group("/journals", protected...))
	journalHandler.RegisterDashboardRoutes(api.Group("/dashboard", protected...))

	runnerHandler := runners.NewHandler(db)
	runnerHandler.RegisterRoutes(api.Group("/runners", protected...))

	// endpoint untuk agent/bot, auth pakai token runner (bukan session user)
	runnerAPI := api.Group("/runner",
		runnerHandler.RequireRunner,
		middleware.UserLimiter(cfg.RateLimit, func(c fiber.Ctx) string {
			return "runner-" + runners.CurrentRunner(c).ID.String()
		}),
	)
	runnerHandler.RegisterRunnerRoutes(runnerAPI)
	journalHandler.RegisterRunnerRoutes(runnerAPI)

	// route yang tidak terdaftar
	api.Use(func(c fiber.Ctx) error {
		return response.Error(c, fiber.StatusNotFound, response.CodeNotFound, "route not found", nil)
	})
}
