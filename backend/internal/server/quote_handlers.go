package server

// تسعيرةُ السلّة قبل الطلب — **رقمٌ يتغيّر أمام العين.**
//
// # المسألة
//
// رسمُ التوصيل صار يتغيّر بعدد المصادر وبُعدها، **ولم يعد رقماً ثابتاً للمنطقة**.
// وكانت السلّةُ تقرأ رسمَ المنطقة وحدَه — **فيرى الزبونُ رقماً ويُحاسَب بآخر.**
//
// **ورقمٌ يتغيّر أمام العين يُقبل، ورقمٌ يظهر عند الدفع يُراجَع**: من أضاف صنفاً
// فارتفع الرسمُ أمامه يفهم أن السببَ إضافتُه، **ومن رآه عند الدفع يظنّ أنه
// خُدع.**
//
// # ولماذا الخادمُ يحسبها لا السلّة
//
// السلّةُ **لا تعرف المصادر** — أخفيناها عنها عمداً، ولا تعرف المسافةَ بينها.
// **والحسبةُ حيث المعرفة**، وهي عند الخادم وحدَه.
//
// **ولا تُنشئ شيئاً**: قراءةٌ محضة، فمن استعرض عشرَ مرّاتٍ لم يترك أثراً.

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleQuote تسعيرةُ سلّةٍ في موقعٍ بعينه.
func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Items []orders.ItemInput `json:"items"`
		Lat   float64            `json:"lat"`
		Lng   float64            `json:"lng"`
		// Expected **ما كان معروضاً على شاشته** — **يُقارَن به ولا
		// يُصدَّق منه حكم** (`CC`، ٢٠٢٦-٠٩-١٥).
		Expected *struct {
			Lines       map[string]int64 `json:"lines"`
			DeliveryFee *int64           `json:"delivery_fee"`
			Discount    *int64           `json:"discount"`
		} `json:"expected"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وسلّةٌ فارغةٌ ليست تسعيرة** — (تدقيقُ الإطلاق ٢٠٢٦-٠٨-١٩ —
	// BUG-006: رُدَّ ٢٠٠ بمجموعٍ صفر).
	//
	// **وصفرٌ يُقرأ سعراً**: تُعرض «الإجماليّ ٠ ل.س» على شاشةٍ لا صنفَ
	// فيها، **فيُظنّ أنّ الطلبَ مجّانيّ** — والصوابُ أن يُقال إنّ
	// السؤالَ نفسَه لا يصحّ.
	if len(req.Items) == 0 {
		s.respondErr(w, errValidation)
		return
	}

	// ══════════════════════════════════════════════════════════════════
	// **وما كان معروضاً يُقارَن به** (`CC`، ٢٠٢٦-٠٩-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **ويُقارَن به ولا يُصدَّق منه حكم** — **الحقيقةُ من المحرّك.**
	var exp orders.Expected
	if req.Expected != nil {
		exp = orders.Expected{
			Lines: req.Expected.Lines, Has: true, DeliveryFee: -1, Discount: -1,
		}
		if req.Expected.DeliveryFee != nil {
			exp.DeliveryFee = *req.Expected.DeliveryFee
		}
		if req.Expected.Discount != nil {
			exp.Discount = *req.Expected.Discount
		}
	}

	q, err := s.orders.Quote(r.Context(), req.Items, req.Lat, req.Lng)
	if err != nil {
		// ══════════════════════════════════════════════════════════════
		// **وصنفٌ نفد لا يُردّ رمزاً أصمّ** (`CC-04`، `CC-19`)
		// ══════════════════════════════════════════════════════════════
		//
		// **والتسعيرةُ تسقط عند أوّل صنفٍ معطوب** — **فتقرأ الشاشةُ
		// «هذا الصنف غير متاح» ولا تعرف أيَّ سطرٍ تُعلّم**، **فتمحو
		// السلّةَ أو تقول «حدث خطأ».**
		//
		// **ومن طلب مقارنةً يُجاب بما تبدّل باسمه** — **ولا تُسعَّر
		// السلّةُ حتّى يُحسَم**: **مجموعٌ يُحسب على ما صحّ وحدَه
		// يُقرأ حذفاً صامتاً للباقي.**
		//
		// **والمنعُ عند الإنشاء كما كان** — **ولم يُمَسّ.**
		if cs := s.changesFor(r.Context(), req.Items, exp, err); len(cs) > 0 {
			httpx.JSON(w, http.StatusOK, &orders.QuoteResult{
				Blocked: true, Changes: cs,
			})
			return
		}
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **وحالُ الإتاحة تُقرأ بالمصادر التي تمنع** (`AV`، ٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وبوّاباتُ الإطلاق والمنصّة تُقرآن هنا بما يقرؤه المنعُ نفسُه**
	// — `launchOpen` و`platform.State` — **فلا يفترق الشرحُ عن الحكم.**
	//
	// **ولا يُمنَع شيءٌ هنا**: **التسعيرةُ تُخبِر، والمنعُ عند
	// الإنشاء.** **وسلّةٌ تنهار لأنّ الوقتَ انتهى سلّةٌ لا تُستعمل.**
	// ══════════════════════════════════════════════════════════════════
	// **وما تبدّل يُقال باسمه** (`CC`، ٢٠٢٦-٠٩-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **والمحرّكُ يعرف أيَّ صنفٍ نفد وأيَّ سعرٍ تبدّل** — **فلا يُردّ
	// رمزٌ واحدٌ لا يقول أيَّها.**
	if exp.Has {
		q.Changes = append(q.Changes, s.orders.CartChanges(r.Context(), s.pg, req.Items, exp)...)
		q.Changes = append(q.Changes, orders.FeeChange(exp, q.DeliveryFee)...)
	}

	gates := s.orderGates(r.Context())
	if av, err := s.orders.AvailabilityAt(r.Context(), s.pg,
		gates, req.Items, req.Lat, req.Lng); err == nil {
		q.Availability = &av
	} else {
		s.logger.Error("تعذّر حسابُ حال الإتاحة", "err", err)
	}
	httpx.JSON(w, http.StatusOK, q)
}

// handlePromoPreview أثرُ كود الخصم قبل الطلب.
//
// **والزبونُ يُسأل عنه لا الزائر**: القواعدُ فيها «مرّةً لكلّ مستخدم»
// و«لأوّل طلبٍ فقط» — **ومعاينةٌ بلا صاحبٍ تَعِد بما لا يقع.**
//
// **ولا تُنشئ شيئاً**: معاملةٌ تُلغى، فمن جرّب عشرةَ أكوادٍ لم يستهلك واحداً.
func (s *Server) handlePromoPreview(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code        string `json:"code"`
		Subtotal    int64  `json:"subtotal"`
		DeliveryFee int64  `json:"delivery_fee"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Code == "" || req.Subtotal <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	out, err := s.orders.PreviewPromo(r.Context(), req.Code, userIDFrom(r), req.Subtotal, req.DeliveryFee)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ══════════════════════════════════════════════════════════════════════
