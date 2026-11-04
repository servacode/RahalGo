package qa

// أقسامُ السوق التي يظهر فيها متجرُ العميل — يضبطها مندوبُه (قرارُ المالك ٢٠٢٦-١١-٠٤).

import "testing"

func newPlatformSection(t *testing.T, h *Harness, name string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO platform_sections (name, active) VALUES ($1, true) RETURNING id::text`,
		name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func sectionIDs(r Res) map[string]bool {
	out := map[string]bool{}
	list, _ := r.JSON()["sections"].([]any)
	for _, x := range list {
		p, _ := x.(map[string]any)
		out[p["id"].(string)] = true
	}
	return out
}

func TestRepStoreSections_SetOnlyForOwnClient(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := h.NewUser("sales")
	other := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	a := newPlatformSection(t, h, "قسم تجربة أ")
	b := newPlatformSection(t, h, "قسم تجربة ب")
	path := "/api/v1/rep/stores/" + m.ID + "/sections"

	// المندوب يقرأ قائمة الأقسام المتاحة للاختيار
	if r := h.GET("/api/v1/rep/platform-sections", rep.Token); r.Code != 200 || !sectionIDs(r)[a] {
		t.Fatalf("أقسامُ السوق للمندوب: %s", r)
	}

	// يضبط قسمين فتظهر عند المتجر وعند المندوب
	if r := h.Call("PUT", path, rep.Token, map[string]any{"sections": []string{a, b}}, nil); r.Code != 200 {
		t.Fatalf("الحفظ: %s", r)
	}
	got := sectionIDs(h.GET(path, rep.Token))
	if len(got) != 2 || !got[a] || !got[b] {
		t.Fatalf("الأقسامُ بعد الحفظ: %v", got)
	}
	if mine := sectionIDs(h.GET("/api/v1/merchant/stores/"+m.ID+"/sections", m.Owner.Token)); !mine[a] || !mine[b] {
		t.Fatalf("صاحبُ المتجر لا يرى ما ضبطه مندوبُه: %v", mine)
	}

	// يزيل أحدهما — الاستبدالُ لا الإضافة
	if r := h.Call("PUT", path, rep.Token, map[string]any{"sections": []string{b}}, nil); r.Code != 200 {
		t.Fatalf("الحفظ الثاني: %s", r)
	}
	if got := sectionIDs(h.GET(path, rep.Token)); len(got) != 1 || !got[b] {
		t.Fatalf("بعد الإزالة: %v", got)
	}

	// مندوبٌ آخر: ٤٠٣ ولا أثر
	if r := h.Call("PUT", path, other.Token, map[string]any{"sections": []string{a}}, nil); r.Code != 403 {
		t.Fatalf("مندوبٌ غريب: %d", r.Code)
	}
	if got := sectionIDs(h.GET(path, rep.Token)); len(got) != 1 || !got[b] {
		t.Fatalf("غريبٌ غيّر الأقسام: %v", got)
	}

	// معرّفٌ غير موجود يُتجاهل كما عند المتجر ولا يُسقط الحفظ
	ghost := "00000000-0000-0000-0000-000000000000"
	if r := h.Call("PUT", path, rep.Token, map[string]any{"sections": []string{ghost, a}}, nil); r.Code != 200 {
		t.Fatalf("معرّفٌ شاخ: %s", r)
	}
	if got := sectionIDs(h.GET(path, rep.Token)); len(got) != 1 || !got[a] {
		t.Fatalf("بعد المعرّف الشائخ: %v", got)
	}
}
