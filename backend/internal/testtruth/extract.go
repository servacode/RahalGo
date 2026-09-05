// المستخرِجات — **ما تقوله الشيفرةُ عن نفسها.**
//
// **ولا سطرَ ممّا هنا مكتوبٌ بيد** — كلُّه يُقرأ عند كلّ توليد، **فما
// تبدّل في الشيفرة تبدّل هنا في الحال.**
//
// # ولماذا نصٌّ لا `go/ast` في أكثرها
//
// **الموجّهُ والمعجمُ والانتقالاتُ بِنًى معلنةٌ بشكلٍ ثابت** — **ونمطٌ
// نصّيٌّ دقيقٌ يكفيها ويقرؤه من يصونه.** **و`go/ast` يلزم حيث يتشعّب
// الشكل** — كجرد دوالّ الاختبار، **وهو ما يُستعمل فيه.**
package testtruth

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Root جذرُ `backend/` — يُشتقّ من موضع هذا الملفّ عند التشغيل.
type Root string

// ══════════════════════════════════════════════════════════════════════
// **الاختباراتُ — بـ`go/ast` لا بنمطٍ نصّيّ**
// ══════════════════════════════════════════════════════════════════════
//
// **الدالّةُ قد تُكتب بأشكال**، **والمحلّلُ يعرفها كلَّها.**

var reTestName = regexp.MustCompile(`^(Test|Benchmark|Fuzz|Example)\w*$`)

// Tests يجرد كلَّ دالّةِ اختبارٍ في الشجرة.
func (r Root) Tests() ([]TestFn, int, error) {
	var out []TestFn
	files := map[string]bool{}
	fset := token.NewFileSet()

	err := filepath.WalkDir(string(r), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// **ما لا يُختبَر لا يُجرَد.**
			switch d.Name() {
			case "vendor", "node_modules", ".git", "build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil // ملفٌّ لا يُحلَّل لا يُسقط الجرد — ويظهر بنقصه
		}
		rel, _ := filepath.Rel(string(r), path)
		rel = filepath.ToSlash(rel)
		files[rel] = true
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !reTestName.MatchString(fn.Name.Name) {
				continue
			}
			out = append(out, TestFn{
				Name:    fn.Name.Name,
				File:    rel,
				Package: f.Name.Name,
			})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Name < out[j].Name
	})
	return out, len(files), err
}

// ══════════════════════════════════════════════════════════════════════
// **الأبوابُ — من الموجّه**
// ══════════════════════════════════════════════════════════════════════

var reRoute = regexp.MustCompile(`(?:^|[.\s])(Get|Post|Patch|Delete|Put)\("(/[^"]*)"`)

// Routes يعدّ الأبوابَ المسجَّلة.
func (r Root) Routes() (int, error) {
	b, err := os.ReadFile(filepath.Join(string(r), "internal/server/server.go"))
	if err != nil {
		return 0, err
	}
	return len(reRoute.FindAllSubmatch(b, -1)), nil
}

// ══════════════════════════════════════════════════════════════════════
// **الانتقالاتُ — من الوثيقة المولَّدة أصلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **`ORDER_TRANSITIONS_55.md` مولَّدةٌ من `statuses.go`** — فقراءتُها
// قراءةُ الشيفرة بخطوةٍ واحدة، **ولا تُنسَخ.**

// **الصيغةُ الحقيقيّةُ عمودان لا سهم**: `| N01 | Pending | Accepted | …`
//
// **وقِيست لا فُرضت** — **ونمطُ السهم الذي افترضتُه أوّلاً ردّ ٣ من ٥٥**،
// لأنّ الوثيقةَ تكتب الحالَ في عمودين لا بسهمٍ بينهما.
var reTransRow = regexp.MustCompile("(?m)^\\|\\s*[A-Z]+\\d+\\s*\\|\\s*`[A-Za-z_]+`\\s*\\|\\s*`[A-Za-z_]+`\\s*\\|")

