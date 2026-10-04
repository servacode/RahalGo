// Package support نظام التذاكر والتعويضات (PLAN §6.7):
// شكوى تُفتح لزبون (مرتبطة بطلب اختيارياً)، خيط ردود، وحلّ بتعويضٍ
// اختياريٍّ **يقترحه الدعمُ وتقرّره الماليّة** لصاحب الشكوى (٢٠٢٦-١٠-٠٤).
package support

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var ErrTicketClosed = httpx.NewError(http.StatusConflict, "ticket_resolved", "errors.ticket_resolved")

// ErrTicketAwaitingFinance **تعويضُها معلَّقٌ عند الماليّة** — لا حلَّ ثانياً حتّى تقرّر
// (قرارُ المالك ٢٠٢٦-١٠-٠٤): الموافقةُ تحلّها، والرفضُ يعيدها إلى الدعم.
var ErrTicketAwaitingFinance = httpx.NewError(http.StatusConflict,
	"ticket_awaiting_finance", "errors.ticket_awaiting_finance")

// StatusAwaitingFinance **«بانتظار المالية»** — شكوى حلُّها تعويضٌ لم تقرّره الماليّةُ بعد.
const StatusAwaitingFinance = "awaiting_finance"

type Reply struct {
	ID        int64     `json:"id"`
	AuthorID  *string   `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Ticket struct {
	ID            string  `json:"id"`
	Number        int64   `json:"number"`
	CustomerID    string  `json:"customer_id"`
	CustomerPhone string  `json:"customer_phone"`
	CustomerName  string  `json:"customer_name"`
	OrderID       *string `json:"order_id"`
	OrderNumber   *int64  `json:"order_number"`
	Subject       string  `json:"subject"`
	// Reason رمزُ سببٍ من `ComplaintReasons` — فارغٌ في تذكرةٍ فتحها موظّف.
	//
	// **والمصنَّفُ يُعدّ**: «كم شكوى ‹لم يصلني طلبي› هذا الشهر» سؤالٌ له جوابٌ
	// الآن، **وكان قبلَه بحثاً في نصوصٍ حرّة.**
	Reason string `json:"reason"`
	// OpenedByCustomer فتحها صاحبُها بنفسه لا موظّفٌ عنه.
	// AgainstUserID **ضدّ مَن هي** — `nil` تعني «على المنصّة».
	//
	// (شكوى المالك ٢٠٢٦-٠٨-٠٩: «مين ضد مين وكلّ شخص ياخذ حقّه».)
	//
	// **ولا يُرسَل إلى الزبون**: هو يعرف ما وقع لا مَن يُنذَر، **ومن رأى اسمَ
	// من اشتُكي عليه قد يقصده خارجَ المنصّة.** (يُقرأ في الخادم للإشعار،
	// ولا يُسمّى في الردّ.)
	AgainstUserID *string `json:"-"`

	OpenedByCustomer bool `json:"opened_by_customer"`
	// ComplainantID **صاحبُ الشكوى** — من يُعوَّض ومن يُخبَر بالردّ.
	//
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «التعويضُ لصاحب الشكوى — السائقُ أو المتجرُ
	// إن كانا هما من اشتكى».) **كان التعويضُ يُقيَّد للزبون دائماً** — وبلاغُ
	// سائقٍ على زبونٍ يُعوِّض الزبونَ المشتكى عليه.
	ComplainantID    string `json:"complainant_id"`
	ComplainantName  string `json:"complainant_name"`
	ComplainantPhone string `json:"complainant_phone"`
	// ComplainantKind `customer` · `driver` · `merchant`.
	ComplainantKind string `json:"complainant_kind"`
	Status          string `json:"status"`
	// Compensation **ما دُفع فعلاً** — لا ما اقتُرح: طلبٌ معلَّقٌ أو مرفوضٌ صفر.
	Compensation int64 `json:"compensation"`
	// CompensationProposed المبلغُ الذي اقترحه الدعمُ على الماليّة (صفرٌ بلا اقتراح).
	CompensationProposed int64 `json:"compensation_proposed"`
	// CompensationStatus حالُ الاقتراح: `pending` · `approved` · `rejected` · وفارغٌ بلا اقتراح.
	CompensationStatus string `json:"compensation_status"`
	// Late **تأخّر الردّ** — مفتوحةٌ لم يردّ عليها المكتبُ بعد مهلة الإعدادات
	// (`support.late_reply_hours`، ساعتان) — تُعلَّم بالأحمر في اللوحة.
	Late       bool       `json:"late"`
	Resolution string     `json:"resolution"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	Replies    []Reply    `json:"replies,omitempty"`
}

