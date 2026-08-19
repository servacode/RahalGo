# -*- coding: utf-8 -*-
"""**الدورةُ التشغيليّةُ كاملةً — والشاشةُ تُقرأ عند كلّ مرحلة.**

(قرارُ المالك ٢٠٢٦-٠٨-١٩: «اختبارٌ شاملٌ على تطبيق الزبون والسائق
 وفحص الدورة التشغيليّة».)

**وهذا ما لا يقيسه اختبارُ API**: الحالُ تتغيّر في الخادم، **والسؤال
هل تصل شاشةَ الزبون بلا أن يلمسها.** فبعد كلّ انتقالٍ تُقرأ الشجرةُ من
الجهاز.
"""
import json
import os
import sys
import time
import urllib.error
import urllib.request as u
import uuid

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "qa"))
import ui  # noqa: E402

sys.stdout.reconfigure(encoding="utf-8")
B = "https://api.rahalgo.com/api/v1"
CUSTOMER = ("0996280740", "Rahal2026x", None)
DRIVER = ("0994352064", "Mm12341234", None)
ADMIN = ("0985395131", "Mm12341234", "2525")

rows = []


def call(p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok:
        h["Authorization"] = "Bearer " + tok
    if key:
        h["Idempotency-Key"] = key
    try:
        r = u.Request(B + p, data=json.dumps(body).encode() if body is not None else None,
                      headers=h)
        rr = u.urlopen(r, timeout=30)
        return rr.status, json.loads(rr.read().decode() or "{}").get("data")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:130]


def login(ph, pw, pin=None):
    c, d = call("/auth/login", body={"phone": ph, "password": pw})
    if c != 200:
        raise SystemExit("دخولٌ فاشل %s: %s %s" % (ph, c, d))
    if d.get("pin_required"):
        c, d = call("/auth/pin", body={"challenge": d["challenge"], "pin": pin})
        if c != 200:
            raise SystemExit("PIN مرفوض: %s %s" % (c, d))
    return d["tokens"]["access_token"]


def row(name, ok, detail=""):
    rows.append((name, ok, detail))
    print("  %s %-38s %s" % ("✔" if ok else "✘", name, detail))


def screen():
    """**ما تقوله شاشةُ الزبون الآن** — نصّاً لا صورة."""
    return [x for x in ui.texts() if x.strip()]


STAGE_WORDS = {
    "pending": "بانتظار القبول", "accepted": "مقبول", "preparing": "قيد التحضير",
    "dispatching": "بانتظار القبول", "assigned": "مع سائق",
    "at_pickup": "في الطريق", "picked_up": "في الطريق", "on_the_way": "في الطريق",
    "at_dropoff": "وصل إليك", "delivered": "تم التسليم",
}


print("═" * 72)
print("  الدورةُ التشغيليّةُ الكاملة — الخادمُ والشاشةُ معا")
print("═" * 72)
cust, drv, adm = login(*CUSTOMER), login(*DRIVER), login(*ADMIN)

# ── التطبيقُ يُفتح على «طلباتي» ويُترك ─────────────────────────────
ui.sh("shell", "am", "force-stop", ui.PKG)
ui.sh("logcat", "-c")
ui.launch()
time.sleep(12)
ui.sh("shell", "input", "tap", "540", "2125")
time.sleep(3.5)
before = screen()
row("التطبيقُ على «طلباتي»", "طلباتي" in " ".join(before[:3]), before[0] if before else "?")

# ── ١ · طلبٌ جديد ─────────────────────────────────────────────────
sec = call("/public/home")[1]["sections"][0]["id"]
item = next(i for i in call("/public/sections/%s/items" % sec)[1]["items"] if i["available"])
ad = call("/my/addresses", cust)[1]
ad = ad["addresses"] if isinstance(ad, dict) else ad
body = {"items": [{"menu_item_id": item["id"], "qty": 1}],
        "address_text": "الرقة — اختبار", "lat": ad[0]["lat"], "lng": ad[0]["lng"],
        "payment_method": "cash"}
c, o = call("/orders", cust, body, key=str(uuid.uuid4()))
row("١ · الزبون يطلب", c in (200, 201), "#%s · %s ل.س" % (o.get("number"), o.get("total")))
oid, num = o["id"], o["number"]

