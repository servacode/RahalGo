// تخويلُ البثّ وحالُ الحساب — **`D14`.**
//
// # ما يقع
//
// **`handleWS` يتحقّق من الجلسة** (`CheckSession`: أحيّةٌ هي؟ وما
// أدوارُها؟) — **ولا يسأل عن حال الحساب.**
//
// **و`RequireAuth` في `REST` تسأل** (`ActiveStatus`): **نشطٌ يمرّ ·
// معلَّقٌ لا يمرّ إلّا في مسارات الإتمام · وما عداهما يُردّ.**
//
// **فالموقوفُ يُردّ في بابٍ ويُقبَل في آخر** — **والقناةُ الدائمةُ
// أطولُ عمراً من نداء.**
//
// # وما لا يعنيه هذا
//
// **الجلسةُ ليست الامتياز**: **جلسةُ الموقوف تبقى حيّةً وتُجدَّد**
// (`XG-39`) — **وذاك عقدٌ قائمٌ لا يُنقَض.** **المقيسُ هنا ما تمنحه
// تلك الجلسةُ من وصولٍ لحظيّ.**
//
// # واستثناءُ الإتمام
//
// **والمعلَّقُ يُتمّ ما بيده** (`suspension.go`): سائقٌ ينقل طلبَه
// ويثبت تسليمَه، ومتجرٌ يقبل ما بين يديه، وزبونٌ يرى طلبَه ويلغيه.
// **فحقُّه في البثّ حقُّه في غرفته هو** — **لا في طابور العمل
// الجديد ولا في غرفة المكتب.**
package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// wsDial **مصافحةٌ حقيقيّةٌ لا نداءٌ عاديّ** — وتردّ الحالَ والوصلة.
func wsDial(t *testing.T, h *Harness, token string) (*websocket.Conn, int) {
	t.Helper()
	url := strings.Replace(h.Srv.URL, "http://", "ws://", 1) +
		"/api/v1/ws?token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, res, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		code := 0
		if res != nil {
			code = res.StatusCode
		}
		return nil, code
	}
	return c, http.StatusSwitchingProtocols
}

// wsReader **قارئٌ واحدٌ للوصلة يجمع كلَّ ما يصلها.**
//
// **ولا تُقرأ الوصلةُ بمهلةٍ ثمّ يُعاد قراءتُها**: **إلغاءُ سياق
// القراءة يُغلق الوصلةَ في هذه المكتبة** — **فأوّلُ انتظارٍ خائبٍ
// يقتل المقبس**، **فيُقرأ ما بعده «مُنع» وهو مقبسٌ ميّت.**
//
// **(وقع في أوّل كتابةِ هذا الفحص: مُنع الطابورُ صحيحاً، ثمّ لم تصل
// غرفتُه — والسببُ أنّ الوصلةَ أُغلقت لا أنّ الغرفةَ مُنعت.)**
func wsReader(c *websocket.Conn) <-chan map[string]any {
	out := make(chan map[string]any, 64)
	go func() {
		defer close(out)
		for {
			_, data, err := c.Read(context.Background())
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(data, &m)
			out <- m
		}
	}()
	return out
}

