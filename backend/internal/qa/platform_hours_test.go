package qa

// ══════════════════════════════════════════════════════════════════════
// **دوامُ المنصّة والإيقافُ المؤقّت — من باب الشبكة** (`PH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وحدودُ الدقيقة وموعدُ العودة مقيسةٌ بالحساب** في `internal/platform`
// — **ثلاثٌ وعشرون حالةً بلا قاعدةٍ ولا خادم.**
//
// **وهذه تقيس ما لا يُقاس هناك**: **أنّ البابَ يردّ فعلاً**، **وأنّ
// المردودَ لا يُخلّف أثراً**، **وأنّ الطلبَ القائمَ يمضي**، **وأنّ
// ترتيبَ الأسبقيّة محفوظٌ مع وضع الإطلاق.**

import (
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ═════════════════ تجهيزٌ ═════════════════

// setHours **يكتب جدولَ الأسبوع مباشرةً** — **ولا يُفعّله.**
func setHours(t *testing.T, h *Harness, ws ...platform.Window) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM platform_hours`); err != nil {
		t.Fatalf("مسحُ الجدول: %v", err)
	}
	for _, w := range ws {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO platform_hours (day_of_week, starts_at, ends_at)
			VALUES ($1, make_time($2, $3, 0), make_time($4, $5, 0))`,
			w.Day, int(w.Start)/60, int(w.Start)%60,
			int(w.End)/60, int(w.End)%60); err != nil {
			t.Fatalf("كتابةُ فترة: %v", err)
		}
	}
}

// around **فترةٌ تبدأ بعد `from` من الآن وتنتهي بعد `to`.**
//
// **واليومُ يُقرأ من لحظة البدء لا من اليوم الجاري** — **فحدُّ منتصف
// الليل يُصيب الفحصَ نفسَه**: **فحصٌ يُشغَّل الحاديةَ عشرةَ ليلاً
// و«بعد ساعتين» يقع في الغد.**
func around(from, to time.Duration) platform.Window {
	loc := platform.Location()
	a := time.Now().In(loc).Add(from)
	b := time.Now().In(loc).Add(to)
	return platform.Window{
		Day:   int(a.Weekday()),
		Start: platform.Minutes(a.Hour()*60 + a.Minute()),
		End:   platform.Minutes(b.Hour()*60 + b.Minute()),
	}
}

// openNow **جدولٌ يجعل هذه اللحظةَ داخلَ الدوام.**
func openNow(t *testing.T, h *Harness) {
	t.Helper()
	setHours(t, h, around(-time.Hour, time.Hour))
	h.Setting(platform.EnforcedKey, "true")
}

// closedNow **جدولٌ يجعل هذه اللحظةَ خارجَ الدوام** — فترةٌ بعد ساعتين.
func closedNow(t *testing.T, h *Harness) {
	t.Helper()
	setHours(t, h, around(2*time.Hour, 3*time.Hour))
	h.Setting(platform.EnforcedKey, "true")
}

