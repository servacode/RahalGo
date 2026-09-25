package platform

// ══════════════════════════════════════════════════════════════════════
// **أوقاتُ مناطق التوصيل — الآلةُ عينُها لا آلةٌ ثانية** (`ZH`)
// ══════════════════════════════════════════════════════════════════════
//
// # ولمَ هنا لا في `orders`
//
// **وجدولُ الأسبوع مكتوبٌ مرّةً** (`hours.go`): **الحدُّ داخلٌ وخارجٌ،
// وعبورُ منتصف الليل، وذيلُ أمس، وتداخلٌ على مدارِ الأسبوع، وموعدٌ
// قادمٌ لا يقع في الماضي.** **وكلُّ ذلك قِيس بواحدٍ وعشرين فحصاً.**
//
// **ونسخةٌ ثانيةٌ منه في `orders` تفترق يوماً** — **فيقول جدولُ المنصّة
// إنّ ١٧:٠٠ مغلقٌ ويقول جدولُ المنطقة إنّه مفتوح**، **والفرقُ سطرٌ
// نُسي عند إصلاحٍ في أحدهما.**
//
// **فالمنطقةُ تستعمل `Schedule` عينَها** — **ولا يتبدّل إلّا من أين
// تُقرأ الصفوف.**
//
// # وما يفترق بحقّ
//
// **وسريانُ جدول المنصّة مفتاحُ إعدادٍ واحد** — **وسريانُ جدول المنطقة
// عمودٌ في المنطقة نفسِها**: **منطقةٌ تعمل ليلَ نهارٍ وأخرى تُغلق
// السادسةَ.** **ولذلك رايةٌ لكلّ منطقة.**

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ZoneState **ما تقوله منطقةٌ عن وقتها** — **ولا شأنَ لها بجغرافيتها.**
//
// **والجغرافيا حُسمت قبلها** (`ZoneAt`) — **وهذه لا تجعل منطقةً مُطفأةً
// صالحةً ولا نقطةً خارجَ التغطية داخلَها.**
type ZoneState struct {
	// Open **أيُوصَّل إلى هذه المنطقة الآن؟**
	//
	// **وغيرُ السارية مفتوحةٌ دائماً** — وهو حالُ كلّ منطقةٍ قائمةٍ
	// اليوم.
	Open bool `json:"open"`
	// Enforced **أيسري جدولُها؟**
	Enforced bool `json:"enforced"`
	// NextOpenAt **متى يعود التوصيلُ إليها** — و`nil` إن لم يُعرَف.
	NextOpenAt *time.Time `json:"next_open_at,omitempty"`
	// NextCloseAt **متى يُغلَق التوصيلُ المفتوحُ الآن** — نهايةُ فترتها
	// الجارية. **يُملأ حين تكون مفتوحةً بجدولٍ سارٍ**، و`nil` وإلّا.
	NextCloseAt *time.Time `json:"next_close_at,omitempty"`
}

