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
}

// EnableCache **يشغّل الذاكرة** بمهلة — للخادم الحيّ وحدَه.
func (s *Store) EnableCache(ttl time.Duration) {
	if s == nil || ttl <= 0 {
		return
	}
	s.c = &memo{ttl: ttl}
}

// Invalidate **تُقرأ الصفوفُ من جديدٍ في السؤال التالي.**
func (s *Store) Invalidate() {
	if s == nil || s.c == nil {
		return
	}
	s.c.mu.Lock()
	s.c.at = time.Time{}
	s.c.mu.Unlock()
}

// cached **قيمةُ مفتاحٍ من الذاكرة** — `ok=false` حين لا ذاكرةَ أو تعثّرت القراءة.
func (s *Store) cached(ctx context.Context, key string) (raw []byte, found, ok bool) {
	if s == nil || s.c == nil || s.db == nil {
		return nil, false, false
	}
	c := s.c
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.vals == nil || time.Since(c.at) > c.ttl {
		rows, err := s.db.Query(ctx, `SELECT key, value FROM app_settings`)
		if err != nil {
			return nil, false, false
		}
		vals := make(map[string][]byte, 256)
		for rows.Next() {
			var k string
			var v []byte
			if err := rows.Scan(&k, &v); err != nil {
				rows.Close()
				return nil, false, false
			}
			vals[k] = v
		}
		rows.Close()
		if rows.Err() != nil {
			return nil, false, false
		}
		c.vals, c.at = vals, time.Now()
	}
	raw, found = c.vals[key]
	return raw, found, true
}
