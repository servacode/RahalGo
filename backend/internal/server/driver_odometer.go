package server

// ══════════════════════════════════════════════════════════════════════
// **عدّاد الدرّاجة عند فتح الدوام وسكّره** — قرارات المالك ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// «اول مايفتح الدوام يطلب منه تصوير العداد الصبح، وعند اغلاق الدوام كمان يطلب منه تصوير
// العداد، وهيك نقدر نعرف شقد مشى من اول الدوام لاخر الدوام.» و«حصراً الصورة بالليل، مشان
// الصبح نعرف إذا هو ماشي بالموتور».
//
//	١ · **الرقمُ المكتوبُ عليه الحساب، والصورةُ دليلُه** — `POST /driver/odometer`.
//	٢ · **رقمٌ أقلُّ من آخر رقمٍ له مرفوض** (`odometer_lower`) — وقفزةٌ فوق
//	    `drivers.odometer_max_jump_km` تُسأل ويؤكّدها (`odometer_check`).
//	٣ · **لا يُفتح الدوامُ ولا يُسكَّر بلاها** في وضع «إلزامي» — **للنسخ التي تعرفها وحدَها**
//	    (`odometerMinBuild`): ٠٫٧ و٠٫٨ تفتحان وتسكّران كما كانتا حتّى يُفرض التحديث.
//	٤ · **وما سكّره غيرُ السائق** (ترك · طارئ · حادث · عمليّات · حساب) يُكتب بسببه، ويُرى
//	    أحمرَ بالمراقبة: «ما صوّر عند الإغلاق».
//	٥ · **وسماحُ العمليّات** يفتح الدوامَ بلا صورةٍ مرّةً واحدة (`drivers.manage`، بسبب).
//	٦ · **وتصحيحُ رقمٍ غلط فيه السائق** بالكتابة — والأصلُ يبقى، والتصحيحُ مُدقَّق.
//
// **ولا مال**: لا قيدَ ولا محفظة.

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// odometerMinBuild **أوّلُ نسخةٍ من تطبيق السائق تعرف العدّاد** (٠٫٩٫٠). وما دونها يفتح
// الدوامَ ويسكّره كما كان — **فتشغيلُ «إلزامي» قبل فرض التحديث لا يقفل على قديمٍ دوامَه.**
const odometerMinBuild = 22

// odometerReadingTTL **كم تبقى قراءةُ البداية صالحةً لفتح الدوام** — رُفض الفتحُ (رقمٌ لم
// يُوثَّق مثلاً) فتُعاد القراءةُ نفسها ولا تُطلب صورةٌ ثانية.
const odometerReadingTTL = 30 * time.Minute

// odometerLateEnd **قراءةُ سكّرٍ تصل بعد سكّرٍ آليّ** (ترك أو طارئ) تُلحق بدوامه إن سُكّر قبل هذا.
const odometerLateEnd = 10 * time.Minute

var (
	errOdometerRequired = httpx.NewError(http.StatusConflict, "odometer_required", "errors.odometer_required")
	errOdometerBadKm    = httpx.NewError(http.StatusBadRequest, "odometer_bad_km", "errors.odometer_bad_km")
	errOdometerPhoto    = httpx.NewError(http.StatusBadRequest, "odometer_photo_required", "errors.odometer_photo_required")
	errOdometerMocked   = httpx.NewError(http.StatusUnprocessableEntity, "odometer_mocked", "errors.odometer_mocked")
	errOdometerPhase    = httpx.NewError(http.StatusConflict, "odometer_wrong_phase", "errors.odometer_wrong_phase")
	errOdometerNote     = httpx.NewError(http.StatusBadRequest, "odometer_note_required", "errors.odometer_note_required")
	errOdometerNotFound = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")
)

// errOdometerLower **رقمٌ أقلّ من آخر رقمٍ له — مرفوض** (قرار المالك)، والتفصيلُ يقول آخرَ رقم.
func errOdometerLower(last float64) error {
	return &httpx.AppError{Status: http.StatusConflict, Code: "odometer_lower",
		MessageKey: "errors.odometer_lower", Details: map[string]any{"last_km": kmString(last)}}
}

// errOdometerJump **قفزةٌ كبيرةٌ عن آخر رقم** — تُسأل، ويعيد السائقُ الإرسالَ بـ`confirm`.
func errOdometerJump(last, km float64) error {
	return &httpx.AppError{Status: http.StatusConflict, Code: "odometer_check",
		MessageKey: "errors.odometer_check", Details: map[string]any{
			"reason": "jump", "last_km": kmString(last), "km": kmString(km), "diff_km": kmString(km - last),
		}}
}

