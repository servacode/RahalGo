# حقيقةُ الاختبار — التقرير

> **مولَّدٌ آليّاً** — `go run ./cmd/testtruth`.
> **ولا يُحرَّر بيد**، وحارسُ `TestTruthIsCurrent` يُسقط البناءَ إن شاخ.
> **والمصدرُ للآلة** [`TEST_TRUTH.json`](TEST_TRUTH.json).

---

# ١ · ما استُخرج من الشيفرة

**ولا رقمَ منه مكتوبٌ بيد.**

| ما هو | العدد |
|---|---|
| أبوابٌ في الموجّه | **291** |
| انتقالاتُ الطلب | **55** |
| أنواعُ قيدِ المحفظة | **13** — `adjustment` · `commission` · `compensation` · `driver_earning` · `merchant_earning` · `order_payment` · `payout` · `penalty` · `platform_expense` · `platform_profit` · `refund` · `reward` · `topup` |
| مواضعُ الإشعار | **49** — منها **11** موجَّهٌ بـ`Apps` |
| غرفُ البثّ | **5** — `customer` · `driver` · `merchant` · `ops` · `user` |
| تعريفاتُ الإعدادات | **118** — منها **95** مُغيِّرٌ للسلوك |
| حقولُ الطلب | **75** — من عقد `P-1` |
| ملفّاتُ اختبار | **202** |
| دوالُّ اختبار | **605** |

---

# ٢ · الاختبارات

```
TOTAL      = 605
MAPPED     = 12
INFRA      = 24
ORPHAN     = 569
```

**واليتيمُ اختبارٌ لا يعرف ماذا يحرس** — **ولا يُسقط البناءَ اليومَ**،
**ويُربَط مرحلةً بعد مرحلة.** وأكثرُها في:

| الحزمة | يتيمٌ |
|---|---|
| `qa` | 121 |
| `server` | 116 |
| `routing` | 76 |
| `orders_test` | 61 |
| `orders` | 54 |
| `identity` | 22 |
| `settings` | 14 |
| `notify` | 13 |
| `push` | 10 |
| `catalog` | 9 |

---

# ٣ · التغطية

| السجلّ | العدد | مربوطٌ | بلا اختبار |
|---|---|---|---|
| **التدفّقات** | 34 | 9 | 25 |
| **العيوب** | 23 | 6 | 17 |
| **المخاطر** | 24 | 1 | 23 |
| **فجواتُ العقد** | 26 | 0 | 26 |
| **إعداداتُ السلوك** | 95 | 0 | 95 |

---

# ٤ · العيوبُ وحارسُها

| ID | العنوان | الحال | الاختبارات |
|---|---|---|---|
| **D1** | فكُّ الإسناد بلا حدثٍ في order_events | `NO_REGRESSION_TEST_YET` | — |
| **D2** | convertLead بلا معاملةٍ واحدة · ٣ كتاباتٍ خطؤه… | `NO_REGRESSION_TEST_YET` | — |
| **D3** | «تذكّرني» تنقلب دائمةً بعد أوّل تجديد | `NO_REGRESSION_TEST_YET` | — |
| **D4** | سقفُ المفتوح داخلَ بوّابة واتساب | `NO_REGRESSION_TEST_YET` | — |
| **D5** | المصروفُ والخزينةُ كتابتان بلا معاملة | `NO_REGRESSION_TEST_YET` | — |
| **D6** | الطلبُ الخاصُّ لا ينادي cashBlocked | `NO_REGRESSION_TEST_YET` | — |
| **D7** | سقفُ النقد يقيس المحصَّل لا المكشوف | `EXPECTED_FAIL` | `TestSampleFactory_DriverOnShiftWithCash` |
| **D8** | الطلبُ الخاصُّ لا يطلب توثيقَ واتساب | `NO_REGRESSION_TEST_YET` | — |
| **D9** | الطلبُ الخاصُّ بلا حدثِ ''→pending | `NO_REGRESSION_TEST_YET` | — |
| **D10** | الاستعادةُ تُبطل نوعَ عميلٍ واحد | `NO_REGRESSION_TEST_YET` | — |
| **D11** | force_password_change بلا بوّابةٍ في أندرويد | `NO_REGRESSION_TEST_YET` | — |
| **D12** | Push.unregister بلا منادٍ | `NO_REGRESSION_TEST_YET` | — |
| **D13** | سردُ /media/ مفتوحٌ — وإثباتُ التسليم فيه | `NO_REGRESSION_TEST_YET` | — |
| **D14** | handleWS لا يفحص ActiveStatus | `EXPECTED_FAIL` | `TestSampleFactory_SuspendedIsRefused` |
| **D15** | AdminCreateUser في خطوتين | `NO_REGRESSION_TEST_YET` | — |
| **D16** | طابورُ المواقع ملفٌّ بلا صاحب | `NO_REGRESSION_TEST_YET` | — |
| **D17** | الملاحةُ لا تعود بعد موت العمليّة | `NO_REGRESSION_TEST_YET` | — |
| **D18** | START_STICKY يعيد الخدمةَ بفترةِ الافتراض | `NO_REGRESSION_TEST_YET` | — |
| **D19** | LiveSocket يعيد الوصلَ بتوكنٍ منتهٍ | `NO_REGRESSION_TEST_YET` | — |
| **D20** | البثُّ الحيُّ يتجاوز redactForMerchant | `EXPECTED_FAIL` | `TestD20_CustomerRealtimeVsREST` · `TestD20_MerchantRealtimeVsREST` · `TestOrderFieldsAllClassified` |
| **D21** | هاتفُ السائق يصل الزبون | `EXPECTED_FAIL` | `TestD21_CustomerRedactionAgainstContract` · `TestForbiddenFieldGuardCatchesLeak` · `TestOrderFieldsAllClassified` |
| **D22** | الطلبُ الخاصُّ لا يُبثّ لصاحبه | `EXPECTED_FAIL` | `TestD22_CustomOrderOwnerChannelContract` · `TestOrderFieldsAllClassified` |
| **D23** | حمولاتُ REST تكشف اقتصاداً داخليّاً | `EXPECTED_FAIL` | `TestD20_MerchantRealtimeVsREST` · `TestD21_CustomerRedactionAgainstContract` · `TestOrderFieldsAllClassified` |

