package server

// سجلُّ السائق — **ما نفّذه، نجح أم فشل.**
//
// # المسألة
//
// شاشتُه كانت «الآن» وحدَه: مهامٌّ جارية وطابورٌ قادم. **وما انتهى اختفى.**
//
// فلا يعرف كم سلّم أمس، ولا يجد طلباً يتذكّره ليقول فيه شيئاً، **ولا يملك أن
// يُبلّغ عن متجرٍ أوقفه نصفَ ساعةٍ أو زبونٍ أعطاه عنواناً وهمياً** — لأنّ
// الطلبَ لم يعد له وجودٌ في تطبيقه.
//
// **وحسابٌ يُقفل بلا سجلٍّ يُقرأ ظلمٌ صامت**: يُخصم منه في آخر الشهر عن طلبٍ
// لا يستطيع أن يراه.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «لازم يكون عندو قسم يشوف طلباته... في حال النجاح
// أو الفشل».)
//
// # والمغلقةُ وحدَها
//
// الجاريةُ في «مهامّي» — **وشاشتان تعرضان الشيءَ نفسَه تفترقان في زرّ**، ثمّ
// يضغط السائقُ على ما لم يعد قائماً.

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// handleDriverHistory طلباتُه المنتهية — الأحدثُ أوّلاً.
func (s *Server) handleDriverHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		DriverID:   userIDFrom(r),
		ClosedOnly: true,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **وأيُّها قيّمتَه سلفاً** — زرٌّ يُعرض على ما قُيّم يُضغط فيُردّ،
	// **ورفضٌ بعد ضغطةٍ يُقرأ عطباً لا قاعدة.**
	rated := map[string]bool{}
	ids := make([]string, 0, len(res.Orders))
	for i := range res.Orders {
		ids = append(ids, res.Orders[i].ID)
	}
	if len(ids) > 0 {
		rows, err := s.pg.Query(r.Context(),
			`SELECT order_id::text FROM merchant_ratings WHERE order_id = ANY($1::uuid[])`, ids)
		if err == nil {
			for rows.Next() {
				var id string
				if rows.Scan(&id) == nil {
					rated[id] = true
				}
			}
			rows.Close()
		}
	}
	// **ومسافةُ كلّ طلبٍ قاده** (قرارُ المالك ٢٠٢٦-١١-٠٥) — «المسافة: ٣ كم»، من كانس المسافات.
	// **ومسافتُه هو** لا مسافةُ غيره إن تبدّل السائق.
	type drive struct{ toPickup, withOrder int }
	drives := map[string]drive{}
	if len(ids) > 0 {
		// **وطلبٌ امتدّ على دوامين** له قطعتان — مسافتُه مجموعُهما.
		rows, err := s.pg.Query(r.Context(), `
			SELECT order_id::text, sum(to_pickup_m)::int, sum(with_order_m)::int FROM order_drives
			 WHERE driver_id = $1 AND order_id = ANY($2::uuid[])
			 GROUP BY order_id`, userIDFrom(r), ids)
		if err == nil {
			for rows.Next() {
				var id string
				var d drive
				if rows.Scan(&id, &d.toPickup, &d.withOrder) == nil {
					drives[id] = d
				}
			}
			rows.Close()
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **والسجلُّ بعين السائق لا بعين المدير** (فحصُ دورة السائق ٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان يُرسَل صفُّ الطلب كاملاً** — وفيه `customer_phone` ومعرّفُ الزبون وإحداثيّاتُ
	// بابه. **وقرارُ المالك (٢٠٢٦-٠٨-٠٩): «ولا رقمَ زبونٍ يصل إلى سائق».** فيمرّ بقائمة
	// السائق (`orders.ViewFor`) — **ثمّ بقائمة السجلّ فوقها** (`historyKeep`).
	out := make([]map[string]any, len(res.Orders))
	for i := range res.Orders {
		o := res.Orders[i]
		full := orders.ViewFor(orders.AudienceDriver, &o)
		v := make(map[string]any, len(historyKeep)+5)
		for k, x := range full {
			if historyKeep[k] {
				v[k] = x
			}
		}
		// **وما تحتاجه شاشةُ السجلّ ممّا ليس في قائمة السائق.**
		v["merchant_name"] = o.MerchantName
		v["number"] = o.Number
		v["total"] = o.Total
		v["merchant_rated"] = rated[o.ID]
		if d, ok := drives[o.ID]; ok {
			v["drive_m"] = d.toPickup + d.withOrder
			v["drive_to_pickup_m"] = d.toPickup
			v["drive_with_order_m"] = d.withOrder
		}
		out[i] = v
		// **وشرطُ التقييم هو شرطُ الخادم نفسُه** — نسخةٌ ثانيةٌ تفترق فيُعرض زرٌّ يُردّ.
		// **ولا يُقيَّم متجرٌ لا وجودَ له** (سأل المالك ٢٠٢٦-٠٨-١٣): الخاصُّ بلا متجرٍ يمرّ
		// بـ`picked_up` عند الشراء، **فكان يصير مؤهَّلاً لتقييم من لا وجودَ له ويُردّ بـ٤٠٤.**
		v["can_rate_merchant"] = o.MerchantID != "" &&
			(o.PickedUpAt != nil ||
				(o.Status == "failed" && o.Fault == "merchant"))
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": out, "total": res.Total, "page": res.Page, "per_page": res.PerPage,
	})
}

// handleDriverReportReasons ما يملك السائقُ اختيارَه، وعلى من هو.
//
// **والقائمةُ من الخادم** — لا تُكتب في التطبيق: قائمةٌ في مكانين تفترق حين
// يُضاف سببٌ في أحدهما، **فيرسل التطبيقُ رمزاً لا يعرفه الخادم** ويُردّ عليه
// بلا أن يفهم لماذا.
// **وتُصفّى بحسب الطلب إن قيل أيُّه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣).
//
// **والطلبُ الخاصُّ بلا متجر**، فأسبابُ المتجر فيه سؤالٌ عمّا لا وجودَ
// له. **وبلا معرّفٍ تُردّ كلُّها** — فلا ينكسر نداءٌ قديمٌ لا يرسله.
func (s *Server) handleDriverReportReasons(w http.ResponseWriter, r *http.Request) {
	kind := ""
	if id := r.URL.Query().Get("order"); id != "" {
		// **وطلبُ غيره لا يُقرأ** — الشرطُ على السائق أيضاً.
		_ = s.pg.QueryRow(r.Context(),
			`SELECT o.kind FROM orders o WHERE o.id = $1 AND o.driver_id = $2`,
			id, userIDFrom(r)).Scan(&kind)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": support.ReasonsForKind(kind)})
}

// handleDriverReport يفتح بلاغَ سائقٍ على متجرٍ أو زبون.
func (s *Server) handleDriverReport(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وملاحظةُ البلاغ تمرّ بالحارس** — الشتيمةُ تُخفى وتُنبَّه بها الإدارة (`text_limits.go`).
	masked, err := s.guardText(r.Context(), tf("note", &req.Note, maxComplaintNote, textguard.Complaint))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	t, err := s.support.DriverReport(r.Context(), userIDFrom(r), orderID, req.Reason, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.alertOffensive(r.Context(), masked, t.Subject, "ticket", t.ID, "/dashboard/tickets")
	// **والعملياتُ تُنبَّه** — بلاغٌ لا يراه أحدٌ حتى يفتح الشاشةَ صدفةً بلاغٌ
	// لم يُقدَّم.
	s.touch("ticket", "ops")
	// **ردٌّ أدنى للمُبلِّغ** (خصوصيّة، Batch 4): إقرارُ تسجيلٍ — المعرّفُ
	// والرقمُ والحالُ لا صفُّ المكتب الكامل: لا هاتفَ زبونٍ ولا اسمَه ولا
	// معرّفَ من أُبلِغ عنه. صفُّ الهويّة الكامل لبابِ الأدمن وحدَه.
	httpx.JSON(w, http.StatusOK, map[string]any{"id": t.ID, "number": t.Number, "status": t.Status})
}

// historyKeep **ما يبقى عند السائق من طلبٍ انتهى — بالسماح لا بالمنع.**
//
// (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٢.) **كانت قائمةَ منعٍ من ستّة حقول** — فبقي الاسمُ
// والعنوانُ والملاحظاتُ ونصُّ الطلب الخاصّ وصاحبُ الأمانة وبابُ البناية وصورةُ الباب:
// **سجلٌّ يُقرأ قائمةَ زبائن.** **والقائمةُ في المحرّك** (`orders.DriverFinishedKeep`) —
// **وبثُّ غرفته يقرؤها نفسَها**، فلا يصله من التحديث ما نُزع من السجلّ. **ونسخُ السائق
// القديمة لا تنكسر**: حقولُ `HistoryOrder` كلُّها بافتراض، وشاشةُ السجلّ لا تعرض الاسمَ
// ولا العنوان.
var historyKeep = orders.DriverFinishedKeep
