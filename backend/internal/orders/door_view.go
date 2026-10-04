package orders

// ══════════════════════════════════════════════════════════════════════
// **ما يراه المكتبُ عند باب الزبون** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
// المكتبُ يقرّر «سلّم الآن» أو «عُد إلى المكتب» — **ولا يقرّر بلا ثلاثة**:
// ماذا قال السائق آخرَ مرّة، وكم ينتظر في الشارع، وهل أُرسل إليه أمرٌ قبلُ.
//
// **قراءةٌ وحدَها** — لا تكتب شيئاً، وتُملأ في ردّ الإدارة وحدَه
// (`server/admin_orders_handlers.go`). **والزبونُ والمتجرُ لا يرونها.**

import (
	"context"
	"strings"
	"time"
)

// DoorView **حالُ الباب كما يقرؤه المكتب.**
type DoorView struct {
	// ReportCode **آخرُ بلاغٍ من السائق بعد الاستلام** — في الطريق أو عند الباب.
	ReportCode string `json:"report_code"`
	// ReportNote كلمةُ السائق مع البلاغ.
	ReportNote string `json:"report_note"`
	// ReportAt متى بلّغ.
	ReportAt *time.Time `json:"report_at"`
	// SuggestedFault **الذنبُ الذي يقترحه البلاغ** — اقتراحٌ لا حكم.
	SuggestedFault string `json:"suggested_fault"`
	// ArrivedAt **متى وصل الباب** — وفارغٌ إن لم يُعرف.
	ArrivedAt *time.Time `json:"arrived_at"`
	// WaitedMin **كم دقيقةً ينتظر عند الباب** — وسالبٌ إن لم يُعرف.
	WaitedMin int64 `json:"waited_min"`
	// Instruction **أمرُ المكتب المُرسَل** — `deliver_now` أو فارغ.
	Instruction string `json:"instruction"`
	// InstructionNote كلمةُ المكتب مع الأمر.
	InstructionNote string `json:"instruction_note"`
	// InstructionAt متى أُرسل.
	InstructionAt *time.Time `json:"instruction_at"`
	// StorePhone **هاتفُ المتجر** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣) — «ينتظر الإدارةَ تتّصل بالمتجر»: كانت
	// اللوحةُ عند المتجر تعرض هاتفَ الزبون وحدَه. **وفارغٌ للخاصّ بلا متجر.**
	StorePhone string `json:"store_phone"`
	// Reasons **أسبابُ الإنهاء الصالحةُ في هذه المرحلة** — من المحرّك لا من الشاشة
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٦). **كانت قائمةُ النافذة البلاغاتِ كلَّها**:
	// أسبابُ المتجر عند الباب، والمحذوفُ بقرار المالك — **والمحرّكُ يردّها
	// «السببُ لا يخصّ هذه المرحلة».**
	Reasons []string `json:"reasons"`
}

