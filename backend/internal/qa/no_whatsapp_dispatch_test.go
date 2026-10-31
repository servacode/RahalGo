package qa

// ══════════════════════════════════════════════════════════════════════
// **الطلب ما بيعلق عند المكتب ولا عند الواتساب** — قرار المالك ٢٠٢٦-١٠-٣١
// ══════════════════════════════════════════════════════════════════════
//
// بوضع المنصّة: بعد `orders.no_whatsapp_dispatch_min` دقيقة من إنشاء الطلب، إذا ما
// حدا قبله أو انقبل وما انبعت للمتجر، **بينزل للسائقين لحاله بلا واتساب** — كل
// الأنواع. انظر `orders/no_whatsapp_dispatch.go`.

import (
	"testing"
)

// nwdNotePrefix **أوّل كلمات سطر السجل** — من `noWhatsAppDispatchNote`.
const nwdNotePrefix = "نزل للسائقين لحاله"

// nwdStandard **طلب عادي معلّق** — بوضع المنصّة وبإرسال المتجر مشتعل (الافتراض).
func nwdStandard(t *testing.T, h *Harness) string {
	t.Helper()
	treasury(t, h)
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.auto_transfer", "false")
	h.Setting("orders.notify_merchant", "true")
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.unattended_auto_accept_min", "0")
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("nwd"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if st := h.statusOf(oid); st != "pending" {
		t.Fatalf("الطلب الجديد حاله %s والمنتظر pending", st)
	}
	return oid
}

// nwdBackdate **بيرجّع وقت الإنشاء لورا** — شوي فوق المهلة لا كتير: الجولة بتاخد الأحدث
// أوّلاً وخمسين بس، وقاعدة الفحص المشتركة فيها طلبات معلّقة من فحوص تانية أقدم منه.
func nwdBackdate(t *testing.T, h *Harness, oid, ago string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET created_at = now() - $2::interval
		WHERE id = $1::uuid`, oid, ago); err != nil {
		t.Fatalf("تأخير وقت الإنشاء: %v", err)
	}
}

func nwdReachedDrivers(st string) bool { return st == "dispatching" || st == "assigned" }

// nwdNoteCount **كم سطر بالسجل فيه «نزل للسائقين لحاله»** — والمهلة مكتوبة فيه.
func nwdNoteCount(t *testing.T, h *Harness, oid string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events
		 WHERE order_id = $1::uuid AND to_status = 'dispatching'
		   AND note LIKE $2 || '%' AND note LIKE '%2 دقيقة%'`, oid, nwdNotePrefix).Scan(&n); err != nil {
		t.Fatalf("قراءة السجل: %v", err)
	}
	return n
}

// TestNWD01_StandardPendingReachesDrivers **عادي ما حدا قبله ⇒ بينزل للسائقين بجولة وحدة.**
func TestNWD01_StandardPendingReachesDrivers(t *testing.T) {
	h := New(t)
	oid := nwdStandard(t, h)
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	nwdBackdate(t, h, oid, "130 seconds")

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	if st := h.statusOf(oid); !nwdReachedDrivers(st) {
		t.Fatalf("**بعد ٣ دقايق بلا قبول الحال %s** — والمنتظر dispatching", st)
	}
	if n := nwdNoteCount(t, h, oid); n != 1 {
		t.Fatalf("سطر «نزل للسائقين لحاله» بالسجل %d مرّة والمنتظر مرّة", n)
	}
}

