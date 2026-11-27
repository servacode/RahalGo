package server

// ══════════════════════════════════════════════════════════════════════
// **قائمة بوت الواتساب** (قرارُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// «أي رسالة توصل لرقمنا يكون في خيارات… نحوّل الرقم من محادثات عادية إلى محادثات منظّمة كشركة
// احترافية». **ستّةُ خياراتٍ بأرقام** (whatsmeow لا يدعم أزرارَ واتساب الرسميّة):
//
//	١ طلبٌ جديدٌ عبر الواتساب   ← نصُّ الطلب ← تعديلٌ؟ (نعم/لا، ويتكرّر) ← المتجر ← الموقع ⇒ طلبٌ معلّق
//	٢ متابعةُ طلبي             ← رقمُ الطلب ⇒ حالُه (لصاحبه وحدَه)
//	٣ محادثةٌ مع الإدارة        ← البوتُ يسكت، وإشعارٌ للموظّفين
//	٤ استعادةُ كلمة المرور      ← رمزٌ بالمسار الحقيقيّ (`RequestPasswordReset`)
//	٥ تحميلُ التطبيق           ← ملفُّ تطبيق الزبون (مسار «ارسل رحال غو»)
//	٦ التسجيلُ ككابتن           ← تسعةُ أسئلةٍ واحداً واحداً ⇒ `captain_applications`
//
// # متى تظهر القائمة
//
// **أوّلُ رسالةٍ للشخص، أو أوّلُ رسالةٍ بعد سكوتٍ** (`whatsapp.bot_menu_silence_min`، قرارُه: ساعة)
// **في الاتّجاهين، أو «0» أو «قائمة».** وغيرُ ذلك لا تقاطع — ولا حين يحادثه موظّف. **وخارجَ دوام المنصّة**
// رسالةُ «انتهى دوامنا» بموعد الفتح مكانَ القائمة، بالقيد نفسِه.
//
// # لمن
//
// **لكلّ رقمٍ عدا الكباتن والمندوبين والمتاجر** (أصحابِها وموظّفيها). والمساراتُ القائمة تسبقها
// (`notify/wa_menu.go`).
//
// # والحالُ في القاعدة
//
// `wa_bot_chats` — فإعادةُ تشغيل الخادم لا تقطع طلباً بنصفه. **وكلُّ رسالةٍ تقفل صفَّ رقمها**
// (`pg_advisory_xact_lock`) فرسالتان متلاحقتان لا تتسابقان.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ── نصوصُ البوت (فصحى مبسّطة — قرارُ المالك) ─────────────────────────────

const (
	waBotMenuText = "أهلاً بك في رحّال غو 👋\nيرجى اختيار رقم الخدمة:\n" +
		"1️⃣ طلب جديد عبر الواتساب\n2️⃣ متابعة طلبي\n3️⃣ محادثة مع الإدارة\n" +
		"4️⃣ استعادة كلمة المرور\n5️⃣ تحميل تطبيق رحّال غو\n6️⃣ التسجيل ككابتن"

	waBotUnclear     = "عذراً، لم أفهم ردّك. يرجى كتابة الجواب بشكل واضح لأتمكّن من مساعدتك."
	waBotAskMenu     = "يرجى إرسال رقم الخدمة من 1 إلى 6، أو 0 لعرض القائمة."
	waBotAskYesNo    = "يرجى الإجابة بـ: نعم أو لا."
	waBotStuck       = "يمكنك كتابة 0 للعودة إلى القائمة، أو 3 للتحدّث مع الإدارة."
	waBotReminder    = "هل ما زلت معنا؟ يرجى إكمال الإجابة لنتابع طلبك."
	waBotWriteOrder  = "عذراً، يرجى كتابة الطلب كتابةً."
	waBotWriteAnswer = "عذراً، يرجى كتابة الجواب كتابةً."

	waBotOrderAsk     = "يرجى كتابة طلبك كاملاً في رسالة مكتوبة، ونرجو عدم إرسال رسائل صوتية.\n\nيرجى العلم أن الطلب الواحد يكون من محل واحد فقط. إذا كان طلبك من محلّين أو أكثر، يُحسب كل محل توصيلة منفصلة بأجرة خاصة."
	waBotOrderEditQ   = "هل لديك تعديل أو إضافة على الطلب؟ أجب بـ: نعم أو لا."
	waBotOrderEditAsk = "يرجى كتابة التعديل أو الإضافة كاملةً."
	waBotOrderStoreQ  = "هل تريد الطلب من متجر أو محل معيّن؟ اكتب اسمه، أو اكتب: لا يوجد."
	waBotOrderLocAsk  = "يرجى إرسال موقعك (العادي أو المباشر)."
	waBotOrderDone    = "شكراً لك، تم تسجيل طلبك برقم #%d، وسنتواصل معك قريباً."
	waBotOrderFailed  = "عذراً، تعذّر تسجيل طلبك الآن. سيتواصل معك أحد موظفينا في أقرب وقت."

	waBotTrackAsk      = "يرجى كتابة رقم طلبك."
	waBotTrackNotFound = "لم نجد طلباً بهذا الرقم مرتبطاً برقمك."

	waBotAdminText = "تفضّل، اكتب رسالتك وسيردّ عليك أحد موظفينا في أقرب وقت."

	waBotResetSent    = "أرسلنا إليك رمز استعادة كلمة المرور، أدخله في التطبيق."
	waBotResetNoAcct  = "هذا الرقم غير مسجّل لدينا. يمكنك إنشاء حساب من تطبيق رحّال غو."
	waBotResetLimited = "عذراً، طُلب الرمز عدّة مرّات. يرجى المحاولة بعد قليل."
	waBotResetFailed  = "عذراً، تعذّر إرسال الرمز الآن. يرجى المحاولة بعد قليل."

	waBotAppText    = "إليك تطبيق رحّال غو لأجهزة أندرويد."
	waBotAppPlay    = "ويمكنك تحميله من متجر Google Play عبر الرابط:\n%s"
	waBotAppMissing = "عذراً، ملف التطبيق غير متاح حالياً. سيتواصل معك أحد موظفينا في أقرب وقت."

	waBotSignupDone = "تم تسجيل جميع معلوماتك، وستتواصل معك الإدارة لاحقاً في الوقت المناسب."

	waBotClosedPrefix = "أهلاً بك في رحّال غو 🌙\n"
	waBotClosedLater  = "انتهى دوامنا لهذا اليوم، ونسعد بخدمتك %s من الساعة %s."
	waBotClosedToday  = "لم يبدأ دوامنا بعد، ونسعد بخدمتك اليوم من الساعة %s."
	waBotClosedNoTime = "نحن خارج أوقات الدوام حالياً، ونسعد بخدمتك في أقرب وقت."
)

