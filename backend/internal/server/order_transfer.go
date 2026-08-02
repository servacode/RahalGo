package server

// **تحويلُ الطلب إلى متجرٍ آخر** — القاعدةُ الاحتياطية.
//
// # متى تُستعمل
//
// المتجرُ يردّ على الواتساب: «هذا الصنفُ غيرُ موجودٍ عندي». **وسياسةُ الإخفاء
// تمنع هذا أصلاً** — صنفٌ غيرُ متاحٍ لا يُعرض. **لكنّ الاحتياطَ يبقى**: قرارُ
// المالك (٢٠٢٦-٠٨-٠٢) «يجب أن نملك دائماً قاعدةً احتياطيةً لشيءٍ طارئ».
//
// # وسعرُ الزبون لا يُمسّ
//
// **الفرقُ على المنصة ولو كان المتجرُ الجديدُ أغلى.** والزبونُ **لا يعلم أنّ
// مصدراً تبدّل** — وهو ما بُني عليه النموذجُ كلُّه: «لن يشعر بأنه اختار من
// مكانين». **وفاتورةٌ تتغيّر بعد الطلب تنقض ذلك في سطر.**
//
// فيُحدَّث `merchant_price` وحدَه — **ما سندفعه للمتجر الجديد** — ويبقى
// `unit_price` كما رآه الزبون. **والهامشُ قد يصير سالباً، وتلك خسارةٌ مقصودة.**
//
// # والمطابقةُ بالاسم — ولا تُخمَّن
//
// **صنفٌ لا يُطابق يُوقف التحويلَ كلَّه ويُقال أيُّه.** والبديلُ أن نُخمّن
// «الأقرب» — **وشاورما بعشرين مكانَ شاورما باثني عشر تصل الزبونَ فيعرف الفرقَ
// ولو لم يعرف السبب.**

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

var (
	errTransferTooLate = httpx.NewError(http.StatusConflict,
		"transfer_too_late", "errors.transfer_too_late")
	errSameMerchant = httpx.NewError(http.StatusConflict,
		"transfer_same_merchant", "errors.transfer_same_merchant")
)

// handleTransferOrder ينقل طلباً إلى متجرٍ آخر قبل خروج البضاعة.
func (s *Server) handleTransferOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		MerchantID string `json:"merchant_id"`
		Note       string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والسببُ إلزاميّ**: تحويلٌ بلا كلمةٍ لا يُقاس ولا يُراجَع، **ولا يُعرف
	// أيُّ متجرٍ يُكثر الاعتذار.**
	note := strings.TrimSpace(req.Note)
	if req.MerchantID == "" || note == "" {
		s.respondErr(w, errValidation)
		return
	}
	orderID := chi.URLParam(r, "id")

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var status, oldMerchant string
	// **القفلُ داخل المعاملة**: تحويلان متزامنان يتركان بنوداً موزّعةً على
	// ثلاثة متاجر.
	if err := tx.QueryRow(r.Context(),
		`SELECT status, merchant_id::text FROM orders WHERE id = $1 FOR UPDATE`,
		orderID).Scan(&status, &oldMerchant); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if oldMerchant == req.MerchantID {
		s.respondErr(w, errSameMerchant)
		return
	}
	// **ولا تحويلَ بعد خروج البضاعة.**
	//
	// المتجرُ الأوّلُ قبض عند الاستلام، **والبضاعةُ في يد السائق** — فتحويلٌ
	// بعدها يعني طلبين لا واحداً: **مالٌ دُفع لمن سلّم، وطعامٌ يُطلب من
	// جديد.** ومن أراد ذلك يُفشل الطلبَ ويُنشئ غيرَه.
	if status != "pending" && status != "accepted" && status != "preparing" &&
		status != "dispatching" && status != "assigned" && status != "at_pickup" {
		s.respondErr(w, errTransferTooLate)
		return
	}

	// **المطابقةُ بالاسم — والفشلُ يُسمّي.**
	// **والمعرّفُ نصٌّ لا رقم**: `order_items.id` من نوع `uuid`. **وقراءتُه في
	// عددٍ صحيحٍ تنهار عند أوّل صفّ** — لا في الترجمة بل في التشغيل، **فيُردّ
	// `500` على تحويلٍ صحيحٍ تماماً.** كشفه فحصٌ حيٌّ لا اختبار.
	rows, err := tx.Query(r.Context(), `
		SELECT oi.id::text, oi.name, ni.id::text, ni.merchant_price
		FROM order_items oi
		LEFT JOIN menu_items ni
		       ON ni.merchant_id = $2 AND ni.name = oi.name AND ni.available
		WHERE oi.order_id = $1`, orderID, req.MerchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	type row struct {
		itemID   string
		name     string
		newID    *string
		newPrice *int64
	}
	list := []row{}
	missing := []string{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.itemID, &x.name, &x.newID, &x.newPrice); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		if x.newID == nil {
			missing = append(missing, x.name)
		}
		list = append(list, x)
	}
	rows.Close()
	if len(missing) > 0 {
		// **ويُقال أيُّها** — «لا يُطابق» وحدَها تترك الموظّفَ يفتح قائمتين
		// ويقارن بعينه.
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{
				"code": "transfer_items_unmatched", "message_key": "errors.transfer_items_unmatched",
				"items": missing,
			}})
		return
	}

	// **سعرُ الشراء وحدَه يُحدَّث** — وسعرُ البيع كما رآه الزبون.
	for _, x := range list {
		if _, err := tx.Exec(r.Context(),
			`UPDATE order_items SET menu_item_id = $2::uuid, merchant_id = $3::uuid,
			        merchant_price = $4 WHERE id = $1::uuid`,
			x.itemID, *x.newID, req.MerchantID, *x.newPrice); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if _, err := tx.Exec(r.Context(),
		`UPDATE orders SET merchant_id = $2, sent_to_merchant_at = NULL,
		        updated_at = now() WHERE id = $1`, orderID, req.MerchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والحدثُ يُكتب في مسار الطلب** — فمن قرأه بعد شهرٍ عرف لماذا تبدّل
	// المصدر، **ولا يجد متجراً في الفاتورة وآخرَ في الدفتر بلا تفسير.**
	actor := userIDFrom(r)
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $2, $3, $4)`,
		orderID, status, actor, "تحويلٌ إلى متجرٍ آخر — "+note); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "ops.order_transferred", "order", orderID, map[string]any{
		"from": oldMerchant, "to": req.MerchantID, "note": note,
	})
	s.touch("order", "ops")
	// **والتبليغُ يعود من الصفر**: `sent_to_merchant_at` صُفّر، **فزرُّ
	// «تحويل للمتجر» يظهر من جديد** — والمتجرُ الجديدُ لم يُبلَّغ بعد.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"transferred": true, "items": len(list), "status": orders.StAccepted,
	})
}
