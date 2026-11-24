package server

// ══════════════════════════════════════════════════════════════════════
// **شحن المحفظة بشام كاش + «شام كاش عند الاستلام»** — قرار المالك ٢٠٢٦-١١-٢٥
// ══════════════════════════════════════════════════════════════════════
//
//	الزبون   GET  /my/shamcash                 الإعدادات (اسم الحساب ورقمه والباركود) وطلباته الأخيرة
//	         POST /my/shamcash/receipt         صورة الإيصال (اختياريّة)
//	         POST /my/shamcash/topups          {amount, txn_ref, receipt_id?} ⇒ طلب معلّق + واتساب للمالك
//	اللوحة   GET  /admin/shamcash-topups       القائمة بمرشّحاتها (finance.read)
//	         POST /admin/shamcash-topups/{id}/approve   «تم، اشحن» (finance.manage) — مرّة وحدة
//	         POST /admin/shamcash-topups/{id}/reject    «مرفوض» بسبب (finance.manage)
//
// **الموافقة بمعاملة وحدة**: قيد `topup` بمحفظة الزبون (مرجعه معرّف الطلب) + حركة `customer_topup`
// داخلة بسجلّ شام كاش + سدّ الدين أوّلاً متل الشحن النقديّ + أثر بالتدقيق. وكبسة تانية = ولا شي.
//
// **والتأكيد الآليّ لاحقاً** من `shamcash.Verifier` (`s.shamcashVerifier()`): اليوم يدويّ.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/shamcash"
)

// errShamcashCustomerOnly **الشحن بشام كاش للزبون بس** — محفظة السائق والمندوب بتنسحب، والشحن
// من برّا عليها باب لتصريف مال.
var errShamcashCustomerOnly = httpx.NewError(http.StatusForbidden, "shamcash_customer_only", "errors.forbidden")

// shamcashVerifier **مين بيأكّد التحويل** — يدويّ لحتى يصير في ربط رسميّ.
func (s *Server) shamcashVerifier() shamcash.Verifier {
	if s.scVerifier != nil {
		return s.scVerifier
	}
	return shamcash.ManualVerifier{}
}

// SetShamcashVerifier **يركّب متحقّقاً رسميّاً** (createBill/webhook) — ما بيتغيّر شي عند الزبون.
func (s *Server) SetShamcashVerifier(v shamcash.Verifier) { s.scVerifier = v }

// shamcashInfo **ما يلزم الزبون ليحوّل** — من الإعدادات وحدها.
type shamcashInfo struct {
	TopupEnabled         bool    `json:"topup_enabled"`
	PayOnDeliveryEnabled bool    `json:"pay_on_delivery_enabled"`
	MinTopup             int64   `json:"min_topup"`
	AccountName          string  `json:"account_name"`
	AccountNumber        string  `json:"account_number"`
	QRURL                *string `json:"qr_url"`
}

func (s *Server) shamcashInfoFor(r *http.Request) shamcashInfo {
	ctx := r.Context()
	info := shamcashInfo{
		TopupEnabled:         s.settings.GetBool(ctx, "shamcash.topup_enabled"),
		PayOnDeliveryEnabled: s.settings.GetBool(ctx, "shamcash.pay_on_delivery_enabled"),
		MinTopup:             s.settings.GetInt(ctx, "shamcash.min_topup"),
		AccountName:          strings.TrimSpace(s.settings.GetString(ctx, "shamcash.account_name")),
		AccountNumber:        strings.TrimSpace(s.settings.GetString(ctx, "shamcash.account_number")),
		QRURL:                s.settingMedia(r, "shamcash.qr"),
	}
	// **بلا رقم حساب ما في شي يتحوّل عليه** — المفتاحين بيتصرّفوا كأنهم طافيين.
	if info.AccountNumber == "" {
		info.TopupEnabled, info.PayOnDeliveryEnabled = false, false
	}
	return info
}

// myTopup **طلب الشحن كما يراه صاحبه** — بلا اسم الموظّف ولا رابط الإيصال.
type myTopup struct {
	ID           string    `json:"id"`
	Amount       int64     `json:"amount"`
	TxnRef       string    `json:"txn_ref"`
	Status       string    `json:"status"`
	RejectReason string    `json:"reject_reason"`
	CreatedAt    time.Time `json:"created_at"`
}

func toMyTopups(list []shamcash.Topup) []myTopup {
	out := make([]myTopup, 0, len(list))
	for _, t := range list {
		out = append(out, myTopup{ID: t.ID, Amount: t.Amount, TxnRef: t.TxnRef, Status: t.Status,
			RejectReason: t.RejectReason, CreatedAt: t.CreatedAt})
	}
	return out
}