type TicketPage struct {
	Tickets []Ticket `json:"tickets"`
	Total   int      `json:"total"`
	// Late **كم شكوى متأخّرةً الآن** — كلُّها لا ما في الصفحة، والمهلةُ `LateHours`.
	Late int `json:"late"`
	// LateHours المهلةُ التي حُسب عليها التأخّر — تقولها الشاشةُ بجانب العدد.
	LateHours int `json:"late_hours"`
	Page      int `json:"page"`
	PerPage   int `json:"per_page"`
}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
	wallet   *wallet.Service
	// settings مهلةُ الشكوى وما يتبعها — يملك المالكُ ضبطَها من اللوحة.
	settings *settings.Store
	// compProposer بابُ اقتراح التعويض — فارغُه طلبُ محفظة (`walletRequestProposer`).
	compProposer CompensationProposer
}

func NewService(db *pgxpool.Pool, identitySvc *identity.Service, walletSvc *wallet.Service) *Service {
	return &Service{db: db, identity: identitySvc, wallet: walletSvc}
}

// SetSettings يحقن مخزن الإعدادات (يُنادى مرّة عند الإقلاع).
func (s *Service) SetSettings(st *settings.Store) { s.settings = st }

// ══════════════════════════════════════════════════════════════════════
// **بابُ اقتراح التعويض — نداءٌ واحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════

// ErrCompensationOverCap مبلغٌ فوق سقف الحركة اليدويّة (`finance.manual_wallet_max`).
var ErrCompensationOverCap = httpx.NewError(http.StatusBadRequest,
	"wallet_over_cap", "errors.wallet_over_cap")

// CompensationProposal ما يقترحه الدعمُ على الماليّة من تذكرة.
type CompensationProposal struct {
	TicketID      string
	BeneficiaryID string // صاحبُ الشكوى
	Amount        int64
	Note          string
	ProposedBy    string
}

// CompensationProposer **يكتب الاقتراحَ معلَّقاً ويردّ معرّفَه** — داخل معاملة الحلّ.
// **ولا يدفع شيئاً**: الدفعُ عند موافقة الماليّة، من الخزينة.
type CompensationProposer interface {
	ProposeCompensation(ctx context.Context, q dbtx.Querier, p CompensationProposal) (string, error)
}

// SetCompensationProposer يبدّل بابَ الاقتراح — لبابِ التعويضات الموحّد حين يُنشر.
func (s *Service) SetCompensationProposer(p CompensationProposer) { s.compProposer = p }

func (s *Service) proposer() CompensationProposer {
	if s.compProposer != nil {
		return s.compProposer
	}
	return walletRequestProposer{}
}

// compensationCap سقفُ الاقتراح الواحد — سقفُ الحركة اليدويّة نفسُه.
func (s *Service) compensationCap(ctx context.Context) int64 {
	if s.settings == nil {
		return 500000
	}
	return s.settings.GetNum(ctx, "finance.manual_wallet_max", 500000)
}

// walletRequestProposer **الافتراض**: طلبُ محفظةٍ من نوع `compensation` في
// `wallet_requests` — تقرّره الماليّةُ من بابه القائم (`/admin/wallet-requests`)،
// **وعند الموافقة يُقيَّد بطرفين: +المحفظة −الخزينة.**
type walletRequestProposer struct{}

