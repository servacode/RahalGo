# رحّال غو — كشفُ المنظومة (Discovery / Read-Only Audit)

> **أُنشئ بطلب المالك نصّاً** (٢٠٢٦-٠٩-٠٤): «أريد أوّلًا أن تفهم المنظومة
> الحالية فهمًا كاملًا ودقيقًا من الكود نفسه… ممنوع الاعتماد على أسماء
> الملفات فقط أو الافتراضات».
>
> **ولم يُعدَّل سطرٌ واحدٌ من الشيفرة في هذه المرحلة** — قراءةٌ وقياسٌ فقط.

---

## علاماتُ الإثبات

| العلامة | معناها |
|---|---|
| **✅ مُثبَت** | قُرئ من الشيفرة أو قِيس من الإنتاج — ومعه مرجعٌ يُعاد إليه |
| **⚠️ مُستنتَج** | استُدلّ عليه ولم يُقرأ صراحةً — **لا يُعامَل حقيقة** |
| **❓ مجهول** | لم يثبت من الشيفرة — ويحتاج قياساً أو قرارَ مالك |

---

# ١ · Executive Summary

**رحّال غو منصّةُ توصيلٍ سوريّة تعمل في الرقّة**، على خادمٍ واحدٍ في
Hetzner، بستّ حاويات، وأربعةِ تطبيقاتِ أندرويد، ولوحةِ إدارةٍ على الويب.

**ما قِيس في هذا الكشف:**

| | العدد | المرجع |
|---|---|---|
| مساراتُ API | **٢٨٤** | `backend/internal/server/server.go` |
| جداولُ الإنتاج | **٧٧** (منها ١٩ لواتساب) | قِيس من الإنتاج |
| هجراتُ القاعدة | **١٢٦** | `backend/internal/migrate/migrations/` |
| اختباراتُ Go | **١٩٨ ملفّاً** | `backend/**/*_test.go` |
| حرّاسُ الويب | **٢٥** | `web/package.json` → `check:guards` |
| اختباراتُ أندرويد | **٥٣ ملفّاً** | موزّعةٌ على الوحدات |
| اختباراتُ الويب | **صفر** | ✅ مُثبَت — لا `*.test.*` في `web/` |
| عمّالُ الخلفيّة | **واحد** | `orders.RunWatchdog` كلَّ ٣٠ ثانية |

**وأخطرُ ما يميّز هذه المنظومة**: **الحقيقةُ المالية دفترٌ مزدوج**
(`wallet_transactions`) **مع عمودِ رصيدٍ مصان** (`wallets.balance`) —
والاثنان يجب أن يتطابقا دائماً.

---

# ٢ · Repository / System Map ✅

```
RahalGo/
├── backend/        محرّك Go — chi · pgx · Redis
│   ├── cmd/        api · apidoc · seed · moneycheck · truthdoc · mediacheck …
│   └── internal/   ٢٩ حزمة (انظر أدناه)
├── web/            Next.js — pnpm/turbo · تطبيقٌ واحد `apps/rahalgo`
│   └── packages/   ui · auth · i18n · tsconfig
├── mobile/         Gradle — ٤ تطبيقات + ٦ وحدات مشتركة
├── maps/           بناءُ الخرائط — Planetiler · PMTiles · نمطٌ واحد
├── routing/        OSRM
├── deploy/         compose.yml · Caddyfile
├── tools/          navigation-voice-pack (ElevenLabs)
├── scripts/        e2e · qa · release-gate
└── docs/           TRUTH · GROUND-RULES · WORKLOG · README · FIELD-TESTS
```

## الحاويات الستّ ✅ (`deploy/compose.yml`)

| الحاوية | ما هي |
|---|---|
| `caddy` | البوّابة — ٨٠/٤٤٣ · TLS |
| `api` | المحرّك (Go) — يُبنى من `backend/` |
| `web` | Next.js — يُبنى من `web/` |
| `postgres` | **PostGIS 16-3.4** — الحقيقة |
| `redis` | ذاكرةٌ عابرة |
| `osrm` | المسارات — بخريطة سوريا |

**ولا منفذَ مكشوفٌ إلّا ٢٢ و٨٠ و٤٤٣** ✅ (`docs/DEPLOY.md`).

## حزمُ المحرّك — ٢٩ ✅

`apidoc · auth · cashbox · catalog · comms · config · database ·
deploycheck · geo · httpx · identity · incentives · media · migrate ·
notifications · notify · offers · orders · pricing · push · qa ·
realtime · referrals · routing · server · settings · support · testdb ·
truthdoc · wallet`

## وحداتُ أندرويد ✅

| الوحدة | ملفّات `main` | اختبارات |
|---|---|---|
| `app-customer` | ٢٤ | ١ |
| `app-driver` | ٣٩ | ٦ |
| `app-merchant` | ١٣ | **٠** |
| `app-rep` | ١٤ | **٠** |
| `ui` (مشترك) | ٦٣ | ٣ |
| `shared` (عقدُ الـAPI) | ٢٢ | ١ |
| `map` | ٢٥ | ١٤ |
| `driver-navigation` | ٤١ | **٢٨** |
| `design` · `brand` | ٥ · ٠ | ٠ |

## الخدماتُ الخارجيّة ✅

| الخدمة | الاستعمال | المرجع |
|---|---|---|
| **FCM** | الإشعاراتُ الدافعة | `internal/push/fcm.go` — **بلا حزمة Firebase**، نداءُ REST مباشر |
| **whatsmeow** | **عميلُ واتساب مضمَّنٌ في المحرّك** | `go.mod` · ١٩ جدولاً في القاعدة |
| **مزوّدُ SMS** | رمزُ التحقّق — احتياطيّ | `internal/notify/sms.go` |
| **ElevenLabs** | توليدُ مقاطع الملاحة — **مرّةً واحدة** | `tools/navigation-voice-pack` |
| **Nominatim (OSM)** | البحثُ عن مكانٍ بالاسم | `internal/geo` → `/geo/search` |
| **OSRM** | المسارات — **ذاتيُّ الاستضافة** | حاوية `osrm` |
| **Geofabrik / OSM** | بياناتُ الخريطة | `maps/config/build.env` → `AREA=syria` |

