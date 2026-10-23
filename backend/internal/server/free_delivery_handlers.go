package server

// ══════════════════════════════════════════════════════════════════════
// **«التوصيل المجاني» في اللوحة** — قرارُ المالك ٢٠٢٦-١٠-٢٣
// ══════════════════════════════════════════════════════════════════════
//
// عرضٌ بمفتاحٍ وبدايةٍ ونهاية، ويختار ما يشمل (العاديّ · المشتريات · الأمانة ·
// «لدي توصيلة»)، وسقفٌ اختياريٌّ لما يُعفى منه. **مرّةٌ لكلّ زبونٍ بكلّ عرض** —
// و«عرضٌ جديد» يُعطي الجميعَ توصيلةً جديدة. الحسابُ في `orders/free_delivery.go`.

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

var freeDeliveryKinds = []string{orders.FreeKindStandard, orders.FreeKindPurchase,
	orders.FreeKindAmanah, orders.FreeKindMerchant}

var errFreeDeliveryInvalid = httpx.NewError(http.StatusBadRequest,
	"free_delivery_invalid", "errors.free_delivery_invalid")

type freeDeliveryCampaign struct {
	ID        string    `json:"id"`
	Active    bool      `json:"active"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Kinds     []string  `json:"kinds"`
	MaxFee    *int64    `json:"max_fee"`
	CreatedAt time.Time `json:"created_at"`
	// Running **شغّالٌ الآن** — مفعَّلٌ والوقتُ بين بدايته ونهايته.
	Running bool `json:"running"`
	// Claims ما انعطى منه — بلا الطلبات الملغاة والمرفوضة.
	Claims int64 `json:"claims"`
	// Waived **ما تحمّلته المنصّة** — مجموعُ المعفى.
	Waived int64            `json:"waived"`
	ByKind map[string]int64 `json:"by_kind"`
}

func (s *Server) loadFreeDeliveryCampaign(ctx context.Context, where string, args ...any) (*freeDeliveryCampaign, error) {
	var c freeDeliveryCampaign
	err := s.pg.QueryRow(ctx, `
		SELECT id::text, active, starts_at, ends_at, kinds, max_fee, created_at,
		       active AND now() >= starts_at AND now() < ends_at
		  FROM free_delivery_campaigns `+where+` ORDER BY created_at DESC LIMIT 1`, args...).
		Scan(&c.ID, &c.Active, &c.StartsAt, &c.EndsAt, &c.Kinds, &c.MaxFee, &c.CreatedAt, &c.Running)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ByKind = map[string]int64{}
	rows, err := s.pg.Query(ctx, `
		SELECT c.kind, count(*), COALESCE(sum(c.waived), 0)
		  FROM free_delivery_claims c JOIN orders o ON o.id = c.order_id
		 WHERE c.campaign_id = $1 AND o.status NOT IN ('cancelled', 'rejected')
		 GROUP BY c.kind`, c.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var n, w int64
		if err := rows.Scan(&k, &n, &w); err != nil {
			return nil, err
		}
		c.ByKind[k] = n
		c.Claims += n
		c.Waived += w
	}
	return &c, rows.Err()
}

// handleGetFreeDelivery **العرضُ الأحدث وما انعطى منه.**
func (s *Server) handleGetFreeDelivery(w http.ResponseWriter, r *http.Request) {
	c, err := s.loadFreeDeliveryCampaign(r.Context(), ``)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"campaign": c})
}

type freeDeliveryInput struct {
	Active   bool      `json:"active"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Kinds    []string  `json:"kinds"`
	MaxFee   *int64    `json:"max_fee"`
}

func (in *freeDeliveryInput) valid() bool {
	if !in.EndsAt.After(in.StartsAt) {
		return false
	}
	if in.MaxFee != nil && *in.MaxFee <= 0 {
		in.MaxFee = nil
	}
	out := []string{}
	for _, k := range in.Kinds {
		if !slices.Contains(freeDeliveryKinds, k) {
			return false
		}
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	in.Kinds = out
	// **عرضٌ مفعَّلٌ لا يشمل شيئاً وعدٌ فارغ.**
	return !in.Active || len(in.Kinds) > 0
}

// handleCreateFreeDelivery **عرضٌ جديد** — والجميعُ يرجع له حقُّ توصيلة.
func (s *Server) handleCreateFreeDelivery(w http.ResponseWriter, r *http.Request) {
	in, err := decode[freeDeliveryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !in.valid() {
		s.respondErr(w, errFreeDeliveryInvalid)
		return
	}
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds, max_fee, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		in.Active, in.StartsAt, in.EndsAt, in.Kinds, in.MaxFee, nilIfEmptyID(userIDFrom(r))).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "admin.free_delivery_create", "free_delivery_campaign", id, map[string]any{
		"active": in.Active, "starts_at": in.StartsAt, "ends_at": in.EndsAt,
		"kinds": in.Kinds, "max_fee": in.MaxFee})
	c, err := s.loadFreeDeliveryCampaign(r.Context(), `WHERE id = $1`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"campaign": c})
}

// handleUpdateFreeDelivery **تعديلُ العرض نفسِه** — ومن نال توصيلتَه لا ينالها ثانية.
func (s *Server) handleUpdateFreeDelivery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	in, err := decode[freeDeliveryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !in.valid() {
		s.respondErr(w, errFreeDeliveryInvalid)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE free_delivery_campaigns
		   SET active = $2, starts_at = $3, ends_at = $4, kinds = $5, max_fee = $6, updated_at = now()
		 WHERE id = $1`, id, in.Active, in.StartsAt, in.EndsAt, in.Kinds, in.MaxFee)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "admin.free_delivery_update", "free_delivery_campaign", id, map[string]any{
		"active": in.Active, "starts_at": in.StartsAt, "ends_at": in.EndsAt,
		"kinds": in.Kinds, "max_fee": in.MaxFee})
	c, err := s.loadFreeDeliveryCampaign(r.Context(), `WHERE id = $1`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"campaign": c})
}

// nilIfEmptyID **معرّفٌ فارغٌ ⇒ NULL** — لا نصٌّ فارغٌ في عمود `uuid`.
func nilIfEmptyID(id string) any {
	if !isUUID(id) {
		return nil
	}
	return id
}