// waBotSignupSteps **أسئلةُ التسجيل ككابتن بالترتيب** (قرارُ المالك) — و`yesno` تُتحقّق.
var waBotSignupSteps = []struct {
	key, ask string
	yesno    bool
}{
	{"name", "يرجى كتابة اسمك الكامل.", false},
	{"age", "كم عمرك؟", false},
	{"area", "ما المنطقة التي تسكن فيها؟", false},
	{"ebike", "هل لديك دراجة كهربائية؟ (نعم/لا)", true},
	{"motorbike", "هل لديك دراجة نارية؟ (نعم/لا)", true},
	{"license", "هل لديك رخصة قيادة؟ (نعم/لا)", true},
	{"android", "هل لديك جوال أندرويد؟ (نعم/لا)", true},
	{"experience", "هل عملت في التوصيل من قبل؟", false},
	{"hours", "ما الساعات التي تستطيع العمل فيها؟", false},
}

// waBotFlowReminder **التذكيرُ بعد ثلاث دقائق سكوتٍ وسطَ طلبٍ أو تسجيل، ثمّ الإسقاطُ بعد ثلاثٍ أخرى.**
const waBotFlowReminder = 3 * time.Minute

// waBotMaxDetails **حدُّ نصّ الطلب والتعديلات** — أطولُ من نموذج اللوحة: الزبونُ يكتب قائمته كلَّها.
const waBotMaxDetails = 2000

// ── الحال ─────────────────────────────────────────────────────────────

type waBotData struct {
	Text   string            `json:"text,omitempty"`
	Edits  []string          `json:"edits,omitempty"`
	Store  string            `json:"store,omitempty"`
	Signup map[string]string `json:"signup,omitempty"`
}

type waBotChat struct {
	phone    string
	state    string
	step     string
	data     waBotData
	unclear  int
	reminded bool
}

// waBotEffect **ما يقع بعد تثبيت الحال** — رسائلُ وإشعاراتٌ لا تُمسك قفلَ الصفّ.
type waBotEffect func(ctx context.Context, reply *notify.MenuReply)

// WABotMenuEnabled **مفتاحُ المالك.**
func (s *Server) WABotMenuEnabled(ctx context.Context) bool {
	return s.settings.GetBool(ctx, "whatsapp.bot_menu_enabled")
}

func (s *Server) waBotSilence(ctx context.Context) time.Duration {
	m := s.settings.GetInt(ctx, "whatsapp.bot_menu_silence_min")
	if m <= 0 {
		m = 60
	}
	return time.Duration(m) * time.Minute
}