func (walletRequestProposer) ProposeCompensation(ctx context.Context, q dbtx.Querier,
	p CompensationProposal) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO wallet_requests (user_id, kind, amount, note, proposed_by)
		VALUES ($1, 'compensation', $2, $3, $4) RETURNING id::text`,
		p.BeneficiaryID, p.Amount, p.Note, p.ProposedBy).Scan(&id)
	return id, err
}

// ComplainantSQL **صاحبُ الشكوى نصّاً واحداً** — على اسم الجدول `t`.
//
// شكوى الزبون (`opened_by_customer`) وتذكرةُ الموظّف صاحبُهما الزبون. **وبلاغُ
// السائق أو المتجر** (سببٌ مصنَّفٌ وليس من الزبون) صاحبُه كاتبُه `created_by`.
const ComplainantSQL = `(CASE WHEN NOT t.opened_by_customer AND COALESCE(t.reason, '') <> ''
	AND t.created_by IS NOT NULL THEN t.created_by ELSE t.customer_id END)`

// PaidCompensationSQL **ما دُفع فعلاً من تعويض التذكرة** — على اسم الجدول `t`.
//
// **عمودُ `compensation` أوّلاً** — دُفع لحظةَ الحلّ قبل ٢٠٢٦-١٠-٠٤، **ويكتبه
// بابُ التعويضات الموحّد عند الموافقة.** وإن بقي صفراً قُرئ طلبُ المحفظة
// المعتمَد (بابُ الاقتراح الافتراضيّ). **ولا شيءَ قبل موافقة الماليّة.**
const PaidCompensationSQL = `(CASE WHEN t.compensation > 0 OR t.compensation_request_id IS NULL
	THEN t.compensation
	ELSE COALESCE((SELECT wrq.amount FROM wallet_requests wrq
		WHERE wrq.id = t.compensation_request_id AND wrq.status = 'approved'), 0) END)`

const ticketSelect = `
	SELECT t.id, t.number, t.customer_id, cu.phone, cu.full_name,
	       t.order_id, o.number, t.subject, COALESCE(t.reason,''), t.against_user_id::text,
	       t.opened_by_customer,
	       cp.id::text, COALESCE(cp.full_name, ''), COALESCE(cp.phone, ''),
	       CASE WHEN cp.id = t.customer_id THEN 'customer'
	            WHEN EXISTS (SELECT 1 FROM merchants mm
	                         WHERE mm.id = o.merchant_id AND mm.owner_user_id = cp.id) THEN 'merchant'
	            ELSE 'driver' END,
	       t.status, ` + PaidCompensationSQL + `,
	       COALESCE(wr.amount, dcr.suggested_amount, 0), COALESCE(wr.status, dcr.status, ''),
	       t.resolution, t.created_at, t.resolved_at
	FROM tickets t
	JOIN users cu ON cu.id = t.customer_id
	JOIN users cp ON cp.id = ` + ComplainantSQL + `
	LEFT JOIN orders o ON o.id = t.order_id
	LEFT JOIN wallet_requests wr ON wr.id = t.compensation_request_id
	-- **وطابورُ التعويضات الموحّد** — حين يُبدَّل بابُ الاقتراح إليه.
	LEFT JOIN driver_compensation_requests dcr ON dcr.id = t.compensation_request_id`

func scanTicket(row pgx.Row) (*Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.Number, &t.CustomerID, &t.CustomerPhone, &t.CustomerName,
		&t.OrderID, &t.OrderNumber, &t.Subject, &t.Reason, &t.AgainstUserID,
		&t.OpenedByCustomer,
		&t.ComplainantID, &t.ComplainantName, &t.ComplainantPhone, &t.ComplainantKind,
		&t.Status, &t.Compensation, &t.CompensationProposed, &t.CompensationStatus,
		&t.Resolution, &t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// LateHours **مهلةُ الردّ على الشكوى بالساعات** — من الإعدادات
// (`support.late_reply_hours`)، **وساعتان** إن لم تُضبط (قرارُ المالك ٢٠٢٦-١٠-٠٤:
// «ساعتين تمام»).
func (s *Service) LateHours(ctx context.Context) int {
	if s.settings == nil {
		return 2
	}
	h := int(s.settings.GetInt(ctx, "support.late_reply_hours"))
	if h <= 0 {
		h = 2
	}
	return h
}

// markLate يعلّم المتأخّرةَ بالشرط نفسِه الذي يعدّها (`ticketWhere`).
func markLate(t *Ticket, hours int, now time.Time) {
	t.Late = t.Status == "open" && t.CreatedAt.Before(now.Add(-time.Duration(hours)*time.Hour))
}

type CreateInput struct {
	CustomerPhone string `json:"customer_phone"`
	OrderNumber   *int64 `json:"order_number"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
}

