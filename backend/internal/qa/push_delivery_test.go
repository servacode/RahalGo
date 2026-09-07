package qa

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **دفعٌ يُعاد أو يُسجَّل — ولا يضيع صامتاً** — `PF-09` · `R23`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`SendToUser` تُنادى مرّةً واحدةً بعد الحفظ**: **ثلاثةُ إخفاقاتٍ
// وثلاثةُ نداءاتٍ ولا رابع** — **لا صفَّ انتظارٍ ولا حالٌ معلَّقةٌ ولا
// سطرٌ يقول إنّه لم يصل.**
//
// # وما يُقاس هنا — الطبقتان الأولى والثانية
//
//	١ نيّةٌ دائمة              — أُحكمت في `PF-07`
//	٢ محاولةٌ وقبولُ المزوّد    — **هذه**
//	٣ عرضُ الجهاز              — **جهازٌ حقيقيّ · `P-8`**
//
// **ولا يُدَّعى أنّ هاتفاً عرض شيئاً.** **`accepted` تعني أنّ المزوّدَ
// قَبِل الطلب.**

// ── أدواتُ القياس ─────────────────────────────────────────────────────

// delivery حالُ صفّ نقلٍ واحد.
type delivery struct {
	State    string
	Attempts int
	Class    string
	Err      string
	Accepted bool
	NextIn   time.Duration
}

// deliveriesOf صفوفُ نقلِ إشعاراتِ حسابٍ بعنوانٍ بعينه.
func deliveriesOf(t *testing.T, h *Harness, uid, title string) map[string]delivery {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT d.token, d.state, d.attempts, d.last_error_class, d.last_error,
		       d.provider_accepted_at IS NOT NULL,
		       GREATEST(d.next_attempt_at - now(), interval '0')
		  FROM notification_deliveries d
		  JOIN notifications n ON n.id = d.notification_id
		 WHERE n.user_id = $1::uuid AND n.title = $2`, uid, title)
	if err != nil {
		t.Fatalf("قراءةُ صفوف النقل: %v", err)
	}
	defer rows.Close()
	out := map[string]delivery{}
	for rows.Next() {
		var tok string
		var d delivery
		if err := rows.Scan(&tok, &d.State, &d.Attempts, &d.Class, &d.Err,
			&d.Accepted, &d.NextIn); err != nil {
			t.Fatalf("مسحُ صفّ النقل: %v", err)
		}
		out[tok] = d
	}
	return out
}

// notifyUser يُنشئ إشعاراً دائماً لحسابٍ عبر قيدِ محفظةٍ إداريّ.
//
// **ومسارٌ حقيقيٌّ لا نداءٌ داخليّ** — **فالفحصُ يقيس ما يقع في المنصّة.**
func notifyUser(t *testing.T, h *Harness, admin *User, uid string, amount int64) {
	t.Helper()
	got := h.POST("/api/v1/admin/users/"+uid+"/wallet", admin.Token,
		map[string]any{"amount": amount, "kind": "topup", "note": "PF-09"})
	if got.Code >= 400 {
		t.Fatalf("قيدُ المحفظة: %s", got)
	}
}

// walletNoticeTitle عنوانُ إشعارِ قيد المحفظة — **يُقرأ من القاعدة لا
// يُخمَّن**، فمن بدّل النصَّ لم يُسقط الفحصَ صامتاً.
func walletNoticeTitle(t *testing.T, h *Harness, uid string) string {
	t.Helper()
	var title string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT title FROM notifications
		 WHERE user_id = $1::uuid ORDER BY created_at DESC LIMIT 1`,
		uid).Scan(&title); err != nil {
		t.Fatalf("عنوانُ الإشعار: %v", err)
	}
	return title
}

