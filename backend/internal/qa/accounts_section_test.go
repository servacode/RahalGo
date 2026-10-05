package qa

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الحسابات — قراراتُ المالك ٢٠٢٦-١٠-٠٤** (`ACC-*`)
// ══════════════════════════════════════════════════════════════════════
//
// كلُّ اختبارٍ هنا يحرس قاعدةً في المحرّك لا في الشاشة — **ويُنادى بالموجّه
// الحقيقيّ** (سياسةُ القدرات وخطوةُ التحقّق ومنعُ التكرار كما في الإنتاج).

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/authz"
)

func walletPropose(h *Harness, by *User, target string, body map[string]any) Res {
	return h.POSTKey("/api/v1/admin/users/"+target+"/wallet", by.Token, uniq("acc"), body)
}

func ledgerByRef(t *testing.T, h *Harness, ref, kind string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = $2`, ref, kind).Scan(&v); err != nil {
		t.Fatalf("الدفتر: %v", err)
	}
	return v
}

// ── ١ · حركةُ المحفظة طلبٌ ثمّ موافقةٌ بطرفين ─────────────────────────────

func TestACC_WalletRequestIsProposalThenTwoSidedApproval(t *testing.T) {
	h := New(t)
	f := h.Factory()
	tid := treasury(t, h)
	fin := h.NewUser("finance")
	approver := h.NewUser("admin")
	target := f.NewUserWith("customer")
	before := treasuryBalance(t, h, tid)

	body := map[string]any{"amount": 5000, "kind": "compensation", "note": "تعويض تأخير"}
	r := walletPropose(h, fin, target.ID, body)
	if r.Code != 201 {
		t.Fatalf("الاقتراح: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if reqID == "" {
		t.Fatalf("لا معرّفَ للطلب: %s", r)
	}
	// **الاقتراحُ لا يحرّك مالاً.**
	if b := walletBalance(t, h, target.ID); b != 0 {
		t.Fatalf("**الاقتراحُ حرّك المال** — الرصيد %d قبل أيّ موافقة", b)
	}

	// **صاحبُ الاقتراح لا يوافق عليه** — وفي المنصّة غيرُه ممّن يملك الموافقة.
	self := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", fin.Token, map[string]any{})
	if self.Code != 403 || self.Err() != "self_approve" {
		t.Fatalf("**وافق صاحبُ الاقتراح على نفسه**: %s", self)
	}

	ok := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", approver.Token, map[string]any{})
	if ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	if b := walletBalance(t, h, target.ID); b != 5000 {
		t.Errorf("**الرصيدُ بعد الموافقة %d والمنتظَر ٥٠٠٠**", b)
	}
	// **والطرفُ الآخر: الخزينةُ نقصت بالقدر نفسِه وبالمرجع نفسِه.**
	if after := treasuryBalance(t, h, tid); before-after != 5000 {
		t.Errorf("**الخزينةُ لم تنقص بقدر التعويض**: قبل %d بعد %d", before, after)
	}
	if v := ledgerByRef(t, h, reqID, "platform_expense"); v != -5000 {
		t.Errorf("**لا قيدَ خزينةٍ بمرجع الطلب** (%d) — مالٌ خُلق من عدم", v)
	}
	if v := ledgerByRef(t, h, reqID, "compensation"); v != 5000 {
		t.Errorf("قيدُ المحفظة بمرجع الطلب %d", v)
	}
	again := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", approver.Token, map[string]any{})
	if again.Code < 400 || walletBalance(t, h, target.ID) != 5000 {
		t.Errorf("**الطلبُ نُفّذ مرّتين**: %s", again)
	}
}

func TestACC_WalletRequestRules(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	fin := h.NewUser("finance")
	target := f.NewUserWith("customer")
	path := "/api/v1/admin/users/" + target.ID + "/wallet"

	// **والمفتاحُ نفسُه مرّتين لا يكتب طلبين** — كبستان سريعتان كانتا تكتبان مرّتين.
	key := uniq("dup")
	body := map[string]any{"amount": 100, "kind": "topup", "note": "x"}
	first := h.POSTKey(path, fin.Token, key, body)
	second := h.POSTKey(path, fin.Token, key, body)
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_requests WHERE user_id = $1`, target.ID).Scan(&n)
	if first.Code != 201 || n != 1 {
		t.Errorf("**المفتاحُ المكرَّرُ كتب %d طلباً** (%s / %s)", n, first, second)
	}
	// **ولا سحبَ من هنا** — بابُه صفحةُ السحب.
	if r := walletPropose(h, fin, target.ID, map[string]any{"amount": 100, "kind": "payout", "note": "x"}); r.Err() != "wallet_payout_not_here" {
		t.Errorf("السحب: %s", r)
	}
	// **والملاحظةُ إلزاميّة.**
	if r := walletPropose(h, fin, target.ID, map[string]any{"amount": 100, "kind": "topup", "note": "  "}); r.Err() != "wallet_note_required" {
		t.Errorf("بلا ملاحظة: %s", r)
	}
	// **وفوق السقف يُرفض.**
	h.Setting("finance.manual_wallet_max", "1000")
	if r := walletPropose(h, fin, target.ID, map[string]any{"amount": 1001, "kind": "topup", "note": "x"}); r.Err() != "wallet_over_cap" {
		t.Errorf("فوق السقف: %s", r)
	}
}

