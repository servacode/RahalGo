package orders

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
	offerDriver, assignedDriver              string
	newOrderMerchant, newOrderOps            string
	accepted, preparing, onTheWay, delivered string
	// **مفصلا الطلب الخاصّ** — كلٌّ منهما يوقف الطلبَ على فعلِ طرفٍ بعينه،
	// **فمن عليه الدورُ يُخبَر أنّ الدورَ عليه.**
	customQuoted, customConfirmed, driverAssignedToCustomer string
	rejected, cancelled, failed, refunded                   string
	merchantDelivered, commission                           string
	targetReached                                           string
	violationsWarn, violationsBanned                        string
	endedOps, warningIssued                                 string
	// **عناوينُ حركات المحفظة** — (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد
	// يتغيّر وما حدا بيعرف ليش»).
	driverEarned, merchantEarned, refunded2, compensated string
	// **وتبدّلُ السائق يُقال** — (قرارُ المالك ٢٠٢٦-١٠-٠٢).
	driverChanged, newDriverIs string
	// **والسائقُ يُخبَر حين يُؤخذ منه طلبُه** — انظر `driver_lost.go`.
	driverLost string
}{
	offerDriver:      "طلب جديد بانتظارك",
	assignedDriver:   "طلب أُسند إليك",
	newOrderMerchant: "طلب جديد قادم إليك",
	newOrderOps:      "طلب جديد في المنصة",
	accepted:         "قبل المتجر طلبك",
	preparing:        "طلبك قيد التحضير",
	onTheWay:         "طلبك في الطريق إليك",
	delivered:        "تم تسليم طلبك",
	// **ونصٌّ يقول الفعلَ المطلوبَ لا الخبرَ وحدَه** — «سعرٌ جاهز» تُقرأ
	// خبراً فيُؤجَّل، **و«أكّده ليبدأ» تُقرأ طلباً فيُفتح التطبيق.**
	customQuoted:             "سعر طلبك جاهز — أكّده ليبدأ الشراء",
	customConfirmed:          "أكّد الزبون السعر — ابدأ الشراء",
	driverAssignedToCustomer: "أُسند سائقٌ لطلبك",
	rejected:                 "اعتذر المتجر عن طلبك",
	cancelled:                "أُلغي طلبك",
	failed:                   "تعذّر تسليم طلبك",
	refunded:                 "استُرجع مبلغ طلبك",
	merchantDelivered:        "سُلّم طلب من متجرك",
	commission:               "عمولة جديدة في محفظتك",
	targetReached:            "أنجزت هدف الشهر — نالتك مكافأته",
	violationsWarn:           "متجرٌ بلغ حدّ المخالفات",
	violationsBanned:         "حُظر متجرٌ لكثرة الإلغاء",
	endedOps:                 "انتهى طلبٌ قبل تسليمه",
	warningIssued:            "إنذارٌ على متجرك",
	driverEarned:             "أجر توصيل في محفظتك",
	merchantEarned:           "مستحق مبيعاتك في محفظتك",
	refunded2:                "أُعيد المبلغ إلى محفظتك",
	compensated:              "تعويض في محفظتك",
	driverChanged:            "تم تغيير السائق",
	newDriverIs:              "سائقك الجديد: ",
	driverLost:               "طلبٌ لم يعد معك",
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
	number int64
	// **عددُ الأصناف والمبلغ** — يُقرآن في إشعار الطلب الجديد، انظر
	// أدناه. **ولا يُنادى بهما نداءٌ ثانٍ**: هما في الصفّ نفسِه.
	itemCount     int
	subtotal      int64
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
		       COALESCE(mm.name, ''), mm.sales_rep_user_id,
		       -- **عددُ الأصناف والمبلغ** — لإشعار الطلب الجديد.
		       -- **ولا نداءَ ثانٍ لهما**: عدُّ الأسطر أرخصُ من رحلةٍ
		       -- ثانيةٍ إلى القاعدة في مسارٍ يقع مع كلّ طلب.
		       COALESCE((SELECT count(*) FROM order_items oi
		                 WHERE oi.order_id = o.id), 0),
		       o.subtotal
		-- **ويُضمّ يساراً** — (٢٠٢٦-٠٨-٠٩): **وضمٌّ صلبٌ يُسكت إشعاراتِ الطلب
		-- الخاصّ كلَّها** — لا الزبونُ يُخبَر ولا العملياتُ، **ولا خطأ يظهر**:
		-- الدالّةُ تردّ «لا صفوف» فيُبتلع.
		FROM orders o LEFT JOIN merchants mm ON mm.id = o.merchant_id
		WHERE o.id = $1`, orderID).
		Scan(&p.number, &p.customerID, &p.merchantOwner, &p.merchantName, &p.repID,
			&p.itemCount, &p.subtotal)
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
		// ══════════════════════════════════════════════════════════════
		// **والنصُّ يقول ما يكفي للقرار**
		// ══════════════════════════════════════════════════════════════
		//
		// (بلاغُ المالك 2026-08-26: «يوصلو إشعار طلب جديد قادم إليك…
		//  مشان يظلّ المتجرُ على اطّلاع».)
		//
		// **وكان الرقمَ وحدَه `#123`** — ولا يقول كم ولا ماذا.
		// **وصاحبُ المتجر يقرأ الإشعارَ وهو يعمل**، فإن لم يفهمه أجّل
		// فتحَ التطبيق، **والطلبُ له مهلةٌ تنتهي.**
		//
		// **وعددُ الأصناف والمبلغ يكفيان**: يعرف أصغيرٌ هو أم كبير،
		// **فيقرّر أيتركُ ما بيده أم يُكمل.**
		body := ref
		if p.itemCount > 0 {
			body = fmt.Sprintf("%s — %d صنفاً · %d ل.س", ref, p.itemCount, p.subtotal)
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: *p.merchantOwner, Kind: notifications.KindOrder,
			Title: t.newOrderMerchant, Body: body,
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
	// ══════════════════════════════════════════════════════════════════
	// **وإسنادُ السائقِ يُخبَر به الزبون**
	// ══════════════════════════════════════════════════════════════════
	//
	// **للطلب الخاصِّ لا خطوةَ قبولِ متجرٍ قبلَه** — فكان أوّلُ ما يصل الزبونَ
	// «طلبك في الطريق»، **وقد اشترى السائقُ بماله قبلها.** (قُيس على دورةٍ
	// حيّةٍ ٢٠٢٦-٠٩-٢٩.)
	//
	// **والعاديُّ ينتفع به كذلك**: «قبل المتجر» ثمّ صمتٌ حتّى الطريق.
	StAssigned:  t.driverAssignedToCustomer,
	StAccepted:  t.accepted,
	StPreparing: t.preparing,
	StOnTheWay:  t.onTheWay,
	StDelivered: t.delivered,
	StRejected:  t.rejected,
	StCancelled: t.cancelled,
	StFailed:    t.failed,
	StRefunded:  t.refunded,
}

// ══════════════════════════════════════════════════════════════════════
// **وأربعةٌ منها ترنّ ولا تُحفَظ**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢.)
//
// **وهذه الأربعةُ تتكرّر مع كلّ طلب**: قُبل، يُجهَّز، في الطريق، سُلّم.
// **وعشرون طلباً في الشهر تعني ثمانين سطراً في صندوقه** — وما عداها يقع
// مرّةً أو مرّتين. **فالضجيجُ كلُّه منها.**
//
// **والقاعدةُ: أيرجع إليه بعد يومين؟** «طلبك في الطريق» لا — **والبطاقةُ
// تقول حالَ الطلب أصدقَ من خبرٍ قديم.** «تعذّر تسليم طلبك» نعم —
// **يُحتجّ به**: لماذا لم يصلني طلبُ الثلاثاء؟
//
// **والانقطاعاتُ الأربعُ تبقى** (اعتذر، أُلغي، تعذّر، استُرجع): فيها
// مالٌ وفيها سؤالٌ يُطرح بعد أيّام.
var passingTitles = map[string]bool{
	StAccepted:  true,
	StPreparing: true,
	StOnTheWay:  true,
	StDelivered: true,
}

// notifyCustomQuoted **السعرُ وُثّق — والدورُ على الزبون.**
//
// **والطلبُ بعدها لا يتقدّم خطوةً حتّى يؤكّد**: لا شراءَ ولا طريقَ ولا تسليم.
// **فإشعارٌ باقٍ لا عابر** — العابرُ يرنّ ويذهب، **وهذا يُنتظَر فعلُه.**
//
// (بلاغُ المالك ٢٠٢٦-٠٩-٢٩: «مشان ما يفكّر حاله إنه طلب وهو يستنّى بدون ما
//
//	يحصل شي».)
func (s *Service) notifyCustomQuoted(ctx context.Context, orderID string, total int64) {
	if s.notify == nil {
		return
	}
	p, err := s.parties(ctx, orderID)
	if err != nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: p.customerID, Kind: notifications.KindOrder,
		Title: t.customQuoted,
		// **والمبلغُ في النصّ** — من قرأه عرف على ماذا يوافق قبل أن يفتح.
		Body:   fmt.Sprintf("#%d — %s ل.س", p.number, groupDigits(total)),
		Entity: "order", EntityID: orderID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	})
}

