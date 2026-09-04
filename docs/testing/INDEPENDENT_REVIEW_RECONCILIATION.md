# رحّال غو — مطابقةُ المراجعة المستقلّة

> **CODE TRUTH BASELINE**: `26f93c5d` — **ولا تغييرَ تشغيليٌّ في هذا العمل.**
>
> **والقاعدة**: **لا يُقبَل ادّعاءُ المراجعة الخارجيّة، ولا يُدافَع عن تقريري.**
> **كلُّ بندٍ قِيس من الشيفرة من جديد.**

---

## ٠ · الجدول

| ID | ادّعاءُ المراجعة | ما كنتُ أقول | إعادةُ القياس | الحكم | الدليل | يغيّر القانونيّ؟ |
|---|---|---|---|---|---|---|
| **R1** | تعريفاتُ الإعدادات **١١٨** لا ١٢٧ | `SETTINGS CATALOG = 127` | **١١٨ تعريفاً · ٩ مراجعَ `ShowWhen` · ١١٨+٩=١٢٧** | **المراجعةُ محقّة** ✅ | تحليلٌ نحويٌّ بـ`go/ast` | **نعم** |
| **R2** | مفاتيحُ مؤثّرةٌ خارجَ الستّين | `BEHAVIOR-CHANGING = 60` | **٩٥ من ١١٨** — و**٣٥ مفتاحاً** أُثبت أثرُها ولم تكن في القائمة | **المراجعةُ محقّة** ✅ | جدولُ ٤ أدناه | **نعم** |
| **R3** | «تذكّرني» تنكسر بعد أوّل تجديد | **لم يُفحص قطّ** | **الادّعاءُ صحيحٌ ومسارُه كاملٌ ومُثبَت** | **PROVEN DEFECT — D3** | `client.ts:97,135` | **نعم** |
| **R4** | فشلٌ جزئيٌّ في قبول السائق | **`accept` سلوكٌ مُثبَتٌ آمن** (القفلُ والحالة) | **الذرّيّةُ صحيحة · ولكنّ ثلاثَ كتاباتٍ خارجَ معاملةٍ واحدة، والتعويضُ ناقصٌ وأفضلُ جهد** | **PROVEN RISK — R7** | `driver_handlers.go:598-625` | **نعم** |

**وحكمُ الأربعة**: **المراجعةُ الخارجيّةُ محقّةٌ في الأربعة.**
**ولم يسقط لها ادّعاءٌ واحد** — وسقط لي **مقامان** و**سلوكٌ ظننتُه آمناً.**

---

# ١ · مقامُ الإعدادات — **١١٨ لا ١٢٧**

## القياس

**لم يُعدّ نصّاً بل نحواً**: `go/parser` على `internal/settings/*.go`، **ويُفصل
`Def{Key: …}` عن `Condition{Key: …}`** بنوع الحرفيّ المركّب لا بشكل السطر.

```
TOTAL SETTING DEFINITIONS   = 118
UNIQUE SETTING KEYS         = 118
SHOWWHEN KEY REFERENCES     =   9
DUPLICATE SETTING DEFINITIONS =  0
```

## وكيف وقع خطئي بالضبط

**عدَدتُ `Key:` بتعبيرٍ نمطيّ.** و`grep -c 'Key:'` يعطي **١٤٩** — فيها
تعليقاتٌ وحقولُ بنية. **وحين نظّفتُها يدويّاً بقي ١٢٧.**

**والمئةُ والسبعةُ والعشرون هي بعينها ١١٨ + ٩:**

| | |
|---|---|
| تعريفاتٌ حقيقيّة | **١١٨** |
| `ShowWhen: &Condition{Key: …}` — **إشارةٌ إلى مفتاحٍ قائمٍ لا تعريفُ جديد** | **٩** |
| **المجموع** | **١٢٧** |

**والتسعةُ كلُّها تشير إلى مفاتيحَ معرَّفةٍ أصلاً** (`platform.background` ·
`auth.background` · `home.banner_auto` · `shop.rail_auto` ·
`drivers.assignment_mode` ×٢ · `drivers.direct_assign` ·
`delivery.by_distance` ×٢) — **فعُدَّ كلُّ واحدٍ منها مرّتين.**

**والدرسُ نفسُه الذي تكرّر**: **العدُّ النصّيُّ يعدّ الشكلَ، والنحويُّ يعدّ
المعنى.** وهو ما أسقط ادّعاءاتي التسعةَ السابقة.

