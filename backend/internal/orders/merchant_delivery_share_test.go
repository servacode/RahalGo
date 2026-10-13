package orders

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// قسمةُ أجرة «لدي توصيلة» — **نصيبُ المنصّة والباقي للسائق.**
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٩ بمثاله نصّاً: «delivery_fee = 5,000 · platform
//
//	share = 500 · driver earning = 4,500».)
//
// **والحسبةُ تُحرَس بالأرقام لا بالقراءة** — **قسمةٌ صحيحةٌ تُقرّب إلى
// الأسفل، ومن جمع النصيبَ والأجرَ فوجدهما دون الأجرة عرف أين ذهبت الليرة.**
func TestMerchantDeliveryShare(t *testing.T) {
	cases := []struct {
		name                  string
		fee, pct              int64
		wantPlatform, wantDrv int64
	}{
		// **مثالُ المالك حرفاً.**
		{"مثالُ المالك", 5000, 10, 500, 4500},
		// **ولا نصيبَ حين تُصفَّر النسبة** — المنصّةُ تمرّ بلا أن تأخذ.
		{"صفرٌ للمنصّة", 5000, 0, 0, 5000},
		// **والكسرُ لا يضيع**: القسمةُ إلى الأسفل، **والباقي للسائق** —
		// **فمجموعُهما يساوي الأجرةَ دائماً ولا ليرةَ تتبخّر.**
		{"كسرٌ يُقرَّب", 333, 10, 33, 300},
		{"أجرةٌ صفر", 0, 10, 0, 0},
		// **والحدُّ الأعلى تسعون** — فلا يُصفَّر أجرُ السائق أبداً.
		{"أقصى نصيب", 1000, 90, 900, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			driver := DriverFeeAfterShare(c.fee, c.pct)
			platform := c.fee - driver
			if platform != c.wantPlatform || driver != c.wantDrv {
				t.Fatalf("أجرةٌ %d بنسبة %d%%: المنصّة %d والسائق %d — والمتوقّع %d و%d",
					c.fee, c.pct, platform, driver, c.wantPlatform, c.wantDrv)
			}
			// **والمجموعُ هو الأجرةُ** — **وليرةٌ تضيع بين الطرفين تضيع في
			// كلّ توصيلةٍ ولا تظهر في شيء.**
			if platform+driver != c.fee {
				t.Fatalf("المجموع %d لا يساوي الأجرة %d — **ليرةٌ تبخّرت**",
					platform+driver, c.fee)
			}
		})
	}
}

// **ومفاتيحُ الحصّة في الفهرس** — لكلّ نوعٍ قيمتُه وطريقتُه (٢٠٢٦-١٠-١٣).
//
// **`GetInt` تردّ الافتراضَ لمفتاحٍ مجهول** — **فخطأٌ مطبعيٌّ يُقرأ صفراً بصمت.**
func TestPlatformShareKeysInCatalog(t *testing.T) {
	for _, k := range []string{ShareStandard, ShareMerchant, SharePurchase, ShareAmanah} {
		for _, key := range []string{SettingShare(k), SettingShare(k) + "_mode"} {
			if _, ok := settings.Lookup(key); !ok {
				t.Errorf("المفتاحُ %q غائبٌ عن الفهرس — **ويُقرأ صفراً بصمت**", key)
			}
		}
	}
}

// **والمبلغُ الثابتُ يُطرح كما هو ولا يتجاوز الأجرة** — ١٥٠ وحصّةٌ ٥٠ ⇒ للسائق ١٠٠.
func TestDriverFeeWithShare(t *testing.T) {
	cases := []struct{ fee, pct, fixed, want int64 }{
		{150, 0, 50, 100},
		{150, 0, 0, 150},
		{150, 10, 0, 135},
		{100, 0, 500, 0},
		{0, 0, 50, 0},
	}
	for _, c := range cases {
		if got := DriverFeeWithShare(c.fee, c.pct, c.fixed); got != c.want {
			t.Errorf("أجرة %d · نسبة %d · ثابت %d ⇒ %d والمنتظَر %d", c.fee, c.pct, c.fixed, got, c.want)
		}
	}
}
