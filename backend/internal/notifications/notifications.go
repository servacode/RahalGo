// Package notifications مركز الإشعارات الموحّد للمنصة.
//
// قاعدة مركزية: أي حدث مهم يمرّ من هنا فقط — يُحفظ في صندوق وارد المستخدم ثم
// يُبثّ حياً. الحفظ أولاً كي لا يضيع الإشعار على من كان غير متصل، والبث ليصل
// فوراً لمن هو متصل بلا أي تحديث للصفحة.
package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher واجهة البث الحي (نفس واجهة محرك الطلبات).
type Publisher interface {
	Publish(topic string, event any)
}

// ══════════════════════════════════════════════════════════════════════
// **والدافعُ واجهةٌ هنا لا استيرادٌ لحزمته**
// ══════════════════════════════════════════════════════════════════════
//
// **`notifications` تقرّر مَن يُبلَّغ، و`push` تقرّر كيف يصل.** ولو
// استوردت هذه تلك لَصار منطقُ المنصّة يعرف جوجل، **وتبديلُ ناقلٍ يمسّ
// كلَّ موضعٍ يُنشئ إشعاراً.**
//
// **والواجهةُ عند المستهلِك لا عند المنتِج** — كما `Publisher` فوقها.
type Pusher interface {
	SendToUser(ctx context.Context, userID string, msg PushMessage)
}

// PushMessage **صورةٌ محلّيّةٌ من رسالة الدفع** — تُبنى في المُهيِّئ.
//
// **ونسخُ الحقول أرخصُ من استيرادِ حزمة**: أربعةُ حقولٍ لا تتبدّل،
// **مقابل تبعيّةٍ في الاتّجاه الخطأ تبقى سنوات.**
type PushMessage struct {
	Title  string
	Body   string
	Data   map[string]string
	Urgent bool
	Apps   []string
}

// ══════════════════════════════════════════════════════════════════════
//
//	**التطبيقاتُ التي يخصّها الإشعار**
//
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١: «لازم يكون هناك فصلٌ بين العمليات والإشعارات،
//
//	الشيءُ المشترَك الوحيد هو المحفظة».)
//
// **صاحبُ المتجر يحمل تطبيقين** — تطبيقَ متجرِه وتطبيقَ الزبون ليطلب
// عشاءه (`FieldRolesAreCustomers`). **و«طلبٌ جديد» يخصّ متجرَه وحدَه.**
//
// **والأسماءُ هي أسماءُ التطبيقات في `identity` نفسِها** — ولو افترقتا
// لَذهب الإشعارُ إلى تطبيقٍ لا وجودَ له، **ولا شيءَ يكشف ذلك إلّا صمتٌ.**
const (
	AppCustomer = "customer"
	AppDriver   = "driver"
	AppMerchant = "merchant"
	AppRep      = "rep"
)

// أنواع الإشعارات — قائمة مركزية تُستعمل في الخادم والواجهة.
const (
	KindOrder   = "order"
	KindTicket  = "ticket"
	KindWallet  = "wallet"
	KindRating  = "rating"
	KindLead    = "lead"
	KindAccount = "account"
)

type Service struct {
	db     *pgxpool.Pool
	hub    Publisher
	logger *slog.Logger
	// **وقد يكون فارغاً** — المنصّةُ عملت بلا دفعٍ حتّى ٢٠٢٦-٠٨-١١،
	// **وجهازُ المطوّر بلا مفتاح.**
	pusher Pusher
}

func New(db *pgxpool.Pool, hub Publisher, logger *slog.Logger) *Service {
	return &Service{db: db, hub: hub, logger: logger}
}

// SetPusher يربط الدفعَ بالإشعارات — **تُنادى مرّةً عند الإقلاع.**
//
// **وبربطٍ لاحقٍ لا في المُنشئ**: `New` يُنادى في ستّة اختباراتٍ لا دفعَ
// فيها، **وحقلٌ إلزاميٌّ في المُنشئ يُجبرها كلَّها على تمرير `nil`.**
func (s *Service) SetPusher(p Pusher) { s.pusher = p }

