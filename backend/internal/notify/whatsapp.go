package notify

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	// ══════════════════════════════════════════════════════════════════
	// **والعميلُ يُستبدَل — فيُقرأ بقفل**
	// ══════════════════════════════════════════════════════════════════
	//
	// **جهازٌ مُحي لا يُحيا** (`store.ErrDeviceDeleted`) — فبعد كلّ فكِّ
	// اقترانٍ يُبنى عميلٌ جديدٌ على جهازٍ جديد. **والحلقةُ تقرؤه في خيطٍ
	// آخر**، فالتبديلُ بلا قفلٍ سباقٌ على مؤشّر.
	clientMu sync.RWMutex
	client   *whatsmeow.Client
	// clientCtx **سياقُ هذا الجيل** — يُلغى حين يُستبدَل العميل.
	clientCtx context.Context
	// clientOff **يُلغيه** — وهو ما يُخرج الحلقةَ من قناةِ جهازٍ مات.
	clientOff context.CancelFunc
	// container **بيتُ الأجهزة** — منه يُولد الجهازُ البديل.
	container *sqlstore.Container
	// ══════════════════════════════════════════════════════════════════
	// **baseCtx — عمرُ الخادم لا عمرُ الطلب**
	// ══════════════════════════════════════════════════════════════════
	//
	// **`Unpair` تُنادى من معالِج HTTP بسياق الطلب** — ولو عُلّق الجيلُ
	// الجديدُ عليه **لَمات لحظةَ انتهاء الردّ.** فتُغلق قناةُ الرموز قبل أن
	// تُفتح، **ولا يظهر رمزٌ أبداً ولا خطأَ يقول لماذا**: اللوحةُ تقول «غير
	// مقترن» وحقلُ الخطأ فارغ.
	//
	// **وقِيس فعلاً** (٢٠٢٦-٠٨-١٠): ستّون ثانيةً بلا رمزٍ بعد كلّ فكّ.
	//
	// **فحياةُ البوت من حياة الخادم** — ومن الطلب يُؤخذ الإلغاءُ لا العمر.
	baseCtx context.Context

	// wake **يوقظ الحلقةَ الآن** — ولا تنتظر بقيّةَ مهلتها.
	wake chan struct{}

	// ══════════════════════════════════════════════════════════════════
	// **pairWanted — ولا رمزَ إلّا بطلب**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الكودُ لا يجب أن يظهر تلقائيّاً — هذا سببُ
	//  رسائل الخطأ. يظهر حين أطلبه بزرّ ربطِ جهاز».)
	//
	// **وكان يُولَّد وحدَه أبداً**: تعرض المكتبةُ ستّةَ رموزٍ في دقيقتين ثمّ
	// تقول `timeout` لمن لم يمسح، **فتعيد الحلقةُ الكرّةَ فوراً** — رمزٌ
	// ينتهي كلَّ دقيقتين وخطأٌ يُسجَّل معه، **ليلاً ونهاراً ولا أحدَ طلبه.**
	//
	// **وذاك أصلُ «pairing ended: timeout»** الذي سأل عنه المالك مرّتين:
	// **لم يكن عطباً، كان أثرَ رمزٍ لم يطلبه أحد.**
	//
	// **ورمزٌ يُعرض بلا طلبٍ خطرٌ أيضاً** — يبقى على شاشةٍ مفتوحةٍ فيمسحه
	// من مرّ بها فيربط هاتفَه هو بالمنصّة.
	pairWanted atomic.Bool
	// qrExpired **انتهى آخرُ رمزٍ ولم يُمسح** — تُعرض دعوةٌ لا إنذار.
	qrExpired atomic.Bool

	logger   *slog.Logger
	template TemplateFunc

	// delay **تمهّلٌ قبل كلّ رسالة** — يُقرأ من الإعدادات عند الإقلاع.
	delay func(context.Context) time.Duration
	// last **وقتُ آخر رسالةٍ خرجت** — التمهّلُ بينها وبين التالية.
	sendMu sync.Mutex
	last   time.Time

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
		container: container,
		baseCtx:   ctx,
		wake:      make(chan struct{}, 1),
		logger:    logger,
		template:  template,
	}
	s.adopt(device)

	// الاتصال بالخلفية مع إعادة محاولة — انقطاع واتساب لا يمنع إقلاع الخادم؛
	// طلبات OTP تفشل بخطأ واضح حتى يعود الاتصال (PLAN.md §6.8).
	go s.connectLoop(ctx)
	return s, nil
}

