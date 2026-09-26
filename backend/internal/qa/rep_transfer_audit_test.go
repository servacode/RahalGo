package qa

// ══════════════════════════════════════════════════════════════════════
// **REP-TRANSFER — نقلُ متجرٍ بين مندوبَين يلزمه سبب، ويُكتب له حدثُ تدقيق**
// ══════════════════════════════════════════════════════════════════════
//
// **`catalog.UpdateMerchant` عبر `PATCH /api/v1/admin/merchants/{id}`**: حين
// يتبدّل `sales_rep_user_id` إلى مندوبٍ آخر، **يُشترط سببٌ غيرُ فارغ**
// (`ErrTransferReasonRequired`)، **ويُكتب صفٌّ في `audit_log` فعلُه
// `admin.merchant_rep_transfer`** وتفاصيلُه تحمل from_rep · to_rep · reason ·
// merchant_id. **وتبديلُ حقلٍ غيرِ المندوب لا يُشترط له سببٌ ولا يُكتب له نقل.**

import (
	"testing"
)

// transferAudits **صفوفُ تدقيق النقل لهذا المتجر** — عدداً وأحدثَ تفاصيلها.
func transferAudits(t *testing.T, h *Harness, merchantID string) (n int, from, to, reason, mid string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*),
		       COALESCE(max(details->>'from_rep'), ''),
		       COALESCE(max(details->>'to_rep'), ''),
		       COALESCE(max(details->>'reason'), ''),
		       COALESCE(max(details->>'merchant_id'), '')
		FROM audit_log
		WHERE action = 'admin.merchant_rep_transfer' AND entity = 'merchant' AND entity_id = $1`,
		merchantID).Scan(&n, &from, &to, &reason, &mid); err != nil {
		t.Fatalf("read transfer audits: %v", err)
	}
	return n, from, to, reason, mid
}

func TestRepTransfer_RequiresReasonAndWritesAudit(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")

	repA := f.RepAccount()
	repB := f.RepAccount()
	m := f.Merchant(OwnedByRep(repA.ID)) // منسوبٌ إلى repA اليوم
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`DELETE FROM audit_log WHERE entity = 'merchant' AND entity_id = $1`, m.ID)
	})

	// ── أ · نقلٌ بلا سبب ⇒ يُرفض ولا يُكتب تدقيقُ نقل ────────────────
	noReason := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
		map[string]any{"sales_rep_code": repB.InviteCode})
	t.Logf("transfer-without-reason: %d %s", noReason.Code, noReason.Err())
	if noReason.Code != 400 {
		t.Errorf("transfer without reason returned %d — expected 400", noReason.Code)
	}
	if noReason.Err() != "transfer_reason_required" {
		t.Errorf("error code %q — expected transfer_reason_required", noReason.Err())
	}
	if n, _, _, _, _ := transferAudits(t, h, m.ID); n != 0 {
		t.Errorf("a rejected transfer wrote %d audit rows — expected 0", n)
	}
	// **ولم يتبدّل المندوب** — الرفضُ قبل الكتابة.
	if got := merchantRep(t, h, m.ID); got != repA.ID {
		t.Errorf("rejected transfer still changed the rep to %q", got)
	}

	// ── ب · نقلٌ بسببٍ ⇒ يمرّ ويُكتب صفٌّ واحدٌ بتفاصيلَ صحيحة ────────
	const reason = "repA left the field, repB takes over"
	ok := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
		map[string]any{"sales_rep_code": repB.InviteCode, "transfer_reason": reason})
	t.Logf("transfer-with-reason: %d %s", ok.Code, ok.Err())
	if ok.Code >= 400 {
		t.Fatalf("transfer with reason rejected: %s", ok)
	}
	if got := merchantRep(t, h, m.ID); got != repB.ID {
		t.Errorf("after transfer the rep is %q — expected repB %q", got, repB.ID)
	}
	n, from, to, gotReason, mid := transferAudits(t, h, m.ID)
	t.Logf("audit: n=%d from=%s to=%s reason=%q merchant=%s", n, from, to, gotReason, mid)
	if n != 1 {
		t.Errorf("transfer wrote %d audit rows — expected exactly 1", n)
	}
	if from != repA.ID {
		t.Errorf("audit from_rep=%q — expected repA %q", from, repA.ID)
	}
	if to != repB.ID {
		t.Errorf("audit to_rep=%q — expected repB %q", to, repB.ID)
	}
	if gotReason != reason {
		t.Errorf("audit reason=%q — expected %q", gotReason, reason)
	}
	if mid != m.ID {
		t.Errorf("audit merchant_id=%q — expected %q", mid, m.ID)
	}
}

// TestRepTransfer_NonRepFieldNeedsNoReason **تبديلُ الاسمِ وحدَه لا نقل.**
func TestRepTransfer_NonRepFieldNeedsNoReason(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")

	repA := f.RepAccount()
	m := f.Merchant(OwnedByRep(repA.ID))
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`DELETE FROM audit_log WHERE entity = 'merchant' AND entity_id = $1`, m.ID)
	})

	// **تبديلُ اسمٍ فقط** — بلا كود مندوب، فلا نقل، فلا سبب مطلوب.
	res := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
		map[string]any{"name": "Renamed QA Store"})
	t.Logf("rename-only: %d %s", res.Code, res.Err())
	if res.Code >= 400 {
		t.Fatalf("name change without rep change rejected: %s", res)
	}
	if n, _, _, _, _ := transferAudits(t, h, m.ID); n != 0 {
		t.Errorf("a name-only edit wrote %d transfer audit rows — expected 0", n)
	}
	// **والمندوبُ لم يتبدّل.**
	if got := merchantRep(t, h, m.ID); got != repA.ID {
		t.Errorf("name-only edit changed the rep to %q", got)
	}
}

// merchantRep **مندوبُ المتجر الحاليّ** — أو فراغٌ إن لا مندوب.
func merchantRep(t *testing.T, h *Harness, merchantID string) string {
	t.Helper()
	var rep *string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT sales_rep_user_id::text FROM merchants WHERE id = $1::uuid`,
		merchantID).Scan(&rep); err != nil {
		t.Fatalf("read merchant rep: %v", err)
	}
	if rep == nil {
		return ""
	}
	return *rep
}