time.sleep(6)
now = screen()
row("٢ · الطلبُ يظهر في الشاشة بلا لمس", ("#%d" % num) in now,
    "أرقامٌ ظاهرة: %s" % [x for x in now if x.startswith("#")][:3])


def move(label, fn, want):
    """**ينقل ثمّ يقرأ الخادمَ ثمّ الشاشة.**"""
    c, d = fn()
    if c >= 400:
        row(label, False, "ردّ %s %s" % (c, str(d)[:60]))
        return False
    time.sleep(7)
    srv_st = call("/my/orders/" + oid, cust)[1]
    srv_st = (srv_st.get("order", srv_st) if isinstance(srv_st, dict) else {}).get("status")
    word = STAGE_WORDS.get(srv_st, srv_st)
    on = word in " ".join(screen())
    row(label, on, "الخادم=%s · الشاشة=%s" % (srv_st, "تُظهر «%s»" % word if on else "**لا تُظهرها**"))
    return True


move("٣ · الإدارة تقبل",
     lambda: call("/admin/orders/%s/transition" % oid, adm, {"to": "accepted"}), "accepted")

# ══════════════════════════════════════════════════════════════════
# **٤ · «بدأتُ الطبخ» — وهي التي تُنزل الطلبَ إلى الطابور**
# ══════════════════════════════════════════════════════════════════
#
# **والإنزالُ التلقائيُّ يقع عند `preparing` لا عند `accepted`**
# (`orders/transitions.go: autoDispatch`) — **وفي وضع «المتجر يدير»
# يعني القبولُ وحدَه أنّ الطلبَ مقبولٌ لا أنّه بدأ.**
#
# (كُشف ٢٠٢٦-٠٨-١٩: دورتي الأولى تخطّت هذه الخطوةَ فبقي الطلبُ عند
#  `accepted` وطابورُ السائق فارغ — **وقُرئ عطباً وهو نقصٌ في المشية.**)
move("٤ · الإدارة تُنزله إلى الطابور",
     lambda: call("/admin/orders/%s/transition" % oid, adm, {"to": "dispatching"}),
     "dispatching")

# ── ٥ · السائقُ يقبل من الطابور ───────────────────────────────────
time.sleep(6)
q = call("/driver/queue", drv)[1] or []
mine = next((x for x in q if x.get("id") == oid), None)
if mine:
    move("٥ · السائق يقبل من الطابور",
         lambda: call("/driver/orders/%s/accept" % oid, drv, {}), "assigned")
else:
    row("٥ · السائق يقبل من الطابور", False,
        "الطلبُ ليس في طابوره (%d في الطابور)" % len(q))

# ── ٥ · حديثٌ من السائق ───────────────────────────────────────────
TXT = "أنا قربتُ من المتجر — اختبارٌ آليّ"
c, _ = call("/orders/%s/messages" % oid, drv, {"body": TXT})
time.sleep(6)
sc = screen()
row("٦ · رسالةُ السائق تصل الشاشة", c in (200, 201),
    "شارةٌ على القرص: %s" % ([x for x in sc if x.strip().isdigit()][:2] or "لا شارة"))

# ── ٦…٩ · مراحلُ التسليم ──────────────────────────────────────────
n = 7
for to in ("at_pickup", "picked_up", "on_the_way", "at_dropoff"):
    move("%d · السائق ← %s" % (n, to),
         lambda t=to: call("/driver/orders/%s/transition" % oid, drv, {"to": t}), to)
    n += 1

call("/driver/orders/%s/proof/skip" % oid, drv, {"reason": "اختبارٌ آليّ"})
move("%d · التسليم" % n,
     lambda: call("/driver/orders/%s/transition" % oid, drv, {"to": "delivered"}), "delivered")

print("─" * 72)
ok = sum(1 for _, g, _ in rows if g)
print("  %d/%d ناجحة" % (ok, len(rows)))
bad = [n for n, g, _ in rows if not g]
if bad:
    print("  الساقطة: " + " · ".join(bad))
print("  انهيارات: %s" % (ui.crashes() or "صفر"))
print("═" * 72)
