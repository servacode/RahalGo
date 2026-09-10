package server

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// exceptPaths يُعفي مساراتٍ بعينها من وسيطٍ يلفّ الباقي.
//
// **الاستثناءُ صريحٌ في مكانٍ واحد** — لا مهلةٌ ثانية داخل المعالج ولا فرعٌ
// مخفيّ فيه. ومن قرأ سطرَ الوسيط عرف من يُعفى منه قبل أن يبحث.
//
// **وما بدأ بنجمةٍ يُطابَق بذيله** (`*/media`) — **ونقاطُ الرفع فيها
// معرّفٌ في وسط المسار** (`/orders/{id}/proof`)، فلا يُطابقها اسمٌ كامل.
func exceptPaths(mw func(http.Handler) http.Handler, paths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pathExcepted(paths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}

func pathExcepted(pats []string, path string) bool {
	for _, p := range pats {
		if suffix, ok := strings.CutPrefix(p, "*"); ok {
			if strings.HasSuffix(path, suffix) {
				return true
			}
			continue
		}
		if p == path {
			return true
		}
	}
	return false
}

// handleWS اتصال البث الحي. المتصفح لا يرسل ترويسات مع WebSocket،
// فالتوكن يصل عبر معامل الاستعلام ويُتحقق منه كأي طلب.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	claims, err := s.tokens.VerifyAccess(r.URL.Query().Get("token"))
	if err != nil {
		httpx.Error(w, errUnauthorized)
		return
	}
	// **ومصافحةٌ جديدةٌ توثَّق كأيّ طلب** — `R16`.
	//
	// **ولا تُخفَّف لأنّها بثّ**: **من رُفض في `HTTP` لا يُقبَل في
	// قناةٍ دائمة** — **وهي أطولُ عمراً من طلب.**
	//
	// **والوصلةُ القائمةُ عقدٌ آخر** (`R14`) — لا تمسّها هذه.
	state, dbRoles, dbCaps, err := s.identity.CheckSession(r.Context(), claims.SID)
	switch {
	case err != nil:
		s.logger.Error("البثّ: تعذّر التحقّقُ من الجلسة",
			"outcome", "auth_validation_unavailable", "error", err)
		httpx.Error(w, errAuthUnavailable)
		return
	case state == identity.SessionRevoked:
		httpx.Error(w, errUnauthorized)
		return
	}

	// ══════════════════════════════════════════════════════════════
	// **وحالُ الحساب يُسأل هنا كما يُسأل في `REST`** — `D14`
	// ══════════════════════════════════════════════════════════════
	//
	// **وكانت المصافحةُ تتحقّق من الجلسة ولا تسأل عن الحساب**:
	// **فالموقوفُ يُردّ بـ٤٠٣ في كلّ نداء، ويُقبَل في القناة الدائمة**
	// — **وهي أطولُ عمراً من نداء.** **قيس: بثٌّ عامٌّ وصل معلَّقاً،
	// وطابورُ العمل الجديد وصل سائقاً معلَّقاً.**
	//
	// **والجلسةُ ليست الامتياز**: **جلسةُ المعلَّق تبقى حيّةً وتُجدَّد**
	// (`XG-39`) — **ولم تُمَسّ.** **المتبدّلُ ما تمنحه من وصولٍ لحظيّ.**
	//
	// **والحكمُ من مصدر `REST` نفسِه** (`identity.ActiveStatus`) —
	// **ولا معجمَ حالاتٍ ثانٍ يُكتب هنا فينحرف.**
	var workRoom string
	status := s.identity.ActiveStatus(r.Context(), claims.Subject)
	if status != "active" && status != "suspended" {
		// **والمحظورُ والمحذوفُ يقفان عند كلّ شيء** — **ولا استثناءَ
		// فيهما** (`middleware.go`).
		httpx.Error(w, errForbidden)
		return
	}

	// **ومواضيعُ البثّ من أدوار اللحظة** — `R15`.
	//
	// **ومصافحةٌ جديدةٌ لا تُشترى بدورٍ سُحب** — **والوصلةُ القائمةُ
	// عقدٌ آخر** (`R14`) لا تمسّه هذه.
	roles := claims.Roles
	if claims.SID != "" {
		roles = dbRoles
	}

	// ══════════════════════════════════════════════════════════════
	// **وغرفةُ العمليّات بقدرةٍ لا باسم دور** — `ADG-1`
	// ══════════════════════════════════════════════════════════════
	//
	// **والقرارُ المركزيُّ نفسُه** — **ولا منطقُ أدوارٍ ثانٍ للبثّ.**
	hasOps := false
	for _, c := range dbCaps {
		if c == string(authz.OrdersRead) {
			hasOps = true
			break
		}
	}

	topics := []string{}
	if hasOps {
		topics = append(topics, realtime.TopicOps)
	}
	// صاحب متجر: يشترك بمواضيع متاجره (بوابة المتجر)
	if slices.Contains(roles, "merchant") {
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
	if slices.Contains(roles, "customer") {
		topics = append(topics, "customer:"+claims.Subject)
	}
	// السائق: موضوعه الشخصي (إسناد الطلبات والنقد) **وإشارةُ الطابور**.
	//
	// والطابورُ موضوعٌ مشترك بين كل السائقين — **لذلك لا حمولةَ فيه**: يقول
	// «تغيّر شيء» فيُعيد التطبيقُ الجلب، وتحكم نقطةُ الطابور ما يُرى.
	if slices.Contains(roles, "driver") {
		topics = append(topics, "driver:"+claims.Subject, realtime.TopicDriverQueue)
	}
	// المندوب: موضوعه الشخصي (طلبات الانضمام والعمولات)
	if slices.Contains(roles, "sales") {
		topics = append(topics, "sales:"+claims.Subject)
	}
	// موضوع الإشعارات الشخصي — لكل مستخدم مهما كان دوره، فلا أحد يبقى بلا بث.
	topics = append(topics, "user:"+claims.Subject)

	// **وإشارةُ اللوحة لكلّ متّصلٍ كذلك** — (شكوى المالك ٢٠٢٦-٠٨-١٨:
	// «أيُّ تعديلٍ من لوحة الأدمن فوراً يُطبَّق حتّى ولو الزبونُ فاتحٌ
	// التطبيق»).
	//
	// **وبلا شرطِ دورٍ**: ما تكتبه اللوحةُ يمسّ الزبونَ والسائقَ والمندوبَ
	// وصاحبَ المتجر — **وقائمةُ أدوارٍ هنا تعني دوراً يُنسى غدا.**
	topics = append(topics, realtime.TopicCatalog)

	// ══════════════════════════════════════════════════════════════
	// **والمعلَّقُ في غرفته وحدَها** — `D14`
	// ══════════════════════════════════════════════════════════════
	//
	// **وله أن يُتمّ ما بيده** (`suspension.go`): سائقٌ ينقل طلبَه
	// ويثبت تسليمَه · ومتجرٌ يقبل ما بين يديه · وزبونٌ يرى طلبَه
	// ويلغيه. **فيصله ما يخصّه هو.**
	//
	// **ولا يصله عملٌ جديد**: `drivers:queue` عروضٌ لا يملك قبولَها،
	// **وغرفةُ المكتب ليست له**، **وإشارةُ اللوحة عملٌ عاديّ.**
	//
	// **والحذفُ بالسماح لا بالمنع**: **غرفةٌ تُضاف غداً لا تصل
	// المعلَّقَ من نفسها.**
	if status == "suspended" {
		own := map[string]bool{"user:" + claims.Subject: true}
		if slices.Contains(roles, "customer") {
			own["customer:"+claims.Subject] = true
		}
		if slices.Contains(roles, "driver") {
			own["driver:"+claims.Subject] = true
		}
		for _, t := range topics {
			if strings.HasPrefix(t, "merchant:") {
				own[t] = true
			}
		}
		kept := topics[:0]
		for _, t := range topics {
			if own[t] {
				kept = append(kept, t)
			}
		}
		topics = kept

		// **وغرفةُ العمل تُفرَد بقناةٍ لتُرشَّح حمولتُها** — `D14`.
		//
		// **والغرفةُ وحدَها لا تكفي**: **العرضُ يُبَثّ فيها كما يُبَثّ
		// في الطابور** (`rotation.go`) — **فيُمنَع الطابورُ ويُنادى
		// من بابه الخاصّ.**
		//
		// **والمِرقاةُ لا تحمل اسمَ غرفتها** — **فتُشترَك على حدة
		// بدل تبديل بنية البثّ لأجل حالٍ واحدة.**
		workRoom = workRoomOf(roles, claims.Subject)
		if workRoom != "" {
			rest := topics[:0]
			for _, t := range topics {
				if t != workRoom {
					rest = append(rest, t)
				}
			}
			topics = rest
		}
		s.logger.Info("البثّ: حسابٌ معلَّقٌ — غرفتُه وحدَها",
			"user", claims.Subject, "rooms", len(topics), "work_room", workRoom)
	}

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

	// **وقناةُ غرفة العمل للمعلَّق وحدَه** — **وللنشط لا وجودَ لها.**
	var workCh <-chan []byte
	if workRoom != "" {
		wc, wcancel := s.hub.Subscribe([]string{workRoom})
		defer wcancel()
		workCh = wc
	}
	filter := suspendedFilter{UserID: claims.Subject}

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
		case msg := <-workCh:
			// **ولا يصل المعلَّقَ من غرفة عمله إلّا ما يحمله الآن.**
			if !filter.allow(msg) {
				continue
			}
			writeCtx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return
			}
		}
	}
}
