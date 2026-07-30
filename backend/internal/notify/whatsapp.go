package notify

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // مشغّل database/sql لمخزن whatsmeow
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// TemplateFunc تعيد نص رسالة OTP (القالب ديناميكي من app_settings).
type TemplateFunc func(ctx context.Context, code string) string

// WhatsAppSender مزوّد OTP الفعلي عبر بوت واتساب (whatsmeow).
// الاقتران الأول: يظهر رمز QR في سجل الخادم ويُتاح عبر نقطة حالة الأدمن —
// يُمسح مرة واحدة من هاتف الرقم المخصص للبوت، وتُخزَّن الجلسة في PostgreSQL.
type WhatsAppSender struct {
	client   *whatsmeow.Client
	logger   *slog.Logger
	template TemplateFunc

	mu       sync.RWMutex
	qr       string
	lastErr  string
	pairedAs string
}

func NewWhatsAppSender(ctx context.Context, databaseURL string, logger *slog.Logger, template TemplateFunc) (*WhatsAppSender, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: open store db: %w", err)
	}

	container := sqlstore.NewWithDB(db, "postgres", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("whatsapp: upgrade store schema: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: get device: %w", err)
	}

	s := &WhatsAppSender{
		client:   whatsmeow.NewClient(device, waLog.Noop),
		logger:   logger,
		template: template,
	}

	// الاتصال بالخلفية مع إعادة محاولة — انقطاع واتساب لا يمنع إقلاع الخادم؛
	// طلبات OTP تفشل بخطأ واضح حتى يعود الاتصال (PLAN.md §6.8).
	go s.connectLoop(ctx)
	return s, nil
}

func (s *WhatsAppSender) connectLoop(ctx context.Context) {
	backoff := 5 * time.Second
	for {
		err := s.connectOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			s.setErr(err.Error())
			s.logger.Warn("whatsapp connect failed, retrying", "error", err, "backoff", backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 2*time.Minute {
			backoff *= 2
		}
	}
}

func (s *WhatsAppSender) connectOnce(ctx context.Context) error {
	if s.client.IsConnected() {
		<-ctx.Done()
		return nil
	}

	if s.client.Store.ID == nil {
		// جهاز غير مقترن بعد — مسار QR (يجب طلب القناة قبل الاتصال)
		qrChan, err := s.client.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		if err := s.client.Connect(); err != nil {
			return err
		}
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				s.setQR(evt.Code)
				s.logger.Info("whatsapp pairing: scan this QR from the bot phone (WhatsApp > Linked Devices)")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			case "success":
				s.setQR("")
				s.setPaired(s.client.Store.ID.User)
				s.logger.Info("whatsapp paired successfully", "as", s.client.Store.ID.User)
				return nil
			default:
				s.setQR("")
				return fmt.Errorf("whatsapp pairing ended: %s", evt.Event)
			}
		}
		return fmt.Errorf("whatsapp qr channel closed")
	}

	if err := s.client.Connect(); err != nil {
		return err
	}
	s.setPaired(s.client.Store.ID.User)
	s.setErr("")
	s.logger.Info("whatsapp connected", "as", s.client.Store.ID.User)
	<-ctx.Done()
	return nil
}

var errWANotReady = fmt.Errorf("whatsapp: bot not connected/paired yet")

func (s *WhatsAppSender) SendOTP(ctx context.Context, phone, code string) error {
	if !s.client.IsLoggedIn() {
		return errWANotReady
	}
	jid := types.NewJID(strings.TrimPrefix(phone, "+"), types.DefaultUserServer)
	text := s.template(ctx, code)
	_, err := s.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return fmt.Errorf("whatsapp: send otp: %w", err)
	}
	return nil
}

// Status حالة البوت — تعرضها نقطة أدمن لمتابعة الاقتران والاتصال.
func (s *WhatsAppSender) Status() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"provider":   "whatsapp",
		"connected":  s.client.IsConnected(),
		"logged_in":  s.client.IsLoggedIn(),
		"paired_as":  s.pairedAs,
		"qr":         s.qr,
		"last_error": s.lastErr,
	}
}

func (s *WhatsAppSender) setQR(v string) {
	s.mu.Lock()
	s.qr = v
	s.mu.Unlock()
}

func (s *WhatsAppSender) setErr(v string) {
	s.mu.Lock()
	s.lastErr = v
	s.mu.Unlock()
}

func (s *WhatsAppSender) setPaired(v string) {
	s.mu.Lock()
	s.pairedAs = v
	s.mu.Unlock()
}
