package server

/*
**الطلبُ الخاصّ — بابان: يُطلب، ويُوثَّق ما اتُّفق عليه.**

(قرارُ المالك ٢٠٢٦-٠٨-٠٩.)
*/

import (
	"context"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// handleCreateCustomOrder **الزبونُ يصف ما يريد.**
//
// **ولا أصنافَ ولا سعر** — ولا يُسأل عن متجر: هو يطلب ما ليس في المنصّة.
func (s *Server) handleCreateCustomOrder(w http.ResponseWriter, r *http.Request) {
	// **والطلبُ المخصَّصُ بابٌ مستقلّ** — يُفتح ويُغلق دون العاديّ.
	//
	// **وقبل قراءةِ الجسم** — فلا يُستهلك مفتاحُ تفرّدٍ لبابٍ مغلق.
	if !s.requireLaunch(w, r, launchCustomerCustomOrders) {
		return
	}
	// **والمخصَّصُ يخضع لدوام المنصّة كالعاديّ** — **ومكتبٌ مغلقٌ لا
	// يحضّر طلباً موصوفاً كما لا يحضّر طلباً من متجر.**
	if !s.requireOrdering(w, r) {
		return
	}
	req, err := decode[struct {
		Request     string  `json:"request"`
		AddressText string  `json:"address_text"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		// Payment **نقدٌ أو محفظة** — والخصمُ عند التسليم لا عند الطلب.
		Payment string `json:"payment_method"`
		// Notes **ملاحظاتٌ للسائق** — `CAF-07`/`CUST-CUSTOM-019`: كانت
		// تُرسَل ولا تُفكّ فتُهمَل صامتةً، **فيشتري السائقُ بلا تعليماتِ
		// صاحبها.** فتُفكّ الآن وتُخزَّن كالعاديّ (عمودُ `orders.notes`).
		Notes string `json:"notes"`
		// Mode **أمانةٌ أم مشتريات** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — وما عداهما مشتريات.
		Mode string `json:"custom_mode"`
		// PickupLat/Lng **موقعُ الأمانة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — الزبونُ يحدّده على الخريطة
		// والسائقُ يراه على خريطته. **للأمانة وحدَها** — ويُهمَل في المشتريات (السائقُ يختار
		// من أين يشتري). **واختياريٌّ في الخادم** — فالتطبيقُ القديمُ لا يرسله.
		PickupLat *float64 `json:"pickup_lat"`
		PickupLng *float64 `json:"pickup_lng"`
		// HolderName/HolderPhone **صاحبُ الأمانة ورقمُه** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — يلغي «الخريطة»:
		// الزبونُ لا يعرف أين الأمانة. **إلزاميّان في الأمانة**، ويُهملان في المشتريات.
		HolderName  string `json:"holder_name"`
		HolderPhone string `json:"holder_phone"`
		// LocationRequestID **«اطلب الموقع منه»** — طلبُ موقعٍ أنشأه قبل الإرسال
		// (`POST /my/location-requests`)، **فيُربط بالطلب**: نقطتُه نقطةُ الاستلام الآن أو حين تصل.
		LocationRequestID string `json:"location_request_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والنصوصُ الحرّةُ تمرّ بالحارس** — انظر `text_limits.go`.
	if _, err := s.guardText(r.Context(),
		tf("request", &req.Request, maxCustomRequest, textguard.Notes),
		tf("address_text", &req.AddressText, maxAddressText, textguard.Address),
		tf("notes", &req.Notes, maxOrderNotes, textguard.Notes),
		tf("holder_name", &req.HolderName, maxPersonName, textguard.Name),
	); err != nil {
		s.respondErr(w, err)
		return
	}
	// **والأمانةُ بلا صاحبها ورقمه لا تُرسَل** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — السائقُ يتّصل به.
	// **والرقمُ يُفحص كبقيّة الأرقام** ويُخزَّن دوليّاً.
	var holderPhone string
	// **والتطبيقُ القديمُ لا يرسلهما** — فيُقبل بلاهما (السائقُ يسأل بالدردشة كما كان)؛ **ومن أرسل
	// أحدَهما فليرسل الآخر.** والإلزامُ في التطبيق الجديد نفسِه.
	hasName, hasPhone := strings.TrimSpace(req.HolderName) != "", strings.TrimSpace(req.HolderPhone) != ""
	if req.Mode == orders.CustomModeAmanah && (hasName || hasPhone) {
		if !hasName || !hasPhone {
			s.respondErr(w, errAmanahHolder)
			return
		}
		p, ok := identity.NormalizePhone(req.HolderPhone)
		if !ok {
			s.respondErr(w, identity.ErrInvalidPhone)
			return
		}
		holderPhone = p
	}
	// **العملُ وعلامةُ تثبيتِ منع التكرار في معاملةٍ واحدة** — `XG-33`.
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		o, after, err := s.orders.CreateCustomTx(ctx, q, userIDFrom(r), req.Request,
			req.AddressText, req.Payment, req.Notes, req.Lat, req.Lng)
		if err != nil {
			return IdempotentBody{}, err
		}
		// **والأمانةُ تُكتب في معاملة الإنشاء نفسِها** — فلا يراها سائقٌ مشترياتٍ لحظة.
		if req.Mode == orders.CustomModeAmanah {
			if _, err := q.Exec(ctx, `UPDATE orders SET custom_mode = $2 WHERE id = $1`,
				o.ID, orders.CustomModeAmanah); err != nil {
				return IdempotentBody{}, err
			}
			// **وأجرتُها أجرتُها** (٢٠٢٦-١٠-١٣) — تُلقَط بدل أجرة المشتريات إن ضُبطت.
			if fee := s.orders.AmanahFeeSnapshot(ctx, q); fee != nil {
				if _, err := q.Exec(ctx, `UPDATE orders SET custom_fee_snapshot = $2
					WHERE id = $1 AND custom_fee_source = 'admin_defined'`, o.ID, *fee); err != nil {
					return IdempotentBody{}, err
				}
				o.CustomFeeSnapshot = fee
			}
			o.CustomMode = orders.CustomModeAmanah
			holderName := strings.TrimSpace(req.HolderName)
			if _, err := q.Exec(ctx, `UPDATE orders SET amanah_holder_name = $2, amanah_holder_phone = $3
				WHERE id = $1`, o.ID, holderName, holderPhone); err != nil {
				return IdempotentBody{}, err
			}
			o.HolderName, o.HolderPhone = holderName, holderPhone
			// **وموقعُه إن طلبه منه** — وصل قبل الإرسال؟ نقطةُ استلامٍ الآن. وإلّا كُتب حين يصل.
			shared := false
			if req.LocationRequestID != "" {
				lat, lng, err := linkAmanahLocationTx(ctx, q, req.LocationRequestID, userIDFrom(r), o.ID)
				if err != nil {
					return IdempotentBody{}, err
				}
				if lat != nil && lng != nil && req.PickupLat == nil {
					req.PickupLat, req.PickupLng, shared = lat, lng, true
				}
			}
			if req.PickupLat != nil && req.PickupLng != nil {
				if err := s.orders.SetAmanahPickupTx(ctx, q, o.ID, *req.PickupLat, *req.PickupLng); err != nil {
					// **وموقعٌ شاركه صاحبُها خارجَ التغطية لا يُسقط الطلب** — السائقُ يتّصل به.
					if !shared {
						return IdempotentBody{}, err
					}
					s.logger.Warn("موقعُ الأمانة المُشارَك لم يُقبل نقطةَ استلام", "order", o.ID, "error", err)
					req.PickupLat, req.PickupLng = nil, nil
				}
			}
			if req.PickupLat != nil && req.PickupLng != nil {
				lat, lng := *req.PickupLat, *req.PickupLng
				o.AmanahPickupLat, o.AmanahPickupLng = &lat, &lng
				o.PickupNote = orders.AmanahPickupNote
			}
		}
		// **وردُّ الخاصّ يُشكَّل كالعاديّ** — **ولا بابَ يُعفى.**
		view, err := orderView(orders.AudienceCustomer, o)
		if err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{Status: http.StatusCreated, Payload: view, AfterCommit: func() {
			// **والبثُّ بعد التثبيت** — بابٌ واحدٌ يبلغ الأطرافَ كلَّها:
			// **العملياتِ التي تنتظر موافقتَها، وصاحبَ الطلب** —
			// **وكان يبلغ المكتبَ وحدَه** (`D22`).
			//
			// **وطلبٌ ينتظر من لا يعلم أنّه ينتظره لا يُخدَم** —
			// **وصاحبُه ينظر إلى شاشةٍ لا تتحرّك.**
			after()
		}}, nil
	})
}

// handleAgreeCustom **السائقُ يوثّق ما اتّفق عليه مع الزبون.**
//
// **بعد المحادثة لا قبلها**: يقول له الثمنَ فيوافق، ويقول له الأجرةَ فيوافق،
// **ثمّ يُوثَّق.** (قرارُ المالك.)
func (s *Server) handleAgreeCustom(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Goods int64 `json:"goods_amount"`
		Fee   int64 `json:"fee"`
		// Step **خطوةُ التوثيق** (٢٠٢٦-١٠-٠٣): `fee` ثمّ `goods` — وفارغٌ الاثنان معاً.
		Step string `json:"step"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	if err := s.orders.AgreeCustomStep(r.Context(), orderID, userIDFrom(r),
		req.Step, req.Goods, req.Fee); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ويصل الزبونَ ما وُثّق باسمه.**
	//
	// **واتّفاقٌ لا يراه صاحبُه ليس اتّفاقاً**: قيل له في المحادثة، **ويبقى
	// مكتوباً حيث يراه** — فمن نسي رجع إليه، ومن خولف احتجّ به.
	var customerID string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&customerID); err == nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: customerID, Kind: notifications.KindOrder,
			Title: notifTitles.customAgreed, Body: "",
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"agreed": true})
}

// handleConfirmQuote **الزبونُ يؤكّد العرضَ ويختار طريقةَ الدفع** — Batch 2a.
//
// (قرارُ المالك: العرضُ يُقفَل بتأكيدٍ من الزبون قبل أن يبدأ الشراء.)
//
// **ويُرسِل الزبونُ ما رآه** — المبلغَ والنسخة — **فإن تبدّل العرضُ بينهما رُدّ
// `quote_changed`** ليقرأ الجديدَ ويؤكّده. **والمحفظةُ لا تُقبَل إلّا إن غطّى
// المتاحُ المبلغَ، ويُحجَز فوراً.**
func (s *Server) handleConfirmQuote(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		// Payment **نقدٌ أو محفظة** — يُختار عند القفل لا عند الطلب.
		Payment string `json:"payment_method"`
		// ExpectedTotal وExpectedQuoteVersion **العرضُ الذي رآه الزبونُ** —
		// يُطابَقان بالحاليّ، وإلّا فتغيّر السعرُ تحت يده.
		ExpectedTotal        int64 `json:"expected_total"`
		ExpectedQuoteVersion int64 `json:"expected_quote_version"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, err := s.orders.ConfirmQuote(r.Context(), orderID, userIDFrom(r),
		req.Payment, req.ExpectedTotal, req.ExpectedQuoteVersion)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := orderView(orders.AudienceCustomer, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleRejectQuote **الزبونُ يرفض السعرَ الجديد** (قرارُ المالك ٢٠٢٦-١٠-١٥).
//
// **لا جسمَ له** — الطلبُ يبقى حيث هو، وتُكتب علامةٌ يراها المكتب.
func (s *Server) handleRejectQuote(w http.ResponseWriter, r *http.Request) {
	o, err := s.orders.RejectQuote(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := orderView(orders.AudienceCustomer, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleAdminOverrideCustomQuote **الأدمنُ يعدّل عرضَ الطلب المخصَّص** — Batch 2a.
//
// (قرارُ المالك ٢٠٢٦-١٠-١٥: صعوداً ونزولاً حتّى التسليم، موثَّقاً بسبب — والزيادةُ
// بعد الاستلام تنتظر تأكيدَ الزبون. انظر `AdminOverrideCustomQuote`.)
//
// **والسببُ إلزاميّ**: تدخّلٌ ماليٌّ بتقدير إنسان، **وبلا كلمةٍ لا يُراجَع.**
// **والتوثيقُ في المحرّك** (`audit_log` داخلَ المعاملة) — لا في المعالِج.
func (s *Server) handleAdminOverrideCustomQuote(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Goods  int64  `json:"goods_amount"`
		Fee    int64  `json:"fee"`
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Goods < 0 || req.Fee < 0 || len(req.Reason) == 0 {
		s.respondErr(w, errValidation)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, err := s.orders.AdminOverrideCustomQuote(r.Context(), orderID, userIDFrom(r),
		req.Goods, req.Fee, req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}
