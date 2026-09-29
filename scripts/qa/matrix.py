# -*- coding: utf-8 -*-
"""**مصفوفةُ القبول المتكاملة — تُشغَّل لا تُقرأ.**

# لماذا تُشغَّل

**مصفوفةٌ في وثيقةٍ تُقرأ مرّةً ثمّ تُصدَّق أبداً.** ومن كتب «✔» بيده
**كتب رأيَه لا قياسَه** — **و«يعمل» تعني عندنا اختباراً رُئي ساقطاً قبل
الإصلاح وناجحاً بعده، أو نداءً حيّاً يُظهر النتيجة** (`CLAUDE.md`، القاعدة
الرابعة).

**فكلُّ صفٍّ هنا يحمل `verify` — كيف يُتحقَّق منه**، وما أمكن تشغيلُه
يُشغَّل:

	contract  · سؤالٌ لملفٍّ مولَّدٍ (`jq`-مثلُ) — يُشغَّل الآن
	gotest    · اختبارُ Go بعينه — يُشغَّل الآن
	script    · سكربتٌ في المستودع — يُشغَّل الآن
	api       · نداءٌ حيٌّ على التجهيز — يُشغَّل الآن
	device    · شاهدُ جهازٍ بـADB — **لا يُشغَّل آليّاً**
	owner     · قبولٌ بصريٌّ من المالك — **لا يُشغَّل آليّاً**

**وما لا يُشغَّل يبقى `PENDING` صريحاً** — **ولا يُعَدُّ ناجحاً لأنّه لم
يُفحص.** وذلك الفرقُ بين مصفوفةٍ صادقةٍ ومصفوفةٍ تُجمّل.

# الأنماط (PROFILES)

**ولا تُشغَّل المصفوفةُ كلُّها في كلّ مرّة** — لأنّ ما يُشغَّل كلَّه لا
يُشغَّل أبداً:

	python scripts/qa/matrix.py --list
	python scripts/qa/matrix.py --profile SMOKE
	python scripts/qa/matrix.py --profile MONEY --run
	python scripts/qa/matrix.py --render        # يكتب الوثيقة من الجدول
"""
import argparse
import datetime
import io
import json
import os
import re
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
import target  # noqa: E402

DATA = os.path.join(ROOT, "docs", "testing", "system", "ACCEPTANCE_MATRIX.json")
DOC = os.path.join(ROOT, "docs", "INTEGRATED-ACCEPTANCE-MATRIX.md")
BACKEND = os.path.join(ROOT, "backend")

AUTO = {"contract", "gotest", "script", "api"}
MANUAL = {"device", "owner"}

STATUS_ORDER = ["FAIL", "BLOCKED", "PENDING", "KNOWN-GAP", "PASS", "N/A"]


def load():
    return json.load(io.open(DATA, encoding="utf-8"))


# ══════════════════════════════════════════════════════════════════════
#  المنفّذون
# ══════════════════════════════════════════════════════════════════════

def _resolve(path):
    """**مسارٌ نسبيٌّ إلى جذر المستودع** — لا إلى مجلّد التشغيل."""
    return os.path.join(ROOT, path.replace("/", os.sep))


def run_contract(spec):
    """**يسأل ملفّاً مولَّداً** — `{"file":…, "query":…, "expect":…}`.

    **و`query` مسارٌ بسيطٌ لا لغةٌ كاملة**: `a.b[0].c`، ومعه `count(...)`
    و`any(...)`. **فلغةٌ كاملةٌ في مصفوفةٍ تصير برنامجاً ثانياً يحتاج
    اختباراتِه.**
    """
    f = _resolve(spec["file"])
    if not os.path.exists(f):
        return False, "الملفُّ غائب: %s" % spec["file"]
    doc = json.load(io.open(f, encoding="utf-8"))
    try:
        got = _query(doc, spec["query"])
    except Exception as exc:
        return False, "الاستعلامُ أخطأ: %s" % exc
    want = spec["expect"]
    return (got == want), "got=%r want=%r" % (got, want)


_IDX = re.compile(r"^(.*?)\[(\d+)\]$")


