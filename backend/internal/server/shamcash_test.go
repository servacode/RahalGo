package server

// **سجلّ شام كاش اليدويّ** (قرارُ المالك ٢٠٢٦-١١-١٠) — التسجيلُ عند التسليم بعملتيه وجزئيّاً وبلا تكرار،
// والأرصدةُ وإيداعُ المالك وسحبُه، و«سلّمت السائق» و«ما وصل» وتراجعُهما، والصلاحيّات —
// **وأنّ المالَ لا يتحرّك**: طلبان متطابقان، أحدُهما بشام كاش، أثرُهما في الدفتر والصندوق والدرج واحد.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// shamcashOrder **طلبٌ نقديٌّ عند باب الزبون** بيد السائق — جاهزٌ لـ«سُلّم».
func (f *driverFixture) shamcashOrder(t *testing.T, driverID string) string {
	t.Helper()
	f.setSetting(t, "drivers.require_delivery_photo", false)
	return f.problemOrderAt(t, "at_dropoff", driverID)
}

func (f *driverFixture) shamcashRows(t *testing.T, orderID string) []string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `
		SELECT kind || ':' || currency || ':' || amount || ':' || status || ':' || driver_id::text
		FROM shamcash_movements WHERE order_id = $1`, orderID)
	if err != nil {
		t.Fatalf("قراءةُ شام كاش: %v", err)
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

func TestShamcash_DeliverRecordsSYPPartial(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	id := f.shamcashOrder(t, d)
	// **جزءٌ بشام كاش والباقي نقد** — الطلبُ ٢٥٠٠٠ وحُوّل ١٠٠٠٠.
	w := f.transitionKeyed(id, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":10000}}`, "sc-1")
	if w.Code != http.StatusOK {
		t.Fatalf("التسليم %d: %s", w.Code, w.Body.String())
	}
	got := f.shamcashRows(t, id)
	want := "driver_transfer:SYP:10000:claimed:" + d
	if len(got) != 1 || got[0] != want {
		t.Fatalf("السجلّ %v — المتوقَّع %s", got, want)
	}
	// **وإعادةُ النداء بالمفتاح نفسِه لا تكرّره** (ضاع الردّ في الشبكة).
	w = f.transitionKeyed(id, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":10000}}`, "sc-1")
	if w.Code != http.StatusOK || w.Header().Get("Idempotent-Replay") != "true" {
		t.Fatalf("الإعادة %d: %s", w.Code, w.Body.String())
	}
	if got := f.shamcashRows(t, id); len(got) != 1 {
		t.Fatalf("الإعادةُ كرّرت السجلّ: %v", got)
	}
	// **والكرتُ في اللوحة يقول «انقبض شام كاش»**.
	info, err := f.srv.orders.BoardInfoOf(context.Background(), []string{id})
	if err != nil || info[id] == nil || info[id].ShamcashAmount == nil || *info[id].ShamcashAmount != 10000 ||
		*info[id].ShamcashCurrency != "SYP" {
		t.Fatalf("معلومةُ الكرت: %+v %v", info[id], err)
	}
}

func TestShamcash_DeliverUSDDecimals(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	id := f.shamcashOrder(t, d)
	// **ثلاثُ منازل مرفوضة** — والطلبُ لا يُسلَّم بسببها.
	if w := f.transitionKeyed(id, d, `{"to":"delivered","shamcash":{"currency":"USD","amount":12.345}}`, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("١٢٫٣٤٥$ قُبلت: %d %s", w.Code, w.Body.String())
	}
	if st, _ := f.orderRow(t, id); st != "at_dropoff" {
		t.Fatalf("الطلبُ تحرّك بمبلغٍ مرفوض: %s", st)
	}
	for _, bad := range []string{
		`{"currency":"EUR","amount":5}`, `{"currency":"USD","amount":0}`,
		`{"currency":"SYP","amount":-5}`, `{"currency":"SYP","amount":10.5}`, `{"currency":"SYP"}`,
	} {
		if w := f.transitionKeyed(id, d, `{"to":"delivered","shamcash":`+bad+`}`, ""); w.Code != http.StatusBadRequest {
			t.Errorf("%s قُبل: %d", bad, w.Code)
		}
	}
	if w := f.transitionKeyed(id, d, `{"to":"delivered","shamcash":{"currency":"USD","amount":12.5}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("١٢٫٥$ %d: %s", w.Code, w.Body.String())
	}
	if got := f.shamcashRows(t, id); len(got) != 1 || got[0] != "driver_transfer:USD:1250:claimed:"+d {
		t.Fatalf("الدولارُ بالسنتات: %v", got)
	}
}

func TestShamcash_OldClientSendsNothing(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	id := f.shamcashOrder(t, d)
	if w := f.transitionKeyed(id, d, `{"to":"delivered"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("التسليم %d: %s", w.Code, w.Body.String())
	}
	if got := f.shamcashRows(t, id); len(got) != 0 {
		t.Fatalf("تطبيقٌ قديمٌ كتب سجلّاً: %v", got)
	}
	// **وحقلٌ مع غير «سُلّم» يُتجاهَل** — لا يُكتب شيءٌ على خطوةٍ أخرى.
	id2 := f.problemOrderAt(t, "picked_up", d)
	if w := f.transitionKeyed(id2, d, `{"to":"on_the_way","shamcash":{"currency":"SYP","amount":1}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("الانطلاق %d: %s", w.Code, w.Body.String())
	}
	if got := f.shamcashRows(t, id2); len(got) != 0 {
		t.Fatalf("سجلٌّ على غير التسليم: %v", got)
	}
}

// moneyFootprint **أثرُ الطلب في كلّ ما يحمل مالاً** — بلا معرّفاتٍ ولا أوقات، فيُقارَن طلبان.
func (f *driverFixture) moneyFootprint(t *testing.T, orderID, driverID string) []string {
	t.Helper()
	ctx := context.Background()
	var out []string
	q := func(label, sql string, args ...any) {
		rows, err := f.pool.Query(ctx, sql, args...)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		defer rows.Close()
		for rows.Next() {
			vals, _ := rows.Values()
			out = append(out, label+fmt.Sprint(vals))
		}
	}
	q("wallet", `SELECT wt.kind, wt.amount, (wt.user_id = $2) FROM wallet_transactions wt
		WHERE wt.ref = $1::text ORDER BY wt.kind, wt.amount`, orderID, driverID)
	q("driver_cash", `SELECT kind, amount FROM driver_cash_entries WHERE ref = $1::text ORDER BY kind, amount`, orderID)
	q("box", `SELECT held FROM driver_cash_boxes WHERE driver_id = $1`, driverID)
	q("office", `SELECT direction, amount, source FROM office_cash_entries WHERE ref = $1::text ORDER BY 1, 2`, orderID)
	q("obligations", `SELECT cause, amount FROM financial_obligations WHERE order_id = $1 ORDER BY 1, 2`, orderID)
	q("order", `SELECT status, cash_due, wallet_paid, total FROM orders WHERE id = $1`, orderID)
	sort.Strings(out)
	return out
}

// TestShamcash_LedgerUntouched **المالُ لا يتحرّك** — طلبان متطابقان لسائقين، أحدُهما بشام كاش كاملاً:
// الأثرُ في المحفظة وصندوق السائق والدرج والديون والطلب واحد، **وثوابتُ المال لا خرقَ جديداً فيها.**
func TestShamcash_LedgerUntouched(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	base, err := fininv.Capture(ctx, f.pool)
	if err != nil {
		t.Fatalf("صورةُ الثوابت: %v", err)
	}
	cash := f.shamcashOrder(t, f.drivers[0])
	sham := f.shamcashOrder(t, f.drivers[1])
	if w := f.transitionKeyed(cash, f.drivers[0], `{"to":"delivered"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("نقدي %d: %s", w.Code, w.Body.String())
	}
	if w := f.transitionKeyed(sham, f.drivers[1], `{"to":"delivered","shamcash":{"currency":"SYP","amount":25000}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("شام كاش %d: %s", w.Code, w.Body.String())
	}
	a, b := f.moneyFootprint(t, cash, f.drivers[0]), f.moneyFootprint(t, sham, f.drivers[1])
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Fatalf("**شام كاش حرّك المال**\nنقدي:\n%s\nشام كاش:\n%s", strings.Join(a, "\n"), strings.Join(b, "\n"))
	}
	if len(a) < 3 {
		t.Fatalf("الأثرُ فارغٌ على غير المتوقَّع — لا يُقارن شيئاً: %v", a)
	}
	after, err := fininv.Run(ctx, f.pool)
	if err != nil {
		t.Fatalf("الثوابت: %v", err)
	}
	// **والعُدّةُ تضع الطلبَ عند الباب بيدها** (بلا استلامٍ حقيقيّ) فقد يخرق الطلبان فحصاً معاً —
	// **والمطلوبُ أنّ شام كاش لا يُحدث خرقاً لا يُحدثه النقد**: ما يصيب أحدَهما يصيب الآخر.
	var nCash, nSham int64
	if err := f.pool.QueryRow(ctx, `SELECT (SELECT number FROM orders WHERE id = $1), (SELECT number FROM orders WHERE id = $2)`,
		cash, sham).Scan(&nCash, &nSham); err != nil {
		t.Fatal(err)
	}
	for _, v := range base.New(after) {
		var hitCash, hitSham int
		for _, r := range v.Rows {
			for _, cell := range r {
				c := fmt.Sprint(cell)
				if c == fmt.Sprint(nCash) || c == cash {
					hitCash++
				}
				if c == fmt.Sprint(nSham) || c == sham {
					hitSham++
				}
			}
		}
		if hitSham != hitCash {
			t.Errorf("خرقٌ يخصّ شام كاش وحدَه (نقدي %d · شام كاش %d): %s", hitCash, hitSham, v)
		}
	}
}

func shamcashCall(h http.HandlerFunc, method, path, id, body, actor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

type shamcashSummaryReply struct {
	Balances []struct {
		Currency      string `json:"currency"`
		Opening       int64  `json:"opening"`
		Balance       int64  `json:"balance"`
		NotArrived    int64  `json:"not_arrived"`
		OwedToDrivers int64  `json:"owed_to_drivers"`
	} `json:"balances"`
	Owed []struct {
		DriverID string `json:"driver_id"`
		Currency string `json:"currency"`
		Amount   int64  `json:"amount"`
	} `json:"owed_by_driver"`
}

func (f *driverFixture) shamcashSummary(t *testing.T, admin string) (syp, usd int64, sypOwed, sypNotArrived int64, owed map[string]int64) {
	t.Helper()
	w := shamcashCall(f.srv.handleShamcashSummary, "GET", "/admin/shamcash/summary", "", "", admin)
	if w.Code != http.StatusOK {
		t.Fatalf("الملخّص %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data shamcashSummaryReply `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	s := env.Data
	for _, b := range s.Balances {
		if b.Currency == "SYP" {
			syp, sypOwed, sypNotArrived = b.Balance, b.OwedToDrivers, b.NotArrived
		} else {
			usd = b.Balance
		}
	}
	owed = map[string]int64{}
	for _, o := range s.Owed {
		owed[o.DriverID+":"+o.Currency] = o.Amount
	}
	return
}

func TestShamcash_BalancesAndToggles(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	admin := testdb.NewUser(t, f.pool, "admin")
	ctx := context.Background()
	// **الجدولان لهذا الاختبار وحدَه** — لا يكتب فيهما غيرُه.
	if _, err := f.pool.Exec(ctx, `DELETE FROM shamcash_movements; DELETE FROM shamcash_opening`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM shamcash_movements; DELETE FROM shamcash_opening`)
	})

	// تحويلٌ قبل تاريخ رصيد البداية — داخلٌ فيه فلا يُعدّ مرّتين.
	old := f.shamcashOrder(t, d)
	if w := f.transitionKeyed(old, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":7000}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if _, err := f.pool.Exec(ctx, `UPDATE shamcash_movements SET occurred_at = now() - interval '3 days' WHERE order_id = $1`, old); err != nil {
		t.Fatal(err)
	}
	yesterday := time.Now().In(damascusLoc()).AddDate(0, 0, -1).Format("2006-01-02")
	if w := shamcashCall(f.srv.handleShamcashOpening, "PUT", "/admin/shamcash/opening", "",
		`{"currency":"SYP","amount":100000,"as_of":"`+yesterday+`"}`, admin); w.Code != http.StatusOK {
		t.Fatalf("رصيدُ البداية %d: %s", w.Code, w.Body.String())
	}
	if w := shamcashCall(f.srv.handleShamcashOpening, "PUT", "/admin/shamcash/opening", "",
		`{"currency":"USD","amount":20.25,"as_of":"`+yesterday+`"}`, admin); w.Code != http.StatusOK {
		t.Fatalf("رصيدُ البداية $ %d: %s", w.Code, w.Body.String())
	}
	if w := shamcashCall(f.srv.handleShamcashOpening, "PUT", "/admin/shamcash/opening", "",
		`{"currency":"USD","amount":1.234,"as_of":"`+yesterday+`"}`, admin); w.Code != http.StatusBadRequest {
		t.Fatalf("ثلاثُ منازل قُبلت: %d", w.Code)
	}
	id := f.shamcashOrder(t, d)
	if w := f.transitionKeyed(id, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":10000}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	add := func(body string) string {
		w := shamcashCall(f.srv.handleShamcashAdd, "POST", "/admin/shamcash/movements", "", body, admin)
		if w.Code != http.StatusCreated {
			t.Fatalf("حركةُ المالك %s: %d %s", body, w.Code, w.Body.String())
		}
		var r struct {
			Data struct {
				Movement struct {
					ID string `json:"id"`
				} `json:"movement"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &r)
		return r.Data.Movement.ID
	}
	dep := add(`{"kind":"deposit","currency":"SYP","amount":5000,"note":"من عندي"}`)
	add(`{"kind":"withdrawal","currency":"SYP","amount":2000}`)
	add(`{"kind":"withdrawal","currency":"USD","amount":0.25}`)
	if w := shamcashCall(f.srv.handleShamcashAdd, "POST", "/admin/shamcash/movements", "",
		`{"kind":"driver_transfer","currency":"SYP","amount":5}`, admin); w.Code != http.StatusBadRequest {
		t.Fatalf("تحويلُ سائقٍ يدويّ قُبل: %d", w.Code)
	}
	if w := shamcashCall(f.srv.handleShamcashAdd, "POST", "/admin/shamcash/movements", "",
		`{"kind":"deposit","currency":"SYP","amount":5,"occurred_at":"2099-01-01T10:00"}`, admin); w.Code != http.StatusBadRequest {
		t.Fatalf("حركةٌ بالمستقبل قُبلت: %d", w.Code)
	}

	syp, usd, owed, notArr, per := f.shamcashSummary(t, admin)
	// ١٠٠٠٠٠ + ١٠٠٠٠ + ٥٠٠٠ − ٢٠٠٠ (والـ٧٠٠٠ قبل البداية لا تُعدّ)
	if syp != 113000 || usd != 2000 {
		t.Fatalf("الرصيد ل.س=%d $=%d — المتوقَّع ١١٣٠٠٠ و٢٠٠٠ سنت", syp, usd)
	}
	// **وما للسائق يُعدّ ولو قبل البداية** — ١٠٠٠٠ + ٧٠٠٠.
	if owed != 17000 || per[d+":SYP"] != 17000 || notArr != 0 {
		t.Fatalf("ما للسائقين %d (%v) ما وصل %d", owed, per, notArr)
	}

	var mid string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM shamcash_movements WHERE order_id = $1`, id).Scan(&mid); err != nil {
		t.Fatal(err)
	}
	act := func(h http.HandlerFunc, mvID, body string, want int) {
		t.Helper()
		if w := shamcashCall(h, "POST", "/admin/shamcash/x", mvID, body, admin); w.Code != want {
			t.Fatalf("%d (المتوقَّع %d): %s", w.Code, want, w.Body.String())
		}
	}
	// «ما وصل» ⇒ يخرج من الرصيد ومن «ما للسائقين».
	act(f.srv.handleShamcashNotArrived, mid, "", http.StatusOK)
	syp, _, owed, notArr, _ = f.shamcashSummary(t, admin)
	if syp != 103000 || owed != 7000 || notArr != 10000 {
		t.Fatalf("بعد «ما وصل»: رصيد %d ما للسائقين %d ما وصل %d", syp, owed, notArr)
	}
	// ولا «سلّمت السائق» على ما لم يصل · والتراجعُ بلا سبب مرفوض.
	act(f.srv.handleShamcashCompensate, mid, "", http.StatusConflict)
	act(f.srv.handleShamcashArrived, mid, "", http.StatusBadRequest)
	act(f.srv.handleShamcashArrived, mid, `{"reason":"وصل متأخر"}`, http.StatusOK)
	act(f.srv.handleShamcashCompensate, mid, "", http.StatusOK)
	act(f.srv.handleShamcashCompensate, mid, "", http.StatusConflict)
	// ولا «ما وصل» على ما سلّمتُ بدله قبل التراجع.
	act(f.srv.handleShamcashNotArrived, mid, "", http.StatusConflict)
	syp, _, owed, _, per = f.shamcashSummary(t, admin)
	if syp != 113000 || owed != 7000 || per[d+":SYP"] != 7000 {
		t.Fatalf("بعد «سلّمت السائق»: رصيد %d ما للسائقين %d %v", syp, owed, per)
	}
	act(f.srv.handleShamcashUncompensate, mid, `{"reason":"غلط"}`, http.StatusOK)
	if _, _, owed, _, _ = f.shamcashSummary(t, admin); owed != 17000 {
		t.Fatalf("بعد التراجع: %d", owed)
	}
	// إلغاءُ الإيداع بسبب · وتحويلُ السائق لا يُلغى.
	act(f.srv.handleShamcashVoid, dep, "", http.StatusBadRequest)
	act(f.srv.handleShamcashVoid, dep, `{"reason":"انكتب مرتين"}`, http.StatusOK)
	act(f.srv.handleShamcashVoid, dep, `{"reason":"تاني"}`, http.StatusConflict)
	act(f.srv.handleShamcashVoid, mid, `{"reason":"x"}`, http.StatusConflict)
	if syp, _, _, _, _ = f.shamcashSummary(t, admin); syp != 108000 {
		t.Fatalf("بعد إلغاء الإيداع: %d", syp)
	}

	// القائمة بمرشّحاتها.
	list := func(qs string) int {
		w := shamcashCall(f.srv.handleShamcashList, "GET", "/admin/shamcash?"+qs, "", "", admin)
		if w.Code != http.StatusOK {
			t.Fatalf("القائمة %s: %d %s", qs, w.Code, w.Body.String())
		}
		var r struct {
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &r)
		return r.Data.Total
	}
	if n := list(""); n != 4 { // تحويلان وسحبان (والإيداعُ الملغى مخفيّ)
		t.Errorf("الكلّ %d", n)
	}
	if n := list("voided=1"); n != 5 {
		t.Errorf("مع الملغى %d", n)
	}
	if n := list("kind=withdrawal"); n != 2 {
		t.Errorf("السحوبات %d", n)
	}
	if n := list("kind=driver_transfer&driver=" + d + "&compensated=no"); n != 2 {
		t.Errorf("تحويلاتُ السائق غيرُ المسلَّمة %d", n)
	}
	if n := list("from=" + yesterday); n != 3 {
		t.Errorf("من أمس %d", n)
	}
	if w := shamcashCall(f.srv.handleShamcashList, "GET", "/admin/shamcash?kind=bogus", "", "", admin); w.Code != http.StatusBadRequest {
		t.Errorf("نوعٌ مجهول قُبل: %d", w.Code)
	}
}

// TestShamcash_Policy **القراءةُ للماليّة والكتابةُ بقدرة إدارتها** — والسياسةُ المركزيّةُ تحكم المسار.
func TestShamcash_Policy(t *testing.T) {
	for _, c := range []struct {
		method, path string
		want         authz.Capability
	}{
		{"GET", "/shamcash", authz.FinanceRead},
		{"GET", "/shamcash/summary", authz.FinanceRead},
		{"PUT", "/shamcash/opening", authz.FinanceManage},
		{"POST", "/shamcash/movements", authz.FinanceManage},
		{"POST", "/shamcash/{id}/compensate", authz.FinanceManage},
		{"POST", "/shamcash/{id}/uncompensate", authz.FinanceManage},
		{"POST", "/shamcash/{id}/not-arrived", authz.FinanceManage},
		{"POST", "/shamcash/{id}/arrived", authz.FinanceManage},
		{"POST", "/shamcash/{id}/void", authz.FinanceManage},
	} {
		got, ok := authz.LookupAdmin(c.method, c.path)
		if !ok || got != c.want {
			t.Errorf("%s %s = %v (%v) — المتوقَّع %v", c.method, c.path, got, ok, c.want)
		}
	}
}
