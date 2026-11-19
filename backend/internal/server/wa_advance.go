package server

// ══════════════════════════════════════════════════════════════════════
// **«سلفة» من الواتساب** (قرار المالك ٢٠٢٦-١١-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// «السائقين والوكلاء احيانا ياخذو مني مصاري… بدي ابعث اسم السائق او المندوب ع واتس اب والمبلغ والبوت
// يسجله كسلفة… لان احيانا انسا من ضغط العمل.»
//
//	«سلفة أحمد الخلف 50»  أو  «سلفة 0933123456 50»  — بمحادثة المالك مع نفسه **بس** (قراره).
//	أكتر من شخص بنفس الاسم  ←  قائمة بأرقام، والمالك بيردّ بالرقم.
//	بينكتب طلب سلفة عادي **معلّق ١٠ دقايق** (`auto_apply_at`) وبعدين بيتطبّق لحالو بطريق الموافقة نفسه
//	(`decideWalletRequestTx`)؛ و«الغاء» بهالوقت بيرفضه. **ولا قيد جديد بالدفتر** (قراره: «تتسجّل بعد ١٠
//	دقايق» بدل قيد إلغاء جديد).
//
// والأهليّة والسقف متل اللوحة بالضبط: سائق أو وكيل أو موظّف، وتحت `finance.manual_wallet_max`.
//
// **وكل ردّ بيبلّش بإشارة** (✅ ❌ ❓ 🚫) — ردّ البوت بيرجع لنفس المحادثة، فلازم ما يبيّن أمر.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/waitems"
)

// waAdvanceDelay **وقت «الغاء»** قبل ما تتطبّق السلفة.
const waAdvanceDelay = 10 * time.Minute

// waAdvanceNote **ملاحظة الطلب** — بتبيّن باللوحة ومحفظة الشخص.
const waAdvanceNote = "سلفة من الواتساب"

type waAdvanceCand struct {
	id, name, role string
}

// waAdvanceChoice **«أيّ واحد؟» معلّق** — مالك واحد، فحالة وحدة بالذاكرة؛ وبتنتهي بعد ١٠ دقايق.
type waAdvanceChoice struct {
	mu     sync.Mutex
	cands  []waAdvanceCand
	amount int64
	until  time.Time
}

var waAdvanceWords = map[string]bool{"سلفة": true, "سلفه": true}
var waCancelWords = map[string]bool{"الغاء": true, "إلغاء": true, "الغي": true, "ألغي": true, "الغاء السلفة": true, "الغاء السلفه": true}

// normArabic **أ/إ/آ ← ا، ة ← ه، ى ← ي** — ومتلها بالـSQL (`translate`).
func normArabic(s string) string {
	return strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي").Replace(strings.ToLower(strings.TrimSpace(s)))
}

// parseWAAdvance **«سلفة <اسم أو رقم> <مبلغ>»** — المبلغ آخر رقم (و«ألف» بعده بتضرب). ok=false: مو أمر سلفة.
func parseWAAdvance(text string) (who string, amount int64, ok bool) {
	words := strings.Fields(waitems.ToASCIIDigits(strings.TrimSpace(text)))
	if len(words) == 0 || !waAdvanceWords[strings.Trim(words[0], ":،,.")] {
		return "", 0, false
	}
	words = words[1:]
	mult := int64(1)
	if n := len(words); n > 0 {
		switch strings.Trim(words[n-1], ".,") {
		case "ألف", "الف", "آلاف", "الاف":
			mult = 1000
			words = words[:n-1]
		case "ليرة", "ليره", "ل.س":
			words = words[:n-1]
		}
	}
	if len(words) < 2 {
		return strings.Join(words, " "), 0, true
	}
	last := strings.ReplaceAll(strings.Trim(words[len(words)-1], ".،"), ",", "")
	v, err := strconv.ParseInt(last, 10, 64)
	if err != nil || v <= 0 {
		return strings.Join(words, " "), 0, true
	}
	return strings.Join(words[:len(words)-1], " "), v * mult, true
}

