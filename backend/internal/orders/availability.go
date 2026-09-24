package orders

// ══════════════════════════════════════════════════════════════════════
// **أيُطلَب الآن — ولِمَ لا — ومتى يُستطاع** (`AV`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// # ولمَ نموذجٌ قارئٌ واحد
//
// **وكانت الشاشةُ تجمعها بنفسها**: تقرأ التسعيرةَ فتعرف التغطية،
// **ولا تعرف دوامَ المنصّة ولا وقتَ المنطقة ولا دوامَ المتجر إلّا حين
// يُردّ الطلبُ عند الإتمام.** **فيملأ الزبونُ سلّةً ليُردَّ في آخرها.**
//
// **وثلاثُ شاشاتٍ تجمعها ثلاثَ مرّاتٍ تفترق** — **فتقول إحداها «مفتوح»
// والأخرى «مغلق» في اللحظة نفسِها.**
//
// # وليست بوّابةً — بل شرحاً لما ستفعله البوّابة
//
// **والمنعُ يبقى حيث هو**: `requireLaunch` و`requireOrdering` في
// المعالِج، **و`RequireServiceable` و`requireZoneOpen` و`OpenNowSQL`
// داخلَ معاملة الإنشاء.** **ولا يُضعَّف واحدٌ منها ليوافق هذا.**
//
// **وهذا يقرأ المصادرَ عينَها** — **فلا يفترق ما يُشرَح عمّا يُمنَع.**
// **ونداءٌ مباشرٌ يتجاوز هذا ولا يتجاوز تلك.**
//
// # ولمَ يُستورَد `platform` هنا
//
// **وبوّابةُ القبول تكتفي بواجهةٍ ضيّقة** (`ZoneHours`) — **جوابٌ
// بنعم أو لا وموعد.** **وهذا يحتاج الجداولَ أنفسَها ليقاطعها**،
// **وأنواعُ الجدول هناك.** **ونسخُها هنا هو الانقسامُ الذي نهربُ منه.**

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// ══════════════════════════════════════════════════════════════════════
// **الأسبابُ — أسماءٌ ثابتةٌ تقرؤها الآلة**
// ══════════════════════════════════════════════════════════════════════
//
// **ورمزُ الخطأ الذي يردّ به الإنشاء يبقى كما هو** — **ولا يُعاد
// تسميةُ عقدٍ منشورٍ لأجل جمال.** **وهذه أسماءُ قراءةٍ فوقَه**،
// **ويحمل كلُّ سببٍ رمزَ إنشائه معه** (`OrderCode`) **فلا يُخمَّن.**
const (
	// ReasonAvailable **يُقبَل الطلبُ الآن.**
	ReasonAvailable = "service_available"
	// ReasonLaunchClosed **بابٌ لم يُفتح للناس بعد** — لا موعدَ له.
	ReasonLaunchClosed = "launch_closed"
	// ReasonTemporarilyUnavailable **إيقافٌ تشغيليٌّ مؤقّت.**
	ReasonTemporarilyUnavailable = "temporarily_unavailable"
	// ReasonPlatformClosedNow **خارجَ دوام المنصّة.**
	ReasonPlatformClosedNow = "platform_closed_now"
	// ReasonInvalidLocation **نقطةٌ ليست نقطة** — رمزُها `bad_point`.
	ReasonInvalidLocation = "invalid_location"
	// ReasonCoverageUnavailable **لا إعدادَ تغطيةٍ صالحاً** — حالُ
	// إعدادٍ لا حكمٌ على العنوان.
	ReasonCoverageUnavailable = "coverage_unavailable"
	// ReasonProvinceNotSupported **محافظةٌ لم تُطلَق فيها الخدمة.**
	ReasonProvinceNotSupported = "province_not_supported"
	// ReasonCityNotSupported **مدينةٌ لم تُطلَق فيها الخدمة.**
	ReasonCityNotSupported = "city_not_supported"
	// ReasonAreaNotSupported **موضعٌ لا تعرفه المنصّةُ أصلاً.**
	//
	// **ولا مدينةَ تحويه ولا محافظةَ تُسمّى له** — **والمحافظاتُ بلا
	// هندسةٍ في المخطَّط**، **والمدنُ وحدَها تحمل مركزاً ونصفَ قطر.**
	// **فيُقال «لم نصل بعد» ولا يُسمّى ما لا يُعرَف.**
	ReasonAreaNotSupported = "area_not_supported"
	// ReasonAddressOutsideCoverage **الخدمةُ مُطلَقةٌ هنا والعنوانُ
	// خارجَ الأشكال** — رمزُها `out_of_zone`.
	ReasonAddressOutsideCoverage = "address_outside_coverage"
	// ReasonZoneClosedNow **المنطقةُ مُغطّاةٌ وخارجَ وقتها.**
	ReasonZoneClosedNow = "zone_closed_now"
	// ReasonMerchantClosedNow **المتجرُ مغلقٌ الآن.**
	ReasonMerchantClosedNow = "merchant_closed_now"
)

