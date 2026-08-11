package orders

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// إشعارات دورة حياة الطلب.
//
// كان نظام الإشعارات موصولاً بأحداث الإدارة فقط، ودورة الطلب — قلب المنصة —
// لا تُشعر أحداً: الزبون لا يعرف أن طلبه قُبل، والمتجر لا يجد أثراً لطلب جديد
// إن أغلق التبويب، والمندوب لا يعلم بعمولة دخلت محفظته.
//
// المبدأ: **إشعار عند ما يهمّ فعلاً** لا عند كل انتقال. حالات المرور الداخلية
// (dispatching، at_pickup، at_dropoff) لا تُشعر الزبون — يتابعها في خط التقدّم
// الحي. الإشعار لما يغيّر انتظاره أو ماله.

// نصوص إشعارات الطلبات — مجمّعة كي لا تتناثر (كنمط notifTitles في الخادم).
var t = struct {
	newOrderMerchant, newOrderOps            string
	accepted, preparing, onTheWay, delivered string
	rejected, cancelled, failed, refunded    string
	merchantDelivered, commission            string
	violationsWarn, violationsBanned         string
	endedOps, warningIssued                  string
	// **عناوينُ حركات المحفظة** — (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد
	// يتغيّر وما حدا بيعرف ليش»).
	driverEarned, merchantEarned, refunded2, compensated string
}{
	newOrderMerchant:  "طلب جديد وصلك",
	newOrderOps:       "طلب جديد في المنصة",
	accepted:          "قبل المتجر طلبك",
	preparing:         "طلبك قيد التحضير",
	onTheWay:          "طلبك في الطريق إليك",
	delivered:         "تم تسليم طلبك",
	rejected:          "اعتذر المتجر عن طلبك",
	cancelled:         "أُلغي طلبك",
	failed:            "تعذّر تسليم طلبك",
	refunded:          "استُرجع مبلغ طلبك",
	merchantDelivered: "سُلّم طلب من متجرك",
	commission:        "عمولة جديدة في محفظتك",
	violationsWarn:    "متجرٌ بلغ حدّ المخالفات",
	violationsBanned:  "حُظر متجرٌ لكثرة الإلغاء",
	endedOps:          "انتهى طلبٌ قبل تسليمه",
	warningIssued:     "إنذارٌ على متجرك",
	driverEarned:      "أجر توصيل في محفظتك",
	merchantEarned:    "مستحق مبيعاتك في محفظتك",
	refunded2:         "أُعيد المبلغ إلى محفظتك",
	compensated:       "تعويض في محفظتك",
}

// endedByLabel من أنهى الطلب — بلفظٍ يُقرأ لا برمزٍ يُفكّ.
//
// **وثلاثةُ أخبارٍ يُخفيها لفظُ «ملغي» وحدَه**: إلغاءُ الزبون خبرٌ لا يستوجب
// شيئاً، **وإلغاءُ المتجر يستوجب اتّصالاً به** وقد يُحتسب عليه مخالفة، وإلغاءُ
// العمليات فعلُنا نحن. **ومن لا يعرف أيُّها وقع يتّصل بالثلاثة أو لا يتّصل
// بأحد.**
var endedByLabel = map[string]string{
	"customer": "ألغاه الزبون",
	"merchant": "ألغاه المتجر",
	"ops":      "ألغته العمليات",
	"admin":    "ألغته الإدارة",
	"driver":   "أنهاه السائق",
}

// orderParties أطراف الطلب الذين قد يُشعَرون.
type orderParties struct {
	number        int64
	customerID    string
	merchantOwner *string
	merchantName  string
	repID         *string
}

