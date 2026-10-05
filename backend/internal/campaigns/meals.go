package campaigns

// ══════════════════════════════════════════════════════════════════════
// **إشعاراتُ الوجبات — فطورٌ وغداءٌ وعشاءٌ كلَّ يوم** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «إشعارات تلقائية: شو بدك تتغدّى، شو بدك تفطر، شو بدك تتعشّى — رسائل
//  تذكير وتحفيزية للزبائن.»
//
// # ولا محرّكَ ثانٍ
//
// **كلُّ وجبةٍ حملةٌ عاديّة** تمرّ بـ`Create` ثمّ `Send` — **فتحترم ساعةَ
// الهدوء وسقفَ اليوم لكلّ حساب** كما تحترمهما حملةُ المالك اليدويّة.
//
// # ومرّةً في اليوم لا مرّتين
//
// **ومفتاحُ عدمِ التكرار** `meal:<الوجبة>:<اليوم>` — **وخادمٌ أُعيد تشغيلُه
// أو عاملان معاً لا يرسلان الغداءَ مرّتين**: الصفُّ الثاني يعود بالأوّل،
// و`claim` لا يلتقط ما أُرسل.
//
// # ومن فاته الموعد لا يُلحَق
//
// **والنافذةُ ساعةٌ بعد الموعد** — **وخادمٌ عاد الخامسةَ عصراً لا يقول
// «شو بدك تفطر».**

import (
	"strconv"
	"strings"
	"time"
)

// Meal **وجبةٌ بموعدها ونصوصها.**
type Meal struct {
	Key   string // breakfast · lunch · dinner
	At    string // «09:00» بتوقيت دمشق — وفارغُه يطفئ الوجبة
	Texts string // سطرٌ لكلّ رسالة، و«العنوان | النصّ» اختياريّ
}

// MealWindow **كم بعد الموعد يُقبل الإرسال** — بعدها تُترك لليوم التالي.
const MealWindow = time.Hour

// parseClock «HH:MM» ← دقائقُ منذ منتصف الليل، و`false` لما لا يُفهم.
func parseClock(v string) (int, bool) {
	h, m, ok := strings.Cut(strings.TrimSpace(v), ":")
	if !ok {
		return 0, false
	}
	hh, e1 := strconv.Atoi(h)
	mm, e2 := strconv.Atoi(m)
	if e1 != nil || e2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, false
	}
	return hh*60 + mm, true
}

// MealDue **أحانت الوجبة؟** — ويعيد يومَها بتوقيت دمشق ليكون مفتاحَ عدمِ التكرار.
func MealDue(m Meal, now time.Time) (day string, due bool) {
	at, ok := parseClock(m.At)
	if !ok {
		return "", false
	}
	local := now.In(damascus)
	start := time.Date(local.Year(), local.Month(), local.Day(), at/60, at%60, 0, 0, damascus)
	if local.Before(start) || !local.Before(start.Add(MealWindow)) {
		return "", false
	}
	return local.Format("2006-01-02"), true
}

// MealText **رسالةُ اليوم** — تدور على السطور يوماً بعد يوم، فلا يملّ الزبون.
func MealText(texts string, day string) (title, body string, ok bool) {
	var lines []string
	for _, l := range strings.Split(texts, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "", "", false
	}
	t, err := time.ParseInLocation("2006-01-02", day, damascus)
	if err != nil {
		return "", "", false
	}
	line := lines[int(t.Unix()/86400)%len(lines)]
	title, body, _ = strings.Cut(line, "|")
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if title == "" {
		return "", "", false
	}
	return title, body, true
}