// wsExpect **يبثّ حتّى يصل أو تنقضي المهلة.**
//
// **والمصافحةُ تعود قبل أن يشترك الخادمُ في الغرف**: `Accept` ثمّ
// `Subscribe`. **فبثٌّ يقع في تلك الفجوة يضيع** — **وغيابُه يُقرأ
// «مُنع» وهو لم يُرسَل أصلاً.**
//
// **ولا نومَ يُضاف لإخفاء سباق**: **يُعاد البثُّ لأنّ المقيسَ وصولُ
// حمولةٍ لا توقيتُ اشتراك.**
func wsExpect(t *testing.T, h *Harness, in <-chan map[string]any,
	room string, within time.Duration) map[string]any {
	t.Helper()
	if in == nil {
		return nil
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(within)
	for {
		h.Hub.Publish(room, map[string]any{"type": "order"})
		select {
		case m, ok := <-in:
			if !ok {
				return nil
			}
			return m
		case <-tick.C:
		case <-deadline:
			return nil
		}
	}
}

// suspend يُعلّق حساباً ويُبطل ذاكرةَ حاله.
//
// **والذاكرةُ تُبطَل صراحةً** — **وإلّا قُرئ الحالُ القديمُ ثلاثين
// ثانيةً فمرّ الفحصُ على حالٍ لم يعُد قائماً.**
func setStatus(t *testing.T, h *Harness, userID, status string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET status = $2 WHERE id = $1::uuid`, userID, status); err != nil {
		t.Fatalf("ضبطُ الحال: %v", err)
	}
	h.Redis().Del(ctxBG(), "ustatus:"+userID)
}

// ══════════════════════════════════════════════════════════════════════
// **`D14` — الموقوفُ يُردّ في `REST` ويُقبَل في البثّ**
// ══════════════════════════════════════════════════════════════════════

func TestD14_SuspendedHasNoBroadRealtimeAccess(t *testing.T) {
	h := New(t)
	cust := h.Customer()

	// ── الأساسُ: نشطٌ يمرّ في البابين ───────────────────────────────
	rest := h.GET("/api/v1/my/orders", cust.Token)
	c0, code0 := wsDial(t, h, cust.Token)
	if c0 != nil {
		defer c0.CloseNow()
	}
	t.Logf("نشطٌ — REST=%d · مصافحةُ البثّ=%d", rest.Code, code0)
	if rest.Code >= 400 || code0 != http.StatusSwitchingProtocols {
		t.Fatalf("**الحالُ السليمُ مكسور** — REST=%d WS=%d", rest.Code, code0)
	}

	// ── ثمّ التعليق ────────────────────────────────────────────────
	setStatus(t, h, cust.ID, "suspended")

	restAfter := h.GET("/api/v1/my/orders", cust.Token)
	c1, code1 := wsDial(t, h, cust.Token)
	if c1 != nil {
		defer c1.CloseNow()
	}
	t.Logf("معلَّقٌ — REST=%d %s · مصافحةُ البثّ=%d",
		restAfter.Code, restAfter.Err(), code1)

	if restAfter.Code < 400 {
		t.Fatalf("**`REST` لم يردّ المعلَّق** — ولا تُقاس مفارقةٌ بلا طرفين")
	}

	// ── والدليلُ حمولةٌ تصل لا مصافحةٌ تُقبَل ───────────────────────
	//
	// **ووصلةٌ صامتةٌ ليست حماية**: **تُبثُّ حمولةٌ حقيقيّةٌ في غرفةٍ
	// عامّة**، **فإن وصلته فقد نال وصولاً لا يملكه.**
	if c1 != nil {
		got := wsExpect(t, h, wsReader(c1), "catalog", 2*time.Second)
		t.Logf("  وصلَ المعلَّقَ في غرفةٍ عامّة: %v", got)
		if got != nil {
			t.Errorf("**وصلةُ المعلَّق تتلقّى بثّاً عامّاً** — "+
				"**و`REST` تردّه بـ%d.** (`D14`)", restAfter.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والمحظورُ والمحذوفُ أشدّ** — ولا استثناءَ فيهما
// ══════════════════════════════════════════════════════════════════════

func TestD14_BlockedAndDeletedHaveNoRealtime(t *testing.T) {
	h := New(t)
	for _, st := range []string{"blocked", "deleted"} {
		t.Run(st, func(t *testing.T) {
			u := h.Customer()
			setStatus(t, h, u.ID, st)

			rest := h.GET("/api/v1/my/orders", u.Token)
			c, code := wsDial(t, h, u.Token)
			if c != nil {
				defer c.CloseNow()
			}
			t.Logf("%s — REST=%d · مصافحةُ البثّ=%d", st, rest.Code, code)
			if rest.Code < 400 {
				t.Fatalf("**`REST` لم يردّ %q**", st)
			}
			if code == http.StatusSwitchingProtocols {
				t.Errorf("**قُبلت مصافحةُ %q في البثّ و`REST` تردّه.** (`D14`)", st)
			}
		})
	}
}

// ══════════════════════════════════════════════════════════════════════
// **واستثناءُ الإتمام مقصورٌ على غرفته**
// ══════════════════════════════════════════════════════════════════════
//
// **والسائقُ المعلَّقُ يُتمّ طلبَه** — **فيصله ما يخصّه.**
// **ولا يصله طابورُ العمل الجديد** — **وذاك ما يمنعه التعليق.**

func TestD14_SuspendedDriverScopedToOwnRoom(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver(OnShift())
	setStatus(t, h, drv.ID, "suspended")

	c, code := wsDial(t, h, drv.Token)
	if c != nil {
		defer c.CloseNow()
	}
	t.Logf("سائقٌ معلَّق — مصافحةُ البثّ=%d", code)
	if code != http.StatusSwitchingProtocols {
		t.Logf("  **رُدَّت مصافحتُه** — ولا يُتمّ طلبَه لحظيّاً")
		return
	}

	in := wsReader(c)

	// **طابورُ العمل الجديد لا يصله.**
	if got := wsExpect(t, h, in, "drivers:queue", 1500*time.Millisecond); got != nil {
		t.Errorf("**وصلَ المعلَّقَ طابورُ العمل الجديد** — %v (`D14`)", got)
	} else {
		t.Logf("  طابورُ العمل الجديد لم يصله — صحيح")
	}

	// **وغرفتُه هو تصله** — وبها يُتمّ ما بيده.
	if got := wsExpect(t, h, in, "driver:"+drv.ID, 2*time.Second); got == nil {
		t.Errorf("**لم تصله غرفتُه** — **والمعلَّقُ يُتمّ ما بيده** " +
			"(`suspension.go`)")
	} else {
		t.Logf("  غرفتُه وصلته — %v", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والنشطُ لا يُمَسّ**
// ══════════════════════════════════════════════════════════════════════

func TestD14_ActiveUserRealtimeUnaffected(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver(OnShift())

	c, code := wsDial(t, h, drv.Token)
	if c != nil {
		defer c.CloseNow()
	}
	if code != http.StatusSwitchingProtocols {
		t.Fatalf("**رُدَّت مصافحةُ سائقٍ نشط** — %d", code)
	}
	if got := wsExpect(t, h, wsReader(c), "drivers:queue", 2*time.Second); got == nil {
		t.Errorf("**سائقٌ نشطٌ لم يصله الطابور** — **وحارسٌ يمنع الجائزَ " +
			"يُطفأ في أوّل شكوى.**")
	} else {
		t.Logf("نشطٌ — الطابورُ وصله: %v", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتكرارُ يكشف ما لا تكشفه مرّة**
// ══════════════════════════════════════════════════════════════════════

func TestD14_RealtimeEligibilityUnderRepetition(t *testing.T) {
	if testing.Short() {
		t.Skip("تكرارٌ — لا يُشغَّل في الوضع القصير")
	}
	h := New(t)
	f := h.Factory()

	// ── معلَّقٌ ×100 — لا مصافحةَ واسعةٌ ولا بثٌّ عامّ ──────────────
	leaked, dialFail := 0, 0
	for i := 0; i < 100; i++ {
		u := h.Customer()
		setStatus(t, h, u.ID, "suspended")
		c, code := wsDial(t, h, u.Token)
		if code != http.StatusSwitchingProtocols {
			dialFail++
			continue
		}
		if wsExpect(t, h, wsReader(c), "catalog", 300*time.Millisecond) != nil {
			leaked++
		}
		c.CloseNow()
	}
	t.Logf("  معلَّقٌ ×100 — بثٌّ عامٌّ وصل %d · رُدَّت مصافحةٌ %d", leaked, dialFail)
	if leaked > 0 {
		t.Errorf("**وصل بثٌّ عامٌّ إلى %d معلَّقاً من 100.** (`D14`)", leaked)
	}

	// ── محظورٌ ×50 — تُردّ المصافحة ────────────────────────────────
	accepted := 0
	for i := 0; i < 50; i++ {
		u := h.Customer()
		setStatus(t, h, u.ID, "blocked")
		c, code := wsDial(t, h, u.Token)
		if c != nil {
			c.CloseNow()
		}
		if code == http.StatusSwitchingProtocols {
			accepted++
		}
	}
	t.Logf("  محظورٌ ×50 — قُبلت مصافحةُ %d", accepted)
	if accepted > 0 {
		t.Errorf("**قُبلت مصافحةُ %d محظوراً من 50.** (`D14`)", accepted)
	}

	// ── ونشطٌ ×100 — لا منعَ كاذب ──────────────────────────────────
	denied, silent := 0, 0
	for i := 0; i < 100; i++ {
		drv := f.Driver(OnShift())
		c, code := wsDial(t, h, drv.Token)
		if code != http.StatusSwitchingProtocols {
			denied++
			continue
		}
		if wsExpect(t, h, wsReader(c), "drivers:queue", time.Second) == nil {
			silent++
		}
		c.CloseNow()
	}
	t.Logf("  نشطٌ ×100 — رُدَّ %d · صامتٌ %d", denied, silent)
	if denied > 0 || silent > 0 {
		t.Errorf("**نشطٌ: رُدَّ %d وصمت %d من 100** — "+
			"**وحارسٌ يمنع الجائزَ يُطفأ في أوّل شكوى.**", denied, silent)
	}
}
