package server

// **وتحويلُ طلبِ انضمامٍ برقمِ سائقٍ يُردّ — بسببٍ يُقال.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # ولماذا صار هذا يقع
//
// **قاعدةُ «دورٌ واحدٌ ومعه الزبون» صارت تُفرض في `GrantRole` نفسِه**
// (٢٠٢٦-٠٨-١٦) — **وطريقُ تحويل الطلب يمرّ به**: يُنشئ المتجرَ فيمنح
// صاحبَه دورَ التاجر.
//
// **فطلبُ انضمامٍ رقمُ صاحبه رقمُ سائقٍ عندك كان يمرّ** — ويخلق سائقاً
// وصاحبَ متجرٍ معاً، **وهو عينُ التعارض الذي كُتبت القاعدةُ لمنعه.**
//
// # وما يُحرَس
//
// **أنّ الردَّ خطأٌ له رمزٌ يُترجَم** — لا نجاحٌ صامتٌ ولا خمسُمئة.
// **وشاشةُ الطلبات كانت تبتلع الخطأ** فيضغط المكتبُ ولا يقع شيء.
//
// **ولا متجرَ يُنشأ على النصف**: من رُدّ طلبُه لا يُترك له متجرٌ بلا صاحب.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
)

func TestConvertLead_RefusesDriverPhoneAndSaysWhy(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver := f.drivers[0]

	var phone string
	if err := f.pool.QueryRow(ctx,
		`SELECT phone::text FROM users WHERE id = $1`, driver).Scan(&phone); err != nil {
		t.Fatalf("تعذّرت قراءةُ الهاتف: %v", err)
	}
	var categoryID string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var leadID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchant_leads (store_name, owner_name, phone, area, category_id, lat, lng)
		VALUES ('متجرُ التعارض', 'صاحبُه', $1, 'الرقة', $2, 35.9528, 39.0079)
		RETURNING id::text`, phone, categoryID).Scan(&leadID); err != nil {
		t.Fatalf("تعذّر الطلب: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM merchant_leads WHERE id = $1`, leadID)
		_, _ = f.pool.Exec(c, `DELETE FROM merchants WHERE name = 'متجرُ التعارض'`)
	})

	err := f.srv.convertLead(ctx, driver, leadID, "1.1.1.1")
	if err == nil {
		t.Fatal("حُوّل طلبٌ برقمِ سائق — **فصار سائقاً وصاحبَ متجرٍ معاً**")
	}
	// **والسببُ رمزٌ يُترجَم** — «لهذا الحساب دورٌ أساسيٌّ بالفعل»، **لا
	// «حدث خطأ» يُعاد معه الضغطُ بلا فائدة.**
	if !errors.Is(err, identity.ErrRoleConflict) {
		t.Fatalf("رُدّ بسببٍ آخر: %v — **ورسالةٌ عامّةٌ لا تُصلَح بها الحال**", err)
	}

	// **ولا يبقى الطلبُ محوَّلاً في القاعدة** — **وطلبٌ يُعدّ منجزاً وقد
	// فشل يختفي من الطابور ولا يُنجَز أبداً.**
	var merchantID *string
	if err := f.pool.QueryRow(ctx,
		`SELECT merchant_id::text FROM merchant_leads WHERE id = $1`, leadID).
		Scan(&merchantID); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if merchantID != nil {
		t.Fatal("وُسم الطلبُ محوَّلاً رغم الفشل — **فيختفي من الطابور ولا يُنجَز**")
	}
}