func (s *Service) parties(ctx context.Context, orderID string) (orderParties, error) {
	var p orderParties
	err := s.db.QueryRow(ctx, `
		SELECT o.number, o.customer_id, mm.owner_user_id,
		       -- **واسمُ المتجر فارغٌ في الطلب الخاصّ** — لا متجرَ له.
		       COALESCE(mm.name, ''), mm.sales_rep_user_id
		-- **ويُضمّ يساراً** — (٢٠٢٦-٠٨-٠٩): **وضمٌّ صلبٌ يُسكت إشعاراتِ الطلب
		-- الخاصّ كلَّها** — لا الزبونُ يُخبَر ولا العملياتُ، **ولا خطأ يظهر**:
		-- الدالّةُ تردّ «لا صفوف» فيُبتلع.
		FROM orders o LEFT JOIN merchants mm ON mm.id = o.merchant_id
		WHERE o.id = $1`, orderID).
		Scan(&p.number, &p.customerID, &p.merchantOwner, &p.merchantName, &p.repID)
	return p, err
}

// notifyCreated المتجر يعرف بطلب جديد ولو أغلق تبويبه، ومكتب المنصة يتابع الحركة.
func (s *Service) notifyCreated(ctx context.Context, o *Order) {
	if s.notify == nil || o == nil {
		return
	}
	p, err := s.parties(ctx, o.ID)
	if err != nil {
		return
	}
	ref := fmt.Sprintf("#%d", p.number)
	if p.merchantOwner != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *p.merchantOwner, Kind: notifications.KindOrder,
			Title: t.newOrderMerchant, Body: ref,
			Entity: "order", EntityID: o.ID, Href: "/portal",
			// **يخصّ متجرَه لا حسابَه كزبون** — وهو يحمل التطبيقين.
			Apps: []string{notifications.AppMerchant},
		})
	}
	s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
		Kind: notifications.KindOrder, Title: t.newOrderOps,
		Body:   ref + " — " + p.merchantName,
		Entity: "order", EntityID: o.ID, Href: "/dashboard/orders",
	})
}

// customerTitles الانتقالات التي تستحق إشعاراً للزبون.
var customerTitles = map[string]string{
	StAccepted:  t.accepted,
	StPreparing: t.preparing,
	StOnTheWay:  t.onTheWay,
	StDelivered: t.delivered,
	StRejected:  t.rejected,
	StCancelled: t.cancelled,
	StFailed:    t.failed,
	StRefunded:  t.refunded,
}

// notifyTransition يُعلم من يخصّه هذا الانتقال. يُستدعى بعد نجاح الإيداع:
// فشل الإشعار لا يُبطل تسليماً وقع فعلاً.
func (s *Service) notifyTransition(ctx context.Context, orderID, to, note, endedBy string) {
	if s.notify == nil {
		return
	}
	title, worth := customerTitles[to]
	if !worth {
		return // حالة مرور داخلية — يتابعها الزبون في خط التقدّم الحي
	}
	p, err := s.parties(ctx, orderID)
	if err != nil {
		return
	}
	// **ولا اسمَ متجرٍ في إشعار الزبون.**
	//
	// حُجب المصدرُ في التصفّح وفي الطلبات وفي التقييمات — **وبقي في الإشعارات
	// وحدها**، وهي أكثرُ ما يُقرأ: تصل بلا أن تُطلب. **وشهده المالكُ في شاشته**
	// (٢٠٢٦-٠٨-٠٣): «يذكر اسم مطعم بيت الرقة».
	//
	// **وحجبٌ في ثلاثة مواضعَ من أربعة ليس حجباً** — يكفي بابٌ واحدٌ مفتوح.
	ref := fmt.Sprintf("#%d", p.number)
	body := ref
	if note != "" && (to == StRejected || to == StCancelled || to == StFailed) {
		body = ref + " — " + note // **والسببُ يُقال**: من أُلغي طلبُه يستحقّ لماذا
	}
	// **والرابطُ إلى القائمة لا إلى صفحةِ طلبٍ منفردة.**
	//
	// صفحةُ التفاصيل حُذفت — **البطاقةُ صارت تحمل كلَّ ما كان فيها.** وإشعارٌ
	// يفتح صفحةً غيرَ موجودة أسوأُ من إشعارٍ بلا رابط: **يُضغط فيصل إلى لا
	// شيء**، فيُقرأ عطباً في المنصة.
	s.notify.Notify(ctx, notifications.Input{
		UserID: p.customerID, Kind: notifications.KindOrder,
		Title: title, Body: body,
		Entity: "order", EntityID: orderID, Href: "/orders",
		// **«طلبُك في الطريق» يخصّ تطبيقَ الزبون** — ولو كان صاحبُه سائقاً.
		Apps: []string{notifications.AppCustomer},
	})

	// **والعملياتُ تُخبَر بمن أنهى** — لا بأنّ الطلب انتهى.
	//
	// كانت لا تُخبَر أصلاً: تكتشف الإلغاءَ حين تنظر في القائمة، **وتقرأ فيها
	// «ملغي» بلا فاعل**. فإن كان المتجرُ هو من ألغى فثمّة مكالمةٌ تستحقّ أن
	// تُجرى ومخالفةٌ تُحتسب، وإن كان الزبونَ فلا شيء.
	//
	// **ولا تُخبَر بفعل نفسِها**: من ألغى بيده لا يُنبَّه أنّ إلغاءً وقع.
	if (to == StCancelled || to == StRejected || to == StFailed) &&
		endedBy != "" && endedBy != "ops" && endedBy != "admin" {
		who := endedByLabel[endedBy]
		if to == StFailed {
			who = "تعذّر التسليم"
		}
		opsBody := ref + " — " + who + " · " + p.merchantName
		if note != "" {
			opsBody += " · " + note
		}
		s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
			Kind: notifications.KindOrder, Title: t.endedOps, Body: opsBody,
			Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
		})
	}

	// التسليم يخصّ المتجر أيضاً (اكتمل التزامه)
	if to == StDelivered && p.merchantOwner != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *p.merchantOwner, Kind: notifications.KindOrder,
			Title: t.merchantDelivered, Body: ref,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		})
	}
}

