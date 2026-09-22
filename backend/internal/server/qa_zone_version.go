package server

import (
	"net/http"
	"sync"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ══════════════════════════════════════════════════════════════════════
// **بذّارُ دوامِ المنطقة والحدِّ الأدنى للنسخة — على التجهيز وحدَه** (٢٠٢٦-٠٩-٢٢)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك: فِخاخٌ زبونيّةٌ ضيّقةٌ لشهود `zone_closed_now` و`update_required`
//  التي لا يفتحها بابُ QA الزبونيّ ولا `qa/setting` المنطقيّ.)
//
// **كلاهما عكوسٌ يحفظ السابقَ ويعيده**، **يسقط مغلقاً في الإنتاج** (الحارسُ في
// `handleQAStagingSeed`)، **QA-scoped على منطقةِ الاختبار/إعدادِ الزبون**، **ولا
// مساسَ ماليّ ولا أثرَ تشغيليّ بعد الاستعادة.**

// qaSavedHours **جدولُ منطقةٍ محفوظٌ قبل الإغلاق ليُعاد بعد الشهادة.**
type qaSavedHours struct {
	enforced bool
	windows  platform.Schedule
}

// qaZoneHoursSaved **في الذاكرة كحاقن الأعطال** — **الإغلاقُ والفتحُ في جلسةٍ
// واحدة.** **ولو سقط الخادمُ بينهما، `zone_reopen` يفتح المنطقةَ دائماً (سريانٌ
// مطفأٌ = مفتوحةٌ دوماً)، فلا تبقى مغلقةً بأثرِ فحص.**
var (
	qaZoneHoursMu    sync.Mutex
	qaZoneHoursSaved = map[string]qaSavedHours{}
)

// qaZoneClose **يُغلق منطقةً الآن حتميّاً** — يفرض `hours_enforced=true` بجدولٍ
// فارغٍ فلا نافذةَ تغطّي هذه اللحظة ⇒ `DecideZone` تردّ مغلقةً ⇒ `zone_closed_now`.
// **يحفظ السابقَ (الرايةَ والنوافذ) للاستعادة.**
func (s *Server) qaZoneClose(w http.ResponseWriter, r *http.Request, zoneID string) {
	if !isUUID(zoneID) {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	var enforced bool
	if err := s.pg.QueryRow(ctx,
		`SELECT hours_enforced FROM delivery_zones WHERE id = $1::uuid`, zoneID).Scan(&enforced); err != nil {
		s.respondErr(w, err)
		return
	}
	ws, err := s.platform.ZoneWindows(ctx, s.pg, zoneID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	qaZoneHoursMu.Lock()
	qaZoneHoursSaved[zoneID] = qaSavedHours{enforced: enforced, windows: ws}
	qaZoneHoursMu.Unlock()

	// **السريانُ بجدولٍ فارغ ⇒ لا نافذةَ مفتوحة ⇒ مغلقةٌ الآن.**
	if _, err := s.platform.SetZoneSchedule(ctx, zoneID, true, nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA zone closed now (staging-only)",
		"zone", zoneID, "prev_enforced", enforced, "prev_windows", len(ws))
	httpx.JSON(w, http.StatusOK, map[string]any{
		"zone_id": zoneID, "closed": true,
		"previous_enforced": enforced, "previous_windows": len(ws),
	})
}

// qaZoneReopen **يُعيد جدولَ المنطقةِ المحفوظَ** — وإن لم يُحفَظ شيءٌ (سقوطٌ بين
// النداءَين) فتحها دائماً (`hours_enforced=false`) فلا تبقى مغلقةً بأثرِ فحص.
func (s *Server) qaZoneReopen(w http.ResponseWriter, r *http.Request, zoneID string) {
	if !isUUID(zoneID) {
		s.respondErr(w, errValidation)
		return
	}
	qaZoneHoursMu.Lock()
	saved, ok := qaZoneHoursSaved[zoneID]
	delete(qaZoneHoursSaved, zoneID)
	qaZoneHoursMu.Unlock()

	enforced := false
	var ws platform.Schedule
	if ok {
		enforced, ws = saved.enforced, saved.windows
	}
	if _, err := s.platform.SetZoneSchedule(r.Context(), zoneID, enforced, ws); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA zone reopened (staging-only)",
		"zone", zoneID, "restored_prior", ok, "enforced", enforced, "windows", len(ws))
	httpx.JSON(w, http.StatusOK, map[string]any{
		"zone_id": zoneID, "reopened": true, "restored_prior": ok,
	})
}

// ══════════════════════════════════════════════════════════════════════
// **فتحُ متجرِ QA الآن حتميّاً — عكوسٌ** (٢٠٢٦-٠٩-٢٣، قرارُ المالك)
// ══════════════════════════════════════════════════════════════════════
//
// **لا قبولَ متجرٍ عامّ.** يفتح متجرَ صنفٍ بعينه على التجهيز ليُنشأ طلبٌ عاديٌّ
// خارجَ دوامه (14-011، 21-006/007، وحالاتُ السلّة). **الفتحُ = حذفُ صفوف
// `merchant_hours` (بلا صفوفٍ ⇒ مفتوحٌ دائماً، `OpenNowSQL`) + رفعُ الإغلاق
// الطارئ** — **يحفظ السابقَ في الذاكرة ويعيده.** **لا أثرَ ماليّ.**

type qaMHRow struct {
	dow         int
	closed      bool
	openT, clsT string // HH:MM:SS
}

type qaSavedMerchant struct {
	emergency bool
	hours     []qaMHRow
}

var (
	qaMerchantMu    sync.Mutex
	qaMerchantSaved = map[string]qaSavedMerchant{}
)

// qaMerchantFromItem **معرّفُ المتجر واسمُه وحالُه من صنفٍ.**
func (s *Server) qaMerchantFromItem(w http.ResponseWriter, r *http.Request, itemID string) (id, name, status string, ok bool) {
	if !isUUID(itemID) {
		s.respondErr(w, errValidation)
		return "", "", "", false
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT m.id::text, m.name, m.status
		FROM merchants m JOIN menu_items i ON i.merchant_id = m.id
		WHERE i.id = $1::uuid`, itemID).Scan(&id, &name, &status)
	if err != nil {
		s.respondErr(w, err)
		return "", "", "", false
	}
	return id, name, status, true
}

// qaMerchantOpen **يفتح متجرَ الصنف الآن** — يحفظ الجدولَ والإغلاقَ الطارئ.
func (s *Server) qaMerchantOpen(w http.ResponseWriter, r *http.Request, itemID string) {
	ctx := r.Context()
	mid, name, status, ok := s.qaMerchantFromItem(w, r, itemID)
	if !ok {
		return
	}
	var emergency bool
	if err := s.pg.QueryRow(ctx, `SELECT emergency_closed FROM merchants WHERE id = $1::uuid`, mid).Scan(&emergency); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx,
		`SELECT day_of_week, closed, open_time::text, close_time::text FROM merchant_hours WHERE merchant_id = $1::uuid`, mid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	saved := qaSavedMerchant{emergency: emergency}
	for rows.Next() {
		var h qaMHRow
		if err := rows.Scan(&h.dow, &h.closed, &h.openT, &h.clsT); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		saved.hours = append(saved.hours, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	qaMerchantMu.Lock()
	qaMerchantSaved[mid] = saved
	qaMerchantMu.Unlock()

	// **الفتح**: حذفُ الجدول (⇒ مفتوحٌ دائماً) ورفعُ الطارئ.
	if _, err := s.pg.Exec(ctx, `DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, mid); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(ctx, `UPDATE merchants SET emergency_closed = false WHERE id = $1::uuid`, mid); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA merchant forced open (staging-only)", "merchant", mid, "prev_emergency", emergency, "prev_hours", len(saved.hours), "status", status)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant_id": mid, "name": name, "status": status,
		"opened": true, "previous_emergency": emergency, "previous_hours": len(saved.hours),
	})
}

