package qa

// ══════════════════════════════════════════════════════════════════════
// **مركزُ الإشعارات — يُعايَن ثمّ يُرسَل مرّةً** (`NT`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُقاس هنا «هل كُتب الصفّ؟» وحدَه** — **بل: أوصل الخبرُ إلى
// صندوق إنسانٍ أم لا؟** **وحارسٌ يقرأ صفَّ حملةٍ ولا يقرأ صندوقَ
// متلقٍّ زينة.**

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **أدواتُ الفحص**
// ══════════════════════════════════════════════════════════════════════

// adminTok **أدمنٌ بقدرة المحتوى** — **وهي قدرةُ من يخاطب الناس.**
func adminTok(t *testing.T, h *Harness) string {
	t.Helper()
	return h.NewUser("admin").Token
}

func mkCampaign(t *testing.T, h *Harness, tok string, body map[string]any) Res {
	t.Helper()
	return h.POST("/api/v1/admin/campaigns", tok, body)
}

// inboxCount **كم خبراً في صندوق هذا الإنسان من هذا النوع.**
func inboxCount(t *testing.T, h *Harness, userID, kind string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND kind = $2`,
		userID, kind).Scan(&n); err != nil {
		t.Fatalf("قراءةُ الصندوق: %v", err)
	}
	return n
}

// interestFor **اشتراكُ «أخبرني» لهذا الإنسان في هذا الهدف.**
func interestFor(t *testing.T, h *Harness, userID, targetKey string, active bool) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO coverage_requests (user_id, at, address_text, source, kind,
		                               target_key, active)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography,
		        'QA', 'app', 'service_interest', $2, $3)`,
		userID, targetKey, active); err != nil {
		t.Fatalf("اشتراكُ الخبر: %v", err)
	}
}

// newCity **مدينةٌ للفحص** — **ومركزُها يُقاس عليه التوفّر.**
func newCity(t *testing.T, h *Harness, name string, lat, lng float64) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO cities (name, center, radius_m, max_delivery_m, active, sort_order)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, 5000, 5000, true, 900)
		RETURNING id::text`, uniq(name+" "), lat, lng).Scan(&id); err != nil {
		t.Fatalf("إنشاءُ مدينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM cities WHERE id = $1::uuid`, id)
	})
	return id
}

// noQuiet **يُصرّح الفحصُ بسياسته** — **ولا يرث ما تركه غيرُه.**
//
// **والمتساويان يعنيان «لا هدوء»** (`campaigns.Quiet`).
func noQuiet(t *testing.T, h *Harness) {
	t.Helper()
	h.Setting("notify.quiet_from", "0")
	h.Setting("notify.quiet_to", "0")
	// **والسقفُ يُصرَّح به كذلك** — **وفحصُ السقف يتركه واحداً**،
	// **فيُحجب ما بعده.** **ولا فحصَ يرث سياسةَ غيره.**
	h.Setting("notify.engagement_daily_cap", "5")
}

