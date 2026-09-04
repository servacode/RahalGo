# رحّال غو — مطابقةُ المراجعة المستقلّة · الدفعةُ الثالثة

> **CODE TRUTH BASELINE**: `26f93c5d` — **ولا سطرَ شيفرةٍ تشغيليّةٍ مسّ.**
>
> **وهذه دفعةُ أمنٍ كلُّها** — والقياسُ من الشيفرة، **ولا ادّعاءَ يُقبل مسبقاً.**

---

## ٠ · الحصيلة

| ID | الادّعاء | الحكم | الرمز |
|---|---|---|---|
| **R3-1** | استعادةُ كلمة المرور تُبطل نوعَ عميلٍ واحدٍ لا الجلساتِ كلَّها | **PROVEN DEFECT** | **D10** |
| **R3-2** | إعادةُ الإدارة لكلمة المرور لا تُبطل شيئاً | **PRODUCT DECISION + RISK** | **D-05 · R13** |
| **R3-3** | `force_password_change` بلا بوّابةٍ في تطبيقات أندرويد الأربعة | **PROVEN DEFECT** | **D11** |
| **R3-4** | رمزُ الدفع لا يُلغى عند الخروج | **PROVEN DEFECT** | **D12** |
| **R3-5** | سردُ مجلّد الوسائط يكشف إثباتاتِ التسليم | **PROVEN SECURITY / PRIVACY DEFECT** | **D13** |
| **R3-6** | الويب سوكت بلا فحص حالةٍ ولا إعادةِ تحقّق | **PROVEN DEFECT (اتّصالٌ جديد) + PROVEN RISK (اتّصالٌ قائم)** | **D14 · R14** |
| **R3-7** | سحبُ الدور لا يسري لأنّ الأدوارَ في التوكن | **PROVEN RISK** | **R15** |
| **R3-8** | إنشاءُ حسابِ موظّفٍ غيرُ ذرّيّ | **PROVEN DEFECT** | **D15** |
| **R3-9** | أخطاءُ Redis تُهمَل والإبطالُ يفشل مفتوحاً | **PROVEN RISK** | **R16** |

**والمراجعةُ محقّةٌ في التسعة.** **وسقط لها تعليلان**: `handleWS` **يفحص**
`SessionRevoked` (لا يغفله)، **و`AdminUpdateUser` يُبطل التوكنات في القاعدة**
— **والثغرةُ في `ActiveStatus` وفي مفاتيح Redis، لا في غياب الإبطال أصلاً.**

---

# R3-1 — **D10** · استعادةُ كلمة المرور تُبقي الجلساتِ الأخرى

## CODE PATH
```
ConfirmPasswordReset      identity/service.go:374
  → repo.SetPassword                        :397   (UPDATE users فقط — لا إبطال)
  → issueFor                                :402
      → NeedsPin? …                         :639
      → issueSession(…, sessionID = "")     :655
          → revokeClientSessions(user, client)  :696
```

## INTENDED CONTRACT
`service.go:400` — **نصّاً**:
> «**جلسة جديدة تُبطل كل ما سبق — من سرق الحساب يخرج فوراً**»

## ACTUAL BEHAVIOR
**`revokeClientSessions` لا `revokeAllSessions`** (`:772`) — **تقرأ
`ClientSessionIDs(userID, client)` وتُبطل نوعَ العميل الحاليَّ وحدَه.**

**والأنواعُ مغلقةٌ في `client_kind.go:95`**: `web` + منصّةٌ مع تطبيق —
`android-customer` · `android-driver` · `android-merchant` · `android-rep`.

## SUCCESS PATH
**صاحبُ متجرٍ له ثلاثُ جلسات** (`web` · `android-merchant` ·
`android-customer`) **يستعيد كلمتَه من `android-merchant`:**

| الجلسة | بعد الاستعادة |
|---|---|
| `android-merchant` | **مُبطَلة** ✅ |
| `web` | **صالحة** ❌ |
| `android-customer` | **صالحة** ❌ |

