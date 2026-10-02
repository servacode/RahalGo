package qa

// ══════════════════════════════════════════════════════════════════════
// **حديثٌ جديدٌ لكلّ سائق** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الحديثُ للطلب وحدَه**: السائقُ الثاني يقرأ كلَّ ما قيل للأوّل،
// **والأوّلُ بعد أن ترك الطلبَ يقرأ ما يقوله الزبونُ للثاني ويَسِمه
// مقروءاً.** **والزبونُ يقرأ «أُسند سائقٌ لطلبك» مرّتين** ولا يعلم أنّ
// الثانيةَ رجلٌ آخر.
//
// **وكلُّه بالمسار الحقيقيّ** — قبولٌ وتركٌ وقبولٌ من الباب نفسِه الذي
// يضغطه السائق، **لا بكتابةٍ في القاعدة.**

import (
	"net/http"
	"strings"
	"testing"
)

// tenureFx **طلبٌ في الطابور وسائقان على وردية.**
type tenureFx struct {
	Cust  *User
	DrvA  *User
	DrvB  *User
	Order string
}

func newTenureFx(t *testing.T, h *Harness) tenureFx {
	t.Helper()
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.cash_limit", "9000000")

	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(h.NewItem(1000), 1))
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("**الطلبُ رُدّ**: %s", made)
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'dispatching' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	return tenureFx{Cust: cust, DrvA: f.Driver(OnShift()), DrvB: f.Driver(OnShift()), Order: oid}
}

func acceptOrder(t *testing.T, h *Harness, drv *User, oid string) {
	t.Helper()
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("**القبولُ رُدّ**: %s", got)
	}
	// **وتحيّتُه وسطرُ خطوته كُتبا باسمه للتوّ** — فحدُّ المعدّل يردّ أوّلَ
	// ما يكتبه بيده. **فيُزاح الزمن** كما في `sayAged`.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE order_messages SET created_at = created_at - interval '1 minute'
		 WHERE order_id = $1::uuid`, oid); err != nil {
		t.Fatalf("إزاحةُ الزمن: %v", err)
	}
}

func releaseOrder(t *testing.T, h *Harness, drv *User, oid string) {
	t.Helper()
	if got := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("**التركُ رُدّ**: %s", got)
	}
}

// chatThread **صفُّ طلبٍ بعينه في سجلّ المحادثات.**
func chatThread(t *testing.T, h *Harness, tok, orderID string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/my/chats", tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**سجلُّ المحادثات رُدّ**: %d", r.Code)
	}
	rows, _ := r.JSON()["threads"].([]any)
	for _, x := range rows {
		m, _ := x.(map[string]any)
		if m["order_id"] == orderID {
			return m
		}
	}
	return nil
}

const releaseApology = "أعتذر: تعذّر عليّ إكمالُ طلبك — سيتابعه سائقٌ آخر بعد قليل."

// TestCHAT_TENURE_FreshChatPerDriver **حديثٌ جديدٌ للثاني، وولايةُ الأوّل
// له وحدَه.**
func TestCHAT_TENURE_FreshChatPerDriver(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)

	acceptOrder(t, h, fx.DrvA, fx.Order)
	sayAged(t, h, fx.DrvA.Token, fx.Order, "أنا عند المتجر")
	sayAged(t, h, fx.Cust.Token, fx.Order, "الطابق الثالث")
	releaseOrder(t, h, fx.DrvA, fx.Order)

	// **والاعتذارُ في ولاية التارك** — لا بلا ولايةٍ ولا في ولاية من بعده.
	var stamped *string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT driver_id::text FROM order_messages
		 WHERE order_id = $1::uuid AND body = $2`, fx.Order, releaseApology).Scan(&stamped); err != nil {
		t.Fatalf("**لا اعتذارَ في الحديث**: %v", err)
	}
	if stamped == nil || *stamped != fx.DrvA.ID {
		t.Fatalf("**الاعتذارُ في غير ولاية التارك**: %v", stamped)
	}

	// **والزبونُ بلا سائقٍ يقرأ آخرَ ولايةٍ مقفلة** — والاعتذارُ آخرُها.
	got := readChat(t, h, fx.Cust.Token, fx.Order)
	if !chatHas(got, "أنا عند المتجر") || !chatHas(got, releaseApology) {
		t.Fatalf("**الزبونُ لا يقرأ حديثَ سائقه السابق**: %v", got)
	}
	if r := say(t, h, fx.Cust.Token, fx.Order, "هل من أحد؟"); r.Code == http.StatusCreated {
		t.Fatalf("**كتب الزبونُ في ولايةٍ مقفلة**")
	}
	if th := chatThread(t, h, fx.Cust.Token, fx.Order); th == nil || th["open"] == true {
		t.Fatalf("**ولايةٌ مقفلةٌ تظهر مفتوحةً للزبون**: %v", th)
	}

	acceptOrder(t, h, fx.DrvB, fx.Order)

	// **والثاني يبدأ حديثاً فارغاً** — إلّا تحيّتَه وسطرَ خطوته.
	got = readChat(t, h, fx.DrvB.Token, fx.Order)
	for _, old := range []string{"أنا عند المتجر", "الطابق الثالث", releaseApology} {
		if chatHas(got, old) {
			t.Fatalf("**قرأ السائقُ الثاني ما قيل للأوّل** (%q): %v", old, got)
		}
	}
	for _, m := range got {
		if m["role"] != "driver" || m["mine"] != true {
			t.Fatalf("**في حديث الثاني سطرٌ ليس منه**: %v", m)
		}
	}

	// **والزبونُ يقرأ ولايةَ سائقه الآن وحدَها.**
	got = readChat(t, h, fx.Cust.Token, fx.Order)
	if chatHas(got, "أنا عند المتجر") || chatHas(got, releaseApology) {
		t.Fatalf("**الزبونُ يقرأ ولايةَ السابق وله سائق**: %v", got)
	}
	sayAged(t, h, fx.Cust.Token, fx.Order, "الباب الأزرق")

	// **والأوّلُ يقرأ ولايتَه وحدَها** — ولا يرى ما قيل بعده.
	got = readChat(t, h, fx.DrvA.Token, fx.Order)
	if !chatHas(got, "الطابق الثالث") || !chatHas(got, releaseApology) {
		t.Fatalf("**السائقُ الأوّلُ لا يقرأ ولايتَه**: %v", got)
	}
	if chatHas(got, "الباب الأزرق") {
		t.Fatalf("**قرأ الأوّلُ ما قيل للثاني**: %v", got)
	}
	// **ولا يَسِم شيئاً** — «قُرئت» منه كذبٌ.
	var unreadNew int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_messages
		 WHERE order_id = $1::uuid AND body = 'الباب الأزرق' AND read_at IS NULL`,
		fx.Order).Scan(&unreadNew); err != nil {
		t.Fatalf("عدّ: %v", err)
	}
	if unreadNew != 1 {
		t.Fatalf("**فتحُ الأوّل وسم رسالةَ الثاني مقروءة**")
	}
	// **وسجلُّه لا يحمل عدّاً ولا سطراً من ولاية غيره.**
	th := chatThread(t, h, fx.DrvA.Token, fx.Order)
	if th == nil {
		t.Fatalf("**ولايةُ الأوّل غابت عن سجلّه**")
	}
	if n, _ := th["unread"].(float64); n != 0 {
		t.Fatalf("**سجلُّ الأوّل يعدّ رسائلَ الثاني**: %v", th)
	}
	if th["last_body"] != releaseApology {
		t.Fatalf("**آخرُ سطرٍ في سجلّ الأوّل ليس من ولايته**: %v", th["last_body"])
	}
	if th["open"] == true {
		t.Fatalf("**ولايةُ التارك مفتوحةٌ له**")
	}
	// **ولا يكتب.**
	if r := say(t, h, fx.DrvA.Token, fx.Order, "ما زلتُ هنا"); r.Code == http.StatusCreated {
		t.Fatalf("**كتب سائقٌ ترك الطلب**")
	}

	// **والثاني يفتح فيَسِم ما وُجّه إليه.**
	_ = readChat(t, h, fx.DrvB.Token, fx.Order)
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_messages
		 WHERE order_id = $1::uuid AND body = 'الباب الأزرق' AND read_at IS NULL`,
		fx.Order).Scan(&unreadNew); err != nil {
		t.Fatalf("عدّ: %v", err)
	}
	if unreadNew != 0 {
		t.Fatalf("**السائقُ الحاملُ فتح ولم يُوسَم ما قرأه**")
	}
}

