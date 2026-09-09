// حارسُ مصفوفة البثّ — **`D20`.**
//
// # ما يحرسه
//
// **`orders.ViewFor` صارت البابَ الوحيدَ الذي تخرج منه حمولةُ البثّ.**
// **وهذا الملفُّ يقيسها بعقد `P-1` نفسِه** — **حقلاً حقلاً، طرفاً طرفاً**،
// **لا بعيّنةٍ من طلبٍ واقعيّ:** طلبٌ واقعيٌّ يترك نصفَ حقوله صفراً،
// **فيمرّ الحقلُ المحظورُ لأنّه كان فارغاً لا لأنّه مُنِع.**
//
// **فيُملأ الطلبُ بالانعكاس حتّى لا يبقى حقلٌ صفريّ** — **ثمّ يُقاس ما
// خرج.**
//
// # واتّجاهان لا واحد
//
//	١ **لا يخرج ما مُنِع** — وهو `D20` نفسُه
//	٢ **ولا يُحجَب ما أُجيز** — **وإلّا «أُصلحت» الخصوصيّةُ ببثٍّ فارغ**،
//	  فتعمى الشاشاتُ ولا يسقط شيء
//
// # ونموُّ البنية
//
// **حقلٌ جديدٌ في `orders.Order` لا يخرج من نفسه** — **الحمولةُ تُبنى
// بالسماح**: ما لم يُذكَر بالاسم لا يمرّ. **وهذا يُقاس هنا لا يُفترَض.**
package qa