func campaignField(t *testing.T, r Res, key string) any {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("**الحملةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
	return r.JSON()[key]
}

// ═════════════════ NT-02 ═════════════════

// TestNT02_CapabilityIsRequired **ولا يخاطب الناسَ من لا يملك ذلك.**
func TestNT02_CapabilityIsRequired(t *testing.T) {
	hh := New(t)
	body := map[string]any{
		"title": "NT-02", "audience_type": "role", "audience_ref": "customer",
	}
	// **وزبونٌ موثَّق.**
	cust := hh.Customer()
	if r := mkCampaign(t, hh, cust.Token, body); r.Code == http.StatusOK {
		t.Fatalf("**أنشأ زبونٌ حملة**")
	}
	// **وموظّفٌ بدورٍ آخرَ لا يملك قدرةَ المحتوى** — **والقراءةُ
	// المحضةُ ليست خطاباً باسم المنصّة.**
	ana := hh.NewUser("analytics")
	if r := mkCampaign(t, hh, ana.Token, body); r.Code != http.StatusForbidden {
		t.Fatalf("**دورُ القراءة فتح بابَ الخطاب**: %d", r.Code)
	}
	// **وبلا هويّةٍ أصلاً.**
	if r := mkCampaign(t, hh, "", body); r.Code == http.StatusOK {
		t.Fatalf("**أُنشئت حملةٌ بلا هويّة**")
	}
}

// ═════════════════ NT-03 · NT-04 ═════════════════

// TestNT03_NT04_DraftAndPreviewSendNothing **والمعاينةُ لا تُرسل.**
//
// **ومن عاين فوجد رسالتَه قد وصلت لا يملك أن يسحبها.**
func TestNT03_NT04_DraftAndPreviewSendNothing(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	// **ولا يرث الفحصُ ساعةَ الحائط** — **وأخطرُ ما في هذا الفحص أنّه
	// ينفي**: **«لم يُرسَل شيء»** — **وساعةُ الهدوء تجعل النفيَ صادقاً
	// بلا سبب**، **فيمرّ ليلاً وهو لا يقيس شيئاً.**
	noQuiet(t, hh)

	r := mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-03", "body": "نصٌّ", "audience_type": "role",
		"audience_ref": "customer",
	})
	if got := campaignField(t, r, "status"); got != "draft" {
		t.Fatalf("**حملةٌ بلا موعدٍ ليست مسوّدة**: %v", got)
	}

	// **والمعاينةُ تعدّ ولا تكتب.**
	p := hh.GET("/api/v1/admin/campaigns/preview?audience_type=role&audience_ref=customer", tok)
	if p.Code != http.StatusOK {
		t.Fatalf("**المعاينةُ رُدّت**: %d", p.Code)
	}
	if n, _ := p.JSON()["count"].(float64); n < 1 {
		t.Fatalf("**المعاينةُ لا تعدّ**: %v", p.JSON())
	}
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 0 {
		t.Fatalf("**المسوّدةُ أو المعاينةُ أرسلت**: %d", got)
	}
}

// ═════════════════ NT-05 · NT-11 · NT-12 ═════════════════

// TestNT05_NT11_NT12_ValidationAtCreation **وما لا يُفهَم يُردّ قبل أن يُحفَظ.**
func TestNT05_NT11_NT12_ValidationAtCreation(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	base := func(over map[string]any) map[string]any {
		b := map[string]any{
			"title": "NT-11", "audience_type": "role", "audience_ref": "customer",
		}
		for k, v := range over {
			b[k] = v
		}
		return b
	}
	// **NT-05 · جمهورٌ لا يُعرَف.**
	if r := mkCampaign(t, hh, tok, base(map[string]any{"audience_ref": "admin"})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل جمهورٌ لا يُخاطَب**: %d", r.Code)
	}
	if r := mkCampaign(t, hh, tok, base(map[string]any{"audience_type": "sql", "audience_ref": "SELECT 1"})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل شرطٌ يُكتب**: %d", r.Code)
	}
	// **NT-11 · موعدٌ مضى.**
	past := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	if r := mkCampaign(t, hh, tok, base(map[string]any{"scheduled_at": past})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل موعدٌ مضى**: %d", r.Code)
	}
	// **NT-12 · وجهةٌ حرّة.**
	if r := mkCampaign(t, hh, tok, base(map[string]any{
		"dest_type": "https://evil.example/pay",
	})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل رابطٌ حرّ**: %d", r.Code)
	}
	if r := mkCampaign(t, hh, tok, base(map[string]any{"dest_type": "offer"})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبلت وجهةُ عرضٍ بلا معرّف**: %d", r.Code)
	}
	// **وعنوانٌ فارغ.**
	if r := mkCampaign(t, hh, tok, base(map[string]any{"title": "   "})); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل عنوانٌ فارغ**: %d", r.Code)
	}
}