**ولا خدمةَ خرائطَ تجاريّة** — لا غوغل ولا ماببوكس. ✅

## التخزين ✅

**الوسائطُ على قرصٍ** (`internal/media`) — مجلّدُ `uploads` مثبَّتٌ
كحجمٍ في Compose. **ولا S3 ولا CDN.**

---

# ٣ · Applications ✅

| التطبيق | الحزمة | الدور |
|---|---|---|
| الزبون | `com.rahalgo.customer` | يطلب ويتابع ويقيّم |
| السائق | `com.rahalgo.driver` | يقبل · يوصّل · يلاحظ الملاحة |
| المتجر | `com.rahalgo.merchant` | يقبل الطلبات ويدير قائمته |
| المندوب | `com.rahalgo.rep` | **يجلب المتاجر — لا يوصّل** |
| لوحةُ الإدارة | `web/apps/rahalgo` → `/dashboard` | التشغيلُ والمالُ والإعدادات |
| الموقعُ العامّ | نفسُ التطبيق → `(site)` | **هويّةٌ وروابطُ تحميل فقط** (٢٠٢٦-٠٩-٠٤) |

---

# ٤ · Actors & Roles ✅

**سبعةُ أدوارٍ مُعرَّفةٌ في القاعدة** — `0002_identity.sql`:

```
customer · driver · merchant · sales · ops · finance · admin
```

**والأدوارُ تُخزَّن في `user_roles`** (مفتاحٌ مركّب `user_id, role_code`) —
**فالمستخدمُ الواحدُ قد يحمل أكثرَ من دور.** ✅ (قِيس في الإنتاج:
حساباتٌ تحمل `customer,merchant` و`customer,admin`.)

## تطبيقُ الصلاحيّات — **في الخادم** ✅

**والحارسُ وسيطٌ على مجموعة المسارات** (`server.go`):

| المجموعة | الشرط |
|---|---|
| `/api/v1/rep/*` | `RequireRoles("sales")` |
| `/api/v1/driver/*` | `RequireRoles("driver")` |
| `/api/v1/merchant/*` | `RequireRoles("merchant")` |
| `/api/v1/admin/*` | `RequireRoles("admin","ops","finance")` |

**وداخلَ الإدارة تضييقٌ ثانٍ لكلّ مسارٍ على حدة** — `r.With(RequireRoles("admin"))`
و`("admin","finance")` و`("admin","ops")`. ✅

**وهذا يعني**: الصلاحيّاتُ **خادميّةٌ لا واجهيّة**. والواجهةُ تُخفي
الأزرارَ لكنّها ليست الحارس. ✅

## ما يخصّ كلَّ دور

| الدور | يرى | ينشئ | يعدّل | يُنهي |
|---|---|---|---|---|
| **customer** | طلباتِه · محفظتَه · شكاواه | طلباً · شكوى · تقييماً | عناوينَه · حسابَه | يُلغي طلبَه في نافذته ✅ |
| **driver** | الطابورَ · طلباتِه · صندوقَه | تقريرَ طلب · طوارئ | ورديّتَه · موضعَه | يقبل/يرفض عرضاً · يُفشل توصيلاً |
| **merchant** | طلباتِ متجره · قائمتَه · تقاريرَه | أصنافاً · أقساماً | ساعاتِه · إعداداتِه | يقبل/يرفض طلباً |
| **sales** | متاجرَه · عمولتَه · هدفَه | فرصةَ متجرٍ (lead) | أصنافَ متاجره ⚠️ | — |
| **ops** | التشغيلَ كلَّه | — | يُسند سائقاً · ينقل طلباً | يُفشل · يقبل/يرفض |
| **finance** | المالَ كلَّه | قيوداً يدويّة | يبتّ طلباتِ الصرف | يسوّي صندوقاً |
| **admin** | **كلَّ شيء** | مستخدمين · متاجر · إعدادات | **كلَّ انتقالٍ في الخارطة** | **كلَّ شيء** |

**والأدمنُ مخوَّلٌ ضمنيّاً بكلّ انتقالٍ في الخارطة** — ✅ مُثبَت في
`internal/orders/statuses.go`:

```go
if r == "admin" {
    // الأدمن مخول بكل الانتقالات المعرفة في الخارطة
    for _, t := range transitionsFor(kind, from) { if t.To == to { return true } }
    return false
}
```

**ولا يستطيع الأدمنُ اختراعَ انتقالٍ ليس في الخارطة** ✅ — يُخوَّل بما
فيها لا بما خارجَها.

---

# ٥ · Core Business Flows

## ٥-١ · الطلبُ العاديّ (متجر) ✅

```
الزبون يطلب                POST /orders   (idempotent ✅)
   ↓  status=pending · wallet_paid/cash_due يُحسبان
المتجر (أو المكتب) يقبل     POST /merchant|admin/orders/{id}/transition → accepted
   ↓  autoPreparing قد ينقلها إلى preparing
طلبُ سائق                  → dispatching
   ↓  OfferNext: عرضٌ لسائقٍ واحدٍ بمهلة  أو  سباقٌ مفتوح
السائق يقبل                POST /driver/orders/{id}/accept → assigned
   ↓  فحوصٌ قبل القبول: on_shift · سقفُ النقد · عددُ الطلبات المفتوحة
السائق عند المتجر          → at_pickup   (driver)
استلم                      → picked_up
انطلق                      → on_the_way
وصل عند الزبون             → at_dropoff
سلّم                       → delivered  ← **هنا تقع التسوية المالية**
```

**والراصدُ يعمل بالتوازي كلَّ ٣٠ ثانية** ✅ (`cmd/api/main.go`):
`SweepExpiredOffers` · `sweepAutoAccept` · `Alerts` + تصعيد.