// TestCHAT_TENURE_CustomerToldDriverChanged **الإسنادُ الثاني يقول إنّه
// سائقٌ آخر — والأوّلُ بنصّه.**
func TestCHAT_TENURE_CustomerToldDriverChanged(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)

	titles := func() []string {
		t.Helper()
		rows, err := h.Pool.Query(ctxBG(), `
			SELECT title || ' | ' || body FROM notifications
			 WHERE user_id = $1::uuid AND entity_id::text = $2 AND kind = 'order'
			 ORDER BY created_at, id`, fx.Cust.ID, fx.Order)
		if err != nil {
			t.Fatalf("قراءةُ الإشعارات: %v", err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			out = append(out, s)
		}
		return out
	}
	count := func(list []string, prefix string) int {
		n := 0
		for _, s := range list {
			if strings.HasPrefix(s, prefix) {
				n++
			}
		}
		return n
	}

	acceptOrder(t, h, fx.DrvA, fx.Order)
	first := titles()
	if count(first, "أُسند سائقٌ لطلبك") != 1 || count(first, "تم تغيير السائق") != 0 {
		t.Fatalf("**الإسنادُ الأوّلُ بغير نصّه**: %v", first)
	}

	releaseOrder(t, h, fx.DrvA, fx.Order)
	acceptOrder(t, h, fx.DrvB, fx.Order)

	var nameB string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT full_name FROM users WHERE id = $1::uuid`, fx.DrvB.ID).Scan(&nameB); err != nil {
		t.Fatalf("اسمُ الثاني: %v", err)
	}
	all := titles()
	if count(all, "أُسند سائقٌ لطلبك") != 1 {
		t.Fatalf("**الإسنادُ الثاني قيل بنصّ الأوّل**: %v", all)
	}
	if count(all, "تم تغيير السائق") != 1 {
		t.Fatalf("**الزبونُ لم يُخبَر أنّ سائقَه تغيّر**: %v", all)
	}
	for _, s := range all {
		if strings.HasPrefix(s, "تم تغيير السائق") && !strings.Contains(s, "سائقك الجديد: "+nameB) {
			t.Fatalf("**الإشعارُ لا يسمّي السائقَ الجديد**: %q", s)
		}
	}
}
