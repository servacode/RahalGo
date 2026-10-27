package catalog

// ══════════════════════════════════════════════════════════════════════
// **خياراتُ الصنف الجاهزة** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — هجرة 0479
// ══════════════════════════════════════════════════════════════════════
//
// القائمةُ الكاملة من اللوحة، ولكلّ تصنيفٍ ما يطلع له افتراضاً، **والمتجرُ
// يختار لنفسه فيعلو اختيارُه على تصنيفه.** وفراغُ اختياره «اتبع تصنيفك».

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// PresetOption خيارٌ داخلَ القالب — بدايةٌ يغيّرها صاحبُ المتجر.
type PresetOption struct {
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
}

// OptionPreset **قالبُ مجموعة خيارات** — «النمرة» بخياراتها ٣٨…٤٥.
type OptionPreset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	MinSelect   int            `json:"min_select"`
	MaxSelect   int            `json:"max_select"`
	Options     []PresetOption `json:"options"`
	SortOrder   int            `json:"sort_order"`
	Active      bool           `json:"active"`
	CategoryIDs []string       `json:"category_ids"`
	// Chosen **أيطلع عند هذا المتجر؟** — يُملأ في قراءة المتجر وحدَها.
	Chosen bool `json:"chosen"`
}

// OptionPresetInput ما ترسله اللوحة عند الإنشاء والتعديل.
type OptionPresetInput struct {
	Name        string         `json:"name"`
	MinSelect   int            `json:"min_select"`
	MaxSelect   int            `json:"max_select"`
	Options     []PresetOption `json:"options"`
	SortOrder   *int           `json:"sort_order"`
	Active      *bool          `json:"active"`
	CategoryIDs *[]string      `json:"category_ids"`
}

var ErrPresetInvalid = httpx.NewError(http.StatusBadRequest, "preset_invalid", "errors.preset_invalid")

func (in *OptionPresetInput) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 40 {
		return ErrPresetInvalid
	}
	if in.MaxSelect < 1 {
		in.MaxSelect = 1
	}
	if in.MinSelect < 0 {
		in.MinSelect = 0
	}
	if in.MinSelect > in.MaxSelect {
		in.MinSelect = in.MaxSelect
	}
	opts := make([]PresetOption, 0, len(in.Options))
	for _, o := range in.Options {
		o.Name = strings.TrimSpace(o.Name)
		if o.Name == "" {
			continue
		}
		if utf8.RuneCountInString(o.Name) > 40 || o.PriceDelta < 0 {
			return ErrPresetInvalid
		}
		opts = append(opts, o)
	}
	in.Options = opts
	return nil
}

// AllOptionPresets **القائمةُ كاملةً للّوحة** — بالمعطَّلة وتصنيفاتِ كلٍّ.
func (s *Service) AllOptionPresets(ctx context.Context) ([]OptionPreset, error) {
	return s.readPresets(ctx, false, "")
}

// StoreOptionPresets **ما يراه المتجرُ** — الفعّالةُ كلُّها، وكلٌّ معلَّمٌ
// بأنّه يطلع عنده أو لا. و`custom` يعني أنّه اختار لنفسه.
func (s *Service) StoreOptionPresets(ctx context.Context, merchantID string) ([]OptionPreset, bool, error) {
	var custom bool
	if err := s.db.QueryRow(ctx,
		`SELECT option_preset_ids IS NOT NULL FROM merchants WHERE id = $1::uuid`, merchantID).
		Scan(&custom); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, httpx.ErrNotFound
		}
		return nil, false, err
	}
	out, err := s.readPresets(ctx, true, merchantID)
	return out, custom, err
}

