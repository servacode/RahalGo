# -*- coding: utf-8 -*-
"""**شهادةُ النموذج ٢ الحيّة على التجهيز** — «البضاعةُ خارجَ كلّ حساب».

(موافقةُ المالك ٢٠٢٦-١١-١٠: جرّب نموذجَ المال الجديد من أوّله لآخره على التجهيز.)

# ماذا يفعل

يشغّل المفتاحَ `finance.goods_outside` (ويحفظ قيمتَه السابقة)، ثمّ يمشي ثمانيةَ سيناريوهاتٍ عبر جسر
QA وحدَه (`/api/v1/qa/*` — غيرُ موجودٍ في الإنتاج)، ويحكم على كلّ واحدٍ من قراءة `GET /qa/money`،
ويطبع سطرَ `PASS`/`FAIL` لكلّ سيناريو، **ثمّ يعيد المفتاحَ كما كان** (إلّا مع `--keep-on`).

    standard_cash        طلبٌ عاديٌّ نقداً — البضاعةُ لا تدخل الدفتر، والكابتنُ يدفع المتجرَ عند الاستلام
    standard_wallet      أجرةٌ من المحفظة — المحفظةُ تدفع التوصيلَ وحدَه
    failure_after_pickup فشلٌ عند الباب — لا خسارةَ على المنصّة ولا سلفةَ ولا دين (للمتجر وللمكتب)
    refund_after_delivery استرجاعٌ بعد التسليم — الأجرةُ المدفوعةُ وحدَها
    custom               مشترياتٌ — لا قيدَ بضاعةٍ ولا سلفة
    whatsapp             طلبُ واتساب من متجر QA — العمولةُ عبر الصندوق، ولا خصمَ من المتجر
    usd                  مشترياتٌ بالدولار من متجرٍ تابع — العمولةُ من المعادل، والسقفُ لا يعدّ الدولار
    edit_after_pickup    طلبُ تعديلٍ من الكابتن بعد الاستلام وموافقةُ المكتب — الفرقُ بين الكابتن والمتجر

وفي كلّ سيناريو: **صفرُ خرقٍ لثوابت المال يذكر الطلب**، وصفرُ قيدِ بضاعة، وصفرُ سلفةِ درج، **وسقفُ نقد
الكابتن يعدّ مالَ المنصّة وحدَه** (نقدُ الطلب ناقصَ حقّ المتجر).

# التشغيل

    python scripts/qa/money_model2_staging.py                 # الكلّ، والمفتاحُ يرجع كما كان
    python scripts/qa/money_model2_staging.py --keep-on       # يبقى المفتاحُ مشغَّلاً بعد الشهادة
    python scripts/qa/money_model2_staging.py --only standard_cash,refund_after_delivery

**والهدفُ من البوّابة** (`target.py`): التجهيزُ افتراضاً، **والإنتاجُ يُرفض** إلّا بالعلَمَين — **ولن
يعمل هناك أصلاً**: جسرُ QA غيرُ مسجَّلٍ في الإنتاج (٤٠٤). **ولا سرَّ في السكربت**: الجسرُ يُصدر جلسةَ
زبون QA وسائقِ QA الثابتَين بنفسه، **ولا رسالةَ واتساب إلّا لرقم QA** (المستلِمُ +963900555001).

# الخاتمة (دائماً، ولو سقط سيناريو)

يسلّم صندوقَ سائق QA للمكتب (إلّا مع `--no-settle`)، ويصفّر محفظةَ زبون QA، ويعيد سياسةَ أجرة
المشتريات، ويطفئ دوامَ سائق QA مع وضع التوزيع السابق، **ويعيد المفتاح.**
"""
import argparse
import json
import sys
import urllib.error
import urllib.request as u
import uuid

import target

try:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
except Exception:  # pragma: no cover
    pass

MODEL2_KEY = "finance.goods_outside"
QA_RECIPIENT = "+963900555001"  # زبونُ QA الثابت — رقمٌ في المدى الوهميّ
QA_LAT, QA_LNG = 35.9506, 39.0094
ITEM_COST = 10000          # كلفةُ صنف متجر QA (حقُّ المتجر قبل العمولة)
STORE_SHARE = 9000         # ما يدفعه الكابتنُ للمتجر عن قطعةٍ واحدة (الكلفة ناقصَ ١٠٪)
COMMISSION = 1000          # عمولةُ المنصّة على قطعةٍ واحدة

