package orders

// توزيعُ الطلبات على السائقين — نظامان.
//
// # الفرق
//
//   - **`queue` — الأسرع التقاطاً**: الطلبُ معروضٌ للجميع ومن سبق أخذ. سريعٌ
//     في الذروة، **ويُجوّع البطيء**: سائقٌ بهاتفٍ قديم أو حيٍّ ضعيف الشبكة لا
//     يصل قبل غيره أبداً، فيرى الطلبات ولا ينال منها.
//
//   - **`rotation` — بالترتيب**: يُعرض على واحدٍ في دوره، فإن لم يأخذه في
//     مهلته انتقل الدورُ إلى التالي. **عدلٌ وثمنُه ثوانٍ.**
//
// # والدورُ يُقاس بالانتظار
//
// **من طال انتظارُه منذ آخر طلبٍ أُسند إليه.** وقائمةٌ ثابتة تجعل من في أوّلها
// يُعرض عليه كلُّ شيء، ومن دخل الدوام متأخراً ينتظر دورةً كاملة.
//
// وهو **يُصحّح نفسه**: من أخذ طلباً هبط إلى آخر الصفّ، ومن رُفض عرضُه بقي في
// أوّله. **ولا يحتاج أحدٌ أن يمسك دفتراً.**
//
// # ولا يُعرض على من لا يستطيع
//
// خارجُ الدوام، أو بلغ سقفَ نقده، أو يحمل أقصى ما يُسمح من طلبات — **عرضٌ عليه
// عرضٌ ضائع**: يمرّ وقتُه كاملاً ثم ينتقل الدور، والزبونُ ينتظر بلا سبب.
//
// # وإن لم يبقَ أحد
//
// **لا يُترك الطلبُ محجوزاً لمن لا يستطيع.** يُفرَّغ العرضُ فيعود معروضاً
// للجميع كما في `queue` — وتراه العملياتُ لتُسنده يدوياً. **وتعطُّلُ الترتيب
// يعيدنا إلى الطابور، لا إلى لا شيء.**

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// AssignmentMode نمطُ التوزيع الحاليّ.
func (s *Service) AssignmentMode(ctx context.Context) string {
	if s.settings == nil {
		return "queue"
	}
	return s.settings.GetString(ctx, "drivers.assignment_mode", "queue")
}

func (s *Service) offerTimeout(ctx context.Context) time.Duration {
	sec := int64(45)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "drivers.offer_timeout_sec"); v > 0 {
			sec = v
		}
	}
	return time.Duration(sec) * time.Second
}

