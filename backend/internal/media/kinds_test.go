package media

// **قائمةُ الأنواع مكتوبةٌ في موضعين** — هنا وفي قيد القاعدة. **والموضعان
// يفترقان بلا صوت.**
//
// # وقد افترقا فعلاً
//
// الهجرةُ ٠٠٦١ أضافت `delivery_proof` هنا وحدَه. **فقبِلت الشيفرةُ ما ترفضه
// القاعدة**: تُفحص الصورةُ وتُصغَّر وتُكتب على القرص، ثمّ يسقط الإدراجُ على
// القيد — خمسمئة، وملفٌّ يتيم. **ولم يُمسك في اختبارٍ ولا مراجعة**، لأنّ
// الاختباراتِ لا ترفع صوراً. ظهر في أوّل رفعةٍ حقيقية.
//
// **فهذا الاختبار يقرأ القيدَ من قاعدةٍ حيّة** ويقابله بالقائمة هنا. من زاد
// نوعاً في أحد الموضعين وحدَه سقط عنده — **لا عند أوّل مستخدم.**

import (
	"context"
	"regexp"
	"sort"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestKindsMatchDatabaseConstraint(t *testing.T) {
	pool := testdb.Pool(t)

	var def string
	err := pool.QueryRow(context.Background(), `
		SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'media'::regclass AND conname = 'media_kind_check'`).Scan(&def)
	if err != nil {
		t.Fatalf("قراءة قيد media_kind_check: %v", err)
	}

	// القيدُ نصٌّ مثل: CHECK ((kind = ANY (ARRAY['banner'::text, ...])))
	inDB := regexp.MustCompile(`'([a-z_]+)'::text`).FindAllStringSubmatch(def, -1)
	if len(inDB) == 0 {
		t.Fatalf("لم أفهم شكلَ القيد: %s", def)
	}
	var db []string
	for _, m := range inDB {
		db = append(db, m[1])
	}

	var code []string
	for k := range validKinds {
		code = append(code, k)
	}
	sort.Strings(db)
	sort.Strings(code)

	if len(db) != len(code) {
		t.Fatalf("افترق الموضعان\n  في Go:      %v\n  في القاعدة: %v\n"+
			"الإصلاح: هجرةٌ جديدة توسّع media_kind_check، أو حذفُ النوع من validKinds", code, db)
	}
	for i := range db {
		if db[i] != code[i] {
			t.Fatalf("افترق الموضعان\n  في Go:      %v\n  في القاعدة: %v", code, db)
		}
	}
}
