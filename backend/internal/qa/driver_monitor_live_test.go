package qa

// الحالة الحيّة لسائق (يمشي · متوقف · غير متصل) — طلب المالك ٢٠٢٦-١١-٠٤.

import (
	"encoding/json"
	"testing"
	"time"
)

type dmLiveRow struct {
	DriverID string `json:"driver_id"`
	Live     *struct {
		State    string     `json:"state"`
		Since    *time.Time `json:"since"`
		LastSeen *time.Time `json:"last_seen"`
		OnShift  bool       `json:"on_shift"`
		HasOrder bool       `json:"has_order"`
	} `json:"live"`
}

func TestDriverMonitor_LiveStateMovingStoppedOffline(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	mover, stopper, gone, idle := f.Driver(), f.Driver(), f.Driver(), f.Driver()
	now := time.Now()
	p := dmSpot{35.6000, 38.6000}
	// السرعة المبلَّغة صفر: الحكم هنا بالإزاحة وحدها.
	put := func(drv string, ago time.Duration, dLat float64) {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
			VALUES ($1::uuid, $2, $3, $4, 5, 0, false) ON CONFLICT DO NOTHING`,
			drv, now.Add(-ago), p.lat+dLat, p.lng); err != nil {
			t.Fatal(err)
		}
	}
	// يمشي: ~٢٢ م كل ٣٠ ث => ~١٠٠ م خلال دقيقتين
	for i := 4; i >= 0; i-- {
		put(mover.ID, time.Duration(i)*30*time.Second, float64(4-i)*0.0002)
	}
	// متوقف: وقف منذ ~٦ دقائق بنبضة كل دقيقة
	for i := 6; i >= 0; i-- {
		put(stopper.ID, time.Duration(i)*time.Minute, 0.00001*float64(i%2))
	}
	// غير متصل: آخر نقطة منذ ١٥ دقيقة
	put(gone.ID, 15*time.Minute, 0)
	// على الدوام بلا أي أثر
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = true WHERE id = $1::uuid`, idle.ID); err != nil {
		t.Fatal(err)
	}

	res := h.GET("/api/v1/admin/driver-monitor", admin.Token)
	if res.Code != 200 {
		t.Fatalf("القائمة: %s", res)
	}
	var env struct {
		Data struct {
			Drivers []dmLiveRow `json:"drivers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	by := map[string]dmLiveRow{}
	for _, r := range env.Data.Drivers {
		by[r.DriverID] = r
	}
	check := func(id, want string) dmLiveRow {
		r, ok := by[id]
		if !ok || r.Live == nil {
			t.Fatalf("السائق %s غائب أو بلا live", id)
		}
		if r.Live.State != want {
			t.Errorf("الحالة = %s والمتوقَّع %s", r.Live.State, want)
		}
		return r
	}
	check(mover.ID, "moving")
	s := check(stopper.ID, "stopped")
	if s.Live.Since == nil || now.Sub(*s.Live.Since) < 5*time.Minute {
		t.Errorf("since للمتوقف يجب أن يرجع لبداية الوقوف (~٦ د): %v", s.Live.Since)
	}
	g := check(gone.ID, "offline")
	if g.Live.LastSeen == nil || now.Sub(*g.Live.LastSeen) < 14*time.Minute {
		t.Errorf("last_seen = %v", g.Live.LastSeen)
	}
	i := check(idle.ID, "offline")
	if !i.Live.OnShift || i.Live.LastSeen != nil {
		t.Errorf("على الدوام بلا أثر: on_shift=%v last_seen=%v", i.Live.OnShift, i.Live.LastSeen)
	}
	// يوم ماضٍ: لا live
	past := time.Now().In(time.UTC).AddDate(0, 0, -3).Format("2006-01-02")
	res = h.GET("/api/v1/admin/driver-monitor?day="+past, admin.Token)
	if res.Code != 200 {
		t.Fatalf("يوم ماضٍ: %s", res)
	}
	var penv struct {
		Data struct {
			Drivers []dmLiveRow `json:"drivers"`
		} `json:"data"`
	}
	_ = json.Unmarshal(res.Body, &penv)
	for _, r := range penv.Data.Drivers {
		if r.Live != nil {
			t.Errorf("يوم ماضٍ يحمل live")
		}
	}
}
