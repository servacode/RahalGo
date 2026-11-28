package server

// ══════════════════════════════════════════════════════════════════════
// **الطرفُ الآخر يشارك موقعَه المباشر من تطبيقنا** (قرارُ المالك ٢٠٢٦-١١-٢٨)
// ══════════════════════════════════════════════════════════════════════
//
// الأمانةُ وتوصيلةُ المتجر بتطلب موقعَ الطرف الآخر برسالة واتساب (`recipient_location.go`). **وإذا رقمُه
// عليه حسابُ زبون** كمان بيوصله إشعارٌ بالتطبيق:
//
//	أمانة    «لديك أمانة من صاحب الرقم ٠٩… — هل تريد استقبال الأمانة؟»
//	توصيلة   «لديك توصيلة من <اسم المتجر> — هل تريد استقبالها؟»
//
// وبالتطبيق والموقع بطاقةٌ فيها زرّين «نعم/لا». **«نعم» موافقةٌ تبدأ بعدها المشاركةُ لحالها**، و«لا»
// تنكتب بسجلّ الطلب ويتبلّغ المكتب. **ورسالةُ الواتساب كما هي** — احتياط.
//
//	GET    /api/v1/my/party-orders                     طلباتٌ أنا فيها الطرفُ الآخر (مفتوحة)
//	GET    /api/v1/my/party-orders/{id}                بطاقةُ طلبٍ وحدَه
//	POST   /api/v1/my/party-orders/{id}/answer         {answer: yes|no}
//	POST   /api/v1/my/party-orders/{id}/live-location  {lat, lng, accuracy_m}
//	DELETE /api/v1/my/party-orders/{id}/live-location  «إيقاف»
//
// # من هو الطرف
//
// **رقمُ حسابه (المُثبَت بكود الدخول) = رقمُ طلب الموقع**، والطلبُ مربوطٌ بطلبٍ وما انتهى عمرُه.
// **وطلبُ غيره «غيرُ موجود» (٤٠٤)** لا «ممنوع» — كباب الموقع المباشر.
//
// # وما تعرضه البطاقة — لا غير (`qa.PartyViewPrivacy`)
//
// رقمُ المرسل (بالأمانة — قرارُ المالك) أو اسمُ المتجر، وحالُ الطلب، **و«كابتن رحّال غو» بلا اسمه**
// (`captain_assigned`)، وكودُ التسليم إذا هو اللي بيعطيه للكابتن. **ولا مبالغ أبداً.**
//
// # الإشعارُ مرّة والتذكيرُ مرّة
//
// **الحجزُ قبل الإرسال** (`party_push_at` ثمّ `party_reminder_at`) — جولتان معاً ما بتبعتو مرّتين.
// والتذكيرُ بعد خمس دقائق بلا جواب ولا موقع، من جولة متابعة الموقع (`wa_location_followup.go`).

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// partyReminderAfter **بعد كم من الإشعار الأوّل يُذكَّر** — خمس دقائق (قرارُ المالك).
const partyReminderAfter = 5 * time.Minute

// partyPickupEvery **أقلُّ فاصلٍ بين كتابتين لنقطة الاستلام** — كلُّ كتابةٍ بثٌّ للكابتن.
const partyPickupEvery = 20 * time.Second

// partyBatch **حدُّ الجولة الواحدة.**
const partyBatch = 50

// partyEntity **وجهةُ الإشعار** — التطبيقُ يفتح بطاقةَ الطرف بمعرّف الطلب.
const partyEntity = "party_order"

// errPartyNotConsented **ما قال «نعم» بعد** — والمشاركةُ لا تبدأ بلا موافقته.
var errPartyNotConsented = httpx.NewError(http.StatusConflict,
	"party_not_consented", "errors.live_location_inactive")

// partyKindSQL **نوعُ الطلب كما يقرؤه الطرف** — `amanah` · `delivery` (توصيلةُ متجر) · `order` (غيرُها:
// طلبُ الواتساب من المكتب وأيُّ نوعٍ يطلب موقعاً بعدين). **والقاعدةُ الرقمُ وحدَه** (قرارُ المالك
// ٢٠٢٦-١١-٢٨): كلُّ طلبِ موقعٍ مربوطٍ بطلب، **أيّاً كان نوعُه**، يمرّ من هون.
const partyKindSQL = `CASE
	  WHEN (o.kind = 'custom' AND o.custom_mode = 'amanah') OR o.wa_amanah THEN 'amanah'
	  WHEN o.kind = 'merchant_delivery' AND o.merchant_id IS NOT NULL THEN 'delivery'
	  ELSE 'order' END`

