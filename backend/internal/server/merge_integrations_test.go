package server

// **ما وصله الدمجُ بين الأقسام** (٢٠٢٦-١٠-٠٤) — كلُّ وصلٍ بفحصه:
//
//   - كلُّ جدول موافقاتٍ مسجَّلٍ في `approvalSources` قائمٌ بأعمدة العقد بعد الهجرات
//     (والناقصُ يُتخطّى صامتاً، فلا يظهر قسمٌ في صفحة الموافقات الموحّدة).
//   - كلفةُ العروض سطورٌ لحالها في الأرباح والكشفُ يتجمّع.
//   - سدادُ الدَّين من شحن المحفظة مالٌ مستردٌّ لا حركةٌ خارج الربح.
//   - الديونُ المشطوبةُ رقمٌ في صفحة الخسائر خارجَ مجموعها.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestMERGE_EveryApprovalSourceTableHasTheContract(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	for _, src := range approvalSources {
		need := []string{"id", "status", "created_at", col(src.AmountCol, "amount"), col(src.NoteCol, "note")}
		if p := col(src.ProposerCol, "proposed_by"); p != "-" {
			need = append(need, p)
		}
		if src.DueCol != "" {
			need = append(need, src.DueCol)
		}
		ok, err := tableHasColumns(ctx, srv.pg, src.Table, need)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("مصدرُ الموافقات %q: الجدول %s غائبٌ أو ينقصه عمودٌ من %v — فيُتخطّى صامتاً", src.Key, src.Table, need)
		}
	}
	for _, want := range []string{"wallet_requests", "expense_requests", "driver_compensation_requests",
		"obligation_requests", "incentive_requests", "dispute_resolutions", "promo_approvals"} {
		found := false
		for _, src := range approvalSources {
			if src.Table == want {
				found = true
			}
		}
		if !found {
			t.Errorf("جدولُ الموافقات %s غيرُ مسجَّلٍ في صفحة الموافقات الموحّدة", want)
		}
	}
}

func TestMERGE_PromoCostsAreTheirOwnProfitLines(t *testing.T) {
	f := newProfitsFx(t, "فحص كلفة العروض")
	ctx := context.Background()
	day := damascusAt(t, "2001-05-11 12:00")
	a := f.order(t, "delivered", day)
	// **توصيلٌ مجانيٌّ بكود** (أُعفي الزبونُ من ٢٠٠٠) · **وصنفٌ بخصمٍ على المنصّة**
	// (٥٠٠ على الوحدة) · **وصنفٌ بخصمٍ على المتجر** (٣٠٠ — لا يمسّ ربحَ المنصّة).
	if _, err := f.srv.pg.Exec(ctx, `
		UPDATE orders SET delivery_fee = 0, promo_delivery_waived = 2000, total = 9500 WHERE id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.pg.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, a); err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.pg.Exec(ctx, `
		INSERT INTO order_items (order_id, merchant_id, name, unit_price, merchant_price, qty, options,
		                         offer_cut, offer_borne_by)
		VALUES ($1, $2, 'صنف بخصم المنصة', 9500, 9000, 1, '[]', 500, 'platform'),
		       ($1, $2, 'صنف بخصم المتجر', 4000, 3500, 1, '[]', 300, 'merchant')`, a, f.merchant); err != nil {
		t.Fatal(err)
	}
	// هامشٌ قبل الخصم ١٥٠٠ + عمولة ٩٠٠ + توصيل ٢٠٠ − كود ٥٠٠ − توصيل مجاني ٢٠٠٠ − خصم صنف ٥٠٠ = −٤٠٠.
	f.post(t, f.treasury, -400, "platform_profit", a, day)

	r := readPlatformProfits(t, f.srv, "2001-05-11")
	want := map[string]int64{
		"margin": 1500, "commission": 900, "delivery_share": 200, "discount": -500,
		"free_delivery": -2000, "item_discounts": -500, "settle_diff": 0,
	}
	for k, v := range want {
		if got := r.line(k); got != v {
			t.Errorf("سطر %s = %d، المتوقَّع %d", k, got, v)
		}
	}
	if r.Net != -400 || r.LinesSum != r.Net || !r.CheckOK {
		t.Fatalf("الصافي %d ومجموع البنود %d والتحقّق %v — المتوقَّع −400 ومتساويان", r.Net, r.LinesSum, r.CheckOK)
	}
}

func TestMERGE_DebtRepaidFromTopupIsRecovered(t *testing.T) {
	f := newProfitsFx(t, "فحص سداد الدين بالشحن")
	day := damascusAt(t, "2001-06-11 12:00")
	// **الشحنُ يسدّ الدَّينَ أوّلاً** (قسمُ الديون): قيدُ الخزينة `platform_profit`
	// مرجعُه طلبُ الشحن — مالٌ مستردٌّ كسداد الدَّين نقداً بالمكتب.
	wr := f.wr(t, "topup", false, 1000)
	f.post(t, f.treasury, 700, "platform_profit", wr, day)
	r := readPlatformProfits(t, f.srv, "2001-06-11")
	if got := r.line("recovered"); got != 700 {
		t.Fatalf("المستردّ %d لا 700 — سدادُ الدَّين من الشحن عُدّ حركةً خارج الربح (%d)", got, r.Outside)
	}
	if r.Outside != 0 || !r.CheckOK {
		t.Fatalf("خارج الربح %d والتحقّق %v", r.Outside, r.CheckOK)
	}
}

func TestMERGE_LossesShowWrittenOffDebtsOutsideTotal(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	rep := testdb.NewUser(t, srv.pg, "sales")
	boss := testdb.NewUser(t, srv.pg, "admin")
	var obl, req string
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO financial_obligations (party_kind, party_id, amount, cause)
		VALUES ('rep', $1, 4200, 'legacy_opening') RETURNING id::text`, rep).Scan(&obl); err != nil {
		t.Fatal(err)
	}
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO obligation_requests (obligation_id, kind, amount, note, status, proposed_by,
		                                 decided_by, decided_at)
		VALUES ($1, 'write_off', 4200, 'لن يُسدَّد', 'approved', $2, $2, '2001-07-11 12:00+03')
		RETURNING id::text`, obl, boss).Scan(&req); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM financial_obligations WHERE id = $1`, obl)
	})
	w := callAs(srv.handlePlatformLosses, "GET", "/x?from=2001-07-11&to=2001-07-11", "", "", financeCaps)
	var out struct {
		Total      int64 `json:"total"`
		WrittenOff int64 `json:"written_off"`
	}
	if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
		t.Fatal(err)
	}
	if out.WrittenOff != 4200 {
		t.Fatalf("الديونُ المشطوبة %d لا 4200", out.WrittenOff)
	}
	if out.Total != 0 {
		t.Fatalf("الشطبُ دخل مجموعَ الخسائر (%d) — ولا قيدَ له", out.Total)
	}
}
