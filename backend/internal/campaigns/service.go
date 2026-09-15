package campaigns

// ══════════════════════════════════════════════════════════════════════
// **الحملة — تُكتب أوّلاً ثمّ تُرسَل** (`NT`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا محرّكَ إشعارٍ ثانٍ
//
// **والإرسالُ كلُّه يمرّ بـ`notifications.Notify`** — **هو من يكتب في
// الصندوق ويرفع نيّةَ الدفع، وعاملُ النقل يتكفّل بالباقي** (`PF-09`).
// **ولا يُنادى `FCM` من هنا إطلاقاً.**
//
// # والصفُّ قبل الفعل
//
// **ومن أرسل ثمّ كتب لا يعرف ماذا أرسل إن مات بينهما** — **فالصفُّ
// يُكتب أوّلاً بحالٍ `sending`، ثمّ يُرسَل، ثمّ تُثبَّت النتيجة.**

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	ErrBadAudience = httpx.NewError(http.StatusBadRequest, "bad_audience", "errors.validation")
	ErrBadDest     = httpx.NewError(http.StatusBadRequest, "bad_destination", "errors.validation")
	ErrBadTime     = httpx.NewError(http.StatusBadRequest, "bad_schedule", "errors.validation")
	ErrNeedsTitle  = httpx.NewError(http.StatusBadRequest, "campaign_needs_title", "errors.validation")
	// ErrNotCancellable **وما بدأ إرسالُه لا يُسحَب.**
	ErrNotCancellable = httpx.NewError(http.StatusConflict, "campaign_not_cancellable", "errors.validation")
	ErrGone           = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")
)

// Notifier **ما تحتاجه الحملةُ من محرّك الإشعارات** — **ولا أكثر.**
//
// **وواجهةٌ ضيّقةٌ تمنع أن يصير هذا محرّكاً ثانياً** — **فليس بيده
// إلّا أن يُرسل خبراً لإنسان.**
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body, entity, entityID string)
}

type Service struct {
	db    *pgxpool.Pool
	notif Notifier
	// Now **ساعةُ الخادم** — **تُحقَن ليُقاس الزمنُ في فحص.**
	Now func() time.Time
	// Quiet و Cap **سياسةٌ تُقرأ من الإعدادات وقتَ التنفيذ.**
	QuietOf func(ctx context.Context) Quiet
	CapOf   func(ctx context.Context) int

	// Serviceable **أوصلت الخدمةُ إلى هذا الهدف فعلاً؟**
	//
	// **ويُحقَن من الخادم** — **ولا تعرف هذه الحزمةُ الطلباتِ ولا
	// المناطق**: **وحزمةٌ تعرف كلَّ شيءٍ تصير محرّكاً ثانياً.**
	//
	// **وفارغُه يعني «لا يُعاد التقييم»** — **وهو حالُ الفحص الذي
	// لا يقيس هذا.**
	Serviceable func(ctx context.Context, targetKey string) (bool, error)

	// LiveOffer **أهذا العرضُ سارٍ الآن؟**
	//
	// **ويُسأل عند الإنشاء وعند الإرسال معاً** — **فحملةٌ جُدولت
	// لعرضٍ انتهى بينهما وعدٌ كاذب.**
	LiveOffer func(ctx context.Context, offerID string) (bool, error)
}

func New(db *pgxpool.Pool, n Notifier) *Service {
	return &Service{
		db: db, notif: n,
		Now:     func() time.Time { return time.Now() },
		QuietOf: func(context.Context) Quiet { return DefaultQuiet },
		CapOf:   func(context.Context) int { return DefaultDailyCap },
	}
}

// Campaign **الحملةُ كما تُقرأ في اللوحة.**
type Campaign struct {
	ID           string     `json:"id"`
	Category     string     `json:"category"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	AudienceType string     `json:"audience_type"`
	AudienceRef  string     `json:"audience_ref"`
	DestType     *string    `json:"dest_type"`
	DestID       *string    `json:"dest_id"`
	Status       string     `json:"status"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
	CreatedAt    time.Time  `json:"created_at"`
	SentAt       *time.Time `json:"sent_at"`
	CancelledAt  *time.Time `json:"cancelled_at"`
	// **والعدُّ يقول ما يُعرَف** — **ولا «وصلت» ولا «قُرئت».**
	Targeted     int    `json:"targeted"`
	InboxCreated int    `json:"inbox_created"`
	Deferred     int    `json:"deferred"`
	Error        string `json:"error"`
}

