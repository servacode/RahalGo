# `P-8` — أندرويد والجهازُ الحقيقيّ · **المِسنَدُ مُنفَّذٌ · والقبولُ لم يُنفَّذ**

> **٢٠٢٦-٠٩-٠٥ · المرحلةُ الثامنة.**
> **ولم تُصلَح عيوبٌ ولم تُمسّ شيفرةُ إنتاج** — `PRODUCT DEFECTS FIXED = 0`.

---

# ١ · ⚠️ الحقيقةُ التي تحكم هذا التقرير كلَّه

```
adb devices  ⇒  List of devices attached
                (فارغة)
```

**لا جهازَ متّصلٌ بهذه الآلة** — **ولا `SM-A525F` ولا غيرُه.**

**فالفصلُ الذي طلبه البند ٥٠ ليس تنظيماً بل ضرورة:**

```
P-8 HARNESS IMPLEMENTATION = COMPLETE
REAL DEVICE ACCEPTANCE     = NOT STARTED
```

**وجهازٌ غيرُ متوفّرٍ ليس `PASS`** (البند ٤٤) — **وحارسٌ في الشيفرة يمنع
أن يُقرأ كذلك.**

---

# ٢ · ما بُني

| الملفّ | ما فيه |
|---|---|
| `backend/internal/androidmap/androidmap.go` | **المصفوفتان** — ٥ أصنافِ أجهزةٍ و٢٧ حالةَ اختبار |
| `backend/internal/androidmap/androidmap_test.go` | **ثلاثةُ حرّاس** — ولا تشغيلَ يُدَّعى بلا جهاز |
| `mobile/testkit/devicerun.sh` | **هويّةُ التشغيل وحارسُ الإنتاج** |
| `mobile/app-driver/src/test/.../LocationContractTest.kt` | **خمسةُ إثباتاتٍ بنيويّة** |

## ولم يُبنَ إطارٌ ثانٍ (البند ١)

**والمقيسُ قبل البناء:**

```
mobile/  → app-customer · app-driver · app-merchant · app-rep
           shared · map · driver-navigation · ui · design · brand
           drivertest/ · merchtest/ · reptest/   ← سكربتاتُ API
اختباراتٌ قائمة: 53 ملفَّ اختبارٍ وحدويّ · 2 على الجهاز
```

**وشُغّلت كلُّها** — **ولا يُبنى على مِسنَدٍ لم يُشغَّل:**

```
ANDROID JVM TESTS = 621  ·  51 صفّاً  ·  0 سقوط  ·  0 تخطٍّ
:app-customer · :app-driver · :app-merchant · :app-rep
:shared · :map · :driver-navigation
```

**والخمسةُ الجديدةُ منها** — `LocationContractTest`.

**و`drivertest` و`merchtest` و`reptest` لا تُستعمل** — **تشير إلى
`https://api.rahalgo.com`**، وهو **الإنتاج**. **ولا اختبارَ كتابةٍ على
الإنتاج** (البند ٣).

---

# ٣ · هويّةُ التشغيل (البند ٢)

**ودليلٌ بلا هويّةٍ لا يُنسَب** — ثلاثةُ تشغيلاتٍ من ثلاثة بناءاتٍ تُقرأ
سواءً.

```
RUN_ID · SOURCE_COMMIT · APK_HASH · APP_VERSION · BACKEND_COMMIT
DEVICE_SERIAL · DEVICE_MODEL · ANDROID_VERSION · BUILD_FINGERPRINT
BATTERY · NETWORK · STARTED_AT · TEST_CASE · RESULT · ENDED_AT
```

**و`begin` يمسح `logcat` ويفتح مجلَّداً، و`end` يسحب السجلَّ ولقطةَ
الشاشة** — فالدليلُ يُجمَع آليّاً لا بيد.

```
RUN IDENTITY = PASS  (الأداةُ مبنيّةٌ ومُختبَرة)
```

## وحارسُ الإنتاج (البند ٣)

**ولا يُكتب سطرٌ على جهازٍ يشير إلى الإنتاج.** يُقرأ العنوانُ **من الحزمة
المثبَّتة نفسِها** — لا يُفترَض:

```bash
strings <apk> | grep -E 'https?://[a-z0-9.-]+/api'
⇒ إن وُجد api.rahalgo.com  ⇒  PRODUCTION ENDPOINT DETECTED — توقّف
```

**وأُثبت أنّه يرفض:**

```
$ bash testkit/devicerun.sh identity
✗ لا جهازَ متّصل — DEVICE NOT AVAILABLE   (rc=1)

$ bash testkit/devicerun.sh safety
✗ لا جهازَ متّصل — DEVICE NOT AVAILABLE   (rc=1)
```

**فهو يتوقّف ولا يدّعي.**

---

# ٤ · مصفوفةُ الأجهزة — `TQ-2`

