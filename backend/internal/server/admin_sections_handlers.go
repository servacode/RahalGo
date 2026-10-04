package server

// أقسامُ المنصة — إدارتُها من اللوحة.
//
// **والأسماءُ تحمل الفرق.** `handleCreateSection` كانت موجودةً لأقسام قائمة
// المتجر، **ومعالِجان باسمٍ واحدٍ لمعنيين مختلفين يُنادى أحدُهما مكان الآخر**
// — فيُنشأ قسمُ منصةٍ حيث أُريد قسمُ قائمة، ولا يظهر الخطأ إلّا في الشاشة.
//
// **والقسمُ غيرُ التصنيف**: `categories` تصف **من نشتري منه** (مطاعم · بقالة)،
// و`platform_sections` تصف **ما نبيعه** (شاورما · بيتزا · خضار). **والزبونُ
// يرى الثاني وحدَه.**

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

type adminPlatformSection struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Icon           string `json:"icon"`
	SortOrder      int    `json:"sort_order"`
	Active         bool   `json:"active"`
	MarginOverride *int64 `json:"margin_override"`
	// Items عددُ الأصناف المربوطة — **لتُعرف قيمةُ القسم قبل حذفه.**
	Items int `json:"items"`
	// ImageMediaID صورةُ القسم — **وجهُه في السوق.**
	//
	// **والأيقونةُ تبقى بجانبها**: هي ما يُعرض قبل أن تُرفع صورة، وفي المواضع
	// الضيّقة. **وقسمٌ بلا صورةٍ لا يظهر فارغاً.**
	ImageMediaID  *string `json:"image_media_id"`
	ImageURL      *string `json:"image_url"`
	ImageThumbURL *string `json:"image_thumb_url"`
}