TERMINAL = {"delivered", "rejected", "cancelled", "failed", "refunded"}
PRE_PICKUP = {"pending", "accepted", "preparing", "dispatching", "assigned", "at_pickup"}
# **سقفُ نقد الزبون يُفكّ مؤقّتاً** (بذرةُ `cod_limit` القائمة، تُرجع السابقَ فيُستعاد في الخاتمة):
# على التجهيز بالليرة الجديدة سقفٌ صغيرٌ يردّ طلبَ الصنف ١١٠٠٠ بـ`cod_limit_exceeded`. **وصفرٌ = بلا
# سقف** (`orders/service.go`: `limit <= 0`) — والفهرسُ لا يقبل فوق ١٠٠٠٠٠٠ أصلاً.
COD_LIMIT_FOR_RUN = 0

SCENARIOS = ["standard_cash", "standard_wallet", "failure_after_pickup", "refund_after_delivery",
             "custom", "whatsapp", "usd", "edit_after_pickup"]


class Api:
    def __init__(self, base):
        self.base = base

    def call(self, method, path, body=None, token=None, key=False, ok=True):
        """`ok=True`: أيُّ ٢xx وإلّا `Step` · `ok=None`: يُرجع الرمزَ بلا حكم."""
        data = None if body is None else json.dumps(body).encode("utf-8")
        req = u.Request(self.base + path, data=data, method=method)
        req.add_header("Accept", "application/json")
        if data is not None:
            req.add_header("Content-Type", "application/json")
        if token:
            req.add_header("Authorization", "Bearer " + token)
        if key:
            req.add_header("Idempotency-Key", "qa-m2-" + uuid.uuid4().hex)
        try:
            with u.urlopen(req, timeout=60) as r:
                code, raw = r.status, r.read()
        except urllib.error.HTTPError as e:
            code, raw = e.code, e.read()
        try:
            out = json.loads(raw.decode("utf-8")) if raw else {}
        except ValueError:
            out = {"raw": raw[:300].decode("utf-8", "replace")}
        if ok and not 200 <= code < 300:
            raise Step("%s %s -> %d %s" % (method, path, code, json.dumps(out, ensure_ascii=False)[:400]))
        # **والردُّ الناجحُ مغلَّفٌ بـ`data`** (`httpx.JSON`) — يُفكّ هنا مرّةً واحدة.
        if isinstance(out, dict) and set(out.keys()) == {"data"}:
            out = out["data"]
        return code, out

    def post(self, path, body=None, **kw):
        return self.call("POST", path, body if body is not None else {}, **kw)[1]

    def get(self, path, **kw):
        return self.call("GET", path, **kw)[1]


class Step(Exception):
    """**خطوةٌ رُدّت** — يسقط السيناريو بسببها الصريح."""


