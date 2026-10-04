package server

// ══════════════════════════════════════════════════════════════════════
// **صفحةُ «طلبات الانضمام» — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
// ══════════════════════════════════════════════════════════════════════
//
//  1. **الموافقةُ ترسل رسالةَ الدخول في المسار نفسِه** — والكلمةُ المؤقّتةُ
//     ومهلتُها في معاملة التحويل؛ ومن له حسابٌ قائمٌ لا تُولَّد له كلمة.
//  2. **الصفحةُ لمتاجر المندوبين وحدَها.**
//  3. **بحثٌ وفلاتر وتحذيراتٌ وحالةُ «بحاجة معلومات»** تعود للمندوب بملاحظة.
//  4. **الهدفُ يُحسب لحظةَ إنشاء المتجر، والعمولةُ لا** — تبدأ من أوّل
//     طلبٍ مُسلَّم (`sales.activation_orders` افتراضُه ١).
//  5. **لا وثائقَ من المتجر** — يُحوَّل الطلبُ باسمٍ ورقمٍ وتصنيفٍ ونقطة.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// leadFx عُدّةُ هذه الفحوص — خادمٌ بفهرسه وهويّته وإشعاراته ومندوبٌ برمز.
type leadFx struct {
	*driverFixture
	ops, rep, category string
}

func newLeadFx(t *testing.T) *leadFx {
	t.Helper()
	f := newDriverFixture(t, 0)
	ops := f.armOps(t)
	rep := testdb.NewUser(t, f.pool, "sales")
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx,
		`UPDATE users SET invite_code = 'LD' || upper(substr(md5(id::text), 1, 8)) WHERE id = $1`,
		rep); err != nil {
		t.Fatalf("رمزُ المندوب: %v", err)
	}
	var cat string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	return &leadFx{driverFixture: f, ops: ops, rep: rep, category: cat}
}

// freshPhone رقمٌ لا حسابَ له ولا طلب.
func (f *leadFx) freshPhone(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `CREATE SEQUENCE IF NOT EXISTS test_phone_seq START 1`); err != nil {
		t.Fatalf("التسلسل: %v", err)
	}
	for i := 0; i < 50; i++ {
		var ph string
		var taken bool
		if err := f.pool.QueryRow(ctx, `
			WITH p AS (SELECT '+9639' || lpad((nextval('test_phone_seq') % 100000000)::text, 8, '0') AS ph)
			SELECT ph, EXISTS (SELECT 1 FROM users WHERE phone = p.ph)
			        OR EXISTS (SELECT 1 FROM merchant_leads WHERE phone = p.ph)
			        OR EXISTS (SELECT 1 FROM merchants WHERE phone = p.ph)
			  FROM p`).Scan(&ph, &taken); err != nil {
			t.Fatalf("رقمٌ جديد: %v", err)
		}
		if !taken {
			return ph
		}
	}
	t.Fatal("لا رقمَ حرّ")
	return ""
}

// newLead طلبٌ مفتوحٌ لمندوب العُدّة (أو بلا مندوبٍ إن كان `rep` فارغاً).
func (f *leadFx) newLead(t *testing.T, name, phone, rep string) string {
	t.Helper()
	var repArg any
	if rep != "" {
		repArg = rep
	}
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO merchant_leads (store_name, owner_name, phone, area, category_id, lat, lng, sales_rep_user_id)
		VALUES ($1, 'صاحبُه', $2, 'مقابل الجامع', $3, 35.9528, 39.0079, $4)
		RETURNING id::text`, name, phone, f.category, repArg).Scan(&id); err != nil {
		t.Fatalf("الطلب: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `UPDATE merchant_leads SET merchant_id = NULL WHERE id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM merchants WHERE lead_id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM merchant_leads WHERE id = $1`, id)
	})
	return id
}

func (f *leadFx) setStatus(t *testing.T, id, body string) (int, map[string]any) {
	t.Helper()
	w := callWithID(f.driverFixture, f.srv.handleAdminLeadStatus, id, f.ops, []string{"ops"}, body)
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Data == nil {
		env.Data = map[string]any{"_raw": w.Body.String()}
	}
	return w.Code, env.Data
}

type leadListRow struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	DecisionNote    string   `json:"decision_note"`
	DuplicatePhone  []string `json:"duplicate_phone"`
	NearbySameName  []string `json:"nearby_same_name"`
	ExistingAccount bool     `json:"existing_account"`
}

func (f *leadFx) list(t *testing.T, query string) []leadListRow {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/leads?per_page=100&"+query, nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, f.ops))
	w := httptest.NewRecorder()
	f.srv.handleAdminLeads(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("القائمة %q ردّت %d: %s", query, w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Leads []leadListRow `json:"leads"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("فكُّ القائمة: %v", err)
	}
	return res.Data.Leads
}

