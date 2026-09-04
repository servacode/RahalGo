# رحّال غو — خريطةُ أثر الإعدادات

> **CODE TRUTH BASELINE**: `26f93c5d`

---

## المقاماتُ — **وتصحيحُ رقمٍ سابق**

| | العدد |
|---|---|
| **مفاتيحُ الكتالوج** (`internal/settings/catalog.go`) | **١٢٧** |
| **مفاتيحُ يقرؤها المحرّكُ فعلاً** | **٨٦** |
| ↳ **تشغيليّةٌ تغيّر السلوك** | **٦٠** |
| ↳ **عرضٌ وهويّةٌ ونصوصُ صفحات** | **٢٦** |
| **مفاتيحُ لا يقرؤها المحرّك** | **٤١** — يقرؤها الويبُ مباشرةً أو غيرُ مستعملة ❓ |

**⚠️ وقياسي السابق قال ٣٨ تشغيليّاً** — **وكان بنمطٍ أضيق** لم يشمل
`intSetting` و`settingInt` ولا `GetFloat`. **والصوابُ ٦٠.**

## متى يسري التغيير — **مُثبَتٌ لجميعها**

**`internal/settings/settings.go`:**

```go
func (s *Store) Get(ctx, key, out) error {
    s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key)
}
```

| | |
|---|---|
| **ذاكرةٌ وسيطة** | **لا شيء** — لا Redis ولا `sync.Map` ولا TTL |
| **الأثر** | **فوريٌّ في النداء التالي** — بلا إعادة تشغيل ✅ |
| **مفتاحٌ غائب** | **يردّ افتراضَ الكتالوج ولا يفشل أبداً** |
| **الثمن** | **استعلامُ قاعدةٍ لكلّ قراءةِ مفتاح** — ⚠️ **حِملٌ لم يُقَس** |

## التوزّعُ التشغيليّ — ٦٠

| المجال | العدد | ما يمسّه |
|---|---|---|
| **`orders.*`** | **١٤** | المهلُ · القبولُ التلقائيّ · نافذةُ إلغاء الزبون · تعدّدُ المصادر |
| **`drivers.*`** | **١٠** | **سقفُ النقد · نمطُ التوزيع · تردّدُ الموقع · مهلةُ العرض · صورةُ التسليم** |
| **`merchants.*`** | **٧** | **العمولة** · حظرُ الإلغاء · مراجعةُ القائمة · زمنُ التحضير |
| **`security.*`** | **٦** | **عمرُ الجلسة · مهلةُ الرمز · طولُ كلمة المرور · حدُّ المحاولات** |
| **`delivery.*`** | **٥** | **الأجرةُ والمسافةُ ونصفُ القطر الافتراضيّ** |
| **`customers.*`** | **٤** | حظرُ النقد · حدُّ العناوين · هديّةُ التسجيل |
| **`auth.*`** · **`sales.*`** | ٣ لكلٍّ | دخولٌ بالرمز · **عمولةُ المندوب وهدفُه** |
| **`referral.*`** | ٢ | مكافآتُ الإحالة |
| **`app` · `payouts` · `platform.orders_mode` · `pricing` · `support` · `whatsapp`** | ١ لكلٍّ | — |

## أخطرُ عشرةٍ — **تغييرُها يقلب سلوكاً**

| المفتاح | يقرؤه | ما يقلبه |
|---|---|---|
| **`drivers.assignment_mode`** | `orders/rotation.go` | **سباقٌ مفتوحٌ أم عرضٌ بالدور** — **وهو `queue` في الإنتاج** |
| **`drivers.cash_limit`** | `cashbox` · `orders` | **من يستطيع قبولَ طلبٍ نقديّ** |
| **`drivers.location_ping_sec`** | `server` | **تردّدُ إرسال الموقع من الجهاز** |
| **`drivers.max_active_orders`** | `orders` · `server` | كم طلباً بيد سائق |
| **`merchants.commission_percent`** | `pricing` | **نصيبُ المنصّة من كلّ طلب** |
| **`sales.commission_percent`** | `pricing` | **عمولةُ المندوب** |
| **`delivery.fee` · `per_km` · `by_distance`** | `pricing` | **ما يدفعه الزبون** |
| **`orders.auto_accept_min`** | `orders/watchdog.go` | **قبولٌ آليٌّ لما نُسي** |
| **`security.session_days`** | `identity` | **متى تنتهي جلسةُ السائق** |
| **`delivery.default_radius_m`** | `server/city_filter.go` | **من يرى السوقَ أصلاً** |

