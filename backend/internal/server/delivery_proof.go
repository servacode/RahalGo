package server

// **إثباتُ التسليم** — صورةٌ وإحداثياتٌ ووقت.
//
// # المسألة
//
// قاعدتُنا: **«الزبونُ يُصدَّق أوّلَ مرّة»** — والمنصةُ تتحمّل. **وقاعدةٌ بلا
// دليلٍ تكلفةٌ بلا سقف**: من عرف أنّ كلمتَه تكفي قالها مرّةً بعد مرّة، **ولا
// يبقى للسائق الصادق ما يدفع به عن نفسه.**
//
// **والمشكلةُ ليست في تصديق الزبون، هي في أنّ لا شيءَ يُقاس عليه.**
//
// # والمعيارُ العالميّ ثلاثةٌ لا واحد
//
//	الصورة       ←  موضعُ التسليم لا الطردُ وحدَه
//	الإحداثيات   ←  تُلتقط آلياً فتثبت أنّه كان هناك
//	الوقت        ←  آليٌّ كذلك — لا يُكتب بيد
//
// **وأهمُّها الإحداثيات**: صورةُ بابٍ قد تكون لأيّ باب، **وصورةٌ بإحداثياتٍ
// على بُعد أمتارٍ من عنوان الزبون بيّنة.**
//
// # ولا يقف التسليمُ على كاميرا — لكنّ التخطّي إذنُ عملياتٍ لا فعلُ سائق
//
// هاتفٌ لا يعمل، أو إذنٌ مرفوض، أو ليلٌ لا يُرى فيه شيء — **وسائقٌ لا يستطيع
// إنهاء طلبٍ سلّمه فعلاً يقف في الشارع.** **لكنّ السائقَ لا يأذن لنفسه بكلمة**
// (قرارُ المالك ٢٠٢٦-٠٩-٢٧): مسارُه العاديُّ يفشل آمناً، **والاستثناءُ يحتاج
// إذنَ أدمن/عمليّاتٍ مُخوَّل** (`handleAuthorizeProofException`) بسببٍ إلزاميٍّ
// مُدقَّقٍ في المعاملة. فمن عطبت كاميرتُه يطلب من العمليّات، وهي تأذن وتُوقّع.

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

var errProofRequired = httpx.NewError(http.StatusConflict,
	"delivery_proof_required", "errors.delivery_proof_required")

// errProofNotAtDoor **صورةُ التسليم عند باب الزبون وحدَه** (٢٠٢٦-١٠-٠٢).
var errProofNotAtDoor = httpx.NewError(http.StatusConflict,
	"proof_not_at_door", "errors.proof_not_at_door")

// errProofMocked **صورةٌ بموقعٍ مزيَّفٍ تُرفض** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً: «منع — تمنع
// احتيالَ السائق، مع تنبيهٍ للسائق»). **كانت تُحفظ بلا موضعٍ موسومةً** — فيمضي التسليم.
var errProofMocked = httpx.NewError(http.StatusUnprocessableEntity,
	"proof_mocked", "errors.proof_mocked")

// proofAccuracyCapM **أكبرُ هامشِ دقّةٍ يُضاف إلى حدّ الصورة** — قرارُ المالك ٢٠٢٦-١٠-٠٣.
const proofAccuracyCapM = 50.0

// errProofTooFar **صورةٌ من مكانٍ بعيدٍ عن الزبون تُرفض** — والمسافةُ والحدُّ في التفصيل
// نصّاً، فيقولهما التطبيقُ للسائق: «أنت بعيدٌ عن الزبون (٣٢٠ م) — اقترب ثمّ صوّر».
func errProofTooFar(distM, maxM int64) error {
	return &httpx.AppError{
		Status: http.StatusUnprocessableEntity, Code: "proof_too_far",
		MessageKey: "errors.proof_too_far",
		Details: map[string]any{
			"distance_m": strconv.FormatInt(distM, 10),
			"max_m":      strconv.FormatInt(maxM, 10),
		},
	}
}

