package server

// **مراجعةُ القائمة قبل النشر — إعدادٌ كان يَعِد ولا يفعل.**
//
// # الحادثة
//
// `merchants.menu_requires_approval` مفتاحٌ مُعرَّفٌ ومحفوظٌ منذ الهجرة ٠٠٠٩،
// **ولا يقرؤه سطرٌ واحدٌ في المشروع.** فُحصت الأربعون مفتاحاً واحداً واحداً
// (٢٠٢٦-٠٨-٠٤) — **وهو الوحيدُ الميّت.**
//
// **وإعدادٌ يكذب أخطرُ من إعدادٍ غائب**: الغائبُ يُسأل عنه، **والكاذبُ يُبنى
// عليه.**
//
// # وما يُراجَع وما لا يُراجَع
//
//	يُراجَع     ←  الاسمُ · الوصفُ · السعرُ · الصورةُ · التصنيف — **ما يراه الزبون**
//	لا يُراجَع  ←  **الإتاحة**: «نفد الصنف» قرارُ مطبخٍ في لحظته
//
// **ومراجعةُ الإتاحة تجعل المتجرَ يبيع ما نفد حتى نستيقظ** — وهو أسوأُ ممّا
// تحرسه المراجعةُ أصلاً.
//
// # والأدمنُ لا يُراجَع نفسَه
//
// الحارسُ في باب المتجر لا في `catalog`: **الأدمنُ هو المُراجِع**، وما يكتبه
// منشورٌ لحظتَه. ولو وُضع في المحرّك لَاحتاج معاملاً يقول «من أنت» يمرّ عبر
// أربع دوالّ ليصل — **وهي علّةٌ وقعنا فيها قبلاً.**

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// menuNeedsApproval أمفتاحُ المراجعة مرفوع؟
func (s *Server) menuNeedsApproval(r *http.Request) bool {
	return s.settings != nil &&
		s.settings.GetBool(r.Context(), "merchants.menu_requires_approval")
}

// touchesContent أيمسّ هذا التعديلُ ما يراه الزبون؟
//
// **والإتاحةُ وحدَها لا تمسّه** — فمن أطفأ صنفاً نفد لا يُعلَّق صنفُه للمراجعة.
func touchesContent(in catalog.MenuItemInput) bool {
	return in.Name != nil || in.Description != nil || in.Price != nil ||
		in.ImageMediaID != nil || in.PlatformSectionID != nil ||
		in.MarginOverride != nil || in.SectionID != nil || in.Modifiers != nil
}

// holdForReview يُنزل عَلَم النشر عن صنفٍ ويُخطر المكتب.
//
// **ويُنادى بعد نجاح الكتابة لا داخلها**: تعثّرُ الإشعار لا يُبطل تعديلاً وقع.
func (s *Server) holdForReview(r *http.Request, itemID string) {
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE menu_items SET approved = false, review_note = '' WHERE id = $1`,
		itemID); err != nil {
		s.logger.Error("menu approval: hold", "error", err, "item", itemID)
		return
	}
	// **والمكتبُ يُخبَر** — طابورٌ لا يعلم به أحدٌ طابورٌ لا يُفرَغ.
	s.notify.NotifyRoles(r.Context(), notifications.OpsDesk, notifications.Input{
		Kind: notifications.KindAccount, Title: notifMenuPending,
		Entity: "menu_item", EntityID: itemID, Href: "/dashboard/sections",
	})
	s.touch("menu", "ops")
}

const (
	notifMenuPending  = "صنفٌ ينتظر المراجعة"
	notifMenuApproved = "نُشر صنفُك"
	notifMenuRejected = "رُدّ صنفُك"
)

type pendingItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        int64     `json:"merchant_price"`
	MerchantID   string    `json:"merchant_id"`
	MerchantName string    `json:"merchant_name"`
	SectionName  string    `json:"section_name"`
	ThumbURL     *string   `json:"thumb_url"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// handlePendingMenuItems طابورُ المراجعة — **بالأقدم أوّلاً.**
//
// **والقسمُ قسمُ السوق** — لا الجدولَ الذي رفعته هجرةُ ٠٠٨٤.
//
// كان يضمّ menu_sections على i.section_id، **وهو عمودٌ لا يُملأ منذ الهجرة**
// — فاسمُ القسم يخرج فارغاً لكلّ صنفٍ جديد، **وطابورُ المراجعة يعرض صفّاً
// بلا قسم** فلا يعرف المراجعُ ما يراجع.
func (s *Server) handlePendingMenuItems(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT i.id::text, i.name, i.description, i.merchant_price,
		       m.id::text, m.name, COALESCE(ps.name, ''), im.thumb_path, i.updated_at
		FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		LEFT JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media im ON im.id = i.image_media_id
		WHERE NOT i.approved
		ORDER BY i.updated_at
		LIMIT 200`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []pendingItem{}
	for rows.Next() {
		var it pendingItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.Price,
			&it.MerchantID, &it.MerchantName, &it.SectionName,
			&it.ThumbURL, &it.UpdatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, it)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
}

// handleReviewMenuItem إقرارُ صنفٍ أو ردُّه.
//
// **والردُّ يلزمه كلمة**: «رُفض» بلا سببٍ يُعاد إرسالُه كما هو، **فيدور المتجرُ
// والمكتبُ في حلقة.**
func (s *Server) handleReviewMenuItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	req, err := decode[struct {
		Approve bool   `json:"approve"`
		Note    string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if !req.Approve && note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}

	// **ويبقى غيرَ منشورٍ عند الردّ** — لا يُحذف: عملُ المتجر لا يُمحى، يُعاد
	// إليه ليصحّحه. وأوّلُ تعديلٍ بعده يُعيده إلى الطابور.
	var ownerID *string
	var name string
	if err := s.pg.QueryRow(r.Context(), `
		UPDATE menu_items i SET approved = $2, review_note = $3, updated_at = now()
		FROM merchants m
		WHERE i.id = $1 AND m.id = i.merchant_id
		RETURNING m.owner_user_id::text, i.name`,
		itemID, req.Approve, note).Scan(&ownerID, &name); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **وصاحبُه يعرف** — ومن لا يعرف أنّ صنفَه معلّقٌ يظنّ أنّ النظام أعطبه.
	if ownerID != nil {
		title := notifMenuApproved
		if !req.Approve {
			title = notifMenuRejected
		}
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *ownerID, Kind: notifications.KindAccount,
			Title: title, Body: name + " — " + note,
			Entity: "menu_item", EntityID: itemID, Href: "/portal/menu",
		})
	}
	s.audit(r, "admin.menu_reviewed", "menu_item", itemID, map[string]any{
		"approve": req.Approve, "note": note,
	})
	s.touch("menu", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"approved": req.Approve})
}
