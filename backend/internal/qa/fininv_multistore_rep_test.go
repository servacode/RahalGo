package qa

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ عمولة المندوبين في الطلب بعدّة محطّات** — `FI-05.e` · `FI-09.a` · `FI-09.b` (الخطّة §7.4)
// ══════════════════════════════════════════════════════════════════════
//
// كأساسات `fininv_multistore_test.go`: صفوفٌ مزروعةٌ بيد — **ولكلّ قاعدةٍ حالٌ صحيحةٌ لا تُخرَق
// وخرقٌ حقيقيٌّ يبقى يُمسَك.**

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

func (f *msFix) multiStop(order string) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `UPDATE orders SET stops = 2 WHERE id = $1::uuid`, order); err != nil {
		f.t.Fatalf("محطّتان: %v", err)
	}
}

func (f *msFix) part(order, merchant string, seq int, rep string, amount int64) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO order_parts (order_id, merchant_id, seq, status, rep_id, rep_commission)
		VALUES ($1::uuid, $2::uuid, $3, 'picked_up', NULLIF($4, '')::uuid, $5)`,
		order, merchant, seq, rep, amount); err != nil {
		f.t.Fatalf("قسم: %v", err)
	}
}

func (f *msFix) held(order, rep string, amount int64, status string) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO rep_held_commissions (rep_id, order_id, amount, status) VALUES ($1::uuid, $2::uuid, $3, $4)`,
		rep, order, amount, status); err != nil {
		f.t.Fatalf("محجوز: %v", err)
	}
}

func (f *msFix) repShareDebt(order, merchant string, amount int64) {
	f.t.Helper()
	if _, err := obligations.CreateRef(f.ctx, f.pool, obligations.PartyMerchant, merchant, amount,
		obligations.CauseRefundRepShare, order, order+":"+merchant, nil); err != nil {
		f.t.Fatalf("دينُ نصيب المندوب: %v", err)
	}
}

// TestFI_RepCommissionPerStoreGuards **FI-05.e لكلّ (طلب، مندوب) · FI-09.a بمجموع الطلب · FI-09.b التفصيل.**
func TestFI_RepCommissionPerStoreGuards(t *testing.T) {
	f := newMSFix(t, "rahalgo_ms3_fi09")
	cust, oa, ob := f.user(), f.user(), f.user()
	r1, r2 := f.user(), f.user()
	a, b := f.store(oa, "wallet"), f.store(ob, "wallet")

	// ── FI-05.e ── مندوبان على طلبٍ واحد ليسا تكراراً، ومندوبٌ مرّتين تكرار.
	two, _ := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.tx(r1, 10, "commission", two)
	f.tx(r2, 40, "commission", two)
	f.wantClean("FI-05.e", two, "عمولتان لمندوبين على طلبٍ بمحطّتين")
	dup, _ := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.tx(r1, 10, "commission", dup)
	f.tx(r1, 10, "commission", dup)
	f.wantFlag("FI-05.e", dup, "عمولتان لمندوبٍ واحد على طلبٍ واحد")

	// ── FI-09.b ── التفصيلُ = الموجبُ + المحجوز (قائماً أو ساقطاً)، والمصروفُ بعد الحجز لا يُعدّ مرّتين.
	okID, okNum := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.multiStop(okID)
	f.part(okID, a, 1, r1, 10)
	f.part(okID, b, 2, r2, 40)
	f.tx(r1, 10, "commission", okID)
	f.tx(r1, -4, "commission", okID) // اقتطاعُ دَين — ليس نقصاً من التفصيل
	f.held(okID, r2, 40, "released")
	f.tx(r2, 40, "commission", okID)
	f.wantClean("FI-09.b", okNum, "تفصيلٌ يطابق الدفتر (اقتطاعُ دَين ومحجوزٌ صُرف)")

	heldID, heldNum := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.multiStop(heldID)
	f.part(heldID, a, 1, r1, 10)
	f.part(heldID, b, 2, r2, 40)
	f.tx(r1, 10, "commission", heldID)
	f.held(heldID, r2, 40, "forfeited")
	f.wantClean("FI-09.b", heldNum, "محجوزٌ سقط يبقى في التفصيل")

	driftID, driftNum := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.multiStop(driftID)
	f.part(driftID, a, 1, r1, 10)
	f.part(driftID, b, 2, r2, 40)
	f.tx(r1, 10, "commission", driftID)
	f.tx(r2, 30, "commission", driftID)
	f.wantFlag("FI-09.b", driftNum, "تفصيلٌ ٤٠ والدفترُ ٣٠")

	ghostID, ghostNum := f.order(cust, a, "standard", "delivered", 0, 3000, 500)
	f.multiStop(ghostID)
	f.part(ghostID, a, 1, r1, 10)
	f.part(ghostID, b, 2, "", 0)
	f.tx(r1, 10, "commission", ghostID)
	f.tx(r2, 40, "commission", ghostID)
	f.wantFlag("FI-09.b", ghostNum, "عمولةٌ لمندوبٍ بلا تفصيل")

	// ── FI-09.a ── دينان على متجرين مجموعُهما ما بقي من العمولة.
	refID, refNum := f.order(cust, a, "standard", "refunded", 0, 3000, 500)
	f.tx(r1, 10, "commission", refID)
	f.tx(r2, 40, "commission", refID)
	f.repShareDebt(refID, a, 10)
	f.repShareDebt(refID, b, 40)
	f.wantClean("FI-09.a", refNum, "نصيبا مندوبين على متجريهما")

	badID, badNum := f.order(cust, a, "standard", "refunded", 0, 3000, 500)
	f.tx(r1, 10, "commission", badID)
	f.tx(r2, 40, "commission", badID)
	f.repShareDebt(badID, a, 10)
	f.repShareDebt(badID, b, 50)
	f.wantFlag("FI-09.a", badNum, "ديونٌ ٦٠ والعمولةُ ٥٠")
}
