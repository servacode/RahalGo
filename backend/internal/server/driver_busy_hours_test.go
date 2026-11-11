package server

// **أفضل أوقات الشغل** — الحكمُ نقيّاً، ثمّ النقطةُ كما يناديها التطبيق.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPickBusiestHours(t *testing.T) {
	var c [24]int
	c[13], c[14], c[19], c[20], c[3] = 50, 40, 60, 40, 5
	got := pickBusiestHours(c, 3, 30)
	if len(got) != 3 || got[0].Hour != 19 || got[1].Hour != 13 || got[2].Hour != 14 {
		t.Fatalf("الترتيب خاطئ (التعادل للأبكر): %+v", got)
	}

	// قلّةُ البيانات لا تُقال رأياً.
	var few [24]int
	few[3] = 4
	if got := pickBusiestHours(few, 3, 30); len(got) != 0 {
		t.Fatalf("أقلُّ من الحدّ يجب أن يردّ فارغاً: %+v", got)
	}

	// ساعتان فقط فيهما طلبات: لا تُملأ الثالثة بصفر.
	var two [24]int
	two[9], two[10] = 20, 15
	if got := pickBusiestHours(two, 3, 30); len(got) != 2 {
		t.Fatalf("لا ساعةَ بصفر طلبات: %+v", got)
	}
}

func TestDriverBusyHoursEndpoint(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	resetBusyHoursCache()
	t.Cleanup(resetBusyHoursCache)

	// ساعةٌ بعيدةٌ عن «الآن» حتّى لا تختلط بطلبات الاختبارات الأخرى.
	damascus, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		t.Skipf("لا قاعدةَ مناطق زمنيّة: %v", err)
	}
	target := (time.Now().In(damascus).Hour() + 12) % 24

	var ids []string
	for i := 0; i < 60; i++ {
		ids = append(ids, f.dispatchingOrder(t, 1000, 100))
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(ctx, `DELETE FROM orders WHERE id = ANY($1::uuid[])`, ids)
	})
	// ثلاثةُ أيّامٍ مضت، في تلك الساعة بتوقيت دمشق.
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET created_at =
			(date_trunc('day', now() AT TIME ZONE 'Asia/Damascus') - interval '3 days'
			 + make_interval(hours => $2)) AT TIME ZONE 'Asia/Damascus'
		WHERE id = ANY($1::uuid[])`, ids, target); err != nil {
		t.Fatalf("تعذّر ضبط الوقت: %v", err)
	}

	call := func() (int, []busyHour) {
		req := httptest.NewRequest(http.MethodGet, "/driver/busy-hours", nil)
		w := httptest.NewRecorder()
		f.srv.handleDriverBusyHours(w, req)
		var body struct {
			Data struct {
				Days int        `json:"days"`
				Top  []busyHour `json:"top"`
			} `json:"data"`
		}
		if w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("ردّ غير مقروء: %v: %s", err, w.Body.String())
			}
			if body.Data.Days != busyHoursDays {
				t.Fatalf("days = %d", body.Data.Days)
			}
		}
		return w.Code, body.Data.Top
	}

	code, top := call()
	if code != http.StatusOK {
		t.Fatalf("الرمز %d", code)
	}
	if len(top) == 0 || len(top) > busyHoursTop {
		t.Fatalf("عدد الساعات %d", len(top))
	}
	if top[0].Hour != target || top[0].Orders < 60 {
		t.Fatalf("الساعةُ الأولى يجب أن تكون %d بـ٦٠ طلباً فأكثر: %+v", target, top)
	}

	// والنداءُ الثاني من الذاكرة: لا يتغيّر ولو تغيّرت القاعدة.
	if _, err := f.pool.Exec(ctx, `DELETE FROM orders WHERE id = ANY($1::uuid[])`, ids); err != nil {
		t.Fatalf("تعذّر الحذف: %v", err)
	}
	_, again := call()
	if len(again) == 0 || again[0].Hour != target {
		t.Fatalf("كان يجب أن يُقرأ من الذاكرة: %+v", again)
	}
}
