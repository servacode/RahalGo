// Package notifications مركز الإشعارات الموحّد للمنصة.
//
// قاعدة مركزية: أي حدث مهم يمرّ من هنا فقط — يُحفظ في صندوق وارد المستخدم ثم
// يُبثّ حياً. الحفظ أولاً كي لا يضيع الإشعار على من كان غير متصل، والبث ليصل
// فوراً لمن هو متصل بلا أي تحديث للصفحة.
package notifications

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher واجهة البث الحي (نفس واجهة محرك الطلبات).
type Publisher interface {
	Publish(topic string, event any)
}

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
}

func New(db *pgxpool.Pool, hub Publisher, logger *slog.Logger) *Service {
	return &Service{db: db, hub: hub, logger: logger}
}

// Notification إشعار واحد كما يُعاد للواجهة.
type Notification struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Href      string `json:"href"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
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
}

// Notify يحفظ الإشعار ويبثّه لصاحبه فوراً. لا يُفشل العملية الأصلية أبداً —
// فشل الإشعار يُسجَّل ولا يُرجع خطأً للمستدعي.
func (s *Service) Notify(ctx context.Context, in Input) {
	if in.UserID == "" || in.Title == "" {
		return
	}
	var id, createdAt string
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
	s.hub.Publish("user:"+in.UserID, map[string]any{
		"type": "notification",
		"notification": Notification{
			ID: id, Kind: in.Kind, Title: in.Title, Body: in.Body,
			Entity: in.Entity, EntityID: in.EntityID, Href: in.Href,
			Read: false, CreatedAt: createdAt,
		},
	})
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

// NotifyOps يبلّغ مكتب المنصة كاملاً (مالك المنصة + العمليات) بلا تكرار.
func (s *Service) NotifyOps(ctx context.Context, in Input) {
	s.NotifyRoles(ctx, OpsDesk, in)
}

// NotifyRoles يرسل الإشعار لحاملي أي من الأدوار المذكورة — مرة واحدة لكل شخص
// مهما تعددت أدواره.
func (s *Service) NotifyRoles(ctx context.Context, roles []string, in Input) {
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
func (s *Service) List(ctx context.Context, userID string, limit int) ([]Notification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, kind, title, body, entity, entity_id, href,
		       (read_at IS NOT NULL), created_at::text
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Entity,
			&n.EntityID, &n.Href, &n.Read, &n.CreatedAt); err != nil {
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
