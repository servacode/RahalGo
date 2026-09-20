package qa

// ══════════════════════════════════════════════════════════════════════
// **الإدارةُ ترى الناسَ لا الضغطات** (`CUST-07-033`)
// ══════════════════════════════════════════════════════════════════════
//
// **وقرارُ التوسّعِ يُبنى على «كم شخصاً ينتظر» لا «كم ضغطةً وقعت».**
// **فمن ضغط الزرَّ عشراً وحدَه لا يصنع سوقاً**، **وعشرةٌ ضغطوا مرّةً
// يصنعونها.**
//
// **و`DemandByPlace` تعدّ `count(DISTINCT user_id)` في `People` وتجمع
// الضغطاتِ في `Signals`** — **فهذا الاختبارُ يُثبت أنّهما يفترقان**:
// حسابٌ يضغط ثلاثاً وآخرُ مرّةً في المدينةِ نفسِها ⇒ `People = 2`
// و`Signals = 4`.

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// TestCUST07033_DemandByPlaceCountsUniquePeople **العدُّ بالأشخاص لا
// بالضغطات.**
func TestCUST07033_DemandByPlaceCountsUniquePeople(t *testing.T) {
	hh := New(t)
	// الرقّةُ مخدومةٌ ودمشقُ خارجَها — فضغطةُ دمشقَ نيّةُ خدمة.
	zoneForDemand(t, hh, "منطقةُ CUST-07-033")

	// **وأساسٌ يُقاس قبلُ** — `New(t)` لا يُفرِغ `coverage_requests`، فقد
	// تسبق طلباتُ اختباراتٍ أخرى في المدينةِ نفسِها. **وحساباتي جُددٌ
	// فرِيدو المعرِّف**، فمساهمتُهم في العدِّ = فرقٌ مضبوطٌ (‎+2 شخص، ‎+4 ضغطة)
	// مهما كان الأساس.
	people0, signals0 := interestTotals(t, hh)

	a := hh.Customer()
	b := hh.Customer()

	// **حسابٌ يضغط ثلاثاً** — ضغطاتٌ مكرّرةٌ في المكانِ نفسِه.
	for i := 0; i < 3; i++ {
		if r := demand(t, hh, a.Token, damA_Lat, damA_Lng, ""); r.Code != 200 {
			t.Fatalf("**ضغطةُ (أ) رُدّت**: %d / %s", r.Code, r.Err())
		}
	}
	// **وحسابٌ آخرُ يضغط مرّةً** في المدينةِ نفسِها.
	if r := demand(t, hh, b.Token, damA_Lat, damA_Lng, ""); r.Code != 200 {
		t.Fatalf("**ضغطةُ (ب) رُدّت**: %d / %s", r.Code, r.Err())
	}

	// **صحّةُ الدمج**: صفٌّ واحدٌ لكلِّ حساب، وضغطاتُ (أ) = 3.
	if rows, reqs := demandRows(t, hh, a.ID, "service_interest"); rows != 1 || reqs != 3 {
		t.Fatalf("**دمجُ (أ) مكسور**: صفوف=%d ضغطات=%d (المنتظَر 1/3)", rows, reqs)
	}
	if rows, reqs := demandRows(t, hh, b.ID, "service_interest"); rows != 1 || reqs != 1 {
		t.Fatalf("**دمجُ (ب) مكسور**: صفوف=%d ضغطات=%d (المنتظَر 1/1)", rows, reqs)
	}

	people1, signals1 := interestTotals(t, hh)

	// **الناسُ اثنان لا أربعة** — التفرّدُ بالحساب لا بالضغطة.
	if dp := people1 - people0; dp != 2 {
		t.Fatalf("**عُدَّت الضغطاتُ أشخاصاً**: ΔPeople=%d (المنتظَر 2) — "+
			"**فيُظنّ سوقٌ حيث ينتظر واحد**", dp)
	}
	// **والضغطاتُ أربعٌ** — يُحفظ العددُ الخام لكن لا يُعرَض «ناساً».
	if ds := signals1 - signals0; ds != 4 {
		t.Fatalf("**مجموعُ الضغطات مكسور**: ΔSignals=%d (المنتظَر 4)", ds)
	}
	// **والعدّان يفترقان** — لو ساواهما لعُرضت الضغطاتُ ناساً.
	if people1-people0 == signals1-signals0 {
		t.Fatalf("**الأشخاصُ = الضغطاتُ** — لم يُفرَّق العدّان")
	}
}

// interestTotals **مجموعُ أشخاصِ نيّةِ الخدمةِ وضغطاتِها** عبر كلِّ الأماكن.
//
// **ويُقاس فرقاً لا مطلقاً** — فالجدولُ مشترَكٌ بين الاختبارات.
func interestTotals(t *testing.T, h *Harness) (people, signals int) {
	t.Helper()
	places, err := opsmap.DemandByPlace(ctxBG(), h.Pool, "service_interest")
	if err != nil {
		t.Fatalf("DemandByPlace: %v", err)
	}
	for _, p := range places {
		if p.Kind != "service_interest" {
			continue
		}
		people += p.People
		signals += p.Signals
	}
	return people, signals
}