// Transitions يعدّ الانتقالاتِ الموصوفة.
func (r Root) Transitions(docsRoot string) (int, error) {
	b, err := os.ReadFile(filepath.Join(docsRoot, "testing/ORDER_TRANSITIONS_55.md"))
	if err != nil {
		return 0, err
	}
	n := len(reTransRow.FindAll(b, -1))
	if n == 0 {
		// **وصمتُ المستخرِج أخطرُ من سقوطه** — ورقمٌ خاطئٌ يُصدَّق.
		return 0, fmt.Errorf("لم أجد صفَّ انتقالٍ واحداً — تبدّلت صيغةُ الوثيقة")
	}
	return n, nil
}

// ══════════════════════════════════════════════════════════════════════
// **أنواعُ القيد — من قيد القاعدة نفسِه**
// ══════════════════════════════════════════════════════════════════════

// **والقائمةُ تمتدُّ أسطراً وفيها تعليقاتٌ عربيّة** — فتُقرأ من `kind IN (`
// **إلى قوسها المقابل، لا إلى أوّل قوسٍ يُصادَف.**
//
// **ونمطٌ توقّف عند أوّل قوسٍ ردّ نوعاً واحداً من تسعة.**
// **ويُمسك الشكلان**: `CREATE TABLE` و`ADD CONSTRAINT` — **ونمطٌ اقتصر
// على الأوّل قرأ `0010` وحدَها فردّ سبعةَ أنواعٍ من تسعة.**
// **والصيغتان كلتاهما**: الهجراتُ القديمةُ تكتب `kind IN (` والأخيرتان
// (`0109` و`0110`) تكتبان `kind = ANY (ARRAY[` — **ومن عرف واحدةً
// قرأ قيداً شائخاً.** (وقع: كان يردّ ثلاثةَ عشرَ نوعاً وفي القاعدة أربعةَ
// عشر، **والناقصُ `operating_expense`** — كُشف في `P-4`.)
var reKindsHead = regexp.MustCompile(`kind (?:IN|=\s*ANY)\s*\(`)
var reQuoted = regexp.MustCompile(`'([a-z_]+)'`)

// LedgerKinds يقرأ أنواعَ قيود المحفظة — **من آخرِ تعريفٍ للقيد لا أوّلِه.**
//
// **وقراءةُ `0010` وحدَها تكذب**: `0033` و`0034` يُسقطان القيدَ ويُعيدان
// بناءَه بنوعين إضافيّين (`merchant_earning` · `driver_earning`).
// **فالهجراتُ تُمشى بترتيبها، وآخرُ ما يُبنى هو الحقيقة.**
func (r Root) LedgerKinds() ([]string, error) {
	dir := filepath.Join(string(r), "internal/migrate/migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files) // **الترتيبُ الرقميُّ هو ترتيبُ التطبيق.**

	var latest []string
	for _, f := range files {
		b, rerr := os.ReadFile(f)
		if rerr != nil {
			continue
		}
		src := string(b)
		for _, loc := range reKindsHead.FindAllStringIndex(src, -1) {
			// **قيدُ المحفظة وحدَه** — و`media` و`promos` لها قيودُ `kind` أيضاً.
			head := src[max0(loc[0]-260):loc[0]]
			if !strings.Contains(head, "wallet_transactions") {
				continue
			}
			var got []string
			for _, q := range reQuoted.FindAllStringSubmatch(callBlock(src, loc[1]-1), -1) {
				got = append(got, q[1])
			}
			if len(got) > 0 {
				latest = got
			}
		}
	}
	if len(latest) == 0 {
		return nil, fmt.Errorf("لم أجد قيدَ أنواع المحفظة — تبدّلت الهجرات")
	}
	sort.Strings(latest)
	return latest, nil
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **الإشعاراتُ — بمطابقة الأقواس لا بنافذةٍ ثابتة**
// ══════════════════════════════════════════════════════════════════════
//
// **ونافذةٌ ثابتةٌ أخطأت أوّلَ مرّة** (٢٠٢٦-٠٩-٠٥): **قرأت `Apps` من
// الكتلة التالية فنسبت التوجيهَ إلى موضعٍ لا توجيهَ فيه.**

var reNotify = regexp.MustCompile(`s\.notify\.(Notify|NotifyRoles|NotifyMany|NotifyOps|NotifyWallet)\s*\(`)

// NotifySites يعدّ مواضعَ الإشعار وكم منها موجَّهٌ بـ`Apps`.
func (r Root) NotifySites() (total, targeted int, err error) {
	err = filepath.WalkDir(filepath.Join(string(r), "internal"), func(path string, d fs.DirEntry, e error) error {
		if e != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		src := string(b)
		for _, loc := range reNotify.FindAllStringIndex(src, -1) {
			total++
			if strings.Contains(callBlock(src, loc[1]-1), "Apps:") {
				targeted++
			}
		}
		return nil
	})
	return total, targeted, err
}

// callBlock نصُّ النداء من قوسه الأوّل إلى قوسه المقابل.
func callBlock(src string, open int) string {
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return src[open : i+1]
			}
		}
	}
	return ""
}