## ٥-٢ · الطلبُ المخصَّص (custom) ✅

`POST /orders/custom` — مسارٌ ثانٍ بخارطةِ حالاتٍ خاصّة
(`customTransitions` في `statuses.go`)، فيه `agree` (توثيقُ السعر والأجرة)
و`buying`. **والسائقُ يشتري ويُوثّق المبلغ.**

## ٥-٣ · فرصةُ المتجر (المندوب) ✅

```
المندوب يسجّل فرصة    → merchant_leads (sales_rep_user_id)
   ↓  إشعارٌ للمندوب وللمكتب
الإدارة تقبل          convertLead → يُنشأ merchant + حساب المالك
   ↓  إشعارُ «تمت الموافقة على عميلك»
   ↓  grantSalesTargetIfAny → مكافأةُ الهدف الشهريّ إن بلغ
أو ترفض               status='rejected' + سببٌ إلزاميّ → إشعارٌ للمندوب
```

## ٥-٤ · الشكوى ✅

`tickets` + `ticket_replies` — شاشةٌ مركزيّةٌ واحدةٌ في التطبيقات الثلاثة
(`ui/TicketsScreen.kt`).

## ٥-٥ · طلبُ الصرف ✅

`payout_requests` → `POST /admin/payouts/{id}/decide` (**idempotent** ✅)
→ قيدٌ في الدفتر + إشعارٌ لصاحبه.

---

# ٦ · Order State Machine ✅

**أربعَ عشرةَ حالةً** — `internal/orders/statuses.go`:

```
pending · accepted · preparing · dispatching · assigned ·
at_pickup · picked_up · on_the_way · at_dropoff · delivered ·
rejected · cancelled · failed · refunded
```

## الخارطةُ الفعليّة (`allowedTransitions`)

| من | إلى | الأدوار |
|---|---|---|
| `pending` | accepted · rejected | merchant, ops |
| | cancelled | customer, ops |
| `accepted` | preparing | merchant, ops |
| | dispatching | ops |
| | cancelled | customer, merchant, ops |
| `preparing` | dispatching | ops |
| | cancelled | merchant, ops |
| `dispatching` | assigned | driver, ops |
| | cancelled | ops |
| `assigned` | at_pickup · **dispatching** (فكُّ الإسناد) | driver, ops |
| | cancelled | ops |
| `at_pickup` | picked_up · failed · dispatching | driver, ops |
| | cancelled | ops |
| `picked_up` | on_the_way | driver, ops |
| | dispatching · failed · cancelled | ops |
| `on_the_way` | at_dropoff | driver, ops |
| | dispatching · failed · cancelled | ops |
| `at_dropoff` | delivered · failed | driver, ops |
| | dispatching · cancelled | ops |
| `delivered` | refunded | **admin فقط** |

**وأربعةُ انتقالاتٍ تلزمها علّةٌ مكتوبة** ✅ (`requiresReason`):
`rejected · cancelled · failed · refunded`.

## اتّساقُ الحالات بين الأطراف ✅ **مُثبَت — لا تناقض**

| الطرف | الحالاتُ المعروفة |
|---|---|
| المحرّك | ١٤ |
| معجمُ الويب (`orders.status`) | **١٤ — مطابق** |
| تطبيقُ الزبون | **١٤ — مطابق** |
| تطبيقُ المتجر | **١٤ — مطابق** |
| تطبيقُ السائق | **٩** — ولا يعرف `pending/accepted/preparing/cancelled/refunded` |

**ونقصُ السائق مقصودٌ لا عيب** ⚠️ مُستنتَج: تلك حالاتٌ تقع قبل أن يدخل
الطلبُ في يده. **ولم يُقرأ نصٌّ يقول ذلك صراحةً** — يحتاج تأكيداً.

## وما تعرضه اللوحةُ من انتقالات ✅

**`OPS_NEXT` في `OrdersScreen.tsx` ثمّ مرشّحُ `DRIVER_ONLY`** — ونتيجتُه
المقيسة: **`pending` وحدَها تُظهر زرَّي قبول/رفض**، و`at_pickup` وما بعدها
تُظهر «فشل التوصيل» (أُضيف ٢٠٢٦-٠٩-٠٢). **ولا زرَّ إلغاءٍ في اللوحة
مطلقاً** — بقرار المالك ٢٠٢٦-٠٨-٠٤.

**فهنا فرقٌ مقصود**: المحرّك يسمح للمكتب بالإلغاء من تسع حالات،
**واللوحةُ لا تعرضه.** ✅ مُثبَت · **وهو قرارٌ لا عطب.**

---

# ٧ · API Map — المسارات التي تغيّر حالاً أو مالاً ✅

| Method · Endpoint | الدور | يكتب | آثارٌ جانبيّة |
|---|---|---|---|
| `POST /orders` **(idem)** | customer | orders · order_items · wallet_transactions · promo_redemptions | إشعارُ المتجر والمكتب · بثُّ `order` |
| `POST /orders/custom` **(idem)** | customer | orders (kind=custom) | كسابقه |
| `POST /orders/{id}/cancel` | customer | orders · دفتر (ردّ) | إشعار · بثّ |
| `POST /driver/orders/{id}/accept` | driver | orders.driver_id · status | يُلغي العرضَ · بثّ |
| `POST /driver/orders/{id}/decline` | driver | — | `OfferNext` للتالي |
| `POST /driver/orders/{id}/transition` | driver | orders · order_events · **الدفتر عند `delivered`** | إشعاراتٌ للطرفين · مكافأةُ الهدف |
| `POST /merchant/orders/{id}/transition` | merchant | orders · order_events | إشعار · بثّ |
| `POST /admin/orders/{id}/transition` | ops/admin | كسابقه | **+ `audit_log`** |
| `POST /admin/orders/{id}/assign` | ops/admin | orders.driver_id | إشعارُ السائق |
| `POST /admin/orders/{id}/transfer` | admin | orders.merchant_id · order_items | تدقيقٌ · فرقٌ على المنصّة |
| `POST /admin/orders/{id}/recompute` | admin/finance | wallet_transactions (قيدٌ مقابل) | **يردّ before/after/delta** |
| `POST /admin/orders/{id}/compensate-driver` | admin/finance | الدفتر | إشعار |
| `POST /admin/orders/{id}/goods` | ops/admin | orders.goods_settled_to · الدفتر | |
| `POST /admin/users/{id}/wallet` **(idem)** | admin/finance | الدفتر + الرصيد | `touchUser` |
| `POST /admin/users/{id}/incentive` **(idem)** | admin/finance | incentives + الدفتر | إشعار |
| `POST /admin/payouts/{id}/decide` **(idem)** | admin/finance | payout_requests + الدفتر | إشعار |
| `POST /admin/drivers/{id}/settle` **(idem)** | admin/finance | driver_cash_entries · driver_cash_boxes.held | |
| `POST /driver/location` · `/location/batch` | driver | users.last_location · driver_track | — |
| `POST /driver/orders/{id}/proof` | driver | orders.pod_* | تدقيق |

