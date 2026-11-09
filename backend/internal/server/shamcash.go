package server

// ══════════════════════════════════════════════════════════════════════
// **الدفع بشام كاش** — قرار المالك ٢٠٢٦-١١-٠٩: «جهّزها وخلّيها جاهزة، وما في شي ينزل هلّق»
// ══════════════════════════════════════════════════════════════════════
//
// الطريقة الرسميّة وحدها (`internal/shamcash`) — ولا خدمة غير رسميّة أبداً.
//
//	الشحن       POST /my/shamcash/topups  → فاتورة → صفحة شام كاش → إشعار مشفّر → تأكيد بـgetBill
//	            → إيداع بالمحفظة مرّة وحدة (`topup_shamcash`) + سطر داخل بدفتر حساب شام كاش
//	الطلب       POST /my/shamcash/orders  → نفس الشي، و**الطلب بينعمل بعد الدفع** من المحفظة
//	            بالطريق الموجود (`payment_method = wallet`) — ما في طريق دفع تاني.
//	            إذا ما اندفعت الفاتورة ما في طلب أصلاً؛ وإذا اندفعت وما انعمل الطلب (سكّر المتجر،
//	            تغيّر السعر) **المبلغ بيضلّ بالمحفظة** والزبون بيتبلّغ.
//	الإرجاع     POST /admin/shamcash/bills/{id}/refund — للأدمن، جزئيّ، بمفتاح منع تكرار،
//	            بيخصم من المحفظة (`refund_shamcash`).
//
// **ومال شام كاش ما بيمرّ بدرج المكتب** (FI-16.c) — الشحن النقديّ بس إله سطر بالدرج (FI-15.g).
//
// **والإشعار والفاحص بيشتغلوا ولو انطفى الإعداد** — فاتورة اندفعت قبل الإطفاء لازم توصل.
// والإطفاء بيمنع فواتير جديدة بس.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/shamcash"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

const (
	shamSettingEnabled = "customers.shamcash_enabled"
	shamSettingMin     = "customers.shamcash_min"
	shamSettingMax     = "customers.shamcash_max"
	// shamBillTTL شام كاش بتنهي الفاتورة غير المدفوعة بعد عشر دقايق.
	shamBillTTL = 10 * time.Minute
	// shamFirstCheck أوّل سؤال لشام كاش عن فاتورة ما وصل إشعارها — بعد انتهائها (الحزمة
	// الرسميّة بتقول: مرّة وحدة بعد عشر دقايق، لا استطلاع).
	shamFirstCheck = 11 * time.Minute
	// shamRecheck وبعدها كلّ خمس دقايق لحدّ ما تنحسم.
	shamRecheck = 5 * time.Minute
	// shamGiveUp فاتورة بعدها «معلّقة» عند شام كاش بعد هالمدّة منعلّمها منتهية عندنا — وإذا
	// وصل بعدها إشعار دفع مؤكّد بتنحسب عادي.
	shamGiveUp = 30 * time.Minute
	// shamDryRunCredit رصيد مؤقّت جوّا معاملة بترجع دايماً — لنحسب مجموع الطلب بنفس طريق الإنشاء.
	shamDryRunCredit int64 = 1 << 40
)

var (
	errShamUnavailable = httpx.NewError(http.StatusServiceUnavailable, "shamcash_unavailable", "errors.shamcash_unavailable")
	errShamRange       = httpx.NewError(http.StatusBadRequest, "shamcash_amount_out_of_range", "errors.shamcash_amount_out_of_range")
	errShamExceeds     = httpx.NewError(http.StatusConflict, "shamcash_refund_exceeds", "errors.shamcash_refund_exceeds")
	errShamNotPaid     = httpx.NewError(http.StatusConflict, "shamcash_not_paid", "errors.shamcash_not_paid")
	errShamNotFound    = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")
)

// WithShamCash يضع عميل شام كاش مبنيّاً بيد الفحص (خادم شام كاش مزيّف).
// **ولا يُنادى في الإنتاج** — العميل هناك من متغيّرات `SHAMCASH_*`.
func WithShamCash(c *shamcash.Client) Option {
	return func(o *options) { o.shamcash = c }
}

// shamClientFromEnv العميل من البيئة — nil إن نقص شيء (والميزة كلّها مطفيّة حينها).
func shamClientFromEnv() *shamcash.Client {
	cfg := shamcash.ConfigFromEnv()
	if !cfg.Complete() {
		return nil
	}
	c, err := shamcash.New(cfg)
	if err != nil {
		return nil
	}
	return c
}

