package qa

// ══════════════════════════════════════════════════════════════════════
// **REP-IDOR — مندوبٌ لا يمسّ قائمةَ متجرٍ ليس عميلَه، عبر الموجّه الحقيقيّ**
// ══════════════════════════════════════════════════════════════════════
//
// **وحدةُ الحارس مُختبَرةٌ في `server/rep_menu_test.go`** — **وهذا يقيسه من
// الباب**: نداءٌ عبر `Router()` الكامل بمساراته الحقيقيّة، بمعرّفٍ في المسار
// يُخمَّن أو يُقرأ من ردٍّ سابق. **والقفلُ في المحرّك أو لا قفل.**
//
//	POST   /api/v1/rep/stores/{id}/menu/items   ← الحارسُ يقرأ `id` (المتجر)
//	PATCH  /api/v1/rep/menu/items/{itemID}      ← الحارسُ يقرأ `itemID` (الصنف)
//	DELETE /api/v1/rep/menu/items/{itemID}
//
// **والأثرُ هو الحَكَم**: بعد كلّ ردٍّ مرفوض، الصنفُ لم يتبدّل ولم يُحذَف.

import (
	"testing"
)

func TestRepIDOR_CannotWriteAnotherRepsMenu(t *testing.T) {
	h := New(t)
	f := h.Factory()

	// **مندوبان بجلستين** — repA هو المهاجم، repB صاحبُ المتجر والصنف.
	repA := h.NewUser("sales")
	repB := h.NewUser("sales")

	// **متجرُ repB وصنفٌ تحته** — من المصنع بنسبةٍ صريحةٍ للمندوب.
	merchantB := f.Merchant(OwnedByRep(repB.ID))
	itemB := h.NewItemFor(merchantB, 1500)

	// **صورةُ الصنف قبل أيّ محاولة** — يُقارَن بها الأثر.
	nameBefore, priceBefore, existed := itemFacts(t, h, itemB.ID)
	if !existed {
		t.Fatalf("setup: item %s was not created", itemB.ID)
	}

	// ── أ · إنشاءُ صنفٍ في متجر repB (الحارسُ يقرأ معرّفَ المتجر) ──────
	create := h.POST("/api/v1/rep/stores/"+merchantB.ID+"/menu/items", repA.Token,
		map[string]any{"name": "IDOR injected item", "price": 999, "platform_section_id": itemB.SectionID})
	t.Logf("create-as-other: %d %s", create.Code, create.Err())
	if create.Code != 403 && create.Code != 404 {
		t.Errorf("create in another rep's store returned %d — expected 403/404", create.Code)
	}

	// ── ب · تعديلُ صنفِ repB (الحارسُ يقرأ معرّفَ الصنف) ─────────────
	patch := h.PATCH("/api/v1/rep/menu/items/"+itemB.ID, repA.Token,
		map[string]any{"price": 111, "available": false, "name": "IDOR renamed"})
	t.Logf("patch-as-other: %d %s", patch.Code, patch.Err())
	if patch.Code != 403 && patch.Code != 404 {
		t.Errorf("patch of another rep's item returned %d — expected 403/404", patch.Code)
	}

	// ── ج · حذفُ صنفِ repB ──────────────────────────────────────────
	del := h.DEL("/api/v1/rep/menu/items/"+itemB.ID, repA.Token)
	t.Logf("delete-as-other: %d %s", del.Code, del.Err())
	if del.Code != 403 && del.Code != 404 {
		t.Errorf("delete of another rep's item returned %d — expected 403/404", del.Code)
	}

	// ── والأثرُ: الصنفُ لم يتبدّل ولم يُحذَف ─────────────────────────
	nameAfter, priceAfter, stillExists := itemFacts(t, h, itemB.ID)
	if !stillExists {
		t.Errorf("IDOR: another rep's item was DELETED")
	}
	if nameAfter != nameBefore {
		t.Errorf("IDOR: item name changed by another rep: %q → %q", nameBefore, nameAfter)
	}
	if priceAfter != priceBefore {
		t.Errorf("IDOR: item price changed by another rep: %d → %d", priceBefore, priceAfter)
	}

	// ── ضابطٌ إيجابيّ: صاحبُ المتجر (repB) يعدّل صنفَه ولا يُردّ ──────
	own := h.PATCH("/api/v1/rep/menu/items/"+itemB.ID, repB.Token,
		map[string]any{"available": true})
	t.Logf("patch-as-owner (repB): %d %s", own.Code, own.Err())
	if own.Code == 403 || own.Code == 404 {
		t.Errorf("owning rep was blocked from its own item (%d) — a lock that rejects its key holder is a bug", own.Code)
	}
}

// itemFacts **اسمُ الصنف وسعرُه ووجودُه** — يُقرأ من القاعدة لا يُفترَض.
func itemFacts(t *testing.T, h *Harness, itemID string) (name string, price int64, exists bool) {
	t.Helper()
	err := h.Pool.QueryRow(ctxBG(),
		`SELECT name, price FROM menu_items WHERE id = $1::uuid`, itemID).Scan(&name, &price)
	if err != nil {
		return "", 0, false
	}
	return name, price, true
}
