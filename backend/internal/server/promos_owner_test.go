package server

// قسمُ «العروض والخصومات» — قراراتُ المالك ٢٠٢٦-١٠-٠٤.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func promoServer(t *testing.T) *Server {
	t.Helper()
	srv := overviewServer(t)
	srv.catalog = catalog.NewService(srv.pg, nil)
	srv.offers = offers.New(srv.pg)
	srv.notify = notifications.New(srv.pg, srv.hub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return srv
}

func promoCall(h http.HandlerFunc, method, body, userID, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func promoErrCode(w *httptest.ResponseRecorder) string {
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Error.Code != "" {
		return env.Error.Code
	}
	return env.Code
}

var promoSeq int64

func uniqCode(p string) string {
	promoSeq++
	return fmt.Sprintf("%s%d%d", p, time.Now().UnixNano()%1_000_000, promoSeq)
}

func createPromo(t *testing.T, s *Server, user, body string) catalog.PromoCode {
	t.Helper()
	w := promoCall(s.handleCreatePromo, "POST", body, user, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الكود ردّ %d: %s", w.Code, w.Body.String())
	}
	var p catalog.PromoCode
	if err := json.Unmarshal(dataOf201(t, w), &p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		s.pg.Exec(ctx, `DELETE FROM promo_approvals WHERE target_id = $1::uuid`, p.ID)
		s.pg.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1::uuid`, p.ID)
	})
	return p
}

func dataOf201(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

// ── ٢ · نسبةٌ ١–٩٠ وسقفٌ بالليرة إجباريّ، ورسائلُ واضحة ─────────────────
func TestPROMOS_ServerValidatesCode(t *testing.T) {
	s := promoServer(t)
	u := testdb.NewUser(t, s.pg, "customer")
	cases := []struct{ body, code string }{
		{`{"code":"` + uniqCode("A") + `","kind":"percent","value":150,"max_discount":1000,"max_uses":10}`, "promo_percent_range"},
		{`{"code":"` + uniqCode("B") + `","kind":"percent","value":10,"max_uses":10}`, "promo_cap_required"},
		{`{"code":"` + uniqCode("C") + `","kind":"fixed","value":-5,"max_uses":10}`, "promo_negative"},
		{`{"code":"AB CD","kind":"fixed","value":100,"max_uses":10}`, "promo_code_invalid"},
	}
	for _, c := range cases {
		w := promoCall(s.handleCreatePromo, "POST", c.body, u, "")
		if w.Code != http.StatusBadRequest || promoErrCode(w) != c.code {
			t.Errorf("%s ⇒ %d %s، والمنتظَر 400 %s", c.body, w.Code, w.Body.String(), c.code)
		}
	}
	// **والفراغاتُ حول الكود تُشال**.
	raw := uniqCode("trim")
	p := createPromo(t, s, u, `{"code":"  `+strings.ToLower(raw)+`  ","kind":"fixed","value":100,"min_order":1000,"max_uses":5}`)
	if p.Code != strings.ToUpper(raw) {
		t.Fatalf("الكودُ حُفظ %q", p.Code)
	}
}

// ── ١ · فوق ٢٠٪ أو ٥٠ استخداماً ينتظر الماليّة، ومن اقترح لا يوافق ──────────
func TestPROMOS_OverLimitNeedsFinanceApproval(t *testing.T) {
	s := promoServer(t)
	content := testdb.NewUser(t, s.pg, "customer")
	finance := testdb.NewUser(t, s.pg, "customer")

	free := createPromo(t, s, content, `{"code":"`+uniqCode("F")+`","kind":"percent","value":20,"max_discount":5000,"max_uses":50}`)
	if free.Status != catalog.PromoActive || free.ApprovalState != "ok" {
		t.Fatalf("كودٌ تحت الحدّ: الحالة %s والموافقة %s", free.Status, free.ApprovalState)
	}

	big := createPromo(t, s, content, `{"code":"`+uniqCode("G")+`","kind":"percent","value":30,"max_discount":5000,"max_uses":50}`)
	if big.Status != catalog.PromoPending || big.Active {
		t.Fatalf("كودٌ فوق الحدّ سرى بلا موافقة: %s active=%v", big.Status, big.Active)
	}
	var apID string
	if err := s.pg.QueryRow(context.Background(), `
		SELECT id::text FROM promo_approvals WHERE target_id = $1::uuid AND status = 'pending'`,
		big.ID).Scan(&apID); err != nil {
		t.Fatalf("لا طلبَ موافقة: %v", err)
	}
	// **وتفعيلُه باليد يُردّ.**
	if w := promoCall(s.handleUpdatePromo, "PATCH", `{"active":true}`, content, big.ID); w.Code != http.StatusConflict {
		t.Fatalf("تفعيلُ كودٍ ينتظر الماليّة ردّ %d", w.Code)
	}
	// **ومن اقترح لا يوافق.**
	if w := promoCall(s.handleDecidePromoApproval(true), "POST", `{}`, content, apID); w.Code != http.StatusForbidden {
		t.Fatalf("المقترحُ وافق على نفسه: %d %s", w.Code, w.Body.String())
	}
	if w := promoCall(s.handleDecidePromoApproval(true), "POST", `{}`, finance, apID); w.Code != http.StatusOK {
		t.Fatalf("موافقةُ الماليّة ردّت %d %s", w.Code, w.Body.String())
	}
	var active bool
	var state string
	if err := s.pg.QueryRow(context.Background(),
		`SELECT active, approval_state FROM promo_codes WHERE id = $1::uuid`, big.ID).Scan(&active, &state); err != nil {
		t.Fatal(err)
	}
	if !active || state != "ok" {
		t.Fatalf("بعد الموافقة: active=%v state=%s", active, state)
	}
}

// ── الحالةُ الحقيقيّة وعدّادُ الرئيسيّة ───────────────────────────────────
func TestPROMOS_RealStatusAndHomeCount(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	two := 2
	if got := catalog.PromoStatusAt(true, "ok", &past, nil, 0, now); got != catalog.PromoExpired {
		t.Errorf("منتهٍ ⇒ %s", got)
	}
	if got := catalog.PromoStatusAt(true, "ok", nil, &two, 2, now); got != catalog.PromoExhausted {
		t.Errorf("خلص سقفه ⇒ %s", got)
	}
	if got := catalog.PromoStatusAt(false, "ok", nil, nil, 0, now); got != catalog.PromoPaused {
		t.Errorf("موقوف ⇒ %s", got)
	}

	s := promoServer(t)
	ctx := context.Background()
	count := func() int64 {
		var n int64
		if err := s.pg.QueryRow(ctx, `SELECT count(*) FROM promo_codes p WHERE `+catalog.PromoLiveSQL).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	var id string
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, value, active, expires_at)
		VALUES ($1, 'fixed', 100, true, now() - interval '1 day') RETURNING id::text`, uniqCode("EXP")).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.pg.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1::uuid`, id) })
	if count() != before {
		t.Fatal("**كودٌ منتهٍ عُدّ سارياً في الرئيسيّة**")
	}
}

// ── التعديلُ يمسح التاريخَ والسقف، والنهايةُ آخرُ يوم دمشق ─────────────────
func TestPROMOS_EditClearsAndEndOfDayDamascus(t *testing.T) {
	end, err := offers.EndOfDay("2026-10-10")
	if err != nil {
		t.Fatal(err)
	}
	if got := end.In(offers.Damascus).Format("2006-01-02 15:04:05"); got != "2026-10-10 23:59:59" {
		t.Fatalf("نهايةُ اليوم = %s", got)
	}
	s := promoServer(t)
	u := testdb.NewUser(t, s.pg, "customer")
	day := time.Now().In(offers.Damascus).Format("2006-01-02")
	p := createPromo(t, s, u, `{"code":"`+uniqCode("E")+`","kind":"fixed","value":100,"min_order":1000,"max_uses":5,"expires_on":"`+day+`"}`)
	if p.ExpiresAt == nil || p.Status != catalog.PromoActive {
		t.Fatalf("اختيارُ اليوم نفسِه: %v %s", p.ExpiresAt, p.Status)
	}
	w := promoCall(s.handleUpdatePromo, "PATCH", `{"clear_expiry":true,"max_uses":3}`, u, p.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("التعديل ردّ %d %s", w.Code, w.Body.String())
	}
	var got catalog.PromoCode
	_ = json.Unmarshal(dataOf(t, w), &got)
	if got.ExpiresAt != nil || got.MaxUses == nil || *got.MaxUses != 3 {
		t.Fatalf("بعد التعديل: %v %v", got.ExpiresAt, got.MaxUses)
	}
}

// ── خصمُ الأصناف من الإدارة ──────────────────────────────────────────────
type offerFix struct {
	s      *Server
	admin  string
	itemID string
	mid    string
}

func newOfferFix(t *testing.T, available bool) offerFix {
	t.Helper()
	s := promoServer(t)
	ctx := context.Background()
	owner := testdb.NewUser(t, s.pg, "merchant")
	var cat, mid, sec, item string
	if err := s.pg.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, status, category_id)
		VALUES ('متجرُ العروض', $1, 'active', $2) RETURNING id::text`, owner, cat).Scan(&mid); err != nil {
		t.Fatal(err)
	}
	if err := s.pg.QueryRow(ctx, `INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قسم') RETURNING id::text`, mid).Scan(&sec); err != nil {
		t.Fatal(err)
	}
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name, price, merchant_price, available, approved)
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1), 'صنفُ عرض', 10000, 10000, $3, true)
		RETURNING id::text`, mid, sec, available).Scan(&item); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.pg.Exec(ctx, `DELETE FROM promo_approvals WHERE target_id IN (SELECT id FROM offers WHERE menu_item_id = $1::uuid)`, item)
		s.pg.Exec(ctx, `DELETE FROM offers WHERE menu_item_id = $1::uuid`, item)
		s.pg.Exec(ctx, `DELETE FROM menu_items WHERE id = $1::uuid`, item)
		s.pg.Exec(ctx, `DELETE FROM menu_sections WHERE id = $1::uuid`, sec)
		s.pg.Exec(ctx, `DELETE FROM merchants WHERE id = $1::uuid`, mid)
	})
	return offerFix{s: s, admin: testdb.NewUser(t, s.pg, "customer"), itemID: item, mid: mid}
}

func TestPROMOS_AdminOfferRules(t *testing.T) {
	f := newOfferFix(t, true)
	post := func(body string) *httptest.ResponseRecorder {
		return promoCall(f.s.handleCreateOffer, "POST", body, f.admin, "")
	}
	// ٤ · الإدارةُ لا تحمّل الخصمَ على المتجر.
	if w := post(`{"title":"خصم","menu_item_id":"` + f.itemID + `","discount_percent":10,"borne_by":"merchant"}`); w.Code != http.StatusBadRequest || promoErrCode(w) != "offer_admin_cannot_charge_store" {
		t.Fatalf("تحميلُ المتجر ردّ %d %s", w.Code, w.Body.String())
	}
	// ١ · فوق ٢٠٪ على المنصّة ينتظر الماليّة.
	w := post(`{"title":"خصم كبير","menu_item_id":"` + f.itemID + `","discount_percent":40}`)
	if w.Code != http.StatusOK {
		t.Fatalf("الإنشاء ردّ %d %s", w.Code, w.Body.String())
	}
	var o offers.Offer
	_ = json.Unmarshal(dataOf(t, w), &o)
	if o.Live || o.ApprovalState != "pending" || o.CreatedByKind != "admin" {
		t.Fatalf("خصمٌ ٤٠٪ سرى بلا موافقة: live=%v state=%s by=%s", o.Live, o.ApprovalState, o.CreatedByKind)
	}
	if w := promoCall(f.s.handleSetOfferActive, "POST", `{"active":true}`, f.admin, o.ID); w.Code != http.StatusConflict {
		t.Fatalf("تفعيلُ ما ينتظر الماليّة ردّ %d", w.Code)
	}
	// ومبلغٌ ثابتٌ تحت الحدّ يسري.
	w = post(`{"title":"خصم صغير","menu_item_id":"` + f.itemID + `","discount_amount":100}`)
	if w.Code != http.StatusOK {
		t.Fatalf("مبلغٌ ثابت ردّ %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(dataOf(t, w), &o)
	if !o.Live || o.DiscountAmount == nil || o.BorneBy == nil || *o.BorneBy != offers.ByPlatform {
		t.Fatalf("المبلغُ الثابت: live=%v", o.Live)
	}
	// وتفعيلُ منتهٍ يُقال سببُه.
	if _, err := f.s.pg.Exec(context.Background(),
		`UPDATE offers SET ends_at = now() - interval '1 hour', active = false WHERE id = $1::uuid`, o.ID); err != nil {
		t.Fatal(err)
	}
	if w := promoCall(f.s.handleSetOfferActive, "POST", `{"active":true}`, f.admin, o.ID); w.Code != http.StatusConflict || promoErrCode(w) != "offer_expired" {
		t.Fatalf("تفعيلُ منتهٍ ردّ %d %s", w.Code, w.Body.String())
	}
}

func TestPROMOS_AdminCannotDiscountUnavailableItem(t *testing.T) {
	f := newOfferFix(t, false)
	w := promoCall(f.s.handleCreateOffer, "POST",
		`{"title":"خصم","menu_item_id":"`+f.itemID+`","discount_percent":10}`, f.admin, "")
	if w.Code != http.StatusConflict || promoErrCode(w) != "offer_item_unavailable" {
		t.Fatalf("خصمٌ على صنفٍ غير متوفر ردّ %d %s", w.Code, w.Body.String())
	}
}

// ── الملخّصُ و«ادعُ صديقاً» يُقرآن ─────────────────────────────────────
func TestPROMOS_SummaryAndReferralsRead(t *testing.T) {
	s := promoServer(t)
	var sum promoSummary
	if err := json.Unmarshal(dataOf(t, promoCall(s.handlePromoSummary, "GET", "", "", "")), &sum); err != nil {
		t.Fatal(err)
	}
	if sum.PlatformCost != sum.Codes+sum.FreeDelivery+sum.PlatformItems+sum.Referrals {
		t.Fatalf("كلفةُ المنصّة لا تساوي مجموعَ بنودها: %+v", sum)
	}
	var ref map[string]any
	if err := json.Unmarshal(dataOf(t, promoCall(s.handleAdminReferrals, "GET", "", "", "")), &ref); err != nil {
		t.Fatal(err)
	}
	if _, ok := ref["referrals"]; !ok {
		t.Fatalf("لا جدولَ دعوات: %v", ref)
	}
}
