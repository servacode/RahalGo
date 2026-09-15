package qa

// ══════════════════════════════════════════════════════════════════════
// **حملاتُ العروض وخبرُ الوصول** (`EN` · `SI-N`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **وعرضٌ لا يُسعَّر به لا يُعلَن عنه** — **يفتحه الزبونُ فيجد السعرَ
// كما كان، فيظنّ أنّنا نكذب.**

import (
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// ═════════════════ EN-01 · EN-02 · EN-03 · EN-04 ═════════════════

// TestEN_OfferCampaignFollowsOfferTruth **والحملةُ تتبع حقيقةَ العرض.**
func TestEN_OfferCampaignFollowsOfferTruth(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(4 * time.Hour)

	mk := func(offerID, title string) Res {
		return mkCampaign(t, hh, tok, map[string]any{
			"title": title, "audience_type": "role", "audience_ref": "customer",
			"dest_type": "offer", "dest_id": offerID,
		})
	}

	// **EN-04 · والسارِي تُقبَل وجهتُه.**
	live := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))
	if r := mk(live, "EN-04 سارٍ"); r.Code != http.StatusOK {
		t.Fatalf("**رُدّت حملةٌ لعرضٍ سارٍ**: %d / %s", r.Code, r.Err())
	}

	// **EN-01 · والمُنزَلُ لا يُعلَن عنه.**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + live + "/stop"
	if r := hh.POST(stop, fx.Tok, map[string]any{}); r.Code != http.StatusOK {
		t.Fatalf("الإنزال: %d", r.Code)
	}
	if r := mk(live, "EN-01 مُنزَل"); r.Code != http.StatusConflict {
		t.Fatalf("**قُبلت حملةٌ لعرضٍ مُنزَل**: %d", r.Code)
	}

	// **EN-02 · والمنتهي كذلك.**
	expired := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 25, nil, &end))
	forceWindow(t, hh, expired, -3*time.Hour, -time.Hour)
	if r := mk(expired, "EN-02 منتهٍ"); r.Code != http.StatusConflict {
		t.Fatalf("**قُبلت حملةٌ لعرضٍ منتهٍ**: %d", r.Code)
	}

	// **EN-03 · والمجدولُ ليس قائماً الآن.**
	//
	// **ووعدٌ لم يحلّ بعدُ لا يُقال إنّه قائم** — **يفتحه فيجد السعرَ
	// كما كان.**
	future := time.Now().Add(48 * time.Hour)
	futureEnd := future.Add(24 * time.Hour)
	sched := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 30, &future, &futureEnd))
	if r := mk(sched, "EN-03 مجدول"); r.Code != http.StatusConflict {
		t.Fatalf("**أُعلن عن عرضٍ مجدولٍ كأنّه سارٍ**: %d", r.Code)
	}
}

// TestEN02_ScheduledCampaignRechecksOfferAtSend **ويُسأل ثانيةً عند الإرسال.**
//
// **وحملةٌ جُدولت لعرضٍ انتهى بينهما وعدٌ كاذب** — **والسؤالُ عند
// الإنشاء وحدَه لا يكفي.**
func TestEN02_ScheduledCampaignRechecksOfferAtSend(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	cust := hh.Customer()
	end := time.Now().Add(4 * time.Hour)
	live := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "EN-02 مجدولة", "audience_type": "role", "audience_ref": "customer",
		"dest_type": "offer", "dest_id": live,
		"scheduled_at": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}), "id").(string)

	// **ثمّ يُنزَل العرضُ قبل موعدها.**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + live + "/stop"
	hh.POST(stop, fx.Tok, map[string]any{})
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE campaigns SET scheduled_at = now() - interval '1 second' WHERE id = $1::uuid`,
		id); err != nil {
		t.Fatalf("تحريكُ الموعد: %v", err)
	}
	hh.API.CampaignsDueOnce(ctxBG())

	if got := inboxCount(t, hh, cust.ID, "promo"); got != 0 {
		t.Fatalf("**أُعلن عن عرضٍ أُنزل قبل الموعد**: %d", got)
	}
	var status string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM campaigns WHERE id = $1::uuid`, id).Scan(&status)
	if status != "failed" {
		t.Fatalf("**حالُ الحملة بعد سقوط العرض**: %q", status)
	}
}

// ═════════════════ SI-N-06 · SI-N-07 · SI-N-08 ═════════════════

// TestSIN06_SIN07_SIN08_ArrivalIsReEvaluated **ولا يُقال «وصلت» قبل أن تصل.**
func TestSIN06_SIN07_SIN08_ArrivalIsReEvaluated(t *testing.T) {
	hh := New(t)
	tok := adminTok(t, hh)
	f := hh.Factory()
	sub := f.NewUserWith("customer")

	// **ومربّعٌ في مكانٍ لا خدمةَ فيه** — **ولا منطقةَ تغطّيه.**
	far := opsmap.TargetKey(opsmap.KindInterest, "", 12, 77)
	interestFor(t, hh, sub.ID, far, true)

	id, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "وصلت الخدمةُ إلى منطقتك", "audience_type": "service_interest",
		"audience_ref": far,
	}), "id").(string)
	hh.POST("/api/v1/admin/campaigns/"+id+"/send", tok, map[string]any{})

	// **SI-N-06 · لم تصل — فلا يُرسَل شيء.**
	if got := inboxCount(t, hh, sub.ID, "promo"); got != 0 {
		t.Fatalf("**قيل «وصلت» ولم تصل**: %d", got)
	}
	// **SI-N-07 · ولا يُعلَّم الاشتراكُ مُخبَراً.**
	//
	// **ومن عُلّم ولم يُخبَر لا يعود إليه الخبرُ أبداً.**
	var notified *time.Time
	_ = hh.Pool.QueryRow(ctxBG(), `
		SELECT notified_at FROM coverage_requests
		 WHERE user_id = $1::uuid AND target_key = $2`, sub.ID, far).Scan(&notified)
	if notified != nil {
		t.Fatalf("**عُلّم اشتراكٌ لم يُخبَر صاحبُه**")
	}

	// ── ثمّ تصل الخدمةُ فعلاً ──
	z := zoneForDemand(t, hh, "منطقةُ SI-N-06")
	near := opsmap.TargetKey(opsmap.KindInterest, "", z.Lat, z.Lng)
	sub2 := f.NewUserWith("customer")
	interestFor(t, hh, sub2.ID, near, true)

	id2, _ := campaignField(t, mkCampaign(t, hh, tok, map[string]any{
		"title": "وصلت الخدمةُ إلى منطقتك", "audience_type": "service_interest",
		"audience_ref": near,
	}), "id").(string)
	r := hh.POST("/api/v1/admin/campaigns/"+id2+"/send", tok, map[string]any{})
	if got := campaignField(t, r, "status"); got != "sent" {
		t.Fatalf("**لم تُرسَل وقد وصلت الخدمة**: %v / %v", got, r.JSON()["error"])
	}
	if got := inboxCount(t, hh, sub2.ID, "promo"); got != 1 {
		t.Fatalf("**وصلت الخدمةُ ولم يُخبَر المشترِك**: %d", got)
	}
	// **SI-N-08 · ويُعلَّم بعد الإرسال.**
	var got2 *time.Time
	_ = hh.Pool.QueryRow(ctxBG(), `
		SELECT notified_at FROM coverage_requests
		 WHERE user_id = $1::uuid AND target_key = $2`, sub2.ID, near).Scan(&got2)
	if got2 == nil {
		t.Fatalf("**أُخبر ولم يُعلَّم اشتراكُه** — **فيُخبَر ثانيةً غداً**")
	}
}
