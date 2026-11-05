package qa

// ══════════════════════════════════════════════════════════════════════
// **صفحةُ المراقبة تعدّ بلاغات «طلب رقمي» والأرقامَ المخفيّة لكلّ سائق** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **بلاغُ الزبون وحدَه لا يكفي** — السائقُ الذي يكرّرها يُرى بعددها، ومعه رسائلُه التي
// أُخفي منها رقمٌ أو واتساب (البند ١). **آخرُ ثلاثين يوماً**، والأكثرُ أوّلاً.

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDriverMonitor_CountsOffPlatformReportsAndContactLeaks(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	drv, day, _ := dmBuildDay(t, h, f, false)
	m := f.Merchant()
	now := time.Now()
	dmOrder(t, h, drv, m.ID, now.Add(-3*time.Hour), now.Add(-150*time.Minute), now.Add(-2*time.Hour))
	dmOrder(t, h, drv, m.ID, now.Add(-90*time.Minute), now.Add(-80*time.Minute), now.Add(-time.Hour))

	rows, err := h.Pool.Query(ctxBG(),
		`SELECT id::text, customer_id::text FROM orders WHERE driver_id = $1::uuid ORDER BY created_at`, drv)
	if err != nil {
		t.Fatal(err)
	}
	type ord struct{ id, cust string }
	var list []ord
	for rows.Next() {
		var o ord
		if err := rows.Scan(&o.id, &o.cust); err != nil {
			t.Fatal(err)
		}
		list = append(list, o)
	}
	rows.Close()
	if len(list) < 3 {
		t.Fatalf("طلباتُ السائق: %d", len(list))
	}
	// **بلاغان على السائق** — وبلاغٌ ثالثٌ بسببٍ آخر لا يُعدّ.
	for i, reason := range []string{"off_platform", "off_platform", "late"} {
		against := any(drv)
		if reason == "late" {
			against = nil
		}
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO tickets (customer_id, order_id, subject, reason, created_by, opened_by_customer, against_user_id)
			VALUES ($1::uuid, $2::uuid, 'بلاغ', $3, $1::uuid, true, $4)`,
			list[i].cust, list[i].id, reason, against); err != nil {
			t.Fatalf("بلاغ: %v", err)
		}
	}
	// **ورسالةٌ مخفيّةٌ منه** — ورسالةُ الزبون المخفيّة لا تُعدّ عليه.
	for _, x := range []struct{ sender, role string }{{drv, "driver"}, {list[0].cust, "customer"}} {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO order_messages (order_id, sender_id, sender_role, body, driver_id, flagged, contact_leak)
			VALUES ($1::uuid, $2::uuid, $3, '***', $4::uuid, true, true)`,
			list[0].id, x.sender, x.role, drv); err != nil {
			t.Fatalf("رسالة: %v", err)
		}
	}

	res := h.GET("/api/v1/admin/driver-monitor?day="+day, admin.Token)
	if res.Code != 200 {
		t.Fatalf("القائمة: %s", res)
	}
	var env struct {
		Data struct {
			Drivers []struct {
				ID           string `json:"driver_id"`
				OffPlatform  int    `json:"off_platform_reports"`
				ContactLeaks int    `json:"contact_leaks"`
			} `json:"drivers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range env.Data.Drivers {
		if d.ID != drv {
			continue
		}
		found = true
		if d.OffPlatform != 2 || d.ContactLeaks != 1 {
			t.Errorf("**العدّ**: بلاغات=%d (والمتوقّع ٢) · أرقام=%d (والمتوقّع ١)", d.OffPlatform, d.ContactLeaks)
		}
	}
	if !found {
		t.Fatalf("**السائقُ ليس في القائمة**: %s", res)
	}
}
