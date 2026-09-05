# مخطّطُ حقيقةِ الاختبار

> §١ · §٤٤ · §٥٤ من طلب المالك.

---

# القاعدةُ الأولى — **لا حقيقةَ ثانية**

**`TEST_TRUTH` ليس مستنداً يُكتب بيد** — **بل ملفٌّ يُولَّد من مصادرَ قائمة،
وحارسٌ يُسقط البناءَ إن شاخ.** **وهو النمطُ نفسُه الذي يحمي `TRUTH.md` اليومَ
بـ`TestTruthDocIsCurrent`.**

## من أين يُولَّد كلُّ حقل

| الحقل | المصدر | كيف |
|---|---|---|
| `ID` | **يدويّ** — `F-01`…`F-34` · `A-*` · `S-*` | **المفاتيحُ وحدَها تُكتب** |
| `OWNER APP` · `SCREEN` · `ACTION` | **وثائقُ إغلاق التطبيقات الخمسة** | **جدولٌ مصدرُه الوثيقة** |
| `API` | **`cmd/apidoc`** | **مولَّدٌ من الموجّه** |
| `BACKEND` | **تحليلُ `go/ast` للمعالج** | **مولَّد** |
| `DB TABLES` | **استخراجُ SQL من جسم المعالج** | **مولَّدٌ تقريبيّاً · ويُراجَع** |
| `STATE EFFECT` | **`ORDER_TRANSITIONS_55.md`** | **مولَّدٌ أصلاً من `statuses.go`** |
| `MONEY EFFECT` | **نداءاتُ `ApplyTx` و`kind`** | **مولَّد** |
| `NOTIFICATION EFFECT` | **جردُ `s.notify.*` بمطابقة الأقواس** | **مولَّد — ٤٩ موضعاً** |
| `REALTIME EFFECT` | **جردُ `pub.Publish` والغرف** | **مولَّد** |
| `CROSS-APP EFFECT` | **كتالوجُ التدفّقات ٣٤** | **من الوثيقة** |
| `KNOWN DEFECTS` · `RISKS` | **`FINAL_STATIC_CLOSEOUT.md`** | **مولَّدٌ من جداولها** |
| `CONTRACT GAPS` | **`CROSS_SYSTEM_FINAL_PRODUCT_DECISIONS.md`** | **من الوثيقة** |
| `REQUIRED TEST TYPES` | **قواعدُ اشتقاقٍ** (أدناه) | **مولَّد** |
| `REQUIRED EVIDENCE` | **معيارُ الدليل** (§٩) | **مولَّدٌ من نوع الاختبار** |
| `RELEASE BLOCKING LEVEL` | **نموذجُ الشدّة** (§١٢) | **مولَّدٌ من الأثر** |

**وثمانيةٌ من خمسةَ عشرَ مولَّدةٌ من الشيفرة مباشرةً** — **فلا تشيخ.**

---

# قواعدُ اشتقاقِ أنواع الاختبار

**تُطبَّق آليّاً على كلّ مدخلة** — **فلا يُنسى نوعٌ لأنّ كاتبَه سها:**

```
MONEY EFFECT ≠ لا          ⇒  + FINANCIAL INVARIANT (L3)
STATE EFFECT ≠ لا          ⇒  + STATE MACHINE (L2/L3)
CROSS-APP EFFECT ≠ لا      ⇒  + END-TO-END (L8)
REALTIME EFFECT ≠ لا       ⇒  + REALTIME CONTRACT (L5)
NOTIFICATION EFFECT ≠ لا   ⇒  + NOTIFICATION MATRIX (L5)
يحمل بياناتِ طرفٍ آخر      ⇒  + PRIVACY CONTRACT (L5) ← ملزِمٌ بعد D20/D21
مُحوِّلٌ (POST/PATCH/DELETE) ⇒  + AUTHZ (L11) + VALIDATION (L4)
في خريطة السباق العشرة     ⇒  + CONCURRENCY (L9)
في خريطة الفشل التسعة      ⇒  + FAILURE INJECTION (L9)
محميٌّ بمفتاح تكرار         ⇒  + IDEMPOTENCY (L4)
يمسّ الجهاز                 ⇒  + INSTRUMENTED (L7)
يمسّ الملاحة                ⇒  + REPLAY (L7) + REAL DEVICE (L7) ← الاثنان معاً
```

---

# شكلُ المدخلة

```yaml
id: F-14
title: تسويةُ التسليم
owner_app: [driver, engine]
screen:  driver/TripScreen
action:  delivered
api:     POST /api/v1/driver/orders/{id}/status
backend: orders.Transition → settle
db_tables: [orders, order_events, wallet_transactions, wallets, cashbox_entries]
state_effect: [on_the_way→at_dropoff, at_dropoff→delivered]
money_effect: [merchant_earning, driver_earning, commission, treasury]
notification_effect: [customer(AppCustomer), merchant(AppMerchant), rep(AppRep)]
realtime_effect: [ops, merchant:*, customer:*, driver:*]
cross_app_effect: [customer, merchant, driver, rep, admin]
known_defects: [D20, D21]
known_risks:   [R4, R23]
contract_gaps: [XG-10, XG-25, XG-26]
required_tests:
  - L3:financial_invariant
  - L3:state_machine
  - L5:privacy_contract
  - L5:notification_matrix
  - L8:end_to_end
  - L9:failure_injection
required_evidence: [db_state, ledger_state, notification_capture, ws_capture]
release_blocking: BLOCKER
```

---

# الملفّاتُ المولَّدة

```
docs/testing/system/
  TEST_TRUTH.yaml            ← مولَّد · لا يُحرَّر بيد
  FLOW_TEST_MATRIX.md        ← مولَّد من TEST_TRUTH
  DEFECT_REGRESSION_MATRIX.md
  RISK_VALIDATION_MATRIX.md
  CONTRACT_GAP_MATRIX.md
  SETTING_TEST_IMPACT.md
  IMPACT_GRAPH.json          ← §١٠
```

**والحارسُ**: `TestTestTruthIsCurrent` — **يعيد التوليدَ ويقارن، ويُسقط
البناءَ عند الفرق.** **وهو النمطُ العاملُ اليومَ لـ`TRUTH.md`.**

---

# ٤٤ · أثرُ الإعدادات

**٩٥ مفتاحاً مُغيِّراً للسلوك من ١١٨.**
**ولكلٍّ منها في `TEST_TRUTH`:**

```yaml
key: sales.commission_percent
group: sales
default: 0
sensitive: true
readers: [pricing.RepCommission]
impacted_flows: [F-14, F-23, F-15]
impacted_apps:  [rep, admin]
money: true
tests:
  - L1:pricing_rep_commission
  - L3:rep_commission_matrix     ← §٨
  - L3:financial_snapshot        ← §٩ · XQ-2
risk: HIGH
runtime_test_required: true
```

## والحارسُ الجديد

```
مفتاحٌ جديدٌ في catalog.go بلا مدخلةٍ في TEST_TRUTH  ⇒  البناءُ يسقط
```

**فلا يدخل إعدادٌ إلى المنصّة بلا أن يقول ما الذي يمسّه.**
**وهذا يحرس نمطاً وقع فعلاً**: **٩٥ مفتاحاً تغيّر السلوكَ ولا شيءَ يعرف
أيَّها يمسّ المال.**