```
DEVICES = 5 · AVAILABLE = 0 · EXECUTED = 0
```

| # | الصنف | الطراز | الحال | التشغيل | لماذا يلزم |
|---|---|---|---|---|---|
| **`DEV-01`** | **SAMSUNG BASELINE** | **`SM-A525F` / Android 14** | ❌ **NOT AVAILABLE** | **NOT EXECUTED** | **جهازُ القبول المرجعيّ** — وقرارُ المالك يُثبته |
| **`DEV-02`** | **XIAOMI/REDMI** | — | ❌ | **NOT EXECUTED** | **أقسى قيودِ خلفيّةٍ في السوق** — تقتل الخدماتِ اللاصقة |
| **`DEV-03`** | **INFINIX/TECNO** | — | ❌ | **NOT EXECUTED** | **شريحةُ سائقي الرقّة الأوسع** — وموتُ عمليّةٍ متكرّر |
| **`DEV-04`** | **OLDER ANDROID** | — | ❌ | **NOT EXECUTED** | **الأذوناتُ تختلف قبل `Android 12`** |
| **`DEV-05`** | **HUAWEI** | — | ❌ | **NOT EXECUTED** | **لا FCM ولا خرائطَ جوجل** — مسارٌ آخرُ كلّيّاً |

**و`Infinix X6833B` لم يُعَد اعتمادُه** (قرارُ المالك) — **ولا يدخل
المصفوفةَ إلّا بتسجيلٍ صريح.**

---

# ٥ · مصفوفةُ الاختبارات

```
CASES = 27 · APPS COVERED = 4/4
STRUCTURAL = 5 · DEVICE = 21 · REAL_WORLD_DRIVE = 1
AUTOMATED = 5 · SEMI_AUTOMATED = 19 · MANUAL = 3
EXPECTED_FAIL = 2 · RISK_CONFIRMED = 3 · NOT_RUN = 22
```

**والحارسُ يمنع ثلاثةَ أكاذيب:**

```
حالةٌ نوعُها DEVICE ونتيجتُها ليست NOT_RUN ولا جهازَ  ⇒  سقوط
حالةٌ بنيويّةٌ بلا اختبارٍ يحرسها                      ⇒  سقوط
حالةٌ لم تُشغَّل ولا تقول ما يلزم لتشغيلها             ⇒  سقوط
```

---

# ٦ · الخمسةُ المُثبَتة — **إثباتٌ بنيويٌّ لا ادّعاء**

**و`PointQueue` و`LocationService` يعتمدان `android.content.Context`** —
**فلا يعملان في آلة Java وحدَها.** **ولا جهازَ.**

**فما أُثبت هو بنيةُ المسار من مصدره** — **كما أُثبت ترتيبُ `D2` في `P-6`
بقراءة المصدر لا بتشغيله.** **وهو دليلٌ حقيقيّ**: يسقط يومَ تتبدّل
الشيفرةُ ويقول أين.

**وصُنّف `STRUCTURAL` لا `DEVICE`** — **فلا يُقرأ إثباتاً على جهاز.**

```
LocationContractTest — 5 tests · 0 failures · 0 errors
```

## `R19` — نافذةُ الطابور

```kotlin
val waiting = queue.all()      // ١ يقرأ الملفّ
api.sendBatch(waiting)         // ٢ نداءُ شبكةٍ يستغرق زمناً
queue.clear()                  // ٣ file.delete() — **الملفُّ كلُّه**
```

**فنقطةٌ تُضاف أثناء نداء الشبكة تُمحى ولم تُرسَل.**
**ولا مؤشّرَ إرسالٍ في الطابور** — لا `sentUpTo` ولا `removeFirst`.

```
R19 LOCATION QUEUE RACE = RISK CONFIRMED (STRUCTURAL)
```

## `R20` — نجاحُ الدفعة يمسح المرفوض

**`sendBatch` تنجح بردّ `2xx`** — **ولا تُقرأ حمولةُ الردّ** لتُعرَف نقطةٌ
رُفضت. **فالمسحُ يقع على المقبولِ والمرفوضِ سواء.**

```
R20 LOCATION BATCH REJECTION = RISK CONFIRMED (STRUCTURAL)
```

## `D16` — الطابورُ بلا صاحب

**`points.jsonl` ملفٌّ واحدٌ للتطبيق** — **لا `driverId` ولا `userId`.**
**ولا مسارَ خروجٍ يمسّه** (قِيس في كلّ مصادر التطبيق).
**فنقاطُ الأوّل تُرفَع باسم الثاني.**

```
D16 LOCATION QUEUE OWNER = EXPECTED FAIL (STRUCTURAL)
```

## `D18` — الفاصلُ يضيع بالإقلاع اللاصق

```kotlin
val seconds = intent?.getLongExtra(EXTRA_PING_SEC, 0L)?.takeIf { it > 0 } ?: DEFAULT_PING_SEC
return START_STICKY
```

