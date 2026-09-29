# -*- coding: utf-8 -*-
"""**خطُّ الأساس قبل الدورة الذهبيّة — ما كان قبل أن نلمس شيئاً.**

(حكمُ المالك الرابعُ ٢٠٢٦-٠٩-٢٩: «قبل إنشاء أيّ طلبٍ ذهبيٍّ جديد، التقط…
 وليكن دليلاً ثابتاً يفرّق بين **ما كان موجوداً** و**ما أحدثته هذه المرحلة**.»)

# لماذا يُلتقط قبلاً لا بعداً

**الدفترُ فيه خرقان قائمان** على التجهيز (`FI-04.a` و`FI-06.d`، صفٌّ لكلٍّ).
**فمن قرأ التقريرَ بعد الدورة وحدَه ظنَّهما منها** — فأصلح ما لم يكسره،
**أو — وهو أسوأ — ظنَّ خرقاً ثالثاً أحدثَته الدورةُ أثراً قديماً فتركه.**

**والفرقُ لا يُعرَف بالتذكّر** — يُعرَف برقمٍ مكتوبٍ قبلَ الفعل.

# وما لا يفعله

**لا يكتب شيئاً** — لا طلبَ ولا قيدَ ولا إعداد. `GET` وحدَها،
**و`POST` واحدةٌ للدخول.** ومن شكَّ فليقرأ: لا `POST` في هذا الملفّ إلّا
`/auth/login` و`/auth/pin`.

# كيف يُشغَّل

    python scripts/qa/baseline.py --label pre
    python scripts/qa/baseline.py --label post   # بعد الدورة، للمقارنة
    python scripts/qa/baseline.py --compare pre post

**والهدفُ التجهيزُ افتراضاً** — عبر [target]. **والحسابُ من البيئة**، أو
من `deploy/staging/.env.staging` بـ`--from-staging-env` (مُهمَلٌ بـgit،
**ولا يُطبع منه سرّ**).
"""
import argparse
import datetime
import io
import json
import os
import shutil
import subprocess
import sys
import urllib.error
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
import target  # noqa: E402

# **الحالاتُ الأربعَ عشرةَ من `internal/orders/statuses.go`** — بترتيبها
# هناك، **فالترتيبُ نفسُه مسارُ الطلب** ومن قرأ العمودَ رأى الرحلة.
STATUSES = ["pending", "accepted", "preparing", "dispatching", "assigned",
            "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered",
            "rejected", "cancelled", "failed", "refunded"]

OUT_DIR = os.path.join(ROOT, "artifacts", "integrated-acceptance")
TIMEOUT = 40


def _req(url, token=None, body=None):
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(url, data=data,
                               method="POST" if data else "GET")
    r.add_header("Accept", "application/json")
    r.add_header("User-Agent", "rahalgo-baseline")
    if data:
        r.add_header("Content-Type", "application/json")
    if token:
        r.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(r, timeout=TIMEOUT) as resp:
            return resp.getcode(), json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        try:
            return e.code, json.loads(raw)
        except ValueError:
            return e.code, {"raw": raw[:400]}
    except Exception as exc:
        return 0, {"error": str(exc)}


def load_staging_env():
    """**يحمّل حسابَ الأدمن من ملفّ التجهيز المُهمَل** — ولا يطبع قيمة."""
    p = os.path.join(ROOT, "deploy", "staging", ".env.staging")
    if not os.path.exists(p):
        raise SystemExit("لا يوجد %s" % p)
    got = []
    for line in io.open(p, encoding="utf-8", errors="replace").read().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        v = v.strip().strip('"').strip("'")
        if k.strip() == "STAGING_ADMIN_PHONE" and v:
            os.environ["RAHALGO_ADMIN_PHONE"] = v
            got.append("phone")
        elif k.strip() == "STAGING_ADMIN_PASSWORD" and v:
            os.environ["RAHALGO_ADMIN_PASSWORD"] = v
            got.append("password")
    print("  staging admin creds loaded: %s" % (", ".join(got) or "NONE"))


