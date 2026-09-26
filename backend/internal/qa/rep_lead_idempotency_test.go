package qa

// ══════════════════════════════════════════════════════════════════════
// **REP-LEAD — إنشاءُ فرصةٍ من المندوب صار محميّاً بمنعِ التكرار**
// ══════════════════════════════════════════════════════════════════════
//
// **`POST /api/v1/rep/leads` لُفّ بـ`s.idempotent` ويُثبّت عبر
// `WithIdempotentTx`** (server.go:708 · leads_handlers.go). فمفتاحٌ واحدٌ
// يُعاد به النداءُ يُنتج فرصةً واحدةً لا اثنتين — تتابعاً وتزامناً وبعد انتهاء
// المهلة — **ويسترجع الردَّ الأوّلَ بمعرّفه نفسِه.** والعقدُ على أثرِ القاعدة
// (عددُ صفوف `merchant_leads`) وعلى دوام الاسترجاع معاً.

import (
	"context"
	"testing"
)

// repLeadEnv **يجهّز مندوباً موثَّقاً وبابَ الضمّ مفتوحاً ومنطقةً وتصنيفاً.**
func repLeadEnv(t *testing.T, h *Harness) (rep *User, districtID, categoryID string) {
	t.Helper()
	// **بابُ الضمّ مفتوح** — وإلّا رُدّ النداءُ قبل قراءة الجسم (503).
	h.Setting("launch.rep_acquisition", "true")
	// **وشرطُ التوثيق مطفأٌ صراحةً** — والمندوبُ موثَّقٌ أصلاً، فالبابان مغلقان.
	h.Setting("sales.require_whatsapp", "false")

	rep = h.NewUser("sales") // موثَّقُ واتساب وله جلسة (لإثبات التأكيد عند الباب)
	districtID = repActiveDistrict(t, h)

	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("rep-lead-cat-")).Scan(&categoryID); err != nil {
		t.Fatalf("category: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})
	// **وفرصُ هذا المندوب تُمحى قبل حسابه** — LIFO: تُسجَّل بعد إنشاء المستخدم
	// فتُنفَّذ قبل حذفه، فلا يعطب حذفُ الحساب بقيدٍ أجنبيّ.
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM merchant_leads WHERE sales_rep_user_id = $1::uuid`, rep.ID)
	})
	return rep, districtID, categoryID
}

// repActiveDistrict **منطقةٌ إداريّةٌ فعّالةٌ تحت محافظةٍ فعّالة** — تُقرأ من
// المبذور، وإلّا صُنعت.
func repActiveDistrict(t *testing.T, h *Harness) string {
	t.Helper()
	var id string
	err := h.Pool.QueryRow(ctxBG(), `
		SELECT d.id::text FROM districts d
		JOIN governorates g ON g.id = d.governorate_id
		WHERE d.active AND g.active LIMIT 1`).Scan(&id)
	if err == nil && id != "" {
		return id
	}
	var gov string
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO governorates (name, active, sort_order) VALUES ($1, true, 0) RETURNING id::text`,
		uniq("rep-gov-")).Scan(&gov); err != nil {
		t.Fatalf("seed governorate: %v", err)
	}
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO districts (governorate_id, name, active, sort_order)
		 VALUES ($1::uuid, $2, true, 0) RETURNING id::text`,
		gov, uniq("rep-dist-")).Scan(&id); err != nil {
		t.Fatalf("seed district: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM districts WHERE id = $1::uuid`, id)
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM governorates WHERE id = $1::uuid`, gov)
	})
	return id
}

// repLeadBody **أدنى جسمٍ صحيحٍ لإنشاء فرصةٍ من المندوب.**
func repLeadBody(districtID, categoryID, phone string) map[string]any {
	return map[string]any{
		"store_name":  "QA Rep Lead Store",
		"owner_name":  "QA Owner",
		"phone":       phone,
		"area":        "near the mosque",
		"district_id": districtID,
		"category_id": categoryID,
		"password":    "qa-passw0rd",
		"lat":         35.9506,
		"lng":         39.0094,
	}
}

// repLeadCount **كم فرصةً لهذا المندوب** — الحقيقةُ في القاعدة لا في الردّ.
func repLeadCount(t *testing.T, h *Harness, repID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM merchant_leads WHERE sales_rep_user_id = $1::uuid`,
		repID).Scan(&n); err != nil {
		t.Fatalf("count leads: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **أ · مفتاحٌ واحدٌ مرّتين بالتتابع ⇒ فرصةٌ واحدةٌ واسترجاعٌ دائم**
// ══════════════════════════════════════════════════════════════════════
//
// **`handleRepCreateLead` صار يُثبّت نتيجتَه عبر `WithIdempotentTx`**
// (leads_handlers.go)، **كما يفعل إنشاءُ الطلب والسحب**: الإدراجُ وكتابةُ
// `committed_at`/`response` في معاملةٍ واحدة. **فالإعادةُ بالمفتاح نفسِه
// تُعيد `201` الأصليَّ بمعرّفه نفسِه (`Idempotent-Replay`)** لا `409`، ولا
// تُنشئ صفّاً ثانياً.
//
// # وحارسان: الأثرُ ودوامُه
//
// **الأثرُ**: مفتاحٌ واحدٌ ⇒ صفٌّ واحد. **ودوامُه**: الاسترجاعُ لا يعتمد على
// مهلة الحيازة — **حتّى بعد انتهاء المهلة تُعيد المحاولةُ المُثبَّتَ الأوّلَ
// ولا تنفّذ** (لأنّ `committed_at` مكتوب، فيُقرأ `FOR UPDATE` فيُعاد).
func TestREPLEAD_Idempotency_SameKey(t *testing.T) {
	h := New(t)
	rep, district, category := repLeadEnv(t, h)
	body := repLeadBody(district, category, uniqPhone())
	key := uniq("replead")
	const endpoint = "POST /api/v1/rep/leads"

	first := h.POSTKey("/api/v1/rep/leads", rep.Token, key, body)
	if first.Code >= 400 {
		t.Fatalf("first create rejected: %s", first)
	}
	firstID, _ := first.JSON()["id"].(string)
	if firstID == "" {
		t.Fatalf("first create returned no id: %s", first)
	}

	second := h.POSTKey("/api/v1/rep/leads", rep.Token, key, body)
	secondID, _ := second.JSON()["id"].(string)
	t.Logf("first=%d(id=%s) second=%d(id=%s) replay=%v err=%q",
		first.Code, firstID, second.Code, secondID, second.Replay(), second.Err())

	// ── الحارسُ الأوّل: فرصةٌ واحدةٌ لا اثنتان ───────────────────────
	if n := repLeadCount(t, h, rep.ID); n != 1 {
		t.Errorf("SAME KEY: got %d lead rows — expected exactly 1", n)
	}

	// ── الحارسُ الثاني: استرجاعٌ دائمٌ للردّ الأوّل، لا 409 ولا صفٌّ جديد ──
	if !second.Replay() {
		t.Errorf("in-window retry was not a durable replay (missing Idempotent-Replay): %s", second)
	}
	if second.Code != first.Code {
		t.Errorf("replay status %d != original %d", second.Code, first.Code)
	}
	if secondID != firstID {
		t.Errorf("replay id %q != original %q — retry did not replay the original lead", secondID, firstID)
	}

	// ══════════════════════════════════════════════════════════════════
	// **ودوامُ المنع — مقيسٌ لا مظنون**
	// ══════════════════════════════════════════════════════════════════
	//
	// **تُقدَّم مهلةُ الحيازة في القاعدة** (كما يزيح `P-3` الزمنَ في البيانة).
	// **والاسترجاعُ مبنيٌّ على `committed_at` لا على المهلة**، فالمحاولةُ بعد
	// انتهائها تبقى استرجاعاً ولا تُنشئ صفّاً ثانياً.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE idempotency_keys SET lease_until = now() - interval '1 second'
		WHERE user_id = $1::uuid AND endpoint = $2 AND key = $3`,
		rep.ID, endpoint, key); err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	third := h.POSTKey("/api/v1/rep/leads", rep.Token, key, body)
	thirdID, _ := third.JSON()["id"].(string)
	after := repLeadCount(t, h, rep.ID)
	t.Logf("after lease expiry: retry=%d(id=%q) replay=%v · lead rows=%d",
		third.Code, thirdID, third.Replay(), after)
	if after != 1 {
		t.Errorf("DURABLE-DEDUP: got %d lead rows after lease expiry — expected exactly 1", after)
	}
	if thirdID != firstID {
		t.Errorf("post-expiry retry id %q != original %q — durable replay broken", thirdID, firstID)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ب · مفتاحٌ واحدٌ بالتزامن ⇒ فرصةٌ واحدةٌ بالضبط**
// ══════════════════════════════════════════════════════════════════════
//
// **والعقدُ على الأثر**: قد يختلف الردّان (إعادةٌ أو `in_progress`)، **والفرصةُ
// يجب أن تكون واحدة.**
func TestREPLEAD_Idempotency_Concurrent(t *testing.T) {
	h := New(t)
	rep, district, category := repLeadEnv(t, h)
	body := repLeadBody(district, category, uniqPhone())
	key := uniq("replead-c")

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "rep-a", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/rep/leads", rep.Token, key, body)
		}},
		Actor{Name: "rep-b", Do: func(ctx context.Context) any {
			return h.POSTKey("/api/v1/rep/leads", rep.Token, key, body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("concurrent creates hung: %s", r)
	}
	for _, o := range r.Outcomes {
		if res, ok := o.Value.(Res); ok && res.Code >= 500 {
			t.Errorf("%s answered %d — a same-key overlap must not 5xx", o.Name, res.Code)
		}
	}
	n := repLeadCount(t, h, rep.ID)
	t.Logf("CONCURRENT — codes=%v · lead rows=%d", r.Codes(), n)
	if n != 1 {
		t.Errorf("SAME KEY CONCURRENT: got %d lead rows — expected exactly 1\n%s", n, r)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ج · مفتاحان متمايزان ⇒ فرصتان (ضابط)**
// ══════════════════════════════════════════════════════════════════════
func TestREPLEAD_Idempotency_DistinctKeys(t *testing.T) {
	h := New(t)
	rep, district, category := repLeadEnv(t, h)

	first := h.POSTKey("/api/v1/rep/leads", rep.Token, uniq("replead-d1"),
		repLeadBody(district, category, uniqPhone()))
	if first.Code >= 400 {
		t.Fatalf("first create rejected: %s", first)
	}
	second := h.POSTKey("/api/v1/rep/leads", rep.Token, uniq("replead-d2"),
		repLeadBody(district, category, uniqPhone()))
	if second.Code >= 400 {
		t.Fatalf("second create rejected: %s", second)
	}
	if n := repLeadCount(t, h, rep.ID); n != 2 {
		t.Errorf("DISTINCT KEYS: got %d lead rows — expected exactly 2", n)
	}
}