// DoorViewOf **حالُ باب طلبٍ** — ولا خطأ: ما تعذّرت قراءتُه يبقى فارغاً،
// **فلا تسقط قائمةُ الطلبات كلُّها لأنّ سطراً في سجلّ التدقيق غاب.**
func (s *Service) DoorViewOf(ctx context.Context, orderID string) *DoorView {
	v := &DoorView{WaitedMin: -1, Reasons: []string{}}

	var code, note *string
	var at *time.Time
	// **وبلاغُ ما قبل المتجر من هذا الطور وحدَه** (٢٠٢٦-١٠-٠٣) — بعد تحويلٍ يعود الطلبُ إلى
	// الطريق، **فلا يُقرأ بلاغُ المتجر القديم بلاغاً قبل الجديد.**
	var cur string
	_ = s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&cur)
	v.Reasons = OfficeReasonsAt(cur)
	statuses := []string{StAtPickup, StPickedUp, StOnTheWay, StAtDropoff}
	if cur == StAssigned {
		statuses = []string{StAssigned}
	}
	since := s.reportsSince(ctx, orderID)
	if err := s.db.QueryRow(ctx, `
		SELECT details->>'code', details->>'note', created_at FROM audit_log
		WHERE action = 'driver.stage_report' AND entity = 'order' AND entity_id = $1
		  AND details->>'status' = ANY($2) AND created_at >= $3
		ORDER BY created_at DESC LIMIT 1`, orderID, statuses, since).
		Scan(&code, &note, &at); err == nil && code != nil && IsTripReport(*code) {
		v.ReportCode = *code
		if note != nil {
			v.ReportNote = *note
		}
		v.ReportAt = at
		v.SuggestedFault = SuggestedFault(*code)
	}

	// **وكم ينتظر عند الباب الذي هو عنده** — بابُ المتجر أو بابُ الزبون (٢٠٢٦-١٠-٠٣).
	waitAt := StAtDropoff
	var status string
	if err := s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err == nil &&
		status == StAtPickup {
		waitAt = StAtPickup
	}
	var arrived *time.Time
	var waited *float64
	if err := s.db.QueryRow(ctx, `
		SELECT max(created_at),
		       floor(EXTRACT(EPOCH FROM now() - max(created_at)) / 60)
		FROM order_events WHERE order_id = $1 AND to_status = $2`,
		orderID, waitAt).Scan(&arrived, &waited); err == nil && arrived != nil && waited != nil {
		v.ArrivedAt = arrived
		v.WaitedMin = int64(*waited)
	}

	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(u.phone::text, '') FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		JOIN users u ON u.id = m.owner_user_id
		WHERE o.id = $1`, orderID).Scan(&v.StorePhone)

	var instr, instrNote *string
	var instrAt *time.Time
	if err := s.db.QueryRow(ctx, `
		SELECT door_instruction, door_instruction_note, door_instruction_at
		FROM orders WHERE id = $1`, orderID).
		Scan(&instr, &instrNote, &instrAt); err == nil {
		if instr != nil {
			v.Instruction = *instr
		}
		if instrNote != nil {
			v.InstructionNote = *instrNote
		}
		v.InstructionAt = instrAt
	}
	return v
}

// reportsSince **من أين تُقرأ البلاغات** — من آخر عودةٍ إلى «في الطريق للمتجر».
//
// **كان المرشّحُ للطور `assigned` وحدَه** (تجربةُ القبول ٢٠٢٦-١٠-٠٣، مرّتين على المحاكي):
// «المتجر مغلق» عند الأوّل ← تحويل ← وصل الجديد ← **فقُرئ بلاغُ القديم بلاغاً عند الجديد**،
// ولوحةُ الإدارة تطلب قراراً حُسم، والسائقُ بلا «استلمت الطلب». **والتحويلُ وحدَه يُعيد
// الطلبَ من عند المتجر إلى الطريق** (`server/order_transfer.go`) — **فكلُّ بلاغٍ قبله يخصّ
// متجراً لم يعد متجرَه.** والإسنادُ الأوّل أقدمُ من كلّ بلاغ، فلا يحجب شيئاً.
func (s *Service) reportsSince(ctx context.Context, orderID string) time.Time {
	var last *time.Time
	_ = s.db.QueryRow(ctx, `
		SELECT max(created_at) FROM order_events
		WHERE order_id = $1 AND to_status = 'assigned' AND from_status <> 'assigned'`,
		orderID).Scan(&last)
	if last == nil {
		return time.Time{}
	}
	return *last
}

// CountAwaitingOffice **كم طلباً ينتظر قرارَ المكتب على بلاغٍ من السائق.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: قسمُ «بانتظار قرارك» في رئيسيّة اللوحة.)
//
// **طلبٌ مع السائق** (عند المتجر · استلم · في الطريق · عند الباب) **آخرُ بلاغٍ
// فيه بلاغُ رحلةٍ** (`IsTripReport`) **ولم يُرسَل بعده أمرٌ من المكتب.**
// **والقاعدةُ نفسُها التي تُظهر لوحةَ البطاقة** — فالرقمُ يطابق ما يراه من
// فتح الطلبات، **ولا يُحسب الحكمُ مرّتين بلفظين.**
func (s *Service) CountAwaitingOffice(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM orders o WHERE `+awaitingOfficeSQL()).Scan(&n)
	return n, err
}

// awaitingOfficeSQL **شرطُ «بلاغٌ ينتظر المكتب» نصّاً واحداً** — للعدّ في
// الرئيسيّة ولمُرشِّح الطلبات معاً (`ListFilter.AwaitingOffice`)، **فلا تقول
// البطاقةُ ثلاثةً وتعرض القائمةُ اثنين.**
//
// **والرموزُ والحالاتُ ثوابتُ الحزمة** (`StageReports`) لا مُدخَلُ أحد —
// فتُكتب في النصّ بلا معاملات، **ويبقى الشرطُ صالحاً في استعلامٍ يعدّ
// معاملاتِه هو.**
func awaitingOfficeSQL() string {
	// **وقبل المتجر أيضاً** (٢٠٢٦-١٠-٠٣: «أكمل الطلب» أو «ألغِ الطلب»).
	statuses := []string{StAssigned, StAtPickup, StPickedUp, StOnTheWay, StAtDropoff}
	codes := []string{}
	for _, r := range StageReports {
		if IsTripReport(r.Code) || r.Code == "customer_cancelled_by_phone" {
			codes = append(codes, r.Code)
		}
	}
	st, cs := sqlTextList(statuses), sqlTextList(codes)
	return `o.closed_at IS NULL AND o.status IN ` + st + `
		AND EXISTS (
			SELECT 1 FROM (
				SELECT a.details->>'code' AS code, a.created_at
				FROM audit_log a
				WHERE a.entity = 'order' AND a.entity_id = o.id::text
				  AND a.action = 'driver.stage_report'
				  AND a.details->>'status' IN ` + st + `
				  -- **وبلاغُ ما قبل آخر عودةٍ إلى الطريق لا يُحسب** — بلاغُ متجرٍ حُوّل عنه (reportsSince).
				  AND a.created_at >= COALESCE((SELECT max(e.created_at) FROM order_events e
				        WHERE e.order_id = o.id AND e.to_status = 'assigned' AND e.from_status <> 'assigned'),
				        '-infinity'::timestamptz)
				ORDER BY a.created_at DESC LIMIT 1) last
			WHERE last.code IN ` + cs + `
			  AND (o.door_instruction_at IS NULL OR o.door_instruction_at < last.created_at))`
}

// sqlTextList قائمةُ نصوصٍ ثابتةٍ بصيغة SQL — **والاقتباسُ يُضاعَف** احتياطاً.
func sqlTextList(xs []string) string {
	if len(xs) == 0 {
		return "('')"
	}
	out := "("
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += "'" + strings.ReplaceAll(x, "'", "''") + "'"
	}
	return out + ")"
}
