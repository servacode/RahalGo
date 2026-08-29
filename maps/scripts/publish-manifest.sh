#!/bin/sh
# ══════════════════════════════════════════════════════════════════════
# **نشرُ النسخة الجديدة — الفهرسُ آخرُ ما يُبدَّل**
# ══════════════════════════════════════════════════════════════════════
#
# (طلبُ المالك ٢٠٢٦-٠٨-٢٤.)
#
# # ولماذا آخرُ ما يُبدَّل
#
# **العملاءُ يقرؤون الفهرسَ ثمّ ينزّلون ما فيه** — فمن بدّله قبل أن
# تستقرّ الملفّاتُ أرسل السائقين إلى عنوانٍ لا شيءَ فيه. **والملفّاتُ
# أوّلاً والفهرسُ آخراً.**
#
# # والقديمُ يبقى
#
# **ولا تُحذف نسخةُ ٢٠٢٦-٠٨-٢٠** — **وجهازٌ لم يُحدِّث فهرسَه بعدُ
# يطلبها**، فيجدها. وتُحذف بعد أسبوعٍ بيد.
#
#   الاستعمال:  sh publish-manifest.sh
set -eu

M=/srv/rahalgo/maps
DV=2026-08-24
R="$M/regions/$DV"

[ -s "$R/region-raqqa.pmtiles" ] || { echo "!! raqqa missing"; exit 3; }
[ -s "$R/region-damascus.pmtiles" ] || { echo "!! damascus missing"; exit 3; }
[ -s "$M/base/$DV/syria.pmtiles" ] || { echo "!! base missing"; exit 3; }

cp -f "$M/manifest.json" "$M/manifest.json.$(date -u +%Y%m%d%H%M).bak"

python3 - "$M" "$DV" <<'PY'
import hashlib
import json
import os
import sys

root, dv = sys.argv[1], sys.argv[2]


def stamp(path: str) -> tuple[int, str]:
    """**الحجمُ والبصمة** — يُقرآن من الملفّ لا يُنسخان من سابق."""
    h = hashlib.sha256()
    size = 0
    with open(path, "rb") as fh:
        while True:
            chunk = fh.read(1 << 20)
            if not chunk:
                break
            size += len(chunk)
            h.update(chunk)
    return size, h.hexdigest()


path = os.path.join(root, "manifest.json")
with open(path, encoding="utf-8") as fh:
    man = json.load(fh)

man["dataVersion"] = dv
# **والبناءُ يُذكر** — فيُعرف من أين جاءت المباني بعد سنة.
man["buildings"] = "Microsoft ML building footprints (2026-08-13)"

base_path = os.path.join(root, "base", dv, "syria.pmtiles")
size, digest = stamp(base_path)
base = man.get("base", {})
base["url"] = f"base/{dv}/syria.pmtiles"
base["bytes"] = size
base["sha256"] = digest
man["base"] = base

for region in man.get("regions", []):
    rid = region["id"]
    rp = os.path.join(root, "regions", dv, f"region-{rid}.pmtiles")
    if not os.path.exists(rp):
        print(f"!! missing region file: {rp}", file=sys.stderr)
        raise SystemExit(4)
    size, digest = stamp(rp)
    region["url"] = f"regions/{dv}/region-{rid}.pmtiles"
    region["bytes"] = size
    region["sha256"] = digest
    region["dataVersion"] = dv
    print(f"   {rid}: {size} bytes")

tmp = path + ".tmp"
with open(tmp, "w", encoding="utf-8") as fh:
    json.dump(man, fh, ensure_ascii=False, indent=2)
    fh.write("\n")
# **ويُستبدل بحركةٍ ذرّيّة** — **وعميلٌ قرأ نصفَ فهرسٍ يسقط**، ولا
# يعرف صاحبُه لماذا.
os.replace(tmp, path)
print(f"   dataVersion={dv}")
PY

echo "-- published"
grep -E '"dataVersion"|"bytes"' "$M/manifest.json" | head -8
