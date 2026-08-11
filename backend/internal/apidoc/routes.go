// Package apidoc **يُولّد عقدَ الـAPI من الشيفرة نفسِها.**
//
// ══════════════════════════════════════════════════════════════════════
// # المسألة
// ══════════════════════════════════════════════════════════════════════
//
// **٢٢٥ نقطةً في المحرّك، و١٦٩ نموذجاً مكتوباً باليد في الويب.** ولا عقدَ
// بينهما — **الواجهةُ تعرف شكلَ الردّ لأنّ من كتبها كتب المحرّك.**
//
// **وكان ذلك محتمَلاً بعميلٍ واحد.** ومع تطبيقَي أندرويد يصير ثلاثةَ عملاء
// يكتبون النماذجَ نفسَها ثلاثَ مرّات، **وحقلٌ يتبدّل في المحرّك ينكسر في
// ثلاثة أماكنَ بصمت** — لا خطأَ في بناء، ولا سطرَ في سجلّ، **إنّما شاشةٌ
// تعرض فراغاً.**
//
// ══════════════════════════════════════════════════════════════════════
// # ولماذا يُولَّد ولا يُكتب
// ══════════════════════════════════════════════════════════════════════
//
// **عقدٌ مكتوبٌ باليد يشيخ.** يُبدَّل حقلٌ في المحرّك، **ولا شيءَ يُجبر
// كاتبَه أن يفتح الملفّ.** فيصير العقدُ يقول شيئاً والمحرّكُ يفعل غيرَه —
// **وهو أسوأُ من ألّا يكون عقد**، لأنّ من قرأه وثق به.
//
// **والمولَّدُ لا يكذب**، وحارسٌ يُسقط البناءَ إن شاخ — كما `truthdoc`.
//
// ══════════════════════════════════════════════════════════════════════
// # وما تبلغه هذه الأداةُ وما لا تبلغه
// ══════════════════════════════════════════════════════════════════════
//
// **تبلغ بدقّة**: المسارَ والطريقةَ وشرطَ المصادقة والأدوارَ المطلوبة،
// **وشكلَ الطلب** حيث كُتب بـ`decode[struct{…}]` (٧٨ موضعاً).
//
// **وتبلغ أسماءَ حقول الردّ** حيث كُتب `httpx.JSON` بخريطةٍ صريحة.
//
// **ولا تبلغ أنواعَ بعض حقول الردّ** — ١٤٣ من ٢٤٢ ردّاً مكتوبةٌ خريطةً
// حرّةً (`map[string]any`)، **والنوعُ فيها غيرُ مصرَّحٍ في الشيفرة أصلاً.**
// فتُكتب `unknown`، **ولا تُخمَّن**: نوعٌ مخمَّنٌ خطأً أسوأُ من نوعٍ مجهولٍ
// معلَنٍ أنّه مجهول.
//
// **والعلاجُ تدريجيٌّ**: كلَّما حُوّل ردٌّ إلى نوعٍ له اسم، عرفته الأداةُ
// من تلقاء نفسِها.
package apidoc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Route نقطةُ نهايةٍ واحدة كما هي في الراوتر.
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	// Handler اسمُ المعالِج — **مفتاحُ الربط بين المسار وشكله.**
	Handler string `json:"handler"`
	// Auth أتحتاج توكناً؟ — من `RequireAuth` في السلسلة.
	Auth bool `json:"auth"`
	// Roles الأدوارُ المخوَّلة — فارغةٌ تعني «كلّ من دخل».
	Roles []string `json:"roles,omitempty"`
	// Idempotent **ملفوفةٌ بمنع التكرار** — يقرؤها العميلُ ليعرف أنّ عليه
	// أن يرسل مفتاحاً. (انظر `server/idempotency.go`.)
	Idempotent bool `json:"idempotent,omitempty"`
}

// httpMethods ما يُعدّ تسجيلَ مسار — **ولا `Use` ولا `Handle`.**
var httpMethods = map[string]bool{
	"Get": true, "Post": true, "Put": true, "Patch": true, "Delete": true,
}

// scope حالةُ التوجيه في نقطةٍ من الشجرة — **تُنسخ عند كلّ تفرّع.**
//
// **ولا تُشارَك بالمرجع**: `Route` داخلَ `Group` يرث ما فوقه **ولا يورّثه
// إخوتَه** — ولو شورك المرجعُ لَتسرّب دورُ قسمٍ إلى القسم الذي يليه.
type scope struct {
	prefix string
	auth   bool
	roles  []string
}

