package server

// **نطاقُ الحديث اللحظيّ هو نطاقُ النداءات نفسُه — ولا يفترقان.**
//
// # ولماذا حارسٌ لهذا
//
// **كانت نطاقاتُ الـWebSocket مكتوبةً في الشيفرة** (`*.rahalgo.com`)
// ومنفصلةً عن `WEB_ORIGINS` التي يقرؤها CORS. **فلمّا رُفعت المنصّةُ على
// نطاقٍ آخر مرّت النداءاتُ العاديّةُ وسقط الحديثُ اللحظيُّ وحدَه.**
//
// **وسقوطُه صامت**: لا رسالةَ خطأٍ ولا شاشةَ تنكسر — **إنّما تتجمّد الأرقامُ
// والحالاتُ في كلّ شاشة.** وقاله المالك بجملةٍ واحدة (٢٠٢٦-٠٨-١١): «بكلّ
// مكانٍ فشل التحديثُ اللحظيّ» — **ولم يكن يعلم أنّ العلّةَ سطرٌ واحد.**
//
// **وقِيس على محرّكِ إنتاجٍ يعمل**: نطاقُ النسخة المرفوعة **يُردّ ٤٠٣**
// و`sub.rahalgo.com` يُقبل — أي عكسُ ما يجب تماماً.
//
// **ومصدران للشيء الواحد يفترقان يوماً** — وقد افترقا.

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/config"
)

// wsOriginPatterns **ما يُمرَّر فعلاً إلى مُصافحة الـWebSocket.**
func wsOriginPatterns(s *Server) []string { return s.allowedOrigins() }

// TestWSOrigins_HandshakeReadsTheSharedSource **والمُصافحةُ تقرأ المصدرَ
// المشترك — لا قائمةً مكتوبةً بجانبها.**
//
// ══════════════════════════════════════════════════════════════════════
// **وحارسٌ يفحص الدالّةَ وحدَها زخرفة**
// ══════════════════════════════════════════════════════════════════════
//
// **كتبتُ أوّلاً حارساً يستدعي `allowedOrigins` ويفحص ردَّها** — وهو صحيحٌ
// ولا يحرس شيئاً: **خُرّب `ws.go` فأُعيدت القائمةُ المكتوبة، ومرّ الحارسُ
// كما هو.** لأنّ العطبَ ليس في الدالّة، **إنّما في ألّا تُنادى.**
//
// **فيُقرأ الملفُّ نفسُه**: أيُمرَّر `allowedOrigins()` إلى `Accept`؟
// **وقائمةٌ مكتوبةٌ هناك تُمسَك مهما بدت سليمة.**
func TestWSOrigins_HandshakeReadsTheSharedSource(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ ws.go: %v", err)
	}
	body := string(src)
	if !strings.Contains(body, "OriginPatterns: s.allowedOrigins()") {
		t.Fatal("مُصافحةُ الـWebSocket لا تقرأ `allowedOrigins()` — " +
			"**فنطاقُ الحديث ينفصل عن نطاق النداءات، ويسقط التحديثُ اللحظيُّ صامتاً**")
	}
	// **ولا نطاقٌ مكتوبٌ بجانبها** — ولو كان صحيحاً اليوم.
	if strings.Contains(body, `OriginPatterns: []string{`) {
		t.Fatal("قائمةُ نطاقاتٍ مكتوبةٌ في `ws.go` — **مصدران للشيء الواحد يفترقان يوماً**")
	}
}

func TestWSOrigins_FollowConfiguredWebOrigins(t *testing.T) {
	want := []string{"https://rahalgo-web.onrender.com", "https://rahalgo.com"}
	srv := &Server{
		cfg:    &config.Config{Env: "production", WebOrigins: want},
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})),
	}
	got := wsOriginPatterns(srv)
	if len(got) != len(want) {
		t.Fatalf("نطاقاتُ الحديث لا تتبع الإعداد: %v — **فيسقط التحديثُ اللحظيُّ وحدَه بلا رسالة**", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("نطاقٌ تبدّل: %q بدل %q", got[i], want[i])
		}
	}
	// **ولا نطاقَ مكتوبٌ في الشيفرة** — وهو ما كان.
	for _, o := range got {
		if strings.Contains(o, "rahalgo.com") && !listHas(want, o) {
			t.Fatalf("نطاقٌ مكتوبٌ في الشيفرة تسرّب: %q", o)
		}
	}
}

func TestWSOrigins_DevelopmentStaysLocal(t *testing.T) {
	srv := &Server{
		cfg:    &config.Config{Env: "development"},
		logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1})),
	}
	got := wsOriginPatterns(srv)
	if len(got) == 0 {
		t.Fatal("التطويرُ بلا نطاقات — **فلا يعمل حديثٌ لحظيٌّ على جهاز أحد**")
	}
	for _, o := range got {
		if !strings.HasPrefix(o, "http://localhost") && !strings.HasPrefix(o, "http://127.0.0.1") {
			t.Fatalf("نطاقٌ غيرُ محلّيٍّ في التطوير: %q", o)
		}
	}
}

func listHas(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
