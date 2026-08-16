// Package server يبني موجّه HTTP وطبقة الوسطاء.
// البنية الطبقية الملزمة: handlers → services → repositories (GROUND-RULES §3).
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/config"
	"github.com/servacode/rahalgo/backend/internal/geo"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/notify"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/push"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/referrals"
	"github.com/servacode/rahalgo/backend/internal/routing"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type Server struct {
	cfg      *config.Config
	logger   *slog.Logger
	pg       *pgxpool.Pool
	rdb      *redis.Client
	tokens   *auth.TokenIssuer
	identity *identity.Service
	catalog  *catalog.Service
	settings *settings.Store
	// comms **حديثُ الطلب** — قناةٌ واحدةٌ لطرفيه بلا رقمٍ بينهما.
	comms      *comms.Service
	wallet     *wallet.Service
	orders     *orders.Service
	cashbox    *cashbox.Service
	support    *support.Service
	incentives *incentives.Service
	offers     *offers.Service
	referrals  *referrals.Service
	media      *media.Service
	hub        *realtime.Hub
	geo        *geo.Service
	// route **محرّكُ المسارات** — وفارغٌ يعني الخطَّ المستقيم.
	route     *routing.Client
	notify    *notifications.Service
	push      *push.Service
	otpStatus func() map[string]any
	// otpUnpair **فكُّ اقتران البوت** — وفارغةٌ لمزوّدٍ لا اقترانَ له.
	otpUnpair func(context.Context) error
	// otpPair **طلبُ رمزِ ربطٍ صريح** — ولا رمزَ بغيره.
	otpPair func()
	// textSender مُرسِلُ الرسائل إلى المتاجر — رسالةٌ نصّية اليوم، وواتسابٌ
	// رسميّ لاحقاً من الواجهة نفسها.
	textSender *notify.SMSSender
}

// SetTextSender يحقن مُرسِل الرسائل (يُنادى مرّة عند الإقلاع).
func (s *Server) SetTextSender(sender *notify.SMSSender) { s.textSender = sender }

