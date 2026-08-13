package push

import (
	"context"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **فحصُ الدفع — يقول لماذا لا يرنّ**
// ══════════════════════════════════════════════════════════════════════
//
// (وقع ٢٠٢٦-٠٨-١٤: جُرّب الإشعارُ على جهازٍ حقيقيّ فلم يرنّ — **والجهازُ
//  مسجَّل، والمفتاحُ مضبوط، والمشروعُ متطابق.** ولم يكن في المنصّة كلِّها
//  ما يقول أين وقفت الرسالة.)
//
// # ولماذا بابٌ لا سطرٌ في السجلّ
//
// **سجلُّ الاستضافة يُقرأ بفتح لوحتها والبحث فيه** — ومن لا يفعل ذلك
// كلَّ يومٍ لا يعلم أنّ إشعاراتِه ماتت منذ أسبوع. **وإشعارٌ لا يصل لا
// يشتكي منه أحد**: السائقُ يظنّ أنّه لا طلبات، **والمكتبُ يظنّ أنّه
// كسول.**
//
// **فيُسأل المحرّكُ فيجيب** — بجملةٍ واحدةٍ تقول أين وقف.

// Diagnosis **ما وجده الفحص.**
type Diagnosis struct {
	// Configured **أمُهيَّأٌ الناقلُ أصلاً** — مفتاحُ الخدمة مقروء.
	Configured bool `json:"configured"`
	// Devices **كم جهازاً مسجَّلاً لهذا الحساب** — وصفرٌ يعني لم يُفتح
	// التطبيقُ بعد الدخول.
	Devices int `json:"devices"`
	// Fresh **وكم منها حديثُ العهد** — الرمزُ يشيخ فيُهمَل.
	Fresh int `json:"fresh"`
	// Sent **كم رسالةً قُبلت** من غوغل.
	Sent int `json:"sent"`
	// Dead **وكم رمزاً رفضته نهائيّاً** — جهازٌ حُذف منه التطبيق.
	Dead int `json:"dead"`
	// Error **ما قالته غوغل حين رفضت** — نصُّها كما هو.
	//
	// **ولا يُترجَم ولا يُلطَّف**: «SENDER_ID_MISMATCH» كلمةٌ تُبحث
	// فتُوجد، **و«تعذّر الإرسال» لا تُوجد.**
	Error string `json:"error"`
}

// Diagnose **يرسل رسالةَ فحصٍ إلى أجهزة صاحبها ويقول ما وقع.**
//
// **ولا تُحفظ في صندوقه** — هي فحصٌ لا خبر.
func (s *Service) Diagnose(ctx context.Context, userID string) Diagnosis {
	out := Diagnosis{Configured: s.Enabled()}
	if s == nil || userID == "" {
		return out
	}

	// **كلُّ أجهزته** — لا المصفّاةَ بالتطبيق: الفحصُ يُظهر ما هو قائمٌ
	// كلَّه، **ومن رأى صفراً عرف أنّ العطبَ قبل الإرسال.**
	rows, err := s.db.Query(ctx, `
		SELECT platform, token, last_seen_at > now() - $2::interval
		FROM device_tokens WHERE user_id = $1`, userID, staleAfter.String())
	if err != nil {
		out.Error = err.Error()
		return out
	}
	byPlatform := map[string][]string{}
	for rows.Next() {
		var platform, token string
		var fresh bool
		if rows.Scan(&platform, &token, &fresh) != nil {
			continue
		}
		out.Devices++
		if !fresh {
			continue
		}
		out.Fresh++
		byPlatform[platform] = append(byPlatform[platform], token)
	}
	rows.Close()

	if !out.Configured || out.Fresh == 0 {
		return out
	}

	msg := Message{
		Title:  "فحصُ الإشعارات",
		Body:   "وصلت — الإشعاراتُ تعمل على هذا الجهاز",
		Urgent: true,
		Data:   map[string]string{"kind": "diagnose"},
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for platform, tokens := range byPlatform {
		t := s.transports[platform]
		if t == nil {
			out.Error = "لا ناقلَ لمنصّة " + platform
			continue
		}
		dead, err := t.Send(ctx, tokens, msg)
		out.Dead += len(dead)
		if err != nil {
			// **وأوّلُ خطأٍ يكفي** — الأسبابُ تتكرّر.
			if out.Error == "" {
				out.Error = err.Error()
			}
			continue
		}
		out.Sent += len(tokens) - len(dead)
	}
	return out
}