**وستّةُ مساراتٍ محميّةٌ بمفتاح تكرار** ✅ — وهي المذكورةُ أعلاه بـ**(idem)**.

**⚠️ مُستنتَج**: `POST /driver/orders/{id}/accept` **غيرُ محميٍّ بمفتاح
تكرار** — يعتمد على الحالة والقفل بدلاً منه. لم يُقرأ نصٌّ يفسّر ذلك.

---

# ٨ · Database Truth ✅

**٧٧ جدولاً في الإنتاج** — منها **١٩ لواتساب** (`whatsmeow_*`) لا تخصّ
منطقَ العمل.

## الجداولُ التشغيليّة

| المجال | الجداول |
|---|---|
| الهويّة | `users · roles · user_roles · otp_codes · refresh_tokens · phone_claims · device_tokens · app_secrets` |
| الطلبات | `orders · order_items · order_events · order_messages · order_ratings · promo_redemptions` |
| المال | `wallets · wallet_transactions · driver_cash_boxes · driver_cash_entries · payout_requests · expenses · expense_categories` |
| المتاجر | `merchants · menu_items · menu_sections · modifier_groups · modifier_options · merchant_hours · merchant_ratings · merchant_warnings · merchant_leads · store_sections` |
| الجغرافيا | `governorates · districts · cities · delivery_zones` |
| التشغيل | `tickets · ticket_replies · disputes · warnings · driver_emergencies · driver_track` |
| التسويق | `offers · promo_codes · banners · categories · platform_sections · referrals` |
| النظام | `app_settings · audit_log · idempotency_keys · media · notifications · schema_migrations · app_opens_daily` |

## مصادرُ الحقيقة ✅

| المعلومة | المصدر |
|---|---|
| **حالةُ الطلب** | `orders.status` — وتاريخُها في `order_events` |
| **المال** | `wallet_transactions` **دفترٌ**، و`wallets.balance` **عمودٌ مصانٌ يجب أن يساويه** |
| **نقدُ السائق** | `driver_cash_entries` دفتراً، و`driver_cash_boxes.held` عموداً مصاناً |
| **دَينُ المتجر** | `merchants.debt` — **عمودٌ مصانٌ بلا دفترٍ خاصٍّ به** ⚠️ |
| **موضعُ السائق** | `users.last_location` + `last_location_at` (الحاليّ) · `driver_track` (الأثر) |
| **الإعدادات** | `app_settings` — كتالوجُها في `internal/settings/catalog.go` |

## القفلُ والتزامن ✅

**`SELECT … FOR UPDATE` في سبعة مواضعَ مقيسة** — أهمُّها:

- `transitions.go`: قفلُ صفّ الطلب قبل أيّ انتقال ✅
- `transitions.go`: قفلُ `merchants.debt` ✅
- `service.go`: قفلُ `promo_codes` — **«من وصل ثانياً ينتظر»** ✅
- `goods.go` · `custom.go`

**ولا `SKIP LOCKED` ولا أقفالٌ استشاريّة.** ✅

## التدقيق ✅

`audit_log` — يُكتب من `s.audit(...)`. **ويُسجَّل انتقالُ الموظّفين لا
انتقالُ الأطراف أنفسهم** ✅ (نصُّ التعليق: «المتجر يقبل مئة طلب في اليوم،
وتسجيلُها يُغرق السجلّ»).

---

# ٩ · Financial Flows ✅

## أنواعُ القيود — ثمانية ✅

```
topup · order_payment · refund · compensation ·
commission · merchant_earning · driver_earning · platform_profit · reward
```

## القاعدةُ المعماريّة ✅

**`wallet.ApplyTx(ctx, q, userID, amount, kind, ref, note, actorID)`** —
تُنادى بـ`Querier` (معاملة)، **فتُكتب في معاملةِ النداء نفسِها**:

1. تُنشئ المحفظةَ إن لم تكن ✅
2. **تُحدّث الرصيد وتقرأه في نداءٍ واحد** (`UPDATE … RETURNING`) ✅
3. **ورصيدٌ سالبٌ يُرفض بقيدِ تحقّقٍ في القاعدة** — `isCheckViolation → ErrInsufficient` ✅

**فالرصيدُ لا يُكتب بيد** — يُشتقّ من القيد.

## مواضعُ الكتابة في الدفتر ✅ (مقيسة)

| الملفّ | عدد النداءات |
|---|---|
| `orders/transitions.go` | **٨** |
| `referrals/referrals.go` | ٤ |
| `orders/treasury.go` | ٣ |
| `server/failure_aftermath.go` | ٢ |
| `orders/goods.go` · `custom.go` · `incentives/*.go` | ٢ لكلٍّ |

## متى يتحرّك المال ✅