**وتوكنُ التجديد في الاثنتين يبقى صالحاً حتّى `security.session_days`** —
**لا خمسَ عشرةَ دقيقة.**

## FAILURE / SECURITY PATH
**ومسارُ الأدمن أضيق**: `issueFor` تقف عند رمز PIN، **و`VerifyPin` تنادي
`issueSession(…, "")`** — **فالإبطالُ بالنوع أيضاً.**

## ⚠️ وقرارُ المالك لا يغطّي هذا
**الإبطالُ صار بالنوع بقرارٍ مكتوب** (`service.go:665`، ٢٠٢٦-٠٨-١١):
> «**كان دخولٌ جديدٌ يُبطل كلَّ الجلسات … ومع الهاتف يصير حلقةً مقفلة**»

**والقرارُ عن الدخول الجديد** — **حلقةُ «يفتح التطبيقَ فيخرج من الويب».**
**واستعادةُ كلمة المرور ليست دخولاً جديداً بل فعلَ استرداد**، **وعلّتُها
مكتوبةٌ في سطرها**: **من سرق الحساب.**

## EXISTING TEST COVERAGE
**لا اختبارَ يتحقّق من إبطال جلسةِ نوعٍ آخرَ بعد الاستعادة** — قِيس.

## VERDICT
> # **PROVEN DEFECT — D10**
> `PASSWORD RESET REVOKES ALL EXISTING SESSIONS = **NO**`
>
> **والعقدُ المكتوبُ في السطر نفسِه يقول نعم.** **ومن سرق الحسابَ من
> التطبيق لا يخرج إن استُعيدت الكلمةُ من الويب — بل يبقى أيّاماً.**

---

# R3-2 — **D-05 · R13** · إعادةُ الإدارة لكلمة المرور لا تمسّ الجلسات

## CODE PATH
`admin_users_handlers.go:246-260`:
```go
UPDATE users SET password_hash = $2, must_change_password = true …
INSERT INTO audit_log … 'admin.password_reset' …
```
**وهذا كلُّ شيء.** **قِيس**: `RevokeAllTokens` · `revokeAllSessions` ·
`AdminLogoutAll` — **لا واحدةَ منها في الدالّة.**

## ACTUAL BEHAVIOR — الأسئلةُ الستّة

| # | السؤال | الجواب |
|---|---|---|
| ٤ | **أيتابع توكنُ الوصول القديم؟** | **نعم** — `RequireAuth` يفحص `ActiveStatus` (لم يتغيّر) و`SessionRevoked` (لا مفتاحَ في Redis) |
| ٥ | **أيُصدر توكنُ التجديد القديم وصولاً جديداً؟** | **نعم** — الصفُّ لم يُبطَل، **والجلسةُ تمتدّ إلى `security.session_days`** |
| ٦ | **أيمنع `must_change_password` النداءات؟** | **لا** — **حقلٌ في الردّ لا حارس** (انظر **R3-3**) |

## INTENDED CONTRACT — **غيرُ مكتوب**
**تعليقُ الدالّة يتكلّم عن الكلمة لا عن الجلسات**:
> «مؤقتة: يُجبَر صاحب الحساب على تبديلها عند أول دخول **فلا تبقى كلمة مرور
> يعرفها غيره**»

**ولا نصَّ يقول أتُقطع الجلساتُ أم تبقى.**

## VERDICT
> # **PRODUCT DECISION REQUIRED — D-05** · **و`R13` خطرٌ قائمٌ حتّى يُحسم**
>
> **والسؤالُ للمالك**: **إعادةُ كلمة المرور من الإدارة — أهي «الموظّف نسي
> كلمتَه» أم «الحسابُ اختُرق»؟**
> **الأولى تُبقي الجلسات، والثانية توجب قطعَها.** **والشيفرةُ تفترض الأولى
> صامتةً.**
>
> **ولو كانت الثانية لَكان الفعلُ عديمَ الأثر أمنيّاً**: **من سرق الجلسةَ
> يبقى فيها، والكلمةُ التي بدّلها الأدمن لا تعنيه.**

