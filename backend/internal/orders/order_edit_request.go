package orders

// ══════════════════════════════════════════════════════════════════════
// **طلب تعديل من الكابتن — والمكتب يوافق أو يرفض** (قرار المالك ٢٠٢٦-١١-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// «كلشي لازم يمشي على المبالغ الحقيقية مو على أرقام الطلب الأصلي»: صنف ناقص بالمتجر، أو الزبون
// لغى صنف، أو انزاد صنف — **والزبون لازم يدفع السعر الحقيقي الأخير.**
//
// **الكابتن ما بيغيّر المصاري بإيده** بالطلب العادي. بيبعت طلب:
//
//	سطر «ناقص»       ←  المتجر ما سلّمه ⇒ حق المتجر بينقص (`EditLine.StoreRefunded` = نعم)
//	سطر «الزبون لغاه» ←  بعد الاستلام حق المتجر كامل والمنصة بتتحمّل (قرار المالك ٢٠٢٦-١١-٠٣، البند ٢)
//	«انزاد صنف»       ←  صنف من خارج القائمة باسمه وسعره (`EditCustom`)
//
// **والموافقة بتمشي من باب تعديل المكتب نفسه** (`EditOrder`) — نفس الحسبة ونفس قيود التصحيح بعد
// الاستلام ونفس الإشعارات (الزبون بمجموعه الجديد، والكابتن بالمبلغ اللي بيقبضه، والمتجر بحقّه)،
// **ونفس القدرة**: بعد الاستلام للأدمن والمالك بس (`orders.edit_after_pickup`، البند ٤).
//
// **والطلب ما بيتغيّر تحت إيد المكتب**: بصمة سطور الطلب ومجموعه بتنحفظ وقت الطلب، وإذا تغيّرت
// قبل الموافقة (تعديل تاني، صنف غير متوفر…) الموافقة بتنرد `edit_request_stale` والطلب بيصير
// «قديم» — والكابتن بيبعت غيره.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// أفعال سطر طلب التعديل.
const (
	EditRequestMissing   = "missing"
	EditRequestCancelled = "cancelled"
)

// حالات طلب التعديل.
const (
	EditRequestPending    = "pending"
	EditRequestApproved   = "approved"
	EditRequestRejected   = "rejected"
	EditRequestSuperseded = "superseded"
	EditRequestStale      = "stale"
	EditRequestClosed     = "closed"
	// EditRequestWithdrawn **الكابتن سحب طلبه بنفسه.**
	EditRequestWithdrawn = "withdrawn"
)

var (
	// ErrEditRequestStage **مو وقته** — من قبول الكابتن لحد التسليم بس.
	ErrEditRequestStage = httpx.NewError(http.StatusConflict,
		"edit_request_stage", "errors.edit_request_stage")
	// ErrEditRequestEmpty **ما في شي يتعدّل** — لا سطر ولا صنف.
	ErrEditRequestEmpty = httpx.NewError(http.StatusBadRequest,
		"edit_request_empty", "errors.edit_request_empty")
	// ErrEditRequestStale **الطلب تغيّر من وقت ما انبعت الطلب** — الكابتن بيبعت غيره.
	ErrEditRequestStale = httpx.NewError(http.StatusConflict,
		"edit_request_stale", "errors.edit_request_stale")
	// ErrEditRequestNotPending **انقرّر أو انبدّل بأحدث.**
	ErrEditRequestNotPending = httpx.NewError(http.StatusConflict,
		"edit_request_not_pending", "errors.edit_request_not_pending")
	// ErrEditRequestRejectReason **سبب الرفض إجباري** — الكابتن لازم يعرف ليش.
	ErrEditRequestRejectReason = httpx.NewError(http.StatusBadRequest,
		"edit_request_reject_reason", "errors.edit_request_reject_reason")
	// ErrEditRequestPendingDeliver **ما بيتسلّم الطلب وطلب تعديل الكابتن بانتظار المكتب** (قرار المالك
	// ٢٠٢٦-١١-١٠) — الزبون لازم يدفع السعر الحقيقي الأخير، فالمكتب يقرّر أو الكابتن يسحب طلبه.
	ErrEditRequestPendingDeliver = httpx.NewError(http.StatusConflict,
		"edit_request_pending_deliver", "errors.edit_request_pending_deliver")
)

