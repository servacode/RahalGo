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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// AssignmentMode نمطُ التوزيع الحاليّ.
// ErrOfferNotYours **عرضٌ لم يعد له** — أخذه غيرُه أو انقضت مهلتُه.
var ErrOfferNotYours = httpx.NewError(http.StatusConflict,
	"offer_not_yours", "errors.offer_not_yours")

func (s *Service) AssignmentMode(ctx context.Context) string {
	if s.settings == nil {
		return "queue"
	}
	return s.settings.GetString(ctx, "drivers.assignment_mode")
}

func (s *Service) offerTimeout(ctx context.Context) time.Duration {
	return time.Duration(s.settingInt(ctx, "drivers.offer_timeout_sec")) * time.Second
}

// silenceTimeout **مهلةُ الصمت بعد الإسناد المباشر — لا مهلةَ الردّ.**
//
// (كشفه جردُ ٢٠٢٦-٠٨-٠٩ بحسابات حقيقيّة: الطلبُ يُسنَد ثمّ يُسترجَع بعد
//
//	خمسٍ وأربعين ثانيةً فيرتدّ إلى الإسناد اليدويّ.)
//
// # ولماذا رقمٌ ثانٍ لا رفعُ الأوّل
//
// **المفتاحُ الواحدُ كان يخدم معنيين**:
//
//   - **في العرض**: «كم يُنتظر ردُّه» — والسائقُ ينظر إلى شاشةٍ ترنّ،
//     **وخمسٌ وأربعون ثانيةً كافية.**
//   - **وفي الإسناد المباشر**: «كم يُحتمل صمتُه» — **والطلبُ وقع في يده وهو
//     على درّاجته لا ينظر.** وقد لا يلمس هاتفه دقيقتين، **فيُسترجَع منه ما
//     هو ذاهبٌ إليه.**
//
// **ورفعُ الأوّل لأجل الثاني يُفسد الأوّل**: عرضٌ ينتظر ثلاث دقائق يجمّد
// الطلبَ عند نائمٍ بينما غيرُه يعمل.
//
// **ورقمان لمعنيين خيرٌ من رقمٍ يخدم أحدَهما ويُساء به إلى الآخر** — وهي
// القاعدةُ التي تكرّرت في هذا المشروع: **معنيان في مفتاحٍ واحدٍ يفترقان.**
func (s *Service) silenceTimeout(ctx context.Context) time.Duration {
	return time.Duration(s.settingInt(ctx, "drivers.assigned_silence_sec")) * time.Second
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

// ProximityEnabled أمُفعَّلٌ التوزيعُ بالقرب — **مطفأً يعود العدلُ/البثُّ
// الصِّرف.** (بلا مخزنِ إعداداتٍ في اختبار: مطفأ.)
func (s *Service) ProximityEnabled(ctx context.Context) bool {
	return s.settings != nil && s.settings.GetBool(ctx, "drivers.proximity_enabled")
}

// DispatchProximity أرقامُ القربِ المركزيّة — **يقرؤها الطابورُ (البثّ) ليحصر
// الرؤيةَ في المؤهَّلين القريبين، بنفسِ سياسة توسّع الدور.**
type DispatchProximity struct {
	CashLimit  int64
	MaxActive  int64
	FreshSec   int64
	InitialM   int64
	StepM      int64
	MaxM       int64
	TimeoutSec int64
}

// DispatchProximity يجمع أرقامَ القرب من الإعدادات المركزيّة.
func (s *Service) DispatchProximity(ctx context.Context) DispatchProximity {
	timeout := int64(s.offerTimeout(ctx).Seconds())
	if timeout < 1 {
		timeout = 1
	}
	return DispatchProximity{
		CashLimit:  s.settingInt(ctx, "drivers.cash_limit"),
		MaxActive:  s.settingInt(ctx, "drivers.max_active_orders"),
		FreshSec:   s.settingInt(ctx, "drivers.location_fresh_sec"),
		InitialM:   s.settingInt(ctx, "drivers.dispatch_radius_initial_m"),
		StepM:      s.settingInt(ctx, "drivers.dispatch_radius_step_m"),
		MaxM:       s.settingInt(ctx, "drivers.dispatch_radius_max_m"),
		TimeoutSec: timeout,
	}
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
	// ══════════════════════════════════════════════════════════════════
	// **ومن ترك الطلبَ مستثنى أبداً — لا لهذه الجولة وحدَها** (قرارُ المالك
	// مساءَ ٢٠٢٦-١٠-٠٢، البند ٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// «لا يعود الطلبُ إلى من تركه أبداً — ولو لم يوجد غيرُه.» **و`offer_passed`
	// سجلُّ جولةٍ يُصفَّر أدناه حين يدور الطابورُ على الجميع** — فكان الطلبُ يعود
	// إلى التارك إن لم يبقَ غيرُه. **فالمستثنى أبداً عمودٌ لا يُصفَّر**
	// (`excluded_drivers`)، **ويُضمّ إلى كلّ استثناءٍ هنا** — في الجولة وفي تصفيرها.
	excluded := s.excludedDrivers(ctx, orderID)
	skip = append(append([]string{}, skip...), excluded...)

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

	// **الاختيارُ بالقرب ثمّ العدل** — انظر `pickRotationCandidate`. والأهليّةُ
	// تُفحص في الاستعلام لا بعده: جلبُ الجميع ثمّ غربلتُهم في Go يعني قراءةَ
	// كلّ سائقٍ في المنصة لاختيار واحد.
	driverID, waitForExpansion, err := s.pickRotationCandidate(ctx, orderID, skip, limit, maxActive)

	// ══════════════════════════════════════════════════════════════════
	// **دارت الجولةُ ولم يأخذه أحد — فتُصفَّر ويعود إلى الجميع**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وهو الوعدُ المكتوبُ في `transitions.go` ولم يكن يُنفَّذ**: «ولا يُحرم
	// منه أبداً: الاستثناءُ لهذه الجولة وحدَها، **فإن دار الطابورُ ولم يأخذه
	// أحد عاد إليه مع الجميع**». **و`offer_passed` كان يُضاف إليه في ثلاثة
	// مواضعَ ولا يُصفَّر في موضعٍ واحدٍ من المشروع.**
	//
	// **فبعتادِ سائقٍ واحدٍ مؤهَّلٍ كان الطلبُ يموت عند أوّل مهلةٍ تنقضي**:
	// المحرّكُ يستثني من مرّ، **وقائمتا تطبيقه تستثنيانه كذلك** — فلا يُعرض
	// ولا يُرى. (قُيس على التجهيز ٢٠٢٦-٠٩-٢٩: مدخلٌ جديدٌ كلَّ ثلاثين ثانيةً —
	// ٩ ثمّ ١٠ ثمّ ١١ — **كلُّها المعرّفُ نفسُه، والطلبُ ساكنٌ لا يتقدّم.**)
	//
	// **والسؤالُ الفارق: أغابَ المرشَّحُ لأنّه مُستثنى أم لأنّه غيرُ مؤهَّل؟**
	// فيُعاد السؤالُ بلا استثناءٍ مرّةً واحدة: **وُجد ⇒ الجولةُ تمّت فتُفتح
	// جديدة. لم يُوجد ⇒ لا أحدَ أصلاً، فيُنتظر كما كان.**
	//
	// **ولا تُصفَّر إلّا ومرشَّحُها في اليد** — تصفيرٌ بلا مرشَّحٍ يمحو تاريخَ
	// الجولة ولا يُقدّم الطلبَ خطوة.
	if errors.Is(err, pgx.ErrNoRows) && len(skip) > len(excluded) {
		if id2, wait2, err2 := s.pickRotationCandidate(ctx, orderID, excluded, limit, maxActive); err2 == nil {
			if _, e := s.db.Exec(ctx,
				`UPDATE orders SET offer_passed = '{}' WHERE id = $1`, orderID); e != nil {
				s.logger.Error("الترتيب: تعذّر تصفيرُ الجولة",
					"order", orderID, "error", e)
			} else {
				driverID, waitForExpansion, err = id2, wait2, nil
			}
		}
	}

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
		// **قريبٌ مؤهَّلٌ لم يُوجد بعد، ونصفُ القطر لم يبلغ أقصاه** — يُترك
		// الطلبُ بلا عرضٍ لتلتقطه الكنسةُ التالية بحلقةٍ أوسع (`offerWaiting`).
		// **لا يُفرَّغ ولا يُهمَل**: التوسّعُ يقاس بانتظاره.
		if waitForExpansion {
			return nil
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
	// **ولا إسنادَ مباشرٌ لطلبٍ خاصّ.**
	//
	// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «أوّلَ شيءٍ لازم السائق يوافق على الطلب،
	//  وما ينزل بشكلٍ مباشر».)
	//
	// **والعاديُّ معروفٌ سلفاً**: متجرٌ وأصنافٌ وأجرةٌ مكتوبة — **فمن أُسند
	// إليه يعرف ما قبِل.** والخاصُّ وصفٌ حرٌّ ومالٌ من جيبه: **قد يكون بعيداً
	// أو مكلفاً أو لا يعرف مصدرَه**، فلا يُلزَم به بلا أن يقرأه.
	if s.directAssign(ctx) && !s.isCustom(ctx, orderID) {
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
		// **والبثُّ يصل شاشةً مفتوحة، والدفعُ يصل جيباً مغلقاً.**
		s.notifyOffer(ctx, orderID, driverID)
	}
	return err
}

// pickRotationCandidate يختار صاحبَ الدور — **بالقرب ثمّ العدل، أو العدلِ
// وحدَه.**
//
// # ثلاثةُ مخارج
//
//	سائقٌ            ←  وُجد مرشّح (قريبٌ مؤهَّلٌ حديثُ الموقع، أو مؤهَّلٌ بالعدل)
//	`waitForExpansion` ←  القربُ مُفعَّلٌ ولا مرشّحَ داخلَ الحلقة بعد، ونصفُ
//	                      القطر دون أقصاه: يُنتظر توسّعُه في الكنسة التالية
//	`pgx.ErrNoRows`    ←  لا مؤهَّلَ أصلاً — يُفرَّغ العرضُ فيراه الجميع
//
// **والقربُ يتوسّع بانتظار الطلب**: نصفُ القطر = الأوّل + الخطوة × عددِ المُهَل
// المنقضية، محدوداً بالأقصى. بلغ الأقصى ولا قريبٌ حديث؟ **يعود العدلُ الصِّرف**
// (شبكةُ الأمان) فلا يبقى طلبٌ عالقاً في منطقةٍ قليلةِ السائقين.
func (s *Service) pickRotationCandidate(ctx context.Context, orderID string, skip []string, limit, maxActive int64) (string, bool, error) {
	// **مطفأً — أو بلا مخزنِ إعداداتٍ في اختبار — يعود العدلُ الصِّرف** كما كان.
	if s.settings == nil || !s.settings.GetBool(ctx, "drivers.proximity_enabled") {
		id, err := s.legacyRotationCandidate(ctx, orderID, skip, limit, maxActive)
		return id, false, err
	}
	hasPickup, waitSec, err := s.orderDispatchInfo(ctx, orderID)
	if err != nil {
		return "", false, err
	}
	// ══════════════════════════════════════════════════════════════════
	// **وكانت هنا كتلةٌ ثانيةٌ `if !hasPickup` تسبق أختَها** — فتُنادي
	// `legacyRotationCandidate` **بلا شرطِ حداثة**، وتُميت الكتلةَ التي
	// تحفظ الحداثة (أدناه). **فحُذفت.**
	//
	// **ونسختان لشرطٍ واحدٍ إحداهما ميّتةٌ أخطرُ من غياب الشرط**: تُقرأ
	// الشيفرةُ فتُرى الحداثةُ محفوظةً، **والمنفَّذُ غيرُ المقروء.**
	//
	// (قُيس ٢٠٢٦-٠٩-٢٩: الاستعلامُ الموروثُ يختار لطلبٍ خاصٍّ سائقاً
	//  موضعُه شائخٌ ١١٫٧ يوماً، **والحديثُ بجانبه يخسر الدور** — لأنّ
	//  ترتيبَه `last_assigned_at NULLS FIRST` لا يعرف الحداثة.)
	// ══════════════════════════════════════════════════════════════════
	fresh := s.settingInt(ctx, "drivers.location_fresh_sec")
	initR := s.settingInt(ctx, "drivers.dispatch_radius_initial_m")
	stepR := s.settingInt(ctx, "drivers.dispatch_radius_step_m")
	maxR := s.settingInt(ctx, "drivers.dispatch_radius_max_m")
	bucket := s.settingInt(ctx, "drivers.proximity_bucket_m")
	timeout := int64(s.offerTimeout(ctx).Seconds())
	if timeout < 1 {
		timeout = 1
	}

	// **بلا نقطةِ التقاطٍ لا مسافةَ تُقاس** — لكنّ الحداثةَ تبقى شرطاً:
	// نختار أقربَ مؤهَّلٍ حديثِ الموقع بالعدل، **ولا نهبط إلى شائخ.**
	if !hasPickup {
		id, err := s.freshFallbackCandidate(ctx, orderID, skip, limit, maxActive, fresh, bucket)
		if err == nil {
			return id, false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, err
		}
		return "", true, pgx.ErrNoRows // **يُنتظر مؤهَّلٌ حديثُ الموقع.**
	}

	radius := initR + stepR*(waitSec/timeout)
	if radius >= maxR {
		radius = maxR
	}
	// **بلغ الأقصى، أو لا خطوةَ توسّعٍ أصلاً** — عندها لا مزيدَ من الحلقات.
	atMax := radius >= maxR || stepR <= 0

	id, err := s.proximityInRadius(ctx, orderID, skip, limit, maxActive, fresh, radius, bucket)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	if atMax {
		// ══════════════════════════════════════════════════════════════
		// **شبكةُ الأمان تبقى حديثةَ الموقع — لا تهبط إلى شائخ** (قرارُ
		// المالك ٢٠٢٦-٠٩-٢٨)
		// ══════════════════════════════════════════════════════════════
		//
		// **بلوغُ الأقصى لا يجعل موضعاً شائخاً مؤهَّلاً**: نُسقط حدَّ المسافة
		// وحدَه ونُبقي الحداثة، فنختار أقربَ مؤهَّلٍ حديثِ الموقع أينما كان.
		// **وإن لم يكن ثمّة حديثٌ مؤهَّل — يُنتظر** حتّى يتحدّث موضعُ أحدهم،
		// **ولا يُعرض على مجهولِ الموضع.** والعدلُ الصِّرفُ الأعمى عن الموضع
		// لا يقع إلّا حين يُطفئ المشغّلُ القربَ صراحةً (`proximity_enabled=false`).
		id, err := s.freshFallbackCandidate(ctx, orderID, skip, limit, maxActive, fresh, bucket)
		if err == nil {
			return id, false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, err
		}
		return "", true, pgx.ErrNoRows // **لا حديثَ مؤهَّل — يُنتظر لا يُهبَط.**
	}
	// **يُنتظر التوسّع.**
	return "", true, pgx.ErrNoRows
}

// orderDispatchInfo يقرأ ما يلزم لحساب الحلقة: **أللطلب نقطةُ التقاطٍ، وكم
// انتظر منذ نزوله إلى الطابور** (`dispatched_at`، وإلّا `created_at`).
func (s *Service) orderDispatchInfo(ctx context.Context, orderID string) (hasPickup bool, waitSec int64, err error) {
	var wait float64
	// ══════════════════════════════════════════════════════════════════
	// **والوصلةُ يسرى لا داخليّة** — `LEFT JOIN`
	// ══════════════════════════════════════════════════════════════════
	//
	// **الطلبُ الخاصُّ لا متجرَ له** (`merchant_id` NULL)، **ووصلةٌ داخليّةٌ
	// تُسقط الصفَّ كلَّه** فتردّ `ErrNoRows`. **ومن فوقنا يقرأ `ErrNoRows`
	// «لا سائقَ مؤهَّلَ أصلاً»** فيُفرّغ العرضَ ويعود — **ولأنّه ليس خطأً لا
	// يُسجَّل سطرٌ واحد.**
	//
	// **فكلُّ طلبٍ خاصٍّ كان لا يُعرض على أحدٍ أبداً، بصمت.** (قُيس على
	// التجهيز ٢٠٢٦-٠٩-٢٩: طلبٌ خاصٌّ ثلاثاً وعشرين دقيقةً في `dispatching`
	// بلا عرضٍ ولا سجلّ، **و١٠١ طلبٍ خاصٍّ في القاعدة كلُّها بلا متجر.**)
	//
	// **وهي العائلةُ المكتوبةُ في `admin_users_handlers.go`**: وصلةٌ صلبةٌ
	// بـ`merchants` على طلبٍ بلا متجر، **أمسكها المشيُ الحيُّ خمسَ مرّاتٍ
	// وكلُّ مرّةٍ تُصلَح واحدةً ويبقى الباقي.** وهذه السادسة — **وموضعُها
	// محرّكُ التوزيع لا شاشةُ عرض.**
	err = s.db.QueryRow(ctx, `
		SELECT `+DispatchAnchorSQL+` IS NOT NULL,
		       GREATEST(0, EXTRACT(EPOCH FROM (now() - COALESCE(o.dispatched_at, o.created_at))))
		FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&hasPickup, &wait)
	if err != nil {
		return false, 0, err
	}
	return hasPickup, int64(wait), nil
}

// legacyRotationCandidate العدلُ الصِّرف — **أطولُ انتظاراً أوّلاً، بلا قُرب.**
// وهو السلوكُ قبل القرب، ويبقى شبكةَ الأمان ووضعَ الإطفاء.
func (s *Service) legacyRotationCandidate(ctx context.Context, orderID string, skip []string, limit, maxActive int64) (string, error) {
	var driverID string
	// **الأهليةُ تُفحص في الاستعلام لا بعده**: جلبُ الجميع ثم غربلتُهم في Go
	// يعني قراءةَ كل سائقٍ في المنصة لاختيار واحد.
	err := s.db.QueryRow(ctx, `
		SELECT u.id
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE u.on_shift AND u.status = 'active'
		  -- **COALESCE لا يُستغنى عنه**: أوّلُ عرضٍ يمرّ بـskip فارغاً، و
		  -- NOT (id = ANY(NULL)) يُنتج NULL لا TRUE فيسقط كلُّ سائق.
		  AND NOT (u.id = ANY(COALESCE($1::uuid[], '{}')))
		  -- **وسقفُ النقد يُقاس بما بحوزته وبنقد هذا الطلب معاً.**
		  AND COALESCE((SELECT b.held FROM driver_cash_boxes b
		                WHERE b.driver_id = u.id), 0)
		      -- **والمُسنَدُ الذي لم يُسلَّم يُحسب** — صيغةُ cashbox.Exposure (فحصُ المتجر
		      --  ٢٠٢٦-١٠-٠١: سائقٌ أُسنِد إليه ثلاثةٌ في ثلاث ثوانٍ فبلغ ٥٦١٬٤٠٠ والسقفُ ٥٠٠٬٠٠٠).
		      + COALESCE((SELECT sum(oi.cash_due) FROM orders oi
		                  WHERE oi.driver_id = u.id AND oi.closed_at IS NULL), 0)
		      + COALESCE((SELECT o.cash_due FROM orders o WHERE o.id = $4), 0) <= $2
		  AND (SELECT count(*) FROM orders o
		       WHERE o.driver_id = u.id AND o.closed_at IS NULL) < $3
		  -- **وعرضٌ حيٌّ واحدٌ لكلّ سائق.**
		  AND NOT EXISTS (
		      SELECT 1 FROM orders o2
		      WHERE o2.offered_driver_id = u.id AND o2.id <> $4
		        AND o2.status = 'dispatching' AND o2.driver_id IS NULL
		        AND o2.offer_expires_at > now())
		-- **ومن لم يأخذ بعدُ يُرتّبون بمن بكّر بالدوام.**
		ORDER BY u.last_assigned_at NULLS FIRST, u.shift_started_at, u.id
		LIMIT 1`, skip, limit, maxActive, orderID).Scan(&driverID)
	return driverID, err
}

// zoneGateClause بوّابةُ المنطقة الاختياريّة — تُضاف حين تُشعَل: موضعُ السائق
// داخلَ منطقةِ الطلب (`orders.zone_id` على هندسة `delivery_zones`، دائرةً أو
// مضلّعاً). **وطلبٌ بلا منطقةٍ يمرّ** — لا نمنع ما لم نُصنّفه.
const zoneGateClause = `
		  AND ( (SELECT o3.zone_id FROM orders o3 WHERE o3.id = $4) IS NULL
		    OR EXISTS (
		        SELECT 1 FROM delivery_zones z
		        WHERE z.id = (SELECT o3.zone_id FROM orders o3 WHERE o3.id = $4)
		          AND z.active
		          AND ( (z.shape = 'radius' AND z.center IS NOT NULL
		                 AND ST_DWithin(z.center, u.last_location, z.radius_m))
		             OR (z.shape = 'polygon' AND z.area IS NOT NULL
		                 AND ST_Covers(z.area, u.last_location)) ) ) )`

// **جسمُ الأهليّة المشترك** — دوامٌ وحالةٌ وتخطٍّ ونقدٌ وعددُ طلباتٍ وعرضٌ حيٌّ
// واحد، **وحداثةُ الموقعِ شرطٌ لا يسقط** (`last_location` حديثٌ). يُبنى فوقه
// استعلامان: داخلَ الحلقة، وحديثٌ بلا حلقة.
const proximityEligibleWhere = `
		WHERE u.on_shift AND u.status = 'active'
		  AND NOT (u.id = ANY(COALESCE($1::uuid[], '{}')))
		  AND COALESCE((SELECT b.held FROM driver_cash_boxes b
		                WHERE b.driver_id = u.id), 0)
		      -- **والمُسنَدُ الذي لم يُسلَّم يُحسب** — صيغةُ cashbox.Exposure (فحصُ المتجر
		      --  ٢٠٢٦-١٠-٠١: سائقٌ أُسنِد إليه ثلاثةٌ في ثلاث ثوانٍ فبلغ ٥٦١٬٤٠٠ والسقفُ ٥٠٠٬٠٠٠).
		      + COALESCE((SELECT sum(oi.cash_due) FROM orders oi
		                  WHERE oi.driver_id = u.id AND oi.closed_at IS NULL), 0)
		      + ord.cash_due <= $2
		  AND (SELECT count(*) FROM orders o2
		       WHERE o2.driver_id = u.id AND o2.closed_at IS NULL) < $3
		  AND NOT EXISTS (
		      SELECT 1 FROM orders o2
		      WHERE o2.offered_driver_id = u.id AND o2.id <> $4
		        AND o2.status = 'dispatching' AND o2.driver_id IS NULL
		        AND o2.offer_expires_at > now())
		  -- **الحداثةُ شرطٌ لا يسقط أبداً حين يكون القربُ مُفعَّلاً** — ولو بلغ
		  --  نصفُ القطر أقصاه: مجهولُ الموضع أو شائخُه لا يُعرض عليه.
		  AND u.last_location IS NOT NULL
		  AND u.last_location_at > now() - make_interval(secs => $5)`

const proximityCTE = `
		SELECT u.id
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		CROSS JOIN (
		    SELECT ` + DispatchAnchorSQL + ` AS pickup,
		           COALESCE(o.cash_due, 0) AS cash_due
		    FROM orders o LEFT JOIN merchants m ON m.id = o.merchant_id
		    WHERE o.id = $4
		) ord`

// proximityInRadius الأقربُ المؤهَّلُ حديثُ الموقع **داخلَ حلقةٍ** — الجغرافيا
// تُرشّح (فهرسُ GIST)، والمسافةُ تُرتّب، والشريحةُ (`$7`) تمنع الجوع.
func (s *Service) proximityInRadius(ctx context.Context, orderID string, skip []string, limit, maxActive, fresh, radius, bucket int64) (string, error) {
	zoneClause := ""
	if s.settings != nil && s.settings.GetBool(ctx, "drivers.zone_gate_enabled") {
		zoneClause = zoneGateClause
	}
	q := proximityCTE + proximityEligibleWhere + `
		  AND ord.pickup IS NOT NULL
		  AND ST_DWithin(u.last_location, ord.pickup, $6)` + zoneClause + `
		ORDER BY floor(ST_Distance(u.last_location, ord.pickup) / GREATEST($7::float8, 1)),
		         u.last_assigned_at NULLS FIRST, u.shift_started_at, u.id
		LIMIT 1`
	var driverID string
	err := s.db.QueryRow(ctx, q, skip, limit, maxActive, orderID, fresh, radius, bucket).Scan(&driverID)
	return driverID, err
}

// freshFallbackCandidate **شبكةُ الأمان — حديثةُ الموقعِ لا شائختُه.**
//
// **تُسقط حدَّ المسافة وحدَه وتُبقي الحداثة** (`proximityEligibleWhere`): تُنادى
// حين بلغ نصفُ القطر أقصاه ولا قريبٌ داخلَه، أو حين لا نقطةَ التقاطٍ تُقاس.
// **فتختار أقربَ مؤهَّلٍ حديثِ الموقع أينما كان، أو لا تختار** — ولا تعرض على
// مجهولِ الموضع. **والعدلُ الأعمى عن الموضع في `legacyRotationCandidate` وحدَها،
// ولا تُنادى إلّا حين يُطفئ المشغّلُ القربَ.**
//
// **والترتيبُ بالمسافة إن وُجدت النقطة، وإلّا بالعدل** — `NULLS LAST` تجعل من لا
// مسافةَ له (لا نقطةَ التقاط) يُرتَّب بالعدل وحدَه.
func (s *Service) freshFallbackCandidate(ctx context.Context, orderID string, skip []string, limit, maxActive, fresh, bucket int64) (string, error) {
	zoneClause := ""
	if s.settings != nil && s.settings.GetBool(ctx, "drivers.zone_gate_enabled") {
		zoneClause = zoneGateClause
	}
	q := proximityCTE + proximityEligibleWhere + zoneClause + `
		ORDER BY floor(ST_Distance(u.last_location, ord.pickup) / GREATEST($6::float8, 1)) NULLS LAST,
		         u.last_assigned_at NULLS FIRST, u.shift_started_at, u.id
		LIMIT 1`
	var driverID string
	err := s.db.QueryRow(ctx, q, skip, limit, maxActive, orderID, fresh, bucket).Scan(&driverID)
	return driverID, err
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
	// ══════════════════════════════════════════════════════════════════
	// **وحارسُ القبول نفسُه قبل الإسناد** — فحصُ المتجر ٢٠٢٦-١٠-٠١
	// ══════════════════════════════════════════════════════════════════
	//
	// **قِيس على التجهيز**: ثلاثةُ طلباتٍ أُسنِدت إلى سائقٍ في ثلاث ثوانٍ،
	// **فبلغ نقدُه ٥٦١٬٤٠٠ والسقفُ ٥٠٠٬٠٠٠.** بابُ «خذ الطلب» يمرّ بـ
	// `AdmitDriverTx` (قفلُ السائق + التعرّضُ كلُّه)، **والإسنادُ التلقائيُّ لم
	// يكن يمرّ** — كان يكتفي بمرشّح الاستعلام، **وإسنادان متتاليان يقرأ كلٌّ
	// منهما ما قبل الآخر.**
	//
	// **والسقفُ والنقدُ يُقرآن قبل فتح المعاملة** (`XG-46`).
	// **وسقفُ العدد لا يُفحص هنا** (`ActiveMax: -1`): المرشِّحُ يحمله، **وطريقُ
	// «في طريقه» أعلى بواحدٍ بقرار المالك** — والحارسُ لا يعرف أيَّ طريقٍ جاء.
	var cashDue int64
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(cash_due, 0) FROM orders WHERE id = $1`, orderID).Scan(&cashDue); err != nil {
		return err
	}
	limit := s.settingInt(ctx, "drivers.cash_limit")
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.AdmitDriverTx(ctx, tx, driverID, Admission{
		CashDue: cashDue, CashLimit: limit, ActiveMax: -1,
	}); err != nil {
		// **لا يُسنَد، ولا يُعلَّق عليه** — يبقى في الطابور لمن يحتمله.
		s.logger.Info("الترتيب: الإسنادُ التلقائيُّ ردّه الحارس",
			"order", orderID, "driver", driverID, "reason", err.Error())
		_, _ = s.db.Exec(ctx, `
			UPDATE orders SET offered_driver_id = NULL, offer_expires_at = NULL
			WHERE id = $1 AND driver_id IS NULL AND offered_driver_id = $2`, orderID, driverID)
		return nil
	}
	// **الشرطُ الذرّيّ يبقى**: سائقٌ ضغط «خذ الطلب» في اللحظة نفسِها يجد صفراً.
	tag, err := tx.Exec(ctx, `
		UPDATE orders
		SET offered_driver_id = $2, driver_id = $2, updated_at = now(),
		    offer_expires_at = now() + make_interval(secs => $3)
		WHERE id = $1 AND status = 'dispatching' AND driver_id IS NULL`,
		orderID, driverID, s.silenceTimeout(ctx).Seconds())
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
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
	// **والمُسنَدُ إليه يُنبَّه** — الطلبُ صار مهمّتَه بلا أن يطلبه،
	// **ومن لم يُنبَّه لم يتحرّك** حتّى تنقضي مهلةُ الصمت فيُنزع منه.
	s.notifyOffer(ctx, orderID, driverID)
	return nil
}

// autoAssignNote نصُّ حدثِ الإسناد التلقائيّ — **يُقرأ في سجلّ الطلب.**
const autoAssignNote = "إسنادٌ تلقائيٌّ بالدور"

// ══════════════════════════════════════════════════════════════════════
// DispatchAnchorSQL **النقطةُ التي يُقاس منها قربُ السائق عند التوزيع**
// ══════════════════════════════════════════════════════════════════════
//
// **موضعُ الاستلام البديلُ إن وُجد، وإلّا المتجر، وإلّا — في الطلب الخاصّ —
// باب الزبون** (٢٠٢٦-١٠-٠٢).
//
// **كان الخاصُّ بلا نقطةٍ تُقاس** (لا متجرَ له) — فيُعرض بالعدل وحدَه على
// سائقٍ في أقصى المدينة **وقريبٌ من الزبون واقفٌ بجانبه.** والسائقُ يشتري من
// سوقٍ قريبٍ من الزبون في الغالب، **فالزبونُ أصدقُ مرساةٍ لا لاشيء.**
//
// **وفي المحرّك وبابِ الطابور نصٌّ واحد** — قياسان يفترقان يعرضان على واحدٍ
// ويُظهران لآخر.
const DispatchAnchorSQL = `COALESCE(o.pickup_override, m.location, CASE WHEN o.kind = 'custom' THEN o.dropoff END)`

// reclaimNote نصُّ حدثِ نزعِ إسنادٍ صامت.
const reclaimNote = "نُزع لعدم التحرّك — عاد إلى الطابور"

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
		// ══════════════════════════════════════════════════════════════
		// **والنزعُ يُكتب في السجلّ ويُقال لصاحبه** (٢٠٢٦-١٠-٠٢)
		// ══════════════════════════════════════════════════════════════
		//
		// **كان يمرّ بلا حدثٍ ولا خبر** — فيفتح السائقُ تطبيقَه فلا يجد طلبَه
		// ولا يعرف لماذا، **وسجلُّ الطلب يقفز من «أُسند» إلى «أُسند» لغيره.**
		if _, e := s.db.Exec(ctx, `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
			VALUES ($1, 'assigned', 'dispatching', NULL, $3, $2)`,
			x.orderID, x.driverID, reclaimNote); e != nil {
			s.logger.Error("الترتيب: تعذّر قيدُ حدثِ النزع", "order", x.orderID, "error", e)
		}
		s.notifyDriverLost(ctx, x.orderID, x.driverID, LossRequeuedSystem)
		s.pub.Publish("driver:"+x.driverID, map[string]any{"type": "order"})
		s.pub.Publish("ops", map[string]any{"type": "order"})
		// **والزبونُ يرى طلبَه عاد إلى الطابور** — كما في كلّ مسارٍ يمرّ
		// بالمحرّك (`publishOrder`). **كان النزعُ وحدَه يسكت عنه**، فتبقى
		// شاشتُه على سائقٍ نُزع منه الطلب حتّى يُحدّثها بيده.
		if o, err := s.GetByID(ctx, x.orderID); err == nil {
			s.publishOrder(o)
		}
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
	s.sweepOfferExpiry(ctx)
	s.offerWaiting(ctx)
}

// ══════════════════════════════════════════════════════════════════════
// **العرضُ ينتقل لحظةَ موته — لا بعد نبضة الراصد** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الانتقالُ مع نبضة الراصد (ثلاثون ثانية)** — فعرضٌ مهلتُه دقيقةٌ يبقى
// ميّتاً عند صاحبه حتّى نصفَ دقيقةٍ أخرى. (قِيس: فجواتٌ نحو عشرين ثانية.)
//
// **فحلقةٌ سريعةٌ للعروض وحدَها** — كلَّ ثانية: **سؤالٌ واحدٌ رخيصٌ** عن عرضٍ
// انقضى، **ولا شيءَ يُفعل إن لم يوجد.** **ومن القاعدة لا من مؤقّتٍ في الذاكرة**:
// إعادةُ التشغيل لا تُضيّع عرضاً، فالحلقةُ تقرأ ما في الصفّ حين تعود.
//
// **وما ينتظر بلا عرضٍ يبقى على نبضة الراصد** (`SweepWaitingOffers`) — هو
// الأثقل (يسأل عن مرشّحٍ لكلّ منتظِر)، **ولا موعدَ ينقضي فيه.**
func (s *Service) RunOfferSweeper(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if s.AssignmentMode(ctx) == "rotation" {
				s.sweepOfferExpiry(ctx)
			}
		}
	}
}

