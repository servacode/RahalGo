package qa

// ═══════════════════════════════════════════════════════════════════════
// **ولا هويّةَ فارغة** (`SI-11`…`SI-16`، ٢٠٢٦-٠٩-١٤)
// ═══════════════════════════════════════════════════════════════════════
//
// **و`NOT NULL` تمنع العدمَ ولا تمنع الفراغ** — **و`''` هويّةٌ قائمةٌ
// في القاعدة لا تُطابق شيئاً.**
//
// **وصفٌّ هويّتُه فارغةٌ**: **لا يُدمَج فيه جديدٌ فيتكاثر**، **ولا
// يُلغى فيُحبَس صاحبُه**، **ولا يُطابقه سؤالُ «من يُخبَر؟»** — **يُعَدّ
// في الكثافة وصاحبُه لا يُخبَر أبداً.**

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// ═════════════════ SI-11 · SI-12 — البناءُ لا يُخرج فارغاً ═════════════════

// TestSI11_SI12_TargetConstructorNeverEmpty **وكلُّ حالٍ مقبولةٍ لها
// هويّةٌ تُقرأ.**
//
// **ولا بديلَ مخترَع** — **هويّةٌ مصنوعةٌ تُخفي العطبَ ولا تُصلحه.**
func TestSI11_SI12_TargetConstructorNeverEmpty(t *testing.T) {
	cases := []struct {
		name   string
		kind   string
		city   string
		cy, cx float64
		prefix string
	}{
		{"مدينةٌ معروفة", opsmap.KindInterest,
			"3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d", 33.51, 36.27, "city:"},
		{"موضعٌ مجهول", opsmap.KindInterest, "", 34.20, 38.60, "cell:"},
		{"طلبُ تغطيةٍ في مدينةٍ معروفة", opsmap.KindCoverage,
			"3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d", 36.03, 39.00, "cell:"},
		{"طلبُ تغطيةٍ بلا مدينة", opsmap.KindCoverage, "", 36.03, 39.00, "cell:"},
		{"خليّةُ الصفر", opsmap.KindInterest, "", 0, 0, "cell:"},
		{"خليّةٌ سالبة", opsmap.KindCoverage, "", -33.5, -70.6, "cell:"},
	}
	for _, c := range cases {
		got := opsmap.TargetKey(c.kind, c.city, c.cy, c.cx)
		if !strings.HasPrefix(got, c.prefix) {
			t.Fatalf("%s: **شكلٌ غيرُ متوقَّع**: %q — المنتظَرُ %q", c.name, got, c.prefix)
		}
		// **والبادئةُ وحدَها ليست هويّة** — **`cell:` بلا إحداثيّةٍ
		// لا تُميّز موضعاً**، **و`city:` بلا معرّفٍ لا تُطابق مدينة.**
		if strings.TrimSpace(strings.TrimPrefix(got, c.prefix)) == "" {
			t.Fatalf("%s: **هويّةٌ بادئةٌ بلا جسد**: %q", c.name, got)
		}
		if !opsmap.ValidTarget(got) {
			t.Fatalf("%s: **البناءُ أخرج هويّةً لا تصلح**: %q", c.name, got)
		}
	}
}

// TestSI12_ValidTargetRejectsBlank **والحارسُ يعرف الفراغَ والبياض.**
func TestSI12_ValidTargetRejectsBlank(t *testing.T) {
	tab, nl := string(rune(9)), string(rune(10))
	for _, bad := range []string{"", " ", "   ", tab, nl, " " + tab + nl + " "} {
		if opsmap.ValidTarget(bad) {
			t.Fatalf("**هويّةٌ بيضاءُ قُبلت**: %q", bad)
		}
	}
	for _, good := range []string{"cell:33.51,36.27", "city:3f2b1c4d"} {
		if !opsmap.ValidTarget(good) {
			t.Fatalf("**هويّةٌ سليمةٌ رُدّت**: %q", good)
		}
	}
}

// ═════════════════ SI-13 … SI-16 — والقاعدةُ تحرس بعده ═════════════════

// TestSI13_SI16_DatabaseRefusesBlankTarget **والكتابةُ بالسطر
// مباشرةً تُردّ.**
//
// **وحارسُ المجال يُتجاوَز بنداءٍ لا يمرّ به** — **مهاجرةٌ أو يدٌ على
// `psql`** — **فالقاعدةُ هي الحدُّ الأخير.**
func TestSI13_SI16_DatabaseRefusesBlankTarget(t *testing.T) {
	hh := New(t)
	u := hh.Customer()

	ins := func(target any) error {
		_, err := hh.Pool.Exec(ctxBG(), `
			INSERT INTO coverage_requests
			  (user_id, at, address_text, source, kind, cell_y, cell_x, target_key)
			VALUES ($1::uuid, ST_SetSRID(ST_MakePoint(39.0, 36.0), 4326)::geography,
			        'برهان', 'customer_app', 'service_interest', 36.0, 39.0, $2)`,
			u.ID, target)
		return err
	}

	// **SI-13 · العدمُ يُردّ.**
	if err := ins(nil); err == nil {
		t.Fatalf("**صفٌّ بهويّةٍ عدمٍ كُتب**")
	}
	// **SI-14 · والفراغُ يُردّ.**
	if err := ins(""); err == nil {
		t.Fatalf("**صفٌّ بهويّةٍ فارغةٍ كُتب** — **لا يُلغى ولا يُدمَج فيه**")
	}
	// **SI-15 · والبياضُ فراغٌ.**
	for _, blank := range []string{" ", "   ", string(rune(9)) + string(rune(10))} {
		if err := ins(blank); err == nil {
			t.Fatalf("**صفٌّ بهويّةٍ بيضاءَ كُتب**: %q", blank)
		}
	}
	// **SI-16 · والسليمُ يمرّ** — **وحدٌّ يمنع الصوابَ حدٌّ خاطئ.**
	if err := ins("city:3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"); err != nil {
		t.Fatalf("**هويّةٌ سليمةٌ رُدّت**: %v", err)
	}
}
