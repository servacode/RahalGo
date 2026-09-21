// أمرُ `flag` — **قلبُ إعدادٍ مسموحٍ في التجهيز وقراءتُه واستعادتُه.**
//
// (طورُ فتحِ القدرات ٢٠٢٦-٠٩-٢١ · الفئة B من دفتر القدرات المتبقّية.)
//
// # لماذا لا `psql` ولا نقطةُ إدارةٍ عامّة
//
// **حملةُ قبولِ الزبون تحتاج قلبَ أعلامِ الإطلاق ثمّ إعادتَها بالضبط** —
// (`CUST-18-001..020`، `CUST-CUSTOM-006`، `ENG-011`). **وقصاصةُ `SQL`
// يدويّةٌ تنسى `WHERE` أو تكتب قيمةً بلا تحقّق**، **ونقطةُ إدارةٍ «تفعل أيَّ
// شيء» بابٌ خلفيّ.** فالأمرُ هنا:
//
//	· يمرّ بحارس الإنتاج أوّلاً (خمسةُ حرّاسٍ — `envguard`)؛
//	· لا يقبل إلّا مفاتيحَ قائمةِ السماح (`qaFlags`)؛
//	· يقرأ القيمةَ القديمةَ ويطبعها قبل الكتابة (دليلُ ما قبل)؛
//	· يكتب عبر `settings.Set` وحدَه (تحقُّقُ الكتالوج + أثرُ `updated_by`)؛
//	· يطبع أمرَ الاستعادةِ الدقيق بعدَها؛
//	· لا `SQL` حرّ، ولا مفتاحٌ خارجَ القائمة، ولا بدائلُ (fail-closed).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// qaFlags **قائمةُ السماح** — لا يُقلَب إلّا ما تحتاجه حملةُ قبولِ الزبون.
//
// **وكلُّ مفتاحٍ هنا مقصودٌ ببندٍ**: أعلامُ إطلاق الزبون وإشعارُها، وإلزامُ
// ساعات المنصّة، وأدنى نسخةِ التطبيق، وحقولُ صفحةِ التواصل. **وما ليس هنا
// يُرفَض** — ولا يُوسَّع إلّا ببندِ قبولٍ يطلبه.
var qaFlags = map[string]string{
	"launch.customer_signup":        "CUST-18-001/002",
	"launch.customer_browse":        "CUST-18-003/004",
	"launch.customer_orders":        "CUST-18-005/006 · 13-018",
	"launch.customer_custom_orders": "CUST-18-007 · CUSTOM-006",
	"launch.notice":                 "إشعارُ الإطلاق",
	"hours.platform_enforced":       "CUST-18-008/009",
	"app.min_version.customer":      "CUST-18-019/020",
	"platform.support_phone":        "CUST-ENG-011",
	"platform.whatsapp":             "CUST-ENG-011",
	"platform.facebook":             "CUST-ENG-011",
	"platform.instagram":            "CUST-ENG-011",
	"platform.telegram":             "CUST-ENG-011",
	"platform.address":              "CUST-ENG-011",
}

// flagCmd ينفّذ `stagingctl flag get|set` — **بعد أن رضي الحارسُ.**
func flagCmd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("الاستعمال: stagingctl flag get <مفتاح>  |  stagingctl flag set <مفتاح> <قيمة>")
	}
	sub, key := args[0], args[1]

	// ── قائمةُ السماح: أوّلُ حارسٍ بعد حارس الإنتاج ──────────────────
	reason, ok := qaFlags[key]
	if !ok {
		return fmt.Errorf("**المفتاحُ %q ليس في قائمة السماح** — حملةُ القبول تُقلّب %d مفتاحاً لا غير", key, len(qaFlags))
	}
	// ── ولا مفتاحٌ خارجَ الكتالوج ولو كان في القائمة (دفاعٌ ثانٍ) ─────
	def, known := settings.Lookup(key)
	if !known {
		return fmt.Errorf("**المفتاحُ %q غيرُ معروفٍ في الكتالوج** — رُفض", key)
	}

	db, err := pool()
	if err != nil {
		return err
	}
	defer db.Close()
	store := settings.NewStore(db)
	ctx := context.Background()

	// **ودليلُ ما قبلَ الكتابة يُقرأ دائماً** — للقراءة وللاستعادة سواء.
	before, _ := store.GetRaw(ctx, key)
	beforeStr := strings.TrimSpace(string(before))
	if beforeStr == "" {
		beforeStr = "(افتراضيّ — لا صفَّ)"
	}

	switch sub {
	case "get":
		fmt.Printf("%s = %s   [%s · %s]\n", key, beforeStr, def.Kind, reason)
		return nil

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("set يحتاج قيمة: stagingctl flag set %s <قيمة>", key)
		}
		val, err := parseByKind(def.Kind, args[2])
		if err != nil {
			return err
		}
		// ── دليلُ ما قبل ────────────────────────────────────────────
		fmt.Printf("PRE   %s = %s\n", key, beforeStr)
		// ── الكتابةُ عبر settings.Set وحدَها (تحقُّقُ الكتالوج) ─────────
		//
		// **و`updated_by` يبقى فارغاً (NULL)** — العمودُ مفتاحٌ أجنبيٌّ إلى
		// `users`، **ولا مستخدمَ لأداةِ تجهيز.** والنسبةُ في مخرَج الأداة
		// وسجلِّ العمل (`WORKLOG`)، لا في صفٍّ يكذب باسمِ مستخدمٍ لا وجودَ له.
		if err := store.Set(ctx, key, val, nil); err != nil {
			return fmt.Errorf("الكتابة: %w", err)
		}
		after, _ := store.GetRaw(ctx, key)
		fmt.Printf("POST  %s = %s\n", key, strings.TrimSpace(string(after)))
		// ── أمرُ الاستعادةِ الدقيق ───────────────────────────────────
		fmt.Printf("RESTORE  stagingctl flag set %s %s\n", key, restoreArg(def.Kind, before))
		return nil

	default:
		return fmt.Errorf("أمرٌ فرعيٌّ مجهول %q — get أو set", sub)
	}
}

// parseByKind يحوّل النصَّ إلى نوع المفتاح — **ويفشل مغلقاً على الغموض.**
func parseByKind(kind settings.Kind, raw string) (any, error) {
	switch kind {
	case settings.KindBool:
		b, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("قيمةٌ منطقيّةٌ متوقَّعة (true/false) لا %q", raw)
		}
		return b, nil
	case settings.KindInt:
		n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("قيمةٌ عدديّةٌ متوقَّعة لا %q", raw)
		}
		return n, nil
	case settings.KindText, settings.KindLongText, settings.KindChoice:
		return raw, nil
	default:
		return nil, fmt.Errorf("نوعُ المفتاح %q لا يُقلَب بهذه الأداة — رُفض", kind)
	}
}

// restoreArg يبني وسيطَ الاستعادةِ من القيمة القديمة الخام.
func restoreArg(kind settings.Kind, before json.RawMessage) string {
	s := strings.TrimSpace(string(before))
	if s == "" {
		// لا صفَّ سابق — الافتراضُ من الكتالوج يعود بحذف الصفّ يدويّاً؛
		// وللأعلام المنطقيّة الافتراضُ false.
		if kind == settings.KindBool {
			return "false"
		}
		return `""`
	}
	// نزعُ علامات الاقتباس من النصّ ليمرّ وسيطاً واحداً.
	if kind == settings.KindText || kind == settings.KindLongText || kind == settings.KindChoice {
		var str string
		if err := json.Unmarshal(before, &str); err == nil {
			return strconv.Quote(str)
		}
	}
	return s
}
