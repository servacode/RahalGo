// Package approval **قاعدةُ «المقترحُ غيرُ الموافق» في المال** — مصدرٌ واحدٌ
// تقرؤه كلُّ أقسام المال.
//
// ══════════════════════════════════════════════════════════════════════
// **قرارُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ الأدوار والصلاحيّات، البند ٦)**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · من اقترح حركةَ مالٍ لا يوافق عليها — يوافق غيرُه ممّن يملك القدرة.
//	٢ · فإن كانت الماليّةُ شخصاً واحداً وافق مديرُ المنصّة (يملك القدرةَ نفسَها).
//	٣ · فإن لم يوجد في المنصّة أحدٌ غيرُه يملك الموافقة: يوافق على نفسه
//	    **المالكُ (`owner_super_admin`) وحدَه**، وتُعلَّم الموافقةُ في السجلّ
//	    بعلامةٍ صريحة (`self_approved` + `self_approval_reason`).
//
// **وكان الحكمُ في حركات المحفظة وحدَها** (`canSelfApprove`) — ويجيز
// لمدير المنصّة أيضاً. **والآن مصدرٌ واحد**: صرفٌ جديدٌ (مصروف، تعويض، سحب)
// يقرأ `approval.Check` ولا يكتب قاعدته من جديد.
package approval

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrSelfApprove **صاحبُ الاقتراح لا يوافق عليه** — والمفتاحُ قائمٌ في المعجم.
var ErrSelfApprove = httpx.NewError(http.StatusForbidden,
	"self_approve", "errors.self_approve")

// SelfApprovalReason **سببُ الموافقة الذاتيّة كما يُكتب في السجلّ.**
const SelfApprovalReason = "owner_alone_no_other_approver"

// Request **موافقةٌ تُطلَب على اقتراح.**
type Request struct {
	// ProposedBy معرّفُ صاحب الاقتراح.
	ProposedBy string
	// Actor معرّفُ من يوافق الآن.
	Actor string
	// Capability **القدرةُ التي تملك الموافقة** — مثلاً `finance.manage`.
	Capability authz.Capability
}

// Verdict **حكمُ الموافقة** — يُكتب في الطلب وفي السجلّ.
type Verdict struct {
	// SelfApproved **وافق صاحبُ الاقتراح على نفسه** — المالكُ وحدَه وبلا بديل.
	SelfApproved bool
}

// AuditFields **ما يُضاف إلى تفاصيل السجلّ** — علامةٌ صريحةٌ لا تُفوَّت.
func (v Verdict) AuditFields() map[string]any {
	out := map[string]any{"self_approved": v.SelfApproved}
	if v.SelfApproved {
		out["self_approval_reason"] = SelfApprovalReason
	}
	return out
}

// Check **أيجوز لهذا الفاعل أن يوافق؟** — يُنادى داخل معاملة الموافقة.
//
// **فاعلٌ غيرُ صاحب الاقتراح يمرّ** (والقدرةُ يحرسها الوسيطُ قبله).
// **وصاحبُ الاقتراح يُردّ** بـ`ErrSelfApprove` — **إلّا** إن كان يحمل دورَ
// المالك الأعلى **ولا يوجد حسابٌ فعّالٌ آخرُ يملك `Capability`.**
func Check(ctx context.Context, q dbtx.Querier, req Request) (Verdict, error) {
	if req.ProposedBy == "" || req.ProposedBy != req.Actor {
		return Verdict{}, nil
	}
	var owner, others bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = $2),
		       EXISTS (SELECT 1 FROM users u
		                 JOIN user_roles ur ON ur.user_id = u.id
		                 JOIN role_capabilities rc ON rc.role_code = ur.role_code
		                WHERE u.id <> $1::uuid AND u.status = 'active'
		                  AND rc.capability_code = $3)`,
		req.Actor, authz.RoleOwnerSuperAdmin, string(req.Capability)).
		Scan(&owner, &others); err != nil {
		return Verdict{}, err
	}
	if owner && !others {
		return Verdict{SelfApproved: true}, nil
	}
	return Verdict{}, ErrSelfApprove
}