// partyRoleSQL **دورُ الطرف** — `orders.PartyDropoff` · `PartyPickup` · `PartyFetch`.
const partyRoleSQL = `CASE
	  WHEN l.target = 'pickup' THEN 'pickup'
	  WHEN o.wa_amanah THEN 'dropoff'
	  WHEN o.kind = 'custom' AND o.custom_mode = 'amanah' THEN
	       CASE WHEN o.amanah_direction = 'send' THEN 'dropoff' ELSE 'pickup' END
	  WHEN o.kind = 'merchant_delivery' AND o.delivery_mode = 'fetch' THEN 'fetch'
	  ELSE 'dropoff' END`

// partyOpenSQL **أما زال دورُه قائماً؟** — طرفُ التسليم لحدّ التسليم، وطرفُ الاستلام لحدّ الاستلام.
const partyOpenSQL = `(o.closed_at IS NULL
	  AND o.status NOT IN ('delivered', 'cancelled', 'failed', 'rejected', 'refunded')
	  AND CASE WHEN r.role = 'dropoff' THEN o.delivered_at IS NULL ELSE o.picked_up_at IS NULL END)`

// partySenderPhoneSQL **رقمُ المرسل بالأمانة** (قرارُ المالك ٢٠٢٦-١١-٢٨: «لديك أمانة من صاحب الرقم») —
// صاحبُ طلب التطبيق، أو رقمُ زبون البوت (بـ«أبعت» هو نقطةُ الاستلام، وبـ«أستقبل» هو المستلم).
const partySenderPhoneSQL = `CASE
	  WHEN o.wa_amanah THEN CASE WHEN o.amanah_direction = 'send' THEN COALESCE(o.pickup_phone, '')
	                             ELSE COALESCE(o.recipient_phone, '') END
	  WHEN o.kind = 'custom' THEN COALESCE(cu.phone::text, '')
	  ELSE '' END`

// partyOrderRow **طلبٌ أنا فيه الطرفُ الآخر** — كما يُقرأ من القاعدة.
type partyOrderRow struct {
	reqID, orderID, role, answer, status, kind string
	senderPhone, storeName, code               string
	captain, received, open                    bool
	number                                     int64
}

// partyView **بطاقةُ الطرف** — كلُّ ما يخرج له، ولا غيره (`qa.PartyViewPrivacy`).
type partyView struct {
	OrderID string `json:"order_id"`
	// Kind **نوعُ الطلب** — `amanah` · `delivery` (متجر) · `order` (غيرُهما: «لديك طلب من رحّال غو»).
	Kind string `json:"kind"`
	// Role **وين هو بالطريق** — `dropoff` · `pickup` · `fetch`.
	Role string `json:"role"`
	// SenderPhone **رقمُ المرسل بالأمانة** — فارغٌ بالتوصيلة.
	SenderPhone string `json:"sender_phone"`
	// StoreName **اسمُ المتجر بالتوصيلة** — فارغٌ بالأمانة.
	StoreName string `json:"store_name"`
	Status    string `json:"status"`
	// CaptainAssigned **صار في كابتن** — والشاشةُ تكتب «كابتن رحّال غو»، **بلا اسمه** (قرارُ المالك).
	CaptainAssigned bool `json:"captain_assigned"`
	// Code **كودُ التسليم** — لطرف التسليم وحدَه، هو اللي بيعطيه للكابتن.
	Code string `json:"code"`
	// Answer **جوابُه** — فارغٌ لسّا ما جاوب، أو `yes` · `no`.
	Answer string `json:"answer"`
	// Shared **وصل موقعُه** (من التطبيق أو بالواتساب).
	Shared bool `json:"shared"`
	// Open **دورُه قائم** — والمشاركةُ تقف حين يصير `false`.
	Open bool `json:"open"`
}

