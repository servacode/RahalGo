package qa

// ══════════════════════════════════════════════════════════════════════
// **لوحةُ معالم السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٥) — يومُ المنصّة سائقاً سائقاً، والشارة،
// والقائمةُ التي يبقى افتراضُها «بانتظار الموافقة».
// ══════════════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"strings"
	"testing"
)

// lmDailyRow **صفُّ سائقٍ في يوم المعالم.**
type lmDailyRow struct {
	DriverID  string `json:"driver_id"`
	Sent      int    `json:"sent"`
	New       int    `json:"new"`
	Duplicate int    `json:"duplicate"`
	Pending   int    `json:"pending"`
	Approved  int    `json:"approved"`
	Rejected  int    `json:"rejected"`
	Worked    bool   `json:"worked"`
	Completed bool   `json:"completed"`
}

// TestAdminLandmarksDaily **يومُ المعالم لكلّ سائق** — أرسل وجديدٌ ومكرّرٌ وبانتظارٍ وموافَقٌ
// ومرفوض، **ومن اشتغل ولم يرسل شيئاً يظهر بأصفاره.**
func TestAdminLandmarksDaily(t *testing.T) {
	h := New(t)
	h.Setting("drivers.landmarks_daily_quota", "2")
	h.Setting("platform.day_start_hour", "9")
	admin := h.NewUser("admin")
	a, b, idle := h.NewUser("driver"), h.NewUser("driver"), h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.7010, 40.3010
	if r := lmPost(h, a.Token, "جامع الهدى "+sfx, "mosque", lat, lng); r.Code != 200 {
		t.Fatalf("أ١: %s", r)
	}
	first := lmPost(h, a.Token, "صيدلية الهدى "+sfx, "pharmacy", lat, lng)
	if first.Code != 200 {
		t.Fatalf("أ٢: %s", first)
	}
	if r := lmPost(h, a.Token, "جامع الهدى "+sfx, "mosque", lat, lng); r.Code != 409 {
		t.Fatalf("أ٣ مكرّر: %s", r)
	}
	id, _ := first.JSON()["id"].(float64)
	if d := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(id))+"/decide", admin.Token,
		map[string]any{"approve": true}); d.Code != 200 {
		t.Fatalf("الموافقة: %s", d)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = $1`, "صيدلية الهدى "+sfx)
	})
	bad := lmPost(h, b.Token, "فرن الهدى "+sfx, "shop", lat+0.01, lng)
	bid, _ := bad.JSON()["id"].(float64)
	if d := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(bid))+"/decide", admin.Token,
		map[string]any{"approve": false}); d.Code != 200 {
		t.Fatalf("الرفض: %s", d)
	}
	// **ويومٌ ثابتٌ لا الساعةُ الحقيقيّة** (مراجعة ٢٠٢٦-١١-٠٥) — كان الأثرُ «قبل دقيقة» واليومُ
	// الافتراضيّ، **فيسقط من شُغّل في الدقيقة الأولى بعد التاسعة.** فما أُرسل يُنقل إلى يومٍ في
	// ٢٠٣١، **والخاملُ أثرُه ظهرَ ذلك اليوم.**
	const day = "2031-07-20"
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE landmark_suggestions SET platform_day = $3::date
		WHERE suggested_by IN ($1::uuid, $2::uuid)`, a.ID, b.ID, day); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_trail (driver_id, recorded_at, lat, lng)
		VALUES ($1::uuid, $2, $3, $4)`, idle.ID, lmAt(2031, 7, 20, 12, 0), lat, lng); err != nil {
		t.Fatal(err)
	}

	res := h.GET("/api/v1/admin/ops-map/landmarks/daily?day="+day, admin.Token)
	if res.Code != 200 {
		t.Fatalf("**لا يومَ للمعالم في اللوحة**: %s", res)
	}
	var body struct {
		Data struct {
			Day     string       `json:"day"`
			Quota   int          `json:"quota"`
			Drivers []lmDailyRow `json:"drivers"`
			Totals  struct {
				Sent int `json:"sent"`
				New  int `json:"new"`
			} `json:"totals"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Quota != 2 || body.Data.Day != day {
		t.Fatalf("اليومُ والمطلوب: %s", res)
	}
	rows := map[string]lmDailyRow{}
	sumSent := 0
	for _, r := range body.Data.Drivers {
		rows[r.DriverID] = r
		sumSent += r.Sent
	}
	if sumSent != body.Data.Totals.Sent {
		t.Fatalf("مجموعُ الصفوف %d ومربّعُ «أرسلوا» %d", sumSent, body.Data.Totals.Sent)
	}
	want := map[string]lmDailyRow{
		a.ID:    {DriverID: a.ID, Sent: 3, New: 2, Duplicate: 1, Pending: 1, Approved: 1, Completed: true},
		b.ID:    {DriverID: b.ID, Sent: 1, Rejected: 1},
		idle.ID: {DriverID: idle.ID, Worked: true},
	}
	for who, w := range want {
		got, ok := rows[who]
		if !ok {
			t.Fatalf("**السائق %s غائبٌ عن يوم المعالم**: %s", who, res)
		}
		got.Worked = got.Worked && w.Worked // **من أرسل قد يكون بلا أثر — لا يُقاس إلّا للخامل**
		if got != w {
			t.Fatalf("صفُّ %s: %+v والمنتظَر %+v", who, got, w)
		}
	}
	if bad := h.GET("/api/v1/admin/ops-map/landmarks/daily?day=amس", admin.Token); bad.Code != 422 && bad.Code != 400 {
		t.Fatalf("يومٌ فاسد: %s", bad)
	}
}

