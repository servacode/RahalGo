package catalog

// ══════════════════════════════════════════════════════════════════════
// **رقمُ المادّة بالأمين — الربطُ اليدويّ** (قرارُ المالك ٢٠٢٦-١١-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// المزامنةُ تطابق بالرقم أوّلاً (`pricesync.go`)، **وصنفٌ أدخله المتجرُ بيده لا رقمَ له**
// فلا يُطابَق إلّا بالاسم إن اتّفق. **فيكتب صاحبُه رقمَ مادّته في نموذج الصنف** فيُربط.
//
// **ورقمٌ واحدٌ لصنفٍ واحدٍ في المتجر الواحد** — يحرسه فهرسٌ فريدٌ في القاعدة (هجرة
// 0459)، **والسؤالُ هنا قبل الكتابة ليُقال اسمُ الصنف الذي يحمله**: «هذا الرقم مربوط
// بصنف آخر: {name}». وطلبان متزامنان يمرّان من السؤال معاً فيردّ الفهرسُ الثاني.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// maxLinkLen **أطولُ رقمٍ أو باركود** بالحروف.
const maxLinkLen = 64

// ErrExternalCodeInvalid **رقمٌ أو باركودٌ أطولُ من الحدّ** (أو فيه فراغٌ داخليّ).
var ErrExternalCodeInvalid = httpx.NewError(http.StatusBadRequest,
	"external_code_invalid", "errors.external_code_invalid")

// errExternalCodeTaken **الرقمُ لصنفٍ آخر** — ويحمل اسمَه إن عُرف.
func errExternalCodeTaken(name string) error {
	e := &httpx.AppError{Status: http.StatusConflict, Code: "external_code_taken",
		MessageKey: "errors.external_code_taken"}
	if name != "" {
		e.Details = map[string]any{"name": name}
	}
	return e
}

// cleanLink **يشذّب الرقمَ المُرسَل** — `nil` يبقى `nil` (بلا تغيير)، والفارغُ فصلُ الربط.
// **وأرقامُه لاتينيّةٌ قبل الفحص والحفظ** (`linkText`).
func cleanLink(p *string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	v := linkText(*p)
	if !validLink(v) {
		return nil, ErrExternalCodeInvalid
	}
	return &v, nil
}

// changedLink **المُرسَلُ إن خالف المحفوظ** — وإلّا `nil` (بلا تغيير: لا فحصَ ولا كتابة).
//
// (مراجعةُ ما قبل الإصدار ٢٠٢٦-١١-٠٥.) **التطبيقُ والويبُ يرسلان الحقلين مع كلّ حفظ**،
// **ورقمٌ قديمٌ لا يقبله النموذج** (كتبته المزامنةُ بفراغٍ في وسطه) **كان يردّ كلَّ تعديلٍ
// على الصنف** — ولو غيّر صاحبُه السعرَ وحدَه.
func changedLink(p *string, stored string) *string {
	if p != nil && strings.TrimSpace(*p) == stored {
		return nil
	}
	return p
}

// linkText **الرقمُ كما يُقارَن ويُحفظ** — مشذَّباً، **وأرقامُه الهنديّةُ والفارسيّةُ لاتينيّة**
// و«٬» «,» و«٫» «.» (مرآةُ `WesternDigits.text` في التطبيق و`westernDigits` في الويب).
//
// (مراجعةُ ما قبل الإصدار ٢٠٢٦-١١-٠٥.) **الكيبوردُ العربيّ يكتب «٠١٧٠٦٤» وملفُّ الأمين
// «017064»** — فلو حُفظ كما كُتب لم يتطابقا أبداً. **وهو للنموذج وللمزامنة معاً.**
func linkText(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r == '٬':
			return ','
		case r == '٫':
			return '.'
		}
		return r
	}, strings.TrimSpace(s))
}

// validLink **أيُحفظ هذا الرقم؟** — بلا فراغٍ في وسطه ولا أطولَ من الحدّ. والفارغُ يصلح (فصل).
// **وقاعدتُه واحدةٌ للنموذج والمزامنة** — وإلّا كتبت المزامنةُ ما لا يُحفظ تعديلُه بعدها.
func validLink(v string) bool {
	return utf8.RuneCountInString(v) <= maxLinkLen && !strings.ContainsAny(v, " \t\r\n")
}

// ensureCodeFree **يسأل: أمربوطٌ هذا الرقمُ بصنفٍ غيرِ هذا في المتجر؟**
func ensureCodeFree(ctx context.Context, q dbtx.Querier, merchantID, itemID, code string) error {
	if code == "" {
		return nil
	}
	// **والمحفوظُ يُقارَن بأرقامٍ لاتينيّة** (`linkText`) — رقمٌ حُفظ «٠١٧٠٦٤» قبل التحويل
	// (٢٠٢٦-١١-٠٥) **هو «017064» نفسُه**، فلا يُربط بصنفٍ ثانٍ.
	var name string
	err := q.QueryRow(ctx, `
		SELECT name FROM menu_items
		 WHERE merchant_id = $1::uuid AND external_code IS NOT NULL AND external_code <> ''
		   AND translate(external_code, '٠١٢٣٤٥٦٧٨٩۰۱۲۳۴۵۶۷۸۹٬٫', '01234567890123456789,.') = $2
		   AND ($3::text = '' OR id::text <> $3)
		 LIMIT 1`, merchantID, code, itemID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return errExternalCodeTaken(name)
}

// mapCodeDup **يحوّل خرقَ الفهرس الفريد إلى الخطأ المصنَّف** — لطلبين متزامنين.
func mapCodeDup(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" && strings.Contains(pe.ConstraintName, "external_code") {
		return errExternalCodeTaken("")
	}
	return err
}
