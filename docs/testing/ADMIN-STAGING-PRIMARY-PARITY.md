# تكافؤُ لوحة الإدارة — التجهيز ↔ الإنتاج

> **يُكتب بطلب المالك صراحةً** (٢٠٢٦-٠٩-١٩): «Staging Admin must be
> functionally equivalent to the primary/Production Admin at 100%, except
> for intentional environment/data differences».
>
> **الإنتاجُ قراءةٌ فقط · وتعديلاتُه صفر.**

---

## الحكم — بعد قرارات المالك الثلاثة (٢٠٢٦-٠٩-١٩)

	سطوحُ الجرد               = 372  (لم يُنقَص منها شيء)
	PARITY_OK                 = 336  (كانت 304)
	ENVIRONMENT_ONLY          = 13   (كانت 10)
	DATA_ONLY                 = 18   (كما هي)
	GENUINE_MISMATCHES        = 0    (في يد التجهيز — لا فرقَ مفتوح)
	BLOCKED_BY_PRODUCTION_VERSION = 5 (G4–G7 — يلزمها نشرُ إنتاج)

**ما بقي فرقاً سببُه واحد**: الإنتاجُ على `79d88245` والتجهيزُ على
`68a45c97`. **ونشرُ الإنتاج غيرُ مأذونٍ بعد** — وفحصُه المسبق في §٩.

| # | الفرق | الحال |
|---|---|---|
| G1 | خرائطُ لوحة التجهيز معطّلة | ✅ **أُصلح** — للتجهيز خرائطُه على مضيفه (§٥) |
| G2 | ويبُ التجهيز منفصلٌ عن محرّكه | ✅ أُصلح (التدقيقُ الأوّل) |
| G3 | تركيبُ الخادم شاخ عن المستودع | ✅ أُصلح (التدقيقُ الأوّل) |
| G4 | حذفُ قسمٍ مشغول: الإنتاج ٥٠٠ | ⛔ يلزمه نشرُ إنتاج |
| G5 | إيقافُ المنصّة بلا سطرِ تدقيقٍ في الإنتاج | ⛔ يلزمه نشرُ إنتاج |
| G6 | اسمُ ذلك الفعل غائبٌ عن سجلّ الإنتاج | ⛔ يلزمه نشرُ إنتاج |
| G7 | الهجرتان ٠١٥٦ و٠١٥٧ | ⛔ يلزمه نشرُ إنتاج |

**وسياسةُ التجهيز صارت سياسةَ الإنتاج**: ٢٩ إعداداً سُوّيت بباب اللوحة
(§٦)، **وبقي فرقان مقصودان** (`site.show_login` · `site.show_shop`).

---

# ١ · المنهج — خمسةُ أبوابٍ لا سادسَ لها

**والصورةُ واحدةٌ للبيئتين** (`/config.js` وقتَ التشغيل). **فلا تفترق
اللوحتان إلّا من خمسة أبواب** — **وكلُّها قِيست:**

| الباب | كيف قِيس |
|---|---|
| **١ · نسخةُ الشيفرة المنشورة** | صورُ الحاويات على الخادم + `git merge-base` |
| **٢ · شيفرةٌ تتفرّع على البيئة** | بحثٌ في الويب والخادم عن كلّ تفرّع |
| **٣ · تهيئةُ التشغيل** | `/config.js` الحيّ + ملفّا التركيب |
| **٤ · البيانات** | الإعداداتُ والأدوارُ والأقسام — قراءةٌ من القاعدتين |
| **٥ · المخطّط** | الهجراتُ المطبَّقة |

**ثمّ برهانٌ تشغيليّ**: كلُّ صفحةٍ وكلُّ بابِ قراءةٍ على البيئتين،
**وكلُّ خريطةٍ في اللوحة بمتصفّحٍ حقيقيّ.**

## الجردُ — ٣٧٢ سطحاً

| السطح | العدد | OK | ENV | DATA | BLOCKED |
|---|---|---|---|---|---|
| صفحاتُ اللوحة | 29 | 26 | 1 | 0 | 2 |
| أبوابُ API (86 قراءة · 89 كتابة) | 175 | 173 | 0 | 0 | 2 |
| مفاتيحُ الإعدادات | 143 | 119 | 8 | 16 | 0 |
| الأدوار | 18 | 16 | 0 | 2 | 0 |
| مفاتيحُ التهيئة | 5 | 1 | 4 | 0 | 0 |
| فهرسُ الأقسام | 1 | 1 | 0 | 0 | 0 |
| حالُ المخطّط | 1 | 0 | 0 | 0 | 1 |
| **المجموع** | **372** | **336** | **13** | **18** | **5** |