// editRequestStages **من قبول الكابتن لحد التسليم.**
var editRequestStages = map[string]bool{
	StAssigned: true, StAtPickup: true, StPickedUp: true, StOnTheWay: true, StAtDropoff: true,
}

// EditRequestLine **سطر قائم بالطلب** — `Qty` كم قطعة بتنشال منه (صفر = السطر كلّه).
type EditRequestLine struct {
	ItemID string `json:"item_id"`
	Action string `json:"action"`
	Qty    int    `json:"qty"`
}

// EditRequestAdd **صنف انزاد** — اسمه وكمّيته وسعر الوحدة بالليرة.
//
// `Price` **سعرُ المتجر** (اللي دفعه الكابتن) — وسعرُ الزبون = سعرُه + هامشُ المنصّة كأيّ صنف يُطلب.
// `MerchantID` **المتجر اللي جاب منه الصنف** — وفاضي: متجر الطلب الرئيسي. (طلب بأكثر من متجر: الكابتن بيختار.)
type EditRequestAdd struct {
	Name         string `json:"name"`
	Qty          int    `json:"qty"`
	Price        int64  `json:"price"`
	MerchantID   string `json:"merchant_id,omitempty"`
	MerchantName string `json:"merchant_name,omitempty"`
	// PriceUSD **سعرُ القطعة بالدولار (سنتات)** إن كتبه الكابتن دولاراً (قرارُ المالك ٢٠٢٦-١١-١٢).
	// **يُحوَّل إلى `Price` بالليرة بسعر صرف المتجر مدوَّراً لأقرب ٥** — والحسابُ كلُّه بالليرة كما كان،
	// **والدولارُ للعرض وحدَه.**
	PriceUSD int64 `json:"price_usd,omitempty"`
}

// DriverEditInput **ما يبعته الكابتن.**
type DriverEditInput struct {
	Lines []EditRequestLine `json:"lines"`
	Adds  []EditRequestAdd  `json:"adds"`
	Note  string            `json:"note"`
}

// editRequestLineRow **سطر محفوظ مع لقطته** — الاسم والكمّية والسعر وقت الطلب للعرض والحساب.
type editRequestLineRow struct {
	ItemID    string `json:"item_id"`
	Action    string `json:"action"`
	Remove    int    `json:"remove"`
	Name      string `json:"name"`
	OldQty    int    `json:"old_qty"`
	UnitPrice int64  `json:"unit_price"`
}

// EditRequestLineView **سطر بالعرض** — قبل وبعد.
type EditRequestLineView struct {
	ItemID    string `json:"item_id"`
	Name      string `json:"name"`
	Action    string `json:"action"`
	OldQty    int    `json:"old_qty"`
	NewQty    int    `json:"new_qty"`
	UnitPrice int64  `json:"unit_price"`
}

// EditRequestView **طلب التعديل بالعرض** — للكابتن والمكتب.
type EditRequestView struct {
	ID           string                `json:"id"`
	OrderID      string                `json:"order_id"`
	Status       string                `json:"status"`
	Lines        []EditRequestLineView `json:"lines"`
	Adds         []EditRequestAdd      `json:"adds"`
	Note         string                `json:"note"`
	OldTotal     int64                 `json:"old_total"`
	NewTotal     int64                 `json:"new_total"`
	RejectReason string                `json:"reject_reason,omitempty"`
	CreatedAt    string                `json:"created_at"`
	DecidedAt    *string               `json:"decided_at,omitempty"`
	// للمكتب وحده: معاينة المحرّك الآن، وهل الطلب بعد الاستلام، وهل تغيّر الطلب.
	AfterPickup  bool        `json:"after_pickup"`
	Stale        bool        `json:"stale"`
	Preview      *EditResult `json:"preview,omitempty"`
	PreviewError string      `json:"preview_error,omitempty"`
}

