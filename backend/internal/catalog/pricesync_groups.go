package catalog

// ══════════════════════════════════════════════════════════════════════
// **كلّ صنف من الأمين يروح على قسمو** (قرار المالك ٢٠٢٦-١١-٠٧)
// ══════════════════════════════════════════════════════════════════════
//
// كانت «أنشئ غير الموجود» تحطّ كلّ صنف جديد بقسم الانتظار المخفي — وبالتجربة
// الحقيقية راح ٥٣٠٨ صنف كلّهم لهنيك. هلّق قسم الصنف الجديد بينحلّ هيك:
//
//  1. **مجموعة الأمين** إذا صاحب المتجر اختار لها قسم (`price_sync_groups`) ⇒ `group`.
//  2. **وإلّا من اسم الصنف** (`SuggestSection`) إذا كان متأكّد ⇒ `name`.
//  3. **وإلّا قسم الانتظار** ⇒ `pending`، ويرتّبه بعدين «وزّع تلقائياً» أو المراجعة بالإيد.
//
// و`section_source = manual` يعني حدا اختار القسم بإيده — **فالمزامنة وتغيير المجموعة
// وزرّ التوزيع ما بيحرّكوه أبداً.** وكلّ تعديل قسم من محرّر الصنف بيقلبه `manual`.
//
// **قسم مقترح مطفي بينحطّ فيه كمان**: الأصناف الجديدة مخفية أصلاً (`available = false`)
// والزبون ما بيشوف قسم مطفي — فبيظهر الصنف أوّل ما ينفتح القسم، بلا ما يرجع حدا يرتّبه.

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// suggestSection **مقترح القسم من الاسم** — متغيّر لتبدّله الاختبارات.
var suggestSection = SuggestSection

// مصادر قسم الصنف.
const (
	SourcePending = "pending"
	SourceGroup   = "group"
	SourceName    = "name"
	SourceManual  = "manual"
)

// maxGroupKey **أطول مفتاح مجموعة.**
const maxGroupKey = 120

// groupKey **مفتاح المجموعة** — اسمها بلا فروق الهمزات والفراغات.
func groupKey(s string) string {
	k := normName(s)
	if utf8.RuneCountInString(k) > maxGroupKey {
		k = string([]rune(k)[:maxGroupKey])
	}
	return k
}

// sectionResolver **اسم القسم ⇒ معرّفه**، بذاكرة للنداء الواحد.
type sectionResolver struct {
	q     dbtx.Querier
	ids   map[string]string
	names map[string]string
}

func newSectionResolver(q dbtx.Querier) *sectionResolver {
	return &sectionResolver{q: q, ids: map[string]string{}, names: map[string]string{}}
}

// id **معرّف القسم بالاسم** — بنفس مفتاح `ensurePendingSection`، والفعّال أوّلاً؛ "" إن غاب.
func (r *sectionResolver) id(ctx context.Context, name string) (string, error) {
	if id, ok := r.ids[name]; ok {
		return id, nil
	}
	var id, real string
	err := r.q.QueryRow(ctx, `
		SELECT id::text, name FROM platform_sections
		 WHERE section_name_key(name) = section_name_key($1) ORDER BY active DESC LIMIT 1`, name).Scan(&id, &real)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	r.ids[name] = id
	if id != "" {
		r.names[id] = real
	}
	return id, nil
}

// suggest **قسم مقترح لاسم صنف** — "" إن ما في اقتراح أكيد أو ما لقى القسم.
func (r *sectionResolver) suggest(ctx context.Context, itemName string) (string, error) {
	sec, ok := suggestSection(itemName)
	if !ok || strings.TrimSpace(sec) == "" {
		return "", nil
	}
	return r.id(ctx, sec)
}

