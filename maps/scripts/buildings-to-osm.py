#!/usr/bin/env python3
"""
══════════════════════════════════════════════════════════════════════
**مبانٍ من GeoJSON إلى OSM — لتدخل خطَّ الإنتاج بلا تغييرٍ فيه**
══════════════════════════════════════════════════════════════════════

(طلبُ المالك ٢٠٢٦-٠٨-٢٤.)

# ولماذا هذا الطريق

**البديلُ مصدرٌ ثانٍ في النمط** — ويقتضي: عنواناً ثانياً في الربط،
وأرشيفاً ثانياً في كلّ حزمةِ منطقة، وحقلاً ثانياً في الفهرس، وتغييراً
في مخزن الحزم. **أربعةُ مواضعَ في التطبيق وواحدٌ في الخادم.**

**وهذا الطريقُ يجعلها مبانيَ OSM عاديّة** — فتمرّ من Planetiler إلى
طبقة `building` القياسيّة. **ولا يتغيّر سطرٌ في التطبيق ولا في النمط
ولا في الفهرس** — والتطبيقُ الذي في جيب السائق يعرضها بلا تحديث.

# والمعرّفاتُ فوق سقف OSM

**ولا تصطدم بمعرّفٍ حقيقيّ** — وانظر `START_NODE`: **السالبةُ مرفوضةٌ
في Planetiler**، فتُختار أرقامٌ أكبرُ من كلّ ما في الخريطة.

# ولا تُجمع الطرقُ في الذاكرة

**وصيغةُ OSM تشترط العقدَ قبل الطرق** — فيُغرى المرءُ بجمع الطرق في
قائمةٍ حتّى تنتهي العقد. **وأربعةُ ملايينِ طريقٍ في ذاكرة بايثون
غيغاباتٌ**، والسيرفرُ سبعةٌ لا غير.

**فتُكتب العقدُ في ملفٍّ والطرقُ في ملفٍّ، ثمّ يُلصقان** — ذاكرةٌ
ثابتةٌ مهما كبر الملفّ.

# ولا يُرفع هذا إلى OpenStreetMap

**بياناتٌ مستخرَجةٌ آليّاً ترفعها آلةٌ تفسد الخريطةَ العامّة** —
ورفعُها يحتاج موافقةَ المجتمع وقواعدَ الاستيراد. **وهذه لخرائطنا
وحدَها.**

    الاستعمال:  python3 buildings-to-osm.py <out.osm> <ملفّ.geojsonl> ...
"""
import json
import os
import sys
import tempfile

# ══════════════════════════════════════════════════════════════════════
# **ومعرّفاتٌ موجبةٌ فوق أكبرِ معرّفٍ حقيقيّ**
# ══════════════════════════════════════════════════════════════════════
#
# **والسالبةُ عرفُ OSM لما لم يُرفع إليها** — وهي أوّلُ ما اخترتُ.
# **وPlanetiler يرفضها صراحةً**: `Negative OSM element IDs not
# supported` (قِيس ٢٠٢٦-٠٨-٢٤، فسقط بناءٌ بعد ساعةٍ من العمل).
#
# **فتُختار أرقامٌ فوق سقف OSM الحقيقيّ**: أكبرُ عقدةٍ اليوم نحو
# ثلاثةَ عشرَ ملياراً، وأكبرُ طريقٍ نحو مليارٍ ونصف. **وعشرون ملياراً
# للعقد وخمسةَ عشرَ للطرق تبعدان عنهما سنينَ من النموّ.**
#
# **ولا يُخشى تصادمٌ في الدمج** — والأرقامُ الأكبرُ تقع بعد الحقيقيّة
# في الترتيب، **فيخرج الملفُّ مرتّباً بلا فرز.**
START_NODE = 20_000_000_000
START_WAY = 15_000_000_000


def main() -> int:
    if len(sys.argv) < 3:
        print(__doc__, file=sys.stderr)
        return 2
    out_path = sys.argv[1]
    sources = sys.argv[2:]

    tmp_dir = os.path.dirname(os.path.abspath(out_path)) or "."
    ways_fd, ways_path = tempfile.mkstemp(dir=tmp_dir, suffix=".ways")
    os.close(ways_fd)

    node_id = START_NODE
    way_id = START_WAY
    kept = 0
    skipped = 0

    with open(out_path, "w", encoding="utf-8", buffering=1 << 20) as nodes, \
         open(ways_path, "w", encoding="utf-8", buffering=1 << 20) as ways:
        nodes.write('<?xml version="1.0" encoding="UTF-8"?>\n')
        nodes.write('<osm version="0.6" generator="rahalgo-buildings">\n')

        for path in sources:
            with open(path, encoding="utf-8") as fh:
                for line in fh:
                    if not line.startswith("{"):
                        continue
                    try:
                        geom = json.loads(line).get("geometry") or {}
                    except json.JSONDecodeError:
                        skipped += 1
                        continue
                    if geom.get("type") != "Polygon":
                        skipped += 1
                        continue
                    rings = geom.get("coordinates") or []
                    if not rings:
                        skipped += 1
                        continue
                    # **والحلقةُ الخارجيّةُ وحدَها** — **والفراغاتُ
                    # الداخليّةُ تحتاج علاقاتٍ (multipolygon)**، وهي
                    # تُضاعف الحجمَ ولا تُرى في خريطةِ سائق.
                    ring = rings[0]
                    if len(ring) < 4:
                        skipped += 1
                        continue
                    # **وآخرُ نقطةٍ تكرارُ الأولى في GeoJSON** — تُحذف
                    # ثمّ تُغلق الحلقةُ بالمرجع، **وإلّا صار لكلّ مبنًى
                    # عقدةٌ زائدة.**
                    if ring[0] == ring[-1]:
                        ring = ring[:-1]

                    first = None
                    refs = []
                    for point in ring:
                        node_id += 1
                        if first is None:
                            first = node_id
                        refs.append(node_id)
                        nodes.write(
                            f'<node id="{node_id}" lat="{point[1]:.7f}" lon="{point[0]:.7f}"/>\n'
                        )
                    way_id += 1
                    ways.write(f'<way id="{way_id}">\n')
                    for ref in refs:
                        ways.write(f'<nd ref="{ref}"/>\n')
                    # **والحلقةُ تُغلق بأوّل عقدة** — شرطُ المضلّع في OSM.
                    ways.write(f'<nd ref="{first}"/>\n')
                    ways.write('<tag k="building" v="yes"/>\n')
                    # **ومصدرُها يُذكر في الوسم** — فيُعرف بعد سنةٍ من
                    # أين جاءت، **ولا تُظنّ مسحاً بشريّاً.**
                    ways.write('<tag k="source" v="Microsoft ML building footprints"/>\n')
                    ways.write('</way>\n')
                    kept += 1

    # **ثمّ تُلصق الطرقُ بعد العقد** — نسخٌ متدفّقٌ بلا ذاكرة.
    with open(out_path, "a", encoding="utf-8", buffering=1 << 20) as out, \
         open(ways_path, encoding="utf-8", buffering=1 << 20) as ways:
        while True:
            chunk = ways.read(1 << 20)
            if not chunk:
                break
            out.write(chunk)
        out.write("</osm>\n")
    os.unlink(ways_path)

    print(f"buildings={kept} skipped={skipped}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
