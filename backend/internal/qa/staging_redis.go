package qa

// ══════════════════════════════════════════════════════════════════════
// **مِسنَدٌ على `Redis` حقيقيٍّ — `P-0` البند ٣٠**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا لم يكن هذا ممكناً في `P-7`
//
// **`miniredis` مكتبةٌ في الذاكرة** — **لا تُسقَط ولا تُقطَع.** و`R16`
// تسأل: **ماذا يفعل التوثيقُ حين تسقط `Redis`؟** **ولا جواب إلّا
// بإسقاطها فعلاً.**
//
// **فأُجِّلت إلى `P-0`** — وهذه بيئتُها.
//
// # ويتخطّى بهدوءٍ إن لم تكن البيئةُ مهيّأة
//
// **ولا يُسقط الحزمةَ على جهازٍ لا `Redis` تجهيزٍ فيه** — **وحزمةٌ
// حمراءُ لسببٍ بيئيٍّ تُقرأ عطباً في المنتج.**

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// StagingRedisAddr عنوانُ ذاكرة التجهيز — **من البيئة لا مكتوباً.**
func StagingRedisAddr() string {
	if a := os.Getenv("STAGING_REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6580"
}

// RealRedis يفتح ذاكرةَ التجهيز، **ويتخطّى الاختبارَ إن لم تُجب.**
//
// **ولا يُنادى إلّا في اختبارات `P-0`** — **وبقيّةُ الحزمة على
// `miniredis` كما كانت**، فلا خادمَ خارجيٌّ يلزم في كلّ تشغيل.
func RealRedis(t *testing.T) *redis.Client {
	t.Helper()
	if os.Getenv("RAHALGO_STAGING") != "1" {
		t.Skip("اختبارُ تجهيزٍ — يحتاج RAHALGO_STAGING=1")
	}
	c := redis.NewClient(&redis.Options{
		Addr: StagingRedisAddr(),
		// **ومهلةٌ قصيرة**: **حين تُسقَط الذاكرةُ نريد الجوابَ سريعاً**
		// — **ومهلةٌ افتراضيّةٌ تجعل الاختبارَ يبدو معلَّقاً.**
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		t.Skipf("لا ذاكرةَ تجهيزٍ على %s: %v", StagingRedisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