// ZoneSchedule **جدولُ منطقةٍ ورايةُ سريانه.**
//
// **ومنطقةٌ بلا معرّفٍ ليست منطقةً** — **وهي حالُ «لا خريطةَ رُسمت
// بعد»** (`ZoneAt` تردّ معرّفاً فارغاً): **فلا جدولَ لها ولا سريان.**
func (s *Service) ZoneSchedule(ctx context.Context, q dbtx.Querier, zoneID string) (Schedule, bool, error) {
	if zoneID == "" {
		return nil, false, nil
	}
	var enforced bool
	if err := q.QueryRow(ctx,
		`SELECT hours_enforced FROM delivery_zones WHERE id = $1::uuid`,
		zoneID).Scan(&enforced); err != nil {
		return nil, false, err
	}
	if !enforced {
		// **ولا تُقرأ صفوفٌ لا تحكم** — **رحلةٌ إلى القاعدة بلا أثر.**
		return nil, false, nil
	}
	rows, err := q.Query(ctx, `
		SELECT day_of_week,
		       EXTRACT(hour FROM starts_at)::int * 60 + EXTRACT(minute FROM starts_at)::int,
		       EXTRACT(hour FROM ends_at)::int   * 60 + EXTRACT(minute FROM ends_at)::int
		FROM delivery_zone_hours
		WHERE zone_id = $1::uuid
		ORDER BY day_of_week, starts_at`, zoneID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := Schedule{}
	for rows.Next() {
		var w Window
		if err := rows.Scan(&w.Day, &w.Start, &w.End); err != nil {
			return nil, false, err
		}
		out = append(out, w)
	}
	return out, true, rows.Err()
}

// ZoneWindows **فتراتُ منطقةٍ كما هي محفوظة — سرت أو لم تسرِ.**
//
// **وتقرؤها اللوحةُ لا القرار** — **فمن كتب جدولاً ثمّ أطفأ سريانَه
// يجب أن يجد ما كتبه حين يعود، لا صفحةً بيضاء.**
func (s *Service) ZoneWindows(ctx context.Context, q dbtx.Querier, zoneID string) (Schedule, error) {
	rows, err := q.Query(ctx, `
		SELECT day_of_week,
		       EXTRACT(hour FROM starts_at)::int * 60 + EXTRACT(minute FROM starts_at)::int,
		       EXTRACT(hour FROM ends_at)::int   * 60 + EXTRACT(minute FROM ends_at)::int
		FROM delivery_zone_hours
		WHERE zone_id = $1::uuid
		ORDER BY day_of_week, starts_at`, zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := Schedule{}
	for rows.Next() {
		var w Window
		if err := rows.Scan(&w.Day, &w.Start, &w.End); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ZoneStateOf **حالُ منطقةٍ في هذه اللحظة.**
func (s *Service) ZoneStateOf(ctx context.Context, q dbtx.Querier, zoneID string) (ZoneState, error) {
	sch, enforced, err := s.ZoneSchedule(ctx, q, zoneID)
	if err != nil {
		return ZoneState{}, err
	}
	return DecideZone(s.now(), enforced, sch), nil
}

// DecideZone **القرارُ خالصاً — لا قاعدةَ ولا ساعةَ نظام.**
func DecideZone(now time.Time, enforced bool, sch Schedule) ZoneState {
	if !enforced {
		// **ومنطقةٌ لا جدولَ ساريَ لها مفتوحةٌ دائماً** — **ولا موعدَ
		// عودةٍ لمن لم يغب.**
		return ZoneState{Open: true}
	}
	st := ZoneState{Enforced: true, Open: sch.OpenAt(now)}
	if st.Open {
		// **ومفتوحةٌ يُعرَف متى تُغلَق** — نهايةُ فترتها الجارية.
		st.NextCloseAt = sch.NextCloseAt(now)
	} else {
		st.NextOpenAt = sch.NextOpenAt(now)
	}
	return st
}

// SetZoneSchedule **يستبدل جدولَ منطقةٍ كلَّه، ويضبط سريانَه.**
//
// **وكلُّه في معاملةٍ واحدة** — **وجدولٌ يُعدَّل صفّاً صفّاً يمرّ
// بحالاتٍ متداخلةٍ وسطَ التعديل**، **ونداءُ زبونٍ يقع في تلك اللحظة
// يُجاب بجدولٍ نصفِ مكتوب.**
func (s *Service) SetZoneSchedule(ctx context.Context, zoneID string,
	enforced bool, ws []Window) (Schedule, error) {
	clean, err := Validate(ws)
	if err != nil {
		return nil, ErrBadSchedule
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **ومنطقةٌ لا وجودَ لها تُردّ** — **ولا يُكتب جدولٌ لمعرّفٍ مخترَع.**
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT true FROM delivery_zones WHERE id = $1::uuid`, zoneID).Scan(&exists); err != nil {
		return nil, ErrZoneNotFound
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM delivery_zone_hours WHERE zone_id = $1::uuid`, zoneID); err != nil {
		return nil, err
	}
	for _, w := range clean {
		if _, err := tx.Exec(ctx, `
			INSERT INTO delivery_zone_hours (zone_id, day_of_week, starts_at, ends_at)
			VALUES ($1::uuid, $2, make_time($3, $4, 0), make_time($5, $6, 0))`,
			zoneID, w.Day, int(w.Start)/60, int(w.Start)%60,
			int(w.End)/60, int(w.End)%60); err != nil {
			return nil, ErrBadSchedule
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE delivery_zones SET hours_enforced = $2 WHERE id = $1::uuid`,
		zoneID, enforced); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return clean, nil
}

// ══════════════════════════════════════════════════════════════════════
// **وموعدُ الطلب تقاطعُ المنصّة والمنطقة** (`ZH-28`)
// ══════════════════════════════════════════════════════════════════════
//
// **ومن قال «يعود التوصيل الثالثة» والمنصّةُ لا تستقبل حتّى الخامسة
// كذب** — **يعود الزبونُ في الثالثة فيجد البابَ مغلقاً، ولا يعود
// ثالثةً.**
//
// # ولمَ البحثُ في المبادئ لا في كلّ دقيقة
//
// **ومجموعُ «مفتوحٌ» يتبدّل عند مبادئِ الفترات وحدَها** — **وبين
// مبدأين لا يتغيّر شيء.** **فتُجمَع مبادئُ الجدولين ومنتهى الإيقاف
// المؤقّت، وتُسأل كلُّ لحظةٍ منها مرّةً.**
//
// **ومسحُ كلّ دقيقةٍ في أسبوعٍ عشرةُ آلافِ سؤال** — **وهذا بضعُ
// عشرات.**

// NextOrderingAt **أوّلُ لحظةٍ يُقبَل فيها طلبٌ إلى هذه المنطقة.**
//
// **وتجمع الطبقات الثلاث**: الإيقافَ المؤقّتَ ودوامَ المنصّة وجدولَ
// المنطقة. **ولا تعرف وضعَ الإطلاق** — **وذاك بابٌ لا موعدَ لفتحه،
// ولا يُقال عنه «نعود الرابعة».**
//
// **وفارغٌ يعني «لا موعدَ في أسبوع»** — **ولا يُخترَع.**
func (s *Service) NextOrderingAt(ctx context.Context, q dbtx.Querier, zoneID string) (*time.Time, error) {
	pSch, err := s.Schedule(ctx, q)
	if err != nil {
		return nil, err
	}
	c, err := s.Closure(ctx, q)
	if err != nil {
		return nil, err
	}
	zSch, zEnforced, err := s.ZoneSchedule(ctx, q, zoneID)
	if err != nil {
		return nil, err
	}
	return NextBothOpen(s.now(), s.settings.GetBool(ctx, EnforcedKey), pSch, c, zEnforced, zSch), nil
}

// Gate **قيدٌ زمنيٌّ واحد** — جدولٌ ورايةُ سريانه.
//
// **وغيرُ السارية لا تقيّد شيئاً** — **ولا تُسأل عن فتراتها أصلاً.**
type Gate struct {
	Enforced bool
	Sch      Schedule
}

// openAt **أيسمح هذا القيدُ بهذه اللحظة؟**
func (g Gate) openAt(t time.Time) bool { return !g.Enforced || g.Sch.OpenAt(t) }

// NextAllOpen **أوّلُ لحظةٍ تجتمع فيها كلُّ القيود الزمنيّة.**
//
// ══════════════════════════════════════════════════════════════════════
// **وليست أصغرَ المواعيد — بل أوّلَ ما تتقاطع** (`AV-15`)
// ══════════════════════════════════════════════════════════════════════
//
// **ومن أخذ أصغرَ موعدٍ في كلّ جدولٍ على حدةٍ أجاب بموعدٍ لا يُطلَب
// فيه شيء**: **المنصّةُ تفتح التاسعةَ والمنطقةُ العاشرةَ والمتجرُ
// الحاديةَ عشرةَ والنصف** — **والجوابُ الحاديةَ عشرةَ والنصف لا
// التاسعة.**
//
// **والبحثُ في المبادئ لا في كلّ دقيقة** — **ومجموعُ «مفتوح» يتبدّل
// عند مبادئِ الفترات ومنتهى الإيقاف وحدَها.** **ومسحُ أسبوعٍ بالدقيقة
// عشرةُ آلافِ سؤال، وهذا بضعُ عشرات.**
//
// **والأفقُ ثمانيةُ أيّامٍ محدودة** — **ولا دورانَ بلا نهاية.**
// **وفارغٌ يعني «لا تقاطعَ يُعرَف»** — **ولا يُخترَع موعد.**
func NextAllOpen(now time.Time, c Closure, gates ...Gate) *time.Time {
	all := func(t time.Time) bool {
		if c.ActiveAt(t) {
			return false
		}
		for _, g := range gates {
			if !g.openAt(t) {
				return false
			}
		}
		return true
	}
	if all(now) {
		t := now
		return &t
	}

	cands := []time.Time{}
	if c.EndsAt != nil && c.EndsAt.After(now) {
		cands = append(cands, *c.EndsAt)
	}
	for _, g := range gates {
		if g.Enforced {
			cands = append(cands, starts(now, g.Sch)...)
		}
	}

	var best *time.Time
	for _, t := range cands {
		if !t.After(now) || !all(t) {
			continue
		}
		if best == nil || t.Before(*best) {
			v := t
			best = &v
		}
	}
	return best
}

// NextBothOpen **تقاطعُ المنصّة والمنطقة** — وجهٌ ضيّقٌ لـ`NextAllOpen`.
//
// **ولا حسبةَ ثانيةٌ تحته** — **واثنتان تفترقان يوماً.**
func NextBothOpen(now time.Time, pEnforced bool, pSch Schedule, c Closure,
	zEnforced bool, zSch Schedule) *time.Time {
	return NextAllOpen(now, c,
		Gate{Enforced: pEnforced, Sch: pSch},
		Gate{Enforced: zEnforced, Sch: zSch})
}

// Snapshot **القيودُ الزمنيّةُ كلُّها مقروءةً مرّةً واحدة.**
//
// **ويقرؤها النموذجُ القارئ ليشرح** — **والبوّابةُ تقرأ ما يخصّها
// وحدَه.** **ومصدرُ الصفوف واحدٌ في الحالين، فلا يفترق ما يُشرَح عمّا
// يُمنَع.**
type Snapshot struct {
	Closure  Closure
	Platform Gate
	Zone     Gate
	// Now **لحظةُ الخادم** — **وبها يُحكَم لا بساعة جهاز.**
	Now time.Time
}

// Snapshot يقرأ القيودَ الزمنيّةَ للمنصّة ولمنطقةٍ بعينها.
func (s *Service) Snapshot(ctx context.Context, q dbtx.Querier, zoneID string) (Snapshot, error) {
	pSch, err := s.Schedule(ctx, q)
	if err != nil {
		return Snapshot{}, err
	}
	c, err := s.Closure(ctx, q)
	if err != nil {
		return Snapshot{}, err
	}
	zSch, zEnforced, err := s.ZoneSchedule(ctx, q, zoneID)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Closure:  c,
		Platform: Gate{Enforced: s.settings.GetBool(ctx, EnforcedKey), Sch: pSch},
		Zone:     Gate{Enforced: zEnforced, Sch: zSch},
		Now:      s.now(),
	}, nil
}

// starts **مبادئُ فترات جدولٍ في الأيّام الثمانية القادمة.**
//
// **وثمانيةٌ لا سبعة** — **فمن سُئل يومَ الأحد مساءً عن فترةٍ تبدأ
// الأحدَ صباحاً وجدها في الأسبوع القادم.**
func starts(now time.Time, sch Schedule) []time.Time {
	loc := Location()
	t := now.In(loc)
	out := make([]time.Time, 0, len(sch)*2)
	for d := 0; d <= 8; d++ {
		day := t.AddDate(0, 0, d)
		dow := int(day.Weekday())
		for _, w := range sch {
			if w.Day != dow {
				continue
			}
			out = append(out, time.Date(day.Year(), day.Month(), day.Day(),
				int(w.Start)/60, int(w.Start)%60, 0, 0, loc))
		}
	}
	return out
}

// ZoneNextCloseAt **متى تُغلَق منطقةٌ مفتوحةٌ الآن** — و`nil` حين لا جدولَ
// سارٍ (مفتوحةٌ بلا حدّ) أو حين تكون مغلقة. **جوابٌ أوّليٌّ للطلبات** فلا
// تستورد `orders` هذه الحزمة (نظيرُ `ZoneOpen`).
func (s *Service) ZoneNextCloseAt(ctx context.Context, q dbtx.Querier, zoneID string) (*time.Time, error) {
	st, err := s.ZoneStateOf(ctx, q, zoneID)
	if err != nil {
		return nil, err
	}
	return st.NextCloseAt, nil
}

// ZoneOpen **أيُوصَّل إلى هذه المنطقة الآن — ومتى يُقبَل طلبٌ إليها.**
//
// **وهو ما تناديه بوّابةُ قبول الطلبات** (`orders.ZoneHours`) —
// **وواجهةٌ بأنواعٍ أوّليّةٍ فلا تستورد `orders` هذه الحزمة.**
//
// **ولا يُحسَب الموعدُ إلّا عند المنع** — **فمسارُ القبول لا يدفع ثمنَ
// سؤالٍ لا يُعرَض.**
//
// **والموعدُ تقاطعُ المنصّةِ والمنطقة** — **لا موعدُ المنطقة وحدَها.**
func (s *Service) ZoneOpen(ctx context.Context, q dbtx.Querier, zoneID string) (bool, *time.Time, error) {
	st, err := s.ZoneStateOf(ctx, q, zoneID)
	if err != nil {
		return false, nil, err
	}
	if st.Open {
		return true, nil, nil
	}
	next, err := s.NextOrderingAt(ctx, q, zoneID)
	if err != nil {
		// **وتعذُّرُ حساب الموعد لا يفتح المنطقةَ** — **والمنعُ قائمٌ
		// بلا موعدٍ يُعرَض.**
		return false, nil, nil
	}
	return false, next, nil
}
