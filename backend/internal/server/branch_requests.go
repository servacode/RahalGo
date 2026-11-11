package server

// ══════════════════════════════════════════════════════════════════════
// **«أضف فرع» — طلباتُ الفروع** (قرارُ المالك ٢٠٢٦-١١-١٠ و«ابدأ كلشي» ٢٠٢٦-١١-١١)
// ══════════════════════════════════════════════════════════════════════
//
// صاحبُ المتجر أو مندوبُه يطلب فرعاً (اسمٌ وموقعٌ على الخريطة وهاتفٌ اختياريّ)، والأدمنُ يوافق
// ويحدّد العمولة فيُنشأ متجرٌ جديدٌ **مربوطٌ بحساب الصاحب نفسِه**، أو يرفض بسببٍ اختياريّ.
//
//   - `POST /merchant/stores/{id}/branch-requests` و`GET /merchant/branch-requests` — صاحبُ المتجر.
//   - `POST /rep/merchants/{id}/branch-requests` و`GET /rep/merchants/{id}/branch-requests` — المندوبُ
//     لمتاجره المسنَدة إليه وحدَها.
//   - `GET /admin/branch-requests` و`/count` و`POST /{id}/approve` و`/{id}/reject` — المكتب (`merchants.verify`).
//
// **والموافقةُ تمرّ بـ`CreateMerchantTx` نفسِه** الذي تنشئ به اللوحةُ متجراً وتحوّل به المرشَّحَ — فالمدينةُ
// تُشتقّ من النقطة والمنطقةُ والتصنيفُ من الأمّ، **ولا فحصَ «الرقمُ له متجر»** هنا لأنّ الفرعَ بصاحبه نفسِه
// بالتعريف. **ولا يمسّ الدفترَ**: يُنسخ من الأمّ ما ليس مالاً جارياً (طريقةُ التسوية، المندوب، العملة).

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

var (
	// errBranchPending **طلبٌ مفتوحٌ بالاسم نفسِه** — يصل التطبيقَ فله نصٌّ في `ApiErrors`.
	errBranchPending = httpx.NewError(http.StatusConflict, "branch_request_pending", "errors.branch_request_pending")
	// errBranchDecided **قُرّر في الطلب** — للمكتب وحدَه.
	errBranchDecided = httpx.NewError(http.StatusConflict, "branch_request_decided", "errors.branch_request_decided")
)

// branchPendingMax **أكثرُ ما يُترك مفتوحاً لمتجرٍ واحد** — يمنع إغراقَ الطابور.
const branchPendingMax = 5

const (
	branchPending  = "pending"
	branchApproved = "approved"
	branchRejected = "rejected"
)

// branchRequestView **صفُّ الطلب كما يراه كلُّ طرف**.
type branchRequestView struct {
	ID               string            `json:"id"`
	ParentMerchantID string            `json:"parent_merchant_id"`
	ParentName       string            `json:"parent_name"`
	Name             string            `json:"name"`
	Phone            *string           `json:"phone"`
	AddressText      *string           `json:"address_text"`
	Lat              float64           `json:"lat"`
	Lng              float64           `json:"lng"`
	Status           string            `json:"status"`
	CommissionPct    *pricing.Percent2 `json:"commission_percent"`
	RejectReason     *string           `json:"reject_reason"`
	RequestedByRole  string            `json:"requested_by_role"`
	RequestedByName  *string           `json:"requested_by_name"`
	OwnerName        *string           `json:"owner_name"`
	OwnerPhone       *string           `json:"owner_phone"`
	BranchMerchantID *string           `json:"branch_merchant_id"`
	CreatedAt        time.Time         `json:"created_at"`
	DecidedAt        *time.Time        `json:"decided_at"`
}

const branchViewSQL = `
	SELECT b.id::text, b.parent_merchant_id::text, pm.name, b.name, b.phone, b.address_text,
	       b.lat, b.lng, b.status, b.commission_percent, b.reject_reason, b.requested_by_role,
	       NULLIF(btrim(ru.full_name), ''), NULLIF(btrim(ou.full_name), ''), NULLIF(ou.phone::text, ''),
	       b.branch_merchant_id::text, b.created_at, b.decided_at
	  FROM branch_requests b
	  JOIN merchants pm ON pm.id = b.parent_merchant_id
	  LEFT JOIN users ru ON ru.id = b.requested_by
	  LEFT JOIN users ou ON ou.id = pm.owner_user_id `

