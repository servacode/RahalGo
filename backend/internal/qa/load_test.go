//go:build load

package qa

// ══════════════════════════════════════════════════════════════════════
// **اختبارُ التحمّل — كم تحتمل المنصّة، وأين تنكسر أوّلاً** (طلبُ المالك ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// «افحصلي المنصة تتحمل ضغط بكل الظروف وأقسى أنواع الظروف.»
//
// **لا يُشغَّل مع الحزمة** (وسمُ البناء `load`) — دقائقُ من الضغط على القاعدة:
//
//	go test -tags load -run TestLoad$ -timeout 60m -v ./internal/qa/
//	QA_REAL_REDIS_ADDR=localhost:6380 go test -tags load -run TestLoadChaos -v ./internal/qa/
//
// # ما يُضغط معاً (خليطٌ يشبه يوماً مزدحماً)
//
//	زبائنُ يتصفّحون   الرئيسيّة · المنصّة · الأقسام · أصنافُ القسم · البحث
//	زبائنُ يطلبون      إنشاءُ طلبٍ حقيقيّ (معاملةٌ · دفترٌ · توزيع) · طلباتي
//	سائقون             موقعٌ · طابورُ العروض
//	المكتب             لوحةُ الطلبات
//
// **`TestLoad`**: مراحلُ تتصاعد حتّى يتجاوز الخطأُ ١٪ أو p95 ثانيتين — **السعةُ
// على هذا الجهاز** لا على الخادم.
//
// **`TestLoadChaos`**: ضغطٌ ثابتٌ ثمّ **يُجمَّد Redis ثمّ القاعدة** (`docker pause`)
// — **أتعود المنصّةُ لحالها؟ وهل ضاع طلبٌ أو صار رصيدٌ سالباً؟**

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type loadOp struct {
	name  string
	do    func(c *http.Client) (int, error)
	share int // وزنُه في الخليط
}

type loadStat struct {
	mu   sync.Mutex
	lat  map[string][]time.Duration
	errs map[string]int
	n    map[string]int
	code map[int]int
}

func newLoadStat() *loadStat {
	return &loadStat{lat: map[string][]time.Duration{}, errs: map[string]int{}, n: map[string]int{}, code: map[int]int{}}
}

func (s *loadStat) add(op string, d time.Duration, code int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n[op]++
	s.lat[op] = append(s.lat[op], d)
	s.code[code]++
	if err != nil || code >= 500 || code == 0 || code == 429 {
		s.errs[op]++
	}
}

