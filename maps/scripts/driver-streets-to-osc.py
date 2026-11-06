#!/usr/bin/env python3
# ══════════════════════════════════════════════════════════════════════
# **شوارعُ مشي السائقين الموافَقُ عليها ← ملفُّ تغييرٍ موصولٌ بالشوارع**
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-١١-٠٦: «هلّق» — الخطوة ٤ من «الخريطة من مشي السائقين».)
#
# المدخل: ملفُّ التصدير من اللوحة (`rahalgo-streets.osm` — صفحة «الخريطة من
# السائقين» ← تصدير OSM، أو `/admin/ops-map/street-candidates/export?format=osm`)
# وطرقُ الخريطة الأصليّة حوله (`osmium tags-filter … w/highway` ثمّ `extract -b`).
#
# المخرج: ملفُّ `osmChange` يُطبَّق بـ`osmium apply-changes` على ملفّ الخريطة.
#
# # لماذا لا `osmium merge` وحده
#
# **طريقٌ جديدٌ لا يلمس الشوارعَ عقدةً بعقدة جزيرةٌ في التوجيه** — Valhalla وOSRM
# يصلان الطرقَ بالعقد المشتركة وحدَها، فيُرسم الشارعُ ولا يمرّ به مسار.
# فهنا يُوصل كلُّ طرفٍ من طرفَي الشارع بأقرب طريقٍ قابلٍ للسير ضمن ٣٥ م:
#   • إن كانت عقدةٌ قائمةٌ ضمن ٣ م من الإسقاط — تُستعمل هي.
#   • وإلّا تُنشأ عقدةُ وصلٍ على القطعة نفسِها وتُدرج في الطريق القائم (`modify`).
# ويُسقط الشارعُ إن صار في الخريطة أصلاً (٨٠٪ من عقده ضمن ١٠ م من طريقٍ قائم) —
# فلا يتكرّر حين تضيفه OpenStreetMap لاحقاً.
#
#   الاستعمال:
#     python3 driver-streets-to-osc.py bbox  streets.osm            # يطبع حدودَ الاستخراج
#     python3 driver-streets-to-osc.py build highways.osm streets.osm out.osc
#     python3 driver-streets-to-osc.py selftest
#
# **مكتبةُ بايثون القياسيّة وحدَها** — كأخواته `name-ar-to-osc.py` و`buildings-to-osm.py`.
import math
import sys
import xml.etree.ElementTree as ET
from xml.sax.saxutils import quoteattr

# الطرقُ التي يسلكها التوجيه (درّاجةٌ ناريّة) — لا يُوصل بممشى ولا درج.
ROUTABLE = {
    "motorway", "motorway_link", "trunk", "trunk_link", "primary", "primary_link",
    "secondary", "secondary_link", "tertiary", "tertiary_link", "unclassified",
    "residential", "service", "living_street", "track", "road",
}
CONNECT_M = 35.0   # أبعدُ طريقٍ يُوصل به طرفُ الشارع
SNAP_M = 3.0       # عقدةٌ قائمةٌ بهذا القرب تُستعمل بدل إنشاء عقدة
MAPPED_M = 10.0    # عقدةُ شارعٍ بهذا القرب من طريقٍ قائمٍ «مرسومةٌ أصلاً»
MAPPED_SHARE = 0.8
BBOX_MARGIN = 0.005  # درجة (~٥٠٠ م) حول الشوارع
# **عقدُ الوصل في نطاقٍ رابع** — التصديرُ: عقدٌ ٢٥ ملياراً وطرقٌ ١٦، والاتّجاهُ ٢٦ و١٧، والمباني ٢٠ و١٥.
JUNCTION_START = 27_000_000_000
CELL = 0.0005      # خليّةُ الفهرس بالدرجة (~٥٠ م)
M_PER_DEG = 111320.0


class Way:
    def __init__(self, wid, refs, tags, version):
        self.id, self.refs, self.tags, self.version = wid, refs, tags, version


