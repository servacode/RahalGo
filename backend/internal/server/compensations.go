package server

// ══════════════════════════════════════════════════════════════════════
// **صفحةُ «التعويضات» — طريقُ موافقةٍ واحدٌ لكلّ تعويض** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
//	١ · صفحةٌ واحدةٌ لكلّ تعويض: السائق · المتجرُ عن بضاعةٍ رُدّت · صاحبُ الشكوى.
//	    الماليّةُ توافق بكلمة السرّ، **ولا يوافق أحدٌ على ما اقترحه**، والمالُ
//	    من الخزينة دائماً (`orders.ApproveCompensationTx`).
//	٢ · سقفٌ لكلّ نوعٍ من الإعدادات — **وفوقه المالكُ وحدَه يوافق** (`owner_super_admin`).
//	٣ · السائقُ يرى «تسوية من الإدارة» ولا يرى ملاحظةَ المكتب.
//	٤ · المعلَّقُ فوق المهلة يحمرّ ويُنبَّه المالكُ والماليّة (`compensations.overdue_hours`).
//
// **وقراراتُ المالك ٢٠٢٦-١٠-١٥**: طريقٌ واحدٌ للتعويض (لا «تعويض» من نافذة المحفظة) ·
// تنبيهُ ما ينتظر للماليّة لكلّ نوع (`orders.AlertCompensationPending`) · المستفيدُ يُخبَر
// عند الموافقة بلا ملاحظة المكتب · المرفوضُ يُقترح ثانيةً · و«موافقة/رفض» لا تظهر لمقترحه
// (`can_decide`) · وبطاقاتٌ تتبع الفلاتر وبطاقةُ التأخّر تفتح المتأخّر.
//	٥ · النسبةُ صفرٌ تكتب طلباً بمقترَحٍ صفرٍ والماليّةُ تقرّر.
//
// **والأزرارُ بمعرّف طلب التعويض لا برقم الطلب** — كان الزرُّ يرسل رقمَ الطلب،
// **فطلبٌ تعذّر مع سائقَين متتاليَين يدفع للأوّل** والنافذةُ تعرض اسمَ الثاني.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
)

var (
	// **فوق السقف المالكُ وحدَه** (البند ٢).
	errCompensationAboveCap = httpx.NewError(http.StatusForbidden,
		"compensation_above_cap", "errors.compensation_above_cap")
	// **للطلب أكثرُ من تعويضٍ معلَّق** — فليُسمَّ المقصودُ بمعرّفه.
	errCompensationAmbiguous = httpx.NewError(http.StatusConflict,
		"compensation_ambiguous", "errors.compensation_ambiguous")
	errCompensationDuplicate = httpx.NewError(http.StatusConflict,
		"compensation_duplicate", "errors.compensation_duplicate")
)

// compensationErr **أخطاءُ المحرّك برموزها** — والباقي كما هو.
func compensationErr(err error) error {
	switch {
	case errors.Is(err, orders.ErrCompensationNotPending):
		return errCompensationNotPending
	case errors.Is(err, orders.ErrAlreadyCompensated):
		return errAlreadyCompensated
	case errors.Is(err, orders.ErrCompensationAmbiguous):
		return errCompensationAmbiguous
	case errors.Is(err, orders.ErrCompensationDuplicate):
		return errCompensationDuplicate
	case errors.Is(err, orders.ErrCompensationInvalid),
		errors.Is(err, orders.ErrGoodsBadCompensation):
		return errValidation
	case errors.Is(err, orders.ErrGoodsCompensationCap):
		return errGoodsCompCap
	case errors.Is(err, orders.ErrGoodsAlreadyCompensated):
		return errGoodsCompensated
	}
	return err
}