class Run:
    def __init__(self, api):
        self.api = api
        self.cust = self.drv = None
        self.item = self.store = self.driver_id = None
        self.held = None

    # ── أدوات ──
    def seed(self, kind, **kw):
        body = {"kind": kind}
        body.update(kw)
        return self.api.post("/qa/seed", body)

    def advance(self, oid, to):
        out = self.seed("order_advance", order_id=oid, target=to)
        if out.get("status") != to:
            raise Step("order_advance %s stopped at %s" % (to, out.get("status")))
        return out

    def money(self, oid):
        m = self.api.get("/qa/money?order_id=" + oid)
        drv = m.get("driver") or {}
        if "held" in drv:
            self.held = drv["held"]
        return m

    def order(self, qty=1, method="cash", scope=""):
        body = {"items": [{"menu_item_id": self.item, "qty": qty}],
                "address_text": "QA model2", "lat": QA_LAT, "lng": QA_LNG,
                "payment_method": method}
        if scope:
            body["wallet_scope"] = scope
        out = self.api.post("/orders", body, token=self.cust, key=True)
        if not out.get("id"):
            raise Step("order create returned no id: %s" % out)
        return out["id"]

    def custom_order(self):
        out = self.api.post("/orders/custom", {
            "request": "QA model2 purchase", "address_text": "QA model2",
            "lat": QA_LAT, "lng": QA_LNG, "payment_method": "cash"}, token=self.cust, key=True)
        oid = out.get("id")
        if not oid:
            raise Step("custom create returned no id: %s" % out)
        self.advance(oid, "assigned")
        return oid

    def confirm_quote(self, oid):
        o = self.money(oid)["order"]
        self.api.post("/orders/%s/confirm-quote" % oid, {
            "payment_method": "cash", "expected_total": o["quote_total"],
            "expected_quote_version": o["quote_version"]}, token=self.cust)

    def close_order(self, oid, why):
        """**يُغلق طلبَ QA مفتوحاً عبر الأبواب الحقيقيّة** — ويردّ `True` إن أُغلق.

        قبل الاستلام: إلغاءُ المكتب (ومن «عند المتجر» يُعاد للطابور أوّلاً ثمّ يُلغى).
        بعد الاستلام بالنموذج ٢: يُساق إلى «سُلّم» (`order_advance`) — **لا إلغاءَ بعد الاستلام.**
        وغيرُ ذلك (نموذج ١ بعد الاستلام، أو بيد سائقٍ آخر): **لا يُفرض شيء** — يُطبع ليُحلّ باليد.
        """
        if not oid:
            return True
        code, m = self.api.call("GET", "/qa/money?order_id=" + oid, ok=None)
        if code >= 400:
            print("    WARN order %s: read-back %s - left as is" % (oid, code))
            return False
        status, model = m["order"]["status"], m["order"]["money_model"]
        if status in TERMINAL:
            return True
        if status in PRE_PICKUP:
            if status == "at_pickup":
                self.api.call("POST", "/qa/orders/%s/transition" % oid,
                              {"to": "dispatching", "note": "QA model2 %s" % why}, ok=None)
            code, _ = self.api.call("POST", "/qa/orders/%s/transition" % oid,
                                    {"to": "cancelled", "note": "QA model2 %s" % why}, ok=None)
            if code < 300:
                return True
        elif model == 2:
            code, _ = self.api.call("POST", "/qa/seed", {"kind": "order_advance", "order_id": oid,
                                                         "target": "delivered"}, ok=None)
            if code < 300:
                return True
        print("    WARN order #%s (%s, model %s, %s) left open - resolve it from the staging dashboard"
              % (m["order"]["number"], oid, model, status))
        return False

    def abandon(self, oid):
        """**طلبٌ سقط سيناريوه في منتصفه.**"""
        self.close_order(oid, "abandon")

    def preflight_cleanup(self):
        """**طلباتُ زبون QA المفتوحة من تشغيلٍ سابق تُغلق أوّلاً** — من قائمته الحقيقيّة (`/my/orders`).

        **وإلّا**: سقفُ النقد عند الاستلام يعدّها، و`custom-agree` يرفض «أكثرَ من طلبٍ مفتوح».
        يردّ عددَ ما بقي مفتوحاً.
        """
        out = self.api.get("/my/orders?open_only=true&per_page=100", token=self.cust)
        ids = [o.get("id") for o in out.get("orders", []) if o.get("id")]
        left = sum(0 if self.close_order(i, "preflight cleanup") else 1 for i in ids)
        print("preflight: QA customer open orders %d, closed %d, left %d" % (len(ids), len(ids) - left, left))
        return left

    def whatsapp_order(self):
        """**طلبُ واتساب بأجرةٍ ضمنَ مدى التجهيز** — يُجرَّب ٣٠٠، وإن ردّ `wa_fee_out_of_range`
        فمنتصفُ المدى الذي أعاده الخادم (لا رقمٌ مفترَض)."""
        body = {"store_id": self.store, "recipient_phone": QA_RECIPIENT, "recipient_name": "QA",
                "address_text": "QA model2", "lat": QA_LAT, "lng": QA_LNG, "fee": 300}
        code, out = self.api.call("POST", "/qa/whatsapp-orders", body, key=True, ok=None)
        err = (out or {}).get("error") or {}
        if code == 422 and err.get("code") == "wa_fee_out_of_range":
            d = err.get("details") or {}
            lo, hi = int(d.get("min", 0)), int(d.get("max", 0))
            body["fee"] = (lo + hi) // 2 // 5 * 5 or lo
            print("    whatsapp fee range on target %s-%s, using %s" % (lo, hi, body["fee"]))
            code, out = self.api.call("POST", "/qa/whatsapp-orders", body, key=True, ok=None)
        if code >= 300:
            raise Step("POST /qa/whatsapp-orders -> %d %s" % (code, json.dumps(out, ensure_ascii=False)[:300]))
        oid = out.get("id") or (out.get("order") or {}).get("id")
        if not oid:
            raise Step("whatsapp create returned no id: %s" % out)
        return oid