// editFingerprintTx **بصمة سطور الطلب ومجموعه** — أي تغيير بالكمّية أو السعر أو السطور أو المجموع بيغيّرها.
func editFingerprintTx(ctx context.Context, q dbtx.Querier, orderID string) (string, error) {
	var fp string
	err := q.QueryRow(ctx, `
		SELECT md5(COALESCE((SELECT string_agg(id::text || ':' || qty || ':' || unit_price || ':' ||
		                                       merchant_price || ':' || returned::text, ',' ORDER BY id)
		                       FROM order_items WHERE order_id = $1), '')
		           || '|' || (SELECT total::text FROM orders WHERE id = $1))`, orderID).Scan(&fp)
	return fp, err
}

// editRequestReason **سبب التعديل بكلمة** — بيوصل الزبون بإشعار التعديل.
func editRequestReason(lines []editRequestLineRow, adds []EditRequestAdd, note string) string {
	var parts []string
	var missing, cancelled bool
	for _, l := range lines {
		if l.Action == EditRequestMissing {
			missing = true
		} else {
			cancelled = true
		}
	}
	if missing {
		parts = append(parts, "صنف ناقص بالمتجر")
	}
	if cancelled {
		parts = append(parts, "الزبون لغى صنف")
	}
	if len(adds) > 0 {
		parts = append(parts, "انزاد صنف")
	}
	reason := "طلب الكابتن: " + strings.Join(parts, "، ")
	if note != "" {
		reason += " — " + note
	}
	return reason
}

// editRequestToEdit **طلب الكابتن بلغة تعديل المكتب** (`EditRequest`).
func editRequestToEdit(lines []editRequestLineRow, adds []EditRequestAdd, note string) EditRequest {
	req := EditRequest{Reason: editRequestReason(lines, adds, note)}
	for _, l := range lines {
		refunded := l.Action == EditRequestMissing
		req.Lines = append(req.Lines, EditLine{ItemID: l.ItemID, Qty: l.OldQty - l.Remove, StoreRefunded: &refunded})
	}
	for _, a := range adds {
		req.Custom = append(req.Custom, EditCustom{Name: a.Name, Qty: a.Qty, Price: a.Price,
			MerchantID: a.MerchantID, PlatformMargin: true})
	}
	req.byDriverRequest = true
	return req
}

func badEditInput() error {
	return httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
}

// PreparedEdit **طلب تعديل انفحص ولسا ما انحفظ** — بين `PrepareOrderEdit` و`SaveOrderEditTx`.
type PreparedEdit struct {
	orderID, driverID, note string
	lines                   []editRequestLineRow
	adds                    []EditRequestAdd
	oldTotal, newTotal      int64
	// بعد الحفظ
	id     string
	number int64
}

// ID **معرّف الطلب بعد الحفظ.**
func (p *PreparedEdit) ID() string { return p.id }

// RequestOrderEdit **الكابتن يبعت طلب تعديل** — فحص ثمّ حفظ بمعاملته ثمّ إخبار. والباب بيستعمل
// الثلاثة لحاله ليحفظ بمعاملة منع التكرار (`WithIdempotentTx`).
func (s *Service) RequestOrderEdit(ctx context.Context, orderID, driverID string, in DriverEditInput) (*EditRequestView, error) {
	p, err := s.PrepareOrderEdit(ctx, orderID, driverID, in)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	v, err := s.SaveOrderEditTx(ctx, tx, p)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.AfterOrderEdit(ctx, p)
	return v, nil
}