// handleMyShamcash **`GET /my/shamcash`** — الإعدادات وآخر عشر طلبات.
func (s *Server) handleMyShamcash(w http.ResponseWriter, r *http.Request) {
	list, _, err := shamcash.ListTopups(r.Context(), s.pg, shamcash.TopupFilter{UserID: userIDFrom(r)}, 10, 0)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"info": s.shamcashInfoFor(r), "requests": toMyTopups(list)})
}

// isCustomer **أيحمل دور الزبون؟**
func (s *Server) isCustomer(ctx context.Context, userID string) bool {
	var ok bool
	_ = s.pg.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1::uuid AND role_code = 'customer')`,
		userID).Scan(&ok)
	return ok
}

// handleMyShamcashReceipt **`POST /my/shamcash/receipt`** — صورة الإيصال (النوع مثبَّت).
func (s *Server) handleMyShamcashReceipt(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	if !s.isCustomer(r.Context(), uid) {
		s.respondErr(w, errShamcashCustomerOnly)
		return
	}
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()
	// **سقف يوميّ** — نقطة رفع بلا سقف بتعبّي القرص.
	var today int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM media WHERE kind = 'shamcash_receipt' AND created_by = $1
		   AND created_at > now() - interval '1 day'`, uid).Scan(&today); err != nil {
		s.respondErr(w, err)
		return
	}
	if today >= 12 {
		s.respondErr(w, errValidation)
		return
	}
	mm, err := s.media.Save(r.Context(), uid, "shamcash_receipt", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":        mm.ID,
		"url":       media.SignedURL(strings.TrimPrefix(mm.URL, "/media/")),
		"thumb_url": media.SignedURL(strings.TrimPrefix(mm.ThumbURL, "/media/")),
	})
}