func TestACC_WalletTopupEntersOfficeCashbox(t *testing.T) {
	h := New(t)
	f := h.Factory()
	tid := treasury(t, h)
	fin := h.NewUser("finance")
	approver := h.NewUser("admin")
	target := f.NewUserWith("customer")
	before := treasuryBalance(t, h, tid)

	r := walletPropose(h, fin, target.ID, map[string]any{"amount": 7000, "kind": "topup", "note": "نقد في المكتب"})
	reqID, _ := r.JSON()["request_id"].(string)
	if ok := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", approver.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	var cash int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'wallet_topup' AND ref = $1 AND direction = 'in'`, reqID).Scan(&cash); err != nil {
		t.Fatal(err)
	}
	if cash != 7000 {
		t.Errorf("**الشحنُ النقديُّ لم يدخل صندوقَ المكتب** (%d)", cash)
	}
	if b := walletBalance(t, h, target.ID); b != 7000 {
		t.Errorf("الرصيد %d", b)
	}
	if after := treasuryBalance(t, h, tid); after != before {
		t.Errorf("الشحنُ النقديُّ لا يمسّ الخزينة: قبل %d بعد %d", before, after)
	}
}

// ── ٤ · الأرصدةُ تُحجب في المحرّك ───────────────────────────────────────

func TestACC_MoneyHiddenFromNonFinanceStaff(t *testing.T) {
	h := New(t)
	f := h.Factory()
	target := f.NewUserWith("customer")
	f.Credit(target.ID, 4321, "topup")
	// **والعمليّاتُ لم تعد تقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — فالثقةُ والأمان
	// تقيس العقدَ نفسَه: تقرأ الحسابَ ولا تملك المال (والدعمُ يراه بقرار ٢٠٢٦-١٠-٠٤).
	ops := h.NewUser("trust_safety")
	// **والماليّةُ لم تعد تقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥، هجرة `0422`) — فقارئُ
	// الحساب الذي يملك المالَ يقيس العقدَ: ثقةٌ وأمانٌ + ماليّة.
	_, finTok := capUser(t, h, "trust_safety", "finance")

	got := h.GET("/api/v1/admin/users/"+target.ID, ops.Token).JSON()
	if got["money_hidden"] != true || got["balance"] != float64(0) {
		t.Errorf("**الرصيدُ وصل العمليّات**: hidden=%v balance=%v", got["money_hidden"], got["balance"])
	}
	list := h.GET("/api/v1/admin/users?query="+target.Name, ops.Token).JSON()
	if list["money_hidden"] != true {
		t.Errorf("**القائمةُ لم تحجب المال**: %v", list["money_hidden"])
	}
	if f := h.GET("/api/v1/admin/users/"+target.ID, finTok).JSON(); f["balance"] != float64(4321) {
		t.Errorf("الماليّةُ لا ترى الرصيد: %v", f["balance"])
	}
}

