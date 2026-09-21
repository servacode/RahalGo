package qa

// **الزبونُ يرى ردودَ شكواه ويردّ عليها** — `PRQ-2` · `CUST-SUP-013`/`014`.
//
// **كانت الردودُ تحت `/admin/tickets` وحدَها** — **فالزبونُ يفتح شكوى ولا
// يرى جوابَ المنصّة ولا يردّ.** فأُضيف `GET /my/tickets/{id}` (بردوده)
// و`POST /my/tickets/{id}/replies` — **معزولان بالملكيّة في الخادم،
// ومحكومان بحال التذكرة** (المحلولةُ لا يُردّ عليها).

import (
	"testing"
)

func TestPRQ2_CustomerTicketRepliesAndIsolation(t *testing.T) {
	h := New(t)
	victim := h.Customer()
	attacker := h.Customer()
	admin := h.NewUser("admin")

	var tid string
	if err := h.Pool.QueryRow(t.Context(),
		`INSERT INTO tickets (customer_id, subject, status) VALUES ($1, 'شكوى اختبار', 'open')
		 RETURNING id::text`, victim.ID).Scan(&tid); err != nil {
		t.Fatalf("PRQ-2 تعذّر إنشاءُ التذكرة: %v", err)
	}
	// **ردُّ المنصّة مبذورٌ ليرى الزبونُ جواباً** (SUP-013).
	if _, err := h.Pool.Exec(t.Context(),
		`INSERT INTO ticket_replies (ticket_id, author_id, body) VALUES ($1, $2, 'ردُّ المنصّة عليك')`,
		tid, admin.ID); err != nil {
		t.Fatalf("PRQ-2 تعذّر بذرُ الردّ: %v", err)
	}

	// ── SUP-013 · صاحبُها يقرؤها بردودها ─────────────────────────────
	got := h.GET("/api/v1/my/tickets/"+tid, victim.Token)
	if got.Code >= 400 {
		t.Fatalf("SUP-013 صاحبُها لا يقرؤها: %s", got)
	}
	if rs, _ := got.JSON()["replies"].([]any); len(rs) != 1 {
		t.Fatalf("SUP-013 عددُ الردود %d — يُنتظر 1", len(rs))
	}

	// ── العزل · غريبٌ لا يقرؤها (404 لا 403) ─────────────────────────
	if o := h.GET("/api/v1/my/tickets/"+tid, attacker.Token); o.Code != 404 {
		t.Errorf("SUP-013 **تذكرةُ غيره قُرئت** — رمز=%d (يُنتظر 404)", o.Code)
	}

	// ── SUP-014 · صاحبُها يردّ، والردُّ يظهر مرّةً ────────────────────
	rep := h.POST("/api/v1/my/tickets/"+tid+"/replies", victim.Token, map[string]any{"body": "شكراً لكم"})
	if rep.Code >= 400 {
		t.Fatalf("SUP-014 صاحبُها لا يردّ: %s", rep)
	}
	after := h.GET("/api/v1/my/tickets/"+tid, victim.Token)
	if rs, _ := after.JSON()["replies"].([]any); len(rs) != 2 {
		t.Fatalf("SUP-014 بعد الردّ عددُ الردود %d — يُنتظر 2 (ظهر مرّةً)", len(rs))
	}

	// ── العزل · غريبٌ لا يردّ (404) ──────────────────────────────────
	if x := h.POST("/api/v1/my/tickets/"+tid+"/replies", attacker.Token,
		map[string]any{"body": "تسلّل"}); x.Code != 404 {
		t.Errorf("SUP-014 **ردَّ غريبٌ على تذكرةِ غيره** — رمز=%d (يُنتظر 404)", x.Code)
	}

	// ── الحالُ · المحلولةُ لا يُردّ عليها ────────────────────────────
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE tickets SET status = 'resolved' WHERE id = $1::uuid`, tid); err != nil {
		t.Fatalf("PRQ-2: %v", err)
	}
	closed := h.POST("/api/v1/my/tickets/"+tid+"/replies", victim.Token,
		map[string]any{"body": "بعد الإغلاق"})
	if closed.Code != 409 {
		t.Errorf("SUP-014 **رُدَّ على تذكرةٍ محلولة** — رمز=%d (يُنتظر 409 ticket_resolved)", closed.Code)
	}
	if code, _ := closed.JSON()["error"].(map[string]any)["code"].(string); code != "ticket_resolved" {
		t.Errorf("SUP-014 رمزُ الإغلاق %q — يُنتظر ticket_resolved", code)
	}
}
