package server

// صفحة سجلّ الأحداث.
//
// كان `audit_log` يُكتب ولا يُقرأ إلا في موضعين ضيّقين: نشاط مستخدم بعينه،
// وسجلّ دخوله. ولا صفحة في اللوحة تقول «ماذا جرى في المنصة اليوم».
//
// **وسجلٌّ لا يُقرأ ليس سجلاً** — هو تكلفةُ كتابةٍ بلا فائدةِ قراءة.
//
// # ما تغيّر (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
//
//   - **يراه كلُّ من ملك `audit.read`** — والمبالغُ تُحذف في الخادم عمّن ليس
//     طرفاً في المال (`canSeeAuditMoney`). كان التعليقُ يقول «للأدمن
//     والمالية» والواقعُ غيرُه.
//   - **التبويبُ الافتراضيُّ «الأفعال الحساسة»** والدخولُ في تبويبه.
//   - **بحثٌ** (اسمٌ · هاتفٌ · رقمُ طلبٍ أو تذكرة · اسمُ متجر · فعل)
//     **وفلترٌ بالشخص وبالفعل**.
//   - **التاريخُ بيوم دمشق** لا بيوم القاعدة.
//   - **الهدفُ يُقرأ**: رقمُ الطلب واسمُ المتجر واسمُ المستخدم.
//   - **عدٌّ محدود**: لا يُعَدّ الجدولُ كلُّه عند كلّ فتح.
//   - **تصديرٌ يُكتب في السجلّ نفسِه.**
//   - **والتصديرُ لمدير المنصّة ومالكها وحدَهما** (`audit.export` — قرارٌ ثانٍ
//     في اليوم نفسِه)، **والعرضُ يبقى لكلّ من ملك `audit.read`** — والماليّةُ
//     منهم منذ الهجرة `0260`.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

type auditEntry struct {
	ID          int64           `json:"id"`
	ActorID     *string         `json:"actor_id"`
	ActorName   *string         `json:"actor_name"`
	ActorPhone  *string         `json:"actor_phone"`
	ActorRoles  []string        `json:"actor_roles"`
	Action      string          `json:"action"`
	Entity      string          `json:"entity"`
	EntityID    *string         `json:"entity_id"`
	TargetLabel string          `json:"target_label"`
	TargetNum   *int64          `json:"target_number"`
	Details     json.RawMessage `json:"details"`
	IP          *string         `json:"ip"`
	UserAgent   string          `json:"user_agent"`
	CreatedAt   time.Time       `json:"created_at"`
	// Redacted **حُذف منه مبلغٌ** — فتقول الشاشةُ «المبلغ مخفي» لا «لا مبلغ».
	Redacted bool `json:"redacted"`
}

// auditCountCap **سقفُ العدّ** (المشكلةُ العاشرة).
//
// كان كلُّ فتحٍ يعدّ الجدولَ كلَّه — **وهو أسرعُ جداول المنصّة نموّاً.**
// والعدُّ هنا لا يتجاوز عشرةَ آلاف؛ وما فوقها تقول الشاشةُ «+١٠٠٠٠»، ومن
// أراد أدقَّ ضيّق التاريخ.
const auditCountCap = 10000

// auditExportCap سقفُ سطور التصدير الواحد — **وما زاد يُقال لا يُقصّ صامتاً.**
const auditExportCap = 20000

// sqlArgs بانٍ صغيرٌ للشروط — **كلُّ قيمةٍ معلَمةٌ لا نصٌّ يُلصق.**
type sqlArgs struct {
	conds []string
	args  []any
}