---

# ٢ · مقامُ القراءة — **٠ مجهول في الاتّجاهين**

## الاتّجاه الأوّل — `SETTING → READERS`

**١١٨ مفتاحاً بُحث عن كلّ ظهورٍ لها في ١٠٥٦ ملفّاً** (`.go .ts .tsx .kt .sql
.js .json`) خارجَ `node_modules` و`.next` و`.git`.

## الاتّجاه الثاني — `READER → KEY`

**٣٧٨ موضعَ نداءٍ لدوالّ القراءة** (`Get` · `GetString` · `GetInt` ·
`GetBool` · `GetNum` · `GetRaw` · `RequireWhatsApp` · `Default` · `Lookup`)،
**مصفّاةً من نداءات `chi.Router.Get` و`url.Values.Get`.**

## الحصيلة

```
DEFINED BUT NEVER READ = 0
READ BUT NOT DEFINED   = 0
UNRESOLVED             = 0
```

## وثلاثةُ أنماطِ قراءةٍ كان حارسي يفوّتها

**والحارسُ القائم** (`read_keys_test.go`) **نمطُه**:
`\.Get(?:String|Int|Bool|Raw)\(…"([a-z_]+\.[a-z_]+)"` — **فيفوّت ثلاثةً:**

| # | النمط | الموضع | المفاتيح |
|---|---|---|---|
| ١ | **بناءٌ ديناميكيٌّ ببادئة الدور** | `incentives/target.go:86-101` — `p := prefixOf(role)` ثمّ `p+"target_2"` | **١٢** |
| ٢ | **غلافٌ يقرأ مفتاحين** | `settings.go:232` — `RequireWhatsApp` تقرأ `auth.require_whatsapp` ثمّ مفتاحَ الدور | **٤** |
| ٣ | **مفتاحٌ يُختار في `switch` ثمّ يُمرَّر متغيّراً** | `min_version.go:52-67` · `incentive_handlers.go:108` · `app_file_handlers.go` | **٦** |

**ولا واحدَ منها خطأ** — لكنّ **الحارسَ لا يراها**، فمفتاحٌ يُقرأ بها ويُحذف
من الكتالوج **لا يُسقط بناءً.**

> **وهذا لا يُصلَح الآن** — يُسجَّل في [`RUNTIME_VALIDATION_REQUIREMENTS.md`](RUNTIME_VALIDATION_REQUIREMENTS.md)
> كـ**فجوةِ حارس**، وقرارُها للمالك.

---

# ٣ · الستّون صارت **٩٥**

```
SETTINGS CATALOG            = 118
BEHAVIOR-CHANGING SETTINGS  =  95 / 118
NON-BEHAVIOR SETTINGS       =  23 / 118
DEFINED BUT NEVER READ      =   0
READ BUT NOT DEFINED        =   0
UNRESOLVED                  =   0
```

## الخمسةُ والثلاثون التي فاتتني — **وكلُّها مُثبَتة**

