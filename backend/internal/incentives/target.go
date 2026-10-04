package incentives

// **مكافأةُ الهدف — تُدفع آليّاً عند بلوغه، ومرّةً في الشهر.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن تكون واضحة وتُدفع بشكل آليّ عند إتمام
//
//	الهدف، مع تأثير للشاشة عند وصول الهدف احتفاليّ للسائق».)
//
// # ولا تُدفع مرّتين — والقاعدةُ هي من يمنع
//
// **يُسلَّم طلبان في ثانيةٍ واحدة، فيقرأ كلاهما «لم يُكافأ بعد» ويكتب كلاهما
// مكافأة.** والفحصُ قبل الكتابة لا يمنعه: **بينهما ثغرةٌ يمرّ فيها الاثنان.**
//
// **فالفهرسُ الفريدُ هو الضمانة** (`incentives_one_target_per_month`)،
// **والثاني يُردّ بتصادمٍ يُبتلع** — لا بخطأٍ يُسقط تسليمَ طلب.
//
// # ولا تُوقف تسليمَ طلبٍ إن أخفقت
//
// **المكافأةُ أثرٌ جانبيٌّ للتسليم لا شرطٌ له.** ودفترٌ مغلقٌ أو خزينةٌ غائبةٌ
// **لا يجوز أن تمنع سائقاً من إقفال طلبٍ سلّمه** — يُقيَّد الخطأُ في السجلّ
// ويمضي التسليم.

import (
	"context"
	"errors"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"strings"
	"time"
)

// damascus **شهرُ السائق ينتهي عنده لا في غرينتش.**
//
// **وطلبٌ سُلّم الواحدةَ بعد منتصف الليل يُحسب على ليلته** لا على شهرٍ جديد.
var damascus = time.FixedZone("Asia/Damascus", 3*60*60)

// PeriodOf **وسمُ الشهر — «2026-08».**
//
// **ونصٌّ لا تاريخ**: الفهرسُ الفريدُ يحتاج تعبيراً ثابتاً، **و`date_trunc`
// بمنطقةٍ زمنيّةٍ ليست ثابتةً** فلا تدخل فهرساً.
func PeriodOf(t time.Time) string { return t.In(damascus).Format("2006-01") }

// Level **مرحلةٌ من مراحل الشهر — عددُها ومكافأتُها.**
type Level struct {
	// N رقمُها: ١ · ٢ · ٣ — **ويدخل وسمَ الشهر** فيمنع الفهرسُ تكرارَها وحدَها.
	N int `json:"n"`
	// Target كم يلزم لبلوغها — **وصفرٌ يعني مطفأة.**
	Target int64 `json:"target"`
	// Reward ما يُدفع عندها — **وصفرٌ يعني عدّاداً بلا مال.**
	Reward int64 `json:"reward"`
}

// prefixOf **بادئةُ مفاتيح الدور** — `drivers.` أو `sales.`
func prefixOf(role string) string {
	switch role {
	case "driver":
		return "drivers."
	case "sales":
		return "sales."
	}
	return ""
}

// levelsFor **مراحلُ الشهر الثلاثُ لهذا الدور.**
//
// # ولماذا ثلاثٌ لا واحدة
//
// **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «الهدفُ برأيي يكون على ٣ مراحل، يعني ٣
// مستويات · **إذا بلغ الأولى يأخذها ثمّ الثانية يأخذها ثمّ الثالثة
// يأخذها** · والسائقُ أيضاً نفسُ الشيء، ولكنّ السائقَ مختلفٌ عن
// المندوب».)
//
// **وهدفٌ واحدٌ يقتل الحافزَ مرّتين في الشهر**: من بلغه في اليوم العاشر
// لا شيءَ يدفعه بعده، **ومن تأخّر ورأى «بقي أربعة» في الخامس والعشرين
// استسلم.** **والمراحلُ تُبقي أمام كلٍّ منهما هدفاً قريبا.**
//
// # والأولى هي القديمةُ باسمها
//
// **`monthly_target` و`target_reward` تبقيان مرحلةً أولى** — **ولو
// أُعيدت تسميتُهما لَسقط ما ضبطه المالكُ من قبل**، ولوحةُ الإعدادات
// معهما.
//
// **والثانيةُ والثالثةُ صفرٌ حتّى يضبطهما** (قرارُه: «اجعلها مطفأة، أنا
// أضبطها لاحقاً») — **ومن نشر مراحلَ بأرقامٍ اخترعها دفع مالاً لم
// يقرّره صاحبُه.**
//
// **وأرقامُ الدورين مستقلّة**: السائقُ يعدّ طلبات، والمندوبُ عملاء.
func (s *Service) levelsFor(ctx context.Context, role string) []Level {
	p := prefixOf(role)
	if p == "" {
		return nil
	}
	return []Level{
		{N: 1,
			Target: s.settings.GetInt(ctx, p+"monthly_target"),
			Reward: s.settings.GetInt(ctx, p+"target_reward")},
		{N: 2,
			Target: s.settings.GetInt(ctx, p+"target_2"),
			Reward: s.settings.GetInt(ctx, p+"reward_2")},
		{N: 3,
			Target: s.settings.GetInt(ctx, p+"target_3"),
			Reward: s.settings.GetInt(ctx, p+"reward_3")},
	}
}

