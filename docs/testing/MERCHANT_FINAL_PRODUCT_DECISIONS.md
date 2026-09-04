# قراراتُ المالك النهائيّة — تطبيقُ المتجر

> **٢٠٢٦-٠٩-٠٤ · قراراتُ مالكٍ نهائيّة.**
> **ليست إصلاحاً ولا تحقّقَ تشغيل** — **ولم يُمسَّ سطرُ كودٍ واحد.**
>
> تُقرأ مع [`MERCHANT_APP_CLOSURE.md`](MERCHANT_APP_CLOSURE.md) و
> [`MERCHANT_OWNER_REVIEW.md`](MERCHANT_OWNER_REVIEW.md) و
> [`SHAM_CASH_PRODUCT_CONTRACT.md`](SHAM_CASH_PRODUCT_CONTRACT.md).

**الأسئلةُ السبعةُ أُغلقت كلُّها**: **ثلاثةٌ أغلقها الكود** (`MD-4` · `MD-6` ·
`MD-7`) و**أربعةٌ حسمها المالكُ اليوم** (`MD-1` · `MD-2` · `MD-3` · `MD-5`).

```
MERCHANT OPEN OWNER DECISIONS = 0
```

---

# `MD-1` — تعديلُ القسم وحذفُه · **APPROVED**

## العقد

**صاحبُ المتجر يملك قائمتَه كاملةً**: `CREATE` · `EDIT` · `DELETE`.
**والإنشاءُ وحدَه غيرُ مقبولٍ كعقدٍ نهائيّ.**

## وحذفُ القسم لا يترك يتيماً

**لا حذفَ صامتاً لقسمٍ فيه أصناف.** إمّا **منعُ الحذف حتّى تُنقل الأصنافُ أو
تُحذف**، وإمّا **خيارٌ صريحٌ للمستخدم كيف تُعالَج**. **ولا يجوز أن ينتهيَ
الحذفُ إلى بياناتٍ يتيمةٍ أو مخفيّة.**

## مقابلةُ القرار بالكود — **٢٠٢٦-٠٩-٠٤**

| ما يوجبه العقد | الحال المقيس | الحكم |
|---|---|---|
| **`PATCH /merchant/menu/sections/{id}`** | **قائمٌ في المحرّك** | ✅ |
| **`DELETE /merchant/menu/sections/{id}`** | **قائمٌ في المحرّك** — `admin_menu_handlers.go:50` | ✅ |
| **حمايةُ الحذف من اليُتم** | **قائمةٌ ومنفَّذة** — `catalog/menu.go:382-390` | ✅ **العقدُ مُستوفىً في المحرّك** |
| **سطحُ التعديل في التطبيق** | **لا زرَّ ولا شاشة** | ❌ `MG-1` |
| **سطحُ الحذف في التطبيق** | **لا زرَّ ولا شاشة** | ❌ `MG-2` |

**والحمايةُ ليست وعداً بل شرطٌ مُنفَّذ:**

```go
// backend/internal/catalog/menu.go:382
func (s *Service) DeleteSection(ctx context.Context, actorID, sectionID, ip string) error {
	var count int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE section_id = $1`, sectionID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrSectionNotEmpty          // ← لا حذفَ صامت
	}
	...
	s.audit(ctx, actorID, "menu.section_delete", "menu_section", sectionID, ip)
