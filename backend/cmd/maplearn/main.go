// Command maplearn **جولةٌ واحدةٌ من «الخريطة من حركة السائقين»** (قرارُ المالك ٢٠٢٦-١١-٠٣).
//
// يقيس بُعدَ نقاط `map_trace_points` غيرِ المقيسة عن أقرب طريقٍ في خريطتنا، ثمّ يكشف الشوارعَ الناقصة
// (نقاطٌ بعيدةٌ تتكرّر في أكثر من يوم) ويحفظها مقترحاتٍ للموافقة في اللوحة. هي نفسُ الجولة التي
// يشغّلها المحرّكُ يوميّاً — هنا للتشغيل اليدويّ.
//
// البيئة: DATABASE_URL (مطلوب) · OSRM_URL أو VALHALLA_URL · ROUTING_ENGINE (osrm|valhalla).
// الخيارات: -batch (نقاط الجولة) · -workers · -far (متر) · -min-days · -min-points.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

func main() {
	opt := maplearn.DefaultOptions()
	flag.IntVar(&opt.Batch, "batch", opt.Batch, "أكثر نقاطٍ تُقاس في الجولة")
	flag.IntVar(&opt.Workers, "workers", opt.Workers, "طلباتٌ متوازية")
	flag.Float64Var(&opt.FarM, "far", opt.FarM, "بُعدٌ (م) يُعدّ بعده الموضعُ خارج الخريطة")
	flag.IntVar(&opt.Detect.MinDays, "min-days", opt.Detect.MinDays, "أقلُّ أيّامٍ مختلفة")
	flag.IntVar(&opt.Detect.MinPoints, "min-points", opt.Detect.MinPoints, "أقلُّ نقاطٍ في الخطّ")
	flag.Parse()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "maplearn: DATABASE_URL مطلوب")
		os.Exit(2)
	}
	engine := strings.ToLower(strings.TrimSpace(os.Getenv("ROUTING_ENGINE")))
	rf := maplearn.NewRoadFinder(engine, os.Getenv("OSRM_URL"), os.Getenv("VALHALLA_URL"))
	if rf == nil {
		fmt.Fprintln(os.Stderr, "maplearn: لا OSRM_URL ولا VALHALLA_URL")
		os.Exit(2)
	}
	ctx := context.Background()
	pg, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "maplearn: تعذّر الاتصال بالقاعدة:", err)
		os.Exit(2)
	}
	defer pg.Close()

	res, err := maplearn.Run(ctx, pg, rf, opt)
	fmt.Printf("maplearn: قِيس %d · نقاطٌ بعيدة %d · شوارعُ مقترحة %d\n", res.Measured, res.FarPoints, res.Candidates)
	if err != nil {
		fmt.Fprintln(os.Stderr, "maplearn:", err)
		os.Exit(1)
	}
}