def common(c, m, name):
    """**قواعدُ كلّ طلبٍ في النموذج ٢.**"""
    o, led = m["order"], m["ledger"]
    c(o["money_model"] == 2, "%s: money_model=%s, want 2" % (name, o["money_model"]))
    c(led["goods_rows"] == 0, "%s: goods ledger rows=%s (goods never hit the ledger)" % (name, led["goods_rows"]))
    c(m["office_cash"].get("advance_return", 0) == 0, "%s: drawer advance_return=%s" % (name, m["office_cash"]))
    v = m["invariants"]["violations"]
    c(m["invariants"]["checked"] > 0 and not v, "%s: invariant violations on this order: %s" % (name, v))


def row0(m):
    rows = m["settlements"]["rows"]
    return rows[0] if rows else {}


# ══════════════════════════════════════════════════════════════════════
# السيناريوهات
# ══════════════════════════════════════════════════════════════════════

def sc_standard(run, c, wallet):
    name = "standard_wallet" if wallet else "standard_cash"
    if wallet:
        run.seed("wallet_fund", value_int=5000)
        oid = run.order(1, "", "delivery")
    else:
        oid = run.order(1, "cash")
    run.oid = oid
    run.advance(oid, "assigned")
    m = run.money(oid)
    o = m["order"]
    # **سقفُ نقد الكابتن يعدّ مالَ المنصّة وحدَه** (B9).
    c(m["platform_cash"] == o["cash_due"] - STORE_SHARE,
      "%s: platform cash %s, want cash_due %s - store share %s" % (name, m["platform_cash"], o["cash_due"], STORE_SHARE))
    before_t = m["ledger"]["treasury_effect"]
    run.advance(oid, "picked_up")
    m = run.money(oid)
    r = row0(m)
    c(r.get("off_ledger") is True and r.get("state") == "cash_paid" and r.get("paid_via") == "driver"
      and r.get("amount") == STORE_SHARE and r.get("has_ledger_tx") is False,
      "%s: settlement row at pickup %s, want an off-ledger record paid by the driver %s" % (name, r, STORE_SHARE))
    c(m["ledger"]["treasury_effect"] == before_t,
      "%s: treasury moved at pickup %s -> %s" % (name, before_t, m["ledger"]["treasury_effect"]))
    run.advance(oid, "delivered")
    m = run.money(oid)
    o, led = m["order"], m["ledger"]
    box = m["driver_box"]["collection"]
    c(box == o["cash_due"] - STORE_SHARE, "%s: driver box %s, want cash_due %s - %s" % (name, box, o["cash_due"], STORE_SHARE))
    want = o["wallet_paid"] + box - o["driver_fee"]
    c(led["platform_profit"] == want, "%s: platform profit %s, want wallet+box-driver fee = %s" % (name, led["platform_profit"], want))
    c(o["platform_commission"] == COMMISSION, "%s: commission %s, want %s" % (name, o["platform_commission"], COMMISSION))
    if wallet:
        c(o["wallet_paid"] == o["fee_paid"], "%s: wallet paid %s, want the delivery fee only %s" % (name, o["wallet_paid"], o["fee_paid"]))
        c(led["by_kind"].get("order_payment", 0) == -o["fee_paid"],
          "%s: customer order_payment %s, want -%s" % (name, led["by_kind"].get("order_payment"), o["fee_paid"]))
        c(o["cash_due"] == o["total"] - o["fee_paid"], "%s: cash_due %s, want total - fee" % (name, o["cash_due"]))
    common(c, m, name)


