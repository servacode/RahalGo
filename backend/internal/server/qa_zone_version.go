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
