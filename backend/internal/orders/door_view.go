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
	"time"
)

// DoorView **حالُ الباب كما يقرؤه المكتب.**
type DoorView struct {
	// ReportCode **آخرُ بلاغٍ من السائق عند الباب** — وفارغٌ إن لم يُبلّغ.
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
}

// DoorViewOf **حالُ باب طلبٍ** — ولا خطأ: ما تعذّرت قراءتُه يبقى فارغاً،
// **فلا تسقط قائمةُ الطلبات كلُّها لأنّ سطراً في سجلّ التدقيق غاب.**
func (s *Service) DoorViewOf(ctx context.Context, orderID string) *DoorView {
	v := &DoorView{WaitedMin: -1}

	var code, note *string
	var at *time.Time
	if err := s.db.QueryRow(ctx, `
		SELECT details->>'code', details->>'note', created_at FROM audit_log
		WHERE action = 'driver.stage_report' AND entity = 'order' AND entity_id = $1
		  AND details->>'status' = $2
		ORDER BY created_at DESC LIMIT 1`, orderID, StAtDropoff).
		Scan(&code, &note, &at); err == nil && code != nil && IsDoorReport(*code) {
		v.ReportCode = *code
		if note != nil {
			v.ReportNote = *note
		}
		v.ReportAt = at
		v.SuggestedFault = SuggestedFault(*code)
	}

	var arrived *time.Time
	var waited *float64
	if err := s.db.QueryRow(ctx, `
		SELECT max(created_at),
		       floor(EXTRACT(EPOCH FROM now() - max(created_at)) / 60)
		FROM order_events WHERE order_id = $1 AND to_status = $2`,
		orderID, StAtDropoff).Scan(&arrived, &waited); err == nil && arrived != nil && waited != nil {
		v.ArrivedAt = arrived
		v.WaitedMin = int64(*waited)
	}

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
