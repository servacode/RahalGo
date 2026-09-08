package settings

// ══════════════════════════════════════════════════════════════════════
// **قراءةٌ واحدةٌ متماسكةٌ لمفاتيحَ تُقرأ معاً** — `XQ-2`
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **`GetInt` و`GetString` كلٌّ منها استعلامٌ بذاته** — **وتبديلٌ
// إداريٌّ يقع بينها يعطي المنادي نصفَ عقدٍ ونصفَ آخر.**
//
// **وقيس** (دورةُ إصلاحٍ ٣٢): **لقطةُ اقتصادِ طلبٍ خرجت `40/10`** —
// **نسبةُ المنصّة من الإعداد الجديد ونسبةُ المندوب من القديم**،
// **وذاك عقدٌ لم يوافق عليه أحد.**
//
// # فاستعلامٌ واحد
//
// **صورةُ القاعدة واحدةٌ في العبارة الواحدة** — **فإمّا القديمُ كلُّه
// أو الجديدُ كلُّه.**
//
// # والافتراضُ من الفهرس كما دائماً
//
// **ولا يأخذ المنادي احتياطيّاً بيده** — القاعدةُ نفسُها التي جمعت
// `GetInt`/`GetString` في موضعٍ واحد.

import (
	"context"
	"encoding/json"
)

// Coherent قراءةُ مفاتيحَ في لحظةٍ واحدة.
//
// **وما لا صفَّ له يأخذ افتراضَ الفهرس** — **وما ليس في الفهرس أصلاً
// لا يُردّ**، فلا يُقرأ اسمٌ لا يعرفه المعجم.
type Coherent map[string]json.RawMessage

// ReadCoherent يقرأ المفاتيحَ المعطاةَ في استعلامٍ واحد.
func (s *Store) ReadCoherent(ctx context.Context, keys ...string) (Coherent, error) {
	out := Coherent{}
	if s == nil || s.db == nil {
		return nil, errNoStore
	}
	rows, err := s.db.Query(ctx,
		`SELECT key, value FROM app_settings WHERE key = ANY($1)`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var raw []byte
		if err := rows.Scan(&k, &raw); err != nil {
			return nil, err
		}
		out[k] = append(json.RawMessage(nil), raw...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Int قيمةٌ عدديّةٌ من القراءة المتماسكة — **والافتراضُ من الفهرس.**
func (c Coherent) Int(key string) int64 {
	if raw, ok := c[key]; ok {
		var v int64
		if json.Unmarshal(raw, &v) == nil {
			return v
		}
	}
	if def, ok := Lookup(key); ok {
		switch n := def.Default.(type) {
		case int:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		}
	}
	return 0
}

// String قيمةٌ نصّيّةٌ من القراءة المتماسكة — **والافتراضُ من الفهرس.**
func (c Coherent) String(key string) string {
	if raw, ok := c[key]; ok {
		var v string
		if json.Unmarshal(raw, &v) == nil && v != "" {
			return v
		}
	}
	if def, ok := Lookup(key); ok {
		if str, ok := def.Default.(string); ok {
			return str
		}
	}
	return ""
}
