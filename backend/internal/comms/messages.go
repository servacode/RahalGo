package comms

import (
	"context"
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
	// Flagged **أفيها لفظٌ لا يُقال؟** — تُعرض للمكتب ولا تُمنع.
	Flagged bool `json:"flagged,omitempty"`
	// FlagWord **أيُّ لفظٍ أوقعها** — للمراجعة، **ولمراجعة الحارس نفسِه**
	// حين يَسِم بريئا. **وللإدارة وحدَها.**
	FlagWord string `json:"flag_word,omitempty"`
}

// List **حديثُ الطلب كما يراه أحدُ طرفيه.**
//
// **ويُقرأ حتّى بعد الإغلاق**: القناةُ تُقفل للكتابة لا للقراءة — **وحديثٌ
// يختفي بانتهاء الطلب يمحو ما يُحتجّ به** عند شكوى.
//
// **وولايةٌ واحدةٌ لا الطلبُ كلُّه** (قرارُ المالك ٢٠٢٦-١٠-٠٢): السائقُ
// الثاني يبدأ حديثاً فارغاً، **والأوّلُ لا يرى ما قيل بعد خروجه.**
func (s *Service) List(ctx context.Context, p *Permission) ([]Message, error) {
	if p.DriverID == "" {
		return []Message{}, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, body, sender_role, sender_id::text = $2, created_at, read_at
		FROM order_messages WHERE order_id = $1 AND driver_id = $3
		ORDER BY created_at`, p.OrderID, p.SelfID, p.DriverID)
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
	// **والتنقيةُ قبل كلّ شيء** — محارفُ تخدع العين تُسقط، **وما بعدها
	// يُقاس على النصّ الذي سيُخزَّن فعلا** لا على ما وصل.
	body = sanitize(body)
	if body == "" {
		return nil, ErrEmptyBody
	}
	if len([]rune(body)) > MaxBody {
		body = string([]rune(body)[:MaxBody])
	}
	// **والشتيمةُ تُوسَم ولا تُمنع** — تصل كما كُتبت، **والشكوى تُحسم
	// بنصٍّ مكتوبٍ لا بكلمةٍ ضدّ كلمة.**
	word := Offense(body)

	// **وحدُّ المعدّل يُفحص في القاعدة لا في الذاكرة** — خادمان يعملان معاً
	// **وذاكرةٌ في أحدهما لا يراها الآخر.**
	var recent bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM order_messages
		               WHERE order_id = $1 AND sender_id = $2 AND NOT auto
		                 AND created_at > now() - $3::interval)`,
		p.OrderID, p.SelfID, MinGap.String()).Scan(&recent); err != nil {
		return nil, err
	}
	if recent {
		return nil, ErrTooFast
	}

	// **والولايةُ من الصلاحية** — سائقُ الطلب لحظةَ قُرئت. **وفراغُها يترك
	// القاعدةَ تقرأ حاملَ الطلب** (`order_messages_tenure`).
	var tenure any
	if p.DriverID != "" {
		tenure = p.DriverID
	}
	var x Message
	if err := s.db.QueryRow(ctx, `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body, flagged, flag_word, driver_id)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		RETURNING id::text, body, sender_role, created_at`,
		p.OrderID, p.SelfID, string(p.Me), body, word != "", word, tenure).
		Scan(&x.ID, &x.Body, &x.Role, &x.CreatedAt); err != nil {
		return nil, err
	}
	x.Mine = true
	x.Flagged = word != ""
	return &x, nil
}