// TestNWD02_AcceptedNotSentReachesDrivers **مقبول وما انبعت للمتجر ⇒ بينزل** — والمبعوث ما بينلمس.
func TestNWD02_AcceptedNotSentReachesDrivers(t *testing.T) {
	h := New(t)
	notSent := nwdStandard(t, h)
	sent := nwdStandard(t, h)
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	for _, oid := range []string{notSent, sent} {
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE orders SET status = 'accepted', accepted_at = now(), driver_id = NULL,
			       created_at = now() - interval '130 seconds'
			 WHERE id = $1::uuid`, oid); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET sent_to_merchant_at = now() WHERE id = $1::uuid`, sent); err != nil {
		t.Fatal(err)
	}

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	if st := h.statusOf(notSent); !nwdReachedDrivers(st) {
		t.Fatalf("**مقبول وما انبعت من ٣ دقايق والحال %s** — والمنتظر dispatching", st)
	}
	if n := nwdNoteCount(t, h, notSent); n != 1 {
		t.Fatalf("سطر «نزل للسائقين لحاله» بالسجل %d مرّة والمنتظر مرّة", n)
	}
	if st := h.statusOf(sent); st != "accepted" {
		t.Fatalf("**طلب انبعت للمتجر انلمس** — الحال %s", st)
	}
}

// TestNWD03_CustomPendingReachesDrivers **الخاص ما إله «مقبول» — بينزل للطابور رأساً.**
func TestNWD03_CustomPendingReachesDrivers(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.unattended_auto_accept_min", "0")
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	got := h.POSTKey("/api/v1/orders/custom", h.Customer().Token, uniq("nwd-c"),
		customBody("كيلو بندورة وخبز من أيّ محلّ"))
	if got.Code >= 400 {
		t.Fatalf("إنشاء الخاص: %s", got)
	}
	oid, _ := got.JSON()["id"].(string)
	if st := h.statusOf(oid); st != "pending" {
		t.Fatalf("الخاص الجديد حاله %s والمنتظر pending", st)
	}
	nwdBackdate(t, h, oid, "130 seconds")

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	if st := h.statusOf(oid); !nwdReachedDrivers(st) {
		t.Fatalf("**الخاص ما نزل للسائقين بعد المهلة** — %s", st)
	}
	if n := nwdNoteCount(t, h, oid); n != 1 {
		t.Fatalf("سطر «نزل للسائقين لحاله» بالسجل %d مرّة والمنتظر مرّة", n)
	}
}

// TestNWD04_MerchantDeliveryReachesDrivers **«لدي توصيلة» بتنزل كمان** — حتى لو الإنزال
// بعد القبول مطفي (`orders.auto_dispatch`)، والأجرة بتنخصم متل القبول العادي.
func TestNWD04_MerchantDeliveryReachesDrivers(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقة NWD-04")
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.unattended_auto_accept_min", "0")
	h.Setting("orders.auto_dispatch", "false")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 1000)
	id := m.create(t, h, "merchant", uniq("nwd04"))
	if st := h.statusOf(id); st != "pending" {
		t.Fatalf("التوصيلة الجديدة حالها %s والمنتظر pending", st)
	}
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	nwdBackdate(t, h, id, "130 seconds")

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	if st := h.statusOf(id); !nwdReachedDrivers(st) {
		t.Fatalf("**التوصيلة ما نزلت للسائقين بعد المهلة** — %s", st)
	}
	if n := nwdNoteCount(t, h, id); n != 1 {
		t.Fatalf("سطر «نزل للسائقين لحاله» بالسجل %d مرّة والمنتظر مرّة", n)
	}
	if got := walletOf(t, h, owner); got != 500 {
		t.Fatalf("**القبول لحاله ما خصم الأجرة**: %d والمنتظر 500", got)
	}
}