---

# R3-3 — **D11** · `force_password_change` بلا بوّابةٍ في أندرويد

## CODE PATH
| الطبقة | ما فيها |
|---|---|
| المحرّك | `applyForcePolicy` (`service.go:158`) — **تُقنّع الحقلَ إن كان الإعدادُ مطفأً.** **ولا حارسَ في `RequireAuth`** — قِيس |
| الويب | **`web/packages/auth/src/PasswordGate.tsx:36`** — `if (loading \|\| !user?.must_change_password) return children` ✅ |
| أندرويد | **`mobile/shared/…/model/Auth.kt:44` — `data class User` بلا الحقل أصلاً** ❌ |

**وقِيس في `mobile/` كلِّه**: `must_change_password` · `mustChangePassword` —
**صفرُ نتائج.**

## INTENDED CONTRACT
`catalog.go:1596`:
> «**والقرارُ في المحرّك لا في الويب**: **خمسُ بوّاباتٍ تقرأ
> `must_change_password`** — ولو قرأ كلٌّ منها الإعدادَ بنفسه لَاختلفت
> واحدةٌ يوماً»

## ACTUAL BEHAVIOR
**بوّابةٌ واحدةٌ لا خمس.** **والخمسُ كانت لوحاتِ الويب للأدوار** —
**وقد حُذفت بقرار المالك** (الويب للأدمن والموظّفين وحدَهم)، **وانتقلت
الأدوارُ إلى أندرويد بلا الحقل.**

```
FORCE PASSWORD CHANGE
  WEB              = ENFORCED ✅
  CUSTOMER ANDROID = NOT ENFORCED ❌
  DRIVER   ANDROID = NOT ENFORCED ❌
  MERCHANT ANDROID = NOT ENFORCED ❌
  REP      ANDROID = NOT ENFORCED ❌
```

## SECURITY PATH
`security.force_password_change = true` **ثمّ يعيد الأدمن كلمةَ سائق**:
**يدخل السائقُ بالكلمة المؤقّتة ويستعمل التطبيقَ كلَّه بلا تبديل.**
**والكلمةُ تبقى معروفةً لطرفٍ ثالثٍ إلى الأبد** — **وهو بعينُه ما وُضع
الحقلُ لمنعه** (`repo.go:395`).

## VERDICT
> # **PROVEN DEFECT — D11**
> **مفتاحٌ في اللوحة لا أثرَ له على أربعةٍ من خمسة.** **والمالكُ يُشعله
> فيظنّ الحمايةَ عمّت.**
>
> **ويدخل في `Configuration Runtime Validation`** — **وهو مثالُ ما يجب أن
> يُقاس أثرُه لا أن يُقرأ اسمُه.**

---

# R3-4 — **D12** · رمزُ الدفع يبقى بعد الخروج

## CODE PATH
| | |
|---|---|
| التعريف | `mobile/ui/…/Push.kt:55` **و** `app-driver/…/push/Push.kt:39` |
| النقطة في المحرّك | **`DELETE /me/devices`** — `server.go:444` ✅ **قائمةٌ وتعمل** |
| **المُنادون** | **صفر** — قِيس: `grep -rn 'Push.unregister' mobile/` **لا نتيجة** |

**والتسجيلُ يُنادى**: `AppFrame.kt:291` و`driver/data/Backend.kt:106`.
**فالبابُ يُفتح ولا يُغلق.**

## INTENDED CONTRACT
**تعليقُ الدالّة نفسِه**:
> «**يُلغى عند الخروج — وإلّا وصلت أخبار حساب خرج إلى جهازه**»

## ACTUAL BEHAVIOR — الخروج
`AuthViewModel.logout()` (`:542`): يقرأ التجديد · **يمسح الجلسةَ محليّاً** ·
`backend.auth.logout(refresh)`. **ولا `Push.unregister`.**
**وسائقُ `HomeViewModel.logout` مثلُه.**

