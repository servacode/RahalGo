package main

// ثلاثة سائقين — لا واحد.
//
// **والعدد ليس اعتباطاً**: الطابور في هذه المنصة **مشترك**، يظهر الطلب فيه لكل
// من بدأ دوامه، ومن ضغط أوّلاً أخذه. وسائقٌ واحد لا يُظهر شيئاً من ذلك:
//
//   - لا تنافُس على الطابور، ولا ردّ «سبقك سائقٌ آخر».
//   - ولا سقفَ الطلبات المتزامنة (٢) يُختبر: من بيده اثنان يُحجب عنه الثالث،
//     وإن كان وحده فالطلب الثالث يبقى معلّقاً بلا آخذ — وهو ما يكشف الحاجة
//     إلى الإسناد اليدوي من غرفة العمليات.
//   - ولا أثرَ لإنهاء الدوام: من انصرف يبقى طلبُه له، ومن بقي يلتقط ما تركه.
//
//	go run ./cmd/seed -drivers
//
// **ويُزرعون خارج الدوام.** التوفّر علَمٌ يرفعه صاحبه لا يُفترض عنه — وهذا
// قرارٌ في بنية المنصة (الترحيل 0040) لا تفصيلٌ في الزراعة. وسائقٌ يُولد
// «على الدوام» يكذب على غرفة العمليات في أوّل لحظة. ورفعُه ضغطةٌ واحدة من
// تطبيقه على المنفذ 3005.

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

var drivers = []struct{ Phone, Name, Password string }{
	{"+963941112233", "عمر الشيخ", "Saeq@2026"},
	{"+963942223344", "بشار الحسن", "Saeq@2026"},
	{"+963943334455", "مصطفى العلي", "Saeq@2026"},
}

func seedDrivers(ctx context.Context, tx pgx.Tx) {
	for _, d := range drivers {
		hash, err := auth.HashPassword(d.Password)
		if err != nil {
			log.Fatal(err)
		}
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (phone, full_name, password_hash)
			VALUES ($1, $2, $3)
			ON CONFLICT (phone) DO UPDATE SET
				full_name = EXCLUDED.full_name, password_hash = EXCLUDED.password_hash
			RETURNING id`, d.Phone, d.Name, hash).Scan(&id); err != nil {
			log.Fatalf("driver %s: %v", d.Phone, err)
		}
		for _, role := range fieldRoles("driver") {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, id, role); err != nil {
				log.Fatal(err)
			}
		}
		// **ولا يُلمس `on_shift`**: تشغيلُ الزراعة مرّةً ثانية بعد أن بدأ سائقٌ
		// دوامه يجب ألّا يُنهيه. الزراعة تُنشئ الحسابات لا تُدير أيامها.
	}

	fmt.Printf("✅ زُرع %d سائقين — كلُّهم **خارج الدوام**:\n", len(drivers))
	for _, d := range drivers {
		fmt.Printf("   %-15s %-16s %s\n", d.Phone, d.Name, d.Password)
	}
	fmt.Println()
	fmt.Println("   التوفّر علَمٌ يرفعه صاحبه: يبدأ كلٌّ دوامه بضغطةٍ من تطبيقه (:3005)،")
	fmt.Println("   وحتى يفعل لا يرى الطابور ولا يُسنَد إليه طلب.")
}