// shamAvailable هل تُقبل فواتير جديدة: مفاتيح موجودة + الإعداد مفتوح.
func (s *Server) shamAvailable(ctx context.Context) bool {
	return s.sham != nil && s.settings.GetBool(ctx, shamSettingEnabled)
}

func (s *Server) shamLimits(ctx context.Context) (int64, int64) {
	return s.settings.GetInt(ctx, shamSettingMin), s.settings.GetInt(ctx, shamSettingMax)
}

// ── الفاتورة كما يراها الزبون ──────────────────────────────────────────

type shamBillView struct {
	ID          string     `json:"id"`
	Number      string     `json:"number"`
	Purpose     string     `json:"purpose"`
	Amount      int64      `json:"amount"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	PaidAt      *time.Time `json:"paid_at"`
	OrderID     *string    `json:"order_id"`
	OrderNumber *int64     `json:"order_number"`
	OrderError  string     `json:"order_error"`
	PaymentURL  string     `json:"payment_url"`
	// للإدارة
	UserID     string  `json:"user_id,omitempty"`
	UserName   *string `json:"user_name,omitempty"`
	UserPhone  *string `json:"user_phone,omitempty"`
	ShamTranID *int64  `json:"sham_tran_id,omitempty"`
	Refunded   int64   `json:"refunded"`
}

const shamBillCols = `b.id::text, b.number, b.purpose, b.amount, b.status, b.created_at, b.expires_at,
	b.paid_at, b.order_id::text, o.number, b.order_error, b.payment_url,
	b.user_id::text, u.full_name, u.phone::text, b.sham_tran_id, b.refunded`

const shamBillFrom = `FROM shamcash_bills b
	JOIN users u ON u.id = b.user_id
	LEFT JOIN orders o ON o.id = b.order_id`

func scanShamBill(row pgx.Row) (shamBillView, error) {
	var v shamBillView
	err := row.Scan(&v.ID, &v.Number, &v.Purpose, &v.Amount, &v.Status, &v.CreatedAt, &v.ExpiresAt,
		&v.PaidAt, &v.OrderID, &v.OrderNumber, &v.OrderError, &v.PaymentURL,
		&v.UserID, &v.UserName, &v.UserPhone, &v.ShamTranID, &v.Refunded)
	return v, err
}

// customerView يخفي حقول الإدارة.
func (v shamBillView) customerView() shamBillView {
	v.UserID, v.UserName, v.UserPhone, v.ShamTranID = "", nil, nil, nil
	return v
}

// ── الزبون ─────────────────────────────────────────────────────────────

// handleMyShamCash GET /my/shamcash — هل يظهر الخيار، وحدّاه.
func (s *Server) handleMyShamCash(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	minA, maxA := s.shamLimits(ctx)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"enabled": s.shamAvailable(ctx), "min": minA, "max": maxA,
		"expires_minutes": int(shamBillTTL / time.Minute),
	})
}

type shamTopupIn struct {
	Amount int64  `json:"amount"`
	Return string `json:"return"`
}

// handleShamTopup POST /my/shamcash/topups — فاتورة شحن.
func (s *Server) handleShamTopup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.shamAvailable(ctx) {
		s.respondErr(w, errShamUnavailable)
		return
	}
	in, err := decode[shamTopupIn](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.shamCheckRange(ctx, in.Amount); err != nil {
		s.respondErr(w, err)
		return
	}
	v, url, err := s.shamOpenBill(ctx, userIDFrom(r), "topup", in.Amount, nil, in.Return)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"bill": v.customerView(), "payment_url": url})
}

func (s *Server) shamCheckRange(ctx context.Context, amount int64) error {
	minA, maxA := s.shamLimits(ctx)
	if amount <= 0 || amount < minA || (maxA > 0 && amount > maxA) {
		return errShamRange
	}
	return nil
}

// shamOrderInput ما يُحفظ مع فاتورة الطلب ليُنشأ الطلبُ بعد الدفع كما لو أرسله الزبون.
type shamOrderInput struct {
	Input orders.CreateInput `json:"input"`
	Roles []string           `json:"roles"`
	IP    string             `json:"ip"`
}

type shamOrderIn struct {
	Order  orders.CreateInput `json:"order"`
	Return string             `json:"return"`
}

// handleShamOrder POST /my/shamcash/orders — فاتورة لطلب؛ الطلب بينعمل بعد الدفع.
func (s *Server) handleShamOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.shamAvailable(ctx) {
		s.respondErr(w, errShamUnavailable)
		return
	}
	// **نفس أبواب «اطلب» الموجودة** — ما منفتح فاتورة لطلب ما بينقبل.
	if !s.requireLaunch(w, r, launchCustomerOrders) || !s.requireLaunch(w, r, launchMerchantOrders) ||
		!s.requireOrdering(w, r) {
		return
	}
	body, err := decode[shamOrderIn](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	in := body.Order
	fields := []textguard.Field{
		tf("address_text", &in.AddressText, maxAddressText, textguard.Address),
		tf("notes", &in.Notes, maxOrderNotes, textguard.Notes),
	}
	for i := range in.Items {
		fields = append(fields, tf("items.note", &in.Items[i].Note, maxItemNote, textguard.Notes))
	}
	if _, err := s.guardText(ctx, fields...); err != nil {
		s.respondErr(w, err)
		return
	}
	uid := userIDFrom(r)
	in.CustomerID, in.CustomerPhone, in.PaymentMethod = uid, "", "wallet"
	saved := shamOrderInput{Input: in, Roles: rolesFrom(r), IP: clientIP(r)}

	total, err := s.shamDryRunTotal(ctx, saved)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.shamCheckRange(ctx, total); err != nil {
		s.respondErr(w, err)
		return
	}
	v, url, err := s.shamOpenBill(ctx, uid, "order", total, &saved, body.Return)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"bill": v.customerView(), "payment_url": url})
}

// shamDryRunTotal **مجموع الطلب بنفس طريق الإنشاء** — معاملة بترجع دايماً: رصيد مؤقّت،
// إنشاء الطلب من المحفظة، قراءة المجموع، ثمّ تراجع. فالسعر والخصم والتوصيل المجّاني نفسهم
// اللي رح ينحسبوا بعد الدفع، وأيّ خطأ بالسلّة (متجر مسكّر، صنف ناقص) بيطلع هلّق قبل الفاتورة.
func (s *Server) shamDryRunTotal(ctx context.Context, in shamOrderInput) (int64, error) {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := s.wallet.ApplyTx(ctx, tx, in.Input.CustomerID, shamDryRunCredit, "topup_shamcash",
		"dry-run-"+uuid.NewString(), "", nil); err != nil {
		return 0, err
	}
	o, _, err := s.orders.CreateTx(ctx, tx, in.Input.CustomerID, in.Roles, in.Input, in.IP)
	if err != nil {
		return 0, err
	}
	if o.Total <= 0 {
		return 0, errValidation
	}
	return o.Total, nil
}

var shamBillNoChars = regexp.MustCompile(`[^A-Z0-9]`)

// shamOpenBill يكتب الفاتورة عندنا أوّلاً ثمّ يطلبها من شام كاش.
func (s *Server) shamOpenBill(ctx context.Context, userID, purpose string, amount int64,
	order *shamOrderInput, ret string) (shamBillView, string, error) {
	var orderJSON []byte
	if order != nil {
		orderJSON, _ = json.Marshal(order)
	}
	number := "RG" + shamBillNoChars.ReplaceAllString(strings.ToUpper(uuid.NewString()), "")[:14]
	var id string
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO shamcash_bills (number, user_id, purpose, amount, currency, order_input, expires_at, history)
		VALUES ($1, $2, $3, $4, $5, $6, now() + $7::interval,
		        jsonb_build_array(jsonb_build_object('at', now(), 'event', 'created', 'return', $8::text)))
		RETURNING id::text`,
		number, userID, purpose, amount, shamcash.CurrencySYP, orderJSON,
		strconv.Itoa(int(shamBillTTL/time.Second))+" seconds", ret).Scan(&id); err != nil {
		return shamBillView{}, "", err
	}
	note := "RahalGo " + purpose + " " + number
	bill, err := s.sham.CreateBill(ctx, number, amount, shamcash.CurrencySYP, note, s.sham.ReturnURLFor(id))
	if err != nil {
		// **عطلُ نقلٍ قد يعني فاتورةً انعملت هناك** — تبقى معلّقةً والفاحصُ يسأل عنها.
		// **وخطأُ عملٍ صريح** يعني لا فاتورة: تُعلَّم فاشلة.
		var ae *shamcash.APIError
		if errors.As(err, &ae) {
			_, _ = s.pg.Exec(ctx, `
				UPDATE shamcash_bills SET status = 'failed',
				       history = history || jsonb_build_array(jsonb_build_object('at', now(), 'event', 'create_failed', 'result', $2::int))
				 WHERE id = $1 AND status = 'pending'`, id, ae.Result)
		}
		s.logger.Warn("shamcash: createBill failed", "bill", number, "err", err)
		return shamBillView{}, "", errShamUnavailable
	}
	if _, err := s.pg.Exec(ctx, `
		UPDATE shamcash_bills SET payment_url = $2,
		       history = history || jsonb_build_array(jsonb_build_object('at', now(), 'event', 'bill_created'))
		 WHERE id = $1`, id, bill.PaymentURL); err != nil {
		return shamBillView{}, "", err
	}
	v, err := s.shamBill(ctx, s.pg, id)
	return v, bill.PaymentURL, err
}