// ═════════════════ MD-03 · MD-04 · MD-06 · MD-07 ═════════════════

// TestMD_MerchantDestinationIsRefusedAtTheDoor **و«المتجر» لم تعد وجهةً.**
//
// # ولماذا عند الباب لا في اللوحة
//
// **ولوحةٌ تُنقَّى وحدَها يتخطّاها نداءٌ مصنوعٌ بيد** (`MD-03`) —
// **ونسخةٌ قديمةٌ من اللوحة تبقى ترسلها** (`MD-07`): **والمحرّكُ هو
// الحَكَم.**
//
// **ولا شاشةَ متجرٍ عند الزبون** (قرارُ المالك ٢٠٢٦-٠٨-٠٥) — **ووجهةٌ
// لا تُفتَح وعدٌ يُرسَل في جيبه ولا يُوفى.**
func TestMD_MerchantDestinationIsRefusedAtTheDoor(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	body := func(over map[string]any) map[string]any {
		b := map[string]any{
			"title": "MD", "audience_type": "role", "audience_ref": "customer",
		}
		for k, v := range over {
			b[k] = v
		}
		return b
	}

	// **MD-03 · نداءٌ مصنوعٌ بيدٍ يقصد المتجر.**
	id := "8b0a8a71-d186-4647-ae3b-9cd3898508bf"
	for _, dest := range []string{"merchant", "MERCHANT", "merchants", " merchant "} {
		r := mkCampaign(t, hh, tok, body(map[string]any{"dest_type": dest, "dest_id": id}))
		if r.Code != http.StatusBadRequest {
			t.Fatalf("**قُبلت وجهةُ متجرٍ**: %q ⇒ %d", dest, r.Code)
		}
	}

	// **MD-07 · ونسخةٌ قديمةٌ من اللوحة ترسلها كما كانت ترسلها** —
	// **بمعرّفِ متجرٍ قائمٍ في القاعدة**: **والوجودُ لا يجعلها وجهة.**
	var merchantID string
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT id::text FROM merchants LIMIT 1`).Scan(&merchantID)
	if merchantID != "" {
		r := mkCampaign(t, hh, tok, body(map[string]any{
			"dest_type": "merchant", "dest_id": merchantID,
		}))
		if r.Code != http.StatusBadRequest {
			t.Fatalf("**قُبلت وجهةُ متجرٍ قائمٍ فعلاً**: %d", r.Code)
		}
	}

	// **ولا صفَّ كُتب** — **والردُّ وحدَه لا يكفي**: **حملةٌ محفوظةٌ
	// بوجهةٍ لا تُفتَح تنتظر من يرسلها.**
	var rows int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM campaigns WHERE dest_type = 'merchant'`).Scan(&rows); err != nil {
		t.Fatalf("عدُّ الحملات: %v", err)
	}
	if rows != 0 {
		t.Fatalf("**حُفظت حملةٌ بوجهة متجر**: %d", rows)
	}

	// **MD-04 · والعرضُ يبقى مقبولاً** — **وحارسٌ يمنع كلَّ شيءٍ ليس
	// حارساً.**
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(4 * time.Hour)
	live := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))
	if r := mkCampaign(t, hh, tok, body(map[string]any{
		"dest_type": "offer", "dest_id": live,
	})); r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("**رُدّت وجهةُ عرضٍ صحيحة**: %d / %s", r.Code, r.Err())
	}
	// **والبيتُ كذلك.**
	if r := mkCampaign(t, hh, tok, body(nil)); r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("**رُدّت وجهةُ البيت**: %d / %s", r.Code, r.Err())
	}

	// **MD-06 · وما لا نعرفه يُردّ كما كان.**
	for _, dest := range []string{"store", "shop", "order_chat", "intent://x", "../../etc"} {
		r := mkCampaign(t, hh, tok, body(map[string]any{"dest_type": dest, "dest_id": id}))
		if r.Code != http.StatusBadRequest {
			t.Fatalf("**قُبلت وجهةٌ لا تُعرَف**: %q ⇒ %d", dest, r.Code)
		}
	}
}