func has(rows []leadListRow, id string) *leadListRow {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

// ── ١ · رسالةُ الدخول في مسار الموافقة نفسِه ─────────────────────────

func TestLeadConvert_NewOwnerGetsTempPasswordInSameFlow(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()
	phone := f.freshPhone(t)
	id := f.newLead(t, "متجرُ الترحيب الجديد", phone, f.rep)

	code, out := f.setStatus(t, id, `{"status":"converted"}`)
	if code != http.StatusOK {
		t.Fatalf("الموافقة ردّت %d: %v", code, out)
	}
	welcome, ok := out["welcome"].(map[string]any)
	if !ok {
		t.Fatalf("**جوابُ الموافقة لا يقول شيئاً عن رسالة الدخول**: %v", out)
	}
	if welcome["existing_owner"] != false {
		t.Errorf("صاحبٌ جديدٌ قيل إنّه قائم: %v", welcome)
	}
	if _, ok := welcome["expires_at"]; !ok {
		t.Errorf("**لا مهلةَ للكلمة المؤقّتة في الجواب**: %v", welcome)
	}
	// **والكلمةُ مؤقّتةٌ بمهلةٍ في القاعدة** — كُتبت في معاملة التحويل.
	var must, hasExp, hasPw bool
	if err := f.pool.QueryRow(ctx, `
		SELECT must_change_password, temp_password_expires_at IS NOT NULL, password_hash IS NOT NULL
		  FROM users WHERE phone = $1`, phone).Scan(&must, &hasExp, &hasPw); err != nil {
		t.Fatalf("حسابُ صاحب المتجر: %v", err)
	}
	if !must || !hasExp || !hasPw {
		t.Errorf("الكلمةُ المؤقّتة: يُجبَر=%v مهلة=%v كلمة=%v — والمطلوب الثلاثة", must, hasExp, hasPw)
	}
	// **والرسالةُ حاولت الخروج في النداء نفسِه** — وأثرُها مكتوب.
	uid, _ := welcome["user_id"].(string)
	waitAudit(t, f.driverFixture, "user", uid, "admin.welcome_message")
}

func TestLeadConvert_ExistingAccountGetsNoNewPassword(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()
	customer := testdb.NewUser(t, f.pool, "customer")
	if _, err := f.pool.Exec(ctx,
		`UPDATE users SET password_hash = 'hash-before', must_change_password = false WHERE id = $1`,
		customer); err != nil {
		t.Fatalf("كلمةُ الزبون: %v", err)
	}
	var phone string
	_ = f.pool.QueryRow(ctx, `SELECT phone::text FROM users WHERE id = $1`, customer).Scan(&phone)
	id := f.newLead(t, "متجرُ الزبون القائم", phone, f.rep)

	code, out := f.setStatus(t, id, `{"status":"converted"}`)
	if code != http.StatusOK {
		t.Fatalf("الموافقة ردّت %d: %v", code, out)
	}
	welcome, ok := out["welcome"].(map[string]any)
	if !ok || welcome["existing_owner"] != true {
		t.Fatalf("**الجوابُ لا يقول إنّ الحسابَ قائم**: %v", out)
	}
	var hash string
	var must bool
	_ = f.pool.QueryRow(ctx, `SELECT password_hash, must_change_password FROM users WHERE id = $1`,
		customer).Scan(&hash, &must)
	if hash != "hash-before" || must {
		t.Errorf("**مُسّت كلمةُ حسابٍ قائم**: %q يُجبَر=%v", hash, must)
	}
	// **ورسالتُه «صار عندك متجر» لا كلمةٌ جديدة.**
	waitAudit(t, f.driverFixture, "user", customer, "admin.store_owner_notified")
}

// ── ٢ و٣ · متاجرُ المندوبين وحدَها · بحثٌ وفلاتر وتحذيرات ──────────────

func TestAdminLeads_RepOnlyFiltersAndWarnings(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()

	orphan := f.newLead(t, "طلبٌ بلا مندوب", f.freshPhone(t), "")
	dupPhone := f.freshPhone(t)
	a := f.newLead(t, "فرن الأمانة التجريبي", dupPhone, f.rep)
	b := f.newLead(t, "بقالةُ الرقمِ المكرّر", dupPhone, f.rep)

	// **متجرٌ قائمٌ بالاسم نفسِه على بعد أمتار** من الطلب الأوّل.
	var near string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, location)
		VALUES ('فرن الأمانة التجريبي', $1, ST_SetSRID(ST_MakePoint(39.0081, 35.9530), 4326)::geography)
		RETURNING id::text`, f.category).Scan(&near); err != nil {
		t.Fatalf("المتجرُ القريب: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, near) })

	all := f.list(t, "")
	if has(all, orphan) != nil {
		t.Error("**طلبٌ بلا مندوبٍ ظهر في الصفحة** — وهي لمتاجر المندوبين وحدَها")
	}
	ra := has(all, a)
	if ra == nil || has(all, b) == nil {
		t.Fatal("طلبا المندوب غائبان عن القائمة")
	}
	if len(ra.DuplicatePhone) == 0 {
		t.Error("**لا تحذيرَ للرقم المكرّر**")
	}
	if len(ra.NearbySameName) == 0 {
		t.Error("**لا تحذيرَ للمتجر القريب بالاسم نفسِه**")
	}
	if ra.ExistingAccount {
		t.Error("رقمٌ بلا حسابٍ قيل إنّ له حساباً")
	}

	// **والبحثُ والمرشِّحات في الخادم.**
	if rows := f.list(t, "q=الأمانة"); has(rows, a) == nil || has(rows, b) != nil {
		t.Errorf("البحثُ بالاسم: %d صفّاً", len(rows))
	}
	if rows := f.list(t, "rep_id="+f.rep); has(rows, a) == nil {
		t.Error("مرشِّحُ المندوب أسقط طلبَه")
	}
	if rows := f.list(t, "rep_id="+f.ops); has(rows, a) != nil {
		t.Error("مرشِّحُ مندوبٍ آخر أظهر طلباً ليس له")
	}
	if rows := f.list(t, "category_id="+f.category); has(rows, a) == nil {
		t.Error("مرشِّحُ التصنيف أسقط طلباً منه")
	}
	if rows := f.list(t, "from=2000-01-01&to=2000-01-02"); has(rows, a) != nil {
		t.Error("مرشِّحُ التاريخ أظهر طلباً من خارجه")
	}
	var gov string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM governorates LIMIT 1`).Scan(&gov); err == nil {
		if rows := f.list(t, "governorate_id="+gov); has(rows, a) != nil {
			t.Error("مرشِّحُ المحافظة أظهر طلباً بلا منطقة")
		}
	}
}

