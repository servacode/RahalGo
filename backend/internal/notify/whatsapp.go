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
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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

// NewWhatsAppSender **يُنشئ البوتَ باسم المنصّة لا باسم المكتبة.**
//
// # ولماذا الاسمُ يُضبط
//
// (سؤالُ المالك ٢٠٢٦-٠٨-١٠ بلقطةٍ من «الأجهزة المرتبطة»: ظهرت الجلسةُ باسم
//
//	`whatsmeow`.)
//
// **المكتبةُ تُعرّف نفسَها باسمها إن لم يُعطَ اسم** — فتظهر جلسةُ المنصّة في
// هاتف صاحبها بين جلسات المتصفّح باسمٍ لا يقول شيئاً.
//
// **وأنفعُ ما فيه أن يعرف أيَّ جلسةٍ لا يُغلق**: بعد شهرٍ تصير القائمةُ
// ثلاثاً، **واسمٌ غامضٌ يجعله يُسجّل خروجَ البوت بيده** — فتتوقّف الرموزُ
// عن الوصول ولا يعرف لماذا.
//
// # والاسمُ من الإعدادات لا من الشيفرة
//
// **قاعدةُ المالك: لا يُكتب اسمُ المنصّة في أيّ موضع** — وله حارسٌ في الويب.
// **وبيعُ النسخ يجعلها ألزم**: نسخةُ الشام تُظهر اسمَها هي، **ولو كُتب هنا
// لَظهر «رحال غو» في هاتف مشترٍ آخر.**
//
// **وفارغٌ يعود إلى اسم المكتبة** — ولا تُخترع تسميةٌ لمنصّةٍ لم تُسمِّ نفسَها.
func NewWhatsAppSender(ctx context.Context, databaseURL string, logger *slog.Logger, template TemplateFunc, deviceName string) (*WhatsAppSender, error) {
	if n := strings.TrimSpace(deviceName); n != "" {
		store.DeviceProps.Os = proto.String(n)
	}
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
	s.watchLogout(ctx)
	go s.connectLoop(ctx)
	return s, nil
}

