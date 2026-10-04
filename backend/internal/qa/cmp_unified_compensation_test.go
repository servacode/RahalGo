package qa

// **تعويضُ الشكوى يدخل طابورَ التعويضات الموحّد** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الدمج).
//
// حلُّ التذكرة بتعويضٍ يكتب طلباً معلَّقاً واحداً في `driver_compensation_requests`
// (نوعُ «شكوى»، مستفيدُه صاحبُ الشكوى، ومربوطٌ بالتذكرة) — **لا طلبَ محفظة،
// ولا مالَ يتحرّك قبل موافقة الماليّة**، ويظهر في صفحة الموافقات الموحّدة.

import "testing"

// openTicketFor تذكرةٌ يفتحها المكتبُ لزبون — وتردّ معرّفَها.
func openTicketFor(t *testing.T, h *Harness, staff, cust *User) string {
	t.Helper()
	res := h.POST("/api/v1/admin/tickets", staff.Token, map[string]any{
		"customer_phone": cust.Phone, "subject": "شكوى فحص", "body": "تفاصيل",
	})
	if res.Code >= 400 {
		t.Fatalf("فتحُ التذكرة: %s", res)
	}
	id, _ := res.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("تذكرةٌ بلا معرّف: %s", res)
	}
	return id
}

func TestCMP_ResolvedComplaintProposesIntoUnifiedQueue(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	staff := h.NewUser("admin")
	cust := h.Customer()
	ticket := openTicketFor(t, h, staff, cust)

	tBefore := treasuryBalance(t, h, tid)
	res := h.POST("/api/v1/admin/tickets/"+ticket+"/resolve", staff.Token,
		map[string]any{"resolution": "تأخّر الطلب", "compensation": 3000})
	if res.Code >= 400 {
		t.Fatalf("الحلّ: %s", res)
	}

	var n, pending int
	var reqID string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), count(*) FILTER (WHERE status = 'pending' AND kind = 'complaint'
		         AND suggested_amount = 3000 AND driver_id = $2::uuid AND amount IS NULL),
		       COALESCE(max(id::text), '')
		  FROM driver_compensation_requests WHERE ticket_id = $1::uuid`,
		ticket, cust.ID).Scan(&n, &pending, &reqID); err != nil {
		t.Fatal(err)
	}
	if n != 1 || pending != 1 {
		t.Fatalf("طابورُ التعويضات: %d صفّاً و%d معلَّقاً لصاحب الشكوى — المطلوبُ واحد", n, pending)
	}
	var linked string
	var paid int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(compensation_request_id::text, ''), compensation FROM tickets WHERE id = $1::uuid`,
		ticket).Scan(&linked, &paid); err != nil {
		t.Fatal(err)
	}
	if linked != reqID {
		t.Fatalf("التذكرةُ مربوطةٌ بـ%q لا بطلب الطابور %q", linked, reqID)
	}
	if paid != 0 {
		t.Fatalf("التذكرةُ تقول «عُوِّض %d» قبل موافقة الماليّة", paid)
	}
	var wr, moved int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_requests
		WHERE user_id = $1::uuid AND kind = 'compensation'`, cust.ID).Scan(&wr)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions WHERE ref = $1`, ticket).Scan(&moved)
	if wr != 0 {
		t.Fatalf("كُتب طلبُ محفظةٍ (%d) — والطابورُ واحد", wr)
	}
	if moved != 0 || treasuryBalance(t, h, tid) != tBefore {
		t.Fatalf("تحرّك مالٌ قبل الموافقة: %d قيداً", moved)
	}

	// **ويظهر في صفحة الموافقات الموحّدة.**
	ap := h.GET("/api/v1/admin/approvals", h.NewUser("admin").Token)
	if ap.Code >= 400 {
		t.Fatalf("الموافقات: %s", ap)
	}
	items, _ := ap.JSON()["items"].([]any)
	found := false
	for _, it := range items {
		if m, _ := it.(map[string]any); m != nil && m["id"] == reqID {
			found = true
		}
	}
	if !found {
		t.Fatal("طلبُ تعويض الشكوى غائبٌ عن صفحة الموافقات الموحّدة")
	}
}

