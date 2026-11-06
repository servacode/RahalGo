package qa

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ المال للطلب من عدّة متاجر** — `FI-05.c` · `FI-05.g` · `FI-06.b` (الخطّة §7.4، Q2)
// ══════════════════════════════════════════════════════════════════════
//
// كلُّ حالٍ هنا مزروعةٌ بصفوفٍ مباشرةٍ في قاعدةٍ خاصّة (كبقيّة أساسات fininv) —
// **لأنّ مسارَ الاستلام بعدّة محطّات يتبدّل الآن**، والحارسُ يُسأل عن شكل الدفتر لا عن المسار.
//
// ولكلّ قاعدةٍ وجهان: **حالٌ صحيحةٌ كانت تُخرَق بالقاعدة القديمة** ولا تُخرَق اليوم،
// **وخرقٌ حقيقيٌّ يبقى يُمسَك.**

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// msFix أساسٌ مزروعٌ بيد — مستخدمون ومتاجرُ وطلباتٌ وقيود.
type msFix struct {
	t     *testing.T
	pool  *pgxpool.Pool
	w     *wallet.Service
	ctx   context.Context
	cat   string
	hold  string
	phone int
}

func newMSFix(t *testing.T, db string) *msFix {
	t.Helper()
	pool := fixtureDB(t, db)
	f := &msFix{t: t, pool: pool, w: wallet.NewService(pool), ctx: context.Background(), phone: 900700000}
	if err := pool.QueryRow(f.ctx,
		`INSERT INTO categories (name) VALUES ('تصنيفُ الأساس') RETURNING id::text`).Scan(&f.cat); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	hid, err := wallet.EnsureCashHolding(f.ctx, pool)
	if err != nil {
		t.Fatalf("الاحتباس: %v", err)
	}
	f.hold = hid
	return f
}

func (f *msFix) user() string {
	f.t.Helper()
	f.phone++
	var id string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (phone, full_name, status) VALUES ($1, 'مستخدمُ الأساس', 'active')
		RETURNING id::text`, fmt.Sprintf("0%d", f.phone)).Scan(&id); err != nil {
		f.t.Fatalf("مستخدم: %v", err)
	}
	return id
}

func (f *msFix) store(owner, method string) string {
	f.t.Helper()
	var id string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO merchants (name, owner_user_id, category_id, status, settlement_method)
		VALUES ($1, $2::uuid, $3::uuid, 'active', $4) RETURNING id::text`,
		fmt.Sprintf("متجر %d", f.phone), owner, f.cat, method).Scan(&id); err != nil {
		f.t.Fatalf("متجر: %v", err)
	}
	f.phone++
	return id
}

// order طلبٌ بحاله وأرقامه — ويُرجع معرّفَه ورقمَه.
func (f *msFix) order(customer, merchant, kind, status string, walletPaid, cashDue, pc int64) (string, string) {
	f.t.Helper()
	pay := "wallet"
	if walletPaid == 0 {
		pay = "cash"
	}
	var id, number string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO orders (customer_id, merchant_id, kind, status, address_text, dropoff, payment_method,
		                    subtotal, delivery_fee, total, wallet_paid, cash_due, platform_commission)
		VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, $4, 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0,35.95),4326)::geography, $5,
		        $6, 0, $6, $7, $8, $9)
		RETURNING id::text, number::text`,
		customer, merchant, kind, status, pay, walletPaid+cashDue, walletPaid, cashDue, pc).
		Scan(&id, &number); err != nil {
		f.t.Fatalf("طلب: %v", err)
	}
	return id, number
}

func (f *msFix) item(order, merchant string, cost int64) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO order_items (order_id, merchant_id, name, unit_price, qty, merchant_price)
		VALUES ($1::uuid, $2::uuid, 'صنف', $3, 1, $3)`, order, merchant, cost); err != nil {
		f.t.Fatalf("بند: %v", err)
	}
}

func (f *msFix) tx(user string, amount int64, kind, ref string) int64 {
	f.t.Helper()
	_, id, err := f.w.ApplyTxID(f.ctx, f.pool, user, amount, kind, ref, "أساس", nil)
	if err != nil {
		f.t.Fatalf("قيد %s %d: %v", kind, amount, err)
	}
	return id
}

