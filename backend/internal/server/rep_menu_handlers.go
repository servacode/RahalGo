package server

// ══════════════════════════════════════════════════════════════════════
// **المندوبُ يبني قائمةَ عميله نيابةً عنه**
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يوجد ميزةٌ مهمّةٌ لحساب المندوب تمّ تجاهلُها
//  ونسيانُها، وهي أن يقوم المندوبُ بإضافة أصناف المنتجات الموجودة لدى
//  المتجر بدلاً عنه… ويستطيع المندوبُ تعديلَ السعر أيضاً وجعلَ المنتج
//  غيرَ متاحٍ أو متاحاً ومتوفّراً وغيرَ متوفّر — يعني نفس الفورم الموجود
//  عند مدير المنصّة والموجود عند المتجر موجودٌ عند المندوب».)
//
// # ولماذا هي مهمّة
//
// **متجرٌ ينضمّ اليومَ ولا يفتح لوحتَه غداً** — صاحبُه في متجره لا في
// حاسوب. **وسوقٌ فيه متاجرُ بلا أصنافٍ سوقٌ فارغ**، والمندوبُ هو من رآه
// وجه‌اً لوجه.
//
// # ولا نسخةَ ثالثةٌ من المنطق
//
// **الإدارةُ لها مسارُها والمتجرُ له مسارُه** — **ونسخةٌ ثالثةٌ تعني
// ثلاثةَ أماكنَ يُصلَح فيها العيبُ ويُنسى ثالثُها.** فهذه المعالجاتُ
// تنادي خدمةَ القائمة نفسَها، **ولا تحمل إلّا سؤالاً واحداً: أهذا
// المتجرُ عميلُه؟**
//
// # وحارسُ الملكيّة غيرُ حارس المتجر
//
// **صاحبُ المتجر يملكه** (`owner_user_id`)، **والمندوبُ جلبه**
// (`sales_rep_user_id`) — **وخلطُ الاثنين يجعل مندوباً يعدّل متجراً ليس
// عميلَه**، أو يمنع صاحبَه من متجره.

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"

	"github.com/go-chi/chi/v5"
)

// repClient **أهذا المتجرُ عميلُ هذا المندوب؟**
//
// **ويُقرأ من القاعدة في كلّ نداء** — **ولائحةٌ تُبنى مرّةً في الجلسة
// تشيخ**: عميلٌ نُقل إلى مندوبٍ آخرَ يبقى الأوّلُ يعدّل قائمتَه.
func (s *Server) repClient(r *http.Request, merchantID string) bool {
	var ok bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM merchants
		              WHERE id = $1 AND sales_rep_user_id = $2)`,
		merchantID, userIDFrom(r)).Scan(&ok)
	return err == nil && ok
}

// repOwnsItem **أصنفٌ في قائمة أحد عملائه؟** — للمسارات التي تحمل
// معرّفَ الصنف لا معرّفَ المتجر.
//
// **ومن عدّل صنفاً بمعرّفه وحدَه بلا هذا السؤال عدّل قائمةَ أيّ متجر** —
// **والمعرّفاتُ تُخمَّن أو تُقرأ من ردٍّ سابق.**
func (s *Server) repOwnsItem(r *http.Request, itemID string) bool {
	var ok bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM menu_items i
		              JOIN merchants m ON m.id = i.merchant_id
		              WHERE i.id = $1 AND m.sales_rep_user_id = $2)`,
		itemID, userIDFrom(r)).Scan(&ok)
	return err == nil && ok
}

// repMenuGuard **يحرس مسارات القائمة عند المندوب.**
//
// **ويقرأ المعرّفَ من المسار نفسِه** — `id` للمتجر و`itemID` للصنف:
// **فحارسٌ واحدٌ يكفي الجميعَ ولا يُنسى في واحد.**
//
// **وكان له فرعٌ ثالثٌ للأقسام** — ذهب مع أبوابها (٢٠٢٦-٠٨-١٨):
// **وفرعُ حارسٍ لا بابَ له يُقرأ على أنّ ثمّة باباً.**
func (s *Server) repMenuGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case chi.URLParam(r, "id") != "":
			if !s.repClient(r, chi.URLParam(r, "id")) {
				s.respondErr(w, errForbidden)
				return
			}
		case chi.URLParam(r, "itemID") != "":
			if !s.repOwnsItem(r, chi.URLParam(r, "itemID")) {
				s.respondErr(w, errForbidden)
				return
			}
		default:
			// **ومسارٌ بلا معرّفٍ لا يمرّ** — **والافتراضُ منعٌ لا سماح**:
			// من أضاف مساراً غداً ونسي معرّفَه يجد باباً مغلقاً لا مفتوحاً.
			s.respondErr(w, errForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleRepMenu **قائمةُ عميلٍ كما يراها المندوب.**
//
// **وسعرُ الشراء يُعرض له** — **بخلاف صاحب المتجر**: المندوبُ يبني
// القائمةَ نيابةً فيحتاج الرقمين، **وصاحبُ المتجر لا يرى ما تبيع به
// المنصّة.**
func (s *Server) handleRepMenu(w http.ResponseWriter, r *http.Request) {
	menu, err := s.catalog.GetMenu(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, menu)
}