import (
	"reflect"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// viewAudience الطرفُ في البثّ مقابلَ الدور في العقد.
//
// **والاسمان يفترقان قصداً**: العقدُ يقول `admin` **وغرفةُ البثّ
// `ops`** — **وهي غرفةُ مكتب العمليّات لا حسابُ أدمن.**
var viewAudience = map[string]orders.Audience{
	RoleCustomer: orders.AudienceCustomer,
	RoleMerchant: orders.AudienceMerchant,
	RoleDriver:   orders.AudienceDriver,
	RoleRep:      orders.AudienceRep,
	RoleAdmin:    orders.AudienceOps,
}

// fillNonZero يملأ كلَّ حقلٍ بقيمةٍ غيرِ صفريّة — **بالانعكاس.**
//
// **ولا قائمةَ مكتوبة**: **من أضاف حقلاً مُلئ في الحال**، **فلا يمرّ
// حقلٌ جديدٌ في الفحص لأنّه بقي فارغاً.**
func fillNonZero(rv reflect.Value, depth int) {
	if depth > 4 {
		return
	}
	switch rv.Kind() {
	case reflect.String:
		rv.SetString("س")
	case reflect.Bool:
		rv.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		rv.SetInt(7)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		rv.SetUint(7)
	case reflect.Float32, reflect.Float64:
		rv.SetFloat(7)
	case reflect.Ptr:
		if rv.Type().Elem() == reflect.TypeOf(time.Time{}) {
			now := time.Now()
			rv.Set(reflect.ValueOf(&now))
			return
		}
		p := reflect.New(rv.Type().Elem())
		fillNonZero(p.Elem(), depth+1)
		rv.Set(p)
	case reflect.Slice:
		e := reflect.New(rv.Type().Elem())
		fillNonZero(e.Elem(), depth+1)
		rv.Set(reflect.Append(reflect.MakeSlice(rv.Type(), 0, 1), e.Elem()))
	case reflect.Map:
		k := reflect.New(rv.Type().Key())
		fillNonZero(k.Elem(), depth+1)
		val := reflect.New(rv.Type().Elem())
		fillNonZero(val.Elem(), depth+1)
		m := reflect.MakeMap(rv.Type())
		m.SetMapIndex(k.Elem(), val.Elem())
		rv.Set(m)
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			rv.Set(reflect.ValueOf(time.Now()))
			return
		}
		for i := 0; i < rv.NumField(); i++ {
			if rv.Type().Field(i).PkgPath != "" {
				continue // غيرُ مُصدَّرٍ — لا يخرج في JSON
			}
			fillNonZero(rv.Field(i), depth+1)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **١ · ما مُنِع لا يخرج** — `D20`
// ══════════════════════════════════════════════════════════════════════

// heap نسخةٌ في الكومة — `ViewFor` تأخذ مؤشّراً.
func heap(o orders.Order) *orders.Order { return &o }

func TestD20_BroadcastViewObeysContract(t *testing.T) {
	o := heap(fullOrder())
	for _, role := range PrivacyRoles {
		aud, ok := viewAudience[role]
		if !ok {
			t.Fatalf("الدورُ %q بلا طرفٍ في البثّ", role)
		}
		view := orders.ViewFor(aud, o)
		if view == nil {
			t.Fatalf("**تعذّر بناءُ حمولةِ %q** — والبثُّ يسقط مغلقاً فتعمى شاشتُه", role)
		}
		vs := CheckPayload(role, ChannelRealtime, view)
		for _, v := range vs {
			t.Errorf("%s", v)
		}
		if len(vs) > 0 {
			t.Errorf("**%d حقلاً محظوراً في حمولة %q** — **والبثُّ ليس قناةً "+
				"مميّزة.** (`D20`)", len(vs), role)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · وما أُجيز لا يُحجَب** — **وإلّا «أُصلحت» الخصوصيّةُ بالعمى**
// ══════════════════════════════════════════════════════════════════════

func TestD20_BroadcastViewIsNotOverNarrow(t *testing.T) {
	o := heap(fullOrder())
	for _, role := range PrivacyRoles {
		view := orders.ViewFor(viewAudience[role], o)
		var lost []string
		for _, f := range OrderFields() {
			if Visible(f, role) != VisAllowed {
				continue
			}
			if _, ok := view[f]; !ok {
				lost = append(lost, f)
			}
		}
		if len(lost) > 0 {
			t.Errorf("**حقولٌ يجيزها العقدُ لـ%q ولا تصلها في البثّ (%d)**: %v\n"+
				"**وحمولةٌ ناقصةٌ لا تسقط من نفسها** — الشاشةُ تعرض قديماً.",
				role, len(lost), lost)
		}
		t.Logf("%s: %d حقلاً في الحمولة", role, len(view))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · نموُّ البنية** — **الحقلُ الجديدُ لا يخرج من نفسه**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا هو الفرقُ بين السماح والمنع**: قائمةُ منعٍ تشيخ **بحقلٍ يُضاف
// بعدها**، **وقائمةُ سماحٍ لا تشيخ** — **الجديدُ ممنوعٌ حتّى يُذكَر.**
//
// **ويُقاس بالانعكاس**: **كلُّ ما يخرج من الحمولة له حكمٌ في العقد** —
// **فما لا حكمَ له لم يخرج.**

func TestD20_UnclassifiedFieldNeverReachesAnyAudience(t *testing.T) {
	o := heap(fullOrder())
	for _, role := range PrivacyRoles {
		for f := range orders.ViewFor(viewAudience[role], o) {
			if _, ok := OrderPrivacy[f]; !ok {
				t.Errorf("**الحقلُ %q خرج إلى %q ولا حكمَ له في العقد** — "+
					"**والسماحُ صار منعاً مقلوباً.** (`D20`)", f, role)
			}
		}
	}

	// **ولا يُصدَّق أنّ المرشَّح يعمل لأنّ الطلبَ كان فارغاً**:
	// **طرفٌ ضيّقٌ يجب أن يُسقط حقولاً فعلاً** — **ومكتبُ العمليّات
	// يرى الطلبَ كلَّه بحكم العقد، فلا يصلح شاهداً.**
	full := len(OrderFields())
	mer := len(orders.ViewFor(orders.AudienceMerchant, o))
	if full == 0 || mer == 0 {
		t.Fatal("لم يُقرأ شيء — الانعكاسُ مكسور")
	}
	if mer >= full {
		t.Errorf("**المتجرُ نال %d من %d حقلاً** — **والمرشَّحُ لا يُسقط شيئاً.**",
			mer, full)
	}
	t.Logf("حقولُ الطلب=%d · تصل المتجرَ=%d · تصل مكتبَ العمليّات=%d",
		full, mer, len(orders.ViewFor(orders.AudienceOps, o)))
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · ويسقط مغلقاً** — **لا حمولةَ عريضةٌ بديلاً**
// ══════════════════════════════════════════════════════════════════════

func TestD20_UnknownAudienceGetsNothing(t *testing.T) {
	o := heap(fullOrder())
	if v := orders.ViewFor(orders.Audience("finance"), o); v != nil {
		t.Errorf("**طرفٌ لا حكمَ له نال %d حقلاً** — **والسقوطُ يجب أن "+
			"يكون مغلقاً.**", len(v))
	}
	if v := orders.ViewFor(orders.AudienceOps, nil); v != nil {
		t.Errorf("**طلبٌ عدمٌ ردّ حمولة** — %v", v)
	}
}
