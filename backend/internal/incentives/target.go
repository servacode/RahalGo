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

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/settings"
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
	return levelsFrom(ctx, s.settings, role)
}

// levelsOn **المراحلُ مقروءةً بالمعاملة الممسوكة** — `XG-48`.
//
// **كانت تُقرأ من المَسبَح ومعاملةُ الصرف ممسكةٌ بوصلة**، فبمَسبَحٍ من وصلةٍ
// واحدةٍ جمد تسليمُ الطلب (الصرفُ بعده يفتح معاملته ثمّ يطلب وصلةً ثانيةً
// للإعدادات). **والمخزنُ الحقيقيُّ يُبدَّل منفّذُه**؛ وغيرُه (اختبارات) يُقرأ كما هو.
func (s *Service) levelsOn(ctx context.Context, q dbtx.Querier, role string) []Level {
	if st, ok := s.settings.(*settings.Store); ok && st != nil {
		return levelsFrom(ctx, st.On(q), role)
	}
	return levelsFrom(ctx, s.settings, role)
}

func levelsFrom(ctx context.Context, st Settings, role string) []Level {
	p := prefixOf(role)
	if p == "" {
		return nil
	}
	return []Level{
		{N: 1,
			Target: st.GetInt(ctx, p+"monthly_target"),
			Reward: st.GetInt(ctx, p+"target_reward")},
		{N: 2,
			Target: st.GetInt(ctx, p+"target_2"),
			Reward: st.GetInt(ctx, p+"reward_2")},
		{N: 3,
			Target: st.GetInt(ctx, p+"target_3"),
			Reward: st.GetInt(ctx, p+"reward_3")},
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
	levels := s.levelsOn(ctx, tx, role)
	if len(levels) == 0 {
		return 0, nil
	}
	state, err := userState(ctx, tx, userID)
	if err != nil || state == stateGone {
		return 0, err
	}
	// **ويُعَدُّ بالمعاملة نفسِها** — `XG-32`: **وإلّا لم يُرَ متجرُها.**
	done, err := s.doneInMonthOn(ctx, tx, userID, role, month)
	if err != nil {
		return 0, err
	}
	// ══════════════════════════════════════════════════════════════════
	// **والموقوفُ لا يُصرف له الآن — ولا يخسر ما بلغه** (قرارُ المالك ٢٠٢٦-١٠-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// «يظل إذا حقق الشروط»: **كان من بلغ مرحلةً وهو موقوفٌ يخسرها للأبد** — العدّادُ
	// لا يتحرّك مرّةً ثانيةً عند العتبة نفسِها. **والآن تُحفظ** (`incentive_pending_grants`)
	// **وتُصرف يومَ يُعاد تفعيلُه** (`PayPending`)، وجولةٌ دوريّةٌ تلتقط ما فات.
	if state == stateHeld {
		return 0, s.holdIfDue(ctx, tx, userID, role, month, levels, done)
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

// حالُ صاحب المكافأة ساعةَ الصرف.
const (
	// stateActive فعّالٌ — يُصرف له.
	stateActive = iota
	// stateHeld موقوفٌ أو غيرُ فعّال — **تُحفظ مكافأتُه حتّى يعود.**
	stateHeld
	// stateGone محذوفٌ أو لا وجودَ له — لا صرفَ ولا حفظ.
	stateGone
)

// userState **أفعّالٌ هو، أم موقوفٌ ينتظر، أم ذهب؟**
func userState(ctx context.Context, q dbtx.Querier, userID string) (int, error) {
	var status string
	var deleted bool
	err := q.QueryRow(ctx, `
		SELECT status, deleted_at IS NOT NULL FROM users WHERE id = $1::uuid`,
		userID).Scan(&status, &deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return stateGone, nil
	}
	if err != nil {
		return stateGone, err
	}
	switch {
	case deleted:
		return stateGone, nil
	case status != "active":
		return stateHeld, nil
	}
	return stateActive, nil
}

// holdIfDue **يحفظ مكافأةً بلغها موقوف** — صفٌّ مفتوحٌ واحدٌ لشهره ودوره.
//
// **ولا يُحفظ ما قُبض**: مرحلةٌ دُفعت قبل الإيقاف لا تفتح صفّاً.
func (s *Service) holdIfDue(ctx context.Context, q dbtx.Querier, userID, role, month string,
	levels []Level, done int64) error {
	for _, l := range levels {
		if l.Target <= 0 || l.Reward <= 0 || done < l.Target {
			continue
		}
		var paid bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM incentives
			                WHERE user_id = $1::uuid AND for_target AND period = $2)`,
			userID, month+"#"+itoa(int64(l.N))).Scan(&paid); err != nil {
			return err
		}
		if paid {
			continue
		}
		_, err := q.Exec(ctx, `
			INSERT INTO incentive_pending_grants (user_id, role, month)
			VALUES ($1::uuid, $2, $3)
			ON CONFLICT (user_id, role, month) WHERE paid_at IS NULL DO NOTHING`,
			userID, role, month)
		return err
	}
	return nil
}

// ══════════════════════════════════════════════════════════════════════
// **إشعارُ المرحلة بلفظها** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كان «أنجزت هدف الشهر» يُرسَل عند كلّ مرحلة** — فمن بلغ الأولى من ثلاثٍ قرأ أنّه
// أتمّ الشهر. **والآن «أنجزت هدف الشهر» عند آخر مرحلةٍ مضبوطةٍ وحدَها**، وما قبلها
// «وصلت للمرحلة {n} — نالتك مكافأتها».

// StageTitle **عنوانُ إشعار المرحلة** — `final` آخرُ مرحلةٍ مضبوطة.
func StageTitle(stage int, final bool) string {
	if final || stage <= 0 {
		return "أنجزت هدف الشهر — نالتك مكافأته"
	}
	return "وصلت للمرحلة " + itoa(int64(stage)) + " — نالتك مكافأتها"
}

// StageNotice **عنوانُ ما صُرف له في شهرٍ** — أعلى مرحلةٍ قُبضت، أهي الأخيرة؟
//
// **ويُقرأ بعد الصرف لا قبله**: مرحلتان تُبلغان معاً (هدفٌ خُفّض) تُقالان بأعلاهما.
func (s *Service) StageNotice(ctx context.Context, userID, role, month string) string {
	last := 0
	for _, l := range s.LevelsOf(ctx, role) {
		if l.N > last {
			last = l.N
		}
	}
	var top int
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(max(NULLIF(split_part(period, '#', 2), '')::int), 0)
		  FROM incentives
		 WHERE user_id = $1::uuid AND for_target AND split_part(period, '#', 1) = $2
		   AND (target_role = $3 OR target_role IS NULL)`, userID, month, role).Scan(&top)
	return StageTitle(top, last > 0 && top >= last)
}

// TargetTitle **عنوانُ إشعار الشهر الجاري** — يقرؤه محرّكُ الطلبات بعد التسليم.
func (s *Service) TargetTitle(ctx context.Context, userID, role string) string {
	return s.StageNotice(ctx, userID, role, PeriodOf(time.Now()))
}

// RoleOfTargetKey **دورُ مفتاح هدفٍ أو مرحلة** — وفراغٌ لغيره.
//
// **وتعديلُه يصرف فوراً لمن صار مؤهَّلاً** (قرارُ المالك ٢٠٢٦-١٠-١٥) — `SweepMonth`.
func RoleOfTargetKey(key string) string {
	for _, role := range []string{"driver", "sales"} {
		p := prefixOf(role)
		for _, k := range []string{"monthly_target", "target_reward", "target_2", "reward_2",
			"target_3", "reward_3"} {
			if key == p+k {
				return role
			}
		}
	}
	return ""
}

// SweepMonth **يصرف للشهر الجاري لكلّ من في الدور بلغ مرحلةً لم يقبضها.**
//
// **خُفّض الهدفُ في منتصف الشهر فصار عشرةٌ مؤهَّلين** — كانوا ينتظرون تسليماً أو
// متجراً جديداً ليتحرّك العدّاد، **ومن وقف لا يقبض أبداً.** والموقوفُ يُحفظ له.
func (s *Service) SweepMonth(ctx context.Context, role string) []Paid {
	if prefixOf(role) == "" {
		return nil
	}
	month := PeriodOf(time.Now())
	// **ومن لم يبلغ أدنى مرحلةٍ مدفوعةٍ لا يُسأل عنه** — العدُّ في استعلامٍ واحد،
	// والصرفُ لمن تجاوز العتبةَ وحدَه.
	var least int64
	for _, l := range s.levelsFor(ctx, role) {
		if l.Target > 0 && l.Reward > 0 && (least == 0 || l.Target < least) {
			least = l.Target
		}
	}
	if least == 0 {
		return nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT u.id::text FROM users u
		  JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = $1
		 WHERE u.deleted_at IS NULL
		   AND `+doneSQL(role, "u.id", "$2::text")+` >= $3`, role, month, least)
	if err != nil {
		s.logf("جولةُ الأهداف تعذّرت", "role", role, "error", err)
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	var out []Paid
	for _, id := range ids {
		paid, err := s.grantForMonth(ctx, id, role, month)
		if err != nil {
			s.RecordFailure(ctx, id, role, month, err)
			continue
		}
		if paid > 0 {
			out = append(out, Paid{UserID: id, Role: role, Amount: paid, Month: month})
		}
	}
	return out
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
		// **وبشهر اكتمال شروطه لا بشهر تسجيله** (قرارُ المالك ٢٠٢٦-١٠-١٣):
		// ساعاتٌ وموقعٌ وأصنافٌ بصورها — `rep_qualified_at`.
		// **ومن المندوبِ الحاليّ لا ممّن فتحه** (قرارُ المالك ٢٠٢٦-١٠-٢٠: «كلشي ننقله الى المندوب الجديد لازم ينحسب») — يُلغي «لمن فتحه للأبد» (٢٠٢٦-١٠-٠٤)، و`opened_by_rep_id` سجلٌّ لا عدّاد.
		return `(SELECT count(*) FROM merchants mm
		          WHERE mm.sales_rep_user_id = ` + who + `
		            AND mm.rep_qualified_at >= ` + monthFromSQL(month) + `
		            AND mm.rep_qualified_at <  ` + monthToSQL(month) + `)`
	}
	return `(SELECT count(*) FROM orders o
	          WHERE o.driver_id = ` + who + ` AND o.status = 'delivered'
	            AND o.delivered_at >= ` + monthFromSQL(month) + `
	            AND o.delivered_at <  ` + monthToSQL(month) + `)`
}

// pendingSQL **متاجرُ فتحها المندوبُ ولم تكتمل شروطُها** — فاعلةٌ؛ وللسائق صفر.
func pendingSQL(role, who string) string {
	if role != "sales" {
		return `0`
	}
	return `(SELECT count(*)::int FROM merchants mm
	          WHERE mm.sales_rep_user_id = ` + who + `
	            AND mm.rep_qualified_at IS NULL AND mm.status = 'active')`
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
	var incID string
	err := tx.QueryRow(ctx, `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, period, created_by,
		                        target_count, target_role, role)
		VALUES ($1, $2, $3, $4, true, $5, NULL, $6, $7, $7)
		ON CONFLICT DO NOTHING
		RETURNING id::text`,
		userID, KindReward, reward, reason, period, l.Target, role).Scan(&incID)
	if errors.Is(err, pgx.ErrNoRows) {
		// **كوفئ عن هذه المرحلة سابقاً** — ولا تكرار.
		return errAlreadyGranted
	}
	if err != nil {
		return err
	}
	// **والمرجعُ صفُّ المكافأة** (قرارُ المالك ٢٠٢٦-١٠-١٥) — فصفحةُ الأرباح تعرف
	// «مكافأةَ هدف» من «مكافأةٍ يدويّة» من «دعوة» بوصلٍ لا بتخمينٍ من فراغ المرجع.
	if _, err := s.wallet.ApplyTx(ctx, tx, userID, reward, KindReward, incID, reason, nil); err != nil {
		return err
	}
	// **والخزينةُ تدفع** — دفترٌ يأخذ من طرفٍ ولا يعطي آخرَ لا يتوازن.
	if tid := s.treasury(ctx); tid != "" {
		if _, err := s.wallet.ApplyTx(ctx, tx, tid, -reward, KindReward, incID,
			"مكافأةُ هدفٍ صُرفت", nil); err != nil {
			return err
		}
	}
	// **ودينُه يُسدّ منها أوّلاً** (قرارُ المالك ٢٠٢٦-١٠-١٥) — بمرجع صفّ المكافأة.
	return s.payDebtFromReward(ctx, tx, userID, reward, incID, nil)
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
