package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **ارتباطُ الأثر — المرحلة ٨ب**
// ══════════════════════════════════════════════════════════════════════

// TestNodeMarginSeparates **الهامشُ يفصل — وهو ما قِيس.**
//
// **وقِيس أنّ تقاطعَ العقد يفصل بهامشٍ وسيطُه ١٫٠٠٠** والأضلاعُ
// الموجَّهةُ ٠٫٥٣٨ **ولا تُحسّن الدقّة** — فالعقدُ هي الحكم.
func TestNodeMarginSeparates(t *testing.T) {
	planned := []int64{1, 2, 3, 4, 5}
	for _, tc := range []struct {
		name    string
		matched []int64
		want    float64
	}{
		{"على المخطَّط تماماً", []int64{2, 3, 4}, 1.0},
		{"على غيره تماماً", []int64{9, 8, 7}, -1.0},
		{"نصفٌ ونصف", []int64{2, 3, 8, 9}, 0.0},
		{"فارغ", nil, 0.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NodeMargin(tc.matched, planned)
			if got != tc.want {
				t.Fatalf("هامش = %.2f — والمنتظر %.2f", got, tc.want)
			}
		})
	}
}

// TestSharedJunctionsFailSafe **عقدٌ مشتركةٌ تُلبِس ولا تُكذّب.**
//
// **قُيس** (٢٠٢٦-٠٨-٢١، ٤٠٠ زوجِ طرقٍ متوازية): **٢٠٪ منها
// تتقاسم عقداً**، ووسيطُ المشترك **١٠٪** وأقصاه **٣٣٪**.
//
// # وما يفعله التلوّث
//
// **يدفع الحكمَ إلى الالتباس لا إلى الخطأ**: بتلوّث ثلاثين
// بالمئة يصير الهامشُ −٠٫٤٠ — **دونَ العتبة فيُعلَن التباساً.**
//
// **وذلك مقصود**: من تلوّث دليلُه لا يُحكم عليه، **ولا يُحكم
// له خطأً أيضاً.** وهو من أسباب أنّ الكشفَ المقيس ٣٤٫٣٪
// لا مئة.
func TestSharedJunctionsFailSafe(t *testing.T) {
	planned := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// ── تلوّثٌ أقصى (٣٠٪) — التباسٌ لا خطأ ─────────
	heavy := NodeMargin([]int64{1, 2, 3, 91, 92, 93, 94, 95, 96, 97}, planned)
	if heavy >= MinNodeMargin {
		t.Fatalf("حُكم له خطأً: %.2f", heavy)
	}
	if heavy <= -MinNodeMargin {
		t.Fatalf("تلوّثٌ ثقيلٌ يجب أن يُلبِس لا أن يحسم: %.2f", heavy)
	}

	// ── وبالتلوّث الوسيط (١٠٪) يُحسم ────────────
	median := NodeMargin([]int64{1, 91, 92, 93, 94, 95, 96, 97, 98, 99}, planned)
	if median > -MinNodeMargin {
		t.Fatalf("التلوّثُ الوسيط منع الحكم: %.2f", median)
	}

	// ── **ولا يُقلَب الحكمُ أبداً** ─────────────────
	onPlanned := NodeMargin([]int64{2, 3, 4, 5, 6, 7, 8, 9, 91, 92}, planned)
	if onPlanned <= -MinNodeMargin {
		t.Fatalf("سائقٌ على المخطّط حُكم عليه موازياً: %.2f", onPlanned)
	}
}

