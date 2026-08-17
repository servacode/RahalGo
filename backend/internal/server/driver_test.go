package server

// اختبارات انحدار لبوابة السائق.
//
// وهي أوّل اختبارات في حزمة `server`: كان كل ما اختُبر حتى الآن في طبقة الخدمات،
// لأن المعالِجات كانت تمريراً. ومعالِجات السائق ليست كذلك — فيها **قرارات لا
// يعرفها المحرّك**: الدوام، وسقف النقد، وذرّية الأخذ من طابورٍ مشترك. وهذه لو
// انكسرت لانكسرت صامتةً: طلبٌ يأخذه سائقان فيصل مرّتين أو لا يصل.
//
// وتُبنى `Server` هنا بحقولها التي تلمسها هذه المعالِجات فقط — لا ضرورة
// للتهيئة الكاملة، فالاختبار يقيس ما يُختبر.
//
// وهذا ثمنُ البناء الجزئيّ: حقلٌ جديد يستعمله معالِجٌ مُختبَر يُسقط الاختبار
// بمؤشّرٍ فارغ لا برسالةٍ مفهومة. وقد وقع فعلاً حين صار الأخذ يقرأ سقف
// الطلبات من الإعدادات. والسقوط أفضل من مرورٍ كاذب — لكنّ الرسالة رديئة،
// فوجب أن تُقرأ هذه الملاحظة قبل الحيرة في «مؤشّر فارغ في السطر ٢٢».

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type driverFixture struct {
	pool       *pgxpool.Pool
	srv        *Server
	merchantID string
	drivers    []string
}

func newDriverFixture(t *testing.T, driverCount int) *driverFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	walletSvc := wallet.NewService(pool)
	settingsStore := settings.NewStore(pool)
	cashboxSvc := cashbox.NewService(pool, settingsStore)
	ident := identity.NewService(identity.NewRepo(pool), nil, nil, nil, "", quiet)
	f := &driverFixture{
		pool: pool,
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      realtime.NewHub(quiet),
			cashbox:  cashboxSvc,
			settings: settingsStore,
			// **والمحفظةُ مركَّبةٌ في العُدّة** — كانت تُنشأ ولا تُسنَد،
			// **فأيُّ اختبارٍ يمسّ المال ينهار بمؤشّرٍ فارغ.**
			wallet: walletSvc,
			orders: orders.NewService(pool, nil, walletSvc, cashboxSvc, nil, quiet),
			// **والفهرسُ مركَّبٌ أيضاً** — **وتحويلُ طلبِ الانضمام يمرّ به**
			// (يُنشئ المتجرَ ويمنح صاحبَه دورَه)، **فأيُّ فحصٍ يمسّه ينهار
			// بمؤشّرٍ فارغ** — وهي علّةُ المحفظة نفسُها قبله.
			// **والهويّةُ تُبنى مرّةً وتُمرَّر للاثنين** — **ونسختان
			// منها في فحصٍ واحدٍ تُخفيان أنّ الخادمَ يحملها أصلاً**،
			// فيسقط كلُّ مسارٍ يقرؤها بمؤشّرٍ فارغ.
			identity: ident,
			catalog:  catalog.NewService(pool, ident),
		},
	}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجر اختبار السائق', $1, 10) RETURNING id`, categoryID).Scan(&f.merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, f.merchantID)
	})

	for i := 0; i < driverCount; i++ {
		f.drivers = append(f.drivers, testdb.NewUser(t, pool, "driver"))
	}
	return f
}

// setSetting يضبط إعداداً للاختبار — عبر المخزن كي يمرّ بتحقّق الكتالوج نفسه.
func (f *driverFixture) setSetting(t *testing.T, key string, v any) {
	t.Helper()
	if err := settings.NewStore(f.pool).SetInternal(context.Background(), key, v); err != nil {
		t.Fatalf("تعذّر ضبط %s: %v", key, err)
	}
}

// onShift يرفع علَم دوام السائق — الحالة التي يفترضها الطابور.
func (f *driverFixture) onShift(t *testing.T, driverID string, on bool) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET on_shift = $2 WHERE id = $1`, driverID, on); err != nil {
		t.Fatalf("تعذّر ضبط الدوام: %v", err)
	}
}

// dispatchingOrder طلبٌ نقديّ في الطابور بلا سائق — هذا ما يتنازع عليه السائقون.
func (f *driverFixture) dispatchingOrder(t *testing.T, subtotal, deliveryFee int64) string {
	t.Helper()
	customer := testdb.NewUser(t, f.pool, "customer")
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, 'dispatching', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $3, $4, $5, 0, $5)
		RETURNING id`, customer, f.merchantID, subtotal, deliveryFee, subtotal+deliveryFee).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return id
}

