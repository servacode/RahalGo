package server

// ══════════════════════════════════════════════════════════════════════
// **سجلّ شام كاش اليدويّ** — قرار المالك ٢٠٢٦-١١-١٠ (`internal/shamcash`)
// ══════════════════════════════════════════════════════════════════════
//
//	السائق     «كيف قبضت؟ نقدي / شام كاش» عند التسليم — حقل `shamcash` اختياريّ بنداء الانتقال
//	المالك     إيداع وسحب من حسابه · رصيد البداية لكل عملة · «سلّمت السائق بدالها» · «ما وصل»
//	الرصيد     البداية + التحويلات (غير «ما وصل») + الإيداعات − السحوبات، لكل عملة لحالها
//
// **ولا شي هون بيمسّ الدفتر ولا المحافظ ولا صندوق السائق ولا درج المكتب ولا الخزينة**: المالك بيعطي
// السائق المبلغ من جيبه، والسائق بيسلّم المكتب كامل المبلغ متل العادة.
//
// القراءة بـ`finance.read` والكتابة بـ`finance.manage` (`authz/policy.go`)، وكل كتابة إلها أثر بالتدقيق
// بنفس المعاملة.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/shamcash"
)

// shamcashClaim **ما يرسله السائق** — المبلغ بالوحدة الكبرى (١٥٠٠٠٠ ليرة · ١٢٫٥ دولار).
type shamcashClaim struct {
	Currency string   `json:"currency"`
	Amount   *float64 `json:"amount"`
}

// shamcashUnits **المبلغ كما يُحفظ** — والدولار بالسنتات.
type shamcashUnits struct {
	Currency string
	Amount   int64
}

// units **يتحقّق**: عملة ليرة أو دولار، مبلغ أكبر من صفر، ليرة صحيحة ودولار بمنزلتين بالأكثر.
func (c *shamcashClaim) units() (*shamcashUnits, error) {
	if c == nil || c.Amount == nil || !shamcash.ValidCurrency(c.Currency) {
		return nil, errValidation
	}
	u, ok := orders.AmountUnits(*c.Amount, c.Currency)
	if !ok || u <= 0 {
		return nil, errValidation
	}
	return &shamcashUnits{Currency: c.Currency, Amount: u}, nil
}

// handleShamcashList **الحركات** — `GET /admin/shamcash?kind&driver&status&compensated&from&to&voided&page`.
func (s *Server) handleShamcashList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := shamcash.Filter{
		Kind: q.Get("kind"), DriverID: q.Get("driver"), Status: q.Get("status"),
		Compensated: q.Get("compensated"), From: q.Get("from"), To: q.Get("to"),
		Voided: q.Get("voided") == "1" || q.Get("voided") == "true",
	}
	switch f.Kind {
	case "", shamcash.KindDriverTransfer, shamcash.KindDeposit, shamcash.KindWithdrawal:
	default:
		s.respondErr(w, errValidation)
		return
	}
	switch f.Status {
	case "", shamcash.StatusClaimed, shamcash.StatusNotArrived:
	default:
		s.respondErr(w, errValidation)
		return
	}
	switch f.Compensated {
	case "", "yes", "no":
	default:
		s.respondErr(w, errValidation)
		return
	}
	for _, d := range []string{f.From, f.To} {
		if d != "" {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				s.respondErr(w, errValidation)
				return
			}
		}
	}
	pg := pagingOf(r, 30)
	list, total, err := shamcash.List(r.Context(), s.pg, f, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, paged("movements", list, total, pg))
}

// handleShamcashSummary **الأرصدة و«ما سلّمته للسائقين لسّا»** — `GET /admin/shamcash/summary`.
func (s *Server) handleShamcashSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := shamcash.GetSummary(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	drivers, err := shamcash.Drivers(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"balances": sum.Balances, "owed_by_driver": sum.Owed, "drivers": drivers,
	})
}

// parseDamascusDay **يوم «YYYY-MM-DD» بتوقيت دمشق** — بداية اليوم.
func parseDamascusDay(v string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(v), damascusLoc())
	return t, err == nil
}

// parseDamascusTime **وقت الحركة** — `YYYY-MM-DDTHH:MM` بتوقيت دمشق أو RFC3339، والفارغ «هلّق».
func parseDamascusTime(v string, now time.Time) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return now, true
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, true
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, v, damascusLoc()); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// handleShamcashOpening **رصيد البداية لعملة** — `PUT /admin/shamcash/opening` `{currency, amount, as_of}`.
func (s *Server) handleShamcashOpening(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Currency string   `json:"currency"`
		Amount   *float64 `json:"amount"`
		AsOf     string   `json:"as_of"`
	}](r)
	if err != nil || req.Amount == nil || !shamcash.ValidCurrency(req.Currency) {
		s.respondErr(w, errValidation)
		return
	}
	units, ok := orders.AmountUnits(*req.Amount, req.Currency)
	if !ok || units < 0 {
		s.respondErr(w, errValidation)
		return
	}
	asOf, ok := parseDamascusDay(req.AsOf)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		if err := shamcash.SetOpening(ctx, q, req.Currency, units, asOf, userIDFrom(r)); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.shamcash_opening", "shamcash_opening", req.Currency,
			map[string]any{"currency": req.Currency, "amount": units, "as_of": req.AsOf})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"currency": req.Currency, "amount": units})
}

