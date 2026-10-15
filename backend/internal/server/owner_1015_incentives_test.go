package server

// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك ٢٠٢٦-١٠-١٥ — الأهدافُ والمكافآت**
// ══════════════════════════════════════════════════════════════════════
//
//	٢ · إشعارُ المرحلة بلفظها — «أنجزت هدف الشهر» عند آخر مرحلةٍ مضبوطةٍ وحدَها
//	٣ · الموقوفُ لا يخسر ما بلغه — يُصرف له يومَ يُعاد تفعيلُه، والجولةُ تلتقط ما فات
//	٤ · خفضُ الهدف في منتصف الشهر يصرف فوراً لمن صار مؤهَّلاً
//	٦ · الطلبُ يُبلَّغ للماليّة والرفضُ لمقترحه
//	٧ · من له الدوران معاً يُعدّ طلبُه في الدور الذي اقتُرح له وحدَه
//	٨ · مكافآتُ الأرباح أربعةُ سطور: أهداف · يدويّة · دعوات · هدايا تسجيل

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// lastTitle **عنوانُ آخر إشعارٍ وصل صاحبَه.**
func (f *incFixture) lastTitle(t *testing.T, uid string) string {
	t.Helper()
	var title string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT title FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		uid).Scan(&title); err != nil {
		t.Fatalf("لا إشعارَ لـ%s: %v", uid, err)
	}
	return title
}

// TestINC1015_StageNotificationSaysTheStage **«أنجزت هدف الشهر» عند آخر مرحلةٍ وحدَها.**
//
// مرحلتان مضبوطتان (والثالثةُ مطفأة): **الأولى تُبشَّر «وصلت للمرحلة ١»** — وكانت
// تقول «أنجزت هدف الشهر» فيظنّ أنّه أتمّ الشهر. **والثانيةُ هي الأخيرة** فتُقال كاملة.
func TestINC1015_StageNotificationSaysTheStage(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"sales.monthly_target": "1", "sales.target_reward": "1000",
		"sales.target_2": "2", "sales.reward_2": "2000",
		"sales.target_3": "0", "sales.reward_3": "0",
	})
	ctx := context.Background()
	rep := f.kept(t, "sales")

	f.srv.grantSalesTargetIfAny(ctx, f.newStore(t, rep))
	if got, want := f.lastTitle(t, rep), "وصلت للمرحلة 1 — نالتك مكافأتها"; got != want {
		t.Fatalf("**إشعارُ المرحلة الأولى**: %q — والمنتظَر %q", got, want)
	}
	f.srv.grantSalesTargetIfAny(ctx, f.newStore(t, rep))
	if got, want := f.lastTitle(t, rep), "أنجزت هدف الشهر — نالتك مكافأته"; got != want {
		t.Fatalf("**إشعارُ آخر مرحلةٍ مضبوطة**: %q — والمنتظَر %q", got, want)
	}
}

// TestINC1015_SuspendedRepPaidOnReactivation **مندوبٌ اكتمل متجرُه وهو موقوفٌ يقبض يومَ يعود.**
//
// **كان يُستثنى المندوبُ غيرُ الفعّال فتضيع مرحلتُه للأبد** — العدّادُ لا يعود إلى
// العتبة نفسِها. «يظل إذا حقق الشروط».
func TestINC1015_SuspendedRepPaidOnReactivation(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"sales.monthly_target": "1", "sales.target_reward": "1500",
		"sales.target_2": "0", "sales.reward_2": "0", "sales.target_3": "0", "sales.reward_3": "0",
	})
	ctx := context.Background()
	rep := f.kept(t, "sales")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET status = 'suspended' WHERE id = $1`, rep); err != nil {
		t.Fatal(err)
	}
	f.srv.grantSalesTargetIfAny(ctx, f.newStore(t, rep))
	if b := f.balance(t, rep); b != 0 {
		t.Fatalf("صُرف لموقوفٍ وهو موقوف: %d", b)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET status = 'active' WHERE id = $1`, rep); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader("{}"))
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, f.approver))
	f.srv.afterStatusChange(req, rep, "active")
	if b := f.balance(t, rep); b != 1500 {
		t.Fatalf("**بعد إعادة التفعيل الرصيدُ %d** — والمنتظَر 1500: مرحلةٌ بلغها وهو موقوف", b)
	}
	if got := f.lastTitle(t, rep); got != "أنجزت هدف الشهر — نالتك مكافأته" {
		t.Fatalf("لم يُبشَّر بما قبض: %q", got)
	}
}