def _query(doc, q):
    """`count(path)` · `any(path==value)` · `path.to.field`"""
    m = re.match(r"^count\((.*)\)$", q)
    if m:
        v = _walk(doc, m.group(1))
        return len(v) if v is not None else 0
    m = re.match(r"^countwhere\((.*?),\s*(.*?)\s*==\s*(.*)\)$", q)
    if m:
        rows = _walk(doc, m.group(1)) or []
        key, val = m.group(2), json.loads(m.group(3))
        return sum(1 for r in rows if isinstance(r, dict) and r.get(key) == val)
    return _walk(doc, q)


def _walk(doc, path):
    cur = doc
    for part in path.split("."):
        if not part:
            continue
        m = _IDX.match(part)
        idx = None
        if m:
            part, idx = m.group(1), int(m.group(2))
        if isinstance(cur, dict):
            cur = cur.get(part)
        else:
            return None
        if idx is not None:
            if not isinstance(cur, list) or idx >= len(cur):
                return None
            cur = cur[idx]
    return cur


def run_gotest(spec):
    """**يشغّل اختبارَ Go بعينه — ويطلب دليلَ أنّه جرى.**

    # وهذا موضعُ كذبٍ وقعتُ فيه

    **`go test -run ^اسمٌ لا وجودَ له$` يردّ صفراً** ويطبع
    `no tests to run` — **فقرأه الحارسُ نجاحاً.** ووقع فعلاً: أربعةُ صفوفٍ
    كتبتُ فيها حزمةً خاطئةً **فخرجت خضراءَ ولم يجرِ منها اختبارٌ واحد.**

    **وكذلك `t.Skip`**: اختبارٌ يحتاج قاعدةً ولا يجدها يُخطّي ويردّ صفراً —
    **فيُقرأ «نجح» وهو لم يُقَس.**

    **فلا يُقبل النجاحُ إلّا بسطرِ `--- PASS:` باسم الاختبار.**
    والمُخطَّى يُقال `BLOCKED` لا `PASS`، **والمفقودُ خطأُ إعدادٍ يُقال.**
    """
    pkg, name = spec["package"], spec.get("test", "")
    cmd = ["go", "test", pkg, "-count=1", "-v"]
    if name:
        cmd += ["-run", "^%s$" % name]
    r = subprocess.run(cmd, cwd=BACKEND, capture_output=True, timeout=900)
    out = (r.stdout + r.stderr).decode("utf-8", "replace")

    if "no tests to run" in out or "no test files" in out:
        return False, ("**لا اختبارَ بهذا الاسم في %s** — راجِعِ الحزمةَ أو "
                       "الاسمَ في المصفوفة" % pkg)

    ran_pass = ("--- PASS: " + name) in out if name else "PASS" in out
    ran_skip = ("--- SKIP: " + name) in out if name else "--- SKIP:" in out
    ran_fail = ("--- FAIL: " + name) in out if name else "--- FAIL:" in out

    if ran_skip and not ran_pass:
        return None, "**مُخطَّى** — شرطُه غائبٌ (قاعدةُ اختبارٍ؟) فلم يُقَس"

    expect_pass = spec.get("expect_pass", True)
    tail = " / ".join(x.strip() for x in out.strip().splitlines()[-3:])
    if expect_pass:
        return ran_pass, ("rc=%d %s" % (r.returncode, tail))[:300]
    # **والمنتظَرُ سقوطُه** — فيُطلب دليلُ السقوط لا مجرّدُ رمزِ خروج.
    return (ran_fail or r.returncode != 0), ("rc=%d %s" % (r.returncode, tail))[:300]


def run_script(spec):
    cmd = spec["cmd"]
    exe = shutil.which(cmd[0]) or cmd[0]
    r = subprocess.run([exe] + cmd[1:], cwd=ROOT, capture_output=True,
                       timeout=spec.get("timeout", 600))
    out = (r.stdout + r.stderr).decode("utf-8", "replace")
    ok = (r.returncode == 0) == spec.get("expect_pass", True)
    if "expect_contains" in spec:
        ok = ok and (spec["expect_contains"] in out)
    tail = " / ".join(x.strip() for x in out.strip().splitlines()[-2:])
    return ok, ("rc=%d %s" % (r.returncode, tail))[:300]


