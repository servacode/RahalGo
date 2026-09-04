# رحّال غو — جدولُ الطفرات الخادميّة ١٥٧/١٥٧

> **مُولَّدٌ آليّاً** من `internal/server/server.go` بالتحليل الشجريّ،
> ومُثرًى بقراءة جسم كلّ مُعالِج.
>
> **CODE TRUTH BASELINE**: `26f93c5d`
> **الفهرس**: [`DISCOVERY_COMPLETENESS_AUDIT.md`](DISCOVERY_COMPLETENESS_AUDIT.md)
> **الأفعالُ الحسّاسةُ موصوفةٌ بالبنود الثلاثين** في
> [`ACTION_TRUTH_CATALOG.md`](ACTION_TRUTH_CATALOG.md)

---

## كيف يُقرأ

**الأثر** — مُصنَّفٌ بمرشّحاتٍ على المسار:

| الرمز | المعنى | العدد |
|---|---|---|
| **`FIN`** | **مالٌ يتحرّك** | **١٤** |
| **`PERM`** | **صلاحيّةٌ أو هويّةٌ أو جلسة** | **٢٨** |
| **`STATE`** | **حالُ طلبٍ أو كيانٍ تتبدّل** | **١٨** |
| **`CONF`** | **ضبطٌ وكتالوجٌ ومحتوى** | **٥٩** |
| **`DATA`** | **بياناتُ صاحبها** (عناوين · تفضيلات · رسائل …) | **٣٨** |
| | **المجموع** | **١٥٧** |

**الرايات الستّ** `I T L A N R` — **حرفٌ يعني الوجود، ونقطةٌ تعني الغياب**:

| الحرف | المعنى | العدد |
|---|---|---|
| **`I`** | **مفتاحُ تكرار** (`idempotent`) | **٦** |
| **`T`** | **معاملةٌ صريحة** (`Begin`) في المُعالِج | **٩** |
| **`L`** | **قفلُ صفّ** (`FOR UPDATE`) في المُعالِج | **٥** |
| **`A`** | **تدقيقٌ** (`s.audit`) في المُعالِج | **٤١** |
| **`N`** | **إشعارٌ** يُرسَل | — |
| **`R`** | **بثٌّ حيّ** (`touch`) | — |

> **⚠️ والرايةُ تقول «في المُعالِج»** — **وغيابُها لا يعني الغياب
> المطلق**: المعاملاتُ والأقفالُ والتدقيقُ تقع كثيراً في **طبقة
> الخدمة** (`internal/orders` · `internal/identity` · `internal/catalog`)،
> **وقد قِيس ذلك: ١٠٤ اسمَ تدقيقٍ في أربع طبقاتٍ مقابل ٤١ في
> المُعالِجات.**

**والحارسُ `OPEN`** يعني **بلا مصادقةٍ إطلاقاً** — **١٧ طفرةً كلُّها
`/auth/*`.**

---

## الجدول