func (s *Service) Create(ctx context.Context, actorID string, in CreateInput, ip string) (*Ticket, error) {
	if in.Subject == "" || in.CustomerPhone == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	customer, err := s.identity.EnsureUserWithRole(ctx, actorID, in.CustomerPhone, "customer", "", "", ip)
	if err != nil {
		return nil, err
	}

	var orderID *string
	if in.OrderNumber != nil {
		var id string
		err := s.db.QueryRow(ctx,
			`SELECT id FROM orders WHERE number = $1 AND customer_id = $2`,
			*in.OrderNumber, customer.ID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		orderID = &id
	}

	var ticketID string
	err = s.db.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		customer.ID, orderID, in.Subject, actorID).Scan(&ticketID)
	if err != nil {
		return nil, err
	}
	if in.Body != "" {
		if _, err := s.db.Exec(ctx, `
			INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, $3)`,
			ticketID, actorID, in.Body); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, ticketID)
}

// TicketFilter **مُرشِّحُ قائمة الشكاوى.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: رئيسيّةُ المدير تعدّ الشكاوى المفتوحةَ
// والمتأخّرةَ، **وكلُّ بطاقةٍ تفتح صفحةً فيها الرقمُ نفسُه.**)
type TicketFilter struct {
	// Status حالٌ بعينها · أو `unresolved` لكلّ ما لم يُحلّ · وفارغٌ للكلّ.
	Status string
	// LateHours **متأخّرةٌ**: لم يُحلّ ولم يردّ عليها أحدٌ من المكتب
	// (حالُها ما زال `open`) **بعد هذا العدد من الساعات** — وصفرٌ يعني بلا شرط.
	LateHours int
}

// ticketWhere **شرطُ القائمة والعدّ نصّاً واحداً** — `$1` الحال و`$2` الساعات.
const ticketWhere = ` WHERE ($1 = '' OR t.status = $1
		OR ($1 = 'unresolved' AND t.status <> 'resolved'))
	AND ($2::int = 0 OR (t.status = 'open'
		AND t.created_at < now() - make_interval(hours => $2::int)))`

