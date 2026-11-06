package qa

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يرى الاسمَ الأوّلَ للزبون وحده** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٥)
// ══════════════════════════════════════════════════════════════════════
//
// **الاسمُ الكاملُ مع الحيّ وتكرارِ الطلبات قائمةُ زبائن** يأخذها السائقُ معه
// برّا المنصّة. **فيُقصّ في الخادم لا في الشاشة** — في كلّ بابٍ يقرأ منه السائقُ
// اسمَ الزبون: الطابور · القبول · طلباتي · رأسُ المحادثة · سجلُّ المحادثات ·
// كشفُ الصندوق · التقييمات. **والزبونُ بلا اسمٍ يُقال «زبون» — لا رقمُه.**

import (
	"encoding/json"
	"strings"
	"testing"
)

const fullCustomerName = "سليمان أحمد الخطيب"

// rowsByID **صفوفُ ردٍّ مصفوفةٍ بمعرّفها** — الطابورُ وطلباتي مصفوفتان، **والمصفوفةُ
// تخرج ملفوفةً بـ`data`** (`httpx.JSON`).
func rowsByID(t *testing.T, r Res, id string) map[string]any {
	t.Helper()
	var wrapped struct {
		Data []map[string]any `json:"data"`
	}
	rows := wrapped.Data
	if err := json.Unmarshal(r.Body, &wrapped); err == nil && wrapped.Data != nil {
		rows = wrapped.Data
	} else if err := json.Unmarshal(r.Body, &rows); err != nil {
		t.Fatalf("ردٌّ غيرُ مصفوفة: %s", r)
	}
	for _, m := range rows {
		if m["id"] == id {
			return m
		}
	}
	return nil
}

