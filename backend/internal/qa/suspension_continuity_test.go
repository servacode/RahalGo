package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-22` — التعليقُ العاديُّ يشلّ إتمامَ طلبٍ قائم**
// ══════════════════════════════════════════════════════════════════════
//
// # ما هو مقيسٌ اليوم
//
// **`RequireAuth` يردّ `403` على كلّ نداءٍ مصادَقٍ فورَ التعليق**
// (`middleware.go:40`: `ActiveStatus != "active"`). **فسائقٌ عُلِّق وهو
// يحمل طلباً لا يستطيع تسليمَه ولا رؤيتَه** — **والطلبُ يبقى معلَّقاً
// بمن لا يقدر.**
//
// # والحالتان في المنتَج ليستا واحدة
//
// **`suspended` و`blocked` حالان مختلفتان** يقبلهما
// `AdminUpdateUser` — **والعاديُّ هو `suspended`.** **و`blocked` هو
// البابُ الاستثنائيُّ** الذي يوقف كلَّ شيءٍ ولا استثناءَ فيه.
//
// # والاستثناءُ ضيّقٌ لا عامّ
//
// **ولا يُقال «معلَّقٌ ومعه طلبٌ فليعمل ما شاء»** — **الإذنُ لفعلٍ
// لازمٍ على طلبٍ بعينه، لا لفتح التطبيق.**

// suspend يعلّق حساباً بالحال المطلوبة.
func suspend(t *testing.T, h *Harness, userID, status string) {
	t.Helper()
	admin := h.NewUser("admin")
	got := h.PATCH("/api/v1/admin/users/"+userID, admin.Token,
		map[string]any{"status": status, "status_reason": "XG-22"})
	if got.Code >= 400 {
		t.Fatalf("تعليقُ الحساب (%s): %s", status, got)
	}
	// **والذاكرةُ القصيرةُ تُبطَل من المسار نفسِه** — فلا انتظار.
}

// activeOrderFor طلبٌ مُسنَدٌ إلى سائقٍ وجاهزٌ للتقدّم.
func activeOrderFor(t *testing.T, h *Harness) (orderID string, drv *User) {
	t.Helper()
	treasury(t, h)
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	orderID, _ = made.JSON()["id"].(string)
	return orderID, h.driverOf(orderID)
}

// ══════════════════════════════════════════════════════════════════════
// **T1 · معلَّقٌ بلا طلبٍ قائم ⇒ يُمنَع**
// ══════════════════════════════════════════════════════════════════════
func TestXG22_T1_SuspendedWithoutActiveOrderIsDenied(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)

	suspend(t, h, cust.ID, "suspended")
	got := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	t.Logf("معلَّقٌ بلا طلبٍ يُنشئ طلباً ⇒ %d", got.Code)
	if got.Code < 400 {
		t.Errorf("**معلَّقٌ أنشأ طلباً** (%d) — **والتعليقُ يمنع نشاطاً جديداً**",
			got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T2 · عُلِّق وهو يحمل طلباً ⇒ يُتمّه**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو `XG-22` بعينه.**
func TestXG22_T2_SuspendedDriverCanFinishActiveOrder(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)

	suspend(t, h, drv.ID, "suspended")
	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"})
	t.Logf("سائقٌ عُلِّق وهو يحمل طلباً ⇒ الانتقال %d %s", got.Code, got.Err())

	if got.Code == 403 {
		t.Errorf("**السائقُ المعلَّقُ لا يستطيع إتمامَ طلبِه** (403) — " +
			"**والطلبُ يبقى معلَّقاً بمن لا يقدر.** (`XG-22`)")
	}
	if got.Code >= 400 {
		t.Errorf("الانتقالُ رُدّ (%d) — والفعلُ لازمٌ لإتمام طلبٍ قائم", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T3 · ولا يمتدّ الإذنُ إلى طلبٍ آخر**
// ══════════════════════════════════════════════════════════════════════
func TestXG22_T3_ExceptionDoesNotLeakToAnotherOrder(t *testing.T) {
	h := New(t)
	mine, drv := activeOrderFor(t, h)
	other, _ := activeOrderFor(t, h)

	suspend(t, h, drv.ID, "suspended")
	got := h.POST("/api/v1/driver/orders/"+other+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"})
	t.Logf("طلبي=%s · طلبٌ آخر=%s ⇒ %d", first8(mine), first8(other), got.Code)
	if got.Code < 400 {
		t.Errorf("**الإذنُ امتدّ إلى طلبٍ ليس له** (%d)", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T4+T5 · ينتهي الإذنُ ببلوغ الحال النهائيّة**
// ══════════════════════════════════════════════════════════════════════
func TestXG22_T4_ExceptionEndsAtTerminalState(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	suspend(t, h, drv.ID, "suspended")

	walkToDropoff(t, h, oid, drv)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليمُ رُدّ: %s", got)
	}
	t.Log("سُلّم الطلبُ والسائقُ معلَّق")

	// **وبعد النهاية لا إذن.**
	after := h.GET("/api/v1/driver/orders", drv.Token)
	t.Logf("بعد التسليم: قائمةُ الطلبات ⇒ %d", after.Code)
	if after.Code < 400 {
		t.Errorf("**المعلَّقُ يعمل بعد انتهاء طلبِه** (%d) — "+
			"**والاستثناءُ ينتهي بانتهاء الطلب.**", after.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T6 · وغيرُ المعلَّق كما كان**
// ══════════════════════════════════════════════════════════════════════
func TestXG22_T6_NormalActorUnchanged(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"})
	list := h.GET("/api/v1/driver/orders", drv.Token)
	t.Logf("سائقٌ نشط: انتقالٌ %d · قائمةٌ %d", got.Code, list.Code)
	if got.Code >= 400 || list.Code >= 400 {
		t.Errorf("**سلوكُ النشط تبدّل**: انتقالٌ %d · قائمةٌ %d", got.Code, list.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T8 · و`blocked` بابٌ استثنائيٌّ لا استثناءَ فيه**
// ══════════════════════════════════════════════════════════════════════
//
// **والحالان في المنتَج أصلاً** — **ولا يُخترَع نظامٌ ثانٍ.**
func TestXG22_T8_BlockedHasNoException(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)

	suspend(t, h, drv.ID, "blocked")
	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"})
	t.Logf("محظورٌ يحمل طلباً ⇒ %d", got.Code)
	if got.Code < 400 {
		t.Errorf("**المحظورُ عمل** (%d) — **والحظرُ بابُ الأمن ولا استثناءَ فيه**",
			got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T9 · ولا يتسرّب الإذنُ بين الأدوار**
// ══════════════════════════════════════════════════════════════════════
func TestXG22_T9_ExceptionDoesNotLeakAcrossRoles(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	suspend(t, h, drv.ID, "suspended")

	// **مسارُ الإدارة ليس من مسارات الاستمرار** — ولو حمل الطلبَ نفسَه.
	admin := h.POST("/api/v1/admin/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "cancelled", "note": "XG-22"})
	me := h.GET("/api/v1/me", drv.Token)
	t.Logf("معلَّقٌ ينادي مسارَ الإدارة ⇒ %d · وملفَّه ⇒ %d", admin.Code, me.Code)
	if admin.Code < 400 {
		t.Errorf("**الإذنُ تسرّب إلى مسارٍ إداريّ** (%d)", admin.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T10 · والإنفاذُ في الخادم لا في الواجهة**
// ══════════════════════════════════════════════════════════════════════
//
// **ونداءٌ مباشرٌ بلا واجهةٍ يُثبته** — وكلُّ ما سبق نداءاتٌ مباشرة.
func TestXG22_T10_EnforcementIsServerSide(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	suspend(t, h, cust.ID, "suspended")

	// **نداءٌ مباشرٌ لمسارٍ ليس من الاستمرار.**
	direct := h.GET("/api/v1/orders", cust.Token)
	create := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	t.Logf("نداءٌ مباشر: قائمةٌ %d · إنشاءٌ %d", direct.Code, create.Code)
	if create.Code < 400 {
		t.Errorf("**إنشاءٌ جديدٌ من معلَّقٍ بنداءٍ مباشر** (%d)", create.Code)
	}
}

var _ = context.Background

// ══════════════════════════════════════════════════════════════════════
// **T5 · والزبونُ الموقوفُ يرى طلبَه الحيَّ ويُلغيه** — `CAF-04`
// ══════════════════════════════════════════════════════════════════════
//
// **كان استثناءُ الرؤية يشير إلى `GET /api/v1/orders/{id}` ولا مسارَ
// زبونيٌّ بهذا الاسم** — **فيبقى الطلبُ غيرَ مرئيٍّ لصاحبه الموقوف
// (المسارُ الحقيقيُّ `GET /api/v1/my/orders/{id}` كان محجوباً).**
// **والإصلاحُ: الاستثناءُ على المسار الحقيقيّ، بحارس `isLiveParticipant`.**
func TestXG22_T5_SuspendedCustomerSeesAndCancelsOwnLiveOrder(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	suspend(t, h, cust.ID, "suspended")

	// **يرى طلبَه الحيَّ عبر المسار الحقيقيّ** — لا يُحجب.
	see := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if see.Code >= 400 {
		t.Errorf("**الموقوفُ لا يرى طلبَه الحيّ** (%d) — CAF-04 لم يُصلَح", see.Code)
	}
	// **ولا تُفتح له القائمةُ العامّة** — نشاطٌ جديدٌ محجوب.
	list := h.GET("/api/v1/my/orders", cust.Token)
	if list.Code < 400 {
		t.Errorf("**القائمةُ العامّةُ مفتوحةٌ لموقوف** (%d) — تسرّبٌ", list.Code)
	}
	// **ويُلغي طلبَه الحيّ** — الاستثناءُ الثاني.
	cancel := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token,
		map[string]any{"reason": "XG-22 T5"})
	if cancel.Code >= 400 {
		t.Errorf("**الموقوفُ لا يُلغي طلبَه الحيّ** (%d)", cancel.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T7 · والإدارةُ تحلّ الطلبَ كما كانت**
// ══════════════════════════════════════════════════════════════════════
//
// **وتعليقُ السائق لا يسلب الإدارةَ سلطتَها على طلبِه** — **فإن لم
// يُتمّه أحدٌ أتمّته هي.**
func TestXG22_T7_OpsCanStillResolveTheOrder(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	suspend(t, h, drv.ID, "suspended")

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "cancelled", "note": "حلٌّ إداريّ"})
	t.Logf("الإدارةُ تُلغي طلبَ سائقٍ معلَّق ⇒ %d %s", got.Code, got.Err())
	if got.Code >= 400 {
		t.Errorf("**الإدارةُ عجزت عن حلّ الطلب** (%d)", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وتعليقٌ يقع مع انتقالٍ — نتيجةٌ حتميّةٌ لا حالٌ فاسدة**
// ══════════════════════════════════════════════════════════════════════
//
// **تداخلٌ حقيقيٌّ مقيس.** **والمقبولُ أحدُ أمرين**: الانتقالُ ثبت قبل
// التعليق، أو التعليقُ سبقه فرُدّ. **والمرفوضُ حالٌ لا يصفها عقد.**
func TestXG22_SuspendDuringTransitionIsDeterministic(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	admin := h.NewUser("admin")

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "انتقالٌ", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
				map[string]any{"to": "at_pickup"})
		}},
		Actor{Name: "تعليقٌ", Do: func(ctx context.Context) any {
			return h.PATCH("/api/v1/admin/users/"+drv.ID, admin.Token,
				map[string]any{"status": "suspended", "status_reason": "XG-22"})
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	var status string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&status)
	t.Logf("تعليقٌ مع انتقال: حالُ الطلب %q · تداخلٌ مقيسٌ %d — %s",
		status, r.Probe.Max(), r)

	if status != "assigned" && status != "at_pickup" {
		t.Errorf("**حالٌ لا يصفها عقد**: %q — والمقبولُ `assigned` أو `at_pickup`",
			status)
	}
	// **وبعده يبقى الاستمرارُ ممكناً** — **ولا يُترَك الطلبُ عالقاً.**
	after := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"})
	t.Logf("وبعد السباق: الاستمرارُ ⇒ %d", after.Code)
	if after.Code == 403 {
		t.Error("**الطلبُ عَلِق بلا مسارِ استمرار** بعد تعليقٍ متزامن")
	}
}