// Count **كم شكوى تحت هذا المُرشِّح** — بالشرط نفسِه الذي تعرضه القائمة.
func (s *Service) Count(ctx context.Context, f TicketFilter) (int, error) {
	var n int
	// **وبوصلة القائمة نفسِها** — فلا يُعدّ ما لا يُعرض.
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM tickets t
		JOIN users cu ON cu.id = t.customer_id`+ticketWhere,
		f.Status, f.LateHours).Scan(&n)
	return n, err
}

func (s *Service) List(ctx context.Context, status string, page, perPage int) (*TicketPage, error) {
	return s.ListFiltered(ctx, TicketFilter{Status: status}, page, perPage)
}

// ListFiltered القائمةُ بمُرشِّحها — انظر TicketFilter.
func (s *Service) ListFiltered(ctx context.Context, f TicketFilter, page, perPage int) (*TicketPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	total, err := s.Count(ctx, f)
	if err != nil {
		return nil, err
	}
	hours := s.LateHours(ctx)
	late, err := s.Count(ctx, TicketFilter{LateHours: hours})
	if err != nil {
		return nil, err
	}
	// **والمتأخّرةُ أوّلاً** — أقدمُها في الرأس، ثمّ غيرُ المحلولة، ثمّ المحلولة.
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤: مهلةُ الردّ ساعتان، وبعدها تُعلَّم بالأحمر.)
	rows, err := s.db.Query(ctx, ticketSelect+ticketWhere+`
		ORDER BY (t.status = 'resolved'),
		         NOT (t.status = 'open' AND t.created_at < now() - make_interval(hours => $5::int)),
		         CASE WHEN t.status = 'open' AND t.created_at < now() - make_interval(hours => $5::int)
		              THEN t.created_at END ASC,
		         t.created_at DESC
		LIMIT $3 OFFSET $4`,
		f.Status, f.LateHours, perPage, (page-1)*perPage, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	tickets := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		markLate(t, hours, now)
		tickets = append(tickets, *t)
	}
	return &TicketPage{Tickets: tickets, Total: total, Late: late, LateHours: hours,
		Page: page, PerPage: perPage}, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	t, err := scanTicket(s.db.QueryRow(ctx, ticketSelect+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	markLate(t, s.LateHours(ctx), time.Now())
	rows, err := s.db.Query(ctx, `
		SELECT id, author_id, body, created_at FROM ticket_replies
		WHERE ticket_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	t.Replies = []Reply{}
	for rows.Next() {
		var r Reply
		if err := rows.Scan(&r.ID, &r.AuthorID, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		t.Replies = append(t.Replies, r)
	}
	return t, rows.Err()
}

func (s *Service) Reply(ctx context.Context, actorID, ticketID, body string) (*Ticket, error) {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM tickets WHERE id = $1`, ticketID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "resolved" {
		return nil, ErrTicketClosed
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, $3)`,
		ticketID, actorID, body); err != nil {
		return nil, err
	}
	// **وردُّ صاحب الشكوى لا يُخرجها من «متأخّرة»** — المهلةُ مهلةُ ردّ
	// المكتب (قرارُ المالك ٢٠٢٦-١٠-٠٤). كان كلُّ ردٍّ يقلبها `in_progress`،
	// **فزبونٌ يكتب «ما زلتُ أنتظر» يُخفي شكواه عن عدّاد التأخّر.**
	if _, err := s.db.Exec(ctx, `
		UPDATE tickets t SET status = 'in_progress', updated_at = now()
		WHERE t.id = $1 AND t.status = 'open' AND $2::uuid <> `+ComplainantSQL,
		ticketID, actorID); err != nil {
		return nil, err
	}
	return s.Get(ctx, ticketID)
}

