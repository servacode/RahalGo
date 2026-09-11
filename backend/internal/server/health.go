package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **البابُ العامُّ يبقى أدنى ما يكون** (دورة ٧٠أ)
// ══════════════════════════════════════════════════════════════════════
//
// # وكان يطبع نصَّ الخطأ للعالم
//
// **`err.Error()` من `pgx` يحمل المضيفَ والمنفذَ واسمَ القاعدة واسمَ
// المستخدم**، **ونصُّ `redis` يحمل عنوانَ الخادم.** **وبابٌ بلا
// مصادقةٍ يطبع ذلك يرسم خريطةَ البنية لمن سأله.**
//
// **وهذا نقصانُ تشدّدٍ لا زيادةُ تشخيص**: **الحقلان يبقيان بأسمائهما**
// — يقرؤهما عقدُ البوّابة (`gate.HealthCovers`) — **وقيمتاهما من
// معجمٍ مغلق.**
//
// # والتفصيلُ خلف بابه
//
// **ومن أراد لماذا سقطت القاعدةُ يسأل `GET /admin/ops/health`**
// بقدرة `observability.read`. **والعامُّ يقول «حيٌّ أم لا» لا غير.**
//
// **و`/healthz` بابُ نجاةٍ مفتوحٌ في بوّابة التحديث** (`min_version.go`)
// — **فما يُضاف إليه يصير عامّاً فعلاً لا نظراً.**

// healthState **معجمُ حالِ التبعيّة** — مغلقٌ بحرفين.
const (
	depOK   = "ok"
	depDown = "down"
)

type healthStatus struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 5*time.Second)
	defer cancel()

	st := healthStatus{Status: "ok", Postgres: depOK, Redis: depOK}
	code := http.StatusOK

	if err := s.pg.Ping(ctx); err != nil {
		// **والسببُ يُسجَّل ولا يُنشَر** — من يملك السجلَّ يملك حقَّه.
		s.logger.Error("الصحّة: القاعدةُ لا تُجيب", "error", err)
		st.Status, st.Postgres = "degraded", depDown
		code = http.StatusServiceUnavailable
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		s.logger.Error("الصحّة: الذاكرةُ لا تُجيب", "error", err)
		st.Status, st.Redis = "degraded", depDown
		code = http.StatusServiceUnavailable
	}

	httpx.JSON(w, code, st)
}
