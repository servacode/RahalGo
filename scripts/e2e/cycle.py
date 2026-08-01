# -*- coding: utf-8 -*-
"""الدورة التشغيلية الكاملة لرحّال — من أول لحظة إلى آخرها، بالتسلسل.

    python scripts/e2e/cycle.py

يفترض أن الخادم يعمل على 8080 وأن حاويات المشروع قائمة (5434/6380).

تُشغَّل على قاعدة التطوير الحيّة وتصنع كياناتها بنفسها (مندوب، متجر، زبون،
سائق)، ثم تحذفها في آخرها. وكل خطوة تُثبت أثرها **في القاعدة** لا في ردّ
الخادم وحده — فردٌّ بـ200 يقول إن النداء نجح، لا إن المال انتقل.

والعربية كلُّها في هذا الملف لا في سطر أوامر: مرّرناها في سلسلة أمرٍ من قبل
فوصلت محارفَ بديلة إلى القاعدة.
"""
import json
import random
import subprocess
import sys

import requests

API = "http://localhost:8080/api/v1"
S = requests.Session()
FAILS, STEP = [], 0


def phase(t):
    print(f"\n{'═' * 76}\n  {t}\n{'═' * 76}")


def step(t):
    global STEP
    STEP += 1
    print(f"\n[{STEP:02d}] {t}")


def ok(m):
    print(f"     ✓ {m}")


def bad(m):
    FAILS.append(m)
    print(f"     ✗ {m}")


def check(c, good, wrong):
    ok(good) if c else bad(wrong)
    return c


def call(method, path, token=None, body=None, expect=None):
    h = {"Content-Type": "application/json"}
    if token:
        h["Authorization"] = f"Bearer {token}"
    r = S.request(method, API + path, headers=h,
                  data=json.dumps(body, ensure_ascii=False).encode("utf-8") if body is not None else None,
                  timeout=30)
    try:
        j = r.json()
    except Exception:
        j = {"raw": r.text[:200]}
    if expect is not None and r.status_code != expect:
        bad(f"{method} {path} → {r.status_code} (توقّعنا {expect}): "
            f"{json.dumps(j, ensure_ascii=False)[:220]}")
    return r.status_code, j.get("data", j)


def login(phone, password):
    _, d = call("POST", "/auth/login", body={"phone": phone, "password": password}, expect=200)
    return d["tokens"]["access_token"]


def sql(q):
    """استعلامٌ مباشر على القاعدة — الحقيقة النهائية، لا ما يقوله الخادم عن نفسه."""
    r = subprocess.run(
        ["docker", "exec", "rahalgo-postgres", "psql", "-U", "rahalgo", "-d", "rahalgo",
         "-tAF|", "-c", q],
        capture_output=True, text=True, encoding="utf-8",
        env={"MSYS_NO_PATHCONV": "1", "PATH": __import__("os").environ["PATH"]})
    if r.returncode != 0:
        bad(f"SQL: {r.stderr[:200]}")
        return []
    return [ln.split("|") for ln in r.stdout.strip().splitlines() if ln]


# ═════════════════════════════════════════════════════════════════════════
RUN = random.randint(10000, 99999)
REP_PHONE, OWNER_PHONE = f"+9639771{RUN}", f"+9639661{RUN}"
CUST_PHONE, DRV_PHONE = f"+9639331{RUN}", f"+9639551{RUN}"
TEMP = "Rahal@2026Temp"
TAG = f"جولة {RUN}"

state = {"drv_phone": DRV_PHONE, "drv_temp_pw": TEMP, "drv_pw": "SaeqRahal@2026", "tag": TAG}

phase("المرحلة ١ — التأسيس: الأدمن يفتح الأدوار")

step("دخول مدير المنصة")
admin = login("+963999000001", "RahalGo@2026")
ok("دخل بكلمة مرور")

step("الإعدادات التي ستحكم هذه الدورة — من اللوحة لا من الشيفرة")
_, settings = call("GET", "/admin/settings", admin, expect=200)
S_MAP = {s["key"]: s["value"] for s in settings}
for k in ("merchants.default_commission_percent", "merchants.default_prep_minutes",
          "sales.commission_percent", "sales.activation_orders",
          "drivers.share_mode", "drivers.share_percent", "drivers.cash_limit",
          "drivers.max_active_orders", "payouts.min_amount",
          "security.password_min_length"):
    ok(f"{k:<38} = {S_MAP.get(k)}")