// Resolve يحلّ التذكرة — **والتعويضُ (إن وُجد) اقتراحٌ للماليّة لا دفع.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الدعم يحوّل التعويض للماليّة، إذا وافقت تدفع،
// وإذا شافت السبب مو مستاهل ترفض».) **ولصاحب الشكوى** لا للزبون دائماً.
func (s *Service) Resolve(ctx context.Context, actorID, ticketID, resolution string, compensation int64, ip string) (*Ticket, error) {
	/* ══════════════════════════════════════════════════════════════════
	   **الحلُّ خطوةٌ واحدةٌ — لا أربع**
	   ══════════════════════════════════════════════════════════════════

	   (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧، وأُصلح بقرار المالك: «نبدأ إذاً».)

	   كان أربعَ خطواتٍ على البِركة مباشرةً: قراءةُ الحالة، ثمّ فحصُها، ثمّ
	   كتابةُ «محلولة» بالتعويض، ثمّ قيدُ المال في معاملةٍ منفصلة.

	   **والترتيبُ مقلوب**: الحالةُ قبل المال. فإن سقط القيدُ **بقيت التذكرةُ
	   تقول «عُوِّض خمسةَ آلاف» ولا خمسةَ آلافٍ في محفظته** — خسارةٌ صامتةٌ
	   للزبون لا يكشفها سجلّ.

	   **والمتزامنان يُعوّضان مرّتين.** قِيس بعشرين جولةً × ثمانية نداءات:
	   **دُفع التعويضُ ثماني مرّاتٍ في جولةٍ واحدة** — والرصيدُ ثمانيةُ
	   أضعافه. خمسُ تشغيلاتٍ من خمس.

	   **والقفلُ هو الحلّ**: `FOR UPDATE` يجعل الثانيةَ تنتظر، **فتقرأ
	   الحالةَ بعد أن كُتبت** فتراها `resolved` وترفض. ومعاملةٌ واحدةٌ تلفّ
	   الثلاثة، **فإن سقط شيءٌ رجع كلُّه.**
	   ══════════════════════════════════════════════════════════════════ */
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status, complainantID string
	var number int64
	err = tx.QueryRow(ctx,
		`SELECT t.status, `+ComplainantSQL+`::text, t.number FROM tickets t WHERE t.id = $1 FOR UPDATE`, ticketID).
		Scan(&status, &complainantID, &number)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "resolved" {
		return nil, ErrTicketClosed
	}
	if status == StatusAwaitingFinance {
		return nil, ErrTicketAwaitingFinance
	}

	if compensation < 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	// **الاقتراحُ قبل الحالة وفي معاملتها** — فلو سقط لم تُغلق الشكوى بلا طلبها.
	// **ونداءٌ واحدٌ لبابٍ واحد** (`CompensationProposer`) — يُبدَّل حين يُنشر
	// بابُ التعويضات الموحّد ولا يُمسّ هنا شيء.
	var requestID *string
	if compensation > 0 {
		if max := s.compensationCap(ctx); compensation > max {
			return nil, ErrCompensationOverCap
		}
		note := fmt.Sprintf("تعويض شكوى #%d", number)
		if r := strings.TrimSpace(resolution); r != "" {
			note += " — " + r
		}
		id, err := s.proposer().ProposeCompensation(ctx, tx, CompensationProposal{
			TicketID: ticketID, BeneficiaryID: complainantID, Amount: compensation,
			Note: note, ProposedBy: actorID,
		})
		if err != nil {
			return nil, err
		}
		requestID = &id
	}

	// **وعمودُ `compensation` يبقى صفراً** — ما دُفع يُقرأ من الطلب بعد الموافقة
	// (`PaidCompensationSQL`)، فلا تقول التذكرةُ «عُوِّض» قبل أن يُدفع.
	//
	// **وبتعويضٍ لا تُغلق** (قرارُ المالك ٢٠٢٦-١٠-٠٤): تبقى «بانتظار المالية» ظاهرةً،
	// **والماليّةُ تحلّها بالموافقة أو تعيدها إلى الدعم بالرفض.** وبلا تعويضٍ تُحلّ الآن.
	if requestID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE tickets SET status = $4, resolution = $2,
				compensation_request_id = $3, resolved_at = NULL, updated_at = now()
			WHERE id = $1`, ticketID, resolution, requestID, StatusAwaitingFinance); err != nil {
			return nil, err
		}
	} else if _, err := tx.Exec(ctx, `
		UPDATE tickets SET status = 'resolved', resolution = $2,
			resolved_at = now(), updated_at = now()
		WHERE id = $1`, ticketID, resolution); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, ticketID)
}

// BackToSupportTx **رفضت الماليّةُ التعويضَ — فتعود الشكوى إلى الدعم** (قرارُ المالك
// ٢٠٢٦-١٠-٠٤): «قيد المعالجة» بلا إغلاق، ويبقى ربطُ الطلب المرفوض ظاهراً حتّى يُقترح غيرُه.
// **وتُنادى في معاملة الرفض نفسِها.**
func BackToSupportTx(ctx context.Context, q dbtx.Querier, ticketID string) error {
	_, err := q.Exec(ctx, `
		UPDATE tickets SET status = 'in_progress', resolved_at = NULL, updated_at = now()
		 WHERE id = $1 AND status = $2`, ticketID, StatusAwaitingFinance)
	return err
}