const selectCampaign = `
	SELECT id::text, category, title, body, audience_type, audience_ref,
	       dest_type, dest_id::text, status, scheduled_at, created_at,
	       sent_at, cancelled_at, targeted, inbox_created, deferred, error
	FROM campaigns`

func scan(row pgx.Row) (*Campaign, error) {
	var c Campaign
	if err := row.Scan(&c.ID, &c.Category, &c.Title, &c.Body, &c.AudienceType,
		&c.AudienceRef, &c.DestType, &c.DestID, &c.Status, &c.ScheduledAt,
		&c.CreatedAt, &c.SentAt, &c.CancelledAt, &c.Targeted, &c.InboxCreated,
		&c.Deferred, &c.Error); err != nil {
		return nil, err
	}
	return &c, nil
}

// Input **ما ترسله اللوحة.**
type Input struct {
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	AudienceType string     `json:"audience_type"`
	AudienceRef  string     `json:"audience_ref"`
	DestType     string     `json:"dest_type"`
	DestID       string     `json:"dest_id"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
}

func (s *Service) validate(in Input) error {
	if strings.TrimSpace(in.Title) == "" {
		return ErrNeedsTitle
	}
	if !ValidAudience(in.AudienceType, in.AudienceRef) {
		return ErrBadAudience
	}
	if !ValidDest(in.DestType, in.DestID) {
		return ErrBadDest
	}
	return nil
}

// ErrOfferNotLive **عرضٌ لا يُسعَّر به لا يُعلَن عنه** (`EN-01`، `EN-02`).
//
// **ومن أعلن عرضاً منتهياً أو مُنزَلاً وعد بما لا يقع** — **يفتحه
// الزبونُ فيجد السعرَ كما كان**، **فيظنّ أنّنا نكذب — وهو محقّ.**
//
// **والمجدولُ كذلك** (`EN-03`) — **وعدٌ لم يحلّ بعدُ لا يُقال إنّه
// قائمٌ الآن.**
var ErrOfferNotLive = httpx.NewError(http.StatusConflict, "offer_not_live", "errors.validation")

// checkOffer **يُسأل عن العرض إن كانت الوجهةُ عرضاً.**
func (s *Service) checkOffer(ctx context.Context, in Input) error {
	if in.DestType != DestOffer || s.LiveOffer == nil {
		return nil
	}
	live, err := s.LiveOffer(ctx, in.DestID)
	if err != nil {
		return ErrBadDest
	}
	if !live {
		return ErrOfferNotLive
	}
	// **وموعدٌ مضى ليس جدولة** — **ومن جدول للأمس أراد الإرسالَ الآن
	// ولم يقله**: **فيُردّ ليقوله.**
	if in.ScheduledAt != nil && !in.ScheduledAt.After(s.Now()) {
		return ErrBadTime
	}
	return nil
}

// Create **يكتب الحملةَ — مسوّدةً أو مجدولة.**
//
// **والهويّةُ تمنع الازدواج** — **وضغطتان تجدان صفّاً واحداً.**
func (s *Service) Create(ctx context.Context, actorID, idemKey string, in Input) (*Campaign, error) {
	if err := s.validate(in); err != nil {
		return nil, err
	}
	if err := s.checkOffer(ctx, in); err != nil {
		return nil, err
	}
	status := StatusDraft
	if in.ScheduledAt != nil {
		status = StatusScheduled
	}
	var destType, destID any
	if in.DestType != "" && in.DestType != DestHome {
		destType, destID = in.DestType, in.DestID
	} else if in.DestType == DestHome {
		destType = DestHome
	}

	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO campaigns (category, title, body, audience_type, audience_ref,
		                       dest_type, dest_id, status, scheduled_at,
		                       created_by, idem_key)
		VALUES ('engagement', $1, $2, $3, $4, $5, $6::uuid, $7, $8, $9, $10)
		ON CONFLICT (created_by, idem_key) DO UPDATE SET updated_at = now()
		RETURNING id::text`,
		strings.TrimSpace(in.Title), in.Body, in.AudienceType,
		strings.TrimSpace(in.AudienceRef), destType, destID, status,
		in.ScheduledAt, actorID, idemKey).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id string) (*Campaign, error) {
	c, err := scan(s.db.QueryRow(ctx, selectCampaign+` WHERE id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGone
	}
	return c, err
}

// List **تاريخُ ما أُرسل** — **والأحدثُ أوّلاً.**
func (s *Service) List(ctx context.Context, limit int) ([]Campaign, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, selectCampaign+` ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Campaign{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// Cancel **يُلغي ما لم يبدأ.**
//
// **ويُقفَل الصفُّ قبل القراءة** — **وإلّا ألغى اثنان معاً ما بدأ
// إرسالُه بينهما.**
func (s *Service) Cancel(ctx context.Context, id string) (*Campaign, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx,
		`SELECT status FROM campaigns WHERE id = $1::uuid FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGone
	}
	if err != nil {
		return nil, err
	}
	if !Cancellable(status) {
		return nil, ErrNotCancellable
	}
	if _, err := tx.Exec(ctx, `
		UPDATE campaigns SET status = 'cancelled', cancelled_at = now(),
		       updated_at = now() WHERE id = $1::uuid`, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// ══════════════════════════════════════════════════════════════════════
// **الجمهور — يُعدّ ويُقرأ من حقيقةٍ قائمة** (البند ٥)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا شرطَ يُكتب في اللوحة** — **والاستعلامان مكتوبان هنا حرفاً،
// والمرجعُ وسيطٌ لا نصٌّ يُلصَق.**
const audienceRoleSQL = `
	SELECT DISTINCT u.id::text FROM users u
	JOIN user_roles ur ON ur.user_id = u.id
	WHERE ur.role_code = $1 AND u.status = 'active'`

// audienceInterestSQL **مشتركو «أخبرني» في هدفٍ بعينه.**
//
// # ولا يُخاطَب من طلب تغطيةً ولم يطلب خبراً (`SI-N-03`)
//
// **و`coverage_request` شكوى موضعٍ لا اشتراكَ خبر** — **والاشتراكُ
// `service_interest` وحدَه**: **ومن خلطهما أرسل إلى من لم يطلب.**
//
// **والمُلغى لا يُخاطَب** — **و`active` هي رايةُ الاشتراك.**
//
// **وصاحبُ الاشتراك قد يكون مجهولاً** (`user_id IS NULL`) — **إشارةُ
// زائرٍ تُعدّ في الخريطة ولا تُخاطَب**: **ولا سبيلَ إلى إخبار من لا
// حسابَ له.**
const audienceInterestSQL = `
	SELECT DISTINCT u.id::text
	FROM coverage_requests cr
	JOIN users u ON u.id = cr.user_id
	WHERE cr.kind = 'service_interest'
	  AND cr.active
	  AND cr.target_key = $1
	  AND u.status = 'active'`

func audienceSQL(kind string) string {
	if kind == AudienceInterest {
		return audienceInterestSQL
	}
	return audienceRoleSQL
}

// Audience **من يقع عليهم الاختيار** — **معرّفاتٌ لا أسماءَ ولا أرقام.**
func (s *Service) Audience(ctx context.Context, kind, ref string) ([]string, error) {
	if !ValidAudience(kind, ref) {
		return nil, ErrBadAudience
	}
	rows, err := s.db.Query(ctx, audienceSQL(kind), strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Count **كم سيصلهم؟** — **يُقرأ قبل الإرسال لا بعده.**
//
// **ولا يُنشئ صفّاً لأحد** — **والمعاينةُ لا تُرسل** (`NT-04`).
func (s *Service) Count(ctx context.Context, kind, ref string) (int, error) {
	ids, err := s.Audience(ctx, kind, ref)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}
