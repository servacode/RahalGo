package server

// سجلّ التدقيق على مستوى المعالِجات.
//
// كان `audit_log` يُكتب من طبقة الخدمات فقط (`catalog`)، فسجّل ثمانية عشر فعلاً
// كلُّها من نوعٍ واحد: إنشاء تصنيف، بانر، كود ترويجي، تعديل ساعات عمل. **وبقيت
// الأفعال الستّ التي قد يُسأل عنها أحدٌ يوماً بلا أثر**:
//
//   - قيدٌ يدويّ في محفظة مستخدم
//   - صرفُ طلب سحب أو رفضُه
//   - تسوية صندوق سائق
//   - حلُّ تذكرة بتعويض
//   - تغييرُ إعداد يحكم المال
//   - تحريكُ حالة طلب بيد موظّف
//
// أي أن **ما لا يمسّ المال كان يُسجَّل، وما يمسّه لا يُسجَّل**.
//
// وهذه الأفعال تعيش في المعالِجات لا في الخدمات — فالسجلّ يحتاج بيتاً هنا.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// audit يقيّد فعلاً في السجلّ.
//
// **لا يُفشل الفعل إن فشل القيد**: القيد أثرٌ لا شرط. وإسقاطُ صرفِ سحبٍ لأن
// كتابة سطرٍ في جدولٍ تعثّرت يعاقب المستخدم على عطبٍ ليس منه. ويُكتب في
// الخلفية بسياقٍ مستقلّ كي لا يُلغى بإلغاء طلب HTTP بعد أن تمّ الفعل.
func (s *Server) audit(r *http.Request, action, entity, entityID string, meta map[string]any) {
	actor := userIDFrom(r)
	ip := clientIP(r)
	var raw []byte
	if len(meta) > 0 {
		raw, _ = json.Marshal(meta)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := s.pg.Exec(ctx, `
			INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
			VALUES (NULLIF($1,'')::uuid, $2, $3, $4, NULLIF($5,''), $6)`,
			actor, action, entity, entityID, ip, raw); err != nil {
			s.logger.Error("audit: تعذّر قيد الحدث", "action", action, "error", err)
		}
	}()
}
