package qa

import (
	"context"
	"testing"
)

// TestStoreAnnounce_OnceWhenVisible **«انضمّ متجر جديد»** (قرارُ المالك ٢٠٢٦-١٠-٢١): يُعلَن
// عن المتجر حين يصير ظاهراً للزبائن — **لا قبل** (بلا دوام)، **ومرّةً واحدة**، ونقرُه يفتح
// صفحتَه. **و«لا ترسل» تمنعه.**
func TestStoreAnnounce_OnceWhenVisible(t *testing.T) {
	h := New(t)
	noQuiet(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	skipped := h.NewItem(1000)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET announce_skip = true WHERE id = $1::uuid`,
		skipped.MerchantID); err != nil {
		t.Fatal(err)
	}
	sweep := func() {
		for i := 0; i < 50 && h.API.SweepStoreAnnounce(context.Background()) > 0; i++ {
		}
	}
	announced := func(mid string) bool {
		var ok bool
		if err := h.Pool.QueryRow(ctxBG(), `SELECT announced_at IS NOT NULL FROM merchants WHERE id = $1::uuid`,
			mid).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}

	// ١ · بلا دوامٍ ليس ظاهراً ⇒ لا إعلان.
	sweep()
	if announced(item.MerchantID) {
		t.Fatal("**أُعلن عن متجرٍ بلا دوام**")
	}

	// ٢ · بدوامٍ ⇒ إعلانٌ واحدٌ بوجهة المتجر، ويصل الزبون.
	for _, mid := range []string{item.MerchantID, skipped.MerchantID} {
		if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1::uuid, 0, false, '09:00', '22:00') ON CONFLICT DO NOTHING`, mid); err != nil {
			t.Fatal(err)
		}
	}
	sweep()
	sweep()
	if !announced(item.MerchantID) {
		t.Fatal("**متجرٌ ظاهرٌ لم يُعلَن عنه**")
	}
	if announced(skipped.MerchantID) {
		t.Fatal("**«لا ترسل» لم تمنع الإعلان**")
	}
	var n, got int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*),
		(SELECT count(*) FROM campaign_recipients r JOIN campaigns c2 ON c2.id = r.campaign_id
		  WHERE c2.dest_id = $1::uuid AND r.user_id = $2::uuid)
		FROM campaigns c WHERE c.dest_type = 'merchant' AND c.dest_id = $1::uuid`,
		item.MerchantID, cust.ID).Scan(&n, &got); err != nil {
		t.Fatal(err)
	}
	if n != 1 || got != 1 {
		t.Fatalf("**الإعلانُ ليس مرّةً واحدةً للزبون**: حملات %d، وصل الزبون %d", n, got)
	}
}

// TestStoreAnnounce_AdminSkip **«لا ترسل» من اللوحة** — تُقرأ وتُضبط، ومتجرٌ مجهولٌ ٤٠٤.
func TestStoreAnnounce_AdminSkip(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	item := h.NewItem(1000)
	path := "/api/v1/admin/merchants/" + item.MerchantID + "/announce"
	if r := h.GET(path, admin.Token); r.Code != 200 || r.JSON()["skip"] != false {
		t.Fatalf("القراءة: %s", r)
	}
	if r := h.Call("PATCH", path, admin.Token, map[string]any{"skip": true}, nil); r.Code != 200 || r.JSON()["skip"] != true {
		t.Fatalf("الضبط: %s", r)
	}
	if r := h.Call("PATCH", path, admin.Token, map[string]any{}, nil); r.Code != 400 && r.Code != 422 {
		t.Fatalf("بلا قيمةٍ يجب أن يُردّ: %s", r)
	}
	if r := h.GET("/api/v1/admin/merchants/00000000-0000-0000-0000-000000000000/announce", admin.Token); r.Code != 404 {
		t.Fatalf("متجرٌ مجهول: %s", r)
	}
}
