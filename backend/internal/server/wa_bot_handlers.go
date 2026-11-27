package server

// ══════════════════════════════════════════════════════════════════════
// **قائمة بوت الواتساب — أبوابُ اللوحة** (قرارُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /admin/whatsapp-orders/{id}/approve     الأجرةُ والموافقة ⇒ الكباتن  (orders.whatsapp)
//	POST /admin/whatsapp-orders/{id}/reject      رفضُ طلب البوت بسبب          (orders.whatsapp)
//	GET  /admin/captain-applications?status=     طلباتُ التسجيل ككابتن         (drivers.read)
//	GET  /admin/captain-applications/count       عددُ الجديد لشارة القائمة      (drivers.read)
//	POST /admin/captain-applications/{id}/status جديد · تم التواصل · مرفوض       (drivers.manage)

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// markWABot **يَسِم طلباتِ البوت في قائمة الواتساب** — `wa_bot` (ومعلَّقُها ينتظر الموافقة).
func (s *Server) markWABot(ctx context.Context, list []orders.Order, views []map[string]any) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	// **وأمانةُ البوت بتفاصيلها** (قرارُ المالك ٢٠٢٦-١١-٢٧) — الاتّجاهُ والطرفُ الآخرُ والمبلغُ وأوصل موقعُه.
	type botRow struct {
		amanah                        bool
		dir, otherName, otherPhone    string
		amount                        *int64
		currency                      string
		otherLocated, customerLocated bool
		chatPhone                     string
	}
	bot := map[string]botRow{}
	// **والاسمُ الافتراضيُّ لا يُعرض اسماً** («زبون» و«صاحب الأمانة») — الزبونُ لم يكتب اسماً.
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, wa_amanah, amanah_direction,
		       COALESCE(NULLIF(NULLIF(CASE WHEN amanah_direction = 'send' THEN recipient_name ELSE pickup_name END,
		                              $2), $3), ''),
		       COALESCE(CASE WHEN amanah_direction = 'send' THEN recipient_phone ELSE pickup_phone END, ''),
		       collect_amount, amount_currency,
		       CASE WHEN amanah_direction = 'send' THEN dropoff_known ELSE pickup_override IS NOT NULL END,
		       CASE WHEN amanah_direction = 'send' THEN pickup_override IS NOT NULL ELSE dropoff_known END,
		       COALESCE(CASE WHEN amanah_direction = 'send' THEN pickup_phone ELSE recipient_phone END, '')
		  FROM orders WHERE id = ANY($1::uuid[]) AND wa_bot`, ids,
		orders.WhatsAppRecipientName, orders.WAAmanahSenderName)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var b botRow
		if err := rows.Scan(&id, &b.amanah, &b.dir, &b.otherName, &b.otherPhone, &b.amount, &b.currency,
			&b.otherLocated, &b.customerLocated, &b.chatPhone); err != nil {
			return err
		}
		bot[id] = b
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range views {
		b, ok := bot[list[i].ID]
		extra := map[string]any{"wa_bot": ok}
		if ok && b.amanah {
			extra["bot_amanah"] = map[string]any{
				"direction":        b.dir,
				"other_name":       b.otherName,
				"other_phone":      b.otherPhone,
				"amount":           b.amount,
				"currency":         b.currency,
				"other_located":    b.otherLocated,
				"customer_located": b.customerLocated,
				"chat_phone":       b.chatPhone,
			}
		}
		views[i] = withExtra(views[i], extra)
	}
	return nil
}

// waBotChatPhone **رقمُ من راسل البوت** — في «إرسال أمانة» هو المرسِل (`pickup_phone`)، وفي غيرها المستلِم.
func (s *Server) waBotChatPhone(ctx context.Context, o *orders.Order) string {
	var p string
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(CASE WHEN wa_amanah AND amanah_direction = 'send' THEN pickup_phone ELSE recipient_phone END, '')
		  FROM orders WHERE id = $1`, o.ID).Scan(&p); err != nil || strings.TrimSpace(p) == "" {
		return strings.TrimSpace(o.CustomerPhone)
	}
	return strings.TrimSpace(p)
}

// waBotRejectedText **رسالةُ الزبون حين يُرفض طلبُه** — وهو راسلنا للتوّ، فالردُّ مسموح.
const waBotRejectedText = "نعتذر، لم نتمكّن من قبول طلبك رقم #%d. يمكنك التواصل معنا بكتابة 3 بعد إرسال 0."

