package server

import (
	"net/http"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// نصوص الإشعارات المركزية — مصدر واحد لكل نصوص الإشعارات في الخادم.
var notifTitles = struct {
	walletCredit, walletDebit, ratingNew, accountSuspended, accountActivated string
}{
	walletCredit:     "إيداع في محفظتك",
	walletDebit:      "خصم من محفظتك",
	ratingNew:        "تقييم جديد على خدمتك",
	accountSuspended: "تم إيقاف حسابك مؤقتاً",
	accountActivated: "تم تفعيل حسابك",
}

// صندوق إشعارات المستخدم — لأي دور، فلا أحد يحتاج تحديث الصفحة ليعرف ما استجدّ.

func (s *Server) handleMyNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, unread, err := s.notify.List(r.Context(), userIDFrom(r), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
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