**وقبل القرارات** (التدقيقُ الأوّل): 304 · 10 · 18 · **40 فيها فرق.**
**وما تحرّك**: ٣ صفحاتِ خرائط ⇐ OK · ٢٩ إعداداً ⇐ OK · إعدادان ⇐ ENV
(قرارٌ مقصود) · نمطُ الخريطة ⇐ ENV (مضيفٌ لكلّ بيئة).

---

# ٢ · النسخُ المنشورة

| المكوّن | الإنتاج | التجهيز |
|---|---|---|
| ويبُ اللوحة | `release-79d88245` | `release-68a45c97` |
| المحرّك | `release-79d88245` · هجرة `0155` | `release-68a45c97` · هجرة `0157` |

	79d88245 ──(13)──▶ 2df3e5ed ──(2)──▶ 68a45c97 ──(10)──▶ HEAD
	الإنتاج                                  التجهيز      وثائقُ وأندرويد وتهيئةُ تجهيز

**تاريخٌ خطّيّ** — فالتجهيزُ لا يمكن أن ينقصه ما في الإنتاج إلّا بحذفٍ
صريح، **ولا حذف.**

---

# ٣ · النتيجةُ بابًا بابًا

## ٣·١ · الشيفرة

**`web/` بين الإنتاج والتجهيز: سطران في `ar.json`** — نصُّ
`section_has_items` وتسميةُ `admin.platform_closure`.

**`backend/` بين المحرّكين (بلا اختبارات)** — التفصيلُ في §٩·٢.
**و`server/server.go` متطابقٌ حرفاً** (قِيس ثانيةً) — **فالموجّهُ واحد.**

## ٣·٢ · التفرّعُ على البيئة

	الويب   = رايةُ «STAGING» (DashboardChrome.tsx:374) + اسمُ البيئة في صفحة ops
	الخادم  = cfg.Env == "development" فقط — يتساوى فيه التجهيزُ والإنتاج

**ولا ميزةَ في اللوحة تُفتح أو تُغلق بالبيئة.** ⇒ `ENVIRONMENT_ONLY` مسموح.

## ٣·٣ · تهيئةُ التشغيل — `/config.js` (قِيس حيّاً)

| المفتاح | الإنتاج | التجهيز | الصنف |
|---|---|---|---|
| `apiUrl` | api.rahalgo.com | staging-api.rahalgo.com | ENVIRONMENT_ONLY |
| `siteUrl` | rahalgo.com | staging-api.rahalgo.com | ENVIRONMENT_ONLY |
| `environment` | production | staging | ENVIRONMENT_ONLY |
| `mapStyleUrl` | maps.rahalgo.com/map-resources/1/style.online.json | **staging-api.rahalgo.com/maps/map-resources/1/style.online.json** | ENVIRONMENT_ONLY — **ويعمل (§٥)** |
| `API_INTERNAL_URL` | http://api:8080 | = | PARITY_OK |

## ٣·٤ · الإعدادات

	مفاتيحُ الفهرس          = 143 في البيئتين — الشيفرةُ نفسُها
	القيمُ الفعليّةُ متطابقة = 119  (كانت 90)
	القيمُ تختلف            = 24   (كانت 53) — §٦

**والقيمةُ الفعليّة = المخزَّنة وإلّا الافتراض** — لا عددُ الصفوف.

## ٣·٥ · الصلاحيّات (قِيست ثانيةً)

	الأدوارُ القياسيّة       = 16 — صلاحيّاتُها متطابقةٌ كلُّها
	أدوارٌ في التجهيز وحدَه  = s1_ops_viewer · s1_orders_only
	                           تجهيزاتُ اختبارٍ يدويّة (٢٠٢٦-٠٩-١٣) · DATA_ONLY

## ٣·٦ · أقسامُ المنصّة (قِيست ثانيةً)

	الإنتاج  = 38 فعّالاً · b4a77017bb32e32812935ec26402f24d
	التجهيز  = 38 فعّالاً · b4a77017bb32e32812935ec26402f24d

**البصمةُ واحدة** (الاسمُ والأيقونةُ والترتيب). **والعقدُ يفترق**: حذفُ
قسمٍ مشغول ٤٠٩ في التجهيز و٥٠٠ في الإنتاج (G4).

