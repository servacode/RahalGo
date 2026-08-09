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
	now := time.Now()
	fresh := now.Add(-1 * time.Minute)
	stale := now.Add(-GraceAfterDelivery - time.Minute)

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

		// **والمسلَّمُ يُقاس بوقته لا بحالته.**
		{"سُلّم قبل دقيقة", "delivered", &fresh, true},
		{"سُلّم قبل ساعة", "delivered", &stale, false},
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

// TestGraceIsReportedSoTheUserSeesIt **وقتُ الإغلاق يُقال لا يُفاجئ.**
//
// **ومن كتب رسالةً فرُدّت بلا سببٍ يعيدها** — ثمّ يظنّ التطبيقَ معطوباً.
func TestGraceIsReportedSoTheUserSeesIt(t *testing.T) {
	at := time.Now().Add(-2 * time.Minute)
	open, closes := channelOpen("delivered", &at, nil)
	if !open {
		t.Fatal("قناةُ طلبٍ سُلّم قبل دقيقتين يجب أن تبقى مفتوحة")
	}
	if closes == nil {
		t.Fatal("لا وقتَ إغلاقٍ يُعرض — فيُفاجأ من كان يكتب")
	}
	if want := at.Add(GraceAfterDelivery); !closes.Equal(want) {
		t.Errorf("الإغلاق %v، والمنتظَر %v", closes, want)
	}
}
