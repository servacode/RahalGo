package server

/*
**حديثُ الطلب — بابٌ واحدٌ لطرفيه.**

(قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
 المنصّة فقط».)

# ولا مسارَ لكلّ دور

**المسارُ واحدٌ يناديه الزبونُ والسائق** — والخادمُ يعرف أيَّهما من جلسته.

**ومساران متطابقان يفترقان**: يُصلَح شرطٌ في أحدهما ويُنسى الآخر، **فيقرأ
السائقُ ما لا يقرؤه الزبونُ من الحديث نفسِه.** وقد وقع في هذا المشروع من قبل
(محفظةٌ لكلّ دور تفعل الشيءَ نفسَه حرفاً بحرف).

# ولا يُقبل معرّفُ طرفٍ في أيّ نداء

**العنوانُ `/orders/{id}/messages` لا `/messages/to/{user}`.** والخادمُ يستخرج
الطرفَ الآخر من الطلب. **فلا يستطيع أحدٌ أن يبلغ من يشاء** بتبديل رقمٍ في نداء
— وهو أوّلُ ما يُجرَّب.
*/

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// handleOrderMessages **يقرأ الحديثَ ويَسِمُ ما وصل.**
//
// **والوسمُ مع القراءة لا بنداءٍ ثانٍ**: من فتح الشاشةَ قرأ، **ونداءٌ ثانٍ
// ليقول «قرأتُ» رحلةٌ زائدةٌ تسقط على شبكةٍ ضعيفة** فتبقى الشارةُ حمراءَ لمن
// قرأ.
func (s *Server) handleOrderMessages(w http.ResponseWriter, r *http.Request) {
	p, err := s.comms.Permit(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := s.comms.List(r.Context(), p)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if n, err := s.comms.MarkRead(r.Context(), p); err == nil && n > 0 {
		// **والطرفُ الآخر يرى «قُرئت»** — فلا يعيد ما وصل.
		s.touch("order", p.PeerID)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"messages": list,
		// **واسمُ الطرف الآخر ولا رقمَ معه** — وهو كلُّ ما يحتاجه من يكتب.
		"peer_name": p.PeerName,
		"open":      p.Open,
		"closes_at": p.ClosesAt,
	})
}

// handleSendOrderMessage **يكتب سطراً ويُشعر الطرفَ الآخر.**
func (s *Server) handleSendOrderMessage(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, err := s.comms.Permit(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	msg, err := s.comms.Send(r.Context(), p, req.Body)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **والإشعارُ إلى الطرف الآخر وحدَه** — يُستخرج من الصلاحية لا من الجسد.
	//
	// **ولا يحمل رقماً**: عنوانُه دورُ المرسِل ونصُّه ما كتب، **والوجهةُ
	// الطلبُ لا الشخص.**
	title := notifTitles.messageFromDriver
	href := "/orders/" + p.OrderID
	if p.Me == comms.RoleCustomer {
		title = notifTitles.messageFromCustomer
		href = "/portal"
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: p.PeerID, Kind: notifications.KindOrder,
		Title: title, Body: clip(msg.Body, 120),
		Entity: "order", EntityID: p.OrderID, Href: href,
	})
	s.touch("order", p.PeerID)
	httpx.JSON(w, http.StatusCreated, msg)
}

// handleMyChats **سجلُّ محادثاتي — المفتوحةُ والمنتهية.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «دردشاتي السابقة… مشان إثبات».)
func (s *Server) handleMyChats(w http.ResponseWriter, r *http.Request) {
	list, err := s.comms.Threads(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"threads": list})
}

// handleAdminOrderChat **حديثُ الطلب للمراجعة — يُقرأ ولا يُكتب.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «يجب أن نضيف دردشات الزبائن والسائقين — في حال
//
//	حصول أيّ تجاوزٍ يمكننا الرجوع إليه».)
//
// **وشكوى «قال لي كذا» كانت كلمةً ضدّ كلمة** — والحديثُ مكتوبٌ في القاعدة
// **ولا بابَ إليه من لوحة الإدارة.** فتحكم العملياتُ بين اثنين لا تملك عن
// أيّهما شيئاً.
func (s *Server) handleAdminOrderChat(w http.ResponseWriter, r *http.Request) {
	th, err := s.comms.Audit(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, th)
}
