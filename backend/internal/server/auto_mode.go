package server

// ══════════════════════════════════════════════════════════════════════
//  **زرُّ «تلقائي/يدوي» لمن يعمل في الشاشة** (ملاحظةُ المالك ٢٠٢٦-١٠-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الزرُّ إعداداً يقرؤه لوحُ الإعدادات** — فلا يراه إلّا من يملك
// `settings.read` وكتابةَ المفتاح، **وحسابُ العمليات لا يملكهما ولا يصحّ
// أن يملكهما**: لوحُ الإعدادات فيه الأسعارُ والعمولاتُ والأسرار.
//
// **فبابٌ ضيّقٌ لمفتاحين لا غير**، كلٌّ بقدرةِ شاشته:
//
//	orders.auto_transfer  ← من يتدخّل في الطلبات (`orders.intervene`)
//	leads.auto_approve    ← من يقرّر طلبات الانضمام (`merchants.verify`)
//
// **والقراءةُ بقدرةِ القراءة، والكتابةُ بقدرةِ الفعل** — ومن يقرأ الطلبات
// ولا يتدخّل يرى الحالَ ولا يبدّله. **ولا مفتاحَ ثالثٌ يمرّ من هنا.**
// **وكلُّ تبديلٍ يُكتب في سجلّ التدقيق** بالقيمتين، كما يكتبه لوحُ الإعدادات.

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type autoModeDoor struct {
	key        string
	minutesKey string
	write      authz.Capability
}

var autoModeDoors = map[string]autoModeDoor{
	"orders": {key: "orders.auto_transfer", minutesKey: "orders.unattended_auto_accept_min", write: authz.OrdersIntervene},
	"leads":  {key: "leads.auto_approve", write: authz.MerchantsVerify},
}

// GET /admin/auto-mode/orders · /admin/auto-mode/leads
//
// **مساران صريحان لا `{kind}`** — جدولُ السياسة يحرس المسارَ بقدرةٍ واحدة،
// **ولكلّ بابٍ قدرتُه.**
func (s *Server) handleGetAutoMode(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { s.getAutoMode(w, r, autoModeDoors[kind]) }
}

func (s *Server) handleSetAutoMode(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { s.setAutoMode(w, r, autoModeDoors[kind]) }
}

func (s *Server) getAutoMode(w http.ResponseWriter, r *http.Request, d autoModeDoor) {
	out := map[string]any{
		"on":       s.settings.GetBool(r.Context(), d.key),
		"editable": s.hasCapability(r, d.write),
	}
	if d.minutesKey != "" {
		out["minutes"] = s.settings.GetInt(r.Context(), d.minutesKey)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// PUT body: {"on": true|false}
func (s *Server) setAutoMode(w http.ResponseWriter, r *http.Request, d autoModeDoor) {
	// **والقدرةُ تُفحص هنا أيضاً** — حارسٌ ثانٍ خلف جدول السياسة.
	if !s.hasCapability(r, d.write) {
		s.respondErr(w, errForbiddenCap)
		return
	}
	req, err := decode[struct {
		On *bool `json:"on"`
	}](r)
	if err != nil || req.On == nil {
		s.respondErr(w, errValidation)
		return
	}
	before := s.settings.GetBool(r.Context(), d.key)
	if before == *req.On {
		httpx.JSON(w, http.StatusOK, map[string]any{"on": before, "updated": false})
		return
	}
	actor := userIDFrom(r)
	// **والتبديلُ وقيدُه في معاملةٍ واحدة** (`AQ-4`) — تبديلٌ يمضي وقيدُه
	// يسقط لا يُعرف من فعله.
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		if err := s.settings.SetTx(ctx, q, d.key, *req.On, &actor); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "admin.setting_update", "setting", d.key, map[string]any{
			"before": before, "after": *req.On, "via": "auto_mode",
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("settings")
	httpx.JSON(w, http.StatusOK, map[string]any{"on": *req.On, "updated": true})
}
