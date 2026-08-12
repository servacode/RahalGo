package orders

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

const orderSelect = `
	SELECT o.id, o.number, o.customer_id, cu.phone, cu.full_name,
	       -- **ونوعُ الطلب** — الشاشاتُ تعرض به، والخارطةُ تختلف عليه.
	       o.kind, COALESCE(o.custom_request, ''),
	       o.custom_goods_amount, o.custom_fee,
	       -- **وفراغٌ لا NULL** — (الطلبُ الخاصّ ٢٠٢٦-٠٨-٠٩): الحقلان نصّان
	       -- في البنية، **وNULL فيهما يُسقط المسحَ كلَّه** لا هذا الحقلَ وحدَه.
	       COALESCE(o.merchant_id::text, ''), COALESCE(mr.name, ''),
	       o.driver_id, dr.phone, NULLIF(dr.full_name, ''),
	       o.status, o.address_text,
	       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry),
	       o.zone_id, z.name,
	       o.payment_method, o.subtotal, o.delivery_fee, o.discount, o.total,
	       o.wallet_paid, o.cash_due, o.promo_code, o.notes, o.cancel_reason, o.created_at,
	       o.sent_to_merchant_at, o.dispatched_at,
	       -- **ومن عُرض عليه ولم يقبل بعد** — يُعرض ما دام العرضُ حيّاً.
	       CASE WHEN o.offer_expires_at > now() THEN od.full_name END,
	       pm.path, o.pod_taken_at,
	       -- **المسافةُ تُقاس ساعةَ السؤال من نقطتين محفوظتين.**
	       COALESCE(ST_Distance(o.pod_at, o.dropoff), -1),
	       o.pod_skip_reason,
	       COALESCE(o.ended_by,''), COALESCE(o.fault,''), COALESCE(o.fail_reason,''),
	       o.returned_at, o.goods_settled_to,
	       -- **وزمنا الطريق كما قالتهما الخريطةُ في لحظتيهما** — ومنهما
	       -- تُقاس مهلةُ الخطّين، **ولا يُقاسان من خطٍّ مستقيمٍ يقصّر
	       -- الطريقَ نصفَه فيتّهم سائقاً سار صحيحا.**
	       o.to_store_eta_sec, o.to_door_eta_sec,
	       -- **طولُ المشوار وبُعدُ السائق** — جوابُ «لماذا تأخّر» و«لماذا هو».
	       -- وسالبٌ يعني «لا يُعرف»: **الجهلُ ليس قرباً.**
	       COALESCE(ST_Distance(COALESCE(o.pickup_override, mr.location), o.dropoff), -1),
	       COALESCE(ST_Distance(
	           (SELECT du.last_location FROM users du
	            WHERE du.id = o.driver_id
	              AND du.last_location_at > now() - interval '15 minutes'),
	           COALESCE(o.pickup_override, mr.location)), -1),
	       -- **أيستردّ كلُّ مصدرٍ في هذا الطلب؟** — bool_and لا bool_or:
	       -- الزرُّ يسترجع من الجميع، **ومن لا يستردّ لا يُسترجع منه.**
	       COALESCE((SELECT bool_and(m2.accepts_returns) FROM merchants m2
	                 WHERE m2.id IN (SELECT COALESCE(oi.merchant_id, o.merchant_id)
	                                 FROM order_items oi WHERE oi.order_id = o.id
	                                 UNION SELECT o.merchant_id)), mr.accepts_returns,
	                -- **والطلبُ الخاصُّ لا متجرَ له فلا استرجاع** — (٢٠٢٦-٠٨-٠٩).
	                --
	                -- **وكانت الطبقةُ الأخيرةُ تنتهي إلى عمود المتجر وهو NULL
	                -- بلا متجر** — فيسقط مسحُ الصفّ كلِّه لا هذا الحقلَ وحدَه.
	                false),
	       o.prep_minutes, o.ready_at, o.accepted_at, o.picked_up_at, o.delivered_at,
	       -- **ومتى أُقفل** — يُكتب في القاعدة عند كلّ نهاية ولم يكن يُقرأ.
	       -- (انظر Order.ClosedAt.)
	       o.closed_at,
	       lm.thumb_path,
	       -- ملخّص الأصناف في القائمة نفسها: «ماذا طلبتُ؟» أول سؤال يسأله صاحب
	       -- الطلب، وكان يلزمه فتح الطلب ليعرف. العدد بالكمّيات لا بالأسطر
	       -- (صنفان من الشيء نفسه سطرٌ واحد وقطعتان)، والمعاينة أول ثلاثة أسماء.
	       COALESCE((SELECT sum(oi.qty) FROM order_items oi WHERE oi.order_id = o.id), 0),
	       COALESCE((SELECT string_agg(x.name, '، ' ORDER BY x.rn)
	                 FROM (SELECT oi.name, row_number() OVER (ORDER BY oi.name) AS rn
	                       FROM order_items oi WHERE oi.order_id = o.id LIMIT 3) x), ''),
	       -- **الأصناف كاملةً في القائمة نفسها.**
	       --
	       -- كانت المعاينةُ ثلاثةَ أسماء بلا كمّيات ولا خيارات، فتُضطر غرفةُ
	       -- العمليات إلى فتح كل طلبٍ لترى ما فيه — وهي تنظر إلى عشرين طلباً
	       -- في الساعة. وجلبُها بنداءٍ لكل بطاقة يعني عشرين نداءً لصفحةٍ واحدة،
	       -- فتُجمَع هنا في استعلامٍ واحد.
	       COALESCE((SELECT json_agg(json_build_object(
	                          'id', oi.id, 'menu_item_id', oi.menu_item_id,
	                          'name', oi.name, 'unit_price', oi.unit_price,
	                          'qty', oi.qty, 'note', oi.note,
	                          'options', COALESCE(oi.options, '[]'::jsonb))
	                        ORDER BY oi.id)
	                 FROM order_items oi WHERE oi.order_id = o.id), '[]'::json)
	FROM orders o
	JOIN users cu ON cu.id = o.customer_id
	-- **والمتجرُ يُضمّ يساراً** — (الطلبُ الخاصّ ٢٠٢٦-٠٨-٠٩): لا متجرَ له.
	--
	-- **وضمٌّ صلبٌ يُسقطه من كلّ قراءة** — لا يُخطئ ولا يُنذر، **إنّما يختفي
	-- الطلبُ كأنّه لم يكن.** (ووقع: أوّلُ طلبٍ خاصٍّ أُنشئ ثمّ ردّ الخادمُ
	-- «غير موجود» — وهو مكتوبٌ في القاعدة.)
	LEFT JOIN merchants mr ON mr.id = o.merchant_id
	LEFT JOIN media lm ON lm.id = mr.logo_media_id
	LEFT JOIN users dr ON dr.id = o.driver_id
	LEFT JOIN users od ON od.id = o.offered_driver_id
	LEFT JOIN delivery_zones z ON z.id = o.zone_id
	LEFT JOIN media pm ON pm.id = o.pod_media_id`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order
	var items []byte
	err := row.Scan(&o.ID, &o.Number, &o.CustomerID, &o.CustomerPhone, &o.CustomerName,
		&o.Kind, &o.CustomRequest, &o.CustomGoodsAmount, &o.CustomFee,
		&o.MerchantID, &o.MerchantName, &o.DriverID, &o.DriverPhone, &o.DriverName,
		&o.Status, &o.AddressText, &o.Lat, &o.Lng, &o.ZoneID, &o.ZoneName,
		&o.PaymentMethod, &o.Subtotal, &o.DeliveryFee, &o.Discount, &o.Total,
		&o.WalletPaid, &o.CashDue, &o.PromoCode, &o.Notes, &o.CancelReason, &o.CreatedAt,
		&o.SentToMerchantAt, &o.DispatchedAt, &o.OfferedDriverName,
		&o.ProofURL, &o.ProofTakenAt, &o.ProofMeters, &o.ProofSkipReason,
		&o.EndedBy, &o.Fault, &o.FailReason, &o.ReturnedAt, &o.GoodsSettledTo,
		&o.ToStoreETASec, &o.ToDoorETASec,
		&o.LegM, &o.DriverToPickupM,
		&o.AcceptsReturns,
		&o.PrepMinutes, &o.ReadyAt, &o.AcceptedAt, &o.PickedUpAt, &o.DeliveredAt, &o.ClosedAt,
		&o.MerchantLogoThumb, &o.ItemsCount, &o.ItemsPreview, &items)
	if err != nil {
		return nil, err
	}
	// أصنافٌ لا تُفكّ لا تُسقط الطلب: البطاقة تعرض ما بقي وتُخفي القائمة وحدها.
	_ = json.Unmarshal(items, &o.Items)
	// بادئة "/media/" تُضاف هنا مرّة واحدة لكل قارئ للطلبات (زبون/متجر/إدارة/سائق)
	// بدل أن يتذكّرها كل معالِج على حدة — ونسيانُها يعني صورةً لا تظهر.
	o.MerchantLogoThumb = media.URLForPtr(o.MerchantLogoThumb)
	// **ومسارُ صورة الإثبات يصير رابطاً** — كسائر الوسائط.
	o.ProofURL = media.URLForPtr(o.ProofURL)
	return &o, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Order, error) {
	o, err := scanOrder(s.db.QueryRow(ctx, orderSelect+` WHERE o.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// **تقديرُ الطريق من الإعدادات** — كان رقماً مكتوباً في شاشة الزبون.
	o.DeliveryEstimateMin = 15
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "orders.delivery_estimate_min"); v > 0 {
			o.DeliveryEstimateMin = int(v)
		}
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, menu_item_id, name, unit_price, qty, note, options
		FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, err
	}
	o.Items = []OrderItem{}
	for rows.Next() {
		var it OrderItem
		var opts []byte
		if err := rows.Scan(&it.ID, &it.MenuItemID, &it.Name, &it.UnitPrice, &it.Qty, &it.Note, &opts); err != nil {
			rows.Close()
			return nil, err
		}
		it.Options = []OptionSnapshot{}
		_ = json.Unmarshal(opts, &it.Options)
		o.Items = append(o.Items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	eRows, err := s.db.Query(ctx, `
		SELECT from_status, to_status, actor_id, note, created_at
		FROM order_events WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()
	o.Events = []Event{}
	for eRows.Next() {
		var e Event
		if err := eRows.Scan(&e.FromStatus, &e.ToStatus, &e.ActorID, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		o.Events = append(o.Events, e)
	}
	if err := eRows.Err(); err != nil {
		return nil, err
	}
	o.Rating = s.ratingFor(ctx, id)
	// **ومرحلتُه تُحسب هنا لا في الشاشة** — موضعٌ واحدٌ لثلاثِ شاشات.
	o.SetStage()
	// **وأحداثُه محمّلةٌ فوق** — فلا استعلامَ ثانٍ لأوقاته.
	o.SetStageTimes(o.Events, s.stageLimits(ctx))
	return o, nil
}

type ListFilter struct {
	Status     string
	MerchantID string
	CustomerID string
	DriverID   string
	Query      string // رقم طلب أو هاتف زبون
	OpenOnly   bool   // الطلبات الجارية فقط
	// ClosedOnly المنتهيةُ وحدَها — **سجلٌّ لا شاشةَ متابعة.**
	//
	// يُستعمل حين تدير المنصةُ الطلبات: **المتجرُ لا يملك زرّاً في الجارية**،
	// وشاشةٌ تُشاهَد ولا تُلمَس تُربك أكثرَ ممّا تُفيد. **وسؤالُه الحقيقيّ
	// «ماذا بعتُ اليومَ وبكم؟» — وجوابُه في السجلّ.**
	ClosedOnly bool
	Page       int
	PerPage    int
}

// fillStageTimes يملأ أوقاتَ المراحل لصفحةِ طلباتٍ كاملة.
//
// ══════════════════════════════════════════════════════════════════════
// **واستعلامٌ واحدٌ للصفحة لا استعلامٌ لكلّ صفّ**
// ══════════════════════════════════════════════════════════════════════
//
// **خمسون طلباً في الصفحة**، ونداءُ أحداثِ كلٍّ على حدة **خمسون رحلةً
// إلى القاعدة** لبطاقاتٍ تُعرض معاً. وهي عائلةُ العطب التي تُسمّى
// «N+1»: **لا تُرى في التطوير حيث الصفّان صفّان**، وتُرى في ذروةٍ فيها
// مئة.
//
// **فمعرّفاتُ الصفحة تُمرَّر دفعةً** (`= ANY`) — رحلةٌ واحدة.
//
// **والترتيبُ بالمعرّف لا بالوقت**: طابعُ الوقت قد يتساوى في حدثين وقعا
// في الثانية نفسِها — **واستلامُ السائق وانطلاقُه يقعان بضغطةٍ واحدة**
// فيتساويان. **والمعرّفُ متسلسلٌ لا يتساوى**، فيحفظ ترتيبَ الوقوع.
func (s *Service) fillStageTimes(ctx context.Context, orders []Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
	}
	rows, err := s.db.Query(ctx, `
		SELECT order_id::text, to_status, created_at
		FROM order_events WHERE order_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	byOrder := map[string][]Event{}
	for rows.Next() {
		var id string
		var e Event
		if err := rows.Scan(&id, &e.ToStatus, &e.CreatedAt); err != nil {
			return err
		}
		byOrder[id] = append(byOrder[id], e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// **والمهلُ تُقرأ مرّةً للصفحة كلِّها** — قيمةٌ واحدةٌ لخمسين صفّا،
	// **وقراءتُها في الحلقة خمسون نداءً لمخزنٍ لا يتبدّل بينها.**
	lim := s.stageLimits(ctx)
	for i := range orders {
		orders[i].SetStageTimes(byOrder[orders[i].ID], lim)
	}
	return nil
}

// stageLimits المهلُ كما ضبطها المالك من لوحته.
//
// **ولا رقمَ مكتوبٌ هنا** — الافتراضاتُ في فهرس الإعدادات، **وموضعان
// يحملان رقماً واحداً يفترقان يوماً.**
//
// **وبلا مخزنٍ لا حكم**: مهلٌ أصفارٌ تعني «لا علامةَ على أيّ خطّ» —
// **وهو الصمتُ الصحيح** حين لا تُعرف القواعد، لا اتّهامٌ ولا تبرئة.
func (s *Service) stageLimits(ctx context.Context) StageLimits {
	if s.settings == nil {
		return StageLimits{}
	}
	return StageLimits{
		Accept:   int(s.settings.GetInt(ctx, "orders.accept_timeout_min")),
		Prep:     0, // **يُملأ من `prep_minutes` لكلّ طلبٍ على حدة.**
		Driver:   int(s.settings.GetInt(ctx, "orders.driver_timeout_min")),
		Handover: int(s.settings.GetInt(ctx, "orders.handover_timeout_min")),
		// **ومهلتا الطريق تُبنيان لكلّ طلبٍ من خريطته** — وهذا هامشُهما.
		RouteMarginPct: int(s.settings.GetInt(ctx, "orders.route_margin_pct")),
	}
}

func (s *Service) List(ctx context.Context, f ListFilter) (*OrderPage, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}
	where := ` WHERE ($1 = '' OR o.status = $1)
		AND ($2 = '' OR o.merchant_id::text = $2)
		AND ($3 = '' OR o.customer_id::text = $3)
		AND ($4 = '' OR o.driver_id::text = $4)
		AND ($5 = '' OR o.number::text = $5 OR cu.phone ILIKE '%'||$5||'%')
		AND (NOT $6 OR o.closed_at IS NULL)
		AND (NOT $7 OR o.closed_at IS NOT NULL)`

	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM orders o JOIN users cu ON cu.id = o.customer_id`+where,
		f.Status, f.MerchantID, f.CustomerID, f.DriverID, f.Query, f.OpenOnly,
		f.ClosedOnly).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, orderSelect+where+`
		ORDER BY o.created_at DESC LIMIT $8 OFFSET $9`,
		f.Status, f.MerchantID, f.CustomerID, f.DriverID, f.Query, f.OpenOnly,
		f.ClosedOnly, f.PerPage, (f.Page-1)*f.PerPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		// **ومرحلتُه معه في القائمة** — البطاقةُ ترسم شريطَها منها،
		// **ولا تطوي الحالاتِ بجدولٍ عندها.**
		o.SetStage()
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.fillStageTimes(ctx, orders); err != nil {
		return nil, err
	}

	// **ولا يُحسب أجرُ السائق هنا.**
	//
	// كان يُحسب لكلّ طلبٍ في القائمة — **قراءةُ إعداداتٍ لكلّ صفٍّ لرقمٍ لا
	// يعرضه أحد.** وقد حُذف من بطاقة الطلب (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لم
	// أطلبها أصلاً»).
	//
	// **وموضعُه دفترُه**: محفظةُ السائق وكشفُ حسابه — حيث يُقرأ مجموعاً.
	page := &OrderPage{Orders: orders, Total: total, Page: f.Page, PerPage: f.PerPage}
	// ══════════════════════════════════════════════════════════════════
	// **والأعدادُ بالشرط نفسِه — بحالٍ فارغ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ولا شرطَ ثانٍ يُكتب هنا**: نصُّ `where` واحدٌ للقائمة وللعدّ،
	// **ومن أضاف مُرشِّحاً غداً أضافه لهما معاً.** ونصّان يفترقان يوماً
	// **فتقول البطاقةُ اثنين وتعرض القائمةُ ثلاثة** — ولا أحدَ يعرف
	// أيُّهما الصواب.
	//
	// **والحالُ يُمرَّر فارغاً** فيسقط شرطُه وحدَه (`$1 = ''`) — وهو
	// بالضبط ما نريد: **عدٌّ لا يراه الترشيح.**
	//
	// **وللمنتهية وحدَها**: شاشةُ العمل لا حالَ منتهيةً فيها،
	// **واستعلامٌ يردّ أصفاراً كلَّها نداءٌ بلا جواب.**
	if f.ClosedOnly {
		cRows, err := s.db.Query(ctx, `
			SELECT o.status, count(*) FROM orders o
			JOIN users cu ON cu.id = o.customer_id`+where+`
			GROUP BY o.status`,
			"", f.MerchantID, f.CustomerID, f.DriverID, f.Query, f.OpenOnly,
			f.ClosedOnly)
		if err != nil {
			return nil, err
		}
		defer cRows.Close()
		counts := map[string]int{}
		for cRows.Next() {
			var st string
			var n int
			if err := cRows.Scan(&st, &n); err != nil {
				return nil, err
			}
			counts[st] = n
		}
		if err := cRows.Err(); err != nil {
			return nil, err
		}
		page.Counts = counts
	}
	return page, nil
}
