package qa

import "testing"

// TestDemand_HourFilterCountsOnlyThoseHours **ساعاتُ اليوم في كثافة الطلب**
// (قرارُ المالك ٢٠٢٦-١٠-٢١): ثلاثةُ طلباتٍ الساعةَ ٩ مساءً واثنان الساعةَ ١٠ صباحاً
// بتوقيت سوريا — **فمن ٨ إلى ١١ مساءً ثلاثة، ومن ١٠ ليلاً إلى ١١ صباحاً (عبورُ
// منتصف الليل) اثنان، وبلا ساعاتٍ خمسة.**
func TestDemand_HourFilterCountsOnlyThoseHours(t *testing.T) {
	h := New(t)
	tok := h.NewUser("admin").Token
	// **نقطةٌ لا يزرع فيها غيرُنا.**
	const lat, lng = 33.323456, 41.787654
	at := func(hour, n int) {
		seedRequests(t, h, lat, lng, n)
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE coverage_requests
			   SET created_at = ((date_trunc('day', now() AT TIME ZONE 'Asia/Damascus') - interval '1 day'
			                     + make_interval(hours => $3)) AT TIME ZONE 'Asia/Damascus')
			 WHERE ST_DWithin(at, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, 5)
			   AND created_at BETWEEN now() - interval '1 minute' AND now() + interval '1 second'`,
			lat, lng, hour); err != nil {
			t.Fatal(err)
		}
	}
	at(21, 3)
	at(10, 2)

	all := demandOf(t, h, tok, "")
	if n := cellAt(all.Requests, lat, lng, all.CellDeg); n != 5 {
		t.Fatalf("بلا ساعات = %d — والمتوقَّع ٥", n)
	}
	eve := demandOf(t, h, tok, "?hour_from=20&hour_to=23")
	if n := cellAt(eve.Requests, lat, lng, eve.CellDeg); n != 3 {
		t.Fatalf("من ٨ إلى ١١ مساءً = %d — والمتوقَّع ٣", n)
	}
	night := demandOf(t, h, tok, "?hour_from=22&hour_to=11")
	if n := cellAt(night.Requests, lat, lng, night.CellDeg); n != 2 {
		t.Fatalf("من ١٠ ليلاً إلى ١١ صباحاً = %d — والمتوقَّع ٢ (عبورُ منتصف الليل)", n)
	}
	// **والساعاتُ الخاطئةُ تُهمَل** — فتُعرض كلُّها لا خريطةٌ بيضاء.
	bad := demandOf(t, h, tok, "?hour_from=25&hour_to=3")
	if n := cellAt(bad.Requests, lat, lng, bad.CellDeg); n != 5 {
		t.Fatalf("ساعاتٌ خاطئة = %d — والمتوقَّع ٥ (تُهمَل)", n)
	}
}