func TestDRV_FIRSTNAME_EveryDriverReadShowsFirstNameOnly(t *testing.T) {
	h := New(t)
	// **بلا قُربٍ** — الطابورُ يُرى بلا موضعٍ للسائق، والسؤالُ هنا عن الاسم لا عن الحلقة.
	h.Setting("drivers.proximity_enabled", "false")
	fx := newTenureFx(t, h)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET full_name = $2 WHERE id = $1::uuid`, fx.Cust.ID, fullCustomerName); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	const want = "سليمان"

	// (١) الطابور
	q := h.GET("/api/v1/driver/queue", fx.DrvA.Token)
	if q.Code != 200 {
		t.Fatalf("**الطابورُ رُدّ**: %s", q)
	}
	if row := rowsByID(t, q, fx.Order); row == nil {
		t.Fatalf("**الطلبُ ليس في الطابور**: %s", q)
	} else if row["customer_name"] != want {
		t.Errorf("**الطابور**: الاسم %v — والمتوقّع %q", row["customer_name"], want)
	}

	// (٢) ردُّ القبول
	acc := h.POST("/api/v1/driver/orders/"+fx.Order+"/accept", fx.DrvA.Token, nil)
	if acc.Code != 200 {
		t.Fatalf("**القبولُ رُدّ**: %s", acc)
	}
	if got := acc.JSON()["customer_name"]; got != want {
		t.Errorf("**ردُّ القبول**: الاسم %v — والمتوقّع %q", got, want)
	}

	// (٣) طلباتي
	mine := h.GET("/api/v1/driver/orders", fx.DrvA.Token)
	if row := rowsByID(t, mine, fx.Order); row == nil {
		t.Fatalf("**الطلبُ ليس في طلباتي**: %s", mine)
	} else if row["customer_name"] != want {
		t.Errorf("**طلباتي**: الاسم %v — والمتوقّع %q", row["customer_name"], want)
	}

	// (٤) رأسُ المحادثة
	chat := h.GET("/api/v1/orders/"+fx.Order+"/messages", fx.DrvA.Token)
	if chat.Code != 200 {
		t.Fatalf("**المحادثةُ رُدّت**: %s", chat)
	}
	if got := chat.JSON()["peer_name"]; got != want {
		t.Errorf("**رأسُ المحادثة**: %v — والمتوقّع %q", got, want)
	}
	// **والزبونُ يرى «كابتن رحال غو» كما كان** — الحجبُ في اتّجاهٍ واحدٍ لم يتغيّر.
	if got := h.GET("/api/v1/orders/"+fx.Order+"/messages", fx.Cust.Token).JSON()["peer_name"]; got == want {
		t.Errorf("**الزبونُ صار يرى اسمَه مكانَ السائق**")
	}

	// (٥) سجلُّ المحادثات
	if th := chatThread(t, h, fx.DrvA.Token, fx.Order); th == nil {
		t.Errorf("**الطلبُ ليس في سجلّ محادثات السائق**")
	} else if th["peer"] != want {
		t.Errorf("**سجلُّ المحادثات**: %v — والمتوقّع %q", th["peer"], want)
	}

	// (٦) كشفُ الصندوق — قيدُ تحصيلٍ على الطلب.
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, ref, note)
		VALUES ($1::uuid, 1000, 'order_collection', $2, 'qa first name')`, fx.DrvA.ID, fx.Order); err != nil {
		t.Fatalf("قيد: %v", err)
	}
	cash := h.GET("/api/v1/driver/cash", fx.DrvA.Token)
	if cash.Code != 200 {
		t.Fatalf("**الكشفُ رُدّ**: %s", cash)
	}
	entries, _ := cash.JSON()["entries"].([]any)
	found := false
	for _, x := range entries {
		e, _ := x.(map[string]any)
		if e["note"] == "qa first name" {
			found = true
			if e["customer_name"] != want {
				t.Errorf("**كشفُ الصندوق**: %v — والمتوقّع %q", e["customer_name"], want)
			}
		}
	}
	if !found {
		t.Errorf("**القيدُ ليس في الكشف**: %s", cash)
	}

	// (٧) التقييمات — والزبونُ الثاني بلا اسمٍ يُقال «زبون» لا رقمُه.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now()
		 WHERE id = $1::uuid`, fx.Order); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_ratings (order_id, customer_id, platform_stars, driver_stars)
		VALUES ($1::uuid, $2::uuid, 5, 5)`, fx.Order, fx.Cust.ID); err != nil {
		t.Fatalf("تقييم: %v", err)
	}
	nameless := h.Customer()
	made := h.POSTKey("/api/v1/orders", nameless.Token, uniq("k"), orderBody(h.NewItem(1000), 1))
	oid2, _ := made.JSON()["id"].(string)
	if oid2 == "" {
		t.Fatalf("**الطلبُ الثاني رُدّ**: %s", made)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET full_name = '' WHERE id = $1::uuid`, nameless.ID); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET driver_id = $2::uuid, status = 'delivered', delivered_at = now(), closed_at = now()
		 WHERE id = $1::uuid`, oid2, fx.DrvA.ID); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_ratings (order_id, customer_id, platform_stars, driver_stars)
		VALUES ($1::uuid, $2::uuid, 4, 4)`, oid2, nameless.ID); err != nil {
		t.Fatalf("تقييم: %v", err)
	}
	rep := h.GET("/api/v1/me/reputation", fx.DrvA.Token)
	if rep.Code != 200 {
		t.Fatalf("**السمعةُ رُدّت**: %s", rep)
	}
	// **ولا رقمَ الزبون في الردّ أبداً** — بأيّ شكلٍ كُتب.
	if strings.Contains(string(rep.Body), strings.TrimPrefix(nameless.Phone, "+")[3:]) {
		t.Errorf("**رقمُ الزبون في ردّ السمعة**: %s", rep)
	}
	reviews, _ := rep.JSON()["reviews"].([]any)
	seen := map[string]bool{}
	for _, x := range reviews {
		rv, _ := x.(map[string]any)
		name, _ := rv["customer_name"].(string)
		seen[name] = true
		if strings.Contains(name, "الخطيب") {
			t.Errorf("**التقييمات**: الاسمُ الكامل %q", name)
		}
	}
	if !seen[want] {
		t.Errorf("**التقييمات**: لا «%s» بين الأسماء %v", want, seen)
	}
	if !seen["زبون"] {
		t.Errorf("**التقييمات**: الزبونُ بلا اسمٍ لم يُقل «زبون» — %v", seen)
	}
}
