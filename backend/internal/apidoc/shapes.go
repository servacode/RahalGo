package apidoc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strings"
)

// Field حقلٌ في طلبٍ أو ردّ.
type Field struct {
	Name string `json:"name"`
	// Type نوعُه كما هو في Go — **أو `unknown` حين لا يُصرَّح.**
	//
	// **ولا يُخمَّن**: ١٤٣ ردّاً من ٢٤٢ مكتوبةٌ خريطةً حرّة، **والنوعُ فيها
	// غيرُ موجودٍ في الشيفرة أصلاً.** ونوعٌ مخمَّنٌ خطأً أسوأُ من نوعٍ
	// مجهولٍ معلَنٍ أنّه مجهول: **الأوّلُ يُبنى عليه فينكسر، والثاني يُسأل
	// عنه فيُصحَّح.**
	Type string `json:"type"`
	// Optional من `omitempty` — **يقرؤها العميلُ ليعرف ما قد يغيب.**
	Optional bool `json:"optional,omitempty"`
}

// Shape شكلُ نقطةٍ: ما تستقبل وما تردّ.
type Shape struct {
	Request  []Field `json:"request,omitempty"`
	Response []Field `json:"response,omitempty"`
	// Statuses رموزُ الحالة التي تردّها — **يعرف العميلُ ما ينتظر.**
	Statuses []string `json:"statuses,omitempty"`
}

// Shapes يقرأ معالِجاتِ الخادم ويُخرج شكلَ كلٍّ منها باسمه.
func Shapes(dir string) (map[string]Shape, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]Shape{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Body == nil {
					continue
				}
				if sh, ok := shapeOf(fn); ok {
					out[fn.Name.Name] = sh
				}
			}
		}
	}
	return out, nil
}

// shapeOf يستخرج شكلَ معالِجٍ واحد.
func shapeOf(fn *ast.FuncDecl) (Shape, bool) {
	var sh Shape
	seen := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch {
		case isDecodeCall(call):
			if f := decodeFields(call); len(f) > 0 && sh.Request == nil {
				sh.Request = f
			}
		case isJSONCall(call) && len(call.Args) == 3:
			if st := exprName(call.Args[1]); st != "" && !seen["s:"+st] {
				seen["s:"+st] = true
				sh.Statuses = append(sh.Statuses, st)
			}
			for _, f := range responseFields(call.Args[2]) {
				if seen[f.Name] {
					continue
				}
				seen[f.Name] = true
				sh.Response = append(sh.Response, f)
			}
		}
		return true
	})
	sort.Strings(sh.Statuses)
	sort.Slice(sh.Response, func(i, j int) bool { return sh.Response[i].Name < sh.Response[j].Name })
	return sh, sh.Request != nil || sh.Response != nil || sh.Statuses != nil
}

// isDecodeCall أهي `decode[struct{…}](r)`؟
//
// **والاستدعاءُ بمعامِلٍ نوعيّ يظهر في الشجرة `IndexExpr`** — الدالّةُ
// `decode` والفهرسُ هو النوع.
func isDecodeCall(call *ast.CallExpr) bool {
	idx, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return false
	}
	id, ok := idx.X.(*ast.Ident)
	return ok && id.Name == "decode"
}

func decodeFields(call *ast.CallExpr) []Field {
	idx := call.Fun.(*ast.IndexExpr)
	st, ok := idx.Index.(*ast.StructType)
	if !ok || st.Fields == nil {
		return nil
	}
	var out []Field
	for _, f := range st.Fields.List {
		name, optional := jsonName(f)
		if name == "" || name == "-" {
			continue
		}
		out = append(out, Field{Name: name, Type: typeString(f.Type), Optional: optional})
	}
	return out
}

// jsonName اسمُ الحقل في JSON — **من الوسم لا من اسمه في Go.**
//
// **والاسمان يفترقان دائماً**: `FullName` في Go و`full_name` في الشبكة.
// **ومن قرأ اسمَ Go كتب حقلاً لا وجودَ له.**
func jsonName(f *ast.Field) (string, bool) {
	if f.Tag == nil {
		if len(f.Names) > 0 {
			return f.Names[0].Name, false
		}
		return "", false
	}
	// **والوسمُ يصل بعلامتين خلفيّتين حوله** — تُقشَّران بالرمز لا بالحرف،
	// **وحارسُ `TestNoBacktickInsideRawStrings` يسقط على علامةٍ فردية في
	// الملفّ** (وهي قاعدةٌ في `CLAUDE.md`: العلامةُ الخلفيّةُ تُنهي النصَّ
	// الخام في Go).
	const backtick = 0x60
	tag := strings.Trim(f.Tag.Value, string(rune(backtick)))
	i := strings.Index(tag, `json:"`)
	if i < 0 {
		if len(f.Names) > 0 {
			return f.Names[0].Name, false
		}
		return "", false
	}
	rest := tag[i+6:]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return "", false
	}
	parts := strings.Split(rest[:j], ",")
	optional := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			optional = true
		}
	}
	return parts[0], optional
}

// responseFields حقولُ الردّ — **من خريطةٍ صريحةٍ أو من نوعٍ له اسم.**
func responseFields(e ast.Expr) []Field {
	switch v := e.(type) {
	case *ast.CompositeLit:
		// `map[string]any{"a": 1, "b": x}` — **المفاتيحُ دقيقةٌ والأنواعُ
		// بحسب ما يظهر.**
		var out []Field
		for _, el := range v.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := stringLit(kv.Key)
			if !ok {
				continue
			}
			out = append(out, Field{Name: key, Type: literalType(kv.Value)})
		}
		return out
	case *ast.Ident, *ast.SelectorExpr, *ast.UnaryExpr, *ast.CallExpr:
		// **متغيّرٌ له نوع** — الاسمُ يُذكر ولا يُفكّ هنا.
		//
		// **وفكُّه يحتاج مُدقّقَ أنواعٍ كاملاً** (`go/types`) يحمّل الحزمةَ
		// وتبعيّاتِها. **وهو ممكنٌ ويستحقّ يوماً** — والاسمُ يكفي اليوم
		// لمن يقرأ الشيفرة، **والحقولُ تظهر حين يُحوَّل الردُّ إلى نوعٍ
		// معرَّفٍ في الحزمة.**
		if n := exprName(e); n != "" {
			return []Field{{Name: "@" + n, Type: "unknown"}}
		}
	}
	return nil
}

// literalType نوعٌ يُقرأ من القيمة نفسِها — **أو `unknown`.**
func literalType(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float"
		case token.STRING:
			return "string"
		}
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" {
			return "bool"
		}
	case *ast.CompositeLit:
		if _, ok := v.Type.(*ast.ArrayType); ok {
			return "array"
		}
		return "object"
	}
	return "unknown"
}

// typeString النوعُ كما كُتب في Go — **مقروءاً لا كاملَ المسار.**
func typeString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "?" + typeString(v.X) // **مؤشّرٌ = قد يكون فارغاً**
	case *ast.ArrayType:
		return "[]" + typeString(v.Elt)
	case *ast.SelectorExpr:
		return typeString(v.X) + "." + v.Sel.Name
	case *ast.MapType:
		return "map[" + typeString(v.Key) + "]" + typeString(v.Value)
	case *ast.InterfaceType:
		return "any"
	case *ast.StructType:
		return "object"
	}
	return "unknown"
}

// isJSONCall أهو `httpx.JSON(w, status, data)`؟
func isJSONCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "JSON" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "httpx"
}
