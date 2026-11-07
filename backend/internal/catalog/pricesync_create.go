package catalog

// ══════════════════════════════════════════════════════════════════════
// **إنشاءُ المواد غير الموجودة أصنافاً مخفيّة** (قرارُ المالك ٢٠٢٦-١١-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// ملفُّ مولٍ حقيقيّ فيه خمسةُ آلاف مادّة ولا صنفَ للمتجر. **فبخيارٍ صريحٍ (مُطفأٍ
// افتراضاً)** تصير كلُّ مادّةٍ لا مقابلَ لها صنفاً جديداً:
//
//   - **مخفيّاً**: `available = false` في قسمٍ غيرِ فعّالٍ اسمُه «من الأمين — بانتظار
//     التصنيف». **والزبونُ يرى الأقسامَ الفعّالةَ وحدَها**، فلا يظهر حتى يضيف المتجرُ
//     صورتَه وينقله إلى قسمٍ حقيقيّ ويُفعّله.
//     **ومن ٢٠٢٦-١١-٠٧ قسم الانتظار آخر حلّ بس**: الصنف بيروح لقسم مجموعته بالأمين
//     أو لقسم من اسمه أوّلاً (`pricesync_groups.go`).
//   - **برقم المادّة** — فرفعُ الملف ثانيةً يطابقه ولا يُنشئ شيئاً (idempotent).
//   - **وبالسعر كما تضعه المزامنةُ**: دولارٌ ⇒ `price_usd_cents` + الليرةُ بسعر الصرف.
//     **وسعرٌ صفرٌ يُنشأ ويبقى مخفيّاً ويُذكر في التقرير.**
//   - **دفعةً واحدةً** في معاملةٍ واحدة (`unnest`) — لا استعلامَ لكلّ صنف.

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// PendingSectionName **قسمُ الانتظار** — يُنشأ إن غاب، **وغيرَ فعّال**.
const PendingSectionName = "من الأمين — بانتظار التصنيف"

var junkPrefix = regexp.MustCompile(`^[\p{Nd}]+\s*-\s*`)

// CleanItemName **يشيل بادئةَ الرقم الزائفة** «017057-طاقة…» ⇒ «طاقة…».
//
// **فقط إن تبعها حرف** — «3ب1 ارمغان» لا شَرطةَ فيه فيبقى، و«2-3 قطع» تبقى (يتبعها رقم).
// ويُقصّ الاسمُ على حدّ الصنف (١٢٠ حرفاً).
func CleanItemName(s string) string {
	s = strings.TrimSpace(s)
	if loc := junkPrefix.FindStringIndex(s); loc != nil {
		rest := s[loc[1]:]
		if r, _ := utf8.DecodeRuneInString(rest); unicode.IsLetter(r) {
			s = strings.TrimSpace(rest)
		}
	}
	if utf8.RuneCountInString(s) > MaxItemName {
		s = string([]rune(s)[:MaxItemName])
	}
	return s
}

// ensurePendingSection **معرّفُ قسم الانتظار** — يُنشأ غيرَ فعّالٍ إن لم يوجد.
func ensurePendingSection(ctx context.Context, q dbtx.Querier) (string, error) {
	find := `SELECT id::text FROM platform_sections
	          WHERE section_name_key(name) = section_name_key($1) ORDER BY active DESC LIMIT 1`
	var id string
	err := q.QueryRow(ctx, find, PendingSectionName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = q.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order, active)
		VALUES ($1, 9999, false)
		ON CONFLICT DO NOTHING
		RETURNING id::text`, PendingSectionName).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = q.QueryRow(ctx, find, PendingSectionName).Scan(&id)
	}
	return id, err
}

// newItem **مادّةٌ ستصير صنفاً** — وقسمُها محلولٌ قبل الإدراج (`placeNewItems`).
type newItem struct {
	code  string
	name  string
	price int64
	cents *int64
	// group **مجموعةُ الأمين** كما وصلت (قد تكون فارغة).
	group string
	// section ومصدرُه: `pending` · `group` · `name` (٢٠٢٦-١١-٠٧).
	section string
	source  string
}

// insertNewItems **يُدرج الأصنافَ الجديدةَ دفعةً واحدة** ويردّ ما أُدرج فعلاً —
// **وتعارضٌ على الرقم (طلبٌ متزامن) يُتجاوز بلا خطأ.** وكلُّ صنفٍ بقسمه المحلول.
func insertNewItems(ctx context.Context, tx dbtx.Querier, merchantID string, items []newItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	names := make([]string, len(items))
	codes := make([]string, len(items))
	prices := make([]int64, len(items))
	cents := make([]*int64, len(items))
	secs := make([]string, len(items))
	sources := make([]string, len(items))
	groups := make([]string, len(items))
	for i, it := range items {
		names[i], codes[i], prices[i], cents[i] = it.name, it.code, it.price, it.cents
		secs[i], sources[i], groups[i] = it.section, it.source, it.group
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO menu_items (merchant_id, name, description, merchant_price, price,
		                        price_usd_cents, available, external_code,
		                        platform_section_id, sort_order, drinks_off,
		                        section_source, external_group)
		SELECT $1::uuid, t.name, '', t.price, t.price, t.cents, false, t.code, t.sec::uuid,
		       COALESCE((SELECT max(sort_order) FROM menu_items
		                  WHERE merchant_id = $1::uuid AND platform_section_id = t.sec::uuid), 0) + t.ord::int,
		       true, t.source, NULLIF(t.grp, '')
		  FROM unnest($2::text[], $3::text[], $4::bigint[], $5::bigint[], $6::text[], $7::text[], $8::text[])
		       WITH ORDINALITY AS t(name, code, price, cents, sec, source, grp, ord)
		ON CONFLICT (merchant_id, external_code)
		  WHERE external_code IS NOT NULL AND external_code <> '' DO NOTHING`,
		merchantID, names, codes, prices, cents, secs, sources, groups)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
