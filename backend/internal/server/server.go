// Package server يبني موجّه HTTP وطبقة الوسطاء.
// البنية الطبقية الملزمة: handlers → services → repositories (GROUND-RULES §3).
package server

import (
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
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/referrals"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type Server struct {
	cfg        *config.Config
	logger     *slog.Logger
	pg         *pgxpool.Pool
	rdb        *redis.Client
	tokens     *auth.TokenIssuer
	identity   *identity.Service
	catalog    *catalog.Service
	settings   *settings.Store
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
	notify     *notifications.Service
	otpStatus  func() map[string]any
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
	hub *realtime.Hub, otpStatus func() map[string]any) *Server {
	notify := notifications.New(pg, hub, logger)
	geoSvc := geo.New(cfg.GeocoderURL, rdb, logger)
	// محرك الطلبات يحتاج الإشعارات (عمولة المندوب) وقد بُني قبلها — نحقنها الآن.
	ordersSvc.SetNotifier(notify)
	// وقواعدَ العمل من اللوحة: اشتراطُ توثيق واتساب قبل الطلب وما يليه.
	ordersSvc.SetSettings(settingsStore)
	// **والقائمةُ تحسب سعرَ البيع من الهامش** — انظر `pricing`.
	catalogSvc.SetSettings(settingsStore)
	srv := &Server{cfg: cfg, logger: logger, pg: pg, rdb: rdb, tokens: tokens,
		identity: identitySvc, catalog: catalogSvc, settings: settingsStore,
		wallet: walletSvc, orders: ordersSvc, cashbox: cashboxSvc, support: supportSvc,
		media: mediaSvc, hub: hub, otpStatus: otpStatus, notify: notify, geo: geoSvc}
	// **والحوافزُ تعرف الخزينةَ من محرّك الطلبات** — مصدرٌ واحدٌ لمن هي،
	// **ولا تُقرأ مرّتين بطريقتين.**
	srv.incentives = incentives.New(pg, walletSvc, settingsStore, ordersSvc.TreasuryID)
	// **والخصمُ يُقرأ لحظةَ بناء الطلب** — لا من ذاكرةٍ محمّلة.
	srv.offers = offers.New(pg)
	ordersSvc.SetOffers(srv.offers)
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
	r.Use(exceptPaths(middleware.Timeout(30*time.Second), "/api/v1/ws"))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.allowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
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

			r.Group(func(r chi.Router) {
				r.Use(s.RequireAuth)
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
		// **هويّةُ المنصة** — خفيفةٌ ومفتوحة، تناديها الخمسةُ وشاشةُ الدخول.
		r.Get("/public/platform", s.handlePublicPlatform)
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
			r.Post("/orders", s.handleCustomerCreateOrder)
			r.Post("/orders/{id}/rating", s.handleRateOrder)
			// إلغاء الزبون — كان حقّاً في خارطة الحالات بلا باب يوصله
			r.Post("/orders/{id}/cancel", s.handleCustomerCancelOrder)
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
			r.Get("/orders", s.handleDriverOrders)
			// **سجلُّه** — ما نفّذه نجح أم فشل. **وما انتهى كان يختفي**، فلا
			// يجد طلباً يتذكّره ليُبلّغ عنه. (انظر `driver_history.go`)
			r.Get("/orders/history", s.handleDriverHistory)
			r.Get("/orders/report-reasons", s.handleDriverReportReasons)
			r.Post("/orders/{id}/report", s.handleDriverReport)
			// **ومن وقف عند بابه يقيّمه** — الزبونُ يرى الطعامَ ولا يرى
			// المطبخ. (`merchant_rating_handlers.go`)
			r.Post("/orders/{id}/rate-merchant", s.handleDriverRateMerchant)
			r.Post("/orders/{id}/accept", s.handleDriverAccept)
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

			r.Post("/media", s.handleUploadMedia)
			r.Get("/users", s.handleAdminListUsers)
			r.Get("/users/stats", s.handleAdminUserRoleCounts)
			r.Get("/users/export", s.handleAdminUsersExport)
			r.Get("/users/{id}", s.handleAdminGetUser)
			r.Get("/users/{id}/activity", s.handleAdminUserActivity)
			r.Get("/users/{id}/feedback", s.handleAdminUserFeedback)
			r.Get("/users/{id}/financials", s.handleAdminUserFinancials)
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
				Post("/users/{id}/wallet", s.handleAdminWalletApply)
			// **وعناوينُه في ملفّه** — من يتابع شكوى «لم يصلني» يحتاج أن يرى
			// أين يسكن قبل أن يسأل.
			r.Get("/users/{id}/addresses", s.handleAdminUserAddresses)

			// الطلبات — غرفة العمليات تدير ولا تُنشئ (قرار 18):
			// الإنشاء حصراً عبر واجهات الزبون (الموقع/التطبيق)
			r.Get("/orders", s.handleListOrders)
			r.Get("/orders/alerts", s.handleOrderAlerts)
			r.Get("/orders/{id}", s.handleGetOrder)
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
				Post("/users/{id}/incentive", s.handleIncentiveGrant)
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
				Post("/payouts/{id}/decide", s.handleDecidePayout)

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
			r.With(s.RequireRoles("admin", "finance")).
				Post("/drivers/{id}/settle", s.handleDriverSettle)
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
				r.Post("/merchants/{id}/warnings", s.handleIssueWarning)
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

func (s *Server) allowedOrigins() []string {
	if s.cfg.Env == "development" {
		return []string{"http://localhost:*", "http://127.0.0.1:*"}
	}
	// تُضبط نطاقات الإنتاج عند النشر (المرحلة 5).
	return []string{"https://*.rahalgo.com"}
}
