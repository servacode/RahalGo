package server

// ══════════════════════════════════════════════════════════════════════
// **ساعي الأمين — ملفٌّ واحدٌ يُضبط من هنا** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// «ملف واحد يشتغل بالخلفية تلقائي، ويشتغل لحاله بعد إعادة التشغيل… ما بدنا المحاسب
// يتلبك». فـ:
//
//	GET  /merchant/stores/{id}/price-sync/agent  البرنامجُ وفيه رمزُ المتجر — دبل كليك وخلص
//	POST /merchant/stores/{id}/price-sync/stop   «إيقاف البرنامج» — يزيل نفسَه من الإقلاع
//	POST /public/price-sync/poll                  يسأل كلَّ دقيقة: يبعث حالَه ويأخذ إعدادَه
//	POST /public/price-sync/schema                جداولُ الأمين — لنضبط القراءةَ بلا زيارة
//	GET  /public/price-sync/agent                 نسختُه الأحدث — يحدّث نفسَه
//	GET/PUT /admin/merchants/{id}/price-sync      إعدادُ القراءة — لفريق المنصّة
//
// **والبرنامجُ يُبنى مع الخادم** (`Dockerfile`) فكلُّ نشرٍ يحمل أحدثَه.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// agentMarker **علامةُ إعداد المتجر في ذيل البرنامج** — يقرؤها الساعي من ملفّه نفسِه.
const agentMarker = "\n##RAHALGO-SYNC-CONFIG##"

var errAgentMissing = httpx.NewError(http.StatusServiceUnavailable, "price_sync_agent_missing", "errors.price_sync_agent_missing")

// errSyncQuery **جملةُ قراءةٍ ليست SELECT خالصة** — والساعي يرفضها كذلك.
var errSyncQuery = httpx.NewError(http.StatusBadRequest, "price_sync_query", "errors.price_sync_query")

func agentDir() string {
	if d := strings.TrimSpace(os.Getenv("AGENT_DIR")); d != "" {
		return d
	}
	return "/app/agent"
}

// agentLatest **نسختُه الأحدث وملفُّه** — وفراغٌ إن لم يُبنَ مع هذا الخادم.
func agentLatest() (string, []byte) {
	ver, _ := os.ReadFile(filepath.Join(agentDir(), "agent-version"))
	exe, _ := os.ReadFile(filepath.Join(agentDir(), "RahalGoSync.exe"))
	return strings.TrimSpace(string(ver)), exe
}

