# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
**مولّدُ الصوت — يُشغَّل مرّةً ويُشحن ناتجُه في الحزمة**
══════════════════════════════════════════════════════════════════════

(قرارُ المالك ٢٠٢٦-٠٨-٢٣: «أريد صوتاً عربيّاً حقيقيّاً، لا صوتي ولا
 صوتَ روبوت».)

# ولماذا يُولَّد ولا يُنادى وقتَ النطق

**السائقُ في حيٍّ ضعيفِ التغطية** — **وجملةٌ تُجلب من الشبكة تصله بعد
المنعطف**، والتعليمةُ المتأخّرةُ أسوأُ من لا تعليمة.

**ويُولَّد مرّةً**: خمسون مقطعاً ≈ ١٥٠٠ حرف، **ومنحةُ Azure المجّانيّة
٥٠٠ ألف حرفٍ شهريّاً** — فالكلفةُ صفر.

# ولماذا صوتٌ سوريّ

`ar-SY-LaithNeural` — **لهجةٌ يعرفها السائقُ**، لا فصحى خليجيّةً
غريبةً على الأذن.

# والصمتُ يُقصّ من الطرفين

**المقاطعُ تُوصَل عند النطق** («بعد مئتي متر» + «انعطف يميناً») —
**وصمتٌ في طرف كلّ مقطعٍ يجعل الجملةَ متقطّعةً كأنّها آلة**، وهو
بعينه ما نهرب منه.

# التشغيل

    set AZURE_SPEECH_KEY=...
    set AZURE_SPEECH_REGION=westeurope
    python build-voice.py

والناتجُ في `../src/main/assets/voice/` بصيغة Ogg/Opus.
"""
import json
import os
import pathlib
import sys
import urllib.request

sys.stdout.reconfigure(encoding="utf-8")

HERE = pathlib.Path(__file__).parent
OUT = HERE.parent / "src" / "main" / "assets" / "voice"

KEY = os.environ.get("AZURE_SPEECH_KEY", "")
REGION = os.environ.get("AZURE_SPEECH_REGION", "westeurope")
VOICE = os.environ.get("RAHALGO_VOICE", "ar-SY-LaithNeural")

# **وصيغةٌ واحدةٌ للجميع** — **واختلافُ الترميز بين مقطعين يُسمع
# فرقاً في النبرة** حين يوصلان.
FORMAT = "ogg-48khz-16bit-mono-opus"


def ssml(text: str) -> str:
    """**ونبرةٌ هادئةٌ وسرعةٌ أبطأُ قليلاً** — التعليمةُ تُقال مرّةً وهو يقود."""
    return (
        '<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" '
        'xml:lang="ar-SY">'
        f'<voice name="{VOICE}">'
        '<prosody rate="-6%">' + text + "</prosody>"
        "</voice></speak>"
    )


def synth(text: str) -> bytes:
    req = urllib.request.Request(
        f"https://{REGION}.tts.speech.microsoft.com/cognitiveservices/v1",
        data=ssml(text).encode("utf-8"),
        headers={
            "Ocp-Apim-Subscription-Key": KEY,
            "Content-Type": "application/ssml+xml",
            "X-Microsoft-OutputFormat": FORMAT,
            "User-Agent": "rahalgo-voice",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as r:
        return r.read()


def main() -> int:
    if not KEY:
        print("لا مفتاح — اضبط AZURE_SPEECH_KEY")
        return 1
    clips = json.loads((HERE / "clips.json").read_text(encoding="utf-8"))
    # **والمفاتيحُ التي تبدأ بشرطةٍ سفليّةٍ تعليقٌ لا مقطع.**
    items = {k: v for k, v in clips.items() if not k.startswith("_")}
    OUT.mkdir(parents=True, exist_ok=True)
    total = 0
    for name, text in items.items():
        data = synth(text)
        (OUT / f"{name}.ogg").write_bytes(data)
        total += len(text)
        print(f"{name:22s}  {len(data):7d} بايت   {text}")
    print(f"\nتمّ: {len(items)} مقطعاً · {total} حرفاً · الصوت {VOICE}")
    print(f"المجلّد: {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