// walletSettle صفُّ تسويةٍ محفظيٌّ مع قيده.
func (f *msFix) walletSettle(order, merchant, owner string, due int64) string {
	f.t.Helper()
	tx := f.tx(owner, due, "merchant_earning", order)
	var id string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO merchant_settlements (order_id, merchant_id, owner_user_id, method, amount, state, earning_tx_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'wallet', $4, 'wallet_credited', $5) RETURNING id::text`,
		order, merchant, owner, due, tx).Scan(&id); err != nil {
		f.t.Fatalf("تسويةٌ محفظيّة: %v", err)
	}
	return id
}

// cashSettle صفُّ تسويةٍ نقديٌّ دفعه السائقُ عند الاستلام.
func (f *msFix) cashSettle(order, merchant, owner string, due int64) string {
	f.t.Helper()
	acc := f.tx(f.hold, due, "merchant_cash_accrued", order)
	paid := f.tx(f.hold, -due, "merchant_cash_paid", order)
	var id string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO merchant_settlements (order_id, merchant_id, owner_user_id, method, amount, state,
		                                  accrued_tx_id, paid_tx_id, paid_via)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'cash', $4, 'cash_paid', $5, $6, 'driver') RETURNING id::text`,
		order, merchant, owner, due, acc, paid).Scan(&id); err != nil {
		f.t.Fatalf("تسويةٌ نقديّة: %v", err)
	}
	return id
}

// flagged هل أمسك الفحصُ صفّاً مفتاحُه الأوّلُ key (مرجعُ الطلب أو رقمُه)؟
func (f *msFix) flagged(id, key string) bool {
	f.t.Helper()
	vs, err := fininv.Run(f.ctx, f.pool, id)
	if err != nil {
		f.t.Fatalf("تشغيلُ %s: %v", id, err)
	}
	for _, v := range vs {
		for _, r := range v.Rows {
			if len(r) > 0 && fmt.Sprint(r[0]) == key {
				return true
			}
		}
	}
	return false
}

func (f *msFix) wantClean(id, key, what string) {
	f.t.Helper()
	if f.flagged(id, key) {
		f.t.Errorf("%s خرقٌ باطل: %s", id, what)
	}
}

func (f *msFix) wantFlag(id, key, what string) {
	f.t.Helper()
	if !f.flagged(id, key) {
		f.t.Errorf("%s لم يُمسك خرقاً حقيقيّاً: %s", id, what)
	}
}

// ── FI-05.c: متجران لمالكٍ واحدٍ في طلبٍ واحد ليسا تكراراً ─────────────────
func TestFI_SameOwnerTwoStoresNotDuplicate(t *testing.T) {
	f := newMSFix(t, "rahalgo_ms3_fi05c")
	cust, owner := f.user(), f.user()
	a, b := f.store(owner, "wallet"), f.store(owner, "wallet")

	// **الحالُ الصحيحة**: صفّا تسويةٍ محفظيّان لمالكٍ واحد — قيدان موجبان.
	ok, _ := f.order(cust, a, "standard", "delivered", 0, 1500, 150)
	f.item(ok, a, 1000)
	f.item(ok, b, 500)
	f.walletSettle(ok, a, owner, 900)
	f.walletSettle(ok, b, owner, 450)
	f.wantClean("FI-05.c", ok, "متجران لمالكٍ واحد، لكلٍّ صفُّه وقيدُه")

	// **خرقٌ حقيقيّ**: قيدٌ ثالثٌ موجبٌ بلا صفّ — المالكُ قُبض له مرّةً زائدة.
	extra, _ := f.order(cust, a, "standard", "delivered", 0, 1500, 150)
	f.item(extra, a, 1000)
	f.item(extra, b, 500)
	f.walletSettle(extra, a, owner, 900)
	f.walletSettle(extra, b, owner, 450)
	f.tx(owner, 900, "merchant_earning", extra)
	f.wantFlag("FI-05.c", extra, "قيدٌ ثالثٌ لمتجرين")

	// **وخرقُ المتجر الواحد يبقى**: صفٌّ واحدٌ وقيدان.
	one, _ := f.order(cust, a, "standard", "delivered", 0, 1000, 100)
	f.item(one, a, 1000)
	f.walletSettle(one, a, owner, 900)
	f.tx(owner, 900, "merchant_earning", one)
	f.wantFlag("FI-05.c", one, "متجرٌ واحدٌ قُيّد مرّتين")

	// **وطلبٌ تراثيٌّ بلا صفوفٍ** يبقى على قيدٍ واحد.
	legacy, _ := f.order(cust, a, "standard", "delivered", 0, 1000, 100)
	f.tx(owner, 900, "merchant_earning", legacy)
	f.wantClean("FI-05.c", legacy, "طلبٌ تراثيٌّ بقيدٍ واحد")
	f.tx(owner, 900, "merchant_earning", legacy)
	f.wantFlag("FI-05.c", legacy, "طلبٌ تراثيٌّ بقيدين")
}

