package server

// **سجلُّ الأحداث بعد فحص ٢٠٢٦-١٠-٠٤** — ستّةُ قراراتٍ للمالك تُقاس هنا.
//
//	١ · المبالغُ تُحذف في الخادم عمّن لا يملك قراءةَ المال
//	٢ · الدخولُ يُحذف بعد مدّته وما سواه يبقى
//	٣ · «الأفعال الحساسة» تبويبٌ افتراضيّ والدخولُ في تبويبه
//	٤ · رفضُ طلب الانضمام وإرجاعُه · الإغلاقُ الطارئ وإعادةُ الفتح · فتحُ
//	    تذكرةٍ والردُّ عليها — كلُّها تُكتب
//	٥ · السجلُّ إضافةٌ فقط في القاعدة نفسِها
//	٦ · التصديرُ يُكتب في السجلّ نفسِه
//
// ومعها: البحثُ والفلاتر ويومُ دمشق.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// auditGet ينادي معالِجَ السجلّ بقدراتٍ وأدوارٍ بعينها.
func auditGet(t *testing.T, f *driverFixture, h http.HandlerFunc, userID string,
	roles, caps []string, query string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/x?"+query, nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	ctx = context.WithValue(ctx, ctxCaps, caps)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	if w.Code != http.StatusOK {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	return out.Data
}

