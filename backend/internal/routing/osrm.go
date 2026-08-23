// Package routing يسأل محرّكَ المسارات: كم الطريقُ بين نقطتين، وكيف يمرّ.
//
// ══════════════════════════════════════════════════════════════════════
// **والخطُّ المستقيمُ كان يكذب**
// ══════════════════════════════════════════════════════════════════════
//
// (أمسكه المالك ٢٠٢٦-٠٨-١٢: «هذه لا تحترم الشوارع ولا أيّ شيء، ترسم خطّاً
//
//	مستقيماً فقط من فوق البيوت وكأنّ المكان فارغ».)
//
// **وقيس على طلبٍ حقيقيّ** (#1005، الرقّة): خطُّنا المستقيم قال ٩٨٢ متراً
// ودقيقتين، **وقالت خرائطُ غوغل ١٫١ كم وسبعَ دقائق** عبر شارع تل أبيض.
// **الوقتُ أقلُّ من الحقيقة ثلاثَ مرّاتٍ ونصفاً** — والزبونُ ينتظر على وعدٍ
// لا يقع، والسائقُ يوازن بين طلبين برقمٍ لا يعني شيئاً.
//
// # وسقوطُه لا يُسقط المنصّة
//
// **ومن جعل رقماً ثانويّاً شرطاً للطلب** أوقف التوزيعَ كلَّه يومَ تنام
// خدمةٌ صغيرة. **فإن لم يردّ: يُقال ذلك في السجلّ ويُحسب بالخطّ المستقيم**
// كما كان.
package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Point نقطةٌ على الأرض.
type Point struct {
	Lat float64
	Lng float64
}

// Route ما يردّه المحرّك.
type Route struct {
	// DistanceM طولُ الطريق بالمتر — **بالشوارع لا بالهواء.**
	DistanceM float64
	// DurationS المدّةُ بالثواني — **محسوبةٌ بسرعات الشوارع نفسِها.**
	DurationS float64
	// Geometry نقاطُ الخطّ كما يمرّ — **لترسمه الشاشةُ على الخريطة.**
	Geometry []Point

	// ══════════════════════════════════════════════════════════════
	// **وزنُ المحرّك — المرحلة ٧، البند ٣**
	// ══════════════════════════════════════════════════════════════
	//
	// **أمرُ المالك نصّاً**: «أريد لكل Route داخليًا: distanceM ·
	// engineDurationS · engineWeight · weightName. حتى نستطيع فهم
	// لماذا أوصى المحرك بالمسار».
	//
	// **و`DurationS` هنا مدّةُ المحرّك الأصليّة** — لا المعدَّلةَ
	// بسرعة السائق. **وتلك تُحسب في `driver_route.go` للعرض وحدَه**،
	// **ولا تدخل مقارنةً ولا ترشيحاً ولا تصنيفاً** (البند ٤).
	EngineWeight float64 `json:",omitempty"`
	WeightName   string  `json:",omitempty"`

	// ══════════════════════════════════════════════════════════════════
	// **وما يلي للملاحة — المرحلة ٢**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وفارغةٌ حالٌ مشروعة**: محرّكٌ ردّ بلا خطوات، أو مسارٌ مخزّنٌ
	// بصيغةٍ قديمة. **والشاشةُ ترسم كما كانت.**

	// CumulativeM مسافةُ كلّ رأسٍ من البداية — **تُحسب مرّةً عند
	// البناء.**
	//
	// **وطولُها طولُ `Geometry`** — `CumulativeM[0] = 0`.
	//
	// **وبها يصير حسابُ المتبقّي طرحاً واحداً** بدل جمعِ آلاف القطع
	// في كلّ قراءةِ موقع.
	CumulativeM []float64 `json:",omitempty"`

	// Maneuvers مناوراتُ الطريق بمفاهيمنا — انظر `maneuver.go`.
	Maneuvers []Maneuver `json:",omitempty"`
}

// HasNavigation **أثمّةَ ما يكفي للملاحة؟**
//
// **ومسارٌ بلا مناوراتٍ يُرسم ولا يُرشِد** — وهي حالُ العميل القديم
// والمحرّكِ الذي لم يردّ خطوات.
func (r *Route) HasNavigation() bool {
	return r != nil && len(r.Maneuvers) > 0 && len(r.CumulativeM) == len(r.Geometry)
}

