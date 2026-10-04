package support_test

// **قراراتُ المالك ٢٠٢٦-١٠-٠٤ — الشكاوى والتقييمات.**
//
//   - التعويضُ يقترحه الدعمُ وتقرّره الماليّة — لا قيدَ لحظةَ الحلّ.
//   - والتعويضُ لصاحب الشكوى — السائقُ إن كان هو من اشتكى، لا الزبون.
//   - ومهلةُ الردّ ساعتان — بعدها «متأخّرة» في أوّل القائمة.
//   - وردُّ صاحب الشكوى لا يُخرجها من «متأخّرة».

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func cmpTicket(t *testing.T, ctx context.Context, sql string, args ...any) string {
	t.Helper()
	pool := testdb.Pool(t)
	var id string
	if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء تذكرة: %v", err)
	}
	return id
}

// TestCMP_ResolveProposesCompensationNotPays الحلُّ يكتب طلباً معلَّقاً ولا يقيّد مالاً.
func TestCMP_ResolveProposesCompensationNotPays(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := support.NewService(pool, nil, wallet.NewService(pool))
	agent := testdb.NewUser(t, pool, "admin")
	customer := testdb.NewUser(t, pool, "customer")

	id := cmpTicket(t, ctx, `
		INSERT INTO tickets (customer_id, subject, reason, created_by, opened_by_customer)
		VALUES ($1, 'late', 'late', $1, true) RETURNING id`, customer)

	tk, err := svc.Resolve(ctx, agent, id, "تأخّر الطلب", 7_000, "127.0.0.1")
	if err != nil {
		t.Fatalf("الحلّ: %v", err)
	}
	var paid int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM wallet_transactions
		WHERE user_id = $1 AND kind = 'compensation'`, customer).Scan(&paid)
	if paid != 0 {
		t.Fatalf("**قُيّد التعويضُ لحظةَ الحلّ (%d قيد)** — والقرار: الدعمُ يقترح والماليّةُ توافق", paid)
	}
	var reqUser, reqStatus, reqKind string
	var reqAmount int64
	if err := pool.QueryRow(ctx, `
		SELECT wr.user_id::text, wr.status, wr.kind, wr.amount
		FROM tickets t JOIN wallet_requests wr ON wr.id = t.compensation_request_id
		WHERE t.id = $1`, id).Scan(&reqUser, &reqStatus, &reqKind, &reqAmount); err != nil {
		t.Fatalf("لا طلبَ تعويضٍ مربوطاً بالتذكرة: %v", err)
	}
	if reqUser != customer || reqStatus != "pending" || reqKind != "compensation" || reqAmount != 7_000 {
		t.Fatalf("الطلب: user=%s status=%s kind=%s amount=%d", reqUser, reqStatus, reqKind, reqAmount)
	}
	if tk.Compensation != 0 || tk.CompensationProposed != 7_000 || tk.CompensationStatus != "pending" {
		t.Fatalf("التذكرةُ تقول دُفع %d واقتُرح %d بحال %q — والمنتظَر ٠ و٧٠٠٠ و pending",
			tk.Compensation, tk.CompensationProposed, tk.CompensationStatus)
	}

	// **وبعد الموافقة يُقرأ ما دُفع** — من الطلب نفسِه.
	if _, err := pool.Exec(ctx, `UPDATE wallet_requests SET status = 'approved'
		WHERE id = (SELECT compensation_request_id FROM tickets WHERE id = $1)`, id); err != nil {
		t.Fatal(err)
	}
	tk, _ = svc.Get(ctx, id)
	if tk.Compensation != 7_000 || tk.CompensationStatus != "approved" {
		t.Fatalf("بعد الموافقة: دُفع %d بحال %q", tk.Compensation, tk.CompensationStatus)
	}
}

// TestCMP_CompensationGoesToComplainant بلاغُ السائق يُعوَّض السائقُ عنه لا زبونُ الطلب.
func TestCMP_CompensationGoesToComplainant(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := support.NewService(pool, nil, wallet.NewService(pool))
	agent := testdb.NewUser(t, pool, "admin")
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")

	id := cmpTicket(t, ctx, `
		INSERT INTO tickets (customer_id, subject, reason, created_by, opened_by_customer)
		VALUES ($1, 'x', 'customer_absent', $2, false) RETURNING id`, customer, driver)

	tk, err := svc.Resolve(ctx, agent, id, "عولجت", 3_000, "127.0.0.1")
	if err != nil {
		t.Fatalf("الحلّ: %v", err)
	}
	if tk.ComplainantID != driver || tk.ComplainantKind != "driver" {
		t.Fatalf("صاحبُ الشكوى %s (%s) — والمنتظَر السائق", tk.ComplainantID, tk.ComplainantKind)
	}
	var beneficiary string
	if err := pool.QueryRow(ctx, `
		SELECT wr.user_id::text FROM tickets t
		JOIN wallet_requests wr ON wr.id = t.compensation_request_id WHERE t.id = $1`, id).
		Scan(&beneficiary); err != nil {
		t.Fatalf("لا طلبَ تعويض: %v", err)
	}
	if beneficiary != driver {
		t.Fatalf("**التعويضُ لزبون الطلب لا لصاحب البلاغ** — المستفيد %s والسائق %s", beneficiary, driver)
	}
}

// TestCMP_LateTicketsFlaggedAndFirst المتأخّرةُ تُعلَّم وتُعدّ وتأتي أوّلاً.
func TestCMP_LateTicketsFlaggedAndFirst(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := support.NewService(pool, nil, wallet.NewService(pool))
	customer := testdb.NewUser(t, pool, "customer")

	late := cmpTicket(t, ctx, `
		INSERT INTO tickets (customer_id, subject, created_by, opened_by_customer, created_at)
		VALUES ($1, 'old', $1, true, now() - interval '3 hours') RETURNING id`, customer)
	fresh := cmpTicket(t, ctx, `
		INSERT INTO tickets (customer_id, subject, created_by, opened_by_customer)
		VALUES ($1, 'new', $1, true) RETURNING id`, customer)

	page, err := svc.ListFiltered(ctx, support.TicketFilter{Status: "unresolved"}, 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if page.LateHours != 2 || page.Late < 1 {
		t.Fatalf("المهلة %d والمتأخّرة %d — والمنتظَر ساعتان وواحدةٌ على الأقلّ", page.LateHours, page.Late)
	}
	posLate, posFresh := -1, -1
	for i, tk := range page.Tickets {
		switch tk.ID {
		case late:
			posLate = i
			if !tk.Late {
				t.Fatal("شكوى عمرُها ثلاثُ ساعاتٍ بلا ردٍّ لم تُعلَّم متأخّرة")
			}
		case fresh:
			posFresh = i
			if tk.Late {
				t.Fatal("شكوى جديدةٌ عُلّمت متأخّرة")
			}
		}
	}
	if posLate < 0 || posFresh < 0 || posLate > posFresh {
		t.Fatalf("المتأخّرة في %d والجديدة في %d — والمتأخّرةُ أوّلاً", posLate, posFresh)
	}
}

// TestCMP_ComplainantReplyKeepsItLate ردُّ الزبون نفسِه لا يُخرج شكواه من «متأخّرة».
func TestCMP_ComplainantReplyKeepsItLate(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := support.NewService(pool, nil, wallet.NewService(pool))
	customer := testdb.NewUser(t, pool, "customer")
	agent := testdb.NewUser(t, pool, "admin")

	id := cmpTicket(t, ctx, `
		INSERT INTO tickets (customer_id, subject, created_by, opened_by_customer, created_at)
		VALUES ($1, 'old', $1, true, now() - interval '3 hours') RETURNING id`, customer)

	tk, err := svc.Reply(ctx, customer, id, "ما زلتُ أنتظر")
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != "open" || !tk.Late {
		t.Fatalf("بعد ردّ الزبون: الحال %q ومتأخّرة=%v — والمنتظَر open ومتأخّرة", tk.Status, tk.Late)
	}
	tk, err = svc.Reply(ctx, agent, id, "نتابع")
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != "in_progress" || tk.Late {
		t.Fatalf("بعد ردّ المكتب: الحال %q ومتأخّرة=%v", tk.Status, tk.Late)
	}
}
