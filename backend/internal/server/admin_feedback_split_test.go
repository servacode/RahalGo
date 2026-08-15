package server

// **شكاواه وشكاوى عليه لا تختلطان.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥ بعد فحصٍ طلبه.)
//
// # العطبُ الذي كان
//
// **الشرطُ كان `t.customer_id` وحدَه** — وهو يقول «تذكرةُ أيّ طلبٍ هذه»
// **لا «من كتبها»**. فبلاغُ السائق على الزبون كان يقع في القائمة نفسِها
// **جنبَ شكاوى الزبون** — ولا يفرّقهما إلّا نصُّ العنوان المحفوظ.
//
// **والحكمان متناقضان**: «هذا الزبونُ عليه ثلاثُ شكاوى» و«هذا الزبونُ
// اشتكى ثلاثاً». **ومن يقرأ هذه الشاشةَ هو من يقرّر الإنذارَ أو الحظر**
// — فيُنذَر من اشتكى، ويسلم من شُكيَ عليه.
//
// **والمحرّكُ كان يحفظ الفرقَ في ثلاثة أعمدة** — `opened_by_customer`
// و`created_by` و`against_user_id` — **ولا يقرأ منها واحداً.**
//
// # وبلاغُ السائق يظهر في ملفّه
//
// **وكان يسكن ملفَّ زبونٍ آخر** — فمن فتح ملفَّ سائقٍ رفع عشرةَ بلاغاتٍ
// **قرأ «لا شكاوى».**

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

type fbTickets struct {
	Data struct {
		Tickets []struct {
			Number int64  `json:"number"`
			Reason string `json:"reason"`
			ByName string `json:"by_name"`
		} `json:"tickets"`
		Against []struct {
			Number int64  `json:"number"`
			Reason string `json:"reason"`
		} `json:"tickets_against"`
		TicketsCount int `json:"tickets_count"`
		AgainstCount int `json:"tickets_against_count"`
	} `json:"data"`
}

func feedbackOf(t *testing.T, f *driverFixture, userID string) fbTickets {
	t.Helper()
	w := asCustomer(f.srv.handleAdminUserFeedback, http.MethodGet, userID, userID)
	if w.Code != 200 {
		t.Fatalf("ردَّ %d", w.Code)
	}
	var out fbTickets
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}
	return out
}

// TestFeedback_SplitsHisComplaintsFromComplaintsAgainstHim
// **شكواه في قائمةٍ والبلاغُ عليه في أخرى — وبلاغُه في ملفّه.**
func TestFeedback_SplitsHisComplaintsFromComplaintsAgainstHim(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	customer, _, orderID := twoCustomers(t, f)
	driver := f.drivers[0]

	// **شكوى فتحها الزبونُ على السائق** — كما يكتبها المحرّك.
	var mine int64
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, order_id, subject, reason, created_by,
		                     opened_by_customer, against_user_id)
		VALUES ($1, $2, 'شكوى على طلب', 'driver_conduct', $1, true, $3)
		RETURNING number`, customer, orderID, driver).Scan(&mine); err != nil {
		t.Fatalf("تعذّرت شكوى الزبون: %v", err)
	}
	// **وبلاغٌ رفعه السائقُ على الزبون** — `customer_id` هو الزبونُ أيضاً،
	// **وهذا بعينه ما كان يخلطهما.**
	var against int64
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO tickets (customer_id, subject, reason, created_by,
		                     opened_by_customer, against_user_id)
		VALUES ($1, 'بلاغُ سائق', 'customer_conduct', $2, false, $1)
		RETURNING number`, customer, driver).Scan(&against); err != nil {
		t.Fatalf("تعذّر بلاغُ السائق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM tickets WHERE number = ANY($1)`, []int64{mine, against})
	})

	// ── ملفُّ الزبون ──────────────────────────────────────────────────
	cust := feedbackOf(t, f, customer)
	if cust.Data.TicketsCount != 1 || len(cust.Data.Tickets) != 1 {
		t.Fatalf("«شكاوى على طلباته» %d — **والبلاغُ عليه ليس شكواه**",
			cust.Data.TicketsCount)
	}
	if cust.Data.Tickets[0].Number != mine {
		t.Fatalf("في قائمته التذكرةُ %d لا %d", cust.Data.Tickets[0].Number, mine)
	}
	if cust.Data.AgainstCount != 1 || len(cust.Data.Against) != 1 ||
		cust.Data.Against[0].Number != against {
		t.Fatalf("«شكاوى عليه» %d — **وبلاغُ السائق عليه لا يظهر**",
			cust.Data.AgainstCount)
	}
	// **والسببُ يصل** — والشاشةُ تعرضه شارةً، **والعنوانُ وحدَه لا يقوله.**
	if cust.Data.Against[0].Reason != "customer_conduct" {
		t.Fatalf("السببُ %q لا customer_conduct — **فيُقرأ «بلاغُ سائق» بلا سبب**",
			cust.Data.Against[0].Reason)
	}
	// **ومن فتحها يُقال** — **وبلاغٌ بلا فاتحٍ يُقرأ شكوى صاحب الملفّ.**
	if cust.Data.Tickets[0].ByName == "" {
		t.Fatal("لا اسمَ لمن فتح الشكوى — **والقائمةُ تخلط من كتب**")
	}

	// ── وملفُّ السائق ─────────────────────────────────────────────────
	//
	// **وبلاغُه كان يسكن ملفَّ الزبون وحدَه** — فمن فتح ملفَّه قرأ «لا شكاوى».
	drv := feedbackOf(t, f, driver)
	found := false
	for _, x := range drv.Data.Tickets {
		if x.Number == against {
			found = true
		}
	}
	if !found {
		t.Fatal("بلاغُ السائق لا يظهر في ملفّه — **فمن رفع عشرةَ بلاغاتٍ قُرئ «لا شكاوى»**")
	}
	// **ولا يُقرأ عليه ما رفعه هو.**
	for _, x := range drv.Data.Against {
		if x.Number == against {
			t.Fatal("بلاغُه عُدَّ عليه — **ومن رفع بلاغاً فوجده شكوى عليه لا يرفع ثانياً**")
		}
	}
	// **والشكوى التي على السائق تُعدّ عليه.**
	if drv.Data.AgainstCount != 1 || drv.Data.Against[0].Number != mine {
		t.Fatalf("«شكاوى عليه» عند السائق %d — **وشكوى الزبون عليه لا تظهر**",
			drv.Data.AgainstCount)
	}
}
