package identity

// **خيارُ إجبار تبديل الكلمة — مُطفَأٌ افتراضاً.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «نتركه خياراً… أحياناً يمرّ المستخدمُ بأكثرَ من
//
//	حركةٍ يعتبرها معقّدةً وربّما يغلق التطبيقَ بدون إكمال الخطوات بسبب
//	الملل… والحالةُ الافتراضيّةُ غيرُ مفعّل».)
//
// # وما يُختبر
//
// **أنّ المُطفأَ يُطفئ**، وأنّ المُشغَّلَ يُبقي العلَمَ، **وأنّ خدمةً بلا لوحةٍ
// لا تُجبر أحداً** — فالافتراضُ صفرٌ لا واحد.
//
// **ولا يُمسّ العلَمُ في القاعدة**: هو واقعةٌ — هذه الكلمةُ وضعها طرفٌ ثالث.
// **والتقنيعُ يخفي البوّابةَ ولا يمحو الخبر**، فمن شغّل الخيارَ بعد شهرٍ
// التقط كلَّ من وُضعت كلمتُه ولم يبدّلها.

import (
	"context"
	"testing"
)

func TestForcePasswordChangeIsOffByDefault(t *testing.T) {
	ctx := context.Background()

	reader := func(v int64) func(context.Context, string, int64) int64 {
		return func(_ context.Context, key string, fallback int64) int64 {
			if key == "security.force_password_change" {
				return v
			}
			return fallback
		}
	}

	cases := []struct {
		name  string
		bind  func(*Service)
		flag  bool
		wantF bool
	}{
		{"بلا لوحةٍ لا إجبار", func(*Service) {}, true, false},
		{"مُطفأً لا إجبار", func(s *Service) { s.SetSettingReader(reader(0)) }, true, false},
		{"مُشغَّلاً يُجبَر", func(s *Service) { s.SetSettingReader(reader(1)) }, true, true},
		{"ومن لا علَمَ عليه لا يُمسّ", func(s *Service) { s.SetSettingReader(reader(1)) }, false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &Service{}
			c.bind(svc)
			u := &User{MustChangePassword: c.flag}
			svc.applyForcePolicy(ctx, u)
			if u.MustChangePassword != c.wantF {
				t.Errorf("العلَمُ بعد السياسة = %v، والمنتظَر %v", u.MustChangePassword, c.wantF)
			}
		})
	}
}
