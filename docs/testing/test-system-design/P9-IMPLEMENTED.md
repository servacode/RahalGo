# `P-9` — محرّكُ أثر التغيير · **مُنفَّذٌ ومُثبَتٌ بضوابطِه**

> **٢٠٢٦-٠٩-٠٥ · المرحلةُ التاسعة.**
> **ولم تُصلَح عيوبٌ ولم تُمسّ شيفرةُ إنتاج** — `PRODUCT DEFECTS FIXED = 0`.

---

# ١ · المشكلةُ التي يحلّها

> **عند تعديل ملفٍّ أو إعدادٍ أو بابٍ أو تدفّق، لا يجوز أن يُقرأ المشروعُ
> كلُّه من الصفر لمعرفة ما انكسر.**

**والقاعدةُ التي تحكم كلَّ سطرٍ فيه:**

```
NO FALSE CONFIDENCE
```

**فما لم يُعرَف أثرُه يُوسَّع لا يُهمَل.** **و«لا أثر» تُقال بدليلٍ أو لا
تُقال.**

---

# ٢ · ما بُني

| الملفّ | ما فيه |
|---|---|
| `backend/internal/impact/model.go` | **الأنواع** — الأصنافُ والأوضاعُ والثقةُ وصنفُ الخطر |
| `backend/internal/impact/rules.go` | **التصنيفُ وتسعُ قواعدِ نطاقٍ مقيسة** |
| `backend/internal/impact/engine.go` | **المحرّك** — يستهلك `TEST_TRUTH` ولا ينسخه |
| `backend/internal/impact/input.go` | **ثلاثةُ مداخل** والتقريرُ البشريّ |
| `backend/cmd/testimpact/` | **الأمر** |
| `backend/Makefile` | `make test-impact` |

## ولا حقيقةَ ثانية (البند ١)

**يُستهلَك ما وُلّد في `P-2…P-8`** — **ولا يُكتب هنا اسمُ اختبارٍ ولا عددُ
تدفّق.** أسماءُ الاختبارات تُقرأ من `TEST_TRUTH`، **وحزمتُها تُشتقّ من
ملفِّها فيه** — فلا مسارَ يُخمَّن.

---

# ٣ · الأرقام

```
CHANGE INPUT MODES = 3   (GIT_RANGE · WORKING_TREE · EXPLICIT_FILES)
FILE CATEGORIES    = 15
DOMAIN RULES       = 9

FLOWS ADDRESSABLE    = 34/34
SETTINGS ADDRESSABLE = 95/95
GAPS ADDRESSABLE     = 23/26
DEFECTS ADDRESSABLE  = 21/27
RISKS ADDRESSABLE    = 13/24

TESTS ADDRESSABLE = 713
ORPHAN TESTS STILL PRESENT = 568
```

## ⚠️ وما لا يبلغه المحرّكُ يُقال (البند ٢٩)

**ستّةُ عيوبٍ وأحدَ عشرَ خطراً لا يبلغها التوسيعُ من التدفّقات** —
**لأنّها غيرُ مربوطةٍ بتدفّقٍ في `TEST_TRUTH`، لا لأنّ المحرّكَ عاجز.**

**والعلاجُ ربطُها في الحقيقة لا في المحرّك** (البند ٣٧: **لا تُعدَّل
الخرائطُ يدويّاً لإجبار نتيجة**). **وحتّى يقع ذلك، `SAFE FALLBACK` يغطّي
ما لا يُعرَف.**

**و٥٦٨ اختباراً يتيماً** — **ولا يُزعَم أنّها تغطّي ميزةً** (البند ٣٨).
**تُختار بحزمتها حين تتبدّل حزمتُها**، لا بادّعاءٍ.

---

# ٤ · الضوابطُ الستّة — **وكلُّها مُثبَتة**

```
CHANGE IMPACT SELF-TESTS = 9/9
```

## `HISTORICAL CHANGE CASES = 8/8` (البند ٣١)

**ولا معرّفَ التزامٍ مثبَّتٌ ليمرّ الاختبار** — **المعرّفُ يشيخ والمسارُ
يبقى.**

