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
// # والمقابلُ يختاره الموظّف — والمطابقةُ تقترح ولا تقرّر
//
// **صنفٌ لا مقابلَ له يُوقف التحويلَ كلَّه ويُقال أيُّه.** وكانت المطابقةُ
// **بالاسم حرفاً بحرف** — فـ«رز مصري» لا يجد «أرز مصري». **وقرارُ المالك
// (٢٠٢٦-١٠-٠٣)**: «المفروض ما يكون نفس الاسم بالضبط». **فصار للتحويل طريقان:**
//
//   - **بمقابلٍ صريح** (`items`): الموظّفُ رأى المقترحَ من
//     `transfer-candidates` (`order_transfer_candidates.go`) **فأكّده أو بدّله
//     بيده** — ويُتحقَّق منه في `transferMapping`.
//   - **وبلا مقابل**: المطابقةُ بالاسم كما كانت — **لمن ينادي المسارَ بالشكل القديم.**
//
// **ولا يُقرّر الخادمُ وحدَه «الأقرب»** — وشاورما بعشرين مكانَ شاورما باثني
// عشر تصل الزبونَ فيعرف الفرقَ ولو لم يعرف السبب. **فالاقتراحُ يُعرض والإنسانُ يقرّر.**

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
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
		// **المقابلُ الذي اختاره الموظّف** — اختياريّ. انظر `transferMapping`.
		Items []transferPick `json:"items"`
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
	//
	// # ولا تحويلَ قبل القبول
	//
	// **التحويلُ علاجٌ بعد التزام**: قبلنا الطلبَ ثمّ تبيّن أنّ المتجر لا
	// يستطيع. **وقبل القبول الأفعالُ ثلاثة**: قبولٌ أو رفضٌ أو إلغاء.
	//
	// **وأخطرُ من الزيادة أنّ التحويلَ يقبل ضمناً**: يردّ هذا المسارُ
	// `StAccepted` — فضغطةٌ على «تحويل» وهو `pending` **تقبل الطلبَ نيابةً عن
	// المالك بلا أن يقول قبلت**، وتبدأ مهلةُ إلغاء الزبون، **ويُحسب القبولُ
	// على المنصة في السجلّ.**
	//
	// شهده المالكُ في شاشته (٢٠٢٦-٠٨-٠٤): «هنا ما في داعي للتحويل لغير متجر
	// لأنّنا أساساً لسّا ما قبلنا الطلب».
	if status != "accepted" && status != "preparing" &&
		status != "dispatching" && status != "assigned" && status != "at_pickup" {
		s.respondErr(w, errTransferTooLate)
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
	if len(req.Items) > 0 {
		// **مقابلٌ اختاره الموظّف** — يُتحقَّق منه ولا يُخمَّن فوقه.
		mapped, miss, err := s.transferMapping(r, tx, orderID, req.MerchantID, req.Items)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		missing = miss
		for _, m := range mapped {
			list = append(list, row{itemID: m.itemID, name: m.name, newID: &m.menuID, newPrice: &m.price})
		}
	} else {
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
	}
	if len(missing) > 0 {
		// **ويُقال أيُّها** — «لا يُطابق» وحدَها تترك الموظّفَ يفتح قائمتين
		// ويقارن بعينه.
		//
		// **والأسماءُ في `error.details.items`** — كان الردُّ يُكتب بـ`httpx.JSON`
		// **فيُدفن الخطأُ في `data`**، فتقرأ اللوحةُ «خطأً داخليّاً» **ولا تعرض
		// الأسماءَ قطّ.** (انظر `httpx.ErrorWith`.)
		httpx.Error(w, &httpx.AppError{
			Status: http.StatusConflict, Code: "transfer_items_unmatched",
			MessageKey: "errors.transfer_items_unmatched",
			Details:    map[string]any{"items": missing},
		})
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
	// ══════════════════════════════════════════════════════════════════
	// **وسائقٌ واقفٌ عند المتجر الأوّل يعود إلى الطريق** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
	// ══════════════════════════════════════════════════════════════════
	//
	// **قِيس على جهازه**: بقي الطلبُ «وصلت المتجر» وهو عند المتجر القديم — **فظهر له
	// «استلمت الطلب» لمتجرٍ لم يصله**، ولا طريقَ إليه. **فيعود «في الطريق للمتجر»**
	// ويبقى الطلبُ معه، ويُرسم طريقُه إلى الجديد ويُسجَّل وصولُه إليه من جديد.
	//
	// **ولا يمرّ بجدول الانتقالات** — `at_pickup → assigned` ليس انتقالاً يطلبه أحد،
	// **وإنّما أثرُ التحويل وحدَه**، فيُكتب هنا ويُسجَّل حدثاً باسمه.
	var driverID *string
	if status == string(orders.StAtPickup) {
		if err := tx.QueryRow(r.Context(), `
			UPDATE orders SET status = 'assigned', updated_at = now(),
			       door_instruction = '', door_instruction_note = '', door_instruction_at = NULL
			WHERE id = $1 RETURNING driver_id::text`, orderID).Scan(&driverID); err != nil {
			s.respondErr(w, err)
			return
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
			VALUES ($1, 'at_pickup', 'assigned', $2, $3)`,
			orderID, actor, "إلى المتجر الجديد بعد التحويل"); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويُخبَر السائقُ باسم المتجر الجديد** — كان اسمُ المتجر يتبدّل في بطاقته صامتاً،
	// **ومن يقود لا يقرأ بطاقته.**
	if driverID == nil {
		_ = s.pg.QueryRow(r.Context(), `SELECT driver_id::text FROM orders WHERE id = $1`,
			orderID).Scan(&driverID)
	}
	if driverID != nil && *driverID != "" {
		var number int64
		var newName string
		_ = s.pg.QueryRow(r.Context(), `
			SELECT o.number, m.name FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1`, orderID).Scan(&number, &newName)
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *driverID, Kind: notifications.KindOrder,
			Title:  notifTitles.driverTransferred,
			Body:   fmt.Sprintf("#%d — اتّجه إلى %s", number, newName),
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		})
		s.hub.Publish("driver:"+*driverID, map[string]any{"type": "order"})
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

// transferPick **صنفُ الطلب ومقابلُه في المتجر الجديد** — كما اختاره الموظّف.
type transferPick struct {
	OrderItemID string `json:"order_item_id"`
	MenuItemID  string `json:"menu_item_id"`
}

type transferMapped struct {
	itemID, name, menuID string
	price                int64
}

var (
	errTransferMapping = httpx.NewError(http.StatusBadRequest,
		"transfer_mapping_invalid", "errors.transfer_mapping_invalid")
	errTransferWrongStore = httpx.NewError(http.StatusUnprocessableEntity,
		"transfer_item_wrong_store", "errors.transfer_item_wrong_store")
	errTransferUnavailable = httpx.NewError(http.StatusConflict,
		"transfer_item_unavailable", "errors.transfer_item_unavailable")
)

// transferMapping **يتحقّق من المقابل الذي اختاره الموظّف** — ويردّه جاهزاً للتحديث.
//
// **وكلُّ شرطٍ يُوقف التحويلَ كلَّه** لا البندَ وحدَه — نصفُ طلبٍ في متجرٍ
// ونصفُه في آخر طلبان لا واحد:
//
//   - **صنفٌ من متجرٍ آخر** (أو لا وجودَ له) ← `422 transfer_item_wrong_store`.
//   - **صنفٌ غيرُ متاحٍ أو غيرُ مُقرّ** ← `409 transfer_item_unavailable` — لا يُشترى
//     ما نفد، **ولا ما لم تراجعه المنصّة.**
//   - **بندٌ من الطلب بلا مقابل** ← يُردّ اسمُه في `missing`، **فيُقال
//     `transfer_items_unmatched` كما في المطابقة بالاسم.**
//   - **معرّفٌ مكرّرٌ أو ليس من هذا الطلب** ← `400 transfer_mapping_invalid`.
//
// **وسعرُ الشراء من الصنف المختار** — وسعرُ الزبون لا يُقرأ هنا أصلاً.
func (s *Server) transferMapping(r *http.Request, tx pgx.Tx, orderID, merchantID string,
	picks []transferPick) ([]transferMapped, []string, error) {
	ctx := r.Context()
	chosen := map[string]string{}
	menuIDs := []string{}
	for _, p := range picks {
		if !isUUID(p.OrderItemID) || !isUUID(p.MenuItemID) {
			return nil, nil, errTransferMapping
		}
		if _, dup := chosen[p.OrderItemID]; dup {
			return nil, nil, errTransferMapping
		}
		chosen[p.OrderItemID] = p.MenuItemID
		menuIDs = append(menuIDs, p.MenuItemID)
	}

	type menuRow struct {
		merchant  string
		available bool
		price     int64
	}
	menu := map[string]menuRow{}
	rows, err := tx.Query(ctx, `
		SELECT id::text, merchant_id::text, available AND approved, merchant_price
		FROM menu_items WHERE id = ANY($1::uuid[])`, menuIDs)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var m menuRow
		if err := rows.Scan(&id, &m.merchant, &m.available, &m.price); err != nil {
			rows.Close()
			return nil, nil, err
		}
		menu[id] = m
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	type orderItem struct{ id, name string }
	items := []orderItem{}
	irows, err := tx.Query(ctx,
		`SELECT id::text, name FROM order_items WHERE order_id = $1 ORDER BY name, id`, orderID)
	if err != nil {
		return nil, nil, err
	}
	for irows.Next() {
		var it orderItem
		if err := irows.Scan(&it.id, &it.name); err != nil {
			irows.Close()
			return nil, nil, err
		}
		items = append(items, it)
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return nil, nil, err
	}

	out := []transferMapped{}
	missing := []string{}
	seen := 0
	for _, it := range items {
		mid, ok := chosen[it.id]
		if !ok {
			missing = append(missing, it.name)
			continue
		}
		seen++
		m, found := menu[mid]
		if !found || m.merchant != merchantID {
			return nil, nil, errTransferWrongStore
		}
		if !m.available {
			return nil, nil, errTransferUnavailable
		}
		out = append(out, transferMapped{itemID: it.id, name: it.name, menuID: mid, price: m.price})
	}
	// **ومعرّفُ بندٍ ليس من هذا الطلب** — خطأُ مُرسِلٍ لا يُتجاهل.
	if seen != len(chosen) {
		return nil, nil, errTransferMapping
	}
	return out, missing, nil
}
