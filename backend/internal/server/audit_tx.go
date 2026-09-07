package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ══════════════════════════════════════════════════════════════════════
// **فعلٌ حسّاسٌ نجح بلا أثرٍ يُقرأ** — `PF-06` · `AQ-4`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`audit()` تكتب في خيطٍ منفصلٍ بمعاملتها** — **فسقوطُها لا يُفشل
// الفعل.** ومقيسٌ بالحقن: **الردُّ `200` · رصيدٌ `12000` · وقيودُ
// التدقيق صفر.**
//
// **ولا مسارَ يستدرك**: **الأثرُ ضاع، ولا شيءَ يقول إنّه ضاع.**
//
// # والعقد
//
//	A SUCCESSFUL SENSITIVE BUSINESS MUTATION
//	MUST HAVE A DURABLE AUDIT RECORD
//
// **فحالان لا ثالث**: **يقعان معاً أو لا يقع أحدُهما.**
//
// # وليس كلُّ سطرٍ تدقيقاً حسّاساً
//
// **ومن جعل كلَّ سطرٍ معامليّاً أسقط بيعاً لأنّ سطرَ سجلٍّ تعذّر.**
// **فالصنفُ `A` محصورٌ فيما يحرّك مالاً** — **وهو الذي لا يُقبَل فيه
// «وقع ولا أثرَ له».**

// criticalAuditActions **الصنفُ `A`** — **أفعالٌ لا تُقبَل بلا أثر.**
//
// # لماذا هذه بعينها
//
// **كلُّها تحرّك مالاً** — قيدٌ في محفظة · حافزٌ · قرارُ سحب · تسويةُ
// نقدِ سائق · مصروفٌ وإلغاؤه. **ومالٌ تحرّك بلا من ولا متى ولا لماذا
// لا يُراجَع ولا يُنازَع فيه.**
//
// **وكلُّها تملك معاملتَها أصلاً** (منسّقُ منع التكرار من دورةِ ٩،
// والمصروفُ من دورةِ ٤) — **فلا معاملةَ جديدةٌ تُفتَح ولا نطاقُ قفلٍ
// يتّسع.**
//
// **وما سواها يبقى أفضلَ جهد**: **إعدادٌ أو تصنيفٌ أو إشعارٌ سقط سطرُه
// لا يُساوي إسقاطَ العملية.**
var criticalAuditActions = map[string]bool{
	"finance.wallet_apply":   true,
	"finance.incentive":      true,
	"finance.payout_decide":  true,
	"finance.driver_settle":  true,
	"finance.expense_added":  true,
	"finance.expense_voided": true,

	// ── نطاقُ `AQ-4` الباقي — دورةُ إصلاحٍ ٢١ (`XG-20`) ──────────
	//
	// **«الإيقافَ والحظر · الإعداداتِ الحسّاسة · وتدخّلاتِ الطلبات
	// الحرجة»** بنصّ العقد.
	//
	// **والأدوارُ والصلاحيّاتُ في `identity.CriticalActions`** —
	// **حيث تعيش أفعالُها**، والعقدُ والصيغةُ واحدة.
	"ops.merchant_suspend": true,
	"admin.setting_update": true,
	"ops.order_transition": true,
}

// conditionalAuditActions **أفعالٌ صنفُها يتقرّر بمعاملها لا باسمها.**
//
// # لماذا وُجد هذا المعجم
//
// **`admin.setting_update` اسمٌ واحدٌ لفعلين**: **تبديلُ عمولةِ
// المنصّة** و**تبديلُ نصّ صفحةٍ** — **والأوّلُ من الصنف `A` والثاني
// لا.** (قرارُ المالك في دورةِ ٢١: «لا تجعل الإعداداتِ الثمانيةَ
// عشرَ ومئةً حسّاسةً بتخمين النطاق».)
//
// # وحارسان قرآ الاسمَ قراءتين
//
// **حارسُ `XG-20` عرف التفريعَ** ونصَّ على شرطه، **وحارسُ `AQ-4` قرأ
// الاسمَ صنفاً واحداً فأدان الفرعَ المشروع** (`XG-41A`).
//
// **والعلّةُ أنّ التفريعَ كان يُقرأ من نصّ الشيفرة في كلّ حارسٍ على
// حدة** — **ونصّان يفترقان يومَ يُعاد تسميةُ دالّة.**
//
// **فصار مكتوباً هنا مرّةً**: القيمةُ اسمُ المصنِّف الكانونيّ الذي
// يحسم كلَّ نداء.
var conditionalAuditActions = map[string]string{
	"admin.setting_update": "criticalSettingKey",
}

// ConditionalAuditClassifier اسمُ المصنِّف الذي يحسم صنفَ هذا الفعل —
// **وفارغٌ يعني أنّ الفعلَ من الصنف `A` بلا شرط.**
//
// **تقرؤه الحرّاس** — **ولا يقرأ أحدُهما نصّاً ويقرأ الآخرُ نصّاً
// ثانياً.**
func ConditionalAuditClassifier(action string) string {
	return conditionalAuditActions[action]
}

// auditTx يقيّد أثراً **في معاملة الفعل نفسِها**.
//
// **ويُنادى داخلَ المعاملة قبل تثبيتها** — **فسقوطُه يُسقط الفعلَ
// كلَّه**، وذلك هو المقصود.
//
// **ولا شبكةَ فيه ولا ملفّ** — **إدخالٌ محلّيٌّ واحد.** **ومن أدخل
// نداءً خارجيّاً في معاملةٍ أطال قفلَها بقدر بُعد الطرف الآخر.**
func (s *Server) auditTx(ctx context.Context, q dbtx.Querier, r *http.Request,
	action, entity, entityID string, meta map[string]any) error {
	raw := []byte("{}")
	if len(meta) > 0 {
		var err error
		if raw, err = json.Marshal(meta); err != nil {
			return err
		}
	}
	// **والفراغُ يصير `NULL` في `Go` لا في `SQL`** — درسُ دورةِ ١٢:
	// **`NULLIF` يقلب استنباطَ النوع.**
	var actor any
	if id := userIDFrom(r); id != "" {
		actor = id
	}
	var ip any
	if v := clientIP(r); v != "" {
		ip = v
	}
	_, err := q.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actor, action, entity, entityID, ip, raw)
	return err
}
