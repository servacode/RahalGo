package server

// ══════════════════════════════════════════════════════════════════════
// **منطقةُ المتجر تُختار من قائمةٍ حيّة — ولا تُكتب بيد**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٣٠.)
//
// **والفحصُ على المنطق لا على الشاشة**: قائمةٌ منسدلةٌ في الواجهة تمنع
// الخطأَ عمّن يستعملها، **ولا تمنع من يرسل بيده.** والمعرّفاتُ تُخمَّن
// أو تُقرأ من ردٍّ سابق.

import (
	"context"
	"net/http"
	"testing"
)

// districtIDs معرّفا منطقةٍ حيّةٍ وأخرى مطفأة.
func liveDistrict(t *testing.T, f *driverFixture) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT d.id::text FROM districts d
		JOIN governorates g ON g.id = d.governorate_id
		WHERE d.active AND g.active LIMIT 1`).Scan(&id); err != nil {
		t.Fatalf("لا منطقةَ حيّة: %v", err)
	}
	return id
}

func TestValidDistrict_AcceptsLiveRejectsOff(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/x", nil)

	// **الحيّةُ تمرّ.**
	live := liveDistrict(t, f)
	got, err := f.srv.validDistrict(req, live)
	if err != nil || got == nil || *got != live {
		t.Fatalf("منطقةٌ حيّةٌ رُدّت: %v", err)
	}

	// **والفراغُ يمرّ فارغاً** — الإلزامُ قرارُ من ينادي لا قرارُ الدالّة.
	if got, err := f.srv.validDistrict(req, ""); err != nil || got != nil {
		t.Errorf("الفراغُ لم يُردّ فارغاً: %v", err)
	}

	// **ونصٌّ ليس معرّفاً يُردّ** — ولا يصل القاعدةَ أصلاً.
	if _, err := f.srv.validDistrict(req, "وسط المدينة"); err == nil {
		t.Error("نصٌّ حرٌّ مرّ كمعرّف منطقة — **وهو ما جئنا نمنعه**")
	}

	// ══════════════════════════════════════════════════════════════════
	// **والمطفأةُ تُردّ وهي موجودةٌ في الجدول**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والمفتاحُ الأجنبيُّ لا يراها** — يحرس الوجودَ لا الحياة. **ومن
	// أرسل معرّفَها من نموذجٍ قديمٍ في جهازه يُقيَّد في محافظةٍ أُغلقت
	// عمداً**، فتظهر متاجرُ حيث قرّر المالكُ ألّا نعمل.
	var off string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO districts (governorate_id, name, active)
		SELECT g.id, 'منطقةُ فحصٍ مطفأة', false FROM governorates g LIMIT 1
		RETURNING id::text`).Scan(&off); err != nil {
		t.Fatalf("تعذّر إنشاءُ منطقةٍ مطفأة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM districts WHERE id = $1::uuid`, off)
	})
	if _, err := f.srv.validDistrict(req, off); err == nil {
		t.Error("منطقةٌ مطفأةٌ قُبلت — **ويُسجَّل متجرٌ في محافظةٍ أُغلقت**")
	}
}

// TestDistrictOffWhenGovernorateOff **وإطفاءُ المحافظة يُطفئ مناطقَها.**
//
// **ومن أطفأ محافظةً أرادها كلَّها** — ولو بقيت مناطقُها مقبولةً
// **لَسُجّل فيها متاجرُ من بابٍ خلفيّ**، والقائمةُ الأولى لا تعرضها.
func TestDistrictOffWhenGovernorateOff(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	var gov, dist string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO governorates (name, active) VALUES ('محافظةُ فحصٍ مطفأة', false)
		RETURNING id::text`).Scan(&gov); err != nil {
		t.Fatalf("تعذّرت المحافظة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM districts WHERE governorate_id = $1::uuid`, gov)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM governorates WHERE id = $1::uuid`, gov)
	})
	// **والمنطقةُ فعّالةٌ في نفسها** — وأمُّها مطفأة.
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO districts (governorate_id, name, active)
		VALUES ($1::uuid, 'منطقةٌ حيّةٌ تحت مطفأة', true) RETURNING id::text`,
		gov).Scan(&dist); err != nil {
		t.Fatalf("تعذّرت المنطقة: %v", err)
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/x", nil)
	if _, err := f.srv.validDistrict(req, dist); err == nil {
		t.Error("منطقةٌ تحت محافظةٍ مطفأةٍ قُبلت — **وإطفاءُ المحافظة لا يعني شيئاً**")
	}
}
