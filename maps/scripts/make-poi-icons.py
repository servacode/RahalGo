"""أيقوناتُ المعالم بشكل غوغل — دبّوسٌ ملوّنٌ وفي وسطه رمزٌ أبيض.

(طلبُ المالك ٢٠٢٦-١٠-٠١: «الخرائط مثل غوغل ماب لأبعد حدّ» — وصورتُه المرجع:
 مستشفى H أحمر، حديقة شجرة خضراء، مطعم شوكة برتقاليّة.)

**والألوانُ لوحةُ غوغل** — كلُّ صنفٍ لونُه، **ويُكتب اسمُ المعلم بلونه** في
النمط. **والرموزُ أشكالٌ هندسيّةٌ بسيطة** — تُقرأ في لمحةٍ بحجم ٢٢ بكسلاً.

    الاستعمال:  python maps/scripts/make-poi-icons.py   ← يكتب maps/sprite/*.svg
"""
from pathlib import Path

OUT = Path(__file__).resolve().parent.parent / "sprite"

PIN = ("M11 27.2c-.4 0-.8-.2-1-.5C8.4 24.6 2 17.6 2 11a9 9 0 0 1 18 0"
       "c0 6.6-6.4 13.6-8 15.7-.2.3-.6.5-1 .5z")

# الرمزُ في مربّع 24×24 حول (12،12) — ويُزاح نقطةً ليتوسّط رأسَ الدبّوس.
GLYPHS = {
    "food": [  # شوكةٌ وسكّين
        "M8.6 6.8v4.1c0 .7.4 1.2 1 1.4V17h1.3v-4.7c.6-.2 1-.7 1-1.4V6.8h-.9v3.5h-.6V6.8h-.8v3.5H9V6.8z",
        "M15.6 6.8c-1.1.7-1.8 2.4-1.8 4.5v1.6h1V17h1.3V6.8z",
    ],
    "cafe": [  # فنجان
        "M7.5 9.5h8v3.6a4 4 0 0 1-8 0z",
        "M15.5 10.3h.8a1.8 1.8 0 0 1 0 3.6h-.9v-1.2h.9a.6.6 0 0 0 0-1.2h-.8z",
        "M7 17.2h9v1H7z",
    ],
    "shop": [  # كيسُ تسوّق
        "M7.5 10h9l-.7 7.5H8.2z",
        "M9.8 10V9a2.2 2.2 0 0 1 4.4 0v1h-1.1V9a1.1 1.1 0 0 0-2.2 0v1z",
    ],
    "hospital": [  # H
        "M8.3 7.3h2.1v3.6h3.2V7.3h2.1v9.4h-2.1V13h-3.2v3.7H8.3z",
    ],
    "pharmacy": [  # صليبٌ طبّيّ
        "M10.7 7.2h2.6v3.5h3.5v2.6h-3.5v3.5h-2.6v-3.5H7.2v-2.6h3.5z",
    ],
    "park": [  # شجرة
        "M12 6.2l4.3 6.1h-2.4l2.9 3.9H7.2l2.9-3.9H7.7z",
        "M11.3 16.1h1.4v1.9h-1.4z",
    ],
    "mosque": [  # هلال
        "M13.6 6.9a5.2 5.2 0 1 0 0 10.2a4.3 4.3 0 1 1 0-10.2z",
    ],
    "school": [  # قبّعةُ تخرّج
        "M12 7.3l6.3 3.1L12 13.5l-6.3-3.1z",
        "M8.6 12.3v2.4c0 1 1.5 1.8 3.4 1.8s3.4-.8 3.4-1.8v-2.4L12 14z",
    ],
    "gov": [  # درع
        "M12 6.6l4.6 1.8v3.6c0 2.9-1.9 4.9-4.6 5.8-2.7-.9-4.6-2.9-4.6-5.8V8.4z",
    ],
    "bank": [  # أعمدة
        "M12 6.8l5.4 2.7H6.6z",
        "M7.6 10.4h1.5v4.8H7.6zM11.25 10.4h1.5v4.8h-1.5zM14.9 10.4h1.5v4.8h-1.5z",
        "M6.8 15.8h10.4v1.4H6.8z",
    ],
    "hotel": [  # سرير
        "M6.8 9.2h1.5v3.4h9v4.2h-1.4v-1.4H8.3v1.4H6.8z",
        "M9 10.9h2.6v1.5H9zM12.2 10.6h3.9a1.2 1.2 0 0 1 1.2 1.2v.8h-5.1z",
    ],
    "transit": [  # حافلة — وزجاجُها بلون الدبّوس
        "M8.3 6.9h7.4c.7 0 1.2.5 1.2 1.2v7.4c0 .5-.3.9-.7 1.1v1.2h-1.4v-1H9.2v1H7.8v-1.2c-.4-.2-.7-.6-.7-1.1V8.1c0-.7.5-1.2 1.2-1.2z",
        ("M8.4 8.4h7.2v3.4H8.4z", "PIN"),
        ("M8.6 13.6h1.4v1.2H8.6zM14 13.6h1.4v1.2H14z", "PIN"),
    ],
    "parking": [  # P
        "M8.8 7h3.9a3.1 3.1 0 0 1 0 6.2h-1.7V17H8.8zM11 9v2.2h1.6a1.1 1.1 0 0 0 0-2.2z",
    ],
    "fuel": [  # مضخّة — ونافذتُها بلون الدبّوس
        "M7.6 7.2h6.2v10.3H7.6z",
        ("M8.6 8.4h4.2v2.8H8.6z", "PIN"),
        "M14.6 9.3l1.6 1.4v4.6a.6.6 0 0 0 1.2 0V9.9l-1.8-1.8-.8.8.9.9z",
    ],
    "landmark": [  # نجمة
        "M12 6.4l1.7 3.7 4 .4-3 2.7.9 4-3.6-2.1-3.6 2.1.9-4-3-2.7 4-.4z",
    ],
    "sport": [  # كرة
        "M12 6.8a5.2 5.2 0 1 0 0 10.4a5.2 5.2 0 0 0 0-10.4zm0 1.3l1.3 1-.5 1.6h-1.6l-.5-1.6zM8.6 10l1.4.4.6 1.6-.9 1.3-1.3-.1a3.9 3.9 0 0 1 .2-3.2zm6.8 0a3.9 3.9 0 0 1 .2 3.2l-1.3.1-.9-1.3.6-1.6z",
    ],
    "generic": [  # نقطة
        "M12 9.3a2.7 2.7 0 1 1 0 5.4a2.7 2.7 0 0 1 0-5.4z",
    ],
}