// LevelsOf **مراحلُ الدور — للشاشات.**
func (s *Service) LevelsOf(ctx context.Context, role string) []Level {
	out := []Level{}
	for _, l := range s.levelsFor(ctx, role) {
		if l.Target > 0 {
			out = append(out, l)
		}
	}
	return out
}

// GrantTargetIfReachedTx كـ`GrantTargetIfReached` **في معاملةٍ مُمرَّرة** — للشهر الجاري.
//
// **وُجدت لأجل تحويل المرشَّح** (`PF-01`): **كانت المكافأةُ تُدفع قبل
// تثبيت التحويل** — **فيُدفَع مالٌ عن تحويلٍ لم يقع.**
//
// **ويُردّ الخطأُ هنا ولا يُبتلَع**: هذه داخلَ عمليّةٍ أكبر، **فصمتُها يترك
// المعاملةَ تُثبَّت بلا مكافأة.**
func (s *Service) GrantTargetIfReachedTx(ctx context.Context, tx dbtx.Querier,
	userID, role string) (int64, error) {
	return s.grantForMonthTx(ctx, tx, userID, role, PeriodOf(time.Now()))
}

// grantForMonthTx **يكافئ على مراحل شهرٍ بعينه** — المصدرُ الواحدُ للصرف الآليّ.
//
// **وشهرٌ بعينه لا «الشهرُ الجاري» دائماً**: إعادةُ مكافأةٍ تعثّرت في آخر
// أيلول تُحسب على أيلول ولو أُعيدت في تشرين.
//
// ══════════════════════════════════════════════════════════════════════
// **وكلُّ مرحلةٍ بلغها تُدفع — لا الأعلى وحدَها** (قرارُ المالك ٢٠٢٦-٠٨-٣١)
// ══════════════════════════════════════════════════════════════════════
//
// **والوسمُ يحمل رقمَ المرحلة** (`2026-08#2`) — فالفهرسُ الفريدُ يمنع تكرارَ
// كلِّ واحدةٍ وحدَها.
//
// **والحسابُ الموقوفُ لا يُكافأ** (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الأهداف):
// مكافأةٌ تُصرف لموقوفٍ تُقرأ مكافأةً على ما أوقف بسببه.
func (s *Service) grantForMonthTx(ctx context.Context, tx dbtx.Querier,
	userID, role, month string) (int64, error) {
	levels := s.levelsFor(ctx, role)
	if len(levels) == 0 {
		return 0, nil
	}
	active, err := isActive(ctx, tx, userID)
	if err != nil || !active {
		return 0, err
	}
	// **ويُعَدُّ بالمعاملة نفسِها** — `XG-32`: **وإلّا لم يُرَ متجرُها.**
	done, err := s.doneInMonthOn(ctx, tx, userID, role, month)
	if err != nil {
		return 0, err
	}
	var paid int64
	for _, l := range levels {
		if l.Target <= 0 || l.Reward <= 0 || done < l.Target {
			// **مطفأةٌ أو لم تُبلَغ** — والهدفُ يبقى عدّاداً بلا مال.
			continue
		}
		if err := s.grantTargetTx(ctx, tx, userID, role, l, month); err != nil {
			// **والمكرَّرُ ليس عثرة** — كوفئ عن هذه المرحلة سابقاً.
			if isDuplicate(err) {
				continue
			}
			return 0, err
		}
		paid += l.Reward
	}
	return paid, nil
}

// GrantTargetIfReached **يُكافئ من بلغ — ويصمت عمّن لم يبلغ أو كوفئ.**
//
// **يردّ المبلغَ إن دُفع الآن، وصفراً في كلّ ما عداه** — فيُعرَف متى يُحتفَل.
//
// **ولا يُنادى إلّا بعد تسليمٍ يُغلَق**: هي اللحظةُ الوحيدةُ التي يتبدّل فيها
// العدّاد. **ونداؤها في كلّ فتحةِ شاشةٍ يجعل القراءةَ تكتب.**
//
// **وعثرتُه لا تُسقط التسليم — ولا تُبلَع**: تُكتب في السجلّ وفي
// `incentive_grant_failures` فتُعاد دوريّاً (`RetryFailures`) ومن زرٍّ في الصفحة.
func (s *Service) GrantTargetIfReached(ctx context.Context, userID, role string) int64 {
	month := PeriodOf(time.Now())
	paid, err := s.grantForMonth(ctx, userID, role, month)
	if err != nil {
		s.RecordFailure(ctx, userID, role, month, err)
		return 0
	}
	return paid
}