---

# ٥ · فجواتُ العقد

| ID | الشدّة | يوقظه | الحال | الاختبارات |
|---|---|---|---|---|
| **XG-5** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-6** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-7** | `HIGH` | `orders.auto_accept_min` | `NOT_IMPLEMENTED` | — |
| **XG-8** | `MEDIUM` | — | `NOT_IMPLEMENTED` | — |
| **XG-9** | `MEDIUM` | — | `NOT_IMPLEMENTED` | — |
| **XG-10** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-11** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-12** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-13** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-14** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-15** | `HIGH` | `sales.activation_orders` | `NOT_IMPLEMENTED` | — |
| **XG-16** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-17** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-18** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-19** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-20** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-21** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-22** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-23** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-24** | `HIGH` | — | `NOT_IMPLEMENTED` | — |
| **XG-25** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-26** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-27** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-28** | `CRITICAL` | — | `NOT_IMPLEMENTED` | — |
| **XG-29** | `BLOCKER` | — | `NOT_IMPLEMENTED` | — |
| **XG-30** | `HIGH` | — | `NOT_IMPLEMENTED` | — |

---

# ٦ · الشيخوخةُ والفجوات

```
STALE REFERENCES = 0
COVERAGE GAPS    = 65
```

## فجواتُ تغطية — **ما يحتاج اختباراً ولا اختبارَ له**

- D1 — لا اختبارَ انحدارٍ بعد
- D10 — لا اختبارَ انحدارٍ بعد
- D11 — لا اختبارَ انحدارٍ بعد
- D12 — لا اختبارَ انحدارٍ بعد
- D13 — لا اختبارَ انحدارٍ بعد
- D15 — لا اختبارَ انحدارٍ بعد
- D16 — لا اختبارَ انحدارٍ بعد
- D17 — لا اختبارَ انحدارٍ بعد
- D18 — لا اختبارَ انحدارٍ بعد
- D19 — لا اختبارَ انحدارٍ بعد
- D2 — لا اختبارَ انحدارٍ بعد
- D3 — لا اختبارَ انحدارٍ بعد
- D4 — لا اختبارَ انحدارٍ بعد
- D5 — لا اختبارَ انحدارٍ بعد
- D6 — لا اختبارَ انحدارٍ بعد
- D8 — لا اختبارَ انحدارٍ بعد
- D9 — لا اختبارَ انحدارٍ بعد
- F-03 (ردُّ إنشاءٍ ضائع) — لا اختبارَ مرتبطٌ به
- F-05 (القبولُ التلقائيّ) — لا اختبارَ مرتبطٌ به
- F-06 (رفضُ المتجر) — لا اختبارَ مرتبطٌ به
- F-07 (عرضُ الطلب على سائق) — لا اختبارَ مرتبطٌ به
- F-09 (انقضاءُ العرض ودورانُه) — لا اختبارَ مرتبطٌ به
- F-10 (عرضُ طلبٍ على الطريق نفسِه) — لا اختبارَ مرتبطٌ به
- F-11 (الوصولُ للاستلام) — لا اختبارَ مرتبطٌ به
- F-12 (الاستلام) — لا اختبارَ مرتبطٌ به
- F-13 (التسليمُ وإثباتُه) — لا اختبارَ مرتبطٌ به
- F-15 (الاسترداد) — لا اختبارَ مرتبطٌ به
- F-16 (تعذّرُ التسليم) — لا اختبارَ مرتبطٌ به
- F-17 (إلغاءُ الزبون) — لا اختبارَ مرتبطٌ به
- F-18 (تحويلُ الطلب لمتجرٍ آخر) — لا اختبارَ مرتبطٌ به
- F-19 (إسنادٌ إداريٌّ وإعادةُ إسناد) — لا اختبارَ مرتبطٌ به
- F-20 (إرسالُ الطلب بواتساب) — لا اختبارَ مرتبطٌ به
- F-21 (تسجيلُ مرشَّحٍ ثمّ تحويلُه) — لا اختبارَ مرتبطٌ به
- F-22 (نقلُ متجرٍ بين مندوبين) — لا اختبارَ مرتبطٌ به
- F-23 (عمولةُ المندوب) — لا اختبارَ مرتبطٌ به
- F-24 (طلبُ سحبٍ وقرارُه) — لا اختبارَ مرتبطٌ به
- F-26 (مصروفٌ وخزينة) — لا اختبارَ مرتبطٌ به
- F-28 (تعليقُ متجر) — لا اختبارَ مرتبطٌ به
- F-31 (شكوى أو بلاغٌ ثمّ حلٌّ بتعويض) — لا اختبارَ مرتبطٌ به
- F-32 (مراجعةُ صنفٍ معلَّق) — لا اختبارَ مرتبطٌ به
- … و25 أخرى