func scanBranchViews(rows pgx.Rows) ([]branchRequestView, error) {
	defer rows.Close()
	out := []branchRequestView{}
	for rows.Next() {
		var v branchRequestView
		if err := rows.Scan(&v.ID, &v.ParentMerchantID, &v.ParentName, &v.Name, &v.Phone, &v.AddressText,
			&v.Lat, &v.Lng, &v.Status, &v.CommissionPct, &v.RejectReason, &v.RequestedByRole,
			&v.RequestedByName, &v.OwnerName, &v.OwnerPhone, &v.BranchMerchantID, &v.CreatedAt, &v.DecidedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// branchCreateBody **جسمُ الطلب** — اسمٌ وموقعٌ إلزاميّان، والهاتفُ والعنوانُ اختياريّان.
type branchCreateBody struct {
	Name        string   `json:"name"`
	Phone       *string  `json:"phone"`
	AddressText *string  `json:"address_text"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
}

// createBranchRequest **الجسمُ المشترك للبابين** (المتجر والمندوب) بعد أن يتحقّق كلٌّ من ملكيّته.
func (s *Server) createBranchRequest(w http.ResponseWriter, r *http.Request, parentID, role string) {
	req, err := decode[branchCreateBody](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || req.Lat == nil || req.Lng == nil {
		s.respondErr(w, errValidation)
		return
	}
	lat, lng := *req.Lat, *req.Lng
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || (lat == 0 && lng == 0) {
		s.respondErr(w, errValidation)
		return
	}
	var phone *string
	if req.Phone != nil && strings.TrimSpace(*req.Phone) != "" {
		p, ok := identity.NormalizePhone(*req.Phone)
		if !ok {
			s.respondErr(w, errValidation)
			return
		}
		phone = &p
	}
	addr := req.AddressText
	if addr != nil && strings.TrimSpace(*addr) == "" {
		addr = nil
	}
	if _, err := s.guardText(r.Context(),
		tf("name", &name, maxStoreName, textguard.Name),
		tf("address_text", addr, maxAddressText, textguard.Address),
	); err != nil {
		s.respondErr(w, err)
		return
	}
	var open int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM branch_requests WHERE parent_merchant_id = $1 AND status = 'pending'`,
		parentID).Scan(&open); err != nil {
		s.respondErr(w, err)
		return
	}
	if open >= branchPendingMax {
		s.respondErr(w, errBranchPending)
		return
	}
	var id string
	err = s.pg.QueryRow(r.Context(), `
		INSERT INTO branch_requests (parent_merchant_id, requested_by, requested_by_role, name, phone, address_text, lat, lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text`,
		parentID, userIDFrom(r), role, name, phone, addr, lat, lng).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "branch_requests_pending_name_uq") {
			s.respondErr(w, errBranchPending)
			return
		}
		s.respondErr(w, err)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "branch.request", "merchant", parentID,
		map[string]any{"request": id, "role": role})
	rows, err := s.pg.Query(r.Context(), branchViewSQL+`WHERE b.id = $1`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := scanBranchViews(rows)
	if err != nil || len(list) == 0 {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, list[0])
}

// handleMerchantBranchCreate **صاحبُ المتجر يطلب فرعاً** — POST /merchant/stores/{id}/branch-requests
func (s *Server) handleMerchantBranchCreate(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, parentID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.createBranchRequest(w, r, parentID, "merchant")
}

// handleMerchantBranchList **كلُّ طلباته عبر متاجره** — GET /merchant/branch-requests
func (s *Server) handleMerchantBranchList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(),
		branchViewSQL+`WHERE pm.owner_user_id = $1 ORDER BY b.created_at DESC LIMIT 200`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := scanBranchViews(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": list})
}

// repOwnsMerchant **أهذا المتجرُ مسنَدٌ إلى هذا المندوب** — وإلّا `errForbidden` بلا كشف الوجود.
func (s *Server) repOwnsMerchant(ctx context.Context, repID, merchantID string) bool {
	if !isUUID(merchantID) {
		return false
	}
	var ok bool
	err := s.pg.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM merchants WHERE id = $1 AND sales_rep_user_id = $2)`,
		merchantID, repID).Scan(&ok)
	return err == nil && ok
}

// handleRepBranchCreate **المندوبُ يطلب فرعاً لعميله** — POST /rep/merchants/{id}/branch-requests
func (s *Server) handleRepBranchCreate(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	if !s.repOwnsMerchant(r.Context(), userIDFrom(r), parentID) {
		s.respondErr(w, errForbidden)
		return
	}
	s.createBranchRequest(w, r, parentID, "sales")
}

// handleRepBranchList **طلباتُ فروع عميله** — GET /rep/merchants/{id}/branch-requests
func (s *Server) handleRepBranchList(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	if !s.repOwnsMerchant(r.Context(), userIDFrom(r), parentID) {
		s.respondErr(w, errForbidden)
		return
	}
	rows, err := s.pg.Query(r.Context(),
		branchViewSQL+`WHERE b.parent_merchant_id = $1 ORDER BY b.created_at DESC LIMIT 100`, parentID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := scanBranchViews(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": list})
}

// handleAdminBranchList **طابورُ المكتب** — افتراضُه «بانتظار الموافقة». GET /admin/branch-requests?status=
func (s *Server) handleAdminBranchList(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	switch status {
	case "":
		status = branchPending
	case branchPending, branchApproved, branchRejected, "all":
	default:
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(),
		branchViewSQL+`WHERE ($1 = 'all' OR b.status = $1) ORDER BY b.created_at DESC LIMIT 300`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	list, err := scanBranchViews(rows)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنسبةُ المقترحةُ لكلّ طلبٍ مفتوح** = النسبةُ الفعليّةُ للمتجر الأمّ (خاصّتُه أو العامّة).
	general := s.generalCommission(r.Context())
	type withDefault struct {
		branchRequestView
		DefaultCommission pricing.Percent2 `json:"default_commission_percent"`
	}
	out := make([]withDefault, 0, len(list))
	for _, v := range list {
		d := general
		if v.Status == branchPending {
			var ov *pricing.Percent2
			if err := s.pg.QueryRow(r.Context(),
				`SELECT commission_percent FROM merchants WHERE id = $1`, v.ParentMerchantID).Scan(&ov); err == nil && ov != nil {
				d = *ov
			}
		}
		out = append(out, withDefault{v, d})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": out, "general_commission_percent": general})
}

// handleAdminBranchCount **شارةُ القائمة** — GET /admin/branch-requests/count
func (s *Server) handleAdminBranchCount(w http.ResponseWriter, r *http.Request) {
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM branch_requests WHERE status = 'pending'`).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"pending": n})
}