// Availability **ما يقوله المحرّكُ قبل أن يضغط الزبون.**
type Availability struct {
	// Available **أيُقبَل طلبٌ جديدٌ لهذا العنوان الآن؟**
	Available bool `json:"available"`
	// Reason **السببُ باسمه** — وفارغٌ حين يُقبَل… لا: `service_available`.
	Reason string `json:"reason"`
	// OrderCode **رمزُ الخطأ الذي سيردّ به الإنشاء فعلاً** — وفارغٌ
	// حين لا رمزَ له (`merchant_closed` مثلاً له رمزُه).
	//
	// **ويُرسَل ليُقارَن لا ليُعرَض** — **والشاشةُ تعرض `Reason`.**
	OrderCode string `json:"order_code,omitempty"`
	// Message **نصُّ المالك** — للإيقاف المؤقّت وحدَه اليوم.
	Message string `json:"message,omitempty"`
	// NextAvailableAt **أوّلُ لحظةٍ تتقاطع فيها القيودُ كلُّها.**
	//
	// **وفارغٌ يعني «لا يُعرَف»** — **ولا يُخترَع موعد.**
	NextAvailableAt *time.Time `json:"next_available_at,omitempty"`
	// PlaceName **اسمُ المدينة أو المحافظة حين يُعرَف** — للرسالة.
	//
	// **ولا يُسمّى ما لا يُعرَف**: `area_not_supported` بلا اسم.
	PlaceName string `json:"place_name,omitempty"`
	// CityID و GovernorateID **معرّفان ثابتان حين يُعرَفان.**
	//
	// **ويُرسَلان ليُبنى عليهما لا ليُعرَضا** — **والاسمُ يتبدّل بتصحيحٍ
	// إملائيٍّ في اللوحة، والمعرّفُ لا يتبدّل.**
	//
	// **ولا يُخزَّن بهما شيءٌ اليوم** — **وطلبُ التوسّع في دفعةٍ قادمة**،
	// **وهذا ما تُبنى عليه هويّتُه.**
	CityID        string `json:"city_id,omitempty"`
	GovernorateID string `json:"governorate_id,omitempty"`
}

// Gates **ما قرأته البوّابةُ قبلَ هذا** — **يُمرَّر ولا يُقرأ ثانيةً.**
//
// **ونداءان لمصدرٍ واحدٍ في الطلب نفسِه قد يفترقان** إن تبدّل الإعدادُ
// بينهما — **والأهمُّ أنّ الشرحَ يجب أن يصف ما منع، لا ما قد يمنع.**
type Gates struct {
	// LaunchOpen **أبوابُ الإطلاق المعنيّةُ مفتوحةٌ كلُّها؟**
	LaunchOpen bool
	// PlatformAvailable **أالمنصّةُ تستقبل الآن؟** (إيقافٌ مؤقّتٌ ودوام).
	PlatformAvailable bool
	// PlatformReason **سببُ منعِ المنصّة** حين تمنع.
	PlatformReason string
	// PlatformMessage **نصُّ المالك** إن كان.
	PlatformMessage string
}