func (s *Server) shamBill(ctx context.Context, q dbtx.Querier, id string) (shamBillView, error) {
	if _, err := uuid.Parse(id); err != nil {
		return shamBillView{}, errShamNotFound
	}
	v, err := scanShamBill(q.QueryRow(ctx, `SELECT `+shamBillCols+` `+shamBillFrom+` WHERE b.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return v, errShamNotFound
	}
	return v, err
}

// handleMyShamBill GET /my/shamcash/bills/{id} — حال الفاتورة من عندنا (لا سؤال لشام كاش بكلّ نقرة).
func (s *Server) handleMyShamBill(w http.ResponseWriter, r *http.Request) {
	v, err := s.shamBill(r.Context(), s.pg, chi.URLParam(r, "id"))
	if err == nil && v.UserID != userIDFrom(r) {
		err = errShamNotFound
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"bill": v.customerView()})
}

// ── إشعار شام كاش ─────────────────────────────────────────────────────

// handleShamWebhook POST /public/shamcash/webhook — مشفّر بالمفتاح السرّي. **ولا يُصدَّق
// «مدفوع» منه وحده**: يُسأل getBill قبل أيّ إيداع. 200 = استلمنا (فلا يعيد)، 400 = مزوّر،
// 500 = عطل مؤقّت (فيعيد شام كاش الإرسال).
func (s *Server) handleShamWebhook(w http.ResponseWriter, r *http.Request) {
	if s.sham == nil {
		s.respondErr(w, errShamUnavailable)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	ev, err := s.sham.ParseWebhook(raw)
	if err != nil {
		s.logger.Warn("shamcash: rejected webhook", "err", err)
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"ok": false})
		return
	}
	var id, status string
	err = s.pg.QueryRow(r.Context(), `SELECT id::text, status FROM shamcash_bills WHERE number = $1`, ev.BillNo).
		Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		// فاتورة مو إلنا — منقول استلمنا حتى ما يضلّ يعيد.
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	_, _ = s.pg.Exec(r.Context(), `
		UPDATE shamcash_bills SET history = history || jsonb_build_array(jsonb_build_object(
		       'at', now(), 'event', 'webhook', 'status', $2::int, 'tran', $3::bigint))
		 WHERE id = $1`, id, ev.StatusID, ev.TranID)
	if err := s.shamReconcile(r.Context(), id); err != nil {
		s.logger.Warn("shamcash: webhook reconcile failed", "bill", ev.BillNo, "err", err)
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"ok": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// shamReconcile يسأل شام كاش عن الفاتورة ويطبّق جوابها — **المصدر الوحيد للحسم**، يناديه
// الإشعار والفاحص معاً.
func (s *Server) shamReconcile(ctx context.Context, id string) error {
	var number, status, userID string
	var amount int64
	var created time.Time
	if err := s.pg.QueryRow(ctx, `
		SELECT number, status, user_id::text, amount, created_at FROM shamcash_bills WHERE id = $1`, id).
		Scan(&number, &status, &userID, &amount, &created); err != nil {
		return err
	}
	if status != "pending" && status != "expired" && status != "failed" {
		return nil // انحسمت — والإشعار المكرّر ما بيعمل شي
	}
	_, _ = s.pg.Exec(ctx, `UPDATE shamcash_bills SET checked_at = now() WHERE id = $1`, id)
	bill, err := s.sham.GetBill(ctx, number)
	if shamcash.IsResult(err, shamcash.ResultBillNotFound) {
		if status == "pending" && time.Since(created) > shamGiveUp {
			return s.shamMark(ctx, id, "failed", "not_found")
		}
		return nil
	}
	if err != nil {
		return err
	}
	switch bill.StatusID {
	case shamcash.StatusPaid, shamcash.StatusPartlyRefunded, shamcash.StatusRefund:
		if bill.TranID == nil || !s.sham.AmountEquals(bill.Amount, amount) || bill.CurrencyID != shamcash.CurrencySYP {
			// **ما منودع شي** — مبلغ أو عملة ما بتطابق: للإدارة.
			s.logger.Error("shamcash: paid bill does not match", "bill", number, "amount", bill.Amount)
			_, _ = s.pg.Exec(ctx, `
				UPDATE shamcash_bills SET history = history || jsonb_build_array(jsonb_build_object(
				       'at', now(), 'event', 'mismatch', 'remote_amount', $2::text, 'currency', $3::int))
				 WHERE id = $1`, id, bill.Amount, bill.CurrencyID)
			return nil
		}
		return s.shamSettle(ctx, id, *bill.TranID)
	case shamcash.StatusExpired:
		return s.shamMark(ctx, id, "expired", "remote_expired")
	default: // معلّقة
		if status == "pending" && time.Since(created) > shamGiveUp {
			return s.shamMark(ctx, id, "expired", "local_timeout")
		}
		return nil
	}
}

func (s *Server) shamMark(ctx context.Context, id, to, why string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE shamcash_bills SET status = $2,
		       history = history || jsonb_build_array(jsonb_build_object('at', now(), 'event', $3::text))
		 WHERE id = $1 AND status = 'pending'`, id, to, why)
	return err
}

// shamSettle **الإيداع مرّة وحدة** — قفل الصفّ + انتقال الحالة + فهرس فريد على مرجع القيد.
// ولفاتورة الطلب: الطلب بينعمل بنفس المعاملة من المحفظة، وإذا فشل بيضلّ المبلغ بالمحفظة.
func (s *Server) shamSettle(ctx context.Context, id string, tranID int64) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID, purpose, status, number string
	var amount int64
	var orderJSON []byte
	if err := tx.QueryRow(ctx, `
		SELECT user_id::text, purpose, status, number, amount, order_input
		  FROM shamcash_bills WHERE id = $1 FOR UPDATE`, id).
		Scan(&userID, &purpose, &status, &number, &amount, &orderJSON); err != nil {
		return err
	}
	if status != "pending" && status != "expired" && status != "failed" {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE shamcash_bills SET status = 'paid', paid_at = now(), sham_tran_id = $2,
		       history = history || jsonb_build_array(jsonb_build_object('at', now(), 'event', 'paid', 'tran', $2::bigint))
		 WHERE id = $1`, id, tranID); err != nil {
		return err
	}
	note := "شام كاش " + number
	if _, err := s.wallet.ApplyTx(ctx, tx, userID, amount, "topup_shamcash", id, note, nil); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO shamcash_account_entries (direction, amount, source, ref)
		VALUES ('in', $1, 'bill_paid', $2) ON CONFLICT (source, ref) DO NOTHING`, amount, id); err != nil {
		return err
	}

	var after func()
	var orderID, orderErr string
	var paidDebt int64
	switch purpose {
	case "topup":
		// **والشحن بيسدّ الدين أوّلاً** — متل الشحن النقديّ (`payDebtFromTopup`). سؤال للمالك.
		if paidDebt, err = s.shamPayDebt(ctx, tx, userID, amount, id); err != nil {
			return err
		}
	case "order":
		var in shamOrderInput
		if err := json.Unmarshal(orderJSON, &in); err != nil {
			return err
		}
		// **نقطةُ رجوع** — فشلُ الطلب ما بيرجّع الإيداع.
		sp, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		o, fn, cerr := s.orders.CreateTx(ctx, sp, in.Input.CustomerID, in.Roles, in.Input, in.IP)
		if cerr == nil {
			cerr = sp.Commit(ctx)
		}
		if cerr != nil {
			_ = sp.Rollback(ctx)
			orderErr = "order_failed"
			var ae *httpx.AppError
			if errors.As(cerr, &ae) {
				orderErr = ae.Code
			}
			s.logger.Warn("shamcash: order after payment failed", "bill", number, "err", cerr)
		} else {
			orderID, after = o.ID, fn
		}
		if _, err := tx.Exec(ctx, `
			UPDATE shamcash_bills SET order_id = NULLIF($2, '')::uuid, order_error = $3 WHERE id = $1`,
			id, orderID, orderErr); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// ── بعد التثبيت ──
	if after != nil {
		after()
		go s.autoTransfer(context.WithoutCancel(ctx), orderID, userID)
	}
	title := notifTitles.shamTopupPaid
	switch {
	case purpose == "order" && orderID != "":
		title = notifTitles.shamOrderPaid
	case purpose == "order":
		title = notifTitles.shamOrderFailed
	}
	body := fmtMoneyAr(amount)
	if paidDebt > 0 {
		body += " · " + fmtMoneyAr(paidDebt)
	}
	s.notify.Notify(context.WithoutCancel(ctx), notifications.Input{
		UserID: userID, Kind: notifications.KindWallet, Title: title, Body: body,
		Entity: "wallet", Href: "/wallet/shamcash/" + id,
	})
	s.touchUser(userID, "wallet")
	s.touch("wallet", "ops")
	return nil
}

// shamPayDebt كـ`payDebtFromTopup` — بلا مُنفِّذ (ما في إنسان؛ القيد من شام كاش).
func (s *Server) shamPayDebt(ctx context.Context, q dbtx.Querier, userID string, amount int64, ref string) (int64, error) {
	var hasDebt bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM financial_obligations o
		                WHERE o.closed_at IS NULL
		                  AND ((o.party_kind IN ('rep', 'driver') AND o.party_id = $1::uuid)
		                    OR (o.party_kind = 'merchant' AND o.party_id IN (
		                          SELECT id FROM merchants WHERE owner_user_id = $1::uuid))))`,
		userID).Scan(&hasDebt); err != nil || !hasDebt {
		return 0, err
	}
	var treasury string
	if err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1), '')`).
		Scan(&treasury); err != nil {
		return 0, err
	}
	return obligations.PayFromCredit(ctx, q, userID, amount, ref, nil, obligations.CreditPay{
		Wallet: s.wallet, TreasuryID: treasury, Method: obligations.MethodTopup, From: "الشحن من شام كاش",
	})
}

