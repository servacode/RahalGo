package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// طلبات انضمام المتاجر عبر رابط المندوب.

// حدود طول الحقول — نقطة عامة بلا حساب، نمنع تخزين حمولات ضخمة لكل صف.
const (
	leadMaxShort = 120  // اسم المتجر/المالك/المنطقة/الهاتف
	leadMaxNote  = 1000 // الملاحظات
	joinMaxPerIP = 10   // طلبات لكل عنوان خلال النافذة
)

// clip يقصّ ويهذّب نصاً إلى حدٍّ أقصى.
func clip(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		return v[:max]
	}
	return v
}

// handlePublicJoin التقاط طلب انضمام متجر عبر رابط/باركود مندوب (عام، بلا حساب).
func (s *Server) handlePublicJoin(w http.ResponseWriter, r *http.Request) {
	// تحديد المعدل حسب العنوان — نقطة عامة قابلة للإغراق (نفس نمط طلب الرمز).
	// نعزل المضيف عن المنفذ العابر كي يكون المفتاح لكل عنوان لا لكل اتصال.
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "join:req:" + ip
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > joinMaxPerIP {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}

	req, err := decode[struct {
		Ref       string `json:"ref"` // كود دعوة المندوب
		StoreName string `json:"store_name"`
		OwnerName string `json:"owner_name"`
		Phone     string `json:"phone"`
		Area      string `json:"area"`
		Note      string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	req.StoreName = clip(req.StoreName, leadMaxShort)
	req.OwnerName = clip(req.OwnerName, leadMaxShort)
	req.Phone = clip(req.Phone, leadMaxShort)
	req.Area = clip(req.Area, leadMaxShort)
	req.Note = clip(req.Note, leadMaxNote)
	if req.StoreName == "" || req.Phone == "" {
		s.respondErr(w, errValidation)
		return
	}
	// المندوب صاحب الكود (اختياري — الرابط قد يُفتح بلا كود)
	var repID *string
	if req.Ref != "" {
		if u, err := s.identity.SalesRepByInviteCode(r.Context(), req.Ref); err == nil {
			repID = &u.ID
		}
	}
	if _, err := s.pg.Exec(r.Context(), `
		INSERT INTO merchant_leads (store_name, owner_name, phone, area, note, sales_rep_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		req.StoreName, req.OwnerName, req.Phone, req.Area, req.Note, repID); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"received": true})
}

type lead struct {
	ID        string    `json:"id"`
	StoreName string    `json:"store_name"`
	OwnerName string    `json:"owner_name"`
	Phone     string    `json:"phone"`
	Area      string    `json:"area"`
	Note      string    `json:"note"`
	RepName   *string   `json:"rep_name"`
	RepCode   *string   `json:"rep_code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func scanLeads(rows interface {
	Next() bool
	Scan(...any) error
}) ([]lead, error) {
	out := []lead{}
	for rows.Next() {
		var l lead
		if err := rows.Scan(&l.ID, &l.StoreName, &l.OwnerName, &l.Phone, &l.Area, &l.Note,
			&l.RepName, &l.RepCode, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

const leadSelect = `
	SELECT l.id, l.store_name, l.owner_name, l.phone, l.area, l.note,
	       NULLIF(COALESCE(u.full_name, u.phone::text), ''), u.invite_code, l.status, l.created_at
	FROM merchant_leads l
	LEFT JOIN users u ON u.id = l.sales_rep_user_id`

// handleAdminLeads كل طلبات الانضمام (ترشيح بالحالة اختياري).
func (s *Server) handleAdminLeads(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	rows, err := s.pg.Query(r.Context(),
		leadSelect+` WHERE ($1 = '' OR l.status = $1) ORDER BY l.created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanLeads(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRepLeads طلبات انضمام المندوب نفسه (بوابة المندوب).
func (s *Server) handleRepLeads(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(),
		leadSelect+` WHERE l.sales_rep_user_id = $1 ORDER BY l.created_at DESC LIMIT 200`,
		userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out, err := scanLeads(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminLeadStatus تحديث حالة طلب (رفض، أو وسمه محوَّلاً يدوياً).
func (s *Server) handleAdminLeadStatus(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Status string `json:"status"`
	}](r)
	if err != nil || (req.Status != "converted" && req.Status != "rejected" && req.Status != "new") {
		s.respondErr(w, errValidation)
		return
	}
	if !isUUID(chi.URLParam(r, "id")) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	tag, err := s.pg.Exec(r.Context(),
		`UPDATE merchant_leads SET status = $2, updated_at = now() WHERE id = $1`,
		chi.URLParam(r, "id"), req.Status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
