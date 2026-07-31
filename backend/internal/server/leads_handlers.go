package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// طلبات انضمام المتاجر عبر رابط المندوب.

// نصوص إشعارات هذا القسم — مجمّعة كي لا تتناثر في الكود.
var m = struct{ leadNew, leadNewOps, leadApproved string }{
	leadNew:      "طلب انضمام جديد عبر رابطك",
	leadNewOps:   "طلب انضمام متجر جديد",
	leadApproved: "تمت الموافقة على عميلك",
}

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
		Ref        string   `json:"ref"` // كود دعوة المندوب
		StoreName  string   `json:"store_name"`
		OwnerName  string   `json:"owner_name"`
		Phone      string   `json:"phone"`
		Area       string   `json:"area"`
		CategoryID string   `json:"category_id"` // تصنيف المتجر
		Password   string   `json:"password"`    // كلمة مرور صاحب المتجر
		Lat        *float64 `json:"lat"`         // موقع المتجر (اختياري)
		Lng        *float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	req.StoreName = clip(req.StoreName, leadMaxShort)
	req.OwnerName = clip(req.OwnerName, leadMaxShort)
	req.Area = clip(req.Area, leadMaxShort)
	if req.StoreName == "" {
		s.respondErr(w, errValidation)
		return
	}
	// الرقم يجب أن يكون رقم موبايل صالح (سيصله رمز الدخول والإشعارات عبر واتساب).
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	// كلمة المرور إلزامية (يدخل بها صاحب المتجر بعد الموافقة) — 8 محارف فأكثر.
	if len(req.Password) < 8 {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "weak_password", "errors.weak_password"))
		return
	}
	pwHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// معرّف التصنيف — اختياري لكنه إن وُجد يجب أن يكون UUID صالحاً.
	var categoryID *string
	if req.CategoryID != "" {
		if !isUUID(req.CategoryID) {
			s.respondErr(w, errValidation)
			return
		}
		categoryID = &req.CategoryID
	}
	// الإسناد: كود مندوب صالح وفعّال → يُنسب له. غير ذلك (لا كود/كود المنصة/كود
	// خاطئ) → تسجيل مباشر منسوب للمنصة (لا رفض) — الإنشاء الفعلي عند موافقة الإدارة.
	var repID *string
	if rep, err := s.identity.SalesRepByInviteCode(r.Context(), req.Ref); err == nil && rep.Status == "active" {
		repID = &rep.ID
	}
	if _, err := s.pg.Exec(r.Context(), `
		INSERT INTO merchant_leads
			(store_name, owner_name, phone, area, category_id, lat, lng, owner_password_hash, sales_rep_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.StoreName, req.OwnerName, phone, req.Area,
		categoryID, req.Lat, req.Lng, pwHash, repID); err != nil {
		s.respondErr(w, err)
		return
	}
	// إشعار فوري: المندوب صاحب الكود والإدارة يعرفان بالطلب بلا تحديث صفحة.
	if repID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadNew, Body: req.StoreName,
			Entity: "lead", Href: "/portal/leads",
		})
	}
	s.notify.NotifyRole(r.Context(), "ops", notifications.Input{
		Kind: notifications.KindLead, Title: m.leadNewOps,
		Body: req.StoreName, Entity: "lead", Href: "/dashboard/leads",
	})
	httpx.JSON(w, http.StatusCreated, map[string]any{"received": true})
}

// handlePublicInvite يعيد كود الدعوة الذي يُعرض في نموذج التسجيل (للقراءة فقط):
// كود المندوب إن كان صالحاً وفعّالاً، وإلا كود المنصة الافتراضي (تسجيل مباشر).
func (s *Server) handlePublicInvite(w http.ResponseWriter, r *http.Request) {
	// تحديد معدل حسب العنوان — نقطة عامة قد تُستغل لتعداد أكواد المندوبين.
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "invite:req:" + ip
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > 60 {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}

	ref := r.URL.Query().Get("ref")
	if ref != "" {
		if rep, err := s.identity.SalesRepByInviteCode(r.Context(), ref); err == nil && rep.Status == "active" {
			httpx.JSON(w, http.StatusOK, map[string]any{"code": ref, "by": "rep"})
			return
		}
	}
	code := s.settings.GetString(r.Context(), "platform.invite_code", "RAHALGO")
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code, "by": "platform"})
}

type lead struct {
	ID           string    `json:"id"`
	StoreName    string    `json:"store_name"`
	OwnerName    string    `json:"owner_name"`
	Phone        string    `json:"phone"`
	Area         string    `json:"area"`
	CategoryName *string   `json:"category_name"`
	CategoryIcon *string   `json:"category_icon"`
	Lat          *float64  `json:"lat"`
	Lng          *float64  `json:"lng"`
	RepName      *string   `json:"rep_name"`
	RepCode      *string   `json:"rep_code"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func scanLeads(rows interface {
	Next() bool
	Scan(...any) error
}) ([]lead, error) {
	out := []lead{}
	for rows.Next() {
		var l lead
		if err := rows.Scan(&l.ID, &l.StoreName, &l.OwnerName, &l.Phone, &l.Area,
			&l.CategoryName, &l.CategoryIcon, &l.Lat, &l.Lng,
			&l.RepName, &l.RepCode, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

const leadSelect = `
	SELECT l.id, l.store_name, l.owner_name, l.phone, l.area,
	       c.name, c.icon, l.lat, l.lng,
	       NULLIF(COALESCE(u.full_name, u.phone::text), ''), u.invite_code, l.status, l.created_at
	FROM merchant_leads l
	LEFT JOIN users u ON u.id = l.sales_rep_user_id
	LEFT JOIN categories c ON c.id = l.category_id`

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

// handleAdminLeadStatus تحديث حالة طلب. "converted" ينشئ المتجر فعلياً (موافقة
// الإدارة هي لحظة الإنشاء — لا متجر قبلها). "rejected"/"new" مجرد وسم.
func (s *Server) handleAdminLeadStatus(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Status string `json:"status"`
	}](r)
	if err != nil || (req.Status != "converted" && req.Status != "rejected" && req.Status != "new") {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if req.Status == "converted" {
		if err := s.convertLead(r.Context(), userIDFrom(r), id, clientIP(r)); err != nil {
			s.respondErr(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
		return
	}
	tag, err := s.pg.Exec(r.Context(),
		`UPDATE merchant_leads SET status = $2, updated_at = now() WHERE id = $1`,
		id, req.Status)
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

// convertLead يحوّل طلب انضمام إلى متجر فعلي: ينشئ/يربط حساب صاحب المتجر (بكلمة
// مروره المحفوظة إن كان جديداً) والمتجر بتصنيفه وموقعه منسوباً للمندوب.
func (s *Server) convertLead(ctx context.Context, actorID, leadID, ip string) error {
	var (
		storeName, ownerName, phone, area string
		categoryID                        *string
		lat, lng                          *float64
		pwHash                            string
		repCode                           *string
		merchantID                        *string
	)
	err := s.pg.QueryRow(ctx, `
		SELECT l.store_name, l.owner_name, l.phone, l.area, l.category_id, l.lat, l.lng,
		       l.owner_password_hash, u.invite_code, l.merchant_id
		FROM merchant_leads l
		LEFT JOIN users u ON u.id = l.sales_rep_user_id
		WHERE l.id = $1`, leadID).
		Scan(&storeName, &ownerName, &phone, &area, &categoryID, &lat, &lng, &pwHash, &repCode, &merchantID)
	if err != nil {
		return httpx.ErrNotFound
	}
	if merchantID != nil {
		return nil // محوّل مسبقاً — لا تكرار
	}
	if categoryID == nil {
		return errValidation // لا متجر بلا تصنيف
	}
	in := catalog.MerchantInput{
		Name:        &storeName,
		CategoryID:  categoryID,
		Phone:       &phone,
		AddressText: &area,
		OwnerPhone:  &phone,
		Lat:         lat,
		Lng:         lng,
	}
	if repCode != nil {
		in.SalesRepCode = repCode
	}
	// هل يملك الرقم حساباً مسبقاً؟ حرج أمنياً: كلمة المرور من نموذج التسجيل يجب ألّا
	// تُطبَّق على حساب قائم (وإلا يمكن لمهاجم "التسجيل" برقم ضحية بلا كلمة مرور ثم
	// يستولي على حسابها عند الموافقة). نطبّق كلمة المرور على الحسابات الجديدة فقط.
	var ownerExisted bool
	_ = s.pg.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`, phone).Scan(&ownerExisted)

	mrch, err := s.catalog.CreateMerchant(ctx, actorID, in, ip)
	if err != nil {
		return err
	}
	// الاسم يُملأ إن كان فارغاً (غير حسّاس). كلمة المرور للحساب الجديد حصراً.
	_, _ = s.pg.Exec(ctx, `
		UPDATE users SET
			full_name  = CASE WHEN full_name = '' THEN $2 ELSE full_name END,
			updated_at = now()
		WHERE id = (SELECT owner_user_id FROM merchants WHERE id = $1)`,
		mrch.ID, ownerName)
	if !ownerExisted && pwHash != "" {
		_, _ = s.pg.Exec(ctx, `
			UPDATE users SET password_hash = $2, updated_at = now()
			WHERE id = (SELECT owner_user_id FROM merchants WHERE id = $1)
			  AND COALESCE(password_hash,'') = ''`,
			mrch.ID, pwHash)
	}
	_, err = s.pg.Exec(ctx, `
		UPDATE merchant_leads SET status = 'converted', merchant_id = $2, updated_at = now()
		WHERE id = $1`, leadID, mrch.ID)
	// المندوب يعرف فوراً أن عميله اعتُمد (مصدر عمولته)
	var repID *string
	_ = s.pg.QueryRow(ctx, `SELECT sales_rep_user_id FROM merchant_leads WHERE id = $1`, leadID).Scan(&repID)
	if repID != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *repID, Kind: notifications.KindLead,
			Title: m.leadApproved, Body: storeName,
			Entity: "merchant", EntityID: mrch.ID, Href: "/portal/merchants",
		})
	}
	return err
}