// ── ٦ · تغييرُ الرقم ───────────────────────────────────────────────────

func TestACC_PhoneChangeNeedsStepUpRevokesSessionsAndAudits(t *testing.T) {
	h := New(t)
	staff := h.NewUser("admin")
	target := h.NewUser("customer")
	newPhone := uniqPhone()

	// **بلا كلمةِ سرِّ الموظّف لا تغيير.**
	act, ok := authz.LookupSensitive("PATCH", "/users/{id}")
	if !ok || act.Conditional != authz.CondUserUpdateStrong {
		t.Fatalf("تغييرُ الرقم ليس فعلاً حسّاساً: %+v", act)
	}
	bare := h.Call("PATCH", "/api/v1/admin/users/"+target.ID, staff.Token,
		map[string]any{"phone": newPhone}, map[string]string{"X-Step-Up": ""})
	if bare.Err() != "step_up_required" {
		t.Fatalf("**الرقمُ يُغيَّر بلا تأكيد**: %s", bare)
	}

	r := h.PATCH("/api/v1/admin/users/"+target.ID, staff.Token, map[string]any{"phone": newPhone})
	if r.Code != 200 {
		t.Fatalf("التغيير: %s", r)
	}
	var phone string
	var live int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT phone FROM users WHERE id = $1`, target.ID).Scan(&phone)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM refresh_tokens WHERE user_id = $1
		AND revoked_at IS NULL AND expires_at > now()`, target.ID).Scan(&live)
	if phone != newPhone {
		t.Errorf("الرقم %q", phone)
	}
	if live != 0 {
		t.Errorf("**الجلساتُ القديمةُ باقية** (%d)", live)
	}
	var details string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT details::text FROM audit_log
		WHERE action = 'admin.phone_change' AND entity_id = $1`, target.ID).Scan(&details)
	if !strings.Contains(details, target.Phone) || !strings.Contains(details, newPhone) {
		t.Errorf("**السجلُّ بلا القديم والجديد**: %s", details)
	}
}

func TestACC_PhoneChangeAboveBalanceNeedsSecondPerson(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("security.phone_change_approval_balance", "1000")
	staff := h.NewUser("admin")
	other := h.NewUser("admin")
	target := f.NewUserWith("customer")
	f.Credit(target.ID, 5000, "topup")
	newPhone := uniqPhone()

	r := h.PATCH("/api/v1/admin/users/"+target.ID, staff.Token, map[string]any{"phone": newPhone})
	pc, _ := r.JSON()["phone_change"].(map[string]any)
	if r.Code != 200 || pc["pending"] != true {
		t.Fatalf("**رصيدٌ فوق الحدّ ولم ينتظر**: %s", r)
	}
	var phone string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT phone FROM users WHERE id = $1`, target.ID).Scan(&phone)
	if phone == newPhone {
		t.Fatalf("**تغيّر الرقمُ قبل الموافقة الثانية**")
	}
	id, _ := pc["request_id"].(string)
	if s := h.POST("/api/v1/admin/phone-requests/"+id+"/approve", staff.Token, map[string]any{}); s.Err() != "second_person_required" {
		t.Errorf("**وافق صاحبُ الطلب على طلبه**: %s", s)
	}
	if o := h.POST("/api/v1/admin/phone-requests/"+id+"/approve", other.Token, map[string]any{}); o.Code != 200 {
		t.Fatalf("موافقةُ الثاني: %s", o)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT phone FROM users WHERE id = $1`, target.ID).Scan(&phone)
	if phone != newPhone {
		t.Errorf("الرقمُ بعد الموافقة %q", phone)
	}
}

// ── ٥ · كلمةُ السرّ يولّدها النظام ─────────────────────────────────────

func TestACC_ResetIssuesSystemTempPasswordWithExpiry(t *testing.T) {
	h := New(t)
	staff := h.NewUser("admin")
	target := h.NewUser("driver")

	r := h.POST("/api/v1/admin/users/"+target.ID+"/password", staff.Token,
		map[string]any{"password": "TypedByStaff1"})
	if r.Code != 200 {
		t.Fatalf("الإعادة: %s", r)
	}
	var hash string
	var must bool
	var exp *time.Time
	_ = h.Pool.QueryRow(ctxBG(), `SELECT password_hash, must_change_password, temp_password_expires_at
		FROM users WHERE id = $1`, target.ID).Scan(&hash, &must, &exp)
	if okTyped, _ := auth.VerifyPassword("TypedByStaff1", hash); okTyped {
		t.Errorf("**قُبلت كلمةٌ كتبها الموظّف**")
	}
	if !must || exp == nil {
		t.Fatalf("**لا إجبارَ ولا مهلة**: must=%v exp=%v", must, exp)
	}
	if d := time.Until(*exp); d < 71*time.Hour || d > 73*time.Hour {
		t.Errorf("**المهلةُ %v والمنتظَر ٧٢ ساعة**", d)
	}
	var live int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM refresh_tokens WHERE user_id = $1
		AND revoked_at IS NULL AND expires_at > now()`, target.ID).Scan(&live)
	if live != 0 {
		t.Errorf("الجلساتُ باقيةٌ بعد الإعادة (%d)", live)
	}
	// **ولا يعيد الموظّفُ كلمتَه من ملفّه.**
	if s := h.POST("/api/v1/admin/users/"+staff.ID+"/password", staff.Token, map[string]any{}); s.Code < 400 {
		t.Errorf("**أعاد الموظّفُ كلمتَه فأخرج نفسه**: %s", s)
	}
}