| المفتاح | يقرؤه | الطريقة | المفتاح ساكنٌ أم ديناميّ | المجال | يحتاج تشغيلاً؟ |
|---|---|---|---|---|---|
| `app.min_version.customer` | `server/min_version.go:55,67` | `GetInt` بمتغيّر | **`switch` ثمّ متغيّر** | **بوّابةُ نسخة** | **نعم** |
| `app.min_version.driver` | `min_version.go:57,67` | كسابقه | كسابقه | بوّابةُ نسخة | نعم |
| `app.min_version.merchant` | `min_version.go:59,67` | كسابقه | كسابقه | بوّابةُ نسخة | نعم |
| `app.min_version.rep` | `min_version.go:61,67` | كسابقه | كسابقه | بوّابةُ نسخة | نعم |
| `auth.otp_channel` | `cmd/api/main.go:157` | `GetString` في مغلّف | ساكن | **قناةُ الرمز** | **نعم** |
| `auth.sms_template` | `main.go:153` | `GetString` | ساكن | نصُّ الرمز المُرسَل | نعم |
| `whatsapp.otp_template` | `main.go:95` | `GetString` | ساكن | نصُّ الرمز المُرسَل | نعم |
| `whatsapp.send_delay_ms` | `main.go:113` | `GetInt` في `SetDelay` | ساكن | **تمهّلُ الإرسال** | نعم |
| `customers.require_whatsapp` | `orders/service.go:335` | `RequireWhatsApp` | **غلاف** | **بوّابةُ إنشاء طلب** | **نعم** |
| `drivers.require_whatsapp` | `driver_handlers.go:213` | `RequireWhatsApp` | غلاف | **بوّابةُ الورديّة** | نعم |
| `sales.require_whatsapp` | `leads_handlers.go:414` | `RequireWhatsApp` | غلاف | بوّابةُ الفرص | نعم |
| `referral.reward_1` | `referrals.go:157` | `GetInt` | ساكن | **مال** | **نعم** |
| `referral.reward_2` | `referrals.go:159` | `GetInt` | ساكن | مال | نعم |
| `referral.reward_3` | `referrals.go:161` | `GetInt` | ساكن | مال | نعم |
| `drivers.monthly_target` | `incentives/target.go:93` | `GetInt` | **`p+"monthly_target"`** | **مال** | **نعم** |
| `drivers.target_reward` | `target.go:94` · `incentive_handlers.go:108` | `GetInt` | ديناميّ + ساكن | مال | نعم |
| `drivers.target_2` | `target.go:96` | `GetInt` | ديناميّ | مال | نعم |
| `drivers.reward_2` | `target.go:97` | `GetInt` | ديناميّ | مال | نعم |
| `drivers.target_3` | `target.go:99` | `GetInt` | ديناميّ | مال | نعم |
| `drivers.reward_3` | `target.go:100` | `GetInt` | ديناميّ | مال | نعم |
| `sales.target_reward` | `target.go:94` · `incentive_handlers.go:110` | `GetInt` | ديناميّ + ساكن | مال | نعم |
| `sales.target_2` | `target.go:96` | `GetInt` | ديناميّ | مال | نعم |
| `sales.reward_2` | `target.go:97` | `GetInt` | ديناميّ | مال | نعم |
| `sales.target_3` | `target.go:99` | `GetInt` | ديناميّ | مال | نعم |
| `sales.reward_3` | `target.go:100` | `GetInt` | ديناميّ | مال | نعم |
| `media.max_upload_mb` | `media/media.go:195` | مغلّف `s.setting` | ساكن | **حدُّ الرفع — يردّ الملفّ** | **نعم** |
| `site.show_login` | `customer_handlers.go:100` | `GetBool` | ساكن | بوّابةُ واجهة | نعم |
| `site.show_shop` | `customer_handlers.go:101` | `GetBool` | ساكن | بوّابةُ واجهة | نعم |
| `site.join_open` | `customer_handlers.go:104` | `GetBool` | ساكن | **بابُ الانضمام** | نعم |
| `home.banner_auto` | `customer_handlers.go:275` | `GetBool` | ساكن | دورانُ الشريط | لا |
| `home.banner_seconds` | `customer_handlers.go:276` | `GetInt ×1000` | ساكن | دورانُ الشريط | لا |
| `shop.rail_auto` | `customer_handlers.go:277` | `GetBool` | ساكن | دورانُ الرفّ | لا |
| `shop.rail_seconds` | `customer_handlers.go:278` | `GetInt ×1000` | ساكن | دورانُ الرفّ | لا |
| `platform.app_url` | `customer_handlers.go:498` | `GetString` | ساكن | **وجهةُ التحميل** | نعم |
| `platform.app_file` | `app_file_handlers.go:53,121,142,161` | `GetString` بثابت | **ثابتٌ متغيّر** | **ملفُّ التحميل** | نعم |

## والثلاثةُ والعشرون التي **لا** تغيّر سلوكاً — **عرضٌ وهويّةٌ لا غير**

```
auth.background · auth.background_dim · auth.background_mobile
platform.background · platform.background_dim · platform.background_mobile
platform.logo · platform.name · platform.address · platform.location
platform.support_phone · platform.facebook · platform.instagram
platform.telegram · platform.whatsapp · platform.invite_code
page.help_text · page.terms_text · page.privacy_text · page.about_text
page.driver_help_text · page.rep_help_text · page.merchant_help_text
```

**و`platform.invite_code` بينها بحدٍّ**: **تُقرأ في `leads_handlers.go:293`
وتُردّ قيمةً تُعرض** — **ولا يُقارَن بها شيءٌ ولا تفتح باباً.** والتحقّقُ من
رموز الدعوة يجري على `users.invite_code` لا عليها.

---

