# -*- coding: utf-8 -*-
"""**حارسُ بوّابة الهدف — فلا يعود سكربتٌ يقصد الإنتاجَ صامتاً.**

(شرطُ المالك الحاجب ٢٠٢٦-٠٩-٢٩.)

**ويقرأ الملفّاتَ لا النيّةَ**: من أعاد عنوانَ الإنتاج حرفيّاً إلى سكربتٍ،
أو أعاد كلمةَ مرورٍ إلى المستودع، **يُسقط هذا الفحص.**

# ولماذا يمشي على المستودع كلِّه

**بحثتُ أوّلاً في `scripts/` وحدَها فخرج أخضرَ** — **وكانت أربعةُ محرّكاتٍ
أخرى تقصد الإنتاجَ افتراضاً** في `mobile/drivertest` و`merchtest`
و`reptest` و`testkit`. **فالحارسُ الذي يحرس مجلّداً واحداً يطمئنك كذباً.**

    python scripts/qa/test_target.py
"""
import hashlib
import io
import os
import re
import shutil
import subprocess
import sys

# **وbash بالاسم المجرّد يفقد البيئةَ على هذا الجهاز** — مقيسٌ ٢٠٢٦-٠٩-٢٩:
#
#     ['bash', '-c', ...]  env={ZZTEST:hello}  ->  zz=[]      argv0=/bin/bash
#     [which('bash'), ...] env={ZZTEST:hello}  ->  zz=[hello] argv0=/usr/bin/bash
#
# **فحارسٌ ينادي `bash` مجرّداً يخرج أخضرَ كذباً** — لأنّ كلَّ متغيّرٍ
# يمرّره يصل فارغاً، فيرى السكربتُ الافتراضَ دائماً.
BASH = shutil.which("bash") or "bash"

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(HERE)
ROOT = os.path.dirname(SCRIPTS)

PROD = "https://api.rahalgo.com"
CONFIRM = "I-KNOW-THIS-IS-PRODUCTION"

# **الملفّاتُ التي يحقُّ لها ذكرُ عنوان الإنتاج** — ولا غيرُها.
#
# **والبوّابةُ نفسُها لا بدَّ أن تعرفَه** لتعرف ما ترفض، **والحرّاسُ
# الآخرون يذكرونه ليؤكّدوا الفصلَ** لا ليقصدوه.
ALLOWED = {
    "scripts/qa/target.py",       # البوّابة
    "scripts/qa/target.sh",       # البوّابة للصدفة
    "scripts/qa/test_target.py",  # هذا الحارس
}

# **السكربتاتُ التي كانت تقصد الإنتاج** — تبقى مرصودةً بالاسم،
# وكلٌّ منها يجب أن يمرّ بالبوّابة بالدالّة المناسبة للغته.
WATCHED = [
    ("scripts/e2e-cycle.py", "target.base_url("),
    ("scripts/perf-baseline.py", "target.base_url("),
    ("scripts/qa/media.py", "target.base_url("),
    ("mobile/testkit/api.py", "target.base_url("),
    ("mobile/drivertest/engine.sh", "rg_base_url"),
    ("mobile/merchtest/engine.sh", "rg_base_url"),
    ("mobile/reptest/engine.sh", "rg_base_url"),
]

SKIP_DIRS = {".git", "node_modules", ".next", "build", "__pycache__",
             ".gradle", "dist", "vendor", ".venv", "artifacts"}