// settle **تنتظر أن تهدأ نبضةُ النقل قبل القياس.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولا يكفي انتظارُ نداءِ الناقل**
// ══════════════════════════════════════════════════════════════════════
//
// **قيس**: `WaitCalls` تعود لحظةَ تسجيل النداء — **وقيدُ نتيجته يقع
// بعدها.** فقُرئ صفٌّ `pending` بلا صنفٍ ولا تأجيل، **وهو في الطريق
// إليهما.** **وفحصٌ يقيس منتصفَ عمليّةٍ يقيس سباقاً لا عقداً.**
//
// **فيُنتظَر الهدوء**: **لا صفَّ محجوزاً** — والحجزُ يُرفَع في كلّ
// مصير: قبولاً أو إعادةً أو نهايةً.
//
// **والعددُ المتوقَّعُ يُمرَّر**: **صفٌّ واحدٌ من إشعارٍ سابقٍ يُرضي
// «ثمّةَ صفوف» فتعود قبل أن يُفرَّع الجديد** — **وقيس ذلك: إشعاران
// وصفٌّ واحد، والثاني في الطريق.**
func settle(t *testing.T, h *Harness, uid string, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var busy, fresh, rows int
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT count(*) FILTER (WHERE d.claimed_until IS NOT NULL
			                          AND d.claimed_until > now()),
			       count(*) FILTER (WHERE d.attempts = 0),
			       count(*)
			  FROM notification_deliveries d
			  JOIN notifications n ON n.id = d.notification_id
			 WHERE n.user_id = $1::uuid`, uid).Scan(&busy, &fresh, &rows); err != nil {
			t.Fatalf("انتظارُ الهدوء: %v", err)
		}
		// **وصفٌّ فُرِّع للتوّ يبدو هادئاً وعملُه لم يبدأ** — **قيس**:
		// **`claim` لم تُثبَّت بعد، فلا حجزَ ولا محاولة.** **فيُشترَط
		// أن يكون كلُّ صفٍّ قد حاول مرّةً.**
		if rows >= want && busy == 0 && fresh == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Logf("**لم تهدأ النبضةُ في مهلتها**: صفوفٌ=%d (المرتقَبُ %d) · "+
				"محجوزٌ=%d · بلا محاولةٍ=%d", rows, want, busy, fresh)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N1 · نيّةٌ دائمةٌ ثمّ قبولُ المزوّد**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N1_ProviderSuccessRecorded(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n1") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 1)
	h.API.DeliverPushOnce(ctxBG())

	got := deliveriesOf(t, h, u.ID, title)
	d := got["pf09-n1"]
	t.Logf("N1: حالٌ=%q · محاولاتٌ=%d · قبولٌ مقيَّد=%v · نداءاتٌ=%d",
		d.State, d.Attempts, d.Accepted, len(fake.Calls()))

	if d.State != "accepted" || !d.Accepted {
		t.Errorf("**قبولُ المزوّد لم يُقيَّد**: حالٌ=%q · قبولٌ=%v",
			d.State, d.Accepted)
	}
	if len(fake.Calls()) == 0 {
		t.Error("**لم يُنادَ الناقلُ أصلاً**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N2+N3+N8 · عابرٌ يُعاد بتراجعٍ ولا يحمى**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N2N3N8_TransientRetriesWithBackoff(t *testing.T) {
	fake := NewFakePush().Script(PushTimeout, PushHTTP500, PushConnReset)
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n2") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 1)

	after := deliveriesOf(t, h, u.ID, title)["pf09-n2"]
	t.Logf("بعد المهلة: حالٌ=%q · محاولاتٌ=%d · صنفٌ=%q · التالي بعد %s",
		after.State, after.Attempts, after.Class, after.NextIn.Round(time.Second))

	if after.State != "pending" {
		t.Errorf("**مهلةٌ عابرةٌ لم تبقَ قابلةً للإعادة**: %q", after.State)
	}
	if after.Class != "TRANSPORT" {
		t.Errorf("**صنفُ الخطأ لم يُقيَّد**: %q", after.Class)
	}
	if after.Err == "" {
		t.Error("**لا نصَّ خطأٍ محفوظ**")
	}

	// **والموعدُ التالي مؤجَّلٌ لا فوريّ** — **وإلّا حلقةٌ محمومة.**
	if after.NextIn <= 0 {
		t.Errorf("**لا تأجيل**: التالي بعد %s — **حلقةٌ محمومة**", after.NextIn)
	}
	if after.NextIn > 35*time.Second {
		t.Errorf("**التأجيلُ الأوّلُ %s وجدولُ المالك ثلاثون ثانية**", after.NextIn)
	}

	// ── الجولةُ التاليةُ لا تلتقطه قبل موعده ──────────────────────
	before := len(fake.Calls())
	h.API.DeliverPushOnce(ctxBG())
	t.Logf("جولةٌ قبل الموعد: نداءاتٌ %d ← %d", before, len(fake.Calls()))
	if len(fake.Calls()) != before {
		t.Errorf("**نُودي قبل موعده** — **والتراجعُ لا يُحترَم** (%d ← %d)",
			before, len(fake.Calls()))
	}

	// ── ويُلتقَط حين يحلّ موعدُه ───────────────────────────────────
	due(t, h, u.ID, title)
	h.API.DeliverPushOnce(ctxBG())
	second := deliveriesOf(t, h, u.ID, title)["pf09-n2"]
	t.Logf("بعد ٥٠٠: حالٌ=%q · محاولاتٌ=%d · التالي بعد %s",
		second.State, second.Attempts, second.NextIn.Round(time.Second))
	if second.Attempts != 2 {
		t.Errorf("**عدُّ المحاولات لم يتقدّم**: %d", second.Attempts)
	}
	if second.NextIn <= after.NextIn {
		t.Errorf("**التراجعُ لا يتّسع**: %s ← %s", after.NextIn, second.NextIn)
	}
}

// due يُحلّ موعدَ صفوف نقلٍ للفحص — **بلا انتظارِ نصفِ ساعةٍ حقيقيّة.**
//
// **ويُقاس التأجيلُ قبل هذا** — **فالإحلالُ لا يُخفي غيابَه.**
func due(t *testing.T, h *Harness, uid, title string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE notification_deliveries d
		   SET next_attempt_at = now() - interval '1 second'
		  FROM notifications n
		 WHERE n.id = d.notification_id
		   AND n.user_id = $1::uuid AND n.title = $2
		   AND d.state = 'pending'`, uid, title); err != nil {
		t.Fatalf("إحلالُ الموعد: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N4+N10 · موتٌ قبل النداء — والعملُ يبقى مكتشَفاً**
// ══════════════════════════════════════════════════════════════════════
//
// **وموتُ المنفّذ يُمثَّل بألّا تُدار جولةٌ أصلاً** — **النبضةُ خيطٌ،
// والخيطُ يموت مع عمليّته.**
func TestPF09_N4N10_RestartFindsPendingWork(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	f := h.Factory()

	var users []string
	for i := 0; i < 3; i++ {
		u := f.NewUserWith("customer")
		if !addToken(t, h, u.ID, fmt.Sprintf("pf09-n4-%d", i)) {
			t.Skip("جدولُ رموز الدفع غيرُ متاح")
		}
		users = append(users, u.ID)
	}
	// **ولا ناقلَ يقبل** — فتبقى معلَّقةً كأنّ المنفّذَ مات قبل النداء.
	fake.Default(PushConnReset)
	for _, uid := range users {
		notifyUser(t, h, admin, uid, 1000)
	}
	// **والنبضةُ خانةٌ واحدة**: **ثلاثةُ إشعاراتٍ في ومضةٍ قد تُولّد
	// جولةً واحدةً تسبق آخرَها** — **فتُدار جولةٌ صريحةٌ تُفرّع الكلَّ
	// وتحاول.** (وهو ما يفعله المُقلِعُ الجديدُ بعينه.)
	h.API.DeliverPushOnce(ctxBG())
	for _, uid := range users {
		settle(t, h, uid, 1)
	}

	// **والإقلاعُ الجديدُ يجد العمل** — لا ذاكرةَ تُفقد.
	for _, uid := range users {
		due(t, h, uid, walletNoticeTitle(t, h, uid))
	}
	fake.Default(PushSuccess)
	h.API.DeliverPushOnce(ctxBG())

	var recovered int
	for i, uid := range users {
		d := deliveriesOf(t, h, uid, walletNoticeTitle(t, h, uid))[fmt.Sprintf("pf09-n4-%d", i)]
		t.Logf("  الحسابُ %d: حالٌ=%q · محاولاتٌ=%d", i, d.State, d.Attempts)
		if d.State == "accepted" {
			recovered++
		}
	}
	t.Logf("N4/N10: تعافى %d من %d", recovered, len(users))
	if recovered != len(users) {
		t.Errorf("**عملٌ معلَّقٌ لم يُستأنَف بعد الإقلاع**: %d من %d",
			recovered, len(users))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N5 · قَبِل المزوّدُ ثمّ مات قبل القيد**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُدَّعى استحالتُه**: **نداءُ شبكةٍ وتثبيتُ قاعدةٍ لا يجتمعان
// ذرّيّاً.** **فالدلالةُ «محاولةٌ مرّةً على الأقلّ»** — **والمقيسُ أنّ
// العملَ لا يضيع، لا أنّه لا يتكرّر.**
func TestPF09_N5_AcceptedThenDeathRetriesNotLoses(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n5") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 1)
	h.API.DeliverPushOnce(ctxBG())

	// **ويُحاكى موتُ المنفّذ بعد قبول المزوّد**: **القبولُ وقع في
	// الشبكة ولم يبلغ القاعدة.**
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE notification_deliveries d
		   SET state = 'pending', provider_accepted_at = NULL,
		       claimed_until = NULL, next_attempt_at = now() - interval '1 second'
		  FROM notifications n
		 WHERE n.id = d.notification_id
		   AND n.user_id = $1::uuid AND n.title = $2`, u.ID, title); err != nil {
		t.Fatalf("محاكاةُ الموت: %v", err)
	}

	before := len(fake.Calls())
	h.API.DeliverPushOnce(ctxBG())
	d := deliveriesOf(t, h, u.ID, title)["pf09-n5"]
	t.Logf("N5: نداءاتٌ %d ← %d · حالٌ=%q · محاولاتٌ=%d",
		before, len(fake.Calls()), d.State, d.Attempts)

	// **والمضمونُ باقٍ في كلّ حال.**
	var live int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`,
		u.ID, title).Scan(&live); err != nil {
		t.Fatalf("عدُّ الإشعارات: %v", err)
	}
	if live == 0 {
		t.Error("**ضاع مضمونُ الإشعار**")
	}
	if d.State != "accepted" {
		t.Errorf("**لم تُعَد المحاولةُ بعد الموت**: %q", d.State)
	}
	if len(fake.Calls()) <= before {
		t.Error("**لم يُعَد النداءُ** — **والعملُ ضاع بموت المنفّذ**")
	}
	t.Log("**وتكرارُ نداءٍ هنا مقبولٌ ومُعلَن** — " +
		"**«محاولةٌ مرّةً على الأقلّ» لا «تسليمٌ مرّةً واحدة».**")
}

// ══════════════════════════════════════════════════════════════════════
// **N6 · جولتان معاً ⇒ محاولةٌ واحدةٌ لكلّ هدف**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N6_TwoWorkersOneAttempt(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n6") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	fake.Default(PushConnReset)
	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 1)
	due(t, h, u.ID, title)

	before := len(fake.Calls())
	beforeAttempts := deliveriesOf(t, h, u.ID, title)["pf09-n6"].Attempts
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "عاملٌ-أ", Do: func(context.Context) any {
			h.API.DeliverPushOnce(ctxBG())
			return nil
		}},
		Actor{Name: "عاملٌ-ب", Do: func(context.Context) any {
			h.API.DeliverPushOnce(ctxBG())
			return nil
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	added := len(fake.Calls()) - before
	d := deliveriesOf(t, h, u.ID, title)["pf09-n6"]
	t.Logf("N6: نداءاتٌ أُضيفت=%d · محاولاتٌ %d ← %d — %s",
		added, beforeAttempts, d.Attempts, r)

	if added > 1 {
		t.Errorf("**عاملان نادَيا الهدفَ نفسَه %d مرّة** — "+
			"**والحجزُ لا يمنع.**", added)
	}
	// **والمقياسُ فرقُ العدّ لا مطلقُه** — **فمحاولةُ النبضةِ سبقت
	// السباقَ، وعدُّها ليس عيباً.**
	if d.Attempts-beforeAttempts > 1 {
		t.Errorf("**عاملان عدّا محاولتين لجولةٍ واحدة**: %d ← %d",
			beforeAttempts, d.Attempts)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N7+M1+M2 · رمزٌ ماتَ يُحذف، والسليمُ يمضي**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N7M1M2_DeadTokenDroppedHealthyDelivered(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-dead") || !addToken(t, h, u.ID, "pf09-live") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	// **والناقلُ يقتل الميّتَ وحدَه** — انظر `FakePush.Kill`.
	fake.Kill("pf09-dead")
	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 2) // **هدفان**
	h.API.DeliverPushOnce(ctxBG())

	got := deliveriesOf(t, h, u.ID, title)
	dead, live := got["pf09-dead"], got["pf09-live"]
	t.Logf("N7: الميّتُ حالٌ=%q صنفٌ=%q · الحيُّ حالٌ=%q",
		dead.State, dead.Class, live.State)

	if dead.State != "dead_token" {
		t.Errorf("**الرمزُ الميّتُ لم يُوسَم**: %q", dead.State)
	}
	if live.State != "accepted" {
		t.Errorf("**الرمزُ السليمُ لم يُسلَّم إليه**: %q", live.State)
	}

	// **ورمزٌ مرفوضٌ يبقى يُنادى عليه في كلّ حدثٍ إلى الأبد.**
	var stillThere bool
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT EXISTS (SELECT 1 FROM device_tokens WHERE token = 'pf09-dead')`).
		Scan(&stillThere); err != nil {
		t.Fatalf("فحصُ الرمز: %v", err)
	}
	if stillThere {
		t.Error("**الرمزُ الميّتُ لم يُحذَف** — يُنادى عليه أبداً")
	}

	// **ولا يُعاد عليه** — الجولةُ التاليةُ لا تلمسه.
	before := len(fake.Calls())
	h.API.DeliverPushOnce(ctxBG())
	if len(fake.Calls()) != before {
		t.Errorf("**أُعيد النداءُ على ميّت** (%d ← %d)", before, len(fake.Calls()))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N9 · إشعارٌ جديدٌ لا تخنقه حالُ سابقه**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N9_NewIntentNotSuppressedByPrior(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n9") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	notifyUser(t, h, admin, u.ID, 1000)
	settle(t, h, u.ID, 1)
	h.API.DeliverPushOnce(ctxBG())

	notifyUser(t, h, admin, u.ID, 2000)
	settle(t, h, u.ID, 2) // **إشعاران**
	h.API.DeliverPushOnce(ctxBG())

	var rows int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notification_deliveries d
		  JOIN notifications n ON n.id = d.notification_id
		 WHERE n.user_id = $1::uuid AND d.token = 'pf09-n9'`, u.ID).
		Scan(&rows); err != nil {
		t.Fatalf("عدُّ صفوف النقل: %v", err)
	}
	t.Logf("N9: صفوفُ نقلٍ للرمز نفسِه = %d لإشعارين", rows)
	if rows < 2 {
		t.Errorf("**الإشعارُ الثاني خُنق بحال الأوّل**: %d", rows)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N11 · انقطاعُ المزوّد لا يُرجع مالاً ولا طلباً**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N11_ProviderOutageDoesNotRollbackBusiness(t *testing.T) {
	fake := NewFakePush().Default(PushConnReset)
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n11") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	got := h.POST("/api/v1/admin/users/"+u.ID+"/wallet", admin.Token,
		map[string]any{"amount": 7000, "kind": "topup", "note": "PF-09"})
	settle(t, h, u.ID, 1)
	h.API.DeliverPushOnce(ctxBG())

	var bal int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT balance FROM wallets WHERE user_id = $1::uuid`, u.ID).Scan(&bal); err != nil {
		t.Fatalf("رصيدُ المحفظة: %v", err)
	}
	title := walletNoticeTitle(t, h, u.ID)
	d := deliveriesOf(t, h, u.ID, title)["pf09-n11"]
	t.Logf("N11: الفعلُ ردّ %d · الرصيدُ=%d · النقلُ حالٌ=%q محاولاتٌ=%d",
		got.Code, bal, d.State, d.Attempts)

	if got.Code >= 400 {
		t.Errorf("**سقطَ الفعلُ بسقوط الدفع**: %s", got)
	}
	if bal != 7000 {
		t.Errorf("**ارتدَّ المالُ بانقطاع المزوّد**: %d", bal)
	}
	if d.State != "pending" {
		t.Errorf("**العملُ لم يبقَ قابلاً للإعادة**: %q", d.State)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **N12 · نفادُ المحاولات ⇒ حالٌ نهائيّةٌ تُقرأ**
// ══════════════════════════════════════════════════════════════════════
func TestPF09_N12_ExhaustedBecomesDiscoverableFailure(t *testing.T) {
	fake := NewFakePush().Default(PushHTTP500)
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-n12") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 1)

	// **وسبعُ محاولاتٍ إجمالاً** — تُدار الجولاتُ بإحلال المواعيد.
	for i := 0; i < 10; i++ {
		due(t, h, u.ID, title)
		h.API.DeliverPushOnce(ctxBG())
		if deliveriesOf(t, h, u.ID, title)["pf09-n12"].State != "pending" {
			break
		}
	}

	d := deliveriesOf(t, h, u.ID, title)["pf09-n12"]
	t.Logf("N12: حالٌ=%q · محاولاتٌ=%d · صنفٌ=%q · خطأٌ=%.60q",
		d.State, d.Attempts, d.Class, d.Err)

	if d.State != "failed" {
		t.Errorf("**لم تُقيَّد حالٌ نهائيّةٌ بعد النفاد**: %q", d.State)
	}
	if d.Attempts > 8 {
		t.Errorf("**تجاوزت المحاولاتُ السقفَ**: %d", d.Attempts)
	}
	if d.Class == "" || d.Err == "" {
		t.Errorf("**السببُ لم يُحفَظ**: صنفٌ=%q · خطأٌ=%q", d.Class, d.Err)
	}

	// **والمضمونُ باقٍ في التطبيق** — سقوطُ النقل لا يمحو الخبر.
	var live int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`,
		u.ID, title).Scan(&live); err != nil {
		t.Fatalf("عدُّ الإشعارات: %v", err)
	}
	if live == 0 {
		t.Error("**حُذف الإشعارُ لأنّ دفعَه أخفق**")
	}

	// **وتُقرأ الحالُ إداريّاً** — لا تختفي صامتة.
	sum, err := h.API.PushHealth(ctxBG())
	if err != nil {
		t.Fatalf("حالُ النقل: %v", err)
	}
	t.Logf("الحالُ المقروءة: معلَّقٌ=%d · مخفقٌ=%d · رمزٌ ميّتٌ=%d · مقبولٌ=%d",
		sum.Pending, sum.Failed, sum.DeadToken, sum.Accepted)
	if sum.Failed == 0 {
		t.Error("**الإخفاقُ النهائيُّ غيرُ مقروءٍ إداريّاً**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **M3+M4 · أجهزةٌ عدّةٌ ومصائرُ مختلفة**
// ══════════════════════════════════════════════════════════════════════
//
// **وحالٌ واحدةٌ على الإشعار كانت تكذب**: **جهازٌ قَبِل وجهازٌ انقطع.**
func TestPF09_M3M4_PerTargetTruthSurvivesRestart(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	u := h.Factory().NewUserWith("customer")
	if !addToken(t, h, u.ID, "pf09-a") || !addToken(t, h, u.ID, "pf09-b") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	fake.Fail("pf09-b", PushHTTP500)
	notifyUser(t, h, admin, u.ID, 1000)
	title := walletNoticeTitle(t, h, u.ID)
	settle(t, h, u.ID, 2) // **هدفان**
	h.API.DeliverPushOnce(ctxBG())

	got := deliveriesOf(t, h, u.ID, title)
	t.Logf("M3: أ حالٌ=%q · ب حالٌ=%q محاولاتٌ=%d",
		got["pf09-a"].State, got["pf09-b"].State, got["pf09-b"].Attempts)

	if got["pf09-a"].State != "accepted" {
		t.Errorf("**الجهازُ السليمُ لم يُقبَل**: %q", got["pf09-a"].State)
	}
	if got["pf09-b"].State != "pending" {
		t.Errorf("**المنقطعُ لم يبقَ قابلاً للإعادة**: %q", got["pf09-b"].State)
	}

	// ── وبعد الإقلاع: حقيقةُ كلّ هدفٍ كما هي ──────────────────────
	fake.Heal("pf09-b")
	due(t, h, u.ID, title)
	h.API.DeliverPushOnce(ctxBG())
	after := deliveriesOf(t, h, u.ID, title)
	t.Logf("M4: أ حالٌ=%q محاولاتٌ=%d · ب حالٌ=%q محاولاتٌ=%d",
		after["pf09-a"].State, after["pf09-a"].Attempts,
		after["pf09-b"].State, after["pf09-b"].Attempts)

	if after["pf09-a"].Attempts != got["pf09-a"].Attempts {
		t.Errorf("**أُعيد النداءُ على هدفٍ مقبول**: %d ← %d",
			got["pf09-a"].Attempts, after["pf09-a"].Attempts)
	}
	if after["pf09-b"].State != "accepted" {
		t.Errorf("**المنقطعُ لم يتعافَ بعد الإقلاع**: %q", after["pf09-b"].State)
	}
}