def parse(path):
    """عقدُ الملفّ وطرقُه: {id: (lat, lon)} و[Way]."""
    nodes, ways = {}, []
    for _, el in ET.iterparse(path, events=("end",)):
        if el.tag == "node":
            nodes[int(el.get("id"))] = (float(el.get("lat")), float(el.get("lon")))
            el.clear()
        elif el.tag == "way":
            refs = [int(n.get("ref")) for n in el.findall("nd")]
            tags = {t.get("k"): t.get("v") for t in el.findall("tag")}
            ways.append(Way(int(el.get("id")), refs, tags, int(el.get("version") or 1)))
            el.clear()
    return nodes, ways


def proj(lat0):
    k = math.cos(math.radians(lat0))
    return lambda lat, lon: (lon * M_PER_DEG * k, lat * M_PER_DEG)


def seg_project(p, a, b):
    """(المسافة، النسبة t) من p إلى القطعة ab بالمتر."""
    ax, ay = a
    bx, by = b
    dx, dy = bx - ax, by - ay
    L = dx * dx + dy * dy
    t = 0.0 if L == 0 else max(0.0, min(1.0, ((p[0] - ax) * dx + (p[1] - ay) * dy) / L))
    qx, qy = ax + t * dx, ay + t * dy
    return math.hypot(p[0] - qx, p[1] - qy), t


class Index:
    """فهرسُ قطع الطرق القابلة للسير بشبكة خلايا."""

    def __init__(self, nodes, ways):
        self.nodes, self.cells = nodes, {}
        for w in ways:
            if w.tags.get("highway") not in ROUTABLE:
                continue
            for i in range(len(w.refs) - 1):
                a, b = w.refs[i], w.refs[i + 1]
                if a not in nodes or b not in nodes:
                    continue
                (la, oa), (lb, ob) = nodes[a], nodes[b]
                for ci in range(int(math.floor(min(la, lb) / CELL)), int(math.floor(max(la, lb) / CELL)) + 1):
                    for cj in range(int(math.floor(min(oa, ob) / CELL)), int(math.floor(max(oa, ob) / CELL)) + 1):
                        self.cells.setdefault((ci, cj), []).append((w, a, b))

    def nearest(self, lat, lon, radius_m):
        """أقربُ قطعة: (المسافة، الطريق، a، b، t) أو None."""
        to = proj(lat)
        p = to(lat, lon)
        r = int(math.ceil(radius_m / (CELL * M_PER_DEG * math.cos(math.radians(lat))))) + 1
        ci0, cj0 = int(math.floor(lat / CELL)), int(math.floor(lon / CELL))
        best, seen = None, set()
        for ci in range(ci0 - r, ci0 + r + 1):
            for cj in range(cj0 - r, cj0 + r + 1):
                for w, a, b in self.cells.get((ci, cj), ()):
                    key = (w.id, a, b)
                    if key in seen:
                        continue
                    seen.add(key)
                    d, t = seg_project(p, to(*self.nodes[a]), to(*self.nodes[b]))
                    if d <= radius_m and (best is None or d < best[0] or (d == best[0] and w.id < best[1].id)):
                        best = (d, w, a, b, t)
        return best