// waBotExcluded **كابتنٌ أو مندوبٌ أو متجر (صاحبُه أو موظّفُه)** — لا قائمةَ لهم (قرارُ المالك).
func (s *Server) waBotExcluded(ctx context.Context, phone string) bool {
	var ex bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id = u.id
		                WHERE (u.phone = $1 OR u.whatsapp_phone = $1)
		                  AND ur.role_code IN ('driver', 'sales', 'merchant'))
		    OR EXISTS (SELECT 1 FROM users u JOIN merchants m ON m.owner_user_id = u.id
		                WHERE u.phone = $1 OR u.whatsapp_phone = $1)
		    OR EXISTS (SELECT 1 FROM merchant_staff st WHERE st.phone = $1)`, phone).Scan(&ex); err != nil {
		// **والشكُّ صمت** — رسالةٌ لكابتنٍ لا تُرسل خطأً أهونُ من قائمةٍ لكابتن.
		s.logger.Warn("قائمة البوت: فحصُ الدور سقط", "error", err)
		return true
	}
	return ex
}

// WABotTakesMedia **أهذا الرقمُ وسطَ طلبٍ أو تسجيلٍ في البوت؟** — فصورتُه أو صوتُه للقائمة لا لمعالِج
// أصناف المتاجر (`waitems`، وهو صامتٌ للغرباء أصلاً).
func (s *Server) WABotTakesMedia(ctx context.Context, from string) bool {
	if !s.WABotMenuEnabled(ctx) {
		return false
	}
	phone, ok := identity.NormalizePhone(from)
	if !ok {
		return false
	}
	var in bool
	_ = s.pg.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wa_bot_chats
		WHERE phone = $1 AND state IN ('order', 'signup', 'track', 'menu'))`, phone).Scan(&in)
	return in
}

