package qa

// **الطبقةُ السابعة — عقدُ الحقول بين التطبيق والمحرّك.**
//
// المعرّفات: `CONTRACT-*` · الوسم: `@api @contract @critical @release`
//
// (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البند ٨: «اختبر الاتّفاق بين Android
//
//	وBackend… أريد اكتشافَ حالاتٍ مثل: Android يتوقّع deliveryFee
//	والخادم يعيد delivery_fee».)
//
// # ولماذا هذه أخطرُ عائلةٍ في المستودع
//
// **`ignoreUnknownKeys = true` تُسكِت الخطأ**: حقلٌ لا يجده المُفكِّك
// **يأخذ قيمتَه الافتراضيّة** — صفراً أو فارغاً أو `null` — **ولا
// استثناءَ ولا سطرَ سجلّ.**
//
// **فسعرٌ يصير صفراً وصورةٌ تصير فراغاً ولا شيءَ يقول لماذا.**
//
// **ووقعت مرّتين**: `link_url` في اللافتات (اسمٌ مخترَع، والصحيح
// `target`)، و`code` مقابل `number` في الطلبات.
//
// # ويُقارَن بالردّ الحيّ لا بالوصف
//
// **`api/contract.json` سطحيٌّ عند الأنواع المتداخلة** (`@st`) — فلا
// يقول ما داخل `banners[]`. **والردُّ الحيُّ يقوله كلَّه.**
//
// # وما يُقاس وما لا يُقاس
//
// **كلُّ حقلٍ يعلنه نموذجُ كوتلن يجب أن يوجد في الردّ.** **ولا يُقاس
// العكس**: حقلٌ في الردّ لا يعرفه التطبيق مقصودٌ كثيراً — المحرّكُ
// يخدم ثلاثةَ تطبيقاتٍ والويب.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// modelRoot **جذرُ نماذجِ العميل** — نسبةً إلى `backend/`.
const modelRoot = "../../../mobile/shared/src/main/kotlin/com/rahalgo/shared"

var (
	classRe  = regexp.MustCompile(`(?m)^data class (\w+)\s*\(`)
	serialRe = regexp.MustCompile(`@SerialName\("([^"]+)"\)`)
	propRe   = regexp.MustCompile(`\bval\s+(\w+)\s*:`)
)

// kotlinFields **الحقولُ كما يراها المُفكِّك** — الاسمُ الموسومُ إن
// وُسم، وإلّا اسمُ الخاصّيّة.
//
// **ويُقرأ الملفُّ نصّاً لا شجرة**: بناءُ محلّلِ كوتلن لأجل هذا أكبرُ
// من العطب، **والصيغةُ في هذا المستودع واحدةٌ في كلّ نموذج.**
func kotlinFields(t *testing.T, file, class string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(modelRoot, file))
	if err != nil {
		t.Skipf("qa: تعذّرت قراءةُ %s (%v) — يُتخطّى العقد", file, err)
	}
	src := string(raw)
	at := strings.Index(src, "data class "+class+"(")
	if at < 0 {
		t.Fatalf("CONTRACT: لم يُوجد النموذجُ %s في %s — تغيّر اسمُه فأصلحْ الحارس",
			class, file)
	}
	// **جسدُ النموذج** — حتّى أوّلِ سطرٍ يُغلق القوسَ في العمود الأوّل.
	rest := src[at:]
	end := strings.Index(rest, "\n)")
	if end < 0 {
		end = len(rest)
	}
	body := rest[:end]

	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "*") ||
			strings.HasPrefix(line, "/*") {
			continue
		}
		if m := serialRe.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
			continue
		}
		if m := propRe.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	if len(out) == 0 {
		t.Fatalf("CONTRACT: %s بلا حقولٍ مقروءة — تغيّرت الصيغةُ فأصلحْ الحارس", class)
	}
	return out
}

// dig **يمشي في الردّ إلى الكائن المقصود** — `orders.0` مثلاً.
func dig(v any, path string) map[string]any {
	if path == "" {
		m, _ := v.(map[string]any)
		return m
	}
	cur := v
	for _, part := range strings.Split(path, ".") {
		switch c := cur.(type) {
		case map[string]any:
			cur = c[part]
		case []any:
			if part != "0" || len(c) == 0 {
				return nil
			}
			cur = c[0]
		default:
			return nil
		}
		if cur == nil {
			return nil
		}
	}
	m, _ := cur.(map[string]any)
	return m
}