## SUCCESS PATH
```
دخول  → device_tokens (token → User A)
خروج  → users/sessions تُبطل · device_tokens لم تُمسّ
النتيجة: الصفّ باقٍ لـUser A
```

**ولا يزول إلّا بثلاثة**: **تسجيلُ الرمز نفسِه لحسابٍ آخر** ·
**تقادمٌ بعد ٦٠ يوماً** (`push/diagnose.go:58`) · **رفضُ FCM نهائيّاً**
(`push.go:231`).

## SECURITY / PRIVACY PATH
**إشعارُ `user:` يحمل نصّاً وكياناً لا إشارةً** — **فيظهر على جهازٍ واقفٍ
على شاشة الدخول**: «طلبُك في الطريق» · «وصلتك حوالة» · اسمُ متجرٍ ومبلغ.
**وعلى جهازٍ مشترَكٍ أو مُعارٍ يقرؤها من ليس صاحبَها.**

## التطبيقاتُ المتأثّرة — **الأربعةُ كلُّها**
| التطبيق | الطريق |
|---|---|
| customer · merchant · rep | **`AuthViewModel` المشترك** |
| driver | **`HomeViewModel.logout` — ونسختُه الخاصّةُ من `Push` معطّلةٌ كذلك** |

## VERDICT
> # **PROVEN DEFECT — D12**
> **دالّةٌ كُتبت لغرضٍ وبقيت بلا منادٍ** — **والنقطةُ في المحرّك جاهزة.**
> **والعقدُ مكتوبٌ فوقها حرفاً.**

---

# R3-5 — **D13** · سردُ مجلّد الوسائط — **وإثباتُ التسليم فيه**

## CODE PATH
`server.go:234`:
```go
r.Handle("/media/*", http.StripPrefix("/media/", s.media.FileServer()))
```
**خارجَ `/api/v1` وخارجَ `RequireAuth`.**

`media.go:490`:
```go
fs := http.FileServer(http.Dir(s.dir))
if strings.Contains(r.URL.Path, "..") { http.NotFound(...) ; return }
w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
```
**والحارسُ الوحيدُ `..`** — **ولا شيءَ يمنع سردَ المجلّدات.**

## القياس — **بمحاكاةٍ خارجَ المستودع، بلا مساسٍ بالشيفرة**
**نُسخت الدالّةُ حرفيّاً إلى برنامجٍ في مجلّدٍ مؤقّتٍ بشجرةٍ مطابقة:**

```
GET /media/                      200  cache="public, max-age=31536000, immutable"
    <pre><a href="2026/">2026/</a></pre>
GET /media/2026/                 200  → <a href="09/">09/</a>
GET /media/2026/09/              200  → <a href="aaaa-bbbb.jpg">aaaa-bbbb.jpg</a>
GET /media/2026/09/aaaa-bbbb.jpg 200  → الملفّ
```

## الأجوبةُ الخمسة
```
1. DIRECTORY LISTING ENABLED            = YES
2. delivery_proof تحت الجذر نفسِه       = YES   (media.go:123 · :304 — yyyy/mm)
3. /media/* بلا مصادقة                  = YES
4. طبقةُ Caddy تمنع السرد               = NO    (grep 'media' في Caddyfile = صفر ·
                                                 و api.rahalgo.com يمرّر كلَّ شيء إلى api:8080)
5. Cache-Control عامٌّ سنةً على الإثباتات = YES
```

## SECURITY / PRIVACY PATH
**نعم** — **زائرٌ بلا حساب**:
1. `GET /media/` → السنوات
2. `GET /media/2026/09/` → **أسماءُ كلّ ملفّات الشهر**
3. **ينزّل ما شاء** — **ومنها صورُ إثبات التسليم.**

**و«المعرّفُ العشوائيُّ حمايةٌ» تسقط كلَّها** — `media.go:7`:
> «أسماء الملفات uuid عشوائية داخل مجلدات yyyy/mm»

