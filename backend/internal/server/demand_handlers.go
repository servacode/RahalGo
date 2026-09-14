package server

// ══════════════════════════════════════════════════════════════════════
// **طلبُ التوسّع و«أخبرني» — والخادمُ يُعيد الحكمَ** (`CR`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا يُصدَّق سببٌ يرسله عميل
//
// **وعميلٌ معدَّلٌ يرسل «عنواني خارجَ التغطية» لعنوانٍ يُخدَم** —
// **فيُلوَّث دفترُ الطلب بإشاراتٍ كاذبة**، **ويُبنى على كثافةٍ مخترَعةٍ
// قرارُ توسّع.**
//
// **فتُعاد قراءةُ الإتاحة هنا بالمحرّك عينِه** (`AvailabilityAt`) —
// **وما تقوله هي الحقيقةُ لا ما في الجسم.**
//
// # وسباقُ الحال
//
// **ورأى الزبونُ «خارجَ التغطية» ثمّ وسّع المكتبُ المنطقةَ قبل أن
// يضغط** — **فلا تُسجَّل إشارةُ طلبٍ لمكانٍ صار مخدوماً.** **ويُردّ
// بحالٍ يقرؤها التطبيقُ فيُحدِّث نفسَه.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/opsmap"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// demandMaxPerHour **سقفُ إشاراتِ حسابٍ في الساعة.**
//
// **وستّون ضغطةً في ساعةٍ أكثرُ ممّا يفعل إنسانٌ يبحث عن خدمة** —
// **وأقلُّ من أن يزرع بها أحدٌ دفترَ طلبٍ كاذباً.**
const demandMaxPerHour = 60

// ErrServiceNowAvailable **صار المكانُ مخدوماً بين القراءة والضغط.**
//
// **و٢٠٩ ليست حالاً** — **فيُردّ ٤٠٩ «تعارضُ حال»**: **الطلبُ صحيحُ
// الشكل، والعالمُ تبدّل تحته.**
var ErrServiceNowAvailable = httpx.NewError(http.StatusConflict,
	"service_now_available", "errors.service_now_available")

// ErrReasonMismatch **نيّةٌ لا توافق حالَ المكان.**
//
// **ومن طلب «أضِف منطقتي» ونحن لم نصل مدينتَه أصلاً طلب ما لا يقع** —
// **والصوابُ «أخبرني».**
var ErrReasonMismatch = httpx.NewError(http.StatusConflict,
	"reason_mismatch", "errors.reason_mismatch")

// demandKindFor **أيُّ نيّةٍ تصلح لهذا السبب؟** — وفارغٌ يعني لا نيّةَ
// جغرافيّةً أصلاً.
//
// ══════════════════════════════════════════════════════════════════════
// **والأسبابُ الزمنيّةُ ليست أسبابَ توسّع**
// ══════════════════════════════════════════════════════════════════════
//
// **ومنصّةٌ خارجَ دوامها ومنطقةٌ خارجَ وقتها ومتجرٌ مغلقٌ** — **كلُّها
// تعود بعد ساعات.** **ومن عُرض عليه «اطلب إضافة منطقتك» لأنّ المتجرَ
// مغلقٌ ظنّ أنّنا لا نصله.**
//
// **و`coverage_unavailable` عطبُ إعدادٍ في اللوحة** — **ولا يُدعى
// الناسُ إلى طلب مناطقهم بسببه** (البند ٢٥).
func demandKindFor(reason string) string {
	switch reason {
	case orders.ReasonAddressOutsideCoverage:
		return opsmap.KindCoverage
	case orders.ReasonCityNotSupported,
		orders.ReasonProvinceNotSupported,
		orders.ReasonAreaNotSupported:
		return opsmap.KindInterest
	default:
		return ""
	}
}

// demandGates يقرأ بوّابتَي الإطلاق والمنصّة كما يقرؤهما المنع.
func (s *Server) demandGates(r *http.Request) orders.Gates {
	g := orders.Gates{
		LaunchOpen: s.launchOpen(r.Context(), launchCustomerOrders) &&
			s.launchOpen(r.Context(), launchMerchantOrders),
		PlatformAvailable: true,
	}
	if st, err := s.platform.State(r.Context(), s.pg); err == nil {
		g.PlatformAvailable = st.OrderingAvailable
		g.PlatformReason = string(st.Reason)
		g.PlatformMessage = st.Message
	}
	return g
}

