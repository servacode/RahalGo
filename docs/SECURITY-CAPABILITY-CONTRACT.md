# عقدُ التخويل — من يملك ماذا، وبأيّ إثبات

> **نصفُ هذه الوثيقة مولَّدٌ من الشيفرة** — كلُّ ما بين `<!-- gen:… -->`
> يُكتب بـ`go run ./cmd/authzdoc` **ولا يُحرَّر بيد**، ويحرسه
> `TestAuthzDocIsCurrent`.
>
> **ومعها ملفٌّ يُقرأ بالآلة**: `docs/testing/system/AUTHZ_CONTRACT.json`.

<!-- gen:counts -->
**31 قدرةً · 185 صفَّ سياسةٍ للمسارات · 2 استثناءً · 24 فعلاً حسّاساً · 7 حقلاً محروساً · 1 قدرةً حقليّةً لا تحرس باباً · 0 قدرةً لا تحرس شيئاً.**
<!-- /gen:counts -->

---

## أربعُ طبقاتٍ لا طبقة

**ومن ظنَّها طبقةً واحدةً أخطأ في كلّ سؤال.**

| الطبقة | ما تحكم | موضعُ الإنفاذ |
|---|---|---|
| معجمُ القدرات | نصوصُ القدرات المعروفة | `internal/authz/catalog.go` |
| سياسةُ المسارات | طريقةٌ + نمطُ مسارٍ ⇒ قدرة | `internal/server/capability.go:163` |
| الإثباتُ (step-up) | فعلٌ + بصمةُ جسمِ الطلب | `internal/server/stepup.go:57` |
| تشكيلُ الردّ | أسماءُ الحقول في الجواب | `internal/server/response_shape.go:53` |

**وترتيبُها ترتيبُ الوسائط نفسُه** (`server.go:934`): جلسةٌ، ثمّ بابٌ
(قدرةٌ واحدةٌ على الأقلّ)، ثمّ قدرةُ المسار، ثمّ الإثبات، ثمّ التشكيل.
**والتشكيلُ آخرُها لزوماً** — لأنّه يقرأ القدراتَ من السياق
(`response_shape.go:52`).

---

## تصحيحٌ يُقال أوّلاً — **الرمزُ السرّيُّ ليس إثباتاً**

**كنتُ أقول إنّ النظامَ «رمزٌ سرّيٌّ ورفعُ صلاحيةٍ بكلمةِ مرورٍ» كأنّهما
بابان لفعلٍ واحد. وهذا خطأ.**

**الرمزُ السرّيُّ عاملٌ ثانٍ عند الدخول** — ويُحكم **بالدور** لا بالمسار:
`NeedsPin(roles)` تردّ صحيحاً لمن حمل `admin` أو `owner_super_admin`
(`identity/admin_pin.go:93`)، **ويُنفَّذ عند إصدار الجلسة**: يردّ المحرّكُ
`{PinRequired, Challenge}` **بلا أيّ رمزِ وصول** (`identity/service.go:819`).
أربعةُ أرقامٍ · خمسُ محاولاتٍ · قفلٌ عشرَ دقائق · مهلةُ تحدٍّ خمسُ دقائق.

**والإثباتُ كلمةُ مرورٍ لا رمزاً** — `IssueStepUp` يتحقّق من
`users.password_hash` وحدَه (`identity/stepup.go:81`). **ولا فرعَ للرمز
السرّيّ في مسار الإثبات إطلاقاً.**

**فالنتيجةُ**: الرمزُ مربوطٌ بالدور وبالدخول، **والإثباتُ مربوطٌ بمسارٍ
وجسمِ طلب**. **ولا نقطةَ واحدةٌ في المنصّة تطلب الرمزَ السرّيّ.**

---

## القدرات

