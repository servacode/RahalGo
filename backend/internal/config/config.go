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

	JWTSecret   string
	OTPProvider string // dev | whatsapp
	AdminPhone  string // هاتف أول أدمن — يُمنح الدور تلقائياً عند الإقلاع
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("APP_ENV", "development"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://rahalgo:rahalgo_dev@localhost:5432/rahalgo?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		OTPProvider: getEnv("OTP_PROVIDER", "dev"),
		AdminPhone:  getEnv("ADMIN_PHONE", ""),
	}

	if cfg.Env == "production" {
		for name, v := range map[string]string{
			"DATABASE_URL": os.Getenv("DATABASE_URL"),
			"REDIS_URL":    os.Getenv("REDIS_URL"),
			"JWT_SECRET":   os.Getenv("JWT_SECRET"),
		} {
			if v == "" {
				return nil, fmt.Errorf("config: %s is required in production", name)
			}
		}
		if cfg.JWTSecret == "dev-secret-change-me" {
			return nil, fmt.Errorf("config: JWT_SECRET must not use the dev default in production")
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
