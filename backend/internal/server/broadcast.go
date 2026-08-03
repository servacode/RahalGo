package server

// **إعلانُ المنصة — أن تخاطب أهلَها.**
//
// # المسألة
//
// كلُّ إشعارٍ في المنصة **يُولد من واقعة**: طلبٌ قُبل، مالٌ دخل، شكوى فُتحت.
// **ولا سبيلَ لقول شيءٍ لا واقعةَ له:**
//
//   - «مغلقون اليومَ للصيانة» · «تأخّرٌ عامٌّ بسبب المطر» · «عرضٌ جديد»
//   - «كلُّ سائقٍ يسلّم صندوقَه اليوم قبل السادسة»
//
// **ومنصّةُ توصيلٍ لا تملك أن تخاطب زبائنَها تُدير أزمتَها بالهاتف** — وواحداً
// واحداً.
//
// # ولماذا بالدور لا بالكلّ
//
// **«الكلُّ» ليست مخاطبةً، هي ضجيج.** رسالةٌ عن تسليم الصناديق تصل زبوناً لا
// شأنَ له بها، **فيتعلّم أن إشعاراتِ المنصة لا تُقرأ** — ثمّ لا يقرأ الذي يهمّه.
//
// # وحارسٌ يمنع الكارثة
//
// **يُعرض العددُ قبل الإرسال ويُطلب تأكيدٌ عليه.** ومن ظنّ أنّه يخاطب ثلاثةَ
// سائقين فوجد ألفَ زبونٍ قد وصلتهم رسالتُه **لا يملك أن يسحبها.**
//
// **ولا يُرسله إلّا الأدمن**: صوتُ المنصة لا يُعار.

import (
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// broadcastRoles الأدوارُ التي يجوز مخاطبتُها — **مرآةُ `user_roles.role_code`.**
var broadcastRoles = map[string]bool{
	"customer": true, "driver": true, "merchant": true, "sales": true,
}

// handleBroadcastCount **كم سيصلهم؟** — يُقرأ قبل الإرسال لا بعده.
func (s *Server) handleBroadcastCount(w http.ResponseWriter, r *http.Request) {
	roles := parseRoles(r.URL.Query().Get("roles"))
	if len(roles) == 0 {
		s.respondErr(w, errValidation)
		return
	}
	var n int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`, roles).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"count": n})
}

// handleBroadcast يُرسل إعلاناً إلى أدوارٍ مختارة.
func (s *Server) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Roles []string `json:"roles"`
		Title string   `json:"title"`
		Body  string   `json:"body"`
		Href  string   `json:"href"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		s.respondErr(w, errValidation)
		return
	}
	roles := []string{}
	for _, x := range req.Roles {
		if broadcastRoles[x] {
			roles = append(roles, x)
		}
	}
	if len(roles) == 0 {
		s.respondErr(w, errValidation)
		return
	}

	// **ويُعدّ قبل الإرسال** — ليُسجَّل في التدقيق كم بلغ، **فمن راجع بعد شهرٍ
	// عرف حجمَ ما أُرسل** لا نصَّه وحدَه.
	var n int
	_ = s.pg.QueryRow(r.Context(), `
		SELECT count(DISTINCT u.id) FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`, roles).Scan(&n)

	s.notify.NotifyRoles(r.Context(), roles, notifications.Input{
		// **`KindAccount` لا `KindOrder`**: إعلانُ منصّةٍ ليس خبرَ طلب،
		// **ومن رشّح إشعاراتِ الطلبات لا يريد أن يجده بينها.**
		Kind:  notifications.KindAccount,
		Title: title,
		Body:  clip(strings.TrimSpace(req.Body), 500),
		Href:  strings.TrimSpace(req.Href),
	})

	s.audit(r, "ops.broadcast", "notification", "", map[string]any{
		"roles": roles, "title": title, "reached": n,
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": n})
}

// parseRoles يفكّ «customer,driver» ويُسقط ما ليس دوراً مسموحاً.
//
// **والإسقاطُ صامتٌ عمداً**: من كتب دوراً لا يُخاطَب لا نُرسل له، **ولا نُفشل
// إرسالاً صحيحاً لأجل كلمةٍ زائدة.**
func parseRoles(raw string) []string {
	out := []string{}
	for _, x := range strings.Split(raw, ",") {
		x = strings.TrimSpace(x)
		if broadcastRoles[x] {
			out = append(out, x)
		}
	}
	return out
}