// approveCompensationTx **الموافقةُ الواحدة** — من الصفحة بمعرّف الطلب، ومن الباب
// القديم (`compensate-driver`). كلُّها هنا: المقترحُ غيرُ الموافق، والسقف،
// والقيدان، والأثر — **في معاملةٍ واحدة.**
func (s *Server) approveCompensationTx(ctx context.Context, q dbtx.Querier, r *http.Request,
	c *orders.CompensationRequest, amount int64, note string) error {
	if c == nil || c.Status != orders.CompensationPending {
		return errCompensationNotPending
	}
	if amount <= 0 || note == "" {
		return errValidation
	}
	actor := userIDFrom(r)
	proposer := ""
	if c.ProposedBy != nil {
		proposer = *c.ProposedBy
	}
	verdict, err := approval.Check(ctx, q, approval.Request{
		ProposedBy: proposer, Actor: actor, Capability: authz.FinanceManage,
	})
	if err != nil {
		return err
	}
	// **فوق السقف المالكُ وحدَه** — والسقفُ صفرٌ «بلا سقف».
	capAmt, err := compensationCapTx(ctx, q, c.Kind)
	if err != nil {
		return err
	}
	if capAmt > 0 && amount > capAmt {
		var owner bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM user_roles
			              WHERE user_id = $1::uuid AND role_code = $2)`,
			actor, authz.RoleOwnerSuperAdmin).Scan(&owner); err != nil {
			return err
		}
		if !owner {
			return errCompensationAboveCap
		}
	}
	if err := s.orders.ApproveCompensationTx(ctx, q, c, amount, actor, clip(note, 300),
		verdict.SelfApproved); err != nil {
		return compensationErr(err)
	}
	entity, entityID := "order", c.OrderID
	if c.Kind == orders.CompKindComplaint {
		entity, entityID = "ticket", c.TicketID
	}
	meta := map[string]any{
		"driver_id": c.DriverID, "beneficiary_id": c.DriverID, "kind": c.Kind,
		"amount": amount, "note": note, "request_id": c.ID,
		"suggested": c.SuggestedAmount, "above_cap": capAmt > 0 && amount > capAmt,
	}
	for k, v := range verdict.AuditFields() {
		meta[k] = v
	}
	return s.auditTx(ctx, q, r, "finance.driver_compensation", entity, entityID, meta)
}

// autoApproveCompensationTx **تعويضٌ اقترحه المالكُ أو المدير ينفُذ فوراً** (قرارُ
// المالك ٢٠٢٦-١٠-٢٧) — بالموافقة الواحدة نفسِها (`approveCompensationTx`) وفي معاملة
// الاقتراح، بمبلغه وملاحظته. **وما فوق سقف النوع لغير المالك يبقى معلَّقاً** كما
// كان — فالسقفُ يُفحص قبل أيّ كتابة. ويُرجع الطلبَ المدفوعَ لإشعار ما بعد التثبيت.
func (s *Server) autoApproveCompensationTx(ctx context.Context, q dbtx.Querier, r *http.Request,
	id string) (*orders.CompensationRequest, int64, error) {
	auto, err := approval.AutoApplies(ctx, q, userIDFrom(r))
	if err != nil || !auto {
		return nil, 0, err
	}
	c, err := s.orders.CompensationByIDTx(ctx, q, id)
	if err != nil || c == nil {
		return nil, 0, err
	}
	amount := c.SuggestedAmount
	if c.Amount != nil && *c.Amount > 0 {
		amount = *c.Amount
	}
	note := strings.TrimSpace(c.Note)
	if note == "" {
		note = "نفذت فوراً بحركة الإدارة"
	}
	if err := s.approveCompensationTx(ctx, q, r, c, amount, note); err != nil {
		if errors.Is(err, errCompensationAboveCap) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return c, amount, nil
}

// compensationCapTx **سقفُ النوع من داخل المعاملة** — لا من المخزن.
//
// **والمخزنُ يأخذ اتّصالاً ثانياً من البِركة**: أربعُ موافقاتٍ متزامنةٍ تمسك كلٌّ
// منها اتّصالاً وتنتظر القفل، **فتستنزف البِركة ويعلق الأوّلُ يطلب الإعداد.**
func compensationCapTx(ctx context.Context, q dbtx.Querier, kind string) (int64, error) {
	key := orders.CompensationCapKey(kind)
	var fallback int64
	if def, ok := settings.Lookup(key); ok {
		if n, ok := def.Default.(int); ok {
			fallback = int64(n)
		}
	}
	var v *float64
	err := q.QueryRow(ctx, `
		SELECT (SELECT CASE WHEN jsonb_typeof(value) = 'number' THEN value::text::float8 END
		        FROM app_settings WHERE key = $1)`, key).Scan(&v)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return fallback, nil
	}
	return int64(*v), nil
}

// compensationPaidTitle **«انصرفلك تعويض X»** — عنوانُ خبر المستفيد (قرارُ المالك ٢٠٢٦-١٠-١٥).
const compensationPaidTitle = "انصرفلك تعويض "

// notifyCompensationPaid **المستفيدُ يُخبَر بتعويضه — في تطبيقه، وبلا ملاحظة المكتب.**
//
// **كانت الموافقةُ صامتة**: يزيد رصيدُ السائق ولا يعرف لماذا. **والتطبيقُ من الدور**:
// تعويضُ السائق لتطبيق السائق، وتعويضُ البضاعة لتطبيق المتجر، وتعويضُ الشكوى لتطبيق
// صاحبها. **والملاحظةُ الداخليّة لا تُكتب أبداً** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٣).
// ويُنادى بعد التثبيت.
func (s *Server) notifyCompensationPaid(ctx context.Context, c *orders.CompensationRequest, amount int64) {
	if c == nil || c.DriverID == "" || amount <= 0 {
		return
	}
	app := notifications.AppCustomer
	switch {
	case c.Kind == orders.CompKindDriver:
		app = notifications.AppDriver
	case c.Kind == orders.CompKindMerchantGoods:
		app = notifications.AppMerchant
	case c.BeneficiaryRole == "driver":
		app = notifications.AppDriver
	case c.BeneficiaryRole == "merchant":
		app = notifications.AppMerchant
	case c.BeneficiaryRole == "rep":
		app = notifications.AppRep
	}
	body := c.ReasonLabel
	if c.OrderNumber > 0 {
		body += " — #" + strconv.FormatInt(c.OrderNumber, 10)
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: c.DriverID, Kind: notifications.KindWallet,
		Title:  compensationPaidTitle + fmtMoneyAr(amount) + " " + currencyWord,
		Body:   body,
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{app},
		// **والتطبيقُ من الدور أعلاه** — AppDriver لتعويض السائق، لا كلُّ تطبيقاته.
	})
	s.touchUser(c.DriverID, "wallet")
}

// handleApproveCompensation **موافقةٌ بمعرّف طلب التعويض** — `{amount, note}`.
func (s *Server) handleApproveCompensation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	var paid *orders.CompensationRequest
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		c, err := s.orders.CompensationByIDTx(ctx, q, id)
		if err != nil {
			return err
		}
		if c == nil {
			return httpx.ErrNotFound
		}
		paid = c
		return s.approveCompensationTx(ctx, q, r, c, req.Amount, note)
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.notifyCompensationPaid(r.Context(), paid, req.Amount)
	s.touch("order", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"compensated": req.Amount, "id": id})
}

// handleRejectCompensationByID **رفضٌ بمعرّف طلب التعويض** — بسببٍ إلزاميّ.
func (s *Server) handleRejectCompensationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.respondErr(w, errValidation)
		return
	}
	var rejected *orders.CompensationRequest
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		c, err := s.orders.CompensationByIDTx(ctx, q, id)
		if err != nil {
			return err
		}
		if c == nil {
			return httpx.ErrNotFound
		}
		rejected = c
		return s.rejectCompensationTx(ctx, q, r, c, note)
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ورفضُ تعويض الشكوى يُخبر الدعم** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — صاحبَ الاقتراح،
	// وإن لم يُعرف فمن يملك إدارةَ الدعم: الشكوى عادت إليهم ليقترحوا ثانيةً أو يحلّوها.
	if rejected != nil && rejected.Kind == orders.CompKindComplaint {
		in := notifications.Input{
			Kind: notifications.KindTicket, Title: notifTitles.complaintCompRejected, Body: note,
			Entity: "ticket", EntityID: rejected.TicketID, Href: "/dashboard/support",
		}
		if rejected.ProposedBy != nil && *rejected.ProposedBy != "" {
			in.UserID = *rejected.ProposedBy
			s.notify.Notify(r.Context(), in)
		} else {
			s.notify.NotifyCaps(r.Context(), []string{string(authz.SupportManage)}, in)
		}
		s.touch("ticket", "ops")
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"rejected": true, "id": id})
}

// rejectCompensationTx **الرفضُ قرارٌ يُكتب** — ولا مالَ يتحرّك.
func (s *Server) rejectCompensationTx(ctx context.Context, q dbtx.Querier, r *http.Request,
	c *orders.CompensationRequest, note string) error {
	if c.Status != orders.CompensationPending {
		return errCompensationNotPending
	}
	if err := s.orders.DecideCompensationTx(ctx, q, c.ID,
		orders.CompensationRejected, 0, userIDFrom(r), clip(note, 300)); err != nil {
		return err
	}
	entity, entityID := "order", c.OrderID
	if c.Kind == orders.CompKindComplaint {
		entity, entityID = "ticket", c.TicketID
		// **والشكوى تعود إلى الدعم** — كانت «بانتظار المالية» (قرارُ المالك ٢٠٢٦-١٠-٠٤).
		if err := support.BackToSupportTx(ctx, q, c.TicketID); err != nil {
			return err
		}
	}
	return s.auditTx(ctx, q, r, "finance.driver_compensation_rejected", entity, entityID, map[string]any{
		"driver_id": c.DriverID, "request_id": c.ID, "kind": c.Kind, "note": note,
	})
}

// handleProposeCompensation **اقتراحُ تعويضِ سائقٍ من المكتب** — لا يدفع.
//
// `{order_id, user_id, amount, fault, reason, note}` — ويوافق عليه غيرُ مقترحه.
func (s *Server) handleProposeCompensation(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		OrderID string `json:"order_id"`
		UserID  string `json:"user_id"`
		Amount  int64  `json:"amount"`
		Fault   string `json:"fault"`
		Reason  string `json:"reason"`
		Note    string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if !isUUID(req.OrderID) || !isUUID(req.UserID) || req.Amount <= 0 || note == "" {
		s.respondErr(w, errValidation)
		return
	}
	var id string
	var paid *orders.CompensationRequest
	var paidAmount int64
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var driverOK bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role_code = 'driver')`,
			req.UserID).Scan(&driverOK); err != nil {
			return err
		}
		if !driverOK {
			return errValidation
		}
		var e error
		id, e = orders.ProposeCompensationTx(ctx, q, orders.CompensationProposal{
			Kind: orders.CompKindDriver, OrderID: req.OrderID, BeneficiaryID: req.UserID,
			Fault: req.Fault, Reason: req.Reason, Amount: req.Amount,
			Note: clip(note, 300), ProposedBy: userIDFrom(r),
		})
		if e != nil {
			return compensationErr(e)
		}
		if err := s.auditTx(ctx, q, r, "finance.compensation_proposed", "order", req.OrderID, map[string]any{
			"request_id": id, "kind": orders.CompKindDriver, "beneficiary_id": req.UserID,
			"amount": req.Amount, "note": note,
		}); err != nil {
			return err
		}
		paid, paidAmount, e = s.autoApproveCompensationTx(ctx, q, r, id)
		return e
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.respondCompensationProposed(w, r, id, paid, paidAmount)
}

