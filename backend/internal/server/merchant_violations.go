package server

// حظرُ المتجر كثيرِ الإلغاء — ورفعُه والعفوُ عنه.
//
// # لماذا الحظر أصلاً
//
// **الثقةُ تُبنى مرّةً وتُهدَم مرّة.** والمنصةُ تتحمّل الخسارة عن طلبٍ أو
// طلبين لأنها لا تريد أن تفقد ثقةَ زبونٍ بسببٍ تافه — لكنّ متجراً يجعل ذلك
// عادةً يُكلّفها ما هو أغلى من الطلبات: **سمعتَها.**
//
// # ولماذا العفوُ خطٌّ لا ممحاة
//
// المخالفاتُ طلباتٌ وقعت فعلاً، ولا يُمحى وقوعُها من السجلّ. **فالعفوُ يُحرّك
// خطّاً زمنياً**: لا يُعدّ إلّا ما بعده. ويبقى الماضي مقروءاً لمن يسأل «كم
// مرّةً سامحناه؟» — **وهو سؤالٌ يُطرح حين يُطلب العفوُ رابعةً.**
//
// # ورفعُ الحظر لا يعني العفو
//
// **فعلان لا فعلٌ واحد**، وفصلُهما مقصود:
//
//   - **رفعُ الحظر وحده**: يعود يعمل **وعدّادُه كما هو** — فمخالفةٌ واحدة
//     تعيده محظوراً. وهذه رسالةٌ صريحة: «أعدناك على وعد».
//   - **العفو**: يُصفَّر العدّاد. صفحةٌ جديدة.
//
// ودمجُهما في زرٍّ واحد يجعل كلَّ رفعِ حظرٍ عفواً — **فيتعلّم المتجرُ أن
// الإلغاء بلا ثمن.**

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMerchantViolations عدّادُ المخالفات وحدُّها — لتراه العملياتُ قبل أن يبلغ.
//
// **الظهورُ قبل البطش**: متجرٌ على ٤ من ٥ تتّصل به العملياتُ فتنقذ الطرفين،
// وحظرٌ يقع فجأةً يُفاجئ من لم يكن يعلم أن هناك عدّاداً.
func (s *Server) handleMerchantViolations(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	n, err := s.orders.MerchantViolations(r.Context(), s.pg, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والوقائعُ مع العدد** — لا عددٌ مجرّدٌ يُقرّر عليه حظرٌ أو عفو.
	//
	// **وتعذّرُ القائمة لا يُسقط العدّاد**: من فتح الملفَّ ليرى أهو على ٤ من ٥
	// يجب أن يرى الرقمَ ولو لم تُقرأ التفاصيل.
	list, err := s.orders.MerchantViolationList(r.Context(), s.pg, id)
	if err != nil {
		list = nil
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"violations": n,
		"items":      list,
		"limit":      s.settings.GetInt(r.Context(), "merchants.cancel_ban_count"),
		"days":       s.settings.GetInt(r.Context(), "merchants.cancel_ban_days"),
		"mode":       s.settings.GetString(r.Context(), "merchants.cancel_ban_mode", "manual"),
	})
}

// handleSuspendMerchant حظرٌ يدويّ أو رفعُه.
//
// **`suspended` لا `inactive`**: الثانيةُ يملكها المتجر — إجازةٌ أو ترميم —
// **ولو حُظر بها لرفع الحظرَ عن نفسه من بوابته. وحظرٌ يرفعه المحظور ليس حظراً.**
func (s *Server) handleSuspendMerchant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		Suspended bool   `json:"suspended"`
		Note      string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **رفعُ الحظر يعيده `active` لا إلى ما كان.** ولو حُفظت حالتُه السابقة
	// وأُعيدت لعاد متجرٌ حُظر وهو مُغلَقٌ إلى الإغلاق — فيظنّ أن الحظر باقٍ.
	status := "active"
	if req.Suspended {
		status = "suspended"
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE merchants SET status = $2 WHERE id = $1`, id, status); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.merchant_suspend", "merchant", id, map[string]any{
		"suspended": req.Suspended, "note": req.Note,
	})
	s.touch("merchant", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status})
}

// handleClearViolations عفوٌ — يُصفَّر العدّاد ولا يُمحى الماضي.
func (s *Server) handleClearViolations(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE merchants SET violations_cleared_at = now() WHERE id = $1`, id); err != nil {
		s.respondErr(w, err)
		return
	}
	// **يُسجَّل**: العفوُ قرارٌ يُعاد النظر فيه حين يُطلب مرّةً أخرى.
	s.audit(r, "ops.merchant_violations_cleared", "merchant", id, nil)
	s.touch("merchant", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"cleared": true})
}

// handleTreasuryCandidates حساباتٌ تصلح لحمل خزينة المنصة.
//
// **نقطةٌ مخصّصة لا ثقبٌ في قائمة المستخدمين**: تلك تستبعد الأدمن عمداً
// (`HAVING NOT bool_or(role_code = 'admin')`) — وتوسيعُها لأجل قائمةٍ واحدة
// **يُضعف حارساً قائماً لأجل راحةٍ عابرة.**
//
// **والأدمن والمالية وحدهما**: خزينةٌ على حساب سائقٍ ليست خطأً يُكتشف، هي
// **مالٌ يُقيَّد لمن لا يخصّه** — ومن يفتح قائمةً يختار من فيها.
//
// وللأدمن وحده: **من يختار حاملَ الخزينة يملك مالَ المنصة كلَّه.**
func (s *Server) handleTreasuryCandidates(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT DISTINCT u.id, COALESCE(NULLIF(u.full_name, ''), u.phone::text), u.phone::text
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code IN ('admin', 'finance') AND u.status = 'active'
		ORDER BY 2`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type candidate struct {
		ID       string `json:"id"`
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}
	out := []candidate{}
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ID, &c.FullName, &c.Phone); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"users": out})
}