// handleShamcashAdd **إيداع أو سحب من المالك** — `POST /admin/shamcash/movements`
// `{kind: deposit|withdrawal, currency, amount, occurred_at?, note}`.
func (s *Server) handleShamcashAdd(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Kind       string   `json:"kind"`
		Currency   string   `json:"currency"`
		Amount     *float64 `json:"amount"`
		OccurredAt string   `json:"occurred_at"`
		Note       string   `json:"note"`
	}](r)
	if err != nil || req.Amount == nil || !shamcash.ValidCurrency(req.Currency) ||
		(req.Kind != shamcash.KindDeposit && req.Kind != shamcash.KindWithdrawal) {
		s.respondErr(w, errValidation)
		return
	}
	units, ok := orders.AmountUnits(*req.Amount, req.Currency)
	if !ok || units <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	now := time.Now()
	at, ok := parseDamascusTime(req.OccurredAt, now)
	// **ولا حركة بالمستقبل** — ساعة سماح لفرق الساعات.
	if !ok || at.After(now.Add(time.Hour)) {
		s.respondErr(w, errValidation)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 300)
	var m shamcash.Movement
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		id, err := shamcash.AddOwner(ctx, q, req.Kind, req.Currency, units, at, note, userIDFrom(r))
		if err != nil {
			return err
		}
		if err := s.auditTx(ctx, q, r, "finance.shamcash_movement_added", "shamcash_movement", id,
			map[string]any{"kind": req.Kind, "currency": req.Currency, "amount": units, "note": note}); err != nil {
			return err
		}
		m, err = shamcash.Get(ctx, q, id)
		return err
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"movement": m})
}

// shamcashAct **خطوة على حركة** بسبب (إلزاميّ أو لا) وأثر بالتدقيق.
func (s *Server) shamcashAct(action string, needReason bool,
	do func(ctx context.Context, q dbtx.Querier, id, reason, actor string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var reason string
		if r.ContentLength != 0 {
			req, err := decode[struct {
				Reason string `json:"reason"`
			}](r)
			if err != nil {
				s.respondErr(w, err)
				return
			}
			reason = clip(strings.TrimSpace(req.Reason), 300)
		}
		if needReason && reason == "" {
			s.respondErr(w, errNoteRequired)
			return
		}
		var m shamcash.Movement
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			if err := do(ctx, q, id, reason, userIDFrom(r)); err != nil {
				return err
			}
			var err error
			if m, err = shamcash.Get(ctx, q, id); err != nil {
				return err
			}
			return s.auditTx(ctx, q, r, action, "shamcash_movement", id, map[string]any{
				"kind": m.Kind, "currency": m.Currency, "amount": m.Amount, "reason": reason,
			})
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"movement": m})
	}
}

// handleShamcashCompensate **«سلّمت السائق بدالها»** — `POST /admin/shamcash/{id}/compensate`.
func (s *Server) handleShamcashCompensate(w http.ResponseWriter, r *http.Request) {
	s.shamcashAct("finance.shamcash_compensated", false,
		func(ctx context.Context, q dbtx.Querier, id, _, actor string) error {
			return shamcash.SetCompensated(ctx, q, id, true, actor)
		})(w, r)
}

// handleShamcashUncompensate **تراجع عن «سلّمت السائق»** بسبب — `POST /admin/shamcash/{id}/uncompensate`.
func (s *Server) handleShamcashUncompensate(w http.ResponseWriter, r *http.Request) {
	s.shamcashAct("finance.shamcash_uncompensated", true,
		func(ctx context.Context, q dbtx.Querier, id, _, actor string) error {
			return shamcash.SetCompensated(ctx, q, id, false, actor)
		})(w, r)
}

// handleShamcashNotArrived **«ما وصل»** — بيطلع من الرصيد — `POST /admin/shamcash/{id}/not-arrived`.
func (s *Server) handleShamcashNotArrived(w http.ResponseWriter, r *http.Request) {
	s.shamcashAct("finance.shamcash_not_arrived", false,
		func(ctx context.Context, q dbtx.Querier, id, _, actor string) error {
			return shamcash.SetNotArrived(ctx, q, id, true, actor)
		})(w, r)
}

// handleShamcashArrived **تراجع عن «ما وصل»** بسبب — `POST /admin/shamcash/{id}/arrived`.
func (s *Server) handleShamcashArrived(w http.ResponseWriter, r *http.Request) {
	s.shamcashAct("finance.shamcash_arrived", true,
		func(ctx context.Context, q dbtx.Querier, id, _, actor string) error {
			return shamcash.SetNotArrived(ctx, q, id, false, actor)
		})(w, r)
}

// handleShamcashVoid **إلغاء إيداع أو سحب** بسبب — `POST /admin/shamcash/{id}/void`.
func (s *Server) handleShamcashVoid(w http.ResponseWriter, r *http.Request) {
	s.shamcashAct("finance.shamcash_voided", true,
		func(ctx context.Context, q dbtx.Querier, id, reason, actor string) error {
			return shamcash.Void(ctx, q, id, reason, actor)
		})(w, r)
}