// Notification إشعار واحد كما يُعاد للواجهة.
//
// ══════════════════════════════════════════════════════════════════════
// **ولا وجهةَ في الإشعار — يُخبِر ولا ينقل**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الإشعاراتُ عند الضغط تفتح صفحاتٍ غيرَ موجودة،
//
//	لذلك يكفي أن تكون مقروءةً وغيرَ مقروءة وتوصل لنا الخبرَ بماذا حدث».)
//
// **وقد صدق: `Href` كانت تُملأ بمساراتٍ لا وجودَ لها** — «‎/portal» في
// معالِج السائقين مثلاً، **وهو مسارٌ من زمنِ التطبيقات الخمسة** قبل أن
// تصير اللوحاتُ أقساماً في بيتٍ واحد. **ولا حارسَ يمنع كتابةَ مسارٍ ميّت
// في نصٍّ حرّ.**
//
// **والشاشةُ توقّفت عن استعمالها** (٢٠٢٦-٠٨-١١)، **لكنّها بقيت تُرسَل** —
// **وحقلٌ يصل الواجهةَ يُستعمَل يوماً**، ولو بعد سنة، ويعود العطبُ نفسُه.
//
// **فلا تُرسَل أصلاً.** ويبقى العمودُ في القاعدة و`Input.Href` في أربعةٍ
// وأربعين موضعاً — **تاريخٌ مكتوبٌ لا يبلغ شاشة**، وتنظيفُه دفعةٌ ميكانيكيّةٌ
// وحدَها.
type Notification struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
	// Transient **يرنّ ولا يُحفَظ** — تعرضه الشاشةُ لحظةً ولا تُضيفه
	// إلى صندوقها. **وغيابُ الحقل في المحفوظ صمتٌ صحيح**: `omitempty`.
	Transient bool `json:"transient,omitempty"`
}

