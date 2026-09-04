# رحّال غو — خريطةُ التدقيق للأفعال الحسّاسة

> **CODE TRUTH BASELINE**: `26f93c5d`
>
> **المطلوب**: `ACTION → AUDIT EVENT` أو `ACTION → NO AUDIT + السبب`.
> **ولا يُعدُّ غيابُ التدقيق عيباً إلّا إن خالف ثابتاً مُثبَتاً.**

---

## ٠ · المقامات

| | العدد |
|---|---|
| **أسماءُ أفعالٍ مدقَّقةٍ فريدة** | **١٠٤** |
| **طبقاتُ الكتابة في `audit_log`** | **٤** |
| **الطفراتُ الحسّاسة** (مالٌ · صلاحيّةٌ · حالةٌ · إعدادات) | **٣٥** |
| ↳ **مدقَّقةٌ بدليلٍ مباشر** | **٢٩** |
| ↳ **بلا تدقيقٍ — بسببٍ مُثبَت** | **٦** |

## ١ · طبقاتُ الكتابة الأربع

| الطبقة | الدالّة | ما تُدقّق |
|---|---|---|
| `internal/server/audit.go` | `s.audit(r, ...)` | أفعالُ المُعالِجات — **٤١ موضعاً** |
| `internal/server/admin_users_handlers.go` | `INSERT` مباشر | `admin.password_reset` |
| **`internal/identity/repo.go`** | `s.repo.Audit(ctx, ...)` | **الهويّةُ والأدوارُ والجلسات** |
| **`internal/catalog/catalog.go`** | `s.audit(ctx, ...)` | **المتاجرُ والمناطقُ والساعاتُ والعروض** |

**وهذا يفسّر ادّعائي الساقط**: **قِست ٤٧ في طبقةٍ واحدةٍ فظننتُها الكلّ.**

---

## ٢ · المال — **١٤ فعلاً · كلُّها مدقَّقة** ✅

| الفعل | حدثُ التدقيق |
|---|---|
| `POST /admin/users/{id}/wallet` | ✅ (`finance.*`) |
| `POST /admin/users/{id}/incentive` | ✅ |
| `POST /admin/payouts/{id}/decide` | ✅ |
| `POST /admin/drivers/{id}/settle` | ✅ |
| **`POST /admin/orders/{id}/recompute`** | ✅ **`finance.settlement_recomputed`** مع `before/after/delta` |
| **`POST /admin/orders/{id}/compensate-driver`** | ✅ **`finance.driver_compensation`** |
| `POST /admin/orders/{id}/goods` | ✅ **`finance.goods_settled`** |
| `POST /admin/disputes` · `/{id}/settle` | ✅ `ops.dispute_opened` · `ops.dispute_settled` |
| `POST /admin/expenses` · `/{id}/void` · `/categories` | ✅ (`finance.*`) |
| `POST /driver/orders/{id}/return` | ✅ **`driver.order_returned`** مع المبلغ |
| **`POST /admin/orders/{id}/settle-goods`** | **NO AUDIT — مُبرَّر**: **مسارٌ مهجورٌ يردّ ٤١٠ قبل أيّ عمل** ✅ |

> **`MONEY ACTIONS AUDITED: 13/13 ACTIVE`** ✅ (والرابعَ عشرَ مهجور)

## ٣ · الصلاحيّاتُ والهويّة — **١٠ · كلُّها مدقَّقة** ✅

**وهي التي بدت بلا تدقيقٍ في قياسٍ سطحيّ — والدليلُ في `internal/identity`:**

| الفعل | حدثُ التدقيق | الموضع |
|---|---|---|
| `POST /admin/users` | **`admin.user_create`** | `identity/admin.go` |
| `POST /admin/users/{id}/roles` | **`admin.role_grant`** | `identity/admin.go` |
| `DELETE /admin/users/{id}/roles/{role}` | **`admin.role_revoke`** | `identity/admin.go` |
| `POST /admin/users/{id}/logout-all` | **`admin.logout_all`** مع عدد الجلسات | `identity/admin.go` |
| `POST /admin/users/{id}/password` | **`admin.password_reset`** | `admin_users_handlers.go` |
| `POST /auth/logout` | **`auth.logout`** | `identity/service.go` |
| **إنشاءُ مالك متجرٍ ضمنَ التحويل** | **`admin.user_create`** | `identity/service.go` — **من داخل `EnsureUserWithRole`** ✅ |
| `PATCH /admin/users/{id}` | ✅ (`admin.*`) | |
| `POST /admin/users/{id}/warnings` · `/merchants/{id}/warnings` | ✅ | |
| `POST /admin/merchants/{id}/suspend` · `/clear-violations` | ✅ **`ops.merchant_suspend`** | |

