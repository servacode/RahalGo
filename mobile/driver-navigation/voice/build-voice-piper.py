# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
**مولّدُ الصوت بـPiper — بلا حسابٍ ولا بطاقةٍ ولا حجب**
══════════════════════════════════════════════════════════════════════

(قرارُ المالك ٢٠٢٦-٠٨-٢٣: «أريد صوتاً عربيّاً حقيقيّاً، لا صوتي ولا
 صوتَ روبوت» — ثمّ: «نعم جرّب» بعد أن قيل له إنّ أزور قد تُحجب.)

# ولماذا بديلٌ لأزور

**أزور تطلب بطاقةً ائتمانيّةً للتحقّق حتّى في طبقتها المجّانيّة**،
**وسوريا ضمن دولها المحجوبة** — فقد يُرفض الحسابُ أو يُقفل بعد شهر.
**ومحرّكٌ يُقفل حسابُه يوقف صوتَ التطبيق كلَّه.**

**وهذا يعمل على الحاسوب بلا شبكةٍ ولا حساب** — ونموذجُه ملفٌّ نملكه.

# والصوتُ `ar_JO-kareem`

**أردنيٌّ — وهو شاميٌّ قريبٌ من لهجة الرقّة**، أقربُ بكثيرٍ من
الفصحى الخليجيّة أو المصريّة التي تعطيها أكثرُ المحرّكات.

**وهو عصبيٌّ (VITS) لا تراكميّ** — وهو الفرقُ الذي شكا منه المالك:
**«نطقٌ آليٌّ أو روبوت».**

# والصمتُ يُقصّ من الطرفين

**المقاطعُ تُوصَل عند النطق** («بعد مئتي متر» + «انعطف يميناً») —
**وصمتٌ في طرف كلّ مقطعٍ يجعل الجملةَ متقطّعةً كأنّها آلة**، وهو
بعينه ما نهرب منه.

# التشغيل

    pip install piper-tts
    python build-voice-piper.py --model <مسار>/ar_JO-kareem-medium.onnx

والناتجُ في `../src/main/assets/voice/` بصيغة WAV ثمّ يُحوَّل.
"""
import argparse
import json
import pathlib
import sys
import wave

sys.stdout.reconfigure(encoding="utf-8")

HERE = pathlib.Path(__file__).parent
OUT = HERE.parent / "src" / "main" / "assets" / "voice"

# **وعتبةُ الصمت** — والقيمةُ نسبةٌ من أعلى سعةٍ في المقطع، لا رقمٌ
# مطلق: **مقطعٌ هادئٌ كلُّه يُقصّ كلُّه بعتبةٍ ثابتة.**
SILENCE_RATIO = 0.02
# **ويُترك هامشٌ يسير** — **وقصٌّ إلى الحرف يبتر أوّلَ الصوت**، فتُقرأ
# «انعطف» كأنّها «نعطف».
KEEP_MS = 25


def trim(samples: bytes, rate: int, width: int) -> bytes:
    """**يقصّ الصمتَ من الطرفين** — ويترك هامشاً."""
    import array

    if width != 2:
        return samples
    a = array.array("h")
    a.frombytes(samples)
    if not a:
        return samples
    peak = max(abs(x) for x in a) or 1
    thr = peak * SILENCE_RATIO
    start, end = 0, len(a) - 1
    while start < end and abs(a[start]) < thr:
        start += 1
    while end > start and abs(a[end]) < thr:
        end -= 1
    pad = int(rate * KEEP_MS / 1000)
    start = max(0, start - pad)
    end = min(len(a) - 1, end + pad)
    return a[start : end + 1].tobytes()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", required=True)
    ap.add_argument("--rate", type=float, default=0.95,
                    help="سرعةُ النطق — أبطأُ قليلاً لأنّه يقود")
    args = ap.parse_args()

    from piper import PiperVoice

    voice = PiperVoice.load(args.model)
    clips = json.loads((HERE / "clips.json").read_text(encoding="utf-8"))
    items = {k: v for k, v in clips.items() if not k.startswith("_")}
    OUT.mkdir(parents=True, exist_ok=True)

    for name, text in items.items():
        path = OUT / f"{name}.wav"
        with wave.open(str(path), "wb") as w:
            voice.synthesize_wav(text, w)
        # **ثمّ يُقصّ الصمت** — يُعاد فتحُ الملفّ لأنّ المحرّك يكتبه كاملاً.
        with wave.open(str(path), "rb") as r:
            ch, width, rate = r.getnchannels(), r.getsampwidth(), r.getframerate()
            data = r.readframes(r.getnframes())
        data = trim(data, rate, width)
        with wave.open(str(path), "wb") as w:
            w.setnchannels(ch)
            w.setsampwidth(width)
            w.setframerate(rate)
            w.writeframes(data)
        print(f"{name:22s}  {len(data):8d} بايت   {text}")

    print(f"\nتمّ: {len(items)} مقطعاً · {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