// ── ٣ · «بحاجة معلومات» تعود للمندوب بملاحظة ──────────────────────────

func TestLeadNeedsInfo_ReturnsToRepWithNote(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()
	id := f.newLead(t, "متجرٌ ينقصه شيء", f.freshPhone(t), f.rep)

	if code, _ := f.setStatus(t, id, `{"status":"needs_info"}`); code < 400 {
		t.Errorf("**«بحاجة معلومات» بلا ملاحظة قُبلت** (%d)", code)
	}
	if code, out := f.setStatus(t, id, `{"status":"needs_info","note":"الموقع على الخريطة غير دقيق"}`); code != http.StatusOK {
		t.Fatalf("«بحاجة معلومات» ردّت %d: %v", code, out)
	}
	waitAudit(t, f.driverFixture, "lead", id, "ops.lead_needs_info")

	// **وتصل المندوبَ إشعاراً بملاحظتها.**
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications
		WHERE user_id = $1 AND entity_id = $2 AND body LIKE '%غير دقيق%'`, f.rep, id).Scan(&n)
	if n == 0 {
		t.Error("**لم يصل المندوبَ إشعارٌ بما ينقص**")
	}

	// **وقائمتُه تقول الحالةَ والملاحظة** — `GET /rep/leads`.
	req := httptest.NewRequest(http.MethodGet, "/rep/leads", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, f.rep))
	w := httptest.NewRecorder()
	f.srv.handleRepLeads(w, req)
	var mine struct {
		Data []leadListRow `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &mine)
	got := has(mine.Data, id)
	if got == nil || got.Status != "needs_info" || got.DecisionNote == "" {
		t.Fatalf("**قائمةُ المندوب لا تقول «بحاجة معلومات» وملاحظتَها**: %+v", got)
	}

	// **ومنها يُوافَق أو يُعاد إلى «جديد»** — لا تُقفَل.
	if code, out := f.setStatus(t, id, `{"status":"new"}`); code != http.StatusOK {
		t.Fatalf("الإعادةُ إلى جديد ردّت %d: %v", code, out)
	}
	if rows := f.list(t, "status=needs_info"); has(rows, id) != nil {
		t.Error("مرشِّحُ «بحاجة معلومات» أظهر طلباً عاد جديداً")
	}
}