// TestINC1015_LoweringTargetPaysNow **خفضُ الهدف يصرف فوراً لمن صار مؤهَّلاً.**
//
// سائقٌ وصّل طلبين والهدفُ خمسة؛ **خُفّض إلى اثنين** — كان ينتظر تسليماً ثالثاً
// ليتحرّك العدّاد، **ومن وقف لا يقبض أبداً.**
func TestINC1015_LoweringTargetPaysNow(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{
		"drivers.monthly_target": "5", "drivers.target_reward": "4000",
		"drivers.target_2": "0", "drivers.reward_2": "0", "drivers.target_3": "0", "drivers.reward_3": "0",
	})
	ctx := context.Background()
	driver := f.kept(t, "driver")
	f.deliveredOrder(t, driver)
	f.deliveredOrder(t, driver)
	if paid := f.srv.incentives.GrantTargetIfReached(ctx, driver, "driver"); paid != 0 {
		t.Fatalf("صُرف قبل بلوغ الهدف: %d", paid)
	}

	req := httptest.NewRequest(http.MethodPut, "/x", strings.NewReader(`{"value":2}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("key", "drivers.monthly_target")
	rctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	rctx = context.WithValue(rctx, ctxUserID, f.approver)
	rctx = context.WithValue(rctx, ctxCaps, []string{string(authz.SettingsFinancialManage),
		string(authz.SettingsGeneralManage)})
	w := httptest.NewRecorder()
	f.srv.handleSetSetting(w, req.WithContext(rctx))
	if w.Code != http.StatusOK {
		t.Fatalf("حفظُ الهدف ردّ %d: %s", w.Code, w.Body.String())
	}
	if b := f.balance(t, driver); b != 4000 {
		t.Fatalf("**خُفّض الهدفُ إلى ما بلغه ولم يُصرف**: الرصيدُ %d والمنتظَر 4000", b)
	}
	if got := f.lastTitle(t, driver); got != "أنجزت هدف الشهر — نالتك مكافأته" {
		t.Fatalf("لم يُبشَّر: %q", got)
	}
}

type incRolePage struct {
	Data struct {
		Standings []struct {
			UserID     string `json:"user_id"`
			ManualPaid int64  `json:"manual_paid"`
		} `json:"standings"`
	} `json:"data"`
}

func (f *incFixture) manualIn(t *testing.T, role, uid string) int64 {
	t.Helper()
	w := f.call(f.proposer, http.MethodGet, "/x", map[string]string{"role": role}, ``,
		f.srv.handleIncentiveStandings)
	if w.Code != http.StatusOK {
		t.Fatalf("الصفحة %d: %s", w.Code, w.Body.String())
	}
	var p incRolePage
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	for _, s := range p.Data.Standings {
		if s.UserID == uid {
			return s.ManualPaid
		}
	}
	t.Fatalf("صاحبُ الدورين غائبٌ عن تبويب %s", role)
	return 0
}

// TestINC1015_DualRoleCountedOnce **سائقٌ مندوبٌ معاً — مكافأتُه في الدور الذي اقتُرحت له.**
func TestINC1015_DualRoleCountedOnce(t *testing.T) {
	f := newIncFixture(t)
	ctx := context.Background()
	u := f.kept(t, "driver")
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'sales') ON CONFLICT DO NOTHING`, u); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": u},
		`{"kind":"reward","amount":900,"reason":"عملاء جدد","role":"sales"}`, f.srv.handleIncentiveGrant)
	if w.Code != http.StatusCreated {
		t.Fatalf("الاقتراح %d: %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if ok := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res.Data.RequestID}, `{}`,
		f.srv.handleDecideIncentiveRequest(true)); ok.Code != http.StatusOK {
		t.Fatalf("الموافقة %d: %s", ok.Code, ok.Body.String())
	}
	if got := f.manualIn(t, "sales", u); got != 900 {
		t.Fatalf("تبويبُ المندوب: %d والمنتظَر 900", got)
	}
	if got := f.manualIn(t, "driver", u); got != 0 {
		t.Fatalf("**عُدّت المكافأةُ مرّتين** — تبويبُ السائق يحمل %d", got)
	}
}