// AvailabilityAt **القرارُ كلُّه في موضعٍ واحد.**
//
// **والترتيبُ من الأعمّ إلى الأخصّ** — **ويُعرَض أوّلُ سببٍ يمنع**:
//
//	١ · وضعُ الإطلاق        بابٌ لم يُفتح بعد — ولا موعدَ له
//	٢ · إيقافٌ مؤقّتٌ         ثمّ دوامُ المنصّة
//	٣ · صحّةُ النقطة
//	٤ · الجغرافيا الإداريّة  محافظةٌ أو مدينةٌ لم تُطلَق
//	٥ · صلاحيّةُ التغطية     إعدادٌ غائبٌ أو معطوب
//	٦ · شكلُ التغطية         العنوانُ خارجَها
//	٧ · وقتُ المنطقة
//	٨ · دوامُ المتجر
//
// **ولا يحجب الأخصُّ ما هو أعمُّ منه** — **ومن قيل له «المتجرُ مغلق»
// والمنصّةُ لم تُفتح بعدُ انتظر فتحَ متجرٍ لن ينفعه.**
func (s *Service) AvailabilityAt(ctx context.Context, q dbtx.Querier,
	g Gates, items []ItemInput, lat, lng float64) (Availability, error) {

	// ١ · **وضعُ الإطلاق** — **ولا موعدَ لبابٍ لم يُفتح بعد.**
	if !g.LaunchOpen {
		return Availability{Reason: ReasonLaunchClosed, OrderCode: "launch_closed"}, nil
	}

	// ٢ · **المنصّةُ** — كما قرأتها البوّابةُ نفسُها.
	if !g.PlatformAvailable {
		out := Availability{
			Reason:    reasonOfPlatform(g.PlatformReason),
			OrderCode: g.PlatformReason,
			Message:   g.PlatformMessage,
		}
		// **وموعدُ العودة يُحسَب بكلّ القيود لا بقيدِ المنصّة وحدَه.**
		out.NextAvailableAt = s.nextAllGates(ctx, q, items, lat, lng)
		return out, nil
	}

	// ٣ · **صحّةُ النقطة قبل سؤال القاعدة** — **ونقطةٌ مشوَّهةٌ تُقال
	// باسمها لا بـ«خارج نطاق التوصيل».**
	if !ValidPoint(lat, lng) {
		return Availability{Reason: ReasonInvalidLocation, OrderCode: "bad_point"}, nil
	}

	// ٤ · **الجغرافيا الإداريّة** — **«لم نصل إلى محافظتك» غيرُ
	// «عنوانُك خارجَ النطاق»**، **والرسالتان لا تتبادلان.**
	place, err := s.classifyPlace(ctx, q, lat, lng)
	if err != nil {
		return Availability{}, err
	}
	if place.Reason != "" {
		return Availability{
			Reason: place.Reason,
			// **ورمزُ الإنشاء = اسمُ السبب** (Batch 3a): الإنشاءُ صار يردّ
			// `province_not_supported`/`city_not_supported`/`area_not_supported`،
			// فيُرسَل هنا ليُقارَن، **فلا يفترق زرُّ الشاشة عن رفض الخادم.**
			OrderCode:     place.Reason,
			PlaceName:     place.Name,
			CityID:        place.CityID,
			GovernorateID: place.GovID,
		}, nil
	}

	// ٥ · و٦ · **التغطيةُ وشكلُها** — **بالمصدر الحاكم نفسِه.**
	zone, zerr := s.ZoneAt(ctx, q, lat, lng)
	switch {
	case zerr == ErrCoverageUnavailable:
		return Availability{
			Reason:    ReasonCoverageUnavailable,
			OrderCode: "coverage_unavailable",
		}, nil
	case zerr == ErrOutOfZone:
		return Availability{
			Reason:        ReasonAddressOutsideCoverage,
			OrderCode:     "out_of_zone",
			PlaceName:     place.Name,
			CityID:        place.CityID,
			GovernorateID: place.GovID,
		}, nil
	case zerr != nil:
		return Availability{}, zerr
	}

	// ٧ · **وقتُ المنطقة** — **وللمنطقة التي سُعِّرت لا لسواها.**
	if zone.HoursEnforced && zone.ID != "" && s.zoneHours != nil {
		open, _, err := s.zoneHours.ZoneOpen(ctx, q, zone.ID)
		if err == nil && !open {
			return Availability{
				Reason:          ReasonZoneClosedNow,
				OrderCode:       "zone_closed_now",
				PlaceName:       zone.Name,
				NextAvailableAt: s.nextAllGates(ctx, q, items, lat, lng),
			}, nil
		}
	}

	// ٨ · **دوامُ المتجر** — **بالنصّ القائم لا بحسبةٍ ثانية.**
	if len(items) > 0 {
		shut, name, err := s.closedSource(ctx, q, items)
		if err != nil {
			return Availability{}, err
		}
		if shut {
			return Availability{
				Reason:          ReasonMerchantClosedNow,
				OrderCode:       "merchant_closed",
				PlaceName:       name,
				NextAvailableAt: s.nextAllGates(ctx, q, items, lat, lng),
			}, nil
		}
	}

	return Availability{Available: true, Reason: ReasonAvailable}, nil
}

// reasonOfPlatform يترجم رمزَ منعِ المنصّة إلى اسم السبب.
func reasonOfPlatform(code string) string {
	if code == "temporarily_unavailable" {
		return ReasonTemporarilyUnavailable
	}
	return ReasonPlatformClosedNow
}