// accept ينادي المعالِج الحقيقي كما يناديه المسار: بمعرف مسار وسياق مستخدم.
func (f *driverFixture) accept(driverID, orderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/driver/orders/"+orderID+"/accept", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})

	w := httptest.NewRecorder()
	f.srv.handleDriverAccept(w, req.WithContext(ctx))
	return w
}

func errCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("ردٌّ غير صالح: %s", w.Body.String())
	}
	return body.Error.Code
}

func (f *driverFixture) assignedDriver(t *testing.T, orderID string) *string {
	t.Helper()
	var id *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT driver_id FROM orders WHERE id = $1`, orderID).Scan(&id); err != nil {
		t.Fatalf("تعذّرت قراءة الطلب: %v", err)
	}
	return id
}

// TestAccept_DriversRace طلبٌ واحد وثمانية سائقين يضغطون معاً — واحدٌ يفوز لا أكثر.
//
// هذا هو الخلل الذي يقع في الإنتاج ولا يظهر في اختبارٍ متسلسل: لو فُحص
// `driver_id IS NULL` في استعلامٍ ثم حُدّث في آخر، لمرّ سائقان من الفحص معاً
// قبل أن يكتب أيٌّ منهما. والنتيجة طلبٌ يظنّ اثنان أنه لهما.
//
// **ولماذا ثمانيةٌ وعشر جولات ولا يكفي اثنان؟** جرّبناه باثنين على نسخةٍ مكسورة
// عمداً (فحصٌ ثم تحديث) فمرّ الاختبار: النافذة بين الفحص والكتابة أضيق من أن
// يُصادفها اثنان. والسباق الذي يقع مرّةً في الألف يقع في الإنتاج ولا يقع في
// اختبار يُشغَّل مرّة. فالتوسيع هنا ليس مبالغةً — هو ما يجعل الاختبار اختباراً.
const raceDrivers, raceRounds = 8, 10

func TestAccept_DriversRace(t *testing.T) {
	f := newDriverFixture(t, raceDrivers)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}
	// **سقفُ الطلبات يُرفع هنا عمداً.**
	//
	// الفائز يحتفظ بطلبه في الجولات التالية، فمن فاز مرّتين يبلغ السقف
	// الافتراضي (٢) ويُمنع في الثالثة — فتسقط جولةٌ بلا فائز ويفشل الاختبار
	// **عشوائياً حسب من فاز**. واختبارٌ يفشل أحياناً يُدرَّب فريقُه على إعادة
	// تشغيله بدل قراءته، فيصير أسوأ من لا اختبار.
	//
	// وهذا الاختبار يقيس **ذرّية الأخذ** لا السقف — فيُرفع السقف عن طريقه،
	// وللسقف اختبارُه المستقلّ أدناه.
	f.setSetting(t, "drivers.max_active_orders", raceRounds+1)

	for round := 0; round < raceRounds; round++ {
		orderID := f.dispatchingOrder(t, 48000, 10000)

		var wg sync.WaitGroup
		codes := make([]int, len(f.drivers))
		start := make(chan struct{})
		for i, d := range f.drivers {
			wg.Add(1)
			go func(i int, driverID string) {
				defer wg.Done()
				<-start // انطلاقةٌ واحدة: بلا هذا يسبق الأول لأنه بدأ أولاً
				codes[i] = f.accept(driverID, orderID).Code
			}(i, d)
		}
		close(start)
		wg.Wait()

		won := 0
		for _, c := range codes {
			if c == http.StatusOK {
				won++
			}
		}
		if won != 1 {
			t.Fatalf("الجولة %d: توقّعنا فائزاً واحداً، والنتيجة %d (الرموز %v)",
				round, won, codes)
		}
		// والفائز يخرج بالطلب فعلاً: لو تدافع اثنان لأسقط التراجعُ الإسنادَ كلَّه
		// فبقي الطلب بلا سائق — ونجاحٌ بلا مُسنَدٍ إليه ليس نجاحاً.
		if f.assignedDriver(t, orderID) == nil {
			t.Fatalf("الجولة %d: الطلب بلا سائق بعد قبولٍ ناجح", round)
		}
	}
}

// TestAccept_RejectsOffShift الدوام شرطُ الأخذ — لا يُسنَد طلبٌ لمن أعلن انصرافه.
func TestAccept_RejectsOffShift(t *testing.T) {
	f := newDriverFixture(t, 1)
	orderID := f.dispatchingOrder(t, 48000, 10000)
	f.onShift(t, f.drivers[0], false)

	w := f.accept(f.drivers[0], orderID)
	if code := errCode(t, w); code != "not_on_shift" {
		t.Fatalf("توقّعنا not_on_shift، والنتيجة %q (حالة %d)", code, w.Code)
	}
	if d := f.assignedDriver(t, orderID); d != nil {
		t.Fatalf("أُسند الطلب رغم الرفض: %s", *d)
	}
}

// TestAccept_RejectsWhenCashLimitWouldBreak السقف يُفحص **قبل** الأخذ لا عنده.
//
// ولو فُحص عند التسليم لحمل السائق طلباً يعجز عن إقفاله: النقد بيده والصندوق
// يرفض قيده. فالرفض عند الباب أرحم.
func TestAccept_RejectsWhenCashLimitWouldBreak(t *testing.T) {
	f := newDriverFixture(t, 1)
	driverID := f.drivers[0]
	f.onShift(t, driverID, true)

	// **والسقفُ من المخزن لا من استعلامٍ يكتب افتراضَه بيده.**
	//
	// اختبارٌ يحمل نسخةً من الافتراض **يمرّ وهو يفحص رقماً غيرَ الذي يعمل به
	// النظام** — فيُصدَّق وهو يكذب.
	limit := f.srv.settings.GetInt(context.Background(), "drivers.cash_limit")

	// بحوزته ما يملأ السقف إلا قليلاً، والطلب أكبر من ذلك القليل
	held := limit - 5000
	if err := f.srv.cashbox.Collect(context.Background(), driverID, held, "", &driverID); err != nil {
		t.Fatalf("تعذّر شحن الصندوق: %v", err)
	}

	orderID := f.dispatchingOrder(t, 48000, 10000)
	w := f.accept(driverID, orderID)
	if code := errCode(t, w); code != "cash_limit_reached" {
		t.Fatalf("توقّعنا cash_limit_reached، والنتيجة %q (حالة %d)", code, w.Code)
	}
	if d := f.assignedDriver(t, orderID); d != nil {
		t.Fatalf("أُسند الطلب رغم تجاوز السقف: %s", *d)
	}

	// وطلبٌ يسع تحت السقف يُقبل — كي لا يمرّ الاختبار لأن كل شيءٍ مرفوض
	small := f.dispatchingOrder(t, 3000, 1000)
	if w := f.accept(driverID, small); w.Code != http.StatusOK {
		t.Fatalf("رُفض طلبٌ يسع تحت السقف: %d — %s", w.Code, w.Body.String())
	}
}

// TestAccept_RejectsOrderAlreadyTaken الطابور معروضٌ للجميع، والمأخوذ ليس فيه.
func TestAccept_RejectsOrderAlreadyTaken(t *testing.T) {
	f := newDriverFixture(t, 2)
	orderID := f.dispatchingOrder(t, 48000, 10000)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	if w := f.accept(f.drivers[0], orderID); w.Code != http.StatusOK {
		t.Fatalf("فشل القبول الأول: %d — %s", w.Code, w.Body.String())
	}
	w := f.accept(f.drivers[1], orderID)
	if code := errCode(t, w); code != "order_taken" {
		t.Fatalf("توقّعنا order_taken، والنتيجة %q (حالة %d)", code, w.Code)
	}
}

// TestAccept_RejectsWhenTooManyActive سقفُ ما بيد السائق من طلبات.
//
// كان يأخذ ما شاء ما دام سقفه النقدي يتّسع — والسقف النقدي لا يمنع تكديس
// الطلبات الصغيرة. وخمسةُ طلبات بيد سائقٍ واحد تعني أربعة زبائن ينتظرون ساعة.
func TestAccept_RejectsWhenTooManyActive(t *testing.T) {
	f := newDriverFixture(t, 1)
	driverID := f.drivers[0]
	f.onShift(t, driverID, true)
	f.setSetting(t, "drivers.max_active_orders", 2)

	for i := 0; i < 2; i++ {
		id := f.dispatchingOrder(t, 3000, 1000)
		if w := f.accept(driverID, id); w.Code != http.StatusOK {
			t.Fatalf("رُفض الطلب %d وهو دون السقف: %d — %s", i+1, w.Code, w.Body.String())
		}
	}

	third := f.dispatchingOrder(t, 3000, 1000)
	w := f.accept(driverID, third)
	if code := errCode(t, w); code != "too_many_active_orders" {
		t.Fatalf("توقّعنا too_many_active_orders، والنتيجة %q (حالة %d)", code, w.Code)
	}
	if d := f.assignedDriver(t, third); d != nil {
		t.Fatalf("أُسند الطلب رغم بلوغ السقف: %s", *d)
	}
}