// cli **العميلُ الحاليُّ وسياقُه** — ومن أمسك القديمَ بعد فكٍّ أمسك جثّة.
func (s *WhatsAppSender) cli() (*whatsmeow.Client, context.Context) {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.client, s.clientCtx
}

// adopt **يتبنّى جهازاً: عميلٌ جديدٌ عليه، ومُراقبُ خروجٍ معه، وجيلٌ يُلغى.**
//
// **والمُراقبُ يُركَّب على العميل لا على المُرسِل** — فعميلٌ جديدٌ بلا مراقبٍ
// **لا يسمع خروجاً من الهاتف بعد أوّل إعادةِ بناء**، فيبقى مقترناً في اللوحة
// وقد فُصل من الهاتف.
//
// ══════════════════════════════════════════════════════════════════════
// **والجيلُ القديمُ يُلغى — وإلّا بقيت الحلقةُ فيه**
// ══════════════════════════════════════════════════════════════════════
//
// **الحلقةُ تقف عند `range qrChan`** تنتظر رمزاً للجهاز القديم. **وتبديلُ
// المؤشّر لا يوقظها**: تبقى في قناةٍ ماتت حتّى تنتهي دورةُ الرموز — **نحوَ
// دقيقتين**، ثمّ مهلةُ تراجعٍ فوقها قد تبلغ دقيقتين أُخريين.
//
// **وقِيس فعلاً** (٢٠٢٦-٠٨-١٠): فُكّ الاقترانُ **فلم يظهر رمزٌ ستّين ثانية**
// — والمالكُ يضغط «ربط الجهاز» فيرى فراغاً، فيضغط ثانيةً وثالثة.
//
// **فيُلغى سياقُ الجيل**: تُغلق المكتبةُ القناةَ، **فتعود الحلقةُ فوراً**
// وتقرأ العميلَ الجديد.
func (s *WhatsAppSender) adopt(device *store.Device) {
	ctx, cancel := context.WithCancel(s.baseCtx)
	c := whatsmeow.NewClient(device, waLog.Noop)
	c.AddEventHandler(func(evt any) {
		if _, ok := evt.(*events.LoggedOut); ok {
			s.logger.Warn("whatsapp logged out from the phone — re-pairing needed")
			_ = s.Unpair(s.baseCtx)
		}
	})
	s.clientMu.Lock()
	off := s.clientOff
	s.client, s.clientCtx, s.clientOff = c, ctx, cancel
	s.clientMu.Unlock()
	// **ويُلغى خارجَ القفل** — الإلغاءُ يوقظ الحلقةَ، **وحلقةٌ تستيقظ فتطلب
	// القفلَ الذي نحمله تقف عليه.**
	if off != nil {
		off()
	}
	s.nudge()
}

// Pair **يطلب رمزاً — وهو البابُ الوحيدُ إليه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «أضغط ربطَ الجهاز ليظهر الكود — هيك الأصول».)
//
// **ولا ينتظر شيئاً**: يرفع الرايةَ ويوقظ الحلقة، **والرمزُ يظهر في اللوحة
// بعد ثانيتين** — ومعالِجُ HTTP لا يُحبَس على شبكةِ واتساب.
func (s *WhatsAppSender) Pair() {
	s.pairWanted.Store(true)
	s.qrExpired.Store(false)
	s.setErr("")
	s.nudge()
}