// ═════════════════ NT-06 · NT-20 ═════════════════

// TestNT06_NT20_SendOnceAndAudited **وضغطتان إرسالٌ واحد.**
func TestNT06_NT20_SendOnceAndAudited(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	// **ولا يرث الفحصُ ساعةَ الحائط** (٢٠٢٦-٠٩-١٦) — **قِيس**:
	// **هذه الفحوصُ تخضرّ نهاراً وتحمرّ ليلاً**، **لأنّ ساعةَ الهدوء
	// الافتراضيّةَ ٢٢→٨ تؤجّل الإرسالَ فتبقى الحملةُ `scheduled`.**
	// **وفحصٌ يتبدّل جوابُه بالساعة ليس فحصاً.**
	noQuiet(t, hh)

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-06", "audience_type": "role", "audience_ref": "customer",
	}), "id").(string)

	send := "/api/v1/admin/campaigns/" + id + "/send"
	first := hh.POST(send, tok, map[string]any{})
	if got := campaignField(t, first, "status"); got != "sent" {
		t.Fatalf("**لم تُرسَل**: %v", got)
	}
	// **والنداءُ الثاني لا يُرسل ثانيةً** — **وهو ما يقع حين تسقط
	// الشبكةُ بعد الإرسال.**
	second := hh.POST(send, tok, map[string]any{})
	if second.Code != http.StatusOK {
		t.Fatalf("**النداءُ الثاني سقط**: %d", second.Code)
	}
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 1 {
		t.Fatalf("**وصل الخبرُ %d مرّةً** — **والمنتظَر واحدة**", got)
	}

	// **NT-20 · وقيدٌ في السجلّ.**
	var n int
	for i := 0; i < 40; i++ {
		_ = hh.Pool.QueryRow(ctxBG(), `
			SELECT count(*) FROM audit_log
			 WHERE entity = 'campaign' AND entity_id = $1
			   AND action = 'ops.campaign_send'`, id).Scan(&n)
		if n > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if n != 1 {
		t.Fatalf("**قيودُ الإرسال %d** — **والمنتظَر واحد**", n)
	}
}

// ═════════════════ NT-22 ═════════════════

// TestNT22_NoPhonesInCampaignOrAudit **ولا رقمَ هاتفٍ في حملةٍ ولا سجلّ.**
//
// **وقائمةٌ محفوظةٌ تشيخ في دقيقة وتُسرَّب في تصديرٍ لا يُنتبَه له.**
func TestNT22_NoPhonesInCampaignOrAudit(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-22", "audience_type": "role", "audience_ref": "customer",
	}), "id").(string)
	hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})

	var blob string
	_ = hh.Pool.QueryRow(ctxBG(), `
		SELECT coalesce(string_agg(details::text, ' '), '')
		  FROM audit_log WHERE entity = 'campaign' AND entity_id = $1`, id).Scan(&blob)
	row := ""
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT to_jsonb(c)::text FROM campaigns c WHERE id = $1::uuid`, id).Scan(&row)

	var phone string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, cust.ID).Scan(&phone)
	for what, text := range map[string]string{"السجلّ": blob, "الحملة": row} {
		if phone != "" && contains(text, phone) {
			t.Fatalf("**رقمُ هاتفٍ في %s**", what)
		}
		if contains(text, cust.ID) {
			t.Fatalf("**معرّفُ متلقٍّ في %s** — **ولا قوائمَ متلقّين**", what)
		}
	}
}

// ═════════════════ NT-09 · NT-10 ═════════════════

// TestNT09_NT10_CancelledNeverSends **والمُلغاةُ لا تُرسَل أبداً.**
func TestNT09_NT10_CancelledNeverSends(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	// **ولا يرث الفحصُ ساعةَ الحائط** — **وأخطرُ ما في هذا الفحص أنّه
	// ينفي**: **«لم يُرسَل شيء»** — **وساعةُ الهدوء تجعل النفيَ صادقاً
	// بلا سبب**، **فيمرّ ليلاً وهو لا يقيس شيئاً.**
	noQuiet(t, hh)
	at := time.Now().Add(2 * time.Hour).Format(time.RFC3339)

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-09", "audience_type": "role", "audience_ref": "customer",
		"scheduled_at": at,
	}), "id").(string)

	c := hh.POST("/api/v1/admin/campaigns/"+id+"/cancel", tok, map[string]any{})
	if got := campaignField(t, c, "status"); got != "cancelled" {
		t.Fatalf("**لم تُلغَ**: %v", got)
	}
	// **NT-10 · ولو استُحقَّ موعدُها.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE campaigns SET scheduled_at = now() - interval '1 minute' WHERE id = $1::uuid`,
		id); err != nil {
		t.Fatalf("تحريكُ الموعد: %v", err)
	}
	if r := hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{}); r.Code == http.StatusOK {
		t.Fatalf("**أُرسلت مُلغاة**")
	}
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 0 {
		t.Fatalf("**وصل خبرُ حملةٍ مُلغاة**: %d", got)
	}
}

