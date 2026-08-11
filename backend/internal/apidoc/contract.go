package apidoc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// Path موضعُ العقد من جذر المستودع.
//
// **وخارجَ `docs/`** — تلك أربعُ وثائقَ تحكم طريقةَ العمل ويقرؤها المالك،
// **وهذا ملفٌّ تقرؤه الآلة.** وخلطُهما يجعل من يبحث عن قاعدةٍ يقرأ جدولَ
// حقول.
const Path = "api/contract.json"

// Endpoint نقطةٌ كاملةً: توجيهُها وشكلُها.
type Endpoint struct {
	Route
	Shape
}

// Contract العقدُ كلُّه.
type Contract struct {
	// Note سطرٌ يقرؤه من يفتح الملفّ — **وملفٌّ مولَّدٌ يُحرَّر باليد يوماً
	// إن لم يقل إنّه مولَّد.**
	Note      string     `json:"_note"`
	Endpoints []Endpoint `json:"endpoints"`
}

const note = "مولَّد من الشيفرة بـ`go run ./cmd/apidoc` — لا يُحرَّر باليد. " +
	"وحارسُ `TestContractIsCurrent` يُسقط البناءَ إن شاخ."

// Build يبني العقدَ من مجلّد الخادم.
func Build(serverDir string) (*Contract, error) {
	routes, err := Routes(serverDir)
	if err != nil {
		return nil, err
	}
	shapes, err := Shapes(serverDir)
	if err != nil {
		return nil, err
	}
	c := &Contract{Note: note}
	for _, r := range routes {
		c.Endpoints = append(c.Endpoints, Endpoint{Route: r, Shape: shapes[r.Handler]})
	}
	return c, nil
}

// Render يُخرج العقدَ نصّاً ثابتَ الترتيب.
//
// **والترتيبُ ثابتٌ عمداً**: ملفٌّ يتبدّل ترتيبُه في كلّ توليدٍ يجعل كلَّ
// `git diff` بحراً، **فلا يُرى فيه ما تغيّر فعلاً.**
func Render(c *Contract) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write يُولّد ويكتب — يُنادى من `cmd/apidoc`.
func Write(root, serverDir string) error {
	c, err := Build(serverDir)
	if err != nil {
		return err
	}
	out, err := Render(c)
	if err != nil {
		return err
	}
	target := filepath.Join(root, Path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, out, 0o644)
}