func auditEntries(t *testing.T, data map[string]any) []map[string]any {
	t.Helper()
	raw, _ := data["entries"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, e := range raw {
		m, _ := e.(map[string]any)
		out = append(out, m)
	}
	return out
}

// auditInsert سطرٌ مصنوعٌ في السجلّ بوقتٍ بعينه.
func auditInsert(t *testing.T, f *driverFixture, actor, action, entity, entityID, details string, at time.Time) {
	t.Helper()
	var who any
	if actor != "" {
		who = actor
	}
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details, created_at)
		VALUES ($1::uuid, $2, $3, $4, '1.1.1.1', $5::jsonb, $6)`,
		who, action, entity, entityID, details, at); err != nil {
		t.Fatalf("تعذّر السطر %s: %v", action, err)
	}
}

// waitAudit ينتظر سطراً يُكتب في الخلفيّة (`s.audit`) — ثلاثَ ثوانٍ لا أكثر.
func waitAudit(t *testing.T, f *driverFixture, entity, entityID, action string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var actor *string
		var details []byte
		err := f.pool.QueryRow(context.Background(), `
			SELECT actor_user_id::text, details FROM audit_log
			WHERE entity = $1 AND entity_id = $2 AND action = $3
			ORDER BY id DESC LIMIT 1`, entity, entityID, action).Scan(&actor, &details)
		if err == nil {
			out := map[string]any{}
			_ = json.Unmarshal(details, &out)
			if actor != nil {
				out["_actor"] = *actor
			}
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("لم يُكتب %s على %s %s — **فعلٌ بلا أثر**", action, entity, entityID)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func callWithID(f *driverFixture, h http.HandlerFunc, id, userID string, roles []string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// ── القرارُ الرابع ─────────────────────────────────────────────────────

// TestAuditLog_MissingActionsAreLogged **ستّةُ أفعالٍ كانت تمضي بلا أثر.**
func TestAuditLog_MissingActionsAreLogged(t *testing.T) {
	f := newDriverFixture(t, 0)
	ops := f.armOps(t)
	ctx := context.Background()

	// ── رفضُ طلب الانضمام وإرجاعُه ─────────────────────────────────
	rep := testdb.NewUser(t, f.pool, "sales")
	var lead string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchant_leads (store_name, phone, sales_rep_user_id)
		VALUES ('متجر فحص السجل', '+963911000111', $1) RETURNING id`, rep).Scan(&lead); err != nil {
		t.Fatalf("تعذّر طلبُ الانضمام: %v", err)
	}
	if w := callWithID(f, f.srv.handleAdminLeadStatus, lead, ops, []string{"ops"},
		`{"status":"rejected","note":"الموقع خارج التغطية"}`); w.Code != http.StatusOK {
		t.Fatalf("الرفضُ ردّ %d: %s", w.Code, w.Body.String())
	}
	got := waitAudit(t, f, "lead", lead, "ops.lead_rejected")
	if got["_actor"] != ops || got["note"] != "الموقع خارج التغطية" {
		t.Fatalf("سطرُ الرفض %v — والمتوقّعُ الفاعلُ %s والسبب", got, ops)
	}
	if w := callWithID(f, f.srv.handleAdminLeadStatus, lead, ops, []string{"ops"},
		`{"status":"new"}`); w.Code != http.StatusOK {
		t.Fatalf("الإرجاعُ ردّ %d: %s", w.Code, w.Body.String())
	}
	waitAudit(t, f, "lead", lead, "ops.lead_reopened")

	// ── الإغلاقُ الطارئ وإعادةُ الفتح ───────────────────────────────
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	if w := callWithID(f, f.srv.handleMerchantEmergency, f.merchantID, owner, []string{"merchant"},
		`{"closed":true}`); w.Code != http.StatusOK {
		t.Fatalf("الإغلاقُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if got := waitAudit(t, f, "merchant", f.merchantID, "merchant.emergency_close"); got["_actor"] != owner {
		t.Fatalf("الإغلاقُ بلا فاعله: %v", got)
	}
	if w := callWithID(f, f.srv.handleMerchantEmergency, f.merchantID, owner, []string{"merchant"},
		`{"closed":false}`); w.Code != http.StatusOK {
		t.Fatalf("إعادةُ الفتح ردّت %d: %s", w.Code, w.Body.String())
	}
	waitAudit(t, f, "merchant", f.merchantID, "merchant.emergency_reopen")

	// ── فتحُ تذكرةٍ من اللوحة والردُّ عليها ─────────────────────────
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	customer := testdb.NewUser(t, f.pool, "customer")
	var phone string
	if err := f.pool.QueryRow(ctx, `SELECT phone::text FROM users WHERE id = $1`, customer).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	w := callWithID(f, f.srv.handleCreateTicket, "", ops, []string{"ops"},
		`{"customer_phone":"`+phone+`","subject":"فحص السجل","body":"وصل ناقصاً"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("فتحُ التذكرة ردّ %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Data.ID == "" {
		t.Fatalf("لا معرّفَ للتذكرة: %s", w.Body.String())
	}
	waitAudit(t, f, "ticket", created.Data.ID, "ops.ticket_created")
	if w := callWithID(f, f.srv.handleTicketReply, created.Data.ID, ops, []string{"ops"},
		`{"body":"نعتذر، يُعاد الصنف"}`); w.Code != http.StatusOK {
		t.Fatalf("الردُّ ردّ %d: %s", w.Code, w.Body.String())
	}
	if got := waitAudit(t, f, "ticket", created.Data.ID, "ops.ticket_replied"); got["note"] != "نعتذر، يُعاد الصنف" {
		t.Fatalf("الردُّ بلا نصّه: %v", got)
	}
}

// ── القرارُ الأوّل ─────────────────────────────────────────────────────

// TestAuditLog_MoneyHiddenWithoutFinanceRead **المبلغُ يُحذف في الخادم.**
func TestAuditLog_MoneyHiddenWithoutFinanceRead(t *testing.T) {
	f := newDriverFixture(t, 0)
	actor := testdb.NewUser(t, f.pool, "finance")
	now := time.Now()
	auditInsert(t, f, actor, "finance.wallet_apply", "user", actor,
		`{"amount": 5000, "kind": "topup", "note": "تعويض"}`, now)
	key := fininv.FinancialSettings[0]
	auditInsert(t, f, actor, "admin.setting_update", "setting", key,
		`{"before": 10, "after": 12}`, now)

	q := "group=all&actor=" + actor
	find := func(es []map[string]any, action string) map[string]any {
		for _, e := range es {
			if e["action"] == action {
				return e
			}
		}
		t.Fatalf("لا سطرَ %s", action)
		return nil
	}

	// **الثقةُ والأمان: يرى السجلّ ولا يرى المبالغ.**
	data := auditGet(t, f, f.srv.handleAdminAudit, actor, []string{"trust_safety"},
		[]string{"audit.read"}, q)
	if data["money_visible"] != false {
		t.Fatalf("money_visible = %v لمن لا يملك قراءةَ المال", data["money_visible"])
	}
	es := auditEntries(t, data)
	w := find(es, "finance.wallet_apply")
	d, _ := w["details"].(map[string]any)
	if _, ok := d["amount"]; ok || w["redacted"] != true {
		t.Fatalf("المبلغُ وصل من لا يملك قراءةَ المال: %v", w)
	}
	if d["note"] != "تعويض" || d["kind"] != "topup" {
		t.Fatalf("حُذف ما ليس مبلغاً: %v", d)
	}
	s := find(es, "admin.setting_update")
	sd, _ := s["details"].(map[string]any)
	if _, ok := sd["after"]; ok {
		t.Fatalf("قيمةُ إعدادٍ ماليٍّ (%s) وصلت: %v", key, sd)
	}

	// **والتصديرُ يحذف كما تحذف الصفحة** — لمن مُنح التصديرَ ولا يقرأ المال.
	exp := auditGet(t, f, f.srv.handleAdminAuditExport, actor, []string{"trust_safety"},
		[]string{"audit.read", "audit.export"}, q)
	for _, e := range auditEntries(t, exp) {
		if dd, _ := e["details"].(map[string]any); dd["amount"] != nil {
			t.Fatalf("التصديرُ حمل المبلغ: %v", e)
		}
	}

	// **ومن ملك قراءةَ المال يراه.**
	data = auditGet(t, f, f.srv.handleAdminAudit, actor, []string{"finance"},
		[]string{"audit.read", "finance.read"}, q)
	w = find(auditEntries(t, data), "finance.wallet_apply")
	if d, _ := w["details"].(map[string]any); d["amount"] != float64(5000) || w["redacted"] == true {
		t.Fatalf("المالُ لم يرَ المبلغ: %v", w)
	}
	// **وأدمنُ المنصّة كذلك.**
	data = auditGet(t, f, f.srv.handleAdminAudit, actor, []string{"admin"},
		[]string{"audit.read"}, q)
	w = find(auditEntries(t, data), "finance.wallet_apply")
	if d, _ := w["details"].(map[string]any); d["amount"] != float64(5000) {
		t.Fatalf("الأدمنُ لم يرَ المبلغ: %v", w)
	}
}

// ── القرارُ الخامس ─────────────────────────────────────────────────────

// TestAuditLog_AppendOnlyAtDatabase **لا تعديلَ ولا حذفَ ولا إفراغ.**
func TestAuditLog_AppendOnlyAtDatabase(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	actor := testdb.NewUser(t, f.pool, "admin")
	auditInsert(t, f, actor, "finance.wallet_apply", "user", actor, `{"amount": 1}`, time.Now())
	auditInsert(t, f, actor, "auth.otp_login", "user", actor, `{}`, time.Now())

	for _, sql := range []string{
		`UPDATE audit_log SET details = '{}' WHERE actor_user_id = $1`,
		`DELETE FROM audit_log WHERE actor_user_id = $1`,
		`DELETE FROM audit_log WHERE actor_user_id = $1 AND action = 'auth.otp_login'`,
	} {
		if _, err := f.pool.Exec(ctx, sql, actor); err == nil ||
			!strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s مرّ (%v) — **والسجلُّ دليلٌ لا يُمحى**", sql, err)
		}
	}
	if _, err := f.pool.Exec(ctx, `TRUNCATE audit_log`); err == nil {
		t.Fatal("أُفرغ السجلُّ كلُّه")
	}

	// **والرايةُ بيدٍ لا تحذف إلّا سطرَ دخول** — سطرُ المال يبقى.
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('rahalgo.audit_prune', 'on', true)`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM audit_log WHERE actor_user_id = $1 AND action = 'finance.wallet_apply'`, actor); err == nil {
		t.Fatal("سطرُ مالٍ حُذف بالراية — والحارسُ يفحص الفعلَ لا الرايةَ وحدَها")
	}
	_ = tx.Rollback(ctx)

	// **والتنظيفُ لا يقبل أقلَّ من ثلاثين يوماً.**
	if _, err := f.pool.Exec(ctx, `SELECT audit_prune_sessions(5)`); err == nil {
		t.Fatal("قُبلت مدّةُ خمسة أيّام")
	}

	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE actor_user_id = $1`, actor).Scan(&n)
	if n != 2 {
		t.Fatalf("بقي %d سطراً لا 2", n)
	}
}