// handleCreateShamcashTopup **`POST /my/shamcash/topups`** `{amount, txn_ref, receipt_id?}`.
func (s *Server) handleCreateShamcashTopup(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount    *float64 `json:"amount"`
		TxnRef    string   `json:"txn_ref"`
		ReceiptID string   `json:"receipt_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	uid := userIDFrom(r)
	if !s.isCustomer(r.Context(), uid) {
		s.respondErr(w, errShamcashCustomerOnly)
		return
	}
	info := s.shamcashInfoFor(r)
	if !info.TopupEnabled {
		s.respondErr(w, shamcash.ErrTopupDisabled)
		return
	}
	// **ليرة صحيحة بس** — `AmountUnits` بيرفض الكسور.
	if req.Amount == nil {
		s.respondErr(w, errValidation)
		return
	}
	amount, ok := amountSYP(*req.Amount)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	if amount < info.MinTopup || amount <= 0 {
		s.respondErr(w, shamcash.ErrTopupMin)
		return
	}
	txn := shamcash.NormalizeTxn(req.TxnRef)
	if txn == "" {
		s.respondErr(w, shamcash.ErrTxnInvalid)
		return
	}
	var t shamcash.Topup
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		id, err := shamcash.CreateTopup(ctx, q, uid, amount, txn, strings.TrimSpace(req.ReceiptID))
		if err != nil {
			return err
		}
		if err := s.auditTx(ctx, q, r, "finance.shamcash_topup_requested", "shamcash_topup", id,
			map[string]any{"amount": amount, "txn_ref": txn, "receipt": req.ReceiptID != ""}); err != nil {
			return err
		}
		t, err = shamcash.GetTopup(ctx, q, id)
		return err
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	s.touchUser(uid, "wallet")
	s.alertShamcashTopup(t)
	s.autoVerifyShamcash(t)
	httpx.JSON(w, http.StatusCreated, map[string]any{"request": toMyTopups([]shamcash.Topup{t})[0]})
}

// amountSYP **مبلغ بالليرة الجديدة** — صحيح موجب.
func amountSYP(v float64) (int64, bool) {
	if v <= 0 || v > 1e12 || v != float64(int64(v)) {
		return 0, false
	}
	return int64(v), true
}

// alertShamcashTopup **واتساب للمالك** بنفس باب تنبيهات المراقبة (`alertOwner`) — وإلّا إشعار اللوحة.
func (s *Server) alertShamcashTopup(t shamcash.Topup) {
	text := "💳 طلب شحن محفظة بشام كاش\n" +
		"الزبون: " + t.UserName + "\n" +
		"المبلغ: " + fmtMoneyAr(t.Amount) + "\n" +
		"رقم العملية: " + t.TxnRef + "\n" +
		"تأكّد إنه وصل لحسابك ووافق من هون: " + s.siteURL() + "/dashboard/shamcash-topups"
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.alertOwner(ctx, text)
	}()
}

// autoVerifyShamcash **الفاصل للتأكيد الآليّ** — اليدويّ بيقول «ما بعرف» فما بيصير شي.
func (s *Server) autoVerifyShamcash(t shamcash.Topup) {
	v := s.shamcashVerifier()
	if _, manual := v.(shamcash.ManualVerifier); manual {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		res, err := v.Verify(ctx, t)
		if err != nil {
			s.logger.Warn("شام كاش: تعذّر التحقّق الآليّ", "topup", t.ID, "error", err)
			return
		}
		switch res.Verdict {
		case shamcash.VerdictConfirmed:
			var out shamcash.Topup
			var paid int64
			if err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
				var err error
				out, _, paid, err = s.approveShamcashTopupTx(ctx, q, t.ID, v.Name(), "")
				return err
			}); err != nil {
				s.logger.Warn("شام كاش: تعذّرت الموافقة الآليّة", "topup", t.ID, "error", err)
				return
			}
			s.afterShamcashDecision(ctx, out, paid)
		case shamcash.VerdictRejected:
			var out shamcash.Topup
			if err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
				var err error
				out, _, err = s.rejectShamcashTopupTx(ctx, q, t.ID, res.Reason, v.Name(), "")
				return err
			}); err != nil {
				return
			}
			s.afterShamcashDecision(ctx, out, 0)
		}
	}()
}

// approveShamcashTopupTx **«تم، اشحن»** — بمعاملة المستدعي. `already` = كانت مقبولة (ولا أثر جديد).
//
// القيد `topup` بمحفظة الزبون ومرجعه معرّف الطلب، والحركة داخلة بالسجلّ، **والشحن بيسدّ الدين أوّلاً**
// متل الشحن النقديّ (`payDebtFromTopup`). **و`actor` فاضي** = متحقّق آليّ (بلا موظّف).
func (s *Server) approveShamcashTopupTx(ctx context.Context, q dbtx.Querier, id, verifier, actor string) (
	t shamcash.Topup, already bool, debtPaid int64, err error) {
	userID, amount, txn, status, err := shamcash.LockTopup(ctx, q, id)
	if err != nil {
		return t, false, 0, err
	}
	switch status {
	case shamcash.TopupApproved:
		t, err = shamcash.GetTopup(ctx, q, id)
		return t, true, 0, err
	case shamcash.TopupRejected:
		return t, false, 0, shamcash.ErrTopupDecided
	}
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}
	note := "شحن بشام كاش · رقم العملية " + txn
	_, txID, err := s.wallet.ApplyTxID(ctx, q, userID, amount, "topup", id, note, actorPtr)
	if err != nil {
		return t, false, 0, err
	}
	movID, err := shamcash.RecordTopupMovement(ctx, q, id, amount, actor)
	if err != nil {
		return t, false, 0, err
	}
	if err := shamcash.MarkApproved(ctx, q, id, txID, movID, verifier, actor); err != nil {
		return t, false, 0, err
	}
	var treasury string
	if err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1), '')`).
		Scan(&treasury); err != nil {
		return t, false, 0, err
	}
	debtPaid, err = obligations.PayFromCredit(ctx, q, userID, amount, id, actorPtr, obligations.CreditPay{
		Wallet: s.wallet, TreasuryID: treasury, Method: obligations.MethodTopup, From: "الشحن",
	})
	if err != nil {
		return t, false, 0, err
	}
	if t, err = shamcash.GetTopup(ctx, q, id); err != nil {
		return t, false, 0, err
	}
	return t, false, debtPaid, nil
}

// rejectShamcashTopupTx **«مرفوض»** بسبب — والمرفوض مرّة تانية ولا شي.
func (s *Server) rejectShamcashTopupTx(ctx context.Context, q dbtx.Querier, id, reason, verifier, actor string) (
	t shamcash.Topup, already bool, err error) {
	_, _, _, status, err := shamcash.LockTopup(ctx, q, id)
	if err != nil {
		return t, false, err
	}
	switch status {
	case shamcash.TopupRejected:
		t, err = shamcash.GetTopup(ctx, q, id)
		return t, true, err
	case shamcash.TopupApproved:
		return t, false, shamcash.ErrTopupDecided
	}
	if err := shamcash.MarkRejected(ctx, q, id, reason, verifier, actor); err != nil {
		return t, false, err
	}
	t, err = shamcash.GetTopup(ctx, q, id)
	return t, false, err
}

