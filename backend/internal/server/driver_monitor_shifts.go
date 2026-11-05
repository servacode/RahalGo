package server

// ══════════════════════════════════════════════════════════════════════
// **الدوامات بيوم السائق — والعدّاد مقابل التطبيق** (قرارات المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// كل دوامٍ مسّ اليوم: متى فُتح ومتى سُكّر ومين سكّره، وصورتا العدّاد ورقماهما (بروابط
// موقَّعة)، وأرقام التطبيق من **التحليل نفسه** اللي بيكتب مسافة كل طلب
// (`drivermonitor.MeasureShift`) — فما في رقمين بيختلفوا.
//
//	حسب العدّاد        = رقم السكّر − رقم الفتح (المصحَّح إن صُحّح)
//	ما سجّلها التطبيق  = حسب العدّاد − القيادة الفعلية
//	مشى خارج الدوام    = رقم هالفتح − رقم سكّر الدوام اللي قبله («لازم يكون صفر»)
//	تقريبي             = بلا صورة سكّر: رقم فتح الدوام اللي بعده − رقم هالفتح
//
// **والأحمر**: فرقٌ كبير بين العدّاد والتطبيق (كيلو ونسبة معاً، من الإعدادات)، وسكّرٌ بلا
// صورة («ما صوّر عند الإغلاق» — صار بلا السائق: ترك أو طارئ أو عمليّات أو حساب أو جوال
// مطفي)، ومشيٌ وهو خارج الدوام.

