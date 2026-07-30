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
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type Server struct {
	cfg       *config.Config
	logger    *slog.Logger
	pg        *pgxpool.Pool
	rdb       *redis.Client
	tokens    *auth.TokenIssuer
	identity  *identity.Service
	catalog   *catalog.Service
	settings  *settings.Store
	wallet    *wallet.Service
	orders    *orders.Service
	cashbox   *cashbox.Service
	support   *support.Service
	media     *media.Service
	hub       *realtime.Hub
	otpStatus func() map[string]any
}

func New(cfg *config.Config, logger *slog.Logger, pg *pgxpool.Pool, rdb *redis.Client,
	tokens *auth.TokenIssuer, identitySvc *identity.Service, catalogSvc *catalog.Service,
	settingsStore *settings.Store, walletSvc *wallet.Service, ordersSvc *orders.Service,
	cashboxSvc *cashbox.Service, supportSvc *support.Service, mediaSvc *media.Service,
	hub *realtime.Hub, otpStatus func() map[string]any) *Server {
	return &Server{cfg: cfg, logger: logger, pg: pg, rdb: rdb, tokens: tokens,
		identity: identitySvc, catalog: catalogSvc, settings: settingsStore,
		wallet: walletSvc, orders: ordersSvc, cashbox: cashboxSvc, support: supportSvc,
		media: mediaSvc, hub: hub, otpStatus: otpStatus}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
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

			r.Group(func(r chi.Router) {
				r.Use(s.RequireAuth)
				r.Get("/me", s.handleMe)
				r.Post("/password", s.handleSetPassword)
			})
		})

		// تقييم الطلب — زبون الطلب نفسه (تُستخدم من التطبيق/الموقع)
		r.With(s.RequireAuth).Post("/orders/{id}/rating", s.handleRateOrder)

		// نقاط الإدارة — أدمن/عمليات فقط، والتعديلات الحساسة للأدمن حصراً
		r.Route("/admin", func(r chi.Router) {
			r.Use(s.RequireAuth)
			r.Use(s.RequireRoles("admin", "ops"))
			r.Get("/whatsapp", func(w http.ResponseWriter, _ *http.Request) {
				httpx.JSON(w, http.StatusOK, s.otpStatus())
			})

			r.Post("/media", s.handleUploadMedia)
			r.Get("/users", s.handleAdminListUsers)
			r.Get("/categories", s.handleListCategories)
			r.Get("/merchants", s.handleListMerchants)
			r.Get("/merchants/{id}/menu", s.handleGetMenu)
			r.Get("/merchants/{id}/hours", s.handleGetHours)
			r.Get("/zones", s.handleListZones)
			r.Get("/promos", s.handleListPromos)
			r.Get("/banners", s.handleListBanners)
			r.Get("/settings", s.handleListSettings)
			r.Get("/stats", s.handleAdminStats)
			r.Get("/reports", s.handleReports)
			r.Get("/users/{id}/wallet", s.handleAdminWalletStatement)
			r.With(s.RequireRoles("admin", "finance")).
				Post("/users/{id}/wallet", s.handleAdminWalletApply)

			// الطلبات — غرفة العمليات تدير ولا تُنشئ (قرار 18):
			// الإنشاء حصراً عبر واجهات الزبون (الموقع/التطبيق)
			r.Get("/orders", s.handleListOrders)
			r.Get("/orders/alerts", s.handleOrderAlerts)
			r.Get("/orders/{id}", s.handleGetOrder)
			r.Post("/orders/{id}/transition", s.handleOrderTransition)
			r.Post("/orders/{id}/assign", s.handleOrderAssign)

			// الأقسام التشغيلية لكل دور (قرار 16)
			r.Get("/customers", s.handleListCustomers)
			r.Get("/salesreps", s.handleListSalesReps)

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
			r.With(s.RequireRoles("admin", "finance")).
				Post("/drivers/{id}/settle", s.handleDriverSettle)
			r.Group(func(r chi.Router) {
				r.Use(s.RequireRoles("admin"))
				r.Post("/users", s.handleAdminCreateUser)
				r.Patch("/users/{id}", s.handleAdminUpdateUser)
				r.Post("/users/{id}/roles", s.handleAdminGrantRole)
				r.Delete("/users/{id}/roles/{role}", s.handleAdminRevokeRole)
				r.Post("/categories", s.handleCreateCategory)
				r.Patch("/categories/{id}", s.handleUpdateCategory)
				r.Post("/merchants", s.handleCreateMerchant)
				r.Patch("/merchants/{id}", s.handleUpdateMerchant)
				r.Post("/merchants/{id}/menu/sections", s.handleCreateSection)
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