// handleDemandSignal **يسجّل نيّةَ توسّعٍ بعد أن يتحقّق منها.**
func (s *Server) handleDemandSignal(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Address string  `json:"address_text"`
		// Kind **نيّةُ العميل** — **تُقارَن بما يقضيه الخادمُ ولا
		// تُصدَّق.**
		Kind string `json:"kind"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **والتفرّدُ ليس حارسَ إساءة** (البند ٢١)
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومن ضغط الزرَّ في الحيّ نفسِه ألفَ مرّةٍ لا يُولَد صفٌّ ثانٍ** —
	// **لكنّه يُشغّل حسبةَ إتاحةٍ ألفَ مرّة.** **وحسابٌ واحدٌ يستطيع أن
	// يزرع ملايينَ الإحداثيّات المختلفة، وكلُّ واحدةٍ خليّةٌ جديدة.**
	//
	// **فيُحَدّ بالحساب لا بالعنوان وحدَه** — **والعدّادُ بالساعة
	// كأخويه في هذا الملفّ.**
	limitKey := "demand:" + userIDFrom(r)
	if userIDFrom(r) == "" {
		limitKey = "demand:ip:" + clientIP(r)
	}
	if n, err := s.incr(r.Context(), limitKey); err == nil {
		if n == 1 && s.rdb != nil {
			s.rdb.Expire(r.Context(), limitKey, time.Hour)
		}
		if n > demandMaxPerHour {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests,
				"rate_limited", "errors.rate_limited"))
			return
		}
	}

	// **والحكمُ يُعاد هنا** — **ولا سببَ يُقرأ من الجسم.**
	av, err := s.orders.AvailabilityAt(r.Context(), s.pg,
		s.demandGates(r), nil, in.Lat, in.Lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if av.Available {
		s.respondErr(w, ErrServiceNowAvailable)
		return
	}
	kind := demandKindFor(av.Reason)
	if kind == "" {
		// **وسببٌ زمنيٌّ أو عطبُ إعدادٍ لا يُسجَّل إشارةَ توسّع.**
		s.respondErr(w, ErrReasonMismatch)
		return
	}
	// **ونيّةٌ مُرسَلةٌ تخالف ما قضاه الخادمُ تُردّ** — **ولا تُقلَب
	// صامتةً**: **من ضغط «أخبرني» ووجد «سجّلنا طلبَ منطقتك» ارتاب.**
	if in.Kind != "" && in.Kind != kind {
		s.respondErr(w, ErrReasonMismatch)
		return
	}

	res, err := opsmap.Record(r.Context(), s.pg, userIDFrom(r), opsmap.Signal{
		Kind:    kind,
		Lat:     in.Lat,
		Lng:     in.Lng,
		Address: in.Address,
		Source:  "customer_app",
		Reason:  av.Reason,
		CityID:  av.CityID,
		GovID:   av.GovernorateID,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"kind":     kind,
		"outcome":  res.Outcome,
		"requests": res.Requests,
		"reason":   av.Reason,
	})
}

// handleDemandCancel **يُلغي اشتراكَ «أخبرني» لهذا المكان.**
//
// **ولا يُمحى طلبُ التغطية** — **وهو واقعةٌ تاريخيّةٌ تبقى محسوبةً في
// الكثافة**، **وإنّما يخرج صاحبُ الاشتراك من دائرة من يُخبَر.**
func (s *Server) handleDemandCancel(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := opsmap.CancelInterest(r.Context(), s.pg, userIDFrom(r), in.Lat, in.Lng); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"active": false})
}

// handleDemandByPlace **كثافةُ الطلب بالمدينة والمحافظة** — للوحة.
func (s *Server) handleDemandByPlace(w http.ResponseWriter, r *http.Request) {
	rows, err := opsmap.DemandByPlace(r.Context(), s.pg, r.URL.Query().Get("kind"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"places": rows})
}