| الحدث | ما يقع |
|---|---|
| **إنشاءُ الطلب** | `order_payment` (خصمٌ من المحفظة) · و`cash_due` للنقد |
| **التسليم** | `merchant_earning` · `driver_earning` · `commission` (المندوب) · `platform_profit` |
| **الإلغاء/الرفض** | `refund` |
| **الفشل** | `failure_aftermath.go` — قرارُ بضاعةٍ وتعويضٍ **بيد المكتب لا آليّاً** ✅ |
| **بلوغُ الهدف** | `reward` — عبر `incentives` |
| **الإحالة** | `referrals` |

## منعُ التكرار ✅

| الطبقة | الآلية |
|---|---|
| **مفتاحُ التكرار** | `idempotency_keys` — ستّةُ مساراتٍ مالية |
| **مكافأةُ الهدف** | فهرسٌ فريدٌ `(user_id, period)` — «التصادمُ ليس خطأً، هو الضمانةُ تعمل» ✅ |
| **التسوية** | `CreditTreasuryTx` **تكتب قيداً مقابلاً بالفرق ولا تمسح** ✅ |
| **الرصيد** | قيدُ تحقّقٍ يمنع السالب |

**⚠️ مُستنتَج**: `merchants.debt` عمودٌ مصانٌ **بلا دفترٍ خاصٍّ به** —
فلا يمكن قراءةُ تاريخه. وقد قِيس أنّه يُصفَّر عند المسح الشامل.

---

# ١٠ · Notifications ✅

**ستُّ فئات** — `internal/notifications/notifications.go`:

```
KindOrder · KindTicket · KindWallet · KindRating · KindLead · KindAccount
```

## التوجيه ✅

**حقلُ `Apps []string`** يقرّر أيُّ تطبيقٍ يرنّ — **وفارغٌ يعني كلَّها**
(«افتراضُ الصمت يجعل الإشعاراتِ تختفي دفعةً واحدةً بلا أثرٍ في سجلّ»).

**والقيمُ**: `AppRep · AppDriver · AppMerchant · AppCustomer` ⚠️ (قِيس
`AppRep` و`AppDriver`؛ الباقي مُستنتَج).

## الناقل ✅

**FCM بنداءِ REST مباشر** — `internal/push/fcm.go`، **بلا حزمة Firebase**
(«تجرّ عشراتِ الحزم»).

**والخدمةُ لا تُفشل العمليةَ الأصلية أبداً** ✅ — عهدٌ مكتوبٌ في الحزمة:
«وذعرٌ يُسقط النداءَ كلَّه أشدُّ من خطأٍ يُرجَع».

## الوجهات ✅

**حارسٌ في الاختبارات** (`notifications/routing_test.go`) **يتحقّق أنّ كلَّ
`Href` له صفحةٌ في الويب فعلاً** — وقد أسقط النداءَ ٢٠٢٦-٠٩-٠٤ حين حُذفت
`/offers`.

---

# ١١ · Realtime ✅

**آليّةٌ داخليّةٌ بسيطة** — `internal/realtime/hub.go`:

```go
Publish(topic string, event any)
Subscribe(topics []string) (<-chan []byte, func())
```

**والمسار `GET /ws`** — بنبضةٍ كلَّ ٣٠ ثانية (`internal/server/ws.go`).

## المواضيع المقيسة ✅

`ops` (غرفةُ العمليات) — يُبثُّ إليها: `account · catalog · dispute ·
driver · lead · menu · merchant · offer · order · rating · settings ·
ticket · wallet`

**و`touchUser(userID, …)` يبثّ إلى صاحب الشأن نفسِه** ✅ — أُضيف لأنّ
«الزبونَ يُعوَّض ثمّ ينظر إلى رصيده فيجده كما كان».

## شكلُ الحدث ✅

**`{"type": "<entity>"}` — إشارةُ تحديثٍ لا حمولةَ بيانات.** فالواجهةُ
**تُعيد الجلب** عند سماعها.

**⚠️ مُستنتَج**: لا ترتيبَ مضمونٌ للأحداث ولا إعادةَ تشغيلٍ بعد انقطاع —
لأنّها إشاراتٌ لا سجلّ. **وسلوكُ إعادة الاتّصال في العميل لم يُقرأ** ❓.

**وحارسُ `check-live-refresh.mjs`** يتحقّق أنّ كلَّ شاشةٍ تجلب من اللوحة
تسمع النبضة ✅ (قِيس: شاشتان).

---

# ١٢ · Authentication & Sessions ✅

## العمليّاتُ المقيسة (`internal/identity`)

```
RequestOTP · VerifyOTP · RequestSignup · ConfirmSignup · VerifySignupCode
LoginPassword · Refresh · Logout · AdminLogoutAll
RequestPasswordReset · ConfirmPasswordReset · VerifyResetCode · SetPassword
RequestPhoneChange · ConfirmPhoneChange
RequestWhatsAppVerify · ConfirmWhatsAppVerify · MarkWhatsAppFromSignup
RequestAccountDeletion · ConfirmAccountDeletion
PinChallenge · VerifyPin · SetPinFirstTime · ChangePin · ResetPinRequest · ResetPinConfirm · HasPin
AdminCreateUser · AdminGrantRole · AdminRevokeRole · AdminUpdateUser · BootstrapAdmin
```

## الأعمار ✅

| | القيمة | المصدر |
|---|---|---|
| توكنُ الوصول | من `NewTokenIssuer(secret, accessTTL)` | `internal/auth/token.go` |
| **الجلسة (refresh)** | **٣٠ يوماً** افتراضاً · تُضبط بـ`security.session_days` | `identity/service.go` |
| رمزُ التحقّق | `security.otp_ttl_min` | إعدادات |
| **رمزُ الأدمن (PIN)** | **٤ أرقام** بقرار المالك | `identity/admin_pin.go` |

## الطبقةُ الثانيةُ للأدمن ✅

`POST /auth/login` يردّ `pin_required` + `challenge` → `POST /auth/pin`.
**ومحاولاتٌ كثيرةٌ تُقفل بمهلةٍ في Redis** (`pinTries` · `pinLockFor`).