**وهي حمايةٌ بالغموض** — **والخادمُ نفسُه يسرد الغموض.**

**وصورةُ إثبات التسليم بابُ زبونٍ وعنوانُه** — **ووسيطٌ عامٌّ مخبّأٌ سنةً.**

## EXISTING TEST COVERAGE
`internal/media/*_test.go` — **تفحص الحفظَ والأنواعَ والمقاسات.**
**ولا اختبارَ للسرد ولا للنفاذ العامّ** — قِيس.

## VERDICT
> # **PROVEN SECURITY / PRIVACY DEFECT — D13**
> **وأخطرُ ما في هذه الدفعة**: **لا يحتاج حساباً ولا عطلاً ولا تزامناً** —
> **نداءُ `curl` واحد.**

---

# R3-6 — **D14 · R14** · الويب سوكت

## CODE PATH — المصافحة
`ws.go:53-62`:
```go
claims, err := s.tokens.VerifyAccess(r.URL.Query().Get("token"))   ✅
if s.identity.SessionRevoked(r.Context(), claims.SID) { … }        ✅
```
**ولا `ActiveStatus`** — **وهو ما يفحصه `RequireAuth`** (`middleware.go:40`).

## CODE PATH — بعد المصافحة
`ws.go:150-172` — **حلقةٌ فيها ثلاثةٌ لا رابع**: `ctx.Done()` ·
`readDone` · نبضةُ ٣٠ ثانية · رسالة. **ولا إعادةَ تحقّقٍ من شيء.**

## الحالاتُ الثلاث

### A — موقوفٌ يفتح اتّصالاً جديداً — **عيب**
`AdminUpdateUser` (`admin.go:158`):
```go
if in.Status != nil && *in.Status != "active" {
    _, _ = s.repo.RevokeAllTokens(ctx, userID)   ← القاعدةُ وحدَها
}
s.invalidateStatusCache(ctx, userID)
```
**ولا مفتاحَ `sessionRevoked` في Redis** — **فـ`SessionRevoked` ترجع `false`.**

**والنتيجة**: `RequireAuth` **يمنعه** بـ`ActiveStatus` ✅، **و`handleWS`
لا يسأل عنها** ⇒ **يمرّ** ❌.

**فيبقى الموقوفُ مشترِكاً في**: `user:` (نصٌّ وكيان) · `customer:` ·
`driver:` + `queue` · `merchant:` · `sales:` · `ops` إن كان موظّفاً ·
`catalog` — **إلى أن ينتهي توكنُه (١٥ دقيقة) ثمّ يبقى الاتّصالُ مفتوحاً بلا حدّ.**

### B — اتّصالٌ قائمٌ ثمّ `logout-all` — **خطر**
`AdminLogoutAll` **يُبطل ويضع المفاتيح** ✅ — **ولا أحدَ يقرؤها في الحلقة.**
**فالقناةُ تبقى تبثّ.**

### C — انتهاءُ توكن الوصول — **خطر**
**لا مهلةَ للاتّصال ولا إعادةَ مصادقة** — **يعيش ما دام العميلُ يردّ النبضة.**

## VERDICT
> # **PROVEN DEFECT — D14** · **اتّصالٌ جديدٌ بعد الإيقاف**
> **`handleWS` يخالف `RequireAuth` في حارسٍ واحد** — **والفرقُ بينهما هو
> الثغرة.** **وموقوفٌ يُمنع من كلّ نداءٍ ويُسمح له بقناة البثّ.**
>
> # **PROVEN RISK — R14** · **اتّصالٌ قائمٌ لا يُراجَع**
> **مصادقةٌ عند الباب لا في الغرفة.** **وهو نمطٌ شائعٌ في الويب سوكت
> ولا نصَّ يقول إنّه مقصود** — **والعقدُ المكتوب يقول «فوراً».**

---

# R3-7 — **R15** · سحبُ الدور لا يسري فوراً

