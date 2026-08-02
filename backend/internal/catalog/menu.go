package catalog

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// نموذج القائمة الكامل: أقسام ← أصناف ← مجموعات مُعدِّلات ← خيارات.

var ErrSectionNotEmpty = httpx.NewError(http.StatusConflict, "section_not_empty", "errors.section_not_empty")

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
	// MerchantPrice **سعرُ الشراء** — ما وضعه المتجر وهو ما يقبضه.
	//
	// **ولا يصل الزبون**: `handlePublicMerchant` تُسقطه. ويصل المتجرَ (هو
	// سعرُه) والأدمن (هو من يضع الهامش).
	MerchantPrice int64 `json:"merchant_price"`
	// MarginOverride تجاوزُ هامش الصنف — **فراغُه «اتبع تصنيفَك» لا «بلا هامش»**.
	MarginOverride *int64          `json:"margin_override"`
	ImageURL       *string         `json:"image_url"`
	ImageThumbURL  *string         `json:"image_thumb_url"`
	Available      bool            `json:"available"`
	SortOrder      int             `json:"sort_order"`
	Modifiers      []ModifierGroup `json:"modifiers"`
}

type MenuSection struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	Items     []MenuItem `json:"items"`
}

// GetMenu يعيد شجرة القائمة كاملة لمتجر.
func (s *Service) GetMenu(ctx context.Context, merchantID string) ([]MenuSection, error) {
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}

	sections := []MenuSection{}
	secIdx := map[string]int{}
	rows, err := s.db.Query(ctx, `
		SELECT id, name, sort_order FROM menu_sections
		WHERE merchant_id = $1 ORDER BY sort_order, created_at`, merchantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sec MenuSection
		if err := rows.Scan(&sec.ID, &sec.Name, &sec.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
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
	var catMargin *int64
	_ = s.db.QueryRow(ctx, `
		SELECT c.margin_override FROM merchants m
		LEFT JOIN categories c ON c.id = m.category_id WHERE m.id = $1`,
		merchantID).Scan(&catMargin)

	rows, err = s.db.Query(ctx, `
		SELECT i.id, i.section_id, i.name, i.description,
		       i.merchant_price, i.margin_override,
		       im.path, im.thumb_path, i.available, i.sort_order
		FROM menu_items i
		LEFT JOIN media im ON im.id = i.image_media_id
		WHERE i.merchant_id = $1 ORDER BY i.sort_order, i.created_at`, merchantID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it MenuItem
		if err := rows.Scan(&it.ID, &it.SectionID, &it.Name, &it.Description,
			&it.MerchantPrice, &it.MarginOverride,
			&it.ImageURL, &it.ImageThumbURL, &it.Available, &it.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		it.Price = rule.SalePrice(it.MerchantPrice, it.MarginOverride, catMargin)
		it.ImageURL = media.URLForPtr(it.ImageURL)
		it.ImageThumbURL = media.URLForPtr(it.ImageThumbURL)
		it.Modifiers = []ModifierGroup{}
		si, ok := secIdx[it.SectionID]
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

func (s *Service) CreateSection(ctx context.Context, actorID, merchantID, name string, ip string) (*MenuSection, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if _, err := s.merchantByID(ctx, merchantID); err != nil {
		return nil, err
	}
	var sec MenuSection
	err := s.db.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name, sort_order)
		VALUES ($1, $2, COALESCE((SELECT max(sort_order)+1 FROM menu_sections WHERE merchant_id=$1), 1))
		RETURNING id, name, sort_order`, merchantID, name).
		Scan(&sec.ID, &sec.Name, &sec.SortOrder)
	if err != nil {
		return nil, err
	}
	sec.Items = []MenuItem{}
	s.audit(ctx, actorID, "menu.section_create", "menu_section", sec.ID, ip)
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
	// Price **سعرُ الشراء** — ما وضعه المتجر وهو ما يقبضه.
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
	Available      *bool  `json:"available"`
	// معرف وسائط الصورة: غير مُرسل = بلا تغيير، "" = إزالة الصورة
	ImageMediaID *string `json:"image_media_id"`
	// إن أُرسلت (حتى فارغة) تُستبدل شجرة المُعدِّلات بالكامل
	Modifiers *[]ModifierGroupInput `json:"modifiers"`
}

func (s *Service) CreateItem(ctx context.Context, actorID, merchantID string, in MenuItemInput, ip string) (string, error) {
	if in.Name == nil || *in.Name == "" || in.SectionID == nil || in.Price == nil || *in.Price < 0 {
		return "", ErrNameRequired
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, name, description,
		                        merchant_price, price, image_media_id, sort_order)
		SELECT $1, $2, $3, COALESCE($4,''), $5, $5, NULLIF(COALESCE($6, ''), '')::uuid,
		       COALESCE((SELECT max(sort_order)+1 FROM menu_items WHERE section_id=$2), 1)
		WHERE EXISTS (SELECT 1 FROM menu_sections WHERE id = $2 AND merchant_id = $1)
		RETURNING id`,
		merchantID, *in.SectionID, *in.Name, in.Description, *in.Price, in.ImageMediaID).Scan(&id)
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

	tag, err := tx.Exec(ctx, `
		UPDATE menu_items SET
			section_id  = COALESCE($2, section_id),
			name        = COALESCE($3, name),
			description = COALESCE($4, description),
			-- **السعرُ المُرسَل سعرُ شراء** — وعمودُ price يتبعه كي لا يبقى
			-- القديمُ يحمل رقماً لا معنى له. **وسعرُ البيع يُحسب عند العرض**
			-- (حزمة pricing) فلا يُقرأ هذا العمودُ في مسارٍ يراه زبون.
			merchant_price = COALESCE($5, merchant_price),
			price          = COALESCE($5, price),
			-- **التجاوزُ يُمحى صراحةً بسالبِ واحد.**
			--
			-- COALESCE وحدَه لا يفرّق بين «لم يُرسَل» و«أُرسل فارغاً» — وكلاهما
			-- NULL. **فمن أراد أن يعيد صنفاً إلى وراثة تصنيفه لم يملك سبيلاً**:
			-- كلُّ إرسالٍ يُقرأ «بلا تغيير».
			margin_override = CASE WHEN $8::bigint IS NULL THEN margin_override
			                       WHEN $8 < 0 THEN NULL
			                       ELSE $8 END,
			available   = COALESCE($6, available),
			image_media_id = CASE WHEN $7::text IS NULL THEN image_media_id
			                      ELSE NULLIF($7, '')::uuid END,
			updated_at  = now()
		WHERE id = $1`,
		itemID, in.SectionID, in.Name, in.Description, in.Price, in.Available,
		in.ImageMediaID, in.MarginOverride)
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