**وأندرويد يُسلّم `intent = null` عند الإقلاع اللاصق** — **ولا يُحفَظ
الفاصلُ خارجَ النيّة** (لا `SharedPreferences` ولا `DataStore`).

```
D18 STICKY RESTART INTERVAL = EXPECTED FAIL (STRUCTURAL)
```

## `R17` — لا إعادةَ تحقّق

**`onStartCommand` تُقلع الواجهةَ وتطلب الموقع** — **ولا تسأل عن ورديّةٍ
ولا جلسةٍ ولا حالِ سائق.** **فخدمةٌ تُقلع بعد إغلاق ورديّةٍ تجمع مواقعَ
لمن ليس على الدوام.**

```
R17 STICKY REVALIDATION = RISK CONFIRMED (STRUCTURAL)
```

---

# ٧ · ما لم يُشغَّل — **اثنتان وعشرون حالة**

**وكلُّها تقول ما يلزمها:**

| الحاجة | العدد |
|---|---|
| **`SM-A525F` وحدَه** | **١٧** |
| **جهازٌ + أداةُ نظام** (`adb dumpsys deviceidle` · `am force-stop`) | **٣** |
| **جهازان أو حسابٌ بتطبيقين** (`XOB-4`) | **١** |
| **قيادةٌ في الشارع** (`REAL-WORLD DRIVE`) | **١** |

**ولا واحدةٌ منها تحتاج `P-0`** — **الناقصُ جهازٌ لا بيئةُ تكامل.**

```
REQUIRES P-0 = 0
REQUIRES ADDITIONAL DEVICE = 22
REAL-WORLD DRIVE REQUIRED = 1
```

---

# ٨ · ما يُثبته `P-7` وما ينتظر `P-8`

**ولا يُعاد إثباتُ ما أُثبت** — **والفصلُ مكتوبٌ في كلّ حالة:**

| العقد | جانبُ الخادم | جانبُ الجهاز |
|---|---|---|
| **`D12`** | **مُثبَتٌ في `P-7`** — الاستهدافُ سليم | **`AND-51`** — سلوكُ الخروج في أندرويد |
| **`R14`** | **مُثبَتٌ في `P-7`** — عزلُ الغرف | **`AND-52`** — وصلةٌ مفتوحةٌ بعد الإبطال |
| **الروابطُ العميقة** | **مُثبَتٌ في `P-7`** — الحمولةُ تحمل الكيان | **`AND-21`** — النقرُ يفتح الشاشةَ الصحيحة |
| **طلبان لا يختلطان** | **مُثبَتٌ في `P-7`** | **`AND-22`** — على الجهاز |
| **`XOB-4`** | **٢٧ من ٤٧ موضعاً بلا توجيهٍ مقيسة** | **`AND-53`** — أيصل الإشعارُ إلى التطبيق الخطأ |

---

# ٩ · `TEST_TRUTH` بعد `P-8`

```
TESTS   704  (كانت 701)
FLOWS     34/34
DEFECTS   27/27
RISKS     24/24
GAPS      26/26
SETTINGS  95/95
TRUTH DRIFT = NONE
```

**واختباراتُ Kotlin لا يراها مستخرِجُ `TEST_TRUTH`** — يقرأ Go وحدَها.
**فتُحصى في `ANDROID_TEST_MATRIX.json`**، **ولا يُدَّعى أنّها في العدّ
الأوّل.**

**والمصفوفاتُ السّت:**

```
FINANCIAL_INVARIANTS.json      ← P-4
CONCURRENCY_MATRIX.json        ← P-5
FAILURE_INJECTION_MATRIX.json  ← P-6
EVENT_CONTRACT_MATRIX.json     ← P-7
ANDROID_DEVICE_MATRIX.json     ← P-8
ANDROID_TEST_MATRIX.json       ← P-8
```

---

# ١٠ · إغلاقاتُ ما قبل `P-8`

## `D26` و`D27` — **جُمّدا بقرار المالك**

```
FROZEN DEFECT COUNT = 27
FROZEN RISK COUNT   = 24
CONTRACT GAP COUNT  = 26
```

**و`R22` و`R23` بقيا في سجلّ المخاطر للتتبّع** —
`R22 → CONFIRMED → D26` · `R23 → CONFIRMED → D27`.

**و`XOB-6` طُوي دليلاً داخل `D26`** — **ولم يصر عيباً مستقلّاً.**

## `XOB-13` — **ملاحظةٌ جديدةٌ سُجّلت**

```
XOB-13 — actor_id يصل الزبونَ لحظيّاً وREST في حقل events
```

**ولا تُجمَّد** — **حتّى يُثبَت أيكشف هويّةَ فاعلٍ أو صلاحيّتَه، أم هو
معرّفٌ داخليٌّ زائدٌ لا غير.**