func TestACC_ExpiredTempPasswordDoesNotOpen(t *testing.T) {
	h := New(t)
	u := h.NewUser("driver")
	hash, _ := auth.HashPassword("Temp12345x")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET password_hash = $2, must_change_password = true,
		temp_password_expires_at = now() - interval '1 hour' WHERE id = $1`, u.ID, hash); err != nil {
		t.Fatal(err)
	}
	r := h.POST("/api/v1/auth/login", "", map[string]any{"phone": u.Phone, "password": "Temp12345x"})
	if r.Err() != "temp_password_expired" {
		t.Fatalf("**كلمةٌ مؤقّتةٌ منتهيةٌ فتحت الحساب**: %s", r)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET temp_password_expires_at = now() + interval '1 hour'
		WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if ok := h.POST("/api/v1/auth/login", "", map[string]any{"phone": u.Phone, "password": "Temp12345x"}); ok.Code != 200 {
		t.Fatalf("كلمةٌ مؤقّتةٌ سارية رُدّت: %s", ok)
	}
}

func TestACC_TempPasswordForcesChangeEvenWhenSwitchIsOff(t *testing.T) {
	h := New(t)
	h.Setting("security.force_password_change", "false")
	u := h.NewUser("customer")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET must_change_password = true,
		temp_password_expires_at = now() + interval '10 hours' WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	r := h.GET("/api/v1/my/orders", u.Token)
	if r.Code != 403 {
		t.Errorf("**كلمةُ النظام المؤقّتة لم تُجبَر على التبديل**: %s", r)
	}
	// **وكلمةٌ وضعها طرفٌ ثالثٌ قبل اليوم (بلا مهلة) تتبع الزرّ كما كانت.**
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET temp_password_expires_at = NULL WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if r2 := h.GET("/api/v1/my/orders", u.Token); r2.Code == 403 {
		t.Errorf("الزرُّ المطفأُ لم يُحترم للكلمات القديمة: %s", r2)
	}
}

// ── ١٧ · قفلُ ما بعد الحادث ────────────────────────────────────────────

func TestACC_AccidentLocksShiftUntilOpsConfirms(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	rel := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
		map[string]any{"reason": "accident", "note": "صدمة خفيفة"})
	if rel.Code >= 400 {
		t.Fatalf("التركُ بسبب حادث: %s", rel)
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Err() != "accident_check_required" {
		t.Fatalf("**فتح السائقُ دوامَه بعد الحادث**: %s", r)
	}
	ops := h.NewUser("operations")
	ok := h.POST("/api/v1/admin/users/"+drv.ID+"/driver-ok", ops.Token, map[string]any{"note": "اتصلت به وهو بخير"})
	if ok.Code != 200 {
		t.Fatalf("«السائقُ بخير»: %s", ok)
	}
	var by string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(accident_cleared_by::text, '') FROM users WHERE id = $1`, drv.ID).Scan(&by)
	if by != ops.ID {
		t.Errorf("**لم يُسجَّل من أكّد** (%q)", by)
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Err() == "accident_check_required" {
		t.Errorf("**بقي مقفولاً بعد التأكيد**: %s", r)
	}
}