def run_api(spec):
    """**نداءٌ حيٌّ على التجهيز** — ويمرّ ببوّابة الهدف لزوماً."""
    import urllib.error
    import urllib.request
    url = target.base_url("/api/v1") + spec["path"]
    req = urllib.request.Request(url, method=spec.get("method", "GET"))
    req.add_header("Accept", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=40) as resp:
            code, body = resp.getcode(), resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        code, body = e.code, e.read().decode("utf-8", "replace")
    except Exception as exc:
        return False, "النداءُ فشل: %s" % exc
    if code != spec.get("expect_status", 200):
        return False, "http=%d والمنتظَرُ %d" % (code, spec.get("expect_status", 200))
    if "query" in spec:
        try:
            got = _query(json.loads(body), spec["query"])
        except Exception as exc:
            return False, "الاستعلامُ أخطأ: %s" % exc
        return (got == spec["expect"]), "got=%r want=%r" % (got, spec["expect"])
    return True, "http=%d" % code


RUNNERS = {"contract": run_contract, "gotest": run_gotest,
           "script": run_script, "api": run_api}


# ══════════════════════════════════════════════════════════════════════
#  التشغيل
# ══════════════════════════════════════════════════════════════════════

def select(rows, profile=None, area=None, ids=None):
    out = []
    for r in rows:
        if ids and r["id"] not in ids:
            continue
        if area and r.get("area") != area:
            continue
        if profile and profile not in (r.get("profiles") or []):
            continue
        out.append(r)
    return out


def execute(rows, do_run):
    results = []
    for r in rows:
        v = r.get("verify") or {}
        kind = v.get("kind")
        if kind in MANUAL or kind not in RUNNERS:
            results.append((r, r.get("status", "PENDING"), v.get("note", "")))
            continue
        # **والمحجوبُ لا يُشغَّل فيُقال «سقط»** — شرطُه غائبٌ لا سلوكُه خطأ.
        # (وقع: صفٌّ يقارن لقطةً لاحقةً لم تُؤخَذ بعد، فقُرئ فشلاً.)
        if r.get("status") == "BLOCKED":
            results.append((r, "BLOCKED", r.get("note", v.get("note", ""))))
            continue
        if not do_run:
            results.append((r, "PENDING", "(لم يُشغَّل — مرّر --run)"))
            continue
        try:
            ok, detail = RUNNERS[kind](v)
        except Exception as exc:
            ok, detail = False, "المنفّذُ انفجر: %s" % exc
        # **و`None` تعني «لم يُقَس»** — لا نجاحاً ولا سقوطاً.
        if ok is None:
            results.append((r, "BLOCKED", detail))
            continue
        # **والصفُّ المُعلَّمُ ثغرةً معروفةً لا يُقال «سقط»** — يُقال
        # «ما زال كما كان»، **فالفرقُ بين انحدارٍ جديدٍ وأثرٍ قديمٍ هو
        # كلُّ ما يُفيد.**
        #
        # **و`ok` هنا تعني «طابق المنتظَر»** — والمنتظَرُ في صفِّ ثغرةٍ
        # **أن تبقى الثغرة.** فمطابقتُه `KNOWN-GAP`، **ومخالفتُه
        # `CLEARED?`** — تغيَّر شيءٌ فليُنظَر أهو إصلاحٌ أم انحدار.
        # (كانت مقلوبةً أوّلاً فقالت «cleared» عن ثغرةٍ قائمة.)
        if r.get("status") == "KNOWN-GAP":
            results.append((r, "KNOWN-GAP" if ok else "CLEARED?", detail))
        else:
            results.append((r, "PASS" if ok else "FAIL", detail))
    return results


def report(results):
    counts = {}
    for _, st, _ in results:
        counts[st] = counts.get(st, 0) + 1
    width = max((len(r["id"]) for r, _, _ in results), default=8)
    for r, st, detail in results:
        line = "  %-8s %-*s %s" % (st, width, r["id"], r["title"])
        print(line)
        if st in ("FAIL", "CLEARED?") and detail:
            print("           %s" % detail[:200])
    print("")
    order = [s for s in STATUS_ORDER + ["CLEARED?"] if s in counts]
    print("  " + " · ".join("%s=%d" % (s, counts[s]) for s in order))
    print("  المجموع: %d" % len(results))
    return counts


# ══════════════════════════════════════════════════════════════════════
#  الوثيقة
# ══════════════════════════════════════════════════════════════════════