# **الأسرارُ التي سُرِّبت فعلاً** — بصماتُها لا نصُّها.
#
# **كانت في `13323ba5` نصّاً صريحاً** (ثلاثةُ حساباتٍ حقيقيّةٍ ومعها رمزُ
# أدمن)، **والمستودعُ عامّ** — فنداءٌ بلا توثيقٍ على `raw.githubusercontent`
# ردَّ ٢٠٠. **وحذفُها من `HEAD` لا يمحوها من التاريخ**، فالعلاجُ تبديلُها
# (فعله المالك ٢٠٢٦-٠٩-٢٩). **وهذا الفحصُ يمنع رجوعَها.**
#
# **وتُخزَّن مُلخَّصةً لا صريحةً** — فالحارسُ نفسُه في مستودعٍ عامّ،
# **ولا يُنشر سرٌّ ليُحرَس.**
#
# **والرمزُ ذو الأربعةِ أرقامٍ لا يُدرَج**: أيُّ `2525` في أيّ ملفٍّ يوقعه،
# **فيصير الحارسُ ضجيجاً فيُهمَل.**
LEAKED_DIGESTS = {
    "589dca372860bd4a": "customer password (13323ba5)",
    "0918026eab8ac795": "driver+admin password (13323ba5)",
    "2bbb9830738f9cf6": "customer phone (13323ba5)",
    "40b201e03a89237e": "driver phone (13323ba5)",
    "96bd6f1c3b786b0d": "admin phone (13323ba5)",
}

_TOKEN = re.compile(r"[A-Za-z0-9_@.+-]{6,64}")
_TEXT_EXT = (".py", ".sh", ".go", ".kt", ".kts", ".ts", ".tsx", ".js", ".jsx",
             ".json", ".yml", ".yaml", ".md", ".sql", ".env", ".example",
             ".txt", ".xml", ".properties", ".toml", ".ps1")

failures = []


def check(name, ok, detail=""):
    print(("  PASS  " if ok else "  FAIL  ") + name
          + (("  -> " + detail) if detail and not ok else ""))
    if not ok:
        failures.append(name)


def run(env_extra, code):
    env = dict(os.environ)
    for k in ("RAHALGO_TARGET", "RAHALGO_ALLOW_PRODUCTION", "RAHALGO_CONFIRM", "API"):
        env.pop(k, None)
    env.update(env_extra)
    return subprocess.run([sys.executable, "-c", code], cwd=HERE, env=env,
                          capture_output=True)


def run_sh(env_extra, code):
    env = dict(os.environ)
    for k in ("RAHALGO_TARGET", "RAHALGO_ALLOW_PRODUCTION", "RAHALGO_CONFIRM", "API"):
        env.pop(k, None)
    env.update(env_extra)
    return subprocess.run([BASH, "-c", ". scripts/qa/target.sh\n" + code],
                          cwd=ROOT, env=env, capture_output=True)


def scripts_in_repo():
    """**كلُّ سكربتٍ في المستودع** — بايثون وصدفة، حيث كان."""
    for base, dirs, files in os.walk(ROOT):
        dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
        for f in files:
            if f.endswith((".py", ".sh")):
                p = os.path.join(base, f)
                yield p, os.path.relpath(p, ROOT).replace("\\", "/")


# ── البوّابةُ في بايثون ──────────────────────────────────────────────

def test_py_default_is_staging():
    r = run({}, "import target;print(target.target(), target.base_url(), target.web_url())")
    out = r.stdout.decode("utf-8", "replace")
    check("py: default is staging (api+web)",
          r.returncode == 0 and "staging-api.rahalgo.com" in out
          and "staging.rahalgo.com" in out, out.strip() or r.stderr.decode("utf-8", "replace"))


def test_py_production_refused():
    for extra, label in [
        ({"RAHALGO_TARGET": "production"}, "no flags"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "1"}, "allow only"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_CONFIRM": CONFIRM}, "confirm only"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "yes",
          "RAHALGO_CONFIRM": CONFIRM}, "allow not exactly 1"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "1",
          "RAHALGO_CONFIRM": "i-know-this-is-production"}, "confirm wrong case"),
    ]:
        r = run(extra, "import target;target.base_url()")
        check("py: production refused (%s)" % label, r.returncode != 0)
        r = run(extra, "import target;target.web_url()")
        check("py: web production refused (%s)" % label, r.returncode != 0)


def test_py_production_allowed_with_both_flags():
    r = run({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "1",
             "RAHALGO_CONFIRM": CONFIRM}, "import target;print(target.base_url())")
    check("py: production allowed with both flags",
          r.returncode == 0 and PROD.encode() in r.stdout)