// ═════════════════ NT-07 · NT-08 ═════════════════

// TestNT07_NT08_DueRunsAtServerTime **والمستحقُّ يُلتقط من القاعدة.**
//
// **ولا جدولةَ في ذاكرة العمليّة** — **والالتقاطُ هو ما يقع بعد
// إعادة التشغيل حرفاً: لا شيءَ في الذاكرة، والصفُّ كما تُرك.**
func TestNT07_NT08_DueRunsAtServerTime(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	at := time.Now().Add(3 * time.Hour).Format(time.RFC3339)
	// **ولا يرث الفحصُ ساعةَ الحائط** (٢٠٢٦-٠٩-١٦) — **قِيس**:
	// **هذه الفحوصُ تخضرّ نهاراً وتحمرّ ليلاً**، **لأنّ ساعةَ الهدوء
	// الافتراضيّةَ ٢٢→٨ تؤجّل الإرسالَ فتبقى الحملةُ `scheduled`.**
	// **وفحصٌ يتبدّل جوابُه بالساعة ليس فحصاً.**
	noQuiet(t, hh)

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-07", "audience_type": "role", "audience_ref": "customer",
		"scheduled_at": at,
	}), "id").(string)

	// **ولم يحن وقتُها** — **فلا تُرسَل ولو نودي العامل.**
	hh.API.CampaignsDueOnce(ctxBG())
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 0 {
		t.Fatalf("**أُرسلت قبل موعدها**: %d", got)
	}

	// **ثمّ يحين بوقت الخادم.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE campaigns SET scheduled_at = now() - interval '1 second' WHERE id = $1::uuid`,
		id); err != nil {
		t.Fatalf("تحريكُ الموعد: %v", err)
	}
	hh.API.CampaignsDueOnce(ctxBG())
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 1 {
		t.Fatalf("**استُحقَّت ولم تُرسَل**: %d", got)
	}
	// **وجولةٌ ثانيةٌ للعامل لا تُرسل ثانيةً** — **وهي حالُ عاملين
	// أو عاملٍ أُعيد تشغيلُه.**
	hh.API.CampaignsDueOnce(ctxBG())
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 1 {
		t.Fatalf("**كرّر العاملُ الإرسال**: %d", got)
	}
}

// ═════════════════ NT-14 · NT-15 ═════════════════

