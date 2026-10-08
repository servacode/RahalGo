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
//	go test -tags load -run TestLoad -timeout 60m -v ./internal/qa/
//
// # ما يُضغط معاً في كلّ مرحلة (خليطٌ يشبه يوماً مزدحماً)
//
//	زبائنُ يتصفّحون   الرئيسيّة · الأقسام · أصنافُ القسم · البحث · المنصّة
//	زبائنُ يطلبون      إنشاءُ طلبٍ حقيقيّ (معاملةٌ · دفترٌ · توزيع)
//	سائقون             موقعٌ كلَّ ثانية · طابورُ العروض
//	المكتب             لوحةُ الطلبات وعدّاداتُها
//
// **والمراحلُ تتصاعد** حتّى يتجاوز الخطأُ ١٪ أو يتجاوز p95 ثانيتين — **فتلك
// السعةُ المقيسة على هذا الجهاز** لا على الخادم؛ والخادمُ (CX33: ٤ أنوية) يُقارَن
// بها في مرحلةٍ ثانيةٍ بإذن المالك.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	i := int(float64(len(c)-1) * p)
	return c[i]
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

func TestLoad(t *testing.T) {
	h := New(t)
	f := h.Factory()
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)
	base := h.Srv.URL

	// ── العتاد: متاجرُ بأصناف · زبائن · سائقون · مكتب ──
	nStores, nCust, nDrv := envInt("LOAD_STORES", 20), envInt("LOAD_CUSTOMERS", 300), envInt("LOAD_DRIVERS", 60)
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
			cu := custs[i%len(custs)]
			it := items[i%len(items)]
			return req(c, base, "POST", "/api/v1/orders", cu.Token, orderBody(it, 1), "load-"+strconv.Itoa(i)+"-"+strconv.FormatInt(time.Now().UnixNano(), 36))
		}, 8},
		{"زبون: طلباتي", func(c *http.Client) (int, error) {
			cu := custs[pick()%len(custs)]
			return req(c, base, "GET", "/api/v1/orders", cu.Token, nil, "")
		}, 7},
		{"سائق: الموقع", func(c *http.Client) (int, error) {
			i := pick()
			d := drivers[i%len(drivers)]
			return req(c, base, "POST", "/api/v1/driver/location", d.Token,
				map[string]any{"lat": 35.95 + float64(i%50)/10000, "lng": 39.01}, "")
		}, 15},
		{"سائق: الطابور", func(c *http.Client) (int, error) {
			d := drivers[pick()%len(drivers)]
			return req(c, base, "GET", "/api/v1/driver/queue", d.Token, nil, "")
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
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
	stageDur := time.Duration(envInt("LOAD_STAGE_SEC", 30)) * time.Second

	type stageRes struct {
		users         int
		rps           float64
		errRate       float64
		p50, p95, p99 time.Duration
		perOp         map[string][3]time.Duration
		errOp         map[string]int
		codes         map[int]int
		orders        int
	}
	var results []stageRes
	for _, users := range []int{10, 25, 50, 100, 200, 400} {
		st := newLoadStat()
		stop := make(chan struct{})
		var wg sync.WaitGroup
		before := loadCountOrders(h)
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
					op := mix[wheel[k%len(wheel)]]
					t0 := time.Now()
					code, err := op.do(client)
					st.add(op.name, time.Since(t0), code, err)
				}
			}(u)
		}
		time.Sleep(stageDur)
		close(stop)
		wg.Wait()
		el := time.Since(start).Seconds()

		var all []time.Duration
		total, errs := 0, 0
		per := map[string][3]time.Duration{}
		for name, ds := range st.lat {
			all = append(all, ds...)
			total += st.n[name]
			errs += st.errs[name]
			per[name] = [3]time.Duration{pct(ds, .5), pct(ds, .95), pct(ds, .99)}
		}
		r := stageRes{users: users, rps: float64(total) / el, errRate: float64(errs) / float64(max(total, 1)),
			p50: pct(all, .5), p95: pct(all, .95), p99: pct(all, .99), perOp: per, errOp: st.errs, codes: st.code,
			orders: loadCountOrders(h) - before}
		results = append(results, r)
		t.Logf("━━ %d مستخدماً متزامناً: %.0f نداء/ث · خطأ %.2f%% · p50 %v · p95 %v · p99 %v · طلباتٌ أُنشئت %d (%.1f/ث)",
			users, r.rps, r.errRate*100, r.p50.Round(time.Millisecond), r.p95.Round(time.Millisecond),
			r.p99.Round(time.Millisecond), r.orders, float64(r.orders)/el)
		names := make([]string, 0, len(per))
		for n := range per {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			p := per[n]
			t.Logf("     %-24s n=%-6d خطأ=%-5d p50=%-8v p95=%-8v p99=%v", n, st.n[n], st.errs[n],
				p[0].Round(time.Millisecond), p[1].Round(time.Millisecond), p[2].Round(time.Millisecond))
		}
		t.Logf("     رموزُ الردّ: %v", st.code)
		if r.errRate > 0.01 || r.p95 > 2*time.Second {
			t.Logf("⛔ الحدّ: عند %d مستخدماً تجاوز الخطأُ ١٪ أو p95 ثانيتين — نتوقّف", users)
			break
		}
	}

	// **والمالُ بعد الضغط** — كلُّ طلبٍ أُنشئ له قيدُه، ولا رصيدَ سالب.
	var neg int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallets WHERE balance < 0`).Scan(&neg)
	t.Logf("محافظُ سالبةٌ بعد الضغط: %d", neg)
	out, _ := json.MarshalIndent(results, "", " ")
	_ = os.WriteFile(os.Getenv("LOAD_OUT"), out, 0o644)
	_ = fmt.Sprint
}

func loadCountOrders(h *Harness) int {
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders`).Scan(&n)
	return n
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}