> **`PERMISSION ACTIONS AUDITED: 10/10`** ✅
> **والتدقيقُ في طبقة الخدمة أمتنُ**: **يقع حيث يقع الفعلُ الحقيقيّ**،
> فلا يُنسى إن نُودي من بابين — وقد وقع ذلك فعلاً: **إنشاءُ المالك
> يُدقَّق سواءٌ جاء من اللوحة أو من تحويل فرصة.**

## ٤ · انتقالاتُ الطلب

| الفعل | التدقيق | السبب |
|---|---|---|
| **`POST /admin/orders/{id}/transition`** | ✅ **`ops.order_transition`** مع `to` و`note` | موظّفٌ يحرّك نيابةً |
| `POST /admin/orders/{id}/assign` | ✅ | |
| `POST /admin/orders/{id}/transfer` | ✅ | |
| **`POST /driver/orders/{id}/transition`** | **NO AUDIT — مُبرَّرٌ بقرارٍ مكتوب** | **«ولا يُسجَّل انتقالُ الأطراف أنفسهم — المتجر يقبل مئة طلب في اليوم، وتسجيلُها يُغرق السجلّ فيصير لا يُقرأ»** |
| **`POST /merchant/orders/{id}/transition`** | **NO AUDIT — نفسُ السبب** | ✅ |
| `POST /driver/orders/{id}/decline` | ✅ **`driver.offer_declined`** — **استثناءٌ مقصود** | رفضُ العرض يُقاس |

> **PROVEN BEHAVIOR** — **وليس عيباً**: الأثرُ محفوظٌ في **`order_events`**
> (من · إلى · الفاعل · العلّة) **لكلّ انتقالٍ بلا استثناء** ✅ —
> **فالتاريخُ لا يضيع، والتدقيقُ لأفعال الموظّفين وحدَها.**
>
> **⚠️ إلّا `DSW-2`** — **فكُّ الإسناد لا يكتب `order_events`** → **D1**.

## ٥ · الإعداداتُ والكتالوج

| الفعل | التدقيق |
|---|---|
| `PUT /admin/settings/{key}` | ✅ (`platform.*`) |
| `POST/PUT/DELETE /admin/zones` | ✅ **`admin.zone_{create,update,delete}`** — `catalog/zones.go` |
| `POST/PATCH /admin/promos` | ✅ **`admin.promo_{create,update}`** |
| `PUT /merchant/stores/{id}/hours` | ✅ **`merchant.hours_update`** |
| **`PATCH /merchant/stores/{id}/settings`** | **NO AUDIT** ⚠️ — **PROVEN RISK**: صاحبُ المتجر يبدّل إعداداتِ متجره بلا أثر |
| `POST /admin/merchants` · `PATCH` | ✅ (`catalog.*`) |

## ٦ · بلا تدقيقٍ — **بسببٍ مُثبَت**

| # | الفعل | السبب | التصنيف |
|---|---|---|---|
| ١ | `POST /auth/password/reset/{request,verify,confirm}` | **أفعالُ صاحبِ الحساب لا موظّف** · والرمزُ في `otp_codes` بأثره | **PROVEN BEHAVIOR** |
| ٢ | `POST /auth/password` | كسابقه | **PROVEN BEHAVIOR** |
| ٣ | `POST /driver/orders/{id}/transition` | **قرارٌ مكتوب** — والأثرُ في `order_events` | **PROVEN BEHAVIOR** |
| ٤ | `POST /merchant/orders/{id}/transition` | كسابقه | **PROVEN BEHAVIOR** |
| ٥ | `POST /admin/orders/{id}/settle-goods` | **مهجورٌ — ٤١٠** | **PROVEN BEHAVIOR** |
| ٦ | **`PATCH /merchant/stores/{id}/settings`** | **لم يُوجد سبب** | **PROVEN RISK** — **قرارُ مالك** |

---

## الحكم

> **`SENSITIVE ACTIONS AUDIT-MAPPED: 35/35`** ✅
> **`AUDITED: 29`** · **`NO-AUDIT WITH PROVEN REASON: 5`** ·
> **`NO-AUDIT WITHOUT REASON: 1`** → **PROVEN RISK**
> **`AUDIT DEFECTS: 0`** — **لا يخالف أيٌّ منها ثابتاً مُثبَتاً**