// nudge **يوقظ الحلقةَ الآن** — ولا تُترك تُكمل مهلةَ تراجعٍ بلغت دقيقتين.
//
// **والقناةُ بسعةِ واحد**: إيقاظان متتاليان إيقاظٌ واحد، **ولا يُحبس مَن
// أيقظ إن كانت الحلقةُ مشغولة.**
func (s *WhatsAppSender) nudge() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
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
		c, _ := s.cli()
		if c.IsConnected() && c.IsLoggedIn() && c.Store.ID != nil {
			// **موصولٌ ومسجَّل** — يُفحص كلَّ نصف دقيقة، **والفحصُ أرخصُ من
			// بوتٍ صامتٍ لا يعلم أحدٌ أنّه سقط.**
			s.setPaired(c.Store.ID.User)
			s.setErr("")
			backoff = 5 * time.Second
			select {
			case <-ctx.Done():
				return
			case <-s.wake:
			case <-time.After(30 * time.Second):
			}
			continue
		}

		// ══════════════════════════════════════════════════════════════
		// **وجهازٌ بلا هويّةٍ ينتظر طلبَ صاحبه — لا يبادر**
		// ══════════════════════════════════════════════════════════════
		//
		// **ولا اتّصالَ أصلاً قبل الطلب**: الوصلُ بخوادم واتساب لجهازٍ بلا
		// هويّةٍ لا يفعل شيئاً إلّا أن يجلب رموزاً تنتهي. **فتقول اللوحةُ
		// «غير متّصل · غير مقترن»** — وهو الصدقُ بعينه، **وقد سأل المالكُ
		// عن تناقض «متّصلٌ وغيرُ مقترن» فكان هذا نصفَ جوابه.**
		if c.Store.ID == nil && !s.pairWanted.Load() {
			if c.IsConnected() {
				c.Disconnect()
			}
			select {
			case <-ctx.Done():
				return
			case <-s.wake:
			case <-time.After(30 * time.Second):
			}
			continue
		}

		if err := s.connectOnce(); err != nil {
			// **وإلغاءُ الجيل ليس عطباً** — فكَّ المالكُ الاقترانَ فأُغلقت
			// قناةُ الجهاز القديم. **وخطأٌ يُعرض على أنّه عطبٌ في لوحةٍ
			// يُقلق صاحبَه بلا سبب** — وقد سأل عن واحدٍ منها بالفعل.
			if errors.Is(err, errQRExpired) {
				// **ولا يُخلَف رمزٌ انتهى إلّا بطلبٍ جديد** — تُطفأ الرايةُ
				// **فتعود الحلقةُ إلى انتظارها.**
				s.pairWanted.Store(false)
				s.qrExpired.Store(true)
				backoff = 5 * time.Second
			} else if ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				s.setErr(err.Error())
				s.logger.Warn("whatsapp connect failed, retrying", "error", err, "backoff", backoff)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
			// **ومن أُيقظ لا يُتابع تراجعَه** — الحالُ تبدّلت، **والمهلةُ
			// كانت لعطبٍ لم يعد قائماً.**
			backoff = 5 * time.Second
			continue
		case <-time.After(backoff):
		}
		if backoff < 2*time.Minute {
			backoff *= 2
		}
	}
}

// errQRExpired **رمزٌ انتهت صلاحيّتُه ولم يُمسح** — حالٌ لا عطب.
//
// **ويُميَّز بقيمةٍ لا بنصّ**: الحلقةُ تقرّر أتعرضه أم تبتلعه، **ومقارنةُ
// نصوصٍ تنكسر بأوّل تبديلِ صياغة.**
var errQRExpired = errors.New("whatsapp: qr expired unscanned")