// ══════════════════════════════════════════════════════════════════════
// **الجغرافيا الإداريّة — ما يُعرَف وما لا يُعرَف**
// ══════════════════════════════════════════════════════════════════════
//
// **والمدنُ وحدَها تحمل هندسة** (`center` و`radius_m`) — **والمحافظاتُ
// اسمٌ ورايةٌ لا شكلَ لها**، **والمدينةُ تُنسَب إلى منطقةٍ إداريّةٍ
// وتلك إلى محافظة** (وقد لا تُنسَب بعد).
//
// **فما يُعرَف يُسمّى، وما لا يُعرَف لا يُخترَع له اسم.**

// placeVerdict حكمُ الجغرافيا الإداريّة.
type placeVerdict struct {
	// Reason فارغٌ يعني «أُطلقت الخدمةُ هنا — امضِ إلى التغطية».
	Reason string
	// Name اسمُ المدينة أو المحافظة حين يُعرَف.
	Name string
	// CityID و GovID **هويّةٌ ثابتةٌ يُبنى عليها** — لا اسمٌ يتبدّل.
	CityID string
	GovID  string
}

// classifyPlace **أأُطلقت الخدمةُ في موضع هذا العنوان؟**
func (s *Service) classifyPlace(ctx context.Context, q dbtx.Querier,
	lat, lng float64) (placeVerdict, error) {

	var (
		cityID   string
		cityName string
		cityOn   bool
		govID    *string
		govName  *string
		govOn    *bool
	)
	// **وأقربُ مدينةٍ تحويه** — **وهي قاعدةُ `city_filter` نفسُها**:
	// **مركزٌ ونصفُ قطر، والأقربُ يفوز عند التداخل.**
	err := q.QueryRow(ctx, `
		SELECT c.id::text, c.name, c.active, g.id::text, g.name, g.active
		FROM cities c
		LEFT JOIN districts d    ON d.id = c.district_id
		LEFT JOIN governorates g ON g.id = d.governorate_id
		WHERE ST_DWithin(c.center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, c.radius_m)
		ORDER BY ST_Distance(c.center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography)
		LIMIT 1`, lat, lng).Scan(&cityID, &cityName, &cityOn, &govID, &govName, &govOn)
	if err != nil {
		// **ولا مدينةَ تحويه** — **ولا محافظةَ تُسمّى له**: المحافظاتُ
		// بلا هندسة. **فيُقال «لم نصل بعد» ولا يُسمّى ما لا يُعرَف.**
		//
		// **وعطبُ قراءةٍ يُقرأ كذلك** — **ولا يُفتح بابٌ لأنّ السؤالَ
		// تعذّر**: البوّابةُ تردّ على كلّ حال.
		return placeVerdict{Reason: ReasonAreaNotSupported}, nil
	}

	// **والمحافظةُ المُطفأةُ تسبق المدينة** — **وهي الأعمّ**: **ومن
	// قيل له «مدينتُك لم تُطلَق» والمحافظةُ كلُّها مُطفأةٌ ظنّ الأمرَ
	// أضيقَ ممّا هو.**
	out := placeVerdict{Name: cityName, CityID: cityID}
	if govID != nil {
		out.GovID = *govID
	}
	if govOn != nil && !*govOn {
		out.Reason = ReasonProvinceNotSupported
		if govName != nil {
			out.Name = *govName
		}
		return out, nil
	}
	if !cityOn {
		out.Reason = ReasonCityNotSupported
		return out, nil
	}
	return out, nil
}

// ══════════════════════════════════════════════════════════════════════
// **دوامُ المتجر — بالنصّ القائم لا بحسبةٍ ثانية**
// ══════════════════════════════════════════════════════════════════════

// closedSource **أفي مصادر السلّة متجرٌ مغلقٌ الآن؟** — واسمُه.
//
// **وبـ`OpenNowSQL` عينِه الذي يمنع عند الإنشاء** — **ونصٌّ ثانٍ
// يفترق يوماً فيقول الشرحُ «مفتوح» ويردّ الإنشاءُ «مغلق».**
func (s *Service) closedSource(ctx context.Context, q dbtx.Querier,
	items []ItemInput) (bool, string, error) {
	src, err := s.SourcesOf(ctx, q, items)
	if err != nil {
		// **وسلّةٌ لا تُقرأ ليست حالَ إتاحة** — يُترك الحكمُ للإنشاء.
		return false, "", nil
	}
	var name string
	err = q.QueryRow(ctx, `
		SELECT m.name FROM merchants m
		WHERE m.id = ANY($1::uuid[]) AND NOT `+OpenNowSQL+`
		LIMIT 1`, src.IDs).Scan(&name)
	if err != nil {
		return false, "", nil
	}
	return true, name, nil
}