// notifyCustomConfirmed **الزبونُ أكّد — والدورُ على السائق.**
//
// **وكان البثُّ وحدَه**، وهو يصل شاشةً مفتوحةً لا جيباً مغلقاً: **فالسائقُ
// ينتظر ولا يعلم أنّ انتظارَه انتهى.**
func (s *Service) notifyCustomConfirmed(ctx context.Context, orderID, driverID string, total int64) {
	if s.notify == nil || driverID == "" {
		return
	}
	var number int64
	if err := s.db.QueryRow(ctx,
		`SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number); err != nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindOrder,
		Title:  t.customConfirmed,
		Body:   fmt.Sprintf("#%d — %s ل.س", number, groupDigits(total)),
		Entity: "order", EntityID: orderID, Href: "/portal",
		// **وتطبيقُ السائق وحدَه يرنّ** — الحسابُ نفسُه قد يكون زبوناً.
		Apps: []string{notifications.AppDriver},
	})
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
	// ══════════════════════════════════════════════════════════════════
	// **وسائقٌ ثانٍ يُقال إنّه ثانٍ** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان الزبونُ يقرأ «أُسند سائقٌ لطلبك» مرّتين** — ولا يعلم أنّ الثانيةَ
	// رجلٌ آخر، **فيكتب لسائقه ما كان يكتبه للأوّل.** **ولا يُقال السبب**:
	// تركه السائقُ أو نزعه المكتبُ أو صمت — شأنُ المنصّة لا الزبون.
	//
	// **والإسنادُ الأوّلُ يبقى بنصّه.**
	if to == StAssigned {
		if changed, name := s.driverChange(ctx, orderID); changed {
			title = t.driverChanged
			body = ref
			if name != "" {
				body = ref + " — " + t.newDriverIs + name
			}
		}
	}
	// **والرابطُ إلى القائمة لا إلى صفحةِ طلبٍ منفردة.**
	//
	// صفحةُ التفاصيل حُذفت — **البطاقةُ صارت تحمل كلَّ ما كان فيها.** وإشعارٌ
	// يفتح صفحةً غيرَ موجودة أسوأُ من إشعارٍ بلا رابط: **يُضغط فيصل إلى لا
	// شيء**، فيُقرأ عطباً في المنصة.
	s.notify.Notify(ctx, notifications.Input{
		UserID: p.customerID, Kind: notifications.KindOrder,
		Title: title, Body: body,
		Entity: "order", EntityID: orderID, Href: "/portal/orders",
		// **«طلبُك في الطريق» يخصّ تطبيقَ الزبون** — ولو كان صاحبُه سائقاً.
		Apps: []string{notifications.AppCustomer},
		// **وتقدّمُ الطلب يرنّ ولا يُحفَظ** — (قرارُ المالك ٢٠٢٦-٠٨-١٢).
		Transient: passingTitles[to],
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

// driverChange **أكان للطلب سائقٌ غيرُ حامله الآن؟** — واسمُ الحامل.
//
// **ومن سجلّ الانتقالات لا من الحديث**: كلُّ إسنادٍ يمرّ بالمحرّك فيُكتب
// حاملُه في حدثه (`order_events.driver_id`)، **والحديثُ قد لا يُكتب فيه سطر.**
// **وسائقٌ يعود إليه الطلبُ نفسُه ليس تبديلاً.**
func (s *Service) driverChange(ctx context.Context, orderID string) (bool, string) {
	var changed bool
	var name string
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM order_events e
		               WHERE e.order_id = o.id AND e.to_status = 'assigned'
		                 AND e.driver_id IS NOT NULL AND e.driver_id <> o.driver_id),
		       COALESCE(u.full_name, '')
		FROM orders o LEFT JOIN users u ON u.id = o.driver_id
		WHERE o.id = $1`, orderID).Scan(&changed, &name); err != nil {
		return false, ""
	}
	return changed, name
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
			Entity: "wallet", EntityID: orderID, Href: "/portal/wallet",
			Apps: []string{c.app},
			// ══════════════════════════════════════════════════════════
			// **ومالُ السائق يُحفَظ ولا يرنّ**
			// ══════════════════════════════════════════════════════════
			//
			// (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا نريد إشعاراتٍ كثيرةً بلا
			//  فائدة».)
			//
			// **وأجرُه يُقيَّد لحظةَ التسليم** — وهو حينئذٍ في التطبيق،
			// يقف عند الباب وقد ضغط «سلّمت» قبل ثانية. **ورنّةٌ تخبره
			// بما فعله للتوّ ضجيجٌ**، ورقمُ محفظته في الشريط فوقه.
			//
			// **ويبقى الصفُّ في صندوقه** — لأنّه مال: يُراجَع بعد يومين
			// **ويُحتجّ به في خلاف.**
			//
			// **وزبونُه يرنّ كما كان**: استرجاعٌ يصله وهو خارج التطبيق،
			// **ومالٌ يعود بلا خبرٍ يُقرأ ضياعا.**
			Silent: c.app == notifications.AppDriver,
		})
	}
}