// Input بيانات إنشاء إشعار.
type Input struct {
	UserID   string
	Kind     string
	Title    string
	Body     string
	Entity   string
	EntityID string
	Href     string
	// Apps **التطبيقاتُ التي يخصّها هذا الإشعار — وفارغةٌ تعني كلَّها.**
	//
	// **والفارغةُ تعني كلَّها لا لا شيء**: أكثرُ المواضع لا تُصرّح،
	// **وافتراضُ الصمت يجعل الإشعاراتِ تختفي دفعةً واحدةً بلا أثرٍ في
	// سجلّ.** فمن عرف جمهورَه صرّح، ومن لم يعرف بقي كما كان.
	//
	// **ولا تمسّ الصندوقَ ولا البثَّ الحيّ** — هي للدفع وحدَه: الخبرُ
	// يُحفظ لصاحبه كاملاً، **والتوجيهُ إنّما يقرّر أيُّ تطبيقٍ يرنّ.**
	Apps []string

	// ══════════════════════════════════════════════════════════════════
	// **Transient — يرنّ ولا يُحفَظ**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «تقدّمُ حالات الطلب لا تلزمنا في
	//  الإشعارات، تظهر كإشعاراتٍ ثمّ تختفي… الزبونُ يصله إشعارٌ «طلبك في
	//  الطريق إليك» بلا أن ينزعج من كثرة الإشعارات ويفوت على صفحةٍ فيها
	//  الكثير بلا فائدة».)
	//
	// # والضجيجُ كلُّه من أربعة
	//
	// **أربعةُ إشعاراتٍ تتكرّر مع كلّ طلب** — قُبل، يُجهَّز، في الطريق،
	// سُلّم. **وعشرون طلباً في الشهر تعني ثمانين سطراً في صندوقه**،
	// **وما عداها يقع مرّةً أو مرّتين.** فصفحةُ الإشعارات تصير كومةً لا
	// تُقرأ، **ومن كفّ عن قراءتها لا يقرأ ما يهمّ.**
	//
	// # والقاعدةُ التي تفصل
	//
	// **أيرجع إليه بعد يومين؟** «خُصم من محفظتك خمسمئة» نعم — سجلٌّ
	// ماليٌّ يُراجَع. **و«طلبك في الطريق» لا** — بعد ساعةٍ لا معنى لها،
	// **والبطاقةُ تقول حالَ الطلب أصدقَ من خبرٍ قديم.**
	//
	// # ولا تُسكَت الرنّة
	//
	// **الخبرُ يصل كما كان**: بثٌّ حيٌّ إلى الشاشة المفتوحة، **ودفعٌ إلى
	// الهاتف المقفل.** والذي يسقط هو الصفُّ في القاعدة وحدَه.
	Transient bool

	// ══════════════════════════════════════════════════════════════════
	// **Silent — يُحفَظ ولا يرنّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا نريد إشعاراتٍ كثيرةً بلا فائدة».)
	//
	// **وهي نقيضُ `Transient`**: تلك تُرنّ ولا تُحفَظ، **وهذه تُحفَظ ولا
	// تُرنّ.** وبينهما القاعدةُ كلُّها:
	//
	//	ما يُنتظَر فيه فعلٌ الآن   →  يرنّ ولا يُحفَظ
	//	ما يُقرأ لاحقاً وله أثر    →  يُحفَظ ولا يرنّ
	//
	// # ولماذا يُسكَت المال
	//
	// **أجرُ التوصيل يُقيَّد لحظةَ التسليم** — والسائقُ حينها في التطبيق،
	// **يقف عند الباب وقد ضغط «سلّمت» قبل ثانية.** **ورنّةٌ تخبره بما
	// فعله للتوّ ضجيجٌ**، ورقمُ محفظته في الشريط فوقه.
	//
	// **ويبقى الصفُّ في صندوقه** — لأنّه مال: يُراجَع بعد يومين، **ويُحتجّ
	// به في خلاف.**
	//
	// # ولا يُسكَت ما ينتظر فعلا
	//
	// **الرنّةُ ثمنُها انتباهُ صاحبها** — ومن دفعه في خبرٍ لا فعلَ فيه
	// **لم يبقَ له ما يدفعه حين يجيء العرض.**
	Silent bool
}

