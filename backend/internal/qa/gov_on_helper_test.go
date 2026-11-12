package qa

import "testing"

// govOn **يشغّل محافظةً مطفأةً طولَ الفحص ثمّ يعيدها كما كانت.**
//
// المحافظاتُ كلُّها مطفأةٌ ما عدا الرقة (الهجرة 0545، قرارُ المالك ٢٠٢٦-١١-١١)، **ففحصٌ يقيس حكمَ
// «مدينةٍ معروفةٍ لم تُطلَق» (دمشق) يحتاج محافظتَها شغّالة** — وإلّا جاء السببُ
// `province_not_supported` قبل أن يصل إلى المدينة. **والإعادةُ في `Cleanup`** فتقع ولو سقط الفحص.
func govOn(t *testing.T, h *Harness, name string) {
	t.Helper()
	var was bool
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT active FROM governorates WHERE name = $1`, name).Scan(&was); err != nil {
		t.Fatalf("قراءةُ المحافظة %s: %v", name, err)
	}
	if was {
		return
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE governorates SET active = true WHERE name = $1`, name); err != nil {
		t.Fatalf("تشغيلُ المحافظة %s: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE governorates SET active = false WHERE name = $1`, name)
	})
}