// SweepWaitingOffers **ما ينتظر بلا عرض** — على نبضة الراصد.
func (s *Service) SweepWaitingOffers(ctx context.Context) {
	if s.AssignmentMode(ctx) != "rotation" {
		return
	}
	s.offerWaiting(ctx)
}

// sweepOfferExpiry **العروضُ المنقضيةُ والإسناداتُ الصامتة** — وحدَها.
func (s *Service) sweepOfferExpiry(ctx context.Context) {
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
			UPDATE orders
			-- **ولا يُوسَم معرّفٌ مرّتين** — النبضةُ تتكرّر كلَّ ثلاثين ثانيةً
			-- على طلبٍ ساكن، **فيصير الصفُّ سجلَّ نبضاتٍ لا سجلَّ جولة.**
			--
			-- **والعرضُ الميّتُ يُفرَّغ مع الوسم** — فلا يراه صاحبُه في طابوره
			-- ولا يستطيع قبولَه، **ولا تعود إليه الحلقةُ في الثانية التالية**:
			-- إن لم يوجد غيرُه صار منتظِراً تلتقطه نبضةُ الراصد.
			SET offer_passed = CASE WHEN $2::uuid = ANY(offer_passed)
			                        THEN offer_passed ELSE offer_passed || $2::uuid END,
			    offered_driver_id = NULL, offer_expires_at = NULL
			-- **والشرطُ يتكرّر في التحديث** — حلقتان أو مُنفّذان يقرآن العرضَ
			-- نفسَه فيأخذه أوّلُهما، **ولا يُنقل الدورُ مرّتين.**
			WHERE id = $1 AND offered_driver_id = $2
			  AND offer_expires_at IS NOT NULL AND offer_expires_at <= now()
			  AND status = 'dispatching' AND driver_id IS NULL
			RETURNING array(SELECT unnest(offer_passed)::text)`,
			e.orderID, e.driverID).Scan(&skip); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				s.logger.Error("الترتيب: تعذّر وسمُ مرور الدور",
					"order", e.orderID, "error", err)
			}
			continue
		}
		if err := s.OfferNext(ctx, e.orderID, skip); err != nil {
			s.logger.Error("الترتيب: تعذّر نقل الدور", "order", e.orderID, "error", err)
		}
	}
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

// groupDigits رقمٌ بفواصلِ ألوف — **2061000 لا تُقرأ، و2,061,000 تُقرأ.**
//
// **والأرقامُ أجنبيّةٌ وفاصلُ الآلاف «,»** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الأرقامُ كلُّها
// بالأجنبيّة في كلّ مكان، مركزيّاً… وفاصلُ الآلاف , لا ٬») — **كانت الفاصلةُ عربيّةً**
// (U+066C) فيقرأ السائقُ «3٬000» في الحديث و«3,000» في شاشته. ولا `golang.org/x/text`
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
			b.WriteByte(',')
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

// ══════════════════════════════════════════════════════════════════════
// **الرفضُ — ينقل الدورَ فوراً بدل أن يُنتظر انقضاؤه**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «بدل خذ الطلب: موافق ورفض — مشان إذا ما بدّه
//
//	يتحوّل لغيره».)
//
// # ولماذا لم يكن
//
// **كان الرفضُ صمتاً**: يترك السائقُ العرضَ حتّى تنقضي مهلتُه فينتقل.
// **والمهلةُ دقيقةٌ أو دقيقتان يقفها الطلبُ بلا سبب** — والزبونُ ينتظر،
// **والسائقُ الذي لا يريده يعرف ذلك في الثانية الأولى.**
//
// # وهو المسارُ نفسُه لا مسارٌ ثانٍ
//
// **يُوسَم مرورُ دوره** (`offer_passed`) **ثمّ يُعرض على من بعده** — وهو
// حرفيّاً ما تفعله `SweepExpiredOffers` حين تنقضي المهلة. **ومسارٌ ثانٍ
// للرفض** يفترق عنها يومَ يتغيّر أحدُهما.
//
// # ولا يُرفض ما ليس معروضاً عليه
//
// **الشرطُ في جملة التحديث نفسِها**: من رفض طلباً معروضاً على غيره
// **لم يمسّ شيئاً** — ولا يُنقل دورُ سائقٍ آخر بضغطةٍ من هذا.
func (s *Service) DeclineOffer(ctx context.Context, orderID, driverID string) error {
	rotation := s.AssignmentMode(ctx) == "rotation"

	// ══════════════════════════════════════════════════════════════════
	// **والرفضُ في «للجميع» إخفاءٌ لا نقل**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «زرُّ الرفض غير موجود كما اتّفقنا».)
	//
	// **ولا دورَ في «للجميع» ينتقل** — الطلبُ معروضٌ على الكلّ. **لكنّ
	// الرفضَ فيه معنىً آخر**: من رآه ولا يريده **يريده أن يختفي عن
	// شاشته**، لا أن يبقى يقرؤه كلَّ مرّة.
	//
	// **وهو يبقى لغيره** — لا يُنزع من الطابور.
	//
	// **والوسمُ واحدٌ في الحالين** (`offer_passed`): في «بالدور» يُعرض
	// على من بعده، **وفي «للجميع» يُحجب عن قائمة من رفضه** — ونقطةُ
	// الطابور تقرؤه.
	where := "id = $1 AND status = 'dispatching' AND driver_id IS NULL"
	if rotation {
		// **ولا يُنقل دورُ سائقٍ آخر بضغطةٍ من هذا.**
		where += " AND offered_driver_id = $2"
	}

	var skip []string
	err := s.db.QueryRow(ctx, `
		UPDATE orders
		SET offer_passed = offer_passed || $2::uuid,
		    offered_driver_id = CASE WHEN offered_driver_id = $2 THEN NULL ELSE offered_driver_id END,
		    offer_expires_at = CASE WHEN offered_driver_id = $2 THEN NULL ELSE offer_expires_at END
		WHERE `+where+`
		RETURNING array(SELECT unnest(offer_passed)::text)`,
		orderID, driverID).Scan(&skip)
	if err != nil {
		// **ولا صفَّ يُصيبه**: العرضُ ليس له، أو أخذه غيرُه، أو انقضى.
		// **وهو ردٌّ طبيعيٌّ لا عطب** — والبطاقةُ اختفت عنه أصلاً.
		return ErrOfferNotYours
	}
	if !rotation {
		return nil
	}
	return s.OfferNext(ctx, orderID, skip)
}

// excludedDrivers **من ترك هذا الطلبَ فلا يعود إليه** — وفارغٌ إن لم يتركه أحد.
func (s *Service) excludedDrivers(ctx context.Context, orderID string) []string {
	var out []string
	if err := s.db.QueryRow(ctx,
		`SELECT array(SELECT unnest(excluded_drivers)::text) FROM orders WHERE id = $1`,
		orderID).Scan(&out); err != nil {
		return nil
	}
	return out
}

// ExcludeDriverTx **يُثبّت أنّ هذا السائقَ ترك الطلب — فلا يعود إليه أبداً.**
//
// **في معاملة الترك نفسِها** — فالعرضُ التالي بعد التثبيت يقرؤه.
func (s *Service) ExcludeDriverTx(ctx context.Context, q dbtx.Querier, orderID, driverID string) error {
	_, err := q.Exec(ctx, `
		UPDATE orders
		SET excluded_drivers = CASE WHEN $2::uuid = ANY(excluded_drivers)
		                            THEN excluded_drivers ELSE excluded_drivers || $2::uuid END
		WHERE id = $1`, orderID, driverID)
	return err
}