// handleAdminBranchApprove **موافقةٌ بعمولة** — POST /admin/branch-requests/{id}/approve
//
// الجسم: `{"commission_percent": 7.5}` أو بلا الحقل = نسبةُ الأمّ (خاصّتُه أو «اتبع العامّ»).
func (s *Server) handleAdminBranchApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	body, err := decode[struct {
		CommissionPct *pricing.Percent2 `json:"commission_percent"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor, ip := userIDFrom(r), clientIP(r)
	ctx := r.Context()

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **القفلُ ثمّ الفحصُ تحته** — نقرتان من مكتبَين لا تصنعان فرعين.
	var (
		parentID, name, role, requester string
		phone, addr                     *string
		lat, lng                        float64
		status                          string
	)
	err = tx.QueryRow(ctx, `
		SELECT parent_merchant_id::text, name, requested_by_role, requested_by::text, phone, address_text, lat, lng, status
		  FROM branch_requests WHERE id = $1 FOR UPDATE`, id).
		Scan(&parentID, &name, &role, &requester, &phone, &addr, &lat, &lng, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if status != branchPending {
		s.respondErr(w, errBranchDecided)
		return
	}

	var (
		categoryID, ownerPhone, parentPhone, settlement, display string
		districtID, repID                                        *string
		override                                                 *pricing.Percent2
		settleAt, returnsAt                                      *time.Time
		acceptsReturns, usesAmeen                                bool
		exchange                                                 *float64
	)
	err = tx.QueryRow(ctx, `
		SELECT m.category_id::text, u.phone, m.phone, m.district_id::text, m.sales_rep_user_id::text,
		       m.commission_percent, m.settlement_method, m.settlement_confirmed_at,
		       m.accepts_returns, m.returns_confirmed_at, m.display_currency, m.exchange_rate::float8, m.uses_ameen
		  FROM merchants m JOIN users u ON u.id = m.owner_user_id
		 WHERE m.id = $1`, parentID).
		Scan(&categoryID, &ownerPhone, &parentPhone, &districtID, &repID,
			&override, &settlement, &settleAt,
			&acceptsReturns, &returnsAt, &display, &exchange, &usesAmeen)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	pct := override
	if body.CommissionPct != nil {
		pct = body.CommissionPct
	}
	contact := parentPhone
	if phone != nil {
		contact = *phone
	}
	in := catalog.MerchantInput{
		Name: &name, CategoryID: &categoryID, Phone: &contact, AddressText: addr,
		OwnerPhone: &ownerPhone, Lat: &lat, Lng: &lng, DistrictID: districtID, CommissionPct: pct,
	}
	branchID, err := s.catalog.CreateMerchantTx(ctx, tx, actor, in, ip)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ما يُنسخ من الأمّ** — طريقةُ التسوية والعملة وسياسةُ الإرجاع والمندوب، كلُّها إعداداتٌ لا مالٌ جارٍ.
	if _, err := tx.Exec(ctx, `
		UPDATE merchants SET settlement_method = $2, settlement_confirmed_at = $3,
		       accepts_returns = $4, returns_confirmed_at = $5, display_currency = $6,
		       exchange_rate = $7, uses_ameen = $8, sales_rep_user_id = $9::uuid
		 WHERE id = $1`,
		branchID, settlement, settleAt, acceptsReturns, returnsAt, display, exchange, usesAmeen, repID); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE branch_requests SET status = 'approved', commission_percent = $2, decided_by = $3,
		       decided_at = now(), branch_merchant_id = $4 WHERE id = $1`,
		id, pct, actor, branchID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}

	s.auditCtx(ctx, actor, ip, "branch.approve", "merchant", branchID,
		map[string]any{"request": id, "parent": parentID})
	// **بعد التثبيت لا قبله** — إشعارٌ خرج ثمّ ارتدّت المعاملةُ كذبٌ لا يُسحَب.
	s.notifyBranchDecision(ctx, requester, role, id, true,
		"تمت الموافقة على فرعك",
		"فرع «"+name+"» صار جاهزاً. بدّل إليه من قائمة المتاجر.")
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "status": branchApproved, "branch_merchant_id": branchID})
}

