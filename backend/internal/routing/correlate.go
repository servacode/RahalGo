package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **ارتباطُ الأثر بالمسار — المرحلة ٨ب**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢١ بعد قياس `TD-PARALLEL-SAMEDIR`.)
//
// # المسألة
//
// **السائقُ قد يكون على طريقٍ موازٍ للمسار المخطَّط**: قريبٍ منه،
// **وفي الاتّجاه نفسِه.** فاتّجاهُه صحيحٌ، وبُعدُه عن الخطّ صغير،
// **والكواشفُ المحلّيّةُ تراه على المسار.**
//
// **وقِيس أنّ الهاتفَ وحدَه لا يحسمها**: بتباعد ١٢م ودقّةٍ ١٥م
// **والضجيجُ مترابطٌ لا مستقلّ**، لا معلومةَ في القراءة تُثبت أيَّ
// طريقٍ يسلكه — **والمعدّلُ لا يُلغي انحيازاً ثابتاً.**
//
// # فيُسأل المحرّكُ عن أثرٍ لا عن نقطة
//
// **`/match` يُطابق سلسلةَ قراءاتٍ على الشبكة** — والسلسلةُ تحمل من
// المعلومة ما لا تحمله نقطةٌ واحدة.
//
// **والمقارنةُ بعقد OSM لا بالهندسة** (قِيس ٢٠٢٦-٠٨-٢١): تقاطعُ
// العقد يفصل بهامشٍ وسيطُه **١٫٠٠٠**، والأضلاعُ الموجَّهةُ **٠٫٥٣٨**
// **ولا تُحسّن الدقّة** — فالعقدُ هي الحكم.
//
// **ولا يخرج من هذا الملفّ شيءٌ يخصّ محرّكاً بعينه** — لا `nodes`
// ولا `hint` ولا `matchings`. **العقدُ إلى الجوّال محايدٌ تماماً.**

// CorrelationContext **ما يُحفظ عن مسارٍ نافذ** — ليُقارَن به الأثر.
//
// **ولا يُرسَل إلى الجوّال** — يبقى في الخادم.
type CorrelationContext struct {
	// Nodes **عقدُ المسار بترتيب السير.**
	//
	// **وحجمُها صغير**: قِيس وسيط ١١ عقدةً وأقصى ١٠٣ — **٠٫١ إلى
	// ٠٫٨ كيلوبايت.**
	Nodes []int64 `json:"nodes"`
	// Fingerprint **بصمةُ المسار** — يُكشف بها تبدّلُه.
	Fingerprint string `json:"fp"`
}

// Usable **أفيه ما يكفي للمقارنة؟**
func (c *CorrelationContext) Usable() bool {
	return c != nil && len(c.Nodes) >= 2
}

// ══════════════════════════════════════════════════════════════════════
// **جلبُ العقد — نداءٌ داخليٌّ لا يُرى من خارج**
// ══════════════════════════════════════════════════════════════════════