// **بوّاباتُ الطلب — تُبنى مرّةً وتُقرأ من موضعين** (`PC`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والتسعيرةُ تقرؤها والإتاحةُ قبلَ السلّة تقرؤها** — **ونسختان
// تفترقان يوماً**: **فتقول شاشةُ السوق «مفتوح» وتقول السلّة «مغلق»
// في اللحظة نفسِها.**

// orderGates وضعُ الإطلاق وحالُ المنصّة كما يقرؤهما المنعُ نفسُه.
func (s *Server) orderGates(ctx context.Context) orders.Gates {
	g := orders.Gates{
		LaunchOpen: s.launchOpen(ctx, launchCustomerOrders) &&
			s.launchOpen(ctx, launchMerchantOrders),
		PlatformAvailable: true,
	}
	if st, err := s.platform.State(ctx, s.pg); err == nil {
		g.PlatformAvailable = st.OrderingAvailable
		g.PlatformReason = string(st.Reason)
		g.PlatformMessage = st.Message
	}
	return g
}

// ══════════════════════════════════════════════════════════════════════
// **أيُقبَل طلبٌ إلى هذا العنوان؟ — قبل أن يملأ سلّة** (`PC`)
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **وكان أوّلُ خبرٍ يبلغه أنّ عنوانَه خارجَ النطاق يأتيه في السلّة** —
// **بعد أن اختار وأضاف وقرأ الأسعار.** **ومن مشى الطريقَ كلَّه ليُردّ
// في آخره يقرأ الردَّ عقوبةً لا خبرا.**
//
// **والزرُّ الذي يبدو صالحاً ثمّ يُردّ أسوأُ من زرٍّ مُعطَّلٍ بسبب.**
//
// # ولا سلّةَ في السؤال
//
// **والسؤالُ عن العنوان لا عن البضاعة** — **فلا صنفَ يُرسَل**،
// **وبوّابةُ المتجر تُتخطّى وحدَها** (`len(items) > 0` في المحرّك).
//
// **ومحرّكُ الإتاحة هو هو** (`AvailabilityAt`، الدفعةُ الثالثة) —
// **ولا محرّكَ ثانٍ يُكتب لشاشةٍ ثانية.**
//
// # ولا يُنشئ شيئاً
//
// **قراءةٌ محضة** — **فمن بدّل عنوانَه عشرَ مرّاتٍ لم يترك أثراً.**
func (s *Server) handlePublicAvailability(w http.ResponseWriter, r *http.Request) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		s.respondErr(w, errValidation)
		return
	}
	av, err := s.orders.AvailabilityAt(r.Context(), s.pg, s.orderGates(r.Context()), nil, lat, lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, av)
}

// changesFor **ما تبدّل حين تسقط التسعيرةُ بصنفٍ معطوب.**
//
// **وفارغةٌ حين لم يطلب مقارنةً أو حين لا يُترجَم العطب** — **فيُردّ
// الرمزُ كما كان، ولا يُبتلَع خطأٌ لا نفهمه.**
func (s *Server) changesFor(
	ctx context.Context, items []orders.ItemInput, exp orders.Expected, cause error,
) []orders.Change {
	if !exp.Has {
		return nil
	}
	switch {
	case errors.Is(cause, orders.ErrItemGone),
		errors.Is(cause, orders.ErrItemUnavailable),
		errors.Is(cause, orders.ErrBadQty):
	default:
		return nil
	}
	return s.orders.CartChanges(ctx, s.pg, items, exp)
}