// TestNWD05_MerchantDeliveryUnpaidIsFlagged **توصيلة رصيدها ما بيكفي بتضل معلّقة — والمكتب بينخبر مرّة.**
//
// رصيد المتجر بيكفي توصيلة وحدة وعنده تنتين معلّقات: الأولى بتنقبل وتنخصم، **والتانية
// بتضل معلّقة** وبينكتب سطر الإخبار مرّة وحدة ولو دارت الجولة مرّتين.
func TestNWD05_MerchantDeliveryUnpaidIsFlagged(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقة NWD-05")
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("orders.unattended_auto_accept_min", "0")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 500)
	a := m.create(t, h, "merchant", uniq("nwd05a"))
	b := m.create(t, h, "merchant", uniq("nwd05b"))
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	nwdBackdate(t, h, a, "130 seconds")
	nwdBackdate(t, h, b, "140 seconds")

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())
	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	// **الأحدث أوّلاً** — فـ a بتنقبل و b بتضل.
	if st := h.statusOf(a); !nwdReachedDrivers(st) {
		t.Fatalf("التوصيلة المدفوعة ما نزلت — %s", st)
	}
	if st := h.statusOf(b); st != "pending" {
		t.Fatalf("**توصيلة بلا أجرة نزلت** — الحال %s", st)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("رصيد المتجر %d والمنتظر 0", got)
	}
	var flags int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events
		 WHERE order_id = $1::uuid AND note LIKE 'محفظةُ المتجر لا تكفي%'`, b).Scan(&flags); err != nil {
		t.Fatal(err)
	}
	if flags != 1 {
		t.Fatalf("إخبار المكتب %d مرّة والمنتظر مرّة وحدة", flags)
	}
}

// TestNWD06_OffFreshAndSelfManageUntouched **صفر ⇒ ما بيتحرّك شي · طلب جديد ⇒ ما بينلمس ·
// وضع المتاجر ⇒ ما بينلمس.**
func TestNWD06_OffFreshAndSelfManageUntouched(t *testing.T) {
	h := New(t)
	old := nwdStandard(t, h)
	fresh := nwdStandard(t, h)
	nwdBackdate(t, h, old, "130 seconds")

	// صفر يعني «لا».
	h.Setting("orders.no_whatsapp_dispatch_min", "0")
	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())
	if st := h.statusOf(old); st != "pending" {
		t.Fatalf("**الإعداد صفر والطلب تحرّك** — %s", st)
	}

	// وضع المتاجر — المتجر بيدير طلباته.
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	h.Setting("platform.orders_mode", `"merchants"`)
	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())
	if st := h.statusOf(old); st != "pending" {
		t.Fatalf("**بوضع المتاجر الطلب تحرّك** — %s", st)
	}

	// ووضع المنصّة: القديم بينزل والجديد بيضل.
	h.Setting("platform.orders_mode", `"platform"`)
	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())
	if st := h.statusOf(fresh); st != "pending" {
		t.Fatalf("**طلب عمره ثواني تحرّك** — %s", st)
	}
	if st := h.statusOf(old); !nwdReachedDrivers(st) {
		t.Fatalf("الطلب القديم بوضع المنصّة ما نزل — %s", st)
	}
}

// TestNWD07_OldForgottenOrderUntouched **الطلب الأقدم من مهلة التوصيل ما بينزل لحاله** — طلبٌ منسيٌّ
// من ساعات (أو موجود يوم ينرفع التعديل) بيضل بيد المكتب، ولا يُبعث سائقٌ لطلبٍ ما حدا ناطره.
func TestNWD07_OldForgottenOrderUntouched(t *testing.T) {
	h := New(t)
	old := nwdStandard(t, h)
	fresh := nwdStandard(t, h)
	h.Setting("orders.no_whatsapp_dispatch_min", "2")
	h.Setting("orders.delivery_timeout_min", "60")
	nwdBackdate(t, h, old, "3 hours")
	nwdBackdate(t, h, fresh, "130 seconds")

	h.Orders.SweepNoWhatsAppDispatchOnce(ctxBG())

	if st := h.statusOf(old); st != "pending" {
		t.Fatalf("**طلب عمره ٣ ساعات صار %s** — والمنتظر يضل pending", st)
	}
	if st := h.statusOf(fresh); !nwdReachedDrivers(st) {
		t.Fatalf("الطلب الجديد حاله %s والمنتظر ينزل للسائقين", st)
	}
}