// handleListPlatformSections الأقسامُ كلُّها — الفعّالةُ والمُطفَأة.
func (s *Server) handleListPlatformSections(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT ps.id, ps.name, ps.icon, ps.sort_order, ps.active, ps.margin_override,
		       count(i.id), ps.image_media_id::text, sm.path, sm.thumb_path
		FROM platform_sections ps
		LEFT JOIN menu_items i ON i.platform_section_id = ps.id
		LEFT JOIN media sm ON sm.id = ps.image_media_id
		GROUP BY ps.id, sm.path, sm.thumb_path
		ORDER BY ps.sort_order, ps.name`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []adminPlatformSection{}
	for rows.Next() {
		var x adminPlatformSection
		if err := rows.Scan(&x.ID, &x.Name, &x.Icon, &x.SortOrder, &x.Active,
			&x.MarginOverride, &x.Items, &x.ImageMediaID, &x.ImageURL, &x.ImageThumbURL); err != nil {
			s.respondErr(w, err)
			return
		}
		x.ImageURL = media.URLForPtr(x.ImageURL)
		x.ImageThumbURL = media.URLForPtr(x.ImageThumbURL)
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sections": out})
}

type platformSectionInput struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	SortOrder *int    `json:"sort_order"`
	Active    *bool   `json:"active"`
	// MarginOverride هامشُ القسم — **وسالبُ واحدٍ يمحوه**.
	//
	// `COALESCE` وحدَه لا يفرّق بين «لم يُرسَل» و«أُرسل فارغاً» — وكلاهما
	// `NULL`. **فمن أراد أن يعيد قسماً إلى الهامش العامّ لم يملك سبيلاً.**
	MarginOverride *int64 `json:"margin_override"`
	// ImageMediaID صورةُ القسم — **وجهُه في السوق**، والفراغُ الصريحُ يرفعها.
	ImageMediaID *string `json:"image_media_id"`
}

func (s *Server) handleCreatePlatformSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[platformSectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		s.respondErr(w, errValidation)
		return
	}
	icon := ""
	if req.Icon != nil {
		icon = *req.Icon
	}
	// **والقسمُ الجديدُ آخرَ القائمة** — لا أوّلَها بصفرٍ يزاحم «شاورما».
	sort := -1
	if req.SortOrder != nil {
		sort = *req.SortOrder
	}
	name := strings.TrimSpace(*req.Name)
	if err := s.sectionNameFree(r, name, ""); err != nil {
		s.respondErr(w, err)
		return
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO platform_sections (name, icon, sort_order, margin_override, image_media_id)
		VALUES ($1, $2,
		        CASE WHEN $3::int < 0
		             THEN (SELECT COALESCE(max(sort_order), 0) + 1 FROM platform_sections)
		             ELSE $3 END,
		        $4, NULLIF(COALESCE($5, ''), '')::uuid) RETURNING id`,
		name, icon, sort, req.MarginOverride,
		req.ImageMediaID).Scan(&id); err != nil {
		s.respondErr(w, sectionWriteErr(err))
		return
	}
	s.audit(r, "catalog.section_create", "section", id, map[string]any{"name": *req.Name})
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdatePlatformSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[platformSectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			s.respondErr(w, errValidation)
			return
		}
		req.Name = &trimmed
		if err := s.sectionNameFree(r, trimmed, id); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE platform_sections SET
			name       = COALESCE($2, name),
			icon       = COALESCE($3, icon),
			sort_order = COALESCE($4, sort_order),
			active     = COALESCE($5, active),
			margin_override = CASE WHEN $6::bigint IS NULL THEN margin_override
			                       WHEN $6 < 0 THEN NULL
			                       ELSE $6 END,
			-- **والفراغُ الصريحُ يرفع الصورة** — لا يُقرأ «بلا تغيير».
			image_media_id = CASE WHEN $7::text IS NULL THEN image_media_id
			                      ELSE NULLIF($7, '')::uuid END
		WHERE id = $1`,
		id, req.Name, req.Icon, req.SortOrder, req.Active, req.MarginOverride,
		req.ImageMediaID)
	if err != nil {
		s.respondErr(w, sectionWriteErr(err))
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "catalog.section_update", "section", id, nil)
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// errSectionHasItems **قسمٌ مشغولٌ لا يُحذف بلا وجهةٍ لأصنافه.**
var errSectionHasItems = httpx.NewError(http.StatusConflict,
	"section_has_items", "errors.section_has_items")

// errSectionNameTaken **اسمٌ موجودٌ ولو اختلف التشكيل** — «حلويات» = «حلويّات».
var errSectionNameTaken = httpx.NewError(http.StatusConflict,
	"section_name_taken", "errors.section_name_taken")

// errSectionMoveTarget **وجهةُ النقل قسمٌ آخرُ موجود** — لا القسمُ نفسُه.
var errSectionMoveTarget = httpx.NewError(http.StatusBadRequest,
	"section_move_target", "errors.section_move_target")

// sectionNameFree **أهذا الاسمُ حرٌّ بعد التطبيع؟**
//
// **والتطبيعُ في القاعدة لا هنا** (`section_name_key`، هجرة ٠٢٠٠) — وهو
// نفسُه ما يحرسه الفهرسُ الفريد. **وتطبيعان في موضعين يفترقان يوماً.**
//
// `except` قسمٌ يُعدَّل — **فإعادةُ حفظه باسمه لا تُرفض.**
func (s *Server) sectionNameFree(r *http.Request, name, except string) error {
	var taken bool
	if err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM platform_sections
			 WHERE section_name_key(name) = section_name_key($1)
			   AND ($2 = '' OR id <> $2::uuid))`, name, except).Scan(&taken); err != nil {
		return err
	}
	if taken {
		return errSectionNameTaken
	}
	return nil
}

// sectionWriteErr **سباقُ طلبين بالاسم نفسِه يردّه الفهرسُ** — فيُقال السبب.
func sectionWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return errSectionNameTaken
	}
	return err
}