## الإبطال ✅

**قائمةُ إبطالٍ في Redis** — `sessionRevokedKey(sid)` بعمر `AccessTTL + دقيقة`.
**فتوكنُ وصولٍ مسروقٌ يموت بعد إبطال الجلسة** ✅.

## ❓ مجهول

**ماذا يقع إن انتهت الجلسةُ أثناء رحلةٍ نشطة؟** لم يُقرأ مسارٌ يعالج ذلك
صراحةً. **⚠️ مُستنتَج**: العميلُ يُجدّد بـ`Refresh` تلقائيّاً
(`packages/auth/src/client.ts` يفعل ذلك في الويب)، **وسلوكُ أندرويد لم
يُقَس.**

---

# ١٣ · Driver Location & Navigation ✅

## الالتقاط والإرسال

| | القيمة | المرجع |
|---|---|---|
| المزوّد | Google Play Services · `PRIORITY_HIGH_ACCURACY` | `LocationService.kt` |
| الفترة | **٢٠ ثانية** افتراضاً · تُضبط من `drivers.location_ping_sec` | ✅ |
| أقلُّ حركة | `MIN_MOVE_M` — «ودونها ضجيج قمرٍ صناعيّ» | ✅ |
| التجميع | `setMaxUpdateDelayMillis(ms*2)` — «يوقظ الجهازَ مرّةً بدل مرّات» | ✅ |
| **الشيخوخة** | **١٥ دقيقة** → النقطةُ كأنّها غيرُ موجودة | `StaleLocationAfter` ✅ |
| الخدمة | أماميّةٌ بنوع `location` · `START_STICKY` | البيان ✅ |

**ومسارٌ ثانٍ للدفعات** — `POST /driver/location/batch` (حتّى ٢٠٠ نقطة)
**لما يُجمَع أثناء انقطاع الشبكة** ✅.

## جودةُ الإشارة ✅

`GpsQuality.grade` — ثلاثُ درجات: `ACCEPTED · DEGRADED · REJECTED`،
وأسبابُ الرفض: `ACCURACY · STALE · TELEPORT`.

## الموقعُ المزيَّف ✅ (أُضيف ٢٠٢٦-٠٩-٠٣)

`isFromMockProvider` → يُرفع مع النقطة → `driver_track.mocked` ·
`orders.pod_mocked`. **وإثباتُ تسليمٍ من موضعٍ مزيَّفٍ تُرفض نقطتُه.**

## الملاحة ✅

**وحدةٌ مستقلّة `driver-navigation` (٤١ ملفّاً · ٢٨ اختباراً)** — وفيها
`NavPipeline · RouteProgress · OffRouteDetector · RerouteEngine ·
VoicePlanner · PathSmoother · ParallelResolver · ReplayDrive`.

**والصوتُ مقاطعُ مسجَّلةٌ مسبقاً — ٥٦٨ ملفّاً في `res/raw`** ✅،
**ولا نداءَ شبكةٍ أثناء القيادة.**

**والخرائطُ PMTiles محلّيّةٌ** — سوريا كاملةً، تُنزَّل في الخلفيّة.

## ❓ مجهول

- **سلوكُ الخدمة عند فقد الإنترنت طويلاً** — الطابورُ موجودٌ، **وحدُّه لم يُقرأ**.
- **كيف يعرف النظامُ أنّ السائقَ متّصل؟** — قِيس `users.on_shift` (يدويّ)
  و`last_location_at` (شيخوخة). **ولا مفهومَ «online» ثالث.** ✅

---

# ١٤ · Representative Workflow ✅

**المندوبُ خارجَ دورة الطلب تماماً** ✅ — لا يظهر في `allowedTransitions`
ولا في أيّ انتقالٍ.

| | |
|---|---|
| **ما يملكه** | متاجرُ جلبها — `merchants.sales_rep_user_id` |
| **كيف يجلب** | `merchant_leads` بكوده · أو تُنشئه الإدارةُ برمزه |
| **عمولتُه** | `sales.commission_percent` (افتراضُه **١٠٪**) — `internal/pricing` |
| **متى تُقيَّد** | **عند تسليم طلبٍ من أحد متاجره** — `notifyCommission` |
| **هدفُه الشهريّ** | **عددُ المتاجر المسجَّلة** (لا الطلبات) — بقرار المالك ٢٠٢٦-٠٨-٣١ |
| **شاشاتُه** | لوحتي · إضافة عميل · عملائي · حسابي · رابط الدعوة · أهدافي · المحفظة |
| **قيدٌ جغرافيّ** | **لا** — «يزور ولا يوصّل» |

---

# ١٥ · Admin Operational Powers ✅

| القدرة | متاح؟ | تدقيق؟ |
|---|---|---|
| تغييرُ حالة الطلب | ✅ ضمن الخارطة | ✅ `ops.order_transition` |
| إسنادُ سائق | ✅ | ✅ |
| **إلغاءُ طلب** | **يسمح به المحرّك · ولا زرَّ في اللوحة** | — |
| نقلُ الطلب إلى متجرٍ آخر | ✅ | ✅ |
| تعديلُ رصيد | ✅ (admin/finance · **idem**) | ✅ |
| **إعادةُ حساب التسوية** | ✅ **بقيدٍ مقابلٍ لا بمسح** | ✅ `finance.settlement_recomputed` |
| تعويضُ سائق | ✅ | ✅ |
| تعليقُ متجرٍ · مسحُ مخالفاته | ✅ | ✅ |
| إنذارُ مستخدم | ✅ | ✅ |
| تعديلُ العمولة والأجرة | ✅ عبر `app_settings` | ✅ |
| منحُ/سحبُ دور | ✅ admin فقط | ✅ |
| إخراجُ كلّ الجلسات | ✅ | ✅ |
| **اختراعُ حالةٍ خارج الخارطة** | **❌ لا** — مُثبَت في `canTransition` | — |

---

# ١٦ · Background Jobs ✅

