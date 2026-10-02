"""النمطُ المرجعيّ بشكل غوغل ماب — يكتب maps/style/rahalgo.style.json.

(طلبُ المالك ٢٠٢٦-١٠-٠١: «الخرائط مثل غوغل ماب لأبعد حدّ»، وصورتُه المرجع:
 أرضٌ رماديّةٌ فاتحة، شوارعُ بيضاءُ بحدودٍ رماديّة حتّى أصغر زقاق، طرقٌ سريعةٌ
 صفراء، حدائقُ خضراء، مستشفياتٌ ورديّة، ودبابيسُ معالم ملوّنةٌ بأسمائها بلونها.)

**وقراراتُ المالك السابقةُ باقية**: لا أبنيةَ مرفوعة (٢٠٢٦-٠٨-٢٤ «بايخة»)،
وتُعرض المعالمُ كلُّها حتّى المطاعم والمتاجر (٢٠٢٦-٠٨-٢٤).

**والعقدُ مع التطبيق ثلاثةُ مراسٍ لا تُمسّ** (`_anchors`) — وما عداها حرّ.

    الاستعمال:  python maps/scripts/make-style.py
"""
import json
from pathlib import Path

OUT = Path(__file__).resolve().parent.parent / "style" / "rahalgo.style.json"
NAME = ["coalesce", ["get", "name:ar"], ["get", "name"]]
Z = ["zoom"]


def lin(*stops):
    return ["interpolate", ["linear"], Z, *stops]


def exp(*stops):
    return ["interpolate", ["exponential", 1.6], Z, *stops]


def cls_in(*vals):
    return ["in", ["get", "class"], ["literal", list(vals)]]


def fill(id_, layer, color, flt=None, minzoom=None, opacity=None, outline=None):
    l = {"id": id_, "type": "fill", "source": "base", "source-layer": layer,
         "paint": {"fill-color": color}}
    if flt is not None:
        l["filter"] = flt
    if minzoom is not None:
        l["minzoom"] = minzoom
    if opacity is not None:
        l["paint"]["fill-opacity"] = opacity
    if outline is not None:
        l["paint"]["fill-outline-color"] = outline
    return l


def line(id_, flt, color, width, minzoom, dash=None, cap="round", layer="transportation"):
    l = {"id": id_, "type": "line", "source": "base", "source-layer": layer,
         "minzoom": minzoom, "filter": flt,
         "layout": {"line-cap": cap, "line-join": "round"},
         "paint": {"line-color": color, "line-width": width}}
    if dash:
        l["paint"]["line-dasharray"] = dash
    return l


# ── الشوارع: عرضُ اللون وعرضُ الحدّ لكلّ درجة ─────────────────────────
# **والحدُّ أعرضُ من اللون بنقطتين تقريباً** — فيبرز الشارعُ الأبيضُ على
# الأرض الرماديّة كما في غوغل، **ولا يذوب فيها.**
W_MINOR = exp(13, 0.6, 16, 4.5, 19, 16)
W_MINOR_C = exp(13, 1.4, 16, 6.3, 19, 19)
W_SERVICE = exp(15, 0.6, 17, 2.6, 19, 8)
W_SERVICE_C = exp(15, 1.4, 17, 3.8, 19, 10)
W_SEC = exp(10, 1.0, 14, 4.2, 19, 22)
W_SEC_C = exp(10, 1.8, 14, 6.0, 19, 25)
W_PRI = exp(8, 1.0, 14, 5.6, 19, 27)
W_PRI_C = exp(8, 1.8, 14, 7.6, 19, 30)
W_TRUNK = exp(6, 1.0, 14, 6.4, 19, 30)
W_TRUNK_C = exp(6, 1.8, 14, 8.6, 19, 33)

ROAD_CASING = "#D5D8DD"
ROAD_FILL = "#FFFFFF"
TRUNK_FILL = "#FCD77A"
TRUNK_CASING = "#E7B04A"