// ── ٤ · الهدفُ عند الإنشاء والعمولةُ عند أوّل تسليم ──────────────────

func TestLeadConvert_TargetAtCreationCommissionAtDelivery(t *testing.T) {
	f := newLeadFx(t)
	ctx := context.Background()
	f.srv.incentives = incentives.New(f.pool, f.srv.wallet, f.srv.settings, f.srv.orders.TreasuryID)
	f.setSetting(t, "sales.monthly_target", 1)
	f.setSetting(t, "sales.target_reward", 5000)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `
			DELETE FROM app_settings WHERE key IN ('sales.monthly_target', 'sales.target_reward')`)
	})

	id := f.newLead(t, "متجرُ هدفِ الشهر", f.freshPhone(t), f.rep)
	if code, out := f.setStatus(t, id, `{"status":"converted"}`); code != http.StatusOK {
		t.Fatalf("الموافقة ردّت %d: %v", code, out)
	}
	var targets, commissions int
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM incentives WHERE user_id = $1 AND for_target`, f.rep).Scan(&targets)
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM wallet_transactions WHERE user_id = $1 AND kind = 'commission'`, f.rep).Scan(&commissions)
	if targets != 1 {
		t.Errorf("**المتجرُ لم يُحسب في الهدف لحظةَ إنشائه**: مكافآتُ الهدف = %d", targets)
	}
	if commissions != 0 {
		t.Errorf("**قُيّدت عمولةٌ قبل أيّ طلبٍ مُسلَّم**: %d", commissions)
	}
	// **والعمولةُ من أوّل طلبٍ ناجح** — عتبةُ التفعيل افتراضُها طلبٌ واحد
	// (هجرة 0098)، **وتسليمُه يحرسه `TestDelivery_CreditsCashAndCommissions`.**
	if d := settings.Default("sales.activation_orders"); d != 1 {
		t.Errorf("عتبةُ التفعيل الافتراضيّة = %d — والقرار «من أوّل طلبٍ ناجح» (١)", d)
	}
}