// PrepareOrderEdit **يفحص طلب الكابتن** — والجديد بيحلّ محل المعلّق وقت الحفظ.
//
// **والمعاينة بالمحرّك نفسه قبل الحفظ** (`EditOrder` بلا تثبيت): طلب ما بيقبله التعديل (آخر صنف،
// تسوية قديمة…) بينرد هلّق مو وقت الموافقة، والمجموع المتوقّع بيوصل الكابتن والمكتب.
func (s *Service) PrepareOrderEdit(ctx context.Context, orderID, driverID string, in DriverEditInput) (*PreparedEdit, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	in.Note = strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(in.Note) > 300 {
		return nil, badEditInput()
	}
	if len(in.Lines) == 0 && len(in.Adds) == 0 {
		return nil, ErrEditRequestEmpty
	}
	if len(in.Lines) > 50 || len(in.Adds) > 20 {
		return nil, badEditInput()
	}
	var kind, status, mainMerchant string
	var driver *string
	err := s.db.QueryRow(ctx, `SELECT kind, status, driver_id::text, COALESCE(merchant_id::text, '')
		FROM orders WHERE id = $1`, orderID).
		Scan(&kind, &status, &driver, &mainMerchant)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if driver == nil || *driver != driverID {
		return nil, httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	if kind != KindStandard {
		return nil, ErrOrderEditKind
	}
	if !editRequestStages[status] {
		return nil, ErrEditRequestStage
	}

	lines := make([]editRequestLineRow, 0, len(in.Lines))
	seen := map[string]bool{}
	for _, l := range in.Lines {
		if _, err := uuid.Parse(l.ItemID); err != nil || seen[l.ItemID] || l.Qty < 0 ||
			(l.Action != EditRequestMissing && l.Action != EditRequestCancelled) {
			return nil, badEditInput()
		}
		seen[l.ItemID] = true
		row := editRequestLineRow{ItemID: l.ItemID, Action: l.Action}
		var returned bool
		err := s.db.QueryRow(ctx, `
			SELECT name, qty, unit_price, returned FROM order_items WHERE id = $1 AND order_id = $2`,
			l.ItemID, orderID).Scan(&row.Name, &row.OldQty, &row.UnitPrice, &returned)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		if returned {
			return nil, badEditInput()
		}
		row.Remove = l.Qty
		if row.Remove == 0 {
			row.Remove = row.OldQty
		}
		if row.Remove > row.OldQty {
			return nil, badEditInput()
		}
		lines = append(lines, row)
	}
	adds := make([]EditRequestAdd, 0, len(in.Adds))
	var stores map[string]bool
	for _, a := range in.Adds {
		a.Name = strings.TrimSpace(a.Name)
		a.MerchantName = ""
		if a.PriceUSD < 0 || a.PriceUSD > 10_000_000 {
			return nil, badEditInput()
		}
		if a.PriceUSD > 0 {
			// **بسعر صرف المتجر الذي جاء منه الصنف** — وبلا سعرٍ له: السعرُ المرجعيّ؛ وبلا هذا ولا ذاك يُرفض.
			store := a.MerchantID
			if store == "" {
				store = mainMerchant
			}
			var rate *float64
			if store != "" {
				if _, perr := uuid.Parse(store); perr == nil {
					if err := s.db.QueryRow(ctx, `SELECT exchange_rate::float8 FROM merchants WHERE id = $1::uuid`, store).
						Scan(&rate); err != nil && !errors.Is(err, pgx.ErrNoRows) {
						return nil, err
					}
				}
			}
			r := 0.0
			if rate != nil {
				r = *rate
			}
			if r <= 0 {
				r = s.usdReferenceRate(ctx, s.db)
			}
			if a.Price = usdToSYP(a.PriceUSD, r); a.Price <= 0 {
				return nil, badEditInput()
			}
		}
		if a.Name == "" || utf8.RuneCountInString(a.Name) > 120 || a.Qty < 1 || a.Qty > 50 ||
			a.Price < 1 || a.Price > 1_000_000_000 {
			return nil, badEditInput()
		}
		// **المتجر اللي جاب منه الصنف**: من متاجر الطلب بس، واسمه بيتحفظ للعرض. وفاضي ⇒ الرئيسي.
		if a.MerchantID != "" {
			if _, err := uuid.Parse(a.MerchantID); err != nil {
				return nil, badEditInput()
			}
			if stores == nil {
				if stores, err = orderStoreSet(ctx, s.db, orderID, mainMerchant); err != nil {
					return nil, err
				}
			}
			if !stores[a.MerchantID] {
				return nil, ErrOrderEditStore
			}
			if err := s.db.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1::uuid`, a.MerchantID).
				Scan(&a.MerchantName); err != nil {
				return nil, err
			}
		}
		adds = append(adds, a)
	}

	// **معاينة المحرّك** — كل قيود التعديل بتنفحص هلّق، والمجموع المتوقّع منها.
	edit := editRequestToEdit(lines, adds, in.Note)
	edit.DryRun = true
	_, preview, err := s.EditOrder(ctx, driverID, orderID, edit, true, nil)
	if err != nil {
		return nil, err
	}

	return &PreparedEdit{orderID: orderID, driverID: driverID, lines: lines, adds: adds, note: in.Note,
		oldTotal: preview.OldTotal, newTotal: preview.NewTotal}, nil
}

// SaveOrderEditTx **يحفظ الطلب بالمعاملة المعطاة** — قفل الطلب، والبصمة، وتعليم المعلّق «انبدّل»،
// وسطر بسجلّ الطلب. وبيرجّع الطلب كما بيشوفه الكابتن.
func (s *Service) SaveOrderEditTx(ctx context.Context, q dbtx.Querier, p *PreparedEdit) (*EditRequestView, error) {
	linesJSON, _ := json.Marshal(p.lines)
	addsJSON, _ := json.Marshal(p.adds)
	var status string
	var driver *string
	if err := q.QueryRow(ctx, `SELECT status, driver_id::text, number FROM orders WHERE id = $1 FOR UPDATE`, p.orderID).
		Scan(&status, &driver, &p.number); err != nil {
		return nil, err
	}
	if driver == nil || *driver != p.driverID || !editRequestStages[status] {
		return nil, ErrEditRequestStage
	}
	fp, err := editFingerprintTx(ctx, q, p.orderID)
	if err != nil {
		return nil, err
	}
	if _, err := q.Exec(ctx, `
		UPDATE order_edit_requests SET status = 'superseded', decided_at = now()
		 WHERE order_id = $1 AND status = 'pending'`, p.orderID); err != nil {
		return nil, err
	}
	r, err := scanEditRequest(q.QueryRow(ctx, `
		INSERT INTO order_edit_requests (order_id, driver_id, lines, adds, note, base_fingerprint,
		                                 old_total, new_total)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+editRequestCols,
		p.orderID, p.driverID, linesJSON, addsJSON, p.note, fp, p.oldTotal, p.newTotal))
	if err != nil {
		return nil, err
	}
	p.id = r.id
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $3)`, p.orderID, status, p.driverID,
		fmt.Sprintf("الكابتن طلب تعديل: %s — المجموع %s ← %s ل.س، بانتظار المكتب",
			editRequestSummary(p.lines, p.adds), groupDigits(p.oldTotal), groupDigits(p.newTotal))); err != nil {
		return nil, err
	}
	return r.view(), nil
}

// AfterOrderEdit **بعد التثبيت**: المكتب بينخبر، والطلب بينبثّ (اللوحة بتحدّث كرتها).
func (s *Service) AfterOrderEdit(ctx context.Context, p *PreparedEdit) {
	if s.notify != nil {
		body := fmt.Sprintf("#%d — %s. المجموع %s ← %s ل.س", p.number, editRequestSummary(p.lines, p.adds),
			groupDigits(p.oldTotal), groupDigits(p.newTotal))
		if p.note != "" {
			body += " — " + p.note
		}
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindOrder, Title: t.editRequestOps, Body: body,
			Entity: "order", EntityID: p.orderID, Href: "/dashboard/orders?id=" + p.orderID,
		})
	}
	if o, err := s.GetByID(ctx, p.orderID); err == nil {
		s.publishOrder(o)
	}
}

// editRequestSummary **السطور بكلمة** — «ناقص: برغر ×١، لغاه الزبون: كولا ×٢، انزاد: خبز ×٣».
func editRequestSummary(lines []editRequestLineRow, adds []EditRequestAdd) string {
	var parts []string
	for _, l := range lines {
		word := "ناقص"
		if l.Action == EditRequestCancelled {
			word = "الزبون لغاه"
		}
		parts = append(parts, fmt.Sprintf("%s: %s ×%d", word, l.Name, l.Remove))
	}
	for _, a := range adds {
		part := fmt.Sprintf("انزاد: %s ×%d بسعر %s", a.Name, a.Qty, groupDigits(a.Price))
		if a.MerchantName != "" {
			part += " من " + a.MerchantName
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "، ")
}

type editRequestRecord struct {
	id, orderID, driverID, status, note, fingerprint, rejectReason string
	lines                                                          []editRequestLineRow
	adds                                                           []EditRequestAdd
	oldTotal, newTotal                                             int64
	createdAt                                                      string
	decidedAt                                                      *string
}

const editRequestCols = `id::text, order_id::text, driver_id::text, status, note, base_fingerprint,
	reject_reason, lines, adds, old_total, new_total,
	to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	to_char(decided_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanEditRequest(row pgx.Row) (*editRequestRecord, error) {
	var r editRequestRecord
	var lj, aj []byte
	if err := row.Scan(&r.id, &r.orderID, &r.driverID, &r.status, &r.note, &r.fingerprint,
		&r.rejectReason, &lj, &aj, &r.oldTotal, &r.newTotal, &r.createdAt, &r.decidedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(lj, &r.lines); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aj, &r.adds); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *editRequestRecord) view() *EditRequestView {
	v := &EditRequestView{
		ID: r.id, OrderID: r.orderID, Status: r.status, Note: r.note,
		OldTotal: r.oldTotal, NewTotal: r.newTotal, RejectReason: r.rejectReason,
		CreatedAt: r.createdAt, DecidedAt: r.decidedAt,
		Lines: []EditRequestLineView{}, Adds: r.adds,
	}
	if v.Adds == nil {
		v.Adds = []EditRequestAdd{}
	}
	for _, l := range r.lines {
		v.Lines = append(v.Lines, EditRequestLineView{ItemID: l.ItemID, Name: l.Name, Action: l.Action,
			OldQty: l.OldQty, NewQty: l.OldQty - l.Remove, UnitPrice: l.UnitPrice})
	}
	return v
}

// DriverEditRequest **آخر طلب تعديل بعته الكابتن على هالطلب** — وفاضي: ما في.
func (s *Service) DriverEditRequest(ctx context.Context, orderID, driverID string) (*EditRequestView, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	r, err := scanEditRequest(s.db.QueryRow(ctx, `SELECT `+editRequestCols+` FROM order_edit_requests
		WHERE order_id = $1 AND driver_id = $2 ORDER BY created_at DESC LIMIT 1`, orderID, driverID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.view(), nil
}

// PendingEditRequest **طلب التعديل المعلّق للمكتب — مع معاينة المحرّك الآن.**
//
// **والمعاينة بنفس باب الموافقة** فالرقم اللي بيشوفه المكتب هو اللي رح يصير. وطلب تغيّر من وقتها
// بينقال «قديم» بلا معاينة.
// `actorID` مين عم يشوف — المعاينة بعد الاستلام بتكتب قيود تصحيح باسمه وبتنطوى.
func (s *Service) PendingEditRequest(ctx context.Context, orderID, actorID string) (*EditRequestView, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	r, err := scanEditRequest(s.db.QueryRow(ctx, `SELECT `+editRequestCols+` FROM order_edit_requests
		WHERE order_id = $1 AND status = 'pending'`, orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := r.view()
	fp, err := editFingerprintTx(ctx, s.db, orderID)
	if err != nil {
		return nil, err
	}
	if fp != r.fingerprint {
		v.Stale = true
		return v, nil
	}
	edit := editRequestToEdit(r.lines, r.adds, r.note)
	edit.DryRun = true
	_, res, err := s.EditOrder(ctx, actorID, orderID, edit, true, nil)
	if err != nil {
		var ae *httpx.AppError
		if errors.As(err, &ae) {
			v.PreviewError = ae.Code
			return v, nil
		}
		return nil, err
	}
	v.Preview = &res
	v.AfterPickup = res.AfterPickup
	return v, nil
}

// ApproveEditRequest **المكتب يوافق ⇒ التعديل بيمشي من باب تعديل الطلب نفسه** (`EditOrder`).
//
// `afterPickupAllowed` قدرة الموافِق بعد الاستلام (البند ٤)، و`hook` أثره بالمعاملة نفسها (التدقيق).
// **وطلب تغيّر تحته بينرد `edit_request_stale` وبيتعلّم «قديم».**
func (s *Service) ApproveEditRequest(ctx context.Context, actorID, orderID, requestID string,
	afterPickupAllowed bool,
	hook func(context.Context, dbtx.Querier, EditResult) error) (*Order, EditResult, error) {
	if _, err := uuid.Parse(requestID); err != nil {
		return nil, EditResult{}, httpx.ErrNotFound
	}
	r, err := scanEditRequest(s.db.QueryRow(ctx, `SELECT `+editRequestCols+` FROM order_edit_requests
		WHERE id = $1 AND order_id = $2`, requestID, orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, EditResult{}, httpx.ErrNotFound
	}
	if err != nil {
		return nil, EditResult{}, err
	}
	if r.status != EditRequestPending {
		return nil, EditResult{}, ErrEditRequestNotPending
	}
	edit := editRequestToEdit(r.lines, r.adds, r.note)
	edit.guard = func(ctx context.Context, tx pgx.Tx) error {
		var st string
		if err := tx.QueryRow(ctx, `SELECT status FROM order_edit_requests WHERE id = $1 FOR UPDATE`,
			requestID).Scan(&st); err != nil {
			return err
		}
		if st != EditRequestPending {
			return ErrEditRequestNotPending
		}
		fp, err := editFingerprintTx(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if fp != r.fingerprint {
			return ErrEditRequestStale
		}
		return nil
	}
	o, res, err := s.EditOrder(ctx, actorID, orderID, edit, afterPickupAllowed,
		func(ctx context.Context, q dbtx.Querier, res EditResult) error {
			resJSON, _ := json.Marshal(res)
			if _, err := q.Exec(ctx, `
				UPDATE order_edit_requests SET status = 'approved', decided_by = NULLIF($2, '')::uuid,
				       decided_at = now(), result = $3
				 WHERE id = $1`, requestID, actorID, resJSON); err != nil {
				return err
			}
			if hook != nil {
				return hook(ctx, q, res)
			}
			return nil
		})
	if errors.Is(err, ErrEditRequestStale) {
		if _, xerr := s.db.Exec(ctx, `
			UPDATE order_edit_requests SET status = 'stale', decided_at = now(), decided_by = NULLIF($2, '')::uuid
			 WHERE id = $1 AND status = 'pending'`, requestID, actorID); xerr != nil {
			s.logger.Warn("طلب تعديل الكابتن: تعذّر تعليمه قديم", "error", xerr)
		}
		s.notifyEditRequestDecision(ctx, orderID, r.driverID,
			"الطلب تغيّر من وقت ما بعتّ — ابعت طلب جديد إذا لسا لازم")
		if o, gerr := s.GetByID(ctx, orderID); gerr == nil {
			s.publishOrder(o)
		}
		return nil, res, err
	}
	if err != nil {
		return nil, res, err
	}
	// **والكابتن بيعرف بالموافقة من إشعار التعديل نفسه** (`editTrailTx`: «تعدّل الطلب الذي معك» بالمبلغ
	// اللي بيقبضه) — ما في إشعار تاني عن نفس الشي.
	return o, res, nil
}

// RejectEditRequest **المكتب يرفض بسبب** — الطلب ما بيتغيّر، والكابتن بيعرف ليش.
func (s *Service) RejectEditRequest(ctx context.Context, actorID, orderID, requestID, reason string,
	hook func(context.Context, dbtx.Querier) error) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrEditRequestRejectReason
	}
	if utf8.RuneCountInString(reason) > 300 {
		return badEditInput()
	}
	if _, err := uuid.Parse(requestID); err != nil {
		return httpx.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var driverID, status string
	err = tx.QueryRow(ctx, `
		UPDATE order_edit_requests r SET status = 'rejected', decided_at = now(),
		       decided_by = NULLIF($3, '')::uuid, reject_reason = $4
		  FROM orders o
		 WHERE r.id = $1 AND r.order_id = $2 AND r.status = 'pending' AND o.id = r.order_id
		RETURNING r.driver_id::text, o.status`, requestID, orderID, actorID, reason).Scan(&driverID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_edit_requests WHERE id = $1 AND order_id = $2)`,
			requestID, orderID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return httpx.ErrNotFound
		}
		return ErrEditRequestNotPending
	}
	if err != nil {
		return err
	}
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5::uuid)`, orderID, status, actor,
		"المكتب رفض طلب تعديل الكابتن — السبب: "+reason, driverID); err != nil {
		return err
	}
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.notifyEditRequestDecision(ctx, orderID, driverID, "السبب: "+reason)
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	return nil
}

// HasPendingEditRequest **في طلب تعديل من الكابتن بانتظار المكتب؟** — بيمنع التسليم (`ErrEditRequestPendingDeliver`).
func (s *Service) HasPendingEditRequest(ctx context.Context, orderID string) (bool, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return false, nil
	}
	var pending bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_edit_requests
		WHERE order_id = $1 AND status = 'pending')`, orderID).Scan(&pending)
	return pending, err
}

// WithdrawEditRequest **الكابتن يسحب طلبه المعلّق بنفسه** — بيرجع يقدر يسلّم أو يبعت غيره. والطلب
// ما بيتغيّر. `hook` أثره بالمعاملة نفسها (التدقيق).
func (s *Service) WithdrawEditRequest(ctx context.Context, driverID, orderID, requestID string,
	hook func(context.Context, dbtx.Querier) error) error {
	if _, err := uuid.Parse(requestID); err != nil {
		return httpx.ErrNotFound
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return httpx.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `
		UPDATE order_edit_requests r SET status = 'withdrawn', decided_at = now()
		  FROM orders o
		 WHERE r.id = $1 AND r.order_id = $2 AND r.driver_id = $3 AND r.status = 'pending' AND o.id = r.order_id
		RETURNING o.status`, requestID, orderID, driverID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		var cur string
		err := tx.QueryRow(ctx, `SELECT status FROM order_edit_requests
			WHERE id = $1 AND order_id = $2 AND driver_id = $3`, requestID, orderID, driverID).Scan(&cur)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		// **وإعادة نداء ضاع ردّه** — سُحب أصلاً فهو نجاح.
		if cur == EditRequestWithdrawn {
			return nil
		}
		return ErrEditRequestNotPending
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $3)`, orderID, status, driverID, "الكابتن سحب طلب التعديل"); err != nil {
		return err
	}
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	return nil
}

// notifyEditRequestDecision **رفض المكتب (أو طلب صار قديم) بيوصل الكابتن.**
func (s *Service) notifyEditRequestDecision(ctx context.Context, orderID, driverID, detail string) {
	if s.notify == nil || driverID == "" {
		return
	}
	var number int64
	_ = s.db.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number)
	title := t.editRequestRejectedDriver
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindOrder, Title: title,
		Body:   fmt.Sprintf("#%d — %s", number, detail),
		Entity: "order", EntityID: orderID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
}

// closeEndedEditRequests **طلبات معلّقة على طلبات انتهت بتتسكّر** — جولة الراصد.
func (s *Service) closeEndedEditRequests(ctx context.Context) {
	if _, err := s.db.Exec(ctx, `
		UPDATE order_edit_requests r SET status = 'closed', decided_at = now()
		  FROM orders o
		 WHERE o.id = r.order_id AND r.status = 'pending'
		   AND (o.closed_at IS NOT NULL OR o.status IN ('delivered','rejected','cancelled','failed','refunded'))`); err != nil {
		s.logger.Warn("طلبات تعديل الكابتن: تعذّر تسكير المنتهي", "error", err)
	}
}
