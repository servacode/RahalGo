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
}

// Client بابُ محرّك المسارات.
//
// **وفارغُ العنوان يعني «لا محرّك»** — فيردّ ErrNoEngine ولا يُنادي شبكة.
type Client struct {
	base string
	http *http.Client
}

// ErrNoEngine لا عنوانَ لمحرّك المسارات.
var ErrNoEngine = fmt.Errorf("routing: لا محرّك مسارات")

func New(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		// **ومهلةٌ قصيرة**: هذا رقمٌ يُحسّن الشاشة لا يصنعها، **وانتظارُ
		// خمسِ ثوانٍ لرسمِ خطٍّ** يُقرأ تعليقاً في التطبيق.
		http: &http.Client{Timeout: 4 * time.Second},
	}
}

// Enabled أثمّة محرّكٌ مضبوط؟
func (c *Client) Enabled() bool { return c != nil && c.base != "" }

// Route يسأل عن الطريق من نقطةٍ إلى نقطة.
//
// **والإحداثيّاتُ في OSRM طولٌ ثمّ عرض** — عكسُ ما تكتبه القاعدةُ والشاشة.
// **ومن قلبها** حصل على مسارٍ في بلدٍ آخر أو على «لا مسار».
func (c *Client) Route(ctx context.Context, from, to Point) (*Route, error) {
	if !c.Enabled() {
		return nil, ErrNoEngine
	}
	url := c.base + "/route/v1/driving/" +
		coord(from.Lng) + "," + coord(from.Lat) + ";" +
		coord(to.Lng) + "," + coord(to.Lat) +
		"?overview=full&geometries=geojson&alternatives=false&steps=false"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("routing: ردّ %d", res.StatusCode)
	}

	var body struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Geometry struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	// **و«لا مسار» ليست خطأً في النداء**: نقطةٌ في الصحراء بلا طريقٍ إليها
	// تردّ `NoRoute` بحالة ٢٠٠. **ومن قرأ الحالةَ وحدَها** ظنّ أنّه نجح.
	if body.Code != "Ok" || len(body.Routes) == 0 {
		return nil, fmt.Errorf("routing: %s", body.Code)
	}

	r := body.Routes[0]
	out := &Route{DistanceM: r.Distance, DurationS: r.Duration}
	for _, c := range r.Geometry.Coordinates {
		if len(c) >= 2 {
			out.Geometry = append(out.Geometry, Point{Lat: c[1], Lng: c[0]})
		}
	}
	return out, nil
}

// coord يكتب الإحداثيَّ بستّ منازل — **نحوُ عشرة سنتيمترات**، وما زاد
// عليها ضجيجٌ يطيل العنوان.
func coord(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }
