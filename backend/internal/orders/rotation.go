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
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// AssignmentMode نمطُ التوزيع الحاليّ.
func (s *Service) AssignmentMode(ctx context.Context) string {
	if s.settings == nil {
		return "queue"
	}
	return s.settings.GetString(ctx, "drivers.assignment_mode")
}

func (s *Service) offerTimeout(ctx context.Context) time.Duration {
	return time.Duration(s.settingInt(ctx, "drivers.offer_timeout_sec")) * time.Second
}

// directAssign أيصير الطلبُ مهمّتَه بلا سؤال.
//
// **ولا معنى له خارج «بالتساوي»** — هناك لا سائقَ مختاراً يُسنَد إليه. والشرطُ
// مكتوبٌ في الفهرس أيضاً (`ShowWhen`)، **وهنا لأنّ الشاشةَ تُخفي والمحرّكَ
// يقرأ**: مفتاحٌ بقي مرفوعاً من وضعٍ سابقٍ لا يصنع إسناداً في وضعٍ لا يحتمله.
func (s *Service) directAssign(ctx context.Context) bool {
	if s.settings == nil {
		return false
	}
	return s.AssignmentMode(ctx) == "rotation" &&
		s.settings.GetBool(ctx, "drivers.direct_assign")
}

// settingInt رقمٌ من الإعدادات — **ولا احتياطيَّ مكتوبٌ هنا.**
//
// كان كلُّ قارئٍ يكتب رقمَه: `sec := 45` و`limit := 500000` و`maxActive := 2`
// — **والفهرسُ يحمل الأرقامَ نفسَها.** فيُغيَّر افتراضُ الفهرس ويبقى القارئُ
// على القديم، **ولا يظهر ذلك إلّا حين يُمحى المفتاحُ من القاعدة** فيعمل
// موضعٌ برقمٍ وموضعٌ بآخر.
//
// **و`GetInt` تقرأ افتراضَ الفهرس أصلاً** — فالحارسُ هنا للمخزن الغائب وحدَه.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «الأرقامُ تصدر من مكانٍ مركزيٍّ واحد».)
func (s *Service) settingInt(ctx context.Context, key string) int64 {
	if s.settings == nil {
		return settings.Default(key)
	}
	return s.settings.GetInt(ctx, key)
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
//
// # وعرضٌ حيٌّ واحدٌ لكلّ سائق — **طلبٌ لكلّ واحدٍ لا ثلاثةٌ لواحد**
//
// كلُّ طلبٍ كان يختار «أطولَ انتظاراً» **مستقلاًّ عن الآخر، ولا يعلم أنّ عرضاً
// حيّاً عند ذاك السائق**. وثلاثةُ طلباتٍ تُحوَّل معاً وثلاثةُ سائقين في الدوام
// **تقع كلُّها على الأوّل**: يراها الثلاثةَ في شاشته، والاثنان الآخران شاشتاهما
// فارغة. ثمّ تنقضي مهلتُه على الثلاثة **فتنتقل كلُّها معاً إلى الثاني** — دورةٌ
// كاملةٌ تُهدر وثلاثةُ زبائنَ ينتظرون.
//
// **وقرارُ المالك (٢٠٢٦-٠٨-٠٣)**: «المفروض الآن يوجد ٣ سائقين، الطلب الأوّل
// يذهب للأوّل والثاني للثاني والثالث للثالث».
//
// **فمن عنده عرضٌ حيٌّ لا يُعرض عليه ثانٍ**: يتوزّع الحملُ من أوّل لحظة،
// **ويقرّر كلُّ سائقٍ في طلبٍ واحدٍ لا في ثلاثة.**
//
// **وما لم يبقَ له سائقٌ ينتظر بلا عرض** — لا يُهمَل: `SweepExpiredOffers`
// يلتقطه حين يتحرّر أحدُهم.
func (s *Service) OfferNext(ctx context.Context, orderID string, skip []string) error {
	// **ونفسُ المسار قبل الدور — وقبل النمط.**
	//
	// **وهو قبل الدور لأنّه ليس منافساً له**: الدورُ يوزّع ما لا صاحبَ له،
	// **وهذا يقول إنّ لهذا الطلب صاحباً بالفعل** — سائقٌ في طريقه إليه.
	//
	// **وقبل النمط لأنّه يعمل في الاثنين**: في «الأسرع» يُنتزع الطلبُ من
	// السباق **فلا يأخذه من هو في آخر المدينة قبل من يقف أمام الباب.**
	//
	// **ولا يُحاوَل إلّا في أوّل مرّة** (`skip` فارغ): طلبٌ مرّ عليه الدورُ
	// **صار له تاريخٌ من الرفض**، وإسنادُه قسراً بعد ذلك يُعيد ما رُفض.
	if len(skip) == 0 && s.TrySameRoute(ctx, orderID) {
		return nil
	}
	if s.AssignmentMode(ctx) != "rotation" {
		return nil
	}

	limit := s.settingInt(ctx, "drivers.cash_limit")
	maxActive := s.settingInt(ctx, "drivers.max_active_orders")

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
		  -- **وعرضٌ حيٌّ واحدٌ لكلّ سائق** — انظر تعليلَه فوق الدالّة.
		  AND NOT EXISTS (
		      SELECT 1 FROM orders o2
		      WHERE o2.offered_driver_id = u.id
		        AND o2.id <> $4
		        AND o2.status = 'dispatching'
		        AND o2.driver_id IS NULL
		        AND o2.offer_expires_at > now())
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

	// **الإسنادُ المباشر — الطلبُ يصير مهمّتَه بلا سؤال.**
	//
	// `offered_driver_id` يبقى مكتوباً: منه يُعرف صاحبُ الدور حين يُنزع الطلبُ
	// منه، **ومنه يُبنى `offer_passed` فلا يعود إليه.** و`offer_expires_at`
	// صار **مهلةَ صمتٍ لا مهلةَ ردّ**.
	//
	// **ودورُه ينتقل إلى آخر الصفّ لحظتَها** (`last_assigned_at`) — فسائقٌ
	// نائمٌ يعطّل طلباً واحداً لا كلَّ الطلبات.
	if s.directAssign(ctx) {
		return s.assignDirectly(ctx, orderID, driverID, autoAssignNote)
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

// assignDirectly يضع الطلبَ في مهامّ صاحب الدور — **بالمسار نفسِه الذي يسلكه
// من يضغط «خذ الطلب»**.
//
// # ولماذا لا يُكتب بيدٍ في جدول الطلبات
//
// كان قيداً واحداً يضع `driver_id` و`status` معاً، **ففاته شيئان لا يُرى
// غيابُهما إلّا بعد أسبوع**:
//
//  1. **لا حدثَ في سجلّ الطلب.** فيُقرأ المسارُ `dispatching → at_pickup`،
//     **ولا يُعرف متى وصل السائقَ الطلبُ ولا كيف** — أخذه بنفسه أم أُسند إليه.
//     وهو أوّلُ ما يُسأل عنه حين يتأخّر طلب. (شهده المالك ٢٠٢٦-٠٨-٠٥.)
//  2. **كان يكتب `accepted_at = now()`** — وهو **وقتُ قبول المتجر** لا وقتُ
//     الإسناد. فيُمحى وقتُ القبول ويُقرأ الطلبُ كأنّ المتجرَ قبله لحظةَ نزوله
//     إلى الطابور، **فيصير قياسُ سرعة المتاجر كذباً.**
//
// **والمحرّكُ يكتب الاثنين وحدَه** — فيُنادى كما يُنادى من الأخذ اليدويّ:
// `driver_id` أوّلاً بشرطٍ ذرّيّ، ثمّ انتقالٌ عاديّ.
func (s *Service) assignDirectly(ctx context.Context, orderID, driverID, note string) error {
	// **الشرطُ الذرّيّ يبقى**: سائقٌ ضغط «خذ الطلب» في اللحظة نفسِها يجد صفراً.
	tag, err := s.db.Exec(ctx, `
		UPDATE orders
		SET offered_driver_id = $2, driver_id = $2, updated_at = now(),
		    offer_expires_at = now() + make_interval(secs => $3)
		WHERE id = $1 AND status = 'dispatching' AND driver_id IS NULL`,
		orderID, driverID, s.offerTimeout(ctx).Seconds())
	if err != nil {
		return err
	}
	// **وسبقَنا إليه غيرُنا** — لا خطأ: الطلبُ في يدٍ أمينة.
	if tag.RowsAffected() == 0 {
		return nil
	}

	if _, e := s.db.Exec(ctx,
		`UPDATE users SET last_assigned_at = now() WHERE id = $1`, driverID); e != nil {
		s.logger.Error("الترتيب: تعذّر تحريكُ الدور", "driver", driverID, "error", e)
	}

	// **والانتقالُ بالمحرّك — فيُكتب الحدثُ ويُبثّ ما يجب.**
	//
	// **والفاعلُ هو السائق** لا «النظام»: الطلبُ صار في يده وهو المسؤولُ عنه،
	// **والنصُّ يقول إنّه لم يختره** فلا يُقرأ الحدثُ أخذاً طوعياً.
	if _, err := s.Transition(ctx, driverID, []string{"driver"},
		orderID, StAssigned, note); err != nil {
		// **وتراجعٌ عن الإسناد** — لولاه بقي الطلبُ محجوزاً لسائقٍ لم يقبله
		// المحرّك، فلا يراه أحدٌ ولا يعمل عليه أحد.
		if _, e := s.db.Exec(ctx, `
			UPDATE orders SET driver_id = NULL, offered_driver_id = NULL,
			                  offer_expires_at = NULL
			WHERE id = $1 AND driver_id = $2`, orderID, driverID); e != nil {
			s.logger.Error("الترتيب: تعذّر التراجعُ عن الإسناد",
				"order", orderID, "error", e)
		}
		return err
	}
	s.pub.Publish(topicDriverQueue, map[string]any{"type": "order"})
	return nil
}

// autoAssignNote نصُّ حدثِ الإسناد التلقائيّ — **يُقرأ في سجلّ الطلب.**
const autoAssignNote = "إسنادٌ تلقائيٌّ بالدور"

// reclaimSilentAssignments ينزع طلباً أُسند مباشرةً ولم يتحرّك صاحبُه.
//
// # المسألة
//
// في العرض، انقضاءُ المهلة ينقل الدورَ وحدَه: الطلبُ ما زال `dispatching` بلا
// سائق. **وفي الإسناد المباشر لا شيءَ ينقضي** — الطلبُ في مهامّه، وهو نائمٌ
// أو هاتفُه في جيبه. **فيقف الزبونُ على من لا يعلم أنّ له مهمّة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «ينتقل إلى التالي بعد مدّة».)
//
// # وما معنى «لم يتحرّك»
//
// **بقاؤه في `assigned`.** ومن فتحه ومضى إلى المتجر صار `at_pickup` — فخرج
// من هذا الشرط ولا يُنزع منه شيءٌ وهو في الطريق. **والحركةُ فعلٌ لا فتحُ
// شاشة**: من نظر ثمّ نام كمن لم ينظر.
//
// # وينزل إلى الطابور لا يُلغى
//
// يعود `dispatching` بلا سائق، **ويُوسَم أنّ الدورَ مرّ عليه** فلا يعود إليه
// — ثمّ تلتقطه الجولةُ التالية لغيره. **وطلبٌ يُنزع ولا يُعرض على أحدٍ أسوأُ
// ممّا كان.**
func (s *Service) reclaimSilentAssignments(ctx context.Context) {
	if !s.directAssign(ctx) {
		return
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, driver_id FROM orders
		WHERE status = 'assigned' AND driver_id IS NOT NULL
		  AND offer_expires_at IS NOT NULL AND offer_expires_at <= now()`)
	if err != nil {
		s.logger.Error("الترتيب: تعذّرت قراءة الإسنادات الصامتة", "error", err)
		return
	}
	type silent struct{ orderID, driverID string }
	var list []silent
	for rows.Next() {
		var x silent
		if rows.Scan(&x.orderID, &x.driverID) == nil {
			list = append(list, x)
		}
	}
	rows.Close()

	for _, x := range list {
		// **النزعُ والوسمُ في قيدٍ واحد** — ولو وقع النزعُ وحدَه لعاد الطلبُ
		// إلى الصامت نفسِه في الجولة التالية: هو ما زال أطولَ انتظاراً.
		//
		// **والشرطُ يتكرّر في التحديث**: بين القراءة والكتابة قد يكون تحرّك،
		// **ونزعُ طلبٍ من سائقٍ صار في الطريق إليه أسوأُ من تركه نائماً.**
		var skip []string
		if err := s.db.QueryRow(ctx, `
			UPDATE orders
			SET driver_id = NULL, status = 'dispatching', accepted_at = NULL,
			    offered_driver_id = NULL, offer_expires_at = NULL,
			    dispatched_at = now(), offer_passed = offer_passed || $2::uuid,
			    updated_at = now()
			WHERE id = $1 AND status = 'assigned' AND driver_id = $2
			RETURNING array(SELECT unnest(offer_passed)::text)`,
			x.orderID, x.driverID).Scan(&skip); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				s.logger.Error("الترتيب: تعذّر نزعُ إسنادٍ صامت",
					"order", x.orderID, "error", err)
			}
			continue
		}
		s.pub.Publish("driver:"+x.driverID, map[string]any{"type": "order"})
		s.pub.Publish("ops", map[string]any{"type": "order"})
		if err := s.OfferNext(ctx, x.orderID, skip); err != nil {
			s.logger.Error("الترتيب: تعذّر نقلُ الدور بعد النزع",
				"order", x.orderID, "error", err)
		}
	}
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
	s.reclaimSilentAssignments(ctx)
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

	s.offerWaiting(ctx)
}

// offerWaiting يعرض ما ينتظر بلا عرض — **حين يتحرّر سائق.**
//
// # لماذا لزمت
//
// **العرضُ الحيُّ واحدٌ لكلّ سائق**، فطلبٌ رابعٌ يأتي وثلاثةُ سائقين مشغولون
// بعروضهم **لا يجد أحداً فيبقى بلا عرض.** ولا شيءَ يوقظه بعدها: الكانسُ كان
// ينظر إلى **العروض المنقضية** وحدَها، **وهذا لا عرضَ له أصلاً** — فيبقى
// ساكناً حتى يمرّ حدثٌ آخرُ بمحض الصدفة.
//
// **وطلبٌ ينتظر بصمتٍ أسوأُ من طلبٍ يُرفض**: الرفضُ يُقرأ ويُعالَج، **والصمتُ
// يُنتظَر.**
//
// فيُنادى مع كلّ كنسة: من تحرّر أخذ ما ينتظر.
func (s *Service) offerWaiting(ctx context.Context) {
	rows, err := s.db.Query(ctx, `
		SELECT id FROM orders
		WHERE status = 'dispatching' AND driver_id IS NULL
		  AND offered_driver_id IS NULL
		-- **والأقدمُ أوّلاً** — ومن انتظر أطولَ يستحقّ أوّلَ سائقٍ يتحرّر.
		ORDER BY dispatched_at NULLS FIRST, created_at
		LIMIT 50`)
	if err != nil {
		s.logger.Error("الترتيب: تعذّرت قراءة المنتظِرين", "error", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	for _, id := range ids {
		// **ومن مرّ عليه الدورُ في هذا الطلب يبقى مستثنى** — يُقرأ من الطلب
		// نفسِه، فلا تُعاد الجولةُ على من رفض.
		var skip []string
		if err := s.db.QueryRow(ctx,
			`SELECT array(SELECT unnest(offer_passed)::text) FROM orders WHERE id = $1`,
			id).Scan(&skip); err != nil {
			continue
		}
		if err := s.OfferNext(ctx, id, skip); err != nil {
			s.logger.Error("الترتيب: تعذّر عرضُ منتظِر", "order", id, "error", err)
		}
	}
}

// groupDigits رقمٌ بفواصلِ ألوف — **٢٠٦١٠٠٠ لا تُقرأ، و٢٬٠٦١٬٠٠٠ تُقرأ.**
//
// **والفاصلةُ عربيّةٌ** (U+066C) لأنّ النصَّ عربيّ. ولا `golang.org/x/text`
// لسطرٍ واحد.
func groupDigits(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	d := strconv.FormatInt(n, 10)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range d {
		if i > 0 && (len(d)-i)%3 == 0 {
			b.WriteRune('٬')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// cashOverLimitMessage **الرسالةُ وحدَها — مفصولةً عن القاعدة لتُختبر.**
//
// **و`NoEligibleReason` تفتح قاعدةً فلا يبلغها اختبار** — ولو كُتبت الرسالةُ
// داخلَها لَبقيت بلا حارس. **وهي بالضبط ما شكا منه الجرد**: نصٌّ يَعِد بشيءٍ
// ولا يفعله، **ولا شيءَ يمسك ذلك.**
func cashOverLimitMessage(cashDue, limit int64) string {
	return "نقدُ الطلب " + groupDigits(cashDue) +
		" وسقفُ السائقين " + groupDigits(limit) + " — لن يلتقطه أحد"
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

	limit := s.settingInt(ctx, "drivers.cash_limit")
	if cashDue > limit {
		// **ويُقال بالرقمين لا بالحكم**: «فوق السقف» تُغلق الباب، **و«٢٬٠٦١٬٠٠٠
		// والسقفُ ٥٠٠٬٠٠٠» تقول أين المخرج** — يُرفع السقفُ أو يُقسَّم الطلب.
		//
		// **وكان يقول الحكمَ وحدَه ستّةَ أيّام** — والتعليقُ فوقه يَعِد بالرقمين.
		// (جردُ ٢٠٢٦-٠٨-٠٩.) **وتعليقٌ يَعِد بما لا تفعله شيفرتُه أسوأُ من لا
		// تعليق**: من قرأه صدّقه ولم يفتح السطر.
		return cashOverLimitMessage(cashDue, limit)
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