// waAdvanceSelf **أمر السلفة من محادثة المالك مع نفسه** — فارغ: مو إلو.
func (s *Server) waAdvanceSelf(ctx context.Context, text string) string {
	t := strings.TrimSpace(strings.Trim(text, "؟?!. "))
	if waCancelWords[t] {
		return s.waAdvanceCancel(ctx)
	}
	if n, err := strconv.Atoi(waitems.ToASCIIDigits(t)); err == nil {
		return s.waAdvancePick(ctx, n)
	}
	who, amount, ok := parseWAAdvance(text)
	if !ok {
		return ""
	}
	if amount <= 0 || who == "" {
		return "❓ اكتبها هيك: سلفة أحمد الخلف 50"
	}
	if max := s.walletManualMax(ctx); amount > max {
		return fmt.Sprintf("🚫 %s أكبر من سقف الحركة اليدويّة (%s). سجّلها من اللوحة.", fmtMoneyAr(amount), fmtMoneyAr(max))
	}
	cands, err := s.waAdvanceFind(ctx, who)
	if err != nil {
		s.logger.Warn("سلفة الواتساب: البحث سقط", "error", err)
		return "❌ صار خطأ، جرّب كمان مرّة."
	}
	switch len(cands) {
	case 0:
		return "❓ ما لقيت كابتن أو وكيل أو موظّف اسمه «" + who + "»."
	case 1:
		return s.waAdvanceCreate(ctx, cands[0], amount)
	}
	s.waChoice.mu.Lock()
	s.waChoice.cands, s.waChoice.amount, s.waChoice.until = cands, amount, time.Now().Add(waAdvanceDelay)
	s.waChoice.mu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "❓ في أكتر من واحد — مين بدّك؟ ردّ بالرقم (سلفة %s):", fmtMoneyAr(amount))
	for i, c := range cands {
		fmt.Fprintf(&b, "\n%d) %s — %s", i+1, c.name, c.role)
	}
	return b.String()
}

func (s *Server) waAdvancePick(ctx context.Context, n int) string {
	s.waChoice.mu.Lock()
	cands, amount, until := s.waChoice.cands, s.waChoice.amount, s.waChoice.until
	if len(cands) == 0 || time.Now().After(until) {
		s.waChoice.mu.Unlock()
		return ""
	}
	if n < 1 || n > len(cands) {
		s.waChoice.mu.Unlock()
		return fmt.Sprintf("❓ اختار رقم من 1 لـ %d", len(cands))
	}
	s.waChoice.cands = nil
	s.waChoice.mu.Unlock()
	return s.waAdvanceCreate(ctx, cands[n-1], amount)
}

