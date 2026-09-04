# رحّال غو — طفراتُ الويب ١٣١/١٣١

> **مُولَّدةٌ آليّاً** من `web/apps/rahalgo/src` و`web/packages`، ومُثراةٌ
> بقراءة خمسٍ وعشرين سطراً قبل كلّ نداءٍ وخمسةَ عشرَ بعده.
>
> **CODE TRUTH BASELINE**: `26f93c5d`

---

## الرايات الأربع `U L E S`

| الحرف | المعنى | العدد |
|---|---|---|
| **`U`** | **يُطلقها فعلُ مستخدمٍ ظاهر** (`onClick` · `onSubmit` · `onChange` · `onConfirm` · `onSave` · `onToggle`) | **٥٦** |
| **`L`** | **حالُ تحميلٍ** (`busy` · `saving` · `pending`) | **٨٠** |
| **`E`** | **معالجةُ خطأٍ ظاهرة** (`setErr` · `catch` · `errorText`) | **١٠٩** |
| **`S`** | **أثرُ نجاحٍ** (رسالة · إعادةُ جلب · تنقّل) | **٧٥** |

## القراءةُ الصحيحةُ لهذه الأرقام

**`U` = ٥٦ من ١٣١ لا تعني أنّ ٧٥ طفرةً خفيّة.** ثلاثةُ أسبابٍ مقيسة:

1. **الطفرةُ تعيش في دالّةٍ مسمّاة** (`save` · `submit` · `remove`)
   **والزرُّ يناديها من موضعٍ أبعدَ من نافذة القراءة** — فهي ظاهرةٌ
   ومُطلَقةٌ بيدِ مستخدم.
2. **مُطلِقاتٌ داخليّةٌ حقيقيّة**: `packages/auth/client.ts` (١٤ طفرة)
   — **تجديدُ التوكن يقع بلا ضغطة** ✅ · و`useLocationBeacon`.
3. **نماذجُ تُرسَل بـ`<form onSubmit>`** خارجَ النافذة.

**فالتصنيفُ الصادق**:

| الصنف | العدد | الدليل |
|---|---|---|
| **ظاهرةٌ للمستخدمِ بيقين** | **٥٦** | راية `U` |
| **داخليّةٌ بيقين** | **~١٦** | `auth/client.ts` ١٤ · `useLocationBeacon` ١ · `routing.ts` ١ |
| **ظاهرةٌ عبر دالّةٍ وسيطة** | **~٥٩** | لها اسمُ دالّةٍ ومعالجةُ خطأٍ وأثرُ نجاح |

**و`E` = ١٠٩ من ١٣١ أهمُّ رقمٍ هنا**: **ثلاثةٌ وثمانون في المئة من
طفرات اللوحة تعالج خطأها.** والاثنتان والعشرون الباقيةُ أكثرُها
في `client.ts` (حيث المعالجةُ في طبقةٍ أعلى) وفي رفع الملفّات.

---

## الجدول