<!-- gen:capabilities -->
| القدرة | تحرس مسارات | تحرس حقولاً | الوصف |
|---|---|---|---|
| `analytics.read` | 8 | — | قراءةُ التحليلات |
| `audit.read` | 1 | — | قراءةُ سجلّ التدقيق |
| `content.manage` | 28 | — | لافتاتٌ وعروضٌ ومحتوى |
| `drivers.manage` | 1 | — | إدارةُ السائقين وتشغيلُهم |
| `drivers.read` | 2 | — | قراءةُ سجلّ السائقين ومواضعهم |
| `emergencies.manage` | 5 | — | قراءةُ الطوارئ واستلامُها |
| `finance.export` | 2 | — | سحبُ الدفتر وكشفِ الطلبات ملفّاً |
| `finance.manage` | 13 | — | قيدُ محفظةٍ ومصروفٌ وخزينة |
| `finance.read` | 16 | — | قراءةُ المال والتقارير الماليّة |
| `finance.recompute` | 1 | — | إعادةُ حساب تسوية طلبٍ مُغلق |
| `merchants.manage` | 10 | — | إدارةُ المتاجر وتعليقُها |
| `merchants.read` | 4 | — | قراءةُ سجلّ المتاجر وقوائمها |
| `merchants.verify` | 2 | — | مراجعةُ المرشَّحين والقوائم |
| `observability.read` | 1 | — | قراءةُ صحّة المنصّة الداخليّة |
| `orders.communications.read` | 3 | — | قراءةُ محادثات الطلب ورسائله |
| `orders.intervene` | 11 | — | تدخّلٌ في طلبٍ نيابةً عن طرفه |
| `orders.read` | 11 | — | قراءةُ الطلبات ولوحةِ العمليّات |
| `payouts.decide` | 1 | — | قرارُ السحب |
| `platform.overview` | 1 | — | رئيسيّةُ مدير المنصّة بأرقامها ومالِها |
| `roles.manage` | 7 | — | منحُ الأدوار وسحبُها |
| `safety.manage` | 8 | — | الإنذاراتُ والمخالفاتُ وتعليقُ المتاجر |
| `settings.financial.manage` | 2 | — | إعداداتٌ تدخل حساباً ماليّاً |
| `settings.general.manage` | 21 | — | إعداداتٌ عامّةٌ ومحتوى |
| `settings.read` | 2 | — | قراءةُ لوح الإعدادات |
| `settings.security.manage` | 3 | — | إعداداتُ الأمن والجلسات |
| `support.manage` | 8 | — | التذاكرُ والنزاعاتُ والطوارئ |
| `users.contact.read` | **0** | **7** | قراءةُ رقم الاتّصال |
| `users.export` | 1 | — | سحبُ دليل الحسابات ملفّاً |
| `users.read` | 6 | — | قراءةُ الحسابات |
| `users.sensitive.read` | 2 | — | قراءةُ عناوين المرء وأثرِه |
| `users.status.manage` | 4 | — | إيقافُ حسابٍ أو حظرُه أو تبديلُ بياناته |
<!-- /gen:capabilities -->

**وقدرةٌ تحرس صفرَ مساراتٍ ليست سهواً بالضرورة.** `users.contact.read` لا
تحرس باباً واحداً — **تحرس حقلاً في الردّ**: `shapeResponse` يحذف الهاتفَ
من الجواب لمن لا يملكها (`response_shape.go:57`). **وهي قرارٌ منصوص**:
«حقلٌ في ردٍّ لا بابٌ في موجّه» (`catalog.go:182`)، **ويستثنيها حارسُ
القدرات اليتيمة صريحاً** (`role_matrix_test.go:490`).

**فالقدرةُ ⇒ المسارُ ليست دالّةً كليّة.** ومن بنى مصفوفةَ قبولٍ على أنّها
كذلك **حسب `users.contact.read` عيباً.**

### سياسةُ الحقول

<!-- gen:field-policy -->
| الحقل | يحتاج |
|---|---|
| `customer_phone` | `users.contact.read` |
| `driver_phone` | `users.contact.read` |
| `owner_phone` | `users.contact.read` |
| `party_phone` | `users.contact.read` |
| `phone` | `users.contact.read` |
| `sales_rep_phone` | `users.contact.read` |
| `user_phone` | `users.contact.read` |
<!-- /gen:field-policy -->

**و`merchant_phone` مستثنىً عمداً** (`catalog.go:226`) — **فهاتفُ المتجر
ليس بيانةً شخصيّة**، ومن حجبه عطّل العملَ.

---

## الاستثناءاتُ — وأسبابُها

<!-- gen:exemptions -->
| النمط | السبب |
|---|---|
| `/settings/{key}` | **القدرةُ تتبع المفتاحَ لا المسار** — عامٌّ أو ماليٌّ أو أمنيّ. **وتُحسَم في المعالِج** (`settingCapability`). |
| `/step-up` | **قدرتُه قدرةُ الفعل المُؤكَّد** — تُقاس في المعالِج من `LookupAdmin` نفسِها. (`ADG-3`.) |
<!-- /gen:exemptions -->

**واثنان لا أكثر**، ولكلٍّ سببٌ مكتوب. **و`api/contract.json` يطويهما في
`«بحسب المفتاح»`** (`apidoc/routes.go:340`) — **فيضيع السبب.** فيُقال هنا.