// ── ١٩ · ٢٠ · ٢١ · المندوب ─────────────────────────────────────────────

func TestACC_SuspendedRepCodeRejectedForNewStores(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	rep := f.RepAccount()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1`, rep.ID); err != nil {
		t.Fatal(err)
	}
	cat := f.category()
	r := h.POST("/api/v1/admin/merchants", admin.Token, map[string]any{
		"name": "متجر برمز موقوف", "category_id": cat, "lat": 35.95, "lng": 39.01,
		"owner_phone": uniqPhone(), "owner_name": "صاحب", "sales_rep_code": rep.InviteCode,
	})
	if r.Err() != "rep_inactive" {
		t.Fatalf("**رمزُ المندوب الموقوف قُبل**: %s", r)
	}
}

func TestACC_SuspendedRepCommissionHeldThenReleased(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("sales.commission_source", `"platform_commission"`)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "0")
	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1`, rep.ID); err != nil {
		t.Fatal(err)
	}
	item := h.NewItemFor(m, 5000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))

	conserved := func(stage string) {
		var net, cash int64
		_ = h.Pool.QueryRow(ctxBG(), `
			SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1), 0),
			       COALESCE((SELECT sum(amount) FROM driver_cash_entries
			                 WHERE ref = $1 AND kind = 'order_collection'), 0)`, oid).Scan(&net, &cash)
		if net != cash {
			t.Errorf("%s: **FI-06.a خُرق** صافي %d ≠ نقد %d", stage, net, cash)
		}
	}
	if c := ledgerByRef(t, h, oid, "commission"); c != 0 {
		t.Fatalf("**المندوبُ الموقوفُ قبض عمولتَه** (%d)", c)
	}
	var held int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM rep_held_commissions
		WHERE rep_id = $1 AND status = 'held'`, rep.ID).Scan(&held)
	if held != 100 {
		t.Fatalf("**العمولةُ لم تُحجز** (%d)", held)
	}
	conserved("محجوزة")

	admin := h.NewUser("admin")
	if r := h.PATCH("/api/v1/admin/users/"+rep.ID, admin.Token, map[string]any{"status": "active"}); r.Code != 200 {
		t.Fatalf("التفعيل: %s", r)
	}
	if c := ledgerByRef(t, h, oid, "commission"); c != 100 {
		t.Errorf("**المحجوزُ لم يُصرف بعد التفعيل** (%d)", c)
	}
	conserved("بعد الصرف")
}

func TestACC_BannedRepStoresMoveToPlatform(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	admin := h.NewUser("admin")
	if r := h.PATCH("/api/v1/admin/users/"+rep.ID, admin.Token,
		map[string]any{"status": "blocked", "status_reason": "احتيال"}); r.Code != 200 {
		t.Fatalf("الحظر: %s", r)
	}
	var rid *string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT sales_rep_user_id::text FROM merchants WHERE id = $1`, m.ID).Scan(&rid)
	if rid != nil {
		t.Errorf("**متجرُ المندوب المحظور بقي باسمه**")
	}
}