## CODE PATH
`AdminRevokeRole` (`admin.go:194`):
```go
s.repo.RevokeRole(ctx, userID, role)
s.repo.Audit(...)
```
**وهذا كلُّ شيء** — **لا إبطالَ جلسةٍ ولا مفتاحَ Redis ولا إصدارَ توكن.**

**والأدوارُ في التوكن**: `auth/token.go:34` `IssueAccess(userID, roles, sid)`
→ `middleware.go` `ctxRoles = claims.Roles` → `RequireRoles` **تقرأ
الادّعاءَ لا القاعدة.**

## ACTUAL BEHAVIOR
| الطريق | المدّة |
|---|---|
| **النداءاتُ العاديّة** | **حتّى انتهاء توكن الوصول — ١٥ دقيقة** (`cmd/api/main.go:71`) |
| **الويب سوكت** | **بلا حدّ** — المواضيعُ حُسبت مرّةً عند المصافحة (**R14**) |

**فمن سُحب منه `finance` يبقى يقرأ `ops` ويكتب في المال ربعَ ساعة**،
**واتّصالُه المفتوحُ يبقى على `ops` إلى أن يُغلقه هو.**

## VERDICT
> # **PROVEN RISK — R15**
> **والشطرُ الأوّل نمطٌ معروفٌ ومحدود** (JWT بمهلةٍ قصيرة) — **ولا يُسمّى
> عيباً وحدَه.**
> **والشطرُ الثاني بلا حدّ** — **وهو ما يجعله خطراً لا سلوكاً.**
>
> **والوسيلةُ قائمةٌ في البيت**: `AdminLogoutAll` تُبطل وتضع المفاتيح —
> **ولا تُنادى عند سحب الدور.**

---

# R3-8 — **D15** · إنشاءُ حسابِ موظّفٍ في خطوتين

## CODE PATH
```
AdminCreateUser                      identity/admin.go:54
  → repo.CreateUserWithRole          :81   ← Begin … Commit داخلها (repo.go:65-91)
      → EnsureInviteCode (sales)            ← بعد الـCommit
  → repo.GrantRole  (الأدوارُ الباقية) :89   ← كتابةٌ مستقلّة
  → auth.HashPassword                :94
  → repo.SetTempPassword             :101  ← كتابةٌ مستقلّة
```

## INTENDED CONTRACT
`admin.go:77` — **في السطر نفسِه**:
> «**إلزامية — لا حساب موظف بلا كلمة مرور**»

## FAILURE WINDOWS
| # | تسقط عند | ما يبقى |
|---|---|---|
| **F1** | `GrantRole` للدور الثاني | **حسابٌ بدورٍ واحدٍ بلا كلمة** |
| **F2** | `HashPassword` (نادر) | **حسابٌ بأدوارٍ بلا كلمة** |
| **F3** | **`SetTempPassword`** — خطأُ قاعدةٍ **أو إلغاءُ سياق الطلب** | **حسابٌ كاملُ الأدوار بلا `password_hash`** |
| **F4** | `EnsureInviteCode` للمندوب | **مندوبٌ بلا رمز دعوة** |

**والردُّ خطأٌ في الأربعة** — **فالأدمن يظنّ أنّ شيئاً لم يقع.**

## الأجوبةُ الثلاثة
```
CAN ADMIN CREATE FAIL WHILE USER REMAINS CREATED = YES
CAN CREATED USER HAVE REQUIRED ROLE BUT NO PASSWORD = YES
CAN RETRY RECOVER AUTOMATICALLY = NO   ← الإعادةُ تردّ ErrPhoneTaken (admin.go:83)
```

**والاستدراكُ يدويٌّ وممكن**: `POST /admin/users/{id}/password` على الحساب
القائم. **ولكنّ الأدمن لا يعلم أنّه قائمٌ أصلاً** — **رأى فشلاً.**