// Nodes **عقدُ المسار بين نقطتين** — بترتيب السير.
//
// **وترتيبُ عقدِ الطريق في OSM اعتباطيّ** (شأنُ من رسمه لا من
// يسلكه)، **فتُؤخذ من ردّ المسار لا من الطريق.**
//
// **وحدُّ الالتقاط يُطبَّق كما في كلّ نداء** (المرحلة ٨أ) — فلا
// يكون طريقٌ محميّاً وآخرُ مكشوفاً.
func (c *Client) Nodes(ctx context.Context, from, to Point) ([]int64, error) {
	if !c.Enabled() {
		return nil, ErrNoEngine
	}
	url := c.base + "/route/v1/driving/" +
		coord(from.Lng) + "," + coord(from.Lat) + ";" +
		coord(to.Lng) + "," + coord(to.Lat) +
		"?overview=false&alternatives=false&steps=false&annotations=nodes"
	if c.snap.bounded() {
		url += "&radiuses=" + c.snap.radiuses()
	}
	var body struct {
		Code   string `json:"code"`
		Routes []struct {
			Legs []struct {
				Annotation struct {
					Nodes []int64 `json:"nodes"`
				} `json:"annotation"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := c.getJSON(ctx, url, &body); err != nil {
		return nil, err
	}
	if body.Code != "Ok" || len(body.Routes) == 0 {
		return nil, mapCode(body.Code)
	}
	var out []int64
	for _, leg := range body.Routes[0].Legs {
		out = append(out, leg.Annotation.Nodes...)
	}
	return out, nil
}

// ══════════════════════════════════════════════════════════════════════
// **مطابقةُ الأثر**
// ══════════════════════════════════════════════════════════════════════

// TracePoint **قراءةُ موضعٍ من الجهاز** — بلا شيءٍ يخصّ محرّكاً.
type TracePoint struct {
	Lat       float64
	Lng       float64
	AccuracyM float64
	AtMs      int64
}

// MatchResult **ما ردّه المحرّكُ عن الأثر.**
type MatchResult struct {
	Confidence float64
	Nodes      []int64
}

// MaxTraceFixes **سقفُ القراءات في الطلب** — البند ٣٢.
//
// **وثمانٍ هي نافذةُ القياس** — والسقفُ ضِعفُها متّسعاً لا أكثر.
const MaxTraceFixes = 16

// MinTraceFixes **ودونها لا معنى للمطابقة.**
const MinTraceFixes = 4

// Match **يُطابق أثراً على الشبكة ويردّ عقدَه.**
//
// **ونصفُ قطر البحث من دقّة كلّ قراءة** — ثلاثةُ أضعافها بحدٍّ أدنى
// عشرة، **وهو ما عُوير عليه القياس.**
func (c *Client) Match(ctx context.Context, trace []TracePoint) (*MatchResult, error) {
	if !c.Enabled() {
		return nil, ErrNoEngine
	}
	if len(trace) < MinTraceFixes || len(trace) > MaxTraceFixes {
		return nil, fmt.Errorf("routing: قراءاتٌ خارجَ الحدّ (%d)", len(trace))
	}
	var coords, radii, stamps strings.Builder
	base := trace[0].AtMs / 1000
	for i, p := range trace {
		if i > 0 {
			// **ومُرمَّزةٌ لا خام** — `net/url` تُسقط المعاملات إن
			// وجدت فاصلةً منقوطةً خاماً (المرحلة ٨أ).
			coords.WriteString("%3B")
			radii.WriteString("%3B")
			stamps.WriteString("%3B")
		}
		coords.WriteString(coord(p.Lng) + "," + coord(p.Lat))
		r := p.AccuracyM * 3
		if r < 10 {
			r = 10
		}
		if r > 100 {
			r = 100
		}
		radii.WriteString(strconv.FormatFloat(r, 'f', 0, 64))
		stamps.WriteString(strconv.FormatInt(p.AtMs/1000-base, 10))
	}
	url := c.base + "/match/v1/driving/" + coords.String() +
		"?overview=false&steps=false&annotations=nodes&gaps=ignore&tidy=false" +
		"&radiuses=" + radii.String() + "&timestamps=" + stamps.String()

	var body struct {
		Code      string `json:"code"`
		Matchings []struct {
			Confidence float64 `json:"confidence"`
			Legs       []struct {
				Annotation struct {
					Nodes []int64 `json:"nodes"`
				} `json:"annotation"`
			} `json:"legs"`
		} `json:"matchings"`
	}
	if err := c.getJSON(ctx, url, &body); err != nil {
		return nil, err
	}
	if body.Code != "Ok" || len(body.Matchings) == 0 {
		return nil, mapCode(body.Code)
	}
	m := body.Matchings[0]
	out := &MatchResult{Confidence: m.Confidence}
	for _, leg := range m.Legs {
		out.Nodes = append(out.Nodes, leg.Annotation.Nodes...)
	}
	return out, nil
}

// ══════════════════════════════════════════════════════════════════════
// **الحكم — هامشُ تقاطع العقد**
// ══════════════════════════════════════════════════════════════════════

// NodeMargin **كم يفوق المخطَّطُ سواه** — في [−١، +١].
//
// **موجبٌ يعني «على المخطَّط»، وسالبٌ «على غيره».**
//
// **ولا يُحسب على التقاطع المطلق بل على الفرق** — فالطريقان
// المتوازيان يتقاسمان عقداً عند التقاطعات: قِيس أنّ **٢٠٪ منهما
// تتقاسم عقداً، ووسيطُ المشترك ١٠٪ وأقصاه ٣٣٪.** **والفرقُ يعبر
// ذلك بأمان.**
func NodeMargin(matched, planned []int64) float64 {
	if len(matched) == 0 {
		return 0
	}
	set := make(map[int64]struct{}, len(planned))
	for _, n := range planned {
		set[n] = struct{}{}
	}
	var on int
	for _, n := range matched {
		if _, ok := set[n]; ok {
			on++
		}
	}
	ratio := float64(on) / float64(len(matched))
	// **على المخطَّط: النسبةُ إلى ١. وعلى غيره: إلى ٠.**
	//
	// **والهامشُ يُمدّ إلى [−١،+١]** فيصير الحكمُ عتبةً واحدة.
	return 2*ratio - 1
}

// ══════════════════════════════════════════════════════════════════════
// **عتباتُ التأكيد — مُعايَرةٌ لا مختارة**
// ══════════════════════════════════════════════════════════════════════
//
// **قِيست على ١٣٨ أثراً من `syria.osm.pbf`** (تسلسل ٤٦٣٥) بضجيجٍ
// مترابطٍ `AR(1)` τ=٣٠ث، وأزواجِ طرقٍ حقيقيّةٍ تباعدُها ٨–٣٠م
// وزاويتُها ≤٥°:
//
//	ثقة ≥٠٫٩٠ وهامش ≥٠٫٨٠  →  كشف ٦٤٫٣٪ · إيجابٌ كاذب ٠/٩٦
//
// **وعتبةُ ٠٫٩٥ تُصفّر الإيجابَ الكاذبَ وتهبط بالكشف إلى نصفه** —
// **ولا يستحقّ.**
const (
	// MinMatchConfidence **ثقةُ المحرّك في مطابقته.**
	//
	// **ولا تُصدَّق وحدَها**: قِيس أنّ ثقةَ ٠٫٩٥ تصيب ٧٠٪ فقط على
	// الحالات القاسية — **فهي شرطٌ لازمٌ لا كافٍ.**
	MinMatchConfidence = 0.90
	// MinNodeMargin **وفصلٌ حاسمٌ في العقد.**
	MinNodeMargin = 0.80
)

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 500 {
		return fmt.Errorf("routing: ردّ %d", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("routing: ردّ %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func mapCode(code string) error {
	switch code {
	case "NoSegment":
		return ErrNoSegment
	case "NoRoute", "NoMatch":
		return ErrNoRoute
	}
	return fmt.Errorf("routing: %s", code)
}
