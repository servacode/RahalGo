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
	rs, _ := got.JSON()["replies"].([]any)
	if len(rs) != 1 {
		t.Fatalf("SUP-013 عددُ الردود %d — يُنتظر 1", len(rs))
	}
	// **ردُّ المنصّة يُوسَم «ليس لي»** — والتمييزُ من الخادم لا من العميل.
	r0, _ := rs[0].(map[string]any)
	if mine, _ := r0["mine"].(bool); mine {
		t.Errorf("SUP-013 **ردُّ المنصّة وُسم mine=true** — يُخلَط بردّ الزبون")
	}
	// **ولا يُكشَف معرّفُ من ردّ** — لا `author_id` في ردّ الزبون.
	if _, leaked := r0["author_id"]; leaked {
		t.Errorf("SUP-013 **تسريب**: author_id ظاهرٌ للزبون")
	}
	// **والتذكرةُ تحمل موضوعَها وحالَها ووقتَ فتحها** — لا رقماً بلا حال.
	if s, _ := got.JSON()["subject"].(string); s == "" {
		t.Errorf("SUP-013 الموضوعُ فارغٌ في التفصيل")
	}
	if c, _ := got.JSON()["created_at"].(string); c == "" {
		t.Errorf("SUP-013 وقتُ الفتح غائبٌ في التفصيل")
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
	ars, _ := after.JSON()["replies"].([]any)
	if len(ars) != 2 {
		t.Fatalf("SUP-014 بعد الردّ عددُ الردود %d — يُنتظر 2 (ظهر مرّةً)", len(ars))
	}
	// **الترتيبُ زمنيٌّ ثابت**: ردُّ المنصّة (المبذور أولاً) ثمّ ردُّ الزبون.
	a0, _ := ars[0].(map[string]any)
	a1, _ := ars[1].(map[string]any)
	if mine, _ := a0["mine"].(bool); mine {
		t.Errorf("SUP-014 الترتيب/التمييز: الأوّلُ يجب أن يكون ردَّ المنصّة (mine=false)")
	}
	if mine, _ := a1["mine"].(bool); !mine {
		t.Errorf("SUP-014 التمييز: ردُّ الزبون يجب أن يكون mine=true")
	}
	// **ولا تكرارَ بإعادة القراءة** — إعادةُ الجلب تردّ اثنين لا ثلاثة.
	reload := h.GET("/api/v1/my/tickets/"+tid, victim.Token)
	if rl, _ := reload.JSON()["replies"].([]any); len(rl) != 2 {
		t.Errorf("SUP-014 **تكرار**: بعد إعادة القراءة صار %d — يُنتظر 2", len(rl))
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