# ── المعالم: الصنفُ ⇒ الأيقونةُ ولونُ الاسم ───────────────────────────
POI_GROUPS = [
    # (الأيقونة، لونُ الاسم، الأصناف)
    ("poi-food", "#B45309", ["restaurant", "fast_food", "bakery", "ice_cream", "bar", "beer"]),
    ("poi-cafe", "#B45309", ["cafe"]),
    ("poi-shop", "#1967D2", ["shop", "grocery", "clothing_store", "alcohol_shop", "laundry",
                             "hairdresser", "jewelry", "mobile_phone", "furniture"]),
    ("poi-hospital", "#C5221F", ["hospital", "doctors", "dentist", "veterinary"]),
    ("poi-pharmacy", "#C5221F", ["pharmacy"]),
    ("poi-park", "#137333", ["park", "garden", "picnic_site", "playground", "campsite", "zoo"]),
    ("poi-sport", "#137333", ["pitch", "stadium", "sports_centre", "swimming", "golf",
                              "horse_racing"]),
    ("poi-mosque", "#5F6368", ["place_of_worship"]),
    ("poi-school", "#7A4F1E", ["school", "college", "library", "kindergarten", "university"]),
    ("poi-gov", "#4A5568", ["town_hall", "police", "fire_station", "post", "prison"]),
    ("poi-bank", "#4A5568", ["bank", "atm"]),
    ("poi-hotel", "#A50E5A", ["lodging"]),
    ("poi-transit", "#1967D2", ["bus", "railway", "aerialway"]),
    ("poi-parking", "#1967D2", ["parking"]),
    ("poi-fuel", "#1967D2", ["fuel"]),
    ("poi-landmark", "#0E7C8C", ["attraction", "castle", "monument", "museum", "art_gallery",
                                 "theatre", "cinema"]),
]
# **وما لا يُهتدى به لا يُرسم** — بوّاباتٌ وأراضٍ مهجورةٌ ومداخل.
POI_HIDDEN = ["gate", "brownfield", "entrance", "office"]
# **والكبرى تظهر أبكر** — يُعطى بها العنوانُ في الرقّة: «جنب الجامع، خلف المستشفى».
POI_MAJOR = ["hospital", "pharmacy", "fuel", "place_of_worship", "school", "college",
             "university", "town_hall", "police", "stadium", "lodging", "bank"]


def poi_match(pick):
    m = ["match", ["get", "class"]]
    for icon, color, classes in POI_GROUPS:
        m += [classes, pick(icon, color)]
    m.append(pick("poi-generic", "#5F6368"))
    return m


def poi_layer(id_, minzoom, flt, size):
    return {
        "id": id_, "type": "symbol", "source": "base", "source-layer": "poi",
        "minzoom": minzoom, "filter": flt,
        "layout": {
            "icon-image": poi_match(lambda icon, color: icon),
            "icon-size": size,
            "icon-anchor": "bottom",
            "text-field": NAME,
            "text-font": ["RahalGo Regular"],
            "text-size": lin(15, 10.5, 18, 12.5),
            # **الاسمُ بجانب الدبّوس لا تحته** — كما في غوغل، ويُبدَّل جانبُه
            # إن ضاق المكان.
            "text-variable-anchor": ["right", "left", "top"],
            "text-radial-offset": 0.9,
            "text-justify": "auto",
            "text-max-width": 8,
            "text-optional": True,
            "text-padding": 6,
            "symbol-sort-key": ["coalesce", ["get", "rank"], 50],
        },
        "paint": {
            "text-color": poi_match(lambda icon, color: color),
            "text-halo-color": "#FFFFFF",
            "text-halo-width": 1.4,
        },
    }


ROUTE_CLASSES_ONEWAY = ["minor", "service", "tertiary", "secondary", "primary", "trunk"]