// handleDeliveryProof يحفظ صورةَ التسليم وموضعَها.
//
// **تُرفع قبل «سُلّم» لا بعده**: بعد الإغلاق يصير الطلبُ تاريخاً، **وصورةٌ
// تُضاف إلى تاريخٍ مغلقٍ تُقرأ إضافةً متأخّرة** — وهي أضعفُ ما يُحتجّ به.
func (s *Server) handleDeliveryProof(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **والصورةُ عند الباب وحدَه** (٢٠٢٦-١٠-٠٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كانت تُقبل في أيّ حال** — فصورةٌ تُلتقط عند المتجر أو في الطريق تُحفظ
	// «إثباتَ تسليم»، **ويُقاس بُعدُها عن الباب فيُقرأ تسليماً بعيداً وهو لم يُسلَّم
	// بعد.** والبيّنةُ التي بُنيت للنزاع تشهد بما لم يقع.
	var status string
	if err := s.pg.QueryRow(r.Context(), `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		s.respondErr(w, err)
		return
	}
	if status != orders.StAtDropoff {
		s.respondErr(w, errProofNotAtDoor)
		return
	}

	// **الإحداثياتُ تأتي مع الصورة لا بعدها.**
	//
	// موضعٌ يُرسَل في نداءٍ ثانٍ **قد يُرسَل من مكانٍ آخر** — والسائقُ يتحرّك.
	lat, errLat := strconv.ParseFloat(r.FormValue("lat"), 64)
	lng, errLng := strconv.ParseFloat(r.FormValue("lng"), 64)
	hasPoint := errLat == nil && errLng == nil

	// ══════════════════════════════════════════════════════════════════
	// **وموقعٌ مزيَّفٌ لا يُقبل إثباتاً — ولا الصورةُ معه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قِيس 2026-09-02: لا فحصَ للتزييف في المنصّة كلِّها.)
	//
	// **وإثباتُ التسليم يقول «على بعد خمسةِ أمتارٍ من العنوان»**
	// محسوبةً من النقطة التي يرسلها الجهازُ نفسُه. **فمن زيّف موضعَه
	// كتب الإثباتَ بيده** — والحارسُ الذي بُني للحماية يشهد له.
	//
	// **وكانت تُرفض النقطةُ وتُحفظ الصورة** — فيمضي التسليمُ بإثباتٍ بلا موضع. **وقرارُ
	// المالك (٢٠٢٦-١٠-٠٣ مساءً): «منع — مع تنبيهٍ للسائق».** فتُرفض كلُّها ويُقال له
	// لماذا: يُطفئ برنامجَ التزييف ثمّ يصوّر.
	//
	// **والفحوصُ قبل حفظ الملفّ** — صورةٌ مرفوضةٌ لا تترك ملفّاً بلا صاحب.
	mocked := r.FormValue("mocked") == "true"
	if mocked {
		s.audit(r, "driver.delivery_proof_rejected", "order", orderID, map[string]any{"why": "mocked"})
		s.respondErr(w, errProofMocked)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **وصورةٌ من بعيدٍ ليست صورةَ تسليم** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً)
	// ══════════════════════════════════════════════════════════════════
	//
	// **أبعدَ من `drivers.proof_max_m` عن نقطة الزبون تُرفض** ويُقال للسائق كم هو بعيد.
	// **وبابٌ لا يُعرف لا يُقاس عليه** — «لدي توصيلة» بلا نقطة (`dropoff_known`).
	//
	// **ويُضاف هامشُ دقّة الجوال لحظتَها** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «١٥٠ متر كثير» فصار ١٥،
	// **مع هامش الدقّة** فلا يُظلَم سائقٌ بين البنايات) — **وسقفُه ٥٠ م**: جوالٌ يقول «دقّتي
	// ألفُ متر» لا يشتري ألفَ متر. **وغيابُها هامشٌ صفر** — نسخةٌ قديمةٌ لا ترسلها تُقاس بالحدّ وحدَه.
	if hasPoint {
		var known bool
		var dist *float64
		if err := s.pg.QueryRow(r.Context(), `
			SELECT dropoff_known,
			       ST_Distance(dropoff, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
			FROM orders WHERE id = $1`, orderID, lng, lat).Scan(&known, &dist); err != nil {
			s.respondErr(w, err)
			return
		}
		maxM := s.settings.GetInt(r.Context(), "drivers.proof_max_m")
		if acc, err := strconv.ParseFloat(r.FormValue("accuracy"), 64); err == nil && acc > 0 {
			maxM += int64(min(acc, proofAccuracyCapM))
		}
		if known && dist != nil && maxM > 0 && *dist > float64(maxM) {
			s.audit(r, "driver.delivery_proof_rejected", "order", orderID, map[string]any{
				"why": "too_far", "distance_m": int64(*dist), "max_m": maxM,
			})
			s.respondErr(w, errProofTooFar(int64(*dist), maxM))
			return
		}
	}

	// **الصورةُ تُرفع كسائر الوسائط** — بالفحص والحدّ نفسِهما.
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer func() { _ = file.Close() }()

	md, err := s.media.Save(r.Context(), userIDFrom(r), "delivery_proof", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	q := `UPDATE orders SET pod_media_id = $2, pod_taken_at = now(), pod_skip_reason = '',
	                       pod_mocked = ` + boolLit(mocked)
	args := []any{orderID, md.ID}
	if hasPoint {
		q += `, pod_at = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography`
		args = append(args, lng, lat)
	}
	q += ` WHERE id = $1`
	if _, err := s.pg.Exec(r.Context(), q, args...); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "driver.delivery_proof", "order", orderID, map[string]any{
		"media": md.ID, "located": hasPoint, "mocked": mocked,
	})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"media_id": md.ID, "located": hasPoint,
	})
}

// handleAuthorizeProofException **إذنُ عملياتٍ بتسليمٍ بلا صورة.**
//
// **قرارُ المالك ٢٠٢٦-٠٩-٢٧**: لا يتخطّى السائقُ إثباتَ التسليم بكلمةٍ يكتبها
// بنفسه — **فيأذن لنفسه.** الاستثناءُ الحقيقيُّ (كاميرا معطّلةٌ أو غيرُ متاحة)
// يحتاج **إذنَ أدمن/عمليّاتٍ مُخوَّل** (قدرةُ `OrdersIntervene`)، والسببُ
// إلزاميّ. **ويُقيَّد في معاملةٍ واحدةٍ**: مَن أذن (الفاعل) وللسائق (`driver_id`)
// ولأيّ طلبٍ (الكيان) والسبب والوقت — دائمٌ يُقرأ يومَ النزاع. **وإعادةُ النداء
// آمنة**: إذنٌ قائمٌ لا يُكتب ثانيةً ولا يُدقَّق مرّتين.
func (s *Server) handleAuthorizeProofException(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}

	actor := userIDFrom(r)
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// **قفلُ الصفّ يجعل فحصَ التكرار صادقاً** — كنظير تعويض السائق.
	var driverID, skipBy *string
	if err := tx.QueryRow(r.Context(),
		`SELECT driver_id::text, pod_skip_by::text FROM orders WHERE id = $1 FOR UPDATE`,
		orderID).Scan(&driverID, &skipBy); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **أُذن من قبل**: إعادةُ النداء لا تكتب ثانيةً ولا تُدقّق مرّتين.
	if skipBy != nil {
		if err := tx.Commit(r.Context()); err != nil {
			s.respondErr(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"authorized": true, "already": true})
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE orders SET pod_skip_by = $2, pod_skip_reason = $3, pod_skip_at = now()
		WHERE id = $1`, orderID, actor, clip(reason, 200)); err != nil {
		s.respondErr(w, err)
		return
	}
	// **الأثرُ في المعاملة نفسِها** (`AQ-4`): تخطٍّ لقاعدة سلامةٍ يجب أن يبقى
	// أثرُه ولو سقط ما بعده.
	if err := s.auditTx(r.Context(), tx, r, "ops.delivery_proof_exception", "order", orderID, map[string]any{
		"driver_id": driverID, "reason": reason,
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"authorized": true})
}

// requireProofBeforeDelivery يمنع «سُلّم» بلا إثباتٍ ولا سببٍ لتخطّيه.
//
// **والحارسُ في الخادم لا في الشاشة**: زرٌّ يُخفى يُلتفّ عليه، **وقاعدةٌ
// تُفرض في المحرّك لا.**
func (s *Server) requireProofBeforeDelivery(r *http.Request, orderID string) error {
	if !s.settings.GetBool(r.Context(), "drivers.require_delivery_photo") {
		return nil
	}
	// **إمّا صورةٌ، وإمّا إذنُ استثناءٍ مُخوَّل** (٢٠٢٦-٠٩-٢٧): `pod_skip_by`
	// يضعه أدمن/عمليّاتٌ مُخوَّلٌ وحدَه، **ولا يقبل الحارسُ كلمةَ السائق
	// (`pod_skip_reason`) بلا إذن.** فالسائقُ العاديُّ لا يأذن لنفسه.
	var mediaID, skipBy *string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT pod_media_id::text, pod_skip_by::text FROM orders WHERE id = $1`,
		orderID).Scan(&mediaID, &skipBy); err != nil {
		return err
	}
	if mediaID == nil && skipBy == nil {
		return errProofRequired
	}
	return nil
}

// boolLit **حرفٌ منطقيٌّ في نصّ الاستعلام** — لا وسيطٌ مرقَّم.
//
// **والوسائطُ هنا مرقّمةٌ بترتيبٍ يتبدّل** (`$3` و`$4` تُضافان مع
// الموضع وحدَه)، **ومن أقحم وسيطاً في المنتصف بدّل ما بعده.** والقيمةُ
// منطقيّةٌ من مقارنةٍ عندنا لا من مدخلٍ خارجيّ، **فلا حقنَ فيها.**
func boolLit(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