// TestAdminLandmarkSuggestions_DefaultStillPending **افتراضُ القائمة «بانتظار الموافقة» بلا
// المكرَّر** — فخريطةُ العمليّات لا تتغيّر؛ **و`status` و`driver` والصفحاتُ تعمل.**
func TestAdminLandmarkSuggestions_DefaultStillPending(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	sfx := uniq("")
	const lat, lng = 34.7110, 40.3110
	name := "مدرسة الأمل " + sfx
	if r := lmPost(h, drv.Token, name, "school", lat, lng); r.Code != 200 {
		t.Fatalf("الأوّل: %s", r)
	}
	if r := lmPost(h, drv.Token, name, "school", lat, lng); r.Code != 409 {
		t.Fatalf("المكرَّر: %s", r)
	}
	ids := func(res Res) (out []string) {
		t.Helper()
		var b struct {
			Data struct {
				Suggestions []struct {
					Name     string  `json:"name"`
					Status   string  `json:"status"`
					DupName  *string `json:"dup_name"`
					Verified bool    `json:"verified"`
				} `json:"suggestions"`
				Total   int  `json:"total"`
				HasMore bool `json:"has_more"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &b); err != nil {
			t.Fatal(err)
		}
		for _, s := range b.Data.Suggestions {
			if s.Name == name {
				out = append(out, s.Status)
			}
		}
		return out
	}
	def := h.GET("/api/v1/admin/ops-map/landmark-suggestions", admin.Token)
	if got := ids(def); def.Code != 200 || strings.Join(got, ",") != "pending" {
		t.Fatalf("**الافتراضُ ليس «بانتظار الموافقة» وحدَها**: %v — %s", got, def)
	}
	all := h.GET("/api/v1/admin/ops-map/landmark-suggestions?status=all&driver="+drv.ID, admin.Token)
	if got := ids(all); len(got) != 2 {
		t.Fatalf("«الكل» لهذا السائق: %v — %s", got, all)
	}
	dup := h.GET("/api/v1/admin/ops-map/landmark-suggestions?status=duplicate&driver="+drv.ID, admin.Token)
	if got := ids(dup); strings.Join(got, ",") != "duplicate" || !strings.Contains(string(dup.Body), `"dup_name":"`+name+`"`) {
		t.Fatalf("المكرَّرُ وما طابقه: %v — %s", got, dup)
	}
	page := h.GET("/api/v1/admin/ops-map/landmark-suggestions?status=all&limit=1&driver="+drv.ID, admin.Token)
	if !strings.Contains(string(page.Body), `"has_more":true`) || !strings.Contains(string(page.Body), `"total":2`) {
		t.Fatalf("الصفحات: %s", page)
	}
	if bad := h.GET("/api/v1/admin/ops-map/landmark-suggestions?status=whatever", admin.Token); bad.Code < 400 || bad.Code >= 500 {
		t.Fatalf("حالةٌ مجهولة: %s", bad)
	}
}

// TestAdminLandmarkCount **شارةُ «الخريطة من السائقين»** — كم ينتظر الموافقة.
func TestAdminLandmarkCount(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	count := func() int {
		t.Helper()
		res := h.GET("/api/v1/admin/ops-map/landmark-suggestions/count", admin.Token)
		n, ok := res.JSON()["pending"].(float64)
		if res.Code != 200 || !ok {
			t.Fatalf("**لا شارةَ للمعالم**: %s", res)
		}
		return int(n)
	}
	before := count()
	r := lmPost(h, drv.Token, uniq("دوار الشارة "), "roundabout", 34.7210, 40.3210)
	if r.Code != 200 {
		t.Fatalf("المعلم: %s", r)
	}
	if n := count(); n != before+1 {
		t.Fatalf("الشارة %d والمنتظَر %d", n, before+1)
	}
	id, _ := r.JSON()["id"].(float64)
	if d := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(id))+"/decide", admin.Token,
		map[string]any{"approve": false}); d.Code != 200 {
		t.Fatalf("الرفض: %s", d)
	}
	if n := count(); n != before {
		t.Fatalf("الشارةُ بعد الرفض %d والمنتظَر %d", n, before)
	}
}