**ومسارٌ لا سياسةَ له ولا استثناءَ ⇒ يُمنَع** (`policy.go:18`) — لا يُسمح.

---

## الأفعالُ الحسّاسة — ما لا يقع بلا إثبات

<!-- gen:sensitive -->
| الفعل | الطريقة | المسار | بصمةُ الجسم | بشرط | القدرة |
|---|---|---|---|---|---|
| `admin.merchant_delivery_credit` | `PATCH` | `/merchants/{id}/delivery-credit` | `limit` | دائماً | `settings.financial.manage` |
| `admin.merchant_settlement_update` | `PATCH` | `/merchants/{id}/settlement-method` | `method` | دائماً | `settings.financial.manage` |
| `admin.password_reset` | `POST` | `/users/{id}/password` | — | دائماً | `users.status.manage` |
| `admin.role_capability_grant` | `POST` | `/roles/{code}/capabilities` | `capability` | دائماً | `roles.manage` |
| `admin.role_capability_revoke` | `DELETE` | `/roles/{code}/capabilities/{cap}` | — | دائماً | `roles.manage` |
| `admin.role_create` | `POST` | `/roles` | `code` | دائماً | `roles.manage` |
| `admin.role_grant` | `POST` | `/users/{id}/roles` | `role` | دائماً | `roles.manage` |
| `admin.role_revoke` | `DELETE` | `/users/{id}/roles/{role}` | — | دائماً | `roles.manage` |
| `admin.setting_update` | `PUT` | `/settings/{key}` | `value` | `settingSensitivity` | *بحسب المفتاح* |
| `admin.user_update` | `PATCH` | `/users/{id}` | `status` | `statusIsStrong` | `users.status.manage` |
| `finance.compensate_driver` | `POST` | `/orders/{id}/compensate-driver` | `amount` | دائماً | `finance.manage` |
| `finance.driver_settle` | `POST` | `/drivers/{id}/settle` | `amount` | دائماً | `finance.manage` |
| `finance.expense_added` | `POST` | `/expenses` | `amount`, `category_id` | دائماً | `finance.manage` |
| `finance.expense_voided` | `POST` | `/expenses/{id}/void` | — | دائماً | `finance.manage` |
| `finance.goods_compensation` | `POST` | `/orders/{id}/goods/compensation` | `amount` | دائماً | `finance.manage` |
| `finance.incentive` | `POST` | `/users/{id}/incentive` | `amount` | دائماً | `finance.manage` |
| `finance.merchant_cash_paid` | `POST` | `/merchant-cash-settlements/{id}/pay` | — | دائماً | `finance.manage` |
| `finance.payout_decide` | `POST` | `/payouts/{id}/decide` | `approve`, `amount` | دائماً | `payouts.decide` |
| `finance.settlement_recomputed` | `POST` | `/orders/{id}/recompute` | — | دائماً | `finance.recompute` |
| `finance.ticket_resolve` | `POST` | `/tickets/{id}/resolve` | `compensation` | دائماً | `support.manage` |
| `finance.wallet_apply` | `POST` | `/users/{id}/wallet` | `amount`, `kind` | دائماً | `finance.manage` |
| `ops.delivery_proof_exception` | `POST` | `/orders/{id}/proof-exception` | `reason` | دائماً | `orders.intervene` |
| `ops.dispute_settled` | `POST` | `/disputes/{id}/settle` | `settlement` | دائماً | `finance.manage` |
| `ops.order_transition` | `POST` | `/orders/{id}/transition` | `to` | `transitionRefund` | `orders.intervene` |
<!-- /gen:sensitive -->

**والبصمةُ هي الفكرةُ كلُّها.** الإذنُ يُربط بـ
`sha256(method + "\n" + path + "\n" + material)` (`identity/stepup.go:44`)،
**و`material` حقولٌ بعينها من الجسم مرتَّبةً** (`authz/sensitive.go:184`).
**فمن أثبت لتحويلِ خمسينَ لا يملك تحويلَ خمسمئة** — تبدّلَ الجسمُ فتبدّلت
البصمةُ فلم يُطابق إذنٌ.

**والمسارُ يدخل البصمةَ كاملاً لا نمطاً** (`stepup.go:41`) — **فإذنٌ على
مستخدمٍ ليس إذناً على غيره**، بلا حاجةٍ إلى فحصِ هدفٍ منفصل.

