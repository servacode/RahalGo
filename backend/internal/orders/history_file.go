package orders

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ الطلب في سجلّ اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// «لمّا افتح السجلّ لازم افهم لكلّ طلب: شو صار، كيف مشي، وحركة المال.»
//
// **شرطٌ واحدٌ للفلتر وللعلامة على الكرت** — «متأخّر» و«عمولته ممحيّة» و«مُحوَّل»
// تُكتب هنا مرّةً، **فلا يقول الكرتُ «متأخّر» والفلترُ لا يجده.**

import "context"

// StoreTransferNote **أوّلُ ملاحظة حدث «تحويلٌ إلى متجرٍ آخر»** — يكتبها التحويلُ
// (`server/order_transfer.go`) ويقرؤها السجلّ ليقول «مُحوَّل».
const StoreTransferNote = "تحويلٌ إلى متجرٍ آخر"

// LateDeliverySQL **سُلّم بعد مهلة عمر الطلب** (`orders.delivery_timeout_min`) — المهلةُ نفسُها
// التي تقول بها اللوحةُ «متأخّر» للطلب الجاري (`TooLongSQL`)، **مقيسةً على وقت التسليم.**
func LateDeliverySQL(l StuckLimits) string {
	return `(o.delivered_at IS NOT NULL AND o.delivered_at > o.created_at + ` + mins(l.DeliveryMin) + `)`
}

// TransferredSQL **حُوّل الطلبُ إلى متجرٍ آخر** — من مساره.
const TransferredSQL = `EXISTS (SELECT 1 FROM order_events te
	WHERE te.order_id = o.id AND te.note LIKE '` + StoreTransferNote + `%')`

// CasesSQL **كم شكوى ومطالبةً وطارئاً على الطلب** — مفتوحةً أو مغلقة.
const CasesSQL = `((SELECT count(*) FROM tickets tk WHERE tk.order_id = o.id)
	+ (SELECT count(*) FROM disputes dp WHERE dp.order_id = o.id)
	+ (SELECT count(*) FROM driver_emergencies em WHERE em.order_id = o.id))::int`

// payFilters **فلترُ طريقة الدفع** — نقدٌ: على الزبون مبلغٌ نقديّ · محفظةٌ: دُفع منها شيء.
// **والمختلطُ يظهر في الاثنين** — فيه الطريقتان.
var payFilters = map[string]string{
	"cash":   `(o.cash_due > 0 OR (o.wallet_paid = 0 AND o.payment_method = 'cash'))`,
	"wallet": `(o.wallet_paid > 0)`,
}

// staffSearchPhone و staffSearchName **بحثُ الموظّفين الأعرض** — هاتفُ السائق والمتجر،
// واسمُ السائق والمتجر وكودُ الخصم. `p` رقمُ معاملٍ مُعدٌّ سلفاً.
func staffSearchPhone(p string) string {
	return `EXISTS (SELECT 1 FROM users sd WHERE sd.id = o.driver_id AND sd.phone::text LIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR EXISTS (SELECT 1 FROM merchants sm WHERE sm.id = o.merchant_id AND sm.phone::text LIKE '%' || ` + p + ` || '%' ESCAPE '\')`
}

func staffSearchName(p string) string {
	return `EXISTS (SELECT 1 FROM users sd WHERE sd.id = o.driver_id AND sd.full_name ILIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR EXISTS (SELECT 1 FROM merchants sm WHERE sm.id = o.merchant_id AND sm.name ILIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR COALESCE(o.promo_code, '') ILIKE '%' || ` + p + ` || '%' ESCAPE '\'`
}

// HistoryMoney **مالُ الطلب على كرت السجلّ** (قرارُ المالك ٢٠٢٦-١٠-٣٠: «كلّ شي عن الطلب على الكرت») —
// نصيبُ المندوب وربحُ المنصّة من الدفتر (قراءةٌ لا كتابة)، وخصمُ العروض بمن تحمّله.
type HistoryMoney struct {
	Rep           int64 `json:"rep"`
	Platform      int64 `json:"platform"`
	Lines         int   `json:"lines"`
	OfferPlatform int64 `json:"offer_platform"`
	OfferMerchant int64 `json:"offer_merchant"`
}

// FillHistory **يملأ ما يحتاجه كرتُ السجلّ لصفحةٍ كاملة** — في رحلةٍ واحدة لا رحلةٍ لكلّ كرت:
// مالُ كلّ طلبٍ واسمُ من أنهاه. (والبنودُ في استعلام القائمة أصلاً.) **وتعذّرُه لا يُسقط القائمة.**
func (s *Service) FillHistory(ctx context.Context, list []Order) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	at := map[string]int{}
	for i := range list {
		ids[i] = list[i].ID
		at[list[i].ID] = i
		if list[i].Board == nil {
			list[i].Board = &BoardInfo{}
		}
	}
	mRows, err := s.db.Query(ctx, `
		SELECT o.id::text,
		       COALESCE(sum(wt.amount) FILTER (WHERE wt.kind = 'commission'), 0),
		       COALESCE(sum(wt.amount) FILTER (WHERE wt.kind = 'platform_profit'), 0),
		       count(wt.id)::int,
		       COALESCE((SELECT sum(oi.offer_cut * oi.qty) FROM order_items oi
		                 WHERE oi.order_id = o.id AND oi.offer_borne_by = 'platform'), 0),
		       COALESCE((SELECT sum(oi.offer_cut * oi.qty) FROM order_items oi
		                 WHERE oi.order_id = o.id AND oi.offer_borne_by = 'merchant'), 0),
		       COALESCE((SELECT NULLIF(u.full_name, '') FROM order_events e JOIN users u ON u.id = e.actor_id
		                 WHERE e.order_id = o.id AND e.to_status = o.status
		                 ORDER BY e.id DESC LIMIT 1), '')
		FROM orders o
		LEFT JOIN wallet_transactions wt ON wt.ref = o.id::text
		WHERE o.id = ANY($1::uuid[])
		GROUP BY o.id`, ids)
	if err != nil {
		return err
	}
	defer mRows.Close()
	for mRows.Next() {
		var oid string
		var hm HistoryMoney
		var endedBy string
		if err := mRows.Scan(&oid, &hm.Rep, &hm.Platform, &hm.Lines, &hm.OfferPlatform, &hm.OfferMerchant,
			&endedBy); err != nil {
			return err
		}
		if i, ok := at[oid]; ok {
			m := hm
			list[i].Board.Money = &m
			list[i].Board.EndedByName = endedBy
		}
	}
	return mRows.Err()
}
