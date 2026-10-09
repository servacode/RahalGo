package server

// ══════════════════════════════════════════════════════════════════════
// **بلاطاتُ الخريطة من سيرفرنا — وسيطٌ يحفظ ما جلب** (طلبُ المالك ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// **رُئي على الجوال**: الخريطةُ تتأخّر حتّى تُفتح. **وقِيس**: كلُّ بلاطةٍ من
// `tile.openstreetmap.org` تأخذ ثانيةً ونصفاً، والشاشةُ تطلب عشرين. **وسياستُهم
// تنهى عن الاستعمال الثقيل** — ونحن نُنزّل لكلّ سائقٍ منطقتَه كاملة.
//
// **فصار التطبيقُ يطلب البلاطةَ منّا**: نعطيها من القرص إن كانت عندنا، **وإلّا
// جلبناها مرّةً من المصدر وحفظناها** — فالزبونُ الثاني يأخذها من عندنا فوراً،
// **ولا يطلب المصدرَ إلّا أوّلُ من رآها.**
//
// # الحدود
//
//   - **التكبيرُ ٠–١٩ والإحداثيّاتُ داخل الشبكة** — وإلّا ٤٠٤ بلا طلبٍ للمصدر.
//   - **طلبان للبلاطة نفسِها في اللحظة نفسِها يجلبانها مرّةً** (`singleflight`).
//   - **والبلاطةُ تُجدَّد بعد ثلاثين يوماً** — والشوارعُ تتبدّل على مهل.
//   - **وتعثّرُ المصدر يُعطي النسخةَ القديمةَ إن وُجدت** — خريطةٌ قديمةٌ خيرٌ من فراغ.
//
// **ولا تحميلَ مسبقاً جماعيّاً**: سياسةُ OSM تمنعه صراحةً. ما يُطلب يُحفظ، لا أكثر.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
)

const tileMaxAge = 30 * 24 * time.Hour

var (
	tileFlight singleflight.Group
	tileClient = &http.Client{Timeout: 8 * time.Second}
)

// tileUpstream **مصدرُ البلاطات خلف الوسيط** — `MAP_TILE_UPSTREAM` أو OSM.
func tileUpstream() string {
	if v := os.Getenv("MAP_TILE_UPSTREAM"); v != "" {
		return v
	}
	return defaultTileURL
}

func (s *Server) tilesDir() string {
	base := "./uploads"
	if s.cfg != nil && s.cfg.UploadsDir != "" {
		base = s.cfg.UploadsDir
	}
	return filepath.Join(base, "tiles")
}

func (s *Server) handleMapTile(w http.ResponseWriter, r *http.Request) {
	z, ez := strconv.Atoi(chi.URLParam(r, "z"))
	x, ex := strconv.Atoi(chi.URLParam(r, "x"))
	yStr := chi.URLParam(r, "y")
	if len(yStr) > 4 && yStr[len(yStr)-4:] == ".png" {
		yStr = yStr[:len(yStr)-4]
	}
	y, ey := strconv.Atoi(yStr)
	if ez != nil || ex != nil || ey != nil || z < 0 || z > 19 || x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.tilesDir(), strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y)+".png")

	fresh := false
	if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) < tileMaxAge {
		fresh = true
	}
	if !fresh {
		key := fmt.Sprintf("%d/%d/%d", z, x, y)
		_, err, _ := tileFlight.Do(key, func() (any, error) {
			return nil, s.fetchTile(context.WithoutCancel(r.Context()), z, x, y, path)
		})
		if err != nil {
			if _, statErr := os.Stat(path); statErr != nil {
				s.logger.Warn("بلاطةُ الخريطة لم تُجلب", "tile", key, "error", err)
				http.Error(w, "tile unavailable", http.StatusBadGateway)
				return
			}
			// **والقديمةُ تُعطى** — خريطةٌ قديمةٌ خيرٌ من فراغ.
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, path)
}

func (s *Server) fetchTile(ctx context.Context, z, x, y int, path string) error {
	url := tileUpstream()
	for _, kv := range [][2]string{{"{z}", strconv.Itoa(z)}, {"{x}", strconv.Itoa(x)}, {"{y}", strconv.Itoa(y)}} {
		url = strings.ReplaceAll(url, kv[0], kv[1])
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// **وسياسةُ OSM تطلب هويّةً تُعرَف** — لا عميلاً مجهولاً.
	req.Header.Set("User-Agent", "RahalGo/1.0 (+https://rahalgo.com; tiles proxy)")
	res, err := tileClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