func (p partyOrderRow) view() partyView {
	v := partyView{
		OrderID: p.orderID, Kind: p.kind, Role: p.role, Status: p.status,
		CaptainAssigned: p.captain, Answer: p.answer, Shared: p.received, Open: p.open,
	}
	switch p.kind {
	case "amanah":
		v.SenderPhone = p.senderPhone
	case "delivery":
		v.StoreName = p.storeName
	}
	if p.role == orders.PartyDropoff && p.open {
		v.Code = p.code
	}
	return v
}

// partyQuery **طلباتُ رقمٍ بعينه** — أحدثُ طلب موقعٍ لكلّ طلب. `orderID` الفارغ: كلُّها.
const partyQuery = `
	SELECT DISTINCT ON (o.id)
	       l.id::text, o.id::text, r.role, COALESCE(l.party_answer, ''), o.status,
	       ` + partySenderPhoneSQL + `, COALESCE(m.name, ''), COALESCE(o.delivery_code, ''),
	       ` + partyKindSQL + `, o.driver_id IS NOT NULL, l.received_at IS NOT NULL,
	       ` + partyOpenSQL + `, o.number
	  FROM recipient_location_requests l
	  JOIN orders o ON o.id = l.order_id
	  LEFT JOIN users cu ON cu.id = o.customer_id
	  LEFT JOIN merchants m ON m.id = o.merchant_id
	  CROSS JOIN LATERAL (SELECT ` + partyRoleSQL + ` AS role) r
	 WHERE l.phone = $1 AND l.expires_at > now()
	   AND o.customer_id IS DISTINCT FROM $3::uuid
	   AND ($2 = '' OR o.id::text = $2)
	 ORDER BY o.id, l.created_at DESC`