// ── الفاحص ────────────────────────────────────────────────────────────

// RunShamCashPoller **فاحص ما فات من إشعارات** — الفواتير المعلّقة بعد انتهائها، والإرجاع
// اللي ما انحسم (بنفس مفتاح منع التكرار).
func (s *Server) RunShamCashPoller(ctx context.Context, interval time.Duration) {
	if s.sham == nil {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.SweepShamCash(ctx)
		}
	}
}

// SweepShamCash جولة واحدة — مكشوفة للفحص.
func (s *Server) SweepShamCash(ctx context.Context) {
	if s.sham == nil {
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text FROM shamcash_bills
		 WHERE status = 'pending' AND created_at < now() - $1::interval
		   AND (checked_at IS NULL OR checked_at < now() - $2::interval)
		 ORDER BY created_at LIMIT 50`,
		strconv.Itoa(int(shamFirstCheck/time.Second))+" seconds",
		strconv.Itoa(int(shamRecheck/time.Second))+" seconds")
	if err != nil {
		s.logger.Warn("shamcash: sweep", "err", err)
		return
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return
	}
	for _, id := range ids {
		if err := s.shamReconcile(ctx, id); err != nil {
			s.logger.Warn("shamcash: sweep reconcile", "bill", id, "err", err)
		}
	}
	rrows, err := s.pg.Query(ctx, `
		SELECT id::text FROM shamcash_refunds WHERE status = 'pending' AND created_at < now() - interval '1 minute'
		 ORDER BY created_at LIMIT 50`)
	if err != nil {
		return
	}
	rids, err := pgx.CollectRows(rrows, pgx.RowTo[string])
	if err != nil {
		return
	}
	for _, id := range rids {
		if err := s.shamSendRefund(ctx, id); err != nil {
			s.logger.Warn("shamcash: sweep refund", "refund", id, "err", err)
		}
	}
}

// ── الإدارة ───────────────────────────────────────────────────────────

// handleAdminShamBills GET /admin/shamcash/bills — قائمة الفواتير ورصيد دفتر شام كاش.
func (s *Server) handleAdminShamBills(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	const per = 50
	status := q.Get("status")
	search := strings.TrimSpace(q.Get("q"))
	where := `WHERE ($1 = '' OR b.status = $1)
	  AND ($2 = '' OR b.number ILIKE '%' || $2 || '%' OR u.phone::text ILIKE '%' || $2 || '%'
	       OR COALESCE(u.full_name, '') ILIKE '%' || $2 || '%')`
	var total int
	if err := s.pg.QueryRow(ctx, `SELECT count(*) `+shamBillFrom+` `+where, status, search).Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, `SELECT `+shamBillCols+` `+shamBillFrom+` `+where+`
		ORDER BY b.created_at DESC LIMIT $3 OFFSET $4`, status, search, per, (page-1)*per)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []shamBillView{}
	for rows.Next() {
		v, err := scanShamBill(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, v)
	}
	var book int64
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(CASE WHEN direction = 'in' THEN amount ELSE -amount END), 0)::bigint
		  FROM shamcash_account_entries`).Scan(&book); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"bills": out, "total": total, "page": page, "per_page": per, "book_balance": book,
		"enabled": s.settings.GetBool(ctx, shamSettingEnabled), "configured": s.sham != nil,
	})
}