// ══════════════════════════════════════════════════════════════════════
// **غرفُ البثّ**
// ══════════════════════════════════════════════════════════════════════

var rePublish = regexp.MustCompile(`\.Publish\("([a-z]+)(?::"|")`)

// PublishRooms يجرد أسماءَ الغرف التي يُبَثّ إليها.
func (r Root) PublishRooms() ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(string(r), "internal"), func(path string, d fs.DirEntry, e error) error {
		if e != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for _, m := range rePublish.FindAllSubmatch(b, -1) {
			seen[string(m[1])] = true
		}
		return nil
	})
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, err
}

// ══════════════════════════════════════════════════════════════════════
// **الإعداداتُ — من معجمها**
// ══════════════════════════════════════════════════════════════════════

var (
	// **ولا يُشترَط ترتيبُ الحقول في السطر.**
	//
	// **نمطٌ اشترط `Key` ثمّ `Group` ثمّ `Kind` متجاورةً ردّ ٩٢ من ١١٨**
	// — **واثنان وعشرون تعريفاً تكتب حقولَها بترتيبٍ آخرَ أو على أسطر.**
	// **فيُلتقَط المفتاحُ وحدَه، ويُقرأ الباقي من جسم التعريف.**
	reKey       = regexp.MustCompile(`\{Key:\s*"([^"]+)"`)
	reGroup     = regexp.MustCompile(`Group:\s*Group(\w+)`)
	reKind      = regexp.MustCompile(`Kind:\s*Kind(\w+)`)
	reCondition = regexp.MustCompile(`Condition\{Key:\s*"([^"]+)"`)
	reSensitive = regexp.MustCompile(`Sensitive:\s*true`)
)

// SettingDef تعريفُ مفتاحٍ كما في المعجم.
type SettingDef struct {
	Key       string
	Group     string
	Kind      string
	Sensitive bool
}

