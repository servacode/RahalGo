package comms

// **متى تُفتح قناةُ الطلب ومتى تُغلق.**
//
// (بُنيت ٢٠٢٦-٠٨-٠٩ على وثيقة المالك في إخفاء الأرقام.)
//
// # ولماذا الحكمُ مفصولٌ عن القاعدة
//
// **`Permit` تفتح قاعدةً فلا يبلغها اختبار** — والحكمُ هو ما يُخطئ فيه:
// **طلبٌ مُلغًى تبقى قناتُه مفتوحة**، أو مُسلَّمٌ تُغلق لحظتَه فينقطع الحديثُ
// وصاحبُه يفتح الكيسَ ويرى نقصاً.
//
// **وحالاتُ رحّال غو لا حالاتُ الوثيقة**: كُتب فيها `DRIVER_ASSIGNED` و
// `DELIVERING` ولا وجودَ لهما. **واسمٌ يُنقل حرفيّاً من ورقةٍ إلى شيفرة يصير
// شرطاً لا يتحقّق أبداً** — فتبقى القناةُ مغلقةً على كلّ طلب، ولا خطأ.

import (
	"testing"
	"time"
)

func TestChannelOpensOnlyWhileTheDriverIsCarrying(t *testing.T) {
	fresh := time.Now().Add(-1 * time.Minute)

	cases := []struct {
		name      string
		status    string
		delivered *time.Time
		want      bool
	}{
		// **ما دام الطلبُ في يده** — الحالاتُ الخمس.
		{"أُسند", "assigned", nil, true},
		{"عند المتجر", "at_pickup", nil, true},
		{"استلم", "picked_up", nil, true},
		{"في الطريق", "on_the_way", nil, true},
		{"عند الباب", "at_dropoff", nil, true},

		// **وقبل الإسناد لا طرفَ ثانيَ أصلاً.**
		{"معلَّق", "pending", nil, false},
		{"قبله المتجر", "accepted", nil, false},
		{"يُوزَّع", "dispatching", nil, false},

		// **والمسلَّمُ يُغلق لحظتَه** — (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «مجرّد
		// تسليم الطلب أو انتهاء الطلب تُغلق الدردشة»).
		//
		// **وكانت ربعُ ساعةٍ مهلةً** — فبقيت القناةُ مفتوحةً بعد أن انتهى
		// الطلب. **وما بعد التسليم بابُه الشكوى.**
		{"سُلّم قبل دقيقة", "delivered", &fresh, false},
		{"سُلّم بلا وقت", "delivered", nil, false},

		// **وما انتهى بغير تسليمٍ مغلق.**
		{"أُلغي", "cancelled", nil, false},
		{"رُفض", "rejected", nil, false},
		{"تعذّر", "failed", nil, false},
		{"استُرجع", "refunded", nil, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := channelOpen(c.status, c.delivered, nil)
			if got != c.want {
				t.Errorf("%q → %v، والمنتظَر %v", c.status, got, c.want)
			}
		})
	}
}

// TestNoChannelSurvivesTheOrder **ولا نهايةَ تُبقي القناةَ مفتوحة.**
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «لا تبقى أيُّ دردشةٍ مفتوحة».)
//
// **والنهاياتُ خمس** — والتسليمُ أخطرُها أن يُنسى: **هو النهايةُ الطبيعيّةُ
// وحدَه**، وكلُّ ما عداه استثناءٌ يُنتبَه له.
func TestNoChannelSurvivesTheOrder(t *testing.T) {
	at := time.Now()
	for _, st := range []string{"delivered", "cancelled", "rejected", "failed", "refunded"} {
		if open, _ := channelOpen(st, &at, &at); open {
			t.Errorf("قناةُ %q بقيت مفتوحةً بعد انتهاء الطلب", st)
		}
	}
}