def build(hw_nodes, hw_ways, st_nodes, st_ways):
    """يردّ (create_nodes, create_ways, modify_ways, report)."""
    idx = Index(hw_nodes, hw_ways)
    nodes_all = dict(hw_nodes)
    created_nodes = {}   # id → (lat, lon)
    modified = {}        # way id → Way (نسخةٌ معدَّلة)
    out_ways = []
    report = {"streets": 0, "already_mapped": 0, "ends_connected": 0, "ends_open": 0, "junctions": 0}
    jid = JUNCTION_START

    for w in sorted(st_ways, key=lambda x: x.id):
        coords = [st_nodes[r] for r in w.refs if r in st_nodes]
        if len(coords) < 2:
            continue
        near = sum(1 for c in coords if idx.nearest(c[0], c[1], MAPPED_M) is not None)
        if near >= MAPPED_SHARE * len(coords):
            report["already_mapped"] += 1
            continue
        report["streets"] += 1
        refs = [r for r in w.refs if r in st_nodes]
        for r in refs:
            created_nodes[r] = st_nodes[r]
        for end in (0, -1):
            lat, lon = st_nodes[refs[end]]
            hit = idx.nearest(lat, lon, CONNECT_M)
            if hit is None:
                report["ends_open"] += 1
                continue
            d, road, a, b, t = hit
            to = proj(lat)
            pa, pb = to(*nodes_all[a]), to(*nodes_all[b])
            seg_len = math.hypot(pb[0] - pa[0], pb[1] - pa[1])
            if t * seg_len <= SNAP_M:
                join = a
            elif (1 - t) * seg_len <= SNAP_M:
                join = b
            else:
                jid += 1
                join = jid
                (la, oa), (lb, ob) = nodes_all[a], nodes_all[b]
                pos = (la + t * (lb - la), oa + t * (ob - oa))
                created_nodes[join] = pos
                nodes_all[join] = pos
                report["junctions"] += 1
                cur = modified.get(road.id) or Way(road.id, list(road.refs), dict(road.tags), road.version + 1)
                # **العقدةُ تُدرج بين a وb في النسخة الحاليّة** — وقد أُدرجت قبلها عقدُ وصلٍ أخرى على القطعة
                # نفسِها، فيُبحث عن الموضع بين أقرب جارين على الخطّ لا بين a وb حرفاً.
                cur.refs = insert_on_segment(cur.refs, a, b, join, nodes_all, pos)
                modified[road.id] = cur
            if join == refs[end]:
                continue
            if end == 0:
                refs = [join] + refs
            else:
                refs = refs + [join]
            report["ends_connected"] += 1
        out_ways.append(Way(w.id, refs, w.tags, 1))
    return created_nodes, out_ways, modified, report


def insert_on_segment(refs, a, b, new, nodes, pos):
    """يُدرج `new` في `refs` على القطعة الأصليّة a→b (أو b→a) بترتيب البُعد عن a."""
    for i in range(len(refs) - 1):
        if refs[i] == a or refs[i] == b:
            start = refs[i]
            other = b if start == a else a
            j = i + 1
            while j < len(refs) and refs[j] != other:
                j += 1
            if j >= len(refs):
                continue
            s = nodes[start]
            dist = lambda n: (nodes[n][0] - s[0]) ** 2 + (nodes[n][1] - s[1]) ** 2
            dn = (pos[0] - s[0]) ** 2 + (pos[1] - s[1]) ** 2
            k = i + 1
            while k < j and dist(refs[k]) < dn:
                k += 1
            return refs[:k] + [new] + refs[k:]
    return refs


def node_xml(nid, lat, lon, version=1):
    return '    <node id="%d" version="%d" lat="%.7f" lon="%.7f"/>\n' % (nid, version, lat, lon)


def way_xml(w):
    s = '    <way id="%d" version="%d">\n' % (w.id, w.version)
    for r in w.refs:
        s += '      <nd ref="%d"/>\n' % r
    for k in sorted(w.tags):
        s += "      <tag k=%s v=%s/>\n" % (quoteattr(k), quoteattr(w.tags[k]))
    return s + "    </way>\n"


def osc(created_nodes, out_ways, modified):
    s = '<?xml version="1.0" encoding="UTF-8"?>\n<osmChange version="0.6" generator="rahalgo-driver-streets">\n'
    s += "  <create>\n"
    for nid in sorted(created_nodes):
        lat, lon = created_nodes[nid]
        s += node_xml(nid, lat, lon)
    for w in sorted(out_ways, key=lambda x: x.id):
        s += way_xml(w)
    s += "  </create>\n"
    if modified:
        s += "  <modify>\n"
        for wid in sorted(modified):
            s += way_xml(modified[wid])
        s += "  </modify>\n"
    return s + "</osmChange>\n"


