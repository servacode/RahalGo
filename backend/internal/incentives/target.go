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

// targetFor الهدفُ ومكافأتُه لهذا الدور — **وصفرٌ يعني بلا مكافأةٍ آليّة.**
func (s *Service) targetFor(ctx context.Context, role string) (target, reward int64) {
	switch role {
	case "driver":
		return s.settings.GetInt(ctx, "drivers.monthly_target"),
			s.settings.GetInt(ctx, "drivers.target_reward")
	case "sales":
		return s.settings.GetInt(ctx, "sales.monthly_target"),
			s.settings.GetInt(ctx, "sales.target_reward")
	}
	return 0, 0
}

// GrantTargetIfReached **يُكافئ من بلغ — ويصمت عمّن لم يبلغ أو كوفئ.**
//
// **يردّ المبلغَ إن دُفع الآن، وصفراً في كلّ ما عداه** — فيُعرَف متى يُحتفَل.
//
// **ولا يُنادى إلّا بعد تسليمٍ يُغلَق**: هي اللحظةُ الوحيدةُ التي يتبدّل فيها
// العدّاد. **ونداؤها في كلّ فتحةِ شاشةٍ يجعل القراءةَ تكتب** — وشاشةٌ تُصرف
// مالاً بمجرّد أن تُفتح لا تُراجَع.
func (s *Service) GrantTargetIfReached(ctx context.Context, userID, role string) int64 {
	target, reward := s.targetFor(ctx, role)
	if target <= 0 || reward <= 0 {
		// **بلا هدفٍ أو بلا مكافأةٍ لا شيءَ يقع** — والهدفُ يبقى عدّاداً
		// والمكافأةُ بيد المالك كما كانت.
		return 0
	}

	done, err := s.doneThisMonth(ctx, userID, role)
	if err != nil || done < target {
		return 0
	}

	// **ثمّ يُكتب — والقاعدةُ تردّ الثاني.**
	period := PeriodOf(time.Now())
	if err := s.grantTarget(ctx, userID, reward, period, target); err != nil {
		// **والتصادمُ ليس خطأً** — هو الضمانةُ تعمل.
		if isDuplicate(err) {
			return 0
		}
		s.logf("مكافأةُ الهدف تعذّرت", "user", userID, "error", err)
		return 0
	}
	return reward
}

// doneThisMonth ما أنجزه في شهره — **بالمعنى الذي يخصّ دورَه.**
func (s *Service) doneThisMonth(ctx context.Context, userID, role string) (int64, error) {
	q := `SELECT count(*) FROM orders o
	      WHERE o.driver_id = $1 AND o.status = 'delivered'
	        AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart
	if role == "sales" {
		q = `SELECT count(*) FROM orders o
		     JOIN merchants mm ON mm.id = o.merchant_id
		     WHERE mm.sales_rep_user_id = $1 AND o.status = 'delivered'
		       AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart
	}
	var n int64
	err := s.db.QueryRow(ctx, q, userID).Scan(&n)
	return n, err
}

// grantTarget القيدُ نفسُه — **بالمعاملة، والخزينةُ الطرفُ المقابل.**
//
// **ولا يُعاد بناءُ ما في `Grant`**: هي تأخذ فاعلاً بشريّاً، **وهذه بلا فاعل**
// — ومن كتب `actor` وهميّاً جعل السجلَّ يقول إنّ إنساناً قرّر.
func (s *Service) grantTarget(ctx context.Context, userID string, reward int64,
	period string, target int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	reason := "مكافأةُ بلوغ هدف الشهر (" + period + ") — " + itoa(target) + " طلباً"

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
	return tx.Commit(ctx)
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