type shamReconRow struct {
	BillNo       string  `json:"bill_no"`
	TranID       *int64  `json:"tran_id"`
	Type         string  `json:"type"`
	RemoteAmount *string `json:"remote_amount"`
	LocalAmount  *int64  `json:"local_amount"`
	Status       string  `json:"status"`
}

// handleAdminShamReconcile GET /admin/shamcash/reconcile?date= — حركات شام كاش لليوم مقابل ما عندنا.
func (s *Server) handleAdminShamReconcile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if s.sham == nil {
		s.respondErr(w, errShamUnavailable)
		return
	}
	date := r.URL.Query().Get("date")
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	remote, err := s.sham.AllTransactions(ctx, date, date)
	if err != nil {
		s.logger.Warn("shamcash: listTransactions", "err", err)
		s.respondErr(w, errShamUnavailable)
		return
	}
	// ما عندنا بنفس اليوم بتوقيت دمشق: مدفوعات بالحركة، وإرجاعات ناجحة.
	type local struct {
		billNo string
		tran   int64
		typ    string
		amount int64
	}
	locals := map[int64]local{}
	rows, err := s.pg.Query(ctx, `
		SELECT b.number, b.sham_tran_id, 'payment', b.amount FROM shamcash_bills b
		 WHERE b.sham_tran_id IS NOT NULL
		   AND (b.paid_at AT TIME ZONE 'Asia/Damascus')::date = $1::date
		UNION ALL
		SELECT b.number, r.sham_tran_id, 'refund', r.amount FROM shamcash_refunds r
		  JOIN shamcash_bills b ON b.id = r.bill_id
		 WHERE r.status = 'done' AND (r.done_at AT TIME ZONE 'Asia/Damascus')::date = $1::date`, day)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var l local
		if err := rows.Scan(&l.billNo, &l.tran, &l.typ, &l.amount); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		locals[l.tran] = l
	}
	rows.Close()

	out := []shamReconRow{}
	matched, mismatched := 0, 0
	seen := map[int64]bool{}
	for _, t := range remote {
		t := t
		typ := "payment"
		if t.TranTypeID == shamcash.TranRefund {
			typ = "refund"
		}
		row := shamReconRow{BillNo: t.BillNo, TranID: &t.TranID, Type: typ, RemoteAmount: &t.Amount}
		seen[t.TranID] = true
		l, ok := locals[t.TranID]
		if !ok {
			// قد تكون حركتنا مسجّلة بيوم تاني (حدود منتصف الليل) — منفتّش بالرقم.
			var amt int64
			q := `SELECT amount FROM shamcash_bills WHERE sham_tran_id = $1`
			if typ == "refund" {
				q = `SELECT amount FROM shamcash_refunds WHERE sham_tran_id = $1 AND status = 'done'`
			}
			if err := s.pg.QueryRow(ctx, q, t.TranID).Scan(&amt); err == nil {
				l, ok = local{billNo: t.BillNo, tran: t.TranID, typ: typ, amount: amt}, true
			}
		}
		switch {
		case !ok:
			row.Status = "missing_local"
			mismatched++
		case !s.sham.AmountEquals(t.Amount, l.amount) || l.typ != typ:
			row.LocalAmount, row.Status = &l.amount, "amount_mismatch"
			mismatched++
		default:
			row.LocalAmount, row.Status = &l.amount, "match"
			matched++
		}
		out = append(out, row)
	}
	for tran, l := range locals {
		if seen[tran] {
			continue
		}
		tran, l := tran, l
		out = append(out, shamReconRow{BillNo: l.billNo, TranID: &tran, Type: l.typ, LocalAmount: &l.amount, Status: "missing_remote"})
		mismatched++
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"date": date, "rows": out,
		"summary": map[string]int{"remote_count": len(remote), "local_count": len(locals),
			"matched": matched, "mismatched": mismatched},
	})
}

