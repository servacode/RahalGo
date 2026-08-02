package orders_test

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestOpenNow_CrossesMidnight دوامٌ يعبر منتصفَ الليل يُقرأ مفتوحاً في نصفَيه.
//
// # الخللُ الذي يمسكه
//
// كان الشرطُ `t BETWEEN open AND close` — **وهو مجالٌ مقلوبٌ لا يصدق على شيء
// حين يكون `close < open`.** فمن يفتح السادسةَ مساءً ويغلق الثانيةَ فجراً
// **يظهر مغلقاً طوالَ دوامه كلِّه**، ولا يعرف لماذا.
//
// **وليست حالةً نادرة**: مطاعمُ الشاورما في الرقّة تعمل ليلاً، وهي أكثرُ ما
// يُطلب.
//
// # ولماذا يُحسب الوقتُ بتوقيت دمشق
//
// الشرطُ يقرأ `now() AT TIME ZONE 'Asia/Damascus'`، **فالاختبارُ يجب أن يبني
// ساعاتِ الدوام حول تلك الساعة لا حول ساعة الخادم** — وإلّا نجح في الرقّة
// وسقط في مُشغّلٍ يعمل بتوقيتٍ آخر.
func TestOpenNow_CrossesMidnight(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	var damascus time.Time
	if err := pool.QueryRow(ctx,
		`SELECT (now() AT TIME ZONE 'Asia/Damascus')`).Scan(&damascus); err != nil {
		t.Fatalf("تعذّرت قراءة الساعة: %v", err)
	}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ('متجرُ الليل', $1, 10, 'active') RETURNING id`, categoryID).
		Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	openNow := func() bool {
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT `+orders.OpenNowSQL+` FROM merchants m WHERE m.id = $1`, merchantID).
			Scan(&v); err != nil {
			t.Fatalf("تعذّر فحص الدوام: %v", err)
		}
		return v
	}

	// **دوامٌ يبدأ قبل ساعتين ويمتدّ إلى ما بعد منتصف الليل** — والساعةُ الآن
	// في وسطه. ويُكتب على يوم البداية: إن كنّا بعد منتصف الليل فبدايتُه أمس.
	start := damascus.Add(-2 * time.Hour)
	setHours := func(day int, open, close string, closed bool) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1, $2, $3, $4::time, $5::time)
			ON CONFLICT (merchant_id, day_of_week)
			DO UPDATE SET closed = excluded.closed, open_time = excluded.open_time,
			              close_time = excluded.close_time`,
			merchantID, day, closed, open, close); err != nil {
			t.Fatalf("تعذّر ضبط الدوام: %v", err)
		}
	}

	// **الحالةُ الحاسمة**: نضع دواماً يعبر منتصفَ الليل صراحةً — يفتح ١٨:٠٠
	// ويغلق ٠٢:٠٠ — على اليوم الذي تقع فيه الساعةُ الحالية داخلَه.
	//
	// إن كانت الساعةُ الآن بين ١٨:٠٠ و٢٣:٥٩ فالصفُّ لليوم؛ وإن كانت بين ٠٠:٠٠
	// و٠١:٥٩ فالصفُّ لأمس. **وفي كلتيهما يجب أن يُقرأ مفتوحاً.**
	hour := damascus.Hour()
	switch {
	case hour >= 18:
		setHours(int(damascus.Weekday()), "18:00", "02:00", false)
	case hour < 2:
		setHours(int(damascus.AddDate(0, 0, -1).Weekday()), "18:00", "02:00", false)
	default:
		// خارج هذا المدى نصنع مجالاً مقلوباً يحيط بالساعة الحالية: يبدأ قبل
		// ساعتين وينتهي بعد ساعتين **من اليوم التالي** — وهو مقلوبٌ بالتعريف.
		setHours(int(damascus.Weekday()),
			start.Format("15:04"), damascus.Add(-4*time.Hour).Format("15:04"), false)
	}
	if !openNow() {
		t.Errorf("دوامٌ يعبر منتصفَ الليل قُرئ مغلقاً والساعةُ في وسطه (%s)",
			damascus.Format("15:04"))
	}

	// **وخارجَه يُقرأ مغلقاً** — وإلّا لكان الشرطُ يقول «مفتوح» دائماً، وهو
	// خللٌ معاكسٌ لا أقلّ ضرراً: **طلبٌ يصل متجراً نائماً.**
	//
	// نافذةٌ انقضت: بدأت قبل ستّ ساعاتٍ وأُغلقت قبل أربع.
	setHours(int(damascus.Weekday()),
		damascus.Add(-6*time.Hour).Format("15:04"),
		damascus.Add(-4*time.Hour).Format("15:04"), false)
	if _, err := pool.Exec(ctx,
		`DELETE FROM merchant_hours WHERE merchant_id = $1 AND day_of_week <> $2`,
		merchantID, int(damascus.Weekday())); err != nil {
		t.Fatalf("تعذّر حذف صفوف الأمس: %v", err)
	}
	// **إلّا أن تكون النافذةُ نفسُها عابرةً لمنتصف الليل** — وذلك حين تقع
	// الساعةُ الحاليةُ في أوّل ست ساعاتٍ من اليوم، فتنقلب الحسبة.
	if damascus.Add(-6*time.Hour).Day() == damascus.Day() && openNow() {
		t.Errorf("نافذةٌ انقضت قُرئت مفتوحة (الساعة %s)", damascus.Format("15:04"))
	}

	// **ويومٌ معلَّمٌ مغلقاً مغلقٌ مهما كانت الساعة.**
	setHours(int(damascus.Weekday()), "00:00", "23:59", true)
	if openNow() {
		t.Error("يومُ عطلةٍ قُرئ مفتوحاً")
	}
}
