# APPLICATION_INVENTORY — تطبيق الزبون · رحّال غو

> **جردٌ مقيسٌ من الشيفرة والحزمة، لا مكتوبٌ من ذاكرة.**
> تاريخ التدقيق: ٢٠٢٦-٠٨-١٩ · الالتزام المفحوص: `16dbf9be`
> الحزمة المفحوصة: `app-customer-release.apk` و`.aab` (نسخة إصدار موقّعة)

---

## ١ · المعمارية

**تطبيقٌ واحد (single-activity) بـJetpack Compose، بلا `Navigation-Compose`.**
التنقّل حالةٌ في `MainActivity`: `Tab` (خمسة) × `Overlay` (خمسة).

```
:app-customer  4٬509 سطراً / 17 ملفّاً   ← شاشات الزبون
:ui            9٬691 سطراً / 48 ملفّاً   ← العدّة المشتركة (الأكبر)
:shared        3٬255 سطراً / 20 ملفّاً   ← عقود الشبكة والنماذج
:design        1٬039 سطراً /  5 ملفّات   ← اللوح والسمة والافتتاح
:map             491 سطراً /  3 ملفّات   ← MapLibre
```

**الاعتماد:** `app-customer → map → ui → design`، و`ui → shared`.

**ملاحظة معمارية:** `:ui` تحمل ثلث الشيفرة وفيها شاشاتُ الحساب والدخول
والمحفظة — **مشتركةٌ بين التطبيقات الثلاثة**، وهو مقصود.

---

## ٢ · الشاشات والتنقّل

| # | الشاشة | الملفّ | النموذج |
|---|---|---|---|
| 1 | التسوّق (سوق) | `shop/ShopScreen.kt` | `ShopViewModel` |
| 2 | السلّة | `cart/CartScreen.kt` | `CartViewModel` |
| 3 | طلباتي | `orders/OrdersScreen.kt` | `OrdersViewModel` |
| 4 | الطلب الخاصّ | `custom/CustomScreen.kt` | `CustomViewModel` |
| 5 | حسابي | `ui/AccountScreen.kt` | `AccountViewModel` |
| 6 | المحفظة | `ui/WalletScreen.kt` | `WalletViewModel` |
| 7 | الدخول | `ui/AuthScreen.kt` | `AuthViewModel` |
| 8 | إنشاء حساب | `ui/SignupScreen.kt` | `AuthViewModel` |
| 9 | استعادة كلمة المرور | `ui/ResetScreen.kt` | `AuthViewModel` |
| 10 | المفضّلة/العروض | `mine/MineScreens.kt` | `MineViewModel` |
| 11 | سجلّ الطلبات | `orders/OrdersScreen.kt` | `OrdersViewModel` |
| 12 | الدردشات | `ui/Chats.kt` | `ChatsViewModel` |
| 13 | صفحات المنصّة | `ui/PlatformPage.kt` | `PagesViewModel` |
| 14 | صندوق الإشعارات | `ui/InboxSheet.kt` | `ShellViewModel` |
| 15 | التقاط الموقع | `map/PickPoint.kt` | `PickPointViewModel` |
| 16 | محرّر العنوان | `ui/Address.kt` | `AccountViewModel` |
| 17 | لوح محادثة الطلب | `ui/OrderChat.kt` | `OrderChatViewModel` |

**التنقّل:** `Tab { Shop, Cart, Orders, Custom, Account }` +
`Overlay { None, Menu(key), Wallet, Inbox, Account, Rating }`.
`BackHandler` في أربعة مواضع.

---

## ٣ · الشبكة

| البند | القيمة |
|---|---|
| العميل | **Ktor 3 + OkHttp** |
| الترميز | `kotlinx.serialization` · `ignoreUnknownKeys = true` |
| الغلاف | `{"data": …}` / `{"error": {code, message_key}}` |
| الأساس | `https://api.rahalgo.com` (ثابت في `Backend.kt`) |
| التجديد | `raw()` تُعيد المحاولة مرّةً بعد ٤٠١ بـ`refresh()` |
| البثّ الحيّ | WebSocket — `LiveSocket` بإعادة وصلٍ متضاعفة |

**نقاط الـAPI التي يناديها التطبيق: ٢٢** (مقيسة من `shared/`).

---

## ٤ · الجلسة والتخزين

| ما هو | أين | الحال |
|---|---|---|
| التوكن | `EncryptedSharedPreferences` (AES256-GCM + MasterKey) | **مشفَّر ✔** |
| السلّة | `SharedPreferences` عاديّة (`rahalgo_cart`) | نصّ — ولا سرَّ فيها |
| رمز الدعوة | `SharedPreferences` (`rahalgo_invite`) | نصّ |
| السمة | `SharedPreferences` | نصّ |
| **لا قاعدة بيانات محلّيّة** | — | لا Room ولا SQLite |

---

## ٥ · الأذونات (من الحزمة)

```
INTERNET · ACCESS_NETWORK_STATE · ACCESS_WIFI_STATE
ACCESS_COARSE_LOCATION · ACCESS_FINE_LOCATION
POST_NOTIFICATIONS
BIND_GET_INSTALL_REFERRER_SERVICE (بلاي)
```

**ولا `ACCESS_BACKGROUND_LOCATION`** — يوفّر أصعب مراجعة في بلاي.

---

## ٦ · مكتبات الطرف الثالث (٢٦)

Compose BOM · Ktor (5) · kotlinx-serialization · Coil 3 (2) ·
MapLibre 13.4.1 · Firebase (messaging + crashlytics) ·
play-services-location · install-referrer · security-crypto ·
splashscreen · lifecycle (2) · coroutines-play-services

---

## ٧ · الإعداد والبناء

| البند | القيمة |
|---|---|
| `minSdk` / `targetSdk` / `compileSdk` | 26 / **37** / 37 |
| `versionCode` / `versionName` | **2** / `1.0.0` |
| تصغير R8 | **مفعَّل** في الإصدار |
| قواعد proguard | `SourceFile,LineNumberTable` + `keepnames Throwable` |
| التوقيع | مفتاح رفع خاصّ (`keystore.properties` خارج المستودع) |
| نسخة تجريبيّة | `applicationIdSuffix=".debug"` + اسم مميّز |
| **AAB** | ‏**٢٣٫٩ ميغا** (APK ‏٥٠٫١) |
| المعماريّات | arm64-v8a · armeabi-v7a · x86 · x86_64 |
| **بيئات** | **واحدة فقط** — لا `staging`، العنوان ثابت في الشيفرة |

---

## ٨ · ما لا يحويه التطبيق

- **لا تحليلات** (Analytics) — Crashlytics فقط
- **لا إعلانات** ولا معرّف إعلانيّ
- **لا WebView**
- **لا مزوّد محتوى** (ContentProvider) خاصّ بنا
- **لا خدمة أماميّة** (السائق وحده يملكها)
- **لا اختبارات آليّة للتطبيق** — لا وحدة ولا واجهة
