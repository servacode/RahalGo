package comms

import (
	"context"
	"strings"
	"time"
)

// MaxBody **حدُّ الرسالة الواحدة.**
//
// **وما يُقال عند الباب قصير**: «أنا تحت البناية» · «الطابق الثالث» ·
// «اتركه عند البوّاب». **وحدٌّ واسعٌ يجعلها بريداً**، وهي ليست بريداً.
const MaxBody = 500

// MinGap **أقلُّ ما بين رسالتين من الطرف نفسه.**
//
// **بديلُ المنع لا المنع**: الوثيقةُ أخرست الزبونَ حتّى يتكلّم السائق —
// **والحاجةُ الأكثرُ شيوعاً عكسُها.** فيُفتح للطرفين ويُحَدّ المعدَّل: من
// أراد المضايقةَ يجدها بطيئةً بلا فائدة، **ومن أراد أن يقول «أنا نازل»
// يقولها في الحال.**
const MinGap = 2 * time.Second

// Message سطرٌ في الحديث.
type Message struct {
	ID        string     `json:"id"`
	Body      string     `json:"body"`
	Role      Role       `json:"role"`
	Mine      bool       `json:"mine"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}

// List **حديثُ الطلب كما يراه أحدُ طرفيه.**
//
// **ويُقرأ حتّى بعد الإغلاق**: القناةُ تُقفل للكتابة لا للقراءة — **وحديثٌ
// يختفي بانتهاء الطلب يمحو ما يُحتجّ به** عند شكوى.
func (s *Service) List(ctx context.Context, p *Permission) ([]Message, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, body, sender_role, sender_id::text = $2, created_at, read_at
		FROM order_messages WHERE order_id = $1
		ORDER BY created_at`, p.OrderID, p.SelfID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Message{}
	for rows.Next() {
		var x Message
		if err := rows.Scan(&x.ID, &x.Body, &x.Role, &x.Mine, &x.CreatedAt, &x.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Send **يكتب سطراً — ويردّ المتلقّي ليُشعَر.**
//
// **ولا يأخذ متلقّياً**: يُستخرج من الصلاحية. **ومن أرسل معرّفاً لا يُقرأ منه.**
func (s *Service) Send(ctx context.Context, p *Permission, body string) (*Message, error) {
	if !p.Open {
		return nil, ErrChannelClosed
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrEmptyBody
	}
	if len([]rune(body)) > MaxBody {
		body = string([]rune(body)[:MaxBody])
	}

	// **وحدُّ المعدّل يُفحص في القاعدة لا في الذاكرة** — خادمان يعملان معاً
	// **وذاكرةٌ في أحدهما لا يراها الآخر.**
	var recent bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM order_messages
		               WHERE order_id = $1 AND sender_id = $2
		                 AND created_at > now() - $3::interval)`,
		p.OrderID, p.SelfID, MinGap.String()).Scan(&recent); err != nil {
		return nil, err
	}
	if recent {
		return nil, ErrTooFast
	}

	var x Message
	if err := s.db.QueryRow(ctx, `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, body, sender_role, created_at`,
		p.OrderID, p.SelfID, string(p.Me), body).
		Scan(&x.ID, &x.Body, &x.Role, &x.CreatedAt); err != nil {
		return nil, err
	}
	x.Mine = true
	return &x, nil
}

// MarkRead **ما وصل يُوسَم** — ويردّ كم وُسم.
//
// **ويوسم رسائلَ الطرف الآخر وحدَها**: من وسم رسائلَه هو جعل «غيرُ مقروء»
// عند صاحبه مقروءاً عنده.
func (s *Service) MarkRead(ctx context.Context, p *Permission) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE order_messages SET read_at = now()
		WHERE order_id = $1 AND sender_role <> $2 AND read_at IS NULL`,
		p.OrderID, string(p.Me))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Unread **كم ينتظره في هذا الطلب** — لشارةٍ في القائمة.
func (s *Service) Unread(ctx context.Context, orderID string, me Role) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM order_messages
		WHERE order_id = $1 AND sender_role <> $2 AND read_at IS NULL`,
		orderID, string(me)).Scan(&n)
	return n, err
}