# ══════════════════════════════════════════════════════════════════════
# **طريقُ القاعدة — قراءةً فقط عبر SSH**
# ══════════════════════════════════════════════════════════════════════
#
# **ولماذا لا بابُ الأدمن**: `bootstrapPassword` لا يضبط كلمةَ البيئة إلّا
# لحسابٍ بلا كلمة، **ويضبطها مؤقّتةً يفرض أوّلُ دخولٍ تبديلَها**
# (`identity/service.go:1374`). **فما في `.env.staging` بذرةُ إقلاعٍ لا
# كلمةٌ حيّة** — ومن جرّبها نال `invalid_credentials`. (والدخولُ يُقفل
# بخمس محاولاتٍ في ربع ساعة، `identity/service.go:76`.)
#
# **فأذِن المالكُ بقراءة القاعدة** (٢٠٢٦-٠٩-٢٩): `SELECT` وحدَها.
#
# **والإنتاجُ على الصندوق نفسِه** — `rahalgo-api-1` و`rahalgo-postgres`
# بجوار `rahalgo-staging-*`. **فالاسمُ مثبَّتٌ هنا حرفيّاً ولا يُبنى من
# متغيّر**، ومن أخطأ حرفاً قرأ قاعدةَ زبائنَ حقيقيّين.
SSH_HOST = "root@195.201.141.130"
SSH_KEY = os.path.expanduser("~/.ssh/rahalgo_claudeops")
PG_CONTAINER = "rahalgo-staging-postgres"   # **لا يُغيَّر** — انظر أعلاه
PG_DB = "rahalgo_staging"

BASELINE_SQL = r"""
SELECT 'orders_by_status', status, count(*)::text FROM orders GROUP BY status
UNION ALL SELECT 'orders_open', (closed_at IS NULL)::text, count(*)::text FROM orders GROUP BY 2
UNION ALL SELECT 'wallets_flagged', 'treasury', count(*)||'/'||COALESCE(sum(balance),0) FROM wallets WHERE is_treasury
UNION ALL SELECT 'wallets_flagged', 'cash_holding', count(*)||'/'||COALESCE(sum(balance),0) FROM wallets WHERE is_cash_holding
UNION ALL SELECT 'wallets_flagged', 'plain', count(*)||'/'||COALESCE(sum(balance),0) FROM wallets WHERE NOT is_treasury AND NOT is_cash_holding
UNION ALL SELECT 'wallet_tx_by_kind', kind, count(*)||'/'||COALESCE(sum(amount),0) FROM wallet_transactions GROUP BY kind
UNION ALL SELECT 'wallet_tx_total', 'all', count(*)||'/'||COALESCE(sum(amount),0) FROM wallet_transactions
UNION ALL SELECT 'driver_cash', 'boxes', count(*)||'/'||COALESCE(sum(held),0) FROM driver_cash_boxes
UNION ALL SELECT 'driver_cash_by_kind', kind, count(*)||'/'||COALESCE(sum(amount),0) FROM driver_cash_entries GROUP BY kind
ORDER BY 1,2;
"""


def read_db_over_ssh():
    """**يقرأ القاعدةَ ولا يكتبها** — `SELECT` وحدَها، ولا `psql -c` بأمرٍ مبنيّ.

    **والاستعلامُ يُمرَّر على المَدخل القياسيّ** لا في سطر أوامر — فالاقتباسُ
    المتشابك هو ما يصنع أمراً غيرَ الذي قُصد.
    """
    ssh = shutil.which("ssh")
    if not ssh:
        raise SystemExit("لا يوجد ssh في المسار")
    cmd = [ssh, "-i", SSH_KEY, "-o", "BatchMode=yes", "-o", "ConnectTimeout=20",
           SSH_HOST,
           "docker exec -i %s psql -U rahalgo -d %s -A -F'|' -t"
           % (PG_CONTAINER, PG_DB)]
    r = subprocess.run(cmd, input=BASELINE_SQL.encode(), capture_output=True,
                       timeout=180)
    out = r.stdout.decode("utf-8", "replace")
    if r.returncode != 0 or "ERROR:" in out:
        raise SystemExit("db read failed rc=%s\n%s\n%s"
                         % (r.returncode, out[:600],
                            r.stderr.decode("utf-8", "replace")[:400]))
    buckets = {}
    for line in out.splitlines():
        parts = line.strip().split("|")
        if len(parts) != 3:
            continue
        bucket, key, val = parts
        buckets.setdefault(bucket, {})[key] = val
    return buckets