# ٤ · **D3** — «تذكّرني» تنقلب دائمةً بعد أوّل تجديد

## المسار كاملاً

| # | الخطوة | الموضع | ما يقع |
|---|---|---|---|
| ١ | **الدخول** | `LoginCard.tsx:212` | `tokenStore.set(result.tokens, remember)` — **والاختيارُ يُمرَّر** ✅ |
| ٢ | **اختيارُ الخزنة** | `client.ts:97-102` | `set(tokens, remember = true)` ثمّ `const store = remember ? localStorage : sessionStorage` |
| ٣ | **`remember=false`** | — | **الجلسةُ في `sessionStorage`** ✅ **كما وُعد** |
| ٤ | **ما يطلق التجديد** | `client.ts:144` · `client.ts:190` | **أيُّ ٤٠١** — **وعمرُ توكن الوصول ١٥ دقيقة** (`cmd/api/main.go:71`) |
| ٥ | **نتيجةُ التجديد** | `client.ts:135` | **`tokenStore.set(result.tokens)`** — **بلا معامل** |
| ٦ | **الافتراض** | `client.ts:97` | **`remember = true`** |
| ٧ | **تنظيفُ القديم** | `client.ts:98` | **`this.clear()` تمسح من الخزنتين** — **فلا يبقى أثرٌ في `sessionStorage`** |
| ٨ | **الخزنةُ الناتجة** | `client.ts:99` | **`localStorage`** ❌ |
| ٩ | **بعد إغلاق المتصفّح** | — | **الجلسةُ باقية** — **وهو ضدّ ما اختاره المستخدم** |

## والموضعان الآخران يفعلان الشيءَ نفسَه

**`provider.tsx:73`** و**`SsoPage.tsx:38`** — **كلاهما `set(tokens)` بلا معامل.**

## الحكم

> # **PROVEN DEFECT — D3**
> **`Remember-Me persistence changes after token refresh`**
>
> **والحالةُ الخاطئةُ قابلةُ الوصول بيقين**: خمسَ عشرةَ دقيقةً من الاستعمال
> تكفي. **ولا تُخطئ إلّا إن أُغلق اللسانُ قبلها.**
>
> **وأثرُه أمنيٌّ لا تجميليّ**: **جهازٌ مشترَكٌ** — وهو بعينُه ما كُتب في
> تعليق `client.ts:85` أنّ الميزةَ وُجدت له.
>
> **ولم يُصلَح** — بأمر المالك.

---

# ٥ · **R7** — قبولُ السائق: ثلاثُ كتاباتٍ خارجَ معاملةٍ واحدة

## حدودُ المعاملة — **لا معاملة**

| # | الكتابة | الموضع | داخلَ معاملة؟ |
|---|---|---|---|
| **W1** | `UPDATE orders SET driver_id, offered_driver_id=NULL, offer_expires_at=NULL WHERE driver_id IS NULL AND status='dispatching'` | `driver_handlers.go:598` | **لا — `Exec` مستقلّة** (وهي **ذرّيّةٌ في ذاتها**) |
| **W2** | `UPDATE users SET last_assigned_at = now()` | `:614` | **لا — والخطأُ يُسجَّل ولا يوقف** |
| **W3** | `orders.Transition(…)` | `:620` | **معاملتُها الخاصّة** |
| **W4** | **التعويض**: `UPDATE orders SET driver_id = NULL` | `:623` | **لا — وأفضلُ جهد: `_, _ =`** |

**فالثلاثةُ ليست في معاملةٍ واحدة، ولا تراجعَ تلقائيّ.**

## نقاطُ الفشل

| # | نقطةُ الفشل | معاملةٌ واحدة؟ | تراجعٌ تلقائيّ؟ | التعويضُ يعيد كلَّ الأعمدة؟ | أثرٌ زمنيٌّ باقٍ؟ | يمسّ الطابور؟ |
|---|---|---|---|---|---|---|
| **F1** | **بعد W1** (سقوطُ العمليّة أو **إلغاءُ سياق الطلب**) | **لا** | **لا** | **لا يعمل أصلاً** | **لا** | **نعم** — انظر أدناه |
| **F2** | **بعد W2** | لا | لا | **`last_assigned_at` لا يُعاد أبداً** | **نعم** | **نعم** — ترتيبٌ لا إسنادَ خلفه |
| **F3** | **قبل W3** | لا | لا | كـF1 | نعم | نعم |
| **F4** | **أثناء W3** | معاملةُ الانتقال ترجع | **للانتقال نعم** | **لا** — يعيد `driver_id` وحدَه، **و`offered_driver_id` و`offer_expires_at` ضاعا** | نعم | **في `rotation` نعم · في `queue` لا** |
| **F5** | **بعد W3 قبل الردّ** | — | — | — | لا | **لا** — والحالةُ صحيحةٌ كاملة |

