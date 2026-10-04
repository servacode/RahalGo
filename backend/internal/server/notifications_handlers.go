package server

import (
	"net/http"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// touch يبثّ إشارة تحديث صامتة: الشاشات المفتوحة تعيد جلب بياناتها فوراً بلا
// صفّ إشعار في صندوق أحد. القاعدة الفاصلة:
//   - إشعار (s.notify) = شخص يحتاج أن *يعلم*.
//   - touch            = شاشة تحتاج أن *تتحدّث*.
//
// entity هو نوع الحدث الذي تشترك به الواجهة: useLiveRefresh(["zone"], load).
func (s *Server) touch(entity string, topics ...string) {
	if len(topics) == 0 {
		topics = []string{"ops"}
	}
	event := map[string]any{"type": entity}
	for _, t := range topics {
		s.hub.Publish(t, event)
	}
}

// touchUser يبثّ إشارةَ تحديثٍ **إلى صاحب الشأن نفسِه** — لا إلى غرفة العمليات.
//
// # المسألة
//
// كلُّ مواضع تغيير الرصيد كانت تبثّ `touch("wallet", "ops")` **وحدَها**: تتحدّث
// لوحةُ المنصة، **ولا يصل صاحبَ المال خبر.** فيُعوَّض الزبونُ ثمّ ينظر إلى
// رصيده فيجده كما كان — **ويُحدّث الصفحةَ بيده أو يظنّ التعويضَ لم يقع.**
//
// **وشهده المالكُ** (٢٠٢٦-٠٨-٠٣): «تمّ التعويض ولكن المحفظة لم تتحدّث بشكل
// فوري».
//
// **وموضوعُ `user:` يشترك فيه كلُّ مستخدمٍ مهما كان دورُه** — فهو الباب الذي
// لا يبقى أحدٌ خارجَه.
func (s *Server) touchUser(userID string, entities ...string) {
	if userID == "" {
		return
	}
	for _, e := range entities {
		s.hub.Publish("user:"+userID, map[string]any{"type": e})
	}
}

// currencyWord **اسمُ العملة في نصّ إشعار** — ولا يُكتب في موضعين.
const currencyWord = "ل.س"

// نصوص الإشعارات المركزية — مصدر واحد لكل نصوص الإشعارات في الخادم.
var notifTitles = struct {
	walletCredit, walletDebit, ratingNew, accountSuspended, accountActivated string
	ticketOpened, ticketNewOps, ticketReply, ticketResolved, driverAssigned  string
	// **شكوى تعويضُها بانتظار المالية · ورفضته المالية فعادت إلى الدعم** (٢٠٢٦-١٠-٠٤).
	ticketAwaitingFinance, complaintCompRejected string
	// driverTransferred **تحوّل طلبُه إلى متجرٍ آخر** — فيتّجه إليه (٢٠٢٦-١٠-٠٣).
	driverTransferred                                                 string
	storeClosed, storeReopened, cashSettled, roleGranted, roleRevoked string
	leadRejected, commissionEarned, passwordReset, sessionsRevoked    string
	// targetReached **بلغ مرحلةً من هدفه الشهريّ فنال مكافأتها.**
	//
	// **والمالُ يُقيَّد في محفظته آليّاً** ولا شيءَ يقول له — **فيراه
	// رقماً زاد بلا سبب.** وهدفٌ لا يُبشَّر ببلوغه لا يحفّز.
	targetReached                               string
	payoutRequested, payoutPaid, payoutRejected string
	// **ولكلّ حالٍ من أحوال السحب عنوانُها** (قسمُ طلبات السحب ٢٠٢٦-١٠-٠٤) —
	// كان الفشلُ والارتدادُ وقيدُ الصرف تُعنوَن «صُرف طلب السحب».
	payoutProcessing, payoutFailed, payoutReversed string
	warningIssued, driverEmergency                 string
	// **سائقٌ ترك طلباً قبل الاستلام بسبب** — ودوامُه أُغلق (مساءَ ٢٠٢٦-١٠-٠٢).
	driverReleased string
	// **رسالةٌ في حديث الطلب — والعنوانُ يقول من كتب لا ماذا كتب.**
	//
	// **ولا رقمَ ولا اسمَ شخصٍ في العنوان**: يُقرأ الإشعارُ على شاشةٍ مقفلة،
	// **واسمٌ يظهر هناك يُعرّف بمن لا يُراد تعريفُه.**
	messageFromDriver, messageFromCustomer string
	// **إنذارٌ عليك** — لأيّ دور.
	//
	// **و`warningIssued` القديمُ للمتاجر وحدَه** ونصُّه «إنذارٌ على متجرك»
	// — **ولا يصلح لسائقٍ ولا لزبون.**
	warningOnYou string
	// emergencyResolved **بلاغُ طوارئك عولج** — (قرارُ المالك ٢٠٢٦-٠٨-١٦).
	//
	// **ومن ضغط الزرَّ ينتظر** — **وانتظارٌ بلا جوابٍ يُقرأ إهمالاً**،
	// ومن قرأه لا يضغط ثانيةً.
	emergencyResolved string
	// **وشكوى فُتحت عليك** — يعرفها من هي عليه لا من فتحها وحدَه.
	complaintOnYou string
	// **وما اتُّفق عليه في الطلب الخاصّ** — يبقى مكتوباً حيث يراه صاحبُه.
	customAgreed string
	// offensiveText **لفظٌ مسيءٌ أُخفي في حديثٍ أو شكوى** — للإدارة (٢٠٢٦-١٠-٠٣).
	offensiveText string
	// **قسمُ الحسابات** (قراراتُ المالك ٢٠٢٦-١٠-٠٤): طلبُ حركةٍ ينتظر الماليّة ·
	// تغييرُ رقمٍ ينتظر شخصاً ثانياً · حظرُ المتجر ورفعُه · السائقُ بخيرٍ بعد حادث ·
	// وإنذاراتٌ بلغت الحدَّ تنبّه الموظّفين ولا توقف أحداً.
	walletRequest, phoneChangeRequest, storeBanned, storeUnbanned string
	driverCleared, warningsThreshold                              string
	// **مكافأةٌ أو عقوبةٌ يدويّةٌ وافقت عليها الماليّة** — والسببُ في المتن
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الأهداف).
	incentiveReward, incentivePenalty string
}{
	incentiveReward:       "مكافأة من الإدارة في محفظتك",
	incentivePenalty:      "عقوبة من الإدارة على حسابك",
	walletRequest:         "طلب حركة على محفظة ينتظر موافقة المالية",
	phoneChangeRequest:    "طلب تغيير رقم حساب ينتظر موافقة ثانية",
	storeBanned:           "حُظر متجرك",
	storeUnbanned:         "رُفع الحظر عن متجرك",
	driverCleared:         "يمكنك فتح دوامك الآن — تأكدت الإدارة أنك بخير",
	warningsThreshold:     "حساب بلغ حد الإنذارات — يحتاج قرار موظف",
	offensiveText:         "لفظ مسيء في حديث أو شكوى — أُخفي عن الطرف الآخر",
	warningOnYou:          "إنذار على حسابك",
	emergencyResolved:     "تابعنا بلاغَ الطوارئ الخاصّ بك",
	complaintOnYou:        "شكوى على خدمتك",
	messageFromDriver:     "رسالة من السائق",
	messageFromCustomer:   "رسالة من الزبون",
	warningIssued:         "إنذارٌ على متجرك",
	driverEmergency:       "طارئٌ لدى سائق",
	driverReleased:        "سائقٌ ترك طلباً قبل الاستلام — ودوامُه أُغلق",
	walletCredit:          "إيداع في محفظتك",
	walletDebit:           "خصم من محفظتك",
	ratingNew:             "تقييم جديد على خدمتك",
	accountSuspended:      "تم إيقاف حسابك مؤقتاً",
	accountActivated:      "تم تفعيل حسابك",
	ticketOpened:          "فُتحت شكواك",
	ticketNewOps:          "شكوى جديدة",
	ticketReply:           "رد جديد على شكواك",
	ticketResolved:        "تم حل شكواك",
	ticketAwaitingFinance: "شكواك بانتظار قرار المالية بالتعويض",
	complaintCompRejected: "رفضت المالية تعويض شكوى — عادت إلى الدعم",
	driverAssigned:        "أُسند إليك طلب جديد",
	driverTransferred:     "تحوّل طلبك إلى متجر آخر",
	targetReached:         "أنجزت هدف الشهر — نالتك مكافأته",
	storeClosed:           "إغلاق طارئ لمتجر",
	storeReopened:         "عاد متجر للعمل",
	cashSettled:           "سُلّم صندوقك النقدي",
	roleGranted:           "أُضيفت صلاحية إلى حسابك",
	roleRevoked:           "سُحبت صلاحية من حسابك",
	leadRejected:          "رُفض طلب انضمام عبر رابطك",
	commissionEarned:      "عمولة جديدة في محفظتك",
	passwordReset:         "غُيّرت كلمة مرور حسابك",
	sessionsRevoked:       "أُنهيت جلساتك — سجّل الدخول من جديد",
	payoutRequested:       "طلب سحب رصيد جديد",
	payoutPaid:            "صُرف طلب السحب",
	payoutRejected:        "رُفض طلب السحب",
	payoutProcessing:      "طلب سحبك قيد الصرف",
	payoutFailed:          "تعذّر صرف طلب السحب",
	payoutReversed:        "ارتدّ طلب السحب",
}

// صندوق إشعارات المستخدم — لأي دور، فلا أحد يحتاج تحديث الصفحة ليعرف ما استجدّ.

func (s *Server) handleMyNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, unread, err := s.notify.List(r.Context(), userIDFrom(r), limit, r.URL.Query().Get("kind"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// عدّاد لكل نوع — تبنى عليه أزرار الترشيح في صفحة الإشعارات
	counts := map[string]int{}
	rows, err := s.pg.Query(r.Context(),
		// **ولا يُعَدّ نوعٌ لا يُعرَض** — **وصفُّ المحادثة حدثُ نقلٍ
		// لا خبرٌ في صندوق** (`notifications.KindChat`).
		`SELECT kind, count(*) FROM notifications
		  WHERE user_id = $1 AND kind <> ALL($2::text[]) GROUP BY kind`,
		userIDFrom(r), notifications.InboxHidden())
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var n int
			if rows.Scan(&k, &n) == nil {
				counts[k] = n
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread, "counts": counts})
}

// handleMarkNotificationRead يعلّم إشعاراً مقروءاً — أو الكل عند تمرير id فارغ.
func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		ID string `json:"id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.notify.MarkRead(r.Context(), userIDFrom(r), req.ID); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