func pct(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	c := append([]time.Duration(nil), ds...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[int(float64(len(c)-1)*p)]
}

func req(c *http.Client, base, method, path, token string, body any, key string) (int, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, base+path, rd)
	if err != nil {
		return 0, err
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	res, err := c.Do(r)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	return res.StatusCode, nil
}

type loadRig struct {
	h      *Harness
	mix    []loadOp
	wheel  []int
	client *http.Client
}

// newLoadRig **العتادُ والخليط** — متاجرُ بأصناف · زبائن · سائقون · مكتب.
func newLoadRig(t *testing.T) *loadRig {
	h := New(t)
	f := h.Factory()
	h.Setting("customers.require_whatsapp", "false")
	// **والحدّان التجاريّان يُرفعان** — ثلاثةُ طلباتٍ مفتوحةٍ للزبون تجعل ٤٠٩ هو ما يُقاس.
	h.Setting("orders.max_open_per_customer", "50")
	treasury(t, h)
	base := h.Srv.URL

	nStores, nCust, nDrv := envInt("LOAD_STORES", 20), envInt("LOAD_CUSTOMERS", 600), envInt("LOAD_DRIVERS", 60)
	items := make([]*Item, 0, nStores*3)
	for i := 0; i < nStores; i++ {
		it := h.NewItem(int64(1000 + i*10))
		items = append(items, it)
		m := &Merchant{ID: it.MerchantID}
		for j := 0; j < 2; j++ {
			items = append(items, h.NewItemFor(m, int64(500+j*100)))
		}
	}
	custs := make([]*User, nCust)
	for i := range custs {
		custs[i] = h.NewUser("customer")
	}
	drivers := make([]*User, nDrv)
	for i := range drivers {
		drivers[i] = f.Driver(OnShift(), LocationAt(35.95+float64(i%20)/1000, 39.01, time.Now()))
	}
	ops := h.NewUser("operations")
	var sectionID string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT platform_section_id::text FROM menu_items WHERE id = $1::uuid`, items[0].ID).Scan(&sectionID)
	t.Logf("العتاد: %d متجراً · %d صنفاً · %d زبوناً · %d سائقاً", nStores, len(items), nCust, nDrv)

	var seq atomic.Int64
	pick := func() int { return int(seq.Add(1)) }
	mix := []loadOp{
		{"تصفّح: الرئيسيّة", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/public/home", "", nil, "")
		}, 20},
		{"تصفّح: المنصّة", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/public/platform", "", nil, "")
		}, 10},
		{"تصفّح: الأقسام", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/public/sections", "", nil, "")
		}, 10},
		{"تصفّح: أصنافُ قسم", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/public/sections/"+sectionID+"/items", "", nil, "")
		}, 15},
		{"تصفّح: بحث", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/public/search?q=QA", "", nil, "")
		}, 5},
		{"زبون: إنشاءُ طلب", func(c *http.Client) (int, error) {
			i := pick()
			return req(c, base, "POST", "/api/v1/orders", custs[i%len(custs)].Token, orderBody(items[i%len(items)], 1),
				"load-"+strconv.Itoa(i)+"-"+strconv.FormatInt(time.Now().UnixNano(), 36))
		}, 8},
		{"زبون: طلباتي", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/my/orders", custs[pick()%len(custs)].Token, nil, "")
		}, 7},
		{"سائق: الموقع", func(c *http.Client) (int, error) {
			i := pick()
			return req(c, base, "POST", "/api/v1/driver/location", drivers[i%len(drivers)].Token,
				map[string]any{"lat": 35.95 + float64(i%50)/10000, "lng": 39.01}, "")
		}, 15},
		{"سائق: الطابور", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/driver/queue", drivers[pick()%len(drivers)].Token, nil, "")
		}, 5},
		{"المكتب: لوحةُ الطلبات", func(c *http.Client) (int, error) {
			return req(c, base, "GET", "/api/v1/admin/orders/board", ops.Token, nil, "")
		}, 5},
	}
	var wheel []int
	for i, op := range mix {
		for k := 0; k < op.share; k++ {
			wheel = append(wheel, i)
		}
	}
	tr := &http.Transport{MaxIdleConns: 2000, MaxIdleConnsPerHost: 2000, IdleConnTimeout: 30 * time.Second}
	return &loadRig{h: h, mix: mix, wheel: wheel, client: &http.Client{Transport: tr, Timeout: 15 * time.Second}}
}

// run **يضغط بعددٍ من المستخدمين مدّةً** — ويردّ الإحصاء.
func (r *loadRig) run(users int, d time.Duration) (*loadStat, float64) {
	st := newLoadStat()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	start := time.Now()
	for u := 0; u < users; u++ {
		wg.Add(1)
		go func(u int) {
			defer wg.Done()
			k := u
			for {
				select {
				case <-stop:
					return
				default:
				}
				k += 7
				op := r.mix[r.wheel[k%len(r.wheel)]]
				t0 := time.Now()
				code, err := op.do(r.client)
				st.add(op.name, time.Since(t0), code, err)
			}
		}(u)
	}
	time.Sleep(d)
	close(stop)
	wg.Wait()
	return st, time.Since(start).Seconds()
}

type loadSummary struct {
	Users   int
	RPS     float64
	ErrRate float64
	P50     time.Duration
	P95     time.Duration
	P99     time.Duration
	Orders  int
}

func summarize(t *testing.T, label string, users int, st *loadStat, el float64, orders int) loadSummary {
	var all []time.Duration
	total, errs := 0, 0
	for name, ds := range st.lat {
		all = append(all, ds...)
		total += st.n[name]
		errs += st.errs[name]
	}
	s := loadSummary{Users: users, RPS: float64(total) / el, ErrRate: float64(errs) / float64(max(total, 1)),
		P50: pct(all, .5), P95: pct(all, .95), P99: pct(all, .99), Orders: orders}
	t.Logf("━━ %s: %.0f نداء/ث · خطأ %.2f%% · p50 %v · p95 %v · p99 %v · طلباتٌ أُنشئت %d (%.1f/ث)",
		label, s.RPS, s.ErrRate*100, s.P50.Round(time.Millisecond), s.P95.Round(time.Millisecond),
		s.P99.Round(time.Millisecond), orders, float64(orders)/el)
	names := make([]string, 0, len(st.lat))
	for n := range st.lat {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		ds := st.lat[n]
		t.Logf("     %-24s n=%-6d خطأ=%-5d p50=%-8v p95=%-8v p99=%v", n, st.n[n], st.errs[n],
			pct(ds, .5).Round(time.Millisecond), pct(ds, .95).Round(time.Millisecond), pct(ds, .99).Round(time.Millisecond))
	}
	t.Logf("     رموزُ الردّ: %v", st.code)
	return s
}

func TestLoad(t *testing.T) {
	r := newLoadRig(t)
	stageDur := time.Duration(envInt("LOAD_STAGE_SEC", 30)) * time.Second
	var results []loadSummary
	for _, users := range []int{10, 25, 50, 100, 200, 400} {
		before := loadCountOrders(r.h)
		st, el := r.run(users, stageDur)
		s := summarize(t, strconv.Itoa(users)+" مستخدماً متزامناً", users, st, el, loadCountOrders(r.h)-before)
		results = append(results, s)
		if s.ErrRate > 0.01 || s.P95 > 2*time.Second {
			t.Logf("⛔ الحدّ: عند %d مستخدماً تجاوز الخطأُ ١٪ أو p95 ثانيتين — نتوقّف", users)
			break
		}
	}
	loadIntegrity(t, r.h)
	if out := os.Getenv("LOAD_OUT"); out != "" {
		b, _ := json.MarshalIndent(results, "", " ")
		_ = os.WriteFile(out, b, 0o644)
	}
}

// TestLoadChaos **ضغطٌ ثابتٌ وعطلٌ في وسطه** — Redis ثمّ القاعدة.
func TestLoadChaos(t *testing.T) {
	if os.Getenv("QA_REAL_REDIS_ADDR") == "" {
		t.Skip("يحتاج Redis حقيقيّاً يُجمَّد: QA_REAL_REDIS_ADDR=localhost:6380")
	}
	r := newLoadRig(t)
	users := envInt("LOAD_CHAOS_USERS", 50)
	phase := func(label string, d time.Duration) {
		before := loadCountOrders(r.h)
		st, el := r.run(users, d)
		summarize(t, label, users, st, el, loadCountOrders(r.h)-before)
	}
	docker := func(args ...string) {
		if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
			t.Fatalf("docker %v: %v %s", args, err, out)
		}
	}
	redisC := envStr("LOAD_REDIS_CONTAINER", "rahalgo-redis")
	pgC := envStr("LOAD_PG_CONTAINER", "rahalgo-postgres")

	phase("١ · طبيعيّ", 20*time.Second)

	docker("pause", redisC)
	phase("٢ · Redis مجمَّد ١٥ث", 15*time.Second)
	docker("unpause", redisC)
	phase("٣ · بعد عودة Redis", 15*time.Second)

	docker("pause", pgC)
	func() {
		defer docker("unpause", pgC)
		phase("٤ · القاعدة مجمَّدة ١٠ث", 10*time.Second)
	}()
	phase("٥ · بعد عودة القاعدة", 20*time.Second)

	loadIntegrity(t, r.h)
}

// loadIntegrity **المالُ والطلباتُ بعد الضغط** — لا رصيدَ سالب، ولا طلبَ بلا بنود.
func loadIntegrity(t *testing.T, h *Harness) {
	var neg, noItems, total int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallets WHERE balance < 0`).Scan(&neg)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders o WHERE NOT EXISTS
		(SELECT 1 FROM order_items i WHERE i.order_id = o.id) AND o.kind = 'standard'`).Scan(&noItems)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders`).Scan(&total)
	t.Logf("السلامة: طلباتٌ %d · محافظُ سالبة %d · طلباتٌ عاديّةٌ بلا بنود %d", total, neg, noItems)
	if neg > 0 || noItems > 0 {
		t.Errorf("**أثرٌ مكسورٌ بعد الضغط**")
	}
}

// loadCountOrders **عددُ الطلبات — بمهلة**: والقاعدةُ مجمَّدةٌ لا يعلق العدُّ معها.
func loadCountOrders(h *Harness) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var n int
	_ = h.Pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&n)
	return n
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
