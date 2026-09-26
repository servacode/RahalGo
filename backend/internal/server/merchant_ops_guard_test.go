package server

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ بوّابة المتجر — الملكيّةُ والإيقافُ والتزامنُ** (A6)
// ══════════════════════════════════════════════════════════════════════
//
// **قرارُ المالك (دفعة المتجر)**: تُسَدّ ثغراتُ الاختبار قبل إغلاق المرحلة —
// **قبولٌ مزدوج · قبولٌ مقابل رفض · قبولٌ مقابل مهلة · جاهزيّةٌ مرّتين ·
// تسلّلٌ على طلبٍ/صنفٍ/قسمٍ لا يملكه · عزلُ مالكٍ متعدّدِ المتاجر · رفضُ
// الكتابة على متجرٍ موقوف.**
//
// **ولا مُنسّقَ يُلَفّ حولها** — يُتّكأ على آلة الحالات وقفلِ `FOR UPDATE`
// القائمَين: القبولُ والرفضُ يمرّان بـ`orders.Transition` الذي يقرأ الحالةَ
// داخلَ المعاملةِ مقفولةً، **فالخاسرُ يرى الحالةَ وقد تبدّلت فيُردّ.**
//
// **وتُبنى `Server` بحقولها التي تلمسها هذه المعالِجات فقط** — كاختبارات
// السائق. **والفرقُ الوحيد**: `orders.SetSettings` **يُنادى هنا** كي يقرأ
// المحرّكُ وضعَ «المتجر يدير» فيُبقيَ للمتجر حقَّ القبول.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type merchantFixture struct {
	pool *pgxpool.Pool
	srv  *Server
	cat  string
	psec string // قسمُ سوقٍ فعّال — `menu_items.platform_section_id` إلزاميّ
}

func newMerchantFixture(t *testing.T) *merchantFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	walletSvc := wallet.NewService(pool)
	settingsStore := settings.NewStore(pool)
	cashboxSvc := cashbox.NewService(pool, settingsStore)
	ident := identity.NewService(identity.NewRepo(pool), nil, nil, nil, "", quiet)
	ordersSvc := orders.NewService(pool, nil, walletSvc, cashboxSvc, nil, quiet)
	// **وهنا وحدَه يُحقَن مخزنُ الإعدادات في المحرّك** — بلا هذا يُفترض
	// «المتجر يدير» صامتاً في بعض المسارات، **ونريده صريحاً في القاعدة.**
	ordersSvc.SetSettings(settingsStore)
	f := &merchantFixture{
		pool: pool,
		srv: &Server{
			pg: pool, logger: quiet, hub: realtime.NewHub(quiet),
			cashbox: cashboxSvc, settings: settingsStore, wallet: walletSvc,
			orders: ordersSvc, identity: ident,
			catalog:  catalog.NewService(pool, ident),
			platform: platform.New(pool, settingsStore),
		},
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO categories (name, icon, active)
		VALUES ('فحصُ حرّاس المتجر', 'other', true) RETURNING id`).Scan(&f.cat); err != nil {
		t.Fatalf("تعذّر التصنيف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, f.cat)
	})
	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, icon, sort_order)
		VALUES ('قسمُ فحصِ الحرّاس', 'food', 98) RETURNING id`).Scan(&f.psec); err != nil {
		t.Fatalf("تعذّر قسمُ السوق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, f.psec)
	})
	return f
}

// merchantsSelfManage يضبط الوضعَ إلى «المتجر يدير» — فيبقى للمتجر حقُّ القبول.
//
// **ويُعاد الوضعُ السابقُ عند الانتهاء** — الإعدادُ عامٌّ في القاعدة، **ولو
// تُرك «المتجر يدير» لَغيّر سلوكَ اختبارٍ لاحقٍ يقرؤه** (الفحوص متسلسلةٌ `-p 1`).
func (f *merchantFixture) merchantsSelfManage(t *testing.T) {
	t.Helper()
	st := settings.NewStore(f.pool)
	prev := st.GetString(context.Background(), "platform.orders_mode")
	if prev == "" {
		prev = orders.ModePlatform
	}
	if err := st.SetInternal(
		context.Background(), "platform.orders_mode", orders.ModeMerchants); err != nil {
		t.Fatalf("تعذّر ضبط الوضع: %v", err)
	}
	t.Cleanup(func() {
		_ = st.SetInternal(context.Background(), "platform.orders_mode", prev)
	})
}