## الحالةُ النهائيّةُ الخاطئةُ القابلةُ للوصول

**`status='dispatching'` و`driver_id` مضبوط** — **وهي حالةٌ لا يبلغها المسارُ
السويّ أبداً.**

**وأخطرُ ما فيها أنّها تفلت من كلّ آليّات الاستدراك:**

| الآليّة | الشرط | تراها؟ |
|---|---|---|
| مسحُ الدور — `rotation.go:417,469,625` | `status='dispatching' AND driver_id IS NULL` | **لا** ❌ |
| إنذارُ «بلا سائق» — `watchdog.go:54` | `driver_id IS NULL` | **لا** ❌ |
| **إنذارُ «طال»** — `watchdog.go:56` | `created_at < now() - delivery_timeout_min` **بلا شرطِ سائق** | **نعم** ✅ **متأخّراً** |
| قائمةُ السائق — `driver_handlers.go:518` | `driver_id = $1 AND closed_at IS NULL` | **نعم** ✅ |
| انتقالُ السائق | `dispatching → assigned` **مسموحٌ لـ`driverOps`** (`statuses.go:116`) | **يُصلحها بيده** ✅ |

**وإعادةُ الضغط على «قبول» تفشل** بـ`errOrderTaken` — لأنّ `driver_id IS NULL`
لم تعد صادقة. **فالسائقُ يقرأ «أُخذ الطلب» وهو صاحبُه.**

## ولماذا هذا قابلُ الوصول لا نظريّ

**الكتاباتُ كلُّها بـ`r.Context()`** — **وانقطاعُ السائق يُلغي السياق.**
**فتنجح W1** (نُفّذت وأُقرّت)، **ثمّ تفشل W3 و W4 معاً بالسياق نفسِه** —
**والتعويضُ يُهمَل خطؤه.** **وسائقٌ في الشارع بشبكةٍ ضعيفة** هو الحالُ لا
الاستثناء.

## الحكم

> # **PROVEN RISK — R7**
> **`Driver accept: non-transactional writes with best-effort compensation`**
>
> **وليس عيباً** — **لأنّ لا ثابتاً مُثبَتاً يُخالَف**: لا مالَ يُمسّ
> (المالُ عند التسليم)، **ولا إسنادَ مزدوجاً** (W1 ذرّيّة)، **والحالةُ
> تُرى وتُصلَح** من السائق ومن الإدارة، **ويُنذَر عنها آخرَ الأمر.**
>
> **وهو خطرٌ لأنّ الاستدراكَ الآليَّ لا يبلغها** — تنتظر إنسانًا.
>
> **ولم يُصلَح** — بأمر المالك.

**وهذا يُصحّح ما قلتُه سابقاً**: صنّفتُ `accept` **`PROVEN BEHAVIOR`** آمناً
بحجّة القفل والحالة. **والقفلُ يمنع الإسنادَ المزدوج ولا يمنع الحالةَ
النصفيّة** — **وسؤالي الأوّل كان عن التزامن وحدَه فلم أسأل عن الفشل.**

---

# ٦ · ما تغيّر في الحقيقة المجمّدة

| المقام | كان | صار |
|---|---|---|
| `SETTINGS CATALOG` | ١٢٧ | **١١٨** |
| `BEHAVIOR-CHANGING SETTINGS` | ٦٠ | **٩٥ / ١١٨** |
| `NON-BEHAVIOR SETTINGS` | ٢٦ | **٢٣ / ١١٨** |
| `DEFINED BUT NEVER READ` | لم يُقس | **٠** |
| `READ BUT NOT DEFINED` | لم يُقس | **٠** |
| `PROVEN DEFECTS` | ٢ | **٣** (+D3) |
| `PROVEN RISKS` | ٦ | **٧** (+R7) |
| `CODE TRUTH BASELINE` | `26f93c5d` | **`26f93c5d` — لم يتغيّر** ✅ |

**والتجميدُ باقٍ**: **لا سطرَ شيفرةٍ تشغيليّةٍ مسّ.**
