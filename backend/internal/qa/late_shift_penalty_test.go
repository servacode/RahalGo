package qa

// ══════════════════════════════════════════════════════════════════════
// **عقوبةُ تأخير فتح الدوام** — قرار المالك ٢٠٢٦-١١-١٤
// ══════════════════════════════════════════════════════════════════════
//
// من لم يفتح دوامه خلال دقائق السماح بعد فتح المنصّة: خصمٌ من محفظته (العدد × `delivery.fee`)
// ومن عدّاد هدفه — مرّةً باليوم. ولا عقوبةَ لمن فتح بالموعد ثمّ سكّر، ولا لمن عليه إجازة، ولا
// والمفتاحُ مطفأ أو المنصّةُ مسكّرة. والإلغاءُ يردّ المالَ ويمحو خصمَ الهدف.

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/incentives"
)

const (
	lateDay          = "2031-04-07"
	lateMonth        = "2031-04"
	lateRemindTitle  = "باقي دقيقتين، افتح دوامك"
	latePenaltyTitle = "عقوبة تأخير فتح الدوام"
)

// lateSetup **المنصّةُ ٩←٢٣ كلَّ يوم، والعقوبةُ مشغّلة**: سماحٌ ٥ دقائق، توصيلتان من المحفظة
// (والأجرةُ ١٠٠٠ ⇒ ٢٠٠٠) وتوصيلةٌ من الهدف.
func lateSetup(t *testing.T, h *Harness) {
	t.Helper()
	shiftRemindHours(t, h)
	h.Setting("drivers.late_shift_penalty_on", "true")
	h.Setting("drivers.late_shift_grace_min", "5")
	h.Setting("drivers.late_shift_wallet_deliveries", "2")
	h.Setting("drivers.late_shift_target_deliveries", "1")
	h.Setting("delivery.fee", "1000")
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM driver_leaves WHERE day = $1::date`, lateDay)
	})
}

// lateIsolate **كلُّ سائقٍ غيرِ هؤلاء بإجازةٍ لليوم** — قاعدةُ الاختبار مشتركة، وسائقو الفحوص
// السابقة لا يدخلون الجولة.
func lateIsolate(t *testing.T, h *Harness, day string, keep ...string) {
	t.Helper()
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM driver_leaves WHERE day = $1::date`, day) })
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_leaves (driver_id, day)
		SELECT r.user_id, $1::date FROM user_roles r
		 WHERE r.role_code = 'driver' AND NOT (r.user_id::text = ANY($2::text[]))
		ON CONFLICT DO NOTHING`, day, keep); err != nil {
		t.Fatal(err)
	}
}

// lateShiftRow **دوامٌ بصفّه** من `from` إلى `to` (`nil` مفتوح) — بلا مسّ `on_shift`.
func lateShiftRow(t *testing.T, h *Harness, driverID string, from time.Time, to *time.Time) {
	t.Helper()
	reason := any(nil)
	if to != nil {
		reason = "qa"
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason)
		VALUES ($1::uuid, $2, $3, $4)`, driverID, from, to, reason); err != nil {
		t.Fatal(err)
	}
}

func lateRound(t *testing.T, h *Harness, at time.Time) (int, int) {
	t.Helper()
	r, p, err := h.API.RunLateShiftRound(ctxBG(), at)
	if err != nil {
		t.Fatal(err)
	}
	return r, p
}

func lateTitles(t *testing.T, h *Harness, driverID, title string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`,
		driverID, title).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func latePenalties(t *testing.T, h *Harness, driverID string) (rows, adjust int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM driver_late_shift_penalties WHERE driver_id = $1::uuid),
		       (SELECT COALESCE(sum(deliveries), 0) FROM driver_target_adjustments WHERE driver_id = $1::uuid)`,
		driverID).Scan(&rows, &adjust); err != nil {
		t.Fatal(err)
	}
	return rows, adjust
}