// ── ٢٢ · ٢٣ · المتجر ───────────────────────────────────────────────────

func TestACC_StoreBanNeedsReasonAndOpenNeverLiftsIt(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	admin := h.NewUser("admin")
	base := "/api/v1/admin/merchants/" + m.ID
	if r := h.POST(base+"/suspend", admin.Token, map[string]any{"suspended": true}); r.Err() != "reason_required" {
		t.Fatalf("**حظرٌ بلا سبب**: %s", r)
	}
	if r := h.POST(base+"/suspend", admin.Token, map[string]any{"suspended": true, "note": "مخالفات متكررة"}); r.Code != 200 {
		t.Fatalf("الحظر: %s", r)
	}
	if r := h.PATCH(base, admin.Token, map[string]any{"status": "active"}); r.Err() != "merchant_ban_locked" {
		t.Fatalf("**«فتح» رفع الحظر**: %s", r)
	}
	var st string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM merchants WHERE id = $1`, m.ID).Scan(&st)
	if st != "suspended" {
		t.Errorf("الحال %q", st)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE user_id = $1`, m.Owner.ID).Scan(&n)
	if n == 0 {
		t.Errorf("**صاحبُ المتجر لم يُبلَّغ بالحظر**")
	}
}

func TestACC_OwnerWithNoStoresLosesMerchantRole(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	other := f.NewUserWith("merchant")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, m.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	var has bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1
		AND role_code = 'merchant')`, m.Owner.ID).Scan(&has)
	if has {
		t.Errorf("**صاحبٌ بلا متجرٍ بقي يحمل دورَ المتجر**")
	}
}

// ── ٧ · ١١ · القائمة والتصدير ──────────────────────────────────────────

func TestACC_SearchFindsOwnerByStoreName(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	admin := h.NewUser("admin")
	got := h.GET("/api/v1/admin/users?query="+urlQ(m.Name), admin.Token).JSON()
	users, _ := got["users"].([]any)
	found := false
	for _, u := range users {
		um, _ := u.(map[string]any)
		if um["id"] == m.Owner.ID {
			found = true
			names, _ := um["store_names"].([]any)
			if len(names) == 0 || names[0] != m.Name {
				t.Errorf("اسمُ المتجر لا يظهر تحت صاحبه: %v", um["store_names"])
			}
		}
	}
	if !found {
		t.Errorf("**البحثُ باسم المتجر لم يجد صاحبَه**")
	}
}

func TestACC_CustomerCardCountsCustomersOnly(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	drv := f.NewUserWith("driver", WithRoles("customer"))
	got := h.GET("/api/v1/admin/users?role=customer&query="+urlQ(drv.Phone), admin.Token).JSON()
	if total, _ := got["total"].(float64); total != 0 {
		t.Errorf("**سائقٌ عُدّ زبوناً** (%v)", total)
	}
}

func TestACC_ExportAdminOnlyArabicInjectionSafeNoTruncation(t *testing.T) {
	h := New(t)
	f := h.Factory()
	if need, _ := authz.LookupAdmin("GET", "/users/export"); need != authz.UsersExport {
		t.Fatalf("التصديرُ بقدرة %q", need)
	}
	ops := h.NewUser("operations")
	if r := h.GET("/api/v1/admin/users/export", ops.Token); r.Code != 403 {
		t.Errorf("**صدّرت العمليّات**: %d", r.Code)
	}
	tag := "صدر" + uniq("x")
	for i := 0; i < 105; i++ {
		f.NewUserWith("customer", Named(tag+" "+itoa(i)))
	}
	evil := f.NewUserWith("customer", Named("=HYPERLINK(\"x\") "+tag))
	_ = evil
	admin := h.NewUser("admin")
	r := h.GET("/api/v1/admin/users/export?query="+urlQ(tag), admin.Token)
	if r.Code != 200 {
		t.Fatalf("التصدير: %s", r)
	}
	body := string(r.Body)
	if !strings.Contains(body, "الاسم") || !strings.Contains(body, "زبون") {
		t.Errorf("**العناوينُ أو الأدوارُ ليست عربيّة**")
	}
	if strings.Contains(body, "\n=HYPERLINK") || !strings.Contains(body, "'=HYPERLINK") {
		t.Errorf("**صيغةُ إكسل خرجت كما هي**")
	}
	lines := strings.Count(strings.TrimSpace(body), "\n")
	if lines < 106 {
		t.Errorf("**التصديرُ قُصّ**: %d سطراً والمنتظَر ١٠٦", lines)
	}
}

// ── ١٤ · الزبونُ الموقوفُ يراسل سائقَ طلبه الحيّ ──────────────────────────

func TestACC_SuspendedCustomerCanChatOnLiveOrder(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	h.driverOf(oid)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1`, cust.ID); err != nil {
		t.Fatal(err)
	}
	h.Redis().Del(ctxBG(), "ustatus:"+cust.ID)
	if r := h.GET("/api/v1/orders/"+oid+"/messages", cust.Token); r.Code == 403 {
		t.Errorf("**الزبونُ الموقوفُ لا يقرأ حديثَ طلبه الحيّ**: %s", r)
	}
	if r := h.GET("/api/v1/my/orders", cust.Token); r.Code != 403 {
		t.Errorf("الموقوفُ فتح غيرَ طلبه الحيّ: %d", r.Code)
	}
}