## ٣·٧ · البرهانُ التشغيليّ (أُعيد ٢٠٢٦-٠٩-١٩ بعد التسوية)

	أبوابُ الإدارة للقراءة (86) بلا توثيق:  الإنتاج 401 · التجهيز 401 ⇐ 86/86
	صفحاتُ اللوحة (29):                     الإنتاج 200 · التجهيز 200 ⇐ 29/29
	أبوابُ الكتابة (89):                    لم تُرسَل إلى الإنتاج · والموجّهُ متطابقٌ حرفاً

---

# ٤ · مصفوفةُ الصفحات — ٢٩

| الصفحة | القدرة | الإنتاج | التجهيز | الصنف |
|---|---|---|---|---|
| `/dashboard` | analytics.read | 200 | 200 | PARITY_OK |
| `/dashboard/orders` | orders.read | 200 | 200 | PARITY_OK |
| `/dashboard/history` | orders.read | 200 | 200 | PARITY_OK |
| `/dashboard/users` | users.read · **منتقي موقع المتجر** (`MerchantModal`) | 200 | 200 | PARITY_OK — **الخريطةُ تعمل** |
| `/dashboard/users/[id]` | (تفصيل) · تعديلُ المتجر بالمكوّن نفسِه | 200 | 200 | PARITY_OK |
| `/dashboard/roles` | roles.manage | 200 | 200 | PARITY_OK |
| `/dashboard/sections` | content.manage | 200 | 200 | **BLOCKED_BY_PRODUCTION_VERSION** (G4) |
| `/dashboard/sections/[id]` | (تفصيل) | 200 | 200 | PARITY_OK |
| `/dashboard/merchants/[id]` | (تفصيل) | 200 | 200 | PARITY_OK |
| `/dashboard/merchants/[id]/menu` | (تفصيل) | 200 | 200 | PARITY_OK |
| `/dashboard/tickets` | support.manage | 200 | 200 | PARITY_OK |
| `/dashboard/emergencies` | support.manage | 200 | 200 | PARITY_OK |
| `/dashboard/leads` | merchants.verify | 200 | 200 | PARITY_OK |
| `/dashboard/opsmap` | orders.read · **خريطة** | 200 | 200 | PARITY_OK — **الخريطةُ تعمل** |
| `/dashboard/wallet` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/expenses` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/profits` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/cash` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/losses` | finance.read · support.manage | 200 | 200 | PARITY_OK |
| `/dashboard/payouts` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/audit` | audit.read | 200 | 200 | **BLOCKED_BY_PRODUCTION_VERSION** (G6) |
| `/dashboard/reports` | analytics.read | 200 | 200 | PARITY_OK |
| `/dashboard/incentives` | finance.read | 200 | 200 | PARITY_OK |
| `/dashboard/promos` | content.manage | 200 | 200 | PARITY_OK |
| `/dashboard/ops` | observability.read | 200 | 200 | ENVIRONMENT_ONLY (اسمُ البيئة) |
| `/dashboard/settings` | settings.*.manage · **ثلاثُ خرائط** | 200 | 200 | PARITY_OK — **الخرائطُ تعمل** |
| `/dashboard/account` | (حسابُ صاحبها) | 200 | 200 | PARITY_OK |
| `/dashboard/notifications` | (صندوقُ صاحبها) | 200 | 200 | PARITY_OK |
| `/adminrahalgo` | عامّةٌ — بابُ الدخول | 200 | 200 | PARITY_OK |

> **تصحيح:** **كتبتُ في التدقيق الأوّل أنّ منتقي موقع المتجر في
> `/dashboard/merchants/[id]`** — **وليس فيها خريطةٌ أصلاً.** **المنتقي في
> `MerchantModal`**، يُفتح من `/dashboard/users` (متجرٌ جديد) ومن
> `/dashboard/users/[id]` (تعديلُ متجر). **والعددُ لم يتغيّر** — صفحةٌ
> واحدةٌ عُدّت معطّلةً في الحالين — **لكنّ النسبةَ كانت خطأ.**

---

# ٥ · G1 — خرائطُ لوحة التجهيز (أُصلح)

## قرارُ المالك

**خرائطُ خاصّةٌ بالتجهيز** — **ولا يُخرَج `maps.rahalgo.com` من مضيفات
الإنتاج، ولا يُضعَف `envguard`، ولا تُقبَل لوحةٌ بلا خرائط، ولا يُشار إلى
مضيف إنتاج.**

## المانعُ الخارجيّ — ولماذا مسارٌ لا مضيف

**لا سجلَّ DNS** لـ`staging-maps.rahalgo.com` ولا لـ`maps-staging.rahalgo.com`
(قِيس). **وإنشاؤه بيد المالك في لوحة النطاق.**

**فخُدمت الخرائطُ من مضيف التجهيز القائم** — `staging-api.rahalgo.com/maps/`
— **وهو يحقّق القرار كلَّه**: مضيفٌ للتجهيز وحدَه، **ونسخةٌ مستقلّةٌ من
الآثار**، **ولا يمرّ شيءٌ بمضيف الإنتاج.** **ومن أراد اسماً مستقلّاً لاحقاً
أضاف سجلَّ DNS وبدّل `STAGING_MAP_STYLE_URL` وحدَه.**

## ما بُني

| الملفّ | ما فيه |
|---|---|
| `deploy/staging/maps-sync.sh` | ينسخ الآثارَ من `/srv/rahalgo/maps` إلى `/srv/rahalgo-staging/maps` **ويُعيد كتابةَ نمط التجهيز** إلى مضيفه · **ويسقط إن بقي `maps.rahalgo.com` في أيّ ملفّ** · ويرفض أن يكون الهدفُ داخلَ المصدر |
| `deploy/staging/Caddyfile.staging` | `handle_path /maps/*` — **قائمةٌ بيضاء** (manifest · base · regions · map-resources · style · sprite) **وما عداها 404** |
| `deploy/staging/compose.staging.yml` | حاملُ الآثار **للقراءة فقط** (`:/srv/maps:ro`) · ونمطُ الويب على مضيف التجهيز |

	نسخةُ التجهيز            = 835M · مستقلّةٌ عن نسخة الإنتاج
	نمطُ التجهيز             = 0 إشارةٍ إلى maps.rahalgo.com
	نمطُ الإنتاج             = 3 إشارات — لم يُمَسّ

## البرهانُ التشغيليّ — متصفّحٌ حقيقيّ · دخولٌ فعليّ

**HTTPS:** النمط 200 · الرموز 200 · الفهرس 200 · الخطوط (`RahalGo Regular|Bold`) 200
· البلاطات (Range) **206** · مسارٌ خارجَ القائمة **404**.

| الخريطة | الموضع | لوحة | بلاطات | بعد التكبير | نداءاتُ الإنتاج |
|---|---|---|---|---|---|
| خريطةُ العمليات | `/dashboard/opsmap` | 1 | 11 | **+25** | **0** |
| المدن | الإعدادات ← المدن | 1 | 9 | **+20** | **0** |
| مناطقُ التغطية | الإعدادات ← مناطق التغطية | 1 | 9 | **+22** | **0** |
| موقعُ المتجر | `/dashboard/users` ← متجر جديد | 1 | 5 | **+9** | **0** |
| موقعُ المكتب | الإعدادات ← إعدادات الموقع ← صفحات الموقع ← تواصل معنا | 1 | 13 | **+6** | **0** |

**ولا حفظَ في شيءٍ منها** — نافذةُ المتجر أُغلقت بلا حفظ، والمتاجرُ ٢ قبلها
و٢ بعدها.

**والخمسُ هي كلُّ خرائط اللوحة** — `OpsMapCanvas` و`ZonesMap` (المدن
والمناطق) و`PickMap` (المتجر والمكتب). **ولا مستهلكَ سادس** (بحثٌ في
`apps/rahalgo/src`).

> **وخريطةُ `/contact` في الموقع العامّ لا تُرسم في التجهيز** — **لأنّ
> `platform.location` فارغٌ فيه** (DATA_ONLY، §٦). **وليست من اللوحة.**

## الحرّاس

	TestStagingMapsAreServedFromStagingItself   نمطُ التجهيز على مضيف التجهيز · يمرّ بـ/maps/
	                                            وكتلةُ Caddy · وحاملُ القراءة · وأداةُ النسخ
	TestStagingConfigNeverNamesProduction       (envguard — لم يُمَسّ)

**والشاهدُ السالب**: جُعل نمطُ التجهيز `https://maps.rahalgo.com/...`
مؤقّتاً ⇒ **سقط الحارسان كلاهما** (`deploycheck` و`envguard`) ⇒ أُعيد
الملفُّ بالبصمة نفسِها.

---

# ٦ · قيمُ الإعدادات — من ٥٣ فرقاً إلى ٢٤

## ٦·١ · قرارُ المالك وتصنيفُه

| الصنف | العدد | المعنى |
|---|---|---|
| **A · PRODUCT_POLICY_MUST_MATCH** | **29** | يجب أن يطابق الإنتاج — **سُوّي** |
| **B · INTENTIONAL_LAUNCH_DIFFERENCE** | **2** | `site.show_login` · `site.show_shop` — **الإنتاجُ قبلَ الافتتاح** (قرارُ ٢٠٢٦-٠٨-١٧) · يبقى |
| C · ENVIRONMENT_PROVIDER_DIFFERENCE | 0 | — |
| D · DATA_CONTENT_DIFFERENCE | 0 | (المحتوى الستّةَ عشر مصنَّفٌ DATA_ONLY منذ التدقيق الأوّل) |
| E · REQUIRES_OWNER_DECISION | 0 | — |

## ٦·٢ · التسوية — بباب اللوحة لا بـSQL

**`PUT /api/v1/admin/settings/{key}`** بحساب المالك على التجهيز. **وستّةَ
عشرَ مفتاحاً ماليّاً طلبت إثباتَ الهويّة (Step-Up) فأُعطي** — **ولم
يُتجاوَز.**

	طُبّق              = 29 / 29
	قيس في القاعدة     = 29 / 29 تساوي الإنتاج
	سجلُّ التدقيق      = 606 ⇐ 636 — 29 admin.setting_update + دخولٌ واحد
	نسخةٌ قبل التبديل  = لكلّ صفّ (27 كانت على الافتراض · 2 مخزَّنة)

| المفتاح | التجهيز قبل | التجهيز بعد = الإنتاج |
|---|---|---|
| `customers.signup_bonus` | 0 | 15 |
| `delivery.fee` | 0 | 100 |
| `pricing.margin_fixed` | 0 | 50 |
| `merchants.commission_percent` | 0 | 10 |
| `merchants.return_support_percent` | 0 | 20 |
| `sales.commission_percent` | 0 | 10 |
| `referral.reward_1` · `_2` · `_3` · `_rest` | 0 | 30 · 20 · 10 · 5 |
| `drivers.monthly_target` · `target_2` · `target_3` · `target_reward` | 0 | 50 · 100 · 200 · 200 |
| `drivers.reward_2` · `reward_3` | 0 | 400 · 700 |
| `sales.target_2` · `target_3` · `target_reward` | 0 | 15 · 25 · 200 |
| `sales.reward_2` · `reward_3` | 0 | 400 · 600 |
| `auth.otp_login` | true | **false** |
| `auth.signup_verify` | false | **true** |
| `auth.require_whatsapp` | true | **false** |
| `drivers.require_whatsapp` | true | **false** |
| `sales.require_whatsapp` | true | **false** |
| `customers.max_addresses` | 10 | 4 |
| `orders.max_sources` | 2 | 1 |
| `shop.rail_auto` | true | false |

## ٦·٣ · ما لم يُمَسّ — وقيس

	الطلب #1050        = on_the_way · delivery_fee 0 · platform_commission 2100
	                     · snap_merchant 0 · snap_rep 0 · pricing_margin · 21000
	                     — **قبل التسوية وبعدها حرفاً** (الطلبُ القائم يحمل لقطتَه)
	المحافظ            = 4 حركات · مجموعُ الأرصدة 0 — قبل وبعد
	الطلبات            = 1 — قبل وبعد
	moneycheck         = 51 فحصاً · 0 خرق (على قاعدة التجهيز بعد التسوية)
	أعلامُ الإطلاق     = لم تُمَسّ في البيئتين
	site.show_login · site.show_shop = لم يُمَسّا (B)

## ٦·٤ · التحقّق — مزوّدُ الرموز

**السياسةُ صارت سياسةَ الإنتاج** — **والمزوّدُ آمنٌ للتجهيز:**

	التجهيز  OTP_PROVIDER = dev  (الرمزُ يُكتب في سجلّ التجهيز · لا يصل أحداً)
	الإنتاج  OTP_PROVIDER = مزوّدُه (٨ أحرف) — **لم يُنسخ إلى التجهيز**
	SMS_URL · SMS_OTP_URL = فارغان في البيئتين

**و`auth.signup_verify = true` يعني**: التسجيلُ في التجهيز يطلب رمزاً، **والرمزُ
في سجلّ خادم التجهيز.** **ومن اختبر التسجيلَ (`C2`) يقرؤه من هناك.**

## ٦·٥ · الفروقُ الباقية — ٢٤ · كلُّها مقصودة

	ENVIRONMENT_ONLY — الإطلاق (6)
	    launch.customer_browse · customer_custom_orders · customer_orders
	    launch.customer_signup · driver_work · merchant_orders
	    الإنتاج = false · التجهيز = true   (الإنتاجُ قبلَ الافتتاح)

	ENVIRONMENT_ONLY — B (2)
	    site.show_login · site.show_shop    الإنتاج = false · التجهيز = true

	DATA_ONLY — الهويّةُ والمحتوى (16)
	    platform.name · logo · background · background_dim · address · location
	    platform.facebook · whatsapp · support_phone
	    page.about_text · help_text · privacy_text · terms_text
	    auth.background_dim · home.banner_seconds · whatsapp.otp_template

---

# ٧ · الحرّاسُ الدائمة

	backend/internal/deploycheck/admin_parity_test.go
	  TestAdminRuntimeConfigSurfaceMatches                     المفاتيحُ نفسُها
	  TestAdminRuntimeConfigHasNoEmptyDefault                  لا فراغَ يعطّل — ولا استثناءَ بعد G1
	  TestAdminRuntimeConfigMatchesProductionExceptEnvironment ما لا يخصّ البيئةَ يطابق
	  TestStagingMapsAreServedFromStagingItself                خرائطُ التجهيز من التجهيز

**و`knownEmptyInStaging` صار فارغاً** — **فأيُّ فراغٍ في تهيئة ويب التجهيز
يُسقط الحارس.**

| البُعد | ولماذا لا حارسَ وحدة |
|---|---|
| قيمُ الإعدادات | **في القاعدة لا في الشيفرة** — تُقاس في التدقيق (§٦) |
| الأدوارُ في القاعدة | يلزمه اتّصالٌ بالقاعدتين |

---

# ٨ · حالُ التجهيز بعد القرارات

	ويبُ اللوحة  = release-68a45c97
	المحرّك      = release-68a45c97 · 0157
	التركيب      = مطابقٌ للمستودع حرفاً (compose · Caddyfile · maps-sync)
	الخرائط      = /srv/rahalgo-staging/maps (835M) · تعمل في الخمس
	الإعدادات    = 29 سُوّيت · 2 بقيت (B)
	البيانات     = #1050 on_the_way · لقطتُه لم تتبدّل · moneycheck 51/51
	الإنتاج      = لم يُمَسّ · release-79d88245 · 0155

---

# ٩ · الفحصُ المسبق لنشر الإنتاج — **لا نشر**

> **قرارُ المالك الثالث**: «PRODUCTION PARITY DEPLOYMENT PREFLIGHT —
> DO NOT DEPLOY.» **والإنتاجُ قُرئ فقط** (`BEGIN READ ONLY … ROLLBACK`).

## ٩·١ · الهدف

	الإنتاجُ الآن     = 79d88245e7c238f92abf661531738a242bb4fa2b · 0155_campaigns.sql
	الهدفُ المقترح    = 68a45c97f18c350344949bc252947d98273a2241 · 0157_section_delete_contract.sql
	المحرّك          = rahalgo-api:release-68a45c97
	                   sha256:e6bb3dfbc48edeba0e28b488d098e11b2d2e80067bc60fde21e005e6e7e06afe
	الويب            = rahalgo-web:release-68a45c97
	                   sha256:3a3e569d960541511fed20dbabcc53e355edba583a0fa6cfe8db4e3d4d84d129

**ولماذا `68a45c97` لا `HEAD`**: **ما بعده لا يمسّ شيفرةَ المحرّك ولا
الويب** (وثائق · أندرويد · حارسٌ · تهيئةُ تجهيز). **وصورتاه هما ما يعمل
في التجهيز الآن وعليهما قِيس كلُّ شيء** — **فيُرقّى الأثرُ عينُه ولا
يُبنى غيرُه** (`promote.sh`). **والصورتان لهما ملفّا إصدارٍ على الخادم
بالبصمة نفسِها** (`/srv/rahalgo/artifacts/release-{api,web}-68a45c97.json`).

## ٩·٢ · الفرق — ١٥ التزاماً · كلُّ ملفٍّ مصنَّف

**من `79d88245` إلى `68a45c97`:**

| الصنف | الملفّات | الأثرُ على الإنتاج |
|---|---|---|
| **محرّك — تشغيل** | `server/admin_sections_handlers.go` | حذفُ قسمٍ مشغول ⇒ **409** بدل 500 (G4) |
| | `server/platform_hours_handlers.go` | إيقافُ المنصّة **يُكتب في التدقيق** (G5) |
| | `server/sections_handlers.go` | **`count` في أقسام الزبون صار «وجودَ محتوى»** لا «مفتوحٌ الآن» · وحقلٌ جديد `orderable_now` — **قرارُ المالك ٢٠٢٦-٠٩-١٦** |
| | `catalog/catalog.go` · `catalog/citysql.go` | مدينةُ المتجر تُشتقّ بتعبيرٍ واحد · **وتُعاد حين يتبدّل موقعُه** |
| **هجرات** | `0156_launch_sections.sql` · `0157_section_delete_contract.sql` | §٩·٣ |
| **ويب** | `i18n/ar.json` (+٢ سطر) | نصُّ `section_has_items` · تسميةُ `admin.platform_closure` (G6) |
| **أداةٌ لا تُشحن** | `cmd/seed/*` | **الصورةُ تبني `./cmd/api` وحدَه** (`backend/Dockerfile`) |
| | `impact/model.go` · `web/scripts/check-app-error-codes.mjs` | أدواتُ فحص |
| **اختبارات** | `qa/*` · `server/*_test.go` · `catalog/*_test.go` · `deploycheck` · `envguard` · `truthdoc` · `impact_test.go` | لا تُشحن |
| **أندرويد وحدَه** | `mobile/*` | لا يمسّ الخادم |
| **وثائق** | `docs/*` · `artifacts/*` · `CLAUDE.md` | لا يمسّ الخادم |
| **تهيئةُ التجهيز** | `deploy/staging/*` | **لا يمسّ الإنتاج** |

**ولا تبديلَ في**: `deploy/compose.yml` (**والذي على الخادم يطابق المستودع
حرفاً** — قِيس) · `deploy/Caddyfile` · `server/server.go` · `settings/` ·
`authz/` · `ledger` · `orders/`.

**وأثرٌ يُقال للمالك**: **تغييرُ `count`** يصل تطبيقاتِ الزبون المثبّتة —
**القسمُ الذي متاجرُه مغلقةٌ بالساعة يبقى ظاهراً** بدل أن يختفي. **وهو ما
قرّره المالك** — **والإنتاجُ الآن بلا متاجرَ ولا أصناف، فلا أثرَ على بيانات.**

## ٩·٣ · الهجرتان على بيانات الإنتاج — محاكاةٌ للقراءة

	platform_sections         = 38 · فعّالةٌ 38
	0156 · يُدرج             = 0   (الأسماءُ الثمانيةُ والثلاثون موجودة)
	0156 · يُبدّل أيقونةً/ترتيباً = 0
	0156 · يُطفئ             = 0   (لا قسمَ من الستّة القديمة فعّال)
	0156 · أسماءٌ فعّالةٌ مكرّرة = 0   (شرطُ الفهرس الفريد)
	0156 · أقسامٌ أضافها المالك = 0   (لا يمسّها أصلاً)
	0157 · menu_items         = 0 · بلا قسم 0 · يتيمة 0
	0157 · المفتاحُ الآن       = n (SET NULL) ⇐ r (RESTRICT)
	merchants · store_sections · orders = 0 · 0 · 0

**فالهجرتان على الإنتاج**: **فهرسٌ واحدٌ يُنشأ ومفتاحٌ يُعاد تعريفُه على
جدولٍ فارغ.** **لا صفَّ يُكتب فوقه ولا صفَّ يُحذف.**

**وكلُّ هجرةٍ في معاملتها** (`migrate.Up`) — **فإن سقطت لم يبقَ منها شيء
ولم يُقلع المحرّك.**

**وسلسلةٌ قائمةٌ لا تُضيفها الهجرتان**: `store_sections → platform_sections`
**CASCADE** — **قديمة**، و`store_sections` في الإنتاج **صفر.** تُذكر ولا تُعدَّل.

## ٩·٤ · التراجع

**بالصورة لا بالقاعدة:**

	promote.sh ... rahalgo-api:release-79d88245   (sha256:187053aedeb3…)
	RAHALGO_WEB_IMAGE=rahalgo-web:release-79d88245 (sha256:ddebbc60e2a6…)

**والمحرّكُ القديم يُقلع على قاعدةٍ في `0157`** — `migrate.Up` **يطبّق
الناقصَ من ملفّاته ولا يسأل عن زائد.** **والمخطّطُ الزائدُ لا يضرّه**:
فهرسٌ فريدٌ على أسماءٍ فريدةٍ أصلاً، ومفتاحٌ `RESTRICT` كان حذفُه يسقط
بـ500 قبلَه.

**وإن لزم إرجاعُ المخطّط نفسِه** (لا يُتوقَّع): **الاستعادةُ من النسخة
المأخوذة قبل النشر** (§٩·٥).

**ونقصٌ يُسدّ قبل النشر**: صورتا `79d88245` **في مخزن الصور بوسمهما**
**لكن بلا أرشيفٍ ولا ملفِّ إصدارٍ في `artifacts/`** — **فيُحفظان أرشيفاً
قبل النشر** كي لا يعتمد التراجعُ على بقاء الوسم.

## ٩·٥ · النسخُ الاحتياطيّ — قبل أيّ تبديل

	١ · pg_dump -Fc لقاعدة الإنتاج ⇐ /srv/rahalgo/backups/rahalgo-pre68a45c97-<UTC>.dump
	    (آخرُ نسختين: 16 أيلول · 2.6MB و2.7MB · المساحةُ الحرّة على / 18G)
	٢ · docker save rahalgo-{api,web}:release-79d88245 ⇐ artifacts/ + sha256
	٣ · نسخةٌ من /srv/rahalgo/deploy/Caddyfile — وإن لم يُمَسّ

## ٩·٦ · الترتيبُ المقترح

	١ · النسخُ الاحتياطيّ (§٩·٥)
	٢ · المحرّك: TARGET_ENV=production promote.sh ... rahalgo-api:release-68a45c97
	    https://api.rahalgo.com/api/v1/public/identity sha256:e6bb3dfb…
	    (الهجرتان تُطبَّقان عند الإقلاع)
	٣ · فحوصُ المحرّك (§٩·٧) — وإن سقط واحدٌ: التراجعُ فوراً
	٤ · الويب: RAHALGO_WEB_IMAGE=rahalgo-web:release-68a45c97
	    docker compose ... up -d --no-build --no-deps web
	٥ · فحوصُ الويب

**المحرّكُ قبل الويب** — **الويبُ الجديدُ لا يطلب شيئاً من المحرّك
القديم**، والعكسُ ينتج نافذةً قصيرةً يُرى فيها مفتاحُ الخطأ خاماً.

## ٩·٧ · فحوصُ ما بعد النشر

| # | الفحص | المنتظَر |
|---|---|---|
| 1 | `/api/v1/public/identity` | production · `68a45c97…` · `0157_section_delete_contract.sql` |
| 2 | سجلُّ المحرّك | `migrations applied count=2` · ولا `migrate:` خطأ |
| 3 | قراءة: `platform_sections` | 38 · فعّالةٌ 38 · البصمةُ `b4a77017…` |
| 4 | قراءة: الفهرسُ والمفتاح | `platform_sections_active_name_uq` موجود · `confdeltype = r` |
| 5 | قراءة: `app_settings` | **بصمتُها قبل النشر = بعده** (أعلامُ الإطلاق خصوصاً) |
| 6 | `moneycheck` على الإنتاج | 51/51 |
| 7 | أبوابُ الإدارة للقراءة (86) بلا توثيق | 401 كلُّها |
| 8 | صفحاتُ اللوحة (29) | 200 كلُّها |
| 9 | `https://rahalgo.com/config.js` | `apiUrl = https://api.rahalgo.com` · `mapStyleUrl = maps.rahalgo.com/...` |
| 10 | `maps.rahalgo.com/.../style.online.json` | 200 |
| 11 | `staging-api.rahalgo.com/api/v1/public/identity` | staging — **التجهيزُ لم ينقطع** |
| 12 | `GET /api/v1/public/sections` | 200 · فيه `orderable_now` (قبل النشر: 200 بلا هذا الحقل — قِيس) |
| 13 | بيدِ المالك في اللوحة | الدخول · الأقسام · سجلُّ التدقيق يسمّي `admin.platform_closure` |

**ولا يُرسَل إلى الإنتاج فعلُ كتابةٍ للفحص** — **الحذفُ ٤٠٩ مثبتٌ في
التجهيز وفي الاختبار** (`admin_section_delete_test.go`).

## ٩·٨ · خطرٌ قائمٌ لا يمسّه هذا النشر

**`Caddyfile` الإنتاج على الخادم = ما في المستودع + كتلةُ `staging-api.rahalgo.com`**
(تُمرّر إلى بوّابة التجهيز). **وهي ليست في المستودع.** **فمن نسخ
`deploy/Caddyfile` من المستودع فوق الخادم قطع التجهيزَ كلَّه** — لا الإنتاج.
**والنشرُ المقترحُ لا يمسّ Caddy.** **وإدخالُها المستودعَ قرارٌ للمالك.**
