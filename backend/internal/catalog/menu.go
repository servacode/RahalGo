package catalog

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// نموذج القائمة الكامل: أقسام ← أصناف ← مجموعات مُعدِّلات ← خيارات.

var ErrSectionNotEmpty = httpx.NewError(http.StatusConflict, "section_not_empty", "errors.section_not_empty")

// ErrSectionRequired صنفٌ بلا قسمِ سوق — **ولا يراه زبونٌ فلا يُقبل.**
var ErrSectionRequired = httpx.NewError(http.StatusBadRequest, "section_required", "errors.section_required")

type ModifierOption struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
	Available  bool   `json:"available"`
	SortOrder  int    `json:"sort_order"`
}

type ModifierGroup struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	MinSelect int              `json:"min_select"`
	MaxSelect int              `json:"max_select"`
	SortOrder int              `json:"sort_order"`
	Options   []ModifierOption `json:"options"`
}

type MenuItem struct {
	ID          string `json:"id"`
	SectionID   string `json:"section_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Price **سعرُ البيع** — ما يدفعه الزبون. يُحسب من `MerchantPrice`
	// والهامش، **ولا يُقرأ من العمود المخزَّن**: الهامشُ إعدادٌ يملك المالكُ
	// تغييرَه، **ولو قُرئ المخزَّنُ لَبِيع بسعر الأمس حتى يُعاد حسابُ ألف صنف.**
	Price int64 `json:"price"`
	// MerchantPrice **سعرُ الشراء** — ما وضعه المتجر، **وليس ما يقبضه**:
	// تُخصم منه عمولةُ المنصة عند التسوية. **وهو أساسُ الحسبتين** — تنزل
	// عليه العمولةُ ويصعد فوقه الهامش.
	//
	// **ولا يصل الزبون**: `handlePublicMerchant` تُسقطه. ويصل المتجرَ (هو
	// سعرُه) والأدمن (هو من يضع الهامش).
	MerchantPrice int64 `json:"merchant_price"`
	// MarginOverride تجاوزُ هامش الصنف — **فراغُه «اتبع تصنيفَك» لا «بلا هامش»**.
	MarginOverride *int64 `json:"margin_override"`
	// SourceClosed مصدرُ الصنف خارجَ دوامه الآن.
	//
	// **والصنفُ لا يُطلب ممّن ينام.** `Available` علَمٌ يرفعه المتجرُ بيده
	// («نفد الصنف»)، **وهذا يقوله الوقتُ عنه** — ومن خلط بينهما جعل المتجرَ
	// يطفئ أصنافَه كلَّ ليلةٍ ويشعلها كلَّ صباح.
	SourceClosed bool `json:"source_closed"`
	// SourceOpensAt متى يعود.
	//
	// **قل متى يعود لا أنه غيرُ متاح**: «متاح من ١٠ صباحاً» موعدٌ يُعاد إليه،
	// و«غير متاح» طريقٌ مسدود.
	SourceOpensAt *time.Time `json:"source_opens_at"`
	// PlatformSectionID قسمُ المنصة الذي يُعرض فيه — **وفراغُه «غيرُ مصنَّف»**.
	//
	// **ولا يُعرض في التصفّح ما لم يُصنَّف**: يبقى قابلاً للطلب من صفحة متجره
	// فلا ينقطع ما كان يعمل، **ويراه الأدمنُ في اللوحة فارغاً فيصنّفه.**
	PlatformSectionID   *string `json:"platform_section_id"`
	PlatformSectionName string  `json:"platform_section_name"`
	ImageURL            *string `json:"image_url"`
	ImageThumbURL       *string `json:"image_thumb_url"`
	Available           bool    `json:"available"`
	// Approved أنُشر الصنفُ للزبائن؟ — **حين يُرفع مفتاحُ مراجعة القائمة.**
	//
	// **ويراه المتجرُ معلّقاً لا مختفياً**: من أضاف صنفاً فلم يجده في قائمته
	// يضيفه ثانيةً وثالثة، **فيمتلئ الطابورُ بنسخٍ من الشيء الواحد.**
	Approved bool `json:"approved"`
	// ReviewNote سببُ الردّ — **يُقال لصاحبه.**
	//
	// «رُدّ» بلا كلمةٍ يُعاد إرسالُه كما هو، **فيدور المتجرُ والمكتبُ في حلقة.**
	ReviewNote string          `json:"review_note"`
	SortOrder  int             `json:"sort_order"`
	Modifiers  []ModifierGroup `json:"modifiers"`
}

type MenuSection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// ImageURL وجهُ القسم — **وسوقٌ يُتصفَّح بالصور لا بالأسماء.**
	//
	// (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «أساساً القسم لازم يكون له صورةٌ واسم».)
	//
	// **وفارغٌ لا يكسر شيئاً**: القسمُ يُعرض باسمه كما كان — وكلُّ قسمٍ قائمٍ
	// في القاعدة اليومَ بلا صورة.
	ImageURL      *string    `json:"image_url"`
	ImageThumbURL *string    `json:"image_thumb_url"`
	SortOrder     int        `json:"sort_order"`
	Items         []MenuItem `json:"items"`
}

// SectionInput ما يُرسَل عند الإنشاء أو التعديل.
//
// **و`ImageMediaID` ثلاثيُّ الحال**: غائبٌ يعني «لا تمسّ الصورة»، وفارغٌ
// يعني «أزِلها»، وقيمةٌ تعني «هذه». **ومن خلط الغائبَ بالفارغ محا صورةَ
// قسمٍ كلَّما بُدّل اسمُه.**
type SectionInput struct {
	Name         string  `json:"name"`
	ImageMediaID *string `json:"image_media_id"`
}

// GetMenu يعيد شجرة القائمة كاملة لمتجر.
func (s *Service) GetMenu(ctx context.Context, merchantID string) ([]MenuSection, error) {
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}

	sections := []MenuSection{}
	secIdx := map[string]int{}
	// ══════════════════════════════════════════════════════════════════
	// **الأقسامُ تُشتقّ من الأصناف — لا من جدولٍ يملكه المتجر**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الأدمنُ هو من يزرع الأقسام، والمتجرُ يجد
	//  أقساماً جاهزة… لأنّ بكرا كلّ متجرٍ رح ينزّل اسمَ قسمٍ مختلفٍ وصورةً
	//  مختلفة».)
	//
	// **فلا جدولَ يُصان ولا شاشةَ تُفتح ولا اختيارَ يُخزَّن**: القسمُ يظهر
	// لأنّ فيه صنفاً، **ويختفي حين يخرج آخرُ صنفٍ منه.** ومطعمٌ لا يبيع
	// بقالةً لا يرى «بقالة» — بلا أن يقول ذلك لأحد.
	//
	// **والاسمُ والصورةُ والترتيبُ من المنصة** — فما يكتبه الأدمنُ مرّةً
	// يراه المتاجرُ كلُّهم والزبونُ سواءً.
	rows, err := s.db.Query(ctx, `
		SELECT ps.id, ps.name, sm.path, sm.thumb_path, ps.sort_order
		FROM platform_sections ps
		LEFT JOIN media sm ON sm.id = ps.image_media_id
		WHERE EXISTS (
			SELECT 1 FROM menu_items i
			WHERE i.merchant_id = $1 AND i.platform_section_id = ps.id
		)
		ORDER BY ps.sort_order, ps.name`, merchantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sec MenuSection
		if err := rows.Scan(&sec.ID, &sec.Name, &sec.ImageURL, &sec.ImageThumbURL, &sec.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		// **ومسارُ الصورة يُبادَأ بـ/media/ كما تُبادَأ مساراتُ الأصناف.**
		//
		// (كشفه المالك 2026-08-08 بلقطةِ لوحة: صورةٌ مكسورةٌ في القائمة.)
		//
		// **كان يخرج مفتاحَ تخزينٍ عارياً** — 2026/08/x.png — **فلا هو رابطٌ
		// ولا مسارُ خادم**: يُطلب من منفذ اللوحة فيردّ 404، وحتى لو حُوّل
		// بـmediaUrl خرج http://…:8080/2026/08/… وردّ 404 كذلك. **والأصنافُ
		// في هذه الدالّة نفسِها تُبادَأ** — فالسطران فاتا لا القاعدة.
		sec.ImageURL = media.URLForPtr(sec.ImageURL)
		sec.ImageThumbURL = media.URLForPtr(sec.ImageThumbURL)
		sec.Items = []MenuItem{}
		secIdx[sec.ID] = len(sections)
		sections = append(sections, sec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	itemIdx := map[string][2]int{} // itemID → (sectionIdx, itemIdx)
	// **تجاوزُ التصنيف يُقرأ مع الصنف** — فالحسبةُ في Go لا في SQL.
	//
	// **ولو حُسب السعرُ في الاستعلام لَصارت المعادلةُ في موضعين**: هنا وفي
	// `priceItems` عند الطلب. **ورقمان لمعنًى واحد يفترقان** — فيرى الزبونُ
	// سعراً في القائمة ويُحاسَب بغيره في السلّة.
	rule := pricing.RuleFrom(ctx, s.settings)
	// **ولا هامشَ واحدٌ للقائمة كلِّها.**
	//
	// أصنافُ متجرٍ واحدٍ تقع في أقسامٍ مختلفة — شاورما وعصير في مطعمٍ واحد —
	// **فقراءةُ قسمٍ واحدٍ للقائمة كلِّها تُسعّر العصيرَ بهامش الشاورما.**
	// ويُقرأ مع كلّ صنفٍ في الاستعلام نفسه.
	// **ودوامُ المصدر يُقرأ مرّةً للقائمة كلِّها** — كلُّ أصنافها من متجرٍ
	// واحد، **وسؤالُ القاعدة لكلّ صنفٍ عن الشيء نفسِه مئةُ استعلامٍ بلا سبب.**
	var open bool
	var opensAt *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT `+orders.OpenNowSQL+`, `+orders.NextOpenSQL+`
		FROM merchants m WHERE m.id = $1`,
		merchantID).Scan(&open, &opensAt)

	// **والعمودُ الميّتُ يُقرأ فارغاً لا يُسقط الصفحة.**
	//
	// (كشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨ على قاعدةٍ نظيفة — انظر `menu_read_test.go`.)
	//
	// هجرةُ ٠٠٨٤ رفعت طبقةَ أقسام المتجر و`CreateItem` لم تعد تكتب
	// `section_id`، **وهذه بقيت تقرؤه في حقلٍ لا يقبل الفراغ** — فأوّلُ صنفٍ
	// يُنشئه متجرٌ اليومَ يجعل قائمتَه كلَّها ٥٠٠.
	//
	// **والصفوفُ القديمةُ كلُّها مملوءة** فلم يسقط قطّ على قاعدةٍ مبذورة.
	rows, err = s.db.Query(ctx, `
		SELECT i.id, COALESCE(i.section_id::text, ''), i.name, i.description,
		       i.merchant_price, i.margin_override, ps.margin_override,
		       i.platform_section_id, COALESCE(ps.name, ''),
		       im.path, im.thumb_path, i.available, i.approved, i.review_note, i.sort_order
		FROM menu_items i
		LEFT JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media im ON im.id = i.image_media_id
		WHERE i.merchant_id = $1 ORDER BY i.sort_order, i.created_at`, merchantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it MenuItem
		var sectionMargin *int64
		if err := rows.Scan(&it.ID, &it.SectionID, &it.Name, &it.Description,
			&it.MerchantPrice, &it.MarginOverride, &sectionMargin,
			&it.PlatformSectionID, &it.PlatformSectionName,
			&it.ImageURL, &it.ImageThumbURL, &it.Available, &it.Approved, &it.ReviewNote, &it.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		it.Price = rule.SalePrice(it.MerchantPrice, it.MarginOverride, sectionMargin)
		it.SourceClosed = !open
		if !open {
			it.SourceOpensAt = opensAt
		}
		it.ImageURL = media.URLForPtr(it.ImageURL)
		it.ImageThumbURL = media.URLForPtr(it.ImageThumbURL)
		it.Modifiers = []ModifierGroup{}
		// **والانتماءُ بقسم السوق** — و`section_id` القديمُ يبقى في الصفّ
		// ولا يُقرأ. **وصنفٌ بلا قسمِ سوقٍ لا يظهر في القائمة**: هو حالةٌ
		// لا تقع اليومَ (صفرٌ من ١٩٤)، **وإظهارُه بلا رأسٍ يُقرأ عطباً.**
		key := ""
		if it.PlatformSectionID != nil {
			key = *it.PlatformSectionID
		}
		si, ok := secIdx[key]
		if !ok {
			continue
		}
		itemIdx[it.ID] = [2]int{si, len(sections[si].Items)}
		sections[si].Items = append(sections[si].Items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	groupIdx := map[string][3]int{} // groupID → (sectionIdx, itemIdx, groupIdx)
	rows, err = s.db.Query(ctx, `
		SELECT g.id, g.item_id, g.name, g.min_select, g.max_select, g.sort_order
		FROM modifier_groups g
		JOIN menu_items i ON i.id = g.item_id
		WHERE i.merchant_id = $1 ORDER BY g.sort_order`, merchantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var g ModifierGroup
		var itemID string
		if err := rows.Scan(&g.ID, &itemID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		g.Options = []ModifierOption{}
		loc, ok := itemIdx[itemID]
		if !ok {
			continue
		}
		item := &sections[loc[0]].Items[loc[1]]
		groupIdx[g.ID] = [3]int{loc[0], loc[1], len(item.Modifiers)}
		item.Modifiers = append(item.Modifiers, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.Query(ctx, `
		SELECT o.id, o.group_id, o.name, o.price_delta, o.available, o.sort_order
		FROM modifier_options o
		JOIN modifier_groups g ON g.id = o.group_id
		JOIN menu_items i ON i.id = g.item_id
		WHERE i.merchant_id = $1 ORDER BY o.sort_order`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o ModifierOption
		var groupID string
		if err := rows.Scan(&o.ID, &groupID, &o.Name, &o.PriceDelta, &o.Available, &o.SortOrder); err != nil {
			return nil, err
		}
		loc, ok := groupIdx[groupID]
		if !ok {
			continue
		}
		g := &sections[loc[0]].Items[loc[1]].Modifiers[loc[2]]
		g.Options = append(g.Options, o)
	}
	return sections, rows.Err()
}

// ---------- الأقسام ----------

func (s *Service) CreateSection(ctx context.Context, actorID, merchantID string, in SectionInput, ip string) (*MenuSection, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}
	var sec MenuSection
	err := s.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO menu_sections (merchant_id, name, sort_order, image_media_id)
			VALUES ($1, $2,
				COALESCE((SELECT max(sort_order)+1 FROM menu_sections WHERE merchant_id=$1), 1),
				NULLIF($3, '')::uuid)
			RETURNING id, name, sort_order, image_media_id
		)
		SELECT ins.id, ins.name, sm.path, sm.thumb_path, ins.sort_order
		FROM ins LEFT JOIN media sm ON sm.id = ins.image_media_id`,
		merchantID, in.Name, derefOr(in.ImageMediaID, "")).
		Scan(&sec.ID, &sec.Name, &sec.ImageURL, &sec.ImageThumbURL, &sec.SortOrder)
	if err != nil {
		return nil, err
	}
	sec.ImageURL = media.URLForPtr(sec.ImageURL)
	sec.ImageThumbURL = media.URLForPtr(sec.ImageThumbURL)
	sec.Items = []MenuItem{}
	s.audit(ctx, actorID, "menu.section_create", "menu_section", sec.ID, ip)
	return &sec, nil
}

// UpdateSection يبدّل اسمَ القسم أو صورتَه.
//
// **وكان القسمُ يُنشأ ويُحذف ولا يُعدَّل** — فمن أخطأ حرفاً في اسمه حذفه
// وأعاد بناءه، **وأصنافُه تمنع الحذف** (`ErrSectionNotEmpty`)، فيبقى الخطأُ
// معروضاً على زبائنه.
//
// **والصورةُ كشفت النقص**: قسمٌ قائمٌ لا سبيلَ لإعطائه وجهاً.
func (s *Service) UpdateSection(ctx context.Context, actorID, sectionID string, in SectionInput, ip string) (*MenuSection, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	var sec MenuSection
	err := s.db.QueryRow(ctx, `
		WITH upd AS (
			UPDATE menu_sections SET
				name = $2,
				-- **والغائبُ لا يمسّ الصورة**: تبديلُ الاسم وحدَه لا يمحو وجهَ القسم.
				image_media_id = CASE
					WHEN $3::text IS NULL THEN image_media_id
					WHEN $3 = '' THEN NULL
					ELSE $3::uuid END
			WHERE id = $1
			RETURNING id, name, sort_order, image_media_id
		)
		SELECT upd.id, upd.name, sm.path, sm.thumb_path, upd.sort_order
		FROM upd LEFT JOIN media sm ON sm.id = upd.image_media_id`,
		sectionID, in.Name, in.ImageMediaID).
		Scan(&sec.ID, &sec.Name, &sec.ImageURL, &sec.ImageThumbURL, &sec.SortOrder)
	if err != nil {
		return nil, err
	}
	sec.ImageURL = media.URLForPtr(sec.ImageURL)
	sec.ImageThumbURL = media.URLForPtr(sec.ImageThumbURL)
	sec.Items = []MenuItem{}
	s.audit(ctx, actorID, "menu.section_update", "menu_section", sec.ID, ip)
	return &sec, nil
}

func (s *Service) DeleteSection(ctx context.Context, actorID, sectionID, ip string) error {
	var count int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE section_id = $1`, sectionID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrSectionNotEmpty
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM menu_sections WHERE id = $1`, sectionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "menu.section_delete", "menu_section", sectionID, ip)
	return nil
}

// ---------- الأصناف (مع شجرة المُعدِّلات في طلب واحد) ----------

type ModifierOptionInput struct {
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
}

type ModifierGroupInput struct {
	Name      string                `json:"name"`
	MinSelect int                   `json:"min_select"`
	MaxSelect int                   `json:"max_select"`
	Options   []ModifierOptionInput `json:"options"`
}

type MenuItemInput struct {
	SectionID   *string `json:"section_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	// Price **سعرُ الشراء** — ما وضعه المتجر، **وليس ما يقبضه**: تُخصم منه
	// عمولةُ المنصة عند التسوية.
	//
	// **والاسمُ يبقى `price` عمداً**: هو ما تُرسله شاشةُ المتجر منذ البداية،
	// **وتغييرُ اسمِ حقلٍ في الواجهة يكسر كلَّ شاشةٍ لم تُحدَّث بعد** — فيُرسل
	// المتجرُ سعراً ويُقرأ فارغاً، **فيصير كلُّ صنفٍ بصفر.**
	//
	// وسعرُ البيع لا يُرسَل أصلاً: **تحسبه المنصةُ ولا يملك المتجرُ وضعَه.**
	Price *int64 `json:"price"`
	// MarginOverride تجاوزُ هامش الصنف — **للأدمن لا للمتجر**.
	//
	// **وفراغُه «اتبع تصنيفَك» لا «بلا هامش»**: مؤشّرٌ لا رقم، فالصفرُ قرارٌ
	// يُفرَّق عن غياب القرار.
	MarginOverride *int64 `json:"margin_override"`
	// PlatformSectionID قسمُ المنصة — **وسالبُ الواحدِ لا يصلح هنا**: هو
	// معرّفٌ نصّي، **والفراغُ الصريح `""` يعني «ارفع التصنيف».**
	PlatformSectionID *string `json:"platform_section_id"`
	Available         *bool   `json:"available"`
	// معرف وسائط الصورة: غير مُرسل = بلا تغيير، "" = إزالة الصورة
	ImageMediaID *string `json:"image_media_id"`
	// إن أُرسلت (حتى فارغة) تُستبدل شجرة المُعدِّلات بالكامل
	Modifiers *[]ModifierGroupInput `json:"modifiers"`
}

func (s *Service) CreateItem(ctx context.Context, actorID, merchantID string, in MenuItemInput, ip string) (string, error) {
	// **وقسمُ السوق شرطٌ لا اختيار.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ الأصناف ستذهب إلى السوق بلوحة الأدمن
	//  بشكلٍ مباشر».)
	//
	// **كان `section_id` هو الشرط** — قسمُ المتجر الذي يكتبه بيده. **وصنفٌ
	// بلا قسمِ سوقٍ لا يراه زبون**، فيبقى في القائمة ولا يُباع.
	if in.Name == nil || *in.Name == "" || in.Price == nil || *in.Price < 0 {
		return "", ErrNameRequired
	}
	if in.PlatformSectionID == nil || *in.PlatformSectionID == "" {
		return "", ErrSectionRequired
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **وقسمُ السوق يُكتب هنا كما يُكتب في التعديل.**
	//
	// كان الحقلُ يُرسَل من النافذة ولا يُذكر في الإدخال، **فيولد كلُّ صنفٍ خارج
	// السوق**: لا يراه الزبونُ ولا يُعَدُّ في قسمه، **حتى يفتحه أحدٌ ويحفظه
	// ثانيةً بلا تغيير** — فالتعديلُ وحدَه كان يكتبه.
	//
	// **وهي عائلةُ الخلل نفسُها**: قاعدةٌ مكتوبةٌ في موضعين افترقت بلا صوت.
	// **ولا `section_id` يُكتب** — العمودُ يبقى في الجدول للصفوف القديمة،
	// **ولا يُملأ لصفٍّ جديد.** (هجرة ٠٠٨٤ رفعت عنه شرطَ الامتلاء.)
	//
	// **والحارسُ صار على قسم السوق**: يوجد وفعّال. وكان على قسم المتجر —
	// **أي أنّ صاحبَ متجرٍ لا يستطيع أن يدسّ صنفاً في قسمِ متجرٍ آخر**،
	// وهذا يبقى محروساً بـ`merchant_id` في الصفّ نفسِه.
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, name, description,
		                        merchant_price, price, image_media_id, sort_order,
		                        platform_section_id)
		SELECT $1, $2, COALESCE($3,''), $4, $4, NULLIF(COALESCE($5, ''), '')::uuid,
		       COALESCE((SELECT max(sort_order)+1 FROM menu_items
		                 WHERE merchant_id=$1 AND platform_section_id=$6::uuid), 1),
		       $6::uuid
		WHERE EXISTS (SELECT 1 FROM platform_sections WHERE id = $6::uuid AND active)
		RETURNING id`,
		merchantID, *in.Name, in.Description, *in.Price, in.ImageMediaID,
		*in.PlatformSectionID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if in.Modifiers != nil {
		if err := insertModifiers(ctx, tx, id, *in.Modifiers); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	s.audit(ctx, actorID, "menu.item_create", "menu_item", id, ip)
	return id, nil
}

func (s *Service) UpdateItem(ctx context.Context, actorID, itemID string, in MenuItemInput, ip string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ══════════════════════════════════════════════════════════════════
	// **ووسيطٌ يُمرَّر ولا يُذكر يُسقط الاستعلام كلَّه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك 2026-08-09: «عند إضافة صورة لصنف يظهر خطأ غير متوقّع».)
	//
	// **وليست الصورةُ سببَه**: أيُّ تعديلِ صنفٍ كان يسقط —
	// `could not determine data type of parameter $2` (SQLSTATE 42P18).
	//
	// **كان `in.SectionID` يُمرَّر ثانيَ الوسائط** من يومَ كان العمودُ
	// `section_id` يُبدَّل. **ثمّ رُفع السطرُ من `SET`** (هجرة 0084: الانتماءُ
	// صار بقسم السوق) **وبقي الوسيطُ في النداء** — فلم يعد `$2` مذكوراً في
	// النصّ، **وبوستغرس يستنبط أنواعَ الوسائط من مواضع ذكرها**: ما لا يُذكر
	// لا يُعرف نوعُه، فيرفض الاستعلامَ قبل أن ينفّذه.
	//
	// **ولا يكشفه بناءٌ ولا مترجِم**: الوسائطُ `...any` — **يسقط في التشغيل
	// وحدَه**، وعند كلّ تعديلِ صنفٍ بلا استثناء.
	//
	// **والترقيمُ أُنقص واحداً** بعد نزعه: الفجوةُ في الترقيم عطبٌ آخرُ من
	// العائلة نفسِها.
	tag, err := tx.Exec(ctx, `
		UPDATE menu_items SET
			-- **ولا عمودُ قسمِ المتجر يُبدَّل** — طبقتُه ذهبت (هجرة ٠٠٨٤)،
			-- **والانتماءُ صار بقسم السوق وحدَه.** والعمودُ يبقى للصفوف القديمة.
			name        = COALESCE($2, name),
			description = COALESCE($3, description),
			-- **السعرُ المُرسَل سعرُ شراء** — وعمودُ price يتبعه كي لا يبقى
			-- القديمُ يحمل رقماً لا معنى له. **وسعرُ البيع يُحسب عند العرض**
			-- (حزمة pricing) فلا يُقرأ هذا العمودُ في مسارٍ يراه زبون.
			merchant_price = COALESCE($4, merchant_price),
			price          = COALESCE($4, price),
			-- **التجاوزُ يُمحى صراحةً بسالبِ واحد.**
			--
			-- COALESCE وحدَه لا يفرّق بين «لم يُرسَل» و«أُرسل فارغاً» — وكلاهما
			-- NULL. **فمن أراد أن يعيد صنفاً إلى وراثة تصنيفه لم يملك سبيلاً**:
			-- كلُّ إرسالٍ يُقرأ «بلا تغيير».
			margin_override = CASE WHEN $7::bigint IS NULL THEN margin_override
			                       WHEN $7 < 0 THEN NULL
			                       ELSE $7 END,
			available   = COALESCE($5, available),
			image_media_id = CASE WHEN $6::text IS NULL THEN image_media_id
			                      ELSE NULLIF($6, '')::uuid END,
			-- **والفراغُ الصريح يرفع التصنيف** — لا يُقرأ «بلا تغيير».
			platform_section_id = CASE WHEN $8::text IS NULL THEN platform_section_id
			                           ELSE NULLIF($8, '')::uuid END,
			updated_at  = now()
		WHERE id = $1`,
		itemID, in.Name, in.Description, in.Price, in.Available,
		in.ImageMediaID, in.MarginOverride, in.PlatformSectionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	if in.Modifiers != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM modifier_groups WHERE item_id = $1`, itemID); err != nil {
			return err
		}
		if err := insertModifiers(ctx, tx, itemID, *in.Modifiers); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.audit(ctx, actorID, "menu.item_update", "menu_item", itemID, ip)
	return nil
}

func (s *Service) DeleteItem(ctx context.Context, actorID, itemID, ip string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "menu.item_delete", "menu_item", itemID, ip)
	return nil
}

func insertModifiers(ctx context.Context, tx pgx.Tx, itemID string, groups []ModifierGroupInput) error {
	for gi, g := range groups {
		if g.Name == "" {
			return ErrNameRequired
		}
		if g.MaxSelect < 1 {
			g.MaxSelect = 1
		}
		if g.MinSelect < 0 {
			g.MinSelect = 0
		}
		if g.MinSelect > g.MaxSelect {
			g.MinSelect = g.MaxSelect
		}
		var groupID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO modifier_groups (item_id, name, min_select, max_select, sort_order)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			itemID, g.Name, g.MinSelect, g.MaxSelect, gi).Scan(&groupID); err != nil {
			return err
		}
		for oi, o := range g.Options {
			if o.Name == "" {
				return ErrNameRequired
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO modifier_options (group_id, name, price_delta, sort_order)
				VALUES ($1, $2, $3, $4)`, groupID, o.Name, o.PriceDelta, oi); err != nil {
				return err
			}
		}
	}
	return nil
}

// ItemModifiers خياراتُ صنفٍ واحد — لصفحة الصنف في تصفّح الأقسام.
//
// **ولا تُنتزع من `GetMenu`.** تلك تقرأ قائمةَ متجرٍ كاملة بثلاثة استعلامات
// ثمّ تربطها، **وقراءةُ قائمةٍ كاملةٍ لعرض صنفٍ واحدٍ حملٌ بلا حاجة** — ومئةُ
// صنفٍ تُقرأ ليُعرض واحد.
func (s *Service) ItemModifiers(ctx context.Context, itemID string) ([]ModifierGroup, error) {
	groups := []ModifierGroup{}
	idx := map[string]int{}
	rows, err := s.db.Query(ctx, `
		SELECT id, name, min_select, max_select, sort_order
		FROM modifier_groups WHERE item_id = $1 ORDER BY sort_order`, itemID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var g ModifierGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.MinSelect, &g.MaxSelect, &g.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		g.Options = []ModifierOption{}
		idx[g.ID] = len(groups)
		groups = append(groups, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}

	rows, err = s.db.Query(ctx, `
		SELECT o.id, o.group_id, o.name, o.price_delta, o.available, o.sort_order
		FROM modifier_options o
		JOIN modifier_groups g ON g.id = o.group_id
		WHERE g.item_id = $1 ORDER BY o.sort_order`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o ModifierOption
		var groupID string
		if err := rows.Scan(&o.ID, &groupID, &o.Name, &o.PriceDelta, &o.Available, &o.SortOrder); err != nil {
			return nil, err
		}
		if i, ok := idx[groupID]; ok {
			groups[i].Options = append(groups[i].Options, o)
		}
	}
	return groups, rows.Err()
}

// derefOr قيمةُ المؤشّر أو بديلُها — **والفارغُ يعني «لا صورة» لا «خطأ».**
func derefOr(p *string, alt string) string {
	if p == nil {
		return alt
	}
	return *p
}
