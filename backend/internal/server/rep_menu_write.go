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
// **والزبونُ يقرأ منه بشرطَي `available AND approved`.**
//
// # لكنّ بابَ المندوب كان يحمل حارساً واحداً
//
// **كان يستعمل معالجاتِ الإدارة حرفاً** (`handleCreateItem` وأختيها)،
// **فورث إعفاءَ الأدمن من المراجعة.** والتعليقُ في `menu_approval.go`
// يقول لماذا أُعفي الأدمن:
//
//     «الحارسُ في باب المتجر لا في `catalog`: **الأدمنُ هو المُراجِع**،
//      وما يكتبه منشورٌ لحظتَه.»
//
// **وهو صحيحٌ في الأدمن — والمندوبُ ليس أدمن.**
//
// **فمفتاحُ `merchants.menu_requires_approval` كان يحرس نصفَ الأبواب**:
// يُشغَّل فيُحجب ما يكتبه المتجر، **ويمرّ ما يكتبه المندوبُ إلى السوق
// بلا مراجعة.**
//
// **وإعدادٌ يحرس نصفَ الأبواب أخطرُ من إعدادٍ لا يحرس شيئاً**: من شغّله
// ظنّ القوائمَ كلَّها محروسة، **فبنى عليه.** (وهي علّةُ الهجرة ٠٠٦٤
// نفسُها في مرآتها — عادت في بابٍ ثانٍ.)
//
// **والمندوبُ أَولى بالمراجعة من المتجر**: هو من يكتب على هاتفه واقفاً
// في السوق بسرعة، **وحجّةُ الهجرة تخصّه قبل غيره**: «سعرٌ كُتب بخطأ
// صفرٍ زائدٍ يُباع به».
//
// # وصاحبُ المتجر يُخطَر بما يجري في قائمته
//
// **كان لا يعلم**: يبدّل المندوبُ سعرَ صنفٍ أو يحذفه، **ولا يصل صاحبَ
// المتجر شيء** — يكتشفه إن فتح تطبيقَه ونظر. **وقائمتُه مصدرُ رزقه، لا
// دفترُ ملاحظاتٍ يُكتب فيه بلا علمه.**
//
// **والقاعدةُ واحدةٌ في الاثنين: ما يُراجَع هو ما يُخطَر به.**
//
// **والتوفّرُ خارجَهما** — «نفد الصنف» قرارُ مطبخٍ في لحظته (الهجرة
// ٠٠٦٤). **ومندوبٌ يقلب التوفّرَ عشرين مرّةً يرسل عشرين إشعاراً**
// فيُطفئها صاحبُ المتجر، **فيخسر الإشعارَ المهمّ معها.**

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
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
// تعديلاً وقع. **وهو النمطُ نفسُه الذي تسير عليه `holdForReview`.**
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
		// **والوجهةُ قائمتُه** — هي نفسُها التي تقصدها مراجعةُ الصنف
		// حين يُنشر أو يُردّ (`menu_approval.go`). **ووجهةٌ ثانيةٌ لشيءٍ
		// واحدٍ تفترق يوماً.**
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
// **ويُراجَع كلُّه** — لا شيءَ منه رآه أحدٌ بعد.
func (s *Server) handleRepCreateItem(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := s.catalog.CreateItem(r.Context(), userIDFrom(r), merchantID, *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	pending := s.menuNeedsApproval(r)
	if pending {
		s.holdForReview(r, id)
	}
	s.notifyMerchantOfRepEdit(r, merchantID, notifRepItemAdded)
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "pending_review": pending})
}

// handleRepUpdateItem **تعديلُ صنفٍ في قائمة عميله.**
func (s *Server) handleRepUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.catalog.UpdateItem(r.Context(), userIDFrom(r), itemID, *req, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والتوفّرُ وحدَه لا يُعلّق ولا يُخطِر** — الشرطُ واحدٌ للاثنين.
	content := touchesContent(*req)
	pending := s.menuNeedsApproval(r) && content
	if pending {
		s.holdForReview(r, itemID)
	}
	if content {
		s.notifyMerchantOfRepEdit(r,
			s.merchantOfItem(r.Context(), itemID), notifRepItemChanged)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true, "pending_review": pending})
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