check(len(settings) == 23, f"٢٣ إعداداً بتعريفها الكامل", f"{len(settings)} إعداداً")

step("إنشاء حساب المندوب والسائق")
_, rep = call("POST", "/admin/users", admin, {
    "phone": REP_PHONE, "full_name": f"مندوب {TAG}", "password": TEMP, "roles": ["sales"]}, expect=201)
state["rep_id"] = rep["id"]
_, drv = call("POST", "/admin/users", admin, {
    "phone": DRV_PHONE, "full_name": f"سائق {TAG}", "password": TEMP, "roles": ["driver"]}, expect=201)
state["driver_id"] = drv["id"]
ok(f"المندوب {rep['id'][:8]} · السائق {drv['id'][:8]}")

phase("المرحلة ٢ — المندوب يجلب متجراً بكوده")

step("دخول المندوب بالكلمة المؤقّتة")
rep_tok = login(REP_PHONE, TEMP)
_, me = call("GET", "/auth/me", rep_tok, expect=200)
check(me.get("must_change_password") is True,
      "الحساب يطلب تغيير الكلمة — كلمةٌ يعرفها الأدمن ليست كلمته",
      "لا يطلب تغيير الكلمة (ثغرة R-60 عادت)")

step("يغيّر كلمته فيُرفع العلَم")
REP_PW = "MandoubRahal@2026"
call("POST", "/auth/password", rep_tok, {"current_password": TEMP, "password": REP_PW}, expect=200)
rep_tok = login(REP_PHONE, REP_PW)
_, me = call("GET", "/auth/me", rep_tok, expect=200)
check(me.get("must_change_password") is False, "صارت كلمته هو", "بقي العلَم مرفوعاً")
state["invite_code"] = me.get("invite_code")
check(bool(state["invite_code"]), f"كود دعوته يُولَد مع الدور: {state['invite_code']}", "بلا كود")

step("كلمةٌ ضعيفة تُرفض بحدّ اللوحة")
sc, d = call("POST", "/auth/password", rep_tok, {"current_password": REP_PW, "password": "12345"})
check((d.get("error") or {}).get("code") == "weak_password",
      f"رُفضت — الحدّ {S_MAP['security.password_min_length']} أحرف من الإعدادات",
      "قُبلت كلمة من خمسة أحرف")

step("صفحة الدعوة تعرّف بالمندوب قبل التسجيل")
_, inv = call("GET", f"/public/invite?ref={state['invite_code']}", expect=200)
ok(f"ردُّ الدعوة: {json.dumps(inv, ensure_ascii=False)}")
check(bool(inv.get("rep_name") or inv.get("full_name") or inv.get("name")),
      "الزائر يرى **اسم** من دعاه",
      "الدعوة تقول «by: rep» ولا تقول اسمه — الزائر لا يعرف بمن يثق")

step("صاحب المتجر يسجّل عبر رابط المندوب")
_, home0 = call("GET", "/public/home", expect=200)
cats = home0["categories"]
STORE = f"مشاوي الفرات — {TAG}"
_, lead = call("POST", "/public/join", body={
    "ref": state["invite_code"], "store_name": STORE, "owner_name": "أبو عمر",
    "phone": OWNER_PHONE, "area": "المشلب", "category_id": cats[0]["id"],
    "password": TEMP, "lat": 35.9528, "lng": 39.0079}, expect=201)
check(lead.get("received") is True,
      "الردّ يقول «وصل» ولا يُعيد معرّفاً — نقطةٌ عامّة لا تُسرّب معرّفات",
      f"ردٌّ غير متوقّع: {json.dumps(lead, ensure_ascii=False)[:120]}")
_, leads = call("GET", "/admin/leads", admin, expect=200)
lrows = leads if isinstance(leads, list) else leads.get("items", [])
mine_l = [x for x in lrows if x["store_name"] == STORE]
check(len(mine_l) == 1, f"الطلب في شاشة الإدارة بحالة «{mine_l[0]['status'] if mine_l else ''}»",
      "الطلب لم يصل شاشة الإدارة")
state["lead_id"] = mine_l[0]["id"]

