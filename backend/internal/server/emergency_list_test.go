package server

// **شاشةُ الطوارئ تُقرأ وفيها بلاغ — لا فارغةً وحدَها.**
//
// # ولماذا حارسٌ لهذه بالذات
//
// **كانت تردّ خمسَمئة كلَّما وُجد بلاغٌ واحد**: `created_at` وقتٌ يُقرأ في
// نصّ، فترمي `cannot scan timestamptz into *string`. **والفارغُ يمرّ** — لا
// صفَّ فلا مسحَ فلا خطأ.
//
// **فبقيت الشاشةُ تُقرأ سليمةً طولَ التطوير**، ولا تسقط إلّا يومَ يقع أوّلُ
// طارئ. (كُشفت في اختبارٍ شامل ٢٠٢٦-٠٨-١٠ ببلاغٍ من سائقٍ حيّ.)
//
// **وهي آخرُ ما يُحتمل سقوطُه**: سائقٌ في ضائقةٍ يضغط الزرَّ، **والعملياتُ
// ترى شاشةَ عطبٍ مكانَ موقعِه ورقمِه.**
//
// # وعائلتُه أوسعُ من موضعِه
//
// **ما لا يُختبر إلّا فارغاً يُقرأ سليماً وهو معطوب.** فالحارسُ يزرع صفّاً
// حقيقيّاً ثمّ يقرأ — **وقائمةٌ تُختبر بلا صفٍّ لا تُختبر.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestEmergencies_ListReadsWithRows(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	driverID := testdb.NewUser(t, f.pool, "driver")

	var emID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO driver_emergencies (driver_id, note, at, status)
		VALUES ($1, 'بلاغُ اختبار',
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography, 'open')
		RETURNING id`, driverID).Scan(&emID); err != nil {
		t.Fatalf("تعذّر زرعُ بلاغ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM driver_emergencies WHERE id = $1`, emID)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/emergencies", nil)
	w := httptest.NewRecorder()
	f.srv.handleOpenEmergencies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("شاشةُ الطوارئ ردّت %d وفيها بلاغٌ واحد — "+
			"**فالسائقُ يستغيث ولا أحدَ يراه**: %s", w.Code, w.Body.String())
	}

	// **والصفُّ يصل كاملاً لا رمزُ حالةٍ وحدَه** — **ردٌّ بمئتين وقائمةٌ
	// فارغةٌ يُقرأ «لا طوارئ» وهو كذب.**
	var out struct {
		Data struct {
			Emergencies []struct {
				ID        string `json:"id"`
				CreatedAt string `json:"created_at"`
			} `json:"emergencies"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ لا يُقرأ: %v — %s", err, w.Body.String())
	}
	if len(out.Data.Emergencies) == 0 {
		t.Fatalf("الشاشةُ ردّت مئتين وقائمتُها فارغةٌ والبلاغُ مزروع")
	}
	if out.Data.Emergencies[0].CreatedAt == "" {
		t.Fatalf("البلاغُ بلا وقت — **ومن لا يعرف متى وقع لا يعرف أيُلاحقه**")
	}
}