## ⚠️ وحسابٌ بلا كلمةٍ ليس حساباً ميّتاً
**`VerifyOTP`** (`service.go:278`) **تُصدر جلسةً لأيّ رقمٍ يملك رمزاً** —
**ولا تشترط `password_hash`.** **فمن بيده الهاتفُ يدخل الحسابَ الناقصَ
بأدواره كاملة** إن كان `auth.otp_login` مشتعلاً.

## VERDICT
> # **PROVEN DEFECT — D15**
> **والثابتُ مكتوبٌ في السطر الذي يفحص الطول** — **ويُخرَق بعده بعشرين سطراً.**
> **وهو `D2` نفسُه في بابٍ آخر** (`convertLead` بلا معاملة).

---

# R3-9 — **R16** · الإبطالُ يفشل مفتوحاً

## CODE PATH
```go
// revokeSession :762 · revokeClientSessions :781 · revokeAllSessions :795
s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
```
**ولا نتيجةَ تُفحص في المواضع الثلاثة.**

```go
// SessionRevoked :745
n, err := s.rdb.Exists(ctx, sessionRevokedKey(sid)).Result()
return err == nil && n > 0        ← خطأٌ ⇒ «غيرُ مُبطَلة»
```

## INTENDED CONTRACT
`service.go:769`:
> «**وRedis شرطٌ لا زينة**: القاعدةُ تُبطل توكنَ التجديد، **وتوكنُ الوصول
> القائمُ يبقى صالحاً حتّى تنتهي مهلتُه** — والوسيطُ يسأل Redis في كلّ طلب»

**و`AdminLogoutAll` (`:213`)**: «**يُبطل كل جلسات الحساب فوراً**».

## FAILURE PATH — **Redis ساقطةٌ أو بطيئة**
| الفعل | الردّ | الحقيقة |
|---|---|---|
| `POST /auth/logout` | **٢٠٠ «تمّ الخروج»** | **توكنُ الوصول صالحٌ ١٥ دقيقة** |
| `POST /admin/users/{id}/logout-all` | **٢٠٠ مع عدد الجلسات** | كسابقه |
| دخولٌ جديدٌ يُبطل سابقَه من النوع نفسِه | **٢٠٠** | **الجلستان تعملان معاً** — **وهو ما يمنع مشاركةَ الحسابات** |

**والقاعدةُ تُبطل توكنَ التجديد دائماً** ✅ — **فالنافذةُ محدودةٌ بـ١٥ دقيقة
لا مفتوحة.**

## VERDICT
> # **PROVEN RISK — R16**
> **وليس عيباً**: **يلزمه عطلُ بنيةٍ تحتيّة، والنافذةُ محدودة، والتجديدُ مقطوع.**
>
> **وهو خطرٌ لأنّ الردَّ يقول «فوراً» والفعلَ ليس فوريّاً** — **ومن أوقف
> موظّفاً مسيئاً وقرأ «تمّ» لا يعلم أنّ له ربعَ ساعة.**

---

# الحصيلةُ النهائيّة

| | كان | صار |
|---|---|---|
| `PROVEN DEFECTS` | ٩ | **١٥** — +D10 · D11 · D12 · D13 · D14 · D15 |
| `PROVEN RISKS` | ١٢ | **١٦** — +R13 · R14 · R15 · R16 |
| `PRODUCT DECISIONS REQUIRED` | ٥ | **٦** — +D-05 |
| `RUNTIME VALIDATION REQUIREMENTS` | ٦ | **٧** — **+RV-7**: **أمسرودٌ `/media/` على الإنتاج فعلاً؟** |
| `CODE TRUTH BASELINE` | `26f93c5d` | **`26f93c5d`** ✅ |

**وثلاثةٌ من الستّة عيوبٌ في العقد لا في المنطق**: **دالّةٌ بلا منادٍ**
(D12) · **بوّابةٌ حُذف نظراؤها** (D11) · **حارسٌ في مسارٍ وليس في أخيه** (D14).

**ولا تغييرَ في شيفرةٍ تشغيليّة.**
