package server

import (
	"net/http"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/httpx"
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

// نصوص الإشعارات المركزية — مصدر واحد لكل نصوص الإشعارات في الخادم.
var notifTitles = struct {
	walletCredit, walletDebit, ratingNew, accountSuspended, accountActivated string
	ticketOpened, ticketNewOps, ticketReply, ticketResolved, driverAssigned  string
	storeClosed, storeReopened, cashSettled, roleGranted, roleRevoked        string
	leadRejected, commissionEarned, passwordReset, sessionsRevoked           string
	payoutRequested, payoutPaid, payoutRejected                              string
}{
	walletCredit:     "إيداع في محفظتك",
	walletDebit:      "خصم من محفظتك",
	ratingNew:        "تقييم جديد على خدمتك",
	accountSuspended: "تم إيقاف حسابك مؤقتاً",
	accountActivated: "تم تفعيل حسابك",
	ticketOpened:     "فُتحت شكواك",
	ticketNewOps:     "شكوى جديدة",
	ticketReply:      "رد جديد على شكواك",
	ticketResolved:   "تم حل شكواك",
	driverAssigned:   "أُسند إليك طلب جديد",
	storeClosed:      "إغلاق طارئ لمتجر",
	storeReopened:    "عاد متجر للعمل",
	cashSettled:      "سُلّم صندوقك النقدي",
	roleGranted:      "أُضيفت صلاحية إلى حسابك",
	roleRevoked:      "سُحبت صلاحية من حسابك",
	leadRejected:     "رُفض طلب انضمام عبر رابطك",
	commissionEarned: "عمولة جديدة في محفظتك",
	passwordReset:    "غُيّرت كلمة مرور حسابك",
	sessionsRevoked:  "أُنهيت جلساتك — سجّل الدخول من جديد",
	payoutRequested:  "طلب سحب رصيد جديد",
	payoutPaid:       "صُرف طلب السحب",
	payoutRejected:   "رُفض طلب السحب",
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
		`SELECT kind, count(*) FROM notifications WHERE user_id = $1 GROUP BY kind`, userIDFrom(r))
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