// handleDeletePlatformSection **يحذف قسماً — وأصنافُه تُنقل أوّلاً.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «يُضاف زرُّ حذف القسم — وقسمٌ فيه أصنافٌ تُنقل
// أصنافُه أوّلاً إلى قسمٍ يختاره الموظّف ثمّ يُحذف؛ والفارغُ يُحذف مباشرة».)
//
//	فارغ                      ←  يُحذف
//	فيه أصناف + `?move_to=`   ←  تُنقل كلُّها إلى الوجهة ثمّ يُحذف — **بمعاملةٍ واحدة**
//	فيه أصنافٌ بلا وجهة       ←  ٤٠٩ `section_has_items` — **ولا صنفَ يُحذف معه أبداً**
//
// **ولا يُفرَّغ قسمُ الصنف**: قرارُ المالك ٢٠٢٦-٠٨-٢٢ جعله إلزاميّاً (هجرة
// ٠١١٨)، **والمفتاحُ `RESTRICT`** (هجرة ٠١٥٧) يمنع حذفاً يترك صنفاً يتيماً.
// **وإعلانُ المتاجر أنّها تبيع في القسم يُنقل مع الأصناف** — وإلّا ضاق
// اختيارُ متجرٍ أعلن قسماً ثمّ اختفى.
func (s *Server) handleDeletePlatformSection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	moveTo := strings.TrimSpace(r.URL.Query().Get("move_to"))
	if moveTo != "" && (!isUUID(moveTo) || moveTo == id) {
		s.respondErr(w, errSectionMoveTarget)
		return
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var moved int64
	if moveTo != "" {
		var ok bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM platform_sections WHERE id = $1)`, moveTo).
			Scan(&ok); err != nil {
			s.respondErr(w, err)
			return
		}
		if !ok {
			s.respondErr(w, errSectionMoveTarget)
			return
		}
		tag, err := tx.Exec(ctx, `
			UPDATE menu_items SET platform_section_id = $2, updated_at = now()
			 WHERE platform_section_id = $1`, id, moveTo)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		moved = tag.RowsAffected()
		if _, err := tx.Exec(ctx, `
			INSERT INTO store_sections (store_id, section_id)
			SELECT store_id, $2 FROM store_sections WHERE section_id = $1
			ON CONFLICT DO NOTHING`, id, moveTo); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	tag, err := tx.Exec(ctx, `DELETE FROM platform_sections WHERE id = $1`, id)
	if err != nil {
		// **والمنعُ حالُ الطالب لا حالُ المنصّة** — `23503` من `RESTRICT`،
		// **و`23502` من قاعدةٍ لم تبلغها `0157` بعد.**
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			(pgErr.Code == "23503" || pgErr.Code == "23502") {
			s.respondErr(w, errSectionHasItems)
			return
		}
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.section_delete", "section", id, map[string]any{
		"move_to": moveTo, "moved_items": moved,
	})
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true, "moved_items": moved})
}

// handleOrderPlatformSections **ترتيبُ الأقسام بالسحب — وهو ترتيبُها عند الزبون.**
//
// **يُرسَل الترتيبُ كلُّه لا خطوةً واحدة** — فلا يبقى قسمان على رقمٍ واحد،
// **ولا يتوقّف الصوابُ على أن تصل الطلباتُ بترتيبها.** والقسمُ الغائبُ عن
// القائمة يُدفع بعدها بترتيبه القديم.
func (s *Server) handleOrderPlatformSections(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		IDs []string `json:"ids"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.IDs) == 0 {
		s.respondErr(w, errValidation)
		return
	}
	seen := map[string]bool{}
	for _, id := range req.IDs {
		if !isUUID(id) || seen[id] {
			s.respondErr(w, errValidation)
			return
		}
		seen[id] = true
	}
	if _, err := s.pg.Exec(r.Context(), `
		WITH wanted AS (
			SELECT t.id::uuid AS id, t.ord::int AS ord
			  FROM unnest($1::text[]) WITH ORDINALITY AS t(id, ord)
		), rest AS (
			SELECT ps.id, ((SELECT count(*) FROM wanted)
			            + row_number() OVER (ORDER BY ps.sort_order, ps.name))::int AS ord
			  FROM platform_sections ps
			 WHERE ps.id NOT IN (SELECT id FROM wanted)
		), all_rows AS (
			SELECT id, ord FROM wanted UNION ALL SELECT id, ord FROM rest
		)
		UPDATE platform_sections ps SET sort_order = a.ord
		  FROM all_rows a WHERE a.id = ps.id`, req.IDs); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.section_order", "section", "", map[string]any{"ids": req.IDs})
	s.touch("catalog", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"ordered": len(req.IDs)})
}