type shamRefundIn struct {
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

type shamRefundView struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

// handleAdminShamRefund POST /admin/shamcash/bills/{id}/refund — للأدمن (treasury.manage).
// الخصم من المحفظة أوّلاً بمعاملة، ثمّ الطلب من شام كاش بمفتاح = معرّف الإرجاع.
func (s *Server) handleAdminShamRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !s.shamAvailable(ctx) {
		s.respondErr(w, errShamUnavailable)
		return
	}
	in, err := decode[shamRefundIn](r)
	if err != nil || in.Amount <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	billID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(billID); err != nil {
		s.respondErr(w, errShamNotFound)
		return
	}
	actor := userIDFrom(r)
	// The DB half (refund row + wallet debit) runs inside the idempotency transaction
	// (XG-33): a double click or a replayed key never makes a second refund. Sham Cash is
	// called after commit with the refund id as its idempotency key.
	s.WithIdempotentTx(w, r, func(ctx context.Context, tx dbtx.Querier) (IdempotentBody, error) {
		var userID, status string
		var amount, open int64
		err := tx.QueryRow(ctx, `
			SELECT b.user_id::text, b.status, b.amount,
			       COALESCE((SELECT sum(r.amount) FROM shamcash_refunds r
			                  WHERE r.bill_id = b.id AND r.status <> 'failed'), 0)::bigint
			  FROM shamcash_bills b WHERE b.id = $1 FOR UPDATE`, billID).Scan(&userID, &status, &amount, &open)
		if errors.Is(err, pgx.ErrNoRows) {
			return IdempotentBody{}, errShamNotFound
		}
		if err != nil {
			return IdempotentBody{}, err
		}
		if status != "paid" && status != "partly_refunded" {
			return IdempotentBody{}, errShamNotPaid
		}
		if open+in.Amount > amount {
			return IdempotentBody{}, errShamExceeds
		}
		v := &shamRefundView{Amount: in.Amount, Status: "pending"}
		if err := tx.QueryRow(ctx, `
			INSERT INTO shamcash_refunds (bill_id, amount, note, created_by)
			VALUES ($1, $2, $3, $4) RETURNING id::text`, billID, in.Amount, strings.TrimSpace(in.Note), actor).
			Scan(&v.ID); err != nil {
			return IdempotentBody{}, err
		}
		if _, err := s.wallet.ApplyTx(ctx, tx, userID, -in.Amount, "refund_shamcash", v.ID,
			strings.TrimSpace(in.Note), &actor); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"refund": v},
			// Runs before the live response is written, so the caller sees done/failed;
			// a replay of the key returns the stored "pending" and the bills list shows the rest.
			AfterCommit: func() {
				bg := context.WithoutCancel(ctx)
				if err := s.shamSendRefund(bg, v.ID); err != nil {
					s.logger.Warn("shamcash: refund send", "refund", v.ID, "err", err)
				}
				_ = s.pg.QueryRow(bg, `SELECT status FROM shamcash_refunds WHERE id = $1`, v.ID).Scan(&v.Status)
			},
		}, nil
	})
}

