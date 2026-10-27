package obligations

// حارسُ ترجمة الأسباب (تدقيقُ ٢٠٢٦-١٠-٢٧): **كلُّ ثابتٍ `Cause*` هنا له لفظٌ عربيّ** في
// `admin.obligations.causes`. نقص `refund_rep_share` فطلع للموظّف بالإنكليزي وغاب عن قائمة الأسباب.
// **لا يحتاج قاعدة** — يقرأ الملفَّين وحدَهما.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEveryCauseHasArabicLabel(t *testing.T) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "obligations.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	causes := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range vs.Names {
			if !strings.HasPrefix(name.Name, "Cause") || i >= len(vs.Values) {
				continue
			}
			if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				v, _ := strconv.Unquote(lit.Value)
				causes[name.Name] = v
			}
		}
		return true
	})
	if len(causes) == 0 {
		t.Fatal("no Cause* constants found")
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ar struct {
		Admin struct {
			Obligations struct {
				Causes map[string]string `json:"causes"`
			} `json:"obligations"`
		} `json:"admin"`
	}
	if err := json.Unmarshal(raw, &ar); err != nil {
		t.Fatal(err)
	}
	for name, v := range causes {
		if strings.TrimSpace(ar.Admin.Obligations.Causes[v]) == "" {
			t.Errorf("%s (%q) has no label in admin.obligations.causes", name, v)
		}
	}
}