**عاملٌ واحدٌ في المحرّك كلِّه** — `cmd/api/main.go`:

```go
go ordersSvc.RunWatchdog(ctx, 30*time.Second)
```

**ويفعل ثلاثة أشياء**: `SweepExpiredOffers` · `sweepAutoAccept` ·
`Alerts` + تصعيد.

**وفي أندرويد**: `MapPackageWorker` (WorkManager) لتنزيل حزم الخرائط.

**ولا طوابيرَ ولا Redis queues ولا cron.** ✅

---

# ١٧ · Existing Tests ✅

| المنظومة | الحجم | ما تختبره | ما لا تختبره |
|---|---|---|---|
| **Go tests** | **١٩٨ ملفّاً** (منها ٢٠ في `internal/qa`) | المنطقَ والحالاتِ والمالَ والعقود | الشاشات · الجهاز |
| **حرّاسُ الويب** | **٢٥** | المركزيّة · النصوص · الأخطاء · الخريطة · التصميم | السلوك |
| **اختباراتُ أندرويد** | **٥٣ ملفّاً** — ٢٨ منها في الملاحة | حساباتِ الملاحة | **الخيطَ الرئيسيّ · الجهاز · الشاشة** |
| `scripts/e2e/cycle.py` | ٣٠٤ + ٢٧٣ سطراً | **٤٦ خطوةً تمشي المنصّةَ كاملةً وتقرأ من القاعدة** | الشاشات |
| `mobile/{merchtest,drivertest,reptest}` | `engine.sh` + `glass.sh` | نداءاتٍ وشاشاتٍ منفصلة | **الوصلاتِ بينها** |
| `mobile/testkit` | `api.py · device.py · guards.py` | أدواتٌ مساعدة | — |
| `docs/FIELD-TESTS.md` | ١٦٦ سطراً | بنودٌ ميدانيّةٌ تُمشى باليد | — |
| **اختباراتُ الويب** | **صفر** ✅ | — | **كلُّ شيء** |

## الحكمُ المقيس

**ولا واحدٍ من عيوب هذه الجلسة أمسكه `merchtest` أو `drivertest`** —
أمسكها المالكُ بعينه أو قياسُ الشيفرة. **وأخطرُ ما فيها أنّ
`drivertest` يعلن «٣٥ ناجح · ٠ ساقط» و٣٩ بنداً من ٦٣ محجوبٌ لم يعمل قطّ.**

**و`scripts/e2e/cycle.py` هو الأقربُ إلى منظومةٍ حقيقيّة** — يمشي الدورةَ
كاملةً **ويقرأ الأثرَ من القاعدة لا من ردّ الخادم** ✅ (مبدأٌ مكتوبٌ في
`README.md`: «ردٌّ بـ٢٠٠ يقول إنّ النداء نجح، لا إنّ المال انتقل»).

---

# ١٨ · Failure Handling ✅

| الحال | ما يقع |
|---|---|
| فشلُ الإشعار | **يُبتلع ولا يُسقط العمليّة** ✅ عهدٌ مكتوب |
| فشلُ الخدمة الخارجيّة (Nominatim) | يردّ فارغاً · وحدُّ معدّلٍ يردّ ٤٢٩ |
| انقطاعُ الشبكة عن السائق | **طابورٌ محلّيّ** ثمّ `location/batch` ✅ |
| تكرارُ الطلب | `idempotency_keys` في ستّة مسارات |
| تصادمُ المكافأة | فهرسٌ فريد — «التصادمُ ليس خطأً» ✅ |
| رصيدٌ غيرُ كافٍ | قيدُ تحقّقٍ في القاعدة → `ErrInsufficient` ✅ |
| طلبٌ عالق | **الراصد** كلَّ ٣٠ ثانية → تنبيهُ تصعيد |
| سقوطُ التطبيق | Crashlytics في الإصدار · **معطَّلٌ في التجريبيّ** ✅ |
| موتُ خدمة الموقع | **بطاقةُ إعفاءِ البطّاريّة** (٢٠٢٦-٠٩-٠٣) + **«موقعه متوقف»** في اللوحة |

## ❓ مجهول

- **إعادةُ المحاولة في عميل أندرويد** — لم يُقرأ مسارٌ عامّ لإعادة النداء.
- **سلوكُ إعادة اتّصال WebSocket** في العملاء.

---

# ١٩ · Cross-App Dependencies ✅

```
shared/        عقدُ الـAPI — تقرؤه التطبيقاتُ الأربعة
ui/            الشاشاتُ المشتركة: الحساب · المحفظة · الشكاوى · الإشعارات ·
               التقييم · الأهداف · الدرج · البطّاريّة · «كم مضى»
map/           لوحةُ الخريطة · التقاطُ النقطة · حزمُ PMTiles
driver-navigation/  الملاحة — **لا يستعملها إلّا السائق**
design/ brand/ التوكنز والهويّة
```

**وحارسٌ يمنع أن يُظلَّ نصٌّ في `ui` بنصٍّ في تطبيق** ✅
(`check-string-shadow.mjs` — أُضيف بعد حادثةٍ وقعت).

**ونمطُ الخريطة ملفٌّ واحدٌ للويب وأندرويد** ✅ (`check-map-style-single`).

---

# ٢٠ · Contradictions & Gaps

## ✅ مُثبَتة