layers = [
    {"id": "background", "type": "background", "paint": {"background-color": "#F2F3F5"}},
    fill("landuse-residential", "landuse", "#EEF0F3",
         cls_in("residential", "neighbourhood", "suburb", "quarter")),
    fill("landuse-commercial", "landuse", "#FCF1DC", cls_in("commercial", "retail")),
    fill("landuse-industrial", "landuse", "#EBECF0", cls_in("industrial", "railway", "military")),
    fill("landuse-hospital", "landuse", "#FCE6E4", cls_in("hospital")),
    fill("landuse-school", "landuse", "#F4EFE4", cls_in("school", "university", "college",
                                                       "kindergarten")),
    fill("landcover", "landcover", ["match", ["get", "class"],
                                    "wood", "#CDE8C6", "grass", "#D9EED2",
                                    "farmland", "#EDF2E2", "sand", "#F5EFDF",
                                    "wetland", "#D5ECE8", "#E4EEDB"], opacity=0.7),
    fill("landuse-green", "landuse", "#CDEACB",
         cls_in("pitch", "grass", "cemetery", "playground", "garden", "stadium")),
    fill("park", "park", "#C6E7C1", opacity=0.9),
    fill("water", "water", "#A8CDF5", ["!=", ["get", "brunnel"], "tunnel"]),
    line("waterway", ["!=", ["get", "brunnel"], "tunnel"], "#A8CDF5",
         lin(9, 0.6, 16, 3), 9, layer="waterway"),
    # **مسطّحةٌ لا مرفوعة** — قرارُ المالك ٢٠٢٦-٠٨-٢٤.
    fill("building", "building", "#E6E7EB", minzoom=15, outline="#D8DADF"),

    # ── الحدود: كلُّها قبل الألوان — وإلّا قطع حدُّ الفرعيّ الرئيسيَّ عند التقاطع ──
    line("road-casing-service", cls_in("service", "track"), ROAD_CASING, W_SERVICE_C, 15),
    line("road-casing-minor", cls_in("minor"), ROAD_CASING, W_MINOR_C, 13),
    line("road-casing-secondary", cls_in("secondary", "tertiary"), ROAD_CASING, W_SEC_C, 10),
    line("road-casing-primary", cls_in("primary"), ROAD_CASING, W_PRI_C, 8),
    line("road-casing-trunk", cls_in("trunk", "motorway"), TRUNK_CASING, W_TRUNK_C, 6),

    line("road-service", cls_in("service", "track"), ROAD_FILL, W_SERVICE, 15),
    line("road-minor", cls_in("minor"), ROAD_FILL, W_MINOR, 12),
    line("road-secondary", cls_in("secondary", "tertiary"), ROAD_FILL, W_SEC, 9),
    line("road-primary", cls_in("primary"), ROAD_FILL, W_PRI, 7),
    line("road-trunk", cls_in("trunk", "motorway"), TRUNK_FILL, W_TRUNK, 5),
    line("road-path", cls_in("path", "pedestrian", "footway", "steps"), "#C3C7CD",
         lin(15, 0.8, 19, 2.6), 15, dash=[2, 1.6], cap="butt"),
    line("rail", cls_in("rail", "transit"), "#BABEC4", lin(12, 0.6, 19, 2.4), 12,
         dash=[3, 2], cap="butt"),

    {"id": "rahalgo-route-anchor", "type": "background",
     "paint": {"background-color": "#000000", "background-opacity": 0}},

    # **وسهمُ الاتّجاه الواحد** — قِيس: ٣٢١ مقطعاً من ٧٣٩ في الرقّة باتّجاهٍ واحد.
    {"id": "road-oneway", "type": "symbol", "source": "base", "source-layer": "transportation",
     "minzoom": 16,
     "filter": ["all", ["==", ["get", "oneway"], 1],
                ["in", ["get", "class"], ["literal", ROUTE_CLASSES_ONEWAY]]],
     "layout": {"symbol-placement": "line", "symbol-spacing": 110, "icon-image": "oneway",
                "icon-size": 0.75, "icon-rotation-alignment": "map",
                "icon-allow-overlap": True, "icon-ignore-placement": True}},

    {"id": "road-label", "type": "symbol", "source": "base",
     "source-layer": "transportation_name", "minzoom": 13,
     "layout": {
         "symbol-placement": "line", "text-field": NAME,
         # **مصفوفةُ أسماءٍ لا تعبير** — قارئُ النمط في التطبيق (`fontstacksOf`)
         # يقرأ أسماءً، والتعبيرُ أسقط الربطَ كلَّه: «بياناتُ الخريطة غيرُ متاحة».
         "text-font": ["RahalGo Regular"],
         "text-size": lin(13, 10, 16, 12, 19, 15),
         "text-rotation-alignment": "map", "symbol-spacing": 420, "text-padding": 12},
     "paint": {"text-color": ["case", cls_in("primary", "trunk", "motorway"), "#3C4043",
                              "#5F6368"],
               "text-halo-color": "#FFFFFF", "text-halo-width": 1.6}},

    # **وأرقامُ البنايات** — قليلةٌ في سوريا اليوم، **وتظهر حيث وُجدت.**
    {"id": "housenumber", "type": "symbol", "source": "base", "source-layer": "housenumber",
     "minzoom": 17.5,
     "layout": {"text-field": ["get", "housenumber"], "text-font": ["RahalGo Regular"],
                "text-size": 10, "text-padding": 2},
     "paint": {"text-color": "#9AA0A6", "text-halo-color": "#F2F3F5", "text-halo-width": 1}},

    {"id": "water-label", "type": "symbol", "source": "base", "source-layer": "water_name",
     "minzoom": 9,
     "layout": {"symbol-placement": "line", "text-field": NAME, "text-font": ["RahalGo Regular"],
                "text-size": lin(9, 10, 15, 14), "text-rotation-alignment": "map"},
     "paint": {"text-color": "#3B6FB6", "text-halo-color": "#EAF2FC", "text-halo-width": 1.2}},

    poi_layer("poi-label", 15,
              ["all", cls_in(*POI_MAJOR)], 0.95),
    poi_layer("poi-other-label", 16,
              ["all", ["!", cls_in(*POI_MAJOR)], ["!", cls_in(*POI_HIDDEN)]], 0.85),

    {"id": "place-village", "type": "symbol", "source": "base", "source-layer": "place",
     "minzoom": 11, "filter": cls_in("village", "suburb", "neighbourhood", "quarter", "hamlet"),
     "layout": {"text-field": NAME, "text-font": ["RahalGo Regular"],
                "text-size": lin(11, 11, 16, 14), "text-max-width": 9, "text-padding": 4},
     "paint": {"text-color": "#5F6368", "text-halo-color": "#FFFFFF", "text-halo-width": 1.6}},
    {"id": "place-city", "type": "symbol", "source": "base", "source-layer": "place",
     "minzoom": 4, "filter": cls_in("city", "town"),
     "layout": {"text-field": NAME, "text-font": ["RahalGo Bold"],
                "text-size": lin(4, 11, 12, 18), "text-max-width": 9},
     "paint": {"text-color": "#202124", "text-halo-color": "#FFFFFF", "text-halo-width": 1.8}},

    {"id": "rahalgo-marker-anchor", "type": "background",
     "paint": {"background-color": "#000000", "background-opacity": 0}},
]