// ── FI-05.g: استردادٌ جزئيٌّ ثمّ نهائيٌّ ليس تكراراً ─────────────────────────
func TestFI_PartialRefundsNotDuplicate(t *testing.T) {
	f := newMSFix(t, "rahalgo_ms3_fi05g")
	cust := f.user()
	owner := f.user()
	a := f.store(owner, "wallet")
	f.tx(cust, 100000, "topup", "")

	// **محفظةٌ ١٠٠٠: صنفٌ غيرُ متوفّر ردّ ٢٠٠، ثمّ أُلغي فرُدّ الباقي ٨٠٠.**
	ok, _ := f.order(cust, a, "standard", "cancelled", 800, 0, 0)
	f.tx(cust, -1000, "order_payment", ok)
	f.tx(cust, 200, "refund", ok)
	f.tx(cust, 800, "refund", ok)
	f.wantClean("FI-05.g", ok, "استردادٌ جزئيٌّ ثمّ نهائيّ")

	// **محفظةٌ ٣٠٠ ونقدٌ ٧٠٠: تعديلٌ ردّ ١٠٠، ثمّ استرجاعٌ بعد التسليم بكامل ما بقي (٢٠٠+٧٠٠).**
	ref, _ := f.order(cust, a, "standard", "refunded", 200, 700, 0)
	f.tx(cust, -300, "order_payment", ref)
	f.tx(cust, 100, "refund", ref)
	f.tx(cust, 900, "refund", ref)
	f.wantClean("FI-05.g", ref, "جزئيٌّ ثمّ استرجاعٌ بعد التسليم مع النقد")

	// **خرقٌ حقيقيّ**: إلغاءٌ رُدّ مرّتين.
	dbl, _ := f.order(cust, a, "standard", "cancelled", 1000, 0, 0)
	f.tx(cust, -1000, "order_payment", dbl)
	f.tx(cust, 1000, "refund", dbl)
	f.tx(cust, 1000, "refund", dbl)
	f.wantFlag("FI-05.g", dbl, "إلغاءٌ رُدّ مرّتين")

	// **ونقدٌ مُسترجَعٌ بعد التسليم رُدّ مرّتين** — بلا دفعِ محفظة.
	cash, _ := f.order(cust, a, "standard", "refunded", 0, 700, 0)
	f.tx(cust, 700, "refund", cash)
	f.tx(cust, 700, "refund", cash)
	f.wantFlag("FI-05.g", cash, "نقدٌ رُدّ مرّتين")

	// **ونقدُ طلبٍ لم يُسترجَع لا يُحسب** — ملغى نقداً ورُدّ له ما لم يدفعه.
	never, _ := f.order(cust, a, "standard", "cancelled", 0, 700, 0)
	f.tx(cust, 700, "refund", never)
	f.wantFlag("FI-05.g", never, "ردٌّ لنقدٍ لم يُقبض")

	// **وغيرُ «العاديّ» يبقى على عدّ القيود**: طلبٌ خاصٌّ باستردادين.
	var custom string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO orders (customer_id, kind, status, address_text, dropoff, payment_method,
		                    subtotal, delivery_fee, total, wallet_paid,
		                    quote_version, quote_confirmed_at, quote_confirmed_total, quote_confirmed_version)
		VALUES ($1::uuid, 'custom', 'cancelled', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0,35.95),4326)::geography, 'wallet',
		        1000, 0, 1000, 1000, 1, now(), 1000, 1)
		RETURNING id::text`, cust).Scan(&custom); err != nil {
		t.Fatalf("طلبٌ خاصّ: %v", err)
	}
	f.tx(cust, -1000, "order_payment", custom)
	f.tx(cust, 200, "refund", custom)
	f.tx(cust, 800, "refund", custom)
	f.wantFlag("FI-05.g", custom, "طلبٌ خاصٌّ باستردادين (قاعدةُ العدّ باقية)")
}

// ── FI-06.b: متجرٌ بالمحفظة ومتجرٌ بالنقد في طلبٍ واحد ────────────────────
func TestFI_MixedWalletCashCommission(t *testing.T) {
	f := newMSFix(t, "rahalgo_ms3_fi06b")
	cust := f.user()
	ownA, ownB := f.user(), f.user()
	a, b := f.store(ownA, "wallet"), f.store(ownB, "cash")

	// **عمولةٌ ١٠٪ لكلٍّ**: أ كلفتُه ١٠٠٠ مستحقُّه ٩٠٠ محفظةً، ب كلفتُه ٥٠٠ مستحقُّه ٤٥٠ نقداً.
	mixed, mixedNo := f.order(cust, a, "standard", "delivered", 0, 1500, 150)
	f.item(mixed, a, 1000)
	f.item(mixed, b, 500)
	f.walletSettle(mixed, a, ownA, 900)
	f.cashSettle(mixed, b, ownB, 450)
	f.wantClean("FI-06.b", mixedNo, "محفظةٌ ونقدٌ في طلبٍ واحد")

	// **واقتطاعُ دَينٍ من المستحقّين لا يمسّ العمولة** — قيدان سالبان بعد القيد الأصليّ.
	off, offNo := f.order(cust, a, "standard", "delivered", 0, 1500, 150)
	f.item(off, a, 1000)
	f.item(off, b, 500)
	f.walletSettle(off, a, ownA, 900)
	f.tx(ownA, -50, "merchant_earning", off)
	// نقداً: احتباسٌ ٤٥٠، يُقتطع منه ١٠٠ دَيناً، ويدفع السائقُ الباقي ٣٥٠.
	acc := f.tx(f.hold, 450, "merchant_cash_accrued", off)
	rev := f.tx(f.hold, -100, "merchant_cash_accrued", off)
	paid := f.tx(f.hold, -350, "merchant_cash_paid", off)
	var cs string
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO merchant_settlements (order_id, merchant_id, owner_user_id, method, amount, state,
		                                  reversed_amount, accrued_tx_id, paid_tx_id, paid_via)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'cash', 450, 'cash_paid', 100, $4, $5, 'driver')
		RETURNING id::text`, off, b, ownB, acc, paid).Scan(&cs); err != nil {
		t.Fatalf("تسويةٌ نقديّةٌ مقتطَعة: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO merchant_settlement_reversals (settlement_id, tx_id, amount, cause, event_ref)
		VALUES ($1::uuid, $2, 100, 'debt_offset', 'debt:1')`, cs, rev); err != nil {
		t.Fatalf("سطرُ الاقتطاع: %v", err)
	}
	f.wantClean("FI-06.b", offNo, "اقتطاعُ دَينٍ محفظةً ونقداً")

	// **ومحوُ العمولة**: متجرٌ نقديٌّ دفعه السائق، والعمولةُ ١٠٠ صارت صفراً في العمود.
	wv, wvNo := f.order(cust, b, "standard", "delivered", 0, 1000, 0)
	f.item(wv, b, 1000)
	ws := f.cashSettle(wv, b, ownB, 900)
	wacc := f.tx(f.hold, 100, "merchant_cash_accrued", wv)
	wpaid := f.tx(f.hold, -100, "merchant_cash_paid", wv)
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE merchant_settlements SET commission_waived = 100, waived_accrued_tx_id = $2, waived_paid_tx_id = $3
		 WHERE id = $1::uuid`, ws, wacc, wpaid); err != nil {
		t.Fatalf("محوُ العمولة: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx,
		`UPDATE orders SET commission_waived = 100, commission_waived_at = now() WHERE id = $1::uuid`, wv); err != nil {
		t.Fatalf("محوُ العمولة: %v", err)
	}
	f.wantClean("FI-06.b", wvNo, "عمولةٌ ممحوّة")

	// **وتصحيحُ متجرٍ نقديٍّ بتعديل الطلب**: كلفةٌ ١٠٠٠ ← ١٢٠٠، مستحقٌّ ٩٠٠ + ١٨٠، عمولةٌ ١٢٠.
	ed, edNo := f.order(cust, b, "standard", "delivered", 0, 1200, 120)
	f.item(ed, b, 1200)
	es := f.cashSettle(ed, b, ownB, 900)
	eacc := f.tx(f.hold, 180, "merchant_cash_accrued", ed)
	epaid := f.tx(f.hold, -180, "merchant_cash_paid", ed)
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO merchant_settlement_adjustments (settlement_id, order_id, delta, accrued_tx_id, paid_tx_id,
		                                             paid_via, reason)
		VALUES ($1::uuid, $2::uuid, 180, $3, $4, 'driver', 'تعديل')`, es, ed, eacc, epaid); err != nil {
		t.Fatalf("سطرُ التصحيح: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx,
		`UPDATE merchant_settlements SET adjusted_amount = 180 WHERE id = $1::uuid`, es); err != nil {
		t.Fatalf("مجموعُ التصحيح: %v", err)
	}
	f.wantClean("FI-06.b", edNo, "تصحيحُ متجرٍ نقديٍّ بتعديل الطلب")

	// **خروقٌ حقيقيّة**: عمولةٌ في العمود لا تطابق الدفتر.
	bad, badNo := f.order(cust, a, "standard", "delivered", 0, 1500, 600)
	f.item(bad, a, 1000)
	f.item(bad, b, 500)
	f.walletSettle(bad, a, ownA, 900)
	f.cashSettle(bad, b, ownB, 450)
	f.wantFlag("FI-06.b", badNo, "عمولةٌ ٦٠٠ والمقيَّدُ يقول ١٥٠")

	cashBad, cashBadNo := f.order(cust, b, "standard", "delivered", 0, 1000, 300)
	f.item(cashBad, b, 1000)
	f.cashSettle(cashBad, b, ownB, 900)
	f.wantFlag("FI-06.b", cashBadNo, "متجرٌ نقديٌّ وحدَه وعمولةٌ ٣٠٠ والمقيَّدُ ١٠٠")

	wvBad, wvBadNo := f.order(cust, b, "standard", "delivered", 0, 1000, 0)
	f.item(wvBad, b, 1000)
	f.cashSettle(wvBad, b, ownB, 900)
	if _, err := f.pool.Exec(f.ctx,
		`UPDATE orders SET commission_waived = 250, commission_waived_at = now() WHERE id = $1::uuid`, wvBad); err != nil {
		t.Fatalf("محوٌ فاسد: %v", err)
	}
	f.wantFlag("FI-06.b", wvBadNo, "محوُ ٢٥٠ وعمولةُ المتجر ١٠٠")

	// **وطلبٌ تراثيٌّ بلا صفوفٍ يبقى على القاعدة القديمة.**
	lg, lgNo := f.order(cust, a, "standard", "delivered", 0, 1000, 100)
	f.item(lg, a, 1000)
	f.tx(ownA, 900, "merchant_earning", lg)
	f.wantClean("FI-06.b", lgNo, "تراثيٌّ صحيح")
	lgBad, lgBadNo := f.order(cust, a, "standard", "delivered", 0, 1000, 50)
	f.item(lgBad, a, 1000)
	f.tx(ownA, 900, "merchant_earning", lgBad)
	f.wantFlag("FI-06.b", lgBadNo, "تراثيٌّ بعمولةٍ خاطئة")
}
