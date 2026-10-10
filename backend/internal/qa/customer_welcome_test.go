package qa

// **ترحيبُ الزبون الجديد** (قرارُ المالك ٢٠٢٦-١٠-٠٥): من سجّل بنفسه يجد في
// صندوق إشعاراته رسالةَ «أهلاً فيك بعائلة رحّال غو» — **والنصُّ المعدَّلُ من الإعدادات هو ما يصل.**

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func cwSignup(t *testing.T, h *Harness) string {
	t.Helper()
	suPolicy(h, true)
	phone := fmt.Sprintf("+9639%08d", time.Now().UnixNano()%100000000)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM users WHERE phone = $1`, phone)
	})
	code := suPlant(t, h, phone)
	res := suConfirm(h, suIP(t, h), phone, code)
	if res.Code != 200 {
		t.Fatalf("التسجيل: %d %s", res.Code, trimBody(res))
	}
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&id); err != nil {
		t.Fatalf("الحساب: %v", err)
	}
	return id
}

func cwInbox(t *testing.T, h *Harness, userID string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `SELECT title FROM notifications WHERE user_id = $1`, userID)
	if err != nil {
		t.Fatalf("الإشعارات: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func TestCW01_NewCustomerGetsWelcomeInApp(t *testing.T) {
	h := New(t)
	id := cwSignup(t, h)
	found := false
	for _, title := range cwInbox(t, h, id) {
		if strings.Contains(title, "عائلة رحّال غو") {
			found = true
		}
	}
	if !found {
		t.Fatalf("لا ترحيبَ في صندوق الزبون الجديد: %v", cwInbox(t, h, id))
	}
}

func TestCW02_EditedTemplateIsWhatArrives(t *testing.T) {
	h := New(t)
	h.Setting("customers.welcome_template", `"مرحبا بك معنا\nنص مجرّب"`)
	id := cwSignup(t, h)
	for _, title := range cwInbox(t, h, id) {
		if title == "مرحبا بك معنا" {
			return
		}
	}
	t.Fatalf("النصُّ المعدَّل لم يصل: %v", cwInbox(t, h, id))
}

// TestCW03_TutorialLinkRidesTheWelcome — **رابطُ فيديو الشرح يصل مع الترحيب**
// (٢٠٢٦-١٠-١٠): متى وُضع `customers.tutorial_url` صار سطرُه في رسالة الزبون الجديد.
func TestCW03_TutorialLinkRidesTheWelcome(t *testing.T) {
	h := New(t)
	h.Setting("customers.tutorial_url", `"https://youtu.be/cw03test"`)
	id := cwSignup(t, h)
	var body string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT coalesce(string_agg(title || ' ' || body, ' | '), '') FROM notifications WHERE user_id = $1`, id).Scan(&body)
	if !strings.Contains(body, "https://youtu.be/cw03test") {
		t.Fatalf("رابطُ الشرح لم يصل مع الترحيب: %q", body)
	}
}