func New(cfg *config.Config, logger *slog.Logger, pg *pgxpool.Pool, rdb *redis.Client,
	tokens *auth.TokenIssuer, identitySvc *identity.Service, catalogSvc *catalog.Service,
	settingsStore *settings.Store, walletSvc *wallet.Service, ordersSvc *orders.Service,
	cashboxSvc *cashbox.Service, supportSvc *support.Service, mediaSvc *media.Service,
	hub *realtime.Hub, otpStatus func() map[string]any,
	otpUnpair func(context.Context) error, otpPair func()) *Server {
	notify := notifications.New(pg, hub, logger)
	// ══════════════════════════════════════════════════════════════════
	// **والدفعُ يُربط إن كان مهيّأً — وإلّا صمتت المنصّةُ ولم تسقط**
	// ══════════════════════════════════════════════════════════════════
	//
	// **مفتاحٌ غائبٌ حالةٌ عاديّة**: جهازُ المطوّر، والمنصّةُ نفسُها حتّى
	// ٢٠٢٦-٠٨-١١. **ومفتاحٌ موجودٌ ومعطوبٌ ليس عاديّاً** — يُصرَّح به،
	// **وسكوتٌ عليه يعني منصّةً تظنّ أنّها تُرسل وهي لا تفعل.**
	//
	// **ولا يُسقط الإقلاعَ أيضاً**: محرّكٌ لا يقلع بسبب إشعاراتٍ أسوأُ من
	// إشعاراتٍ لا تصل. **إنّما يُسجَّل خطأً لا معلومة.**
	pushSvc := push.New(pg, logger)
	if fcm, err := push.NewFCM(logger); err != nil {
		logger.Error("الدفع: تعذّرت التهيئة — الإشعاراتُ لن تصل إلى الأجهزة", "error", err)
	} else if fcm != nil {
		pushSvc = push.New(pg, logger, fcm)
	}
	notify.SetPusher(pushAdapter{pushSvc})
	geoSvc := geo.New(cfg.GeocoderURL, rdb, logger)
	routeClient := routing.New(cfg.OSRMURL)
	if !routeClient.Enabled() {
		logger.Warn("المسارات: لا محرّك — المسافةُ بخطٍّ مستقيم")
	}
	// محرك الطلبات يحتاج الإشعارات (عمولة المندوب) وقد بُني قبلها — نحقنها الآن.
	ordersSvc.SetNotifier(notify)
	// وقواعدَ العمل من اللوحة: اشتراطُ توثيق واتساب قبل الطلب وما يليه.
	ordersSvc.SetSettings(settingsStore)
	// **والقائمةُ تحسب سعرَ البيع من الهامش** — انظر `pricing`.
	catalogSvc.SetSettings(settingsStore)
	srv := &Server{cfg: cfg, logger: logger, pg: pg, rdb: rdb, tokens: tokens,
		identity: identitySvc, catalog: catalogSvc, settings: settingsStore,
		comms:  comms.New(pg),
		wallet: walletSvc, orders: ordersSvc, cashbox: cashboxSvc, support: supportSvc,
		media: mediaSvc, hub: hub, otpStatus: otpStatus, otpUnpair: otpUnpair, otpPair: otpPair,
		notify: notify, push: pushSvc, geo: geoSvc, route: routeClient}
	// **والحوافزُ تعرف الخزينةَ من محرّك الطلبات** — مصدرٌ واحدٌ لمن هي،
	// **ولا تُقرأ مرّتين بطريقتين.**
	srv.incentives = incentives.New(pg, walletSvc, settingsStore, ordersSvc.TreasuryID)
	srv.incentives.SetLogger(logger)
	// **ومكافأةُ الهدف تُدفع عند التسليم** — محرّكُ الطلبات يناديها.
	ordersSvc.SetTargetGranter(srv.incentives)
	// **والخصمُ يُقرأ لحظةَ بناء الطلب** — لا من ذاكرةٍ محمّلة.
	srv.offers = offers.New(pg)
	ordersSvc.SetOffers(srv.offers)
	// **والخريطةُ تُسأل عن زمن الطريق لحظةَ الإسناد والاستلام** — تُلتقط
	// إجابتُها وتُجمَّد، **فلا تُعاد سؤالاً بعد أن يتحرّك السائق.**
	ordersSvc.SetRouter(orderRouter{srv})
	// **ومكافأةُ من دعا** — تُصرف عند أوّل طلبٍ يُسلَّم للمدعوّ.
	srv.referrals = referrals.New(pg, walletSvc, settingsStore, ordersSvc.TreasuryID, notify)
	ordersSvc.SetReferrals(srv.referrals)
	return srv
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// **وترويساتُ الأمان على كلّ ردّ** — قبل التوجيه، فتشمل الوسائطَ
	// والصحّةَ كما تشمل الواجهة. (انظر security_headers.go.)
	r.Use(securityHeaders)
	// **المهلةُ لكل طلبٍ إلّا قناة البثّ.**
	//
	// كانت تلفّ `/ws` معها، فتُلغي سياقَ الطلب بعد ثلاثين ثانية — و`handleWS`
	// يقرأ `<-ctx.Done()` فينصرف. **فكانت كلُّ شاشةٍ في المنصة تموت بثّياً بعد
	// نصف دقيقة من فتحها**: العمليات والمتجر والزبون والسائق.
	//
	// ولم يشتكِ شيء. الشاشةُ تبقى معروضةً بآخر ما جلبت، **والجمودُ يبدو هدوءاً**
	// — حتى يُحدّث أحدٌ الصفحةَ فيرى ما فاته.
	//
	// وأثرُه كان في السجلّ طوال الوقت: `WriteHeader on hijacked connection` —
	// الوسيطُ يحاول كتابة 504 على اتصالٍ خطفه WebSocket. **رسالةٌ قيلت ولم
	// تُقرأ.**
	// ══════════════════════════════════════════════════════════════════
	// **ونقاطُ الرفع تُعفى — مهلةُ الطلب تشمل زمنَ الرفع نفسِه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٥ على الخادم الجديد: «حدث خطأٌ غير متوقّع»
	//  عند رفع خلفيّة الموقع وصورة الحساب.)
	//
	// **والسجلُّ يقول `context deadline exceeded`** — لا «قرصٌ ممتلئ»
	// ولا «صلاحيّة»: **المهلةُ نفسُها انقضت.**
	//
	// **وثلاثون ثانيةً تُقاس من أوّل بايتٍ يصل** — لا من لحظة اكتمال
	// الجسد. **فصورةٌ بأربعة ميغا على وصلةٍ بطيئة تستهلكها في الرفع
	// وحدَه**، ثمّ يُقتل الطلبُ قبل أن تُعالَج الصورة.
	//
	// **وهي تحمي من معالجٍ يعلق، لا من زبونٍ يرفع** — وحدُّ الحجم
	// (`MaxBytesReader`) هو حارسُ الرفع، وقد وُضع.
	r.Use(exceptPaths(middleware.Timeout(30*time.Second),
		"/api/v1/ws",
		// **وما بدأ بنجمةٍ يُطابَق بذيله** — لأنّ في مسار الإثبات
		// معرّفَ طلبٍ لا يُعرف مسبقاً.
		"*/media", "*/proof", "*/app-file", "*/avatar",
	))
	// **ونوعُ العميل يُقرأ مرّةً هنا** — فتراه كلُّ نقطةٍ تُصدر جلسة
	// بلا أن تسأل عنه. (انظر `identity/client_kind.go`.)
	r.Use(clientKind)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: s.allowedOrigins(),
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// **و`X-RahalGo-Client` في المسموح** — ترويسةٌ غيرُ معلَنةٍ يرفضها
		// المتصفّحُ في الفحص المبدئيّ، **فيسقط النداءُ قبل أن يصل.**
		// (والتطبيقُ لا يمرّ بـCORS، لكنّ الويبَ قد يُصرّح بنفسه يوماً.)
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-RahalGo-Client"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", s.handleHealth)

	// الوسائط المرفوعة (صور عامة بأسماء uuid) — تخزين مؤقت طويل
	r.Handle("/media/*", http.StripPrefix("/media/", s.media.FileServer()))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ws", s.handleWS)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/otp/request", s.handleOTPRequest)
			r.Post("/otp/verify", s.handleOTPVerify)
			r.Post("/login", s.handlePasswordLogin)
			r.Post("/refresh", s.handleRefresh)
			r.Post("/logout", s.handleLogout)
			r.Post("/sso", s.handleSSO) // استبدال رمز التسليم بجلسة (عام)
			r.Post("/password/reset/request", s.handleResetRequest)
			// **والتحقّقُ من رمز الاستعادة قبل نموذج الكلمة الجديدة.**
			r.Post("/password/reset/verify", s.handleResetVerify)
			r.Post("/password/reset/confirm", s.handleResetConfirm)
			r.Post("/signup/request", s.handleSignupRequest) // إنشاء حساب زبون فقط
			// **والتحقّقُ من الرمز قبل النموذج** — لا يستهلكه ولا يُنشئ شيئاً.
			r.Post("/signup/verify", s.handleSignupVerify)
			r.Post("/signup/confirm", s.handleSignupConfirm)

			// ══════════════════════════════════════════════════════════
			// **رمزُ الأدمن — خطوةٌ ثانيةٌ عامّةٌ لأنّها قبل الجلسة**
			// ══════════════════════════════════════════════════════════
			//
			// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «رمزُ دخولٍ ثانٍ من ٤ أرقام… فقط
			//  للأدمن، لأنّه بنفس اللوحة تحسّباً للاختراق».)
			//
			// **ولا حارسَ جلسةٍ عليها** — من يبلغها لم تُصدَر له جلسةٌ بعد.
			// **وحارسُها التحدّي**: رمزٌ عشوائيٌّ يعرف صاحبَه، يعيش خمسَ
			// دقائقَ ويُستهلك مرّة، **ولا يُمنَح إلّا لمن عرف كلمةَ المرور.**
			r.Post("/pin", s.handlePinVerify)
			r.Post("/pin/setup", s.handlePinSetup)

			r.Group(func(r chi.Router) {
				r.Use(s.RequireAuth)
				// **وتبديلُ الرمز واستعادتُه من داخل الجلسة** — يبدّله صاحبُه
				// وهو داخل، **ولا يُبدَّل بلا القديم.**
				r.Get("/pin/state", s.handlePinState)
				r.Post("/pin/change", s.handlePinChange)
				r.Post("/pin/reset/request", s.handlePinResetRequest)
				r.Post("/pin/reset/confirm", s.handlePinResetConfirm)
				r.Get("/me", s.handleMe)
				r.Post("/password", s.handleSetPassword)
				r.Get("/my-logins", s.handleMyLogins)
				// **رمزُ دعوته ورابطُه** — ومن جلب يُكافأ.
				r.Get("/referral", s.handleMyReferral)
				r.Post("/handoff", s.handleHandoff) // إنشاء رمز تسليم SSO
				r.Post("/phone/request", s.handlePhoneChangeRequest)
				r.Post("/phone/confirm", s.handlePhoneChangeConfirm)
				// توثيق رقم واتساب — قناة التواصل، لا هوية الدخول
				r.Post("/whatsapp/request", s.handleWhatsAppVerifyRequest)
				r.Post("/whatsapp/confirm", s.handleWhatsAppVerifyConfirm)
				// حذف الحساب — بتأكيد رمز على هاتف صاحبه
				r.Post("/account/delete/request", s.handleDeleteAccountRequest)
				r.Post("/account/delete/confirm", s.handleDeleteAccountConfirm)
			})
		})

		// واجهة التصفح العامة — بلا حساب
		// **والعروضُ عامّةٌ كالتصفّح** — تُرى قبل الدخول، **ومن رأى عرضاً سجّل.**
		r.Get("/public/offers", s.handlePublicOffers)
		r.Get("/public/home", s.handlePublicHome)
		r.Get("/public/hero", s.handlePublicHero)
		// **هويّةُ المنصة** — خفيفةٌ ومفتوحة، تناديها الخمسةُ وشاشةُ الدخول.
		r.Get("/public/platform", s.handlePublicPlatform)
		// **وأسلوبُ الخريطة** — يقرؤه العارضُ والمنزِّلُ معاً.
		r.Get("/public/map-style.json", s.handleMapStyle)
		// **وتنزيلُ التطبيق عامٌّ** — يُضغط قبل أن يكون هناك حساب.
		r.Get("/public/app", s.handleDownloadApp)
		r.Get("/public/zone", s.handlePublicZone)
		// **هويّةُ المنصة للشروط والخصوصية** — عامّةٌ لأنّ من يقرؤها قد لا
		// يكون دخل بعد، **ومن سُئل أن يوافق قبل أن يقرأ لم يوافق.**
		r.Get("/public/contact", s.handlePublicContact)
		// **تسعيرةُ السلّة قبل الطلب** — رقمٌ يتغيّر أمام العين يُقبل، ورقمٌ
		// يظهر عند الدفع يُراجَع. (انظر `quote_handlers.go`)
		r.Post("/public/quote", s.handleQuote)
		// البحث عام كالتصفّح — من يشتهي صنفاً لا يعرف اسم المتجر الذي يصنعه
		// **التصفّحُ بالأصناف لا بالمتاجر** — الزبونُ يشتهي شاورما ولا يعرف
		// من يصنع أفضلَها. (انظر `sections_handlers.go`)
		r.Get("/public/sections", s.handlePublicSections)
		r.Get("/public/sections/{id}/items", s.handlePublicSectionItems)
		r.Get("/public/items/{id}", s.handlePublicItem)
		r.Get("/public/search/items", s.handleSearchItems)
		r.Get("/public/search", s.handlePublicSearch)
		r.Get("/public/invite", s.handlePublicInvite)
		r.Post("/public/join", s.handlePublicJoin)

		// نقاط الزبون — الطلب حصراً من هنا (قرار 18)
		r.Group(func(r chi.Router) {
			r.Use(s.RequireAuth)
			// **ومحميٌّ من الإعادة** — انظر `idempotency.go`. **أهمُّ فعلٍ
			// في تطبيق الزبون**: ضغطةٌ على شبكةٍ سيّئة تُنشئ طلبين وخصمين.
			r.Post("/orders", s.idempotent(s.handleCustomerCreateOrder))
			// **والطلبُ الخاصّ** — ما ليس في المنصّة. (قرارُ المالك ٢٠٢٦-٠٨-٠٩.)
			r.Post("/orders/custom", s.handleCreateCustomOrder)
			// **أثرُ كود الخصم قبل الطلب** — والقواعدُ نفسُها لا نسخةٌ منها.
			r.Post("/promo/preview", s.handlePromoPreview)
			r.Post("/orders/{id}/rating", s.handleRateOrder)
			// إلغاء الزبون — كان حقّاً في خارطة الحالات بلا باب يوصله
			r.Post("/orders/{id}/cancel", s.handleCustomerCancelOrder)

			// ── حديثُ الطلب — بابٌ واحدٌ لطرفيه ──────────────────────
			//
			// **وهو هنا لا في مجموعة السائق ولا مجموعة الزبون**: المسارُ
			// واحدٌ يناديه الاثنان، **والخادمُ يعرف أيَّهما من جلسته.**
			//
			// **ومساران متطابقان يفترقان**: يُصلَح شرطٌ في أحدهما ويُنسى
			// الآخر، **فيقرأ السائقُ ما لا يقرؤه الزبونُ من الحديث نفسِه.**
			//
			// **ولا دورَ يُشترَط هنا** — `Permit` تردّ ٤٠٤ لمن ليس طرفاً،
			// **وحارسُ الدور يمنع سائقاً أن يقرأ حديثَ طلبه** لو وُضع.
			// **وإنذاراتي — يراها صاحبُها أيَّ دورٍ كان.**
			r.Get("/my/warnings", s.handleMyWarnings)
			// **وسجلُّ المحادثات** — المفتوحةُ والمنتهية، حجّةً عند الخلاف.
			r.Get("/my/chats", s.handleMyChats)
			r.Get("/orders/{id}/messages", s.handleOrderMessages)
			r.Post("/orders/{id}/messages", s.handleSendOrderMessage)
			r.Get("/my/orders", s.handleMyOrders)
			r.Get("/my/orders/{id}", s.handleMyOrder)
			r.Get("/my/wallet", s.handleMyWallet)
			// **بابُ الشكوى عند الطلب لا في رقم هاتف.**
			//
			// كانت التذاكرُ تُفتح من لوحة الإدارة وحدَها: يتّصل الزبونُ فيفتح
			// موظّفٌ تذكرةً بالنيابة عنه. **فمن لم يجد من يردّ لم يجد باباً** —
			// وشكواه تضيع ونحن لا نعلم أنّها وقعت. **والصمتُ يُقرأ رضاً وهو
			// ليس رضاً.**
			r.Get("/my/orders/{id}/complaint-reasons", s.handleComplaintReasons)
			r.Get("/my/orders/{id}/complaint", s.handleMyComplaint)
			r.Post("/my/orders/{id}/complaint", s.handleOpenComplaint)
			// عناوينه المحفوظة — يكتبها مرّة ويستعملها دائماً
			r.Get("/my/addresses", s.handleMyAddresses)
			r.Post("/my/addresses", s.handleCreateAddress)
			r.Delete("/my/addresses/{id}", s.handleDeleteAddress)
			r.Post("/my/addresses/{id}/default", s.handleSetDefaultAddress)
			// المفضّلة — زرٌّ واحد ينقلب، فنقطةٌ واحدة تقلبه
			r.Get("/my/favorites", s.handleMyFavorites)
			r.Post("/my/favorites/{id}", s.handleToggleFavorite)
			// بيانات التوب بار الموحّدة لأي مستخدم (اسم، صورة، رصيد) وإدارة صورته
			r.Get("/me/summary", s.handleMeSummary)
			r.Patch("/me/name", s.handleSetMyName)
			r.Post("/me/avatar", s.handleMyAvatar)
			r.Delete("/me/avatar", s.handleDeleteMyAvatar)
			r.Get("/my/ratings", s.handleMyRatings)
			// **شكاواه هو** — والتذاكرُ كلُّها كانت تحت /admin، فلا يرى صاحبُها حالَها
			r.Get("/my/tickets", s.handleMyTickets)
			r.Get("/me/reputation", s.handleMeReputation)
			r.Get("/me/notifications", s.handleMyNotifications)
			// **أجهزةُ الدفع — لكلّ دورٍ لا للسائق وحدَه.**
			//
			// **الزبونُ ينتظر «طلبُك في الطريق»، والمتجرُ ينتظر طلباً**،
			// **والسائقُ ينتظر عرضاً.** وثلاثتُهم يغلقون الشاشة.
			// ══════════════════════════════════════════════════════
			// **وفحصُ الإشعارات — يقول لماذا لا يرنّ**
			// ══════════════════════════════════════════════════════
			//
			// (وقع ٢٠٢٦-٠٨-١٤: جُرّب على جهازٍ حقيقيٍّ فلم يرنّ،
			//  **والجهازُ مسجَّلٌ والمفتاحُ مضبوطٌ والمشروعُ متطابق** —
			//  ولم يكن في المنصّة ما يقول أين وقفت الرسالة.)
			//
			// **ولكلّ حسابٍ على نفسه** — لا للإدارة وحدَها: **السائقُ
			// الذي لا تصله إشعاراتٌ يفحص بنفسه** ويقول للمكتب ما ردّ.
			r.Post("/me/devices/test", s.handlePushTest)
			r.Post("/me/devices", s.handleDeviceRegister)
			r.Delete("/me/devices", s.handleDeviceUnregister)
			// العنونة: مساعدة لتحديد المواقع — لأي مستخدم مسجّل
			r.Get("/geo/reverse", s.handleGeoReverse)
			r.Get("/geo/search", s.handleGeoSearch)
			r.Post("/me/notifications/read", s.handleMarkNotificationRead)
			// طلبات سحب الرصيد — لأي صاحب رصيد (مندوب اليوم، سائق مع تطبيقه)
			r.Get("/me/payouts", s.handleMyPayouts)
			r.Post("/me/payouts", s.handleCreatePayout)
		})

		// لوحة المندوب — دور المبيعات حصراً (قراءة: كوده ومتاجره وعمولاته)
		r.Route("/rep", func(r chi.Router) {
			r.Use(s.RequireAuth)
			r.Use(s.RequireRoles("sales"))
			r.Get("/me", s.handleRepMe)
			r.Get("/merchants", s.handleRepMerchants)
			// تفاصيل عميل: طلباته وعمولة المندوب عن كلٍّ منها — شفافية العمولة
			r.Get("/merchants/{id}", s.handleRepMerchantDetail)
			r.Get("/wallet", s.handleRepWallet)
			r.Get("/leads", s.handleRepLeads)
			// يسجّل عميلاً باسمه من الميدان — يبقى معلّقاً حتى موافقة الإدارة
			r.Post("/leads", s.handleRepCreateLead)
			r.Get("/categories", s.handleListCategories) // تصنيفات المتاجر للنموذج
			// **وهدفُ المندوب كهدف السائق** — المقياسُ يختلف والمعنى واحد.
			r.Get("/incentives", s.handleMyIncentives)
		})

		// بوابة السائق — كان الطرف الوحيد بلا باب رغم أن الخارطة تخوّله سبعة انتقالات
		r.Route("/driver", func(r chi.Router) {
			r.Use(s.RequireAuth)
			r.Use(s.RequireRoles("driver"))
			r.Get("/me", s.handleDriverMe)
			r.Post("/shift", s.handleDriverShift)
			// **موضعُه — يُرسله هو ولا يُخمَّن.** ومنه تُقاس المسافةُ إلى
			// المتجر، **والترتيبُ بالدور عدلٌ في الوقت أعمى في المكان.**
			r.Post("/location", s.handleDriverLocation)
			// **ودفعةٌ لِما جُمع بلا شبكة** — انظر `driver_location_batch.go`.
			// **والمفردةُ تبقى للمتصفّح**: نبضةٌ في ثانيتها لا تحتاج طابوراً.
			r.Post("/location/batch", s.handleDriverLocationBatch)
			r.Get("/queue", s.handleDriverQueue)
			// **أسبابُ التعذّر من الخادم** — قائمةٌ تُكرَّر في مكانين تفترق
			// حين يُضاف سببٌ في أحدهما (driver_return.go)
			r.Get("/fail-reasons", s.handleFailReasons)
			// **الطارئ** — ضغطةٌ واحدة: موقعٌ يُلتقط، وعملياتٌ تُنبَّه، وطلبٌ
			// يُحرَّر. **ومن كُسرت يدُه لا يملأ ثلاث شاشات.**
			r.Post("/orders/{id}/emergency", s.handleDriverEmergency)
			// **إثباتُ التسليم** — صورةٌ وإحداثياتٌ ووقت. والمعيارُ العالميّ
			// ثلاثةٌ لا واحد. (انظر `delivery_proof.go`)
			r.Post("/orders/{id}/proof", s.handleDeliveryProof)
			r.Post("/orders/{id}/proof/skip", s.handleSkipDeliveryProof)
			// **إرجاعُ البضاعة** — لمتاجرِ الاسترداد وحدها
			r.Post("/orders/{id}/return", s.handleDriverReturn)
			// **وتوثيقُ ما اتُّفق عليه في الطلب الخاصّ** — بعد المحادثة.
			r.Post("/orders/{id}/agree", s.handleAgreeCustom)
			r.Get("/orders", s.handleDriverOrders)
			// **مسارُ الطرف الحاليّ** — خطُّ الشوارع ومسافتُه ومدّتُه.
			// **ونداءٌ وحدَه لا حقلٌ في القائمة**: القائمةُ خمسةُ طلبات
			// تُقرأ كلَّ ثوان، **والمسارُ يلزم لواحدٍ في يده.**
			r.Get("/orders/{id}/route", s.handleDriverOrderRoute)
			// **سجلُّه** — ما نفّذه نجح أم فشل. **وما انتهى كان يختفي**، فلا
			// يجد طلباً يتذكّره ليُبلّغ عنه. (انظر `driver_history.go`)
			r.Get("/orders/history", s.handleDriverHistory)
			r.Get("/orders/report-reasons", s.handleDriverReportReasons)
			r.Post("/orders/{id}/report", s.handleDriverReport)
			// **ومن وقف عند بابه يقيّمه** — الزبونُ يرى الطعامَ ولا يرى
			// المطبخ. (`merchant_rating_handlers.go`)
			r.Post("/orders/{id}/rate-merchant", s.handleDriverRateMerchant)
			r.Post("/orders/{id}/accept", s.handleDriverAccept)
			// **والرفضُ ينقل الدورَ فوراً** — بدل انتظار انقضاء المهلة.
			r.Post("/orders/{id}/decline", s.handleDriverDecline)
			r.Post("/orders/{id}/transition", s.handleDriverTransition)
			r.Post("/orders/{id}/release", s.handleDriverRelease)
			r.Get("/cash", s.handleDriverCash)
			// **هدفُه وما ناله** — ومكافأةٌ لا يراها صاحبُها لم تُصرف في نظره.
			r.Get("/incentives", s.handleMyIncentives)
		})

		// بوابة المتجر — صاحب المتجر حصراً، وكل نقطة تتحقق من الملكية
		r.Route("/merchant", func(r chi.Router) {
			r.Use(s.RequireAuth)
			r.Use(s.RequireRoles("merchant"))
			// **وصورةُ الصنف تُرفع من بوّابته** — كان الرفعُ للإدارة وحدَها،
			// **فصاحبُ المطعم لا يملك أن يضع صورةً لصنفه.** (طلبُ المالك
			// ٢٠٢٦-٠٨-٠٧.) والأنواعُ محصورةٌ في `merchantKinds`.
			r.Post("/media", s.handleMerchantUploadMedia)
			r.Get("/stores", s.handleMerchantStores)
			// **ومن أُنذر يرى إنذارَه.** إشعارٌ يمرّ في الشريط يُقرأ مرّةً
			// ويُنسى، **ثمّ يُحظر المتجرُ ولم يعلم أنّ عليه شيئاً.**
			r.Get("/warnings", s.handleMerchantWarnings)
			r.Get("/stores/{id}/orders", s.handleMerchantOrders)
			r.Get("/stores/{id}/menu", s.handleMerchantMenu)
			r.Get("/stores/{id}/reports", s.handleMerchantReports)
			r.Post("/stores/{id}/emergency", s.handleMerchantEmergency)
			r.Get("/orders/{id}", s.handleMerchantGetOrder)
			r.Post("/orders/{id}/transition", s.handleMerchantTransition)
			// حلقة المطبخ: الجاهزية علامةٌ يرفعها من يعرف، وساعاته وإعداداته بيده
			r.Post("/orders/{id}/ready", s.handleMerchantReady)
			// ══════════════════════════════════════════════════════════
			// **وبابُ شكوى المتجر** — (قرارُ المالك ٢٠٢٦-٠٨-١٦)
			// ══════════════════════════════════════════════════════════
			//
			// **وكان يُشتكى عليه ولا يشتكي**: يُنذَر ويُحظَر بعدّاد
			// مخالفات، **ولا يُسمع منه.**
			r.Get("/report-reasons", s.handleMerchantReportReasons)
			r.Post("/orders/{id}/report", s.handleMerchantReport)
			r.Get("/stores/{id}/hours", s.handleMerchantGetHours)
			r.Put("/stores/{id}/hours", s.handleMerchantSetHours)
			r.Patch("/stores/{id}/settings", s.handleMerchantSettings)
			r.Patch("/menu/items/{itemID}/availability", s.handleMerchantItemAvailability)
			// القائمة بضاعته: يضيف ويعدّل ويحذف بنفسه — الحارس مختلف والعملية واحدة
			r.Post("/stores/{id}/menu/sections", s.handleMerchantCreateSection)
			// **والقسمُ يُعدَّل** — كان يُنشأ ويُحذف لا غير، فمن أخطأ حرفاً في
			// اسمه لم يملك تصحيحَه، **وأصنافُه تمنع حذفَه.** (٢٠٢٦-٠٨-٠٧.)
			r.Patch("/menu/sections/{sectionID}", s.handleMerchantUpdateSection)
			r.Delete("/menu/sections/{sectionID}", s.handleMerchantDeleteSection)
			// أقسامُ السوق ليختار الصنفُ موضعَه — **والمراجعةُ تبقى الحارس.**
			r.Get("/platform-sections", s.handleMerchantPlatformSections)
			r.Post("/stores/{id}/menu/items", s.handleMerchantCreateItem)
			r.Patch("/menu/items/{itemID}", s.handleMerchantUpdateItem)
			r.Delete("/menu/items/{itemID}", s.handleMerchantDeleteItem)
		})

		// نقاط الإدارة — أدمن/عمليات فقط، والتعديلات الحساسة للأدمن حصراً
		r.Route("/admin", func(r chi.Router) {
			r.Use(s.RequireAuth)
			// المالية عضو في مكتب المنصة: كانت المجموعة تسمح للأدمن والعمليات فقط،
			// فتُحجب المالية عند الباب — وتصير المسارات المعلَّمة "أدمن/مالية"
			// (المحفظة، تسوية الصندوق، حلّ التذاكر، صرف السحوبات) غير قابلة للوصول
			// لمن أُنشئت له. الحراسة الدقيقة تبقى على كل مسار حسّاس بذاته.
			r.Use(s.RequireRoles("admin", "ops", "finance"))
			r.Get("/whatsapp", func(w http.ResponseWriter, _ *http.Request) {
				httpx.JSON(w, http.StatusOK, s.otpStatus())
			})
			// **وفكُّ الاقتران بابُ إعادة الربط** — (قرارُ المالك ٢٠٢٦-٠٨-١٠:
			// «إذا تمّ فصلُ الاقتران لا يوجد زرٌّ لإعادة ربط الجهاز»).
			//
			// **والرمزُ لا يُولَّد إلّا لجهازٍ بلا هويّة** — فمن أراد ربطاً
			// جديداً يفكّ القديمَ أوّلاً. **وللأدمن وحدَه**: فكُّه يوقف كلَّ
			// رموز التحقّق حتّى يُمسح رمزٌ جديد.
			r.With(s.RequireRoles("admin")).
				Post("/whatsapp/unpair", func(w http.ResponseWriter, r *http.Request) {
					if s.otpUnpair == nil {
						s.respondErr(w, httpx.ErrNotFound)
						return
					}
					if err := s.otpUnpair(r.Context()); err != nil {
						s.respondErr(w, err)
						return
					}
					s.audit(r, "admin.whatsapp_unpair", "platform", "", map[string]any{})
					httpx.JSON(w, http.StatusOK, map[string]any{"unpaired": true})
				})
			// ══════════════════════════════════════════════════════════
			// **وطلبُ الرمز صريحٌ — ولا يُولَّد وحدَه**
			// ══════════════════════════════════════════════════════════
			//
			// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الكودُ لا يجب أن يظهر تلقائيّاً —
			//  يظهر حين أطلبه بزرّ ربطِ جهاز، هيك الأصول».)
			//
			// **ورمزٌ يُولَّد بلا طلبٍ ينتهي بلا مسح** — فيُسجَّل «انتهت
			// المهلة» كلَّ دقيقتين، **ويقرؤه صاحبُ المنصّة عطباً دائماً.**
			//
			// **وهو بابٌ يُفتح لا مجرّدَ زرّ**: من مسح رمزاً معروضاً على
			// شاشةٍ منسيّةٍ ربط هاتفَه هو ببوت المنصّة.
			r.With(s.RequireRoles("admin")).
				Post("/whatsapp/pair", func(w http.ResponseWriter, r *http.Request) {
					if s.otpPair == nil {
						s.respondErr(w, httpx.ErrNotFound)
						return
					}
					s.otpPair()
					s.audit(r, "admin.whatsapp_pair", "platform", "", map[string]any{})
					httpx.JSON(w, http.StatusOK, map[string]any{"pairing": true})
				})

			r.Post("/media", s.handleUploadMedia)
			r.Get("/users", s.handleAdminListUsers)
			r.Get("/users/stats", s.handleAdminUserRoleCounts)
			r.Get("/users/export", s.handleAdminUsersExport)
			r.Get("/users/{id}", s.handleAdminGetUser)
			r.Get("/users/{id}/activity", s.handleAdminUserActivity)
			r.Get("/users/{id}/feedback", s.handleAdminUserFeedback)
			r.Get("/users/{id}/financials", s.handleAdminUserFinancials)
			// **وإنذاراتُ الحساب — لأيّ دورٍ كان.**
			//
			// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «اجعله لكلّ الأدوار حتّى الزبون».)
			r.Get("/users/{id}/warnings", s.handleAdminUserWarnings)
			r.Get("/users/{id}/warn-reasons", s.handleWarnReasons)
			r.Get("/categories", s.handleListCategories)
			// **أقسامُ المنصة** — ما نبيعه، لا من نشتري منه.
			r.Get("/sections", s.handleListPlatformSections)
			// **وأصنافُ القسم كما هي** — لا كما يراها الزبون: من يفتح قسماً
			// ليقرّر إطفاءَه يريد ما فيه كلَّه، **بما لا يظهر ولماذا.**
			r.Get("/sections/{id}/items", s.handleSectionItems)
			r.Get("/merchants", s.handleListMerchants)
			// **ومتجرٌ بعينه لملفّه** — كان يُبحث عنه بالاسم في القائمة،
			// **ومتجران متشابها الاسم يُخلطان.**
			r.Get("/merchants/{id}", s.handleAdminGetMerchant)
			r.Get("/merchants/{id}/menu", s.handleGetMenu)
			r.Get("/merchants/{id}/hours", s.handleGetHours)
			r.Get("/merchants/{id}/warnings", s.handleAdminMerchantWarnings)
			r.Get("/zones", s.handleListZones)
			r.Get("/promos", s.handleListPromos)
			r.Get("/banners", s.handleListBanners)
			r.Get("/settings", s.handleListSettings)
			r.Get("/stats", s.handleAdminStats)
			r.Get("/reports", s.handleReports)
			// سجلّ الأحداث — للأدمن والمالية دون العمليات: يحوي مبالغ التعويضات
			// والسحوبات وأرصدة المحافظ، وموظّف العمليات ليس طرفاً في المال.
			r.With(s.RequireRoles("admin", "finance")).
				Get("/audit", s.handleAdminAudit)
			// حاملو الخزينة المحتملون — للأدمن وحده (merchant_violations.go)
			r.With(s.RequireRoles("admin")).
				Get("/treasury-candidates", s.handleTreasuryCandidates)
			r.Get("/users/{id}/wallet", s.handleAdminWalletStatement)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/users/{id}/wallet", s.idempotent(s.handleAdminWalletApply))
			// **وعناوينُه في ملفّه** — من يتابع شكوى «لم يصلني» يحتاج أن يرى
			// أين يسكن قبل أن يسأل.
			r.Get("/users/{id}/addresses", s.handleAdminUserAddresses)
			// **وأحاديثُه في ملفّه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥): من يحكم
			// في نزاعٍ يعرف اسمَ الإنسان لا رقمَ الطلب.
			r.Get("/users/{id}/chats", s.handleAdminUserChats)

			// الطلبات — غرفة العمليات تدير ولا تُنشئ (قرار 18):
			// الإنشاء حصراً عبر واجهات الزبون (الموقع/التطبيق)
			r.Get("/orders", s.handleListOrders)
			r.Get("/orders/alerts", s.handleOrderAlerts)
			r.Get("/orders/{id}", s.handleGetOrder)
			// **وحديثُ طرفيه — يُقرأ ولا يُكتب.** (قرارُ المالك ٢٠٢٦-٠٨-١٠:
			// «في حال حصول أيّ تجاوزٍ يمكننا الرجوع إليه».)
			//
			// **ومسارٌ ثانٍ لا توسعةُ صلاحية الطرفين**: لو صارت الإدارةُ طرفاً
			// لَاستطاعت أن تكتب، **فيقرأ الزبونُ سطراً باسم سائقه لم يقله.**
			r.Get("/orders/{id}/chat", s.handleAdminOrderChat)
			// إرسال الطلب إلى المتجر على واتساب — في وضع «المنصة تدير»
			r.Get("/orders/{id}/message", s.handleOrderMessagePreview)
			r.Post("/orders/{id}/whatsapp", s.handleSendOrderToMerchant)
			// **تفصيلُ مال الطلب** — مقروءاً من الدفتر (order_breakdown.go).
			// للمالية والأدمن: يحوي أنصبةَ الأطراف وربحَ المنصة.
			r.With(s.RequireRoles("admin", "finance")).
				Get("/orders/{id}/breakdown", s.handleOrderBreakdown)
			// **وتصحيحُ تسويةٍ قديمة بقيدٍ مقابل** — لا بتصفير البيانات.
			// **والأدمنُ وحدَه**: قيدٌ ماليٌّ يُنشأ بيد.
			r.With(s.RequireRoles("admin")).
				Post("/orders/{id}/recompute", s.handleRecomputeSettlement)
			// **تحويلُ الطلب إلى متجرٍ آخر** — قاعدةٌ احتياطية، وسعرُ الزبون
			// لا يُمسّ. (انظر `order_transfer.go`)
			r.With(s.RequireRoles("admin", "ops")).
				Post("/orders/{id}/transfer", s.handleTransferOrder)
			r.Post("/orders/{id}/transition", s.handleOrderTransition)
			r.Post("/orders/{id}/assign", s.handleOrderAssign)

			// **ما بعد فشل الطلب** — من يحمل الخسارة (failure_aftermath.go).
			//
			// التعويضُ للمالية والأدمن لا للعمليات: **مالٌ يخرج من المنصة
			// بتقدير إنسان**، وموظّفُ العمليات ليس طرفاً في المال — وهو
			// الفصلُ نفسه المطبَّق على سجلّ الأحداث وحركات المحفظة.
			r.With(s.RequireRoles("admin", "finance")).
				Post("/orders/{id}/compensate-driver", s.handleCompensateDriver)
			// **المكافآتُ والعقوبات** — مالٌ يخرج بتقدير إنسان،
			// **وموظّفُ العمليات ليس طرفاً في المال**: الحارسُ نفسُه الذي
			// على تعويض السائق.
			r.With(s.RequireRoles("admin", "finance")).
				Post("/users/{id}/incentive", s.idempotent(s.handleIncentiveGrant))
			// ومصيرُ البضاعة تحسمه العملياتُ: **هي من يستلمها في المكتب**
			// وتعرف أاستردّها المتجرُ أم رفض. والقيدُ المالي يتبع قرارَها.
			r.Post("/orders/{id}/settle-goods", s.handleSettleGoods) // مهجورة — 410
			r.Post("/orders/{id}/goods", s.handleGoods)

			// الأقسام التشغيلية لكل دور (قرار 16)
			// **العروضُ والخصومات** — لافتةٌ تُرى وخصمٌ يُطبَّق في الدفتر.
			r.Get("/offers", s.handleAdminOffers)
			r.With(s.RequireRoles("admin")).Post("/offers", s.handleCreateOffer)
			r.With(s.RequireRoles("admin")).Post("/offers/{id}/active", s.handleSetOfferActive)
			r.Get("/customers", s.handleListCustomers)
			// **الأهدافُ تُقرأ ولا تُدفع** — تقول من بلغ، ولا تُعطي.
			r.Get("/incentives/{role}", s.handleIncentiveStandings)
			r.Get("/users/{id}/incentives", s.handleIncentiveList)
			r.Get("/salesreps", s.handleListSalesReps)
			r.Get("/leads", s.handleAdminLeads)
			r.Post("/leads/{id}/status", s.handleAdminLeadStatus)

			// طلبات سحب الرصيد: القراءة لمكتب المنصة، والصرف للأدمن والمالية
			r.Get("/payouts", s.handleAdminPayouts)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/payouts/{id}/decide", s.idempotent(s.handleDecidePayout))

			// التذاكر والتعويضات — الحل المالي للأدمن/المالية حصراً
			r.Get("/tickets", s.handleListTickets)
			r.Post("/tickets", s.handleCreateTicket)
			r.Get("/tickets/{id}", s.handleGetTicket)
			r.Post("/tickets/{id}/replies", s.handleTicketReply)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/tickets/{id}/resolve", s.handleTicketResolve)

			// السائقون والصندوق النقدي
			r.Get("/drivers", s.handleListDrivers)
			r.Get("/drivers/{id}/cash", s.handleDriverCashStatement)
			// **وإغلاقُ دوامٍ نُسي** — من ذهب إلى بيته وعلَمُه مرفوعٌ يبقى في
			// الدور، فيتأخّر كلُّ طلبٍ بمقدار غيابه. **إغلاقٌ فقط لا تشغيل.**
			r.Post("/drivers/{id}/end-shift", s.handleAdminEndShift)
			// **ما في الشارع مجموعاً** — مالٌ لا يُرى مجموعاً لا يُطالَب به.
			r.Get("/cash/outstanding", s.handleCashOutstanding)
			// **نزاعاتُ المنصة مع الأربعة** — المتجرِ والسائقِ والمندوبِ
			// والزبون. **ونزاعٌ لا يُرى مجموعاً لا يُتابَع**، وثلاثةٌ منها لم
			// يكن لها مكانٌ إطلاقاً قبل هجرة `0063`.
			// **إعلانُ المنصة** — أن تخاطب أهلَها بلا واقعةٍ تُولّده.
			// **والأدمنُ وحدَه**: صوتُ المنصة لا يُعار.
			// **وتصديرُ الأصل لا المجاميع** — من شكّ في مجموعٍ عاد إلى السطور.
			// وللمالية والأدمن: فيه أنصبةُ الأطراف وربحُ المنصة.
			r.With(s.RequireRoles("admin", "finance")).
				Get("/orders/export", s.handleOrdersExport)
			r.With(s.RequireRoles("admin", "finance")).
				Get("/ledger/export", s.handleLedgerExport)
			// **التقييماتُ مجموعةً** — «أيُّ سائقٍ يشكو منه الناس؟» سؤالٌ لا
			// جوابَ له إلّا بفتح عشرين ملفّاً، **فلا يُفتح فلا يُعرف.**
			r.Get("/ratings", s.handleAdminRatings)
			// **طابورُ مراجعة القائمة** — يعمل حين يُرفع مفتاحُ
			// `merchants.menu_requires_approval`، وكان المفتاحُ يَعِد ولا يفعل.
			r.Get("/menu/pending", s.handlePendingMenuItems)
			r.With(s.RequireRoles("admin")).
				Post("/menu/items/{itemID}/review", s.handleReviewMenuItem)
			r.Get("/broadcast/count", s.handleBroadcastCount)
			r.With(s.RequireRoles("admin")).Post("/broadcast", s.handleBroadcast)

			// **وملفُّ التطبيق قرارُ هويّةٍ لا قرارُ مال** — للإدارة وحدَها.
			r.With(s.RequireRoles("admin")).Post("/app-file", s.handleUploadAppFile)
			r.With(s.RequireRoles("admin")).Delete("/app-file", s.handleDeleteAppFile)
			r.Get("/disputes", s.handleListDisputes)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/disputes", s.handleCreateDispute)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/disputes/{id}/settle", s.handleSettleDispute)
			// **الطوارئُ مجموعةً** — ولا تُغلق بمرور الوقت: طارئٌ يختفي وحدَه
			// يُنسى، **ومن سأل عنه بعد يومين لم يجد من يقول ماذا جرى.**
			r.Get("/emergencies", s.handleOpenEmergencies)
			r.Post("/emergencies/{id}/resolve", s.handleResolveEmergency)
			// **الخسارةُ الفعلية من الدفتر** — لا من إعادة حسابٍ لما حُسب.
			r.Get("/reports/losses", s.handlePlatformLosses)
			// **والأرباحُ بتبويباتها** — (قرارُ المالك ٢٠٢٦-٠٨-١٦):
			// المنصّةُ والزبائنُ والمندوبون والسائقون والمتاجر.
			//
			// **وللمالك والماليّة** — فيها أنصبةُ الناس وأرباحُ المنصّة.
			r.With(s.RequireRoles("admin", "finance")).
				Get("/profits", s.handleProfits)
			// ══════════════════════════════════════════════════════════
			// **ومصروفاتُ التشغيل** — (قرارُ المالك ٢٠٢٦-٠٨-١٦)
			// ══════════════════════════════════════════════════════════
			//
			// **إيجارُ المكتب والرواتبُ والكهرباء** — كلفةُ تشغيلٍ
			// مخطَّطة، **لا خسارةَ عملٍ**. **وتخرج من الخزينة** بقرار
			// المالك: «مصروفٌ تابعٌ للمنصّة».
			//
			// **وللمالك والماليّة** — كسائر ما يمسّ المال.
			// **والقراءةُ محجوبةٌ عن العمليات كذلك** — **وموظّفُ
			// العمليات ليس طرفاً في المال**، وهو الفصلُ نفسُه المطبَّق
			// على الخزينة والخسائر وتعويض السائق.
			r.With(s.RequireRoles("admin", "finance")).
				Get("/expenses", s.handleListExpenses)
			r.With(s.RequireRoles("admin", "finance")).
				Get("/expenses/categories", s.handleExpenseCategories)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/expenses", s.handleCreateExpense)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/expenses/categories", s.handleSaveExpenseCategory)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/expenses/{id}/void", s.handleVoidExpense)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/drivers/{id}/settle", s.idempotent(s.handleDriverSettle))
			r.Group(func(r chi.Router) {
				r.Use(s.RequireRoles("admin"))
				r.Post("/sections", s.handleCreatePlatformSection)
				r.Patch("/sections/{id}", s.handleUpdatePlatformSection)
				r.Delete("/sections/{id}", s.handleDeletePlatformSection)
				r.Post("/users", s.handleAdminCreateUser)
				r.Patch("/users/{id}", s.handleAdminUpdateUser)
				r.Post("/users/{id}/roles", s.handleAdminGrantRole)
				r.Post("/users/{id}/password", s.handleAdminResetPassword)
				r.Post("/users/{id}/logout-all", s.handleAdminLogoutAll)
				r.Delete("/users/{id}/roles/{role}", s.handleAdminRevokeRole)
				r.Post("/categories", s.handleCreateCategory)
				r.Patch("/categories/{id}", s.handleUpdateCategory)
				r.Post("/merchants", s.handleCreateMerchant)
				r.Patch("/merchants/{id}", s.handleUpdateMerchant)
				// **الحظرُ والعفو** — merchant_violations.go
				r.Get("/merchants/{id}/violations", s.handleMerchantViolations)
				r.Post("/merchants/{id}/warnings", s.handleIssueMerchantWarning)
				// **والإنذارُ على حسابٍ** — سائقاً كان أو زبوناً أو مندوباً.
				r.Post("/users/{id}/warnings", s.handleIssueUserWarning)
				r.Post("/merchants/{id}/suspend", s.handleSuspendMerchant)
				r.Post("/merchants/{id}/clear-violations", s.handleClearViolations)
				r.Post("/merchants/{id}/menu/sections", s.handleCreateSection)
				r.Patch("/menu/sections/{sectionID}", s.handleUpdateSection)
				r.Delete("/menu/sections/{sectionID}", s.handleDeleteSection)
				r.Post("/merchants/{id}/menu/items", s.handleCreateItem)
				r.Patch("/menu/items/{itemID}", s.handleUpdateItem)
				r.Delete("/menu/items/{itemID}", s.handleDeleteItem)
				r.Put("/merchants/{id}/hours", s.handleSetHours)
				r.Post("/zones", s.handleCreateZone)
				r.Patch("/zones/{id}", s.handleUpdateZone)
				r.Delete("/zones/{id}", s.handleDeleteZone)
				r.Post("/promos", s.handleCreatePromo)
				r.Patch("/promos/{id}", s.handleUpdatePromo)
				r.Post("/banners", s.handleCreateBanner)
				r.Patch("/banners/{id}", s.handleUpdateBanner)
				r.Delete("/banners/{id}", s.handleDeleteBanner)
				r.Put("/settings/{key}", s.handleSetSetting)
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, httpx.ErrNotFound)
	})

	return r
}