def test_py_unknown_target_rejected():
    r = run({"RAHALGO_TARGET": "prod-ish"}, "import target;target.base_url()")
    check("py: unknown target rejected", r.returncode != 0)


def test_py_creds_require_env():
    r = run({}, "import target;target.creds('admin')")
    check("py: credentials refuse to default", r.returncode != 0)
    r = run({"RAHALGO_ADMIN_PHONE": "0900000000", "RAHALGO_ADMIN_PASSWORD": "x"},
            "import target;print(target.creds('admin'))")
    check("py: credentials read from env", r.returncode == 0)


# ── البوّابةُ في الصدفة ──────────────────────────────────────────────

def test_sh_default_is_staging():
    r = run_sh({}, 'rg_base_url /api/v1')
    check("sh: default is staging",
          r.returncode == 0 and b"staging-api.rahalgo.com/api/v1" in r.stdout,
          r.stdout.decode("utf-8", "replace"))


def test_sh_production_refused():
    for extra, label in [
        ({"RAHALGO_TARGET": "production"}, "no flags"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "1"}, "allow only"),
        ({"RAHALGO_TARGET": "production", "RAHALGO_CONFIRM": CONFIRM}, "confirm only"),
        # **ومن مرّر عنوانَ الإنتاج في `API` صريحاً يُرفض كذلك** — فالثغرةُ
        # الواضحةُ أن يبقى `API` بابَ تجاوزٍ بلا حارس.
        ({"API": PROD + "/api/v1"}, "API= forced to production"),
    ]:
        r = run_sh(extra, 'rg_base_url /api/v1')
        check("sh: production refused (%s)" % label,
              r.returncode != 0 and b"REFUSED" in r.stderr)


def test_sh_production_allowed_with_both_flags():
    r = run_sh({"RAHALGO_TARGET": "production", "RAHALGO_ALLOW_PRODUCTION": "1",
                "RAHALGO_CONFIRM": CONFIRM}, 'rg_base_url /api/v1')
    check("sh: production allowed with both flags",
          r.returncode == 0 and (PROD + "/api/v1").encode() in r.stdout)


def test_sh_explicit_local_api_honoured():
    r = run_sh({"API": "http://localhost:8080/api/v1"}, 'rg_base_url /api/v1')
    check("sh: explicit local API honoured",
          r.returncode == 0 and b"localhost:8080" in r.stdout)


def test_sh_unknown_target_rejected():
    r = run_sh({"RAHALGO_TARGET": "prod-ish"}, 'rg_base_url /api/v1')
    check("sh: unknown target rejected", r.returncode != 0)


# ── المستودعُ كلُّه ─────────────────────────────────────────────────

def test_no_bare_production_url_in_any_script():
    """**ولا عنوانَ إنتاجٍ حرفيٍّ في أيّ سكربتٍ خارجَ البوّابة.**"""
    offenders = []
    for path, rel in scripts_in_repo():
        if rel in ALLOWED:
            continue
        if PROD in io.open(path, encoding="utf-8", errors="ignore").read():
            offenders.append(rel)
    check("repo: no bare production URL in any .py/.sh", not offenders,
          ", ".join(sorted(offenders)))


def test_no_production_default_pattern():
    """**ولا نمطَ «الإنتاجُ افتراضاً»** — وهو ما أخفى الأربعةَ عنّي أوّلاً.

    `${API:-https://api...}` و`os.environ.get(..., "https://api...")`
    **كلاهما يقرأ كأنّه مُعامَل، وكلاهما يضرب الإنتاجَ بلا متغيّر.**
    """
    pats = [re.compile(r'\$\{[A-Z_]+:-\s*https://api\.rahalgo\.com'),
            re.compile(r'environ\.get\([^)]*["\']https://api\.rahalgo\.com'),
            re.compile(r'getenv\([^)]*["\']https://api\.rahalgo\.com')]
    offenders = []
    for path, rel in scripts_in_repo():
        if rel in ALLOWED:
            continue
        txt = io.open(path, encoding="utf-8", errors="ignore").read()
        if any(p.search(txt) for p in pats):
            offenders.append(rel)
    check("repo: no production-as-default pattern", not offenders,
          ", ".join(sorted(offenders)))