// connectOnce **محاولةٌ واحدة — تعود دائماً ولا تحبس نفسَها.**
//
// **وسياقُها سياقُ الجيل لا سياقُ الخادم** — فقناةُ الرموز تُغلق حين يُستبدَل
// العميل، **ولا تبقى الحلقةُ تنتظر رمزاً لجهازٍ حُذف.**
func (s *WhatsAppSender) connectOnce() error {
	c, ctx := s.cli()
	// **وجهازٌ بلا هويّةٍ يحتاج اقتراناً** — والقناةُ تُطلب قبل الاتصال.
	if c.Store.ID == nil {
		qrChan, err := c.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		if err := c.Connect(); err != nil {
			return err
		}
		// ══════════════════════════════════════════════════════════════
		// **ولا تُترك الحلقةُ رهينةَ قناةٍ لا نملك إغلاقَها**
		// ══════════════════════════════════════════════════════════════
		//
		// **`range` على قناةِ المكتبة ينتظر إغلاقَها — وقد لا تُغلق أبداً.**
		// قرأتُ مصدرَها (٢٠٢٦-٠٨-١٠): باعثُ الرموز يخرج صامتاً على
		// `expectedDisconnect` **بلا `close`** — ونحن نفصل العميلَ بأنفسنا
		// في `Unpair`، **فيموت الباعثُ وتبقى القناةُ مفتوحةً إلى الأبد.**
		//
		// **وقِيست فعلاً مرّتين**: فُكّ الاقترانُ فبقيت الحلقةُ واقفةً
		// ستّين ثانيةً بلا رمزٍ ولا خطأ — **والمالكُ يضغط الزرَّ ويرى فراغاً.**
		//
		// **فيُنتظر الاثنان معاً**: رمزٌ يأتي أو سياقٌ يُلغى. **وحلقةٌ لا
		// تملك خروجَها ليست حلقة.**
		for {
			var evt whatsmeow.QRChannelItem
			var ok bool
			select {
			case <-ctx.Done():
				s.setQR("")
				return ctx.Err()
			case evt, ok = <-qrChan:
				if !ok {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					return fmt.Errorf("whatsapp qr channel closed")
				}
			}
			switch evt.Event {
			case "code":
				s.setQR(evt.Code)
				// **ورمزٌ جديدٌ يمحو خطأ ما قبله.**
				//
				// (سؤالُ المالك ٢٠٢٦-٠٨-١٠: «ما زال يظهر آخرُ خطأ، لماذا؟»)
				//
				// **الحقلُ كان يُمحى في موضعين فقط**: نجاحُ الاقتران وفكُّه.
				// **فيبقى خطأُ محاولةٍ ماتت معروضاً فوق رمزٍ حيٍّ صالح** —
				// ومن رآه ظنّ الرمزَ عاطلاً فلم يمسحه، **فانتهى هو أيضاً
				// فظهر خطؤه**، وهكذا.
				//
				// **والحقلُ اسمُه «آخر خطأ» ويُقرأ «الخطأ الآن»** — ولا أحدَ
				// يقرأ صفةً في اسمِ حقل.
				s.setErr("")
				s.logger.Info("whatsapp pairing: scan this QR from the bot phone (WhatsApp > Linked Devices)")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			case "success":
				s.setQR("")
				// **والرايةُ تُطفأ بالنجاح** — وإلّا طلبت الحلقةُ رمزاً
				// لجهازٍ صار له هويّة.
				s.pairWanted.Store(false)
				s.logger.Info("whatsapp paired successfully", "as", c.Store.ID.User)
				// **ولا تُنهي الحلقةَ هنا** — الاقترانُ ليس اتّصالاً:
				// **تعود فتفحص الحالَ وتُكمل ما نقص.**
				return nil
			case "timeout":
				// ══════════════════════════════════════════════════════
				// **وانتهاءُ صلاحيّةِ رمزٍ ليس عطباً**
				// ══════════════════════════════════════════════════════
				//
				// **الرموزُ تنتهي بطبعها**: تعرض المكتبةُ ستّةً يتعاقبون في
				// نحوِ دقيقتين، **ثمّ تقول `timeout` لمن لم يمسح.** وهذا هو
				// حالُ كلِّ من فتح اللوحةَ وتركها.
				//
				// **وعرضُه «آخر خطأ» يقول للمالك إنّ في منصّته عطباً وليس
				// فيها عطب** — سأل عنه مرّتين.
				//
				// **فيُقال في السجلّ ولا يُعرض**: الحلقةُ تطلب رمزاً جديداً
				// في ثوانٍ، **وما يُصلح نفسَه لا يُنذَر به.**
				s.setQR("")
				s.logger.Info("whatsapp qr expired unscanned — requesting a new one")
				return errQRExpired
			default:
				// **وكلُّ نهايةٍ تُطفئ الراية** — **ومحاولةٌ تُعاد بلا طلبٍ
				// تُعيد الخطأَ نفسَه إلى الأبد** فيُقرأ عطباً مستمرّاً.
				s.setQR("")
				s.pairWanted.Store(false)
				return fmt.Errorf("whatsapp pairing ended: %s", evt.Event)
			}
		}
	}

	// **ومقترنٌ غيرُ موصولٍ يُوصَل** — ولا ينتظر شيئاً بعدها.
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return err
		}
	}
	if c.Store.ID != nil {
		s.setPaired(c.Store.ID.User)
		s.logger.Info("whatsapp connected", "as", c.Store.ID.User)
	}
	return nil
}