def login(base):
    phone, password, pin = target.creds("admin")
    code, body = _req(base + "/auth/login", body={"phone": phone,
                                                  "password": password})
    if code != 200:
        raise SystemExit("login failed http=%s %s" % (code, str(body)[:300]))
    d = body.get("data") or body
    tok = d.get("token") or d.get("access_token")
    if not tok:
        raise SystemExit("login gave no token: %s" % list(d.keys()))
    return tok, d


def capture(base, token=None, via_ssh=False):
    """**يقرأ ولا يكتب** — وكلُّ رقمٍ هنا بصمةٌ تُقارَن لاحقاً."""
    snap = {}

    code, ident = _req(base + "/public/identity")
    snap["identity"] = {"http": code, "data": (ident.get("data") or ident)}

    code, rec = _req(base + "/qa/reconcile")
    snap["reconcile"] = {"http": code, "data": (rec.get("data") or rec)}

    # ── عددُ الطلبات بكلّ حالة ──────────────────────────────────────
    #
    # **ولا تُحذف المئةُ وتسعةَ عشرَ ملغىً** (نصُّ المالك) — **تُعَدُّ فتصير
    # خطَّ أساسٍ**، فمن رأى العددَ ينمو عرف أنّ الدورةَ ألغت طلباً.
    if via_ssh:
        b = read_db_over_ssh()
        snap["_source"] = "ssh-readonly"
        raw = b.get("orders_by_status", {})
        # **وكلُّ حالةٍ تُذكر ولو بصفر** — فغيابُ مفتاحٍ يُقرأ «لم يُقَس»
        # **وصفرٌ يُقرأ «قِيس فكان صفراً»**، وهما سؤالان.
        snap["orders_by_status"] = {st: int(raw.get(st, 0)) for st in STATUSES}
        snap["orders_total_summed"] = sum(snap["orders_by_status"].values())
        snap["orders_total_reported"] = snap["orders_total_summed"]
        snap["orders_open_closed"] = b.get("orders_open", {})
        snap["money"] = {
            "wallets_flagged": b.get("wallets_flagged", {}),
            "wallet_tx_by_kind": b.get("wallet_tx_by_kind", {}),
            "wallet_tx_total": b.get("wallet_tx_total", {}),
            "driver_cash": b.get("driver_cash", {}),
            "driver_cash_by_kind": b.get("driver_cash_by_kind", {}),
        }
        return snap

    snap["_source"] = "admin-api"
    by_status, total_seen = {}, 0
    for st in STATUSES:
        code, body = _req("%s/admin/orders?status=%s&per_page=1" % (base, st), token)
        if code != 200:
            by_status[st] = {"http": code, "error": str(body)[:160]}
            continue
        d = body.get("data") or body
        n = d.get("total")
        if n is None:
            n = len(d.get("orders") or [])
        by_status[st] = n
        if isinstance(n, int):
            total_seen += n
    snap["orders_by_status"] = by_status
    snap["orders_total_summed"] = total_seen

    code, body = _req(base + "/admin/orders?per_page=1", token)
    d = (body.get("data") or body) if code == 200 else {}
    snap["orders_total_reported"] = d.get("total") if code == 200 else {"http": code}

    code, st = _req(base + "/admin/stats", token)
    snap["stats"] = {"http": code, "data": (st.get("data") or st) if code == 200 else st}

    return snap


def write_snapshot(label, snap):
    d = os.path.join(OUT_DIR, "baseline-" + label)
    os.makedirs(d, exist_ok=True)
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    snap["_captured_at_utc"] = stamp
    snap["_target"] = target.target()
    p = os.path.join(d, "snapshot.json")
    io.open(p, "w", encoding="utf-8", newline="").write(
        json.dumps(snap, ensure_ascii=False, indent=1) + "\n")
    print("  wrote %s" % os.path.relpath(p, ROOT).replace("\\", "/"))
    return p