// qaMerchantRestore **يعيد جدولَ المتجر والإغلاقَ الطارئ المحفوظَين.**
func (s *Server) qaMerchantRestore(w http.ResponseWriter, r *http.Request, itemID string) {
	ctx := r.Context()
	mid, _, _, ok := s.qaMerchantFromItem(w, r, itemID)
	if !ok {
		return
	}
	qaMerchantMu.Lock()
	saved, had := qaMerchantSaved[mid]
	delete(qaMerchantSaved, mid)
	qaMerchantMu.Unlock()
	if !had {
		httpx.JSON(w, http.StatusOK, map[string]any{"merchant_id": mid, "restored": false, "note": "nothing saved"})
		return
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, mid); err != nil {
		s.respondErr(w, err)
		return
	}
	for _, h := range saved.hours {
		if _, err := tx.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1::uuid, $2, $3, $4::time, $5::time)`, mid, h.dow, h.closed, h.openT, h.clsT); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE merchants SET emergency_closed = $2 WHERE id = $1::uuid`, mid, saved.emergency); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA merchant restored (staging-only)", "merchant", mid, "hours", len(saved.hours), "emergency", saved.emergency)
	httpx.JSON(w, http.StatusOK, map[string]any{"merchant_id": mid, "restored": true, "hours": len(saved.hours)})
}

// qaMinVersion **يضبط الحدَّ الأدنى لنسخة الزبون** (`app.min_version.customer`)
// لشهود 426 `update_required`. **يُرجع السابقَ للاستعادة** — نداءٌ ثانٍ بقيمته
// يُعيد الحال. **إعدادٌ رقميٌّ لا يقلبه `qa/setting` المنطقيّ.**
func (s *Server) qaMinVersion(w http.ResponseWriter, r *http.Request, value int64) {
	const key = "app.min_version.customer"
	prev := s.settings.GetInt(r.Context(), key)
	if err := s.settings.Set(r.Context(), key, int(value), nil); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA min_version set (staging-only)", "key", key, "previous", prev, "set", value)
	httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "previous": prev, "set": value})
}