**وخمسُ دقائقَ ومرّةٌ واحدة** (`StepUpTTL`, `identity/stepup.go:25`).
**ولا يُرَدُّ الاستهلاكُ إن فشل الفعلُ بعده** — قرارٌ منصوص
(`stepup.go:132`): **لأنّ إذناً يُرَدُّ عند الفشل إذنٌ يُعاد استعمالُه
بإفشالٍ مقصود.**

**والملاحظاتُ الحرّةُ خارجَ البصمة** — `note` ليست في `Params` عمداً
(`sensitive.go:47`): **فتبديلُ التعليل لا يُبطل إذناً، وتبديلُ المبلغِ
يُبطله.**

**وشرطٌ لا يُعرَف اسمُه ⇒ يُطلب الإثبات** (`stepup.go:151`) — **يُغلَق عند
الشكّ لا يُفتَح.**

---

## الأدوار — تصنيفُها وسلطةُ منحها

<!-- gen:role-classes -->
| الصنف | الأدوار | سلطةُ المنح | يُنشأ بالتسجيل؟ |
|---|---|---|---|
| `protected` | `owner_super_admin` | `owner` | لا |
| `elevated` | `admin` | `owner` | لا |
| `staff` | `analytics`, `customer_support`, `driver_verification`, `finance`, `marketing_content`, `merchant_verification`, `observability`, `operations`, `platform_monitor`, `trust_safety` | `roles.manage` | لا |
| `account_type` | `customer`, `driver`, `merchant`, `sales` | `roles.manage` | **نعم** |
| `legacy` | `ops` | `never` | لا |
<!-- /gen:role-classes -->

**و`ops` دورٌ متقاعد**: قائمُه يعمل **ولا يُمنَح جديداً**
(`role_grant_retired`). **وقدراتُه الأربعَ عشرةَ أوسعُ من `operations`
التسع** — فمن بقي عليه أوسعُ ممّن انتقل.

### ⚠️ ودورٌ مصنَّفٌ لا وجودَ له

**`observability` في التصنيف** (`roleclass.go:78`) **ولا هجرةَ تُنشئه.**
**و`observability.read` لا تُمنَح لدورٍ واحدٍ في أيّ هجرة** — حتّى
`owner_super_admin` يملك سبعاً وعشرينَ من ثمانٍ وعشرين، **والناقصةُ هي
هذه بعينها.**

**فالنتيجةُ على قاعدةٍ مهاجَرةٍ حديثاً**:
`GET /api/v1/admin/ops/health` **لا يبلغه أحد — ولا المالك** — حتّى يُنشئ
أحدٌ دوراً من اللوحة ويمنحه القدرة.

> **يُسجَّل بنداً في المصفوفة** (`GAP-SEC-01`) — **ولا يُصلَح قبل أن تُثبت
> الدورةُ حاجتَه** (حكمُ المالك السادس).

---

## مصفوفةُ الدور ⇒ القدرة — **ليست في هذا الملفّ، ولها سبب**

**لأنّها ليست في الشيفرة.** `role_capabilities` جدولٌ في القاعدة
(`0137_role_capabilities.sql:28`)، **يُبدّله كلُّ من يملك `roles.manage`
من اللوحة** (`rbac_admin_handlers.go:113`).

**فما تبذره الهجراتُ السبعُ حالةٌ أولى لا عقد**: **١٠٦ صفّاً على ١١ دوراً**
بعد تشغيلها كلِّها بالترتيب، وكلُّها `ON CONFLICT DO NOTHING` تُشغَّل مرّةً
**فلا تدهس تعديلَ اللوحة** (`0138_canonical_roles.sql:7`).

**ومولِّدٌ يقرأ قاعدةً يكذب مرّتين**: يكذب على من لا قاعدةَ له، **ويُثبّت
في وثيقةٍ ما يتغيّر بنقرة.** فيُقرأ الحاضرُ من القاعدة لا من هنا:

```sql
SELECT role_code, count(*) FROM role_capabilities GROUP BY 1 ORDER BY 2 DESC;
```

**والقدراتُ تُقرأ لكلّ طلبٍ من القاعدة لا من الرمز** (`repo.go:1191`) —
**فنزعُ دورٍ يسري في الطلب التالي**، ولا ينتظر انتهاءَ رمزٍ.

---

## طبقاتُ خريطة العمليات

