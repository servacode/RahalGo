package server

// ══════════════════════════════════════════════════════════════════════
// **«سلّمت البضاعة» — نهايةُ مشوار الإرجاع** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /driver/orders/{id}/goods-handed
//	{lat?, lng?}  ←  {handed_at}
//
// **الزبونُ رفض أو ألغى بعد الاستلام فأعاده المكتبُ بالبضاعة** — والطلبُ أُنهي
// فشلاً، **والسائقُ في مشوارٍ مرسومٍ إلى المكتب أو المتجر.** هذا الزرُّ يكتب
// وقتَ التسليم وموضعَه، **فتراه الإدارةُ على بطاقة الطلب ويخرج المشوارُ من يده.**
//
// **والحسابُ كلُّه في المحرّك** (`orders/return_trip.go`): لسائق الطلب وحدَه،
// ومرّةً واحدة، **وعلى مشوارٍ قائمٍ لا غير.** والموضعُ اختياريّ — **وغيابُه يُقرأ
// من آخر موضعٍ حديثٍ عند الخادم.**

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleDriverGoodsHanded(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	}
	// **والجسدُ اختياريّ** — زرٌّ بلا موضعٍ لا يُردّ لأنّ الموقعَ لم يُلتقط.
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(&req); err != nil &&
		!errors.Is(err, io.EOF) {
		s.respondErr(w, errValidation)
		return
	}
	// **وزوجٌ صالحٌ أو لا شيء** — نصفُ نقطةٍ ليس نقطة.
	if (req.Lat == nil) != (req.Lng == nil) {
		s.respondErr(w, errBadPoint)
		return
	}
	if req.Lat != nil {
		la, ln := *req.Lat, *req.Lng
		if math.IsNaN(la) || math.IsNaN(ln) || math.IsInf(la, 0) || math.IsInf(ln, 0) ||
			la < -90 || la > 90 || ln < -180 || ln > 180 {
			s.respondErr(w, errBadPoint)
			return
		}
	}
	oid := chi.URLParam(r, "id")
	at, err := s.orders.HandGoods(r.Context(), oid, userIDFrom(r), req.Lat, req.Lng,
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "driver.goods_handed", "order", oid, map[string]any{
				"has_point": req.Lat != nil,
			})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"handed_at": at})
}