| التغيير | ما اختاره المحرّك |
|---|---|
| **`customer_privacy.go`** | `SECURITY` `REALTIME` · **و`D21`/`D23`** |
| **`wallet/wallet.go`** | `FINANCIAL` `CONCURRENCY` `FAILURE` · **و`internal/fininv`** |
| **`driver_handlers.go`** | `CONCURRENCY` · **و`D24`/`R10`** |
| **`push/push.go`** | `REALTIME` `FAILURE` · **وجهازٌ مطلوب** |
| **`PointQueue.kt`** | `ANDROID` · **ودَينُ `P-8` مذكور** |
| **هجرةُ قاعدة** | `FULL` `FINANCIAL` |
| **`settings/catalog.go`** | **إعادةُ توليد الحقيقة** |
| **`deploy/nginx.conf`** | **`STAGING VALIDATION REQUIRED`** |

## `DEFECT IMPACT CASES = 4/4` (البند ٣٢)

```
D20 ← orders/service.go          متأثّرٌ ✓ · حارسٌ مختارٌ ✓
D24 ← server/driver_handlers.go  متأثّرٌ ✓ · حارسٌ مختارٌ ✓
D26 ← orders/watchdog.go         متأثّرٌ ✓ · حارسٌ مختارٌ ✓
D27 ← push/push.go               متأثّرٌ ✓ · حارسٌ مختارٌ ✓
```

**و`D27` سقط أوّلَ تشغيل** — **لأنّ قاعدةَ الأحداث لم تكن تشمل `F-07`
و`F-14`.** **والإصلاحُ في القاعدة لا في الاختبار** — وسّعتُ القاعدةَ
لتشمل تدفّقَي العرضِ والتسوية، **وهما حيث يقع الدفعُ فعلاً.**

## `NEGATIVE CONTROL = PASS` (البند ٣٣)

```
README.md + docs/WORKLOG.md  ⇒  modes=[FAST] · tests=0 · risk=LOW
```

**ولا انفجارَ بلا سبب** — **والمحافظةُ لا تعني تشغيلَ كلّ شيءٍ في كلّ
تعديل.**

**ووثيقةُ الحقيقة ليست وثيقةً عاديّة** (البند ٢٥):

```
docs/testing/FINAL_STATIC_CLOSEOUT.md  ⇒  internal/testtruth (حرّاسُ الانحراف)
```

## `UNKNOWN CHANGE SELF-TEST = PROVEN` (البندان ١٠ و٣٤)

```
some/new/unmapped/area/thing.go
⇒ fallback = "UNKNOWN IMPACT → SAFE FULL FALLBACK"
⇒ confidence = LOW · risk = HIGH · modes = [FULL]
⇒ الأمرُ الموصى به يشمل ./...
```

**ولا `0 TESTS` لملفٍّ مجهول.**

## `STALE MAPPING GUARD = PASS` (البند ٣٦)

```
0 خريطةً شائخةً · 0 ملفّاً مفقوداً
```

**ويُفحَص الاتّجاهان**: خريطةٌ تشير إلى اختبارٍ زال، **وملفُّ اختبارٍ لم
يعد موجوداً.**

## `WHY-SELECTED TRACE = PASS` (البندان ٢٧ و٢٨)

```
54 اختباراً · مباشرٌ 2 · متعدٍّ 16
```

**ولا اختبارَ يُختار بلا سبب** — كلُّ هدفٍ يحمل `Rule` و`Path` و`Depth`.
**والمباشرُ يُميَّز عن المتعدّي** (البند ٦).

## `REPEATABILITY = PASS` (البند ٤٧)

**خمسةُ تشغيلاتٍ متطابقةٌ حرفاً** — **ولا وقتٌ في المحتوى المعياريّ** يكسر
الإعادة.

---

# ٥ · الزمن (البند ٤٢)

```
AVERAGE LOCAL IMPACT TIME = 816µs
MAX LOCAL IMPACT TIME     = 2.556ms
```

**ولا يُقرأ المشروعُ كلُّه** — **الحقيقةُ مولَّدةٌ فيُقرأ ملفٌّ واحد.**
**ولا `SLO` عشوائيٌّ يُخترَع**: هذا مقيسٌ لا موعود.

---

# ٦ · مثالٌ كامل

