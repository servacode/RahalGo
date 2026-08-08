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

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/database"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/migrate"
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
	otpStatus := func() map[string]any { return map[string]any{"provider": "dev"} }
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
			})
		if err != nil {
			return err
		}
		otpSender = wa
		otpStatus = wa.Status
	default:
		return fmt.Errorf("unknown OTP_PROVIDER %q (expected dev or whatsapp)", cfg.OTPProvider)
	}

	identitySvc := identity.NewService(identity.NewRepo(pg), rdb, tokens, otpSender, cfg.JWTSecret, logger)
	// طول كلمة المرور وسقف محاولات الدخول من اللوحة — قرارا أمانٍ يتّخذهما
	// المالك لا قرارا نشرٍ ينتظران مبرمجاً. (يُربط هنا كي لا تعتمد حزمةُ
	// الهوية على حزمة الإعدادات.)
	identitySvc.SetSettingReader(func(ctx context.Context, key string, fallback int64) int64 {
		var v float64
		if err := settingsStore.Get(ctx, key, &v); err != nil {
			return fallback
		}
		return int64(v)
	})
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
	go ordersSvc.RunWatchdog(ctx, 30*time.Second)
	supportSvc := support.NewService(pg, identitySvc, walletSvc)
	supportSvc.SetSettings(settingsStore)
	mediaSvc, err := media.NewService(pg, cfg.UploadsDir)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: func() http.Handler {
			srv := server.New(cfg, logger, pg, rdb, tokens, identitySvc, catalogSvc,
				settingsStore, walletSvc, ordersSvc, cashboxSvc, supportSvc, mediaSvc, hub, otpStatus)
			// **إبلاغُ المتاجر برسالةٍ نصّية لا ببوت واتساب**: البوت غيرُ رسميّ
			// ويُحظَر إن أكثر من الإرسال الآليّ. وواتساب الرسميّ لاحقاً — يدخل
			// من الواجهة نفسها (`notify.TextSender`) بلا تغييرٍ فيمن يستعملها.
			srv.SetTextSender(notify.NewSMSSender(notify.SMSConfig{
				URL:         cfg.SMSURL,
				Method:      cfg.SMSMethod,
				Body:        cfg.SMSBody,
				ContentType: cfg.SMSContentType,
				AuthHeader:  cfg.SMSAuthHeader,
				Sender:      cfg.SMSSender,
			}, logger))
			return srv.Router()
		}(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server started", "addr", cfg.HTTPAddr, "env", cfg.Env)
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
