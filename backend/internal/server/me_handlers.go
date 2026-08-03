package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleMeSummary بيانات التوب بار الموحّدة لأي مستخدم: الاسم، الصورة، رصيد المحفظة.
func (s *Server) handleMeSummary(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var out struct {
		FullName    string  `json:"full_name"`
		AvatarThumb *string `json:"avatar_thumb_url"`
		Balance     int64   `json:"balance"`
		// قناة التواصل الموثّقة — تُعرض في «حسابي» وتفتح أدوات المندوب
		WhatsAppPhone    *string `json:"whatsapp_phone"`
		WhatsAppVerified bool    `json:"whatsapp_verified"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.full_name,
		       (SELECT m.thumb_path FROM media m WHERE m.id = u.avatar_media_id),
		       COALESCE((SELECT w.balance FROM wallets w WHERE w.user_id = u.id), 0),
		       u.whatsapp_phone, u.whatsapp_verified_at IS NOT NULL
		FROM users u WHERE u.id = $1`, uid).
		Scan(&out.FullName, &out.AvatarThumb, &out.Balance,
			&out.WhatsAppPhone, &out.WhatsAppVerified)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out.AvatarThumb = media.URLForPtr(out.AvatarThumb) // "/media/" prefix (نمط الوسائط)
	httpx.JSON(w, http.StatusOK, out)
}

// handleSetMyName يغيّر المستخدمُ اسمَه بنفسه.
//
// **لم يكن له بابٌ ألبتّة**: صفحةُ «حسابي» تقرأ الاسمَ وتعرضه **ولا تكتبه** —
// ومن أخطأ في اسمه عند التسجيل، أو كُتب له بيد موظّفٍ في طلبٍ هاتفيّ، **يبقى
// عليه إلى الأبد.** (شهده المالك ٢٠٢٦-٠٨-٠٣.)
func (s *Server) handleSetMyName(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		FullName string `json:"full_name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetOwnName(r.Context(), userIDFrom(r), req.FullName); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويُسجَّل**: الاسمُ يُقرأ في الفاتورة وعند باب الزبون، **ومن غيّره مرّاتٍ
	// يُقرأ ذلك حين يُسأل عن طلبٍ باسمٍ لا يطابق.**
	s.audit(r, "user.rename_self", "user", userIDFrom(r), nil)
	httpx.JSON(w, http.StatusOK, map[string]any{"full_name": req.FullName})
}

// handleMyAvatar يرفع صورة المستخدم لنفسه (نوع avatar) ويضبطها، ويعيد رابط المصغّرة.
func (s *Server) handleMyAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+64<<10)
	if err := r.ParseMultipartForm(media.MaxUploadBytes); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	uid := userIDFrom(r)
	// النوع مثبّت "avatar" — لا نثق بقيمة العميل على نقطة عامة للمستخدمين.
	mm, err := s.media.Save(r.Context(), uid, "avatar", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetOwnAvatar(r.Context(), uid, mm.ID); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"avatar_thumb_url": mm.ThumbURL})
}

// handleDeleteMyAvatar يزيل صورة المستخدم.
func (s *Server) handleDeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.SetOwnAvatar(r.Context(), userIDFrom(r), ""); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}

// handleMyRatings طلبات الزبون المُسلَّمة مع حالة تقييم كل منها (لصفحة "تقييماتي").
func (s *Server) handleMyRatings(w http.ResponseWriter, r *http.Request) {
	// **ولا اسمَ متجرٍ هنا** — الزبونُ يعرف طلبَه بما طلب لا بمن طبخه، **وصفحةُ
	// تقييماتٍ تسمّي المطعم تهدم ما تحرسه صفحةُ الطلب.** (انظر `customer_privacy.go`)
	rows, err := s.pg.Query(r.Context(), `
		SELECT o.id::text, o.number,
		       COALESCE((SELECT string_agg(x.name, '، ' ORDER BY x.rn)
		                 FROM (SELECT oi.name, row_number() OVER (ORDER BY oi.name) AS rn
		                       FROM order_items oi WHERE oi.order_id = o.id LIMIT 3) x), ''),
		       (o.driver_id IS NOT NULL),
		       COALESCE(rt.platform_stars, 0), rt.driver_stars, COALESCE(rt.comment, ''),
		       (rt.order_id IS NOT NULL), o.created_at
		FROM orders o
		LEFT JOIN order_ratings rt ON rt.order_id = o.id
		WHERE o.customer_id = $1 AND o.status = 'delivered'
		ORDER BY o.created_at DESC LIMIT 100`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type ratedOrder struct {
		OrderID string `json:"order_id"`
		Number  int64  `json:"number"`
		// ItemsPreview **بدل اسم المتجر** — «ماذا طلبتُ؟» لا «من طبخه؟»
		ItemsPreview  string    `json:"items_preview"`
		HasDriver     bool      `json:"has_driver"`
		PlatformStars int       `json:"platform_stars"`
		DriverStars   *int      `json:"driver_stars"`
		Comment       string    `json:"comment"`
		Rated         bool      `json:"rated"`
		CreatedAt     time.Time `json:"created_at"`
	}
	out := []ratedOrder{}
	for rows.Next() {
		var o ratedOrder
		if err := rows.Scan(&o.OrderID, &o.Number, &o.ItemsPreview, &o.HasDriver,
			&o.PlatformStars, &o.DriverStars, &o.Comment, &o.Rated, &o.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, o)
	}
	httpx.JSON(w, http.StatusOK, out)
}