// Client بابُ محرّك المسارات.
//
// **وفارغُ العنوان يعني «لا محرّك»** — فيردّ ErrNoEngine ولا يُنادي شبكة.
type Client struct {
	base string
	http *http.Client

	// snap **حدُّ الالتقاط** — TD-SNAP-RADIUS، المرحلة ٨أ.
	//
	// **وهو سياسةُ خادمٍ لا خيارُ هاتف** (البند ٤): لا يصل من الطلب،
	// **ولا يُوسَّع عند الفشل** (البند ٧).
	snap SnapPolicy
}

// ErrNoEngine لا عنوانَ لمحرّك المسارات.
var ErrNoEngine = fmt.Errorf("routing: لا محرّك مسارات")

func New(baseURL string) *Client {
	return NewWithSnap(baseURL, DefaultSnapPolicy())
}

// NewWithSnap **بحدٍّ مُعطَى** — للاختبار والمعايرة.
//
// **ولا يُنادى من الخادم بغير `DefaultSnapPolicy`** — فالسياسةُ
// مركزيّةٌ واحدة، وكلُّ ما يمرّ بـ`Backend` يمرّ بها.
func NewWithSnap(baseURL string, snap SnapPolicy) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		// **ومهلةٌ قصيرة**: هذا رقمٌ يُحسّن الشاشة لا يصنعها، **وانتظارُ
		// خمسِ ثوانٍ لرسمِ خطٍّ** يُقرأ تعليقاً في التطبيق.
		http: &http.Client{Timeout: 4 * time.Second},
		snap: snap,
	}
}

// SnapPolicy **ما يُطبَّق فعلاً** — تقرؤه الاختبارات.
func (c *Client) SnapPolicy() SnapPolicy {
	if c == nil {
		return SnapPolicy{}
	}
	return c.snap
}

// Enabled أثمّة محرّكٌ مضبوط؟
func (c *Client) Enabled() bool { return c != nil && c.base != "" }

// Route يسأل عن الطريق من نقطةٍ إلى نقطة.
//
// **والإحداثيّاتُ في OSRM طولٌ ثمّ عرض** — عكسُ ما تكتبه القاعدةُ والشاشة.
// **ومن قلبها** حصل على مسارٍ في بلدٍ آخر أو على «لا مسار».
func (c *Client) Route(ctx context.Context, from, to Point) (*Route, error) {
	res, err := c.fetch(ctx, from, to, 0)
	if err != nil {
		return nil, err
	}
	return res[0], nil
}

// ══════════════════════════════════════════════════════════════════════
// **مجموعةُ المسارات — موصًى بها وبدائل**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
//
// **تردّ ما ردَّه المحرّكُ بترتيبه** — والأوّلُ هو الموصى به.
// **والترشيحُ ليس هنا**: هذا جالبٌ، و`FilterAlternatives` تقرّر.
func (c *Client) RouteSet(ctx context.Context, from, to Point) ([]*Route, error) {
	return c.fetch(ctx, from, to, EngineMaxAlternatives)
}

// ══════════════════════════════════════════════════════════════════════
// **ولا `overview` — المرحلة ٧، البند ١٩**
// ══════════════════════════════════════════════════════════════════════
//
// **أمرُ المالك نصّاً**: «لا نرسل Geometry مكررة لمجرد العادة».
//
// **والمرحلةُ ٢ تبني الهندسةَ من `steps[].geometry`** لا من
// `routes[].geometry` — انظر `buildFromSteps`. **فـ`overview` كانت
// تُطلب وتُرمى.**
//
// **وقِيس التكافؤُ على أربع عيّنات** (٢٠٢٦-٠٨-٢١، OSRM v26.8.0):
// هندسةٌ ومناوراتٌ ومسافةٌ ومدّةٌ **متطابقةٌ بين `full` و`false`**،
// **والتوفيرُ ٣٧٪ من الحمولة المضغوطة** في الطويلة — دمشق←اللاذقيّة
// بثلاثة مسارات: ١٦٧ك ← ١٠٤ك.
//
// **وشبكةُ الأمان تبقى**: لو ردَّ المحرّكُ بلا خطوات — ولم يقع في
// القياس — **يُعاد الطلبُ مرّةً واحدةً بـ`overview=full`** فتُقرأ
// الهندسةُ الإجماليّة. **فالتوفيرُ لا يشتري هشاشة.**
func (c *Client) fetch(ctx context.Context, from, to Point, alternatives int) ([]*Route, error) {
	if !c.Enabled() {
		return nil, ErrNoEngine
	}
	out, err := c.request(ctx, from, to, alternatives, "false")
	if err != nil {
		return nil, err
	}
	if len(out) > 0 && len(out[0].Geometry) >= 2 {
		return out, nil
	}
	return c.request(ctx, from, to, alternatives, "full")
}