def sc_failure(run, c):
    name = "failure_after_pickup"
    reasons = run.api.get("/driver/fail-reasons?at=at_dropoff", token=run.drv).get("reasons", [])
    code = next((x["code"] for x in reasons if x.get("kind") == "report" and x.get("code")), "")
    for variant in ("store", "office"):
        oid = run.order(1, "cash")
        run.oid = oid
        run.advance(oid, "at_dropoff")
        if code:
            run.api.post("/driver/orders/%s/report" % oid, {"code": code, "note": "QA model2"}, token=run.drv)
        run.api.post("/qa/orders/%s/door-resolution" % oid, {
            "action": "return_to_office", "fault": "customer", "customer_action": "none",
            "return_to": variant, "note": "QA model2 failure"})
        run.api.post("/driver/orders/%s/goods-handed" % oid, {}, token=run.drv)
        run.api.post("/qa/orders/%s/goods" % oid, {"to": "merchant" if variant == "store" else "platform"})
        m = run.money(oid)
        if variant == "store":
            c(row0(m).get("state") == "goods_returned", "%s/store: row state %s, want goods_returned" % (name, row0(m).get("state")))
        else:
            c(m["order"]["stuck_goods_amount"] == STORE_SHARE and row0(m).get("state") == "held_by_driver",
              "%s/office: stuck %s state %s, want %s held_by_driver" % (name, m["order"]["stuck_goods_amount"], row0(m).get("state"), STORE_SHARE))
            before = m["ledger"]["treasury_effect"]
            run.api.post("/qa/orders/%s/stuck-goods" % oid, {"decision": "customer"})
            m = run.money(oid)
            c(m["ledger"]["treasury_effect"] == before,
              "%s/office: 'customer' decision moved the treasury %s -> %s" % (name, before, m["ledger"]["treasury_effect"]))
        c(m["order"]["status"] == "failed", "%s/%s: status %s" % (name, variant, m["order"]["status"]))
        c(m["ledger"]["platform_profit"] >= 0, "%s/%s: platform loss %s" % (name, variant, m["ledger"]["platform_profit"]))
        c(not m["obligations"], "%s/%s: debts created %s" % (name, variant, m["obligations"]))
        common(c, m, "%s/%s" % (name, variant))
        run.oid = None


def sc_refund(run, c):
    name = "refund_after_delivery"
    oid = run.order(1, "cash")
    run.oid = oid
    run.advance(oid, "delivered")
    before = run.money(oid)["ledger"]
    run.api.post("/qa/orders/%s/transition" % oid, {"to": "refunded", "note": "QA model2 refund"})
    m = run.money(oid)
    led, o = m["ledger"], m["order"]
    refund = led["by_kind"].get("refund", 0)
    c(refund == o["fee_paid"] and refund > 0, "%s: refunded %s, want the delivery fee paid %s" % (name, refund, o["fee_paid"]))
    c(led["driver_net"] == before["driver_net"], "%s: driver ledger moved %s -> %s" % (name, before["driver_net"], led["driver_net"]))
    c(led["by_party"].get("merchant", 0) == 0 and not m["obligations"], "%s: store touched / debt %s" % (name, m["obligations"]))
    common(c, m, name)


def sc_custom(run, c):
    name = "custom"
    oid = run.custom_order()
    run.oid = oid
    run.api.post("/qa/custom-agree", {"goods": 300, "fee": 150})
    run.confirm_quote(oid)
    run.advance(oid, "delivered")
    m = run.money(oid)
    o, led = m["order"], m["ledger"]
    box = m["driver_box"]["collection"]
    c(o["custom_goods_amount"] == 300 and o["custom_fee"] == 150, "%s: quote %s/%s" % (name, o["custom_goods_amount"], o["custom_fee"]))
    want = o["wallet_paid"] + box - o["driver_fee"]
    c(led["platform_profit"] == want, "%s: platform profit %s, want %s" % (name, led["platform_profit"], want))
    common(c, m, name)


def sc_usd(run, c):
    name = "usd"
    oid = run.custom_order()
    run.oid = oid
    run.api.post("/qa/orders/%s/partner" % oid, {"merchant_id": run.store})
    run.api.post("/qa/custom-agree", {"goods": 0, "fee": 150, "goods_usd_cents": 1250})
    run.confirm_quote(oid)
    m = run.money(oid)
    c(m["platform_cash"] == m["order"]["cash_due"],
      "%s: platform cash %s, want SYP cash %s (USD goods not counted)" % (name, m["platform_cash"], m["order"]["cash_due"]))
    run.advance(oid, "delivered")
    m = run.money(oid)
    o = m["order"]
    c(o["custom_goods_usd"] == 1250 and o["goods_usd_syp"] == 1250,
      "%s: usd %s syp %s, want 1250 cents at rate 100" % (name, o["custom_goods_usd"], o["goods_usd_syp"]))
    c(o["platform_commission"] == 125, "%s: commission %s, want 125 (10%% of the SYP equivalent)" % (name, o["platform_commission"]))
    common(c, m, name)


