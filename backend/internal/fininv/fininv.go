// Package fininv محرّكُ الثوابت الماليّة — **مصدرُ حقيقةٍ واحدٌ للمال.**
//
// **ولا إطارَ ماليٌّ ثانٍ**: `cmd/moneycheck` كان يحمل ثلاثةَ عشرَ فحصاً
// مكتوبةً فيه، **فنُقلت كما هي إلى هنا** — وصار الأمرُ قشرةً تنادي هذه
// الحزمة، **واختباراتُ `internal/qa` تنادي الحزمةَ نفسَها.**
//
// **وهذا هو المقصود**: أمرٌ واحدٌ ودالّةٌ واحدةٌ تفحصان الحقيقةَ الماليّةَ
// لحالةٍ اختباريّةٍ أو لقاعدةِ تشغيل، **بالسؤال نفسِه لا بسؤالين يفترقان.**
//
// # لماذا استعلامٌ يردّ صفوفاً = خرق
//
// **لأنّ الجوابَ يحمل دليلَه.** فحصٌ يردّ `false` يقول «فيه خلل» ولا يقول
// أين، **وصفٌّ يردّ رقمَ الطلب والمبلغَ يضع الإصبعَ على الموضع.**
package fininv

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier ما يكفي للفحص — **بركةٌ أو معاملة، فيُفحَص داخلَ معاملةٍ لم تُودَع.**
type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Family عائلةُ الثابت — **الاثنتا عشرةَ التي حسمها المالك.**
type Family string

const (
	FI01 Family = "FI-01" // سلامةُ مرجعِ الدفتر
	FI02 Family = "FI-02" // مطابقةُ رصيدِ المحفظة
	FI03 Family = "FI-03" // لا خلقَ للمال
	FI04 Family = "FI-04" // لا ضياعَ للمال
	FI05 Family = "FI-05" // أثرٌ ماليٌّ مرّةً واحدةً بالضبط
	FI06 Family = "FI-06" // حفظُ اقتصاد الطلب
	FI07 Family = "FI-07" // اللقطةُ الاقتصاديّة
	FI08 Family = "FI-08" // سلامةُ الاسترداد
	FI09 Family = "FI-09" // عكسُ عمولة المندوب
	FI10 Family = "FI-10" // مطابقةُ نقدِ السائق
	FI11 Family = "FI-11" // حفظُ السحب
	FI12 Family = "FI-12" // الخزينةُ والمصروف
	FI13 Family = "FI-13" // تتبّعُ الالتزامات — `XG-31`
)

// Status حالُ إثباتِ الثابت — **ولا `PASS` وهميّ.**
type Status string

const (
	// ProvableNow يُثبَت بالاستعلام على قاعدةٍ حقيقيّةٍ الآن.
	ProvableNow Status = "PROVABLE_NOW"
	// DeferredP5 يحتاج تزامناً حقيقيّاً لإثباته.
	DeferredP5 Status = "DEFERRED_TO_P5"
	// DeferredP6 يحتاج إسقاطَ خطوةٍ بعينها لإثبات الفشل الجزئيّ.
	DeferredP6 Status = "DEFERRED_TO_P6"
	// NotImplemented العقدُ مقرَّرٌ والشيفرةُ لا تنفّذه بعد.
	NotImplemented Status = "NOT_IMPLEMENTED"
)

// Check فحصٌ واحد — **واستعلامٌ يردّ صفّاً واحداً فأكثرَ يعني خرقاً.**
type Check struct {
	// ID معرّفٌ مستقرٌّ — `FI-06.a`. **يُكتب في التقارير ولا يتبدّل.**
	ID string
	// Family العائلة.
	Family Family
	// Name اسمٌ يُقرأ في الطرفيّة.
	Name string
	// Why ما الذي ينكسر في الواقع إن سقط — **لا إعادةُ صياغةِ الاسم.**
	Why string
	// SQL الاستعلام. **صفرُ صفوفٍ = سليم.**
	SQL string
	// Status هل يُثبَت الآن أم يُنتظَر.
	Status Status
	// Flows التدفّقاتُ التي يحرسها — `F-14`.
	Flows []string
	// Registers ما يمسّه من السجلّات المجمَّدة — `D5` · `XG-10`.
	Registers []string
	// Kinds أنواعُ القيد التي يغطّيها.
	Kinds []string
	// Ops هل يُشغَّل في `moneycheck` على قاعدةِ تشغيل.
	//
	// **وبعضُ الفحوص لا معنى لها هناك**: فحصٌ يفترض خزينةً في قاعدةٍ
	// اختباريّةٍ بلا خزينةٍ يُنذر كذباً — **وأداةٌ تُنذر كذباً تُهمَل ثمّ
	// لا تُقرأ يومَ تصدق.**
	Ops bool
}

// Violation خرقٌ واحدٌ بصفوفه.
type Violation struct {
	Check Check
	Cols  []string
	Rows  [][]any
}

func (v Violation) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s — %d صفّاً", v.Check.ID, v.Check.Name, len(v.Rows))
	for i, r := range v.Rows {
		if i >= 5 {
			fmt.Fprintf(&b, "\n  ... و%d غيرُها", len(v.Rows)-5)
			break
		}
		fmt.Fprintf(&b, "\n  %v", r)
	}
	return b.String()
}

// Run يفحص الحقيقةَ الماليّة — **الدالّةُ الواحدةُ التي طُلبت.**
//
// بلا معرّفاتٍ: كلُّ ما هو `PROVABLE_NOW`.
// بمعرّفاتٍ: ما طُلب وحدَه (`FI-06` تعني كلَّ فحوص العائلة).
func Run(ctx context.Context, q Querier, ids ...string) ([]Violation, error) {
	var out []Violation
	for _, c := range Select(ids...) {
		rows, err := q.Query(ctx, c.SQL)
		if err != nil {
			return out, fmt.Errorf("%s: %w", c.ID, err)
		}
		v := Violation{Check: c}
		for _, fd := range rows.FieldDescriptions() {
			v.Cols = append(v.Cols, fd.Name)
		}
		for rows.Next() {
			vals, verr := rows.Values()
			if verr != nil {
				rows.Close()
				return out, fmt.Errorf("%s: %w", c.ID, verr)
			}
			v.Rows = append(v.Rows, vals)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, fmt.Errorf("%s: %w", c.ID, err)
		}
		if len(v.Rows) > 0 {
			out = append(out, v)
		}
	}
	return out, nil
}

// Select ينتقي الفحوصَ — **وما ليس `PROVABLE_NOW` لا يُشغَّل بلا طلبٍ صريح.**
func Select(ids ...string) []Check {
	var out []Check
	for _, c := range All {
		if len(ids) == 0 {
			if c.Status == ProvableNow {
				out = append(out, c)
			}
			continue
		}
		for _, want := range ids {
			if c.ID == want || string(c.Family) == want {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// Families عوائلُ الثوابت مرتَّبةً.
func Families() []Family {
	seen := map[Family]bool{}
	var out []Family
	for _, c := range All {
		if !seen[c.Family] {
			seen[c.Family] = true
			out = append(out, c.Family)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