func (f *merchantFixture) owner(t *testing.T) string {
	t.Helper()
	return testdb.NewUser(t, f.pool, "merchant")
}

func (f *merchantFixture) store(t *testing.T, ownerID, name, status string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO merchants (name, owner_user_id, category_id, commission_percent, status)
		VALUES ($1, $2, $3, 10, $4) RETURNING id`,
		name, ownerID, f.cat, status).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء المتجر %q: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
	})
	return id
}

// pendingOrder طلبٌ نقديٌّ في «pending» — **هذا ما يُقبل أو يُرفض.**
func (f *merchantFixture) pendingOrder(t *testing.T, merchantID string) string {
	t.Helper()
	customer := testdb.NewUser(t, f.pool, "customer")
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, 'pending', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 48000, 10000, 58000, 0, 58000,
			`+qaSnapSQL()+`)
		RETURNING id`, customer, merchantID).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return id
}

func (f *merchantFixture) status(t *testing.T, orderID string) string {
	t.Helper()
	var st string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&st); err != nil {
		t.Fatalf("تعذّرت قراءة الحالة: %v", err)
	}
	return st
}

// transition ينادي معالِجَ الانتقال كما يناديه المسار: بمعرّفٍ وسياقِ مالك.
func (f *merchantFixture) transition(ownerID, orderID, to, note string) *httptest.ResponseRecorder {
	body := `{"to":"` + to + `","note":"` + note + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/merchant/orders/"+orderID+"/transition", strings.NewReader(body))
	return f.driveOrder(req, ownerID, orderID, f.srv.handleMerchantTransition)
}

func (f *merchantFixture) ready(ownerID, orderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/merchant/orders/"+orderID+"/ready", nil)
	return f.driveOrder(req, ownerID, orderID, f.srv.handleMerchantReady)
}

func (f *merchantFixture) getOrder(ownerID, orderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/merchant/orders/"+orderID, nil)
	return f.driveOrder(req, ownerID, orderID, f.srv.handleMerchantGetOrder)
}