def cmd_bbox(streets):
    nodes, _ = parse(streets)
    if not nodes:
        sys.exit("لا شوارعَ في الملفّ")
    lats = [v[0] for v in nodes.values()]
    lons = [v[1] for v in nodes.values()]
    print("%.5f,%.5f,%.5f,%.5f" % (min(lons) - BBOX_MARGIN, min(lats) - BBOX_MARGIN,
                                   max(lons) + BBOX_MARGIN, max(lats) + BBOX_MARGIN))


def cmd_build(highways, streets, out):
    hn, hw = parse(highways)
    sn, sw = parse(streets)
    cn, ow, mod, rep = build(hn, hw, sn, sw)
    with open(out, "w", encoding="utf-8", newline="\n") as f:
        f.write(osc(cn, ow, mod))
    print("driver-streets: streets=%(streets)d already_mapped=%(already_mapped)d "
          "ends_connected=%(ends_connected)d ends_open=%(ends_open)d junctions=%(junctions)d" % rep)


def selftest():
    # طريقٌ أفقيٌّ قائم (عقدتان بينهما ~٢٠٠ م)، وشارعٌ جديدٌ عموديٌّ يبدأ على بُعد ~٢٠ م منه.
    hn = {1: (35.9500, 39.0000), 2: (35.9500, 39.0022)}
    hw = [Way(100, [1, 2], {"highway": "residential"}, 3), Way(101, [1, 2], {"highway": "footway"}, 1)]
    sn = {25_000_000_001: (35.95018, 39.0011), 25_000_000_002: (35.9520, 39.0011)}
    sw = [Way(16_000_000_001, [25_000_000_001, 25_000_000_002], {"highway": "residential", "name": "حارة"}, 1)]
    cn, ow, mod, rep = build(hn, hw, sn, sw)
    assert rep["streets"] == 1 and rep["junctions"] == 1 and rep["ends_connected"] == 1, rep
    assert rep["ends_open"] == 1, rep
    j = JUNCTION_START + 1
    assert ow[0].refs[0] == j, ow[0].refs
    assert mod[100].refs == [1, j, 2] and mod[100].version == 4, mod
    assert 101 not in mod, "وُصل بممشى"
    # شارعٌ مرسومٌ أصلاً يُسقط.
    sn2 = {25_000_000_010: (35.95002, 39.0002), 25_000_000_011: (35.95002, 39.0020)}
    sw2 = [Way(16_000_000_002, [25_000_000_010, 25_000_000_011], {"highway": "residential"}, 1)]
    _, ow2, _, rep2 = build(hn, hw, sn2, sw2)
    assert rep2["already_mapped"] == 1 and not ow2, rep2
    # طرفٌ قريبٌ من عقدةٍ قائمة يُوصل بها بلا عقدةٍ جديدة.
    sn3 = {25_000_000_020: (35.95015, 39.00001), 25_000_000_021: (35.9515, 39.0000)}
    sw3 = [Way(16_000_000_003, [25_000_000_020, 25_000_000_021], {"highway": "residential"}, 1)]
    _, ow3, mod3, rep3 = build(hn, hw, sn3, sw3)
    assert rep3["junctions"] == 0 and ow3[0].refs[0] == 1 and not mod3, (rep3, ow3[0].refs)
    out = osc(cn, ow, mod)
    ET.fromstring(out.encode("utf-8"))
    assert "<modify>" in out and 'k="name" v="حارة"' in out
    print("selftest ok")


if __name__ == "__main__":
    a = sys.argv[1:]
    if a[:1] == ["bbox"] and len(a) == 2:
        cmd_bbox(a[1])
    elif a[:1] == ["build"] and len(a) == 4:
        cmd_build(a[1], a[2], a[3])
    elif a[:1] == ["selftest"]:
        selftest()
    else:
        sys.exit(__doc__ or "usage: bbox STREETS | build HIGHWAYS STREETS OUT | selftest")
