package server

// ══════════════════════════════════════════════════════════════════════
// **كتابةُ المندوب في قائمة عميله — بحارسَين لا بواحد**
// ══════════════════════════════════════════════════════════════════════
//
// (سؤالُ المالك ٢٠٢٦-٠٨-٣٠: «ماذا يحصل إذا مندوبٌ ضغط على متوفّر أو غير
//  متوفّر لصنفٍ في متجرٍ تابعٍ له أو عدّل سعرَه أو صورتَه؟ هذه أيضاً من
//  صلاحيات المندوب، **يجب أن تنعكس على الجميع** مثل المعلومات بلوحة
//  الأدمن والمتجر نفسه والزبون أيضاً».)
//
// **والجوابُ نعم — وفوراً**: الثلاثةُ يكتبون في `menu_items` نفسِه،
// **والزبونُ يقرأ منه بشرط `available`.**
//
// # ولا مراجعةَ قبل النشر
//
// **كان هنا حارسُ مراجعةٍ يقرأ `merchants.menu_requires_approval`** —
// **ورُفع هو ومفتاحُه** بقرار المالك ٢٠٢٦-١٠-٠٤: «ما في داعي للموافقة
// على الصنف أساساً». **فما يكتبه المندوبُ أو المتجرُ يظهر فوراً.**
//
// # وصاحبُ المتجر يُخطَر بما يجري في قائمته
//
// **كان لا يعلم**: يبدّل المندوبُ سعرَ صنفٍ أو يحذفه، **ولا يصل صاحبَ
// المتجر شيء** — يكتشفه إن فتح تطبيقَه ونظر. **وقائمتُه مصدرُ رزقه، لا
// دفترُ ملاحظاتٍ يُكتب فيه بلا علمه.**
//
// **والتوفّرُ لا يُخطَر به** — «نفد الصنف» قرارُ مطبخٍ في لحظته (الهجرة
// ٠٠٦٤). **ومندوبٌ يقلب التوفّرَ عشرين مرّةً يرسل عشرين إشعاراً**
// فيُطفئها صاحبُ المتجر، **فيخسر الإشعارَ المهمّ معها.**

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// نصوصُ إشعار صاحب المتجر — **يقول ماذا وقع في قائمته لا أنّ «شيئاً وقع».**
const (
	notifRepItemAdded   = "مندوبك أضاف صنفاً إلى قائمتك"
	notifRepItemChanged = "مندوبك عدّل صنفاً في قائمتك"
	notifRepItemRemoved = "مندوبك حذف صنفاً من قائمتك"
)

// notifyMerchantOfRepEdit **يُخبر صاحبَ المتجر.**
//
// **ويُنادى بعد نجاح الكتابة لا داخلها** — تعثّرُ الإشعار لا يُبطل
// تعديلاً وقع.
func (s *Server) notifyMerchantOfRepEdit(r *http.Request, merchantID, title string) {
	if merchantID == "" || s.notify == nil {
		return
	}
	var owner *string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`,
		merchantID).Scan(&owner); err != nil || owner == nil {
		// **ومتجرٌ بلا صاحبٍ بعدُ لا يُخطَر** — يُنشأ بالموافقة على الطلب،
		// **وقد يبني المندوبُ قائمتَه قبل أن يدخل صاحبُه أوّلَ مرّة.**
		return
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: *owner,
		Kind:   notifications.KindAccount, Title: title,
		// **والوجهةُ قائمتُه.**
		Entity: "merchant", EntityID: merchantID, Href: "/portal/menu",
	})
}

// merchantOfItem **متجرُ الصنف** — يُقرأ قبل الحذف، إذ لا يبقى بعده صفّ.
func (s *Server) merchantOfItem(ctx context.Context, itemID string) string {
	var id string
	if err := s.pg.QueryRow(ctx,
		`SELECT merchant_id::text FROM menu_items WHERE id = $1`, itemID).Scan(&id); err != nil {
		return ""
	}
	return id
}

// handleRepCreateItem **صنفٌ جديدٌ يبنيه المندوبُ لعميله.**
//
// **ويظهر فوراً بلا مراجعة** — كصنف المتجر (قرارُ المالك ٢٠٢٦-١٠-٠٤).
func (s *Server) handleRepCreateItem(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنصوصُ تمرّ بالحارس المركزيّ** — انظر `text_limits.go`.
	if err := s.guardMenuItem(r, req); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والإدراجُ وعلامةُ منع التكرار في معاملةٍ واحدة** (`DUP-LEAD`) —
	// صنفٌ أُدرج وضاع ردُّه فأُعيد **لا يُدرج ثانيةً**: يُعاد ردُّ الأوّل.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		id, err := s.catalog.CreateItemIn(ctx, q, merchantID, *req)
		if err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"id": id},
			AfterCommit: func() {
				s.catalog.AuditItemCreate(r.Context(), userIDFrom(r), id, clientIP(r))
				s.touch("menu", "ops")
				s.notifyMerchantOfRepEdit(r, merchantID, notifRepItemAdded)
			},
		}, nil
	})
}

// touchesContent أيمسّ هذا التعديلُ ما يراه الزبون؟ — **والإتاحةُ وحدَها لا تمسّه.**
func touchesContent(in catalog.MenuItemInput) bool {
	return in.Name != nil || in.Description != nil || in.Price != nil ||
		in.ImageMediaID != nil || in.PlatformSectionID != nil ||
		in.MarginOverride != nil || in.SectionID != nil || in.Modifiers != nil
}

// handleRepUpdateItem **تعديلُ صنفٍ في قائمة عميله.**
func (s *Server) handleRepUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنصوصُ تمرّ بالحارس المركزيّ** — انظر `text_limits.go`.
	if err := s.guardMenuItem(r, req); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.catalog.UpdateItem(r.Context(), userIDFrom(r), itemID, *req, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والتوفّرُ وحدَه لا يُخطِر** — «نفد» قرارُ مطبخٍ في لحظته.
	if touchesContent(*req) {
		s.notifyMerchantOfRepEdit(r,
			s.merchantOfItem(r.Context(), itemID), notifRepItemChanged)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleRepDeleteItem **حذفُ صنفٍ من قائمة عميله.**
func (s *Server) handleRepDeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	// **ويُقرأ المتجرُ قبل الحذف** — بعده لا صفَّ يُقرأ منه.
	merchantID := s.merchantOfItem(r.Context(), itemID)
	if err := s.catalog.DeleteItem(r.Context(), userIDFrom(r), itemID, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.notifyMerchantOfRepEdit(r, merchantID, notifRepItemRemoved)
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
