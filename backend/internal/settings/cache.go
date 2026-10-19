package settings

// ══════════════════════════════════════════════════════════════════════
// **الإعداداتُ في ذاكرة الخادم — قراءةٌ واحدةٌ كلَّ ثانيتين** (طلبُ المالك ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// **كشفه اختبارُ التحمّل**: `‎/public/platform` — أوّلُ ما يناديه كلُّ تطبيقٍ —
// **يسأل القاعدةَ نحو ثلاثين سؤالاً** لأنّ كلَّ مفتاحٍ يُقرأ وحدَه، فكان أبطأَ
// بابٍ في المنصّة (ثانيةٌ عند مئة مستخدم). **فتُقرأ الصفوفُ كلُّها بسؤالٍ واحد**
// وتبقى في الذاكرة ثانيتين.
//
// # والطزاجة
//
//   - **كلُّ كتابةٍ من هذا المخزن تُبطل الذاكرة** — فمن حفظ يرى ما حفظ.
//   - **وإشارةُ «settings» الحيّة تُبطلها** بعد التثبيت (`Server.touch`) — فكتابةٌ
//     داخل معاملةٍ لا تُقرأ قديمةً بعد أن تُثبَّت.
//   - **وما كُتب من خارج العمليّة** (عمليّةٌ ثانية، يدٌ في القاعدة) يظهر بعد ثانيتين
//     على الأكثر.
//
// **ولا ذاكرةَ إلّا لمن طلبها** (`EnableCache`) — الخادمُ وحدَه. الاختباراتُ تكتب
// في القاعدة مباشرةً وتنتظر أثرَها فوراً، **والمخزنُ المربوطُ بمعاملة** (`On`) يقرأ
// من المعاملة دائماً.
//
// **وتعثّرُ القراءة الجماعيّة لا يُسقط شيئاً**: يُقرأ المفتاحُ وحدَه كما كان.
//
// # ولا قفلَ والقاعدةُ تُسأل (مراقبةُ الإنتاج ٢٠٢٦-١٠-١٩)
//
// **كان التجديدُ يمسك القفلَ وهو ينتظر القاعدة** — فكلُّ قارئٍ بعده يقف، **ومنهم
// من يمسك وصلةً في معاملة.** فإن وقف عشرون والمسبحُ عشرون **لم يجد التجديدُ وصلةً
// وعلق الكلّ** حتّى تتعب الطلباتُ وتُلغى: «طلبات تعبت من الانتظار ٧١» على الإنتاج
// في ثلاث ساعات، ومعها ٥٠٣ مساءَ ٢٠٢٦-١٠-١٨.
//
// **فالآن قارئٌ واحدٌ يجدّد خارجَ القفل، والباقون يقرؤون ما في الذاكرة** — قديماً
// بثانيتين على الأكثر. **والقارئُ الذي يجدّد يرى الجديد**، **وإبطالٌ وقع والتجديدُ
// جارٍ لا يُعلَّم الجوابَ القديمَ طازجاً.**

import (
	"context"
	"sync"
	"time"
)

type memo struct {
	mu   sync.Mutex
	ttl  time.Duration
	at   time.Time
	vals map[string][]byte
	// loading **تجديدٌ جارٍ** — فلا يبدأ ثانٍ، ويقرأ الباقون ما في الذاكرة.
	loading bool
	// gen **رقمُ الإبطال** — يزيد مع كلّ `Invalidate`، **فجوابٌ بدأ قبله لا يُعلَّم طازجاً.**
	gen uint64
}

// EnableCache **يشغّل الذاكرة** بمهلة — للخادم الحيّ وحدَه.
func (s *Store) EnableCache(ttl time.Duration) {
	if s == nil || ttl <= 0 {
		return
	}
	s.c = &memo{ttl: ttl}
}

// Invalidate **يُبطل الذاكرة** — القراءةُ التالية تجدّد.
func (s *Store) Invalidate() {
	if s == nil || s.c == nil {
		return
	}
	s.c.mu.Lock()
	s.c.at = time.Time{}
	s.c.gen++
	s.c.mu.Unlock()
}

// cached **المفتاحُ من الذاكرة** — و`ok=false` إن لم تكن ذاكرةٌ أو تعذّر ملؤها أوّلَ مرّة.
func (s *Store) cached(ctx context.Context, key string) (raw []byte, found, ok bool) {
	if s == nil || s.c == nil || s.db == nil {
		return nil, false, false
	}
	c := s.c
	c.mu.Lock()
	if c.vals != nil && (time.Since(c.at) <= c.ttl || c.loading) {
		raw, found = c.vals[key]
		c.mu.Unlock()
		return raw, found, true
	}
	c.loading = true
	gen := c.gen
	c.mu.Unlock()

	vals, err := s.loadAll(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.loading = false
	if err != nil {
		if c.vals == nil {
			return nil, false, false
		}
		raw, found = c.vals[key]
		return raw, found, true
	}
	c.vals = vals
	if c.gen == gen {
		c.at = time.Now()
	}
	raw, found = vals[key]
	return raw, found, true
}

// loadAll **الصفوفُ كلُّها بسؤالٍ واحد** — بلا قفل.
func (s *Store) loadAll(ctx context.Context) (map[string][]byte, error) {
	rows, err := s.db.Query(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vals := make(map[string][]byte, 256)
	for rows.Next() {
		var k string
		var v []byte
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		vals[k] = v
	}
	return vals, rows.Err()
}