| الفعل | المسار | الحارس | الأثر | `I T L A N R` | المُعالِج |
|---|---|---|---|---|---|
| `POST` | `/admin/drivers/{id}/end-shift` | admin,ops,finance | DATA | `···ANR` | `handleAdminEndShift` |
| `POST` | `/admin/emergencies/{id}/resolve` | admin,ops,finance | STATE | `···ANR` | `handleResolveEmergency` |
| `POST` | `/admin/leads/{id}/status` | admin,ops,finance | STATE | `····NR` | `handleAdminLeadStatus` |
| `POST` | `/admin/media` | admin,ops,finance | CONF | `······` | `handleUploadMedia` |
| `POST` | `/admin/orders/{id}/assign` | admin,ops,finance | STATE | `···AN·` | `handleOrderAssign` |
| `POST` | `/admin/orders/{id}/goods` | admin,ops,finance | FIN | `···A·R` | `handleGoods` |
| `POST` | `/admin/orders/{id}/settle-goods` | admin,ops,finance | FIN | `······` | `handleSettleGoods` |
| `POST` | `/admin/orders/{id}/transition` | admin,ops,finance | STATE | `···A··` | `handleOrderTransition` |
| `POST` | `/admin/orders/{id}/whatsapp` | admin,ops,finance | CONF | `···A·R` | `handleSendOrderToMerchant` |
| `POST` | `/admin/tickets` | admin,ops,finance | DATA | `····N·` | `handleCreateTicket` |
| `POST` | `/admin/tickets/{id}/replies` | admin,ops,finance | DATA | `····N·` | `handleTicketReply` |
| `POST` | `/admin/app-file` | admin,ops,finance|admin | CONF | `···A··` | `handleUploadAppFile` |
| `DELETE` | `/admin/app-file` | admin,ops,finance|admin | CONF | `···A··` | `handleDeleteAppFile` |
| `POST` | `/admin/banners` | admin,ops,finance|admin | CONF | `······` | `handleCreateBanner` |
| `PATCH` | `/admin/banners/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateBanner` |
| `DELETE` | `/admin/banners/{id}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteBanner` |
| `POST` | `/admin/broadcast` | admin,ops,finance|admin | CONF | `···A··` | `handleBroadcast` |
| `POST` | `/admin/categories` | admin,ops,finance|admin | CONF | `······` | `handleCreateCategory` |
| `PATCH` | `/admin/categories/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateCategory` |
| `POST` | `/admin/cities` | admin,ops,finance|admin | CONF | `······` | `handleCreateCity` |
| `PUT` | `/admin/cities/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateCity` |
| `DELETE` | `/admin/cities/{id}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteCity` |
| `POST` | `/admin/districts` | admin,ops,finance|admin | CONF | `······` | `handleCreateDistrict` |
| `PUT` | `/admin/districts/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateDistrict` |
| `DELETE` | `/admin/districts/{id}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteDistrict` |
| `POST` | `/admin/governorates` | admin,ops,finance|admin | CONF | `······` | `handleCreateGovernorate` |
| `PUT` | `/admin/governorates/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateGovernorate` |
| `DELETE` | `/admin/governorates/{id}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteGovernorate` |
| `PATCH` | `/admin/menu/items/{itemID}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateItem` |
| `DELETE` | `/admin/menu/items/{itemID}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteItem` |
| `POST` | `/admin/menu/items/{itemID}/review` | admin,ops,finance|admin | CONF | `···ANR` | `handleReviewMenuItem` |
| `PATCH` | `/admin/menu/sections/{sectionID}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateSection` |
| `DELETE` | `/admin/menu/sections/{sectionID}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteSection` |
| `POST` | `/admin/merchants` | admin,ops,finance|admin | CONF | `······` | `handleCreateMerchant` |
| `PATCH` | `/admin/merchants/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateMerchant` |
| `POST` | `/admin/merchants/{id}/clear-violations` | admin,ops,finance|admin | PERM | `···A·R` | `handleClearViolations` |
| `PUT` | `/admin/merchants/{id}/hours` | admin,ops,finance|admin | CONF | `······` | `handleSetHours` |
| `POST` | `/admin/merchants/{id}/menu/items` | admin,ops,finance|admin | CONF | `······` | `handleCreateItem` |
| `POST` | `/admin/merchants/{id}/menu/sections` | admin,ops,finance|admin | CONF | `······` | `handleCreateSection` |
| `POST` | `/admin/merchants/{id}/suspend` | admin,ops,finance|admin | PERM | `···A·R` | `handleSuspendMerchant` |
| `POST` | `/admin/merchants/{id}/warnings` | admin,ops,finance|admin | PERM | `······` | `handleIssueMerchantWarning` |
| `POST` | `/admin/offers` | admin,ops,finance|admin | CONF | `···A·R` | `handleCreateOffer` |
| `POST` | `/admin/offers/{id}/active` | admin,ops,finance|admin | CONF | `···A·R` | `handleSetOfferActive` |
| `POST` | `/admin/orders/{id}/recompute` | admin,ops,finance|admin | FIN | `·T·A·R` | `handleRecomputeSettlement` |
| `POST` | `/admin/promos` | admin,ops,finance|admin | CONF | `······` | `handleCreatePromo` |
| `PATCH` | `/admin/promos/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdatePromo` |
| `POST` | `/admin/sections` | admin,ops,finance|admin | CONF | `···A·R` | `handleCreatePlatformSection` |
| `PATCH` | `/admin/sections/{id}` | admin,ops,finance|admin | CONF | `···A·R` | `handleUpdatePlatformSection` |
| `DELETE` | `/admin/sections/{id}` | admin,ops,finance|admin | CONF | `···A·R` | `handleDeletePlatformSection` |
| `PUT` | `/admin/settings/{key}` | admin,ops,finance|admin | CONF | `···A·R` | `handleSetSetting` |
| `POST` | `/admin/users` | admin,ops,finance|admin | PERM | `······` | `handleAdminCreateUser` |
| `PATCH` | `/admin/users/{id}` | admin,ops,finance|admin | PERM | `····N·` | `handleAdminUpdateUser` |
| `POST` | `/admin/users/{id}/logout-all` | admin,ops,finance|admin | PERM | `······` | `handleAdminLogoutAll` |
| `POST` | `/admin/users/{id}/password` | admin,ops,finance|admin | PERM | `······` | `handleAdminResetPassword` |
| `POST` | `/admin/users/{id}/roles` | admin,ops,finance|admin | PERM | `····NR` | `handleAdminGrantRole` |
| `DELETE` | `/admin/users/{id}/roles/{role}` | admin,ops,finance|admin | PERM | `····NR` | `handleAdminRevokeRole` |
| `POST` | `/admin/users/{id}/warnings` | admin,ops,finance|admin | PERM | `······` | `handleIssueUserWarning` |
| `POST` | `/admin/whatsapp/pair` | admin,ops,finance|admin | CONF | `······` | `inline` |
| `POST` | `/admin/whatsapp/unpair` | admin,ops,finance|admin | CONF | `······` | `inline` |
| `POST` | `/admin/zones` | admin,ops,finance|admin | CONF | `······` | `handleCreateZone` |
| `PATCH` | `/admin/zones/{id}` | admin,ops,finance|admin | CONF | `······` | `handleUpdateZone` |
| `DELETE` | `/admin/zones/{id}` | admin,ops,finance|admin | CONF | `······` | `handleDeleteZone` |
| `POST` | `/admin/disputes` | admin,ops,finance|admin,finance | FIN | `···A·R` | `handleCreateDispute` |
| `POST` | `/admin/disputes/{id}/settle` | admin,ops,finance|admin,finance | FIN | `·TLA·R` | `handleSettleDispute` |
| `POST` | `/admin/drivers/{id}/settle` | admin,ops,finance|admin,finance | FIN | `I··ANR` | `handleDriverSettle` |
| `POST` | `/admin/expenses` | admin,ops,finance|admin,finance | FIN | `···A·R` | `handleCreateExpense` |
| `POST` | `/admin/expenses/categories` | admin,ops,finance|admin,finance | FIN | `···A··` | `handleSaveExpenseCategory` |
| `POST` | `/admin/expenses/{id}/void` | admin,ops,finance|admin,finance | FIN | `···A·R` | `handleVoidExpense` |
| `POST` | `/admin/orders/{id}/compensate-driver` | admin,ops,finance|admin,finance | FIN | `·TLA·R` | `handleCompensateDriver` |
| `POST` | `/admin/payouts/{id}/decide` | admin,ops,finance|admin,finance | FIN | `ITLANR` | `handleDecidePayout` |
| `POST` | `/admin/tickets/{id}/resolve` | admin,ops,finance|admin,finance | STATE | `···ANR` | `handleTicketResolve` |
| `POST` | `/admin/users/{id}/incentive` | admin,ops,finance|admin,finance | FIN | `I··A·R` | `handleIncentiveGrant` |
| `POST` | `/admin/users/{id}/wallet` | admin,ops,finance|admin,finance | FIN | `I··ANR` | `handleAdminWalletApply` |
| `POST` | `/admin/orders/{id}/transfer` | admin,ops,finance|admin,ops | STATE | `·TLA·R` | `handleTransferOrder` |
| `POST` | `/driver/location` | driver | DATA | `······` | `handleDriverLocation` |
| `POST` | `/driver/location/batch` | driver | DATA | `······` | `handleDriverLocationBatch` |
| `POST` | `/driver/orders/{id}/accept` | driver | STATE | `······` | `handleDriverAccept` |
| `POST` | `/driver/orders/{id}/agree` | driver | DATA | `····NR` | `handleAgreeCustom` |
| `POST` | `/driver/orders/{id}/decline` | driver | STATE | `···A·R` | `handleDriverDecline` |
| `POST` | `/driver/orders/{id}/emergency` | driver | STATE | `···A·R` | `handleDriverEmergency` |
| `POST` | `/driver/orders/{id}/proof` | driver | STATE | `···A·R` | `handleDeliveryProof` |
| `POST` | `/driver/orders/{id}/proof/skip` | driver | STATE | `···A··` | `handleSkipDeliveryProof` |
| `POST` | `/driver/orders/{id}/rate-merchant` | driver | DATA | `·····R` | `handleDriverRateMerchant` |
| `POST` | `/driver/orders/{id}/release` | driver | STATE | `···A·R` | `handleDriverRelease` |
| `POST` | `/driver/orders/{id}/report` | driver | DATA | `·····R` | `handleDriverReport` |
| `POST` | `/driver/orders/{id}/return` | driver | STATE | `·TLA·R` | `handleDriverReturn` |
| `POST` | `/driver/orders/{id}/road-correlation` | driver | DATA | `······` | `handleRoadCorrelation` |
| `POST` | `/driver/orders/{id}/transition` | driver | STATE | `······` | `handleDriverTransition` |
| `POST` | `/driver/shift` | driver | DATA | `·····R` | `handleDriverShift` |
| `POST` | `/merchant/media` | merchant | CONF | `······` | `handleMerchantUploadMedia` |
| `PATCH` | `/merchant/menu/items/{itemID}` | merchant | CONF | `······` | `handleMerchantUpdateItem` |
| `DELETE` | `/merchant/menu/items/{itemID}` | merchant | CONF | `······` | `handleMerchantDeleteItem` |
| `PATCH` | `/merchant/menu/items/{itemID}/availability` | merchant | CONF | `·····R` | `handleMerchantItemAvailability` |
| `PATCH` | `/merchant/menu/sections/{sectionID}` | merchant | CONF | `······` | `handleMerchantUpdateSection` |
| `DELETE` | `/merchant/menu/sections/{sectionID}` | merchant | CONF | `······` | `handleMerchantDeleteSection` |
| `POST` | `/merchant/orders/{id}/ready` | merchant | STATE | `·····R` | `handleMerchantReady` |
| `POST` | `/merchant/orders/{id}/report` | merchant | DATA | `·····R` | `handleMerchantReport` |
| `POST` | `/merchant/orders/{id}/transition` | merchant | STATE | `······` | `handleMerchantTransition` |
| `POST` | `/merchant/stores/{id}/emergency` | merchant | STATE | `·····R` | `handleMerchantEmergency` |
| `PUT` | `/merchant/stores/{id}/hours` | merchant | CONF | `······` | `handleMerchantSetHours` |
| `POST` | `/merchant/stores/{id}/menu/items` | merchant | CONF | `······` | `handleMerchantCreateItem` |
| `POST` | `/merchant/stores/{id}/menu/sections` | merchant | CONF | `······` | `handleMerchantCreateSection` |
| `PUT` | `/merchant/stores/{id}/sections` | merchant | CONF | `·T····` | `handleMerchantSetStoreSections` |
| `PATCH` | `/merchant/stores/{id}/settings` | merchant | CONF | `······` | `handleMerchantSettings` |
| `POST` | `/rep/leads` | sales | DATA | `·····R` | `handleRepCreateLead` |
| `POST` | `/rep/media` | sales | CONF | `······` | `handleUploadMedia` |
| `PATCH` | `/rep/menu/items/{itemID}` | sales | CONF | `······` | `handleRepUpdateItem` |
| `DELETE` | `/rep/menu/items/{itemID}` | sales | CONF | `······` | `handleRepDeleteItem` |
| `POST` | `/rep/stores/{id}/menu/items` | sales | CONF | `······` | `handleRepCreateItem` |
| `POST` | `/auth/account/delete/confirm` | AUTH | DATA | `······` | `handleDeleteAccountConfirm` |
| `POST` | `/auth/account/delete/request` | AUTH | DATA | `······` | `handleDeleteAccountRequest` |
| `POST` | `/auth/handoff` | AUTH | DATA | `······` | `handleHandoff` |
| `POST` | `/auth/login` | OPEN | PERM | `······` | `handlePasswordLogin` |
| `POST` | `/auth/logout` | OPEN | PERM | `······` | `handleLogout` |
| `POST` | `/auth/otp/request` | OPEN | PERM | `······` | `handleOTPRequest` |
| `POST` | `/auth/otp/verify` | OPEN | PERM | `······` | `handleOTPVerify` |
| `POST` | `/auth/password` | AUTH | PERM | `······` | `handleSetPassword` |
| `POST` | `/auth/password/reset/confirm` | OPEN | PERM | `······` | `handleResetConfirm` |
| `POST` | `/auth/password/reset/request` | OPEN | PERM | `······` | `handleResetRequest` |
| `POST` | `/auth/password/reset/verify` | OPEN | PERM | `······` | `handleResetVerify` |
| `POST` | `/auth/phone/confirm` | AUTH | DATA | `······` | `handlePhoneChangeConfirm` |
| `POST` | `/auth/phone/request` | AUTH | DATA | `······` | `handlePhoneChangeRequest` |
| `POST` | `/auth/pin` | OPEN | PERM | `······` | `handlePinVerify` |
| `POST` | `/auth/pin/change` | AUTH | PERM | `···A··` | `handlePinChange` |
| `POST` | `/auth/pin/reset/confirm` | AUTH | PERM | `···A··` | `handlePinResetConfirm` |
| `POST` | `/auth/pin/reset/request` | AUTH | PERM | `······` | `handlePinResetRequest` |
| `POST` | `/auth/pin/setup` | OPEN | PERM | `······` | `handlePinSetup` |
| `POST` | `/auth/refresh` | OPEN | PERM | `······` | `handleRefresh` |
| `POST` | `/auth/signup/confirm` | OPEN | PERM | `······` | `handleSignupConfirm` |
| `POST` | `/auth/signup/request` | OPEN | PERM | `······` | `handleSignupRequest` |
| `POST` | `/auth/signup/verify` | OPEN | PERM | `······` | `handleSignupVerify` |
| `POST` | `/auth/sso` | OPEN | PERM | `······` | `handleSSO` |
| `POST` | `/auth/wa/ticket` | OPEN | DATA | `······` | `handleWATicket` |
| `POST` | `/auth/whatsapp/confirm` | AUTH | CONF | `······` | `handleWhatsAppVerifyConfirm` |
| `POST` | `/auth/whatsapp/request` | AUTH | CONF | `······` | `handleWhatsAppVerifyRequest` |
| `POST` | `/me/avatar` | AUTH | DATA | `······` | `handleMyAvatar` |
| `DELETE` | `/me/avatar` | AUTH | DATA | `······` | `handleDeleteMyAvatar` |
| `POST` | `/me/devices` | AUTH | DATA | `······` | `handleDeviceRegister` |
| `DELETE` | `/me/devices` | AUTH | DATA | `······` | `handleDeviceUnregister` |
| `POST` | `/me/devices/test` | AUTH | DATA | `······` | `handlePushTest` |
| `PATCH` | `/me/name` | AUTH | DATA | `···A··` | `handleSetMyName` |
| `POST` | `/me/notifications/read` | AUTH | DATA | `······` | `handleMarkNotificationRead` |
| `POST` | `/me/payouts` | AUTH | FIN | `·····R` | `handleCreatePayout` |
| `POST` | `/my/addresses` | AUTH | DATA | `·T····` | `handleCreateAddress` |
| `PATCH` | `/my/addresses/{id}` | AUTH | DATA | `······` | `handleUpdateAddress` |
| `DELETE` | `/my/addresses/{id}` | AUTH | DATA | `······` | `handleDeleteAddress` |
| `POST` | `/my/addresses/{id}/default` | AUTH | DATA | `·T····` | `handleSetDefaultAddress` |
| `POST` | `/my/favorites/{id}` | AUTH | DATA | `······` | `handleToggleFavorite` |
| `POST` | `/my/orders/{id}/complaint` | AUTH | DATA | `···AN·` | `handleOpenComplaint` |
| `POST` | `/orders` | AUTH | DATA | `I·····` | `handleCustomerCreateOrder` |
| `POST` | `/orders/custom` | AUTH | DATA | `I····R` | `handleCreateCustomOrder` |
| `POST` | `/orders/{id}/cancel` | AUTH | STATE | `······` | `handleCustomerCancelOrder` |
| `POST` | `/orders/{id}/messages` | AUTH | DATA | `····NR` | `handleSendOrderMessage` |
| `POST` | `/orders/{id}/rating` | AUTH | DATA | `····N·` | `handleRateOrder` |
| `POST` | `/promo/preview` | AUTH | DATA | `······` | `handlePromoPreview` |
| `POST` | `/public/join` | OPEN | DATA | `····N·` | `handlePublicJoin` |
| `POST` | `/public/quote` | OPEN | DATA | `······` | `handleQuote` |