// TestNT14_NT15_CapHoldsEngagementAndSparesOrders **والسقفُ للتفاعل وحدَه.**
func TestNT14_NT15_CapHoldsEngagementAndSparesOrders(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	// **وساعةُ الهدوء تُرفَع أوّلاً ثمّ يُوضَع السقفُ المقصود** —
	// **و`noQuiet` تضع سقفاً سخيّاً، فترتيبُ السطرين هو الفحص.**
	// **ولا يرث الفحصُ ساعةَ الحائط**: **٢٢→٨ تؤجّل فلا يُقاس سقفٌ
	// أصلاً** (قِيس ٢٠٢٦-٠٩-١٦).
	noQuiet(t, hh)
	hh.Setting("notify.engagement_daily_cap", "1")

	for i, title := range []string{"NT-14 أولى", "NT-14 ثانية"} {
		id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
			"title": title, "audience_type": "role", "audience_ref": "customer",
		}), "id").(string)
		r := hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})
		if r.Code != http.StatusOK {
			t.Fatalf("الإرسال %d: %d", i, r.Code)
		}
	}
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 1 {
		t.Fatalf("**تجاوز التفاعلُ سقفَه**: %d", got)
	}

	// **NT-15 · وخبرُ الطلب لا يُعدّ في السقف.**
	//
	// **ومن أُسكت خبرُ طلبه لأنّه رأى عرضين اليومَ فقد طلبَه.**
	before := inboxCount(t, hh, cust.ID, "order")
	for _, title := range []string{"طلبك في الطريق", "طلبك سُلّم", "طلبٌ ثالث"} {
		hh.API.NotifyForTest(ctxBG(), cust.ID, "order", title)
	}
	if got := inboxCount(t, hh, cust.ID, "order") - before; got != 3 {
		t.Fatalf("**حُجب خبرُ طلبٍ بسقف التسويق**: %d", got)
	}
}

// ═════════════════ NT-16 · NT-17 ═════════════════

// TestNT16_NT17_QuietDefersEngagementOnly **وساعةُ الهدوء للتفاعل وحدَه.**
func TestNT16_NT17_QuietDefersEngagementOnly(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	cust := hh.Customer()
	// **وتُجعل الساعةُ كلُّها هدوءاً** — **فيُقاس التأجيلُ بلا انتظار
	// ساعةِ حائط.**
	hh.Setting("notify.quiet_from", "0")
	hh.Setting("notify.quiet_to", "23")
	// **ويُعيد ما بدّل** — **وإعدادٌ عامٌّ يُترَك يُسكت فحصاً بعده.**
	t.Cleanup(func() { noQuiet(t, hh) })

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "NT-16", "audience_type": "role", "audience_ref": "customer",
	}), "id").(string)
	r := hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})
	if got := campaignField(t, r, "status"); got != "scheduled" {
		t.Fatalf("**لم تُؤجَّل في ساعة الهدوء**: %v", got)
	}
	if got := inboxCount(t, hh, cust.ID, "promo"); got != 0 {
		t.Fatalf("**أُرسل تفاعلٌ في ساعة الهدوء**: %d", got)
	}
	// **ولم تُسقَط** — **لها موعدٌ يُقرأ.**
	if campaignField(t, hh.GET("/api/v1/admin/campaigns", tok), "campaigns") == nil {
		t.Fatalf("**ذهبت الحملةُ بلا أثر**")
	}

	// **NT-17 · وخبرُ الطلب يمضي في ساعة الهدوء.**
	hh.API.NotifyForTest(ctxBG(), cust.ID, "order", "طلبك في الطريق")
	if got := inboxCount(t, hh, cust.ID, "order"); got < 1 {
		t.Fatalf("**أُخّر خبرُ طلبٍ بساعة هدوء**")
	}
}

// ═════════════════ SI-N-01 … SI-N-10 ═════════════════

