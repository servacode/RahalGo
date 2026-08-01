# -*- coding: utf-8 -*-
"""تكملة الدورة: من الطلب إلى المال إلى الأثر. تُستورد من e2e.py."""
import json


def run(ctx):
    (call, login, phase, step, ok, bad, check, state, admin, rep_tok,
     own_tok, cust_tok, S_MAP, sql) = ctx

    COMMISSION = S_MAP["merchants.default_commission_percent"]
    REP_PCT = S_MAP["sales.commission_percent"]
    ACTIVATION = S_MAP["sales.activation_orders"]
    DRV_PCT = S_MAP["drivers.share_percent"]
    MIN_PAYOUT = S_MAP["payouts.min_amount"]
    DRV_PHONE, DRV_PW = state["drv_phone"], state["drv_pw"]

    # ═════════════════════════════════════════════════════════════════════
    phase("المرحلة ٥ — دورة الطلب الأولى: من المطبخ إلى الباب")

    step("المتجر يرى الطلب في بوابته ويقبله")
    _, mo = call("GET", f"/merchant/stores/{state['merchant_id']}/orders", own_tok, expect=200)
    rows = mo if isinstance(mo, list) else mo.get("orders", mo.get("items", []))
    check(any(x["id"] == state["order_id"] for x in rows),
          f"الطلب #{state['order_no']} في بوابة المتجر", "الطلب لم يصل بوابة المتجر")
    call("POST", f"/merchant/orders/{state['order_id']}/transition", own_tok,
         {"to": "accepted"}, expect=200)
    ok("قُبل الطلب — وبدأ عدّاد وقت التحضير")

    step("الوقت المتوقّع صار معلوماً للزبون")
    _, o = call("GET", f"/my/orders/{state['order_id']}", cust_tok, expect=200)
    check(o.get("accepted_at") and o.get("prep_minutes"),
          f"الزبون يرى: قُبل في {str(o.get('accepted_at'))[11:19]} · تحضير {o.get('prep_minutes')} دقيقة "
          f"— «قيد التحضير» وحدها لا تقول عشر دقائق أم ساعة",
          "لا وقت متوقّع للزبون")
    check(o.get("prep_minutes") == 15,
          "وقتُ التحضير من إعدادات هذا المتجر (١٥) لا من افتراض المنصة",
          f"وقت التحضير {o.get('prep_minutes')} لا ١٥")

    step("المتجر يبدأ التحضير ثم يعلن الجاهزية")
    call("POST", f"/merchant/orders/{state['order_id']}/transition", own_tok,
         {"to": "preparing"}, expect=200)
    call("POST", f"/merchant/orders/{state['order_id']}/ready", own_tok, {}, expect=200)
    r = sql(f"SELECT status, ready_at IS NOT NULL FROM orders WHERE id = '{state['order_id']}'")
    check(r[0][1] == "t",
          f"الجاهزية طابعٌ زمنيّ لا حالة — الحالة بقيت «{r[0][0]}» وسُجّل وقتُ الجاهزية",
          "الجاهزية لم تُسجَّل")

    step("العمليات تطلب سائقاً")
    call("POST", f"/admin/orders/{state['order_id']}/transition", admin,
         {"to": "dispatching"}, expect=200)
    ok("الطلب في الطابور")

    step("السائق يغيّر كلمته ثم يبدأ دوامه")
    drv_tok = login(DRV_PHONE, state["drv_temp_pw"])
    call("POST", "/auth/password", drv_tok,
         {"current_password": state["drv_temp_pw"], "password": DRV_PW}, expect=200)
    drv_tok = login(DRV_PHONE, DRV_PW)
    _, me = call("GET", "/driver/me", drv_tok, expect=200)
    check(me["on_shift"] is False, "السائق خارج الدوام — التوفّر علَمٌ يرفعه هو", "دوامه مرفوع تلقائياً")

    step("الطابور محجوبٌ عمّن لم يبدأ دوامه")
    sc, d = call("POST", f"/driver/orders/{state['order_id']}/accept", drv_tok)
    check((d.get("error") or {}).get("code") == "not_on_shift",
          "رُدّ عليه: ابدأ دوامك أولاً", f"قُبل الأخذ بلا دوام: {sc}")

    call("POST", "/driver/shift", drv_tok, {"on": True}, expect=200)
    _, q = call("GET", "/driver/queue", drv_tok, expect=200)
    mine = [x for x in q if x["id"] == state["order_id"]]
    check(bool(mine),
          f"الطلب في طابوره: {mine[0]['merchant_name']} → {mine[0]['customer_name']} · "
          f"يقبض {mine[0]['cash_due']:,} نقداً" if mine else "",
          "الطلب ليس في الطابور")

    step("السائق يأخذ الطلب")
    call("POST", f"/driver/orders/{state['order_id']}/accept", drv_tok, expect=200)
    r = sql(f"SELECT status, driver_id IS NOT NULL FROM orders WHERE id = '{state['order_id']}'")
    check(r[0][0] == "assigned" and r[0][1] == "t", "أُسند إليه وصار في مهامّه", f"الحالة {r[0]}")

    step("الطلب خرج من الطابور فوراً — لا يأخذه اثنان")
    _, q2 = call("GET", "/driver/queue", drv_tok, expect=200)
    check(not any(x["id"] == state["order_id"] for x in q2),
          "لم يعد معروضاً لأحد", "ما زال في الطابور بعد أخذه")

    step("رحلة السائق بأربع ضغطات")
    for to, label in [("at_pickup", "وصل المتجر"), ("picked_up", "استلم الطلب"),
                      ("on_the_way", "انطلق"), ("at_dropoff", "وصل الزبون")]:
        call("POST", f"/driver/orders/{state['order_id']}/transition", drv_tok,
             {"to": to}, expect=200)
        ok(label)

    step("إنهاء الدوام وبيده طلب — مرفوض")
    sc, d = call("POST", "/driver/shift", drv_tok, {"on": False})
    check((d.get("error") or {}).get("code") == "has_active_orders",
          "من بدأ يُنهي — لا يُترك زبونٌ معلّقاً", "سُمح بإنهاء الدوام والطلب جارٍ")

    step("التسليم")
    call("POST", f"/driver/orders/{state['order_id']}/transition", drv_tok,
         {"to": "delivered"}, expect=200)
    ok(f"سُلّم الطلب #{state['order_no']}")

    # ═════════════════════════════════════════════════════════════════════
    phase("المرحلة ٦ — المال: أين ذهبت كل ليرة")

    sub, fee, total = state["price"], state["delivery_fee"], state["total"]
    exp_comm = sub * COMMISSION // 100
    exp_merch = sub - exp_comm
    exp_drv = fee * DRV_PCT // 100
    exp_net = exp_comm + fee - exp_drv

    step("قيود الطلب في الدفتر")
    rows = sql(f"""SELECT u.full_name, t.kind, t.amount FROM wallet_transactions t
                   JOIN users u ON u.id = t.user_id
                   WHERE t.ref = '{state['order_id']}' ORDER BY t.kind""")
    got = {r[1]: int(r[2]) for r in rows}
    for r in rows:
        ok(f"{r[0]:<28} {r[1]:<18} {int(r[2]):>10,}")

    check(got.get("merchant_earning") == exp_merch,
          f"المتجر {exp_merch:,} = بضاعة {sub:,} − عمولة {COMMISSION}٪ ({exp_comm:,}) "
          f"— **رسم التوصيل ليس منه**",
          f"مستحقّ المتجر {got.get('merchant_earning')} لا {exp_merch}")
    check(got.get("driver_earning") == exp_drv,
          f"السائق {exp_drv:,} = {DRV_PCT}٪ من رسم التوصيل {fee:,} — أجرُ توصيلٍ لا حصةٌ من بيع",
          f"أجر السائق {got.get('driver_earning')} لا {exp_drv}")
    check("commission" not in got,
          f"**لا عمولة للمندوب بعد** — عتبة التفعيل {ACTIVATION} طلبات، وهذا الأول. "
          f"تمنع تسجيل متاجر لا تعمل",
          "قُيّدت عمولة المندوب قبل بلوغ عتبة التفعيل")

    step("ذمّة السائق النقدية")
    r = sql(f"""SELECT COALESCE(held,0) FROM driver_cash_boxes
                WHERE driver_id = '{state['driver_id']}'""")
    held = int(r[0][0]) if r else 0
    check(held == total,
          f"بذمّته {held:,} — **المبلغ كاملاً** يسلّمه للمكتب، وأجره {exp_drv:,} "
          f"مستقلٌّ في محفظته. دفتران لا يلتقيان",
          f"ذمّته {held} لا {total}")

    step("صافي المنصة من هذا الطلب")
    ok(f"عمولة {exp_comm:,} + توصيل {fee:,} − أجر السائق {exp_drv:,} = **{exp_net:,}**")

    step("الزبون يقيّم")
    call("POST", f"/orders/{state['order_id']}/rating", cust_tok,
         {"merchant_stars": 5, "driver_stars": 5, "comment": "وصل ساخناً وبسرعة"}, expect=201)
    r = sql(f"""SELECT merchant_stars, driver_stars FROM order_ratings
                WHERE order_id = '{state['order_id']}'""")
    check(bool(r) and r[0][0] == "5" and r[0][1] == "5",
          "قُيّد تقييمان: للمتجر وللسائق — طرفان مختلفان في خدمةٍ واحدة",
          f"التقييم لم يُقيَّد: {r}")
    sc, d = call("POST", f"/orders/{state['order_id']}/rating", cust_tok,
                 {"merchant_stars": 1})
    check((d.get("error") or {}).get("code") == "already_rated",
          "تقييمٌ واحد لكل طلب — ولا يُعاد", "قُبل تقييمٌ ثانٍ")

    # ═════════════════════════════════════════════════════════════════════
    phase("المرحلة ٧ — عتبة تفعيل المندوب: أربعة طلبات أخرى")

    step(f"تشغيل {ACTIVATION - 1} طلبات إضافية حتى تُفتح عمولة المندوب")
    for i in range(2, ACTIVATION + 1):
        oid = full_cycle(ctx, drv_tok, i)
        state[f"order_{i}"] = oid
    ok(f"صار للمتجر {ACTIVATION} طلبات مُسلَّمة")

    step("عمولة المندوب فُتحت عند بلوغ العتبة")
    rows = sql(f"""SELECT count(*), COALESCE(sum(amount),0) FROM wallet_transactions
                   WHERE user_id = '{state['rep_id']}' AND kind = 'commission'""")
    cnt, tot = int(rows[0][0]), int(rows[0][1])
    exp_rep = exp_comm * REP_PCT // 100
    check(cnt == 1 and tot == exp_rep,
          f"قيدٌ واحد بـ{tot:,} = {REP_PCT}٪ من عمولة المنصة ({exp_comm:,}) — "
          f"**عن الطلب الخامس وحده**، لا بأثرٍ رجعيّ على الأربعة",
          f"قيود المندوب: {cnt} بمجموع {tot:,} (توقّعنا 1 بـ{exp_rep:,})")

    # ═════════════════════════════════════════════════════════════════════
    phase("المرحلة ٨ — المالية: السحب والتسوية")

    step("السائق يطلب سحباً دون الحدّ")
    _, dm = call("GET", "/driver/me", drv_tok, expect=200)
    bal = dm["balance"]
    ok(f"رصيده {bal:,} · نقدٌ بذمّته {dm['cash_held']:,} من سقف {dm['cash_limit']:,}")
    sc, d = call("POST", "/me/payouts", drv_tok, {"amount": 1000, "note": ""})
    check((d.get("error") or {}).get("code") == "payout_below_min",
          f"رُفض: دون الحدّ الأدنى {MIN_PAYOUT:,}", "قُبل طلبٌ دون الحدّ")

    step("السائق يسحب رصيده كلَّه — الاستثناء يعمل")
    _, pr = call("POST", "/me/payouts", drv_tok, {"amount": bal, "note": ""}, expect=201)
    state["payout_id"] = pr["id"]
    ok(f"قُبل طلب سحب {bal:,} — من بقي له مبلغٌ صغير لا يُحبس عنه")

    step("المالية تصرف الطلب")
    fin = login("+963955444555", "Finance@2026")
    call("POST", f"/admin/payouts/{state['payout_id']}/decide", fin,
         {"status": "paid", "decision": "سُلّم نقداً في المكتب"}, expect=200)
    _, dm2 = call("GET", "/driver/me", drv_tok, expect=200)
    check(dm2["balance"] == 0,
          f"صار رصيده {dm2['balance']:,} — والقيد `payout` في دفتره",
          f"رصيده {dm2['balance']} بعد الصرف")

    step("المالية تسوّي صندوق السائق")
    call("POST", f"/admin/drivers/{state['driver_id']}/settle", fin,
         {"amount": dm2["cash_held"], "note": "تسليم حصيلة اليوم"}, expect=200)
    _, dm3 = call("GET", "/driver/me", drv_tok, expect=200)
    check(dm3["cash_held"] == 0, "خلَت ذمّته النقدية", f"بقي {dm3['cash_held']}")

    step("الآن يستطيع إنهاء دوامه")
    call("POST", "/driver/shift", drv_tok, {"on": False}, expect=200)
    ok("انتهى الدوام")

    # ═════════════════════════════════════════════════════════════════════
    phase("المرحلة ٩ — الأثر: ماذا بقي بعد انصراف الجميع")

    step("سجلّ الأحداث يحمل كل فعلٍ ماليّ")
    _, aud = call("GET", "/admin/audit?prefix=finance&limit=20", fin, expect=200)
    acts = [a["action"] for a in aud]
    for want, label in [("finance.payout_decide", "صرف طلب السحب"),
                        ("finance.driver_settle", "تسوية صندوق السائق")]:
        check(want in acts, f"مُسجَّل: {label}", f"غير مُسجَّل: {label}")
    for a in aud[:4]:
        ok(f"{a['action']:<24} {a.get('actor_name') or '—':<14} "
           f"{json.dumps(a.get('details'), ensure_ascii=False)[:70]}")

    step("العمليات محجوبة عن السجلّ المالي")
    ops = login("+963955333444", "Ops@2026")
    sc, d = call("GET", "/admin/audit", ops)
    check(sc == 403, "مُنعت — ليست طرفاً في المال", f"وصلت إلى السجلّ ({sc})")

    step("لوحة القيادة تعكس اليوم")
    _, st = call("GET", "/admin/stats", admin, expect=200)
    ok(f"الآن  : جارية {st['orders_open']} · عالق {st['orders_stuck']} · "
       f"على الدوام {st['drivers_on_shift']} · متاجر مفتوحة {st['merchants_open']}")
    ok(f"اليوم : طلبات {st['orders_today']} · سُلّم {st['delivered_today']} · "
       f"مبيعات {st['sales_today']:,} · **صافي المنصة {st['net_today']:,}** · "
       f"نقدٌ بالذمم {st['cash_held_total']:,}")
    check(st["delivered_today"] >= ACTIVATION,
          f"الطلبات الخمسة ظاهرةٌ في حصيلة اليوم", "الحصيلة لا تعكس الطلبات")

    step("المندوب يرى عميله وعمولته")
    _, rm = call("GET", "/rep/merchants", rep_tok, expect=200)
    rr = rm if isinstance(rm, list) else rm.get("items", [])
    mine = [x for x in rr if x["id"] == state["merchant_id"]]
    if mine:
        ok(f"عميله «{mine[0]['name']}»: {json.dumps(mine[0], ensure_ascii=False)[:170]}")
    _, rw = call("GET", "/rep/wallet", rep_tok, expect=200)
    ok(f"محفظته: رصيد {rw.get('balance', 0):,}")

    step("صاحب المتجر يرى مستحقّه")
    _, mw = call("GET", "/my/wallet", own_tok, expect=200)
    check(mw.get("balance") == exp_merch * ACTIVATION,
          f"رصيده {mw.get('balance'):,} = {ACTIVATION} × {exp_merch:,}",
          f"رصيده {mw.get('balance')} لا {exp_merch * ACTIVATION}")

    return exp_net * ACTIVATION


