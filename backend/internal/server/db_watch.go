package server

// ══════════════════════════════════════════════════════════════════════
// **القاعدةُ متوقّفة؟ يُقال فوراً لا بعد ربع دقيقة** (اختبارُ التحمّل ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس بتجميد القاعدة عشرَ ثوانٍ**: كلُّ نداءٍ علق خمسَ عشرةَ ثانيةً حتّى قطعه
// الهاتف — **شاشةٌ تدور ولا تقول شيئاً**، ثمّ يعيد الزبونُ فيتراكم الضغط.
//
// **فحارسٌ يسأل القاعدةَ كلَّ ثانية** (`Ping` بمهلة ثانية): **فشلتان متتاليتان**
// تُعلنانها متوقّفة، **فيُردّ كلُّ نداءٍ فوراً ٥٠٣ «المنصّة مشغولة»**؛ **ونجاحٌ واحدٌ**
// يعيدها. **وأبوابُ الصحّة والهويّة تمرّ** — هي ما يسأل عن الحال.
//
// **ولا يعمل إلّا حيث شُغّل** (`StartDBWatch` في الخادم الحيّ) — الاختباراتُ بلاه.

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errServiceBusy = httpx.NewError(http.StatusServiceUnavailable,
	"service_busy", "errors.service_busy")

// dbDown **أمتوقّفةٌ القاعدةُ الآن؟** — يكتبه الحارسُ ويقرؤه الوسيط.
var dbDownFlag atomic.Bool

// StartDBWatch **يبدأ الحارس** — ويقف بانتهاء السياق.
func (s *Server) StartDBWatch(ctx context.Context) {
	if s == nil || s.pg == nil {
		return
	}
	go func() {
		fails := 0
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			pctx, cancel := context.WithTimeout(ctx, time.Second)
			err := s.pg.Ping(pctx)
			cancel()
			if err == nil {
				if dbDownFlag.Swap(false) {
					s.logger.Warn("القاعدةُ عادت — تُستقبَل النداءاتُ من جديد")
				}
				fails = 0
				continue
			}
			fails++
			if fails >= 2 && !dbDownFlag.Swap(true) {
				s.logger.Error("القاعدةُ لا تردّ — تُردّ النداءاتُ فوراً «المنصّة مشغولة»", "error", err)
			}
		}
	}()
}

// dbGate **وسيطٌ يردّ فوراً والقاعدةُ متوقّفة.**
func dbGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dbDownFlag.Load() && !strings.HasPrefix(r.URL.Path, "/health") &&
			!strings.HasSuffix(r.URL.Path, "/public/identity") {
			w.Header().Set("Retry-After", "5")
			httpx.Error(w, errServiceBusy)
			return
		}
		next.ServeHTTP(w, r)
	})
}