// shamSendRefund يرسل الإرجاع المعلّق لشام كاش ويحسمه:
//
//	نجح        → done + سطر طالع بدفتر شام كاش + الفاتورة refunded/partly_refunded
//	رفض صريح   → failed + يرجع الخصم للمحفظة (نفس النوع ونفس المرجع)
//	عطل نقل    → يبقى pending والفاحص يعيد بنفس المفتاح
func (s *Server) shamSendRefund(ctx context.Context, refundID string) error {
	var billNo, billID, userID, status, note string
	var amount int64
	if err := s.pg.QueryRow(ctx, `
		SELECT b.number, b.id::text, b.user_id::text, r.status, r.amount, r.note
		  FROM shamcash_refunds r JOIN shamcash_bills b ON b.id = r.bill_id WHERE r.id = $1`, refundID).
		Scan(&billNo, &billID, &userID, &status, &amount, &note); err != nil {
		return err
	}
	if status != "pending" {
		return nil
	}
	tranID, rerr := s.sham.RefundBill(ctx, billNo, amount, refundID, note)
	var ae *shamcash.APIError
	if rerr != nil && !errors.As(rerr, &ae) {
		return rerr // غير محسوم — يعاد لاحقاً
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT status FROM shamcash_refunds WHERE id = $1 FOR UPDATE`, refundID).
		Scan(&status); err != nil || status != "pending" {
		return err
	}
	if rerr == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE shamcash_refunds SET status = 'done', sham_tran_id = $2, done_at = now() WHERE id = $1`,
			refundID, tranID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO shamcash_account_entries (direction, amount, source, ref)
			VALUES ('out', $1, 'refund_sent', $2) ON CONFLICT (source, ref) DO NOTHING`, amount, refundID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE shamcash_bills SET refunded = refunded + $2,
			       status = CASE WHEN refunded + $2 >= amount THEN 'refunded' ELSE 'partly_refunded' END,
			       history = history || jsonb_build_array(jsonb_build_object('at', now(), 'event', 'refunded',
			                 'amount', $2::bigint, 'tran', $3::bigint))
			 WHERE id = $1`, billID, amount, tranID); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE shamcash_refunds SET status = 'failed', error = $2, done_at = now() WHERE id = $1`,
			refundID, strconv.Itoa(ae.Result)+" "+ae.Message); err != nil {
			return err
		}
		if _, err := s.wallet.ApplyTx(ctx, tx, userID, amount, "refund_shamcash", refundID,
			"رجع الخصم — شام كاش رفضت الإرجاع", nil); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if rerr == nil {
		s.notify.Notify(context.WithoutCancel(ctx), notifications.Input{
			UserID: userID, Kind: notifications.KindWallet, Title: notifTitles.shamRefunded,
			Body: fmtMoneyAr(amount), Entity: "wallet", Href: "/wallet",
		})
	}
	s.touchUser(userID, "wallet")
	return nil
}
