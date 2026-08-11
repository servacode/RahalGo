package identity

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حارسُ نوع العميل — والقيدُ يُضيَّق ولا يُلغى**
// ══════════════════════════════════════════════════════════════════════
//
// **ما يُحرَس هنا شيئان متعاكسان**، وسقوطُ أيّهما يكسر شيئاً حقيقيّاً:
//
// **الأوّل** أن يبقى المتصفّحُ والتطبيقُ منفصلين — **وإلّا عادت الحلقةُ
// المقفلة**: السائقُ يفتح التطبيقَ فيخرج من الويب.
//
// **والثاني** أن تبقى القائمةُ مغلقة — **وإلّا سقط القيدُ كلُّه وهو يبدو
// قائماً**: من يستطيع أن يخترع نوعاً بترويسةٍ من عنده يفتح مئةَ جلسةٍ
// بمئة كلمة، **فيتشارك سائقان حساباً واحداً ولا تعرف المنصّةُ من سلّم.**

// TestClientKind_UnknownFallsBackToWeb **والمجهولُ متصفّحٌ لا نوعٌ جديد.**
func TestClientKind_UnknownFallsBackToWeb(t *testing.T) {
	for _, raw := range []string{
		"", "  ", "WEB", "Android", "android-v2", "desktop", "ios ",
		// ══════════════════════════════════════════════════════════════
		// **ومنصّةٌ بلا تطبيقٍ ليست نوعاً**
		// ══════════════════════════════════════════════════════════════
		//
		// **`android` كانت نوعاً حتّى ٢٠٢٦-٠٨-١١** — وهو صحيحٌ لتطبيقٍ
		// واحد. **ومع أربعةِ تطبيقاتٍ يجمعها كلَّها في جلسةٍ واحدةٍ
		// يقتل بعضُها بعضاً**: المتجرُ يفتح تطبيقَ الزبون ليطلب عشاءه
		// فيخرج من تطبيق متجره.
		"android", "ios", "android-", "-customer", "-",
		// **وتطبيقٌ مجهولٌ على منصّةٍ معروفةٍ يُردّ** — وإلّا فتح كلُّ
		// نصٍّ يُخترَع جلسةً مستقلّة.
		"android-hacker", "windows-customer", "android-customer-x",
		"'; DROP TABLE refresh_tokens; --", "android\x00",
	} {
		if got := normalizeClient(raw); got != ClientWeb {
			t.Fatalf("نوعٌ مجهولٌ %q صار %q — **فيفتح جلسةً مستقلّةً بكلمةٍ يخترعها**، "+
				"ويسقط المنعُ من مشاركة الحسابات وهو يبدو قائماً", raw, got)
		}
	}
}

// TestClientKind_KnownKindsSurvive **والمعروفُ يمرّ كما هو** — وبلا ذلك
// يصير كلُّ شيءٍ متصفّحاً، **فيعود التطبيقُ يقتل جلسةَ الويب.**
func TestClientKind_KnownKindsSurvive(t *testing.T) {
	known := []string{ClientWeb}
	for platform := range clientPlatforms {
		for app := range clientApps {
			known = append(known, platform+"-"+app)
		}
	}
	// **وتسعةٌ لا أقلّ**: متصفّحٌ ومنصّتان × أربعةُ تطبيقات. **ونقصانُ
	// واحدٍ يعني تطبيقاً يُقرأ متصفّحاً فيُخرج صاحبَه من لوحته.**
	if len(known) != 9 {
		t.Fatalf("الأنواعُ المعروفةُ %d لا ٩", len(known))
	}
	for _, k := range known {
		if got := normalizeClient(k); got != k {
			t.Fatalf("نوعٌ معروفٌ %q صار %q — **فيتصادم التطبيقُ مع المتصفّح**", k, got)
		}
	}
}

// TestClientKind_ContextRoundTrip **والسياقُ يحمله سليماً** — وهو الطريقُ
// الوحيدُ بين الوسيط و`issueSession`.
func TestClientKind_ContextRoundTrip(t *testing.T) {
	if got := ClientFrom(context.Background()); got != ClientWeb {
		t.Fatalf("سياقٌ بلا نوعٍ أعطى %q — **والافتراضُ متصفّحٌ**: كلُّ نداءٍ من الويب "+
			"لا يحمل الترويسةَ ولن يحملها", got)
	}
	ctx := WithClient(context.Background(), "android-driver")
	if got := ClientFrom(ctx); got != "android-driver" {
		t.Fatalf("النوعُ ضاع في السياق: %q — **فيُسجَّل التطبيقُ متصفّحاً "+
			"ويُبطل جلسةَ صاحبه على الويب**", got)
	}
	// **والتطبيعُ يقع عند الوضع لا عند القراءة** — فلا يصل إلى القاعدة
	// نصٌّ لم يُفحص.
	if got := ClientFrom(WithClient(context.Background(), "hack")); got != ClientWeb {
		t.Fatalf("نصٌّ غيرُ مفحوصٍ عبر إلى السياق: %q", got)
	}
}