// respondCompensationProposed **ردُّ الاقتراح** — وإن نفذ فوراً (حركةُ مالكٍ أو مدير)
// أُخبر المستفيدُ ولا تُنبَّه الماليّةُ على ما لا ينتظرها.
func (s *Server) respondCompensationProposed(w http.ResponseWriter, r *http.Request, id string,
	paid *orders.CompensationRequest, amount int64) {
	if paid != nil {
		s.notifyCompensationPaid(r.Context(), paid, amount)
		s.touch("order", "ops")
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusAccepted, map[string]any{"proposed": id, "status": "approved",
			"applied": true, "compensated": amount})
		return
	}
	// **والماليّةُ تُنبَّه** — بعد التثبيت (قرارُ المالك ٢٠٢٦-١٠-١٥).
	s.orders.AlertCompensationPending(r.Context(), id)
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusAccepted, map[string]any{"proposed": id, "status": "pending"})
}

// handleListCompensations **القائمةُ بتبويباتها وفلاترها وبطاقاتها.**
//
// `?status=pending|approved|rejected|all&kind=&fault=&person=&from=&to=&overdue=1&page=`
func (s *Server) handleListCompensations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := orders.CompensationFilter{
		Status: q.Get("status"), Kind: q.Get("kind"), Fault: q.Get("fault"),
		Person: clip(q.Get("person"), 80), From: q.Get("from"), To: q.Get("to"),
		// **بطاقةُ «تأخّر» تفتح المتأخّرَ وحدَه** (قرارُ المالك ٢٠٢٦-١٠-١٥).
		Overdue: q.Get("overdue") == "1" || q.Get("overdue") == "true",
	}
	switch f.Status {
	case "", "pending", "approved", "rejected", "all":
	default:
		s.respondErr(w, errValidation)
		return
	}
	switch f.Kind {
	case "", orders.CompKindDriver, orders.CompKindMerchantGoods, orders.CompKindComplaint:
	default:
		s.respondErr(w, errValidation)
		return
	}
	if f.Fault != "" && !orders.IsFault(f.Fault) {
		s.respondErr(w, errValidation)
		return
	}
	for _, d := range []string{f.From, f.To} {
		if d == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", d); err != nil {
			s.respondErr(w, errValidation)
			return
		}
	}
	pg := pagingOf(r, 20)
	list, total, err := s.orders.Compensations(r.Context(), f, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **«موافقة/رفض» لا تظهر لمقترحه** (قرارُ المالك ٢٠٢٦-١٠-١٥) — بحكم `approval` نفسِه
	// والمالكُ وحدَه مستثنى. والقدرةُ (`finance.manage`) تحرسها الشاشةُ والوسيط.
	actor := userIDFrom(r)
	ownerAlone, err := approval.OwnerAlone(r.Context(), s.pg, actor, authz.FinanceManage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for i := range list {
		proposer := ""
		if list[i].ProposedBy != nil {
			proposer = *list[i].ProposedBy
		}
		list[i].CanDecide = list[i].Status == orders.CompensationPending &&
			approval.Allowed(actor, ownerAlone, proposer)
	}
	sum, err := s.orders.CompensationsSummary(r.Context(), f)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out := paged("compensations", list, total, pg)
	out["summary"] = sum
	httpx.JSON(w, http.StatusOK, out)
}

// RunCompensationOverdueSweeper **ينبّه المالكَ والماليّةَ إلى ما تأخّر القرارُ فيه** — البند ٤.
func (s *Server) RunCompensationOverdueSweeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.orders.SweepOverdueCompensations(ctx)
		}
	}
}
