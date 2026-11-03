"""يحوّل جدولَ التعريب إلى ملفّ تعديلات OSM يُطبَّق قبل Planetiler.

(قرارُ المالك ٢٠٢٦-١١-٠٣: «كلشي بالعربي لازم».)

كلُّ عنصرٍ اسمُه (`name`) في `maps/config/name-ar-overrides.csv` وليس له `name:ar`
يأخذ `name:ar` من الجدول — فيُخبَز العربيُّ في البلاطات نفسها، وتصير كلُّ
الأنماط (ويب، أندرويد) تقرؤه دون جدولٍ في النمط.

    osmium tags-filter merged.osm.pbf nwr/name -o named.osm.pbf --overwrite
    osmium cat named.osm.pbf -f osm -o named.osm --overwrite
    python3 name-ar-to-osc.py named.osm names.osc
    osmium apply-changes merged.osm.pbf names.osc -o merged-ar.osm.pbf --overwrite

(مُدمَجةٌ في `rebuild-all.sh` بعد الدمج. لم تُجرَّب على اللقطة الكاملة بعد.)
"""
import csv
import os
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

HERE = Path(__file__).resolve().parent
# NAME_AR_CSV overrides; repo layout maps/config; or the csv next to the script on the server.
CSV = next(
    (p for p in (
        Path(os.environ.get("NAME_AR_CSV", "")),
        HERE.parent / "config" / "name-ar-overrides.csv",
        HERE / "name-ar-overrides.csv",
    ) if p.is_file()),
    HERE.parent / "config" / "name-ar-overrides.csv",
)


def main(src: str, dst: str) -> None:
    with open(CSV, encoding="utf-8", newline="") as f:
        table = {r["name"]: r["name_ar"] for r in csv.DictReader(f) if r["name"] and r["name_ar"]}
    changed = 0
    with open(dst, "w", encoding="utf-8") as out:
        out.write('<?xml version="1.0" encoding="UTF-8"?>\n')
        out.write('<osmChange version="0.6" generator="rahalgo-name-ar">\n<modify>\n')
        for _, el in ET.iterparse(src, events=("end",)):
            if el.tag not in ("node", "way", "relation"):
                continue
            tags = {t.get("k"): t.get("v") for t in el.findall("tag")}
            ar = table.get(tags.get("name", ""))
            if ar and not tags.get("name:ar"):
                ET.SubElement(el, "tag", {"k": "name:ar", "v": ar})
                out.write(ET.tostring(el, encoding="unicode"))
                out.write("\n")
                changed += 1
            el.clear()
        out.write("</modify>\n</osmChange>\n")
    print(f"name:ar added to {changed} elements", file=sys.stderr)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