// with يُفرّع الحالةَ — **والأدوارُ تُقاطَع لا تُجمَع.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا التقاطعُ هو الصحيح**
// ══════════════════════════════════════════════════════════════════════
//
// **`RequireRoles("a","b")` تعني «أحدُهما يكفي».** ووسيطان في سلسلةٍ
// واحدة يعملان معاً — **فمن يمرّ لا بدّ أن يُرضيَ الاثنين.**
//
// **مثالٌ من الشيفرة**: قسمُ الإدارة كلُّه لـ`admin · ops · finance`،
// **وسجلُّ الأحداث داخلَه مقيَّدٌ بـ`admin · finance`** لأنّه يحوي مبالغَ
// التعويضات — **وموظّفُ العمليات ليس طرفاً في المال.**
//
// **فلو جُمعت** لَقال العقدُ إنّ `ops` يصل إلى السجلّ **وهو لا يصل** —
// وذاك عقدٌ يكذب في الصلاحيات، **وهو أخطرُ ما يكذب فيه عقد.**
func (s scope) with(prefix string, auth bool, roles []string) scope {
	out := scope{prefix: s.prefix + prefix, auth: s.auth || auth}
	switch {
	case len(roles) == 0:
		out.roles = append([]string{}, s.roles...)
	case len(s.roles) == 0:
		out.roles = append([]string{}, roles...)
	default:
		have := map[string]bool{}
		for _, r := range s.roles {
			have[r] = true
		}
		for _, r := range roles {
			if have[r] {
				out.roles = append(out.roles, r)
			}
		}
	}
	return out
}

// Routes يقرأ ملفَّ الراوتر ويُخرج كلَّ نقطةٍ فيه مرتَّبة.
func Routes(dir string) ([]Route, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(dir, "server.go"), nil, 0)
	if err != nil {
		return nil, err
	}
	var out []Route
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Router" {
			return true
		}
		walk(fn.Body, scope{}, &out)
		return false
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out, nil
}

// walk يمشي في جسمٍ من الشيفرة جامعاً المسارات — **بحالةٍ موروثة.**
func walk(body *ast.BlockStmt, sc scope, out *[]Route) {
	if body == nil {
		return
	}
	for _, stmt := range body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		walkCall(call, sc, out)
	}
}

// walkCall يفكّ نداءً واحداً — **وقد يكون سلسلةً** (`r.With(…).Route(…)`).
func walkCall(call *ast.CallExpr, sc scope, out *[]Route) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	name := sel.Sel.Name

	// **والسلسلةُ تُقرأ من داخلها إلى خارجها**: `r.With(x).Get(p, h)` —
	// `With` هي المستقبِل، **فتُقرأ أوّلاً لتُعرف الأدوارُ قبل التسجيل.**
	inner := sc
	if recv, ok := sel.X.(*ast.CallExpr); ok {
		if rsel, ok := recv.Fun.(*ast.SelectorExpr); ok && rsel.Sel.Name == "With" {
			inner = sc.with("", hasRequireAuth(recv.Args), rolesFrom(recv.Args))
		}
	}

	switch {
	case name == "Use":
		// **`Use` تُغيّر ما بعدها في الكتلة نفسِها** — تُعالَج في `walkBlock`.
		return

	case name == "Route" && len(call.Args) == 2:
		prefix, okp := stringLit(call.Args[0])
		fn, okf := call.Args[1].(*ast.FuncLit)
		if okp && okf {
			walkBlock(fn.Body, inner.with(prefix, false, nil), out)
		}

	case name == "Group" && len(call.Args) == 1:
		if fn, ok := call.Args[0].(*ast.FuncLit); ok {
			walkBlock(fn.Body, inner, out)
		}

	case httpMethods[name] && len(call.Args) == 2:
		path, okp := stringLit(call.Args[0])
		if !okp {
			return
		}
		handler, idem := handlerName(call.Args[1])
		*out = append(*out, Route{
			Method: strings.ToUpper(name), Path: inner.prefix + path,
			Handler: handler, Auth: inner.auth, Roles: dedupe(inner.roles),
			Idempotent: idem,
		})
	}
}

// walkBlock مثل `walk` لكنّه يلتقط `Use` **قبل** ما بعدها.
//
// **و`Use` تسري على الكتلة كلِّها لا على ما يليها فقط** — هكذا يعمل chi،
// **ولو قُرئت بالترتيب لَخرجت نقاطٌ بلا مصادقةٍ وهي محميّة.**
func walkBlock(body *ast.BlockStmt, sc scope, out *[]Route) {
	if body == nil {
		return
	}
	for _, stmt := range body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Use" {
			continue
		}
		sc = sc.with("", hasRequireAuth(call.Args), rolesFrom(call.Args))
	}
	walk(body, sc, out)
}

func hasRequireAuth(args []ast.Expr) bool {
	for _, a := range args {
		if strings.Contains(exprName(a), "RequireAuth") {
			return true
		}
	}
	return false
}

// rolesFrom الأدوارُ من `RequireRoles("a","b")`.
func rolesFrom(args []ast.Expr) []string {
	var out []string
	for _, a := range args {
		call, ok := a.(*ast.CallExpr)
		if !ok || !strings.Contains(exprName(call.Fun), "RequireRoles") {
			continue
		}
		for _, ra := range call.Args {
			if s, ok := stringLit(ra); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// handlerName اسمُ المعالِج — **ويكشف لفَّ منع التكرار.**
func handlerName(e ast.Expr) (string, bool) {
	if call, ok := e.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "idempotent" {
			if len(call.Args) == 1 {
				n, _ := handlerName(call.Args[0])
				return n, true
			}
		}
	}
	name := exprName(e)
	// **ومعالِجٌ مكتوبٌ في موضعه** (`func(w,r){…}`) لا اسمَ له.
	if name == "" {
		return "inline", false
	}
	return name, false
}

func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.CallExpr:
		return exprName(v.Fun)
	}
	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	return s, err == nil
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