style = {
    "_note": "النمطُ المرجعيُّ الواحد — لا يُحرَّر باليد: يُولَّد من maps/scripts/make-style.py. "
             "تُبدَّل فيه القوالبُ الثلاثة ({{TILES}} · {{GLYPHS}} · {{SPRITE}}) فيصير نمطاً أونلاين "
             "أو دونَ اتّصال. انظر maps/scripts/bind-style.mjs.",
    "_contract": "معرّفاتُ المراسي الثلاثة ثابتة — يعتمد عليها الأندرويدُ في إدراج خطّ المسار والعلامات. "
                 "انظر _anchors. وما عداها حرّ.",
    "_google": "قرارُ المالك ٢٠٢٦-١٠-٠١: «الخرائط مثل غوغل ماب لأبعد حدّ». ألوانُ غوغل وترتيبُه: "
               "أرضٌ رماديّةٌ فاتحة، شوارعُ بيضاءُ بحدٍّ رماديّ، طرقٌ سريعةٌ صفراء، دبابيسُ معالم ملوّنة.",
    "_buildings": "مسطّحةٌ لا مرفوعة — قرارُ المالك ٢٠٢٦-٠٨-٢٤ نصّاً: «أشكال الأبنية… بايخة». "
                  "ولا تُرفع إلّا بطلبه.",
    "_allPoi": "تُعرض المعالمُ كلُّها حتّى المطاعم والمتاجر — قرارُ المالك ٢٠٢٦-٠٨-٢٤. "
               "وما لا يُهتدى به (بوّاباتٌ، أراضٍ مهجورة، مداخل، مكاتبُ شركات) لا يُرسم.",
    "_anchors": {"belowLabels": "place-city", "route": "rahalgo-route-anchor",
                 "markers": "rahalgo-marker-anchor"},
    "version": 8,
    "name": "RahalGo",
    "metadata": {"rahalgo:styleVersion": "3", "rahalgo:schema": "openmaptiles-3.16.0"},
    "sources": {"base": {"type": "vector", "url": "{{TILES}}",
                         "attribution": "© <a href=\"https://www.openstreetmap.org/copyright\">"
                                        "مساهمو OpenStreetMap</a> · "
                                        "<a href=\"https://openmaptiles.org/\">OpenMapTiles</a>"}},
    "glyphs": "{{GLYPHS}}",
    "sprite": "{{SPRITE}}",
    "layers": layers,
}

if __name__ == "__main__":
    OUT.write_text(json.dumps(style, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(len(layers), "طبقة ⇒", OUT)