// TestSIN_ServiceInterestTargeting **ومن طلب الخبرَ يُخبَر — وحدَه.**
func TestSIN_ServiceInterestTargeting(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	// **ولا يُورَث إعدادُ غيره** — **وفحصُ ساعة الهدوء يترك المنصّةَ
	// هادئةً كلَّها**، **فيُؤجَّل هذا ولا يُرسَل.** **والمتساويان يعنيان
	// لا هدوء.**
	noQuiet(t, hh)
	// **ومدينتان حقيقيّتان** — **وخبرُ الوصول يُعاد تقييمُه بمحرّك
	// التوفّر**: **فهدفٌ لا يُعرَف موضعُه لا يُقال عنه «وصلت».**
	z := zoneForDemand(t, hh, "منطقةُ SI-N")
	f := hh.Factory()

	want := f.NewUserWith("customer")  // **مشترِكٌ فاعلٌ حيث وصلت**
	gone := f.NewUserWith("customer")  // **ألغى اشتراكَه**
	other := f.NewUserWith("customer") // **مشترِكٌ في مدينةٍ أخرى**
	cover := f.NewUserWith("customer") // **طلب تغطيةً ولم يطلب خبراً**

	damascus := "city:" + newCity(t, hh, "مدينةُ SI-N أ", z.Lat, z.Lng)
	// **والثانيةُ بعيدةٌ لا تصلها الخدمة** — **ولا يبلغها خبرُ الأولى.**
	aleppo := "city:" + newCity(t, hh, "مدينةُ SI-N ب", 12, 77)
	interestFor(t, hh, want.ID, damascus, true)
	interestFor(t, hh, gone.ID, damascus, false)
	interestFor(t, hh, other.ID, aleppo, true)
	if _, err := hh.Pool.Exec(ctxBG(), `
		INSERT INTO coverage_requests (user_id, at, address_text, source, kind,
		                               target_key, active)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography,
		        'QA', 'app', 'coverage_request', $2, true)`,
		cover.ID, damascus); err != nil {
		t.Fatalf("طلبُ التغطية: %v", err)
	}

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "وصلت الخدمةُ إلى منطقتك", "audience_type": "service_interest",
		"audience_ref": damascus,
	}), "id").(string)
	r := hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})
	if r.Code != http.StatusOK {
		t.Fatalf("الإرسال: %d / %s", r.Code, r.Err())
	}

	// **SI-N-01 · الفاعلُ يُخبَر.**
	if got := inboxCount(t, hh, want.ID, "promo"); got != 1 {
		t.Fatalf("**لم يُخبَر المشترِك**: %d", got)
	}
	// **SI-N-02 · والمُلغى لا يُخبَر.**
	if got := inboxCount(t, hh, gone.ID, "promo"); got != 0 {
		t.Fatalf("**أُخبر من ألغى اشتراكَه**: %d", got)
	}
	// **SI-N-03 · وطالبُ التغطية ليس مشترِكاً في الخبر.**
	if got := inboxCount(t, hh, cover.ID, "promo"); got != 0 {
		t.Fatalf("**أُخبر من طلب تغطيةً ولم يطلب خبراً**: %d", got)
	}
	// **SI-N-10 · ودمشقُ غيرُ حلب.**
	if got := inboxCount(t, hh, other.ID, "promo"); got != 0 {
		t.Fatalf("**بلغ خبرُ دمشقَ مشترِكَ حلب**: %d", got)
	}
	// **SI-N-09 · وإعادةُ المحاولة لا تُكرّر.**
	hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})
	if got := inboxCount(t, hh, want.ID, "promo"); got != 1 {
		t.Fatalf("**كرّرت الإعادةُ خبرَ الوصول**: %d", got)
	}
	// **والعدُّ يقول الحقيقة** — **واحدٌ استُهدف وواحدٌ كُتب.**
	final := hh.GET("/api/v1/admin/campaigns", tok)
	rows, _ := final.JSON()["campaigns"].([]any)
	if len(rows) == 0 {
		t.Fatalf("**لا حملةَ في التاريخ**")
	}
	first, _ := rows[0].(map[string]any)
	if first["targeted"] != float64(1) || first["inbox_created"] != float64(1) {
		t.Fatalf("**العدُّ لا يطابق ما وقع**: %v", first)
	}
}

