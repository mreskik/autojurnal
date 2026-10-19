package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	SessionTTL time.Duration
	// URL publik API yang bisa diakses agent, dipakai di AGENT.md. Kosong = dari alamat request.
	PublicAPIURL string
	// origin frontend yang boleh memanggil API dari browser (CORS)
	CORSAllowedOrigins []string
	// IP Nginx/reverse proxy yang dipercaya untuk header X-Real-IP. Kosong = tidak di belakang proxy.
	TrustedProxies []string
	RateLimit      RateLimitConfig
	DB             DBConfig
}

type RateLimitConfig struct {
	Enabled     bool
	LoginMax    int
	LoginWindow time.Duration
	IPMax       int
	UserMax     int
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Pass     string
	SSLMode  string
	MaxConns int
	Debug    bool
}

// Load membaca .env (kalau ada) lalu environment variable.
func Load() (*Config, error) {
	// .env opsional, di production biasanya pakai env asli
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:            getEnv("APP_PORT", "1000"),
		SessionTTL:         time.Duration(getEnvInt("SESSION_TTL_HOURS", 168)) * time.Hour,
		TrustedProxies:     getEnvList("TRUSTED_PROXIES"),
		PublicAPIURL:       strings.TrimRight(os.Getenv("PUBLIC_API_URL"), "/"),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS"),
		RateLimit: RateLimitConfig{
			Enabled:     getEnv("RATE_LIMIT_ENABLED", "true") == "true",
			LoginMax:    getEnvInt("RATE_LIMIT_LOGIN_MAX", 5),
			LoginWindow: time.Duration(getEnvInt("RATE_LIMIT_LOGIN_WINDOW_MINUTES", 15)) * time.Minute,
			IPMax:       getEnvInt("RATE_LIMIT_IP_MAX", 300),
			UserMax:     getEnvInt("RATE_LIMIT_USER_MAX", 120),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Pass:     os.Getenv("DB_PASS"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 10),
			Debug:    getEnv("DB_DEBUG", "false") == "true",
		},
	}

	if len(cfg.CORSAllowedOrigins) == 0 {
		// default untuk development (Vite)
		cfg.CORSAllowedOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}

	if cfg.DB.Name == "" || cfg.DB.User == "" {
		return nil, fmt.Errorf("DB_NAME and DB_USER are required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}

// getEnvList memecah nilai dipisah koma, contoh "127.0.0.1, 10.0.0.0/8".
func getEnvList(key string) []string {
	var list []string
	for _, v := range strings.Split(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			list = append(list, v)
		}
	}
	return list
}