// SettingDefs يقرأ تعريفاتِ الإعدادات — **ويميّز `Def` عن `Condition`.**
//
// **والخلطُ بينهما أخطأ مرّةً**: **١٢٧ ذِكراً لـ`Key:` منها ٩ شروطُ ظهور**
// ⇒ **١١٨ تعريفاً لا ١٢٧.**
func (r Root) SettingDefs() ([]SettingDef, error) {
	b, err := os.ReadFile(filepath.Join(string(r), "internal/settings/catalog.go"))
	if err != nil {
		return nil, err
	}
	src := string(b)

	// **مواضعُ الشروطِ تُستثنى بمواضعها لا بأسمائها** — فقد يتشابه مفتاح.
	cond := map[int]bool{}
	for _, loc := range reCondition.FindAllStringIndex(src, -1) {
		cond[loc[0]] = true
	}

	var out []SettingDef
	for _, m := range reKey.FindAllStringSubmatchIndex(src, -1) {
		start := m[0]
		if cond[start] || (start > 12 && strings.Contains(src[start-12:start], "Condition")) {
			continue
		}
		// **جسمُ التعريفِ حتّى مطلعِ التالي** — ومنه تُقرأ بقيّةُ الحقول.
		body := src[start:min(start+700, len(src))]
		if i := strings.Index(body[1:], "{Key:"); i > 0 {
			body = body[:i+1]
		}
		out = append(out, SettingDef{
			Key:       src[m[2]:m[3]],
			Group:     lowerSub(reGroup, body),
			Kind:      lowerSub(reKind, body),
			Sensitive: reSensitive.MatchString(body),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// lowerSub أوّلُ التقاطٍ بحروفٍ صغيرة — وفراغٌ إن لم يوجد.
func lowerSub(re *regexp.Regexp, body string) string {
	if m := re.FindStringSubmatch(body); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ══════════════════════════════════════════════════════════════════════
// **حقولُ الطلب — من `P-1`**
// ══════════════════════════════════════════════════════════════════════
//
// **لا يُعاد استخراجُها هنا** — **`qa.OrderFields()` تفعلها بالانعكاس،
// وتُستهلَك.** (البند ١٧ و٢٠ من طلب المالك: لا حقيقةَ مكرّرة.)

// ══════════════════════════════════════════════════════════════════════
// **العيوبُ والمخاطرُ — من السجلّ المجمَّد**
// ══════════════════════════════════════════════════════════════════════

var (
	reDefectRow = regexp.MustCompile("(?m)^\\| \\*\\*(D\\d+)\\*\\* \\| ([^|]+) \\|")
	reRiskRow   = regexp.MustCompile("(?m)^\\| \\*\\*(R\\d+)\\*\\* \\| ([^|]+) \\|")
)

// RegisterRow سطرٌ من السجلّ المجمَّد.
type RegisterRow struct {
	ID    string
	Title string
}

// Register يقرأ العيوبَ والمخاطرَ من السجلّ — **ولا يُنسَخ عددٌ بيد.**
func Register(docsRoot string) (defects, risks []RegisterRow, err error) {
	b, err := os.ReadFile(filepath.Join(docsRoot, "testing/FINAL_STATIC_CLOSEOUT.md"))
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for _, m := range reDefectRow.FindAllSubmatch(b, -1) {
		id := string(m[1])
		if seen[id] {
			continue
		}
		seen[id] = true
		defects = append(defects, RegisterRow{ID: id, Title: clean(string(m[2]))})
	}
	seen = map[string]bool{}
	for _, m := range reRiskRow.FindAllSubmatch(b, -1) {
		id := string(m[1])
		if seen[id] {
			continue
		}
		seen[id] = true
		risks = append(risks, RegisterRow{ID: id, Title: clean(string(m[2]))})
	}
	sort.Slice(defects, func(i, j int) bool { return num(defects[i].ID) < num(defects[j].ID) })
	sort.Slice(risks, func(i, j int) bool { return num(risks[i].ID) < num(risks[j].ID) })
	return defects, risks, nil
}

func clean(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	// **والعلامةُ الخلفيّةُ تُكتب برمزها لا بحرفها** — `\x60`.
	//
	// **وحارسُ `TestNoBacktickInsideRawStrings` يعدّها في الملفّ كلِّه بلا
	// تمييزِ سياق**، فعلامةٌ واحدةٌ في نصٍّ مقتبَسٍ تجعل العدَّ فردياً
	// **فيُنذر بنصٍّ خامٍّ لم يُغلق وليس ثمّةَ شيء.** (كان يسقط منذ `P-2`.)
	//
	// **ولم يُضعَّف الحارسُ ليمرّ**: هو محقٌّ في تشدّده — **علامةٌ خلفيّةٌ
	// داخل نصٍّ خامٍّ في Go تكسر البناءَ فعلاً** (وهي مزلقةٌ مكتوبةٌ في
	// `CLAUDE.md`). **فالمكتوبُ هو ما تبدّل، لا القاعدة.**
	s = strings.ReplaceAll(s, "\x60", "")
	return strings.TrimSpace(s)
}

func num(id string) int {
	n := 0
	for _, c := range id[1:] {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