// grantForMonth **بمعاملته** — للمنادي المنفرد.
func (s *Service) grantForMonth(ctx context.Context, userID, role, month string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	paid, err := s.grantForMonthTx(ctx, tx, userID, role, month)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return paid, nil
}

// isActive **أفعّالٌ هو؟** — الموقوفُ والمحذوفُ لا يُكافآن.
func isActive(ctx context.Context, q dbtx.Querier, userID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM users
		                WHERE id = $1::uuid AND status = 'active' AND deleted_at IS NULL)`,
		userID).Scan(&ok)
	return ok, err
}

// monthRangeSQL **حدّا الشهر بتوقيت دمشق** من وسمٍ «2026-09» في المعلَمة `$N`.
//
// **تعبيرٌ واحدٌ للعرض وللصرف وللتنبيه** — ورقمٌ يُرى غيرُ الذي يُدفع عليه
// أسوأُ من رقمٍ لا يُرى.
func monthFromSQL(p string) string {
	return `((` + p + ` || '-01')::timestamp AT TIME ZONE 'Asia/Damascus')`
}

func monthToSQL(p string) string {
	return `(((` + p + ` || '-01')::timestamp + interval '1 month') AT TIME ZONE 'Asia/Damascus')`
}

// doneSQL **ما أنجزه في شهرٍ — بالمعنى الذي يخصّ دورَه.**
//
// **السائقُ يُنجز بما وصّل** — والمسترجَعُ لم يعد مسلَّماً فيخرج.
//
// **والمندوبُ يُنجز بما فتح من متاجر** (قرارُ المالك ٢٠٢٦-٠٨-٣١) — **ويُحسب
// للمندوب الذي فتحه للأبد** (`opened_by_rep_id`، قرارُ المالك ٢٠٢٦-١٠-٠٤):
// **كان العدُّ على المندوب الحاليّ، فالمتجرُ المنقولُ يدخل عدّادَ الجديد
// والقديمُ قبض عليه — مكافأةٌ مرّتين.** والنقلُ ينقل العمولةَ القادمةَ وحدَها.
//
// `who` تعبيرُ المستخدم، و`month` تعبيرُ وسم الشهر.
func doneSQL(role, who, month string) string {
	if role == "sales" {
		return `(SELECT count(*) FROM merchants mm
		          WHERE mm.opened_by_rep_id = ` + who + `
		            AND mm.created_at >= ` + monthFromSQL(month) + `
		            AND mm.created_at <  ` + monthToSQL(month) + `)`
	}
	return `(SELECT count(*) FROM orders o
	          WHERE o.driver_id = ` + who + ` AND o.status = 'delivered'
	            AND o.delivered_at >= ` + monthFromSQL(month) + `
	            AND o.delivered_at <  ` + monthToSQL(month) + `)`
}

// doneThisMonthOn العدُّ للشهر الجاري **بالمنفّذ المُمرَّر** — `XG-32`.
//
// **كان العدُّ يقرأ من المَسبَح دائماً** — والمتجرُ الذي يُنشأ داخلَ معاملة
// التحويل غيرُ مُثبَّتٍ بعدُ فلا يراه. **فالقراءةُ هي التي تنضمّ إلى
// المعاملة، لا العملُ الذي يخرج منها.**
func (s *Service) doneThisMonthOn(ctx context.Context, q dbtx.Querier,
	userID, role string) (int64, error) {
	return s.doneInMonthOn(ctx, q, userID, role, PeriodOf(time.Now()))
}

func (s *Service) doneInMonthOn(ctx context.Context, q dbtx.Querier,
	userID, role, month string) (int64, error) {
	var n int64
	err := q.QueryRow(ctx, `SELECT `+doneSQL(role, "$1::uuid", "$2::text"), userID, month).Scan(&n)
	return n, err
}

// grantTarget القيدُ نفسُه — **بالمعاملة، والخزينةُ الطرفُ المقابل.**
//
// **ولا يُعاد بناءُ ما في `Grant`**: هي تأخذ فاعلاً بشريّاً، **وهذه بلا فاعل**
// — ومن كتب `actor` وهميّاً جعل السجلَّ يقول إنّ إنساناً قرّر.
// grantTarget يفتح معاملتَه — للمنادي المنفرد.
func (s *Service) grantTarget(ctx context.Context, userID, role string,
	l Level, period string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.grantTargetTx(ctx, tx, userID, role, l, period); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// grantTargetTx يكتب في معاملةٍ مُمرَّرة — **ولا يثبّتها.**
func (s *Service) grantTargetTx(ctx context.Context, tx dbtx.Querier,
	userID, role string, l Level, period string) error {

	// **والوحدةُ بوحدة الدور** — **وكان يقول «طلباً» للمندوب أيضاً**
	// وهدفُه عملاءُ لا طلبات. **وقيدٌ يسمّي غيرَ ما وقع يُقرأ في كشف
	// حسابه فيظنّ أنّ المكافأةَ عن شيءٍ آخر.**
	unit := " طلباً"
	if role == "sales" {
		unit = " عميلاً"
	}
	reason := "مكافأةُ المرحلة " + itoa(int64(l.N)) + " (" + period + ") — " +
		itoa(l.Target) + unit
	// **والوسمُ يحمل رقمَ المرحلة** — انظر `GrantTargetIfReached`.
	period += "#" + itoa(int64(l.N))
	reward := l.Reward

	// **والصفُّ أوّلاً**: هو ما يحمل الفهرسَ الفريد، **فيُردّ المكرَّرُ قبل أن
	// يمسّ الدفتر.** ولو كُتب المالُ أوّلاً لَخرج ثمّ رُدّ القيد.
	// ══════════════════════════════════════════════════════════════
	// **ولا يُترَك القيدُ يُفسد معاملةَ غيرِه** — `XG-32`
	// ══════════════════════════════════════════════════════════════
	//
	// **`incentives_one_target_per_month` هو الحارسُ الأخير** ويبقى.
	// **لكنّ خرقَه داخلَ معاملةٍ مشتركةٍ يُجهضها كلَّها** — فيسقط
	// التحويلُ بـ`500` وإن كانت المكافأةُ وحدَها هي المكرَّرة.
	// (مقيس: تحويلٌ ثانٍ بعد بلوغ الهدف يُردّ `500`.)
	//
	// **و`continue` بعد الخرق لا ينفع**: **بوستغرس يُجهض المعاملةَ عند
	// أوّل خرق**، فكلُّ ما بعده يسقط.
	//
	// **فيُسأل القيدُ قبل أن يُخرَق**: `ON CONFLICT DO NOTHING` —
	// **والحارسُ في المخطَّط كما هو، والمعاملةُ تمضي.** **ومن لم يُدخَل
	// له صفٌّ لا يُقيَّد له مال.**
	tag, err := tx.Exec(ctx, `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, period, created_by,
		                        target_count, target_role)
		VALUES ($1, $2, $3, $4, true, $5, NULL, $6, $7)
		ON CONFLICT DO NOTHING`,
		userID, KindReward, reward, reason, period, l.Target, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// **كوفئ عن هذه المرحلة سابقاً** — ولا تكرار.
		return errAlreadyGranted
	}
	if _, err := s.wallet.ApplyTx(ctx, tx, userID, reward, KindReward, "", reason, nil); err != nil {
		return err
	}
	// **والخزينةُ تدفع** — دفترٌ يأخذ من طرفٍ ولا يعطي آخرَ لا يتوازن.
	if tid := s.treasury(ctx); tid != "" {
		if _, err := s.wallet.ApplyTx(ctx, tx, tid, -reward, KindReward, "",
			"مكافأةُ هدفٍ صُرفت", nil); err != nil {
			return err
		}
	}
	return nil
}

// isDuplicate **أهو تصادمُ الفهرس الفريد؟**
//
// **والنصُّ لا الرمزُ** لأنّ `pgconn.PgError` يقتضي استيراداً لا تحتاجه الحزمة
// لغيره — **والاسمُ مكتوبٌ في الهجرة فلا يتبدّل صامتاً.**
// errAlreadyGranted **كوفئ عن هذه المرحلة سابقاً** — ليس عطباً.
//
// **ولا يُقرأ من نصّ خطأ القاعدة**: `isDuplicate` تفحص اسمَ القيد في
// نصّ الخطأ، **وذلك يقع بعد أن تكون المعاملةُ أُجهضت.** **وهذا يُرجَع
// قبل أيّ خرق.**
var errAlreadyGranted = errors.New("incentives: كوفئ عن هذه المرحلة سابقاً")

func isDuplicate(err error) bool {
	if errors.Is(err, errAlreadyGranted) {
		return true
	}
	return err != nil && strings.Contains(err.Error(), "incentives_one_target_per_month")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

var errNoLogger = errors.New("incentives: لا سجلّ")

// logf **يُقال ما وقع ولا يُسقَط شيء.**
func (s *Service) logf(msg string, args ...any) {
	if s.logger == nil {
		_ = errNoLogger
		return
	}
	s.logger.Error(msg, args...)
}
