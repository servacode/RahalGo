package server

// إشعاراتُ الطلب الخاصّ — **من عليه الدورُ يُخبَر أنّ الدورَ عليه.**
//
// # لماذا لزمت
//
// **للطلب الخاصِّ مفصلان ينتظران فعلَ طرفٍ بعينه:**
//
//	السائقُ يوثّق السعر   ⇐  **الزبونُ يؤكّد**   — والطلبُ واقفٌ حتّى يفعل
//	الزبونُ يؤكّد          ⇐  **السائقُ يشتري**  — والطلبُ واقفٌ حتّى يفعل
//
// **وكلاهما كان بلا إشعارٍ البتّة** (قُيس ٢٠٢٦-٠٩-٢٩ على دورةٍ حيّةٍ كاملة):
// `AgreeCustom` لا تنادي إشعاراً، و`ConfirmQuote` تنادي البثَّ وحدَه —
// **والبثُّ يصل الشاشاتِ المفتوحةَ فقط** (علّةُ `watchdog.go`: «فمن أغلق
// اللوحة ليلاً لم يصله شيء»).
//
// **وبلاغُ المالك نصّاً** (٢٠٢٦-٠٩-٢٩): «لازم يجي تنبيه للزبون مشان تأكيد
// الاتفاق والدفع… مشان ما يفكّر حاله إنه طلب وهو يستنّى بدون ما يحصل شي».
//
// **وثالثةٌ**: `assigned` ليست في عناوين الزبون، **وللخاصِّ لا خطوةَ قبولِ
// متجر** — فأوّلُ إشارةٍ كانت تصله «في الطريق»، **بعد أن اشترى السائقُ
// فعلاً بماله.**
//
// # وما ليس ثغرةً — يُكتب كي لا يُعاد اكتشافُه خطأً
//
// `on_the_way` و`delivered` **تُشعران فعلاً**، لكنّهما **عابرتان**
// (`passingTitles`): ترنّ ولا تُحفَظ صفّاً — **بقرار المالك ٢٠٢٦-٠٨-١٢.**
// **فعدُّ صفوف `notifications` يقيس الشيءَ الخطأ فيهما.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// armCustomNotify عُدّةٌ تُشعِر فعلاً — المخزنُ والناقلُ وخدمةُ الإشعارات.
func armCustomNotify(t *testing.T, f *driverFixture) {
	t.Helper()
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	f.srv.orders.SetNotifier(notifications.New(f.pool, f.srv.hub, f.srv.logger))
}

// customerOf صاحبُ الطلب — تُقرأ من الصفّ لا تُخمَّن.
func (f *driverFixture) customerOf(t *testing.T, orderID string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT customer_id FROM orders WHERE id = $1`, orderID).Scan(&id); err != nil {
		t.Fatalf("تعذّرت قراءةُ صاحب الطلب: %v", err)
	}
	return id
}

// notifsFor عددُ الإشعارات الباقيةِ لشخصٍ في طلبٍ بعينه.
func (f *driverFixture) notifsFor(t *testing.T, userID, orderID string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM notifications
		WHERE user_id = $1 AND entity = 'order' AND entity_id = $2`,
		userID, orderID).Scan(&n); err != nil {
		t.Fatalf("تعذّر عدُّ الإشعارات: %v", err)
	}
	return n
}

// ── أ · السائقُ يوثّق السعر ⇒ الزبونُ يُخبَر ليؤكّد ─────────────────────────
//
// **وهو الأهمّ**: الطلبُ بعدها لا يتقدّم إلّا بفعل الزبون.
func TestCustomNotify_QuoteReachesCustomer(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	armCustomNotify(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	cust := f.customerOf(t, ord)
	f.offer(t, ord, nil)
	if w := f.accept(d, ord); w.Code != 200 {
		t.Fatalf("القبول ردّ %d", w.Code)
	}
	before := f.notifsFor(t, cust, ord)

	if err := f.srv.orders.AgreeCustom(context.Background(), ord, d, 12000, 5000); err != nil {
		t.Fatalf("تعذّر توثيقُ الاتّفاق: %v", err)
	}

	if after := f.notifsFor(t, cust, ord); after == before {
		t.Fatalf("وُثّق السعرُ ولم يُخبَر الزبون — الإشعاراتُ %d قبلَه و%d بعدَه. "+
			"**والطلبُ واقفٌ على فعلِه.**", before, after)
	}
}

// ── ب · الزبونُ يؤكّد ⇒ السائقُ يُخبَر ليشتري ───────────────────────────────
func TestCustomNotify_ConfirmReachesDriver(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	armCustomNotify(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	cust := f.customerOf(t, ord)
	f.offer(t, ord, nil)
	if w := f.accept(d, ord); w.Code != 200 {
		t.Fatalf("القبول ردّ %d", w.Code)
	}
	if err := f.srv.orders.AgreeCustom(context.Background(), ord, d, 12000, 5000); err != nil {
		t.Fatalf("تعذّر توثيقُ الاتّفاق: %v", err)
	}
	before := f.notifsFor(t, d, ord)

	// **والمبلغُ والنسخةُ يُمرَّران كما يمرّرهما التطبيق** — تأكيدٌ على ما رآه
	// لا على ما صار: **سعرٌ تبدّل بين العرض والضغط لا يمرّ صامتاً.**
	if _, err := f.srv.orders.ConfirmQuote(
		context.Background(), ord, cust, "cash", 17000, 1); err != nil {
		t.Fatalf("تعذّر تأكيدُ الزبون: %v", err)
	}

	if after := f.notifsFor(t, d, ord); after == before {
		t.Fatalf("أكّد الزبونُ ولم يُخبَر السائق — الإشعاراتُ %d قبلَه و%d بعدَه. "+
			"**والشراءُ واقفٌ على فعلِه.**", before, after)
	}
}

// ── ج · وإسنادُ السائق يصل الزبون ──────────────────────────────────────────
//
// **وللخاصِّ لا قبولَ متجرٍ قبلَه** — فبدونه لا يعلم الزبونُ بشيءٍ حتّى
// «في الطريق»، **وقد اشترى السائقُ بماله قبلها.**
func TestCustomNotify_AssignmentReachesCustomer(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	armCustomNotify(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	cust := f.customerOf(t, ord)
	before := f.notifsFor(t, cust, ord)

	f.offer(t, ord, nil)
	if w := f.accept(d, ord); w.Code != 200 {
		t.Fatalf("القبول ردّ %d", w.Code)
	}

	if after := f.notifsFor(t, cust, ord); after == before {
		t.Fatalf("أُسند سائقٌ ولم يُخبَر الزبون — %d قبلَه و%d بعدَه", before, after)
	}
}
