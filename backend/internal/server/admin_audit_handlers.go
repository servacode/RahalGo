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

	// ══════════════════════════════════════════════════════════════════
	// **وتجديدُ الجلسة يُخفى — تكتبه الساعةُ لا الإنسان**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٦، وقياسٌ على خادمه: **أربعةٌ وأربعون من
	//  ثمانين سطراً `auth.refresh`** — خمسةٌ وخمسون بالمئة.)
	//
	// **وأُصلح هذا بعينه في سجلّ نشاط الحساب أمس** — **والطاولةُ نفسُها**،
	// فبقيَ هنا.
	//
	// **ومئتا سطرٍ فيها مئةٌ وعشرة تجديداتٍ لا يفعلها إنسان** — ويُدفَع
	// الفعلُ الحقيقيُّ خارجَ الصفحة.
	//
	// **ومُرشِّحُ `auth` يجمع التجديدَ مع الدخول والخروج وتبديل كلمة
	// السرّ** فلا يفصل. **ولا يُحذف من القاعدة**: أثرُ أمانٍ بعنوانٍ ووقت،
	// يُخفى ويُطلب.
	withRefresh := q.Get("refresh") == "true"

	// ══════════════════════════════════════════════════════════════════
	// **ومدًى بالتاريخ — وسجلٌّ بلا تاريخٍ يُقلَّب لا يُبحَث**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومن سأل «ماذا جرى الأسبوع الماضي؟» لم يكن له بابٌ** إلّا أن يقلّب
	// مئتين مئتين.
	from, to := q.Get("from"), q.Get("to")

	// **والشرطُ واحدٌ للعدّ وللقائمة** — نصّان يفترقان يوماً **فيقول
	// العنوانُ ألفاً وتعرض القائمةُ تسعمئة.**
	const scope = `
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE ($1 = '' OR a.action LIKE $1 || '.%')
		  AND ($2 OR a.action <> 'auth.refresh')
		  AND ($3 = '' OR a.created_at >= $3::date)
		  AND ($4 = '' OR a.created_at < ($4::date + 1))`

	// **وصفحةٌ محدودةٌ بعدّ** — **وهذا أسرعُ ما يُكتب في المنصّة**: كلُّ
	// دخولٍ وتعديلِ إعدادٍ وإنذارٍ وقيدٍ يدويّ. **وسجلٌّ يُقرأ منه آخرُ
	// مئتين ويصمت عن الباقي لمحةٌ باسم سجلّ.**
	var count int
	if err := s.pg.QueryRow(r.Context(), `SELECT count(*)`+scope,
		prefix, withRefresh, from, to).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT a.id, a.actor_user_id, u.full_name, a.action, a.entity, a.entity_id,
		       a.details, a.ip, a.created_at`+scope+`
		ORDER BY a.id DESC
		LIMIT $5 OFFSET $6`, prefix, withRefresh, from, to, limit, (page-1)*limit)
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
	// **والعدُّ والصفحةُ معه** — **وردٌّ مصفوفةٌ مجرّدةٌ لا حقلَ فيه يقول
	// «هناك أكثر».**
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entries": out, "total": count, "page": page, "per_page": limit,
	})
}