def test_no_hardcoded_credentials():
    """**ولا كلمةَ مرورٍ ولا رمزَ أدمن في سكربتٍ مُتابَع.**

    **والنمطُ: ثلاثيٌّ من نصوصٍ يبدأ برقم هاتفٍ سوريّ** — وهو الشكلُ الذي
    كان في `e2e-cycle.py` حرفيّاً.
    """
    triple = re.compile(r'\(\s*"0\d{9}"\s*,\s*"[^"]{4,}"\s*,')
    offenders = []
    for rel, _ in WATCHED:
        txt = io.open(os.path.join(ROOT, rel), encoding="utf-8", errors="ignore").read()
        if triple.search(txt):
            offenders.append(rel)
    check("repo: no hardcoded credential triples", not offenders, ", ".join(offenders))


def test_watched_scripts_use_the_gate():
    offenders = []
    for rel, marker in WATCHED:
        txt = io.open(os.path.join(ROOT, rel), encoding="utf-8", errors="ignore").read()
        if marker not in txt:
            offenders.append("%s (wants %s)" % (rel, marker))
    check("repo: watched scripts go through the gate", not offenders,
          ", ".join(offenders))


def test_leaked_secrets_absent():
    """**ولا سرٌّ من المسرَّبين يرجع إلى ملفٍّ مُتابَع.**

    **ويُقاس على المتابَعِ بـgit لا على القرص** — فالمُهمَلُ (`.tmp/`)
    لا يُدفَع، **والخطرُ في ما يُدفَع.**
    """
    try:
        listed = subprocess.run(["git", "ls-files", "-z"], cwd=ROOT,
                                capture_output=True, timeout=120)
        rels = [x for x in listed.stdout.decode("utf-8", "replace").split("\0") if x]
    except Exception as exc:                                   # pragma: no cover
        check("secrets: leaked values absent from tracked files", False,
              "git ls-files failed: %s" % exc)
        return

    offenders = []
    for rel in rels:
        if not rel.endswith(_TEXT_EXT) or rel in ALLOWED:
            continue
        path = os.path.join(ROOT, rel)
        try:
            txt = io.open(path, encoding="utf-8", errors="ignore").read()
        except OSError:
            continue
        for tok in set(_TOKEN.findall(txt)):
            d = hashlib.sha256(tok.encode()).hexdigest()[:16]
            if d in LEAKED_DIGESTS:
                offenders.append("%s (%s)" % (rel, LEAKED_DIGESTS[d]))
    check("secrets: leaked values absent from tracked files",
          not offenders, "; ".join(sorted(set(offenders))))


def test_watched_engines_refuse_production():
    """**والمحرّكاتُ تُشغَّل فعلاً** — فقراءةُ سطرٍ ليست إثباتاً."""
    env = dict(os.environ)
    env.pop("API", None)
    env["RAHALGO_TARGET"] = "production"
    env.pop("RAHALGO_ALLOW_PRODUCTION", None)
    env.pop("RAHALGO_CONFIRM", None)
    for rel, marker in WATCHED:
        if marker != "rg_base_url":
            continue
        r = subprocess.run([BASH, rel], cwd=ROOT, env=env, capture_output=True,
                           timeout=60)
        check("run: %s refuses production" % rel,
              r.returncode != 0 and b"REFUSED" in (r.stderr + r.stdout))


if __name__ == "__main__":
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass
    print("target gate guard")
    for fn in sorted(
            [v for k, v in list(globals().items()) if k.startswith("test_")],
            key=lambda f: f.__code__.co_firstlineno):
        fn()
    print("")
    if failures:
        print("FAILED: %d" % len(failures))
        sys.exit(1)
    print("all green")