// ── القرارُ الثاني ─────────────────────────────────────────────────────

// TestAuditLog_RetentionPrunesOnlyOldSessionRows **الدخولُ يُحذف بعد مدّته.**
func TestAuditLog_RetentionPrunesOnlyOldSessionRows(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	// **القائمتان واحدة** — الشيفرةُ والقاعدة.
	var sqlList []string
	if err := f.pool.QueryRow(ctx, `SELECT audit_session_actions()`).Scan(&sqlList); err != nil {
		t.Fatal(err)
	}
	goList := append([]string(nil), auditSessionActions...)
	sort.Strings(sqlList)
	sort.Strings(goList)
	if strings.Join(sqlList, ",") != strings.Join(goList, ",") {
		t.Fatalf("قائمتا الدخول افترقتا:\nالقاعدة %v\nالشيفرة %v", sqlList, goList)
	}

	actor := testdb.NewUser(t, f.pool, "ops")
	day := 24 * time.Hour
	now := time.Now()
	auditInsert(t, f, actor, "auth.otp_login", "user", "old-login", `{}`, now.Add(-100*day))
	auditInsert(t, f, actor, "auth.refresh", "user", "old-refresh", `{}`, now.Add(-95*day))
	auditInsert(t, f, actor, "auth.otp_login", "user", "new-login", `{}`, now.Add(-10*day))
	auditInsert(t, f, actor, "finance.wallet_apply", "user", "old-money", `{"amount": 1}`, now.Add(-400*day))
	auditInsert(t, f, actor, "auth.phone_change", "user", "old-phone", `{}`, now.Add(-400*day))
	auditInsert(t, f, actor, "admin.role_grant", "user", "old-role", `{}`, now.Add(-400*day))

	left := func() []string {
		rows, err := f.pool.Query(ctx, `SELECT entity_id FROM audit_log WHERE actor_user_id = $1 ORDER BY entity_id`, actor)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			out = append(out, s)
		}
		return out
	}

	f.setSetting(t, auditRetentionKey, 120)
	t.Cleanup(func() { f.setSetting(t, auditRetentionKey, 90) })
	if _, err := f.srv.PruneAuditSessions(ctx); err != nil {
		t.Fatalf("التنظيفُ سقط: %v", err)
	}
	if got := strings.Join(left(), ","); got != "new-login,old-login,old-money,old-phone,old-refresh,old-role" {
		t.Fatalf("بمدّة ١٢٠ بقي %s — حُذف ما لم يبلغ مدّته", got)
	}

	f.setSetting(t, auditRetentionKey, 90)
	if _, err := f.srv.PruneAuditSessions(ctx); err != nil {
		t.Fatalf("التنظيفُ سقط: %v", err)
	}
	if got := strings.Join(left(), ","); got != "new-login,old-money,old-phone,old-role" {
		t.Fatalf("بمدّة ٩٠ بقي %s — والمتوقّعُ حذفُ الدخول القديم وحدَه", got)
	}
}