## ما يلزم إعادةُ تشغيله عند تغيير كلٍّ — **تسجيلٌ لا بناء**

| إن تغيّر | يلزم إعادةُ التحقّق من |
|---|---|
| `drivers.assignment_mode` | **دورةُ الإسناد كاملةً** — الطابورُ والعرضُ والمهلةُ والراصد |
| `drivers.cash_limit` | قبولُ السائق · تسويةُ الصندوق |
| `delivery.*` · `merchants.commission_percent` · `sales.commission_percent` | **التسعيرُ والتسويةُ والثوابتُ المالية الثلاثةَ عشرة** |
| `orders.*` المهل | الراصدُ والتصعيدُ والقبولُ التلقائيّ |
| `security.*` | الدخولُ والتجديدُ وانتهاءُ الجلسة |
| `delivery.default_radius_m` · مضلّعاتُ المناطق | **ترشيحُ السوق ومنعُ الطلب** |

---

## الجدولُ الكامل — ٦٠ مفتاحاً تشغيليّاً

| المفتاح | من يقرؤه |
|---|---|
| `app.max_file_mb` | server |
| `auth.otp_login` | identity, server |
| `auth.require_whatsapp` | settings |
| `auth.signup_verify` | identity, server |
| `customers.cash_ban_days` | orders |
| `customers.cash_ban_failures` | orders |
| `customers.max_addresses` | server |
| `customers.signup_bonus` | referrals |
| `delivery.by_distance` | pricing |
| `delivery.default_radius_m` | server |
| `delivery.fee` | pricing |
| `delivery.max_fee` | pricing |
| `delivery.per_km` | pricing |
| `drivers.assigned_silence_sec` | orders |
| `drivers.assignment_mode` | orders |
| `drivers.cash_limit` | cashbox, orders, server |
| `drivers.direct_assign` | orders |
| `drivers.failed_compensation_percent` | orders |
| `drivers.location_ping_sec` | server |
| `drivers.max_active_orders` | orders, server |
| `drivers.offer_timeout_sec` | orders |
| `drivers.require_delivery_photo` | server |
| `drivers.same_route_extra` | orders |
| `merchants.cancel_ban_count` | orders, server |
| `merchants.cancel_ban_days` | catalog, orders, server |
| `merchants.cancel_ban_mode` | orders, server |
| `merchants.commission_percent` | pricing |
| `merchants.default_prep_minutes` | catalog |
| `merchants.menu_requires_approval` | server |
| `merchants.return_support_percent` | orders |
| `orders.accept_timeout_min` | orders, server |
| `orders.auto_accept_min` | orders |
| `orders.auto_dispatch` | orders |
| `orders.auto_transfer` | server |
| `orders.customer_cancel_window_sec` | orders |
| `orders.delivery_estimate_min` | orders |
| `orders.delivery_timeout_min` | orders, server |
| `orders.driver_timeout_min` | orders, server |
| `orders.extra_source_fee` | orders |
| `orders.handover_timeout_min` | orders |
| `orders.max_open_per_customer` | orders |
| `orders.max_sources` | orders |
| `orders.route_margin_pct` | orders |
| `orders.source_proximity_m` | orders |
| `payouts.min_amount` | server |
| `platform.orders_mode` | orders |
| `pricing.margin_fixed` | pricing |
| `referral.reward_on` | referrals |
| `referral.reward_rest` | referrals |
| `sales.activation_orders` | orders, server |
| `sales.commission_percent` | pricing |
| `sales.monthly_target` | server |
| `security.force_password_change` | identity |
| `security.login_max_attempts` | identity |
| `security.otp_max_per_phone` | identity |
| `security.otp_ttl_min` | identity |
| `security.password_min_length` | identity, server |
| `security.session_days` | identity |
| `support.complaint_window_hours` | server, support |
| `whatsapp.order_template` | server |
