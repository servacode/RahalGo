package media

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Purge **يمحو وسائطَ صنفٍ أقدمَ من حدّ — الصفَّ وملفّاته معاً** (فيبقى `mediacheck` سليماً:
// لا صفَّ بلا ملفّ ولا ملفَّ بلا صفّ). ويردّ كم مُحي.
//
// **صورُ عدّاد الدرّاجة بعد مدّة حفظها** (قرارُ المالك ٢٠٢٦-١١-٠٥): الرقمُ يبقى في
// `driver_odometer_readings` والصورةُ تُمحى (`media_id` يصير فارغاً بقيده).
//
// **والصفُّ يُحذف أوّلاً** — ملفٌّ بقي بعد صفٍّ محذوفٍ لا يُخدَم أبداً (مجهولُ النسب يُحجَب،
// `IsProtected`)، وصفٌّ بقي بلا ملفّ يُعرض صورةً مكسورة.
func (s *Service) Purge(ctx context.Context, kind string, before time.Time, limit int) (int, error) {
	if !validKinds[kind] || limit <= 0 {
		return 0, nil
	}
	rows, err := s.db.Query(ctx, `
		DELETE FROM media
		 WHERE id IN (SELECT id FROM media WHERE kind = $1 AND created_at < $2
		               ORDER BY created_at LIMIT $3)
		RETURNING path, thumb_path`, kind, before, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var path, thumb string
		if err := rows.Scan(&path, &thumb); err != nil {
			return n, err
		}
		full := filepath.Join(s.dir, filepath.FromSlash(path))
		_ = os.Remove(full)
		_ = os.Remove(filepath.Join(s.dir, filepath.FromSlash(thumb)))
		for _, w := range VariantWidths {
			_ = os.Remove(variantPath(full, w))
		}
		s.protCache.Delete(path)
		n++
	}
	return n, rows.Err()
}