def summary(snap):
    m = (snap.get("reconcile", {}).get("data") or {}).get("money") or {}
    print("")
    print("  money checks      : %s/%s passed" % (m.get("checks_passed"),
                                                  m.get("checks_total")))
    for f in (m.get("failed") or []):
        print("    PRE-EXISTING %-9s rows=%-3s %s" % (
            f.get("id"), f.get("violating_rows"), f.get("by_status")))
    print("  orders by status  :")
    for st in STATUSES:
        v = snap.get("orders_by_status", {}).get(st)
        if isinstance(v, int) and v:
            print("    %-13s %d" % (st, v))
    print("  orders total      : reported=%s summed=%s" % (
        snap.get("orders_total_reported"), snap.get("orders_total_summed")))
    res = (snap.get("reconcile", {}).get("data") or {}).get("residue") or {}
    print("  qa residue        : %s" % json.dumps(res, ensure_ascii=False))


def compare(a, b):
    """**الفرقُ وحدَه** — فمن قرأ لقطتين كاملتين لم يرَ ما تغيّر."""
    pa = os.path.join(OUT_DIR, "baseline-" + a, "snapshot.json")
    pb = os.path.join(OUT_DIR, "baseline-" + b, "snapshot.json")
    for p in (pa, pb):
        if not os.path.exists(p):
            raise SystemExit("لا توجد لقطة: %s" % p)
    A = json.load(io.open(pa, encoding="utf-8"))
    B = json.load(io.open(pb, encoding="utf-8"))
    print("  %s -> %s" % (A.get("_captured_at_utc"), B.get("_captured_at_utc")))
    print("  orders by status (changed only):")
    quiet = True
    for st in STATUSES:
        x, y = A["orders_by_status"].get(st), B["orders_by_status"].get(st)
        if x != y:
            quiet = False
            print("    %-13s %s -> %s" % (st, x, y))
    if quiet:
        print("    (none)")
    fa = {f["id"]: f for f in ((A["reconcile"]["data"].get("money") or {}).get("failed") or [])}
    fb = {f["id"]: f for f in ((B["reconcile"]["data"].get("money") or {}).get("failed") or [])}
    print("  invariants:")
    for k in sorted(set(fa) | set(fb)):
        if k not in fa:
            print("    INTRODUCED  %s rows=%s" % (k, fb[k].get("violating_rows")))
        elif k not in fb:
            print("    CLEARED     %s" % k)
        elif fa[k].get("violating_rows") != fb[k].get("violating_rows"):
            print("    CHANGED     %s %s -> %s" % (k, fa[k].get("violating_rows"),
                                                   fb[k].get("violating_rows")))
        else:
            print("    unchanged   %s rows=%s (pre-existing)" % (
                k, fa[k].get("violating_rows")))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--label", default="pre")
    ap.add_argument("--from-staging-env", action="store_true")
    ap.add_argument("--via-ssh", action="store_true",
                    help="يقرأ الطلباتَ والمالَ من القاعدة عبر SSH قراءةً فقط")
    ap.add_argument("--compare", nargs=2, metavar=("A", "B"))
    a = ap.parse_args()

    if a.compare:
        compare(*a.compare)
        return

    print(target.banner())
    base = target.base_url("/api/v1")

    if a.via_ssh:
        # **ولا يُقرأ الإنتاجُ من هذا الطريق أبداً** — الحاويةُ مثبَّتةٌ على
        # التجهيز، **فمن أراد الإنتاجَ فليس هذا بابَه.**
        if target.target() != "staging":
            raise SystemExit("--via-ssh للتجهيز وحدَه")
        snap = capture(base, via_ssh=True)
        print("  db read over ssh (read-only)")
    else:
        if a.from_staging_env:
            load_staging_env()
        token, _ = login(base)
        print("  admin logged in")
        snap = capture(base, token)

    write_snapshot(a.label, snap)
    summary(snap)


if __name__ == "__main__":
    try:
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    except Exception:
        pass
    main()