// Notify يحفظ الإشعار ويبثّه لصاحبه فوراً. لا يُفشل العملية الأصلية أبداً —
// فشل الإشعار يُسجَّل ولا يُرجع خطأً للمستدعي.
func (s *Service) Notify(ctx context.Context, in Input) {
	// **وخدمةٌ غيرُ مهيّأة تصمت ولا تُسقط.**
	//
	// وهو عهدُ هذه الحزمة المكتوبُ أعلاه: **«لا يُفشل العملية الأصلية أبداً».**
	// **وذعرٌ يُسقط النداءَ كلَّه أشدُّ من خطأٍ يُرجَع** — يُبطل الطارئَ الذي
	// سُجّل، والانتقالَ الذي وقع، **ويردّ خمسمئة على فعلٍ نجح.**
	if s == nil || in.UserID == "" || in.Title == "" {
		return
	}
	var id, createdAt string
	if in.Transient {
		// **ولا صفَّ له** — يرنّ ويمضي. **والمعرّفُ يبقى فارغاً**: شاشةٌ
		// تحاول أن تعلّمه مقروءاً تنادي على ما لا وجودَ له، **فتُردّ
		// بأربعمئةٍ على فعلٍ لا يعني شيئا.**
		createdAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		err := s.db.QueryRow(ctx, `
			INSERT INTO notifications (user_id, kind, title, body, entity, entity_id, href)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at::text`,
			in.UserID, in.Kind, in.Title, in.Body, in.Entity, in.EntityID, in.Href).
			Scan(&id, &createdAt)
		if err != nil {
			s.logger.Error("notify: insert", "error", err, "user", in.UserID)
			return
		}
	}
	s.hub.Publish("user:"+in.UserID, map[string]any{
		"type": "notification",
		"notification": Notification{
			ID: id, Kind: in.Kind, Title: in.Title, Body: in.Body,
			Entity: in.Entity, EntityID: in.EntityID,
			Read: false, CreatedAt: createdAt,
			// **وتُوسَم عابرةً في البثّ** — الشاشةُ ترفعها لحظةً ولا
			// تضيفها إلى صندوقها، **ولا تزيد عدّادَ غير المقروء.**
			Transient: in.Transient,
		},
	})

	// ══════════════════════════════════════════════════════════════════
	// **ويُدفَع إلى أجهزته — لِما يصل والشاشةُ مقفلة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **البثُّ الحيُّ يكفي المتصفّح** — تبويبٌ مفتوحٌ فيصل، ومغلقٌ فيُقرأ
	// عند العودة. **والهاتفُ مقفلٌ أكثرَ ممّا هو مفتوح**، **والسائقُ الذي
	// يعلم بطلبٍ بعد دقيقتين طلبٌ ضائع.**
	//
	// **وبعد الحفظ لا قبله**: الصندوقُ هو الحقيقة، **والدفعُ تنبيهٌ إليها.**
	// فإن سقط الدفعُ بقي الخبرُ محفوظاً يُقرأ عند الفتح.
	//
	// **والعاجلُ ما يُنتظَر فيه فعل**: طلبٌ ينتظر سائقاً أو متجراً.
	// **ولا تُرفع الأولويّةُ لكلّ شيء** — جوجل تخفض حصّةَ من يُسيء
	// استعمالها **فتتأخّر رسائلُه كلُّها، بما فيها العاجل.**
	// **ولا رنّةَ لِما لا فعلَ فيه** — انظر `Silent` أعلاه.
	if s.pusher != nil && !in.Silent {
		s.pusher.SendToUser(ctx, in.UserID, PushMessage{
			Title: in.Title,
			Body:  in.Body,
			Data: map[string]string{
				"kind":      in.Kind,
				"entity":    in.Entity,
				"entity_id": in.EntityID,
			},
			Urgent: in.Kind == KindOrder,
			Apps:   in.Apps,
		})
	}
}

// NotifyMany يرسل الإشعار نفسه لعدة مستخدمين (مثلاً كل العمليات).
func (s *Service) NotifyMany(ctx context.Context, userIDs []string, in Input) {
	for _, uid := range userIDs {
		in.UserID = uid
		s.Notify(ctx, in)
	}
}

// OpsDesk أدوار مكتب المنصة — من يجب أن يعرف بأي حركة تشغيلية جديدة.
// مصدر واحد: لا يقرر كل معالِج بنفسه من يُبلَّغ.
var OpsDesk = []string{"admin", "ops"}

// NotifyWallet إشعار حركة مالية — يُرضي واجهة orders.Notifier.
func (s *Service) NotifyWallet(ctx context.Context, userID, title, body, href string) {
	s.Notify(ctx, Input{
		UserID: userID, Kind: KindWallet, Title: title, Body: body,
		Entity: "wallet", Href: href,
	})
}

// NotifyRole يرسل الإشعار لكل حاملي دور معيّن.
func (s *Service) NotifyRole(ctx context.Context, role string, in Input) {
	s.NotifyRoles(ctx, []string{role}, in)
}