def sc_whatsapp(run, c):
    name = "whatsapp"
    oid = run.whatsapp_order()
    run.oid = oid
    run.advance(oid, "at_pickup")
    run.api.post("/driver/orders/%s/fetch-amount" % oid, {"amount": 2400}, token=run.drv)
    run.advance(oid, "delivered")
    m = run.money(oid)
    o, led = m["order"], m["ledger"]
    box = m["driver_box"]["collection"]
    c(o["platform_commission"] == 240, "%s: commission %s, want 240" % (name, o["platform_commission"]))
    fee = o["fee_paid"]
    c(box == fee + 240, "%s: driver box %s, want fee %s + commission 240" % (name, box, fee))
    c(led["by_party"].get("merchant", 0) == 0 and not m["obligations"],
      "%s: store wallet moved %s / debt %s" % (name, led["by_party"].get("merchant"), m["obligations"]))
    common(c, m, name)
    # **والاسترجاعُ بعد التسليم لا شيءَ في التطبيق** (جوابُ المالك).
    run.api.post("/qa/orders/%s/transition" % oid, {"to": "refunded", "note": "QA model2 wa refund"})
    m2 = run.money(oid)
    c(m2["ledger"]["by_kind"].get("refund", 0) == 0 and m2["ledger"]["by_party"].get("merchant", 0) == 0,
      "%s: refund moved money %s" % (name, m2["ledger"]["by_kind"]))


def sc_edit(run, c):
    name = "edit_after_pickup"
    oid = run.order(2, "cash")
    run.oid = oid
    run.advance(oid, "picked_up")
    line = run.money(oid)["items"][0]["id"]
    req = run.api.post("/driver/orders/%s/edit-request" % oid, {
        "lines": [{"item_id": line, "action": "missing", "qty": 1}], "note": "QA model2 edit"}, token=run.drv, key=True)
    rid = (req.get("request") or {}).get("id")
    if not rid:
        raise Step("edit request returned no id: %s" % req)
    code, _ = run.api.call("POST", "/qa/seed", {"kind": "order_advance", "order_id": oid, "target": "delivered"}, ok=None)
    c(code == 409, "%s: delivery while the edit request is pending returned %s, want 409" % (name, code))
    run.api.post("/qa/orders/%s/edit-request/approve" % oid, {"request_id": rid})
    run.advance(oid, "delivered")
    m = run.money(oid)
    s = m["settlements"]
    c(s["adjustments"] == 1 and s["adjustments_with_ledger_tx"] == 0,
      "%s: adjustments %s (with ledger tx %s), want 1 without ledger" % (name, s["adjustments"], s["adjustments_with_ledger_tx"]))
    box = m["driver_box"]["collection"]
    c(box == m["order"]["cash_due"] - STORE_SHARE,
      "%s: driver box %s, want cash_due after edit %s - %s (paid 18000, took back 9000)" % (name, box, m["order"]["cash_due"], STORE_SHARE))
    common(c, m, name)


RUNNERS = {
    "standard_cash": lambda run, c: sc_standard(run, c, False),
    "standard_wallet": lambda run, c: sc_standard(run, c, True),
    "failure_after_pickup": sc_failure,
    "refund_after_delivery": sc_refund,
    "custom": sc_custom,
    "whatsapp": sc_whatsapp,
    "usd": sc_usd,
    "edit_after_pickup": sc_edit,
}