func lateDebt(t *testing.T, h *Harness, driverID string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		 WHERE party_id = $1::uuid AND closed_at IS NULL`, driverID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

type lateStub struct{}

func (lateStub) GetInt(context.Context, string) int64 { return 10 }

// lateDone **عدّادُ الهدف كما تقرؤه لوحةُ الأهداف وتطبيقُ السائق** — `StandingsFor`.
func lateDone(t *testing.T, h *Harness, driverID string) int {
	t.Helper()
	svc := incentives.New(h.Pool, nil, lateStub{}, func(context.Context) string { return "" })
	rows, err := svc.StandingsFor(ctxBG(), "driver", lateMonth)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.UserID == driverID {
			return r.Done
		}
	}
	t.Fatalf("السائق %s ليس بالقائمة", driverID)
	return 0
}

// lateDelivered **طلباتٌ سلّمها بنيسان ٢٠٣١** — لعدّاد الهدف.
func lateDelivered(t *testing.T, h *Harness, driverID string, n int) {
	t.Helper()
	cust := h.Customer()
	item := h.NewItem(5000)
	for i := 0; i < n; i++ {
		made := h.POSTKey("/api/v1/orders", cust.Token, uniq("late"), orderBody(item, 1))
		oid, _ := made.JSON()["id"].(string)
		if oid == "" {
			t.Fatalf("تعذّر إنشاءُ الطلب: %d %s", made.Code, trimBody(made))
		}
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE orders SET status = 'delivered', driver_id = $2::uuid,
			       delivered_at = $3, closed_at = $3
			 WHERE id = $1::uuid`, oid, driverID, lmAt(2031, 4, 2, 12, 0)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestLateShift_PenaltyOnceAndCancel **القلبُ كلُّه**: التذكيرُ قبل الموعد بدقيقتين لمن لم يفتح ·
// العقوبةُ بعد الموعد للمتأخّر وحدَه (المبلغُ ٢ × ١٠٠٠ والهدفُ ناقصٌ واحداً) · وما زاد على
// رصيده دين · وجولةٌ ثانيةٌ لا تكرّر · ومن فتح بالموعد أو فتح ثمّ سكّر أو عليه إجازة لا يُعاقَب ·
// والإلغاءُ يردّ المالَ ويُسقط الدينَ ويُرجع الهدف، ولا يتكرّر.
func TestLateShift_PenaltyOnceAndCancel(t *testing.T) {
	h := New(t)
	lateSetup(t, h)
	onTime := h.NewUser("driver")
	closer := h.NewUser("driver")
	late := h.NewUser("driver")
	poor := h.NewUser("driver")
	leave := h.NewUser("driver")
	lateIsolate(t, h, lateDay, onTime.ID, closer.ID, late.ID, poor.ID, leave.ID)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_leaves (driver_id, day) VALUES ($1::uuid, $2::date)`, leave.ID, lateDay); err != nil {
		t.Fatal(err)
	}
	if err := h.walletApply(late.ID, 10_000, "topup", ""); err != nil {
		t.Fatal(err)
	}
	lateDelivered(t, h, late.ID, 3)
	lateShiftRow(t, h, onTime.ID, lmAt(2031, 4, 7, 8, 58), nil)
	closedAt := lmAt(2031, 4, 7, 9, 30)
	lateShiftRow(t, h, closer.ID, lmAt(2031, 4, 7, 9, 1), &closedAt)

	if r, p := lateRound(t, h, lmAt(2031, 4, 7, 8, 30)); r+p != 0 {
		t.Fatalf("قبل فتح المنصّة: تذكير %d عقوبة %d — المتوقَّع لا شيء", r, p)
	}
	if r, p := lateRound(t, h, lmAt(2031, 4, 7, 9, 2)); r+p != 0 {
		t.Fatalf("٩:٠٢: تذكير %d عقوبة %d — المتوقَّع لا شيء", r, p)
	}
	lateRound(t, h, lmAt(2031, 4, 7, 9, 3))
	lateRound(t, h, lmAt(2031, 4, 7, 9, 4))
	for _, d := range []*User{late, poor} {
		if n := lateTitles(t, h, d.ID, lateRemindTitle); n != 1 {
			t.Errorf("التذكير لمن لم يفتح: %d — المتوقَّع واحد", n)
		}
	}
	for _, d := range []*User{onTime, closer, leave} {
		if n := lateTitles(t, h, d.ID, lateRemindTitle); n != 0 {
			t.Errorf("تذكيرٌ لمن فتح أو عليه إجازة: %d — المتوقَّع لا شيء", n)
		}
	}

	if _, p := lateRound(t, h, lmAt(2031, 4, 7, 9, 6)); p != 2 {
		t.Fatalf("بعد الموعد: عوقب %d — المتوقَّع اثنان", p)
	}
	lateRound(t, h, lmAt(2031, 4, 7, 9, 10))
	lateRound(t, h, lmAt(2031, 4, 7, 15, 0))

	if b := walletBalance(t, h, late.ID); b != 8_000 {
		t.Errorf("رصيدُ المتأخّر %d — المتوقَّع ٨٠٠٠ (١٠٠٠٠ − ٢ × ١٠٠٠)", b)
	}
	if d := lateDebt(t, h, poor.ID); d != 2_000 {
		t.Errorf("دينُ من لا رصيدَ له %d — المتوقَّع ٢٠٠٠", d)
	}
	for _, d := range []*User{late, poor} {
		if rows, adj := latePenalties(t, h, d.ID); rows != 1 || adj != 1 {
			t.Errorf("المتأخّر: صفوف %d وخصمُ هدف %d — المتوقَّع ١ و١ (مرّةً باليوم)", rows, adj)
		}
		if n := lateTitles(t, h, d.ID, latePenaltyTitle); n != 1 {
			t.Errorf("إشعارُ العقوبة: %d — المتوقَّع واحد", n)
		}
	}
	for _, d := range []*User{onTime, closer, leave} {
		if rows, _ := latePenalties(t, h, d.ID); rows != 0 {
			t.Errorf("عوقب من فتح بالموعد أو فتح ثمّ سكّر أو عليه إجازة")
		}
	}
	if got := lateDone(t, h, late.ID); got != 2 {
		t.Errorf("عدّادُ الهدف %d — المتوقَّع ٢ (٣ مسلَّمة − ١ خصم)", got)
	}
	if got := lateDone(t, h, poor.ID); got != 0 {
		t.Errorf("عدّادُ من لا طلبَ له %d — المتوقَّع ٠ لا سالب", got)
	}

	// ── الإلغاء ──
	admin := h.NewUser("admin")
	var latePID, poorPID string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM driver_late_shift_penalties WHERE driver_id = $1::uuid`, late.ID).Scan(&latePID)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM driver_late_shift_penalties WHERE driver_id = $1::uuid`, poor.ID).Scan(&poorPID)
	if res := h.POST("/api/v1/admin/late-shift-penalties/"+latePID+"/cancel", admin.Token,
		map[string]any{"reason": "qa"}); res.Code != 200 {
		t.Fatalf("الإلغاء: %d %s", res.Code, trimBody(res))
	}
	if b := walletBalance(t, h, late.ID); b != 10_000 {
		t.Errorf("بعد الإلغاء رصيدُه %d — المتوقَّع ١٠٠٠٠", b)
	}
	if got := lateDone(t, h, late.ID); got != 3 {
		t.Errorf("بعد الإلغاء عدّادُ الهدف %d — المتوقَّع ٣", got)
	}
	if res := h.POST("/api/v1/admin/late-shift-penalties/"+latePID+"/cancel", admin.Token,
		map[string]any{}); res.Code != 409 {
		t.Errorf("إلغاءٌ ثانٍ: %d — المتوقَّع ٤٠٩", res.Code)
	}
	if res := h.POST("/api/v1/admin/late-shift-penalties/"+poorPID+"/cancel", admin.Token,
		map[string]any{}); res.Code != 200 {
		t.Fatalf("إلغاءُ عقوبة الدين: %d %s", res.Code, trimBody(res))
	}
	if d := lateDebt(t, h, poor.ID); d != 0 {
		t.Errorf("بعد الإلغاء دينُه %d — المتوقَّع ٠", d)
	}
	if b := walletBalance(t, h, poor.ID); b != 0 {
		t.Errorf("بعد إلغاء عقوبة الدين رصيدُه %d — المتوقَّع ٠ (لم يُقبض منه شيء)", b)
	}
	if _, adj := latePenalties(t, h, poor.ID); adj != 0 {
		t.Errorf("بعد الإلغاء خصمُ الهدف %d — المتوقَّع ٠", adj)
	}
	var voided int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM incentives
		WHERE user_id = ANY($1::uuid[]) AND kind = 'penalty' AND voided_at IS NOT NULL`,
		[]string{late.ID, poor.ID}).Scan(&voided)
	if voided != 2 {
		t.Errorf("صفوفُ الحوافز الملغاة %d — المتوقَّع ٢", voided)
	}
	// وبعد الإلغاء لا تعود العقوبةُ بجولةٍ ثالثة
	lateRound(t, h, lmAt(2031, 4, 7, 16, 0))
	if rows, adj := latePenalties(t, h, late.ID); rows != 1 || adj != 0 {
		t.Errorf("جولةٌ بعد الإلغاء: صفوف %d وخصم %d — المتوقَّع ١ و٠", rows, adj)
	}
}