// partyFind **طلباتي كطرفٍ آخر** — برقم حسابي. `orderID` الفارغ: كلُّها.
func (s *Server) partyFind(ctx context.Context, uid, orderID string) ([]partyOrderRow, error) {
	var phone string
	if err := s.pg.QueryRow(ctx,
		`SELECT phone::text FROM users WHERE id = $1 AND status = 'active'`, uid).Scan(&phone); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rows, err := s.pg.Query(ctx, partyQuery, phone, orderID, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []partyOrderRow
	for rows.Next() {
		var p partyOrderRow
		if err := rows.Scan(&p.reqID, &p.orderID, &p.role, &p.answer, &p.status,
			&p.senderPhone, &p.storeName, &p.code, &p.kind, &p.captain, &p.received,
			&p.open, &p.number); err != nil {
			return nil, err
		}
		p.senderPhone = partyLocalPhone(p.senderPhone)
		out = append(out, p)
	}
	return out, rows.Err()
}

// partyOne **طلبٌ وحدَه** — وطلبُ غيري أو معرّفٌ مجهول ٤٠٤.
func (s *Server) partyOne(ctx context.Context, uid, orderID string) (*partyOrderRow, error) {
	if !isUUID(orderID) {
		return nil, httpx.ErrNotFound
	}
	list, err := s.partyFind(ctx, uid, orderID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, httpx.ErrNotFound
	}
	return &list[0], nil
}

// partyLocalPhone **الرقمُ كما يكتبه الناس** — `+9639…` ← `09…`؛ وغيرُ السوريّ كما هو.
func partyLocalPhone(p string) string {
	if strings.HasPrefix(p, "+963") && len(p) > 4 {
		return "0" + p[4:]
	}
	return p
}

// handlePartyOrders — `GET /my/party-orders` — المفتوحةُ وحدَها.
func (s *Server) handlePartyOrders(w http.ResponseWriter, r *http.Request) {
	list, err := s.partyFind(r.Context(), userIDFrom(r), "")
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out := make([]partyView, 0, len(list))
	for _, p := range list {
		if p.open {
			out = append(out, p.view())
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out})
}

// handlePartyOrder — `GET /my/party-orders/{id}`.
func (s *Server) handlePartyOrder(w http.ResponseWriter, r *http.Request) {
	p, err := s.partyOne(r.Context(), userIDFrom(r), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p.view())
}

// partyRefusedNote **سطرُ «لا» بسجلّ الطلب** — للمكتب (سطرٌ حالُه قبلُ وبعدُ واحدة، فالزبونُ لا يراه).
func partyRefusedNote(kind string) string {
	if kind == "amanah" {
		return "الطرف الآخر ضغط «لا» على «هل تريد استقبال الأمانة؟» بالتطبيق — طلب الموقع بالواتساب باقٍ"
	}
	return "المستلم ضغط «لا» على «هل تريد استقباله؟» بالتطبيق — طلب الموقع بالواتساب باقٍ"
}

// handlePartyAnswer — `POST /my/party-orders/{id}/answer` {answer: yes|no}.
//
// **«لا» مرّةً تنكتب بالسجلّ ويتبلّغ المكتب**، وتكرارُها لا يكرّر. **و«نعم» بعد «لا» مقبولة** — غيّر رأيه.
func (s *Server) handlePartyAnswer(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Answer string `json:"answer"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Answer != "yes" && req.Answer != "no" {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	uid := userIDFrom(r)
	p, err := s.partyOne(ctx, uid, chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !p.open {
		s.respondErr(w, orders.ErrLiveInactive)
		return
	}
	var prev *string
	if err := s.pg.QueryRow(ctx, `
		WITH old AS (SELECT party_answer FROM recipient_location_requests WHERE id = $1 FOR UPDATE)
		UPDATE recipient_location_requests l
		   SET party_answer = $2, party_answered_at = now()
		  FROM old WHERE l.id = $1
		RETURNING old.party_answer`, p.reqID, req.Answer).Scan(&prev); err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Answer == "no" && (prev == nil || *prev != "no") {
		if _, err := s.pg.Exec(ctx, `
			INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
			VALUES ($1, $2, $2, $3, $4)`, p.orderID, p.status, uid, partyRefusedNote(p.kind)); err != nil {
			s.logger.Warn("الطرف الآخر: سطرُ الرفض بالسجلّ سقط", "order", p.orderID, "error", err)
		}
		if s.notify != nil {
			title := "المستلم رفض استقبال الطلب من التطبيق"
			if p.kind == "amanah" {
				title = "الطرف الآخر رفض استقبال الأمانة من التطبيق"
			}
			s.notify.NotifyOps(ctx, notifications.Input{
				Kind: notifications.KindOrder, Title: title,
				Body:   "#" + strconv.FormatInt(p.number, 10) + " — طلب الموقع بالواتساب باقٍ",
				Entity: "order", EntityID: p.orderID, Href: "/dashboard/orders",
			})
		}
		s.touch("orders", "ops")
	}
	p.answer = req.Answer
	httpx.JSON(w, http.StatusOK, p.view())
}

// handlePartyLiveLocation — `POST /my/party-orders/{id}/live-location` {lat, lng, accuracy_m}.
//
// **أوّلُ نقطةٍ تملأ طلبَ الموقع كما يملؤه ردُّ الواتساب** (`locReqReceived`: تُكتب في الطلب، وينزل
// للكباتن إن كان ينتظرها، ويُخبَر الكابتن). **وما بعدها** حسب دوره (`orders/party_live.go`).
func (s *Server) handlePartyLiveLocation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	uid := userIDFrom(r)
	req, err := decode[struct {
		Lat       float64  `json:"lat"`
		Lng       float64  `json:"lng"`
		AccuracyM *float64 `json:"accuracy_m"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !orders.ValidPoint(req.Lat, req.Lng) {
		s.respondErr(w, errValidation)
		return
	}
	acc := locReqAcc(req.AccuracyM)
	ctx := r.Context()
	// **وحدُّ المعدّل كحدّ صاحب الطلب** (`liveAllowed`) — بمفتاحٍ منفصل.
	if !s.liveAllowed(ctx, uid, "party:"+orderID) {
		s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
		return
	}
	p, err := s.partyOne(ctx, uid, orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !p.open {
		s.respondErr(w, orders.ErrLiveInactive)
		return
	}
	if p.answer != "yes" {
		s.respondErr(w, errPartyNotConsented)
		return
	}
	ok := func() {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"saved":          true,
			"max_accuracy_m": s.liveSetting(ctx, orders.SettingLiveMaxAccuracyM),
		})
	}
	if !p.received {
		var ownerID, customerID, oid, mode string
		err := s.pg.QueryRow(ctx, `
			UPDATE recipient_location_requests l
			   SET lat = $2, lng = $3, accuracy_m = $4, received_at = now(), party_live_at = now()
			 WHERE l.id = $1 AND l.received_at IS NULL
			RETURNING COALESCE((SELECT m.owner_user_id::text FROM merchants m WHERE m.id = l.merchant_id), ''),
			          COALESCE(l.customer_id::text, ''), COALESCE(l.order_id::text, ''), `+locReqModeSQL,
			p.reqID, req.Lat, req.Lng, acc).Scan(&ownerID, &customerID, &oid, &mode)
		if err == nil {
			s.locReqReceived(ctx, ownerID, customerID, oid, mode, req.Lat, req.Lng)
			if p.role == orders.PartyDropoff {
				s.partySaveDropoffLive(ctx, p.orderID, uid, req.Lat, req.Lng, acc)
			}
			ok()
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			s.respondErr(w, err)
			return
		}
		// **سبقه ردُّ الواتساب بنفس اللحظة** — يكمل كنقطةٍ لاحقة.
	}
	switch p.role {
	case orders.PartyDropoff:
		if err := s.partySaveDropoffLive(ctx, p.orderID, uid, req.Lat, req.Lng, acc); err != nil {
			s.respondErr(w, err)
			return
		}
	default:
		// **نقطةُ الاستلام بتنكتب كلَّ عشرين ثانية بالأكثر** — كلُّ كتابةٍ بثٌّ للكابتن.
		tag, err := s.pg.Exec(ctx, `
			UPDATE recipient_location_requests SET party_live_at = now()
			 WHERE id = $1 AND (party_live_at IS NULL OR party_live_at < now() - make_interval(secs => $2))`,
			p.reqID, int(partyPickupEvery.Seconds()))
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if tag.RowsAffected() == 1 {
			moved, err := s.orders.MovePartyPickup(ctx, p.orderID, p.role, req.Lat, req.Lng)
			if err != nil {
				s.respondErr(w, err)
				return
			}
			if moved {
				s.touch("orders", "ops")
			}
		}
	}
	ok()
}

