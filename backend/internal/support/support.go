// Package support نظام التذاكر والتعويضات (PLAN §6.7):
// شكوى تُفتح لزبون (مرتبطة بطلب اختيارياً)، خيط ردود، وحلّ بتعويض
// اختياري يُقيَّد لمحفظته فوراً — أسرع طريقة لإرضاء زبون غاضب.
package support

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var ErrTicketClosed = httpx.NewError(http.StatusConflict, "ticket_resolved", "errors.ticket_resolved")

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

	OpenedByCustomer bool       `json:"opened_by_customer"`
	Status           string     `json:"status"`
	Compensation     int64      `json:"compensation"`
	Resolution       string     `json:"resolution"`
	CreatedAt        time.Time  `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	Replies          []Reply    `json:"replies,omitempty"`
}

type TicketPage struct {
	Tickets []Ticket `json:"tickets"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	PerPage int      `json:"per_page"`
}

type Service struct {
	db       *pgxpool.Pool
	identity *identity.Service
	wallet   *wallet.Service
	// settings مهلةُ الشكوى وما يتبعها — يملك المالكُ ضبطَها من اللوحة.
	settings *settings.Store
}

func NewService(db *pgxpool.Pool, identitySvc *identity.Service, walletSvc *wallet.Service) *Service {
	return &Service{db: db, identity: identitySvc, wallet: walletSvc}
}

// SetSettings يحقن مخزن الإعدادات (يُنادى مرّة عند الإقلاع).
func (s *Service) SetSettings(st *settings.Store) { s.settings = st }

const ticketSelect = `
	SELECT t.id, t.number, t.customer_id, cu.phone, cu.full_name,
	       t.order_id, o.number, t.subject, COALESCE(t.reason,''), t.against_user_id::text,
	       t.opened_by_customer,
	       t.status, t.compensation, t.resolution,
	       t.created_at, t.resolved_at
	FROM tickets t
	JOIN users cu ON cu.id = t.customer_id
	LEFT JOIN orders o ON o.id = t.order_id`

func scanTicket(row pgx.Row) (*Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.Number, &t.CustomerID, &t.CustomerPhone, &t.CustomerName,
		&t.OrderID, &t.OrderNumber, &t.Subject, &t.Reason, &t.AgainstUserID,
		&t.OpenedByCustomer,
		&t.Status, &t.Compensation, &t.Resolution,
		&t.CreatedAt, &t.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
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

func (s *Service) List(ctx context.Context, status string, page, perPage int) (*TicketPage, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	where := ` WHERE ($1 = '' OR t.status = $1)`

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM tickets t`+where, status).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, ticketSelect+where+`
		ORDER BY (t.status = 'resolved'), t.created_at DESC LIMIT $2 OFFSET $3`,
		status, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, *t)
	}
	return &TicketPage{Tickets: tickets, Total: total, Page: page, PerPage: perPage}, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (*Ticket, error) {
	t, err := scanTicket(s.db.QueryRow(ctx, ticketSelect+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
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
	if _, err := s.db.Exec(ctx, `
		UPDATE tickets SET status = 'in_progress', updated_at = now()
		WHERE id = $1 AND status = 'open'`, ticketID); err != nil {
		return nil, err
	}
	return s.Get(ctx, ticketID)
}

// Resolve يحل التذكرة — والتعويض (إن وُجد) يُقيَّد لمحفظة الزبون فوراً.
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

	var status, customerID string
	var number int64
	err = tx.QueryRow(ctx,
		`SELECT status, customer_id, number FROM tickets WHERE id = $1 FOR UPDATE`, ticketID).
		Scan(&status, &customerID, &number)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "resolved" {
		return nil, ErrTicketClosed
	}

	// **والمالُ قبل الحالة** — فلو سقط لم يبقَ سطرٌ يقول «عُوِّض» بلا تعويض.
	if compensation > 0 {
		if _, err := s.wallet.ApplyTx(ctx, tx, customerID, compensation, "compensation",
			ticketID, fmt.Sprintf("تعويض تذكرة #%d", number), &actorID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tickets SET status = 'resolved', resolution = $2, compensation = $3,
			resolved_at = now(), updated_at = now()
		WHERE id = $1`, ticketID, resolution, compensation); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, ticketID)
}
