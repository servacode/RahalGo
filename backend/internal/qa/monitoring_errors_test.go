package qa

// ══════════════════════════════════════════════════════════════════════
//  **بابُ أخطاء الخادم — لمالك قدرة الرصد وحدَه** (مراقبةُ المنصّة ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// `GET /admin/monitoring/errors` **تفصيلٌ تقنيٌّ كصحّة المنصّة** — أنماطُ
// مساراتٍ ورموزٌ وأعداد. **فيُسأل بالموجّه الحقيقيّ**: بلا جلسةٍ ٤٠١،
// وبقدرةٍ أخرى (الطلبات، التحليلات) ٤٠٣، وبـ`observability.read` ٢٠٠.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

const monitoringErrorsPath = "/api/v1/admin/monitoring/errors"

func TestMonitoringErrors_ObservabilityOnly(t *testing.T) {
	hh := New(t)

	if r := hh.GET(monitoringErrorsPath, ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("**بلا جلسةٍ يجب ٤٠١** — وردَّ %d", r.Code)
	}

	for _, c := range []struct {
		role string
		cap  authz.Capability
	}{
		{"qa-mon-orders", authz.OrdersRead},
		{"qa-mon-analytics", authz.AnalyticsRead},
	} {
		capRole(t, hh, c.role, c.cap)
		_, tok := capUser(t, hh, c.role)
		if r := hh.GET(monitoringErrorsPath, tok); r.Code != http.StatusForbidden {
			t.Fatalf("**`%s` فتحت بابَ الأخطاء** — وردَّ %d", c.cap, r.Code)
		}
	}

	capRole(t, hh, "qa-mon-seeing", authz.ObservabilityRead)
	_, tok := capUser(t, hh, "qa-mon-seeing")
	r := hh.GET(monitoringErrorsPath+"?hours=6", tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**مالكُ القدرة يُقرئ** — وردَّ %d: %s", r.Code, r.Body)
	}
	var env struct {
		Data struct {
			Total  *int64           `json:"total"`
			Groups []map[string]any `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("الردّ: %v — %s", err, r.Body)
	}
	if env.Data.Total == nil || env.Data.Groups == nil {
		t.Fatalf("**الردُّ بلا شكله** (total/groups) — %s", r.Body)
	}
	// **ولا رابطَ خامّاً ولا هاتف** — أنماطُ مساراتٍ وحدَها.
	if strings.Contains(string(r.Body), "+963") {
		t.Fatalf("**رقمُ هاتفٍ في ردّ الأخطاء** — %s", r.Body)
	}
}
