// Package config يحمّل إعدادات التشغيل التقنية من متغيرات البيئة حصراً.
// ملاحظة: إعدادات العمل (رسوم، عمولات، مهل...) ليست هنا — تلك ديناميكية
// تُخزَّن في قاعدة البيانات وتُدار من لوحة الأدمن (راجع GROUND-RULES §1.3).
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env         string // development | staging | production
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("APP_ENV", "development"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://rahalgo:rahalgo_dev@localhost:5432/rahalgo?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
	}

	if cfg.Env == "production" {
		for name, v := range map[string]string{
			"DATABASE_URL": os.Getenv("DATABASE_URL"),
			"REDIS_URL":    os.Getenv("REDIS_URL"),
		} {
			if v == "" {
				return nil, fmt.Errorf("config: %s is required in production", name)
			}
		}
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