// ── القرارُ الثالث والبحثُ والفلاتر ───────────────────────────────────

// TestAuditLog_SearchFiltersAndDamascusDay **بحثٌ وشخصٌ وفعلٌ وتبويبٌ ويومُ دمشق.**
func TestAuditLog_SearchFiltersAndDamascusDay(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	actor, _, orderID := twoCustomers(t, f)
	tag := strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	name := "فاحص السجل " + tag
	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name = $2 WHERE id = $1`, actor, name); err != nil {
		t.Fatal(err)
	}
	var number int64
	var phone string
	_ = f.pool.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number)
	_ = f.pool.QueryRow(ctx, `SELECT phone::text FROM users WHERE id = $1`, actor).Scan(&phone)

	now := time.Now()
	auditInsert(t, f, actor, "ops.order_transition", "order", orderID, `{"to": "cancelled"}`, now)
	auditInsert(t, f, actor, "finance.wallet_apply", "user", actor, `{"amount": 10}`, now)
	auditInsert(t, f, actor, "auth.otp_login", "user", actor, `{}`, now)
	auditInsert(t, f, actor, "auth.refresh", "user", actor, `{}`, now)
	auditInsert(t, f, actor, "admin.merchant_update", "merchant", f.merchantID, `{}`, now)
	// **ساعةٌ ونصفٌ بعد منتصف ليل دمشق** = ١١ كانون الثاني بدمشق، ١٠ منه بغرينتش.
	late := time.Date(2026, 1, 10, 22, 30, 0, 0, time.UTC)
	auditInsert(t, f, actor, "admin.zone_create", "zone", "z", `{}`, late)

	admin := []string{"admin"}
	caps := []string{"audit.read"}
	count := func(q string) int {
		t.Helper()
		return len(auditEntries(t, auditGet(t, f, f.srv.handleAdminAudit, actor, admin, caps, q)))
	}
	by := "&actor=" + actor

	// **الافتراضيُّ «الأفعال الحساسة»**: الانتقالُ والمحفظة لا غير.
	data := auditGet(t, f, f.srv.handleAdminAudit, actor, admin, caps, strings.TrimPrefix(by, "&"))
	if data["group"] != "sensitive" || len(auditEntries(t, data)) != 2 {
		t.Fatalf("الافتراضيُّ %v بـ%d سطراً — والمتوقّعُ sensitive بسطرين",
			data["group"], len(auditEntries(t, data)))
	}
	if n := count("group=login" + by); n != 1 {
		t.Fatalf("تبويبُ الدخول %d لا 1 (والتجديدُ مخفيّ)", n)
	}
	if n := count("group=all" + by); n != 5 {
		t.Fatalf("الكلُّ %d لا 5", n)
	}
	if n := count("group=all&action=finance.wallet_apply" + by); n != 1 {
		t.Fatalf("فلترُ الفعل %d لا 1", n)
	}
	if n := count("group=all&actor=system&q=" + tag); n != 0 {
		t.Fatalf("فلترُ «النظام» جاء بسطورِ شخص: %d", n)
	}

	// **البحثُ بالاسم · بالهاتف المحلّيّ · برقم الطلب · بالاسم العربيّ للفعل.**
	if n := count("group=all&q=" + tag); n != 5 {
		t.Fatalf("البحثُ بالاسم %d لا 5", n)
	}
	local := "0" + strings.TrimPrefix(phone, "+963")
	if n := count("group=all&q=" + local + by); n != 5 {
		t.Fatalf("البحثُ بالهاتف %s جاء بـ%d لا 5", local, n)
	}
	es := auditEntries(t, auditGet(t, f, f.srv.handleAdminAudit, actor, admin, caps,
		"group=all&q=%23"+strconv.FormatInt(number, 10)+by))
	if len(es) != 1 || es[0]["action"] != "ops.order_transition" || es[0]["target_number"] != float64(number) {
		t.Fatalf("البحثُ برقم الطلب #%d جاء بـ%v", number, es)
	}
	if n := count("group=all&q=" + url.QueryEscape("شحن") + "&q_actions=finance.wallet_apply" + by); n != 1 {
		t.Fatalf("البحثُ باسم الفعل %d لا 1", n)
	}

	// **والهدفُ يُقرأ**: اسمُ المتجر في سطره.
	for _, e := range auditEntries(t, auditGet(t, f, f.srv.handleAdminAudit, actor, admin, caps,
		"group=merchant"+by)) {
		if e["entity"] == "merchant" && e["target_label"] == "" {
			t.Fatalf("سطرُ المتجر بلا اسمه: %v", e)
		}
	}

	// **ويومُ دمشق لا يومُ القاعدة** (المشكلةُ التاسعة).
	if n := count("group=all&from=2026-01-11&to=2026-01-11" + by); n != 1 {
		t.Fatalf("يومُ ١١ بدمشق جاء بـ%d لا 1 — **قُرئ اليومُ بغير منطقة المنصّة**", n)
	}
	if n := count("group=all&from=2026-01-10&to=2026-01-10" + by); n != 0 {
		t.Fatalf("يومُ ١٠ بدمشق جاء بـ%d لا 0", n)
	}

	// **والعدُّ محدودٌ بسقف** — ويقول إن بلغه.
	if data["total_capped"] != false {
		t.Fatalf("total_capped = %v على سطرين", data["total_capped"])
	}
}

// ── القرارُ السادس ─────────────────────────────────────────────────────

// TestAuditLog_ExportWritesItself **من صدّر السجلَّ يُكتب فيه.**
func TestAuditLog_ExportWritesItself(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	exporter := testdb.NewUser(t, f.pool, "admin")
	subject := testdb.NewUser(t, f.pool, "ops")
	auditInsert(t, f, subject, "ops.order_assign", "order", "x", `{}`, time.Now())

	data := auditGet(t, f, f.srv.handleAdminAuditExport, exporter, []string{"admin"},
		[]string{"audit.read", "audit.export"}, "group=all&actor="+subject)
	if es := auditEntries(t, data); len(es) != 1 || es[0]["action"] != "ops.order_assign" {
		t.Fatalf("التصديرُ حمل %v", es)
	}
	var details []byte
	if err := f.pool.QueryRow(ctx, `
		SELECT details FROM audit_log
		WHERE actor_user_id = $1 AND action = 'admin.audit_exported'
		ORDER BY id DESC LIMIT 1`, exporter).Scan(&details); err != nil {
		t.Fatalf("التصديرُ لم يُكتب في السجلّ: %v", err)
	}
	var meta map[string]any
	_ = json.Unmarshal(details, &meta)
	if meta["rows"] != float64(1) || meta["actor"] != subject || meta["group"] != "all" {
		t.Fatalf("سطرُ التصدير بلا مرشّحاته: %v", meta)
	}
}

// ── قرارا المالك الثانيان (2026-10-04) ─────────────────────────────────
//
//	الماليّةُ تقرأ السجلَّ (`audit.read` — هجرة 0260) وترى مبالغَه
//	والتصديرُ لمدير المنصّة ومالكها وحدَهما (`audit.export`)

// TestAuditLog_FinanceReadsAndSeesMoney **الماليّةُ تملك `audit.read` وترى المبالغ.**
func TestAuditLog_FinanceReadsAndSeesMoney(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	for _, c := range []string{"audit.read", "finance.read"} {
		var ok bool
		if err := f.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM role_capabilities
			WHERE role_code = 'finance' AND capability_code = $1)`, c).Scan(&ok); err != nil || !ok {
			t.Fatalf("الماليّةُ لا تملك %s (err=%v)", c, err)
		}
	}
	if need, ok := authz.LookupAdmin("GET", "/audit"); !ok || need != authz.AuditRead {
		t.Fatalf("بابُ السجلّ بقدرة %q لا audit.read", need)
	}
	fin := testdb.NewUser(t, f.pool, "finance")
	auditInsert(t, f, fin, "finance.wallet_apply", "user", fin, `{"amount": 7000}`, time.Now())
	data := auditGet(t, f, f.srv.handleAdminAudit, fin, []string{"finance"},
		[]string{"audit.read", "finance.read"}, "group=all&actor="+fin)
	if data["money_visible"] != true {
		t.Fatalf("الماليّةُ لا ترى المبالغ: money_visible=%v", data["money_visible"])
	}
	es := auditEntries(t, data)
	if len(es) != 1 {
		t.Fatalf("سطورُ الماليّة: %v", es)
	}
	if d, _ := es[0]["details"].(map[string]any); d["amount"] != float64(7000) {
		t.Fatalf("المبلغُ حُذف عن الماليّة: %v", es[0])
	}
}