```

**فالمحرّكُ اختار السلوكَ الأوّل من الاثنين اللذين أجازهما المالك**: **يمنع
الحذفَ حتّى يفرغ القسم.** **وهو مطابقٌ للعقد ولا يحتاج تبديلاً.**
**والفجوةُ في التطبيق وحدَه**: **سطحان ناقصان لا منطقٌ ناقص.**

**والحذفُ مُدقَّقٌ أصلاً** (`menu.section_delete`) — **وهو ما يفتقده
`MD-3`.**

```
MD-1 = CLOSED
MD-1 CONTRACT GAPS = 2   (MG-1 سطحُ التعديل · MG-2 سطحُ الحذف)
MD-1 BACKEND READY = YES
```

---

# `MD-2` — إشعارُ المتجر بإسناد السائق ووصوله · **APPROVED**

## العقد

**المتجرُ يبقى على اطّلاعٍ بحركة الطلب التشغيليّة**، وأقلُّه:

| الحدث | ما يصل |
|---|---|
| **إسنادُ سائق** | «تمّ تعيينُ سائقٍ للطلب» |
| **وصولُه لنقطة الاستلام** | «وصل السائقُ لاستلام الطلب» |

**والإشعارُ `ORDER-SCOPED`** — **يُضغط فيفتح الطلبَ بعينه.**

## والخصوصيّةُ لا تتبدّل

**لا اسمَ سائقٍ ولا رقمَه ولا بياناتِه.** **الحالةُ التشغيليّةُ وحدَها.**
**وهذا يثبّت `MD-4` ولا ينقضه**: `redactForMerchant` تبقى كما هي.

## مقابلةُ القرار بالكود — **٢٠٢٦-٠٩-٠٤**

**والمتجرُ يُشعَر في موضعين اثنين لا غير** — قِيسا بجرد كلّ
`notifications.AppMerchant` في المحرّك:

| # | الموضع | الحدث |
|---|---|---|
| ١ | `orders/notify.go:142` | **طلبٌ جديد** — ومعه عددُ الأصناف والمبلغ |
| ٢ | `orders/notify.go:257` | **تمّ التسليم** |

**وبينهما لا شيء.** **والحالاتُ الأربعَ عشرةَ تمرّ صامتةً على المتجر** —
ومنها `assigned` و`at_pickup` اللتان أوجبهما هذا القرار.

**والسببُ في السطر ١٩٣:**

```go
// backend/internal/orders/notify.go:189
func (s *Service) notifyTransition(ctx context.Context, orderID, to, note, endedBy string) {
	...
	title, worth := customerTitles[to]      // ← معجمُ الزبون وحدَه
	if !worth {
		return
	}
	...
	Apps: []string{notifications.AppCustomer},
```

**فمعجمُ العناوين معجمُ زبون**، **والمتجرُ ملحقٌ بحالتين مستثنَيَتين.**
**ولا معجمَ متجرٍ أصلاً.**

## والفجوةُ الثالثةُ تمسّ العقدَ كلَّه

**«يفتح الطلبَ المحدَّد» غيرُ ممكنٍ اليومَ في أيّ تطبيق:**

```kotlin
// mobile/ui/.../push/RahalPushService.kt:115
val open = PendingIntent.getActivity(
    ..., Intent(this, home()).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP), ...)