// merchantGates **قيودُ دوامِ مصادر السلّة** — لتقاطع المواعيد.
//
// **ولا صفوفَ دوامٍ تعني مفتوحاً دائماً** — **وهو ما يقوله
// `OpenNowSQL` حرفاً**، **ومن نسيه هنا جعل متجراً بلا جدولٍ مغلقاً
// أبداً فلا يتقاطع موعدٌ مع شيء.**
//
// **والإغلاقُ الطارئُ لا موعدَ لرفعه** — **فمتجرٌ مُغلَقٌ طارئاً يُعاد
// بقيدٍ لا يُفتح أبداً**، **وهو الصدق: لا نعرف متى يعود.**
func (s *Service) merchantGates(ctx context.Context, q dbtx.Querier,
	items []ItemInput) []platform.Gate {
	if len(items) == 0 {
		return nil
	}
	src, err := s.SourcesOf(ctx, q, items)
	if err != nil {
		return nil
	}
	out := []platform.Gate{}
	for _, id := range src.IDs {
		var emergency bool
		var status string
		if err := q.QueryRow(ctx,
			`SELECT emergency_closed, status FROM merchants WHERE id = $1::uuid`,
			id).Scan(&emergency, &status); err != nil {
			continue
		}
		if emergency || status != "active" {
			// **قيدٌ لا يُفتح** — جدولٌ سارٍ بلا فترةٍ واحدة.
			out = append(out, platform.Gate{Enforced: true, Sch: platform.Schedule{}})
			continue
		}
		rows, err := q.Query(ctx, `
			SELECT day_of_week,
			       EXTRACT(hour FROM open_time)::int * 60 + EXTRACT(minute FROM open_time)::int,
			       EXTRACT(hour FROM close_time)::int * 60 + EXTRACT(minute FROM close_time)::int
			FROM merchant_hours
			WHERE merchant_id = $1::uuid AND NOT closed
			ORDER BY day_of_week, open_time`, id)
		if err != nil {
			continue
		}
		sch := platform.Schedule{}
		any := false
		for rows.Next() {
			var w platform.Window
			if err := rows.Scan(&w.Day, &w.Start, &w.End); err != nil {
				break
			}
			sch = append(sch, w)
			any = true
		}
		rows.Close()
		var total int
		if err := q.QueryRow(ctx,
			`SELECT count(*) FROM merchant_hours WHERE merchant_id = $1::uuid`,
			id).Scan(&total); err != nil {
			continue
		}
		if total == 0 {
			// **بلا صفوفٍ ⇒ مفتوحٌ دائماً** — ولا يقيّد التقاطع.
			continue
		}
		_ = any
		out = append(out, platform.Gate{Enforced: true, Sch: sch})
	}
	return out
}

// nextAllGates **أوّلُ لحظةٍ تتقاطع فيها القيودُ كلُّها** — و`nil` إن
// لم تُعرَف.
//
// **ولا تُنادى إلّا عند المنع** — **فمسارُ القبول لا يدفع ثمنَ سؤالٍ
// لا يُعرَض.**
func (s *Service) nextAllGates(ctx context.Context, q dbtx.Querier,
	items []ItemInput, lat, lng float64) *time.Time {
	snap, ok := s.snapshotFor(ctx, q, lat, lng)
	if !ok {
		return nil
	}
	gates := []platform.Gate{snap.Platform, snap.Zone}
	gates = append(gates, s.merchantGates(ctx, q, items)...)
	return platform.NextAllOpen(snap.Now, snap.Closure, gates...)
}

// snapshotFor يقرأ قيودَ المنصّة ومنطقةِ هذا العنوان.
func (s *Service) snapshotFor(ctx context.Context, q dbtx.Querier,
	lat, lng float64) (platform.Snapshot, bool) {
	snaps, ok := s.zoneHours.(interface {
		Snapshot(context.Context, dbtx.Querier, string) (platform.Snapshot, error)
	})
	if !ok {
		return platform.Snapshot{}, false
	}
	zoneID := ""
	if ValidPoint(lat, lng) {
		if z, err := s.ZoneAt(ctx, q, lat, lng); err == nil {
			zoneID = z.ID
		}
	}
	snap, err := snaps.Snapshot(ctx, q, zoneID)
	if err != nil {
		return platform.Snapshot{}, false
	}
	return snap, true
}