// waAdvanceFind **مين بيستاهل سلفة بهالاسم أو الرقم** — متل `advanceEligible`، وتسعة على الأكتر.
func (s *Server) waAdvanceFind(ctx context.Context, who string) ([]waAdvanceCand, error) {
	phone, isPhone := "", false
	if digits := strings.Trim(who, "+ "); digits != "" && strings.Trim(digits, "0123456789") == "" {
		if p, ok := identity.NormalizePhone(who); ok {
			phone, isPhone = p, true
		}
	}
	rows, err := s.pg.Query(ctx, `
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
		       CASE WHEN bool_or(ur.role_code = 'driver') THEN 'كابتن'
		            WHEN bool_or(ur.role_code = 'sales') THEN 'وكيل'
		            ELSE 'موظّف' END
		  FROM users u JOIN user_roles ur ON ur.user_id = u.id
		 WHERE u.deleted_at IS NULL AND u.status = 'active'
		   AND ur.role_code NOT IN ('customer', 'merchant')
		   AND CASE WHEN $2 THEN u.phone = $3
		            ELSE translate(lower(COALESCE(u.full_name, '')), 'أإآةى', 'اااهي') LIKE '%' || $1 || '%' END
		 GROUP BY u.id, u.full_name, u.phone
		 ORDER BY u.full_name LIMIT 9`, normArabic(who), isPhone, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []waAdvanceCand
	for rows.Next() {
		var c waAdvanceCand
		if err := rows.Scan(&c.id, &c.name, &c.role); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// waOwner **المالك الأعلى** — هو صاحب الاقتراح والموافقة (`approval.AutoApplies`).
func (s *Server) waOwner(ctx context.Context, q dbtx.Querier) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		SELECT u.id::text FROM users u JOIN user_roles ur ON ur.user_id = u.id
		 WHERE ur.role_code = $1 AND u.status = 'active' AND u.deleted_at IS NULL
		 ORDER BY u.created_at LIMIT 1`, authz.RoleOwnerSuperAdmin).Scan(&id)
	return id, err
}

// waActorRequest **طلبٌ مركّب للسجلّ** — فاعله المالك، ومن الواتساب.
func waActorRequest(ctx context.Context, actor string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/wa/advance", nil)
	r.RemoteAddr = "127.0.0.1:0"
	r.Header.Set("User-Agent", "whatsapp-bot")
	return r.WithContext(context.WithValue(ctx, ctxUserID, actor))
}

func (s *Server) waAdvanceCreate(ctx context.Context, c waAdvanceCand, amount int64) string {
	err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
		owner, err := s.waOwner(ctx, q)
		if err != nil {
			return err
		}
		if err := advanceEligible(ctx, q, c.id); err != nil {
			return err
		}
		var id string
		if err := q.QueryRow(ctx, `
			INSERT INTO wallet_requests (user_id, kind, debit, amount, note, proposed_by, auto_apply_at)
			VALUES ($1, 'advance', true, $2, $3, $4, now() + make_interval(secs => $5))
			RETURNING id::text`, c.id, amount, waAdvanceNote, owner, waAdvanceDelay.Seconds()).Scan(&id); err != nil {
			return err
		}
		return s.auditTx(ctx, q, waActorRequest(ctx, owner), "finance.wallet_request", "user", c.id,
			map[string]any{"request_id": id, "kind": "advance", "debit": true, "amount": amount,
				"note": waAdvanceNote, "source": "whatsapp"})
	})
	if err != nil {
		s.logger.Warn("سلفة الواتساب: التسجيل سقط", "error", err)
		return "❌ ما انسجّلت السلفة — جرّب من اللوحة."
	}
	s.touch("wallet", "ops")
	return fmt.Sprintf("✅ سلفة %s لـ %s (%s) — بتنخصم من محفظته بعد ١٠ دقايق.\nإذا غلطت اكتب: الغاء",
		fmtMoneyAr(amount), c.name, c.role)
}

// waAdvanceCancel **«الغاء»** — آخر سلفة واتساب لسا ما تطبّقت.
func (s *Server) waAdvanceCancel(ctx context.Context) string {
	var name string
	var amount int64
	err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
		var id string
		if err := q.QueryRow(ctx, `
			SELECT wr.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''), wr.amount
			  FROM wallet_requests wr JOIN users u ON u.id = wr.user_id
			 WHERE wr.status = 'pending' AND wr.auto_apply_at IS NOT NULL
			 ORDER BY wr.created_at DESC LIMIT 1`).Scan(&id, &name, &amount); err != nil {
			return err
		}
		owner, err := s.waOwner(ctx, q)
		if err != nil {
			return err
		}
		_, _, err = s.decideWalletRequestTx(ctx, q, waActorRequest(ctx, owner), id, owner, false, "انلغت من الواتساب")
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "❓ ما في سلفة من الواتساب ناطرة لتنلغى (يا إمّا تطبّقت يا إمّا ما في)."
	}
	if err != nil {
		s.logger.Warn("سلفة الواتساب: الإلغاء سقط", "error", err)
		return "❌ ما انلغت — شيلها من اللوحة."
	}
	s.touch("wallet", "ops")
	return fmt.Sprintf("✅ انلغت سلفة %s لـ %s.", fmtMoneyAr(amount), name)
}

// SweepWAAdvanceOnce **تطبيق السلف اللي خلص وقتها** — بطريق الموافقة نفسه، وإشعار صاحبها.
func (s *Server) SweepWAAdvanceOnce(ctx context.Context) {
	rows, err := s.pg.Query(ctx, `
		SELECT id::text FROM wallet_requests
		 WHERE status = 'pending' AND auto_apply_at IS NOT NULL AND auto_apply_at <= now()
		 ORDER BY auto_apply_at LIMIT 20`)
	if err != nil {
		s.logger.Warn("سلفة الواتساب: القراءة سقطت", "error", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		var wr walletRequestRow
		var r *http.Request
		err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
			owner, err := s.waOwner(ctx, q)
			if err != nil {
				return err
			}
			r = waActorRequest(ctx, owner)
			wr, _, err = s.decideWalletRequestTx(ctx, q, r, id, owner, true, "")
			return err
		})
		if errors.Is(err, errRequestDecided) {
			continue
		}
		if err != nil {
			// **ما انطبقت** (متلاً انشال دوره) — بتنرفض بسببها وبينقال للمالك، لا بتضل تعيد كل نبضة.
			s.logger.Warn("سلفة الواتساب: التطبيق سقط", "request", id, "error", err)
			var name string
			var amount int64
			_ = s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
				if err := q.QueryRow(ctx, `SELECT COALESCE(NULLIF(u.full_name, ''), u.phone, ''), wr.amount
					FROM wallet_requests wr JOIN users u ON u.id = wr.user_id WHERE wr.id = $1`, id).Scan(&name, &amount); err != nil {
					return err
				}
				owner, err := s.waOwner(ctx, q)
				if err != nil {
					return err
				}
				_, _, err = s.decideWalletRequestTx(ctx, q, waActorRequest(ctx, owner), id, owner, false, "ما انطبقت من الواتساب")
				return err
			})
			if s.alertOut != nil {
				_ = s.alertOut.SendToSelf(ctx, fmt.Sprintf("❌ سلفة %s لـ %s ما انطبقت — سجّلها من اللوحة.", fmtMoneyAr(amount), name))
			}
			continue
		}
		s.afterWalletDecision(r, wr, true, "", 0)
	}
}

// RunWAAdvance **الراصد** — كل نبضة بيطبّق اللي خلص وقتها.
func (s *Server) RunWAAdvance(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.SweepWAAdvanceOnce(ctx)
		}
	}
}