func (f *merchantFixture) driveOrder(req *http.Request, ownerID, orderID string,
	h http.HandlerFunc) *httptest.ResponseRecorder {
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, ownerID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"merchant"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// reqAs طلبٌ فارغٌ بسياقِ فاعلٍ بعينه — لنداء الحرّاس مباشرةً.
func reqAs(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	return req.WithContext(context.WithValue(req.Context(), ctxUserID, userID))
}

// ══════════════════════════════════════════════════════════════════════
// **قبولٌ مزدوج — طلبٌ واحدٌ يُقبل مرّةً لا مرّتين**
// ══════════════════════════════════════════════════════════════════════

func TestMerchant_DoubleAccept(t *testing.T) {
	f := newMerchantFixture(t)
	f.merchantsSelfManage(t)
	owner := f.owner(t)
	m := f.store(t, owner, "متجرُ القبول المزدوج", "active")
	order := f.pendingOrder(t, m)

	if w := f.transition(owner, order, "accepted", ""); w.Code != http.StatusOK {
		t.Fatalf("القبولُ الأوّل رُدّ بـ%d: %s", w.Code, w.Body.String())
	}
	// **وبعد القبول لم تعد الحالةُ pending** — تقدّمت (تحضيرٌ تلقائيٌّ ثمّ
	// إنزالٌ إن كان مُفعَّلاً)، **والمهمُّ أنّها غادرت pending فلا تُقبل ثانية.**
	if st := f.status(t, order); st == "pending" {
		t.Fatalf("بعد القبول بقيت الحالةُ pending — والقبولُ يجب أن يقدّمها")
	}
	// **والقبولُ الثاني يُردّ** — لم تعد الحالةُ pending، **فلا انتقالَ إليه ثانية.**
	w := f.transition(owner, order, "accepted", "")
	if w.Code == http.StatusOK {
		t.Fatal("قُبل الطلبُ مرّتين — **وطلبٌ مقبولٌ مرّتين طلبٌ يُطبخ مرّتين**")
	}
	if code := errCode(t, w); code != "invalid_transition" {
		t.Fatalf("رمزُ القبول الثاني %q، والمتوقّع invalid_transition", code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **قبولٌ مقابل رفض — واحدٌ يفوز والآخرُ يُردّ**
// ══════════════════════════════════════════════════════════════════════
//
// **N جولةٍ، وفي كلٍّ خيطُ قبولٍ وخيطُ رفضٍ يضغطان معاً** — يجب أن ينتهيَ
// الطلبُ إلى مصيرٍ واحد: إمّا مقبولٌ (فـ«preparing») وإمّا مرفوض، **ولا حالٌ
// بينهما، ولا نجاحان.**

func TestMerchant_AcceptVsReject(t *testing.T) {
	f := newMerchantFixture(t)
	f.merchantsSelfManage(t)
	owner := f.owner(t)
	m := f.store(t, owner, "متجرُ القبول والرفض", "active")

	const rounds = 12
	for i := 0; i < rounds; i++ {
		order := f.pendingOrder(t, m)
		var wg sync.WaitGroup
		codes := make([]int, 2)
		start := make(chan struct{})
		acts := []struct{ to, note string }{{"accepted", ""}, {"rejected", "الصنف غير متوفر"}}
		for j, a := range acts {
			wg.Add(1)
			go func(j int, to, note string) {
				defer wg.Done()
				<-start
				codes[j] = f.transition(owner, order, to, note).Code
			}(j, a.to, a.note)
		}
		close(start)
		wg.Wait()

		wins := 0
		for _, c := range codes {
			if c == http.StatusOK {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("جولة %d: عددُ الرابحين %d (قبول=%d رفض=%d) — والمتوقّع واحد",
				i, wins, codes[0], codes[1])
		}
		// **مصيرٌ واحدٌ لا حالٌ بينهما**: إمّا رُفض، وإمّا قُبل فتقدّم — **وأيّاً
		// كان لم يبقَ pending، ولم يجمع بين القبول والرفض.**
		if st := f.status(t, order); st == "pending" {
			t.Fatalf("جولة %d: بقي الطلبُ pending رغم رابحٍ واحد", i)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **قبولٌ مقابل مهلة — قبولُ المتجر يسابق القبولَ التلقائيّ**
// ══════════════════════════════════════════════════════════════════════
//
// **القبولُ التلقائيُّ بعد المهلة** (`sweepAutoAccept`) **ينادي
// `Transition` بدور `ops` وفاعلٍ فارغ** — فيُحاكى هنا بندائه نفسِه متزامناً
// مع قبول المتجر. **قفلُ `FOR UPDATE` يفصل بينهما**: واحدٌ يقبل والآخرُ يرى
// الطلبَ وقد قُبل فيُردّ، **ولا يُقبل الطلبُ مرّتين ولو تزامن الإنسانُ والآلة.**

func TestMerchant_AcceptVsTimeout(t *testing.T) {
	f := newMerchantFixture(t)
	f.merchantsSelfManage(t)
	owner := f.owner(t)
	m := f.store(t, owner, "متجرُ القبول والمهلة", "active")

	const rounds = 12
	for i := 0; i < rounds; i++ {
		order := f.pendingOrder(t, m)
		var wg sync.WaitGroup
		start := make(chan struct{})
		var merchantCode int
		var timeoutErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			merchantCode = f.transition(owner, order, "accepted", "").Code
		}()
		go func() {
			defer wg.Done()
			<-start
			// **ما يفعله الكانسُ حرفيّاً**: قبولٌ بدور ops وفاعلٍ فارغ.
			_, timeoutErr = f.srv.orders.Transition(
				context.Background(), "", []string{"ops"}, order, orders.StAccepted, "قبولٌ تلقائيّ")
		}()
		close(start)
		wg.Wait()

		merchantWon := merchantCode == http.StatusOK
		timeoutWon := timeoutErr == nil
		if merchantWon == timeoutWon {
			t.Fatalf("جولة %d: المتجر=%v المهلة=%v — والمتوقّع فائزٌ واحدٌ لا اثنان ولا صفر",
				i, merchantWon, timeoutWon)
		}
		// **وأيّاً فاز، الطلبُ قُبل فتقدّم** — لا حالٌ فاسدةٌ ولا بقاءٌ على pending.
		if st := f.status(t, order); st == "pending" {
			t.Fatalf("جولة %d: بقي الطلبُ pending رغم قبولٍ رابح", i)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **جاهزيّةٌ مرّتين — الوقتُ يُثبَّت مرّةً واحدة**
// ══════════════════════════════════════════════════════════════════════

func TestMerchant_ReadyTwiceKeepsFirstTime(t *testing.T) {
	f := newMerchantFixture(t)
	f.merchantsSelfManage(t)
	owner := f.owner(t)
	m := f.store(t, owner, "متجرُ الجاهزيّة", "active")
	order := f.pendingOrder(t, m)

	if w := f.transition(owner, order, "accepted", ""); w.Code != http.StatusOK {
		t.Fatalf("القبولُ رُدّ بـ%d: %s", w.Code, w.Body.String())
	}
	if w := f.ready(owner, order); w.Code != http.StatusOK {
		t.Fatalf("الجاهزيّةُ الأولى رُدّت بـ%d: %s", w.Code, w.Body.String())
	}
	var first *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT ready_at::text FROM orders WHERE id = $1`, order).Scan(&first); err != nil {
		t.Fatalf("تعذّرت قراءة ready_at: %v", err)
	}
	if first == nil {
		t.Fatal("ready_at لم يُكتب بعد الجاهزيّة الأولى")
	}
	// **إعلانٌ ثانٍ لا يعيد ضبطَ الوقت** — لحظةُ الجاهزيّة واحدةٌ، وإعادتُها
	// تُفسد قياسَ زمن التحضير.
	if w := f.ready(owner, order); w.Code != http.StatusOK {
		t.Fatalf("الجاهزيّةُ الثانية رُدّت بـ%d: %s", w.Code, w.Body.String())
	}
	var second *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT ready_at::text FROM orders WHERE id = $1`, order).Scan(&second); err != nil {
		t.Fatalf("تعذّرت قراءة ready_at الثانية: %v", err)
	}
	if second == nil || *second != *first {
		t.Fatalf("تبدّل ready_at من %v إلى %v — ولحظةُ الجاهزيّة يجب أن تثبت", first, second)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تسلّلٌ على طلبٍ لا يملكه — يُردّ بـ«غير موجود»**
// ══════════════════════════════════════════════════════════════════════

func TestMerchant_OrderIDOR(t *testing.T) {
	f := newMerchantFixture(t)
	f.merchantsSelfManage(t)
	ownerA, ownerB := f.owner(t), f.owner(t)
	mA := f.store(t, ownerA, "متجرُ ألف", "active")
	f.store(t, ownerB, "متجرُ باء", "active")
	order := f.pendingOrder(t, mA)

	// **ألف يرى طلبَه** — قفلٌ لا يُفتح بمفتاحه عطبٌ لا حماية.
	if w := f.getOrder(ownerA, order); w.Code != http.StatusOK {
		t.Fatalf("صاحبُ الطلب رُدّ بـ%d عن طلبه", w.Code)
	}
	// **وباء لا يراه ولا يحرّكه** — ومعرّفٌ يُخمَّن أو يُقرأ من ردٍّ لا يكفي.
	if w := f.getOrder(ownerB, order); w.Code != http.StatusNotFound {
		t.Fatalf("مالكٌ أجنبيٌّ قرأ طلبَ غيره — رُدّ بـ%d والمتوقّع 404", w.Code)
	}
	if w := f.transition(ownerB, order, "accepted", ""); w.Code != http.StatusNotFound {
		t.Fatalf("مالكٌ أجنبيٌّ حرّك طلبَ غيره — رُدّ بـ%d والمتوقّع 404", w.Code)
	}
	if st := f.status(t, order); st != "pending" {
		t.Fatalf("تبدّلت حالةُ الطلب إلى %q بفعلِ أجنبيّ — والمتوقّع pending", st)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ الكتابة: مصفوفةُ الملكيّةِ والإيقاف** (صنفٌ · قسمٌ · متجر)
// ══════════════════════════════════════════════════════════════════════
//
// **يُنادى الحارسُ مباشرةً** — فالمعالِجاتُ تناديه أوّلَ سطر، **والمصفوفةُ
// أوضحُ عند مصدره**: مملوكٌ نشطٌ يمرّ، وموقوفٌ يُردّ `store_suspended`،
// وأجنبيٌّ يُردّ `forbidden` بلا كشفِ وجود.

func TestMerchant_WriteGuardMatrix(t *testing.T) {
	f := newMerchantFixture(t)
	ownerA, ownerB := f.owner(t), f.owner(t)
	active := f.store(t, ownerA, "متجرٌ نشط", "active")
	suspended := f.store(t, ownerA, "متجرٌ موقوف", "suspended")
	foreign := f.store(t, ownerB, "متجرُ غيري", "active")

	// صنفٌ وقسمٌ في كلٍّ من النشط والموقوف والأجنبيّ.
	activeItem, activeSec := f.itemAndSection(t, active)
	suspendedItem, suspendedSec := f.itemAndSection(t, suspended)
	foreignItem, foreignSec := f.itemAndSection(t, foreign)

	cases := []struct {
		name string
		got  error
		want error
	}{
		{"متجرٌ نشطٌ مملوك", f.srv.merchantWriteGuard(reqAs(ownerA), active), nil},
		{"متجرٌ موقوفٌ مملوك", f.srv.merchantWriteGuard(reqAs(ownerA), suspended), errStoreSuspended},
		{"متجرٌ أجنبيّ", f.srv.merchantWriteGuard(reqAs(ownerA), foreign), errForbidden},
		{"متجرٌ غيرُ موجود", f.srv.merchantWriteGuard(reqAs(ownerA), active[:len(active)-1]+"0"), errForbidden},
		{"صنفٌ نشطٌ مملوك", f.srv.itemWriteGuard(reqAs(ownerA), activeItem), nil},
		{"صنفٌ موقوفٌ مملوك", f.srv.itemWriteGuard(reqAs(ownerA), suspendedItem), errStoreSuspended},
		{"صنفٌ أجنبيّ", f.srv.itemWriteGuard(reqAs(ownerA), foreignItem), errForbidden},
		{"قسمٌ نشطٌ مملوك", f.srv.sectionWriteGuard(reqAs(ownerA), activeSec), nil},
		{"قسمٌ موقوفٌ مملوك", f.srv.sectionWriteGuard(reqAs(ownerA), suspendedSec), errStoreSuspended},
		{"قسمٌ أجنبيّ", f.srv.sectionWriteGuard(reqAs(ownerA), foreignSec), errForbidden},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: الحارسُ ردّ %v، والمتوقّع %v", c.name, c.got, c.want)
		}
	}
}

// itemAndSection يُنشئ قسماً وصنفاً في متجرٍ ويعيد معرّفَيهما.
func (f *merchantFixture) itemAndSection(t *testing.T, merchantID string) (item, section string) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قسمٌ للفحص')
		RETURNING id`, merchantID).Scan(&section); err != nil {
		t.Fatalf("تعذّر إنشاء قسم: %v", err)
	}
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
			name, merchant_price, price, available)
		VALUES ($1, $2, $3, 'صنفٌ للفحص', 10000, 10000, true) RETURNING id`,
		merchantID, section, f.psec).Scan(&item); err != nil {
		t.Fatalf("تعذّر إنشاء صنف: %v", err)
	}
	return item, section
}

// ══════════════════════════════════════════════════════════════════════
// **رفضُ الكتابة على متجرٍ موقوف — عبر المعالِج كاملاً** (الإتاحة)
// ══════════════════════════════════════════════════════════════════════
//
// **معالِجُ الإتاحة يكتب بـSQL خامٍّ لا عبر الكتالوج** — فيُختبَر كاملاً بلا
// تركيبِ نصفِ المنصّة: النشطُ يمرّ، والموقوفُ يُردّ `store_suspended`،
// والأجنبيُّ `forbidden`، **والقراءةُ (المتاجر) تبقى مفتوحةً للموقوف.**

func TestMerchant_SuspendedAvailabilityRejected(t *testing.T) {
	f := newMerchantFixture(t)
	ownerA := f.owner(t)
	active := f.store(t, ownerA, "متجرُ الإتاحة النشط", "active")
	suspended := f.store(t, ownerA, "متجرُ الإتاحة الموقوف", "suspended")
	activeItem, _ := f.itemAndSection(t, active)
	suspendedItem, _ := f.itemAndSection(t, suspended)

	setAvail := func(ownerID, itemID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch,
			"/merchant/menu/items/"+itemID+"/availability",
			strings.NewReader(`{"available":false}`))
		rc := chi.NewRouteContext()
		rc.URLParams.Add("itemID", itemID)
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		ctx = context.WithValue(ctx, ctxUserID, ownerID)
		ctx = context.WithValue(ctx, ctxRoles, []string{"merchant"})
		w := httptest.NewRecorder()
		f.srv.handleMerchantItemAvailability(w, req.WithContext(ctx))
		return w
	}

	if w := setAvail(ownerA, activeItem); w.Code != http.StatusOK {
		t.Fatalf("إتاحةُ صنفٍ نشطٍ رُدّت بـ%d: %s", w.Code, w.Body.String())
	}
	w := setAvail(ownerA, suspendedItem)
	if w.Code == http.StatusOK {
		t.Fatal("قبِل متجرٌ موقوفٌ تبديلَ الإتاحة — **والموقوفُ يُقرأ ولا يُكتب فيه**")
	}
	if code := errCode(t, w); code != "store_suspended" {
		t.Fatalf("رمزُ الرفض %q، والمتوقّع store_suspended", code)
	}

	// **والقراءةُ تبقى مفتوحةً** — المالكُ يرى متجرَيه الموقوفَ والنشطَ معاً.
	stores := f.stores(t, ownerA)
	if len(stores) != 2 {
		t.Fatalf("المالكُ يرى %d متجراً، والمتوقّع 2 (النشطُ والموقوفُ كلاهما يُقرأ)", len(stores))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **عزلُ مالكٍ متعدّدِ المتاجر — يرى متاجرَه لا متاجرَ غيره**
// ══════════════════════════════════════════════════════════════════════

func TestMerchant_MultiStoreIsolation(t *testing.T) {
	f := newMerchantFixture(t)
	ownerA, ownerB := f.owner(t), f.owner(t)
	m1 := f.store(t, ownerA, "متجرُ ألف الأوّل", "active")
	m2 := f.store(t, ownerA, "متجرُ ألف الثاني", "active")
	mB := f.store(t, ownerB, "متجرُ باء", "active")

	got := f.stores(t, ownerA)
	ids := map[string]bool{}
	for _, s := range got {
		ids[s["id"].(string)] = true
		// **والحالةُ الفعليّةُ محسوبةٌ في الخادم** (A3): بلا ساعاتٍ = مفتوحٌ دائماً.
		if _, ok := s["open_now"]; !ok {
			t.Errorf("المتجر %v بلا حقل open_now — والتطبيقُ يحتاجه للحالة الفعليّة", s["id"])
		}
		if _, ok := s["next_open"]; !ok {
			t.Errorf("المتجر %v بلا حقل next_open", s["id"])
		}
	}
	if !ids[m1] || !ids[m2] {
		t.Fatalf("مالكٌ لا يرى أحدَ متجرَيه — رأى %v", ids)
	}
	if ids[mB] {
		t.Fatal("مالكٌ رأى متجرَ غيره في قائمة متاجره — **تسرّبٌ بين المُلّاك**")
	}
	if len(got) != 2 {
		t.Fatalf("عددُ متاجر المالك %d، والمتوقّع 2", len(got))
	}
}

// stores يقرأ متاجرَ مالكٍ عبر المعالِج ويعيدها خرائطَ حرّة.
func (f *merchantFixture) stores(t *testing.T, ownerID string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/merchant/stores", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, ownerID)
	w := httptest.NewRecorder()
	f.srv.handleMerchantStores(w, req.WithContext(ctx))
	if w.Code != http.StatusOK {
		t.Fatalf("قائمةُ المتاجر رُدّت بـ%d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			Stores []map[string]any `json:"stores"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ غيرُ مقروء: %v — %s", err, w.Body.String())
	}
	return out.Data.Stores
}