var errWANotReady = fmt.Errorf("whatsapp: bot not connected/paired yet")

func (s *WhatsAppSender) SendOTP(ctx context.Context, phone, code string) error {
	c, _ := s.cli()
	if !c.IsLoggedIn() {
		return errWANotReady
	}
	s.pace(ctx)
	jid := types.NewJID(strings.TrimPrefix(phone, "+"), types.DefaultUserServer)
	text := s.template(ctx, code)
	_, err := c.SendMessage(ctx, jid, &waE2E.Message{
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
	c, _ := s.cli()
	if !c.IsLoggedIn() {
		return errWANotReady
	}
	s.pace(ctx)
	jid := types.NewJID(strings.TrimPrefix(phone, "+"), types.DefaultUserServer)
	if _, err := c.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	}); err != nil {
		return fmt.Errorf("whatsapp: send text: %w", err)
	}
	return nil
}

// Status حالة البوت — تعرضها نقطة أدمن لمتابعة الاقتران والاتصال.
func (s *WhatsAppSender) Status() map[string]any {
	c, _ := s.cli()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"provider":  "whatsapp",
		"connected": c.IsConnected(),
		"logged_in": c.IsLoggedIn(),
		// **وحالُ الربط تُقال للوحة**: أطُلب رمزٌ الآن؟ وهل انتهى آخرُه؟
		// **وزرٌّ لا يعرف حالَه يُضغط مرّتين** — والثانيةُ تُبطل الأولى.
		"pair_wanted": s.pairWanted.Load(),
		"qr_expired":  s.qrExpired.Load(),
		"paired_as":   s.pairedAs,
		"qr":          s.qr,
		"last_error":  s.lastErr,
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
	// ══════════════════════════════════════════════════════════════════
	// **والخروجُ من المنصّة خروجٌ من الهاتف أيضاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «عند تسجيل الخروج من الهاتف يجب تسجيلُ
	//  الخروج من المنصّة، وإذا تمّ تسجيلُ الخروج من المنصّة يجب أن يتمّ
	//  تسجيلُ الخروج من الهاتف أيضاً».)
	//
	// **وفصلٌ محلّيٌّ وحدَه يترك الجهازَ معلَّقاً في هاتفه** — يراه في
	// «الأجهزة المرتبطة» ويظنّه يعمل، **ويتراكم واحدٌ في كلّ مرّة.**
	//
	// **و`Logout` تُبلّغ واتساب فيُزال الجهازُ من القائمة** — ثمّ تُمحى
	// الهويّةُ محليّاً. **وإن تعذّر الإبلاغُ (لا شبكة) يمضي الفكُّ محليّاً**:
	// **مالكٌ لا يستطيع أن يفكّ اقترانَه لأنّ الشبكةَ انقطعت عالقٌ.**
	c, _ := s.cli()
	if c.IsLoggedIn() {
		if err := c.Logout(ctx); err != nil {
			s.logger.Warn("whatsapp: remote logout failed — unpairing locally", "error", err)
		}
	}
	if c.IsConnected() {
		c.Disconnect()
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
	//
	// ══════════════════════════════════════════════════════════════════
	// **وجهازٌ مُحي لا يُحيا — فيُبنى بديلُه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **`Store.Delete` تَسِمُ الجهازَ ميّتاً في الذاكرة** (`ErrDeviceDeleted`
	// في المكتبة) — **وتصفيرُ `Store.ID` لا يرفع الوسم.** فكلُّ محاولةِ
	// اتّصالٍ بعدها تُردّ بـ«invalid use of deleted device»، **والحلقةُ
	// تعيدها كلَّ دقيقتين إلى الأبد ولا رمزَ يظهر.**
	//
	// **وقِيس فعلاً** (٢٠٢٦-٠٨-١٠، بعد الإصلاح الأوّل): فُكّ الاقترانُ
	// **فمات البوتُ حتّى أُعيد تشغيلُ الخادم** — واللوحةُ تقول «متّصل» و«غير
	// مقترن» وزرُّ الربط يعرض فراغاً.
	//
	// **فيُولَد جهازٌ جديدٌ من الحاوية وعميلٌ جديدٌ عليه** — وهو بلا هويّة،
	// **فتطلب الحلقةُ له رمزاً في أوّل دورة.** ولا إعادةَ تشغيلٍ لأحد.
	if c.Store.ID != nil {
		if err := c.Store.Delete(ctx); err != nil {
			return err
		}
	}
	s.adopt(s.container.NewDevice())
	// **ولا رمزَ بعد الفكّ حتّى يُطلب** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	s.pairWanted.Store(false)
	s.qrExpired.Store(false)
	s.setPaired("")
	s.setQR("")
	s.setErr("")
	s.logger.Info("whatsapp unpaired — awaiting a new QR scan")
	return nil
}

// **ومُراقبُ الخروج صار في `adopt`** — لأنّه يُركَّب على كلّ عميلٍ يُولَد،
// **لا على أوّلِ عميلٍ وحدَه.**

// pace **يُمهل بين رسالةٍ وأخرى — لأنّ الرقمَ يُحظر لا الخادم.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «جهّز رسائلَ بشريّةً بحيث تخفّف عمليّةَ الحظر».)
//
// **ورقمٌ يُرسل عشرين رسالةً في الدقيقة بلا توقّفٍ يُقرأ آلة** — والإنسانُ
// يكتب ثمّ يتوقّف.
//
// **ولا أَعِدُ أنّ هذا يمنع الحظر**: لا أحدَ يعرف ما يفحصه واتساب، **ومن
// وعد بذلك خمّن.** إنّما يُقلّل النمطَ الآليَّ الظاهر.
//
// **والتمهّلُ بين الرسائل لا قبل كلّ واحدة**: من أرسل رسالةً وحيدةً بعد ساعةٍ
// لا ينتظر شيئاً — **وتأخيرُ رمزِ تحقّقٍ بلا سببٍ يجعل صاحبَه يطلبه ثانيةً**،
// فتصير رسالتان مكان واحدة.
func (s *WhatsAppSender) pace(ctx context.Context) {
	if s.delay == nil {
		return
	}
	d := s.delay(ctx)
	if d <= 0 {
		return
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if wait := d - time.Since(s.last); wait > 0 && !s.last.IsZero() {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
	s.last = time.Now()
}

// SetDelay **يربط التمهّلَ بالإعدادات** — ويُقرأ عند كلّ إرسالٍ لا مرّةً
// عند الإقلاع: **رقمٌ يُبدَّل في اللوحة يجب أن يعمل بلا إعادة تشغيل.**
func (s *WhatsAppSender) SetDelay(f func(context.Context) time.Duration) { s.delay = f }
