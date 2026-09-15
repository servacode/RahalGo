package server

// ══════════════════════════════════════════════════════════════════════
// **عروضُ المتجر — يبنيها صاحبُه ومندوبُه** (`OF`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا محرّكَ ثانيا
//
// **الجدولُ نفسُه** (`offers`)، **وشرطُ السريان نفسُه** (`offers.LiveCond`)،
// **والدالّةُ نفسُها التي يناديها بناءُ الطلب** (`LiveDiscount`).
//
// **ومحرّكٌ ثانٍ للعروض يعني سعرين**: **واحداً يُعرض وواحداً يُقيَّد** —
// **وهو بعينه ما مُنع في `AfterDiscount`**: «حسبةٌ في موضعين تفترق يوماً».
//
// # والفرقُ كلُّه نطاقٌ لا منطق
//
// **وبابُ الإدارة باقٍ كما هو بقدرته** (`content.manage`) — **ولم يُوسَّع
// صاحبُ المتجر ليصير أدمن**: **بابٌ آخرُ يسأل سؤالاً آخر**: **أهذا
// الصنفُ في قائمتك؟**
//
//	الأدمن    ←  قدرةُ المحتوى، وأيُّ صنفٍ في السوق
//	المتجر    ←  merchants.owner_user_id
//	المندوب   ←  merchants.sales_rep_user_id
//
// **والاثنان الأخيران حقيقتان قائمتان في القاعدة** — **لم تُخترَع
// لهذه الدفعة**: **الأولى تحرس قائمةَ المتجر منذ أوّل يوم، والثانية
// تحرس ما يكتبه المندوبُ في قائمة عميله** (`rep_menu_write.go`).
//
// # ومن يتحمّل الخصم ليس خيارَه
//
// **وصاحبُ المتجر يخصم من مستحقّه هو** — **ولو مَلَك أن يقول «تتحمّله
// المنصّة» لأنفق من هامشِ غيره بضغطة.** **والقرارُ بيد الإدارة وحدَها**
// كما كان (الهجرة ٠٠٧٤: «أقرّر لكلّ عرضٍ على حدة»).

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/offers"
)

// scopeOfferActor **من الفاعلُ ولأيّ متجرٍ يُؤذَن له؟**
//
// **ويُقرأ من الهُويّة المُوثَّقة والمسار المحروس** — **لا من الحمولة.**
type offerActor struct {
	role       string // `merchant` أو `rep` — **يُقيَّد في السجلّ.**
	merchantID string
}

// merchantOfferScope **متجرُ صاحب الحساب** — **أو منعٌ.**
func (s *Server) merchantOfferScope(r *http.Request) (offerActor, bool) {
	id := chi.URLParam(r, "id")
	if id == "" || !s.ownsMerchant(r, id) {
		return offerActor{}, false
	}
	return offerActor{role: "merchant", merchantID: id}, true
}

// repOfferScope **متجرُ عميلِه** — **أو منعٌ.**
func (s *Server) repOfferScope(r *http.Request) (offerActor, bool) {
	id := chi.URLParam(r, "id")
	if id == "" || !s.repClient(r, id) {
		return offerActor{}, false
	}
	return offerActor{role: "rep", merchantID: id}, true
}

// offerByID **عرضٌ بمعرّفه، بعد سؤال: أهو في نطاق هذا الفاعل؟**
//
// **ويُسأل عن متجر العرض من القاعدة** — **ولا يُصدَّق أنّ من بلغ المسارَ
// يملك ما فيه**: **فمعرّفُ العرض يُقرأ من ردٍّ سابقٍ أو يُخمَّن.**
func (s *Server) offerInScope(r *http.Request, who func(*http.Request) (offerActor, bool)) (offerActor, string, bool) {
	a, ok := who(r)
	if !ok {
		return offerActor{}, "", false
	}
	offerID := chi.URLParam(r, "offerID")
	if offerID == "" {
		return offerActor{}, "", false
	}
	owner, err := s.offers.OwnerOf(r.Context(), offerID)
	if err != nil || owner != a.merchantID {
		return offerActor{}, "", false
	}
	return a, offerID, true
}

// ══════════════════════════════════════════════════════════════════════
// **القراءة**
// ══════════════════════════════════════════════════════════════════════

func (s *Server) listScopedOffers(w http.ResponseWriter, r *http.Request,
	who func(*http.Request) (offerActor, bool)) {
	a, ok := who(r)
	if !ok {
		s.respondErr(w, errForbidden)
		return
	}
	rows, err := s.offers.ListForMerchant(r.Context(), a.merchantID, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"offers": rows})
}

func (s *Server) handleMerchantOffers(w http.ResponseWriter, r *http.Request) {
	s.listScopedOffers(w, r, s.merchantOfferScope)
}

func (s *Server) handleRepOffers(w http.ResponseWriter, r *http.Request) {
	s.listScopedOffers(w, r, s.repOfferScope)
}

// ══════════════════════════════════════════════════════════════════════
// **الإنشاء**
// ══════════════════════════════════════════════════════════════════════

func (s *Server) createScopedOffer(w http.ResponseWriter, r *http.Request,
	who func(*http.Request) (offerActor, bool)) {
	a, ok := who(r)
	if !ok {
		s.respondErr(w, errForbidden)
		return
	}
	in, err := decode[offers.Input](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.offers.CreateScoped(r.Context(), userIDFrom(r), a.merchantID, *in, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ومن فعلها يُقيَّد بدوره** — **«أنزل المندوبُ عرضاً» و«أنزله
	// صاحبُ المتجر» سؤالان مختلفان يُسألان بعد شهر.**
	s.audit(r, "catalog.offer_create", "offer", o.ID, map[string]any{
		"kind": o.Kind, "title": o.Title, "by": a.role,
		"merchant_id": a.merchantID, "percent": o.DiscountPercent,
	})
	s.touch("offer", "ops")
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleMerchantCreateOffer(w http.ResponseWriter, r *http.Request) {
	s.createScopedOffer(w, r, s.merchantOfferScope)
}

func (s *Server) handleRepCreateOffer(w http.ResponseWriter, r *http.Request) {
	s.createScopedOffer(w, r, s.repOfferScope)
}

// ══════════════════════════════════════════════════════════════════════
// **الإيقاف** — **ولا حذف**
// ══════════════════════════════════════════════════════════════════════
//
// **وعرضٌ حُذف لا يُقرأ في تقريرٍ لاحق** — وهي قاعدةُ `SetActive` نفسُها.
//
// **ويُعاد إيقافُه بلا أثرٍ ثانٍ**: **`active = false` مرّتين حالٌ واحدة**
// — **ومن ضغط مرّتين لأنّ الشبكةَ تأخّرت لا يُعاقَب.**
func (s *Server) stopScopedOffer(w http.ResponseWriter, r *http.Request,
	who func(*http.Request) (offerActor, bool)) {
	a, offerID, ok := s.offerInScope(r, who)
	if !ok {
		s.respondErr(w, errForbidden)
		return
	}
	o, err := s.offers.SetActive(r.Context(), offerID, false, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.offer_active", "offer", o.ID, map[string]any{
		"active": false, "by": a.role, "merchant_id": a.merchantID,
	})
	s.touch("offer", "ops")
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleMerchantStopOffer(w http.ResponseWriter, r *http.Request) {
	s.stopScopedOffer(w, r, s.merchantOfferScope)
}

func (s *Server) handleRepStopOffer(w http.ResponseWriter, r *http.Request) {
	s.stopScopedOffer(w, r, s.repOfferScope)
}
