package server

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/routing"
)

// ══════════════════════════════════════════════════════════════════════
// **ارتباطُ الأثر بالمسار — المرحلة ٨ب**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢١.)
//
// **يسأل التطبيقُ: أنا على المسار الذي أعطيتَني أم على غيره؟**
// **ويردّ الخادمُ حكماً محايداً** — لا عقدَ ولا ثقةَ محرّكٍ ولا JSON
// منه. **والجوّالُ لا يعرف أنّ ثمّة محرّكَ مسارات.**

// ── دلالاتُ الردّ — البند ١٦ ─────────────────────────────────────────
const (
	// correlationOnPlanned **الأثرُ على المسار المخطَّط.**
	correlationOnPlanned = "ON_PLANNED_ROUTE"
	// correlationParallel **الأثرُ على طريقٍ آخر.**
	correlationParallel = "PARALLEL_ROUTE"
	// correlationAmbiguous **لا يُحسم** — ولا قرارَ يُبنى عليه.
	correlationAmbiguous = "AMBIGUOUS"
	// correlationInsufficient **لا سياقَ أو لا بيانات.**
	correlationInsufficient = "INSUFFICIENT_DATA"
)

// correlationTTL **عمرُ سياق الارتباط.**
//
// ══════════════════════════════════════════════════════════════════════
// **وليس عمرَ لوحة الاختيار — البند ٥**
// ══════════════════════════════════════════════════════════════════════
//
// **`RouteChoices` تُقادم بعد عشر دقائق** لأنّ الاختيارَ يشيخ.
// **والمسارُ المختارُ يُقاد أطولَ من ذلك بكثير** — رحلةٌ بين المدن
// ساعات.
//
// **فالعمرُ مربوطٌ بالرحلة لا بالاختيار**: يُمسح عند تركيب مسارٍ
// جديدٍ أو تبدّلِ وجهةٍ أو انتهاء الطلب، **وهذا سقفٌ للمهجور فقط**
// (البند ٥: `bounded orphan cleanup`).
const correlationTTL = 6 * time.Hour

// correlationKey **مفتاحٌ يشمل الطلبَ والسائقَ والوجهةَ والمسار.**
//
// **ولا يُقبل `routeId` وحدَه** (البند ٤): المفتاحُ يُبنى من هويّة
// السائق المُصادَق عليها وطلبِه الحاليّ، **فمعرّفٌ صحيحٌ لطلبٍ آخرَ
// لا يفتح شيئاً.**
func correlationKey(driverID, orderID, target, routeID string) string {
	return "corr:v1:" + driverID + ":" + orderID + ":" + target + ":" + routeID
}

// storeCorrelation **يحفظ عقدَ المسار ليُقارَن بها لاحقاً.**
//
// **ويُنادى لكلّ مسارٍ يُسلَّم بـ`correlation=true`** — من الخبيئة أو
// من المحرّك سواء (البند ٦): **فلا يكون مسارٌ قابلاً للمقارنة وآخرُ
// لا لمجرّد أنّه جاء مخبوءاً.**
func (s *Server) storeCorrelation(
	ctx context.Context, driverID, orderID, target, routeID string,
	from, to routing.Point, fingerprint string,
) {
	if s.rdb == nil || routeID == "" {
		return
	}
	key := correlationKey(driverID, orderID, target, routeID)
	// **وإن كان محفوظاً فلا يُعاد جلبُه** — النداءُ إلى المحرّك
	// يُوفَّر، **والعقدُ لا تتبدّل لمسارٍ واحد.**
	if n, err := s.rdb.Exists(ctx, key).Result(); err == nil && n > 0 {
		s.rdb.Expire(ctx, key, correlationTTL)
		return
	}
	nodes, err := s.routeNodes(ctx, from, to)
	if err != nil || len(nodes) < 2 {
		return
	}
	raw, err := json.Marshal(routing.CorrelationContext{
		Nodes: nodes, Fingerprint: fingerprint,
	})
	if err != nil {
		return
	}
	s.rdb.Set(ctx, key, raw, correlationTTL)
}

// routeNodes **عقدُ المسار من المحرّك** — وفارغٌ إن لم يكن ثمّة محرّك.
func (s *Server) routeNodes(ctx context.Context, from, to routing.Point) ([]int64, error) {
	type nodeSource interface {
		Nodes(context.Context, routing.Point, routing.Point) ([]int64, error)
	}
	src, ok := s.route.(nodeSource)
	if !ok {
		return nil, routing.ErrNoEngine
	}
	return src.Nodes(ctx, from, to)
}

// ══════════════════════════════════════════════════════════════════════
// **المنفذ**
// ══════════════════════════════════════════════════════════════════════

type correlationRequest struct {
	RouteID string `json:"route_id"`
	Fixes   []struct {
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
		AccuracyM float64 `json:"accuracy_m"`
		AtMs      int64   `json:"at_ms"`
	} `json:"fixes"`
}