// contracts **العقودُ المرصودة** — القائمةُ صريحةٌ لا مُخمَّنة.
//
// **ونموذجٌ لا بابَ له لا يُحرَس** — تُزاد كلَّما رُبط نموذجٌ بباب.
var contracts = []struct {
	ID, File, Class, Path, JSONPath string
	// Skip **حقولٌ يحسبها العميلُ ولا يرسلها المحرّك** — تُسمّى
	// صراحةً بسببها، **وقائمةُ استثناءاتٍ بلا سببٍ تصير ثغرة.**
	Skip []string
}{
	{ID: "CONTRACT-001", File: "model/Shop.kt", Class: "Item",
		Path: "ITEMS", JSONPath: "items.0"},
	{ID: "CONTRACT-002", File: "customer/CustomerApi.kt", Class: "MyOrder",
		Path: "/api/v1/my/orders", JSONPath: "orders.0"},
	{ID: "CONTRACT-003", File: "model/Chat.kt", Class: "ChatMessage",
		Path: "MESSAGES", JSONPath: "messages.0"},
	{ID: "CONTRACT-004", File: "model/Chat.kt", Class: "ChatThreadRow",
		Path: "/api/v1/my/chats", JSONPath: "threads.0"},
}

// TestCONTRACT_FieldsExist **كلُّ حقلٍ يتوقّعه التطبيقُ موجودٌ في الردّ.**
func TestCONTRACT_FieldsExist(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1234)

	// **وتُصنع بياناتٌ حقيقيّة** — **وردٌّ فارغٌ يجعل الحارسَ يمرّ على
	// كلّ شيء**، وهو أسوأُ من غيابه.
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("CONTRACT: تعذّر تجهيزُ طلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	if got := h.POST("/api/v1/orders/"+oid+"/messages", drv.Token,
		map[string]any{"body": "رسالةُ عقد"}); got.Code >= 400 {
		t.Fatalf("CONTRACT: تعذّر تجهيزُ رسالة: %s", got)
	}

	for _, c := range contracts {
		t.Run(c.ID+"/"+c.Class, func(t *testing.T) {
			path := c.Path
			switch path {
			case "ITEMS":
				path = "/api/v1/public/sections/" + item.SectionID + "/items"
			case "MESSAGES":
				path = "/api/v1/orders/" + oid + "/messages"
			}
			res := h.GET(path, cust.Token)
			if res.Code >= 400 {
				t.Fatalf("%s نداءُ %s رُدّ: %s", c.ID, path, res)
			}
			var body any
			if err := json.Unmarshal(res.Body, &body); err != nil {
				t.Fatalf("%s ردٌّ غيرُ JSON: %v", c.ID, err)
			}
			data := dig(body, "data")
			obj := dig(data, c.JSONPath)
			if obj == nil {
				t.Fatalf("%s لا كائنَ عند %q — **ردٌّ فارغٌ يجعل الحارسَ كاذبا**\n    %s",
					c.ID, c.JSONPath, res)
			}

			skip := map[string]bool{}
			for _, s := range c.Skip {
				skip[s] = true
			}
			var missing []string
			for _, f := range kotlinFields(t, c.File, c.Class) {
				if skip[f] {
					continue
				}
				if _, ok := obj[f]; !ok {
					missing = append(missing, f)
				}
			}
			if len(missing) > 0 {
				t.Errorf("%s **حقولٌ يتوقّعها %s ولا يرسلها المحرّك**: %s\n"+
					"    وقيمتُها ستصير الافتراضَ صامتةً — صفراً أو فارغا.\n"+
					"    ما يرسله المحرّك: %s",
					c.ID, c.Class, strings.Join(missing, " · "), keysOf(obj))
			}
		})
	}
}

func keysOf(m map[string]any) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return strings.Join(out, " · ")
}

// TestCONTRACT_ModelsAreParsed **والحارسُ يقرأ نماذجَ حقيقيّة.**
//
// **وحارسٌ لا يجد ما يقرؤه يمرّ صامتا** — فيُقاس أنّ الملفّاتِ موجودةٌ
// وأنّ فيها نماذجَ تُقرأ.
func TestCONTRACT_ModelsAreParsed(t *testing.T) {
	for _, c := range contracts {
		raw, err := os.ReadFile(filepath.Join(modelRoot, c.File))
		if err != nil {
			t.Skipf("لا نماذجَ في هذا الفرع (%v)", err)
		}
		if !classRe.MatchString(string(raw)) {
			t.Errorf("%s لا نموذجَ يُقرأ في %s", c.ID, c.File)
		}
	}
}
