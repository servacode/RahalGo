package identity

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ══════════════════════════════════════════════════════════════════════
// **فعلٌ حسّاسٌ نجح وأثرُه سقط** — `XG-20` · `AQ-4`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ بنصّه
//
//	CRITICAL ACTION SUCCESS ⇒ CRITICAL AUDIT SUCCESS
//
// **«في المعاملة نفسِها أو في وحدة النجاح المنطقيّة نفسِها.»**
// **ويشمل الأدوارَ والصلاحيّاتِ والإيقافَ والحظر.**
//
// # وما كان
//
// **`repo.Audit` تُنادى بعد الكتابة على المَسبَح وتبتلع خطأها**
// (`repo.go:738` — `_, _ = r.db.Exec`). **فسحبُ دورٍ ينجح وأثرُه
// يسقط**، **ولا يبقى من يُسأل: من سحبه ومتى ولماذا.**
//
// # ولماذا هنا لا في `server`
//
// **الأفعالُ الثلاثةُ تعيش في `identity`** — منحُ دورٍ وسحبُه وتبديلُ
// حال. **ولا تستوردُ حزمةُ الهويّة الخادمَ**، **ومنسّقُ `auditTx` في
// `server` يخدم مسارات `HTTP` وحدَها.**
//
// **والعقدُ واحدٌ والصيغةُ واحدة** — **جدولٌ واحدٌ وحقولٌ هي هي**،
// ويحرسهما فحصٌ واحد.

// AuditTx **يقيّد أثراً في معاملة الفعل نفسِها** — ويُرجع خطأه.
//
// **وخلافُ `Audit` أنّه لا يبتلع** — **فسقوطُه يُسقط الفعل**، وذلك هو
// المقصود.
//
// **ولا يُنادى بعد التثبيت** — **الفجوةُ بين كتابتين هي عينُ العلّة.**
func AuditTx(ctx context.Context, q dbtx.Querier, actorID *string,
	action, entity, entityID, ip string, details map[string]any) error {
	d := details
	if d == nil {
		d = map[string]any{}
	}
	// **وعنوانٌ فارغٌ يُكتب فارغاً لا `NULL`** — العمودُ نصٌّ في المخطَّط.
	if _, err := q.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actorID, action, entity, entityID, ip, d); err != nil {
		return fmt.Errorf("تدقيقٌ حسّاسٌ لم يُقيَّد (%s): %w", action, err)
	}
	return nil
}

// CriticalActions **معجمُ الأفعال الحسّاسة في هذه الحزمة** — الصنفُ `A`.
//
// **مشتقٌّ من نصّ `AQ-4`**: «الأدوارَ والصلاحيّات · الإيقافَ والحظر».
//
// **ويحرسه فحصٌ دائم**: **من أضاف فعلاً هنا ولم يمرّ بـ`AuditTx` سقط
// بناؤه.**
var CriticalActions = map[string]bool{
	"admin.role_grant":  true,
	"admin.role_revoke": true,
	"admin.user_update": true,
}