// NotifyShoppers **يبلّغ من يتسوّق وحدَه — لا فريقَ العمل.**
//
// (بلاغُ المالك ٢٠٢٦-٠٨-١٩: «إشعاراتُ العروض تأتي إلى حساب السائق وهذا
//
//	غلط لأنّه لا يوجد تسوّقٌ بحساب السائق».)
//
// # ولماذا لا يكفي `NotifyRole("customer")`
//
// **`customer` دورُ أساسٍ يحمله كلُّ حساب** — والسائقُ والمتجرُ والمندوبُ
// كلُّهم زبائنُ في القاعدة. **فبثٌّ إلى «كلّ من يحمل دورَ الزبون» يبلغ
// فريقَ العمل كلَّه.**
//
// **وقِيس على الإنتاج**: وصل السائقَ «عرض حاص — شاورما غنم ١٥٪» مرّتين.
//
// # والمتسوّقُ من لا دورَ تشغيليَّ له
//
// **والتعريفُ قائمٌ في المستودع** (`identity.primaryRoles`): الزبونُ
// دورُ أساسٍ وما عداه أصليّ. **فمن حمل أصليّاً فهو من فريق العمل.**
//
// **ولا تُعدَّد الأدوارُ المستثناةُ بيد** — **وقائمةٌ تُكتب بيدٍ تنسى
// دوراً يُضاف غدا.**
func (s *Service) NotifyShoppers(ctx context.Context, in Input) {
	s.notifyQuery(ctx, in, `
		SELECT u.id FROM users u
		JOIN user_roles c ON c.user_id = u.id AND c.role_code = 'customer'
		WHERE u.status = 'active'
		  AND NOT EXISTS (
		      SELECT 1 FROM user_roles o
		      WHERE o.user_id = u.id AND o.role_code <> 'customer'
		  )`)
}

// NotifyOps يبلّغ مكتب المنصة كاملاً (مالك المنصة + العمليات) بلا تكرار.
func (s *Service) NotifyOps(ctx context.Context, in Input) {
	s.NotifyRoles(ctx, OpsDesk, in)
}

// NotifyRoles يرسل الإشعار لحاملي أي من الأدوار المذكورة — مرة واحدة لكل شخص
// مهما تعددت أدواره.
// notifyQuery **يبلّغ من يردّه استعلامٌ يرجع معرّفات** — أساسُ
// `NotifyRoles` و`NotifyShoppers`.
//
// **ونسختان من حلقةٍ واحدةٍ تفترقان يومَ يُضاف شرط.**
func (s *Service) notifyQuery(ctx context.Context, in Input, query string, args ...any) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error("notify: query", "error", err)
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	s.NotifyMany(ctx, ids, in)
}

func (s *Service) NotifyRoles(ctx context.Context, roles []string, in Input) {
	// **والحارسُ هنا كما في `Notify`** — هذه تمسّ القاعدةَ بنفسها فلا يحميها
	// حارسُ تلك.
	if s == nil {
		return
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`, roles)
	if err != nil {
		s.logger.Error("notify: role query", "error", err, "roles", roles)
		return
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	s.NotifyMany(ctx, ids, in)
}

// List إشعارات المستخدم (الأحدث أولاً) مع عدد غير المقروء.
// kind فارغ = كل الأنواع (الجرس يطلب الأحدث، والصفحة الكاملة تفلتر وتوسّع الحد).
func (s *Service) List(ctx context.Context, userID string, limit int, kind string) ([]Notification, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, kind, title, body, entity, entity_id,
		       (read_at IS NOT NULL), created_at::text
		FROM notifications WHERE user_id = $1 AND ($3 = '' OR kind = $3)
		ORDER BY created_at DESC LIMIT $2`, userID, limit, kind)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Entity,
			&n.EntityID, &n.Read, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	var unread int
	_ = s.db.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`,
		userID).Scan(&unread)
	return out, unread, rows.Err()
}

// MarkRead يعلّم إشعاراً (أو الكل عند تمرير معرّف فارغ) كمقروء.
func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if id == "" {
		_, err := s.db.Exec(ctx,
			`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID)
		return err
	}
	_, err := s.db.Exec(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND id = $2`, userID, id)
	return err
}