// allowedOrigins **من أين يُقبل النداء.**
//
// ══════════════════════════════════════════════════════════════════════
// **والنطاقُ يُقرأ من البيئة لا يُكتب في الشيفرة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «نجهّز الموقع نرفعه على Render للتجربة، ثمّ
//
//	لاحقاً استضافةٌ حقيقيّةٌ ودومين».)
//
// **كان مكتوباً `https://*.rahalgo.com`** — ولا يُقلع عليه رفعٌ تجريبيّ:
// عنوانُ Render شيءٌ آخر، **فيُحجب كلُّ نداءٍ من الويب إلى المحرّك**
// والشاشةُ تُقرأ فارغةً بلا خطأٍ مفهوم.
//
// **وبيعُ النسخ يجعلها ألزم**: نسخةُ الشام نطاقُها هي، **ونطاقٌ مكتوبٌ في
// الشيفرة يعني بناءً لكلّ مشترٍ.**
//
// # وفراغُ المتغيّر في الإنتاج لا يُفتح للكلّ
//
// **`*` تعني أنّ أيَّ موقعٍ في الشبكة يستطيع مناداةَ محرّكك بجلسة زائره.**
// فإن لم يُضبط شيءٌ **يُغلق الباب** — ويُقال في السجلّ عند الإقلاع.
//
// **والتطويرُ يبقى مفتوحاً للمنافذ المحلّيّة** — لا شيءَ يُضبط على جهاز.
func (s *Server) allowedOrigins() []string {
	if s.cfg.Env == "development" {
		return []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	if len(s.cfg.WebOrigins) > 0 {
		return s.cfg.WebOrigins
	}
	// ══════════════════════════════════════════════════════════════════
	// **وقائمةٌ فارغةٌ تفتح البابَ للجميع — لا تغلقه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **قِيس ٢٠٢٦-٠٨-١٠**: أُقلع المحرّكُ في وضع الإنتاج بلا `WEB_ORIGINS`،
	// **فردّ `Access-Control-Allow-Origin: *` لكلّ نطاقٍ سُئل** — والمكتبةُ
	// تقرأ الفراغَ «بلا قيدٍ» لا «بلا سماح».
	//
	// **وهو أخطرُ ما يقع في نسيان متغيّر**: أيُّ موقعٍ في الشبكة ينادي
	// محرّكَك بجلسة زائره. **ونسيانُه وارد** — قيمتُه لا تُعرف قبل أوّل
	// نشرةٍ للويب.
	//
	// **فيُردّ نطاقٌ لا يطابق شيئاً** — بابٌ مغلقٌ بصوتٍ عالٍ في السجلّ.
	s.logger.Error("WEB_ORIGINS غير مضبوط في الإنتاج — كلُّ نداءٍ من المتصفّح سيُحجب")
	return []string{"https://origins.not.configured.invalid"}
}
