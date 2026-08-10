// Package config يحمّل إعدادات التشغيل التقنية من متغيرات البيئة حصراً.
// ملاحظة: إعدادات العمل (رسوم، عمولات، مهل...) ليست هنا — تلك ديناميكية
// تُخزَّن في قاعدة البيانات وتُدار من لوحة الأدمن (راجع GROUND-RULES §1.3).
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// loadDotEnv **يقرأ `.env` من مجلّد التشغيل — إن وُجد.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا صار لازماً**
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ إعدادٍ يُمرَّر في سطر الأوامر يضيع مع النافذة.** ومن شغّل بـ
// `make run` أو ضغط زرّاً في محرّره **يعود إلى الافتراضات صامتاً**:
// `OTP_PROVIDER=dev` فتُطبع رموزُ الدخول في السجلّ بدل أن تصل بواتساب،
// **والمنصّةُ تعمل فلا شيءَ يقول إنّ شيئاً تبدّل.**
//
// **وقِيس ٢٠٢٦-٠٨-١٠**: البوتُ مقترنٌ ويعمل، **وأوّلُ إعادة تشغيلٍ من
// طريق المالك تعيده إلى `dev`.**
//
// # وما يُقرأ لا يطغى على ما كُتب
//
// **المتغيّرُ المضبوطُ في البيئة أقوى من الملفّ** — فمن مرّر شيئاً في سطر
// الأوامر أراده لهذه المرّة. **والملفُّ افتراضٌ لا أمر.**
//
// # ولا يُشترط وجودُه
//
// **ملفٌّ مفقودٌ ليس خطأً** — الافتراضاتُ في الشيفرة تكفي للتطوير، **وحرّاسُ
// الإنتاج تحت تمنع الإقلاع بها.**
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// **وعلامتا الاقتباس تُنزعان** — من كتب `X="ي"` أراد `ي` لا `"ي"`.
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' ||
			val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

func Load() (*Config, error) {
	// **ويُقرأ قبل أوّل `getEnv`** — وإلّا قُرئت الافتراضاتُ ثمّ جاء الملفّ.
	loadDotEnv(".env")
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

	// ══════════════════════════════════════════════════════════════════
	// **ومسارُ الرفع يُحلّ مطلقاً — لا يُترك نسبيّاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (كشفه جردُ لوحة الزبون 2026-08-09: سبعُ صورٍ مكسورةٍ في ثلاث صفحات.)
	//
	// **الافتراضُ `./uploads` — وهو نسبيٌّ إلى مجلّد التشغيل لا إلى المشروع.**
	// **فمن أقلع المحرّكَ من مجلّدٍ آخرَ كتب صورَه في مكانٍ ثانٍ** — وقُسم
	// فوُجد نصفُ الملفّات في `backend/uploads` ونصفُها في `web/uploads`،
	// **والقاعدةُ تشير إليها كلِّها.**
	//
	// **ولا خطأ ولا سجلّ**: الرفعُ ينجح والصفُّ يُكتب، **والصورةُ تُطلب فتردّ
	// 404** — ولا يعرف أحدٌ لماذا. **وفي الإنتاج يعني هذا فقدَ كلّ صورةٍ
	// رُفعت** يومَ يتبدّل مجلّدُ الخدمة.
	//
	// **فيُحلّ مطلقاً هنا** — ويُذكر في السجلّ عند الإقلاع، **فمن رآه في غير
	// موضعه عرف قبل أن يرفع صورةً واحدة.**
	if abs, err := filepath.Abs(cfg.UploadsDir); err == nil {
		cfg.UploadsDir = abs
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
		// **ومزوّدُ الرمز لا يكون `dev` في الإنتاج.**
		//
		// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧.)
		//
		// `DevSender` **يطبع الرمزَ في السجلّ ولا يرسله**. ومن أقلع ونسي
		// المتغيّرَ **فتح البابَ لمن يقرأ السجلّ**: رمزُ دخولِ أيِّ حساب
		// مطبوعاً بجانب رقمه. **والخادمُ يقلع والدخولُ يعمل** — فلا شيءَ
		// يقول إنّ الرمزَ يصل من لا يجب.
		//
		// **و`JWT_SECRET` له حارسُه فوق** — ونُسي أخوه.
		if cfg.OTPProvider == "dev" {
			return nil, fmt.Errorf("config: OTP_PROVIDER=dev prints codes to the log — not allowed in production")
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
