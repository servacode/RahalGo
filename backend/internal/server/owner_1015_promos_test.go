package server

// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك ٢٠٢٦-١٠-١٥ — العروضُ والأكواد وبقايا الديون**
// ══════════════════════════════════════════════════════════════════════
//
//	ب١ · عرضٌ واحدٌ قائمٌ على الصنف: عرضُ رحّال غو يغلب، والإدارةُ تستبدل عرضَ المتجر بتأكيد
//	ب٢ · الكودُ المنتهي يُعدَّل ولا يُفعَّل
//	ب٣ · رفعُ «للطلب الأوّل» أو «مرّةً للزبون» يعيد الكودَ للماليّة
//	ب٥ · سقفُ مبلغ خصم الصنف سعرُ البيع لا سعرُ الشراء
//	ج  · رفضُ الشطب بقدرة الموافقة · ومرشّحُ «ينتظر الموافقة»

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// asIs **سعرُ البيع = سعرُ الشراء** — لاختبارٍ لا يسأل عن الهامش.
func asIs(cost int64, _, _ *int64) int64 { return cost }

func (f offerFix) owner(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.s.pg.QueryRow(context.Background(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`, f.mid).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f offerFix) merchantOffer(t *testing.T, owner string) (*offers.Offer, error) {
	t.Helper()
	pct := 10
	return f.s.offers.CreateScoped(context.Background(), owner, f.mid, offers.Input{
		Title: "عرض المتجر", MenuItemID: &f.itemID, DiscountPercent: &pct,
	}, asIs)
}

func (f offerFix) active(t *testing.T, id string) bool {
	t.Helper()
	var on bool
	if err := f.s.pg.QueryRow(context.Background(),
		`SELECT active FROM offers WHERE id = $1::uuid`, id).Scan(&on); err != nil {
		t.Fatal(err)
	}
	return on
}

func appCode(err error) (string, map[string]any) {
	var e *httpx.AppError
	if errors.As(err, &e) {
		return e.Code, e.Details
	}
	return "", nil
}

// TestOFFER1015_PlatformOfferWinsOverMerchant **عرضُ رحّال غو يغلب — والمتجرُ يُردّ بتاريخه.**
//
// **كان عرضُ المتجر الجديدُ يُنزل عرضَ المنصّة** — خصمٌ قرّرته المنصّةُ يسقط بضغطة.
func TestOFFER1015_PlatformOfferWinsOverMerchant(t *testing.T) {
	f := newOfferFix(t, true)
	end := time.Now().In(offers.Damascus).AddDate(0, 0, 3).Format("2006-01-02")
	w := promoCall(f.s.handleCreateOffer, "POST", `{"title":"عرض المنصة","menu_item_id":"`+
		f.itemID+`","discount_amount":100,"ends_on":"`+end+`"}`, f.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("عرضُ المنصّة ردّ %d %s", w.Code, w.Body.String())
	}
	var platform offers.Offer
	_ = json.Unmarshal(dataOf(t, w), &platform)

	_, err := f.merchantOffer(t, f.owner(t))
	code, details := appCode(err)
	if code != "offer_platform_live" {
		t.Fatalf("**عرضُ المتجر قُبل وعلى الصنف عرضُ رحّال غو**: %v", err)
	}
	if details["until"] != end {
		t.Fatalf("الرفضُ بلا تاريخ النهاية: %v — والمنتظَر %s", details, end)
	}
	if !f.active(t, platform.ID) {
		t.Fatal("**نزل عرضُ المنصّة** بعرض المتجر")
	}
}

// TestOFFER1015_AdminReplaceNeedsConfirmAndTellsMerchant **الإدارةُ تستبدل بتأكيدٍ وتُخبر المتجر.**
func TestOFFER1015_AdminReplaceNeedsConfirmAndTellsMerchant(t *testing.T) {
	f := newOfferFix(t, true)
	ctx := context.Background()
	owner := f.owner(t)
	notes := func() int {
		var n int
		_ = f.s.pg.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND title = $2`, owner, "نزل عرضك — رحّال غو عملت عرض على نفس الصنف").Scan(&n)
		return n
	}
	mine, err := f.merchantOffer(t, owner)
	if err != nil {
		t.Fatalf("عرضُ المتجر: %v", err)
	}
	body := `{"title":"عرض المنصة","menu_item_id":"` + f.itemID + `","discount_amount":100`
	w := promoCall(f.s.handleCreateOffer, "POST", body+`}`, f.admin, "")
	if w.Code != http.StatusConflict || promoErrCode(w) != "offer_merchant_live" {
		t.Fatalf("**استُبدل عرضُ المتجر بلا تأكيد**: %d %s", w.Code, w.Body.String())
	}
	if !f.active(t, mine.ID) {
		t.Fatal("نزل عرضُ المتجر بلا تأكيد")
	}
	w = promoCall(f.s.handleCreateOffer, "POST", body+`,"replace":true}`, f.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("الاستبدالُ بالتأكيد ردّ %d %s", w.Code, w.Body.String())
	}
	if f.active(t, mine.ID) {
		t.Fatal("عرضُ المتجر باقٍ بعد الاستبدال")
	}
	if n := notes(); n != 1 {
		t.Fatalf("**صاحبُ المتجر لم يُخبَر** — %d إشعار", n)
	}

	// **وموافقةُ الماليّة على عرض منصّةٍ ينتظر تستبدل كذلك وتُخبر.**
	if _, err := f.s.pg.Exec(ctx, `UPDATE offers SET active = false
		WHERE menu_item_id = $1::uuid AND borne_by = 'platform'`, f.itemID); err != nil {
		t.Fatal(err)
	}
	mine, err = f.merchantOffer(t, owner)
	if err != nil {
		t.Fatalf("عرضُ المتجر الثاني: %v", err)
	}
	w = promoCall(f.s.handleCreateOffer, "POST", `{"title":"خصم كبير","menu_item_id":"`+
		f.itemID+`","discount_percent":40,"replace":true}`, f.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("عرضٌ ينتظر الماليّة ردّ %d %s", w.Code, w.Body.String())
	}
	var big offers.Offer
	_ = json.Unmarshal(dataOf(t, w), &big)
	if !f.active(t, mine.ID) {
		t.Fatal("نزل عرضُ المتجر قبل موافقة الماليّة")
	}
	var apID string
	if err := f.s.pg.QueryRow(ctx, `SELECT id::text FROM promo_approvals
		WHERE target_id = $1::uuid AND status = 'pending'`, big.ID).Scan(&apID); err != nil {
		t.Fatalf("لا طلبَ موافقة: %v", err)
	}
	finance := testdb.NewUser(t, f.s.pg, "customer")
	if w := promoCall(f.s.handleDecidePromoApproval(true), "POST", `{}`, finance, apID); w.Code != http.StatusOK {
		t.Fatalf("الموافقة %d %s", w.Code, w.Body.String())
	}
	if f.active(t, mine.ID) {
		t.Fatal("**وافقت الماليّةُ وبقي عرضُ المتجر**")
	}
	if n := notes(); n != 2 {
		t.Fatalf("الموافقةُ استبدلت ولم تُخبر — %d إشعار", n)
	}
}

// TestOFFER1015_FixedCutCappedBySalePrice **السقفُ سعرُ البيع لا سعرُ الشراء.**
//
// صنفٌ يُشترى بـ١٠٬٠٠٠ ويُباع بـ١٢٬٠٠٠: **خصمٌ بـ١١٬٠٠٠ كان يُردّ** لأنّه فوق سعر
// الشراء — والزبونُ يدفع ١٢٬٠٠٠.
func TestOFFER1015_FixedCutCappedBySalePrice(t *testing.T) {
	f := newOfferFix(t, true)
	owner := f.owner(t)
	sale := func(cost int64, _, _ *int64) int64 { return cost * 12 / 10 }
	create := func(amount int64) error {
		_, err := f.s.offers.CreateScoped(context.Background(), owner, f.mid, offers.Input{
			Title: "خصم ثابت", MenuItemID: &f.itemID, DiscountAmount: &amount,
		}, sale)
		return err
	}
	if err := create(11_000); err != nil {
		t.Fatalf("**خصمٌ تحت سعر البيع رُدّ**: %v", err)
	}
	if code, _ := appCode(create(12_000)); code != "offer_amount_over_price" {
		t.Fatalf("خصمٌ يبلغ سعرَ البيع قُبل: %q", code)
	}
}

// TestPROMO1015_ExpiredCodeEditableNotActivatable **الكودُ المنتهي يُعدَّل ولا يُفعَّل.**
func TestPROMO1015_ExpiredCodeEditableNotActivatable(t *testing.T) {
	s := promoServer(t)
	u := testdb.NewUser(t, s.pg, "customer")
	p := createPromo(t, s, u, `{"code":"`+uniqCode("X")+`","kind":"fixed","value":100,"min_order":1000,"max_uses":5}`)
	yday := time.Now().In(offers.Damascus).AddDate(0, 0, -1).Format("2006-01-02")
	end, _ := offers.EndOfDay(yday)
	if _, err := s.pg.Exec(context.Background(),
		`UPDATE promo_codes SET expires_at = $2 WHERE id = $1::uuid`, p.ID, end); err != nil {
		t.Fatal(err)
	}
	// **والنافذةُ تعيد إرسالَ تاريخه كما هو** — وهذا لا يمنع تعديلاً.
	w := promoCall(s.handleUpdatePromo, "PATCH",
		`{"min_order":2000,"max_uses":5,"expires_on":"`+yday+`"}`, u, p.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("**كودٌ منتهٍ لا يُعدَّل**: %d %s", w.Code, w.Body.String())
	}
	var got catalog.PromoCode
	_ = json.Unmarshal(dataOf(t, w), &got)
	if got.MinOrder != 2000 {
		t.Fatalf("الحدُّ الأدنى %d", got.MinOrder)
	}
	w = promoCall(s.handleUpdatePromo, "PATCH", `{"active":true}`, u, p.ID)
	if w.Code != http.StatusConflict || promoErrCode(w) != "promo_expired_activate" {
		t.Fatalf("**فُعّل كودٌ منتهٍ**: %d %s", w.Code, w.Body.String())
	}
}

// TestPROMO1015_BroadeningGoesBackToApproval **رفعُ قيد الجمهور يعيده للماليّة.**
func TestPROMO1015_BroadeningGoesBackToApproval(t *testing.T) {
	s := promoServer(t)
	ctx := context.Background()
	content := testdb.NewUser(t, s.pg, "customer")
	finance := testdb.NewUser(t, s.pg, "customer")
	for _, c := range []struct{ create, broaden string }{
		{`"first_order_only":true`, `{"first_order_only":false}`},
		{`"once_per_user":true`, `{"once_per_user":false}`},
	} {
		p := createPromo(t, s, content, `{"code":"`+uniqCode("W")+
			`","kind":"percent","value":30,"max_discount":5000,"max_uses":50,`+c.create+`}`)
		var apID string
		if err := s.pg.QueryRow(ctx, `SELECT id::text FROM promo_approvals
			WHERE target_id = $1::uuid AND status = 'pending'`, p.ID).Scan(&apID); err != nil {
			t.Fatal(err)
		}
		if w := promoCall(s.handleDecidePromoApproval(true), "POST", `{}`, finance, apID); w.Code != http.StatusOK {
			t.Fatalf("الموافقة %d", w.Code)
		}
		w := promoCall(s.handleUpdatePromo, "PATCH", c.broaden, content, p.ID)
		if w.Code != http.StatusOK {
			t.Fatalf("التوسيع %s ردّ %d %s", c.broaden, w.Code, w.Body.String())
		}
		var got catalog.PromoCode
		_ = json.Unmarshal(dataOf(t, w), &got)
		if got.ApprovalState != offers.ApprovalPending || got.Active {
			t.Fatalf("**وُسّع الكودُ بلا موافقة** (%s): الموافقة %s والسريان %v",
				c.broaden, got.ApprovalState, got.Active)
		}
	}
}

// ── ج · بقايا الديون ────────────────────────────────────────────────────

func oblFixture(t *testing.T, srv *Server, driver string, n int) []string {
	t.Helper()
	ctx := context.Background()
	var ids []string
	for i := 0; i < n; i++ {
		var src string
		if err := srv.pg.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&src); err != nil {
			t.Fatal(err)
		}
		if _, err := obligations.CreateRef(ctx, srv.pg, obligations.PartyDriver, driver, 500,
			obligations.CausePenaltyRemainder, "", src, nil); err != nil {
			t.Fatal(err)
		}
		var id string
		if err := srv.pg.QueryRow(ctx, `SELECT id::text FROM financial_obligations
			WHERE source_ref = $1`, src).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = srv.pg.Exec(c, `DELETE FROM obligation_requests WHERE obligation_id::text = ANY($1)`, ids)
		_, _ = srv.pg.Exec(c, `DELETE FROM financial_obligations WHERE id::text = ANY($1)`, ids)
		_, _ = srv.pg.Exec(c, `UPDATE users SET driver_debt = 0 WHERE id = $1`, driver)
	})
	return ids
}