var forbiddenSQL = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|create|truncate|merge|exec|execute|grant|revoke|deny|backup|restore|shutdown|dbcc|into|openrowset|opendatasource|xp_\w+|sp_\w+)\b`)

// validSyncQuery **قراءةٌ لا كتابة** — `SELECT` واحدةٌ بلا فاصلةٍ منقوطةٍ ولا كلمةٍ تكتب.
// **وكمبيوترُ المول يحمل حساباتِه** — فجملةٌ تمحو لا تمرّ ولو كتبها أحدٌ في اللوحة.
func validSyncQuery(q string) bool {
	t := strings.TrimSpace(q)
	if t == "" {
		return true
	}
	low := strings.ToLower(t)
	return (strings.HasPrefix(low, "select") || strings.HasPrefix(low, "with")) &&
		!strings.Contains(t, ";") && !strings.Contains(t, "--") && !strings.Contains(t, "/*") &&
		!forbiddenSQL.MatchString(t)
}

// handleMerchantPriceSyncAgentDownload **البرنامجُ برمز المتجر في ذيله.**
//
// **وكلُّ تحميلٍ رمزٌ جديد** — فالبرنامجُ القديمُ إن بقي على جهازٍ آخر يتوقّف، **ولا يُعرف
// رمزٌ قديم** (يُحفظ مجزّأً). ومعه يُرفع «الإيقاف» ويُطلب إرسالُ الجداول.
func (s *Server) handleMerchantPriceSyncAgentDownload(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	_, exe := agentLatest()
	if len(exe) == 0 {
		s.respondErr(w, errAgentMissing)
		return
	}
	tok, err := newSyncToken()
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET price_sync_token_hash = $2, price_sync_stop = false, price_sync_want_schema = true
		 WHERE id = $1`, merchantID, syncTokenHash(tok)); err != nil {
		s.respondErr(w, err)
		return
	}
	cfg, _ := json.Marshal(map[string]string{"server": "https://" + r.Host, "token": tok})
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "merchant.price_sync_token", "merchant", merchantID, map[string]any{"via": "download"})
	w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
	w.Header().Set("Content-Disposition", `attachment; filename="RahalGoSync.exe"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(exe)
	_, _ = w.Write([]byte(agentMarker))
	_, _ = w.Write(cfg)
}

// handleMerchantPriceSyncStop **«إيقاف البرنامج»** — يقرؤه الساعي في سؤاله التالي فيزيل نفسَه.
func (s *Server) handleMerchantPriceSyncStop(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE merchants SET price_sync_stop = true WHERE id = $1`, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// agentConfig **ما يعمل به الساعي** — يُعاد في كلّ سؤال.
type agentConfig struct {
	SyncNow       bool   `json:"sync_now"`
	Interval      int    `json:"interval_minutes"`
	HasMapping    bool   `json:"has_mapping"`
	Stop          bool   `json:"stop"`
	SQLQuery      string `json:"sql_query"`
	SQLConnection string `json:"sql_connection"`
	ProbeTable    string `json:"probe_table"`
	Folder        string `json:"folder"`
	Currency      string `json:"currency"`
	WantSchema    bool   `json:"want_schema"`
	LatestVersion string `json:"latest_version"`
}

// handlePriceSyncPollV2 **الساعي يبعث حالَه ويأخذ إعدادَه** — كلَّ دقيقة.
func (s *Server) handlePriceSyncPollV2(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.syncAgentMerchant(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	req, err := decode[struct {
		Version string         `json:"version"`
		Error   string         `json:"error"`
		Status  map[string]any `json:"status"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.Version) > 40 {
		req.Version = req.Version[:40]
	}
	if len(req.Error) > 1000 {
		req.Error = req.Error[:1000]
	}
	status, _ := json.Marshal(req.Status)
	var c agentConfig
	var query, conn, folder *string
	if err := s.pg.QueryRow(r.Context(), `
		UPDATE merchants SET price_sync_agent_seen_at = now(),
		       price_sync_agent_version = NULLIF($2, ''),
		       price_sync_agent_error   = NULLIF($3, ''),
		       price_sync_agent_status  = $4
		 WHERE id = $1
		RETURNING price_sync_requested_at IS NOT NULL, price_sync_interval_minutes,
		          price_sync_mapping IS NOT NULL, price_sync_stop, price_sync_sql_query,
		          price_sync_sql_connection, price_sync_probe_table, price_sync_folder,
		          price_sync_currency, price_sync_want_schema`,
		merchantID, req.Version, req.Error, status).Scan(&c.SyncNow, &c.Interval, &c.HasMapping, &c.Stop,
		&query, &conn, &c.ProbeTable, &folder, &c.Currency, &c.WantSchema); err != nil {
		s.respondErr(w, err)
		return
	}
	if query != nil {
		c.SQLQuery = *query
	}
	if conn != nil {
		c.SQLConnection = *conn
	}
	if folder != nil {
		c.Folder = *folder
	}
	c.LatestVersion, _ = agentLatest()
	httpx.JSON(w, http.StatusOK, c)
}

// handlePriceSyncSchema **جداولُ الأمين كما وجدها الساعي** — يُضبط بها سطرُ القراءة بلا زيارة.
func (s *Server) handlePriceSyncSchema(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.syncAgentMerchant(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	raw, err := decode[json.RawMessage](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET price_sync_schema = $2, price_sync_schema_at = now(), price_sync_want_schema = false
		 WHERE id = $1`, merchantID, []byte(*raw)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": true})
}

// handlePriceSyncAgentBinary **أحدثُ نسخةٍ للساعي** — بلا إعداد؛ إعدادُه محفوظٌ عنده.
func (s *Server) handlePriceSyncAgentBinary(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.syncAgentMerchant(w, r); !ok {
		return
	}
	_, exe := agentLatest()
	if len(exe) == 0 {
		s.respondErr(w, errAgentMissing)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
	_, _ = w.Write(exe)
}

// handleAdminPriceSyncGet **إعدادُ القراءة وما بعثه الساعي** — لفريق المنصّة.
func (s *Server) handleAdminPriceSyncGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var out struct {
		SQLQuery      *string          `json:"sql_query"`
		SQLConnection *string          `json:"sql_connection"`
		ProbeTable    string           `json:"probe_table"`
		Folder        *string          `json:"folder"`
		Currency      string           `json:"currency"`
		Stop          bool             `json:"stop"`
		WantSchema    bool             `json:"want_schema"`
		AgentSeenAt   *time.Time       `json:"agent_seen_at"`
		AgentVersion  *string          `json:"agent_version"`
		AgentError    *string          `json:"agent_error"`
		AgentStatus   *json.RawMessage `json:"agent_status"`
		Schema        *json.RawMessage `json:"schema"`
		SchemaAt      *time.Time       `json:"schema_at"`
	}
	var status, schema []byte
	if err := s.pg.QueryRow(r.Context(), `
		SELECT price_sync_sql_query, price_sync_sql_connection, price_sync_probe_table, price_sync_folder,
		       price_sync_currency, price_sync_stop, price_sync_want_schema, price_sync_agent_seen_at,
		       price_sync_agent_version, price_sync_agent_error, price_sync_agent_status,
		       price_sync_schema, price_sync_schema_at
		  FROM merchants WHERE id = $1`, id).Scan(&out.SQLQuery, &out.SQLConnection, &out.ProbeTable, &out.Folder,
		&out.Currency, &out.Stop, &out.WantSchema, &out.AgentSeenAt, &out.AgentVersion, &out.AgentError,
		&status, &schema, &out.SchemaAt); err != nil {
		s.respondErr(w, err)
		return
	}
	if len(status) > 0 {
		m := json.RawMessage(status)
		out.AgentStatus = &m
	}
	if len(schema) > 0 {
		m := json.RawMessage(schema)
		out.Schema = &m
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleAdminPriceSyncPut **يضبط القراءة** — وجملةٌ تكتب تُردّ.
func (s *Server) handleAdminPriceSyncPut(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	req, err := decode[struct {
		SQLQuery      *string `json:"sql_query"`
		SQLConnection *string `json:"sql_connection"`
		ProbeTable    *string `json:"probe_table"`
		Folder        *string `json:"folder"`
		Currency      *string `json:"currency"`
		WantSchema    *bool   `json:"want_schema"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.SQLQuery != nil && !validSyncQuery(*req.SQLQuery) {
		s.respondErr(w, errSyncQuery)
		return
	}
	if req.Currency != nil && *req.Currency != "SYP" && *req.Currency != "USD" {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET
			price_sync_sql_query      = COALESCE(NULLIF($2, ''), CASE WHEN $2 = '' THEN NULL ELSE price_sync_sql_query END),
			price_sync_sql_connection = COALESCE(NULLIF($3, ''), CASE WHEN $3 = '' THEN NULL ELSE price_sync_sql_connection END),
			price_sync_probe_table    = COALESCE(NULLIF($4, ''), price_sync_probe_table),
			price_sync_folder         = COALESCE(NULLIF($5, ''), CASE WHEN $5 = '' THEN NULL ELSE price_sync_folder END),
			price_sync_currency       = COALESCE($6, price_sync_currency),
			price_sync_want_schema    = COALESCE($7, price_sync_want_schema)
		 WHERE id = $1`, id, req.SQLQuery, req.SQLConnection, req.ProbeTable, req.Folder, req.Currency, req.WantSchema); err != nil {
		s.respondErr(w, err)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "admin.price_sync_config", "merchant", id, map[string]any{})
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