| السطح | الملفّ:السطر | الدالّة | الفعل | المسار | `U L E S` |
|---|---|---|---|---|---|
| لوحة: cash | `page.tsx:217` | `submit` | `POST` | `/api/v1/admin/drivers/{v}/settle` | `·LES` |
| لوحة: emergencies | `page.tsx:224` | `submit` | `POST` | `/api/v1/admin/emergencies/{v}/resolve` | `·LE·` |
| لوحة: expenses | `page.tsx:236` | `ExpensesPage` | `POST` | `/api/v1/admin/expenses/{v}/void` | `U·ES` |
| لوحة: expenses | `page.tsx:303` | `submit` | `POST` | `/api/v1/admin/expenses` | `·LES` |
| لوحة: expenses | `page.tsx:395` | `save` | `POST` | `/api/v1/admin/expenses/categories` | `ULES` |
| لوحة: incentives | `page.tsx:105` | `submit` | `POST` | `/api/v1/admin/users/{v}/incentive` | `·LES` |
| لوحة: leads | `page.tsx:136` | `setStatus` | `POST` | `/api/v1/admin/leads/{v}/status` | `·LES` |
| لوحة: merchants | `page.tsx:109` | `MerchantProfilePage` | `PATCH` | `/api/v1/admin/merchants/{v}` | `·LES` |
| لوحة: merchants | `page.tsx:243` | `MerchantProfilePage` | `PATCH` | `/api/v1/admin/merchants/{v}/hours` | `U··S` |
| لوحة: payouts | `page.tsx:320` | `submit` | `POST` | `/api/v1/admin/payouts/{v}/decide` | `ULE·` |
| لوحة: promos | `page.tsx:130` | `toggleActive` | `PATCH` | `/api/v1/admin/promos/{v}` | `··ES` |
| لوحة: promos | `page.tsx:267` | `submit` | `POST` | `/api/v1/admin/promos` | `ULE·` |
| لوحة: sections | `page.tsx:181` | `toggle` | `PATCH` | `/api/v1/admin/menu/items/{v}` | `··ES` |
| لوحة: sections | `page.tsx:501` | `submit` | `POST` | `/api/v1/admin/merchants/{v}/menu/items` | `ULE·` |
| لوحة: sections | `page.tsx:617` | `submit` | `PATCH` | `/api/v1/admin/menu/items/{v}` | `ULE·` |
| لوحة: sections | `page.tsx:78` | `toggle` | `PATCH` | `/api/v1/admin/sections/{v}` | `U··S` |
| لوحة: sections | `page.tsx:235` | `submit` | `PATCH` | `/api/v1/admin/sections/{v}` | `ULE·` |
| لوحة: sections | `page.tsx:237` | `submit` | `POST` | `/api/v1/admin/sections/{v}` | `ULE·` |
| لوحة: settings | `page.tsx:547` | `save` | `PUT` | `/api/v1/admin/settings/{v}` | `ULE·` |
| لوحة: users | `page.tsx:265` | `setStatus` | `PATCH` | `/api/v1/admin/users/{v}` | `··ES` |
| لوحة: users | `page.tsx:596` | `has` | `POST` | `/api/v1/admin/users/{v}/logout-all` | `U··S` |
| لوحة: users | `page.tsx:1395` | `submit` | `POST` | `/api/v1/admin/users/{v}/password` | `ULES` |
| لوحة: users | `page.tsx:1457` | `submit` | `PATCH` | `/api/v1/admin/users/{v}` | `ULE·` |
| لوحة: users | `page.tsx:1528` | `NotesEditor` | `PATCH` | `/api/v1/admin/users/{v}` | `UL·S` |
| مكوّن | `PasswordGate.tsx:46` | `submit` | `POST` | `/api/v1/auth/password` | `·LE·` |
| مكوّن | `client.ts:132` | `refreshTokens` | `POST` | `/api/v1/auth/refresh` | `··ES` |
| مكوّن | `client.ts:215` | `call` | `POST` | `/api/v1/auth/login` | `··ES` |
| مكوّن | `client.ts:220` | `call` | `POST` | `/api/v1/auth/login` | `··E·` |
| مكوّن | `client.ts:225` | `call` | `POST` | `/api/v1/auth/otp/request` | `··E·` |
| مكوّن | `client.ts:231` | `call` | `POST` | `/api/v1/auth/password/reset/request` | `··E·` |
| مكوّن | `client.ts:236` | `call` | `POST` | `/api/v1/auth/password/reset/request` | `····` |
| مكوّن | `client.ts:242` | `call` | `POST` | `/api/v1/auth/password/reset/verify` | `····` |
| مكوّن | `client.ts:257` | `call` | `POST` | `/api/v1/auth/pin` | `····` |
| مكوّن | `client.ts:263` | `call` | `POST` | `/api/v1/auth/pin/setup` | `····` |
| مكوّن | `client.ts:270` | `call` | `POST` | `/api/v1/auth/signup/request` | `····` |
| مكوّن | `client.ts:280` | `call` | `POST` | `/api/v1/auth/signup/verify` | `····` |
| مكوّن | `client.ts:292` | `call` | `POST` | `/api/v1/auth/signup/confirm` | `···S` |
| مكوّن | `client.ts:297` | `call` | `POST` | `/api/v1/auth/signup/confirm` | `··ES` |
| مكوّن | `client.ts:306` | `call` | `POST` | `/api/v1/auth/me` | `··ES` |
| مكوّن | `routing.ts:122` | `goTo` | `POST` | `/api/v1/auth/handoff` | `····` |
| مكوّن | `AccountSettings.tsx:180` | `onUpload` | `POST` | `/api/v1/me/avatar` | `··E·` |
| مكوّن | `AccountSettings.tsx:194` | `onRemovePhoto` | `DELETE` | `/api/v1/me/avatar` | `·LE·` |
| مكوّن | `AccountSettings.tsx:211` | `onPassword` | `POST` | `/api/v1/auth/password` | `·LE·` |
| مكوّن | `AccountSettings.tsx:231` | `reqPhone` | `POST` | `/api/v1/auth/phone/request` | `·LE·` |
| مكوّن | `AccountSettings.tsx:247` | `confirmPhone` | `POST` | `/api/v1/auth/phone/confirm` | `··E·` |
| مكوّن | `AccountSettings.tsx:268` | `reqWhatsApp` | `POST` | `/api/v1/auth/whatsapp/request` | `··E·` |
| مكوّن | `AccountSettings.tsx:283` | `confirmWhatsApp` | `POST` | `/api/v1/auth/whatsapp/confirm` | `··E·` |
| مكوّن | `AccountSettings.tsx:304` | `reqDelete` | `POST` | `/api/v1/auth/account/delete/request` | `··E·` |
| مكوّن | `AccountSettings.tsx:319` | `confirmDelete` | `POST` | `/api/v1/auth/account/delete/confirm` | `··E·` |
| مكوّن | `AccountSettings.tsx:341` | `onName` | `PATCH` | `/api/v1/me/name` | `··E·` |
| مكوّن | `AddressBook.tsx:96` | `save` | `POST` | `/api/v1/my/addresses` | `·LES` |
| مكوّن | `AddressBook.tsx:173` | `save` | `POST` | `/api/v1/my/addresses/{v}/default` | `U··S` |
| مكوّن | `AddressBook.tsx:184` | `save` | `DELETE` | `/api/v1/my/addresses/{v}` | `U··S` |
| مكوّن | `DashboardChrome.tsx:185` | `shopAsCustomer` | `POST` | `/api/v1/auth/handoff` | `··E·` |
| مكوّن | `Favorites.tsx:102` | `useFavorites` | `POST` | `/api/v1/my/favorites/{v}` | `··ES` |
| مكوّن | `FileUpload.tsx:56` | `upload` | `POST` | `—` | `ULE·` |
| مكوّن | `FileUpload.tsx:70` | `remove` | `DELETE` | `—` | `ULE·` |
| مكوّن | `ImageUpload.tsx:93` | `upload` | `POST` | `—` | `ULE·` |
| مكوّن | `MenuManager.tsx:205` | `toggleAvailable` | `PATCH` | `—` | `··ES` |
| مكوّن | `MenuManager.tsx:228` | `deleteItem` | `DELETE` | `—` | `·LES` |
| مكوّن | `MenuManager.tsx:567` | `submit` | `PATCH` | `—` | `ULE·` |
| مكوّن | `MenuManager.tsx:569` | `submit` | `POST` | `—` | `ULE·` |
| مكوّن | `Notifications.tsx:293` | `connect` | `POST` | `/api/v1/me/notifications/read` | `··ES` |
| مكوّن | `NotificationsPage.tsx:103` | `NotificationsPage` | `POST` | `/api/v1/me/notifications/read` | `···S` |
| مكوّن | `NotificationsPage.tsx:110` | `NotificationsPage` | `POST` | `/api/v1/me/notifications/read` | `···S` |
| مكوّن | `OrderChat.tsx:142` | `submit` | `POST` | `/api/v1/orders/{v}/messages` | `·LES` |
| مكوّن | `StoreHours.tsx:87` | `save` | `PUT` | `—` | `ULE·` |
| مكوّن | `WalletPage.tsx:567` | `submit` | `POST` | `/api/v1/me/payouts` | `ULES` |
| مكوّن | `WhatsAppVerify.tsx:55` | `request` | `POST` | `/api/v1/auth/whatsapp/request` | `·LE·` |
| مكوّن | `WhatsAppVerify.tsx:84` | `confirm` | `POST` | `/api/v1/auth/whatsapp/confirm` | `ULE·` |
| مكوّن | `useLocationBeacon.ts:50` | `send` | `POST` | `/api/v1/driver/location` | `··E·` |
| مكوّن إداريّ | `BroadcastPanel.tsx:73` | `send` | `POST` | `/api/v1/admin/broadcast` | `·LE·` |
| مكوّن إداريّ | `DiscountsTab.tsx:112` | `submit` | `POST` | `/api/v1/admin/offers` | `·LE·` |
| مكوّن إداريّ | `DiscountsTab.tsx:139` | `toggle` | `POST` | `/api/v1/admin/offers/{v}/active` | `ULES` |
| مكوّن إداريّ | `HoursModal.tsx:70` | `save` | `PUT` | `/api/v1/admin/merchants/{v}/hours` | `ULES` |
| مكوّن إداريّ | `HoursModal.tsx:75` | `save` | `PATCH` | `/api/v1/admin/merchants/{v}/hours` | `ULES` |
| مكوّن إداريّ | `ManageRolesModal.tsx:79` | `apply` | `DELETE` | `/api/v1/admin/users/{v}/roles/{v}?reason={v}` | `ULES` |
| مكوّن إداريّ | `ManageRolesModal.tsx:84` | `apply` | `POST` | `/api/v1/admin/users/{v}/roles/{v}?reason={v}` | `ULES` |
| مكوّن إداريّ | `MenuReviewQueue.tsx:85` | `review` | `POST` | `/api/v1/admin/menu/items/{v}/review` | `·LES` |
| مكوّن إداريّ | `MerchantModal.tsx:254` | `submit` | `PATCH` | `/api/v1/admin/merchants/{v}` | `ULE·` |
| مكوّن إداريّ | `MerchantModal.tsx:258` | `submit` | `POST` | `/api/v1/admin/merchants/{v}` | `ULE·` |
| مكوّن إداريّ | `MerchantModal.tsx:583` | `addCategory` | `POST` | `/api/v1/admin/categories` | `U·ES` |
| مكوّن إداريّ | `MerchantModal.tsx:599` | `toggleActive` | `PATCH` | `/api/v1/admin/categories/{v}` | `U·ES` |
| مكوّن إداريّ | `PinSection.tsx:110` | `change` | `POST` | `/api/v1/pin/change` | `·LES` |
| مكوّن إداريّ | `PinSection.tsx:128` | `askCode` | `POST` | `/api/v1/pin/reset/request` | `·LES` |
| مكوّن إداريّ | `PinSection.tsx:147` | `confirmReset` | `POST` | `/api/v1/pin/reset/confirm` | `·LES` |
| مكوّن إداريّ | `ProfileRoleTabs.tsx:538` | `settle` | `POST` | `/api/v1/admin/drivers/{v}/settle` | `·LES` |
| مكوّن إداريّ | `ProfileRoleTabs.tsx:564` | `endShift` | `POST` | `/api/v1/admin/drivers/{v}/end-shift` | `·L·S` |
| مكوّن إداريّ | `StoreActions.tsx:111` | `call` | `POST` | `—` | `U···` |
| مكوّن إداريّ | `StoreActions.tsx:121` | `call` | `PATCH` | `—` | `U···` |
| مكوّن إداريّ | `StoreActions.tsx:135` | `call` | `POST` | `—` | `U··S` |
| مكوّن إداريّ | `ViolationsModal.tsx:100` | `issue` | `POST` | `/api/v1/admin/merchants/{v}/warnings` | `ULES` |
| مكوّن إداريّ | `WalletModal.tsx:64` | `apply` | `POST` | `/api/v1/admin/users/{v}/wallet` | `·LES` |
| مكوّن إداريّ | `WarningsSection.tsx:104` | `issue` | `POST` | `/api/v1/admin/users/{v}/warnings` | `·LES` |
| مكوّن إداريّ | `all.tsx:147` | `setStatus` | `PATCH` | `/api/v1/admin/users/{v}` | `··ES` |
| مكوّن إداريّ | `all.tsx:613` | `submit` | `POST` | `/api/v1/admin/users` | `·LE·` |
| مكوّن إداريّ | `disputes.tsx:289` | `submit` | `POST` | `/api/v1/admin/disputes/{v}/settle` | `·LES` |
| مكوّن إداريّ | `disputes.tsx:347` | `submit` | `POST` | `/api/v1/admin/disputes` | `·LE·` |
| مكوّن إداريّ | `OrdersScreen.tsx:1481` | `forwardToMerchant` | `POST` | `/api/v1/admin/orders/{v}/whatsapp` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1504` | `settleGoods` | `POST` | `/api/v1/admin/orders/{v}/goods` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1542` | `recompute` | `POST` | `/api/v1/admin/orders/{v}/recompute` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1576` | `compensate` | `POST` | `/api/v1/admin/orders/{v}/compensate-driver` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1602` | `assign` | `POST` | `/api/v1/admin/orders/{v}/assign` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1633` | `transfer` | `POST` | `/api/v1/admin/orders/{v}/transfer` | `ULES` |
| مكوّن إداريّ | `OrdersScreen.tsx:1656` | `go` | `POST` | `/api/v1/admin/orders/{v}/transition` | `ULES` |
| مكوّن إداريّ | `banners.tsx:87` | `toggleActive` | `PATCH` | `/api/v1/admin/banners/{v}` | `··ES` |
| مكوّن إداريّ | `banners.tsx:98` | `remove` | `DELETE` | `/api/v1/admin/banners/{v}` | `U·ES` |
| مكوّن إداريّ | `banners.tsx:207` | `submit` | `PATCH` | `/api/v1/admin/banners/{v}` | `ULE·` |
| مكوّن إداريّ | `banners.tsx:211` | `submit` | `POST` | `/api/v1/admin/banners/{v}` | `ULE·` |
| مكوّن إداريّ | `cities.tsx:161` | `save` | `PUT` | `/api/v1/admin/cities/{v}` | `·LES` |
| مكوّن إداريّ | `cities.tsx:165` | `save` | `POST` | `/api/v1/admin/cities/{v}` | `·LES` |
| مكوّن إداريّ | `cities.tsx:187` | `toggleActive` | `PUT` | `/api/v1/admin/cities/{v}` | `·LES` |
| مكوّن إداريّ | `cities.tsx:207` | `remove` | `DELETE` | `/api/v1/admin/cities/{v}` | `U·ES` |
| مكوّن إداريّ | `divisions.tsx:106` | `toggleGov` | `PUT` | `/api/v1/admin/governorates/{v}` | `··ES` |
| مكوّن إداريّ | `divisions.tsx:122` | `toggleDist` | `PUT` | `/api/v1/admin/districts/{v}` | `··ES` |
| مكوّن إداريّ | `divisions.tsx:138` | `removeGov` | `DELETE` | `/api/v1/admin/governorates/{v}` | `··ES` |
| مكوّن إداريّ | `divisions.tsx:149` | `removeDist` | `DELETE` | `/api/v1/admin/districts/{v}` | `··ES` |
| مكوّن إداريّ | `divisions.tsx:384` | `submit` | `PUT` | `/api/v1/admin/governorates/{v}` | `ULE·` |
| مكوّن إداريّ | `divisions.tsx:388` | `submit` | `POST` | `/api/v1/admin/governorates/{v}` | `ULE·` |
| مكوّن إداريّ | `divisions.tsx:454` | `submit` | `PUT` | `/api/v1/admin/districts/{v}` | `ULE·` |
| مكوّن إداريّ | `divisions.tsx:458` | `submit` | `POST` | `/api/v1/admin/districts/{v}` | `ULE·` |
| مكوّن إداريّ | `whatsapp.tsx:41` | `unpair` | `POST` | `/api/v1/admin/whatsapp/unpair` | `·L·S` |
| مكوّن إداريّ | `whatsapp.tsx:54` | `pair` | `POST` | `/api/v1/admin/whatsapp/pair` | `·L·S` |
| مكوّن إداريّ | `zones.tsx:104` | `save` | `PATCH` | `/api/v1/admin/zones/{v}` | `·LES` |
| مكوّن إداريّ | `zones.tsx:106` | `save` | `POST` | `/api/v1/admin/zones/{v}` | `·LES` |
| مكوّن إداريّ | `zones.tsx:121` | `toggleActive` | `PATCH` | `/api/v1/admin/zones/{v}` | `·LES` |
| مكوّن إداريّ | `zones.tsx:136` | `deleteZone` | `DELETE` | `/api/v1/admin/zones/{v}` | `ULES` |
| مكوّن إداريّ | `tickets.tsx:373` | `submit` | `POST` | `/api/v1/admin/tickets` | `·LE·` |
| مكوّن إداريّ | `tickets.tsx:482` | `sendReply` | `POST` | `/api/v1/admin/tickets/{v}/replies` | `ULES` |
| مكوّن إداريّ | `tickets.tsx:502` | `resolve` | `POST` | `/api/v1/admin/tickets/{v}/resolve` | `ULES` |
| موقع: join | `page.tsx:137` | `submit` | `POST` | `/api/v1/public/join` | `·LE·` |
