package server

import (
	"net/http"
)

/*
**أسلوبُ الخريطة — يُخدَم من المحرّك لا من داخل التطبيق.**

(البندُ الرابع في قائمة المالك ٢٠٢٦-٠٨-١٢: «الخريطة تشتغل بلا إنترنت».)

# لماذا من هنا

**منزِّلُ المناطق في MapLibre يقرأ عنواناً شبكيّاً وحدَه.** جُرّب
`asset://` فردّ «تعذّر تحليل العنوان»، وجُرّب `file://` فردّ مثلَها
(قيسا على الجهاز ٢٠٢٦-٠٨-١٢) — **فبقي التنزيلُ على صفرٍ بلا سببٍ ظاهرٍ
في الشاشة.**

# وفائدةٌ ثانيةٌ أهمّ

**مصدرُ البلاطات يُبدَّل من هنا بلا نسخةٍ جديدةٍ من التطبيق.**
وسياسةُ `openstreetmap.org` تمنع الاستعمالَ الثقيل: **يوم يكبر عددُ
السائقين نضع خادمَنا** — سطرٌ واحدٌ في هذا الملفّ، **لا تحديثٌ ينتظره
كلُّ سائقٍ في المدينة.**

# ومفتوحٌ بلا توثيق

**الخريطةُ تُرسم قبل أن يدخل أحد** — كما `public/platform`.
*/

// tileURL مصدرُ البلاطات — **نفسُ ما يرسم به الويب** (`ui/map.tsx`).
const tileURL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"

func (s *Server) handleMapStyle(w http.ResponseWriter, r *http.Request) {
	style := `{
  "version": 8,
  "name": "RahalGo",
  "sources": {
    "osm": {
      "type": "raster",
      "tiles": ["` + tileURL + `"],
      "tileSize": 256,
      "minzoom": 0,
      "maxzoom": 19,
      "attribution": "© OpenStreetMap"
    }
  },
  "layers": [
    { "id": "bg", "type": "background", "paint": { "background-color": "#EDE7DF" } },
    { "id": "osm", "type": "raster", "source": "osm" }
  ]
}`

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// **ويُخزَّن يوماً في الوسيط** — الأسلوبُ لا يتبدّل كلَّ ساعة،
	// **وطلبُه مع كلّ فتحةِ خريطةٍ رحلةٌ زائدة.**
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(style))
}