// TestINC1015_ProposalNotifiesFinanceRejectionNotifiesProposer **الطلبُ للماليّة والرفضُ لمقترحه.**
func TestINC1015_ProposalNotifiesFinanceRejectionNotifiesProposer(t *testing.T) {
	f := newIncFixture(t)
	ctx := context.Background()
	driver := f.kept(t, "driver")
	count := func(uid string) int {
		var n int
		_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND href = '/dashboard/incentives'`, uid).Scan(&n)
		return n
	}
	w := f.call(f.proposer, http.MethodPost, "/x", map[string]string{"id": driver},
		`{"kind":"penalty","amount":300,"reason":"تأخير"}`, f.srv.handleIncentiveGrant)
	if w.Code != http.StatusCreated {
		t.Fatalf("الاقتراح %d: %s", w.Code, w.Body.String())
	}
	if n := count(f.approver); n != 1 {
		t.Fatalf("**الماليّةُ لم تعلم بطلبٍ ينتظرها** — %d إشعار", n)
	}
	if n := count(f.proposer); n != 0 {
		t.Fatalf("أُشعر المقترحُ بطلبه هو: %d", n)
	}
	var res struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if rj := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res.Data.RequestID},
		`{"note":"لا دليل"}`, f.srv.handleDecideIncentiveRequest(false)); rj.Code != http.StatusOK {
		t.Fatalf("الرفض %d: %s", rj.Code, rj.Body.String())
	}
	var title, body string
	if err := f.pool.QueryRow(ctx, `SELECT title, body FROM notifications
		WHERE user_id = $1 AND href = '/dashboard/incentives' ORDER BY created_at DESC LIMIT 1`,
		f.proposer).Scan(&title, &body); err != nil {
		t.Fatalf("**المقترحُ لم يعلم أنّ طلبَه رُفض**: %v", err)
	}
	if title != "انرفض طلب العقوبة اللي اقترحته" || !strings.Contains(body, "لا دليل") {
		t.Fatalf("إشعارُ الرفض: %q / %q", title, body)
	}
}

// ── ٨ · سطورُ المكافآت في الأرباح ──────────────────────────────────────

// postNote **قيدٌ بنصٍّ ومرجعٍ بعينهما في يومٍ بعينه** — ويُمحى بعده.
func (f profitsFx) postNote(t *testing.T, user string, amount int64, kind, ref, note string, at time.Time) {
	t.Helper()
	ctx := context.Background()
	_, id, err := f.srv.wallet.ApplyTxID(ctx, f.srv.pg, user, amount, kind, ref, note, nil)
	if err != nil {
		t.Fatalf("قيد %s %d: %v", kind, amount, err)
	}
	if _, err := f.srv.pg.Exec(ctx, `UPDATE wallet_transactions SET created_at = $2 WHERE id = $1`,
		id, at); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(),
			`UPDATE wallets SET balance = balance - $2 WHERE user_id = $1`, user, amount)
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM wallet_transactions WHERE id = $1`, id)
	})
}

// incentiveRow **صفُّ حافزٍ يُشار إليه بمرجع القيد.**
func (f profitsFx) incentiveRow(t *testing.T, user string, forTarget bool) string {
	t.Helper()
	var period any
	if forTarget {
		period = "2001-09#1"
	}
	var id string
	if err := f.srv.pg.QueryRow(context.Background(), `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, period)
		VALUES ($1, 'reward', 1, 'فحص الأرباح', $2, $3) RETURNING id::text`,
		user, forTarget, period).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM incentives WHERE id = $1`, id)
	})
	return id
}

// TestProfits1015_RewardLinesSplitBySource **أربعةُ سطورٍ بمصدرها لا بفراغ مرجعها.**
//
// **كان «مرجعٌ فارغ = هدف»**: دعوةٌ بوضع التسجيل (مرجعُها فارغ) تُعدّ هدفاً، وهديّةُ
// التسجيل (مرجعُها الزبون) تُعدّ دعوة، واليدويّةُ تختلط بالأهداف.
func TestProfits1015_RewardLinesSplitBySource(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ١٠١٥")
	day := damascusAt(t, "2001-09-14 12:00")
	target := f.incentiveRow(t, f.driver, true)
	manual := f.incentiveRow(t, f.driver, false)

	f.postNote(t, f.treasury, -400, "reward", target, "مكافأةُ هدفٍ صُرفت", day)
	f.postNote(t, f.treasury, -300, "reward", manual, "مكافأةٌ صُرفت", day)
	// **دعوةٌ بوضع التسجيل — مرجعُها فارغ.**
	f.postNote(t, f.treasury, -200, "reward", "", "مكافأةُ دعوةٍ صُرفت", day)
	// **هديّةُ التسجيل — مرجعُها الزبون.**
	f.postNote(t, f.treasury, -100, "reward", f.customer, "هديّةُ حسابٍ جديدٍ صُرفت", day)

	r := readPlatformProfits(t, f.srv, "2001-09-14")
	for k, v := range map[string]int64{
		"targets": -400, "manual_rewards": -300, "referrals": -200, "signup_bonuses": -100,
	} {
		if got := r.line(k); got != v {
			t.Errorf("سطر %s = %d، المتوقَّع %d", k, got, v)
		}
	}
	if r.Net != -1000 || r.LinesSum != r.Net || !r.CheckOK {
		t.Fatalf("الصافي %d ومجموع البنود %d والتحقّق %v — المتوقَّع −1000 ومتساويان",
			r.Net, r.LinesSum, r.CheckOK)
	}
}