// MarkRead **ما وصل يُوسَم** — ويردّ كم وُسم.
//
// **ويوسم رسائلَ الطرف الآخر وحدَها**: من وسم رسائلَه هو جعل «غيرُ مقروء»
// عند صاحبه مقروءاً عنده.
//
// **وفي ولايته وحدَها، ولمن وُجّهت إليه وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٢):
// **سائقٌ ترك الطلبَ يفتح الحديثَ فلا يَسِم شيئاً** — كان يَسِم رسائلَ الزبون
// إلى السائق الثاني مقروءةً، **فيرى الزبونُ «قُرئت» ممّن لم يقرأها.**
func (s *Service) MarkRead(ctx context.Context, p *Permission) (int64, error) {
	if p.Former || p.DriverID == "" {
		return 0, nil
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE order_messages SET read_at = now()
		WHERE order_id = $1 AND driver_id = $3 AND sender_role <> $2 AND read_at IS NULL`,
		p.OrderID, string(p.Me), p.DriverID)
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

// Thread **محادثةٌ كما تُعرض في القائمة.**
type Thread struct {
	OrderID   string     `json:"order_id"`
	Number    int64      `json:"number"`
	Peer      string     `json:"peer"`
	Open      bool       `json:"open"`
	Unread    int        `json:"unread"`
	LastBody  string     `json:"last_body"`
	LastAt    *time.Time `json:"last_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// Threads **كلُّ محادثاتِ هذا الحساب — المفتوحةُ والمنتهية.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن يكون هناك دردشاتي السابقة… مشان إثبات».)
//
// # ولماذا المنتهيةُ معها
//
// **الحديثُ حجّةٌ عند الخلاف** — ومن اتُّفق معه على سعرٍ ثمّ أُنكر يرجع إليه.
// **ومحادثةٌ تختفي بانتهاء الطلب تمحو الدليلَ في اللحظة التي يُحتاج فيها**:
// لا يُختلَف أثناء الطلب، **إنّما بعده.**
//
// # ونداءٌ واحدٌ يخدم الفقّاعةَ والسجلّ
//
// **والفقّاعةُ تأخذ المفتوحَ وحدَه، والسجلُّ يأخذ الكلّ** — ونداءان لقائمةٍ
// واحدةٍ يفترقان: **يُصلَح عدُّ غيرِ المقروء في أحدهما ويبقى الآخرُ يكذب.**
func (s *Service) Threads(ctx context.Context, userID string) ([]Thread, error) {
	// ══════════════════════════════════════════════════════════════════
	// **ويُبدأ من الرسائل لا من الطلبات**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٥: «الدردشاتُ السابقةُ تتأخّر كثيراً لتفتح،
	//  تبقى جاري التحميل».)
	//
	// **وكان الاستعلامُ يبدأ من `orders`** ويسأل عن كلّ طلبٍ في القاعدة
	// **«أفيه رسالة؟»** — **ثمّ يرتّب بـ`max(created_at)` محسوباً في
	// جملة الترتيب نفسِها**، فيُعاد حسابُه لكلّ صفّ.
	//
	// **وأربعُ جملٍ مرتبطةٍ لكلّ صفّ** — والصفوفُ كلُّ طلبات المنصّة.
	// **فالكلفةُ تنمو بعدد الطلبات لا بعدد محادثاته هو**: من له
	// محادثتان يدفع ثمنَ عشرة آلاف طلب.
	//
	// **والرسائلُ هي الجدولُ الصغير** — وفيها فهرسان (`order_id` و
	// `sender_id`). **فتُجمَّع مرّةً واحدةً ثمّ تُضمّ الطلباتُ إليها.**
	//
	// **ولا يتبدّل ما تراه**: الشرطُ نفسُه — طرفٌ في الطلب **أو** كتب
	// فيه بيده (سائقٌ سُحب منه الطلب). **وسجلٌّ يراه طرفٌ ولا يراه
	// الآخرُ ليس إثباتاً** — إنّما حجّةٌ في يدٍ واحدة.
	//
	// # وولايةٌ واحدةٌ لكلّ طلب (قرارُ المالك ٢٠٢٦-١٠-٠٢)
	//
	// **الزبونُ يرى ولايةَ سائقه الآن** — أو آخرَ ولايةٍ إن لم يكن له سائق.
	// **والسائقُ يرى ولايتَه هو** ولو ترك الطلب: **لا عدَّ غيرِ مقروءٍ ولا
	// آخرَ سطرٍ من حديث من جاء بعده.**
	rows, err := s.db.Query(ctx, `
		WITH ids AS (
		    SELECT o.id AS order_id,
		           COALESCE(o.driver_id,
		                    (SELECT x.driver_id FROM order_messages x
		                     WHERE x.order_id = o.id AND x.driver_id IS NOT NULL
		                     ORDER BY x.created_at DESC LIMIT 1)) AS tenure
		    FROM orders o
		    WHERE o.customer_id = $1
		    UNION
		    SELECT DISTINCT om.order_id, om.driver_id
		    FROM order_messages om WHERE om.driver_id = $1
		), t AS (
		    SELECT om.order_id, ids.tenure,
		           max(om.created_at) AS last_at,
		           count(*) FILTER (
		               WHERE om.read_at IS NULL AND om.sender_id <> $1
		           ) AS unread,
		           -- **وآخرُ ما قيل** — يُلتقط في المرور نفسِه لا بجملةٍ
		           -- ثانيةٍ لكلّ صفّ.
		           (array_agg(om.body ORDER BY om.created_at DESC))[1] AS last_body
		    FROM order_messages om
		    JOIN ids ON ids.order_id = om.order_id AND om.driver_id = ids.tenure
		    GROUP BY om.order_id, ids.tenure
		)
		SELECT o.id::text, o.number, o.status, o.delivered_at, o.closed_at,
		       -- **والطرفُ الآخر بحسب من يسأل** — كلٌّ يرى الآخر.
		       --
		       -- **والزبونُ يرى صاحبَ الولاية** — لا حاملَ الطلب الآن وحدَه:
		       -- **ومن سُحب الطلبُ من يده يبقى اسمُه على ولايته.**
		       COALESCE(CASE WHEN o.customer_id = $1 THEN dr.full_name
		                     ELSE cu.full_name END, ''),
		       -- **وطرفٌ الآن أم خرج** — سائقٌ ترك الطلبَ لا تُفتح له
		       -- القناةُ ولو كان الطلبُ في الطريق مع غيره.
		       COALESCE((o.customer_id = $1 AND o.driver_id IS NOT NULL)
		                OR o.driver_id = $1, false),
		       -- **ومن ترك الطلبَ لا شارةَ له** — لا يَسِم شيئاً بعد خروجه
		       -- (MarkRead)، **فعدٌّ يبقى في سجلّه أحمرَ لا يُطفأ أبداً.**
		       CASE WHEN o.customer_id = $1 OR o.driver_id = $1
		            THEN t.unread ELSE 0 END,
		       COALESCE(t.last_body, ''), t.last_at, o.created_at
		FROM t
		JOIN orders o ON o.id = t.order_id
		JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN users dr ON dr.id = t.tenure
		ORDER BY t.last_at DESC
		LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Thread{}
	for rows.Next() {
		var t Thread
		var status string
		var deliveredAt, closedAt *time.Time
		var partyNow bool
		if err := rows.Scan(&t.OrderID, &t.Number, &status, &deliveredAt, &closedAt,
			&t.Peer, &partyNow, &t.Unread, &t.LastBody, &t.LastAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		// **والحكمُ من `channelOpen` نفسِها** — لا شرطٌ يشبهه.
		t.Open, _ = channelOpen(status, deliveredAt, closedAt)
		t.Open = t.Open && partyNow
		out = append(out, t)
	}
	return out, rows.Err()
}
