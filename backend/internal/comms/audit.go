package comms

/*
**حديثُ الطلب كما تقرؤه الإدارة — من بابها لا من باب طرفيه.**

(قرارُ المالك ٢٠٢٦-٠٨-١٠: «يجب أن نضيف دردشات الزبائن والسائقين — في حال حصول
 أيّ تجاوزٍ يمكننا الرجوع إليه».)

# ولماذا بابٌ ثانٍ لا توسعةُ الأوّل

**`Permit` تردّ الإدارةَ عمداً**: هي ليست طرفاً، **ولو صارت طرفاً لَحملت
`PeerID` ولاستطاعت أن تكتب** — فيقرأ الزبونُ سطراً باسم سائقه لم يقله.

**والقراءةُ فعلٌ آخرُ غيرُ المشاركة.** فهذا بابٌ يقرأ ولا يكتب، **ولا يمرّ
بالصلاحية أصلاً** — فلا يستطيع أحدٌ أن يبلغه بتبديل دور.

# ولا وسمَ قراءةٍ يُكتب

**من قرأته الإدارةُ لا يصير «قُرئت» عند صاحبه** — **وعلامةٌ زرقاءُ يضعها
طرفٌ ثالثٌ كذبٌ يُحتجّ به**: يظنّ المرسِلُ أنّ الآخرَ قرأ، فيبني عليه لومَه.

# وحقُّ الوصول يُفحص في المسار لا هنا

**المسارُ تحت `/admin` بحارسه** — ومن بلغ هذه الدالّة فقد جاز الحارس.
*/

import (
	"context"
	"time"
)

// AuditLine **سطرٌ في الحديث كما تراه الإدارة — بقائله لا بـ«لي/له».**
//
// **و`mine` لا معنى لها لمن ليس طرفاً** — فيُكتب الدورُ والاسم.
type AuditLine struct {
	ID        string     `json:"id"`
	Body      string     `json:"body"`
	Role      string     `json:"role"`
	Sender    string     `json:"sender"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
	// Flagged **أفيها لفظٌ لا يُقال؟** — (قرارُ المالك ٢٠٢٦-٠٨-١٢).
	//
	// **وهي أوّلُ ما يُبحث عنه عند شكوى**: «قال لي كذا» كانت كلمةً ضدّ
	// كلمة، **والسطرُ الموسومُ يحسمها في ثانية.**
	Flagged  bool   `json:"flagged"`
	FlagWord string `json:"flag_word,omitempty"`
}

// AuditThread **حديثُ طلبٍ كاملاً — للمراجعة عند الخلاف.**
type AuditThread struct {
	Customer string      `json:"customer"`
	Driver   string      `json:"driver"`
	Lines    []AuditLine `json:"lines"`
}

// Audit **يقرأ حديثَ الطلب لمن يراجع.**
func (s *Service) Audit(ctx context.Context, orderID string) (*AuditThread, error) {
	out := &AuditThread{Lines: []AuditLine{}}

	// **والطرفان يُقرآن ولو أُغلق الطلب** — ومن سُحب منه الطلبُ يبقى اسمُه
	// على سطره، **فالسطرُ يحمل قائلَه لا حاملَ الطلب اليوم.**
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(cu.full_name, ''), COALESCE(dr.full_name, '')
		FROM orders o
		LEFT JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		WHERE o.id = $1`, orderID).Scan(&out.Customer, &out.Driver); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT x.id::text, x.body, x.sender_role, COALESCE(u.full_name, ''),
		       x.created_at, x.read_at, x.flagged, COALESCE(x.flag_word, '')
		FROM order_messages x
		LEFT JOIN users u ON u.id = x.sender_id
		WHERE x.order_id = $1
		ORDER BY x.created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l AuditLine
		if err := rows.Scan(&l.ID, &l.Body, &l.Role, &l.Sender, &l.CreatedAt, &l.ReadAt,
			&l.Flagged, &l.FlagWord); err != nil {
			return nil, err
		}
		out.Lines = append(out.Lines, l)
	}
	return out, rows.Err()
}