// kmString رقمٌ بمنزلةٍ عشريّةٍ على الأكثر وبأرقامٍ غربيّة.
func kmString(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
}

// parseOdometerKm **رقمُ العدّاد كما كتبه السائق** — أرقامٌ غربيّةٌ أو عربيّة، وفاصلٌ واحد
// (`.` أو `,` أو `٫`) ومنزلةٌ واحدة، وسبعُ خاناتٍ صحيحةٍ على الأكثر (`numeric(8,1)`).
func parseOdometerKm(raw string) (float64, bool) {
	var b strings.Builder
	ints, decimals, sep := 0, 0, false
	for _, c := range strings.TrimSpace(raw) {
		d := rune(-1)
		switch {
		case c >= '0' && c <= '9':
			d = c
		case c >= '٠' && c <= '٩':
			d = '0' + (c - '٠')
		case c >= '۰' && c <= '۹':
			d = '0' + (c - '۰')
		}
		switch {
		case d >= 0 && !sep:
			if ints++; ints > 7 {
				return 0, false
			}
			b.WriteRune(d)
		case d >= 0:
			if decimals++; decimals > 1 {
				return 0, false
			}
			b.WriteRune(d)
		case (c == '.' || c == ',' || c == '٫') && !sep && ints > 0:
			sep = true
			b.WriteByte('.')
		default:
			return 0, false
		}
	}
	s := strings.TrimSuffix(b.String(), ".")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

// odometerMode **وضعُ العدّاد لهذا النداء** — `off` لنسخةٍ لا تعرفه (والويب والاختبارات بلا
// ترويسة)، وإلّا الإعداد (`drivers.odometer_mode`).
func (s *Server) odometerMode(r *http.Request) string {
	if appBuild(r) < odometerMinBuild {
		return "off"
	}
	switch m := s.settings.GetString(r.Context(), "drivers.odometer_mode"); m {
	case "off", "ask", "require":
		return m
	}
	return "require"
}

// lastOdometerKm **آخرُ رقمٍ للسائق** (المصحَّحُ إن صُحّح) — وفارغٌ قبل أوّل قراءة.
func lastOdometerKm(ctx context.Context, q dbtx.Querier, driverID string) (*float64, error) {
	var km float64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(corrected_km, km)::float8 FROM driver_odometer_readings
		 WHERE driver_id = $1::uuid ORDER BY created_at DESC LIMIT 1`, driverID).Scan(&km)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &km, nil
}

// handleDriverOdometer **POST /driver/odometer** — صورةُ العدّاد ورقمُه (متعدّد الأجزاء:
// `file` · `km` · `phase` start|end · `lat` · `lng` · `accuracy` · `mocked` · `confirm`).
//
// **والفحوصُ قبل حفظ الصورة** — قراءةٌ مرفوضةٌ لا تترك ملفّاً بلا صاحب (كإثبات التسليم).
func (s *Server) handleDriverOdometer(w http.ResponseWriter, r *http.Request) {
	if !s.requireLaunch(w, r, launchDriverWork) {
		return
	}
	ctx := r.Context()
	uid := userIDFrom(r)
	phase := r.FormValue("phase")
	if phase != "start" && phase != "end" {
		s.respondErr(w, errValidation)
		return
	}
	km, ok := parseOdometerKm(r.FormValue("km"))
	if !ok {
		s.respondErr(w, errOdometerBadKm)
		return
	}
	// **والموقعُ المزيَّف يُرفض** — كصورة التسليم (قرارُ المالك ٢٠٢٦-١٠-٠٣).
	if r.FormValue("mocked") == "true" {
		s.audit(r, "driver.odometer_rejected", "user", uid, map[string]any{"why": "mocked", "phase": phase})
		s.respondErr(w, errOdometerMocked)
		return
	}
	var lat, lng, acc *float64
	if la, err1 := strconv.ParseFloat(r.FormValue("lat"), 64); err1 == nil {
		if ln, err2 := strconv.ParseFloat(r.FormValue("lng"), 64); err2 == nil &&
			!math.IsNaN(la) && !math.IsNaN(ln) && !math.IsInf(la, 0) && !math.IsInf(ln, 0) &&
			la >= -90 && la <= 90 && ln >= -180 && ln <= 180 {
			lat, lng = &la, &ln
		}
	}
	if a, err := strconv.ParseFloat(r.FormValue("accuracy"), 64); err == nil && a > 0 && !math.IsInf(a, 0) {
		acc = &a
	}
	confirm := r.FormValue("confirm") == "true"

	// ── المرحلةُ والحال: صورةُ الفتح والدوامُ مسكَّر، وصورةُ السكّر والدوامُ مفتوح ──
	var onShift bool
	if err := s.pg.QueryRow(ctx, `SELECT on_shift FROM users WHERE id = $1`, uid).Scan(&onShift); err != nil {
		s.respondErr(w, err)
		return
	}
	attachLate := ""
	switch {
	case phase == "start" && onShift:
		s.respondErr(w, errOdometerPhase)
		return
	case phase == "end" && !onShift:
		// **سكّرٌ آليٌّ قبل لحظات** (ترك أو طارئ): تُلحق قراءةُ الليل بدوامه.
		err := s.pg.QueryRow(ctx, `
			SELECT id::text FROM driver_shifts
			 WHERE driver_id = $1::uuid AND closed_at > now() - make_interval(secs => $2)
			   AND end_reading_id IS NULL
			 ORDER BY closed_at DESC LIMIT 1`, uid, odometerLateEnd.Seconds()).Scan(&attachLate)
		if err != nil {
			s.respondErr(w, errOdometerPhase)
			return
		}
	}

	// ── الرقمُ مقابل آخر رقم ──
	last, err := lastOdometerKm(ctx, s.pg, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	check := "first"
	if last != nil {
		check = "ok"
		if km < *last {
			s.respondErr(w, errOdometerLower(*last))
			return
		}
		if maxJump := float64(s.settings.GetInt(ctx, "drivers.odometer_max_jump_km")); maxJump > 0 && km-*last > maxJump {
			if !confirm {
				s.respondErr(w, errOdometerJump(*last, km))
				return
			}
			check = "jump_confirmed"
		}
	}

	// ── الصورةُ — من الكاميرا، وتُرفع كسائر الوسائط بفحصها وحدّها ──
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errOdometerPhoto)
		return
	}
	defer func() { _ = file.Close() }()
	md, err := s.media.Save(ctx, uid, "odometer", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var build any
	if b := appBuild(r); b > 0 {
		build = b
	}
	var id string
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO driver_odometer_readings (driver_id, phase, km, check_status, prev_km, media_id,
		                                      lat, lng, accuracy_m, app_build)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $7, $8, $9, $10)
		RETURNING id::text`, uid, phase, km, check, last, md.ID, lat, lng, acc, build).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	if attachLate != "" {
		if _, err := s.pg.Exec(ctx, `
			UPDATE driver_shifts SET end_reading_id = $2::uuid
			 WHERE id = $1::uuid AND end_reading_id IS NULL`, attachLate, id); err != nil {
			s.logger.Warn("العدّاد: تعذّر إلحاقُ قراءة السكّر", "shift", attachLate, "error", err)
		}
	}
	s.audit(r, "driver.odometer_reading", "user", uid, map[string]any{
		"reading": id, "km": km, "phase": phase, "check": check,
	})
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "km": km, "check": check})
}