// OfferNext يعرض الطلبَ على صاحب الدور، أو يُفرّغ العرضَ إن لم يبقَ أحد.
//
// `skip` سائقون مرّ عليهم الدورُ في هذا الطلب فلا يُعادون إليه — **وإلّا دار
// العرضُ على الأوّل أبداً**: هو أطولُ انتظاراً وسيبقى كذلك ما دام لم يأخذ.
//
// # وسقفُ النقد يُقاس بما بحوزته **وبنقد هذا الطلب معاً**
//
// كان الشرطُ هنا `held < limit` وحدَه، **والقبولُ يفحص `held + cash_due >
// limit`** — قاعدةٌ واحدةٌ مكتوبةٌ في موضعين، **فافترقا.**
//
// **فيُعرض الطلبُ على من لا يستطيع أخذَه**: سائقٌ حوزتُه صفرٌ مؤهَّلٌ للعرض،
// وطلبٌ نقدُه فوق السقف **يُردّ عند الضغط.** ويدور العرضُ على الجميع بمهلته
// كاملةً ثمّ يسقط إلى «لا أحد» — **والعملياتُ ترى «جارٍ إسناد سائق» وتنتظر من
// لن يأتي.**
//
// **ووقع أمام المالك** (٢٠٢٦-٠٨-٠٣): طلبٌ نقدُه ٦٢٦٬٠٠٠ وسقفُ السائق ٥٠٠٬٠٠٠
// — عُرض على سائقٍ حوزتُه صفر.
//
// **ولم يُمسك لأنّ الاثنين يعملان**: العرضُ يعرض والقبولُ يردّ، وكلٌّ صحيحٌ
// وحدَه. **والخللُ في أنّهما لا يتّفقان** — وهو ما لا يراه اختبارٌ يفحص أحدَهما.
func (s *Service) OfferNext(ctx context.Context, orderID string, skip []string) error {
	if s.AssignmentMode(ctx) != "rotation" {
		return nil
	}

	limit := int64(500000)
	maxActive := int64(2)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "drivers.cash_limit"); v > 0 {
			limit = v
		}
		if v := s.settings.GetInt(ctx, "drivers.max_active_orders"); v > 0 {
			maxActive = v
		}
	}

	// **الأهليةُ تُفحص في الاستعلام لا بعده**: جلبُ الجميع ثم غربلتُهم في Go
	// يعني قراءةَ كل سائقٍ في المنصة لاختيار واحد.
	var driverID string
	err := s.db.QueryRow(ctx, `
		SELECT u.id
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE u.on_shift
		  AND u.status = 'active'
		  -- **COALESCE لا يُستغنى عنه**: أوّلُ عرضٍ يمرّ بـskip فارغاً،
		  -- و NOT (id = ANY(NULL)) يُنتج NULL لا TRUE — **فيسقط كلُّ سائقٍ
		  -- في المنصة ويعود الطلبُ مشاعاً وكأن لا أحدَ أهلٌ له.**
		  AND NOT (u.id = ANY(COALESCE($1::uuid[], '{}')))
		  -- **وسقفُ النقد يُقاس بما بحوزته وبنقد هذا الطلب معاً** — انظر
		  -- تعليلَه فوق الدالّة.
		  AND COALESCE((SELECT b.held FROM driver_cash_boxes b
		                WHERE b.driver_id = u.id), 0)
		      + COALESCE((SELECT o.cash_due FROM orders o WHERE o.id = $4), 0) <= $2
		  AND (SELECT count(*) FROM orders o
		       WHERE o.driver_id = u.id AND o.closed_at IS NULL) < $3
		-- **ومن لم يأخذ بعدُ يُرتّبون بمن بكّر بالدوام.**
		--
		-- كان الفاصلُ u.id — **معرّفٌ عشوائيٌّ لا معنى له**: من سُجّل أوّلاً
		-- يسبق من بكّر بالدوام. **وقاعدةُ المالك: من فتح دوامَه أوّلاً يستحقّ
		-- أوّلَ طلب** — وهو ما يجعل التبكير مجدياً.
		ORDER BY u.last_assigned_at NULLS FIRST, u.shift_started_at, u.id
		LIMIT 1`, skip, limit, maxActive, orderID).Scan(&driverID)

	if err != nil {
		// **خطأُ الاستعلام لا يُقرأ «لا أحد».**
		//
		// كانا يُعاملان سواءً، **فعمودٌ أُعيدت تسميتُه يجعل كلَّ الطلبات تبدو
		// بلا سائقٍ مؤهّل** — وتعود مشاعةً بهدوء، ولا يظهر في أيّ سجلّ أن
		// الترتيب معطَّل. وهو أسوأ من عطبٍ يصرخ.
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.Error("الترتيب: تعذّر اختيار صاحب الدور",
				"order", orderID, "error", err)
			return err
		}
		// **لا أحدَ أهلٌ — فليَرَه الجميع.** الطلبُ لا يُحجَز لمن لا يستطيع.
		_, e := s.db.Exec(ctx, `
			UPDATE orders SET offered_driver_id = NULL, offer_expires_at = NULL
			WHERE id = $1`, orderID)
		return e
	}

	_, err = s.db.Exec(ctx, `
		UPDATE orders
		SET offered_driver_id = $2, offer_expires_at = now() + make_interval(secs => $3)
		WHERE id = $1 AND status = 'dispatching' AND driver_id IS NULL`,
		orderID, driverID, s.offerTimeout(ctx).Seconds())
	if err == nil {
		// **إشارةٌ بلا حمولة** — كما في `drivers:queue`: تقول «تغيّر شيء»،
		// وتحكم نقطةُ الطابور ما يراه كلُّ سائق.
		s.pub.Publish(topicDriverQueue, map[string]any{"type": "order"})
		s.pub.Publish("driver:"+driverID, map[string]any{"type": "order"})
	}
	return err
}