// ── ١٢ · ١٣ · الملاحظاتُ ومنعُ النقد ──────────────────────────────────

func TestACC_NotesAreAppendOnlyWithAuthor(t *testing.T) {
	h := New(t)
	f := h.Factory()
	// **والعمليّاتُ لم تعد تقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — فموظّفُ الدعم
	// يقيس العقدَ نفسَه: يقرأ الحسابَ ولا يملك المال.
	ops := h.NewUser("customer_support")
	target := f.NewUserWith("customer")
	if r := h.POST("/api/v1/admin/users/"+target.ID+"/notes", ops.Token, map[string]any{"body": "اتصل يشتكي"}); r.Code != 201 {
		t.Fatalf("الملاحظة: %s", r)
	}
	var author string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT author_id::text FROM user_notes WHERE user_id = $1`, target.ID).Scan(&author)
	if author != ops.ID {
		t.Errorf("بلا كاتب")
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE user_notes SET body = 'محو' WHERE user_id = $1`, target.ID); err == nil {
		t.Errorf("**الملاحظةُ عُدّلت** — والسجلُّ لا يُمحى")
	}
}

func TestACC_CashBanVisibleAndOnlyAdminLiftsWithReason(t *testing.T) {
	h := New(t)
	h.Setting("customers.cash_ban_failures", "1")
	h.Setting("customers.cash_ban_days", "30")
	cust := h.Customer()
	item := h.NewItem(2000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'failed', fault = 'customer' WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO order_events (order_id, from_status, to_status)
		VALUES ($1, 'on_the_way', 'failed')`, oid); err != nil {
		t.Fatal(err)
	}
	admin := h.NewUser("admin")
	path := "/api/v1/admin/users/" + cust.ID + "/cash-ban"
	if b := h.GET(path, admin.Token).JSON(); b["blocked"] != true || b["until"] == nil {
		t.Fatalf("**المنعُ لا يظهر بسببه وانتهائه**: %v", b)
	}
	ops := h.NewUser("operations")
	if r := h.POST(path+"/lift", ops.Token, map[string]any{"reason": "x"}); r.Code != 403 {
		t.Errorf("**رفعت العمليّاتُ المنع**: %s", r)
	}
	if r := h.POST(path+"/lift", admin.Token, map[string]any{"reason": " "}); r.Err() != "reason_required" {
		t.Errorf("رفعٌ بلا سبب: %s", r)
	}
	if r := h.POST(path+"/lift", admin.Token, map[string]any{"reason": "خطأ من السائق"}); r.Code != 200 {
		t.Fatalf("الرفع: %s", r)
	}
	if b := h.GET(path, admin.Token).JSON(); b["blocked"] != false {
		t.Errorf("**المنعُ باقٍ بعد رفعه**: %v", b)
	}
}

func urlQ(v string) string { return url.QueryEscape(v) }

// adminWallet **حركةُ محفظةٍ يدويّةٌ كما صارت** (قرارُ المالك ٢٠٢٦-١٠-٠٤): يقترحها موظّفٌ
// ماليٌّ ويوافق عليها `admin` — ويُرجع ردَّ الموافقة (أو ردَّ الاقتراح إن رُدّ).
func (h *Harness) adminWallet(admin *User, uid string, body map[string]any) Res {
	h.T.Helper()
	proposer := h.NewUser("finance")
	r := h.POSTKey("/api/v1/admin/users/"+uid+"/wallet", proposer.Token, uniq("w"), body)
	if r.Code >= 400 {
		return r
	}
	id, _ := r.JSON()["request_id"].(string)
	return h.POST("/api/v1/admin/wallet-requests/"+id+"/approve", admin.Token, map[string]any{})
}

// setKnownPassword **كلمةٌ معروفةٌ لاختبار** — الإدارةُ لم تعد تكتب كلمات (قرارُ المالك
// ٢٠٢٦-١٠-٠٤)، فتُكتب بصمتُها هنا كما يكتبها المحرّكُ بلا مهلةٍ ولا إجبار.
func setKnownPassword(t *testing.T, h *Harness, userID, pw string) {
	t.Helper()
	hash, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("بصمُ الكلمة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET password_hash = $2,
		must_change_password = false, temp_password_expires_at = NULL WHERE id = $1::uuid`,
		userID, hash); err != nil {
		t.Fatalf("ضبطُ الكلمة: %v", err)
	}
}

