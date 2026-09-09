package server

// حمولةُ الطلب في `REST` — **بالسماح لا بالمنع.**
//
// # ما كان
//
// **`redactForCustomer` تمحو أربعةَ حقولٍ و`redactForMerchant` ستّةَ
// عشر** — **ثمّ يُسلسَل الكائنُ الداخليُّ كلُّه.** **فما لم يُذكَر في
// المحو خرج**، **وحقلٌ يُضاف إلى `orders.Order` غداً يخرج من نفسه.**
//
// **وقيس على الأبواب الأربعة** (دورةُ ٤٥): **الزبونُ ١٦ خرقاً في
// تفصيله و١٤ في قائمته · والمتجرُ ١٤ و١٦** — **وهاتفُ السائق يصل
// الزبونَ قيمةً لا اسمَ حقل.**
//
// **وثلاثةُ أبوابٍ للزبون بلا تنقيةٍ إطلاقاً**: إنشاءُ الطلب وإلغاؤه
// وإنشاءُ الخاصّ.
//
// # وما صار
//
// **الحمولةُ تُبنى من `orders.ViewFor`** — **المرشَّحُ نفسُه الذي
// يبثّ به المحرّك** (دورةُ ٤٤)، **مولَّدٌ من عقد `P-1`.**
//
// **ولا قائمةَ ثانيةٌ هنا**: **قائمتان تنحرفان** — **وذاك `D23`
// بعينه.**
//
// # والغلافُ يبقى غلافاً
//
// **`REST` تزيد ما ليس في الطلب**: مهلةُ الإلغاء · مسارُه بأوقاته.
// **وهي عرضٌ لا حقولُ كائن** — **تُضاف فوق الحمولة الآمنة، ولا
// تُفتح بها ثغرة**: **ما يُضاف يُسمّى بالاسم هنا.**
//
// # ويسقط مغلقاً
//
// **تعذّرَ بناءُ الحمولة فلا يُسلسَل الكائنُ الخام** — **يُردّ عطبٌ.**
// **حمولةٌ لا تصل تُعاد، وتسريبٌ لا يُسحَب.**

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// errPayloadUnsafe **تعذّر بناءُ حمولةٍ آمنة** — ولا بديلَ عريض.
var errPayloadUnsafe = httpx.NewError(http.StatusInternalServerError,
	"payload_unsafe", "errors.internal")

// orderView حمولةُ طلبٍ واحدٍ لطرفٍ بعينه.
//
// **ويُردّ خطأٌ لا حمولةٌ ناقصة** — **فمن ردّ نصفَ الطلب أعمى شاشةً
// صامتاً، ومن ردّ عطباً أيقظ من ينظر.**
func orderView(a orders.Audience, o *orders.Order) (map[string]any, error) {
	v := orders.ViewFor(a, o)
	if v == nil {
		return nil, errPayloadUnsafe
	}
	return v, nil
}

// orderViews حمولةُ قائمة — **وواحدةٌ تسقط فتسقط القائمة.**
//
// **ولا يُتخطّى الطلبُ الذي تعذّر بناؤه**: **قائمةٌ ينقص منها طلبٌ
// تُقرأ «لا طلبَ لك»**، **وهو كذبٌ أهدأُ من العطب وأضرّ.**
func orderViews(a orders.Audience, list []orders.Order) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		v, err := orderView(a, &list[i])
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// withExtra يضيف حقولَ الغلاف فوق الحمولة الآمنة.
//
// **وأسماؤها تُكتب في النداء** — **فمن قرأ الباب رأى ما يزيده عليه.**
func withExtra(v map[string]any, extra map[string]any) map[string]any {
	for k, val := range extra {
		v[k] = val
	}
	return v
}