// handleAdminBranchReject **رفضٌ بسببٍ اختياريّ** — POST /admin/branch-requests/{id}/reject
func (s *Server) handleAdminBranchReject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	body, err := decode[struct {
		Reason *string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var reason *string
	if body.Reason != nil && strings.TrimSpace(*body.Reason) != "" {
		t := strings.TrimSpace(*body.Reason)
		if _, err := s.guardText(r.Context(), tf("reason", &t, maxAddressText, textguard.Notes)); err != nil {
			s.respondErr(w, err)
			return
		}
		reason = &t
	}
	var (
		name, role, requester string
	)
	err = s.pg.QueryRow(r.Context(), `
		UPDATE branch_requests SET status = 'rejected', reject_reason = $2, decided_by = $3, decided_at = now()
		 WHERE id = $1 AND status = 'pending'
		RETURNING name, requested_by_role, requested_by::text`,
		id, reason, userIDFrom(r)).Scan(&name, &role, &requester)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = s.pg.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM branch_requests WHERE id = $1)`, id).Scan(&exists)
		if exists {
			s.respondErr(w, errBranchDecided)
		} else {
			s.respondErr(w, httpx.ErrNotFound)
		}
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "branch.reject", "branch_request", id, nil)
	body2 := "تم رفض طلب فرع «" + name + "»."
	if reason != nil {
		body2 += " السبب: " + *reason
	}
	s.notifyBranchDecision(r.Context(), requester, role, id, false, "رُفض طلب الفرع", body2)
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "status": branchRejected})
}

// notifyBranchDecision **يبلّغ صاحبَ الطلب بقرار المكتب** في التطبيق الذي طلب منه.
func (s *Server) notifyBranchDecision(ctx context.Context, userID, role, requestID string, approved bool, title, body string) {
	app := notifications.AppMerchant
	if role == "sales" {
		app = notifications.AppRep
	}
	_ = approved
	s.notify.Notify(ctx, notifications.Input{
		UserID: userID, Kind: notifications.KindAccount, Title: title, Body: body,
		Entity: "branch_request", EntityID: requestID,
		Apps: []string{app},
	})
}