// partySaveDropoffLive **موقعٌ مباشرٌ لطرف التسليم** — بالجدول اللي يقرؤه الكابتن، ونبضةٌ لغرفته وللمكتب.
func (s *Server) partySaveDropoffLive(ctx context.Context, orderID, uid string, lat, lng float64, acc *float64) error {
	driverID, err := s.orders.SavePartyLive(ctx, orderID, uid, lat, lng, acc)
	if err != nil {
		return err
	}
	s.notifyCustomerLive(driverID, orderID)
	s.publishCustomerLiveToOps(orderID, &lat, &lng, acc)
	return nil
}

// handleStopPartyLiveLocation — `DELETE /my/party-orders/{id}/live-location` — «إيقاف».
func (s *Server) handleStopPartyLiveLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := userIDFrom(r)
	p, err := s.partyOne(ctx, uid, chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	driverID, err := s.orders.StopPartyLive(ctx, p.orderID, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notifyCustomerLive(driverID, p.orderID)
	httpx.JSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// ── الإشعارُ والتذكير ─────────────────────────────────────────────────

// partyInviteTitle **عنوانُ الإشعار** — نصُّ المالك.
func partyInviteTitle(kind, senderPhone, store string) string {
	switch {
	case kind == "amanah" && senderPhone != "":
		return "لديك أمانة من صاحب الرقم " + senderPhone + " — هل تريد استقبال الأمانة؟"
	case kind == "amanah":
		return "لديك أمانة من رحّال غو — هل تريد استقبال الأمانة؟"
	case kind == "delivery" && store != "":
		return "لديك توصيلة من " + store + " — هل تريد استقبالها؟"
	}
	return "لديك طلب من رحّال غو — هل تريد استقباله؟"
}

// partyReminderBody **سطرُ التذكير تحت العنوان.**
const partyReminderBody = "تذكير: ما وصلنا ردّك لسّا — اضغط لتردّ"

// partyInviteBody **سطرُ الإشعار الأوّل.**
const partyInviteBody = "اضغط لتردّ — بموافقتك يوصل إليك كابتن رحّال غو بسهولة"

// partyCandidatesSQL **من يُبلَّغ** — طلبُ موقعٍ مربوطٌ بطلبٍ مفتوح، ما وصل موقعُه ولا جاوب، **ورقمُه
// عليه حسابُ زبونٍ فعّال** غيرُ صاحب الطلب. `$1` الفارغ: الكلّ، وإلّا طلبٌ أو طلبُ موقعٍ بعينه.
const partyCandidatesSQL = `
	SELECT l.id::text, u.id::text, o.id::text, ` + partyKindSQL + `,
	       ` + partySenderPhoneSQL + `, COALESCE(m.name, '')
	  FROM recipient_location_requests l
	  JOIN orders o ON o.id = l.order_id
	  JOIN users u ON u.phone = l.phone AND u.status = 'active'
	  LEFT JOIN users cu ON cu.id = o.customer_id
	  LEFT JOIN merchants m ON m.id = o.merchant_id
	  CROSS JOIN LATERAL (SELECT ` + partyRoleSQL + ` AS role) r
	 WHERE l.expires_at > now() AND l.received_at IS NULL AND l.party_answer IS NULL
	   AND ` + partyOpenSQL + `
	   AND u.id IS DISTINCT FROM o.customer_id
	   AND EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code = 'customer')
	   AND ($1 = '' OR o.id::text = $1 OR l.id::text = $1)`

// partyNotify **يحجز ثمّ يرسل** — `column` هو `party_push_at` أو `party_reminder_at`.
func (s *Server) partyNotify(ctx context.Context, query string, args []any, column, body string) int {
	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		s.logger.Warn("الطرف الآخر: قراءةُ المرشَّحين سقطت", "error", err)
		return 0
	}
	type cand struct {
		req, user, order, kind string
		senderPhone, store     string
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.req, &c.user, &c.order, &c.kind, &c.senderPhone, &c.store); err != nil {
			rows.Close()
			s.logger.Warn("الطرف الآخر: قراءةُ صفٍّ سقطت", "error", err)
			return 0
		}
		cands = append(cands, c)
	}
	rows.Close()
	sent := 0
	for _, c := range cands {
		if ctx.Err() != nil {
			break
		}
		tag, err := s.pg.Exec(ctx, `UPDATE recipient_location_requests SET `+column+` = now()
			 WHERE id = $1 AND `+column+` IS NULL`, c.req)
		if err != nil || tag.RowsAffected() == 0 {
			continue
		}
		if s.notify != nil {
			s.notify.Notify(ctx, notifications.Input{
				UserID: c.user, Kind: notifications.KindOrder,
				Title:  partyInviteTitle(c.kind, partyLocalPhone(c.senderPhone), c.store),
				Body:   body,
				Entity: partyEntity, EntityID: c.order, Href: "/shop?party=" + c.order,
				Apps: []string{notifications.AppCustomer},
			})
		}
		s.touchUser(c.user, "party_orders")
		sent++
	}
	return sent
}