def main():
    ap = argparse.ArgumentParser(description="model 2 money witness on staging (QA bridge only)")
    ap.add_argument("--keep-on", action="store_true", help="leave finance.goods_outside ON at the end")
    ap.add_argument("--no-settle", action="store_true", help="do not hand the QA driver box to the office at the end")
    ap.add_argument("--only", default="", help="comma list of scenarios (default: all)")
    # **تجربةُ السكربت نفسِه على محرّكٍ محلّيٍّ بهويّة التجهيز** (APP_ENV=staging و RAHALGO_STAGING=1)
    # — منفذٌ على 127.0.0.1 وحدَه، فلا يصير بابَ تجاوزٍ إلى غير التجهيز.
    ap.add_argument("--local", type=int, default=0, metavar="PORT",
                    help="run against http://127.0.0.1:PORT (a local API started in staging mode)")
    args = ap.parse_args()

    chosen = [s.strip() for s in args.only.split(",") if s.strip()] or SCENARIOS
    bad = [s for s in chosen if s not in RUNNERS]
    if bad:
        raise SystemExit("unknown scenario(s): %s — known: %s" % (", ".join(bad), ", ".join(SCENARIOS)))

    if args.local:
        base = "http://127.0.0.1:%d/api/v1" % args.local
        print("target=local  url=%s" % base)
    else:
        print(target.banner())
        base = target.base_url("/api/v1")
    api = Api(base)
    run = Run(api)
    results = []
    previous = None
    shift = None
    fee_policy = None
    cod = None
    try:
        sw = api.post("/qa/setting", {"key": MODEL2_KEY, "value": True})
        previous = bool(sw.get("previous"))
        print("switch %s: previous=%s -> ON" % (MODEL2_KEY, previous))
        run.cust = api.post("/qa/session", {})["tokens"]["access_token"]
        run.drv = api.post("/qa/driver-session", {})["tokens"]["access_token"]
        st = run.seed("m2_store_setup")
        run.item, run.store, run.driver_id = st["item_id"], st["merchant_id"], st["driver_id"]
        print("QA model-2 store %s item %s (open 24h, commission 10%%)" % (run.store, run.item))
        shift = run.seed("driver_shift", value_bool=True)
        cod = run.seed("cod_limit", value_int=COD_LIMIT_FOR_RUN)
        print("customers.cod_limit: previous=%s -> %s (restored at the end)" % (cod.get("previous"), COD_LIMIT_FOR_RUN))
        leftover = run.preflight_cleanup()
        if any(s in chosen for s in ("custom", "usd")):
            pol = api.post("/qa/custom-fee-policy", {"source": "driver_defined", "fee": 0, "driver_may_change": False})
            fee_policy = pol.get("previous")
    except Step as e:
        print("FAIL  setup: %s" % e)
        cleanup(api, run, args, previous, shift, fee_policy, cod)
        sys.exit(1)
    try:
        for name in chosen:
            fails = []
            run.oid = None

            def c(cond, msg, fails=fails):
                if not cond:
                    fails.append(msg)
            try:
                if name in ("custom", "usd") and leftover:
                    raise Step("%d QA customer order(s) still open from before - custom-agree needs exactly one; "
                               "close them from the staging dashboard and re-run" % leftover)
                RUNNERS[name](run, c)
            except Step as e:
                fails.append("step refused: %s" % e)
                run.abandon(run.oid)
            except (KeyError, TypeError, IndexError) as e:
                fails.append("unexpected read-back shape: %r" % (e,))
                run.abandon(run.oid)
            results.append((name, fails))
            print(("PASS  " if not fails else "FAIL  ") + name)
            for f in fails:
                print("        - " + f)
    finally:
        cleanup(api, run, args, previous, shift, fee_policy, cod)

    failed = [n for n, f in results if f]
    print("")
    print("%d/%d scenarios passed" % (len(results) - len(failed), len(chosen)))
    sys.exit(1 if failed or len(results) != len(chosen) else 0)


def cleanup(api, run, args, previous, shift, fee_policy, cod=None):
    """**الخاتمةُ دائماً** — وكلُّ خطوةٍ وحدَها، فسقوطُ واحدةٍ لا يترك الباقيات."""
    def soft(label, fn):
        try:
            fn()
        except Exception as e:  # noqa: BLE001
            print("    WARN cleanup %s: %s" % (label, e))

    if not args.no_settle and run.driver_id and run.held:
        soft("settle QA driver box", lambda: api.post(
            "/qa/drivers/%s/settle" % run.driver_id,
            {"amount": run.held, "note": "QA model2 witness - box handed to office"}, key=True))
    soft("drain QA wallet", lambda: run.seed("wallet_drain"))
    if cod is not None:
        soft("restore customers.cod_limit", lambda: run.seed("cod_limit", value_int=int(cod.get("previous") or 0)))
    if fee_policy:
        soft("restore custom fee policy", lambda: api.post("/qa/custom-fee-policy", {
            "source": fee_policy.get("source") or "driver_defined", "fee": fee_policy.get("fee") or 0,
            "driver_may_change": bool(fee_policy.get("driver_may_change"))}))
    if shift is not None:
        soft("QA driver shift off", lambda: run.seed("driver_shift", value_bool=False,
                                                     value_str=shift.get("previous_mode") or ""))
    if previous is not None:
        if args.keep_on:
            print("switch %s left ON (--keep-on); previous was %s" % (MODEL2_KEY, previous))
        else:
            soft("restore switch", lambda: api.post("/qa/setting", {"key": MODEL2_KEY, "value": previous}))
            print("switch %s restored to %s" % (MODEL2_KEY, previous))


if __name__ == "__main__":
    main()