| # | التناقض | الموضع |
|---|---|---|
| **C1** | **المحرّك يسمح للمكتب بإلغاء طلبٍ من تسع حالات — واللوحةُ لا تعرض زرَّ إلغاءٍ قطّ** | `statuses.go` مقابل `OrdersScreen.tsx` · **قرارُ مالك ٢٠٢٦-٠٨-٠٤** |
| **C2** | **`merchants.debt` عمودٌ مصانٌ بلا دفتر** — بينما كلُّ مالٍ آخرَ له قيد | `transitions.go` |
| **C3** | **٢٨ مقطعاً صوتيّاً (`keep_*`) مسجَّلٌ ومربوطٌ ولن يُنطق** — لأنّ بيانات سوريا بلا `turn:lanes` | قِيس من OSRM الحيّ |
| **C4** | **`exit_roundabout` صار شبهَ ميّت** بعد طيّ مناورة الخروج (٢٠٢٦-٠٩-٠٣) | `steps.go` |
| **C5** | **اختباراتُ الويب صفر** — بينما فيه لوحةُ الإدارة كلُّها | `web/` |
| **C6** | **`app-merchant` و`app-rep` بلا اختبارٍ واحد** | `mobile/` |
| **C7** | **`drivertest` يعلن نجاحاً و٣٩ بنداً من ٦٣ محجوب** | `mobile/drivertest/TASKS.md` |
| **C8** | **`cmd/hashgen/main.go` خارجَ التنسيق** منذ زمن | `gofmt -l` |

## ⚠️ مُستنتَجة — تحتاج إثباتاً

| # | الشبهة |
|---|---|
| **I1** | **`accept` غيرُ محميٍّ بمفتاح تكرار** — يعتمد على القفل والحالة. **لم يُثبَت أنّه آمنٌ تحت ضغط.** |
| **I2** | **بثُّ Realtime إشارةٌ بلا ترتيبٍ ولا إعادةِ تشغيل** — فحدثٌ ضاع أثناء انقطاعٍ لا يُعوَّض إلّا بجلبٍ يدويّ |
| **I3** | **تطبيقُ السائق لا يعرف ٥ حالات** — يُرجَّح أنّه مقصود، ولا نصَّ يقوله |
| **I4** | **لا حدَّ مقروءٌ لطابور المواقع** عند انقطاعٍ طويل |

## ❓ مجهولة

- سلوكُ الجلسة إن انتهت أثناء رحلةٍ نشطة (أندرويد).
- إعادةُ اتّصال WebSocket في العملاء.
- سلوكُ `whatsmeow` عند فقد الجلسة — **١٩ جدولاً في الإنتاج ولم يُقرأ مسارُه.**
- **هل `AppCustomer` و`AppMerchant` مستعملان فعلاً** في توجيه الإشعارات؟

---

# ٢١ · High-Risk Areas

**مرتّبةٌ بالأثر لو انكسرت:**

| # | المنطقة | لماذا |
|---|---|---|
| **R1** | **تسويةُ التسليم** (`transitions.go` — ٨ نداءات دفتر) | أربعةُ قيودٍ في معاملةٍ واحدة · وخطأٌ فيها يفسد مالَ ثلاثة أطراف |
| **R2** | **إسنادُ السائق** (`rotation.go` + `accept`) | تزامنٌ · وطلبٌ لسائقين أو لا أحد |
| **R3** | **موضعُ السائق** | يُقتل في الخلفيّة · ويُزيَّف · وعليه يقوم الإسنادُ والإثبات |
| **R4** | **الراصدُ الوحيد** كلَّ ٣٠ ثانية | **نقطةُ فشلٍ واحدة**: لو توقّف، لا عروضَ تنتقل ولا تصعيد |
| **R5** | **صحّةُ الدفتر** (`balance` = مجموعُ القيود) | يُقاس بـ`moneycheck` — **١٣ فحصاً** |
| **R6** | **الملاحةُ الصوتيّة** | ٥٦٨ مقطعاً · وعبارةٌ خاطئةٌ تُسمع في كلّ رحلة |
| **R7** | **الهجرات** ١٢٦ | لا تراجعَ مكتوب ⚠️ |

---

# ٢٢ · ما يحتاج قراراً قبل تصميم منظومة التحقّق

**١) ما تعريفُ «الجواب الشافي»؟**
هل هو: «دورةٌ كاملةٌ مشت والمالُ صحيحٌ والانحرافُ صفر»؟ أم أوسع؟

**٢) ما الذي يُقاس آليّاً وما الذي يُمشى بالعين؟**
**والقاعدةُ المقترحة**: ما يُقرأ من القاعدة يُؤتمت، **وما يُرى بالعين
يُكتب قائمةً ولا يُدَّعى أنّه فُحص.**

**٣) مصيرُ الموجود:**
`merchtest` · `drivertest` · `reptest` · `testkit` — **تُحذف أم تُدمج؟**
**و`scripts/e2e/cycle.py` هو الوحيدُ الذي يمشي دورةً حقيقيّةً ويقرأ من
القاعدة — أيُبنى عليه؟**

**٤) البيئة:**
هل تمشي المنظومةُ على **الإنتاج** (كما يفعل `drivertest`) أم على بيئةٍ
منفصلة؟ **والأولى تُلوّث، والثانيةُ لا تُثبت ما يقع فعلاً.**

**٥) الأجهزةُ الأربعة:**
هل تُقاد آليّاً (adb) أم يمشيها المالكُ بيده والمنظومةُ تقرأ الأثر؟

**٦) كم مرّة؟** كلَّ نشرة؟ يوميّاً؟ عند الطلب؟

---

# ٢٣ · حدودٌ مقترحةٌ لمنظومة التحقّق المقبلة

**لا تُبنى الآن — هذه حدودٌ للنقاش:**

1. **طلبٌ حقيقيٌّ واحدٌ يمشي إلى آخره في كلّ تشغيل** — لا فحوصٌ منفصلة.
2. **كلُّ تأكيدٍ يُقرأ من القاعدة أو من الدفتر** — لا من ردّ الخادم.
3. **وفحصُ المال يُختم بالانحراف**: `balance` مقابل مجموع القيود = **صفر**.
4. **وما لا يُقاس يُكتب في قائمةٍ ميدانيّة** — ولا يُحسب نجاحاً.
5. **ولا رقمَ يُعلَن ما لم يُذكر مقامُه**: «٣٥ من ٦٣ · و٢٨ محجوبٌ لأنّ…»

---

**انتهى الكشف. ولم تُبنَ منظومة، ولم تُصلَح مشكلة، ولم تُضف تبعيّة.**