def render():
    """**يكتب الوثيقةَ من الجدول** — فلا نسختان تتناقضان."""
    m = load()
    rows = m["rows"]
    out = []
    out.append("<!-- gen:matrix -->")
    out.append("")
    # ملخّصٌ بالمجالات
    areas = {}
    for r in rows:
        a = r.get("area", "?")
        areas.setdefault(a, []).append(r)
    out.append("| المجال | صفوف | يُشغَّل آليّاً | شاهدُ جهاز | قبولُ مالك | ثغرةٌ معروفة |")
    out.append("|---|---|---|---|---|---|")
    for a in sorted(areas):
        rs = areas[a]
        auto = sum(1 for r in rs if (r.get("verify") or {}).get("kind") in AUTO)
        dev = sum(1 for r in rs if (r.get("verify") or {}).get("kind") == "device")
        own = sum(1 for r in rs if (r.get("verify") or {}).get("kind") == "owner")
        gap = sum(1 for r in rs if r.get("status") == "KNOWN-GAP")
        out.append("| `%s` | %d | %d | %d | %d | %s |"
                   % (a, len(rs), auto, dev, own, ("**%d**" % gap) if gap else "—"))
    out.append("")
    for a in sorted(areas):
        out.append("### `%s`" % a)
        out.append("")
        out.append("| ID | البند | الأنماط | كيف يُتحقَّق | الحالة |")
        out.append("|---|---|---|---|---|")
        for r in areas[a]:
            v = r.get("verify") or {}
            st = r.get("status", "PENDING")
            if st == "KNOWN-GAP":
                st = "**KNOWN-GAP**"
            out.append("| `%s` | %s | `%s` | `%s` | %s |"
                       % (r["id"], r["title"],
                          ",".join(r.get("profiles") or []),
                          v.get("kind", "—"), st))
        out.append("")
    out.append("<!-- /gen:matrix -->")
    body = "\n".join(out) + "\n"

    raw = io.open(DOC, encoding="utf-8").read()
    new = re.sub(r"(?s)<!-- gen:matrix -->.*?<!-- /gen:matrix -->", body.rstrip("\n"), raw)
    if new != raw:
        io.open(DOC, "w", encoding="utf-8", newline="").write(new)
        print("  wrote %s" % os.path.relpath(DOC, ROOT).replace("\\", "/"))
    else:
        print("  (الوثيقةُ حاضرةٌ بلا تغيير)")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--profile")
    ap.add_argument("--area")
    ap.add_argument("--id", nargs="*")
    ap.add_argument("--run", action="store_true")
    ap.add_argument("--list", action="store_true")
    ap.add_argument("--render", action="store_true")
    a = ap.parse_args()

    if a.render:
        render()
        return

    m = load()
    if a.list:
        profs = {}
        for r in m["rows"]:
            for p in (r.get("profiles") or []):
                profs[p] = profs.get(p, 0) + 1
        print("  الأنماط:")
        for p in sorted(profs, key=lambda k: -profs[k]):
            print("    %-10s %d صفّاً" % (p, profs[p]))
        print("")
        areas = {}
        for r in m["rows"]:
            areas[r.get("area", "?")] = areas.get(r.get("area", "?"), 0) + 1
        print("  المجالات:")
        for k in sorted(areas):
            print("    %-12s %d" % (k, areas[k]))
        print("")
        print("  المجموع: %d صفّاً" % len(m["rows"]))
        return

    rows = select(m["rows"], a.profile, a.area, set(a.id) if a.id else None)
    if not rows:
        raise SystemExit("لا صفوفَ بهذا الاختيار")
    print(target.banner())
    print("  المختارُ: %d صفّاً%s" % (len(rows),
                                     (" · النمط %s" % a.profile) if a.profile else ""))
    print("")
    res = execute(rows, a.run)
    counts = report(res)
    if a.run:
        stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        d = os.path.join(ROOT, "artifacts", "integrated-acceptance", "runs")
        os.makedirs(d, exist_ok=True)
        p = os.path.join(d, "run-%s.json" % stamp)
        io.open(p, "w", encoding="utf-8", newline="").write(json.dumps({
            "at": stamp, "target": target.target(), "profile": a.profile,
            "counts": counts,
            "rows": [{"id": r["id"], "status": st, "detail": d2}
                     for r, st, d2 in res],
        }, ensure_ascii=False, indent=1) + "\n")
        print("  أثرٌ مكتوب: %s" % os.path.relpath(p, ROOT).replace("\\", "/"))
    if counts.get("FAIL"):
        sys.exit(1)


if __name__ == "__main__":
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass
    main()
