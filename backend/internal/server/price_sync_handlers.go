package server

// ══════════════════════════════════════════════════════════════════════
// **مزامنةُ الأسعار من الأمين — الأبواب** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
//	GET  /merchant/stores/{id}/price-sync          الاختيارُ المحفوظ وآخرُ مزامنة
//	POST /merchant/stores/{id}/price-sync/file     ملفٌّ: بلا اختيارٍ ⇒ الرأسُ وعيّنة،
//	                                               وباختيارٍ ⇒ «جرّب» أو «طبّق»
//	POST /merchant/stores/{id}/price-sync/token    رمزُ الساعي — يُعطى مرّةً
//	POST /public/price-sync                        الساعي برمزه — بلا جلسة
//
// **والقلبُ واحدٌ للبابين** — `catalog.SyncPrices`.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// maxSyncFile **أكبرُ ملفّ** — قائمةُ مولٍ بعشرات الآلاف أقلُّ من هذا بكثير.
const maxSyncFile = 10 << 20

func syncTokenHash(tok string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(tok)))
	return hex.EncodeToString(h[:])
}

// handleMerchantPriceSyncInfo **الاختيارُ المحفوظ، وأيوجد رمزٌ للساعي، وآخرُ مزامنة.**
func (s *Server) handleMerchantPriceSyncInfo(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	var mapping []byte
	var hasToken bool
	if err := s.pg.QueryRow(r.Context(), `
		SELECT price_sync_mapping, price_sync_token_hash IS NOT NULL
		  FROM merchants WHERE id = $1`, merchantID).Scan(&mapping, &hasToken); err != nil {
		s.respondErr(w, err)
		return
	}
	var last map[string]any
	var lastReport []byte
	var lastAt time.Time
	var source string
	err := s.pg.QueryRow(r.Context(), `
		SELECT source, report, created_at FROM price_sync_runs
		 WHERE merchant_id = $1 ORDER BY created_at DESC LIMIT 1`, merchantID).Scan(&source, &lastReport, &lastAt)
	if err == nil {
		last = map[string]any{"source": source, "report": json.RawMessage(lastReport), "at": lastAt}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, err)
		return
	}
	var m any
	if len(mapping) > 0 {
		_ = json.Unmarshal(mapping, &m)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"mapping": m, "has_token": hasToken, "last_run": last})
}

// handleMerchantPriceSyncFile **ملفُّ الأمين.**
//
// بلا `mapping` ⇒ **الرأسُ وعشرةُ صفوف** ليختار الأعمدة. وبه ⇒ تقريرُ «جرّب»
// (`dry_run=1`) أو التطبيق — **ويُحفظ الاختيارُ عند التطبيق** فلا يُسأل ثانيةً.
func (s *Server) handleMerchantPriceSyncFile(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSyncFile+64<<10)
	if err := r.ParseMultipartForm(maxSyncFile); err != nil {
		s.respondErr(w, catalog.ErrSyncFileType)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	sheet, err := catalog.ReadSheet(hdr.Filename, data)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	raw := strings.TrimSpace(r.FormValue("mapping"))
	if raw == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"headers": sheet.Headers, "sample": sheet.Sample(10), "rows": len(sheet.Rows),
		})
		return
	}
	var m catalog.SyncMapping
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	rows, err := sheet.SyncRows(m)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	dry := r.FormValue("dry_run") == "1"
	rep, err := s.catalog.SyncPrices(r.Context(), merchantID, rows, catalog.SyncOptions{
		Currency: m.Currency, DryRun: dry, Source: "file", ActorID: userIDFrom(r),
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !dry {
		if _, err := s.pg.Exec(r.Context(),
			`UPDATE merchants SET price_sync_mapping = $2 WHERE id = $1`, merchantID, raw); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("menu", "merchant:"+merchantID)
	}
	httpx.JSON(w, http.StatusOK, rep)
}

// handleMerchantPriceSyncToken **رمزُ الساعي — يُعطى مرّةً ويُحفظ مُجزَّأً.**
//
// **ورمزٌ جديدٌ يُبطل القديم** — من فقد رمزه أو شكّ فيه يطلب غيره.
func (s *Server) handleMerchantPriceSyncToken(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		s.respondErr(w, err)
		return
	}
	tok := "rgs_" + hex.EncodeToString(b)
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE merchants SET price_sync_token_hash = $2 WHERE id = $1`, merchantID, syncTokenHash(tok)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "merchant.price_sync_token", "merchant", merchantID, map[string]any{})
	httpx.JSON(w, http.StatusOK, map[string]any{"token": tok})
}

// handlePriceSyncAgent **الساعي يبعث القائمة برمزه** — بلا جلسة.
//
// **والرمزُ يدلّ على المتجر** — فلا يُرسَل معرّفُه ولا يُصدَّق إن أُرسل.
func (s *Server) handlePriceSyncAgent(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimSpace(r.Header.Get("X-Sync-Token"))
	if tok == "" {
		s.respondErr(w, errUnauthorized)
		return
	}
	var merchantID string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT id::text FROM merchants
		 WHERE price_sync_token_hash = $1 AND status <> 'suspended'`, syncTokenHash(tok)).Scan(&merchantID); err != nil {
		s.respondErr(w, errUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSyncFile)
	req, err := decode[struct {
		Rows     []catalog.SyncRow `json:"rows"`
		Currency string            `json:"currency"`
		DryRun   bool              `json:"dry_run"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	rep, err := s.catalog.SyncPrices(ctx, merchantID, req.Rows, catalog.SyncOptions{
		Currency: req.Currency, DryRun: req.DryRun, Source: "agent",
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !req.DryRun {
		s.touch("menu", "merchant:"+merchantID)
	}
	httpx.JSON(w, http.StatusOK, rep)
}