// ═════════════════ SI-N-04 · SI-N-05 ═════════════════

// TestSIN04_SIN05_CellAndCityScopes **والمربّعُ كالمدينة في الدقّة.**
func TestSIN04_SIN05_CellAndCityScopes(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	noQuiet(t, hh)
	f := hh.Factory()
	inCell := f.NewUserWith("customer")
	inCity := f.NewUserWith("customer")
	interestFor(t, hh, inCell.ID, "cell:35.95,39.01", true)
	interestFor(t, hh, inCity.ID, "city:33333333-3333-3333-3333-333333333333", true)

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "SI-N-05", "audience_type": "service_interest",
		"audience_ref": "cell:35.95,39.01",
	}), "id").(string)
	hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})

	if got := inboxCount(t, hh, inCell.ID, "promo"); got != 1 {
		t.Fatalf("**لم يُخبَر صاحبُ المربّع**: %d", got)
	}
	if got := inboxCount(t, hh, inCity.ID, "promo"); got != 0 {
		t.Fatalf("**بلغ خبرُ مربّعٍ مشترِكَ مدينة**: %d", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ولا فحصَ يتبدّل جوابُه بساعة الحائط** (٢٠٢٦-٠٩-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع
//
// **جرى الفحصُ الكاملُ الواحدةَ بعد منتصف الليل فسقطت أربعةٌ** —
// `NT-06` و`NT-07` و`NT-14` و`SI-N-06`: **كلُّها تقول «لم تُرسَل»
// والحملةُ `scheduled`.**
//
// **ولا عطبَ في المنتج**: **ساعةُ الهدوء الافتراضيّةُ ٢٢→٨** (الدفعة
// الثامنة) **تؤجّل التفاعلَ إلى الصباح** — **وهي تعمل كما أُقرّت.**
//
// **والعطبُ في الفحص**: **ورث سياسةَ المنصّة ولم يعلن سياستَه** —
// **فيخضرّ نهاراً ويحمرّ ليلاً.** **وفحصٌ كذلك لا يُوثَق به في
// الاتّجاهين**: **لا حين يخضرّ ولا حين يحمرّ.**
//
// # ولماذا حارسٌ لا إصلاحُ الأربعة
//
// **والخامسُ يُكتب غداً** — **ومن أصلح ما وقع وحدَه انتظر وقوعَه
// ثانيةً.**
func TestNTQ_SendingTestsDeclareTheirQuietPolicy(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "backend", "internal", "qa")
	files, err := filepath.Glob(filepath.Join(dir, "campaigns*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("**لم تُقرأ ملفّاتُ فحوص الحملات**: %v", err)
	}

	// **وما يدلّ على أنّ الفحصَ ينتظر إرسالاً فعليّاً.**
	wants := []string{`!= "sent"`, `"promo") ; got != 0`, `inboxCount(`}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("قراءة %s: %v", f, err)
		}
		src := string(b)
		for _, block := range strings.Split(src, "\nfunc Test")[1:] {
			name := block[:strings.Index(block, "(")]
			// **والفحصُ الذي يقيس التأجيلَ نفسَه يعلن هدوءَه بيده.**
			if strings.Contains(block, "notify.quiet_from") {
				continue
			}
			sends := false
			for _, w := range wants {
				if strings.Contains(block, w) {
					sends = true
					break
				}
			}
			if !sends {
				continue
			}
			if !strings.Contains(block, "noQuiet(t, hh)") {
				t.Errorf("**%s ينتظر إرسالاً ولا يعلن سياسةَ الهدوء** — "+
					"**فيخضرّ نهاراً ويحمرّ ليلاً.**", name)
			}
		}
	}
}