import (
	"context"
	"math"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// longOpenHours **دوامٌ مفتوحٌ أطول من هذا** يُعلَّم — جوالٌ مطفي أو نسيان.
const longOpenHours = 16

// odoReading قراءةُ عدّادٍ كما تراها المراقبة.
type odoReading struct {
	ID             string    `json:"id"`
	Km             float64   `json:"km"`
	CorrectedKm    *float64  `json:"corrected_km"`
	EffectiveKm    float64   `json:"effective_km"`
	CheckStatus    string    `json:"check_status"`
	CorrectionNote string    `json:"correction_note"`
	PhotoURL       *string   `json:"photo_url"`
	ThumbURL       *string   `json:"thumb_url"`
	CreatedAt      time.Time `json:"created_at"`
	AppBuild       *int      `json:"app_build"`
}

// monitorShift دوامٌ بيوم السائق.
type monitorShift struct {
	ID          string     `json:"id"`
	OpenedAt    time.Time  `json:"opened_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	CloseReason string     `json:"close_reason"`
	OpenedBuild *int       `json:"opened_build"`
	Migrated    bool       `json:"migrated"`
	// أرقام التطبيق (متر) — القيادة الفعلية وتقسيمها.
	DriveM      int  `json:"drive_m"`
	OrdersM     int  `json:"orders_m"`
	NoOrderM    int  `json:"no_order_m"`
	NotDrivingM int  `json:"not_driving_m"`
	GapMin      int  `json:"gap_min"`
	OrdersN     int  `json:"orders_n"`
	Final       bool `json:"final"`
	// OpenHours كم ساعة له مفتوحاً (للمفتوح وحده).
	OpenHours int `json:"open_hours,omitempty"`

	// ── العدّاد ──
	StartReading *odoReading `json:"start_reading"`
	EndReading   *odoReading `json:"end_reading"`
	OdometerKm   *float64    `json:"odometer_km"`
	ApproxKm     *float64    `json:"approx_km"`
	OffBeforeKm  *float64    `json:"off_before_km"`
	UnseenKm     *float64    `json:"unseen_km"`
	ExemptByName string      `json:"exempt_by_name,omitempty"`
	ExemptReason string      `json:"exempt_reason,omitempty"`
	Flags        []string    `json:"flags"`
}

// odoDay ملخّصُ العدّاد ليوم السائق — للقائمة ورأس اليوم.
type odoDay struct {
	// OnKm مجموعُ «حسب العدّاد» للدوامات التي لها صورتا فتحٍ وسكّر.
	OnKm *float64 `json:"on_km"`
	// OffKm مجموعُ «مشى وهو خارج الدوام».
	OffKm *float64 `json:"off_km"`
	// AppKm قيادةُ التطبيق للدوامات نفسها (التي لها «حسب العدّاد»).
	AppKm *float64 `json:"app_km"`
	// UnseenKm OnKm − AppKm.
	UnseenKm     *float64 `json:"unseen_km"`
	MissingClose int      `json:"missing_close"`
	BigGap       bool     `json:"big_gap"`
	OffMoved     bool     `json:"off_moved"`
	Shifts       int      `json:"shifts"`
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func f64p(v float64) *float64 { return &v }

// odoThresholds عتباتُ الأعلام من الإعدادات.
type odoThresholds struct{ gapKm, gapPct, offKm float64 }

func (s *Server) odoThresholds(ctx context.Context) odoThresholds {
	return odoThresholds{
		gapKm:  float64(s.settings.GetInt(ctx, "drivers.odometer_gap_alert_km")),
		gapPct: float64(s.settings.GetInt(ctx, "drivers.odometer_gap_alert_pct")),
		offKm:  float64(s.settings.GetInt(ctx, "drivers.odometer_off_alert_km")),
	}
}

// bigGap **فرقٌ كبير**: ما سجّلها التطبيق ≥ الكيلومترات **و** ≥ النسبة من العدّاد.
func (t odoThresholds) bigGap(odoKm, unseenKm float64) bool {
	return t.gapKm > 0 && unseenKm >= t.gapKm && odoKm > 0 && unseenKm*100 >= t.gapPct*odoKm
}

// monitorShifts دوامات السائق التي تمسّ [from, to) — بقراءاتها وأعلامها. `measure` يحسب
// أرقام التطبيق الآن (يوم السائق)، وبلاه تُقرأ من حساب الكانس (القائمة).
func (s *Server) monitorShifts(ctx context.Context, driverID string, from, to, now time.Time, measure bool) ([]monitorShift, odoDay, error) {
	// **والدوامُ قبلها وبعدها** — لـ«خارج الدوام» و«التقريبي».
	rows, err := s.pg.Query(ctx, `
		WITH win AS (
			SELECT s.* FROM driver_shifts s
			 WHERE s.driver_id = $1::uuid AND s.opened_at < $3 AND COALESCE(s.closed_at, $4) >= $2
		), around AS (
			SELECT * FROM win
			UNION
			(SELECT s.* FROM driver_shifts s WHERE s.driver_id = $1::uuid AND s.opened_at < $2
			  ORDER BY s.opened_at DESC LIMIT 1)
			UNION
			(SELECT s.* FROM driver_shifts s WHERE s.driver_id = $1::uuid AND s.opened_at >= $3
			  ORDER BY s.opened_at LIMIT 1)
		)
		SELECT a.id::text, a.opened_at, a.closed_at, COALESCE(a.close_reason, ''), a.opened_build,
		       a.migrated, a.final, COALESCE(a.drive_m, 0), COALESCE(a.orders_m, 0),
		       COALESCE(a.no_order_m, 0), COALESCE(a.not_driving_m, 0), COALESCE(a.gap_min, 0),
		       COALESCE(a.orders_n, 0), COALESCE(eu.full_name, ''), a.exempt_reason,
		       a.id IN (SELECT id FROM win), a.computed_at IS NOT NULL,
		       sr.id::text, sr.km::float8, sr.corrected_km::float8, sr.check_status, sr.correction_note,
		       sm.path, sm.thumb_path, sr.created_at, sr.app_build,
		       er.id::text, er.km::float8, er.corrected_km::float8, er.check_status, er.correction_note,
		       em.path, em.thumb_path, er.created_at, er.app_build
		  FROM around a
		  LEFT JOIN users eu ON eu.id = a.exempt_by
		  LEFT JOIN driver_odometer_readings sr ON sr.id = a.start_reading_id
		  LEFT JOIN media sm ON sm.id = sr.media_id
		  LEFT JOIN driver_odometer_readings er ON er.id = a.end_reading_id
		  LEFT JOIN media em ON em.id = er.media_id
		 ORDER BY a.opened_at`, driverID, from, to, now)
	if err != nil {
		return nil, odoDay{}, err
	}
	type row struct {
		m        monitorShift
		inside   bool
		computed bool
	}
	var all []row
	for rows.Next() {
		var x row
		var sID, eID, sCheck, eCheck, sNote, eNote, sPath, sThumb, ePath, eThumb *string
		var sKm, sCor, eKm, eCor *float64
		var sAt, eAt *time.Time
		var sBuild, eBuild *int
		m := &x.m
		if err := rows.Scan(&m.ID, &m.OpenedAt, &m.ClosedAt, &m.CloseReason, &m.OpenedBuild, &m.Migrated,
			&m.Final, &m.DriveM, &m.OrdersM, &m.NoOrderM, &m.NotDrivingM, &m.GapMin, &m.OrdersN,
			&m.ExemptByName, &m.ExemptReason, &x.inside, &x.computed,
			&sID, &sKm, &sCor, &sCheck, &sNote, &sPath, &sThumb, &sAt, &sBuild,
			&eID, &eKm, &eCor, &eCheck, &eNote, &ePath, &eThumb, &eAt, &eBuild); err != nil {
			rows.Close()
			return nil, odoDay{}, err
		}
		m.StartReading = makeReading(sID, sKm, sCor, sCheck, sNote, sPath, sThumb, sAt, sBuild)
		m.EndReading = makeReading(eID, eKm, eCor, eCheck, eNote, ePath, eThumb, eAt, eBuild)
		all = append(all, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, odoDay{}, err
	}

	th := s.odoThresholds(ctx)
	out := []monitorShift{}
	var day odoDay
	for i := range all {
		if !all[i].inside {
			continue
		}
		m := all[i].m
		if measure {
			km, _, err := drivermonitor.MeasureShift(ctx, s.pg, drivermonitor.ShiftSpan{
				ID: m.ID, DriverID: driverID, OpenedAt: m.OpenedAt, ClosedAt: m.ClosedAt,
			}, now)
			if err != nil {
				return nil, odoDay{}, err
			}
			m.DriveM, m.OrdersM, m.NoOrderM = km.DriveM, km.OrdersM, km.NoOrderM
			m.NotDrivingM, m.GapMin, m.OrdersN = km.NotDrivingM, km.GapMin, km.OrdersN
		}
		if m.ClosedAt == nil {
			m.OpenHours = int(now.Sub(m.OpenedAt) / time.Hour)
		}
		var prev, next *monitorShift
		if i > 0 {
			prev = &all[i-1].m
		}
		if i+1 < len(all) {
			next = &all[i+1].m
		}
		m.Flags = []string{}
		flag := func(f string) { m.Flags = append(m.Flags, f) }
		// **وأرقامُ التطبيق معروفةٌ** إن حُسبت الآن أو حسبها الكانس — وإلّا لا «فرقَ» يُقال.
		known := measure || all[i].computed
		if m.StartReading != nil && m.EndReading != nil {
			odo := round1(m.EndReading.EffectiveKm - m.StartReading.EffectiveKm)
			m.OdometerKm = f64p(odo)
			if known {
				unseen := round1(odo - float64(m.DriveM)/1000)
				m.UnseenKm = f64p(unseen)
				if th.bigGap(odo, unseen) {
					flag("big_gap")
				}
			}
		} else if m.StartReading != nil && m.EndReading == nil && next != nil && next.StartReading != nil {
			m.ApproxKm = f64p(round1(next.StartReading.EffectiveKm - m.StartReading.EffectiveKm))
		}
		if m.StartReading != nil && prev != nil && prev.EndReading != nil {
			off := round1(m.StartReading.EffectiveKm - prev.EndReading.EffectiveKm)
			m.OffBeforeKm = f64p(off)
			if th.offKm > 0 && off >= th.offKm {
				flag("off_duty_moved")
			}
		}
		if m.StartReading == nil {
			flag("no_open_photo")
		}
		if m.ClosedAt != nil && m.EndReading == nil {
			flag("no_close_photo")
		}
		if (m.StartReading != nil && m.StartReading.CheckStatus == "jump_confirmed") ||
			(m.EndReading != nil && m.EndReading.CheckStatus == "jump_confirmed") {
			flag("jump_confirmed")
		}
		if m.OpenedBuild == nil || *m.OpenedBuild < odometerMinBuild {
			flag("old_app")
		}
		if m.ExemptByName != "" || m.ExemptReason != "" {
			flag("exempt")
		}
		if m.ClosedAt == nil && m.OpenHours >= longOpenHours {
			flag("long_open")
		}
		if m.Migrated {
			flag("migrated")
		}
		// ── ملخّصُ اليوم ──
		day.Shifts++
		if m.ClosedAt != nil && m.EndReading == nil {
			day.MissingClose++
		}
		if m.OdometerKm != nil && known {
			day.OnKm = f64p(round1(val(day.OnKm) + *m.OdometerKm))
			day.AppKm = f64p(round1(val(day.AppKm) + float64(m.DriveM)/1000))
		}
		if m.OffBeforeKm != nil {
			day.OffKm = f64p(round1(val(day.OffKm) + *m.OffBeforeKm))
		}
		out = append(out, m)
	}
	if day.OnKm != nil && day.AppKm != nil {
		day.UnseenKm = f64p(round1(*day.OnKm - *day.AppKm))
		day.BigGap = th.bigGap(*day.OnKm, *day.UnseenKm)
	}
	day.OffMoved = day.OffKm != nil && th.offKm > 0 && *day.OffKm >= th.offKm
	return out, day, nil
}

func val(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// makeReading قراءةٌ من أعمدتها — والصورةُ برابطٍ موقَّع (صنفٌ محميّ).
func makeReading(id *string, km, corrected *float64, check, note, path, thumb *string, at *time.Time, build *int) *odoReading {
	if id == nil || km == nil {
		return nil
	}
	r := &odoReading{ID: *id, Km: *km, CorrectedKm: corrected, EffectiveKm: *km, AppBuild: build}
	if corrected != nil {
		r.EffectiveKm = *corrected
	}
	if check != nil {
		r.CheckStatus = *check
	}
	if note != nil {
		r.CorrectionNote = *note
	}
	if at != nil {
		r.CreatedAt = *at
	}
	r.PhotoURL = media.SignedURLPtr(path)
	r.ThumbURL = media.SignedURLPtr(thumb)
	return r
}