// handleSectionItems أصنافُ قسمٍ بعينه — **كما هي لا كما يراها الزبون.**
//
// # لماذا نقطةٌ خاصّة
//
// نقطةُ التصفّح العامّة (`/public/sections/{id}/items`) **تُرشِّح**: متجرٌ فعّالٌ
// وقسمٌ فعّالٌ وصنفٌ مُقَرّ. **وهي الصواب للزبون وخطأٌ للإدارة**: من يفتح قسماً
// ليقرّر إطفاءَه يريد أن يرى ما فيه كلَّه — **بما فيه ما لا يظهر ولماذا لا
// يظهر.**
//
// **وقسمٌ يبدو فارغاً في اللوحة وفيه عشرةُ أصنافٍ من متجرٍ مُطفَأ يُحذف بلا
// علمٍ بما فيه.**
func (s *Server) handleSectionItems(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **وصفحةٌ محدودةٌ بعدٍّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **وقسمُ السوق يجمع أصنافَ كلّ المتاجر** — ينمو بعدد المتاجر لا بعدد
	// الأقسام. **وخمسُمئةٍ صامتةٌ تعني أنّ صنفاً لا يُوافَق عليه لأنّ أحداً
	// لم يره.**
	pg := pagingOf(r, 50)
	// ══════════════════════════════════════════════════════════════════
	// **والبحثُ والترشيحُ في المحرّك لا في الشاشة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
	//
	// **كانا يعملان على الصفحة المعروضة وحدَها** — فمن بحث عن صنفٍ في
	// الصفحة الثالثة **قرأ «لا أصناف»**. **وبحثٌ يقول «غيرُ موجود» عمّا
	// هو موجودٌ أخطرُ من رقمٍ يكذب**: فيُضاف الصنفُ مرّتين.
	//
	// **وترشيحٌ في الشاشة فوق صفحةٍ وعدٌ بترشيح.**
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	// **والحالُ رمزٌ لا نصٌّ معروض** — **والشاشةُ كانت تُرشِّح بنصّ الشارة
	// العربيّ**، فمن بدّل كلمةً في المعجم كسر الترشيح.
	state := r.URL.Query().Get("state")

	// **وترتيبُ الحال هو ترتيبُ الشاشة نفسُه** — يُقرأ أوّلُ سببٍ يمنع
	// الظهور. **ولا حالَ «ينتظر المراجعة»** — رُفعت المراجعة (٢٠٢٦-١٠-٠٤).
	const stateExpr = `CASE
		WHEN m.status <> 'active' THEN 'store_off'
		WHEN NOT i.available THEN 'out'
		ELSE 'live' END`
	const scope = `
		FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		WHERE i.platform_section_id = $1
		  AND ($2 = '' OR i.name ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%')
		  AND ($3 = '' OR ` + stateExpr + ` = $3)`

	// ══════════════════════════════════════════════════════════════════
	// **والبطاقاتُ تعدّ القسمَ لا الصفحة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **كانت تُحسب من طول الصفحة** — فقسمٌ فيه ثلاثمئة يقول «الكلّ: ٥٠»،
	// **والترقيمُ أسفلَه يقول «١ / ٦»**: رقمان متناقضان في شاشةٍ واحدة.
	//
	// **والمعروضُ فعلاً لا المسجَّل**: قسمٌ فيه اثنا عشر ويُعرض منه ثلاثةٌ
	// **حالةٌ تُعالَج، ورقمٌ واحدٌ يخفيها.**
	//
	// **وتُحسب قبل الترشيح بالحال** — **وبطاقةٌ تتبع مُرشِّحَها تقول
	// «المعروضُ صفر» لمن رشّح «نفد»**، وهي لا تخصّه.
	var count, all, live int
	if err := s.pg.QueryRow(r.Context(), `SELECT count(*)`+scope, id, q, state).
		Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*), count(*) FILTER (WHERE `+stateExpr+` = 'live')`+scope,
		id, q, "").Scan(&all, &live); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT i.id::text, i.name, i.merchant_price, i.available,
		       m.name, m.status, im.thumb_path, im.path,
		       m.commission_percent, i.margin_override, ps.margin_override
		FROM menu_items i
		JOIN merchants m ON m.id = i.merchant_id
		JOIN platform_sections ps ON ps.id = i.platform_section_id
		LEFT JOIN media im ON im.id = i.image_media_id
		WHERE i.platform_section_id = $1
		  AND ($2 = '' OR i.name ILIKE '%'||$2||'%' OR m.name ILIKE '%'||$2||'%')
		  AND ($3 = '' OR `+stateExpr+` = $3)
		ORDER BY m.name, i.name
		LIMIT $4 OFFSET $5`, id, q, state, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// MerchantPrice **سعرُ الشراء** — ما وضعه المتجر، وأصلُ الحسبتين.
		MerchantPrice  int64   `json:"merchant_price"`
		Available      bool    `json:"available"`
		MerchantName   string  `json:"merchant_name"`
		MerchantStatus string  `json:"merchant_status"`
		ThumbURL       *string `json:"thumb_url"`
		// ImageURL **الأصلُ للبطاقة، والمصغَّرةُ للسطر.**
		//
		// المصغَّرةُ حدُّها ٤٠٠ بكسل، **وبطاقةٌ تمطّها تبهت** — وهو ما حدث في
		// بطاقة القسم قبلها.
		ImageURL *string `json:"image_url"`

		// --- الحسبتان: واحدةٌ تنزل على المتجر وأخرى تصعد على الزبون ---

		// CommissionPct نسبةُ عمولة المنصة النافذة — **بعد الوراثة**:
		// تجاوزُ المتجر إن كان، وإلّا العامّ. **ورقمُ الإعدادات وحدَه يكذب
		// على متجرٍ اتُّفق معه على غيره.**
		CommissionPct int `json:"commission_percent"`
		// CommissionMode «نسبة» أم «مقطوع» — **فلا تُكتب «٪» على رقمٍ بالليرة.**
		CommissionMode string `json:"commission_mode"`
		// Commission قيمةُ العمولة بالليرة — **تُقتطع من المتجر لا تُضاف للزبون.**
		Commission int64 `json:"commission"`
		// MerchantNet **ما يقبضه المتجر فعلاً** — سعرُ الشراء ناقصَ العمولة.
		MerchantNet int64 `json:"merchant_net"`
		// MarginValue هامشُ الصنف النافذ بالليرة — **بعد الوراثة**: تجاوزُ الصنف، فتجاوزُ
		// قسمه، فالعام. **ورقمُ الإعدادات وحدَه يكذب على من خُصّ بغيره.**
		MarginValue int64 `json:"margin_value"`
		// Margin قيمةُ الهامش بالليرة — **يشمل أثرَ التقريب**، فهو الفرقُ
		// المحسوب لا حاصلُ ضربٍ يُعاد. **ورقمٌ يُحسب مرّتين يفترق.**
		Margin int64 `json:"margin"`
		// SalePrice **ما يدفعه الزبون** — من `pricing` وحدَها.
		SalePrice int64 `json:"sale_price"`
		// MarginOverride تجاوزُ هذا الصنف — **وفراغُه «اتبع قسمَك».**
		MarginOverride *int64 `json:"margin_override"`
	}
	// **والسعرُ من `pricing` لا من هنا.** المعادلةُ (نمطٌ ووراثةٌ وتقريب) في
	// حزمةٍ واحدة، **ولو حُسبت هنا لَافترقت عن حسبة القائمة وحسبة الطلب** —
	// فيُقرأ في اللوحة رقمٌ ويُباع بغيره.
	rule := pricing.RuleFrom(r.Context(), s.settings)
	out := []item{}
	for rows.Next() {
		var x item
		var sectionMargin *int64
		var commOverride *int64
		if err := rows.Scan(&x.ID, &x.Name, &x.MerchantPrice, &x.Available,
			&x.MerchantName, &x.MerchantStatus, &x.ThumbURL, &x.ImageURL,
			&commOverride, &x.MarginOverride, &sectionMargin); err != nil {
			s.respondErr(w, err)
			return
		}
		x.ThumbURL = media.URLForPtr(x.ThumbURL)
		x.ImageURL = media.URLForPtr(x.ImageURL)

		// **والعمولةُ من `pricing` كالهامش** — نمطاً ووراثةً.
		comm := pricing.MerchantCommission(r.Context(), s.settings, commOverride)
		x.CommissionPct = int(comm.Value)
		x.CommissionMode = comm.Mode
		x.Commission = comm.Of(x.MerchantPrice)
		x.MerchantNet = x.MerchantPrice - x.Commission

		x.MarginValue = rule.Value
		switch {
		case x.MarginOverride != nil:
			x.MarginValue = *x.MarginOverride
		case sectionMargin != nil:
			x.MarginValue = *sectionMargin
		}
		x.SalePrice = rule.SalePrice(x.MerchantPrice, x.MarginOverride, sectionMargin)
		x.Margin = x.SalePrice - x.MerchantPrice
		out = append(out, x)
	}
	// **والهامشُ ثابتٌ دائماً** — لا نمطَ يُرسَل. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
	// **و`count` عددُ الكلّ لا طولُ الصفحة** — **كان `len(out)` وكان صادقاً
	// حين تُعرض كلُّها**، ويصير مع الترقيم عددَ ما في الشاشة. **ورقمٌ يقول
	// «٥٠ صنفاً في القسم» وفيه أربعُمئةٍ يُبنى عليه قرارُ عرض.**
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": out, "count": count, "page": pg.Page, "per_page": pg.PerPage,
		// **وعددا القسم كلِّه** — تقرؤهما البطاقات، **ولا تعدّ الصفحة.**
		"all": all, "live": live,
	})
}
