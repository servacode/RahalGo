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

	// بوّابةُ الرسائل النصّية — إبلاغُ المتاجر بطلباتها.
	//
	// **رسالةٌ نصّية لا بوت واتساب**: البوت عندنا غيرُ رسميّ، يحتاج اقتراناً
	// بهاتفٍ وينقطع، **ويُحظَر حسابُه إن أكثر من الإرسال الآليّ**. والرسالةُ
	// النصّية تصل أيَّ هاتفٍ بلا اقتران. (وواتساب الرسميّ لاحقاً — والواجهة
	// `TextSender` تقبله بلا تغييرٍ فيمن يستعملها.)
	//
	// **وأسرارُها في البيئة لا في الإعدادات**: جدولُ `app_settings` تقرؤه
	// نقطةٌ متاحة للأدمن والعمليات والمالية — فمفتاحُ المزوّد فيها يُسلَّم
	// لثلاثة أدوار لا شأن لاثنين منها به.
	SMSURL         string
	SMSMethod      string
	SMSBody        string
	SMSContentType string
	SMSAuthHeader  string
	SMSSender      string
	AdminPhone     string // هاتف أول أدمن — يُمنح الدور تلقائياً عند الإقلاع
	UploadsDir     string // مجلد تخزين الوسائط المرفوعة (خارج الحاوية في الإنتاج)
	// خدمة العنونة (Nominatim) — تُستبدل بنسخة ذاتية الاستضافة عند النشر
	GeocoderURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:      getEnv("APP_ENV", "development"),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		// **المنفذان ٥٤٣٤ و٦٣٨٠ — لا ٥٤٣٢ و٦٣٧٩.**
		//
		// هذان منفذا حاويتَي المشروع في `docker-compose.yml`. والمنفذان
		// القياسيّان يشغلهما مشروعٌ آخر على الجهاز نفسه، **فافتراضُهما هنا
		// يجعل تشغيلاً بلا بيئةٍ يمدّ يدَه إلى قاعدة غيرنا** — وهو ما وقع
		// فعلاً (نجا بفشل استيثاق، لا بتصميم).
		//
		// **والافتراضُ يوافق ما في `docker-compose.yml` لا ما هو شائع.**
		DatabaseURL: getEnv("DATABASE_URL", "postgres://rahalgo:rahalgo_dev@localhost:5434/rahalgo?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6380/0"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		OTPProvider: getEnv("OTP_PROVIDER", "dev"),

		SMSURL:         getEnv("SMS_URL", ""),
		SMSMethod:      getEnv("SMS_METHOD", "POST"),
		SMSBody:        getEnv("SMS_BODY", ""),
		SMSContentType: getEnv("SMS_CONTENT_TYPE", "application/json"),
		SMSAuthHeader:  getEnv("SMS_AUTH_HEADER", ""),
		SMSSender:      getEnv("SMS_SENDER", ""),
		AdminPhone:     getEnv("ADMIN_PHONE", ""),
		UploadsDir:     getEnv("UPLOADS_DIR", "./uploads"),
		GeocoderURL:    getEnv("GEOCODER_URL", "https://nominatim.openstreetmap.org"),
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