func (s *Service) readPresets(ctx context.Context, activeOnly bool, merchantID string) ([]OptionPreset, error) {
	// **والمختارُ يُحسب في الاستعلام نفسِه**: اختيارُ المتجر إن وُجد،
	// وإلّا ربطُ تصنيفه.
	rows, err := s.db.Query(ctx, `
		SELECT p.id::text, p.name, p.min_select, p.max_select, p.options, p.sort_order, p.active,
		       COALESCE((SELECT array_agg(cop.category_id::text) FROM category_option_presets cop
		                  WHERE cop.preset_id = p.id), '{}'),
		       CASE WHEN $2::text = '' THEN false
		            ELSE COALESCE((
		              SELECT CASE WHEN m.option_preset_ids IS NOT NULL
		                          THEN p.id = ANY (m.option_preset_ids)
		                          ELSE EXISTS (SELECT 1 FROM category_option_presets c
		                                        WHERE c.preset_id = p.id AND c.category_id = m.category_id)
		                     END
		                FROM merchants m WHERE m.id = NULLIF($2, '')::uuid), false)
		       END
		  FROM option_presets p
		 WHERE (NOT $1::bool OR p.active)
		 ORDER BY p.sort_order, p.name`, activeOnly, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OptionPreset{}
	for rows.Next() {
		var p OptionPreset
		var raw []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.MinSelect, &p.MaxSelect, &raw, &p.SortOrder,
			&p.Active, &p.CategoryIDs, &p.Chosen); err != nil {
			return nil, err
		}
		p.Options = []PresetOption{}
		_ = json.Unmarshal(raw, &p.Options)
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetStoreOptionPresets **يستبدل اختيارَ المتجر** — و`nil` يعيده إلى تصنيفه.
func (s *Service) SetStoreOptionPresets(ctx context.Context, actorID, merchantID string, ids *[]string, ip string) error {
	var arg any
	if ids != nil {
		clean := []string{}
		for _, id := range *ids {
			if strings.TrimSpace(id) != "" {
				clean = append(clean, id)
			}
		}
		arg = clean
	}
	// **والمعرّفُ الشائخُ يُسقَط لا يُسقط الحفظ** — يُحفظ ما وُجد من القائمة.
	tag, err := s.db.Exec(ctx, `
		UPDATE merchants SET option_preset_ids = CASE WHEN $2::text[] IS NULL THEN NULL ELSE
		    ARRAY(SELECT p.id FROM option_presets p WHERE p.id::text = ANY ($2::text[]))
		  END, updated_at = now()
		 WHERE id = $1::uuid`, merchantID, arg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "menu.option_presets", "merchant", merchantID, ip)
	return nil
}

// SaveOptionPreset **ينشئ قالباً أو يعدّله** — و`id` فارغٌ يعني جديداً.
func (s *Service) SaveOptionPreset(ctx context.Context, actorID, id string, in OptionPresetInput, ip string) (string, error) {
	if err := in.clean(); err != nil {
		return "", err
	}
	raw, _ := json.Marshal(in.Options)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if id == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO option_presets (name, min_select, max_select, options, sort_order, active)
			VALUES ($1, $2, $3, $4, COALESCE($5, (SELECT COALESCE(max(sort_order), 0) + 1 FROM option_presets)),
			        COALESCE($6, true))
			RETURNING id::text`, in.Name, in.MinSelect, in.MaxSelect, raw, in.SortOrder, in.Active).Scan(&id)
	} else {
		var tag interface{ RowsAffected() int64 }
		tag, err = tx.Exec(ctx, `
			UPDATE option_presets SET name = $2, min_select = $3, max_select = $4, options = $5,
			       sort_order = COALESCE($6, sort_order), active = COALESCE($7, active)
			 WHERE id = $1::uuid`, id, in.Name, in.MinSelect, in.MaxSelect, raw, in.SortOrder, in.Active)
		if err == nil && tag.RowsAffected() == 0 {
			return "", httpx.ErrNotFound
		}
	}
	if err != nil {
		return "", err
	}
	if in.CategoryIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM category_option_presets WHERE preset_id = $1::uuid`, id); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_option_presets (category_id, preset_id)
			SELECT c.id, $1::uuid FROM categories c WHERE c.id::text = ANY ($2::text[])
			ON CONFLICT DO NOTHING`, id, *in.CategoryIDs); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	s.audit(ctx, actorID, "menu.option_preset_save", "option_preset", id, ip)
	return id, nil
}

// DeleteOptionPreset **حذفُ قالب** — والأصنافُ التي نُسخ إليها تبقى كما هي:
// القالبُ بدايةٌ تُنسخ لا رابطٌ حيّ.
func (s *Service) DeleteOptionPreset(ctx context.Context, actorID, id, ip string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM option_presets WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	_, _ = s.db.Exec(ctx, `
		UPDATE merchants SET option_preset_ids = array_remove(option_preset_ids, $1::uuid)
		 WHERE $1::uuid = ANY (option_preset_ids)`, id)
	s.audit(ctx, actorID, "menu.option_preset_delete", "option_preset", id, ip)
	return nil
}