<!-- gen:ops-map-perms -->
| القدرة | الطبقاتُ التي تفتحها |
|---|---|
| `analytics.read` | `VIEW_DEMAND_ANALYTICS` |
| `drivers.read` | `VIEW_DRIVER_LOCATIONS` |
| `finance.read` | `VIEW_MAP_FINANCIALS` |
| `merchants.manage` | `VIEW_REP_ACTIVITY` |
| `orders.read` | `VIEW_OPERATIONS_MAP`, `VIEW_MERCHANT_LOCATIONS`, `VIEW_ACTIVE_ORDERS` |
| `settings.general.manage` | `MANAGE_COVERAGE`, `MANAGE_BRANCHES` |
<!-- /gen:ops-map-perms -->

**وهذه بوّابةٌ ثانيةٌ فوق سياسة المسارات** لا بدلاً منها
(`opsmap_handlers.go:33`). **و`capPerms` غيرُ مُصدَّرةٍ في Go** — فهذا الجدولُ
مُستنتَجٌ بسؤال `opsmap.Allows` عن كلّ قدرةٍ وكلّ طبقة، **لا منسوخٌ في
قائمةٍ ثانيةٍ تشيخ.**

---

## المنعُ هو الأصل — في ستّة مواضع

1. **قدرةٌ مجهولةٌ ⇒ منع** قبل أيّ بحث (`capability.go:80`).
2. **قدرةٌ مجهولةٌ في بناء المسار ⇒ لا يُقلِع المحرّكُ أصلاً** — `panic`
   (`capability.go:58`). **فالخطأُ يظهر عند الإقلاع لا عند الطلب.**
3. **مسارٌ إداريٌّ بلا سياسةٍ ولا استثناءٍ ⇒ 403** (`capability.go:172`).
4. **صفرُ قدراتٍ ⇒ لا يُدخَل السطحُ الإداريُّ أصلاً** (`capability.go:141`).
5. **دورٌ بلا صفوفٍ لا يملك شيئاً**، والدورُ الجديدُ يبدأ فارغاً
   (`rbac_admin_handlers.go:196`).
6. **رمزٌ بلا جلسةٍ ⇒ قدراتٌ فارغة**، **وعطبُ البنيةِ ⇒ 503 لا سماح**
   (`session_check.go:130`).

**ورسالةُ المنع لا تُفصح عن القدرة الناقصة** (`capability.go:41`) —
**تُسجَّل ولا تُقال**، فلا يُستدَلُّ بالرفض على الخريطة.

---

## وأينَ يبقى الخطرُ فعلاً

**فليس في السطح الإداريّ.** كلُّ نقطةٍ إداريّةٍ تحمل قدرةً أو استثناءً
مكتوباً، **ويحرسها اختبارٌ يمشي على الموجّه الحقيقيّ**
(`role_matrix_test.go:583`) فيُسقط البناء.

**الخطرُ في تسعٍ وخمسينَ نقطةً موثَّقةً بلا دورٍ وبلا قدرة**:
`/my/*` · `/me/*` · `/auth/*` · `/orders/*` · `/geo/*` · `/demand/*` ·
`/promo/*`. **تخويلُها منطقُ ملكيّةٍ داخل كلّ معالِج** — **لا جدولَ مركزيّاً
له ولا حارسَ يمشي على الموجّه.** فمن نسي فحصَ ملكيّةٍ في واحدةٍ منها
**لم يُسقط بناءً.**

> **يُسجَّل بنداً في المصفوفة** (`GAP-SEC-02`).

**وسبعون نقطةً أخرى محروسةٌ بالدور لا بالقدرة** (`merchant` ٣٠ ·
`driver` ٢٣ · `sales` ١٧) — **وذلك مقصود**: سطوحُ التطبيقات لا يحكمها
معجمُ القدرات.

### وبابُ QA يتخطّى السطحَ الإداريَّ كلَّه

**`/api/v1/qa/*` مسجَّلةٌ خارجَ `/admin`** (`server.go:499`) — **فلا
`RequireAnyCapability` ولا سياسةَ مسارٍ ولا إثبات** — **وتنادي معالِجاتِ
الأدمن الحقيقيّةَ عينَها.**

**ولا تُسجَّل في الإنتاج أصلاً** (`if s.qaStagingEnabled()`, `server.go:480`)
⇒ **404 لا 403**، ومعها حارسٌ ثانٍ في المعالِج وتحذيرٌ عند الإقلاع.
**فالتخطّي مقصودٌ ومحدودٌ ببيئةٍ** — **ويُذكر هنا لأنّ عقداً لا يذكره
يُقرأ كأنّ هذه المسارات لا وجودَ لها.**

---

## كيف يُعاد التوليد

```
cd backend && go run ./cmd/authzdoc
go test ./internal/authzdoc/ -count=1
```
