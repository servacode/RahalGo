package server

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ تغطيةِ المنطقة الحيّ — بنيةُ اختبارٍ على التجهيز وحدَه** (Batch 3d)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٥: «Staging QA لا يملك مصادقةَ أدمن، فأنشئ قدرةً
//  ضيّقةً جدًّا تمرّ بنفسِ مسارِ خدمةِ المناطق الحقيقيّ لينطلقَ
//  `notifyAreaCoverage` الحقيقيّ — لا حقنَ SQL بديلاً عن الشاهد».)
//
// # لماذا هذه القدرةُ آمنةٌ — **بنيةُ اختبارٍ لا تغييرُ منتَج**
//
// **لا تُصدِر توكن أدمن، ولا تقبل معرّفَ منطقةٍ من الطلب، ولا تُنشئ إلّا
// منطقةً واحدةً باسمٍ ثابتٍ حول نقطةٍ ثابتة.** **وتسقط مغلقةً في الإنتاج**
// (المنادي `handleQAStagingSeed` يحرس `qaStagingEnabled`، وهنا حارسٌ ثانٍ
// في كلّ معالج). **والتعديلُ والحذفُ بالاسم الثابت لا بمعرّفٍ يأتي من
// الخارج** — فلا تمسّ منطقةً أخرى مهما كان الطلب.
//
// **وتمرّ بنفسِ `catalog.CreateZone/UpdateZone/DeleteZone` ثمّ
// `notifyAreaCoverage`** — نفسُ ما يفعله بابُ الأدمن الحقيقيّ
// (`admin_zones_handlers.go`)، **فالشاهدُ حقيقيٌّ لا مُصطنَعٌ بحقنِ صفٍّ.**

const (
	// qaCoverageZoneName **الاسمُ الثابتُ الوحيد** — لا تُنشأ ولا تُعدَّل ولا
	// تُحذَف منطقةٌ بغيره. **حارسُ «لا منطقةَ عشوائيّة».**
	qaCoverageZoneName   = "QA-3d-coverage-test"
	qaCoverageZoneLat    = 36.0006
	qaCoverageZoneLng    = 39.0594
	qaCoverageZoneRadius = 500
)

// qaCoverageActor **هويّةُ أدمن ثابتةٌ للتدقيق — لا توكن.** معرّفُ موظّفٍ
// قائمٍ يُكتب كاتبَ الأثر (audit) فحسب، لا جلسةٌ ولا صلاحيّة.
func (s *Server) qaCoverageActor(ctx context.Context) string {
	id, _ := s.qaNonCustomerAuthor(ctx, "00000000-0000-0000-0000-000000000000")
	return id
}

// qaCoverageZoneID **يجد المنطقةَ الاختباريّةَ بالاسم الثابت وحدَه** —
// **فلا يمسّ أيَّ منطقةٍ أخرى مهما كان.** فارغٌ ⇒ غيرُ موجودة.
func (s *Server) qaCoverageZoneID(ctx context.Context) (string, error) {
	zones, err := s.catalog.ListZones(ctx)
	if err != nil {
		return "", err
	}
	for i := range zones {
		if zones[i].Name == qaCoverageZoneName {
			return zones[i].ID, nil
		}
	}
	return "", nil
}

// qaCoverageZoneCreate يُنشئ منطقةَ التغطيةِ الاختباريّةَ الثابتةَ **عبر خدمةِ
// الأدمن الحقيقيّة** ثمّ يُطلق `notifyAreaCoverage` الحقيقيّ.
func (s *Server) qaCoverageZoneCreate(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	if id, err := s.qaCoverageZoneID(ctx); err != nil {
		s.respondErr(w, err)
		return
	} else if id != "" {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_coverage_zone_exists", "errors.conflict"))
		return
	}
	actor := s.qaCoverageActor(ctx)
	name := qaCoverageZoneName
	lat, lng := float64(qaCoverageZoneLat), float64(qaCoverageZoneLng)
	radius := qaCoverageZoneRadius
	active := true
	fee := int64(100)
	minOrder := int64(0)
	z, err := s.catalog.CreateZone(ctx, actor, catalog.ZoneInput{
		Name: &name, Lat: &lat, Lng: &lng, RadiusM: &radius,
		DeliveryFee: &fee, MinOrder: &minOrder, Active: &active,
	}, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **تُضمَن الفعّاليّة** — `CreateZone` لا يكتب عمودَ `active`، فيُثبَّت
	// بـ`UpdateZone` إن لزم (المسارُ نفسُه الذي يستعمله الأدمن).
	if !z.Active {
		z, err = s.catalog.UpdateZone(ctx, actor, z.ID, catalog.ZoneInput{Active: &active}, clientIP(r))
		if err != nil {
			s.respondErr(w, err)
			return
		}
	}
	// **الإشعارُ الحقيقيّ** — نفسُ نداءِ `handleCreateZone`.
	s.notifyAreaCoverage(ctx)
	s.logger.Warn("QA coverage test zone created (staging-only)", "zone", z.ID, "active", z.Active)
	httpx.JSON(w, http.StatusOK, map[string]any{"zone_id": z.ID, "active": z.Active, "name": z.Name, "kind": "coverage_zone_create"})
}

// qaCoverageZoneResave يُعيد حفظَ **نفسِ** المنطقةِ الاختباريّةِ (بالاسم) ثمّ
// يُطلق `notifyAreaCoverage` — لشهودِ «لا إشعارَ مكرّر».
func (s *Server) qaCoverageZoneResave(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	id, err := s.qaCoverageZoneID(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if id == "" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	actor := s.qaCoverageActor(ctx)
	name := qaCoverageZoneName
	lat, lng := float64(qaCoverageZoneLat), float64(qaCoverageZoneLng)
	radius := qaCoverageZoneRadius
	active := true
	z, err := s.catalog.UpdateZone(ctx, actor, id, catalog.ZoneInput{
		Name: &name, Lat: &lat, Lng: &lng, RadiusM: &radius, Active: &active,
	}, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **الإشعارُ الحقيقيّ** — نفسُ نداءِ `handleUpdateZone`.
	s.notifyAreaCoverage(ctx)
	s.logger.Warn("QA coverage test zone resaved (staging-only)", "zone", z.ID)
	httpx.JSON(w, http.StatusOK, map[string]any{"zone_id": z.ID, "active": z.Active, "kind": "coverage_zone_resave"})
}

// qaCoverageZoneDelete يحذف **نفسَ** المنطقةِ الاختباريّةِ (بالاسم) وحدَها —
// تنظيفٌ يُعيد الحالَ إلى ما قبلَ الشاهد.
func (s *Server) qaCoverageZoneDelete(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	id, err := s.qaCoverageZoneID(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if id == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"deleted": false, "note": "no such zone"})
		return
	}
	if err := s.catalog.DeleteZone(ctx, s.qaCoverageActor(ctx), id, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA coverage test zone deleted (staging-only)", "zone", id)
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true, "zone_id": id, "kind": "coverage_zone_delete"})
}
