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

// GrantTargetIfReached **يُكافئ من بلغ — ويصمت عمّن لم يبلغ أو كوفئ.**
//
// **يردّ المبلغَ إن دُفع الآن، وصفراً في كلّ ما عداه** — فيُعرَف متى يُحتفَل.
//
// **ولا يُنادى إلّا بعد تسليمٍ يُغلَق**: هي اللحظةُ الوحيدةُ التي يتبدّل فيها
// العدّاد. **ونداؤها في كلّ فتحةِ شاشةٍ يجعل القراءةَ تكتب** — وشاشةٌ تُصرف
// مالاً بمجرّد أن تُفتح لا تُراجَع.
// GrantTargetIfReachedTx كـ`GrantTargetIfReached` **في معاملةٍ مُمرَّرة**.
//
// **وُجدت لأجل تحويل المرشَّح** (`PF-01`): **كانت المكافأةُ تُدفع قبل
// تثبيت التحويل** — **فيُدفَع مالٌ عن تحويلٍ لم يقع.**
//
// **ويُردّ الخطأُ هنا ولا يُبتلَع**: **القائمةُ تصمت عن العثرة لأنّها
// خارجَ عمليّةٍ أكبر** — **وهذه داخلَها، فصمتُها يترك المعاملةَ تُثبَّت
// بلا مكافأة.**
func (s *Service) GrantTargetIfReachedTx(ctx context.Context, tx dbtx.Querier,
	userID, role string) (int64, error) {
	levels := s.levelsFor(ctx, role)
	if len(levels) == 0 {
		return 0, nil
	}
	done, err := s.doneThisMonth(ctx, userID, role)
	if err != nil {
		return 0, err
	}
	period := PeriodOf(time.Now())
	var paid int64
	for _, l := range levels {
		if l.Target <= 0 || l.Reward <= 0 || done < l.Target {
			continue
		}
		if err := s.grantTargetTx(ctx, tx, userID, role, l, period); err != nil {
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

func (s *Service) GrantTargetIfReached(ctx context.Context, userID, role string) int64 {
	levels := s.levelsFor(ctx, role)
	if len(levels) == 0 {
		return 0
	}

	done, err := s.doneThisMonth(ctx, userID, role)
	if err != nil {
		return 0
	}

	// ══════════════════════════════════════════════════════════════════
	// **وكلُّ مرحلةٍ بلغها تُدفع — لا الأعلى وحدَها**
	// ══════════════════════════════════════════════════════════════════
	//
	// **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «إذا بلغ الأولى يأخذها، ثمّ الثانية
	// يأخذها، ثمّ الثالثة يأخذها».)
	//
	// **ومن قفز من صفرٍ إلى العشرين في يومٍ واحدٍ نال الثلاثَ معاً** —
	// بلغها كلَّها فعلاً. **ومن يُحرَم ما استحقّه لأنّه تجاوزه يتعلّم أن
	// يقف عند الحدّ.**
	//
	// **والوسمُ يحمل رقمَ المرحلة** (`2026-08#2`) — **فالفهرسُ الفريدُ
	// القائمُ يمنع تكرارَ كلِّ واحدةٍ وحدَها**، بلا فهرسٍ جديدٍ ولا هجرة.
	period := PeriodOf(time.Now())
	var paid int64
	for _, l := range levels {
		if l.Target <= 0 || l.Reward <= 0 || done < l.Target {
			// **مطفأةٌ أو لم تُبلَغ** — والهدفُ يبقى عدّاداً بلا مال.
			continue
		}
		if err := s.grantTarget(ctx, userID, role, l, period); err != nil {
			// **والتصادمُ ليس خطأً** — هو الضمانةُ تعمل: نالها من قبل.
			if isDuplicate(err) {
				continue
			}
			s.logf("مكافأةُ الهدف تعذّرت", "user", userID, "level", l.N, "error", err)
			continue
		}
		paid += l.Reward
	}
	return paid
}

// doneThisMonth ما أنجزه في شهره — **بالمعنى الذي يخصّ دورَه.**
//
// # والمعنيان مختلفان اختلافاً تامّاً
//
// **السائقُ يُنجز بما وصّل** — فعلُه ينتهي بالتسليم.
//
// **والمندوبُ يُنجز بما فتح من متاجر** — **وعملُه ينتهي يومَ يوقّع
// العميل**، ولا يملك بعدها أن يجعله يبيع.
//
// # ولماذا تبدّل
//
// **كان يعدّ طلبات متاجره المسلَّمة** — **فمندوبٌ فتح عشرةَ متاجرَ في
// أسبوعٍ عدّادُه صفر** حتّى يشتري الناسُ منها. **وذلك يقيس السوقَ لا
// المندوب.**
//
// **(قرارُ المالك ٢٠٢٦-٠٨-٣١:** «الهدفُ الشهريّ هو عددُ العملاء
// المسجَّلين» · «كلُّ عميل — بالأساس كلُّ عميلٍ راح توافق عليه
// الإدارة، لن ترفض أيَّ عميلٍ أصلاً».) **فلا يُشترط قبولٌ ولا بيع.**
//
// **وأنا من ثبّت الخطأ**: رأيتُ اللوحةَ تقول «عميل» والشيفرةَ تعدّ
// طلبات، **فجعلتُ الكلمةَ تتبع الشيفرة** بدل أن تتبع الشيفرةُ القصد.
func (s *Service) doneThisMonth(ctx context.Context, userID, role string) (int64, error) {
	q := `SELECT count(*) FROM orders o
	      WHERE o.driver_id = $1 AND o.status = 'delivered'
	        AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart
	if role == "sales" {
		// **ولا تُستثنى حالة** — `active` و`inactive` و`suspended`
		// كلُّها متاجرُ فتحها، **وليس في الجدول حذفٌ ناعمٌ أصلاً.**
		// **ومتجرٌ عُوقب بعد شهرٍ لا يُسحب من رصيد من جلبه.**
		q = `SELECT count(*) FROM merchants m
		     WHERE m.sales_rep_user_id = $1
		       AND m.created_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart
	}
	var n int64
	err := s.db.QueryRow(ctx, q, userID).Scan(&n)
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, period, created_by)
		VALUES ($1, $2, $3, $4, true, $5, NULL)`,
		userID, KindReward, reward, reason, period); err != nil {
		return err
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
func isDuplicate(err error) bool {
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