// HandleWABotMenu **معالِجُ القائمة** — يُربط بالبوت (`notify.SetMenu`).
func (s *Server) HandleWABotMenu(ctx context.Context, in notify.MenuInbound) notify.MenuReply {
	if !s.WABotMenuEnabled(ctx) {
		return notify.MenuReply{}
	}
	phone, ok := identity.NormalizePhone(in.From)
	if !ok || s.waBotExcluded(ctx, phone) {
		return notify.MenuReply{}
	}
	// **نشاطٌ وحسب** — ردُّ موظّفٍ يُسكت القائمةَ المعروضة، وما عالجه غيرُها يُحسب نشاطاً.
	if in.Outgoing || in.Handled {
		st := "wa_bot_chats.state"
		if in.Outgoing {
			st = `CASE WHEN wa_bot_chats.state IN ('menu', 'closed', 'track') THEN 'admin' ELSE wa_bot_chats.state END`
		}
		initial := "idle"
		if in.Outgoing {
			initial = "admin"
		}
		if _, err := s.pg.Exec(ctx, `
			INSERT INTO wa_bot_chats (phone, state, last_activity_at) VALUES ($1, $2, now())
			ON CONFLICT (phone) DO UPDATE SET last_activity_at = now(), updated_at = now(),
			       state = `+st, phone, initial); err != nil {
			s.logger.Warn("قائمة البوت: تسجيلُ النشاط سقط", "error", err)
		}
		return notify.MenuReply{}
	}

	var reply notify.MenuReply
	var effects []waBotEffect
	err := pgx.BeginFunc(ctx, s.pg, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('wa_bot:' || $1))`, phone); err != nil {
			return err
		}
		c := waBotChat{phone: phone, state: "idle"}
		var raw []byte
		var last *time.Time
		err := tx.QueryRow(ctx, `
			SELECT state, step, data, unclear, reminded_at IS NOT NULL, last_activity_at
			  FROM wa_bot_chats WHERE phone = $1`, phone).
			Scan(&c.state, &c.step, &raw, &c.unclear, &c.reminded, &last)
		fresh := false
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			fresh = true
		case err != nil:
			return err
		default:
			_ = json.Unmarshal(raw, &c.data)
			fresh = last == nil || time.Since(*last) >= s.waBotSilence(ctx)
		}
		prevStep := c.state + "/" + c.step
		reply, effects = s.waBotStep(ctx, tx, &c, in, fresh)
		// **السؤالُ الجاري يُؤقَّت من جديد** متى تبدّل أو أجاب عليه — والتذكيرُ يُصفَّر.
		stepChanged := prevStep != c.state+"/"+c.step || reply.Text != ""
		data, _ := json.Marshal(c.data)
		_, err = tx.Exec(ctx, `
			INSERT INTO wa_bot_chats (phone, state, step, data, unclear, step_at, reminded_at, last_activity_at)
			VALUES ($1, $2, $3, $4, $5, now(), NULL, now())
			ON CONFLICT (phone) DO UPDATE SET state = $2, step = $3, data = $4, unclear = $5,
			       step_at = CASE WHEN $6 THEN now() ELSE wa_bot_chats.step_at END,
			       reminded_at = CASE WHEN $6 THEN NULL ELSE wa_bot_chats.reminded_at END,
			       last_activity_at = now(), updated_at = now()`,
			phone, c.state, c.step, data, c.unclear, stepChanged)
		return err
	})
	if err != nil {
		s.logger.Warn("قائمة البوت: المعالجةُ سقطت", "error", err)
		return notify.MenuReply{}
	}
	for _, e := range effects {
		e(ctx, &reply)
	}
	return reply
}

// waBotStep **آلةُ الحال** — تعدّل `c` وتردّ النصَّ والآثار.
func (s *Server) waBotStep(ctx context.Context, tx pgx.Tx, c *waBotChat, in notify.MenuInbound,
	fresh bool) (notify.MenuReply, []waBotEffect) {
	t := waBotNorm(in.Text)
	isText := in.Kind == notify.MenuText && strings.TrimSpace(in.Text) != ""
	if fresh || (isText && (t == "0" || t == "قائمة" || t == "القائمة")) {
		*c = waBotChat{phone: c.phone}
		if txt, closed := s.waBotClosedText(ctx); closed {
			c.state = "closed"
			return notify.MenuReply{Text: txt}, nil
		}
		c.state = "menu"
		return notify.MenuReply{Text: waBotMenuText}, nil
	}
	switch c.state {
	case "menu":
		return s.waBotMenuChoice(ctx, c, in, t, isText)
	case "order":
		return s.waBotOrderStep(c, in, t, isText)
	case "track":
		return s.waBotTrackStep(ctx, c, t, isText)
	case "signup":
		return s.waBotSignupStep(c, in, t, isText)
	}
	// idle · closed · admin · done ⇒ لا مقاطعة.
	return notify.MenuReply{}, nil
}

// unclear **ردٌّ غيرُ مفهوم** — توضيحٌ وإعادةُ السؤال مرّتين، ثمّ دلالةُ «0 أو 3» في الثالثة، ثمّ صمت.
func (c *waBotChat) unclearReply(lead, question string) notify.MenuReply {
	c.unclear++
	switch {
	case c.unclear <= 2:
		return notify.MenuReply{Text: lead + "\n" + question}
	case c.unclear == 3:
		return notify.MenuReply{Text: waBotStuck}
	}
	return notify.MenuReply{}
}

func (c *waBotChat) ask(state, step, text string) notify.MenuReply {
	c.state, c.step, c.unclear = state, step, 0
	return notify.MenuReply{Text: text}
}

// ── ١ … ٦ ─────────────────────────────────────────────────────────────

func (s *Server) waBotMenuChoice(ctx context.Context, c *waBotChat, in notify.MenuInbound, t string,
	isText bool) (notify.MenuReply, []waBotEffect) {
	if !isText {
		return c.unclearReply(waBotUnclear, waBotAskMenu), nil
	}
	switch t {
	case "1":
		c.data = waBotData{}
		return c.ask("order", "text", waBotOrderAsk), nil
	case "2":
		return c.ask("track", "number", waBotTrackAsk), nil
	case "3":
		phone := c.phone
		return c.ask("admin", "", waBotAdminText), []waBotEffect{func(ctx context.Context, _ *notify.MenuReply) {
			s.notify.NotifyCaps(ctx, []string{string(authz.OrdersWhatsApp)}, notifications.Input{
				Kind: notifications.KindOrder, Title: waBotNotifAdminTitle,
				Body: fmt.Sprintf(waBotNotifAdminBody, phone), Href: "/dashboard/whatsapp",
			})
		}}
	case "4":
		phone := c.phone
		r := c.ask("done", "", "")
		return r, []waBotEffect{func(ctx context.Context, reply *notify.MenuReply) {
			err := s.identity.RequestPasswordReset(ctx, phone, "")
			switch {
			case err == nil:
				reply.Text = waBotResetSent
			case errors.Is(err, identity.ErrNoAccount):
				reply.Text = waBotResetNoAcct
			case errors.Is(err, identity.ErrOTPRateLimited):
				reply.Text = waBotResetLimited
			default:
				s.logger.Warn("قائمة البوت: رمزُ الاستعادة سقط", "error", err)
				reply.Text = waBotResetFailed
			}
		}}
	case "5":
		r := c.ask("done", "", "")
		return r, []waBotEffect{func(ctx context.Context, reply *notify.MenuReply) {
			play := strings.TrimSpace(s.settings.GetString(ctx, "release.customer.play_url"))
			_, err := s.CustomerAppFile(ctx)
			hasFile := err == nil
			switch {
			case hasFile && play != "":
				reply.Text = waBotAppText + "\n" + fmt.Sprintf(waBotAppPlay, play)
			case hasFile:
				reply.Text = waBotAppText
			case play != "":
				reply.Text = fmt.Sprintf(waBotAppPlay, play)
			default:
				reply.Text = waBotAppMissing
			}
			reply.SendApp = hasFile
		}}
	case "6":
		c.data = waBotData{Signup: map[string]string{}}
		return c.ask("signup", waBotSignupSteps[0].key, waBotSignupSteps[0].ask), nil
	}
	return c.unclearReply(waBotUnclear, waBotAskMenu), nil
}

// waBotOrderStep **طلبٌ موجَّه**: النصّ ← تعديل؟ ← المتجر ← الموقع.
func (s *Server) waBotOrderStep(c *waBotChat, in notify.MenuInbound, t string,
	isText bool) (notify.MenuReply, []waBotEffect) {
	raw := strings.TrimSpace(in.Text)
	switch c.step {
	case "text", "edit_text":
		if !isText {
			q := waBotOrderAsk
			if c.step == "edit_text" {
				q = waBotOrderEditAsk
			}
			return c.unclearReply(waBotWriteOrder, q), nil
		}
		if c.step == "text" {
			c.data.Text = raw
		} else {
			c.data.Edits = append(c.data.Edits, raw)
		}
		return c.ask("order", "edit_q", waBotOrderEditQ), nil
	case "edit_q":
		switch waBotYesNo(t, isText) {
		case 1:
			return c.ask("order", "edit_text", waBotOrderEditAsk), nil
		case 0:
			return c.ask("order", "store", waBotOrderStoreQ), nil
		}
		return c.unclearReply(waBotUnclear, waBotAskYesNo), nil
	case "store":
		if !isText {
			return c.unclearReply(waBotWriteAnswer, waBotOrderStoreQ), nil
		}
		if waBotNoStore(t) {
			c.data.Store = ""
		} else {
			c.data.Store = raw
		}
		return c.ask("order", "location", waBotOrderLocAsk), nil
	case "location":
		if in.Kind != notify.MenuLocation || !orders.ValidPoint(in.Loc.Lat, in.Loc.Lng) {
			return c.unclearReply(waBotUnclear, waBotOrderLocAsk), nil
		}
		data, phone, lat, lng := c.data, c.phone, in.Loc.Lat, in.Loc.Lng
		r := c.ask("done", "", "")
		c.data = waBotData{}
		return r, []waBotEffect{func(ctx context.Context, reply *notify.MenuReply) {
			reply.Text = s.waBotCreateOrder(ctx, phone, data, lat, lng)
		}}
	}
	// خطوةٌ مجهولة (حالٌ قديمة) ⇒ من أوّل الطلب.
	return c.ask("order", "text", waBotOrderAsk), nil
}

// waBotDetails **نصُّ الطلب كما يُكتب على الطلب** — الأصلُ ثمّ التعديلاتُ بترتيبها.
func waBotDetails(d waBotData) string {
	var b strings.Builder
	b.WriteString(d.Text)
	for i, e := range d.Edits {
		fmt.Fprintf(&b, "\n\nتعديل %d: %s", i+1, e)
	}
	out := b.String()
	if r := []rune(out); len(r) > waBotMaxDetails {
		out = string(r[:waBotMaxDetails])
	}
	return out
}

// waBotCreateOrder **ينشئ الطلبَ المعلّق ويُشعر الموظّفين** — ويردّ نصَّ الزبون.
func (s *Server) waBotCreateOrder(ctx context.Context, phone string, d waBotData, lat, lng float64) string {
	c, err := s.orders.CreateWABotOrder(ctx, orders.WABotOrderInput{
		Phone: phone, Details: waBotDetails(d), StoreName: d.Store, Lat: lat, Lng: lng,
	})
	if err != nil {
		s.logger.Error("قائمة البوت: إنشاءُ الطلب سقط", "error", err)
		return waBotOrderFailed
	}
	s.notify.NotifyCaps(ctx, []string{string(authz.OrdersWhatsApp)}, notifications.Input{
		Kind: notifications.KindOrder, Title: waBotNotifOrderTitle,
		Body:   fmt.Sprintf(waBotNotifOrderBody, c.Number, phone),
		Entity: "order", EntityID: c.ID, Href: "/dashboard/whatsapp",
	})
	s.touch("orders", "ops")
	return fmt.Sprintf(waBotOrderDone, c.Number)
}

// waBotTrackStep **متابعةُ طلب** — برقمه، ولصاحبه وحدَه.
func (s *Server) waBotTrackStep(ctx context.Context, c *waBotChat, t string,
	isText bool) (notify.MenuReply, []waBotEffect) {
	n, ok := waBotOrderNumber(t)
	if !isText || !ok {
		return c.unclearReply(waBotUnclear, waBotTrackAsk), nil
	}
	status, found, err := s.orders.WABotTrack(ctx, c.phone, n)
	if err != nil {
		s.logger.Warn("قائمة البوت: قراءةُ الطلب سقطت", "error", err)
		return notify.MenuReply{}, nil
	}
	if !found {
		// **ويبقى في المتابعة** — قد يكون أخطأ في رقم.
		c.unclear = 0
		return notify.MenuReply{Text: waBotTrackNotFound}, nil
	}
	return c.ask("done", "", waBotStatusText(status, n)), nil
}

// waBotStatusText **حالُ الطلب بكلامٍ رسميّ.**
func waBotStatusText(status string, n int64) string {
	switch status {
	case orders.StPending:
		return fmt.Sprintf("طلبك رقم #%d قيد المراجعة لدى الإدارة.", n)
	case orders.StAccepted, orders.StPreparing:
		return fmt.Sprintf("طلبك رقم #%d مقبول وقيد التجهيز.", n)
	case orders.StDispatching:
		return fmt.Sprintf("طلبك رقم #%d مقبول، ونبحث له عن كابتن الآن.", n)
	case orders.StAssigned:
		return fmt.Sprintf("تم تعيين كابتن لطلبك رقم #%d، وهو في طريقه لاستلامه.", n)
	case orders.StAtPickup:
		return fmt.Sprintf("الكابتن الآن في مكان استلام طلبك رقم #%d.", n)
	case orders.StPickedUp, orders.StOnTheWay:
		return fmt.Sprintf("طلبك رقم #%d في الطريق إليك.", n)
	case orders.StAtDropoff:
		return fmt.Sprintf("وصل الكابتن بطلبك رقم #%d إلى عنوانك.", n)
	case orders.StDelivered:
		return fmt.Sprintf("تم تسليم طلبك رقم #%d. شكراً لك.", n)
	case orders.StRejected:
		return fmt.Sprintf("نعتذر، لم يتم قبول طلبك رقم #%d.", n)
	case orders.StCancelled:
		return fmt.Sprintf("طلبك رقم #%d ملغى.", n)
	case orders.StFailed:
		return fmt.Sprintf("نعتذر، تعذّر تسليم طلبك رقم #%d.", n)
	case orders.StRefunded:
		return fmt.Sprintf("تم استرجاع طلبك رقم #%d.", n)
	}
	return fmt.Sprintf("طلبك رقم #%d قيد المتابعة.", n)
}

// waBotSignupStep **التسجيلُ ككابتن** — سؤالٌ سؤال.
func (s *Server) waBotSignupStep(c *waBotChat, in notify.MenuInbound, t string,
	isText bool) (notify.MenuReply, []waBotEffect) {
	idx := -1
	for i, st := range waBotSignupSteps {
		if st.key == c.step {
			idx = i
			break
		}
	}
	if idx < 0 {
		c.data = waBotData{Signup: map[string]string{}}
		return c.ask("signup", waBotSignupSteps[0].key, waBotSignupSteps[0].ask), nil
	}
	cur := waBotSignupSteps[idx]
	if !isText {
		return c.unclearReply(waBotWriteAnswer, cur.ask), nil
	}
	raw := strings.TrimSpace(in.Text)
	if r := []rune(raw); len(r) > 200 {
		raw = string(r[:200])
	}
	val := raw
	switch {
	case cur.yesno:
		switch waBotYesNo(t, true) {
		case 1:
			val = "yes"
		case 0:
			val = "no"
		default:
			return c.unclearReply(waBotUnclear, waBotAskYesNo), nil
		}
	case cur.key == "age":
		if !waBotValidAge(t) {
			return c.unclearReply(waBotUnclear, cur.ask), nil
		}
		val = waBotDigits(t)
	case cur.key == "name":
		if len([]rune(raw)) < 3 || !strings.ContainsFunc(raw, unicode.IsLetter) {
			return c.unclearReply(waBotUnclear, cur.ask), nil
		}
	}
	if c.data.Signup == nil {
		c.data.Signup = map[string]string{}
	}
	c.data.Signup[cur.key] = val
	if idx+1 < len(waBotSignupSteps) {
		nx := waBotSignupSteps[idx+1]
		return c.ask("signup", nx.key, nx.ask), nil
	}
	ans, phone := c.data.Signup, c.phone
	r := c.ask("done", "", waBotSignupDone)
	c.data = waBotData{}
	return r, []waBotEffect{func(ctx context.Context, reply *notify.MenuReply) {
		s.waBotSaveApplication(ctx, phone, ans, reply)
	}}
}

func (s *Server) waBotSaveApplication(ctx context.Context, phone string, a map[string]string, reply *notify.MenuReply) {
	var id string
	err := s.pg.QueryRow(ctx, `
		INSERT INTO captain_applications (phone, full_name, age, area, has_ebike, has_motorbike,
		                                  has_license, has_android, experience, hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id::text`,
		phone, a["name"], a["age"], a["area"], a["ebike"] == "yes", a["motorbike"] == "yes",
		a["license"] == "yes", a["android"] == "yes", a["experience"], a["hours"]).Scan(&id)
	if err != nil {
		s.logger.Error("قائمة البوت: حفظُ طلب التسجيل سقط", "error", err)
		reply.Text = waBotOrderFailed
		return
	}
	s.notify.NotifyCaps(ctx, []string{string(authz.DriversRead)}, notifications.Input{
		Kind: notifications.KindAccount, Title: waBotNotifSignupTitle,
		Body:   fmt.Sprintf(waBotNotifSignupBody, a["name"], phone),
		Entity: "captain_application", EntityID: id, Href: "/dashboard/captain-applications",
	})
	s.touch("captain_application", "ops")
}

// ── إشعاراتُ الموظّفين ──────────────────────────────────────────────────

const (
	waBotNotifOrderTitle  = "طلب واتساب من البوت بانتظار الموافقة"
	waBotNotifOrderBody   = "الطلب #%d من الرقم %s ينتظر كتابة الأجرة والموافقة."
	waBotNotifAdminTitle  = "زبون ينتظر الردّ على الواتساب"
	waBotNotifAdminBody   = "الرقم %s اختار محادثة الإدارة من قائمة البوت."
	waBotNotifSignupTitle = "طلب تسجيل كابتن جديد"
	waBotNotifSignupBody  = "%s — %s"
)

// ── خارجَ الدوام ────────────────────────────────────────────────────────

// waBotClosedText **رسالةُ خارج الدوام بموعد الفتح** — و`false` إن كانت المنصّةُ تستقبل.
//
// **ودوامُ المنصّة وحدَه** (`platform_closed_now`) — الإيقافُ المؤقّتُ لا يُقال عنه «انتهى دوامنا».
func (s *Server) waBotClosedText(ctx context.Context) (string, bool) {
	if s.platform == nil {
		return "", false
	}
	st, err := s.platform.State(ctx, s.pg)
	if err != nil || st.OrderingAvailable || st.Reason != platform.ReasonPlatformClosedNow {
		return "", false
	}
	return waBotClosedMessage(s.platform.Now(), st.NextAvailableAt), true
}

// waBotClosedMessage **نصُّ «انتهى دوامنا» بيوم الفتح وساعته بتوقيت دمشق** — دالّةٌ خالصة.
func waBotClosedMessage(now time.Time, next *time.Time) string {
	if next == nil {
		return waBotClosedPrefix + waBotClosedNoTime
	}
	loc := platform.Location()
	n, o := now.In(loc), next.In(loc)
	clock := waBotClock(o)
	dn := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	do := time.Date(o.Year(), o.Month(), o.Day(), 0, 0, 0, 0, loc)
	days := int(do.Sub(dn).Hours()/24 + 0.5)
	switch days {
	case 0:
		return waBotClosedPrefix + fmt.Sprintf(waBotClosedToday, clock)
	case 1:
		return waBotClosedPrefix + fmt.Sprintf(waBotClosedLater, "غداً", clock)
	}
	return waBotClosedPrefix + fmt.Sprintf(waBotClosedLater, "يوم "+waBotWeekday[o.Weekday()], clock)
}

var waBotWeekday = [...]string{"الأحد", "الاثنين", "الثلاثاء", "الأربعاء", "الخميس", "الجمعة", "السبت"}

// waBotClock «9:00 صباحاً» / «4:30 مساءً».
func waBotClock(t time.Time) string {
	h := t.Hour() % 12
	if h == 0 {
		h = 12
	}
	part := "صباحاً"
	if t.Hour() >= 12 {
		part = "مساءً"
	}
	return fmt.Sprintf("%d:%02d %s", h, t.Minute(), part)
}

// ── قراءةُ الأجوبة ──────────────────────────────────────────────────────

// waBotNorm **يوحّد ما يكتبه الناس** — الأرقامُ الهنديّة واللاتينيّة، والتشكيلُ والتطويل، وأشكالُ
// الألف، ورموزُ الأرقام (1️⃣)، والترقيمُ في الآخر.
func waBotNorm(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= 0x064B && r <= 0x0652, r == 'ـ', r == 0xFE0F, r == 0x20E3, r == 0x200F, r == 0x200E:
		case r == 'أ', r == 'إ', r == 'آ':
			b.WriteRune('ا')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	out := strings.TrimRight(strings.TrimSpace(b.String()), " .،,!؟?")
	return strings.Join(strings.Fields(out), " ")
}

var waBotYes = map[string]bool{"نعم": true, "اي": true, "ايه": true, "ايوه": true, "ايوا": true,
	"اكيد": true, "يس": true, "yes": true, "y": true, "نعم يوجد": true, "اي نعم": true}
var waBotNo = map[string]bool{"لا": true, "لاء": true, "كلا": true, "no": true, "n": true, "لا يوجد": true,
	"لا شكرا": true, "لاا": true}

// waBotYesNo ١ نعم · ٠ لا · −١ غيرُ مفهوم.
func waBotYesNo(t string, isText bool) int {
	switch {
	case !isText:
		return -1
	case waBotYes[t]:
		return 1
	case waBotNo[t]:
		return 0
	}
	return -1
}

func waBotNoStore(t string) bool {
	switch t {
	case "لا يوجد", "لايوجد", "لا", "لاء", "no", "لا شي", "لا شيء", "لا احد", "اي متجر", "اي محل":
		return true
	}
	return false
}

func waBotDigits(t string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, t)
}

func waBotValidAge(t string) bool {
	d := waBotDigits(t)
	n, err := strconv.Atoi(d)
	return err == nil && len(d) <= 2 && n >= 14 && n <= 80
}

// waBotOrderNumber **رقمُ الطلب** — أرقامٌ وحدَها، و«#» قبلها مقبولة.
func waBotOrderNumber(t string) (int64, bool) {
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "#"))
	if t == "" || len(t) > 12 {
		return 0, false
	}
	n, err := strconv.ParseInt(t, 10, 64)
	return n, err == nil && n > 0
}

// ── التذكيرُ والإسقاط ───────────────────────────────────────────────────

// RunWABotMenu **جولةُ التذكير كلَّ نصف دقيقة.**
func (s *Server) RunWABotMenu(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepWABotMenuOnce(ctx)
		}
	}
}

// SweepWABotMenuOnce **جولةٌ واحدة**: تذكيرٌ مرّةً بعد ثلاث دقائق سكوتٍ وسطَ طلبٍ أو تسجيل، **ثمّ
// إسقاطٌ صامتٌ بعد ثلاثٍ أخرى** (قرارُ المالك: «تذكير بعد ٣ دقايق، وإذا ما ردّ نتجاهله»). ويردّ عددَ
// التذكيرات التي وصلت.
//
// **الحجزُ قبل الإرسال** (`reminded_at`) — جولتان لا تذكّران مرّتين، **وتذكيرٌ لم يصل لا يُعاد.**
func (s *Server) SweepWABotMenuOnce(ctx context.Context) int {
	secs := int(waBotFlowReminder.Seconds())
	if _, err := s.pg.Exec(ctx, `
		UPDATE wa_bot_chats SET state = 'idle', step = '', data = '{}'::jsonb, unclear = 0, updated_at = now()
		 WHERE state IN ('order', 'signup') AND reminded_at IS NOT NULL
		   AND reminded_at <= now() - make_interval(secs => $1)`, secs); err != nil {
		s.logger.Warn("قائمة البوت: الإسقاطُ سقط", "error", err)
	}
	if !s.WABotMenuEnabled(ctx) || !s.merchantReady() {
		return 0
	}
	rows, err := s.pg.Query(ctx, `
		UPDATE wa_bot_chats SET reminded_at = now(), last_activity_at = now(), updated_at = now()
		 WHERE phone IN (SELECT phone FROM wa_bot_chats
		                  WHERE state IN ('order', 'signup') AND reminded_at IS NULL
		                    AND step_at <= now() - make_interval(secs => $1)
		                  ORDER BY step_at LIMIT 50)
		   AND reminded_at IS NULL
		RETURNING phone`, secs)
	if err != nil {
		s.logger.Warn("قائمة البوت: قراءةُ التذكير سقطت", "error", err)
		return 0
	}
	var phones []string
	for rows.Next() {
		var p string
		if rows.Scan(&p) == nil {
			phones = append(phones, p)
		}
	}
	rows.Close()
	sent := 0
	for _, p := range phones {
		if ctx.Err() != nil {
			break
		}
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		if s.sendText(sendCtx, p, waBotReminder) {
			sent++
		}
		cancel()
	}
	return sent
}
