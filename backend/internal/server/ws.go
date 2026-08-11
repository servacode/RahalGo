package server

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// exceptPaths يُعفي مساراتٍ بعينها من وسيطٍ يلفّ الباقي.
//
// **الاستثناءُ صريحٌ في مكانٍ واحد** — لا مهلةٌ ثانية داخل المعالج ولا فرعٌ
// مخفيّ فيه. ومن قرأ سطرَ الوسيط عرف من يُعفى منه قبل أن يبحث.
func exceptPaths(mw func(http.Handler) http.Handler, paths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slices.Contains(paths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}

// handleWS اتصال البث الحي. المتصفح لا يرسل ترويسات مع WebSocket،
// فالتوكن يصل عبر معامل الاستعلام ويُتحقق منه كأي طلب.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	claims, err := s.tokens.VerifyAccess(r.URL.Query().Get("token"))
	if err != nil {
		httpx.Error(w, errUnauthorized)
		return
	}
	// جلسة أُنهيت لا تبقى لها قناة بث مفتوحة
	if s.identity.SessionRevoked(r.Context(), claims.SID) {
		httpx.Error(w, errUnauthorized)
		return
	}

	topics := []string{}
	if slices.ContainsFunc(claims.Roles, func(role string) bool {
		return role == "admin" || role == "ops" || role == "finance"
	}) {
		topics = append(topics, realtime.TopicOps)
	}
	// صاحب متجر: يشترك بمواضيع متاجره (بوابة المتجر)
	if slices.Contains(claims.Roles, "merchant") {
		rows, err := s.pg.Query(r.Context(),
			`SELECT id FROM merchants WHERE owner_user_id = $1`, claims.Subject)
		if err == nil {
			for rows.Next() {
				var id string
				if rows.Scan(&id) == nil {
					topics = append(topics, "merchant:"+id)
				}
			}
			rows.Close()
		}
	}
	// الزبون: موضوعه الشخصي (تتبع طلباته حياً من الموقع/التطبيق)
	if slices.Contains(claims.Roles, "customer") {
		topics = append(topics, "customer:"+claims.Subject)
	}
	// السائق: موضوعه الشخصي (إسناد الطلبات والنقد) **وإشارةُ الطابور**.
	//
	// والطابورُ موضوعٌ مشترك بين كل السائقين — **لذلك لا حمولةَ فيه**: يقول
	// «تغيّر شيء» فيُعيد التطبيقُ الجلب، وتحكم نقطةُ الطابور ما يُرى.
	if slices.Contains(claims.Roles, "driver") {
		topics = append(topics, "driver:"+claims.Subject, realtime.TopicDriverQueue)
	}
	// المندوب: موضوعه الشخصي (طلبات الانضمام والعمولات)
	if slices.Contains(claims.Roles, "sales") {
		topics = append(topics, "sales:"+claims.Subject)
	}
	// موضوع الإشعارات الشخصي — لكل مستخدم مهما كان دوره، فلا أحد يبقى بلا بث.
	topics = append(topics, "user:"+claims.Subject)

	// ══════════════════════════════════════════════════════════════════
	// **ونطاقُ الحديث اللحظيّ هو نطاقُ النداءات نفسُه**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-١١: «بكلّ مكانٍ فشل التحديثُ اللحظيّ».)
	//
	// **كانت النطاقاتُ مكتوبةً هنا** (`*.rahalgo.com`) — **ومنفصلةً عن
	// `WEB_ORIGINS`** التي يقرؤها CORS. فلمّا رُفعت المنصّةُ على نطاقٍ آخر
	// **مرّت النداءاتُ العاديّةُ وسقط الحديثُ اللحظيُّ وحدَه.**
	//
	// **وسقوطُه صامت**: لا رسالةَ خطأٍ ولا شاشةَ تنكسر — **إنّما تتجمّد
	// الأرقامُ والحالات**، فيُقرأ عطباً في كلّ صفحةٍ ولا يُعرف مصدرُه.
	//
	// **ومصدران للشيء الواحد يفترقان يوماً** — وقد افترقا. **فصارا واحداً**:
	// من ضبط نطاقَه للنداءات ضبطه للحديث معاً، **ولا يُنسى الثاني.**
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: s.allowedOrigins(),
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ch, cancel := s.hub.Subscribe(topics)
	defer cancel()

	ctx := r.Context()
	// قارئ لاكتشاف الإغلاق من الطرف الآخر
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case <-ping.C:
			pingCtx, pcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			pcancel()
			if err != nil {
				return
			}
		case msg := <-ch:
			writeCtx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return
			}
		}
	}
}