# الصنفُ ⇒ (الرمز، اللون). **وألوانُ غوغل** نصّاً.
ICONS = {
    "poi-food": ("food", "#E8710A"),
    "poi-cafe": ("cafe", "#E8710A"),
    "poi-shop": ("shop", "#1A73E8"),
    "poi-hospital": ("hospital", "#D93025"),
    "poi-pharmacy": ("pharmacy", "#D93025"),
    "poi-park": ("park", "#188038"),
    "poi-sport": ("sport", "#188038"),
    "poi-mosque": ("mosque", "#5F6368"),
    "poi-school": ("school", "#9C6B30"),
    "poi-gov": ("gov", "#5E6C84"),
    "poi-bank": ("bank", "#5E6C84"),
    "poi-hotel": ("hotel", "#C2185B"),
    "poi-transit": ("transit", "#1A73E8"),
    "poi-parking": ("parking", "#1A73E8"),
    "poi-fuel": ("fuel", "#1A73E8"),
    "poi-landmark": ("landmark", "#12A4B8"),
    "poi-generic": ("generic", "#80868B"),
}


def pin(color: str, glyph: list[str]) -> str:
    def one(g):
        d, fill = (g, "#FFFFFF") if isinstance(g, str) else (g[0], color if g[1] == "PIN" else g[1])
        return f'<path d="{d}" fill="{fill}"/>'
    paths = "".join(one(g) for g in glyph)
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" width="22" height="28" viewBox="0 0 22 28">'
        f'<path d="{PIN}" fill="{color}" stroke="#FFFFFF" stroke-width="1.3"/>'
        f'<g transform="translate(-1 -1)">{paths}</g>'
        "</svg>\n"
    )


# **سهمُ الاتّجاه الواحد** — رماديٌّ هادئٌ على الشارع، يتّجه مع رسمِ الخطّ.
ONEWAY = (
    '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="8" viewBox="0 0 14 8">'
    '<path d="M0 3.2h9.5V0L14 4l-4.5 4V4.8H0z" fill="#9AA0A6"/></svg>\n'
)

if __name__ == "__main__":
    OUT.mkdir(exist_ok=True)
    for name, (g, color) in ICONS.items():
        (OUT / f"{name}.svg").write_text(pin(color, GLYPHS[g]), encoding="utf-8")
    (OUT / "oneway.svg").write_text(ONEWAY, encoding="utf-8")
    print(len(ICONS) + 1, "أيقونة ⇒", OUT)