// notifyCommission المندوب يعرف بعمولته لحظة قيدها — مصدر دخله لا يُترك للاكتشاف.
func (s *Service) notifyCommission(ctx context.Context, repID, orderID string, amount int64) {
	if s.notify == nil || amount <= 0 {
		return
	}
	p, err := s.parties(ctx, orderID)
	if err != nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: repID, Kind: notifications.KindWallet,
		Title:  t.commission,
		Body:   fmt.Sprintf("#%d — %s", p.number, p.merchantName),
		Entity: "wallet", EntityID: orderID, Href: "/portal/wallet",
		// **عمولةُ المندوب تخصّ تطبيقَه** — وهو يحمل تطبيقَ الزبون أيضاً.
		Apps: []string{notifications.AppRep},
	})
}

// notifyCredits يُخبر كلَّ من تحرّكت محفظتُه — **بعد الإيداع لا داخلَه.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا لم تكن موجودة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا بيعرف ليش».)
//
// **السائقُ يوصّل ويقبض أجرَه فلا يصله شيء** — يرى الرقمَ في شريطه يزيد
// **ولا يعرف عمّاذا** إلّا إن فتح المحفظةَ وقرأ السطر. **وهو غالباً لا
// يفتحها**، فيتراكم عنده شكٌّ في الحساب لا سببَ له.
//
// **والمبلغُ في العنوان لا في النصّ وحدَه**: إشعارٌ يُقرأ من شاشةٍ مقفلةٍ
// في سطرٍ ونصف، **ومن أراد التفصيل فتح المحفظة.**
func (s *Service) notifyCredits(ctx context.Context, orderID string, credits []walletCredit) {
	if s.notify == nil || len(credits) == 0 {
		return
	}
	var ref string
	if p, err := s.parties(ctx, orderID); err == nil {
		ref = fmt.Sprintf("#%d", p.number)
	}
	for _, c := range credits {
		sign := "+"
		amount := c.amount
		if amount < 0 {
			sign = "−"
			amount = -amount
		}
		body := fmt.Sprintf("%s%d ل.س", sign, amount)
		if ref != "" {
			body += " — طلب " + ref
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: c.userID, Kind: notifications.KindWallet,
			Title: c.title, Body: body,
			Entity: "wallet", EntityID: orderID, Href: "/wallet",
			Apps: []string{c.app},
		})
	}
}