// closure **يضبط الإيقافَ المؤقّت مباشرةً.**
func closure(t *testing.T, h *Harness, active bool, msg string, ends *time.Time) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE service_closure SET active = $1, message = $2, ends_at = $3 WHERE id`,
		active, msg, ends); err != nil {
		t.Fatalf("ضبطُ الإيقاف: %v", err)
	}
}

// ordersOpen **يفتح أبوابَ الإطلاق كلَّها للطلب ويمسح ما قبلَه.**
//
// **والقاعدةُ مشتركةٌ بين الفحوص** — **وصفُّ الإيقاف واحدٌ للمنصّة
// كلِّها**: **فحصٌ يُفعّله يتركه مُفعَّلاً لمن بعده**، **فيسقط فحصٌ
// سليمٌ بسبب جارِه ويُطارَد العطبُ في غير موضعه.** (وقع فعلاً أوّلَ
// تشغيل.)
//
// **فتُمسَح الحالُ في مبتدأ كلّ فحصٍ لا في خاتمته** — **وخاتمةٌ لا
// تُنفَّذ إن سقط الفحصُ قبلها.**
func ordersOpen(t *testing.T, h *Harness) {
	t.Helper()
	h.Setting("launch.customer_orders", "true")
	h.Setting("launch.merchant_orders", "true")
	h.Setting("launch.customer_custom_orders", "true")
	closure(t, h, false, "", nil)
	setHours(t, h)
	h.Setting(platform.EnforcedKey, "false")
}

// phCustomBody طلبٌ مخصَّصٌ صالح.
func phCustomBody() map[string]any {
	return map[string]any{
		"request":        "كيلو بندورة",
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            35.9506,
		"lng":            39.0094,
		"payment_method": "cash",
	}
}

// isClosedNow **أهذا ردُّ «خارجَ الدوام»؟**
func isClosedNow(r Res) bool {
	return r.Code == http.StatusServiceUnavailable && r.Err() == "platform_closed_now"
}

// isTempClosed **أهذا ردُّ «موقوفٌ مؤقّتاً»؟**
func isTempClosed(r Res) bool {
	return r.Code == http.StatusServiceUnavailable && r.Err() == "temporarily_unavailable"
}

// ═════════════════ PH-17 · PH-18 — الجدولُ يمنع ═════════════════

// TestPH17_WeeklyClosedBlocksNormalOrder **خارجَ الدوام لا يُقبَل طلبٌ
// عاديّ.**
func TestPH17_WeeklyClosedBlocksNormalOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if !isClosedNow(r) {
		t.Fatalf("**طلبٌ مرّ خارجَ دوام المنصّة**: %d / %s", r.Code, r.Err())
	}
}

// TestPH18_WeeklyClosedBlocksCustomOrder **والمخصَّصُ كذلك** — **ومكتبٌ
// مغلقٌ لا يحضّر طلباً موصوفاً كما لا يحضّر طلباً من متجر.**
func TestPH18_WeeklyClosedBlocksCustomOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)

	u := hh.Customer()
	r := hh.POST("/api/v1/orders/custom", u.Token, phCustomBody())
	if !isClosedNow(r) {
		t.Fatalf("**طلبٌ مخصَّصٌ مرّ خارجَ دوام المنصّة**: %d / %s", r.Code, r.Err())
	}
}

// **وداخلَ الدوام يمضي** — **وحارسٌ يمنع دائماً ليس حارساً.**
func TestPH_OpenScheduleAdmitsOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if r.Code >= 400 {
		t.Fatalf("**طلبٌ رُدّ داخلَ الدوام**: %d / %s", r.Code, r.Err())
	}
}

// **وجدولٌ غيرُ مفعَّلٍ لا يمنع شيئاً** — **وهو ما يجعل الهجرةَ لا
// تُغلق المنصّةَ يومَ تُنشر.**
func TestPH_DormantScheduleAdmitsOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	setHours(t, hh, around(2*time.Hour, 3*time.Hour))
	hh.Setting(platform.EnforcedKey, "false")

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if r.Code >= 400 {
		t.Fatalf("**جدولٌ غيرُ مفعَّلٍ ردّ طلباً**: %d / %s — "+
			"**فنشرُ الهجرة يوقف المنصّة**", r.Code, r.Err())
	}
}

// ═════════════════ PH-15 · PH-16 — الإيقافُ المؤقّت يمنع ═════════════════

// TestPH15_ClosureBlocksNormalOrder **إيقافٌ مؤقّتٌ يمنع الطلبَ العاديّ
// ولو كنّا في وسط الدوام.**
func TestPH15_ClosureBlocksNormalOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)
	closure(t, hh, true, "صيانةٌ مؤقّتة", nil)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if !isTempClosed(r) {
		t.Fatalf("**طلبٌ مرّ والخدمةُ موقوفة**: %d / %s", r.Code, r.Err())
	}
	// **ونصُّ المالك يصل** — **ورسالةٌ عامّةٌ تُضيع ما كُتب.**
	if got, _ := r.JSON()["error"].(map[string]any); got != nil {
		d, _ := got["details"].(map[string]any)
		if d == nil || d["notice"] != "صيانةٌ مؤقّتة" {
			t.Fatalf("**نصُّ المالك لم يصل العميل**: %v", d)
		}
	}
}

// TestPH16_ClosureBlocksCustomOrder **والمخصَّصُ كذلك.**
func TestPH16_ClosureBlocksCustomOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)
	closure(t, hh, true, "", nil)

	u := hh.Customer()
	r := hh.POST("/api/v1/orders/custom", u.Token, phCustomBody())
	if !isTempClosed(r) {
		t.Fatalf("**طلبٌ مخصَّصٌ مرّ والخدمةُ موقوفة**: %d / %s", r.Code, r.Err())
	}
}

// **وإيقافٌ انقضى وقتُه يُرفَع بنفسه** — **ولا مهمّةَ دوريّةً تُطفئه.**
func TestPH_ExpiredClosureAdmitsOrder(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)
	past := time.Now().Add(-time.Minute)
	closure(t, hh, true, "انتهت", &past)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if r.Code >= 400 {
		t.Fatalf("**إيقافٌ انقضى وقتُه ما زال يمنع**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ PH-14 — أسبقيّةُ وضع الإطلاق ═════════════════

// TestPH14_LaunchClosedPrecedence **وضعُ الإطلاق يبقى هو الجواب.**
//
// **وبابٌ لم يُفتح بعدُ ليس بابَ مغلقٍ مؤقّتاً** — **ومن استبدل رمزَه
// قال للناس «نعود الرابعة» عن بابٍ لا موعدَ لفتحه.**
func TestPH14_LaunchClosedPrecedence(t *testing.T) {
	hh := New(t)
	hh.Setting("launch.customer_orders", "false")
	hh.Setting("launch.customer_custom_orders", "false")
	hh.Setting("launch.merchant_orders", "true")
	// **والطبقتان الأخريان مغلقتان أيضاً** — **فلو سبقت إحداهما لَظهر
	// رمزُها.**
	closedNow(t, hh)
	closure(t, hh, true, "صيانة", nil)

	u := hh.Customer()
	it := hh.NewItem(900)
	normal := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if normal.Err() != "launch_closed" {
		t.Fatalf("**رمزُ وضع الإطلاق استُبدل**: %d / %s", normal.Code, normal.Err())
	}
	custom := hh.POST("/api/v1/orders/custom", u.Token, phCustomBody())
	if custom.Err() != "launch_closed" {
		t.Fatalf("**رمزُ وضع الإطلاق استُبدل في المخصَّص**: %d / %s",
			custom.Code, custom.Err())
	}
}

// **والإيقافُ المؤقّتُ يسبق الجدول** — **والأخصُّ يسبق الأعمّ.**
func TestPH_ClosurePrecedesSchedule(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)
	closure(t, hh, true, "صيانة", nil)

	u := hh.Customer()
	it := hh.NewItem(900)
	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if !isTempClosed(r) {
		t.Fatalf("**الجدولُ سبق الإيقافَ المؤقّت**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ PH-19 — المردودُ بلا أثر ═════════════════

// TestPH19_RejectedOrderLeavesNothing **ولا صفَّ ولا قيدَ ولا إشعار.**
//
// **والمنعُ قبل قراءةِ الجسم وقبل المعاملة** — **فلو نُودي داخلَها
// لَكان إرجاعاً لا منعاً**، **والإرجاعُ يستهلك مفتاحَ التفرّد فيُحبَس
// الزبونُ عن إعادة المحاولة.**
func TestPH19_RejectedOrderLeavesNothing(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)

	u := hh.Customer()
	it := hh.NewItem(900)

	before := hh.CountOrders(u.ID)
	ledgerBefore := phCount(t, hh, `SELECT count(*) FROM wallet_transactions`)
	notifyBefore := phCount(t, hh, `SELECT count(*) FROM notifications`)

	key := uniq("ph19")
	r := hh.POSTKey("/api/v1/orders", u.Token, key, orderBody(it, 1))
	if !isClosedNow(r) {
		t.Fatalf("**لم يُردّ الطلب**: %d / %s", r.Code, r.Err())
	}

	if got := hh.CountOrders(u.ID); got != before {
		t.Fatalf("**صفُّ طلبٍ وُلد من نداءٍ مردود**: %d ← %d", before, got)
	}
	if got := phCount(t, hh, `SELECT count(*) FROM wallet_transactions`); got != ledgerBefore {
		t.Fatalf("**قيدٌ ماليٌّ وُلد من نداءٍ مردود**: %d ← %d", ledgerBefore, got)
	}
	if got := phCount(t, hh, `SELECT count(*) FROM notifications`); got != notifyBefore {
		t.Fatalf("**إشعارٌ وُلد من نداءٍ مردود**: %d ← %d", notifyBefore, got)
	}

	// **ومفتاحُ التفرّد لم يُستهلَك** — **فالزبونُ يعيد المحاولةَ به
	// حين نعود، ولا يُحبَس بمفتاحٍ أُحرق على بابٍ مغلق.**
	openNow(t, hh)
	again := hh.POSTKey("/api/v1/orders", u.Token, key, orderBody(it, 1))
	if again.Code >= 400 {
		t.Fatalf("**المفتاحُ أُحرق على بابٍ مغلق**: %d / %s", again.Code, again.Err())
	}
}

// phCount عدّادٌ قصير.
func phCount(t *testing.T, h *Harness, q string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), q).Scan(&n); err != nil {
		t.Fatalf("عدٌّ: %v", err)
	}
	return n
}

// ═════════════════ PH-20 · PH-21 — الطلبُ القائمُ يمضي ═════════════════

// TestPH20_ExistingOrderSurvivesSchedule **طلبٌ صحّ قبل الإغلاق يبقى
// صحيحاً بعده.**
//
// **والمنعُ على الاستقبال وحدَه** — **ومن ألغى طلباً قائماً لأنّ الدوامَ
// انتهى ترك زبوناً ينتظر طعاماً لن يأتي، وسائقاً في الطريق.**
func TestPH20_ExistingOrderSurvivesSchedule(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)

	fx := populatedOrder(t, hh)
	statusBefore := phStatus(t, hh, fx.OrderID)

	// **ثمّ ينتهي الدوام.**
	closedNow(t, hh)

	if got := phStatus(t, hh, fx.OrderID); got != statusBefore {
		t.Fatalf("**حالُ طلبٍ قائمٍ تبدّلت بانتهاء الدوام**: %q ← %q",
			statusBefore, got)
	}
	if r := hh.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token); r.Code != http.StatusOK {
		t.Fatalf("**الزبونُ لا يرى طلبَه القائمَ بعد الإغلاق**: %d / %s",
			r.Code, r.Err())
	}
	// **والمتجرُ ما زال يقدر أن يعمل عليه** — **وإلّا فالطلبُ مجمَّدٌ
	// لا قائم.**
	if r := hh.GET("/api/v1/merchant/orders/"+fx.OrderID, fx.MerchToken); r.Code != http.StatusOK {
		t.Fatalf("**المتجرُ حُجب عن طلبٍ قائمٍ بعد الإغلاق**: %d / %s",
			r.Code, r.Err())
	}
}

// TestPH21_ExistingOrderSurvivesClosure **والإيقافُ المؤقّتُ كذلك.**
func TestPH21_ExistingOrderSurvivesClosure(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)

	fx := populatedOrder(t, hh)
	statusBefore := phStatus(t, hh, fx.OrderID)

	closure(t, hh, true, "صيانةٌ طارئة", nil)

	if got := phStatus(t, hh, fx.OrderID); got != statusBefore {
		t.Fatalf("**حالُ طلبٍ قائمٍ تبدّلت بالإيقاف المؤقّت**: %q ← %q",
			statusBefore, got)
	}
	if r := hh.GET("/api/v1/my/orders/"+fx.OrderID, fx.Cust.Token); r.Code != http.StatusOK {
		t.Fatalf("**الزبونُ لا يرى طلبَه بعد الإيقاف**: %d / %s", r.Code, r.Err())
	}
}

// phStatus حالُ الطلب من القاعدة.
func phStatus(t *testing.T, h *Harness, id string) string {
	t.Helper()
	var st string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, id).Scan(&st); err != nil {
		t.Fatalf("قراءةُ حال الطلب: %v", err)
	}
	return st
}

// ═════════════════ PH-22 — التصفّحُ يبقى ═════════════════

// TestPH22_BrowsingStaysOpenWhenClosed **الإغلاقُ استقبالٌ لا إطفاءُ
// محرّك.**
//
// **ومن أغلق المنصّةَ كلَّها خارجَ الدوام منع الناسَ من أن يعرفوا متى
// نعود** — **والشاشةُ البيضاءُ لا تقول شيئاً.**
func TestPH22_BrowsingStaysOpenWhenClosed(t *testing.T) {
	hh := New(t)
	hh.Setting("launch.customer_browse", "true")
	ordersOpen(t, hh)
	closedNow(t, hh)
	closure(t, hh, true, "صيانة", nil)

	u := hh.Customer()
	for _, p := range []string{
		"/api/v1/public/home",
		"/api/v1/public/platform",
		"/api/v1/public/cities",
	} {
		if r := hh.GET(p, ""); r.Code != http.StatusOK {
			t.Errorf("**بابُ تصفّحٍ أُغلق مع الاستقبال**: %s ⇒ %d / %s",
				p, r.Code, r.Err())
		}
	}
	if r := hh.GET("/api/v1/my/orders", u.Token); r.Code != http.StatusOK {
		t.Errorf("**الزبونُ حُجب عن طلباته**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ PH-27 · PH-28 — الردُّ العامُّ يقول السبب ═════════════════

// TestPH27_PublicPlatformCarriesReason **والشاشةُ تقرأ السببَ لا تخمّنه.**
//
// **وحقلٌ يُضاف لا عقدٌ يُكسَر** — **والحقولُ القائمةُ تبقى كما هي.**
func TestPH27_PublicPlatformCarriesReason(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	closedNow(t, hh)

	body := hh.GET("/api/v1/public/platform", "").JSON()
	// **والعقدُ القديمُ لم يُكسَر.**
	if _, ok := body["name"]; !ok {
		t.Fatal("**حقلٌ قائمٌ اختفى من الردّ العامّ**: name")
	}
	data, _ := body["ordering"].(map[string]any)
	if data == nil {
		t.Fatal("**حالُ الاستقبال غائبةٌ عن الردّ العامّ** — فالشاشةُ تخمّن")
	}
	if data["ordering_available"] != false {
		t.Fatalf("**الردُّ يقول مفتوحٌ والمحرّكُ يردّ**: %v", data["ordering_available"])
	}
	if data["reason"] != "platform_closed_now" {
		t.Fatalf("**سببٌ غيرُ الذي يردّ به الباب**: %v", data["reason"])
	}
	if data["timezone"] != platform.TZ {
		t.Fatalf("**منطقةٌ غيرُ منطقة المنصّة**: %v", data["timezone"])
	}
	if _, ok := data["server_time"].(string); !ok {
		t.Fatal("**لحظةُ الخادم غائبة** — فالشاشةُ تحسب بساعة الجهاز")
	}
	if _, ok := data["next_available_at"].(string); !ok {
		t.Fatal("**موعدُ العودة غائبٌ وهو معلوم** — فلا يُقال للزبون متى نعود")
	}

	// **ومفتوحاً يقول مفتوحاً.**
	openNow(t, hh)
	open, _ := hh.GET("/api/v1/public/platform", "").JSON()["ordering"].(map[string]any)
	if open["ordering_available"] != true {
		t.Fatalf("**الردُّ يقول مغلقٌ والبابُ يقبل**: %v", open)
	}
}

// TestPH29_StaleClientCannotSubmitAfterClose **عميلٌ قرأ «مفتوح» ثمّ
// أرسل بعد الإغلاق يُردّ.**
//
// **وهو سباقُ الحدّ الذي طلبه المالك**: **يرى ١٦:٥٩:٥٩ مفتوحاً ويصل
// نداؤه ١٧:٠٠.** **والحكمُ لحظةَ وصول النداء لا لحظةَ رسمِ الشاشة.**
func TestPH29_StaleClientCannotSubmitAfterClose(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	openNow(t, hh)

	u := hh.Customer()
	it := hh.NewItem(900)

	// **العميلُ يقرأ الحالَ فيجدها مفتوحة.**
	seen, _ := hh.GET("/api/v1/public/platform", "").JSON()["ordering"].(map[string]any)
	if seen["ordering_available"] != true {
		t.Fatalf("مقدّمةٌ مكسورة: الحالُ مغلقةٌ قبل البدء: %v", seen)
	}

	// **ثمّ ينتهي الدوامُ قبل أن يضغط.**
	closedNow(t, hh)

	r := hh.POST("/api/v1/orders", u.Token, orderBody(it, 1))
	if !isClosedNow(r) {
		t.Fatalf("**حالٌ قديمةٌ في الشاشة تجاوزت المحرّك**: %d / %s", r.Code, r.Err())
	}
}

// ═════════════════ PH-25 · PH-26 — اللوحةُ ترفض ما لا يصلح ═════════════════

// TestPH25_AdminRejectsOverlap **والتحقّقُ في المحرّك لا في المتصفّح.**
func TestPH25_AdminRejectsOverlap(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")

	bad := map[string]any{"windows": []any{
		map[string]any{"day_of_week": 0, "start": "09:00", "end": "14:00"},
		map[string]any{"day_of_week": 0, "start": "13:00", "end": "18:00"},
	}}
	if r := hh.Call("PUT", "/api/v1/admin/platform/hours", tok, bad, nil); r.Code < 400 {
		t.Fatalf("**جدولٌ متداخلٌ قُبل**: %d", r.Code)
	}
}

// TestPH26_AdminRejectsMalformed **وساعةٌ لا تُقرأ تُردّ.**
func TestPH26_AdminRejectsMalformed(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")

	for _, bad := range []map[string]any{
		{"day_of_week": 0, "start": "9", "end": "17:00"},
		{"day_of_week": 0, "start": "24:00", "end": "01:00"},
		{"day_of_week": 9, "start": "09:00", "end": "17:00"},
		{"day_of_week": 0, "start": "09:00", "end": "09:00"},
	} {
		body := map[string]any{"windows": []any{bad}}
		if r := hh.Call("PUT", "/api/v1/admin/platform/hours", tok, body, nil); r.Code < 400 {
			t.Errorf("**قُبل ما لا يُقبَل**: %v ⇒ %d", bad, r.Code)
		}
	}
}

// **والجدولُ يُكتب ويُقرأ كما كُتب** — **ودورةٌ كاملةٌ عبر الشبكة.**
func TestPH_AdminRoundTrip(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")

	body := map[string]any{"windows": []any{
		map[string]any{"day_of_week": 0, "start": "09:00", "end": "14:00"},
		map[string]any{"day_of_week": 0, "start": "17:00", "end": "01:00"},
		map[string]any{"day_of_week": 2, "start": "10:00", "end": "02:00"},
	}}
	if r := hh.Call("PUT", "/api/v1/admin/platform/hours", tok, body, nil); r.Code != http.StatusOK {
		t.Fatalf("**جدولٌ صالحٌ رُدّ**: %d / %s", r.Code, r.Err())
	}
	got := hh.GET("/api/v1/admin/platform/hours", tok).JSON()
	ws, _ := got["windows"].([]any)
	if len(ws) != 3 {
		t.Fatalf("**الجدولُ لم يُحفظ كما كُتب**: %v", got["windows"])
	}
	first, _ := ws[0].(map[string]any)
	if first["start"] != "09:00" || first["end"] != "14:00" {
		t.Fatalf("**فترةٌ عادت مبدَّلة**: %v", first)
	}
}

// **وموعدُ عودةٍ مضى يُردّ** — **وإيقافٌ ينتهي قبل أن يبدأ لا يوقف
// شيئاً، فيظنّ المالكُ الخدمةَ متوقّفةً وهي تعمل.**
func TestPH_AdminRejectsPastClosureEnd(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")

	body := map[string]any{
		"active":  true,
		"message": "صيانة",
		"ends_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
	}
	if r := hh.Call("PUT", "/api/v1/admin/platform/closure", tok, body, nil); r.Code < 400 {
		t.Fatalf("**موعدُ عودةٍ مضى قُبل**: %d", r.Code)
	}
}