// TestACC_DeletedAccountsHaveTheirOwnBox **المحذوفُ لا يُخلط بالأحياء** (قرارُ المالك
// ٢٠٢٦-١٠-٠٦): يغيب عن القائمة العامّة ويُعدّ في بطاقته ويظهر بترشيحها.
func TestACC_DeletedAccountsHaveTheirOwnBox(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	gone := f.NewUserWith("customer")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET status = 'deleted', full_name = $2 WHERE id = $1`, gone.ID, "محذوف-"+gone.ID[:8]); err != nil {
		t.Fatalf("حذف: %v", err)
	}
	has := func(q string) bool {
		got := h.GET("/api/v1/admin/users?"+q, admin.Token).JSON()
		users, _ := got["users"].([]any)
		for _, u := range users {
			if um, _ := u.(map[string]any); um["id"] == gone.ID {
				return true
			}
		}
		return false
	}
	if has("query=" + urlQ("محذوف-"+gone.ID[:8])) {
		t.Errorf("**المحذوفُ ظهر في القائمة العامّة**")
	}
	if !has("status=deleted&query=" + urlQ("محذوف-"+gone.ID[:8])) {
		t.Errorf("**المحذوفُ لا يظهر في مربّعه**")
	}
	counts := h.GET("/api/v1/admin/users/stats", admin.Token).JSON()
	roles, _ := counts["roles"].(map[string]any)
	if n, _ := roles["deleted"].(float64); n < 1 {
		t.Errorf("**بطاقةُ المحذوفة لا تعدّه**: %v", roles["deleted"])
	}
}
