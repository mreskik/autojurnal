package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // data timezone ikut di binary (untuk timezone jurnal)

	"autojurnal/internal"
	"autojurnal/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := config.NewDatabase(cfg.DB)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	appCfg := fiber.Config{AppName: "autojurnal"}
	if len(cfg.TrustedProxies) > 0 {
		// di belakang Nginx: IP asli dibaca dari X-Real-IP,
		// hanya kalau request datang dari proxy yang terdaftar
		appCfg.TrustProxy = true
		appCfg.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: cfg.TrustedProxies}
		appCfg.ProxyHeader = "X-Real-IP"
		appCfg.EnableIPValidation = true
	}

	app := fiber.New(appCfg)

	// frontend (browser) di origin lain, misal Vite localhost:5173
	app.Use(cors.New(cors.Config{
		AllowOrigins:  cfg.CORSAllowedOrigins,
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Authorization", "Content-Type", "X-Journal-Token"},
		ExposeHeaders: []string{"Content-Disposition", "Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		MaxAge:        600,
	}))
	internal.SetupRouter(app, db, cfg)

	// graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Println("shutting down...")
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Printf("server stopped: %v", err)
	}
}
