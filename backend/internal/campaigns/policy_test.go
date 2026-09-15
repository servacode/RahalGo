package campaigns

// ══════════════════════════════════════════════════════════════════════
// **سياسةُ الإزعاج — قرارٌ يُقاس بلا قاعدة** (`NT`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════

import (
	"testing"
	"time"
)

func local(h int) time.Time {
	return time.Date(2026, 9, 15, h, 0, 0, 0, damascus)
}

// ═════════════════ NT-16 · NT-17 ═════════════════

// TestQuietCrossesMidnight **وساعةُ الهدوء تعبر منتصفَ الليل.**
//
// **ومن قارن `>= from && < to` وحدَه أسكت النهارَ كلَّه** — **٢٢→٨
// ليست مجالاً تصاعديّاً.**
func TestQuietCrossesMidnight(t *testing.T) {
	q := DefaultQuiet
	quiet := []int{22, 23, 0, 3, 7}
	loud := []int{8, 9, 12, 17, 21}
	for _, h := range quiet {
		if !q.InQuiet(local(h)) {
			t.Fatalf("**الساعةُ %d ليست هدوءاً وهي منه**", h)
		}
	}
	for _, h := range loud {
		if q.InQuiet(local(h)) {
			t.Fatalf("**الساعةُ %d عُدّت هدوءاً** — **فيُسكَت النهار**", h)
		}
	}
}

// TestQuietEqualBoundsMeansNoQuiet **ومتساويان يعني لا هدوء.**
//
// **ولا يُسكَت اليومُ كلُّه بخطإِ ضبط.**
func TestQuietEqualBoundsMeansNoQuiet(t *testing.T) {
	q := Quiet{From: 9, To: 9}
	for h := 0; h < 24; h++ {
		if q.InQuiet(local(h)) {
			t.Fatalf("**سُكت اليومُ كلُّه**: %d", h)
		}
	}
}

// TestNextAllowedDefersNotDrops **والمؤجَّلُ يُؤجَّل ولا يُسقَط.**
func TestNextAllowedDefersNotDrops(t *testing.T) {
	q := DefaultQuiet
	// **ليلاً** — **يُؤجَّل إلى الثامنة صباحاً.**
	at := local(23)
	next := q.NextAllowed(at)
	if !next.After(at) {
		t.Fatalf("**لم يُؤجَّل**: %v", next)
	}
	if next.In(damascus).Hour() != 8 {
		t.Fatalf("**أُجّل إلى غير الثامنة**: %v", next)
	}
	// **وبعد منتصف الليل** — **إلى ثامنة اليوم نفسِه لا الغد.**
	early := time.Date(2026, 9, 15, 3, 0, 0, 0, damascus)
	n2 := q.NextAllowed(early)
	if n2.In(damascus).Day() != early.Day() || n2.In(damascus).Hour() != 8 {
		t.Fatalf("**أُجّل يوماً كاملاً بلا سبب**: %v", n2)
	}
	// **ونهاراً لا يُؤجَّل شيء.**
	noon := local(12)
	if !q.NextAllowed(noon).Equal(noon) {
		t.Fatalf("**أُجّل ما لا يُؤجَّل**")
	}
}

// TestTransactionalIsNotEngagement **والمعاملةُ لا تخضع للسياسة.**
//
// **ومن أخّر «وصل سائقُك» إلى الصباح أفسد التوصيل.**
func TestTransactionalIsNotEngagement(t *testing.T) {
	if Engagement(CategoryTransactional) {
		t.Fatalf("**خضعت المعاملةُ لسياسة الإزعاج**")
	}
	if !Engagement(CategoryEngagement) {
		t.Fatalf("**أُعفي التفاعلُ من السياسة**")
	}
	// **وما لا يُعرَف صنفُه يُسلَّم ولا يُؤجَّل** — **والافتراضُ
	// تسليمٌ**: **ومن جهل صنفَ خبرٍ فأخّره أخّر خبرَ طلب.**
	if Engagement("something_new") {
		t.Fatalf("**أُجّل ما لا يُعرَف**")
	}
}

// ═════════════════ NT-14 ═════════════════

func TestCap(t *testing.T) {
	if !UnderCap(0, 2) || !UnderCap(1, 2) {
		t.Fatalf("**مُنع من لم يبلغ سقفَه**")
	}
	if UnderCap(2, 2) || UnderCap(5, 2) {
		t.Fatalf("**مرّ من بلغ سقفَه**")
	}
	// **وصفرٌ يمنع كلَّ تفاعل** — **ضبطٌ مشروعٌ في أزمة.**
	if UnderCap(0, 0) {
		t.Fatalf("**السقفُ صفرٌ ومرّ**")
	}
}

// ═════════════════ NT-12 ═════════════════

// TestValidDest **ووجهةٌ لا نعرفها تُردّ.**
//
// **ورابطٌ حرٌّ في إشعارٍ بابُ تصيّدٍ في جيب الزبون.**
func TestValidDest(t *testing.T) {
	ok := [][2]string{
		{"", ""}, {DestHome, ""},
		{DestOffer, "8b0a8a71-d186-4647-ae3b-9cd3898508bf"},
		{DestMerchant, "8b0a8a71-d186-4647-ae3b-9cd3898508bf"},
	}
	for _, c := range ok {
		if !ValidDest(c[0], c[1]) {
			t.Fatalf("**رُدّت وجهةٌ صحيحة**: %v", c)
		}
	}
	bad := [][2]string{
		{"https://evil.example/pay", ""},
		{"intent://x", "y"},
		{DestOffer, ""},       // **بلا معرّفٍ لا تُفتَح**
		{DestHome, "some-id"}, // **والبيتُ لا معرّفَ له**
		{"order", "x"},        // **وخبرُ الطلب ليس حملةً**
		{"../../etc/passwd", "x"},
	}
	for _, c := range bad {
		if ValidDest(c[0], c[1]) {
			t.Fatalf("**قُبلت وجهةٌ لا تُعرَف**: %v", c)
		}
	}
}

// ═════════════════ NT-05 · NT-13 ═════════════════

func TestValidAudience(t *testing.T) {
	if !ValidAudience(AudienceRole, "customer") ||
		!ValidAudience(AudienceInterest, "city:8b0a8a71") ||
		!ValidAudience(AudienceInterest, "cell:35.95,39.01") {
		t.Fatalf("**رُدّ جمهورٌ مفهوم**")
	}
	bad := [][2]string{
		{AudienceRole, "admin"},       // **ولا يُخاطَب موظّفٌ بحملة**
		{AudienceRole, "' OR 1=1 --"}, // **ولا شرطَ يُكتب**
		{AudienceInterest, "everyone"},
		{"sql", "SELECT 1"},
		{"", ""},
	}
	for _, c := range bad {
		if ValidAudience(c[0], c[1]) {
			t.Fatalf("**قُبل جمهورٌ لا يُعرَف**: %v", c)
		}
	}
}

// ═════════════════ NT-09 · NT-10 ═════════════════

func TestCancellable(t *testing.T) {
	if !Cancellable(StatusDraft) || !Cancellable(StatusScheduled) {
		t.Fatalf("**مُنع إلغاءُ ما لم يبدأ**")
	}
	// **وما بدأ إرسالُه لا يُسحَب** — **الرسالةُ في جيوبهم.**
	for _, st := range []string{StatusSending, StatusSent, StatusCancelled, StatusFailed} {
		if Cancellable(st) {
			t.Fatalf("**أُلغي ما لا يُلغى**: %s", st)
		}
	}
}
