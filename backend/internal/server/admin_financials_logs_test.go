package server

// **أربعةُ سجلّاتٍ صارت في ملفّه — بعد أن كانت في أربع شاشات.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «نعم ضمن الماليّة».)
//
// # ولماذا حارسٌ لأربعِ قراءات
//
// **كلٌّ منها كان له شاشةٌ مستقلّة** — `/incentives` و`/payouts`
// و`/losses` و`/emergencies` — **فمن أراد أن يعرف سائقاً خرج من ملفّه
// وبحث عن اسمه في أربعة أماكن.**
//
// **وهذا ما بُني الملفُّ ليمنعه**: «يجمع كلَّ شيءٍ يخصّه — ولا نريد
// خسارةَ أيّ ميزة» (قرارُ المالك ٢٠٢٦-٠٨-٠٣).
//
// **والعقوبةُ تُقرأ سالبةً**: `incentives.amount` موجبٌ دائماً والنوعُ
// يقول الاتجاه — **وموجبان في عمودٍ واحدٍ يُجمعان خطأً**، فيبدو من عوقب
// كمن كوفئ.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// TestFinancialsLogs_FourRecordsLandInTheProfile
// **حافزٌ وعقوبةٌ وسحبٌ ونزاعٌ وبلاغُ طوارئ — كلُّها تُقرأ من ملفّه.**
func TestFinancialsLogs_FourRecordsLandInTheProfile(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	customer, _, _ := twoCustomers(t, f)
	driver := f.drivers[0]

	var orderID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
			dropoff, payment_method, subtotal, delivery_fee, total, cash_due)
		VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 3000, 13000, 13000)
		RETURNING id::text`, customer, f.merchantID, driver).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM incentives WHERE user_id = $1`, driver)
		_, _ = f.pool.Exec(c, `DELETE FROM payout_requests WHERE user_id = $1`, driver)
		_, _ = f.pool.Exec(c, `DELETE FROM disputes WHERE party_user_id = $1`, driver)
		_, _ = f.pool.Exec(c, `DELETE FROM driver_emergencies WHERE driver_id = $1`, driver)
		_, _ = f.pool.Exec(c, `DELETE FROM orders WHERE id = $1`, orderID)
	})

	exec := func(what, q string, args ...any) {
		if _, err := f.pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("تعذّر %s: %v", what, err)
		}
	}
	exec("المكافأة", `INSERT INTO incentives (user_id, kind, amount, reason, created_by)
		VALUES ($1, 'reward', 5000, 'أسرعُ تسليمٍ هذا الأسبوع', $2)`, driver, customer)
	exec("العقوبة", `INSERT INTO incentives (user_id, kind, amount, reason, created_by)
		VALUES ($1, 'penalty', 2000, 'تأخّرٌ متكرّر', $2)`, driver, customer)
	exec("طلبَ السحب", `INSERT INTO payout_requests (user_id, amount, note)
		VALUES ($1, 40000, 'تحويل')`, driver)
	exec("النزاع", `INSERT INTO disputes (party_role, party_user_id, order_id, reason, amount)
		VALUES ('driver', $1, $2, 'بضاعةٌ لم تُسلَّم', 7000)`, driver, orderID)
	exec("بلاغَ الطوارئ", `INSERT INTO driver_emergencies (driver_id, order_id, note)
		VALUES ($1, $2, 'عطلٌ في الدرّاجة')`, driver, orderID)

	var fin struct {
		Data struct {
			Incentives []struct {
				Amount int64  `json:"amount"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			} `json:"incentives"`
			Payouts     []struct{ Status string } `json:"payouts"`
			Disputes    []struct{ Ref string }    `json:"disputes"`
			Emergencies []struct {
				Ref    string `json:"ref"`
				Amount int64  `json:"amount"`
			} `json:"emergencies"`
			IncentivesN  int `json:"incentives_count"`
			PayoutsN     int `json:"payouts_count"`
			DisputesN    int `json:"disputes_count"`
			EmergenciesN int `json:"emergencies_count"`
		} `json:"data"`
	}
	w := asCustomer(f.srv.handleAdminUserFinancials, http.MethodGet, driver, driver)
	if w.Code != 200 {
		t.Fatalf("ردَّ %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &fin); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}
	d := fin.Data

	if d.IncentivesN != 2 || d.PayoutsN != 1 || d.DisputesN != 1 || d.EmergenciesN != 1 {
		t.Fatalf("الأعدادُ %d/%d/%d/%d — **وسجلٌّ لا يبلغ ملفَّه يُبحث عنه في شاشةٍ أخرى**",
			d.IncentivesN, d.PayoutsN, d.DisputesN, d.EmergenciesN)
	}

	// **والعقوبةُ سالبة** — وهو ما يفصلها عن المكافأة في عمودٍ واحد.
	var reward, penalty bool
	for _, x := range d.Incentives {
		if x.Status == "reward" && x.Amount == 5000 {
			reward = true
		}
		if x.Status == "penalty" && x.Amount == -2000 {
			penalty = true
		}
	}
	if !reward || !penalty {
		t.Fatalf("المكافأةُ والعقوبةُ لا تفترقان: %+v — **فيبدو من عوقب كمن كوفئ**",
			d.Incentives)
	}

	// **ومرجعُ النزاع رقمُ الطلب لا معرّفُه** — والشاشةُ تبني منه بحثاً،
	// **وبحثُ الطلبات يطابق الرقمَ لا المعرّف فيردّ «لا نتائج».**
	if len(d.Disputes) != 1 || d.Disputes[0].Ref == "" || len(d.Disputes[0].Ref) > 12 {
		t.Fatalf("مرجعُ النزاع %q — **ومعرّفٌ في خانة رقمٍ يُضغط فيردّ فراغا**",
			d.Disputes[0].Ref)
	}
	// **ولا مالَ في بلاغ الطوارئ** — **وصفرٌ معروضٌ يُقرأ مبلغا.**
	if len(d.Emergencies) != 1 || d.Emergencies[0].Amount != 0 {
		t.Fatalf("بلاغُ الطوارئ بمبلغ %d", d.Emergencies[0].Amount)
	}
	if d.Emergencies[0].Ref == "" {
		t.Fatal("بلاغٌ بلا رقم طلبٍ وله طلب — **ومن قرأه لا يعرف أين وقع**")
	}
}