// afterShamcashDecision **إشعار الزبون** بعد التثبيت: «انشحنت محفظتك بـ X» أو السبب.
func (s *Server) afterShamcashDecision(ctx context.Context, t shamcash.Topup, debtPaid int64) {
	in := notifications.Input{UserID: t.UserID, Kind: notifications.KindWallet,
		Entity: "wallet", Href: "/portal/wallet"}
	if t.Status == shamcash.TopupApproved {
		in.Title = "انشحنت محفظتك بـ " + fmtMoneyAr(t.Amount)
		in.Body = "وصل تحويل شام كاش (رقم العملية " + t.TxnRef + ") وانضاف المبلغ لمحفظتك."
		if debtPaid > 0 {
			in.Body += " وانسدّ منه " + fmtMoneyAr(debtPaid) + " من الدين اللي عليك."
		}
	} else {
		in.Title = "ما انشحنت محفظتك بشام كاش"
		in.Body = fmtMoneyAr(t.Amount) + " · رقم العملية " + t.TxnRef + " — السبب: " + t.RejectReason
	}
	s.notify.Notify(ctx, in)
	s.touchUser(t.UserID, "wallet")
	s.touch("wallet", "ops")
}

// handleListShamcashTopups **`GET /admin/shamcash-topups?status&q&from&to&page`**.
func (s *Server) handleListShamcashTopups(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	f := shamcash.TopupFilter{Status: qs.Get("status"), Search: clip(qs.Get("q"), 80), From: qs.Get("from"), To: qs.Get("to")}
	switch f.Status {
	case "", shamcash.TopupPending, shamcash.TopupApproved, shamcash.TopupRejected:
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
	list, total, err := shamcash.ListTopups(r.Context(), s.pg, f, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for i := range list {
		list[i].ReceiptURL = media.SignedURLPtr(list[i].ReceiptPath)
	}
	pending, err := shamcash.CountPending(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	body := paged("requests", list, total, pg)
	body["pending"] = pending
	httpx.JSON(w, http.StatusOK, body)
}

// handleApproveShamcashTopup **«تم، اشحن»** — `POST /admin/shamcash-topups/{id}/approve`. مرّة وحدة.
func (s *Server) handleApproveShamcashTopup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actor := userIDFrom(r)
	var t shamcash.Topup
	var already bool
	var paid int64
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var err error
		t, already, paid, err = s.approveShamcashTopupTx(ctx, q, id, s.shamcashVerifierNameManual(), actor)
		if err != nil || already {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.shamcash_topup_approved", "user", t.UserID, map[string]any{
			"topup_id": id, "amount": t.Amount, "txn_ref": t.TxnRef, "debt_paid": paid,
			"movement_id": t.MovementID, "verifier": t.Verifier,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if !already {
		s.afterShamcashDecision(r.Context(), t, paid)
	}
	t.ReceiptURL = media.SignedURLPtr(t.ReceiptPath)
	httpx.JSON(w, http.StatusOK, map[string]any{"request": t, "already": already})
}

// shamcashVerifierNameManual **قرار الموظّف بإيده** دايماً `manual` — حتى لو في متحقّق آليّ مركّب.
func (s *Server) shamcashVerifierNameManual() string { return shamcash.ManualVerifier{}.Name() }

// handleRejectShamcashTopup **«مرفوض»** بسبب إلزاميّ — `POST /admin/shamcash-topups/{id}/reject` `{reason}`.
func (s *Server) handleRejectShamcashTopup(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := clip(strings.TrimSpace(req.Reason), 300)
	if reason == "" {
		s.respondErr(w, errNoteRequired)
		return
	}
	id := chi.URLParam(r, "id")
	var t shamcash.Topup
	var already bool
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var err error
		t, already, err = s.rejectShamcashTopupTx(ctx, q, id, reason, s.shamcashVerifierNameManual(), userIDFrom(r))
		if err != nil || already {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.shamcash_topup_rejected", "user", t.UserID, map[string]any{
			"topup_id": id, "amount": t.Amount, "txn_ref": t.TxnRef, "reason": reason,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if !already {
		s.afterShamcashDecision(r.Context(), t, 0)
	}
	t.ReceiptURL = media.SignedURLPtr(t.ReceiptPath)
	httpx.JSON(w, http.StatusOK, map[string]any{"request": t, "already": already})
}
