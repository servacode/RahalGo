package settings

// ══════════════════════════════════════════════════════════════════════
// **مفاتيحُ مرتبطةٌ تُفحص معاً** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات، البند ١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ مفتاحٍ صحيحٌ وحدَه والاثنان معاً عطب**:
//
//   - أدنى أجرةِ الطلب الخاصّ ٥٠ ألفاً وأعلاها ١٠ آلاف ⇒ **يُرفض كلُّ رقمٍ
//     يكتبه السائق فتعلق الطلباتُ الخاصّةُ كلُّها.**
//   - القبولُ التلقائيُّ قبل مهلة التنبيه ⇒ **يُقبل الطلبُ قبل أن يُنبَّه أحد** —
//     والشرحُ نفسُه يقول «تنبيهٌ ثمّ فعل».
//
// **والفحصُ بالقيمة الجديدة مقابل المحفوظة** — فمن يرفع الأعلى قبل الأدنى لا
// يُحبس.

import (
	"context"
	"fmt"
)

// ErrConflict قيمةٌ صحيحةٌ وحدَها تتعارض مع مفتاحٍ مرتبط.
type ErrConflict struct {
	Key  string
	With string
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("settings: %s يتعارض مع %s", e.Key, e.With)
}

// IntReader ما يكفي لقراءة الطرف الآخر.
type IntReader interface {
	GetInt(ctx context.Context, key string) int64
}

// CheckRelated **يفحص القيمةَ الجديدةَ مقابل المفاتيح المرتبطة بها.**
//
// `v` القيمةُ بعد `Validate` (مطبَّعة).
func CheckRelated(ctx context.Context, st IntReader, key string, v any) error {
	n, ok := v.(int64)
	if !ok {
		return nil
	}
	switch key {
	case "delivery.custom_fee_min":
		if mx := st.GetInt(ctx, "delivery.custom_fee_max"); n > mx {
			return ErrConflict{Key: key, With: "delivery.custom_fee_max"}
		}
	case "delivery.custom_fee_max":
		if mn := st.GetInt(ctx, "delivery.custom_fee_min"); n < mn {
			return ErrConflict{Key: key, With: "delivery.custom_fee_min"}
		}
	case "orders.auto_accept_min":
		if n != 0 && n <= st.GetInt(ctx, "orders.accept_timeout_min") {
			return ErrConflict{Key: key, With: "orders.accept_timeout_min"}
		}
	case "orders.accept_timeout_min":
		if auto := st.GetInt(ctx, "orders.auto_accept_min"); auto != 0 && n >= auto {
			return ErrConflict{Key: key, With: "orders.auto_accept_min"}
		}
	}
	return nil
}
