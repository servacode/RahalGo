package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **سجلُّ «انتهت جلستك»** (قرارُ المالك ٢٠٢٦-١٠-٢١، هجرة ٠٤٦٨)
// ══════════════════════════════════════════════════════════════════════
//
// **يُكتب حين يُردّ الحسابُ بما يُخرجه** — تجديدٌ فشل أو جلسةٌ انتهت — لا لكلّ رمزِ
// وصولٍ منتهٍ (ذاك يُجدَّد عاديّاً كلَّ ربع ساعة). **والكتابةُ لا تؤخّر الردّ**: بسياقٍ
// منفصلٍ ومهلةٍ قصيرة، **وتعثّرُها لا يُسقط شيئاً.**

const authFailureKeepDays = 30

var authFailurePrune struct {
	sync.Mutex
	last time.Time
}

// authErrCode **رمزُ الخطأ كما وصل الجهاز.**
func authErrCode(err error) string {
	var ae *httpx.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return "internal"
}

// recordAuthFailure **صفٌّ واحدٌ لكلّ ردٍّ يُخرج الحساب.**
func (s *Server) recordAuthFailure(r *http.Request, kind, code, userID, sessionID, detail string) {
	var uid, sid *string
	if userID != "" {
		uid = &userID
	}
	if sessionID != "" {
		sid = &sessionID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := s.pg.Exec(ctx, `
		INSERT INTO auth_failures (kind, code, user_id, session_id, path, client, app_version, ip, detail)
		VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6, $7, $8, $9)`,
		kind, code, uid, sid, r.URL.Path, r.Header.Get(clientKindHeader), r.Header.Get(versionHeader),
		clientIP(r), detail); err != nil {
		s.logger.Warn("سجلُّ الخروج: تعذّرت الكتابة", "error", err)
		return
	}
	authFailurePrune.Lock()
	due := time.Since(authFailurePrune.last) > time.Hour
	if due {
		authFailurePrune.last = time.Now()
	}
	authFailurePrune.Unlock()
	if due {
		_, _ = s.pg.Exec(ctx, `DELETE FROM auth_failures WHERE at < now() - make_interval(days => $1)`,
			authFailureKeepDays)
	}
}