// TestLateShift_NothingWhenOffOrClosed **لا شيء**: المفتاحُ مطفأ · زرُّ «مسكّرين» · زرُّ
// «مفتوحين» · جدولٌ لا يسري · يومٌ بلا فترة · مفتاحٌ شُغّل بعد موعد اليوم.
func TestLateShift_NothingWhenOffOrClosed(t *testing.T) {
	h := New(t)
	lateSetup(t, h)
	drv := h.NewUser("driver")
	lateIsolate(t, h, lateDay, drv.ID)
	lateIsolate(t, h, "2031-04-08", drv.ID)
	at := lmAt(2031, 4, 7, 9, 20)
	check := func(what string) {
		t.Helper()
		if r, p := lateRound(t, h, at); r+p != 0 {
			t.Fatalf("%s: تذكير %d عقوبة %d — المتوقَّع لا شيء", what, r, p)
		}
		if rows, _ := latePenalties(t, h, drv.ID); rows != 0 {
			t.Fatalf("%s: عوقب", what)
		}
	}
	h.Setting("drivers.late_shift_penalty_on", "false")
	check("المفتاحُ مطفأ")
	h.Setting("drivers.late_shift_penalty_on", "true")
	h.Setting("hours.platform_switch", `"closed"`)
	check("زرّ «مسكّرين»")
	h.Setting("hours.platform_switch", `"open"`)
	check("زرّ «مفتوحين»")
	h.Setting("hours.platform_switch", `"schedule"`)
	h.Setting("hours.platform_enforced", "false")
	check("جدولٌ لا يسري")
	h.Setting("hours.platform_enforced", "true")
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE app_settings SET updated_at = $2 WHERE key = $1`,
		"drivers.late_shift_penalty_on", lmAt(2031, 4, 7, 9, 10)); err != nil {
		t.Fatal(err)
	}
	check("شُغّل بعد الموعد")
	// وفي الغد يسري
	if _, p := lateRound(t, h, lmAt(2031, 4, 8, 9, 20)); p != 1 {
		t.Fatalf("اليومُ التالي لتشغيله: عوقب %d — المتوقَّع واحد", p)
	}
}

// TestLateShift_LeaveButton **زرُّ «إجازة اليوم»** يُعلَّم ويُرى ويُتراجَع عنه — ولغير السائق ٤٠٤.
func TestLateShift_LeaveButton(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	cust := h.Customer()
	base := "/api/v1/admin/drivers/" + drv.ID
	if res := h.POST(base+"/leave", admin.Token, map[string]any{}); res.Code != 200 {
		t.Fatalf("الإجازة: %d %s", res.Code, trimBody(res))
	}
	if res := h.GET(base+"/late-shift", admin.Token); res.Code != 200 || res.JSON()["on_leave"] != true {
		t.Fatalf("بعد الإجازة: %d %s — المتوقَّع on_leave", res.Code, trimBody(res))
	}
	if res := h.DEL(base+"/leave", admin.Token); res.Code != 200 {
		t.Fatalf("إلغاءُ الإجازة: %d %s", res.Code, trimBody(res))
	}
	if res := h.GET(base+"/late-shift", admin.Token); res.JSON()["on_leave"] != false {
		t.Fatalf("بعد إلغاء الإجازة: %s", trimBody(res))
	}
	if res := h.POST("/api/v1/admin/drivers/"+cust.ID+"/leave", admin.Token, map[string]any{}); res.Code != 404 {
		t.Errorf("إجازةٌ لزبون: %d — المتوقَّع ٤٠٤", res.Code)
	}
	if res := h.POST(base+"/leave", drv.Token, map[string]any{}); res.Code == 200 {
		t.Errorf("السائقُ يمنح نفسه إجازة: %d — المتوقَّع رفض", res.Code)
	}
}