// handleApproveWABotOrder `POST /admin/whatsapp-orders/{id}/approve` — `{"fee": 0, "store_name": ""}`.
func (s *Server) handleApproveWABotOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Fee       int64  `json:"fee"`
		StoreName string `json:"store_name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.guardText(r.Context(), tf("store_name", &req.StoreName, maxStoreName, textguard.Name)); err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	if !isUUID(orderID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	o, err := s.orders.ApproveWABotOrder(r.Context(), userIDFrom(r), orderID, req.Fee, req.StoreName,
		func(ctx context.Context, q pgx.Tx) error {
			return s.auditTx(ctx, q, r, "ops.whatsapp_bot_order_approve", "order", orderID, map[string]any{
				"fee": req.Fee, "store_name": strings.TrimSpace(req.StoreName),
			})
		})
	if err != nil {
		s.respondErr(w, waReply(err))
		return
	}
	s.touch("orders", "ops")
	// **والزبونُ يُقال له رقمُ طلبه وأجرتُه** (قرارُ المالك ٢٠٢٦-١١-٢٧) — وهو راسلنا للتوّ.
	if phone := s.waBotChatPhone(r.Context(), o); phone != "" {
		num, fee := o.Number, o.DeliveryFee
		go func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 20*time.Second)
			defer cancel()
			s.sendText(ctx, phone, fmt.Sprintf(waBotApprovedText, num, fee))
		}()
	}
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}

// waBotApprovedText **رسالةُ الزبون حين يُوافَق على طلبه** — رقمُه وأجرةُ التوصيل.
const waBotApprovedText = "تمت الموافقة على طلبك رقم #%d. أجرة التوصيل: %d ل.س. نبحث لك الآن عن كابتن."

// waOrderFeeText **رسالةُ الزبون حين يُدخل الموظّفُ طلبَه من اللوحة** — رقمُه وأجرةُ التوصيل.
const waOrderFeeText = "تم تسجيل طلبك رقم #%d. أجرة التوصيل: %d ل.س."

// handleRejectWABotOrder `POST /admin/whatsapp-orders/{id}/reject` — `{"reason": "…"}`.
func (s *Server) handleRejectWABotOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.guardText(r.Context(), tf("reason", &req.Reason, maxOrderNotes, textguard.Notes)); err != nil {
		s.respondErr(w, err)
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		s.respondErr(w, errValidation)
		return
	}
	orderID := chi.URLParam(r, "id")
	if !isUUID(orderID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	o, err := s.orders.RejectWABotOrder(r.Context(), userIDFrom(r), orderID, req.Reason)
	if err != nil {
		s.respondErr(w, waReply(err))
		return
	}
	s.audit(r, "ops.whatsapp_bot_order_reject", "order", orderID, map[string]any{"reason": strings.TrimSpace(req.Reason)})
	s.touch("orders", "ops")
	// **والزبونُ يُقال له** — سطرٌ واحد، وإن لم يصل فالموظّفُ يراه مرفوضاً.
	if phone := s.waBotChatPhone(r.Context(), o); phone != "" {
		num := o.Number
		go func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 20*time.Second)
			defer cancel()
			s.sendText(ctx, phone, fmt.Sprintf(waBotRejectedText, num))
		}()
	}
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}

// captainApplication **طلبُ تسجيلٍ ككابتن كما تعرضه اللوحة.**
type captainApplication struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	FullName     string    `json:"full_name"`
	Age          string    `json:"age"`
	Area         string    `json:"area"`
	HasEbike     bool      `json:"has_ebike"`
	HasMotorbike bool      `json:"has_motorbike"`
	HasLicense   bool      `json:"has_license"`
	HasAndroid   bool      `json:"has_android"`
	Experience   string    `json:"experience"`
	Hours        string    `json:"hours"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const captainAppCols = `id::text, phone, full_name, age, area, has_ebike, has_motorbike, has_license,
	has_android, experience, hours, status, created_at, updated_at`

func scanCaptainApp(row pgx.Row) (captainApplication, error) {
	var a captainApplication
	err := row.Scan(&a.ID, &a.Phone, &a.FullName, &a.Age, &a.Area, &a.HasEbike, &a.HasMotorbike,
		&a.HasLicense, &a.HasAndroid, &a.Experience, &a.Hours, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

var captainAppStatuses = map[string]bool{"new": true, "contacted": true, "rejected": true}

// handleListCaptainApplications `GET /admin/captain-applications?status=` — آخرُ مئتين، الأحدثُ أوّلاً.
func (s *Server) handleListCaptainApplications(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !captainAppStatuses[status] {
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(), `SELECT `+captainAppCols+` FROM captain_applications
		WHERE ($1 = '' OR status = $1) ORDER BY created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []captainApplication{}
	for rows.Next() {
		a, err := scanCaptainApp(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"applications": out})
}

// handleCountCaptainApplications `GET /admin/captain-applications/count` — `{"new": n}`.
func (s *Server) handleCountCaptainApplications(w http.ResponseWriter, r *http.Request) {
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM captain_applications WHERE status = 'new'`).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"new": n})
}

// handleCaptainApplicationStatus `POST /admin/captain-applications/{id}/status` — `{"status": "contacted"}`.
func (s *Server) handleCaptainApplicationStatus(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Status string `json:"status"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !captainAppStatuses[req.Status] {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	a, err := scanCaptainApp(s.pg.QueryRow(r.Context(), `
		UPDATE captain_applications SET status = $2, updated_by = NULLIF($3, '')::uuid, updated_at = now()
		 WHERE id = $1 RETURNING `+captainAppCols, id, req.Status, userIDFrom(r)))
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.captain_application_status", "captain_application", id, map[string]any{"status": req.Status})
	s.touch("captain_application", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"application": a})
}