// ══════════════════════════════════════════════════════════════════════
// **إشعارُ السائق بطلبٍ معروض — وبلاه لا يعمل التطبيقُ أصلاً**
// ══════════════════════════════════════════════════════════════════════
//
// (البندُ الأوّل في قائمة المالك ٢٠٢٦-٠٨-١٢.)
//
// # ولماذا لا يكفي البثُّ الحيّ
//
// **`drivers:queue` تعمل والتطبيقُ مفتوحٌ وحدَه** — وأكثرُ وقت السائق
// شاشتُه مطفأةٌ والجوّالُ في جيبه. **فبلا دفعٍ يفتح تطبيقَه كلَّ دقيقتين**،
// **والطلبُ يفوت لمن رآه أوّلاً.**
//
// # ومن يُشعَر
//
// **في «بالدور» واحدٌ**: الطلبُ معروضٌ عليه هو وله مهلة — **وإشعارُ
// الجميع بطلبٍ لا يستطيعونه إزعاجٌ محض.**
//
// **وفي «للجميع» كلُّ من على وردية**: هو معروضٌ عليهم فعلاً، **ومن سبق
// أخذ.**
//
// # ولا يُشعَر من ليس على وردية
//
// **ورديّةٌ مغلقةٌ تعني «لست في العمل»** — وإشعارٌ يرنّ في بيته ليلاً
// يجعله يُطفئ الإشعاراتِ كلَّها، **فيفقد الطلبَ يومَ يعمل.**
func (s *Service) notifyOffer(ctx context.Context, orderID, driverID string) {
	if s.notify == nil {
		return
	}

	var merchant string
	var number int64
	var cash int64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(m.name, ''), o.number, o.cash_due
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&merchant, &number, &cash); err != nil {
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **وفي «للجميع» لا يرنّ إلّا لمن يرى الطلبَ في طابوره** (٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان يرنّ عند كلّ سائقٍ على ورديّة في المدينة** — قريباً أو بعيداً،
	// مؤهَّلاً أو بلغ سقفَه. **فيفتح البعيدُ طلباً لا يجده في طابوره.** والسؤالُ
	// هو سؤالُ بابِ الطابور نفسُه (`queueAudience`) — **لا شرطٌ ثانٍ يشبهه.**
	targets := []string{}
	if driverID != "" {
		targets = append(targets, driverID)
	} else {
		ids, err := s.queueAudience(ctx, orderID)
		if err != nil {
			s.logger.Error("العرض: تعذّرت قراءةُ من يرى الطلب", "order", orderID, "error", err)
			return
		}
		targets = ids
	}

	title := t.offerDriver
	if driverID != "" && s.directAssign(ctx) {
		title = t.assignedDriver
	}
	// ══════════════════════════════════════════════════════════════════
	// **والعرضُ يرنّ ولا يُحفَظ**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا نريد إشعاراتٍ كثيرةً بلا فائدة».)
	//
	// **وقيس صندوقُ سائقٍ فكان عشرةً من سبعةَ عشرَ سطراً «طلب جديد
	// بانتظارك»** — لطلباتٍ انقضت مهلتُها وذهبت لغيره.
	//
	// **وهو خبرٌ يموت بعد دقيقتين**: العرضُ ينقضي **والسطرُ يبقى أسبوعاً**
	// — فيفتح صندوقَه ليقرأ ما يعنيه فيجد كومةً عن طلباتٍ مضت، **وبينها
	// أجرُ توصيلٍ يعنيه فعلاً.**
	//
	// **ومن وصله عشرةٌ لا يفتح الحادي عشر.**
	//
	// **ولا تُسكَت رنّتُه**: هو أعجلُ ما يصل السائق، **والعرضُ الذي يُقرأ
	// بعد دقيقتين طلبٌ ضائع.** والذي يسقط الصفُّ في القاعدة وحدَه.
	//
	// **والإسنادُ المباشرُ يُحفَظ**: ليس عرضاً ينقضي، **إنّما طلبٌ صار
	// في يده** — ويُسأل عنه غدا.
	//
	// ══════════════════════════════════════════════════════════════════
	// **ولا يُسكَت دفعُه — صفٌّ بنوعٍ لا يُعرَض** (٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان «يرنّ ولا يُحفَظ» يعني عمليّاً «لا يرنّ والتطبيقُ مغلق»**: العابرُ
	// لا صفَّ له، **وعاملُ النقل يقرأ الصفوف.** (قِيس على جهاز المالك.) فصار
	// صفّاً بنوع `order_offer` **يُدفَع عاجلاً ويُحجَب عن الصندوق** — وقرارُ
	// ٢٠٢٦-٠٨-١٤ باقٍ. **وأجلُه مهلةُ العرض** فلا يصل بعد موته، **ومفتاحُ طيّه
	// الطلب** فعرضان له صورةٌ واحدة.
	offer := title == t.offerDriver
	kind := notifications.KindOrder
	var ttl time.Duration
	collapse := ""
	if offer {
		kind = notifications.KindOrderOffer
		ttl = s.offerTTL(ctx, orderID)
		collapse = "offer:" + orderID
	}
	body := merchant
	if cash > 0 {
		body = fmt.Sprintf("%s · تقبض %d", merchant, cash)
	}

	for _, id := range targets {
		s.notify.Notify(ctx, notifications.Input{
			UserID: id, Kind: kind,
			Title: title, Body: body,
			Entity: "order", EntityID: orderID,
			Href: "/portal", // **لوحتُه أيّاً كانت** — حُذفت `/driver` من الويب ٢٠٢٦-٠٨-٢٣
			// **وتطبيقُ السائق وحدَه يرنّ** — الحسابُ نفسُه قد يكون
			// زبوناً، **وطلبُ عملٍ يرنّ في تطبيق الزبون** خبرٌ في غير
			// مكانه.
			Apps: []string{"driver"},
			// **يُدفَع ولا يُعرَض في الصندوق، وله أجل** — انظر أعلاه.
			TTL: ttl, Collapse: collapse,
		})
	}
}

// offerTTL **ما بقي من مهلة العرض** — من الصفّ إن كان معروضاً على أحدٍ
// بعينه، وإلّا مهلةُ العرض كاملة. **ولا أقلَّ من ثانية**: الصفرُ «بلا أجل».
func (s *Service) offerTTL(ctx context.Context, orderID string) time.Duration {
	var left *float64
	_ = s.db.QueryRow(ctx, `
		SELECT EXTRACT(EPOCH FROM (offer_expires_at - now()))
		FROM orders WHERE id = $1 AND offer_expires_at IS NOT NULL`, orderID).Scan(&left)
	if left != nil {
		if *left < 1 {
			return time.Second
		}
		return time.Duration(*left * float64(time.Second))
	}
	if d := s.offerTimeout(ctx); d > 0 {
		return d
	}
	return time.Minute
}

// notifyTargetReached **السائقُ يعرف أنّه بلغ مرحلةً فنال مكافأتَها.**
//
// (قِيس 2026-09-02: المكافأةُ تُقيَّد آليّاً ولا شيءَ يقول له.)
//
// **وهدفٌ لا يُبشَّر ببلوغه لا يحفّز** — يراه السائقُ رقماً زاد في
// محفظته بلا سبب، **فلا يربطه بما فعل.**
func (s *Service) notifyTargetReached(ctx context.Context, driverID string, amount int64) {
	if s.notify == nil || amount <= 0 {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindWallet,
		Title:  t.targetReached,
		Body:   strconv.FormatInt(amount, 10) + " " + currencyWord,
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{notifications.AppDriver},
	})
}

// currencyWord **اسمُ العملة في نصّ إشعار.**
const currencyWord = "ل.س"