```
$ go run ./cmd/testimpact -files backend/internal/orders/transitions.go

CHANGED (1)
  PRODUCT_BACKEND backend/internal/orders/transitions.go

IMPACTED
  FLOWS    [F-01 F-04 F-07 F-08 F-12 F-13 F-14 F-15 F-16 F-17 …+6]
  APPS     [admin customer driver merchant rep]
  DEFECTS  [D1 D13 D19 D20 D23 D24 D27 D4 D5 D7]
  RISKS    [R10 R21 R23 R4 R7 R8]
  GAPS     [XG-10 XG-11 XG-12 …+3]
  SETTINGS 56 مفتاحاً

WHY
  [DIRECT] FINANCIAL — مسٌّ لمسارٍ ماليّ ⇒ ثوابتُ P-4 وسباقاتُها وحقنُ فشلها
  [DIRECT] ORDER_LIFECYCLE — دورةُ حياة الطلب تمسّ أربعةَ تطبيقاتٍ ولو تبدّل ملفٌّ واحد
  [TRANSITIVE] FLOW_EXPANSION — F-14 ⇒ تطبيقاتُه [merchant driver rep admin] وسجلّاتُه

RUN THESE TESTS   (إلزاميٌّ 54)
  go test -count=1 -p 1 ./internal/fininv ./internal/orders ./internal/qa

MODES       [CONCURRENCY FAILURE FINANCIAL]
CONFIDENCE  HIGH
RISK CLASS  CRITICAL

DEVICE REQUIRED?  NO
STAGING REQUIRED? NO
UNKNOWN IMPACT?   NO
SAFE TO USE IMPACTED MODE? YES

ELAPSED = 2ms
```

**وملفٌّ واحدٌ في المحرّك يفتح خمسةَ تطبيقاتٍ وستَّ عشرةَ تدفّقاً** —
**وهذا هو التوسيعُ عبرَ التدفّقات الذي طلبه البند ١٨.**

---

# ٧ · التصعيداتُ الستّة — **وكلُّها مُثبَتة**

| التصعيد | البند | الدليل |
|---|---|---|
| **FINANCIAL** | ١٤ | `wallet.go` ⇒ `FINANCIAL` + `internal/fininv` |
| **PRIVACY** | ١٥ | `customer_privacy.go` ⇒ `SECURITY` + `D21`/`D23` |
| **SECURITY** | ١٦ | `auth/` · `identity/` · `realtime/` ⇒ `SECURITY` + بثّ |
| **ANDROID DEVICE** | ١٧ · ٤٤ | أيُّ ملفِّ أندرويد ⇒ **`REAL DEVICE ACCEPTANCE = NOT STARTED`** |
| **STAGING** | ٢٦ · ٤٥ | `deploy/` ⇒ **`STAGING VALIDATION REQUIRED`** · و`R16` مع الهويّة |
| **LATENT GAP** | ١٣ | فجوةٌ `BLOCKER`/`CRITICAL` ⇒ **صنفُ الخطر يرتفع** |

---

# ٨ · الدَّينان محفوظان

```
P-8 REAL DEVICE DEBT PRESERVED = YES
P-0 / STAGING DEBT PRESERVED   = YES
```

**فأيُّ تبديلٍ في أندرويد يُخرج:**

> **`REAL DEVICE ACCEPTANCE = NOT STARTED`** — **و٦٢١ اختباراً محلّيّاً
> لا تُغني عنه.**

**وأيُّ تبديلٍ في الهويّة أو النشر يُخرج `STAGING VALIDATION REQUIRED`** —
**ولا يُدَّعى اكتمالٌ محلّيّ.**

---

# ٩ · المخرَجاتُ والأمر

```
docs/testing/system/CHANGE_IMPACT.json        ← كلُّ تشغيل
docs/testing/system/CHANGE_IMPACT_RULES.json  ← القواعدُ آليّةً
```

```
make test-impact                    شجرةُ العمل
make impact BASE=HEAD~1 HEAD=HEAD   مدىً في git
go run ./cmd/testimpact -files a,b  ملفّاتٌ صريحة
go run ./cmd/testimpact -json       الصورةُ الآليّةُ فقط
```

**ولا تعيش قواعدُ الاختيار في Markdown وحدَه** (البند ٤٩).

---

# ١٠ · `TEST_TRUTH` بعد `P-9`

```
TESTS   713  (كانت 704)
FLOWS     34/34
DEFECTS   27/27
RISKS     24/24
GAPS      26/26
SETTINGS  95/95
TRUTH DRIFT = NONE
```

**والمصفوفاتُ الثمان:**

```
FINANCIAL_INVARIANTS · CONCURRENCY_MATRIX · FAILURE_INJECTION_MATRIX
EVENT_CONTRACT_MATRIX · ANDROID_DEVICE_MATRIX · ANDROID_TEST_MATRIX
CHANGE_IMPACT_RULES · CHANGE_IMPACT
```