def full_cycle(ctx, drv_tok, n):
    """دورةٌ كاملة مختصرة لطلبٍ إضافي — لبلوغ عتبة التفعيل."""
    (call, login, phase, step, ok, bad, check, state, admin, rep_tok,
     own_tok, cust_tok, S_MAP, sql) = ctx
    _, o = call("POST", "/orders", cust_tok, {
        "merchant_id": state["merchant_id"],
        "items": [{"menu_item_id": state["item_id"], "qty": 1}],
        "address_text": "شارع تل أبيض — خلف الجامع، الطابق الثاني",
        "lat": 35.9528, "lng": 39.0079, "payment_method": "cash",
    }, expect=201)
    oid = o["id"]
    call("POST", f"/merchant/orders/{oid}/transition", own_tok, {"to": "accepted"}, expect=200)
    call("POST", f"/merchant/orders/{oid}/transition", own_tok, {"to": "preparing"}, expect=200)
    call("POST", f"/admin/orders/{oid}/transition", admin, {"to": "dispatching"}, expect=200)
    call("POST", f"/driver/orders/{oid}/accept", drv_tok, expect=200)
    for to in ("at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered"):
        call("POST", f"/driver/orders/{oid}/transition", drv_tok, {"to": to}, expect=200)
    ok(f"الطلب #{o['number']} — دورةٌ كاملة")
    return oid