// TestMatchSendsGuardedRadii **نصفُ القطر من الدقّة لا مفتوح.**
func TestMatchSendsGuardedRadii(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "Ok",
			"matchings": []any{map[string]any{
				"confidence": 0.95,
				"legs": []any{map[string]any{
					"annotation": map[string]any{"nodes": []int64{1, 2, 3}},
				}},
			}},
		})
	}))
	defer srv.Close()

	trace := []TracePoint{
		{Lat: 35.95, Lng: 39.01, AccuracyM: 5, AtMs: 1_000},
		{Lat: 35.951, Lng: 39.011, AccuracyM: 8, AtMs: 2_000},
		{Lat: 35.952, Lng: 39.012, AccuracyM: 40, AtMs: 3_000},
		{Lat: 35.953, Lng: 39.013, AccuracyM: 5, AtMs: 4_000},
	}
	res, err := New(srv.URL).Match(context.Background(), trace)
	if err != nil {
		t.Fatalf("مطابقة: %v", err)
	}
	if res.Confidence != 0.95 || len(res.Nodes) != 3 {
		t.Fatalf("ردٌّ غيرُ مقروء: %+v", res)
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("استعلامٌ لا يُقرأ: %v — %q", err, raw)
	}
	rad := q.Get("radiuses")
	if rad == "" {
		t.Fatalf("بلا نصفِ قطر: %q", raw)
	}
	if strings.Contains(rad, "unlimited") {
		t.Fatalf("نصفُ قطرٍ مفتوح: %q", rad)
	}
	// **ثلاثةُ أضعاف الدقّة بحدٍّ أدنى عشرة وأقصى مئة.**
	if rad != "15;24;100;15" {
		t.Fatalf("radiuses = %q", rad)
	}
	if strings.Contains(raw, ";") {
		t.Fatalf("فاصلةٌ خامّةٌ تُسقط المعاملات: %q", raw)
	}
}

// TestMatchRejectsBadTraceSize **سقفُ القراءات — البند ٣٢.**
func TestMatchRejectsBadTraceSize(t *testing.T) {
	c := New("http://x")
	if _, err := c.Match(context.Background(), nil); err == nil {
		t.Fatal("قُبل أثرٌ فارغ")
	}
	big := make([]TracePoint, MaxTraceFixes+1)
	if _, err := c.Match(context.Background(), big); err == nil {
		t.Fatal("قُبل أثرٌ فوق السقف")
	}
}

// TestNodesUsesSnapRadius **وجلبُ العقد محميٌّ كسائر النداءات.**
func TestNodesUsesSnapRadius(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "Ok",
			"routes": []any{map[string]any{
				"legs": []any{map[string]any{
					"annotation": map[string]any{"nodes": []int64{7, 8}},
				}},
			}},
		})
	}))
	defer srv.Close()
	ns, err := New(srv.URL).Nodes(context.Background(),
		Point{35.95, 39.01}, Point{35.96, 39.02})
	if err != nil || len(ns) != 2 {
		t.Fatalf("عقد: %v %v", ns, err)
	}
	if !strings.Contains(raw, "radiuses=150%3B250") &&
		!strings.Contains(raw, "radiuses=150;250") {
		t.Fatalf("بلا حدِّ التقاط: %q", raw)
	}
	if !strings.Contains(raw, "annotations=nodes") {
		t.Fatalf("بلا تعليقاتِ عقد: %q", raw)
	}
}

// TestCorrelationThresholdsAreMeasured **العتباتُ مُعايَرة.**
func TestCorrelationThresholdsAreMeasured(t *testing.T) {
	// **ثقة ≥٠٫٩٠ وهامش ≥٠٫٨٠ → كشف ٦٤٫٣٪ · إيجابٌ كاذب ٠/٩٦.**
	if MinMatchConfidence != 0.90 {
		t.Fatalf("عتبةُ الثقة تبدّلت: %.2f", MinMatchConfidence)
	}
	if MinNodeMargin != 0.80 {
		t.Fatalf("عتبةُ الهامش تبدّلت: %.2f", MinNodeMargin)
	}
	// **وثمانٍ هي نافذةُ القياس، والسقفُ ضِعفُها.**
	if MaxTraceFixes < 8 || MaxTraceFixes > 32 {
		t.Fatalf("سقفُ القراءات خارجَ المعقول: %d", MaxTraceFixes)
	}
}