// partyInvite **الإشعارُ الأوّل** — لطلبٍ أو لطلب موقعٍ بعينه (`key`)، أو للكلّ إن فرغ. ويردّ كم أُرسل.
//
// **يُنادى بعد ربط طلب الموقع بالطلب** (إنشاءُ الأمانة والتوصيلة وطلبُ البوت) **وكلَّ جولة** — فما فات
// لحظتَه يلحقه بعد نصف دقيقة.
func (s *Server) partyInvite(ctx context.Context, key string) int {
	return s.partyNotify(ctx, partyCandidatesSQL+` AND l.party_push_at IS NULL
	 ORDER BY l.created_at LIMIT `+itoa(partyBatch), []any{key}, "party_push_at", partyInviteBody)
}

// partyRemind **التذكيرُ الوحيد** — بعد خمس دقائق من الإشعار بلا جوابٍ ولا موقع.
func (s *Server) partyRemind(ctx context.Context) int {
	return s.partyNotify(ctx, partyCandidatesSQL+` AND l.party_push_at IS NOT NULL
	   AND l.party_push_at <= now() - make_interval(secs => $2) AND l.party_reminder_at IS NULL
	 ORDER BY l.party_push_at LIMIT `+itoa(partyBatch),
		[]any{"", int(partyReminderAfter.Seconds())}, "party_reminder_at", partyReminderBody)
}

// SweepPartyInvites **جولةٌ واحدة** — الإشعارُ الفائت ثمّ التذكير. **يُنادى من الفحص مباشرةً.**
func (s *Server) SweepPartyInvites(ctx context.Context) (invited, reminded int) {
	return s.partyInvite(ctx, ""), s.partyRemind(ctx)
}
