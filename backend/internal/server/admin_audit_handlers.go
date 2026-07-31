package server

// صفحة سجلّ الأحداث.
//
// كان `audit_log` يُكتب ولا يُقرأ إلا في موضعين ضيّقين: نشاط مستخدم بعينه،
// وسجلّ دخوله. ولا صفحة في اللوحة تقول «ماذا جرى في المنصة اليوم».
//
// **وسجلٌّ لا يُقرأ ليس سجلاً** — هو تكلفةُ كتابةٍ بلا فائدةِ قراءة.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type auditEntry struct {
	ID        int64           `json:"id"`
	ActorID   *string         `json:"actor_id"`
	ActorName *string         `json:"actor_name"`
	Action    string          `json:"action"`
	Entity    string          `json:"entity"`
	EntityID  *string         `json:"entity_id"`
	Details   json.RawMessage `json:"details"`
	IP        *string         `json:"ip"`
	CreatedAt time.Time       `json:"created_at"`
}

// handleAdminAudit سجلّ الأحداث — للأدمن والمالية.
//
// **ومحجوبٌ عن العمليات**: السجلّ يحوي مبالغ التعويضات والسحوبات وأرصدة
// المحافظ، وموظّف العمليات ليس طرفاً في المال. ومن يملك حقّ القراءة هنا يقرأ
// كل قرارٍ ماليّ اتُّخذ في المنصة.
func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 100
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 500 {
		limit = n
	}
	// الترشيح بالبادئة لا بالفعل الكامل: «كل ما فعلته المالية» سؤالٌ أقرب
	// إلى ذهن من يراجع من «كل من صرف طلب سحب».
	prefix := q.Get("prefix") // finance | ops | admin | auth | menu | user

	rows, err := s.pg.Query(r.Context(), `
		SELECT a.id, a.actor_user_id, u.full_name, a.action, a.entity, a.entity_id,
		       a.details, a.ip, a.created_at
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE ($1 = '' OR a.action LIKE $1 || '.%')
		ORDER BY a.id DESC
		LIMIT $2`, prefix, limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []auditEntry{}
	for rows.Next() {
		var e auditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.Action, &e.Entity,
			&e.EntityID, &e.Details, &e.IP, &e.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, e)
	}
	httpx.JSON(w, http.StatusOK, out)
}
