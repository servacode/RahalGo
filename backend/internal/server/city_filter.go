package server

import (
	"context"
	"net/http"
	"strconv"
)

// ══════════════════════════════════════════════════════════════════════
// **ما يراه الزبونُ من سوقِ مدينته وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «مو معقول شخصٌ بالشام يطلب من الرقّة… ولا
//
//	زبونٌ بأوّل الشام من مطعمٍ بآخر الشام».)
//
// # وما كان يقع قبل هذا
//
// **الدمشقيُّ يرى متاجرَ الرقّة كلَّها ويملأ سلّتَه** — **ثمّ يُردّ عند
// الإرسال بـ`out_of_zone`.** وهذا أسوأُ من المنع: أضاع وقتَه، **وقرأها
// عطباً في التطبيق لا حدّاً للخدمة.**
//
// # وطبقتان
//
//	المدينةُ  ←  أيَّ سوقٍ يرى     (دمشقُ لا حلب)
//	المسافةُ  ←  أيَّ متجرٍ يصله   (المزّةُ لا جرمانا)
//
// **والمدينةُ وحدَها لا تكفي في مدينةٍ كبيرة.**
//
// # ومدىً متدرّجٌ ثلاثيّ — يعمل بلا ضبط
//
//	مدى المتجر  إن ضُبط  (استثناءٌ نادر)
//	  ↓ وإلّا مدى المدينة إن ضُبط  (دمشقُ رقمٌ واحد)
//	    ↓ وإلّا افتراضُ المنصّة `delivery.default_radius_m`
//	      ↓ وصفرٌ يعني بلا حدّ  (الرقّةُ لا تحتاج شيئا)
//
// # ولا موقعَ يعني لا ترشيح
//
// **وعميلٌ قديمٌ لا يرسل موقعَه يرى ما كان يراه** — **وترشيحٌ يُفرض على
// من لا يعرف عنه يُخفي السوقَ كلَّه عن نسخةٍ قديمة.**

// geoScope **موضعُ الزبون كما أرسله** — وفارغٌ يعني «لا ترشيح».
type geoScope struct {
	lat, lng float64
	on       bool
}

// scopeFrom **يقرأ الموضعَ من النداء.**
//
// **ويُقرأ من الاستعلام لا من الحساب**: الضيفُ يتصفّح بلا حساب،
// **وعنوانُه المحفوظُ قد لا يكون مدينتَه اليوم** — من يزور أهلَه يطلب
// من حيث هو.
func scopeFrom(r *http.Request) geoScope {
	q := r.URL.Query()
	lat, e1 := strconv.ParseFloat(q.Get("lat"), 64)
	lng, e2 := strconv.ParseFloat(q.Get("lng"), 64)
	if e1 != nil || e2 != nil || (lat == 0 && lng == 0) {
		return geoScope{}
	}
	// **وإحداثيٌّ خارجَ الأرض يُهمَل ولا يُردّ** — تصفّحٌ لا فعل،
	// **ورفضُ التصفّح على قراءةِ موقعٍ سيّئةٍ يُقرأ عطبا.**
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return geoScope{}
	}
	return geoScope{lat: lat, lng: lng, on: true}
}

// cityWhere **شرطُ المدينةِ والمدى** — يُضاف إلى استعلام التصفّح.
//
// **ويردّ فارغاً حين لا موضع** — فيبقى الاستعلامُ كما كان حرفاً بحرف.
//
// # ولماذا الشرطُ نصٌّ يُضاف لا جدولٌ يُضمّ
//
// **الضمُّ يغيّر عددَ الصفوف إن تعدّدت المدنُ حول نقطة** — **وشرطٌ في
// `WHERE` لا يضاعف شيئا.**
func (s *Server) cityWhere(ctx context.Context, g geoScope, argN int) (string, []any) {
	if !g.on {
		return "", nil
	}
	// **وافتراضُ المنصّةِ يُقرأ مرّةً هنا** — لا في كلّ صفّ.
	def := s.settings.GetInt(ctx, "delivery.default_radius_m")
	// **والمدينةُ أقربُ مدينةٍ تحويه** — ولا تُقرأ بيدٍ ولا تُخمَّن.
	//
	// **ومن وقع خارجَ كلّ مدينةٍ لا يرى شيئا**: `city_id` لا يطابق،
	// **وهو الصواب** — لا سوقَ له بعد.
	cond := `
	  AND m.city_id = (
	      SELECT c.id FROM cities c
	       WHERE c.active
	         AND ST_DWithin(c.center, ST_SetSRID(ST_MakePoint($` +
		itoa(argN+1) + `, $` + itoa(argN) + `), 4326)::geography, c.radius_m)
	       ORDER BY ST_Distance(c.center, ST_SetSRID(ST_MakePoint($` +
		itoa(argN+1) + `, $` + itoa(argN) + `), 4326)::geography)
       LIMIT 1)
	  AND (
	      COALESCE(
	          m.max_delivery_m,
	          (SELECT c.max_delivery_m FROM cities c WHERE c.id = m.city_id),
	          $` + itoa(argN+2) + `
	      ) = 0
	   OR ST_DWithin(
	          m.location,
	          ST_SetSRID(ST_MakePoint($` + itoa(argN+1) + `, $` + itoa(argN) + `), 4326)::geography,
	          COALESCE(
	              m.max_delivery_m,
	              (SELECT c.max_delivery_m FROM cities c WHERE c.id = m.city_id),
	              $` + itoa(argN+2) + `
	          )
	      )
	  )`
	return cond, []any{g.lat, g.lng, def}
}

func itoa(n int) string { return strconv.Itoa(n) }

// inScope **أيُّ هذه المتاجر يصلُ هذا الموضعَ** — للقوائم التي تُبنى في
// الذاكرة لا في استعلامٍ واحد.
//
// **والعروضُ منها**: تُقرأ من حزمةٍ لا تعرف الجغرافيا، **وتمريرُ نصِّ
// شرطٍ بين الحزم بابٌ يُفتح ولا يُغلق.** فيُسأل عن المعرّفات وحدَها.
//
// **ويردّ `nil` حين لا موضع** — والمنادي يقرؤها «مرِّر الكلّ».
func (s *Server) inScope(ctx context.Context, g geoScope, ids []string) map[string]bool {
	if !g.on || len(ids) == 0 {
		return nil
	}
	where, args := s.cityWhere(ctx, g, 2)
	rows, err := s.pg.Query(ctx, `
		SELECT m.id::text FROM merchants m
		 WHERE m.id = ANY($1::uuid[])`+where, ids, args[0], args[1], args[2])
	if err != nil {
		// **وخطأٌ هنا لا يحجب السوق** — ردٌّ فارغٌ يُقرأ «لا عروضَ عندكم»
		// وهو كذب. **والحجبُ الصامتُ أسوأُ من ترشيحٍ لم يقع.**
		return nil
	}
	defer rows.Close()
	ok := map[string]bool{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ok[id] = true
		}
	}
	return ok
}