step("والمندوب يرى عميله المحتمل في لوحته")
_, rleads = call("GET", "/rep/leads", rep_tok)
rl = rleads if isinstance(rleads, list) else rleads.get("items", [])
check(any(x.get("store_name") == STORE for x in rl),
      "الطلب في «عملائي» عند المندوب بحالته المعلّقة",
      f"المندوب لا يرى طلبه: {json.dumps(rleads, ensure_ascii=False)[:120]}")

step("لا متجر قبل موافقة الإدارة")
_, home = call("GET", "/public/home", expect=200)
check(STORE not in json.dumps(home, ensure_ascii=False),
      "لا يظهر للزبائن قبل الموافقة — **الموافقة هي لحظة الإنشاء**",
      "ظهر للزبائن قبل أن يوافق أحد")

step("الإدارة توافق فيُنشأ المتجر منسوباً للمندوب")
call("POST", f"/admin/leads/{state['lead_id']}/status", admin, {"status": "converted"}, expect=200)
r = sql(f"SELECT id, default_prep_minutes, commission_percent, sales_rep_user_id IS NOT NULL "
        f"FROM merchants WHERE name = '{STORE}'")
check(len(r) == 1, "أُنشئ المتجر", "لم يُنشأ")
state["merchant_id"] = r[0][0]
check(r[0][1] == str(S_MAP["merchants.default_prep_minutes"]),
      f"ورث وقت التحضير الافتراضي من اللوحة ({r[0][1]} دقيقة) لا من افتراض العمود",
      f"وقت التحضير {r[0][1]} لا {S_MAP['merchants.default_prep_minutes']}")
check(r[0][2] == str(S_MAP["merchants.default_commission_percent"]),
      f"وورث نسبة العمولة ({r[0][2]}٪)", f"العمولة {r[0][2]}")
check(r[0][3] == "t", "ومنسوبٌ للمندوب", "غير منسوب للمندوب")

phase("المرحلة ٣ — المتجر يجهّز نفسه")

step("دخول صاحب المتجر")
own_tok = login(OWNER_PHONE, TEMP)
ok("دخل بالكلمة التي كتبها عند التسجيل")

step("ضبط وقت التحضير والحدّ الأدنى")
call("PATCH", f"/merchant/stores/{state['merchant_id']}/settings", own_tok,
     {"default_prep_minutes": 15, "min_order": 20000}, expect=200)
ok("تحضير ١٥ دقيقة · حدٌّ أدنى ٢٠٬٠٠٠ — المتجر أعرف بمطبخه من المنصة")

step("إضافة قسم وصنف")
_, sec = call("POST", f"/merchant/stores/{state['merchant_id']}/menu/sections", own_tok,
              {"name": "المشاوي", "sort_order": 1}, expect=201)
_, item = call("POST", f"/merchant/stores/{state['merchant_id']}/menu/items", own_tok,
               {"section_id": sec["id"], "name": "فروج مشوي", "price": 60000,
                "description": "فروج كامل مع الثوم", "available": True}, expect=201)
state["item_id"], state["price"] = item["id"], 60000
ok("«المشاوي» ← «فروج مشوي» بـ٦٠٬٠٠٠")

step("ظهر للزبائن فوراً")
_, menu = call("GET", f"/public/merchants/{state['merchant_id']}", expect=200)
check("فروج مشوي" in json.dumps(menu, ensure_ascii=False), "الصنف في القائمة العامة", "لم يظهر")

phase("المرحلة ٤ — الزبون يدخل ويطلب")

step("إنشاء حساب الزبون ودخوله")
_, cu = call("POST", "/admin/users", admin, {
    "phone": CUST_PHONE, "full_name": f"زبون {TAG}", "password": TEMP,
    "roles": ["customer"]}, expect=201)
state["customer_id"] = cu["id"]
cust_tok = login(CUST_PHONE, TEMP)
call("POST", "/auth/password", cust_tok, {"current_password": TEMP, "password": "ZaboonRahal@2026"}, expect=200)
cust_tok = login(CUST_PHONE, "ZaboonRahal@2026")
ok("دخل الزبون")

step("يحفظ عنوانه")
call("POST", "/my/addresses", cust_tok, {
    "label": "البيت", "address_text": "شارع تل أبيض — خلف الجامع، الطابق الثاني",
    "lat": 35.9528, "lng": 39.0079}, expect=201)
_, addrs = call("GET", "/my/addresses", cust_tok, expect=200)
check(len(addrs) == 1 and addrs[0]["is_default"],
      "حُفظ وصار افتراضياً — أوّل عنوانٍ لا يُسأل صاحبه أهو الافتراضي",
      "لم يُحفظ افتراضياً")