```

**كلُّ إشعارٍ يفتح الشاشةَ الأولى.** **و`EntityID` يُرسَل من المحرّك ولا
يُقرأ في الجهاز.** **وحتّى `Href` في الإشعارين القائمين `/portal` — قائمةٌ لا
طلب.**

**فالعقدُ `ORDER-SCOPED` يحتاج طرفين**: **معجمَ متجرٍ في المحرّك** و**قراءةَ
`entity_id` في التطبيق.**

| المطلوب | الحال | الحكم |
|---|---|---|
| إشعارٌ عند `assigned` | **لا شيء** | ❌ `MG-3` |
| إشعارٌ عند `at_pickup` | **لا شيء** | ❌ `MG-4` |
| فتحُ الطلب بعينه | **`home()` وحدَها في كلّ التطبيقات** | ❌ `MG-5` |
| بلا هويّةِ سائق | **`redactForMerchant` تمحوها** | ✅ **قائم** |

```
MD-2 = CLOSED
MD-2 CONTRACT GAPS = 3   (MG-3 · MG-4 · MG-5)
```

---

# `MD-3` — تدقيقُ إعدادات المتجر · **APPROVED**

## العقد

**كلُّ تبديلِ إعدادٍ يُسجَّل** — **وأخصُّها `PATCH /merchant/stores/{id}/settings`.**
**ولا يتبدّل إعدادٌ تشغيليٌّ أو تجاريٌّ بلا أثر.**

**ويُحفظ ما أمكن**: **الفاعل** · **دورُه** · **المتجر** · **مفتاحُ الإعداد** ·
**القيمةُ السابقة** · **القيمةُ الجديدة** · **الوقت** · **القناة** ·
**والسببُ حين تقتضيه العملية.**

**والغايةُ**: **أن تعرف الإدارةُ لاحقاً مَن غيّر ومتى ومن أيّ قيمةٍ إلى أيّ قيمة.**

## مقابلةُ القرار بالكود — **٢٠٢٦-٠٩-٠٤**

**`handleMerchantSettings` (`merchant_ops_handlers.go:118`) يبدّل ستّةَ حقول**
— `name` · `default_prep_minutes` · `min_order` · `address_text` · `lat` ·
`lng` — **ثمّ يردّ `{"updated": true}` ولا يكتب سطراً:**

```go
// backend/internal/server/merchant_ops_handlers.go:186-203  (مختصَر)
if _, err := s.pg.Exec(r.Context(), `
	UPDATE merchants SET name = COALESCE($7, name), ... updated_at = now()
	WHERE id = $1`, merchantID, ...); err != nil {
	s.respondErr(w, err)
	return
}
httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})   // ← لا s.audit
```

**ولا قراءةَ للقيمة القديمة قبل الكتابة** — **فحتّى لو أُضيف نداءُ تدقيقٍ
اليومَ لَما وجد «من أيّ قيمة».** **والعقدُ يوجب الطرفين.**

**والدفترُ الذي يستقبل هذا قائمٌ ويكفي:**

```sql
-- internal/migrate/migrations/0002_identity.sql:63
CREATE TABLE audit_log (
    actor_user_id uuid REFERENCES users (id),
    action        text  NOT NULL,
    entity        text  NOT NULL DEFAULT '',
    entity_id     text  NOT NULL DEFAULT '',
    details       jsonb NOT NULL DEFAULT '{}',
    ip            text  NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
```

| حقلٌ يوجبه العقد | أيسعُه الدفترُ القائم؟ |
|---|---|
| الفاعل | ✅ `actor_user_id` |
| المتجر | ✅ `entity` + `entity_id` |
| الوقت | ✅ `created_at` |
| القناة | ⚠️ **`ip` وحدَه** — ولا نوعَ عميل |
| دورُ الفاعل · المفتاح · السابق · الجديد · السبب | ✅ **يسعها `details jsonb`** بلا هجرةٍ جديدة |

**فلا حاجةَ إلى جدولٍ جديد** — **والفجوةُ نداءٌ غائبٌ وقراءةٌ سابقةٌ غائبة،
لا بنيةٌ ناقصة.** **و`menu.section_delete` تثبت أنّ النمطَ مستعمَلٌ في المشروع
أصلاً.**

```
MD-3 = CLOSED
MD-3 CONTRACT GAPS = 1   (MG-6 — نداءُ التدقيق وقراءةُ القيمة السابقة)
MD-3 SCHEMA MIGRATION NEEDED = NO
```

**ويسقط بهذا وصفُ `R5`**: «الفعلُ الوحيدُ بلا سببٍ مُثبَتٍ من ٣٥» **صار له
عقدٌ يحسمه.** **ولا يُصلَح الآن.**

---

# `MD-5` — القبولُ التلقائيُّ لا يكون صامتاً · **APPROVED**

## العقد

**إن قَبِل النظامُ الطلبَ نيابةً عن المتجر** — بـ`orders.auto_accept_min` أو
بأيّ سياسةٍ لاحقة — **فلا يقع ذلك بصمت.**

```
AUTO ACCEPT → MERCHANT INFORMED IMMEDIATELY
```

**ويعرف المتجرُ صراحةً**: **أنّ الطلبَ قُبل تلقائيّاً** · **وأنّ التجهيزَ صار
مسؤوليّتَه** · **وأنّ الحالَ بدّلته سياسةُ النظام لا هو.**
مثالاً: «تمّ قبولُ الطلب تلقائيّاً، ابدأ تجهيزَ الطلب.»

**والإشعارُ**: `Push`/`Realtime` بحسب القنوات · `ORDER-SCOPED` · **يفتح الطلبَ.**

**وإن سقطت قناةٌ فالحقيقةُ تبقى مرئيّةً داخل التطبيق** — **ولا يصير الطلبُ
المقبولُ مجهولاً لصاحبه.**

## مقابلةُ القرار بالكود — **٢٠٢٦-٠٩-٠٤**

**`sweepAutoAccept` (`orders/watchdog.go:191`) يقرأ خمسين طلباً `pending` مضت
مهلتُها ويمرّرها إلى `Transition(... StAccepted ...)` بدور `ops`.**
**وينتهي بسطر سجلٍّ للخادم:**

```go
// backend/internal/orders/watchdog.go:224
s.logger.Info("قُبل تلقائيّاً بعد المهلة", "order", id, "minutes", mins)
```

**والسجلُّ يقرؤه المشغّلُ لا صاحبُ المتجر.**

**وتعليقُ الدالّة يَعِد بغير ما تفعل:**

> «**فيمضي الطلبُ كما لو ضغطه موظّف**: **يُخطَر المتجرُ** ويدخل التحضيرَ في
> وضع المتاجر…»

**والقياسُ يكذّب الوعد**: **`notifyTransition` لا تُشعر متجراً عند `accepted`**
— لأنّها لا تعرف إلّا `customerTitles`. **فالطلبُ يُقبل باسم صاحب المتجر ولا
يعلم.**

**وهو أخطرُ من `MD-2`**: **هناك يفوته خبرٌ**، **وهنا يفوته التزامٌ عليه** —
**ومهلةُ التحضير تعدّ.**

| المطلوب | الحال | الحكم |
|---|---|---|
| إشعارٌ فوريٌّ بالقبول التلقائيّ | **سطرُ سجلٍّ للخادم وحدَه** | ❌ `MG-7` |
| `ORDER-SCOPED` | **`home()`** | ❌ **يشترك مع `MG-5`** |
| **الحقيقةُ مرئيّةٌ داخل التطبيق ولو سقطت القناة** | ⚠️ **الطلبُ يظهر في القائمة** — **ولا علامةَ أنّ النظامَ قبِله** | ❌ **جزءٌ من `MG-7`** |

**ويُربط خاصّةً بـ`R21`** (سقوطُ قناتَي الإشعار) **و`D19`** — **ولا يُصلَحان
الآن.** **وفحصُ التشغيل `RV-M1` يغطّيهما.**

```
MD-5 = CLOSED
MD-5 CONTRACT GAPS = 1   (MG-7)
MD-5 LINKED = R21 · D19 · RV-M1
```

---

# الثلاثةُ المغلقةُ بالكود — **تبقى كما هي**

| ID | الحكمُ السابق | أتبدّل اليوم؟ |
|---|---|---|
| **`MD-4`** | `redactForMerchant` تمحو اسمَ السائق ورقمَه عمداً | **لا** — **و`MD-2` يثبّته نصّاً** |
| **`MD-6`** | `WarningsScreen` قائمة · والإنذارُ يسبق التعليق | **لا** |
| **`MD-7`** | `menu_approval.go` يحفظ الحالَ والسبب · `MenuScreen.kt:283` تقرؤهما | **لا** |

---

# ما يبقى محفوظاً بقرار المالك

## الدرجُ الجانبيّ

**في تطبيق المتجر**: **فتحٌ بالسحب** · **إغلاقٌ بالسحب** · **إغلاقٌ بلمس
العتمة** · **زرُّ الرجوع يغلق الدرجَ أوّلاً** · **وزرُّ القائمة يفتحه.**

**ولا تُعطَّل الإيماءاتُ عالميّاً من أجل `PickPoint` النادرة**
(`MainActivity.kt:237`). **`CONTRACT GAP` تحت `PC-10` — ولا يُصلَح الآن.**

## أوّلُ فتحٍ للخريطة

**يبقى `UX / PERFORMANCE GAP`** يقيسه `P-M1`.
**وغيابُ الخرائط دون اتّصالٍ في تطبيق المتجر ليس سبباً تلقائيّاً لنسخ بنية
السائق إليه.** **والمطلوبُ لاحقاً تحسينُ `PickPoint` نفسِها بقدر الحاجة
الحقيقيّة.**

## فجوةُ أقسام القائمة

**بعد حسم `MD-1` صار غيابُ التعديل والحذف من التطبيق فجوةَ عقدٍ صريحة.**
**ولا تُصلَح الآن.**

## شام كاش

**العقدُ معتمدٌ كما هو** ([`SHAM_CASH_PRODUCT_CONTRACT.md`](SHAM_CASH_PRODUCT_CONTRACT.md)):
**رؤيةُ المستحقّات · طلبُ السحب · ربطُ حسابٍ موثَّق · السحبُ عبر شام كاش ·
متابعةُ الحالة** بحالاتها الخمس `PENDING`/`PROCESSING`/`COMPLETED`/`FAILED`/`REVERSED`.

```
MERCHANT SHAM CASH CONTRACT GAPS = 5   (بلا تبديل)
```

**ولم تقع مصالحةٌ توجب تصحيحَ العدد** — **القائمُ اثنان (`WalletScreen` ·
`POST /me/payouts`) والناقصُ خمسة.** **ولا يُنفَّذ الآن.**

---

# فجواتُ العقد المتراكمةُ لتطبيق المتجر

| ID | الفجوة | مصدرُها | الموضع |
|---|---|---|---|
| **`MG-1`** | **لا سطحَ لتعديل قسم** | `MD-1` | تطبيقُ المتجر |
| **`MG-2`** | **لا سطحَ لحذف قسم** | `MD-1` | تطبيقُ المتجر |
| **`MG-3`** | **لا إشعارَ عند إسناد السائق** | `MD-2` | `orders/notify.go` |
| **`MG-4`** | **لا إشعارَ عند وصوله للاستلام** | `MD-2` | `orders/notify.go` |
| **`MG-5`** | **الإشعارُ يفتح الشاشةَ الأولى لا الطلب** | `MD-2` · `MD-5` | `ui/push/RahalPushService.kt:115` |
| **`MG-6`** | **`PATCH settings` بلا تدقيقٍ وبلا قيمةٍ سابقة** | `MD-3` | `merchant_ops_handlers.go:186` |
| **`MG-7`** | **القبولُ التلقائيُّ صامتٌ عن المتجر** | `MD-5` | `orders/watchdog.go:224` |
| **`PC-10`/متجر** | **الإيماءاتُ معطَّلةٌ عالميّاً** | مراجعةُ المالك | `MainActivity.kt:237` |

```
MERCHANT CONTRACT GAPS AGAINST CODE = 8
```

**و`MG-5` مشتركةٌ مع التطبيقات الأخرى** — **تُصلَح في `:ui` مرّةً واحدة.**
**ولا يُصلَح شيءٌ الآن.**

---

# ولا عدّادَ مجمَّدٌ تبدّل

| البند | أيصير عيباً أو خطراً؟ | لماذا |
|---|---|---|
| **`MG-1`…`MG-7`** | **لا** | **عقودٌ وُضعت اليوم ولم تُبنَ بعد** — `PRODUCT REQUIREMENT` |
| **تعليقُ `sweepAutoAccept` يخالف فعلَها** | **لا** | **وثيقةٌ داخليّةٌ متقادمة** — **تُصحَّح مع بناء `MG-7`، ولا سلوكَ يتبدّل بها** |

```
NEW PROVEN DEFECTS = 0
NEW PROVEN RISKS   = 0
FROZEN COUNTERS    = 19 عيباً · 24 خطراً — بلا تبديل
CODE TRUTH BASELINE = 26f93c5d — سليم
```

---

# الخلاصة

```
MERCHANT APP FINAL OWNER CLOSEOUT COMPLETE

MD-1 MENU SECTION EDIT/DELETE               = CLOSED
MD-2 DRIVER ASSIGNMENT/ARRIVAL NOTIFICATION = CLOSED
MD-3 MERCHANT SETTINGS AUDIT                = CLOSED
MD-5 AUTO-ACCEPT NOTIFICATION               = CLOSED

MERCHANT OPEN OWNER DECISIONS           = 0
MERCHANT OWNER PRODUCT DECISIONS CLOSED = YES

MERCHANT PRODUCT CONTRACT    = COMPLETE
MERCHANT CODE UNDERSTANDING  = COMPLETE
MERCHANT OWNER MANUAL REVIEW = COMPLETE

MERCHANT APP PRODUCT REVIEW = CLOSED

MERCHANT CONTRACT GAPS AGAINST CODE = 8    (MG-1…MG-7 · PC-10/متجر)
MERCHANT PRODUCT REQUIREMENTS       = 4    (MD-1 · MD-2 · MD-3 · MD-5)
MERCHANT SHAM CASH CONTRACT GAPS    = 5    (بلا تبديل)
MERCHANT UX/PERFORMANCE GAPS        = 2    (الدرج · أوّلُ خريطة)
MERCHANT RUNTIME CHECKS REQUIRED    = 4    (P-M1 · P-M2 · RV-4 · RV-M1)

MERCHANT FUNCTIONAL VALIDATION     = PENDING
MERCHANT PROGRAMMATIC VALIDATION   = PENDING
MERCHANT SECURITY/ABUSE VALIDATION = PENDING
MERCHANT PERFORMANCE/UX VALIDATION = PENDING
MERCHANT REAL DEVICE VALIDATION    = PENDING

MERCHANT APP VERIFIED         = NO
MERCHANT READY FOR PRODUCTION = NOT YET EVALUATED

OPERATIONAL CODE CHANGES = 0
```
