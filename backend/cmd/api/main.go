package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/database"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/migrate"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/server"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pg.Close()

	rdb, err := database.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	applied, err := migrate.Up(ctx, pg)
	if err != nil {
		return err
	}
	if applied > 0 {
		logger.Info("migrations applied", "count", applied)
	}

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, 15*time.Minute)
	settingsStore := settings.NewStore(pg)

	var otpSender notify.OTPSender
	// **وبوتُ واتساب يُبلّغ المتاجر أيضاً** — انظر حيث يُحقن.
	//
	// **ويُصرَّح به بنوعه لا بواجهته**: **مؤشّرٌ فارغٌ داخل واجهةٍ ليس
	// واجهةً فارغة** — فيمرّ فحصُ `!= nil` ثمّ ينهار عند أوّل نداء.
	var waBot *notify.WhatsAppSender
	otpStatus := func() map[string]any { return map[string]any{"provider": "dev"} }
	// **وفكُّ الاقتران لا وجودَ له في مزوّد التطوير** — ولا يُخترع زرٌّ لا يفعل.
	var otpUnpair func(context.Context) error
	var otpPair func()
	switch cfg.OTPProvider {
	case "dev":
		otpSender = &notify.DevSender{Logger: logger}
	case "whatsapp":
		// **والقالبُ من الفهرس لا من هنا.**
		//
		// كان مكتوباً في هذا الملفّ **وفي الفهرس** — نصّان لرسالةٍ واحدة.
		// **فيُصحَّح أحدُهما ويبقى الآخرُ** يُرسَل لمن لم يُخزَّن له قالبٌ بعد،
		// **ولا يظهر الفرقُ إلّا في هاتف زبون.**
		wa, err := notify.NewWhatsAppSender(ctx, cfg.DatabaseURL, logger,
			func(ctx context.Context, code string) string {
				tpl := settingsStore.GetString(ctx, "whatsapp.otp_template")
				return strings.ReplaceAll(tpl, "{code}", code)
			},
			// **واسمُ الجهاز اسمُ المنصّة** — يظهر في «الأجهزة المرتبطة»
			// بهاتف البوت. **ومن الإعدادات لا من الشيفرة**: نسخةُ كلّ مشترٍ
			// تُظهر اسمَها هي.
			settingsStore.GetString(ctx, "platform.name"))
		if err != nil {
			return err
		}
		otpSender = wa
		waBot = wa
		otpStatus = wa.Status
		otpUnpair = wa.Unpair
		otpPair = wa.Pair
		// **والتمهّلُ من الإعدادات ويُقرأ عند كلّ إرسال** — رقمٌ يُبدَّل في
		// اللوحة يعمل بلا إعادة تشغيل.
		wa.SetDelay(func(ctx context.Context) time.Duration {
			return time.Duration(settingsStore.GetInt(ctx, "whatsapp.send_delay_ms")) * time.Millisecond
		})
	default:
		return fmt.Errorf("unknown OTP_PROVIDER %q (expected dev or whatsapp)", cfg.OTPProvider)
	}

	// ══════════════════════════════════════════════════════════════════
	// **وبأيّ طريقٍ يصل الرمز — يُبدَّل من اللوحة لا بنشرٍ جديد**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «ميزة تحقّق SMS… نقدر نفعّلها أو نطفّيها
	//  من لوحة التحكّم».)
	//
	// **وبوّابةُ الرسائل تُبنى مرّةً هنا** — كانت تُبنى داخلَ بانيَ
	// المُوجِّه وحدَه، **ونسختان من ضبطٍ واحدٍ تفترقان يومَ يُزاد حقل.**
	//
	// **ولا يُلَفُّ مُرسِلُ التطوير**: يطبع الرمزَ في الطرفيّة، **ومن
	// يطوّر بلا واتسابَ ولا بوّابةٍ لا يعنيه التبديل.** (انظر
	// `notify/channel.go`.)
	smsSender := notify.NewSMSSender(notify.SMSConfig{
		URL:         cfg.SMSURL,
		Method:      cfg.SMSMethod,
		Body:        cfg.SMSBody,
		ContentType: cfg.SMSContentType,
		AuthHeader:  cfg.SMSAuthHeader,
		Sender:      cfg.SMSSender,
	}, logger)
	// **وبوّابةُ الرمز تُبنى وحدَها إن ضُبطت** — انظر `config.go`.
	otpSMS := smsSender
	if cfg.SMSOTPURL != "" {
		otpSMS = notify.NewSMSSender(notify.SMSConfig{
			URL:        cfg.SMSOTPURL,
			Body:       cfg.SMSOTPBody,
			AuthHeader: cfg.SMSOTPAuthHeader,
			Sender:     cfg.SMSSender,
		}, logger)
	}
	if cfg.OTPProvider == "whatsapp" {
		otpSender = notify.NewOTPChannel(otpSender, otpSMS,
			func(code string) string {
				tpl := settingsStore.GetString(context.Background(), "auth.sms_template")
				return strings.ReplaceAll(tpl, "{code}", code)
			},
			func(ctx context.Context) string {
				return settingsStore.GetString(ctx, "auth.otp_channel")
			}, logger)
	}

	// readSetting **جسرٌ إلى اللوحة تعبره الحزمُ الدنيا.**
	//
	// قرارات الأمان والحدود يتّخذها المالك لا مبرمجٌ في نصّ. **ويُربط هنا كي
	// لا تعتمد حزمةُ الهوية ولا حزمةُ الوسائط على حزمة الإعدادات** — وكلتاهما
	// أدنى منها في الترتيب.
	// **والترجمةُ في `settings.GetNum` لا هنا** — تقبل العدديَّ والمنطقيَّ
	// معاً، **ودالّةٌ مجهولةٌ داخلَ دالّةِ إقلاعٍ لا يبلغها اختبار.**
	readSetting := settingsStore.GetNum

	identitySvc := identity.NewService(identity.NewRepo(pg), rdb, tokens, otpSender, cfg.JWTSecret, logger)

	// ══════════════════════════════════════════════════════════════════
	// **ومن راسلنا يُردّ عليه** — انظر `identity/wa_inbound.go`.
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك 2026-08-25 بعد أن قُيّد رقمُ المنصّة مرّتين في يوم.)
	//
	// **و`notify` لا تعرف حساباً ولا رمزا** — فتُحقن الدالّةُ هنا كما
	// حُقنت `delay` و`template`. **ويبقى الترتيبُ سليماً: الأدنى لا
	// يعرف الأعلى.**
	//
	// **وتُربط بعد إنشاء الخدمة لا قبله** — ومن ربطها فوقُ ربط عدما.
	if waBot != nil {
		waBot.SetInbound(func(c context.Context, from, text string) string {
			return identitySvc.HandleWAInbound(c, from, text)
		})
	}
	identitySvc.SetSettingReader(readSetting)
	if err := identitySvc.BootstrapAdmin(ctx, cfg.AdminPhone); err != nil {
		return err
	}
	catalogSvc := catalog.NewService(pg, identitySvc)
	walletSvc := wallet.NewService(pg)

	// **ولا تعمل المنصةُ بلا خزينة** — انظر الشرحَ عند `EnsureTreasury`.
	//
	// **ويُقال في السجلّ**: من عيّنها آليّاً يحقّ له أن يعرف لمن.
	if tid, err := wallet.EnsureTreasury(ctx, pg); err != nil {
		return err
	} else if tid == "" {
		logger.Warn("لا خزينةَ للمنصة ولا إداريَّ بعد — مصروفُ المنصة لن يُقيَّد")
	}
	cashboxSvc := cashbox.NewService(pg, settingsStore)
	hub := realtime.NewHub(logger)
	ordersSvc := orders.NewService(pg, identitySvc, walletSvc, cashboxSvc, hub, logger)

	// ══════════════════════════════════════════════════════════════
	// **ومكتبٌ فارغٌ يعني راصداً لا يُنذر أحداً** — `PF-07` · `W9`
	// ══════════════════════════════════════════════════════════════
	//
	// **العقدُ**: **لا يُوسَم طلبٌ «أُنذر» ولا مستقبِلَ** — **وسمٌ بلا
	// مُنذَرٍ كذبٌ يُسكته أبداً.** **فالراصدُ يمتنع ويُعيد.**
	//
	// **وامتناعُه صحيحٌ وصامت** — **فيُقال عند الإقلاع لا في السجلّ
	// الدوريّ**: من شغّل خدمةً بلا مكتبٍ يحقّ له أن يعرف **قبل** أن
	// يعلق طلبٌ بلا مُنذِر.
	//
	// **و`BootstrapAdmin` يضمنه إن ضُبط `ADMIN_PHONE`** — **والتحقّقُ
	// يقيس النتيجةَ لا النيّة.**
	if n, err := opsDeskSize(ctx, pg); err != nil {
		logger.Warn("تعذّر عدُّ مكتب العمليّات", "error", err)
	} else if n == 0 {
		logger.Warn("**لا موظّفَ عمليّاتٍ فاعل** — " +
			"**الراصدُ لن يُنذر ولن يسم، والطلبُ العالقُ يبقى مستحقّاً " +
			"حتّى يُعيَّن أحد** (`PF-07`)")
	} else {
		logger.Info("مكتبُ العمليّات", "recipients", n)
	}

	go ordersSvc.RunWatchdog(ctx, 30*time.Second)
	supportSvc := support.NewService(pg, identitySvc, walletSvc)
	supportSvc.SetSettings(settingsStore)
	// **وسرُّ توقيع الوسائط** — `D13`: **الشخصيُّ يُخدَم برابطٍ موقَّعٍ
	// محدودِ الأجل**، لأنّ وسمَ `<img>` لا يحمل ترويسةَ مصادقة.
	// **وبلا سرٍّ يُحجَب المحميُّ كلُّه** — سقوطٌ مغلقٌ لا مفتوح.
	media.SetSigningKey(cfg.JWTSecret)
	mediaSvc, err := media.NewService(pg, cfg.UploadsDir)
	if err != nil {
		return err
	}
	mediaSvc.SetSettingReader(readSetting)

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: func() http.Handler {
			srv := server.New(cfg, logger, pg, rdb, tokens, identitySvc, catalogSvc,
				settingsStore, walletSvc, ordersSvc, cashboxSvc, supportSvc, mediaSvc, hub, otpStatus, otpUnpair, otpPair)
			// **إبلاغُ المتاجر برسالةٍ نصّية لا ببوت واتساب**: البوت غيرُ رسميّ
			// ويُحظَر إن أكثر من الإرسال الآليّ. وواتساب الرسميّ لاحقاً — يدخل
			// من الواجهة نفسها (`notify.TextSender`) بلا تغييرٍ فيمن يستعملها.
			// **والمتاجرُ تُبلَّغ بالبوت لا برسالةٍ نصّيّة.**
			//
			// (قرارُ المالك ٢٠٢٦-٠٨-٢٠.)
			//
			// **والمتاجرُ أرقامٌ قليلةٌ معروفةٌ تتكرّر** — وهو أهونُ ما
			// يحمله بوتٌ غيرُ رسميّ. **وخطرُ الحظر في مراسلة أرقامٍ
			// جديدةٍ لم تُراسَل من قبل**، وتلك رموزُ التحقّق وحدَها —
			// **ولها قناتُها التي تُبدَّل من اللوحة.**
			if waBot != nil {
				srv.SetMerchantNotifier(waBot)
			}
			// **ويستأنف ما عَلِق من التحويل التلقائيّ** — `PF-08`.
			//
			// **والخيطُ السريعُ بعد الإنشاء تعجيلٌ لا مصدرَ حقيقة**:
			// **حالُ الطلب هي مصدرُ العمل**، فمن مات خيطُه استأنفه
			// الكانس. **وبعد ضبط المُبلِّغ** — فيرى الكانسُ ما يراه
			// المسارُ الحيّ.
			go srv.RunAutoTransferSweeper(ctx, 30*time.Second)

			// **وعاملُ دفع الإشعارات** — `PF-09`.
			//
			// **والنبضةُ بعد كلّ إشعارٍ تعجيلٌ لا مصدرَ حقيقة**:
			// **صفوفُ النقل هي العمل**، فمن ماتت نبضتُه أو مات
			// منفّذُه التقطته الجولةُ الدوريّة.
			//
			// **وثلاثون ثانيةً كنبضة الراصد والكانس** — **إيقاعٌ
			// واحدٌ في المنصّة أسهلُ في قراءة السجلّ.**
			go srv.RunPushDeliveryWorker(ctx, 30*time.Second)
			return srv.Router()
		}(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		// **ومجلّدُ الوسائط يُذكر في السجلّ.**
		//
		// (كشفه جردُ 2026-08-09: مجلّدا رفعٍ لا واحد — انظر `config.go`.)
		//
		// **ومن رآه في غير موضعه عرف قبل أن يرفع صورةً واحدة** — ولا يُكتشف
		// الخطأُ بعد شهرٍ بصورةٍ مكسورةٍ لا يُعرف سببُها.
		logger.Info("http server started",
			"addr", cfg.HTTPAddr, "env", cfg.Env, "uploads", cfg.UploadsDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// opsDeskSize عددُ من يصلهم إنذارُ الراصد — **الاستعلامُ نفسُه الذي
// يستعمله `NotifyOpsTx`**، فلا يفترقان.
func opsDeskSize(ctx context.Context, pg *pgxpool.Pool) (int, error) {
	var n int
	err := pg.QueryRow(ctx, `
		SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`,
		notifications.OpsDesk).Scan(&n)
	return n, err
}