func oblRequest(t *testing.T, srv *Server, obligationID, kind, by string) string {
	t.Helper()
	var id string
	if err := srv.pg.QueryRow(context.Background(), `
		INSERT INTO obligation_requests (obligation_id, kind, amount, note, proposed_by)
		VALUES ($1, $2, 500, 'فحص', $3) RETURNING id::text`, obligationID, kind, by).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestDEBT1015_RejectWriteoffNeedsApproveCapability **رفضُ الشطب كموافقته — بقدرة الشطب.**
func TestDEBT1015_RejectWriteoffNeedsApproveCapability(t *testing.T) {
	srv := promoServer(t)
	driver := testdb.NewUser(t, srv.pg, "driver")
	clerk := testdb.NewUser(t, srv.pg, "finance")
	boss := testdb.NewUser(t, srv.pg, "admin")
	ids := oblFixture(t, srv, driver, 1)
	req := oblRequest(t, srv, ids[0], oblReqWriteOff, clerk)

	w := callAs(srv.handleDecideObligationRequest(false), "POST", "/x", boss, req,
		[]string{string(authz.FinanceManage)})
	if w.Code != http.StatusForbidden || promoErrCode(w) != "writeoff_owner_only" {
		t.Fatalf("**رفض الشطبَ من لا يملك قدرتَه**: %d %s", w.Code, w.Body.String())
	}
	w = callAs(srv.handleDecideObligationRequest(false), "POST", "/x", boss, req,
		[]string{string(authz.FinanceManage), string(authz.FinanceWriteoffApprove)})
	if w.Code != http.StatusOK {
		t.Fatalf("رفضُ صاحب القدرة ردّ %d %s", w.Code, w.Body.String())
	}
}

// TestDEBT1015_PendingFilter **«ينتظر الموافقة» يعرض ما عليه طلبٌ وحدَه.**
func TestDEBT1015_PendingFilter(t *testing.T) {
	srv := promoServer(t)
	driver := testdb.NewUser(t, srv.pg, "driver")
	clerk := testdb.NewUser(t, srv.pg, "finance")
	ids := oblFixture(t, srv, driver, 2)
	oblRequest(t, srv, ids[1], oblReqOfficeCash, clerk)

	w := callAs(srv.handleListObligations, "GET",
		"/x?party_kind=driver&party_id="+driver+"&pending=1", clerk, "",
		[]string{string(authz.FinanceRead)})
	var out struct {
		Obligations []struct {
			ID string `json:"id"`
		} `json:"obligations"`
	}
	if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Obligations) != 1 || out.Obligations[0].ID != ids[1] {
		t.Fatalf("**المرشّحُ لا يعرض ما ينتظر الموافقة وحدَه**: %+v — والمنتظَر %s", out.Obligations, ids[1])
	}
}
