package server

// **عناوينُ الملفّ عناوينُ صاحبه — كلُّها ولا واحدَ من غيرها.**
//
// (سؤالُ المالك ٢٠٢٦-٠٨-١٥: «نتأكّد أنّها تجلب العناوينَ الخاصّةَ
//  بالمستخدم».)
//
// # ولماذا حارسٌ لسطرٍ واحد
//
// **المعرّفُ يأتي من المسار لا من الجلسة** — والمكتبُ يقرأ ملفَّ غيره
// بحكم دوره. **فشرطُ `user_id` هو كلُّ ما يفصل ملفّاً عن ملفّ**، ومن
// أسقطه يوماً **لم يسقط شيءٌ في الشاشة**: تظهر عناوينُ أكثر، وتُقرأ
// «هذا زبونٌ كثيرُ العناوين».
//
// **وتسريبُ عنوانِ بيتٍ أخطرُ من تسريب مبلغ** — المالُ يُردّ.
//
// **والقائمةُ كاملةٌ بلا سقف**: عشرةٌ حدُّها في المحرّك
// (`customers.max_addresses`) **فلا صفحةَ تلزمها**، لكن ما دون الحدّ
// يُقرأ كلُّه — **ومن حُجب عنوانُه الأخيرُ لا يعرف أنّه حُجب.**

import (
	"context"
	"net/http"
	"testing"
)

// TestAdminAddresses_OnlyThisUsersAndAllOfThem **كلُّها له، ولا شيءَ لغيره.**
func TestAdminAddresses_OnlyThisUsersAndAllOfThem(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner, other, _ := twoCustomers(t, f)

	// **ثلاثةٌ لصاحب الملفّ** — أحدُها الافتراضيّ، **وواحدٌ لجاره.**
	mk := func(user, label, text string, def bool) {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO user_addresses
			    (user_id, label, area_building, address_text, location, is_default)
			VALUES ($1, $2, $3, $3,
			        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography, $4)`,
			user, label, text, def); err != nil {
			t.Fatalf("تعذّر إنشاءُ العنوان %q: %v", label, err)
		}
	}
	mk(owner, "بيتي", "شارعُ الرشيد — بناءُ ٧", true)
	mk(owner, "المكتب", "دوّارُ النعيم", false)
	mk(owner, "بيتُ أهلي", "حيُّ المشلب", false)
	mk(other, "بيتُ الجار", "عنوانٌ لا يخصّ هذا الملفّ", true)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM user_addresses WHERE user_id = ANY($1)`, []string{owner, other})
	})

	w := asCustomer(f.srv.handleAdminUserAddresses, http.MethodGet, owner, owner)
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — **وملفٌّ لا تُقرأ عناوينُه يُقرأ «لا عنوانَ له»**", w.Code)
	}
	body := w.Body.String()

	// **الثلاثةُ كلُّها** — ونقصُ واحدٍ صامتٌ لا يُنذر.
	for _, want := range []string{"شارعُ الرشيد", "دوّارُ النعيم", "حيُّ المشلب"} {
		if !contains(body, want) {
			t.Fatalf("سقط العنوان %q — **والنقصُ لا يُرى: قائمةٌ تعمل وتَنقُص**", want)
		}
	}
	// **ولا عنوانَ لغيره** — وهو ما يحرسه شرطُ `user_id` وحدَه.
	if contains(body, "عنوانٌ لا يخصّ هذا الملفّ") {
		t.Fatal("ظهر عنوانُ مستخدمٍ آخرَ في هذا الملفّ — **وتسريبُ عنوانِ بيتٍ لا يُردّ**")
	}
	// **والافتراضيُّ يُقال** — الشاشةُ تسمه بشارة، **ومن لم يُعرف افتراضيُّه
	// لم يُعرف أين يصل طلبُه إن لم يختر.**
	if !contains(body, `"is_default":true`) {
		t.Fatal("لا افتراضيَّ في الردّ — **والشارةُ في الشاشة تُقرأ منه**")
	}
}
