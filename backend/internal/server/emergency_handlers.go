package server

// **الطارئ** — حين يقع للسائق ما لا يحتمل شاشة.
//
// # المسألة
//
// قرارُ المالك: «ربما ياخذ السائق الطلب ثم يتحرك فجاءة يصبح معه حادث… زر اسمه
// طارئ».
//
// **ولم يكن له بابٌ ألبتّة.** من وقع له حادثٌ وهو حاملٌ الطعام لا يملك في
// شاشته إلّا «تعذّر التسليم» — **فيُقفل طلبٌ كان يمكن أن يصل**، ويُحسب ذنبٌ لم
// يقع، **ويبقى هو مشغولاً بشاشةٍ وهو في حالٍ لا تحتمل الشاشات.**
//
// # وثلاثةُ أشياءَ تقع بضغطةٍ واحدة
//
//	الموقعُ يُلتقط    →  البضاعةُ حيث وقف لا حيث بدأت
//	العملياتُ تُنبَّه  →  فوراً، لا حين تفتح اللوحة
//	الطلبُ يُحرَّر     →  إلى الطابور، ومنه يستأنفه غيرُه
//
// **وواحدةٌ لا ثلاث**: من كُسرت يدُه لا يملأ ثلاث شاشات. **وكلُّ حقلٍ نطلبه
// منه في تلك اللحظة حقلٌ لن يُملأ** — فيُترك الزرُّ ولا يُضغط، ويبقى الطلبُ
// معه ولا نعلم.
//
// # ولماذا يُحرَّر بدور «العمليات» لا بدوره
//
// التحريرُ بعد الاستلام **قرارُ منصةٍ لا قرارُ سائق** — وإلّا لَترك كلُّ من
// ثقل عليه طلبٌ طلبَه بحجّة الطارئ. **والمنصةُ هي التي تحرّره عنه، لا هو.**

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleDriverEmergency طارئٌ على طلبٍ بيده.
func (s *Server) handleDriverEmergency(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	req, err := decode[struct {
		Lat  *float64 `json:"lat"`
		Lng  *float64 `json:"lng"`
		Note string   `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	driverID := userIDFrom(r)
	ctx := r.Context()

	var status string
	var number int64
	if err := s.pg.QueryRow(ctx,
		`SELECT status, number FROM orders WHERE id = $1`, orderID).
		Scan(&status, &number); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}

	// **الموقعُ اختياريٌّ ولا يُوقف الطارئ.**
	//
	// جهازٌ مرفوضُ الإذن، أو داخلَ بناءٍ لا إشارةَ فيه — **وطارئٌ يُردّ لأن
	// الموقعَ لم يُقرأ طارئٌ ضاع.** والعملياتُ تتّصل به فتعرف أين هو.
	hasPoint := req.Lat != nil && req.Lng != nil
	var emergencyID string
	if hasPoint {
		err = s.pg.QueryRow(ctx, `
			INSERT INTO driver_emergencies (driver_id, order_id, at, note)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5)
			RETURNING id`, driverID, orderID, *req.Lng, *req.Lat, clip(req.Note, 500)).
			Scan(&emergencyID)
	} else {
		err = s.pg.QueryRow(ctx, `
			INSERT INTO driver_emergencies (driver_id, order_id, note)
			VALUES ($1, $2, $3) RETURNING id`,
			driverID, orderID, clip(req.Note, 500)).Scan(&emergencyID)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **نقطةُ الاستلام البديلة — إن كانت البضاعةُ قد خرجت.**
	//
	// قبل الاستلام الطعامُ في المتجر، **فالبديلُ يذهب إليه كما كان.** وبعده
	// الطعامُ مع المصاب، **فمن ذهب إلى المطعم استلم طلباً ثانياً من مطبخٍ
	// حضّر واحداً** — فتُدفع البضاعةُ مرّتين.
	afterPickup := status == orders.StPickedUp || status == orders.StOnTheWay ||
		status == orders.StAtDropoff
	if afterPickup {
		if hasPoint {
			if _, err := s.pg.Exec(ctx, `
				UPDATE orders
				SET pickup_override = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
				    pickup_override_note = $4
				WHERE id = $1`,
				orderID, *req.Lng, *req.Lat, "استلامٌ من موضع طارئ — البضاعةُ مع السائق"); err != nil {
				s.respondErr(w, err)
				return
			}
		} else if _, err := s.pg.Exec(ctx, `
			UPDATE orders SET pickup_override_note = $2 WHERE id = $1`,
			orderID,
			"البضاعةُ مع سائقٍ سابقٍ وقع له طارئ — لا تذهب إلى المتجر، اتّصل بالعمليات",
		); err != nil {
			// **وكلمةٌ بلا نقطةٍ خيرٌ من نقطةٍ لم تُلتقط.**
			//
			// كان الشرطُ `afterPickup && hasPoint` — **فإن رُفض إذنُ الموقع أو
			// كان السائقُ داخل بناءٍ لم يُكتب شيءٌ إطلاقاً**، ويذهب الثاني إلى
			// مطعمٍ سلّم بضاعتَه وقبض ثمنَها. **فتُدفع البضاعةُ مرّتين.**
			//
			// والطارئُ نفسُه لا يُردّ لغياب الموقع — وهو صواب: «طارئٌ يُردّ لأن
			// الموقعَ لم يُقرأ طارئٌ ضاع». **لكنّ البضاعةَ حينها كانت بلا عنوان
			// ولا كلمة.** والآن لها كلمةٌ يقرؤها من يلتقطه.
			s.respondErr(w, err)
			return
		}
	}

	// **والتحريرُ بدور العمليات لا بدوره** — المنصةُ تحرّره عنه.
	//
	// ويُنفَّذ بعد تسجيل الطارئ: **تعثّرُ التحرير يجب ألّا يبتلع النداء.**
	// فلو سقط الانتقالُ لسببٍ ما بقي الإنذارُ مسجّلاً والعملياتُ تُخبَر،
	// **وإنسانٌ يتصرّف خيرٌ من صمتٍ لأن آلةً تعثّرت.**
	released := true
	if _, err := s.orders.Transition(ctx, driverID, []string{"ops"},
		orderID, orders.StDispatching, "طارئٌ لدى السائق"); err != nil {
		released = false
		s.logger.Error("الطارئ: تعذّر تحرير الطلب", "order", orderID, "error", err)
	}

	// **ودوامُه يُغلق.**
	//
	// من وقع له حادثٌ لا يُعرض عليه طلبٌ تالٍ بعد دقيقة. **وترتيبُ الطابور
	// يقرأ `on_shift`** — فمن بقي عليه ظلّ في الدور وهو في المستشفى.
	if _, err := s.pg.Exec(ctx,
		`UPDATE users SET on_shift = false WHERE id = $1`, driverID); err != nil {
		s.logger.Error("الطارئ: تعذّر إغلاق الدوام", "driver", driverID, "error", err)
	}

	// **والعملياتُ تُنبَّه فوراً** — لا حين تفتح اللوحة.
	body := "#" + strconv.FormatInt(number, 10)
	if req.Note != "" {
		body += " — " + clip(req.Note, 160)
	}
	if hasPoint {
		body += " · " + strconv.FormatFloat(*req.Lat, 'f', 5, 64) +
			"," + strconv.FormatFloat(*req.Lng, 'f', 5, 64)
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind: notifications.KindOrder, Title: notifTitles.driverEmergency, Body: body,
		Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
	})
	s.audit(r, "driver.emergency", "order", orderID, map[string]any{
		"emergency_id": emergencyID, "released": released, "status_was": status,
	})
	s.touch("order", "ops")
	s.touch("driver", "ops")

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"emergency_id": emergencyID, "released": released,
	})
}

// handleOpenEmergencies الطوارئُ المفتوحة — لتراها العملياتُ مجموعةً.
func (s *Server) handleOpenEmergencies(w http.ResponseWriter, r *http.Request) {
	// **وصفحةٌ محدودةٌ بعدٍّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	//
	// **والمفتوحُ منها لا يُقفل نفسَه**: يتراكم حتّى تعالجه العملياتُ سطراً
	// سطراً. **ومئةٌ صامتةٌ تعني أنّ سائقاً في ضائقةٍ لا يراه أحد** — وهو
	// آخرُ ما يُحتمل صمتُه في هذه المنصّة.
	pg := pagingOf(r, 20)
	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM driver_emergencies WHERE status = 'open'`).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT e.id, u.full_name, u.phone, o.number, e.note,
		       ST_Y(e.at::geometry), ST_X(e.at::geometry), e.created_at
		FROM driver_emergencies e
		JOIN users u ON u.id = e.driver_id
		LEFT JOIN orders o ON o.id = e.order_id
		WHERE e.status = 'open'
		ORDER BY e.created_at DESC LIMIT $1 OFFSET $2`, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	// ══════════════════════════════════════════════════════════════════
	// **والوقتُ يُقرأ وقتاً — لا نصّاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// كان `created_at` يُقرأ في `string`، **فتردّ الشاشةُ خمسَمئة كلَّما
	// وُجد بلاغٌ واحد**: `cannot scan timestamptz into *string`.
	//
	// **والفارغُ كان يمرّ** — لا صفَّ فلا مسحَ فلا خطأ. **فبقي العطبُ خفيّاً
	// إلى أن يقع أوّلُ طارئ**، وهو آخرُ ما يُحتمل سقوطُه في هذه المنصّة:
	// **سائقٌ في ضائقةٍ يضغط الزرَّ ولا يراه أحدٌ في اللوحة.**
	//
	// (قِيس ٢٠٢٦-٠٨-١٠ في اختبارٍ شامل: بلاغٌ واحدٌ من سائقٍ حيٍّ فسقطت
	//  الشاشةُ كلُّها.)
	//
	// **وعائلتُه: ما لا يُختبر إلّا فارغاً يُقرأ سليماً وهو معطوب.**
	type row struct {
		ID          string    `json:"id"`
		DriverName  string    `json:"driver_name"`
		DriverPhone string    `json:"driver_phone"`
		OrderNumber *int64    `json:"order_number"`
		Note        string    `json:"note"`
		Lat         *float64  `json:"lat"`
		Lng         *float64  `json:"lng"`
		CreatedAt   time.Time `json:"created_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		var name *string
		if err := rows.Scan(&x.ID, &name, &x.DriverPhone, &x.OrderNumber,
			&x.Note, &x.Lat, &x.Lng, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		if name != nil {
			x.DriverName = *name
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, paged("emergencies", out, count, pg))
}

// handleResolveEmergency تُغلقها العملياتُ بعد أن تطمئنّ.
//
// **ولا تُغلق بمرور الوقت.** طارئٌ يختفي من الشاشة وحدَه يُنسى، **ومن سأل عنه
// بعد يومين لم يجد من يقول ماذا جرى.**
func (s *Server) handleResolveEmergency(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE driver_emergencies
		SET status = 'resolved', resolved_at = now(), resolved_by = $2
		WHERE id = $1 AND status = 'open'`, id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.audit(r, "ops.emergency_resolved", "emergency", id, nil)
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"resolved": true})
}