// handleRoadCorrelation **يحكم على أثرٍ مقابلَ المسار النافذ.**
//
// **والحمايةُ في الخادم لا في العميل** (البند ٣٢): سائقٌ مُصادَقٌ
// عليه، وطلبٌ له، وسقفُ قراءات، وإحداثيّاتٌ وأزمنةٌ تُفحص.
func (s *Server) handleRoadCorrelation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	driverID := userIDFrom(r)

	var req correlationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}
	if req.RouteID == "" ||
		len(req.Fixes) < routing.MinTraceFixes ||
		len(req.Fixes) > routing.MaxTraceFixes {
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}

	// ── الطلبُ لهذا السائق فعلاً؟ — البند ٤ ─────────────────────────
	//
	// **ولا يكفي أنّ المعرّفَ يحوي رقمَ الطلب**: البصمةُ ليست إذناً،
	// **والإذنُ من القاعدة.**
	var status string
	var pickLat, pickLng, dropLat, dropLng float64
	err := s.pg.QueryRow(r.Context(), `
		SELECT o.status,
		       ST_Y(COALESCE(o.pickup_override, m.location)::geometry),
		       ST_X(COALESCE(o.pickup_override, m.location)::geometry),
		       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry)
		FROM orders o
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1::uuid AND o.driver_id = $2::uuid`,
		orderID, driverID).
		Scan(&status, &pickLat, &pickLng, &dropLat, &dropLng)
	if err != nil {
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}
	target := "pickup"
	if status != "assigned" && status != "at_pickup" {
		target = "dropoff"
	}

	// ── سياقُ الارتباط ─────────────────────────────────────────────
	ctxData := s.loadCorrelation(r.Context(), driverID, orderID, target, req.RouteID)
	if !ctxData.Usable() {
		// **ولا تخمينَ عند فقد الهويّة** (البند ١١ من الإغلاق):
		// **لا مقارنةَ بآخرِ مسارٍ في الخبيئة ولا بموصًى بها قديمة.**
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}

	// ── الأثرُ يُفحص ثمّ يُطابَق ────────────────────────────────────
	trace := make([]routing.TracePoint, 0, len(req.Fixes))
	var newest int64
	for _, f := range req.Fixes {
		if f.Lat < -90 || f.Lat > 90 || f.Lng < -180 || f.Lng > 180 ||
			math.IsNaN(f.Lat) || math.IsNaN(f.Lng) {
			s.correlationReply(w, correlationInsufficient, 0)
			return
		}
		if f.AtMs <= 0 {
			s.correlationReply(w, correlationInsufficient, 0)
			return
		}
		if f.AtMs > newest {
			newest = f.AtMs
		}
		trace = append(trace, routing.TracePoint{
			Lat: f.Lat, Lng: f.Lng, AccuracyM: f.AccuracyM, AtMs: f.AtMs,
		})
	}

	type matcher interface {
		Match(context.Context, []routing.TracePoint) (*routing.MatchResult, error)
	}
	m, ok := s.route.(matcher)
	if !ok {
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}
	res, err := m.Match(r.Context(), trace)
	if err != nil || res == nil || len(res.Nodes) == 0 {
		if err != nil && !errors.Is(err, routing.ErrNoRoute) {
			s.logger.Warn("الارتباط: تعذّرت المطابقة", "error", err)
		}
		s.correlationReply(w, correlationInsufficient, 0)
		return
	}

	age := time.Now().UnixMilli() - newest
	if age < 0 {
		age = 0
	}

	// ── الحكم ──────────────────────────────────────────────────────
	//
	// **والثقةُ شرطٌ لازمٌ لا كافٍ** — قِيس أنّ ٠٫٩٥ تصيب ٧٠٪ فقط
	// على الحالات القاسية.
	if res.Confidence < routing.MinMatchConfidence {
		s.correlationReply(w, correlationAmbiguous, age)
		return
	}
	margin := routing.NodeMargin(res.Nodes, ctxData.Nodes)
	switch {
	case margin >= routing.MinNodeMargin:
		s.correlationReply(w, correlationOnPlanned, age)
	case margin <= -routing.MinNodeMargin:
		s.correlationReply(w, correlationParallel, age)
	default:
		s.correlationReply(w, correlationAmbiguous, age)
	}
}

func (s *Server) loadCorrelation(
	ctx context.Context, driverID, orderID, target, routeID string,
) *routing.CorrelationContext {
	if s.rdb == nil {
		return nil
	}
	raw, err := s.rdb.Get(ctx, correlationKey(driverID, orderID, target, routeID)).Bytes()
	if err != nil {
		return nil
	}
	var c routing.CorrelationContext
	if json.Unmarshal(raw, &c) != nil {
		return nil
	}
	return &c
}

// correlationReply **ردٌّ محايدٌ لا يحمل شيئاً من المحرّك.**
//
// **ولا `offsetM` ولا عقدَ ولا ثقةَ خام** (البند ١٦): الهاتفُ لا
// يحتاجها، **وما لا يُحتاج لا يُرسَل.**
func (s *Server) correlationReply(w http.ResponseWriter, status string, ageMs int64) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":          status,
		"evidence_age_ms": ageMs,
	})
}
