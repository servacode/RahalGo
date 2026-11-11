package server

// ══════════════════════════════════════════════════════════════════════
// **أفضل أوقات الشغل** (قرارُ المالك ٢٠٢٦-١١-١١)
// ══════════════════════════════════════════════════════════════════════
//
// بطاقةٌ في لوحة السائق تقول له متى تكثر الطلباتُ في المدينة: **أكثرُ ثلاثِ
// ساعاتٍ في اليوم طلباً خلال آخر ١٤ يوماً**، بتوقيت دمشق.
//
// # أرقامٌ مجمَّعةٌ لا بيانات أشخاص
//
// **عدّادٌ لكلّ ساعة (٠–٢٣)** يُحسب من عدد الطلبات المنشأة فيها — لا زبونَ ولا
// متجرَ ولا سائقَ في الردّ. **وهو للمدينة كلّها** لا لمتجرٍ ولا لسائق.
//
// # ولا يُحسب عند كلّ فتحة
//
// الاستعلامُ يمسح أسبوعين من الطلبات، **والسائقون يفتحون لوحتَهم مئاتِ المرّات
// في اليوم**؛ فيُحفظ الحاصلُ في الذاكرة ساعةً (`busyHoursTTL`).
//
// # وقلّةُ البيانات لا تُقال رأياً
//
// **أقلُّ من `busyHoursMinOrders` طلباً في أسبوعين ضجيج**: «أكثرُ وقتٍ الثالثةُ
// فجراً» بسبب أربعة طلبات تضلّل سائقاً. فيُردّ `top` فارغاً **والتطبيقُ يُخفي
// البطاقة.**

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const (
	// busyHoursDays **نافذةُ العدّ** بالأيّام.
	busyHoursDays = 14
	// busyHoursTop **كم ساعةً تُعرض.**
	busyHoursTop = 3
	// busyHoursMinOrders **أقلُّ عددٍ يُعتدّ به** في النافذة كلّها.
	busyHoursMinOrders = 30
	// busyHoursTTL **عمرُ الحاصل في الذاكرة.**
	busyHoursTTL = time.Hour
)

// busyHour **ساعةٌ (٠–٢٣ بتوقيت دمشق) وعددُ ما أُنشئ فيها من طلبات.**
type busyHour struct {
	Hour   int `json:"hour"`
	Orders int `json:"orders"`
}

var busyHoursCache struct {
	mu   sync.Mutex
	at   time.Time
	data []busyHour
}

// resetBusyHoursCache **يمسح الحاصلَ المحفوظ** — للاختبارات.
func resetBusyHoursCache() {
	busyHoursCache.mu.Lock()
	busyHoursCache.at = time.Time{}
	busyHoursCache.data = nil
	busyHoursCache.mu.Unlock()
}

// pickBusiestHours **أكثرُ `n` ساعاتٍ طلباً** من عدّادات الساعات، بالأكثر أوّلاً
// (والتعادلُ للأبكر). **وفارغٌ إن لم يبلغ المجموعُ `minTotal`.**
func pickBusiestHours(counts [24]int, n, minTotal int) []busyHour {
	total := 0
	for _, c := range counts {
		total += c
	}
	if total < minTotal || total == 0 {
		return []busyHour{}
	}
	used := [24]bool{}
	out := make([]busyHour, 0, n)
	for len(out) < n {
		best := -1
		for h, c := range counts {
			if used[h] || c <= 0 {
				continue
			}
			if best == -1 || c > counts[best] {
				best = h
			}
		}
		if best == -1 {
			break
		}
		used[best] = true
		out = append(out, busyHour{Hour: best, Orders: counts[best]})
	}
	return out
}

// loadBusyHours **يعدّ الطلبات بالساعة** في آخر `busyHoursDays` يوماً.
func loadBusyHours(ctx context.Context, pg *pgxpool.Pool, now time.Time) ([]busyHour, error) {
	rows, err := pg.Query(ctx, `
		SELECT extract(hour FROM created_at AT TIME ZONE 'Asia/Damascus')::int, count(*)::int
		FROM orders
		WHERE created_at >= $1
		GROUP BY 1`, now.AddDate(0, 0, -busyHoursDays))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var counts [24]int
	for rows.Next() {
		var h, c int
		if err := rows.Scan(&h, &c); err != nil {
			return nil, err
		}
		if h >= 0 && h < 24 {
			counts[h] = c
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pickBusiestHours(counts, busyHoursTop, busyHoursMinOrders), nil
}

// handleDriverBusyHours **أفضل أوقات الشغل** — `GET /driver/busy-hours`.
func (s *Server) handleDriverBusyHours(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	busyHoursCache.mu.Lock()
	if busyHoursCache.data != nil && now.Sub(busyHoursCache.at) < busyHoursTTL {
		data := busyHoursCache.data
		busyHoursCache.mu.Unlock()
		httpx.JSON(w, http.StatusOK, map[string]any{"days": busyHoursDays, "top": data})
		return
	}
	busyHoursCache.mu.Unlock()

	data, err := loadBusyHours(r.Context(), s.pg, now)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	busyHoursCache.mu.Lock()
	busyHoursCache.at, busyHoursCache.data = now, data
	busyHoursCache.mu.Unlock()
	httpx.JSON(w, http.StatusOK, map[string]any{"days": busyHoursDays, "top": data})
}