// pendingSectionID **قسم الانتظار بلا إنشاء** — "" إن ما في.
func pendingSectionID(ctx context.Context, q dbtx.Querier) (string, error) {
	var id string
	err := q.QueryRow(ctx, `SELECT id::text FROM platform_sections
	          WHERE section_name_key(name) = section_name_key($1) ORDER BY active DESC LIMIT 1`,
		PendingSectionName).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// loadGroupMap **مجموعات المتجر اللي إلها قسم** — مفتاح ⇒ معرّف القسم.
func loadGroupMap(ctx context.Context, q dbtx.Querier, merchantID string) (map[string]string, error) {
	rows, err := q.Query(ctx, `
		SELECT group_key, platform_section_id::text FROM price_sync_groups
		 WHERE merchant_id = $1::uuid AND platform_section_id IS NOT NULL`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, sec string
		if err := rows.Scan(&k, &sec); err != nil {
			return nil, err
		}
		out[k] = sec
	}
	return out, rows.Err()
}

// upsertGroups **كلّ مجموعة شافتها المزامنة بتنحفظ** (بلا قسم) لتنعرض بالصفحة.
func upsertGroups(ctx context.Context, q dbtx.Querier, merchantID string, seen map[string]string) error {
	if len(seen) == 0 {
		return nil
	}
	keys := make([]string, 0, len(seen))
	names := make([]string, 0, len(seen))
	for k, n := range seen {
		keys, names = append(keys, k), append(names, n)
	}
	_, err := q.Exec(ctx, `
		INSERT INTO price_sync_groups (merchant_id, group_key, group_name)
		SELECT $1::uuid, k, n FROM unnest($2::text[], $3::text[]) AS t(k, n)
		ON CONFLICT (merchant_id, group_key) DO UPDATE SET group_name = EXCLUDED.group_name`,
		merchantID, keys, names)
	return err
}

// addStoreSections **متجر أعلن أقسامه بينضاف إلها القسم الجديد** — متل النقل من اللوحة؛
// والمتجر اللي ما أعلن (= الكلّ) ما بينمسّ.
func addStoreSections(ctx context.Context, q dbtx.Querier, merchantID string, secs []string) error {
	if len(secs) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO store_sections (store_id, section_id)
		SELECT DISTINCT $1::uuid, x::uuid FROM unnest($2::text[]) AS x
		 WHERE EXISTS (SELECT 1 FROM store_sections s WHERE s.store_id = $1::uuid)
		ON CONFLICT DO NOTHING`, merchantID, secs)
	return err
}

// placeNewItems **يحلّ قسم كلّ صنف جديد**: المجموعة ثمّ الاسم ثمّ الانتظار — ويعدّ.
// قسم الانتظار بينعمل بس إذا احتاجه صنف وما كان «جرّب».
func placeNewItems(ctx context.Context, q dbtx.Querier, merchantID string, items []newItem, dry bool, rep *SyncReport) error {
	if len(items) == 0 {
		return nil
	}
	gm, err := loadGroupMap(ctx, q, merchantID)
	if err != nil {
		return err
	}
	res := newSectionResolver(q)
	pending := ""
	for i := range items {
		it := &items[i]
		if sec := gm[groupKey(it.group)]; it.group != "" && sec != "" {
			it.section, it.source = sec, SourceGroup
			rep.CreatedByGroup++
			continue
		}
		sec, err := res.suggest(ctx, it.name)
		if err != nil {
			return err
		}
		if sec != "" {
			it.section, it.source = sec, SourceName
			rep.CreatedByName++
			continue
		}
		rep.CreatedPending++
		it.source = SourcePending
		if dry {
			continue
		}
		if pending == "" {
			if pending, err = ensurePendingSection(ctx, q); err != nil {
				return err
			}
		}
		it.section = pending
	}
	return nil
}

// ── الصفحة: مجموعات الأمين ─────────────────────────────────────────────

// SyncGroup **مجموعة أمين** كما تنعرض بالصفحة.
type SyncGroup struct {
	Key                  string  `json:"group_key"`
	Name                 string  `json:"group_name"`
	Items                int     `json:"items"`
	Waiting              int     `json:"waiting"`
	SectionID            *string `json:"platform_section_id"`
	SectionName          *string `json:"platform_section_name"`
	SuggestedSectionID   *string `json:"suggested_section_id"`
	SuggestedSectionName *string `json:"suggested_section_name"`
}

// SyncGroupsView **ردّ صفحة التوزيع** — المجموعات وكم صنف لسّا بالانتظار.
type SyncGroupsView struct {
	Groups  []SyncGroup `json:"groups"`
	Pending int         `json:"pending"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// pendingCount **كم صنف للمتجر بقسم الانتظار.**
func pendingCount(ctx context.Context, q dbtx.Querier, merchantID string) (int, error) {
	pend, err := pendingSectionID(ctx, q)
	if err != nil || pend == "" {
		return 0, err
	}
	var n int
	err = q.QueryRow(ctx, `SELECT count(*) FROM menu_items
	                        WHERE merchant_id = $1::uuid AND platform_section_id = $2::uuid`,
		merchantID, pend).Scan(&n)
	return n, err
}

// PriceSyncGroups **مجموعات الأمين للمتجر**: عدد أصنافها وقسمها والقسم المقترح
// (الأكثر بين اقتراحات أسماء أصنافها، وإلّا من اسم المجموعة نفسها).
func (s *Service) PriceSyncGroups(ctx context.Context, merchantID string) (*SyncGroupsView, error) {
	view := &SyncGroupsView{Groups: []SyncGroup{}}
	rows, err := s.db.Query(ctx, `
		SELECT g.group_key, g.group_name, g.platform_section_id::text, ps.name
		  FROM price_sync_groups g
		  LEFT JOIN platform_sections ps ON ps.id = g.platform_section_id
		 WHERE g.merchant_id = $1::uuid ORDER BY g.group_name`, merchantID)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for rows.Next() {
		var g SyncGroup
		if err := rows.Scan(&g.Key, &g.Name, &g.SectionID, &g.SectionName); err != nil {
			rows.Close()
			return nil, err
		}
		idx[g.Key] = len(view.Groups)
		view.Groups = append(view.Groups, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pend, err := pendingSectionID(ctx, s.db)
	if err != nil {
		return nil, err
	}
	items, err := s.db.Query(ctx, `
		SELECT name, external_group, COALESCE(platform_section_id::text, '') FROM menu_items
		 WHERE merchant_id = $1::uuid AND COALESCE(external_group, '') <> ''`, merchantID)
	if err != nil {
		return nil, err
	}
	res := newSectionResolver(s.db)
	votes := map[string]map[string]int{}
	type row struct{ name, group, sec string }
	var all []row
	for items.Next() {
		var r row
		if err := items.Scan(&r.name, &r.group, &r.sec); err != nil {
			items.Close()
			return nil, err
		}
		all = append(all, r)
	}
	items.Close()
	if err := items.Err(); err != nil {
		return nil, err
	}
	for _, r := range all {
		k := groupKey(r.group)
		i, ok := idx[k]
		if !ok {
			idx[k] = len(view.Groups)
			i = idx[k]
			view.Groups = append(view.Groups, SyncGroup{Key: k, Name: r.group})
		}
		view.Groups[i].Items++
		if pend != "" && r.sec == pend {
			view.Groups[i].Waiting++
		}
		if sec, err := res.suggest(ctx, r.name); err != nil {
			return nil, err
		} else if sec != "" {
			if votes[k] == nil {
				votes[k] = map[string]int{}
			}
			votes[k][sec]++
		}
	}
	for i := range view.Groups {
		g := &view.Groups[i]
		best, n := "", 0
		for sec, c := range votes[g.Key] {
			if c > n || (c == n && sec < best) {
				best, n = sec, c
			}
		}
		if best == "" {
			if best, err = res.suggest(ctx, g.Name); err != nil {
				return nil, err
			}
		}
		if best != "" {
			g.SuggestedSectionID = strPtr(best)
			g.SuggestedSectionName = strPtr(res.names[best])
		}
	}
	if view.Pending, err = pendingCount(ctx, s.db, merchantID); err != nil {
		return nil, err
	}
	return view, nil
}

// GroupChoice **قسم لمجموعة** — والفارغ يشيل الاختيار.
type GroupChoice struct {
	Key       string `json:"group_key"`
	SectionID string `json:"platform_section_id"`
}

// checkSection **قسم موجود وفعّال** — وإلّا `ErrSectionRequired` («اختر القسم»).
func checkSection(ctx context.Context, q dbtx.Querier, id string) error {
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM platform_sections
	                             WHERE id::text = $1 AND active)`, id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrSectionRequired
	}
	return nil
}

// SetPriceSyncGroups **يحفظ قسم كلّ مجموعة وينقل أصنافها** — اللي مصدرها pending أو
// group أو name بس؛ **واليدويّ ما بينمسّ.** ويردّ كم صنف انتقل.
func (s *Service) SetPriceSyncGroups(ctx context.Context, merchantID, actorID string, in []GroupChoice) (int, error) {
	if len(in) == 0 || len(in) > 2000 {
		return 0, ErrSyncEmpty
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	want := map[string]string{}
	for _, c := range in {
		k := groupKey(c.Key)
		if k == "" {
			return 0, ErrSyncMapping
		}
		sec := strings.TrimSpace(c.SectionID)
		if sec != "" {
			if err := checkSection(ctx, tx, sec); err != nil {
				return 0, err
			}
		}
		want[k] = sec
		if _, err := tx.Exec(ctx, `
			INSERT INTO price_sync_groups (merchant_id, group_key, group_name, platform_section_id, updated_at)
			VALUES ($1::uuid, $2, $3, NULLIF($4, '')::uuid, now())
			ON CONFLICT (merchant_id, group_key) DO UPDATE
			   SET platform_section_id = EXCLUDED.platform_section_id, updated_at = now()`,
			merchantID, k, strings.TrimSpace(c.Key), sec); err != nil {
			return 0, err
		}
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, external_group FROM menu_items
		 WHERE merchant_id = $1::uuid AND COALESCE(external_group, '') <> ''
		   AND section_source IN ('pending', 'group', 'name')`, merchantID)
	if err != nil {
		return 0, err
	}
	var ids, secs []string
	for rows.Next() {
		var id, g string
		if err := rows.Scan(&id, &g); err != nil {
			rows.Close()
			return 0, err
		}
		if sec := want[groupKey(g)]; sec != "" {
			ids, secs = append(ids, id), append(secs, sec)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	moved, err := moveItems(ctx, tx, merchantID, ids, secs, SourceGroup)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.audit(ctx, actorID, "menu.price_sync_groups", "merchant", merchantID, "")
	return moved, nil
}

// moveItems **ينقل كلّ صنف لقسمه** دفعةً واحدة — واليدويّ ما بينمسّ إلّا إذا المصدر الجديد يدويّ.
func moveItems(ctx context.Context, q dbtx.Querier, merchantID string, ids, secs []string, source string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := q.Exec(ctx, `
		UPDATE menu_items i SET platform_section_id = t.sec::uuid, section_source = $4, updated_at = now()
		  FROM unnest($2::text[], $3::text[]) AS t(id, sec)
		 WHERE i.id = t.id::uuid AND i.merchant_id = $1::uuid
		   AND ($4 = 'manual' OR i.section_source <> 'manual')`,
		merchantID, ids, secs, source)
	if err != nil {
		return 0, err
	}
	uniq := map[string]bool{}
	var list []string
	for _, sec := range secs {
		if !uniq[sec] {
			uniq[sec] = true
			list = append(list, sec)
		}
	}
	if err := addStoreSections(ctx, q, merchantID, list); err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// AutoSortResult **نتيجة «وزّع تلقائياً».**
type AutoSortResult struct {
	ByGroup int `json:"by_group"`
	ByName  int `json:"by_name"`
	Waiting int `json:"waiting"`
}

// AutoSortPending **يوزّع أصناف قسم الانتظار**: مجموعتها إن إلها قسم، وإلّا من اسمها.
// والباقي بيضلّ بالانتظار للمراجعة بالإيد.
func (s *Service) AutoSortPending(ctx context.Context, merchantID, actorID string) (*AutoSortResult, error) {
	out := &AutoSortResult{}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	pend, err := pendingSectionID(ctx, tx)
	if err != nil || pend == "" {
		return out, err
	}
	gm, err := loadGroupMap(ctx, tx, merchantID)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, name, COALESCE(external_group, '') FROM menu_items
		 WHERE merchant_id = $1::uuid AND platform_section_id = $2::uuid AND section_source <> 'manual'`,
		merchantID, pend)
	if err != nil {
		return nil, err
	}
	type row struct{ id, name, group string }
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.group); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	res := newSectionResolver(tx)
	var gIDs, gSecs, nIDs, nSecs []string
	for _, r := range all {
		if sec := gm[groupKey(r.group)]; r.group != "" && sec != "" {
			gIDs, gSecs = append(gIDs, r.id), append(gSecs, sec)
			continue
		}
		sec, err := res.suggest(ctx, r.name)
		if err != nil {
			return nil, err
		}
		if sec != "" && sec != pend {
			nIDs, nSecs = append(nIDs, r.id), append(nSecs, sec)
		}
	}
	if out.ByGroup, err = moveItems(ctx, tx, merchantID, gIDs, gSecs, SourceGroup); err != nil {
		return nil, err
	}
	if out.ByName, err = moveItems(ctx, tx, merchantID, nIDs, nSecs, SourceName); err != nil {
		return nil, err
	}
	if out.Waiting, err = pendingCount(ctx, tx, merchantID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "menu.price_sync_autosort", "merchant", merchantID, "")
	return out, nil
}

// ── المراجعة بالإيد: أصناف الانتظار ────────────────────────────────────

// PendingItem **صنف بقسم الانتظار** مع القسم المقترح من اسمه.
type PendingItem struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Code                 *string `json:"external_code"`
	Group                *string `json:"external_group"`
	Price                int64   `json:"price"`
	SuggestedSectionID   *string `json:"suggested_section_id"`
	SuggestedSectionName *string `json:"suggested_section_name"`
}

// PendingPage **صفحة من أصناف الانتظار** — و`next` فارغ بآخر صفحة.
type PendingPage struct {
	Items []PendingItem `json:"items"`
	Total int           `json:"total"`
	Next  string        `json:"next"`
}

// MaxPendingPage **أكبر صفحة.**
const MaxPendingPage = 100

// PendingItems **أصناف المتجر بقسم الانتظار** بترتيب الملفّ، صفحة صفحة، وبحث بالاسم أو الرقم.
// `after` هو `next` الصفحة اللي قبل («ترتيب_معرّف»).
func (s *Service) PendingItems(ctx context.Context, merchantID, after, q string, limit int) (*PendingPage, error) {
	if limit <= 0 || limit > MaxPendingPage {
		limit = 50
	}
	page := &PendingPage{Items: []PendingItem{}}
	pend, err := pendingSectionID(ctx, s.db)
	if err != nil || pend == "" {
		return page, err
	}
	afterOrd, afterID := -1, ""
	if o, id, ok := strings.Cut(after, "_"); ok {
		if n, err := strconv.Atoi(o); err == nil && isUUIDText(id) {
			afterOrd, afterID = n, id
		}
	}
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > 80 {
		q = string([]rune(q)[:80])
	}
	like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
	filter := `merchant_id = $1::uuid AND platform_section_id = $2::uuid
	           AND ($3 = '' OR name ILIKE $4 OR COALESCE(external_code, '') ILIKE $4)`
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE `+filter,
		merchantID, pend, q, like).Scan(&page.Total); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, external_code, external_group, merchant_price, sort_order
		  FROM menu_items WHERE `+filter+`
		   AND ($5::int < 0 OR (sort_order, id) > ($5::int, NULLIF($6, '')::uuid))
		 ORDER BY sort_order, id LIMIT $7`,
		merchantID, pend, q, like, afterOrd, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	var ords []int
	for rows.Next() {
		var it PendingItem
		var ord int
		if err := rows.Scan(&it.ID, &it.Name, &it.Code, &it.Group, &it.Price, &ord); err != nil {
			rows.Close()
			return nil, err
		}
		page.Items = append(page.Items, it)
		ords = append(ords, ord)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.Next = strconv.Itoa(ords[limit-1]) + "_" + page.Items[limit-1].ID
	}
	res := newSectionResolver(s.db)
	for i := range page.Items {
		sec, err := res.suggest(ctx, page.Items[i].Name)
		if err != nil {
			return nil, err
		}
		if sec != "" {
			page.Items[i].SuggestedSectionID = strPtr(sec)
			page.Items[i].SuggestedSectionName = strPtr(res.names[sec])
		}
	}
	return page, nil
}

// MaxMoveItems **أكبر نقلة وحدة.**
const MaxMoveItems = 500

// MoveSyncItems **صاحب المتجر بينقل أصناف لقسم بإيده** ⇒ `manual`، فالمزامنة ما بترجّعهم.
func (s *Service) MoveSyncItems(ctx context.Context, merchantID, actorID string, ids []string, section string) (int, error) {
	if len(ids) == 0 || len(ids) > MaxMoveItems {
		return 0, ErrSyncEmpty
	}
	for _, id := range ids {
		if !isUUIDText(id) {
			return 0, ErrSyncEmpty
		}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := checkSection(ctx, tx, section); err != nil {
		return 0, err
	}
	secs := make([]string, len(ids))
	for i := range secs {
		secs[i] = section
	}
	n, err := moveItems(ctx, tx, merchantID, ids, secs, SourceManual)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.audit(ctx, actorID, "menu.price_sync_move", "merchant", merchantID, "")
	return n, nil
}

// isUUIDText **نصّ بشكل معرّف.**
func isUUIDText(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return false
			}
		}
	}
	return true
}