// TestAuditLog_ExportOnlyForPlatformAdmin **من يقرأ لا يُصدّر.**
func TestAuditLog_ExportOnlyForPlatformAdmin(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	if need, ok := authz.LookupAdmin("GET", "/audit/export"); !ok || need != authz.AuditExport {
		t.Fatalf("بابُ التصدير بقدرة %q لا audit.export", need)
	}
	// **في القاعدة: الأدمنُ والمالكُ الأعلى وحدَهما.**
	rows, err := f.pool.Query(ctx, `SELECT role_code FROM role_capabilities
		WHERE capability_code = 'audit.export' ORDER BY role_code`)
	if err != nil {
		t.Fatal(err)
	}
	var roles []string
	for rows.Next() {
		var r string
		_ = rows.Scan(&r)
		roles = append(roles, r)
	}
	rows.Close()
	if strings.Join(roles, ",") != "admin,owner_super_admin" {
		t.Fatalf("audit.export لأدوار %v — والقرارُ الأدمنُ والمالكُ وحدَهما", roles)
	}

	reader := testdb.NewUser(t, f.pool, "finance")
	call := func(h http.HandlerFunc, roles, caps []string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/x?group=all", nil)
		c := context.WithValue(req.Context(), ctxUserID, reader)
		c = context.WithValue(c, ctxRoles, roles)
		c = context.WithValue(c, ctxCaps, caps)
		w := httptest.NewRecorder()
		h(w, req.WithContext(c))
		return w
	}
	for _, role := range []string{"finance", "trust_safety", "platform_monitor"} {
		w := call(f.srv.handleAdminAuditExport, []string{role}, []string{"audit.read", "finance.read"})
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s صدّر السجلَّ بـaudit.read وحدَها: %d", role, w.Code)
		}
	}
	// **وزرُّ التصدير يُخفى لمن لا يملكه** — والعرضُ له.
	data := auditGet(t, f, f.srv.handleAdminAudit, reader, []string{"finance"},
		[]string{"audit.read", "finance.read"}, "group=all")
	if data["can_export"] != false {
		t.Fatalf("can_export = %v لمن يقرأ فقط", data["can_export"])
	}
	data = auditGet(t, f, f.srv.handleAdminAudit, reader, []string{"admin"},
		[]string{"audit.read", "audit.export"}, "group=all")
	if data["can_export"] != true {
		t.Fatalf("can_export = %v للأدمن", data["can_export"])
	}
	if w := call(f.srv.handleAdminAuditExport, []string{"admin"}, []string{"audit.read", "audit.export"}); w.Code != http.StatusOK {
		t.Fatalf("الأدمنُ لم يُصدّر: %d — %s", w.Code, w.Body.String())
	}
}
