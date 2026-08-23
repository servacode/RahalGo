# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
**مولّدُ الصوت بـElevenLabs**
══════════════════════════════════════════════════════════════════════

(قرارُ المالك ٢٠٢٦-٠٨-٢٣: «ElevenLabs أقوى جدّاً من هذا المحرّك الذي
 اخترتَه أنت».)

**وهو محقّ.** اخترتُ Piper لأنّه لا يحتاج حساباً — **وذاك تفضيلُ
الراحة على الجودة، وهو خطأٌ حين تكون الجودةُ هي السؤال.**

# ولماذا يُولَّد مرّةً ويُشحن

**السائقُ في حيٍّ ضعيفِ التغطية** — **وجملةٌ تُجلب من الشبكة تصله بعد
المنعطف**، والتعليمةُ المتأخّرةُ أسوأُ من لا تعليمة.

**واثنتان وأربعون عبارةً ≈ ١٣٠٠ حرف** — ومنحةُ الطبقة المجّانيّة
عشرةُ آلافٍ شهريّاً. **فالكلفةُ صفرٌ والتوليدُ يُعاد كما نشاء.**

# والصمتُ يُقصّ من الطرفين

**المقاطعُ تُوصَل عند النطق** («بعد مئتي متر» + «انعطف يميناً») —
**وصمتٌ في طرف كلّ مقطعٍ يجعل الجملةَ متقطّعةً كأنّها آلة.**

# والتشكيلُ يُجرَّب ولا يُفترض

**نفع التشكيلُ مع `espeak`** لأنّه يفَنِم الحرفَ حرفاً. **ونماذجُ
ElevenLabs العصبيّةُ مدرَّبةٌ على نصٍّ طبيعيٍّ غالبُه بلا تشكيل** —
فقد يعينها وقد يشوّشها. **فيُقاس على عبارةٍ واحدةٍ قبل الأربعين.**

# التشغيل

    set ELEVENLABS_API_KEY=...
    python build-voice-eleven.py --list          # لعرض الأصوات
    python build-voice-eleven.py --voice <id>    # للتوليد
    python build-voice-eleven.py --voice <id> --only d_200,act_right
"""
import argparse
import json
import os
import pathlib
import sys
import urllib.error
import urllib.request

sys.stdout.reconfigure(encoding="utf-8")

HERE = pathlib.Path(__file__).parent
OUT = HERE.parent / "src" / "main" / "assets" / "voice"

KEY = os.environ.get("ELEVENLABS_API_KEY", "")
BASE = "https://api.elevenlabs.io/v1"

# **والنموذجُ متعدّدُ اللغات** — وهو الذي يعرف العربيّة.
MODEL = os.environ.get("RAHALGO_EL_MODEL", "eleven_multilingual_v2")

# **وصيغةٌ واحدةٌ للجميع** — **واختلافُ الترميز بين مقطعين يُسمع
# فرقاً في النبرة** حين يوصلان.
FORMAT = "mp3_44100_128"


def req(path: str, data=None, method="GET"):
    r = urllib.request.Request(
        BASE + path,
        data=json.dumps(data).encode("utf-8") if data is not None else None,
        headers={
            "xi-api-key": KEY,
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        method=method,
    )
    with urllib.request.urlopen(r, timeout=90) as resp:
        return resp.read()


def voices():
    """**أصواتُ الحساب** — ومنها يُختار."""
    d = json.loads(req("/voices"))
    for v in d.get("voices", []):
        labels = v.get("labels") or {}
        tags = " · ".join(f"{k}={x}" for k, x in labels.items())
        print(f"{v['voice_id']:24s} {v.get('name',''):20s} {tags}")


def synth(voice_id: str, text: str) -> bytes:
    r = urllib.request.Request(
        f"{BASE}/text-to-speech/{voice_id}?output_format={FORMAT}",
        data=json.dumps(
            {
                "text": text,
                "model_id": MODEL,
                # **وثباتٌ عالٍ لتعليمةِ ملاحة** — **والصوتُ المتقلّب
                # يُقرأ ممثّلاً لا مرشدا.**
                "voice_settings": {
                    "stability": 0.75,
                    "similarity_boost": 0.75,
                    "style": 0.0,
                    "use_speaker_boost": True,
                },
            }
        ).encode("utf-8"),
        headers={
            "xi-api-key": KEY,
            "Content-Type": "application/json",
            "Accept": "audio/mpeg",
        },
        method="POST",
    )
    with urllib.request.urlopen(r, timeout=120) as resp:
        return resp.read()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--list", action="store_true")
    ap.add_argument("--voice")
    ap.add_argument("--only", default="", help="مقاطعُ بعينها، مفصولةً بفاصلة")
    args = ap.parse_args()

    if not KEY:
        print("لا مفتاح — اضبط ELEVENLABS_API_KEY")
        return 1
    if args.list:
        voices()
        return 0
    if not args.voice:
        print("اختر صوتاً: --voice <id>   (أو --list لعرضها)")
        return 1

    clips = json.loads((HERE / "clips.json").read_text(encoding="utf-8"))
    items = {k: v for k, v in clips.items() if not k.startswith("_")}
    if args.only:
        want = {x.strip() for x in args.only.split(",") if x.strip()}
        items = {k: v for k, v in items.items() if k in want}
        if not items:
            print("لا مقطعَ بهذه الأسماء")
            return 1

    OUT.mkdir(parents=True, exist_ok=True)
    total = 0
    for name, text in items.items():
        try:
            data = synth(args.voice, text)
        except urllib.error.HTTPError as e:
            # **والخطأُ يُقال ولا يُبتلع** — **ومقطعٌ يسقط صامتاً يجعل
            # السائقَ يسمع محرّكاً آليّاً في منعطفٍ وبشراً في الذي
            # يليه**، والتبدّلُ يُقرأ عطبا.
            print(f"{name:22s} سقط: {e.code} {e.read()[:200]!r}")
            return 1
        (OUT / f"{name}.mp3").write_bytes(data)
        total += len(text)
        print(f"{name:22s} {len(data):8d} بايت   {text}")

    print(f"\nتمّ: {len(items)} مقطعاً · {total} حرفاً · النموذج {MODEL}")
    print(f"المجلّد: {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