// shiftReading **قراءةٌ صالحةٌ لهذا الفتح أو السكّر** — له، وبمرحلته، وغيرُ مربوطةٍ بدوام،
// وعمرُها دون نصف ساعة. وغيرُ ذلك فراغ (فيُطلب منه أن يصوّر).
func (s *Server) shiftReading(ctx context.Context, driverID, readingID, phase string) (string, error) {
	if readingID == "" || !isUUID(readingID) {
		return "", nil
	}
	var id string
	err := s.pg.QueryRow(ctx, `
		SELECT r.id::text FROM driver_odometer_readings r
		 WHERE r.id = $1::uuid AND r.driver_id = $2::uuid AND r.phase = $3
		   AND r.created_at > now() - make_interval(secs => $4)
		   AND NOT EXISTS (SELECT 1 FROM driver_shifts s
		                    WHERE s.start_reading_id = r.id OR s.end_reading_id = r.id)`,
		readingID, driverID, phase, odometerReadingTTL.Seconds()).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// pendingExemption **سماحُ العمليّات القائم** — معرّفُه ومن سمح وسببُه، أو فراغ.
func (s *Server) pendingExemption(ctx context.Context, driverID string) (id, by, reason string, err error) {
	var byID *string
	err = s.pg.QueryRow(ctx, `
		SELECT id::text, granted_by::text, reason FROM driver_odometer_exemptions
		 WHERE driver_id = $1::uuid AND used_at IS NULL`, driverID).Scan(&id, &byID, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", nil
	}
	if byID != nil {
		by = *byID
	}
	return id, by, reason, err
}

// ══════════════════════════════════════════════════════════════════════
// **تصحيحُ رقمٍ غلط فيه السائق** — `PATCH /admin/driver-odometer/{id}` (`drivers.monitor`)
// ══════════════════════════════════════════════════════════════════════

func (s *Server) handleAdminOdometerCorrect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, errOdometerNotFound)
		return
	}
	req, err := decode[struct {
		// **رقمٌ أو نصّ** — الحقلُ في الشاشة نصٌّ قد يُرسَل كما كُتب.
		Km   any    `json:"km"`
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var kmText string
	switch v := req.Km.(type) {
	case float64:
		kmText = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		kmText = v
	}
	km, ok := parseOdometerKm(kmText)
	if !ok {
		s.respondErr(w, errOdometerBadKm)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 300)
	if note == "" {
		s.respondErr(w, errOdometerNote)
		return
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var driverID string
	var old float64
	if err := tx.QueryRow(ctx, `
		SELECT driver_id::text, COALESCE(corrected_km, km)::float8 FROM driver_odometer_readings
		 WHERE id = $1::uuid FOR UPDATE`, id).Scan(&driverID, &old); err != nil {
		s.respondErr(w, errOdometerNotFound)
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE driver_odometer_readings
		   SET corrected_km = $2, corrected_by = $3::uuid, corrected_at = now(), correction_note = $4
		 WHERE id = $1::uuid`, id, km, userIDFrom(r), note); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.auditTx(ctx, tx, r, "ops.odometer_corrected", "user", driverID, map[string]any{
		"reading": id, "old": old, "new": km, "note": note,
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "km": km})
}

// ══════════════════════════════════════════════════════════════════════
// **سماحٌ بفتح الدوام بلا صورة — مرّة وحدة** — `POST /admin/drivers/{id}/odometer-exemption`
// ══════════════════════════════════════════════════════════════════════
//
// كاميرا مكسورة أو عدّادٌ معطّل: **العمليّاتُ تسمح بسببٍ مكتوب**، فيُفتح الدوامُ التالي بلا
// صورة ويُكتب على الدوام من سمح وليش (ويُرى بالمراقبة «سمحت له العمليات بلا صورة»).
// **ولا يُسمح بالسكّر بلا صورة** — صورةُ الليل حصراً؛ ومن تعذّرت عليه تسكّر العمليّاتُ دوامَه.
func (s *Server) handleAdminOdometerExemption(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := clip(strings.TrimSpace(req.Reason), 300)
	if reason == "" {
		s.respondErr(w, errOdometerNote)
		return
	}
	ctx := r.Context()
	var isDriver bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1::uuid AND role_code = 'driver')`,
		id).Scan(&isDriver); err != nil || !isDriver {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var exID string
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO driver_odometer_exemptions (driver_id, granted_by, reason)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (driver_id) WHERE used_at IS NULL
		DO UPDATE SET granted_by = EXCLUDED.granted_by, reason = EXCLUDED.reason, created_at = now()
		RETURNING id::text`, id, userIDFrom(r), reason).Scan(&exID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.odometer_exemption", "user", id, map[string]any{"reason": reason, "exemption": exID})
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"exemption_id": exID})
}

// ══════════════════════════════════════════════════════════════════════
// **صورُ العدّاد تُمحى بعد مدّة حفظها، والرقمُ يبقى** — مرّةً بالساعة من كانس المسافات
// ══════════════════════════════════════════════════════════════════════

var odoPruneGate struct {
	sync.Mutex
	last time.Time
}

func (s *Server) pruneOdometerPhotos(ctx context.Context, now time.Time) {
	odoPruneGate.Lock()
	due := now.Sub(odoPruneGate.last) >= time.Hour
	if due {
		odoPruneGate.last = now
	}
	odoPruneGate.Unlock()
	if !due || s.media == nil {
		return
	}
	days := s.settings.GetInt(ctx, "drivers.odometer_photo_retention_days")
	if days < 1 {
		return
	}
	if n, err := s.media.Purge(ctx, "odometer", now.AddDate(0, 0, -int(days)), 500); err != nil {
		s.logger.Warn("العدّاد: تعذّر محوُ الصور القديمة", "error", err)
	} else if n > 0 {
		s.logger.Info("العدّاد: مُحيت صورٌ مضت مدّةُ حفظها", "count", n)
	}
}
