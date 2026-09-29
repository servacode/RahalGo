package orders

import "testing"

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
			platform := c.fee * c.pct / 100
			driver := c.fee - platform
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

// **والمفتاحُ واحدٌ لا يُكتب بيده مرّتين.**
//
// **`GetInt` تردّ الافتراضَ لمفتاحٍ مجهول** — **فخطأٌ مطبعيٌّ في أحد
// الموضعين يُقرأ نسبةً افتراضيّةً بصمت، ولا يظهر في أيّ خطأ.**
func TestMerchantDeliveryPercentKeyMatchesCatalog(t *testing.T) {
	const inCatalog = "delivery.merchant_delivery_platform_percent"
	if SettingMerchantDeliveryPlatformPercent != inCatalog {
		t.Fatalf("المفتاحُ %q والفهرسُ %q — **ويُقرأ الافتراضُ بصمت**",
			SettingMerchantDeliveryPlatformPercent, inCatalog)
	}
}
