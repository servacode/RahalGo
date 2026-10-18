package qa

import (
	"encoding/json"
	"testing"
)

// TestHours24h_API **«مفتوح ٢٤ ساعة» يُكتب بالراية ويُقرأ بها — والقديمُ بساعتيه.**
//
// (قرارُ المالك ٢٠٢٦-١٠-١٨.)
//
//   - التطبيقُ الجديد يرسل `open_24h: true` ⇒ تُكتب ٠٠:٠٠ ← ٠٠:٠٠
//   - التطبيقُ القديم يرسل ٠٠:٠٠ ← ٠٠:٠٠ بلا راية ⇒ يُقرأ `open_24h: true`
//   - ويومٌ عاديٌّ يبقى كما هو و`open_24h: false`
//   - **ومتجرٌ مفتوحٌ ٢٤ ساعة لا موعدَ فتحٍ له** (`next_open: null`)
func TestHours24h_API(t *testing.T) {
	h := New(t)
	m := h.Factory().Merchant()
	path := "/api/v1/merchant/stores/" + m.ID + "/hours"

	days := []map[string]any{}
	for d := 0; d < 7; d++ {
		day := map[string]any{"day_of_week": d, "closed": false, "open_time": "09:00", "close_time": "23:00"}
		switch d {
		case 0, 2, 3, 4, 5, 6:
			// **الجديد**: الرايةُ وحدَها، والساعتان المرسَلتان معها لا تُقرآن.
			day["open_24h"] = true
		case 1:
			// **القديم**: ساعتان صفرٌ بلا راية.
			day["open_time"], day["close_time"] = "00:00", "00:00"
		}
		days = append(days, day)
	}
	if res := h.Call("PUT", path, m.Owner.Token, map[string]any{"days": days}, nil); res.Code >= 400 {
		t.Fatalf("حفظُ الدوام: %s", res)
	}

	res := h.GET(path, m.Owner.Token)
	if res.Code >= 400 {
		t.Fatalf("قراءةُ الدوام: %s", res)
	}
	var got []struct {
		Day     int    `json:"day_of_week"`
		Open    string `json:"open_time"`
		Close   string `json:"close_time"`
		Open24h *bool  `json:"open_24h"`
	}
	if err := json.Unmarshal(rawData(res), &got); err != nil {
		t.Fatalf("فكُّ الدوام: %v — %s", err, res)
	}
	if len(got) != 7 {
		t.Fatalf("سبعةُ أيّامٍ منتظَرة: %s", res)
	}
	for _, d := range got {
		if d.Open24h == nil {
			t.Fatalf("يوم %d بلا حقل open_24h: %s", d.Day, res)
		}
		if !*d.Open24h || d.Open != "00:00" || d.Close != "00:00" {
			t.Errorf("يوم %d: open_24h=%v %s←%s — والمنتظَر ٢٤ ساعة ٠٠:٠٠←٠٠:٠٠",
				d.Day, *d.Open24h, d.Open, d.Close)
		}
	}

	// **ومفتوحٌ الآن بلا موعد** — لا «يفتح الساعة ١٢:٠٠ ص».
	stores := h.GET("/api/v1/merchant/stores", m.Owner.Token)
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(rawData(stores), &list); err != nil {
		var wrapped struct {
			Stores []map[string]json.RawMessage `json:"stores"`
		}
		_ = json.Unmarshal(rawData(stores), &wrapped)
		list = wrapped.Stores
	}
	found := false
	for _, s := range list {
		var id string
		_ = json.Unmarshal(s["id"], &id)
		if id != m.ID {
			continue
		}
		found = true
		if string(s["open_now"]) != "true" {
			t.Errorf("متجرٌ ٢٤ ساعة open_now=%s", s["open_now"])
		}
		if string(s["next_open"]) != "null" {
			t.Errorf("متجرٌ مفتوحٌ ٢٤ ساعة قيل له «يفتح» %s", s["next_open"])
		}
	}
	if !found {
		t.Fatalf("المتجر ليس في قائمة صاحبه: %s", stores)
	}

	// **ويومٌ عاديٌّ يعود عاديّاً** — الرايةُ لا تلتصق.
	days[0] = map[string]any{"day_of_week": 0, "closed": false, "open_time": "09:00", "close_time": "23:00", "open_24h": false}
	if res := h.Call("PUT", path, m.Owner.Token, map[string]any{"days": days}, nil); res.Code >= 400 {
		t.Fatalf("حفظُ الدوام ثانيةً: %s", res)
	}
	res = h.GET(path, m.Owner.Token)
	_ = json.Unmarshal(rawData(res), &got)
	if len(got) == 7 && (got[0].Open24h == nil || *got[0].Open24h || got[0].Open != "09:00") {
		t.Errorf("الأحد بعد إطفاء الراية: %+v", got[0])
	}
}