// SweepExpiredOffers ينقل الدورَ عن العروض التي انقضت مهلتها.
//
// **يُنادى من الراصد** — لا من نداءِ سائقٍ للطابور: لو انتظرنا من يسأل لبقي
// طلبٌ محجوزاً لسائقٍ نائمٍ حتى يفتح غيرُه التطبيق. **والزبونُ لا ينتظر أن
// يتذكّر أحدٌ أن ينظر.**
func (s *Service) SweepExpiredOffers(ctx context.Context) {
	if s.AssignmentMode(ctx) != "rotation" {
		return
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, offered_driver_id FROM orders
		WHERE status = 'dispatching' AND driver_id IS NULL
		  AND offer_expires_at IS NOT NULL AND offer_expires_at <= now()`)
	if err != nil {
		s.logger.Error("الترتيب: تعذّرت قراءة العروض المنقضية", "error", err)
		return
	}
	type expired struct{ orderID, driverID string }
	var list []expired
	for rows.Next() {
		var e expired
		if rows.Scan(&e.orderID, &e.driverID) == nil {
			list = append(list, e)
		}
	}
	rows.Close()

	for _, e := range list {
		// **يُسجَّل أنه مرّ عليه قبل أن يُعرض على غيره** — ولو عُكس الترتيب
		// لعاد الدورُ إليه فوراً لأنه ما زال أطولَ انتظاراً.
		var skip []string
		if err := s.db.QueryRow(ctx, `
			UPDATE orders SET offer_passed = offer_passed || $2::uuid
			WHERE id = $1 RETURNING array(SELECT unnest(offer_passed)::text)`,
			e.orderID, e.driverID).Scan(&skip); err != nil {
			s.logger.Error("الترتيب: تعذّر وسمُ مرور الدور",
				"order", e.orderID, "error", err)
			continue
		}
		if err := s.OfferNext(ctx, e.orderID, skip); err != nil {
			s.logger.Error("الترتيب: تعذّر نقل الدور", "order", e.orderID, "error", err)
		}
	}
}

// NoEligibleReason لماذا لا يلتقط الطلبَ أحد — **حين لا يلتقطه أحد.**
//
// # المسألة
//
// طلبٌ فوق سقف نقد السائقين **لا يظهر لأحدٍ منهم**، فيبقى في الطابور صامتاً.
// **والعملياتُ ترى «جارٍ إسناد سائق» ولا تعرف أنّ أحداً لن يأتي** — تنتظر،
// ثمّ يوقظها الحارسُ بعد عشر دقائق بـ«لا سائق»، **ولا يقول لها لماذا.**
//
// ووقع فعلاً في التجربة الحيّة (طلب #1001، ٢٠٢٦-٠٨-٠٢): نقدُه ٢٬٠٦١٬٠٠٠
// وسقفُ السائق ٥٠٠٬٠٠٠ — **فما كان ليلتقطه أحدٌ أبداً.**
//
// **والصمتُ هنا أسوأُ من الرفض**: الرفضُ يُقرأ ويُعالَج، **والصمتُ يُنتظَر.**
//
// يعيد نصّاً فارغاً حين لا مشكلة.
func (s *Service) NoEligibleReason(ctx context.Context, orderID string) string {
	var cashDue int64
	var hasDriver bool
	if err := s.db.QueryRow(ctx,
		`SELECT cash_due, driver_id IS NOT NULL FROM orders WHERE id = $1`,
		orderID).Scan(&cashDue, &hasDriver); err != nil || hasDriver {
		return ""
	}

	limit := int64(500000)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "drivers.cash_limit"); v > 0 {
			limit = v
		}
	}
	if cashDue > limit {
		// **ويُقال بالرقمين لا بالحكم**: «فوق السقف» تُغلق الباب، **و«٢٬٠٦١٬٠٠٠
		// والسقفُ ٥٠٠٬٠٠٠» تقول أين المخرج** — يُرفع السقفُ أو يُقسَّم الطلب.
		return "نقدُ الطلب فوق سقف السائقين — لن يلتقطه أحد"
	}

	// **ثمّ: أعلى الدوامَ أحدٌ أصلاً؟**
	//
	// طلبٌ ينزل ليلاً ولا سائقَ على الدوام يبقى صامتاً كذلك، **والسببُ آخرُ
	// تماماً**: هذا يُحلّ بمكالمةٍ لا برفع سقف.
	var onShift int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM users u
		 JOIN user_roles r ON r.user_id = u.id AND r.role_code = 'driver'
		 WHERE u.on_shift AND u.status = 'active'`).Scan(&onShift); err == nil && onShift == 0 {
		return "لا سائقَ على الدوام الآن"
	}
	return ""
}