// connectLoop **يصل ويبقى موصولاً — ولا يقف عند أوّل نجاح.**
//
// # عطبان كشفهما أوّلُ اقترانٍ حقيقيّ (٢٠٢٦-٠٨-١٠)
//
// **الأوّل: نجاحُ الاقتران كان نهايةَ المطاف.** تعود الدالّةُ عند «success»،
// ثمّ تُنادى ثانيةً فتجد `IsConnected()` صادقةً **فتقف عند `<-ctx.Done()`
// إلى الأبد.** **فإن سقط الاتصالُ بعدها لا يُعاد وصلُه** — ويبقى البوتُ
// ميّتاً حتّى يُعاد تشغيلُ الخادم كلِّه.
//
// **والثاني: الاقترانُ نجح والحالةُ تقول `logged_in: false`** — الرمزُ مُسح
// من الهاتف والجلسةُ لم تكتمل، **فيُردّ كلُّ رمزِ تحقّقٍ بـ«البوت غير
// مقترن».**
//
// # فصارت الحلقةُ تسأل لا تفترض
//
// **تفحص الحالَ كلَّ حين**: موصولٌ ومسجَّلٌ فتنام، وإلّا تصل من جديد.
// **ولا تقف عند حالٍ ظنّتها دائمة** — والشبكةُ في الرقّة تنقطع، وهذا ما
// سيقع لا ما قد يقع.
func (s *WhatsAppSender) connectLoop(ctx context.Context) {
	backoff := 5 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if s.client.IsConnected() && s.client.IsLoggedIn() {
			// **موصولٌ ومسجَّل** — يُفحص كلَّ نصف دقيقة، **والفحصُ أرخصُ من
			// بوتٍ صامتٍ لا يعلم أحدٌ أنّه سقط.**
			s.setPaired(s.client.Store.ID.User)
			s.setErr("")
			backoff = 5 * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
			continue
		}

		if err := s.connectOnce(ctx); err != nil {
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

// connectOnce **محاولةٌ واحدة — تعود دائماً ولا تحبس نفسَها.**
func (s *WhatsAppSender) connectOnce(ctx context.Context) error {
	// **وجهازٌ بلا هويّةٍ يحتاج اقتراناً** — والقناةُ تُطلب قبل الاتصال.
	if s.client.Store.ID == nil {
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
				s.logger.Info("whatsapp paired successfully", "as", s.client.Store.ID.User)
				// **ولا تُنهي الحلقةَ هنا** — الاقترانُ ليس اتّصالاً:
				// **تعود فتفحص الحالَ وتُكمل ما نقص.**
				return nil
			default:
				s.setQR("")
				return fmt.Errorf("whatsapp pairing ended: %s", evt.Event)
			}
		}
		return fmt.Errorf("whatsapp qr channel closed")
	}

	// **ومقترنٌ غيرُ موصولٍ يُوصَل** — ولا ينتظر شيئاً بعدها.
	if !s.client.IsConnected() {
		if err := s.client.Connect(); err != nil {
			return err
		}
	}
	if s.client.Store.ID != nil {
		s.setPaired(s.client.Store.ID.User)
		s.logger.Info("whatsapp connected", "as", s.client.Store.ID.User)
	}
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

// SendText يُرسل رسالةً حرّة — يستعمله إرسالُ الطلب إلى المتجر.
//
// **ونفسُ حارس `SendOTP`**: بوتٌ غيرُ مقترن يرمي خطأً واضحاً بدل أن يبتلع
// الرسالة صامتاً. ورسالةٌ ابتُلعت أسوأ من رسالةٍ لم تُرسَل: الأولى يظنّ صاحبُها
// أنها وصلت.
func (s *WhatsAppSender) SendText(ctx context.Context, phone, text string) error {
	if !s.client.IsLoggedIn() {
		return errWANotReady
	}
	jid := types.NewJID(strings.TrimPrefix(phone, "+"), types.DefaultUserServer)
	if _, err := s.client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	}); err != nil {
		return fmt.Errorf("whatsapp: send text: %w", err)
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

// Unpair **يفكّ الاقتران فيعود رمزُ الربط.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «إذا تمّ فصلُ الاقتران لا يوجد زرٌّ لإعادة ربط
//
//	الجهاز».)
//
// # ولماذا لم يكن يعود من تلقاء نفسه
//
// **الرمزُ لا يُولَّد إلّا لجهازٍ بلا هويّة.** ومن فُصل من هاتفه تبقى هويّتُه
// مخزَّنةً في القاعدة — **فتحاول الحلقةُ الاتّصالَ بها إلى الأبد ولا تعرض
// رمزاً**، ولوحةُ الإدارة تقول «غير مقترن» بلا سبيلٍ إلى الاقتران.
//
// **فبابان يُفتحان**: هذا — يضغطه المالكُ متى شاء — **وحدثُ الخروج** الذي
// يفكّه آليّاً حين يفصله واتساب.
func (s *WhatsAppSender) Unpair(ctx context.Context) error {
	if s.client.IsConnected() {
		s.client.Disconnect()
	}
	// ══════════════════════════════════════════════════════════════════
	// **وتُمحى من القاعدة ومن الذاكرة معاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومحوُها من القاعدة وحدَها لا يكفي**: العميلُ في الذاكرة يبقى يحمل
	// هويّتَه، **فتقول الحلقةُ «مقترنٌ» وتحاول الاتّصالَ إلى الأبد ولا تعرض
	// رمزاً** — والزرُّ يردّ «تمّ» ولا يتبدّل شيء.
	//
	// **وقِيس فعلاً** (٢٠٢٦-٠٨-١٠): بعد الفكّ صار الصفُّ صفراً في القاعدة
	// **و`logged_in` ما زالت صادقةً ولا رمز.**
	if s.client.Store.ID != nil {
		if err := s.client.Store.Delete(ctx); err != nil {
			return err
		}
		s.client.Store.ID = nil
	}
	s.setPaired("")
	s.setQR("")
	s.setErr("")
	s.logger.Info("whatsapp unpaired — awaiting a new QR scan")
	return nil
}

// watchLogout **يفكّ الاقتران حين يفصله واتساب.**
//
// **ومن فُصل من هاتفه لا يعرف أنّ عليه أن يضغط زرّاً** — يفتح اللوحةَ فيجد
// «غير مقترن» ولا رمز. **فيُفكّ آليّاً فيظهر الرمزُ وحدَه.**
func (s *WhatsAppSender) watchLogout(ctx context.Context) {
	s.client.AddEventHandler(func(evt any) {
		if _, ok := evt.(*events.LoggedOut); ok {
			s.logger.Warn("whatsapp logged out from the phone — re-pairing needed")
			_ = s.Unpair(ctx)
		}
	})
}