// ticketState حالُ التذكرة وما يتبعها.
func ticketState(t *testing.T, h *Harness, ticket string) (status, reqID string, paid int64, resolved bool) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, COALESCE(compensation_request_id::text, ''), compensation, resolved_at IS NOT NULL
		  FROM tickets WHERE id = $1::uuid`, ticket).Scan(&status, &reqID, &paid, &resolved); err != nil {
		t.Fatal(err)
	}
	return
}

// TestCMP_PendingCompensationWaitsForFinance **قرارا المالك ٢٠٢٦-١٠-٠٤ بعد الدمج:**
//
//  1. شكوى بتعويضٍ معلَّقٍ تبقى ظاهرةً «بانتظار المالية» (`awaiting_finance`)
//     ولا تُغلق إلّا بقرار الماليّة: الموافقةُ تحلّها بالمبلغ.
//  2. رفضُ الماليّة يعيدها إلى الدعم (`in_progress`) ويُخبر صاحبَ الاقتراح،
//     **والدعمُ يقترح ثانيةً على التذكرة نفسِها.**
func TestCMP_PendingCompensationWaitsForFinance(t *testing.T) {
	h := New(t)
	treasury(t, h)
	staff := h.NewUser("admin")
	fin := h.NewUser("admin")
	cust := h.Customer()
	ticket := openTicketFor(t, h, staff, cust)

	res := h.POST("/api/v1/admin/tickets/"+ticket+"/resolve", staff.Token,
		map[string]any{"resolution": "نعتذر", "compensation": 3000})
	if res.Code >= 400 {
		t.Fatalf("الحلّ: %s", res)
	}
	st, first, paid, resolved := ticketState(t, h, ticket)
	if st != "awaiting_finance" || resolved || paid != 0 || first == "" {
		t.Fatalf("بعد الاقتراح: حال %q · محلولة %v · مدفوع %d — المطلوبُ «بانتظار المالية» بلا إغلاق", st, resolved, paid)
	}
	// **ولا اقتراحَ ثانٍ والأوّلُ معلَّق.**
	if again := h.POST("/api/v1/admin/tickets/"+ticket+"/resolve", staff.Token,
		map[string]any{"resolution": "ثانية", "compensation": 1000}); again.Code != 409 {
		t.Fatalf("اقتراحٌ ثانٍ والأوّلُ معلَّق ردّ %s", again)
	}

	// **الرفضُ يعيدها إلى الدعم ويُخبر المقترِح.**
	if rj := h.POST("/api/v1/admin/compensations/"+first+"/reject", fin.Token,
		map[string]any{"note": "لا يستحقّ"}); rj.Code >= 400 {
		t.Fatalf("الرفض: %s", rj)
	}
	if st, _, _, resolved = ticketState(t, h, ticket); st != "in_progress" || resolved {
		t.Fatalf("بعد الرفض: حال %q · محلولة %v — المطلوبُ عودتُها إلى الدعم", st, resolved)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2`, staff.ID, ticket).Scan(&told)
	if told == 0 {
		t.Fatal("رُفض التعويضُ ولم يُخبَر الدعم")
	}

	// **والدعمُ يقترح ثانيةً على التذكرة نفسِها.**
	if res := h.POST("/api/v1/admin/tickets/"+ticket+"/resolve", staff.Token,
		map[string]any{"resolution": "نعتذر ثانية", "compensation": 2000}); res.Code >= 400 {
		t.Fatalf("الاقتراحُ الثاني: %s", res)
	}
	st, second, _, _ := ticketState(t, h, ticket)
	if st != "awaiting_finance" || second == "" || second == first {
		t.Fatalf("الاقتراحُ الثاني: حال %q · طلب %q (الأوّل %q)", st, second, first)
	}

	// **والموافقةُ تحلّها بالمبلغ.**
	if ok := h.POST("/api/v1/admin/compensations/"+second+"/approve", fin.Token,
		map[string]any{"amount": 2000, "note": "موافقة"}); ok.Code >= 400 {
		t.Fatalf("الموافقة: %s", ok)
	}
	st, _, paid, resolved = ticketState(t, h, ticket)
	if st != "resolved" || !resolved || paid != 2000 {
		t.Fatalf("بعد الموافقة: حال %q · محلولة %v · مدفوع %d", st, resolved, paid)
	}
}