check("خلف الجامع" in addrs[0]["address_text"],
      "والنصّ الحرّ محفوظ: «خلف الجامع، الطابق الثاني» أدلّ من أي إحداثية في الرقة",
      "النصّ الحرّ ضاع")

step("يبحث عن الصنف باسمه لا باسم المتجر")
_, hits = call("GET", "/public/search?q=فروج", expect=200)
h = [x for x in hits if x["id"] == state["merchant_id"]]
check(bool(h), f"ظهر المتجر — وسببُ ظهوره معلن: «{h[0].get('matched_items') if h else ''}»",
      "البحث لم يجده باسم صنفه")

step("طلبٌ نقديّ")
_, o = call("POST", "/orders", cust_tok, {
    "merchant_id": state["merchant_id"],
    "items": [{"menu_item_id": state["item_id"], "qty": 1}],
    "address_text": addrs[0]["address_text"], "lat": 35.9528, "lng": 39.0079,
    "payment_method": "cash"}, expect=201)
state["order_id"], state["order_no"] = o["id"], o["number"]
state["delivery_fee"], state["total"] = o["delivery_fee"], o["total"]
ok(f"وُلد #{state['order_no']} — {state['total']:,} "
   f"(بضاعة {state['price']:,} + توصيل {state['delivery_fee']:,})")

# ── التكملة ───────────────────────────────────────────────────────────────
sys.path.insert(0, __import__("os").path.dirname(__file__))
import cycle_part2 as e2e2  # noqa: E402

ctx = (call, login, phase, step, ok, bad, check, state, admin, rep_tok,
       own_tok, cust_tok, S_MAP, sql)
try:
    net = e2e2.run(ctx)
finally:
    phase("التنظيف — لا تُترك بيانات اختبارٍ تبدو حقيقية")
    # الحذف هنا حذفٌ صلب لأنها بيانات اختبار. **وحذف الحساب في التطبيق نفسه
    # تجريدٌ لا حذف** (`AnonymizeUser`) — يبقى الصفّ لتماسك القيود ويسقط عنه
    # كل ما يعرّف صاحبه. ولذلك يجب رفع قيود السجلّ يدوياً هنا وحدها.
    phones = "'" + "','".join([REP_PHONE, OWNER_PHONE, CUST_PHONE, DRV_PHONE]) + "'"
    sql(f"DELETE FROM order_ratings WHERE order_id IN (SELECT id FROM orders WHERE merchant_id = '{state.get('merchant_id')}')")
    sql(f"DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE merchant_id = '{state.get('merchant_id')}')")
    sql(f"DELETE FROM order_events WHERE order_id IN (SELECT id FROM orders WHERE merchant_id = '{state.get('merchant_id')}')")
    sql(f"DELETE FROM merchant_leads WHERE store_name LIKE '%{TAG}%'")
    sql(f"DELETE FROM orders WHERE merchant_id = '{state.get('merchant_id')}'")
    sql(f"DELETE FROM merchant_hours WHERE merchant_id = '{state.get('merchant_id')}'")
    sql(f"DELETE FROM menu_items WHERE merchant_id = '{state.get('merchant_id')}'")
    sql(f"DELETE FROM menu_sections WHERE merchant_id = '{state.get('merchant_id')}'")
    sql(f"DELETE FROM merchants WHERE id = '{state.get('merchant_id')}'")
    sql(f"DELETE FROM audit_log WHERE actor_user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM payout_requests WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM driver_cash_entries WHERE driver_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM driver_cash_boxes WHERE driver_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM wallet_transactions WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM wallets WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM user_addresses WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone IN ({phones}))")
    sql(f"DELETE FROM users WHERE phone IN ({phones})")
    left = sql(f"SELECT count(*) FROM merchants WHERE name LIKE '%{TAG}%'")
    ok(f"حُذفت كيانات الجولة (متبقٍّ: {left[0][0] if left else '?'})")

print(f"\n{'═' * 76}")
if FAILS:
    print(f"  ✗ سقط {len(FAILS)} تحقّقاً من {STEP} خطوة:")
    for f in FAILS:
        print(f"      · {f}")
    sys.exit(1)
print(f"  ✓ الدورة كاملة: {STEP} خطوة، كلُّها سليمة.")