func (b *sqlArgs) arg(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

func (b *sqlArgs) where(c string) { b.conds = append(b.conds, c) }

func (b *sqlArgs) clause() string {
	if len(b.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.conds, " AND ")
}

// likeEscape يُبطل محارفَ `LIKE` في نصّ الباحث — `%` ليس «أيَّ شيء» هنا.
func likeEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// auditFilter **مرشّحاتُ الصفحة والتصدير — نصٌّ واحد.**
//
// **والشرطُ واحدٌ للعدّ وللقائمة وللتصدير** — نصّان يفترقان يوماً **فيقول
// العنوانُ ألفاً وتعرض القائمةُ تسعمئة، ويحمل الملفُّ ثالثاً.**
func auditFilter(r *http.Request) (*sqlArgs, map[string]string, error) {
	q := r.URL.Query()
	b := &sqlArgs{}
	echo := map[string]string{}

	// ── التبويب ──────────────────────────────────────────────────────
	group := q.Get("group")
	if group == "" {
		group = q.Get("prefix") // الاسمُ القديم
	}
	if group == "" {
		group = "sensitive"
	}
	g, ok := auditGroupByKey(group)
	if !ok {
		return nil, nil, errValidation
	}
	echo["group"] = g.Key
	switch {
	case g.Key == "sensitive":
		b.where("(a.action LIKE 'finance.%' OR a.action = ANY(" + b.arg(auditSensitiveActions()) +
			") OR (a.action = 'admin.setting_update' AND a.entity_id = ANY(" +
			b.arg(auditSensitiveSettingKeys()) + ")))")
	case len(g.Prefixes) > 0:
		b.where("split_part(a.action, '.', 1) = ANY(" + b.arg(g.Prefixes) + ")")
	}

	// ── وتجديدُ الجلسة مخفيٌّ إلّا بطلب ─────────────────────────────
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٦: ٤٤ من ٨٠ سطراً `auth.refresh` تكتبه
	//  الساعةُ لا الإنسان.)
	if q.Get("refresh") != "true" {
		b.where("a.action <> 'auth.refresh'")
	}

	// ── الفعلُ بعينه ───────────────────────────────────────────────
	if a := strings.TrimSpace(q.Get("action")); a != "" {
		b.where("a.action = " + b.arg(a))
		echo["action"] = a
	}
	// ── الشخصُ بعينه ───────────────────────────────────────────────
	if who := q.Get("actor"); who != "" {
		if who == "system" {
			b.where("a.actor_user_id IS NULL")
		} else {
			if !isUUID(who) {
				return nil, nil, errValidation
			}
			b.where("a.actor_user_id = " + b.arg(who) + "::uuid")
		}
		echo["actor"] = who
	}
	// ── الهدفُ بعينه (رابطٌ من صفحةٍ أخرى) ───────────────────────────
	if ent := q.Get("entity"); ent != "" {
		b.where("a.entity = " + b.arg(ent))
		echo["entity"] = ent
	}
	if eid := q.Get("entity_id"); eid != "" {
		b.where("a.entity_id = " + b.arg(eid))
		echo["entity_id"] = eid
	}

	// ── التاريخُ بيوم دمشق (المشكلةُ التاسعة) ───────────────────────
	//
	// كان `created_at >= $3::date` — **بمنطقة القاعدة لا بمنطقة المنصّة**،
	// فأحداثُ ما بين منتصف الليل والثالثة تقع في اليوم الخطأ.
	loc := platform.Location()
	if from := q.Get("from"); from != "" {
		t, err := time.ParseInLocation("2006-01-02", from, loc)
		if err != nil {
			return nil, nil, errValidation
		}
		b.where("a.created_at >= " + b.arg(t))
		echo["from"] = from
	}
	if to := q.Get("to"); to != "" {
		t, err := time.ParseInLocation("2006-01-02", to, loc)
		if err != nil {
			return nil, nil, errValidation
		}
		b.where("a.created_at < " + b.arg(t.AddDate(0, 0, 1)))
		echo["to"] = to
	}

	// ── البحث (المشكلةُ السادسة) ─────────────────────────────────────
	//
	// اسمُ الفاعل أو هاتفُه · رقمُ طلبٍ أو تذكرة · اسمُ المتجر أو المستخدم
	// الهدف · رمزُ الفعل — **و`q_actions` أفعالٌ طابق اسمُها العربيُّ ما
	// كُتب** (الأسماءُ في معجم الويب لا هنا).
	if raw := strings.TrimSpace(q.Get("q")); raw != "" {
		if len([]rune(raw)) > 100 {
			raw = string([]rune(raw)[:100])
		}
		echo["q"] = raw
		like := b.arg("%" + likeEscape(raw) + "%")
		ors := []string{
			"u.full_name ILIKE " + like,
			"a.action ILIKE " + like,
			"a.entity_id = " + b.arg(raw),
			"(a.entity = 'merchant' AND a.entity_id IN (SELECT m.id::text FROM merchants m WHERE m.name ILIKE " + like + "))",
			"(a.entity = 'user' AND a.entity_id IN (SELECT tu.id::text FROM users tu WHERE tu.full_name ILIKE " + like + "))",
		}
		// **الهاتفُ بأرقامه** — `0944…` و`+963944…` رقمٌ واحد. **وسبعةُ أرقامٍ
		// على الأقلّ**: رقمُ طلبٍ قصيرٌ («١٢٣٤») لا يُقرأ قطعةَ هاتفٍ فيجرّ
		// أفعالَ كلِّ من في رقمه «١٢٣٤».
		digits := strings.Map(func(c rune) rune {
			if c >= '0' && c <= '9' {
				return c
			}
			if c >= '٠' && c <= '٩' {
				return '0' + (c - '٠')
			}
			return -1
		}, raw)
		if trimmed := strings.TrimLeft(digits, "0"); len(trimmed) >= 7 {
			ph := b.arg("%" + trimmed + "%")
			ors = append(ors,
				"u.phone::text LIKE "+ph,
				"(a.entity = 'user' AND a.entity_id IN (SELECT tu.id::text FROM users tu WHERE tu.phone::text LIKE "+ph+"))")
		}
		// **رقمُ الطلب أو التذكرة** — كما يقوله الموظّفُ «#١٢٣٤».
		bare := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
		allDigits := bare != "" && len([]rune(bare)) == len(digits)
		if n, err := strconv.ParseInt(digits, 10, 64); err == nil && n > 0 && allDigits {
			num := b.arg(n)
			ors = append(ors,
				"(a.entity = 'order' AND a.entity_id IN (SELECT o.id::text FROM orders o WHERE o.number = "+num+"))",
				"(a.entity = 'ticket' AND a.entity_id IN (SELECT tk.id::text FROM tickets tk WHERE tk.number = "+num+"))",
				"(a.details->>'order_id') IN (SELECT o.id::text FROM orders o WHERE o.number = "+num+")")
		}
		if qa := q.Get("q_actions"); qa != "" {
			var acts []string
			for _, a := range strings.Split(qa, ",") {
				if a = strings.TrimSpace(a); a != "" && len(acts) < 50 {
					acts = append(acts, a)
				}
			}
			if len(acts) > 0 {
				ors = append(ors, "a.action = ANY("+b.arg(acts)+")")
			}
		}
		b.where("(" + strings.Join(ors, " OR ") + ")")
	}
	return b, echo, nil
}

// auditSelect **السطرُ كما يُعرض** — بالفاعل ودوره، وبالهدف مقروءاً.
//
// (المشكلةُ الرابعة: «تغيير حالة طلب — أحمد» بلا أيّ طلب.)
const auditSelect = `
	SELECT a.id, a.actor_user_id::text, NULLIF(u.full_name, ''), u.phone::text,
	       COALESCE((SELECT array_agg(ur.role_code ORDER BY ur.role_code)
	                 FROM user_roles ur WHERE ur.user_id = a.actor_user_id), '{}'),
	       a.action, a.entity, NULLIF(a.entity_id, ''),
	       COALESCE(CASE WHEN a.entity_id ~ '^[0-9a-fA-F-]{36}$' THEN
	           CASE a.entity
	           WHEN 'merchant' THEN (SELECT m.name FROM merchants m WHERE m.id = a.entity_id::uuid)
	           WHEN 'user' THEN (SELECT COALESCE(NULLIF(tu.full_name, ''), tu.phone::text)
	                             FROM users tu WHERE tu.id = a.entity_id::uuid)
	           END END, ''),
	       CASE WHEN a.entity_id ~ '^[0-9a-fA-F-]{36}$' THEN
	           CASE a.entity
	           WHEN 'order' THEN (SELECT o.number FROM orders o WHERE o.id = a.entity_id::uuid)
	           WHEN 'ticket' THEN (SELECT tk.number FROM tickets tk WHERE tk.id = a.entity_id::uuid)
	           END END,
	       a.details, NULLIF(a.ip, ''), a.user_agent, a.created_at
	FROM audit_log a
	LEFT JOIN users u ON u.id = a.actor_user_id`

func (s *Server) scanAudit(r *http.Request, sql string, args []any) ([]auditEntry, error) {
	rows, err := s.pg.Query(r.Context(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	money := s.canSeeAuditMoney(r)
	out := []auditEntry{}
	for rows.Next() {
		var e auditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.ActorPhone, &e.ActorRoles,
			&e.Action, &e.Entity, &e.EntityID, &e.TargetLabel, &e.TargetNum,
			&e.Details, &e.IP, &e.UserAgent, &e.CreatedAt); err != nil {
			return nil, err
		}
		if !money {
			eid := ""
			if e.EntityID != nil {
				eid = *e.EntityID
			}
			e.Details, e.Redacted = redactAuditMoney(e.Action, eid, e.Details)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// handleAdminAudit سجلّ الأحداث — **لكلّ من ملك `audit.read`**.
//
// **والمبالغُ لمن ملك قراءةَ المال أو كان أدمن المنصّة** — وتُحذف لغيره
// هنا في الخادم (قرارُ المالك الأوّل ٢٠٢٦-١٠-٠٤).
func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 200 {
		limit = n
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	b, echo, err := auditFilter(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **عدٌّ محدودٌ لا عدُّ الجدول** (المشكلةُ العاشرة).
	var count int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM (SELECT 1 FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id`+b.clause()+`
		LIMIT `+strconv.Itoa(auditCountCap+1)+`) c`, b.args...).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	capped := count > auditCountCap
	if capped {
		count = auditCountCap
	}

	lim, off := b.arg(limit), b.arg((page-1)*limit)
	out, err := s.scanAudit(r, auditSelect+b.clause()+`
		ORDER BY a.id DESC LIMIT `+lim+` OFFSET `+off, b.args)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entries": out, "total": count, "total_capped": capped,
		"page": page, "per_page": limit, "group": echo["group"],
		"money_visible": s.canSeeAuditMoney(r),
		// **زرُّ التصدير يظهر لمن يملكه وحدَه** — والخادمُ يرفض غيرَه على أيّ حال.
		"can_export": s.hasCapability(r, authz.AuditExport),
	})
}

// handleAdminAuditActors **مَن فعل شيئاً في آخر تسعين يوماً** — لفلتر الشخص.
//
// **ولا دخولَ ولا تجديد**: من دخل ولم يفعل شيئاً ليس فاعلاً يُبحث عنه.
func (s *Server) handleAdminAuditActors(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT a.actor_user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''),
		       count(*)
		FROM audit_log a
		JOIN users u ON u.id = a.actor_user_id
		WHERE a.created_at > now() - interval '90 days'
		  AND NOT (a.action = ANY ($1))
		GROUP BY 1, 2
		ORDER BY 3 DESC
		LIMIT 300`, auditSessionActions)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type actor struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	out := []actor{}
	for rows.Next() {
		var a actor
		if err := rows.Scan(&a.ID, &a.Name, &a.Count); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"actors": out})
}

// handleAdminAuditExport **تصديرُ ما يراه** — بالمرشّحات نفسِها.
//
// (قرارُ المالك السادس: «تصدير لمن معه الصلاحية، والتصدير نفسه ينكتب
// بالسجل».) **والصلاحيّةُ `audit.export`** — للأدمن والمالك وحدَهما.
//
// **والتصديرُ يُقيَّد قبل أن يُسلَّم** — في المعاملة نفسِها لا في الخلفيّة:
// ملفٌّ خرج بلا أثرٍ هو بعينه ما وُجد السجلُّ ليمنعه. فإن تعذّر القيدُ
// لم يخرج الملفّ.
//
// **ويُسلَّم سطوراً لا ملفّاً**: أسماءُ الأفعال العربيّة في معجم الويب
// وحدَه، فالشاشةُ تبني الملفَّ بعناوين عربيّة وتوقيت دمشق وحمايةٍ من حقن
// الصيغ. **والمبالغُ محذوفةٌ هنا كما في الصفحة.**
func (s *Server) handleAdminAuditExport(w http.ResponseWriter, r *http.Request) {
	// **والقدرةُ تُفحص هنا أيضاً لا في الجدول وحدَه** (قرارُ المالك 2026-10-04:
	// التصديرُ للأدمن والمالك فقط) — فمعالِجٌ يُركَّب غداً خلف وسيطٍ آخر لا
	// يُخرج السجلَّ ملفّاً لمن يقرؤه فقط.
	if !s.hasCapability(r, authz.AuditExport) {
		s.respondErr(w, errForbidden)
		return
	}
	b, echo, err := auditFilter(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	lim := b.arg(auditExportCap + 1)
	out, err := s.scanAudit(r, auditSelect+b.clause()+`
		ORDER BY a.id DESC LIMIT `+lim, b.args)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	truncated := len(out) > auditExportCap
	if truncated {
		out = out[:auditExportCap]
	}
	meta := map[string]any{"rows": len(out), "truncated": truncated,
		"money_visible": s.canSeeAuditMoney(r)}
	for k, v := range echo {
		meta[k] = v
	}
	if err := s.auditTx(r.Context(), s.pg, r, "admin.audit_exported", "audit", "", meta); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entries": out, "truncated": truncated, "cap": auditExportCap,
	})
}