func (c *Client) request(
	ctx context.Context, from, to Point, alternatives int, overview string,
) ([]*Route, error) {
	// **و`alternatives=1` ليست «واحداً»** — OSRM تفهمها «ابحث عن
	// بديلٍ واحدٍ زيادة». **فالمفرَدُ `false` صريحاً.**
	alt := "false"
	if alternatives > 1 {
		alt = strconv.Itoa(alternatives)
	}

	url := c.base + "/route/v1/driving/" +
		coord(from.Lng) + "," + coord(from.Lat) + ";" +
		coord(to.Lng) + "," + coord(to.Lat) +
		// ══════════════════════════════════════════════════════════════
		// **و`steps=true` منذ المرحلة ٢**
		// ══════════════════════════════════════════════════════════════
		//
		// (أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
		//
		// **بها تأتي المناوراتُ وهندستُها** — وبلاها خطٌّ يُرسم ولا
		// يُرشد.
		"?overview=" + overview + "&geometries=geojson" +
		"&alternatives=" + alt + "&steps=true"

	// ══════════════════════════════════════════════════════════════════
	// **وحدُّ الالتقاط يُرسَل — TD-SNAP-RADIUS**
	// ══════════════════════════════════════════════════════════════════
	//
	// **و`radiuses` تُرسَل في كلّ طلب**: مفردٌ وبدائلُ
	// وإعادةُ حساب — **فلا يكون طريقٌ محميّاً وآخرُ مكشوفاً**
	// (البندان ٩ و١٠). **وهذا الموضعُ هو الممرّ الوحيد.**
	if c.snap.bounded() {
		url += "&radiuses=" + c.snap.radiuses()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// ══════════════════════════════════════════════════════════════════
	// **والرفضُ يأتي بـ٤٠٠ لا بـ٢٠٠ — المرحلة ٨أ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **وهذا ما كشفه الاختبارُ الحيُّ ولم يكشفه المزيّف**:
	// OSRM يردّ `NoSegment` **بحالة ٤٠٠ وجسمٍ JSON فيه السبب**.
	//
	// **فمن خرج عند الحالة ضيّع السبب** — وصار «إحداثيّةٌ
	// خارجَ الحدّ» و«المحرّكُ معطوب» خطأً واحداً.
	//
	// **والحالاتُ الخمسمئيّةُ تبقى أعطاباً** — ولا جسمَ يُقرأ فيها.
	if res.StatusCode >= 500 {
		return nil, fmt.Errorf("routing: ردّ %d", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusBadRequest {
		return nil, fmt.Errorf("routing: ردّ %d", res.StatusCode)
	}

	var body struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			// ══════════════════════════════════════════════════════
			// **الوزنُ واسمُه — دلالةٌ محسومةٌ بالقياس**
			// ══════════════════════════════════════════════════════
			//
			// (المرحلة ٣، ثمّ حُسمت في ٨أ ٢٠٢٦-٠٨-٢١.)
			//
			// **وثلاثةٌ لا تُخلط**:
			//
			//   - `DurationS` **زمنٌ يُعرَض للسائق.**
			//   - `EngineWeight` **ما وازن به المحرّك** — وليس زمناً.
			//   - **والترتيبُ** من المحرّك، `routes[0]` هو
			//     الموصى به — **ولا نُعيد ترتيبه** (المرحلة ٧).
			//
			// # ولماذا يفترقان
			//
			// **`routability` ليست `duration`** وإن ساوتها غالباً.
			// **الوزنُ = المسافة ÷ rate**، و`rate = السرعة × غرامة`
			// (`way_handlers.lua:471`). **فحيث لا غرامةَ يتساويان.**
			//
			// **وقُيس على رفيدة المرحلة ٨** (١٨٩ استعلاماً،
			// v26.8.0، PBF تسلسل ٤٦٣٥): **يختلفان في ٤٫٨٪ من
			// المسارات الرئيسة** (٩ من ١٨٩) و٤٫٦٪ من كلّ المسارات.
			//
			// **والفرقُ كلّه في أوزان القطع لا في الدوران** —
			// قُيس بالتعليقات السطريّة: فرقُ الدوران `+0.0` في
			// التسع جميعاً. **والنسبةُ تتجمّع عند قيمٍ منفصلة**:
			//
			//	٢٫٠×  ←  service_penalties = 0.5  (parking_aisle · alley · driveway)
			//	١٫٢٥× ←  side_road_multiplier = 0.8
			//
			// **وهي ثوابتُ `car.lua` بعينها** — فالدلالةُ مقروءةٌ
			// من الملفّ الشخصيّ لا مُستنتَجة.
			//
			// **وتقريري الأوّلُ للمرحلة ٨ قال «`weight == duration`»** —
			// **وكان خطأً**: بُني على عيّنةٍ واحدة. **والمرحلةُ ٣
			// كانت مُصيبة.**
			Weight     float64 `json:"weight"`
			WeightName string  `json:"weight_name"`
			Geometry   struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
			Legs []struct {
				Steps []osrmStep `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	// **و«لا مسار» ليست خطأً في النداء**: نقطةٌ في الصحراء بلا طريقٍ إليها
	// تردّ `NoRoute` بحالة ٢٠٠. **ومن قرأ الحالةَ وحدَها** ظنّ أنّه نجح.
	if body.Code != "Ok" || len(body.Routes) == 0 {
		// ══════════════════════════════════════════════════════════════
		// **والفشلُ يُسمّى — المرحلة ٨أ، البند ٧**
		// ══════════════════════════════════════════════════════════════
		//
		// **`NoSegment` تعني: إحداثيّةٌ خارجَ الحدّ** — تثبيتٌ
		// فاسدٌ أو موضعٌ بعيدٌ عن كلّ طريق. **وتُردّ خطأً
		// مُسمًّى لا نصّاً**، فيميّزها من ناداه.
		//
		// **ولا يُعاد الطلبُ بلا حدّ** — فذلك يُعيد العيبَ
		// الصامتَ بعينه: مسارٌ واثقٌ من إحداثيّةٍ لا تعني شيئاً.
		switch body.Code {
		case "NoSegment":
			return nil, ErrNoSegment
		case "NoRoute":
			return nil, ErrNoRoute
		}
		return nil, fmt.Errorf("routing: %s", body.Code)
	}

	// **وكلُّها تُفكّ لا الأوّلُ وحدَه** — **وترتيبُ المحرّك يُحفظ**
	// (البند ١٣): لا يُعاد الترتيبُ بالمدّة ولا بالمسافة.
	all := make([]*Route, 0, len(body.Routes))
	for _, r := range body.Routes {
		out := &Route{
			DistanceM:    r.Distance,
			DurationS:    r.Duration,
			EngineWeight: r.Weight,
			WeightName:   r.WeightName,
		}

		var steps []osrmStep
		for _, leg := range r.Legs {
			steps = append(steps, leg.Steps...)
		}
		// **والهندسةُ تُبنى من الخطوات حين توجد** — انظر `buildFromSteps`.
		if built := buildFromSteps(steps); built != nil {
			out.Geometry = built.Geometry
			out.CumulativeM = built.CumulativeM
			out.Maneuvers = built.Maneuvers
		} else {
			// **وإلّا فالخطُّ الإجماليُّ كما كان** — **ومحرّكٌ لم يردّ
			// خطواتٍ يُرسم ولا يُرشِد**، ولا يسقط شيء.
			for _, c := range r.Geometry.Coordinates {
				if len(c) >= 2 {
					out.Geometry = append(out.Geometry, Point{Lat: c[1], Lng: c[0]})
				}
			}
		}
		all = append(all, out)
	}
	return all, nil
}

// coord يكتب الإحداثيَّ بستّ منازل — **نحوُ عشرة سنتيمترات**، وما زاد
// عليها ضجيجٌ يطيل العنوان.
func coord(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }
