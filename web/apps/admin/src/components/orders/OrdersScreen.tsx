"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtTime,
  fmtDateTime,
} from "@rahalgo/i18n";
import {
  IconNote,
  IconEdit,
  Stars,
  useLiveEvent,
  useLiveStatus,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconOrder,
  IconSearch,
  IconUser,
  IconStore,
  IconWallet,
  IconDriver,
  IconCamera,
  IconStatus,
  IconLocation,
  IconStar,
  IconBalance,
  IconPhone,
  IconWhatsApp,
  IconSwap,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

// ---------- الأنواع ----------

/**
 * تفصيلُ توزيع مال الطلب — **مقروءاً من الدفتر لا محسوباً هنا.**
 *
 * لو حُسبت الأنصبةُ في الواجهة بالمعادلات لظهرت طلباتُ الأمس بأجرٍ لم يُقبض
 * حين تتغيّر نسبةُ السائق اليوم. **وشاشةٌ تقرأ الدفتر لا تكذب عليه.**
 */
type Breakdown = {
  total: number;
  subtotal: number;
  delivery_fee: number;
  discount: number;
  to_parties: number;
  platform: number;
  lines: {
    party: "merchant" | "driver" | "sales" | "platform" | "customer" | "other";
    name: string;
    kind: string;
    amount: number;
    note: string;
  }[];
};

/** سطرٌ في تفصيل التوزيع — عنوانٌ يميناً ومبلغٌ يساراً بخانةٍ ثابتة. */
function Row({
  label,
  value,
  strong,
  danger,
}: {
  label: string;
  value: number;
  strong?: boolean;
  danger?: boolean;
}) {
  return (
    <div
      className={`flex items-center justify-between gap-3 ${
        strong ? "font-bold" : ""
      } ${danger ? "text-danger" : ""}`}
    >
      <span className="truncate">{label}</span>
      <span dir="ltr" className="shrink-0 tabular-nums">
        {fmtNum(value)} {m.common.currency}
      </span>
    </div>
  );
}

/** رسالةُ المتجر كما يبنيها الخادم — نصّاً ورابطاً معاً، فلا يفترقان. */
type MerchantMessage = {
  text: string;
  phone: string;
  /** رابطُ واتساب جاهزاً — فارغٌ إن كان رقمُ المتجر غيرَ صالح */
  wa_link: string;
  /** أمُهيَّأةٌ بوّابةُ الرسائل النصّية؟ */
  sms_ready: boolean;
};

interface DriverRow {
  id: string;
  full_name: string;
  phone: string;
  status: string;
  on_shift: boolean;
  open_orders: number;
}

interface OrderRow {
  id: string;
  number: number;
  customer_phone: string;
  customer_name: string;
  merchant_name: string;
  merchant_id: string;
  driver_phone: string | null;
  driver_name: string | null;
  /** من عُرض عليه الطلبُ ولم يقبل بعد — **يُعرض ما دام العرضُ حيّاً.** */
  offered_driver_name: string | null;
  /** إثباتُ التسليم — صورةٌ ومسافةٌ ووقت. */
  proof_url?: string | null;
  proof_taken_at?: string | null;
  /** بُعدُ موضع التسليم عن عنوان الزبون — و`-1` تعني «لا موضعَ محفوظ». */
  proof_meters?: number;
  proof_skip_reason?: string;
  /** متى حُوِّل الطلب إلى المتجر — فارغٌ يعني لم يُحوَّل بعد */
  sent_to_merchant_at: string | null;
  /** متى نزل إلى طابور السائقين — ومنه تُقاس مهلةُ زرّ الإسناد */
  dispatched_at: string | null;
  /** لماذا لا يلتقطه أحد — **وفارغٌ حين لا مشكلة**. */
  blocked_reason?: string;
  /** مصيرُ بضاعة طلبٍ فشل — فارغٌ يعني لم يُحسم بعد */
  goods_settled_to: "merchant" | "platform" | null;
  /** أجرُ السائق — تقديرٌ قبل التسليم وواقعٌ بعده، من مصدر الحساب نفسه */
  driver_fee: number;
  status: string;
  payment_method: "cash" | "wallet";
  subtotal: number;
  delivery_fee: number;
  discount: number;
  total: number;
  wallet_paid: number;
  cash_due: number;
  promo_code: string | null;
  address_text: string;
  notes: string;
  cancel_reason: string;
  /** الدورُ الذي أنهى الطلب — يُقرأ ولا يُرسَل. */
  ended_by?: string;
  fail_reason?: string;
  fault?: string;
  created_at: string;
  items?: {
    id: string;
    name: string;
    unit_price: number;
    qty: number;
    note: string;
    options: { group: string; name: string; price_delta: number }[];
  }[];
  events?: {
    from_status: string;
    to_status: string;
    note: string;
    created_at: string;
  }[];
  rating?: {
    platform_stars: number;
    driver_stars: number | null;
    comment: string;
    created_at: string;
  };
}

interface OrderPage {
  orders: OrderRow[];
  total: number;
  page: number;
  per_page: number;
}

interface Alert {
  order_id: string;
  number: number;
  status: string;
  merchant_name: string;
  customer_phone: string;
  reason: "no_accept" | "no_driver" | "too_long";
  minutes: number;
}

const STATUS_LABELS: Record<string, string> = m.orders.status;
const ENDED_BY: Record<string, string> = m.admin.ordersPage.endedBy;
const ACTION_LABELS: Record<string, string> = m.admin.ordersPage.actions;
const PAYMENT_LABELS: Record<string, string> = m.orders.payment;

const STATUS_VARIANT: Record<
  string,
  "warning" | "primary" | "success" | "danger" | "neutral"
> = {
  pending: "warning",
  accepted: "primary",
  preparing: "primary",
  dispatching: "warning",
  assigned: "primary",
  at_pickup: "primary",
  picked_up: "primary",
  on_the_way: "primary",
  at_dropoff: "primary",
  delivered: "success",
  rejected: "danger",
  cancelled: "danger",
  failed: "danger",
  refunded: "neutral",
};

/**
 * الأفعالُ الهدّامة — تُطلب لها ضغطةٌ ثانية على البطاقة.
 *
 * كلُّها تُنهي الطلب أو تعكس مالاً: الرفضُ والإلغاءُ يُرجعان ما دُفع، والفشلُ
 * يُغلق بلا تسليم، والاسترجاعُ يعكس تسويةً تمّت. **وما لا يُستدرَك لا يُترك
 * لضغطةٍ واحدة.**
 */
const DESTRUCTIVE = new Set(["rejected", "cancelled", "failed", "refunded"]);

/**
 * **الحالاتُ المنتهية** — مرآةُ `terminal()` في المحرّك.
 *
 * تلزم هنا لأنّ في الصفحة **مُرشِّحَين يستطيعان التناقض**: قائمةُ الحالة
 * و«الجارية فقط». **واختيارُ «مُسلَّم» والمربّعُ مؤشَّرٌ يُخرج لا شيء** —
 * `AND (NOT $6 OR o.closed_at IS NULL)` — **ولا كلمةَ تقول لماذا.**
 *
 * وقع فعلاً (٢٠٢٦-٠٨-٠٣): بحث المالكُ عن طلبٍ مُسلَّمٍ ليسترجعه فلم يجد شيئاً.
 * **وقائمةٌ فارغةٌ تُقرأ «لا طلباتِ لديك» لا «مُرشِّحاك يتنازعان».**
 */
const CLOSED_STATUSES = new Set([
  "delivered",
  "rejected",
  "cancelled",
  "failed",
  "refunded",
]);

/**
 * **الحالاتُ التي جرت فيها تسويةٌ ماليّة** — وهي وحدَها تُعاد حسبتُها.
 *
 * الملغى والمرفوض **لا أنصبةَ لهما**: لم تخرج بضاعةٌ ولم يُقبض مال. **وزرُّ
 * تصحيحٍ على طلبٍ لا شيءَ فيه يُضغط فلا يقع شيء** — فيُظنّ أنّه معطوب.
 */
const SETTLED_STATUSES = new Set(["delivered", "failed", "refunded"]);

/**
 * **الحالاتُ التي يكون فيها لإثبات التسليم معنى.**
 *
 * الإثباتُ لا يوجد إلّا بعد أن يسلّم السائقُ أو يتعذّر عليه. **وسطرٌ فارغٌ في
 * طلبٍ لم يُقبل بعد يُسأل عنه ولا جواب** — «كيف والطلبُ لسّا ما وافقنا عليه
 * أساساً؟» (المالك ٢٠٢٦-٠٨-٠٤).
 */
/**
 * **إثباتُ التسليم — ولا يُعرض إلّا حيث يقع.**
 *
 * كان يظهر من «السائق وصل»: **صفٌّ فارغٌ يسأل عن صورةٍ لم تُلتقط بعد**، فيُقرأ
 * نقصاً ويُبحث عن سببه. **والفراغُ الذي لا يعني شيئاً يُعلّم العينَ أن تتخطّى
 * الصفَّ** — فلا تراه حين يعني.
 *
 * **ولا في الفشل**: طلبٌ لم يُسلَّم لا إثباتَ تسليمٍ له، **وسببُ التعذّر مكتوبٌ
 * في مكانه.**
 *
 * **ويبقى في المسترجَع**: الاسترجاعُ يقع بعد تسليمٍ وقع — **والصورةُ بيّنتُه
 * حين يُنازَع فيه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «إثباتُ التسليم يظهر فقط بحالة تمّ التسليم».)
 */
const PROOF_STATUSES = new Set(["delivered", "refunded"]);

/**
 * **الحالاتُ التي يكون فيها للسائق معنى.**
 *
 * الطلبُ لا يبلغ السائقَ إلّا بعد أن يُقبل ويُحوَّل ويُعلن جاهزاً — **فحقلُ
 * سائقٍ في طلبٍ «بانتظار التأكيد» يسأل عمّن لا وجودَ له بعد.**
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٤): «إسنادُ السائق لا يظهر في هذه المرحلة أصلاً —
 * الطلبُ ما زال بالانتظار، كيف يكون اسمُ السائق موجوداً؟ وغيرُ احترافيٍّ
 * وجودُه بهذه المرحلة».
 *
 * **وهي القاعدةُ نفسُها التي أخفت إثباتَ التسليم**: حقلٌ يظهر فارغاً قبل أوانه
 * **يُتعلَّم تجاهلُه، ثمّ يمتلئ يوماً فلا يُنظر إليه.**
 */
const DRIVER_STATUSES = new Set([
  "dispatching",
  "assigned",
  "at_pickup",
  "picked_up",
  "on_the_way",
  "at_dropoff",
  "delivered",
  "failed",
  "refunded",
]);

/**
 * **سعرُ الصنف عارياً من إضافاته.**
 *
 * # وتصحيحُ خطأ
 *
 * `order_items.unit_price` **يشمل فروقَ الخيارات أصلاً**: المحرّكُ يجمعها فيه
 * عند الإنشاء (`it.UnitPrice += delta` في `priceItems`)، **و`subtotal` حاصلُ
 * ضربه في الكمّية.**
 *
 * فكتبتُ دالّةً تجمعها **مرّةً ثانية**، فظهر السطرُ `٣٧٬٥٠٠` وقيمةُ الطلب
 * `٣٢٬٥٠٠` — **ورقمان لواقعةٍ واحدةٍ يتناقضان في شاشةٍ واحدة.** كشفه المالكُ
 * في فاتورته (٢٠٢٦-٠٨-٠٤).
 *
 * **والصوابُ الطرحُ لا الجمع**: الصنفُ عارياً = المحفوظ − مجموعُ الفروق.
 */
function basePrice(it: {
  unit_price: number;
  options?: { price_delta: number }[];
}): number {
  return (
    it.unit_price -
    (it.options ?? []).reduce((s, x) => s + (x.price_delta || 0), 0)
  );
}

// أزرار الانتقال المتاحة للعمليات/الأدمن حسب الحالة (مرآة لخارطة الخادم)
const OPS_NEXT: Record<string, string[]> = {
  // **وزرٌّ واحدٌ للإنهاء قبل القبول.**
  //
  // كان «رفض المتجر» و«إلغاء الطلب» جنباً إلى جنبٍ في طلبٍ لم يُقبل بعد —
  // **وأثرُهما عند الزبون واحد**: طلبٌ انتهى قبل أن يبدأ، ولا مالَ تحرّك، ولا
  // مخالفةَ تُحسب (المنهي هو العمليات لا المتجر). **وزرّان لفعلٍ واحدٍ يجعلان
  // من يضغط يتردّد ثمّ يختار عشوائياً.**
  //
  // **وبقي «الرفض» لا «الإلغاء»** — بقرار المالك (٢٠٢٦-٠٨-٠٤): «نخلّي رفض
  // المتجر، مشان إذا فعّلنا المتاجر لاحقاً: رفضُ المتجر لا يذهب للزبون مباشرةً
  // بل يعود للمنصة وتحوّله لمتجرٍ آخر». **فاللفظُ يحمل مستقبلَه.**
  //
  // **واسمُه اليومَ «رفض» مجرّداً**: «قبولٌ ورفض» زوجٌ يُقرأ بلا تفكير،
  // **و«رفض المتجر» تسأل عمّن رفض** — والرافضُ في وضعنا هو المنصة.
  //
  // **والإلغاءُ يبقى بعد القبول** — هناك يختلفان: أُعلن للزبون أنّ طلبَه قُبل،
  // فإنهاؤه إلغاءٌ لالتزامٍ لا ردٌّ لطلب.
  pending: ["accepted", "rejected"],
  // **ولا «بدء تحضير» في المنصة — ولا إلغاءَ بعد القبول.**
  //
  // # «بدء التحضير» لا وجودَ له
  //
  // **المطبخُ خارج النظام**: يُبلَّغ على واتساب ولا يضغط شيئاً. **فإعلانُ
  // «بدأ يحضّر» عنه ادّعاءُ ما لا نعلمه** — وقد يكون الطبّاخُ لم يفتح الرسالة.
  //
  // والفعلُ الحقيقيُّ اسمُه ما هو: **تحويلُ الطلب إلى المتجر**، أو إلى متجرٍ
  // آخر. **والمحرّكُ يسمح `accepted → dispatching` رأساً** — دون المرور بحالٍ
  // لا يعلمها أحد.
  //
  // # والإلغاءُ بعد القبول مايصير
  //
  // **قبلنا الطلبَ فالتزمنا.** ومتجرٌ لا يستطيع لا يُلغي الطلبَ بل يُحوَّل إلى
  // غيره — **والمنصةُ لم تُفتح لترفض طلبات.**
  //
  // **ويبقى الإلغاءُ لغيرها**: للزبون في نافذته، وللمتجر حين ينفد صنفُه —
  // **وهما يعلمان ما لا تعلمه.**
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٤ — وقد صُحّحت هذه البطاقةُ مرّاتٍ قبله: «ما في
  // شي بالمنصة اسمه بدء التحضير، خلص اسمه تحويل للمتجر أو تحويل لمتجر آخر.
  // وإلغاء الطلب ما يصير أساساً لأنّنا وافقنا على طلب — حتى لو ما في عندي
  // متجرٌ معيّن نجيب من متجرٍ ثانٍ. ونحن ما فتحنا المنصة مشان نرفض طلبات».)
  accepted: [],
  preparing: [], // + إسناد سائق
  dispatching: [], // + إسناد سائق
  assigned: ["at_pickup"],
  at_pickup: ["picked_up"],
  picked_up: ["on_the_way"],
  on_the_way: ["at_dropoff"],
  at_dropoff: ["delivered", "failed"],
  delivered: ["refunded"],
};

/** مراحلُ الطريق — لا يملكها إلّا من يسير فيها (مرآةُ `driverOnly`). */
/** الخطوةُ التالية في مسار السائق — لنافذة التدخّل اليدويّ. */
const NEXT_AFTER: Record<string, string> = {
  assigned: "at_pickup",
  at_pickup: "picked_up",
  picked_up: "on_the_way",
  on_the_way: "at_dropoff",
  at_dropoff: "delivered",
};

/**
 * **من أين يجوز التدخّلُ اليدويّ — وأين يقف.**
 *
 * # القاعدة
 *
 * **بمجرّد أن يستلم السائقُ البضاعةَ تصير المنصةُ مراقبةً لا متحكّمة.**
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لا يجوز تدخّلُ المنصة — قلنا المنصة بمجرّد أن
 * السائق استلم الطلب هي تراقب فقط عمليةَ التسليم، لم يعد بإمكانها التحكّم
 * بشيء».)
 *
 * # ولماذا هنا بالذات
 *
 * **قبل الاستلام لا شيءَ بيد أحد**: طلبٌ أُسند ولم يتحرّك له سائق، **والمنصةُ
 * تملك أن تُنقذه** — تعلن وصولَه إلى المتجر إن كان السائقُ هناك ولا شبكة.
 *
 * **وبعد الاستلام البضاعةُ بيده**: من يقول «تمّ التسليم» يقول ما لا يعلمه —
 * **والمالُ يتحرّك على قوله**: يُقيَّد للمتجر ولنا وللمندوب، ويُحمَّل السائقُ
 * نقداً لم يقبضه. **وتصحيحُ قيدٍ وقع أصعبُ من انتظار من يعرف.**
 *
 * # وإن تعطّل هاتفُ السائق؟
 *
 * **بابُه الاستغاثة لا زرُّ التسليم**: يُفتح على الطلب أثرٌ يُقرأ ويُعالَج،
 * **ولا يُكتب في الدفتر تسليمٌ لم يشهده أحد.**
 */
const MANUAL_STEP_FROM = new Set(["assigned", "at_pickup"]);

/** الحالاتُ التي يجوز فيها التحويل — **قبل خروج البضاعة**. */
/**
 * **الحالاتُ التي يجوز فيها التحويل** — **قبل خروج البضاعة.**
 *
 * # ولا يُعرض في وضع «المنصة تدير»
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٤): «احذف التحويل لمتجرٍ آخر، نمشي حركةً حركة» —
 * ثمّ في النفَس نفسِه: «نُبقي رفضَ المتجر، **مشان إذا فعّلنا المتاجر لاحقاً:
 * رفضُ المتجر لا يذهب للزبون مباشرةً بل يعود للمنصة وتحوّله لمتجرٍ آخر**».
 *
 * **فالتحويلُ ليس زائداً — هو سابقٌ لأوانه.** ومعناه يولد حين يملك المتجرُ
 * أن يرفض: عندها يعود الطلبُ إلى المنصة **ولها أن تُنقذه بمتجرٍ ثانٍ بدل أن
 * تعتذر للزبون.** وفي وضعنا اليوم لا رفضَ من متجرٍ أصلاً، **فالزرُّ يعرض
 * علاجاً لمرضٍ لا وجودَ له.**
 *
 * **ولم تُحذف شيفرتُه** — تُحذف الميزةُ فتُعاد كتابتُها من الذاكرة ناقصة.
 *
 * # ولا قبل القبول
 *
 * `pending` نُزعت: **التحويلُ يقبل الطلبَ ضمناً** (يردّ المسارُ `accepted`)،
 * **فضغطةٌ عليه تقبل نيابةً عن المالك بلا أن يقول قبلت** وتبدأ مهلةُ إلغاء
 * الزبون. والحارسُ في المحرّك أيضاً لا في الشاشة وحدَها.
 */
const TRANSFERABLE = new Set([
  "accepted",
  "preparing",
  "dispatching",
  "assigned",
  "at_pickup",
]);

const DRIVER_ONLY = new Set([
  "at_pickup",
  "picked_up",
  "on_the_way",
  "at_dropoff",
  "delivered",
  "failed",
]);

/** ما بلغه الطلبُ بعد تحويله إلى المتجر (مرآةُ `afterHandoff`). */
const AFTER_HANDOFF = new Set([
  "dispatching",
  "assigned",
  "at_pickup",
  "picked_up",
  "on_the_way",
  "at_dropoff",
]);

/**
 * ما تملكه العملياتُ فعلاً — **مرآةُ `orders/modes.go`**.
 *
 * # القاعدةُ الحاكمة
 *
 * **بعد التحويل إلى المتجر: المنصةُ عينٌ لا يد.** قامت بدورها — قبلت وحوّلت —
 * وانتهى عملُها. وما بعدها يقع في مطبخٍ لا تراه وعلى طريقٍ لا تسلكه.
 *
 * # والخادمُ هو الحَكَم
 *
 * هذه الدالة **تُخفي ما سيُردّ** فلا يضغط الموظّفُ زرّاً يعتذر. ولو انحرفت عن
 * الخادم لظهر زرٌّ لا يعمل — **وهو أخفُّ ضرراً من زرٍّ يعمل ولا يجب أن يعمل**.
 *
 * # والأدمن فوقها
 *
 * المالكُ يبقى قادراً وكلُّ فعلٍ له مُسجَّل. **ونظامٌ بلا تجاوزٍ في أيّ موضع
 * يُصلَح بجراحةٍ في قاعدة البيانات حين يقع ما لم يُحسب.**
 */
function opsNext(
  status: string,
  selfManage: boolean,
  hasDriver: boolean,
  isAdmin: boolean,
): string[] {
  let next = OPS_NEXT[status] ?? [];

  // **ولا تُعلن العملياتُ ولا المالكُ بدءَ تحضيرٍ لم يبدأه أحدٌ منهما.**
  //
  // **قبل استثناء الأدمن لا بعده**: كان الاستثناءُ يسبق كلَّ حارس، **فسقط
  // أهمُّها عمّن أحدث المشكلة** — وشكوى المالك قالت «ضُغطت بيد الأدمن».
  //
  // **وهي مسألةُ معرفةٍ لا صلاحية**: كونُك المالكَ لا يمنحك عِلماً بما يجري في
  // مطبخِ غيرك. والمحرّكُ يفرضها أيضاً (`modes.go`) — **والشاشةُ تُخفي ما
  // يرفضه الخادم، فلا زرَّ يَعِد بما يُعتذر عنه.**
  if (!selfManage) next = next.filter((t) => t !== "preparing");

  // **ومراحلُ الطريق مقروءةٌ للمنصة لا ملموسة** — نصُّ قرار المالك. ولا
  // استثناءَ له: **لا هو ولا موظّفُه يعلم أنّ السائقَ وصل أو استلم.**
  next = next.filter((t) => !DRIVER_ONLY.has(t));

  // **وما بقي فسلطةٌ يملكها المالك** — تدخّلٌ مسجَّلٌ في سجلّ التدقيق:
  // القبولُ نيابةً عن متجرٍ لا يستجيب، والإلغاءُ بعد التحويل.
  if (isAdmin) return next;

  // **المتجر يدير**: القبولُ والرفضُ اختصاصُه.
  if (selfManage && status === "pending") {
    next = next.filter((t) => t !== "accepted" && t !== "rejected");
  }

  // **ولا تُعلن العملياتُ بدءَ تحضيرٍ لم تبدأه** — التحضيرُ فعلٌ في مطبخٍ
  // خارج النظام. وقد وقعت فعلاً في تجربةٍ حيّة: ضُغطت بعد ثانيةٍ من القبول
  // **والمتجرُ لم يُبلَّغ بعد**.
  next = next.filter((t) => t !== "preparing");

  // **وبعد التحويل لا تُلغي طلباً** — ولو لم يمسكه سائقٌ بعد. وما قبله بيدها:
  // طلبٌ لم يعلم به مطبخٌ ولا تحرّك له سائق.
  if (AFTER_HANDOFF.has(status)) {
    next = next.filter((t) => t !== "cancelled");
  }
  return next;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError
    ? translateKey(err.body.message_key)
    : m.errors.internal;
}

// ---------- الشاشة الرئيسية ----------

export default function OrdersScreen({ mode }: { mode: "live" | "history" }) {
  const params = useSearchParams();
  const initialQ = params.get("q") ?? "";
  const [data, setData] = useState<OrderPage | null>(null);
  const [status, setStatus] = useState("");
  const [query, setQuery] = useState(initialQ);
  /**
   * **شاشتان لا شاشةٌ بمربّع.**
   *
   * كانت الجاريةُ والمنتهيةُ في قائمةٍ واحدةٍ يفصلهما مربّعُ اختيار، **ومربّعٌ
   * افتراضيٌّ ينازع قائمةَ الحالة**: من اختار «مُسلَّم» خرجت قائمتُه فارغةً
   * ولا كلمةَ تقول لماذا.
   *
   * وأعمقُ من ذلك: **هما سؤالان مختلفان.** «ما الذي يحتاجني الآن؟» شاشةُ عمل
   * تُفتح كلَّ دقيقة، **و«ماذا جرى لهذا الطلب؟» شاشةُ بحثٍ تُفتح عند شكوى.**
   * وخلطُهما يجعل الأولى تُقرأ سجلّاً والثانية تُقرأ عملاً.
   *
   * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «يجب أن نفصل بين الطلبات الجديدة والسابقة —
   * شاشةُ طلباتٍ فقط للجديدة، وقسمٌ للطلبات بعد التعامل معها لمتابعة حالتها
   * وإذا حصل خطأٌ أو شكوى».
   *
   * **والبحثُ بالرقم يعبر الشاشتين**: من كتب رقماً يريد ذلك الطلبَ بعينه
   * **حيثما كان** — فلا يُحبَس في نصف القائمة.
   */
  const searching = query.trim() !== "";
  const live = mode === "live";
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [alerts, setAlerts] = useState<Alert[]>([]);
  /**
   * أيُدير المتجرُ طلباته بنفسه؟
   *
   * **`null` تعني «لم نعرف بعد»** لا «المنصة تدير»: لو بدأناها `false` لظهر
   * زرُّ الإرسال لحظةً في كل بطاقة ثم اختفى — **ووميضُ زرٍّ كاذب يُفقد الثقة
   * بكل زرّ**.
   */
  const [selfManage, setSelfManage] = useState<boolean | null>(null);
  /** مهلةُ ظهور زرّ الإسناد اليدويّ — من الإعدادات لا من الشيفرة */
  const [assignAfterMin, setAssignAfterMin] = useState(10);
  /**
   * **كم سائقاً في الدوام الآن** — يُقرأ قبل التحويل لا بعده.
   *
   * التحويلُ يُبلّغ المطعمَ فيبدأ الطبخ، **ثمّ ينزل الطلبُ إلى من يحمله.**
   * فإن لم يكن أحدٌ في الدوام **طُبخ طعامٌ لا حاملَ له** — ويبرد بينما تنتظر
   * العملياتُ سائقاً لا يأتي. **و`null` مجهولٌ لا صفر**: تعذُّرُ القراءة لا
   * يوقف العمل بإنذارٍ كاذب.
   */
  const [onShift, setOnShift] = useState<number | null>(null);
  // **الأدمن فوق قاعدة «عينٌ لا يد»** — تجاوزُ المالك، وكلُّ فعلٍ له مُسجَّل.
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");

  useEffect(() => {
    api<{ settings: { key: string; value: unknown }[] }>("/api/v1/admin/settings")
      .then(({ settings: all }) => {
        const delay = all.find(
          (x) => x.key === "orders.manual_assign_after_min",
        );
        setAssignAfterMin(typeof delay?.value === "number" ? delay.value : 10);
        // **ومن المفتاح الحيّ لا المحذوف.**
        //
        // كان يقرأ `merchants.self_manage_orders` — **وقد صار
        // `platform.orders_mode` بوضعين.** فلم يُوجَد الصفُّ فسقط على `true`،
        // **فظنّت الشاشةُ أنّ المتاجر تدير وهي لا تدير** — فعادت أزرارُ وضعٍ
        // آخر إلى بطاقةٍ صُحّحت مرّاتٍ من قبل.
        //
        // **ومفتاحٌ يُقرأ باسمه القديم لا يصرخ**: لا خطأَ ولا سجلّ، **بل
        // احتياطيٌّ صامتٌ يقلب السلوك.**
        const row = all.find((x) => x.key === "platform.orders_mode");
        setSelfManage(row?.value === "merchants");
      })
      .catch(() => setSelfManage(false));
  }, []);
  const [view, setView] = useViewMode("orders");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        status,
        query,
        // **والبحثُ يعبر الشاشتين** — من كتب رقماً يريده حيثما كان.
        open: live && !searching ? "1" : "",
        closed: !live && !searching ? "1" : "",
        page: String(page),
        per_page: "12",
      });
      setData(await api<OrderPage>(`/api/v1/admin/orders?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
    // **ويُقرأ مع كلّ تحديث** — سائقٌ يفتح دوامَه أو يُغلقه لا يُنتظر تحديثُ صفحة.
    try {
      const res = await api<{ drivers: DriverRow[] } | DriverRow[]>(
        "/api/v1/admin/drivers",
      );
      const list = Array.isArray(res) ? res : res.drivers;
      setOnShift(
        list.filter((x) => x.on_shift && x.status === "active").length,
      );
    } catch {
      setOnShift(null);
    }
  }, [status, query, live, searching, page]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  // البث الحي: تحديثات الطلبات تعيد التحميل فوراً، والتنبيهات تُستبدل مباشرة
  useLiveEvent((event) => {
    if (event.type === "order") void load();
    if (event.type === "alerts") setAlerts((event.alerts as Alert[]) ?? []);
  });
  const liveConnected = useLiveStatus();
  useEffect(() => {
    api<Alert[]>("/api/v1/admin/orders/alerts")
      .then(setAlerts)
      .catch(() => undefined);
  }, []);

  const totalPages = data
    ? Math.max(1, Math.ceil(data.total / data.per_page))
    : 1;

  const columns: DataColumn<OrderRow>[] = [
    {
      // **الرقمُ يميناً والحالةُ يساراً في سطرٍ واحد.**
      //
      // قرارُ المالك (٢٠٢٦-٠٨-٠٤): «الحالةُ فوق بالأعلى يسار رقم الطلب أفضل
      // وأحترافيّ، وتحته الوقتُ والتاريخُ الذي تمّ به الطلب».
      //
      // **وسطرٌ مُعنوَنٌ للحالة كان يُنزلها بين الحقول** — ومن يمسح عشرين
      // بطاقةً يقرأ الحالةَ من طرف عينه، **لا يبحث عنها في قائمةِ حقول.**
      id: "number",
      header: m.admin.ordersPage.number,
      icon: <IconOrder />,
      primary: true,
      cell: (o) => (
        <span className="flex items-center justify-between gap-2">
          <span className="font-bold">#{o.number}</span>
          <span className="inline-flex flex-wrap items-center gap-1">
            <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>
              {STATUS_LABELS[o.status]}
            </Badge>
            {/* **ومن أنهاه بجانب أنّه انتهى** — «ملغي» بلا فاعلٍ ثلاثةُ أخبار. */}
            {o.ended_by && ENDED_BY[o.ended_by] && (
              <Badge variant="neutral">{ENDED_BY[o.ended_by]}</Badge>
            )}
          </span>
        </span>
      ),
      // **وفي الجدول الرقمُ وحدَه** — الحالةُ عمودٌ له رأسُه.
      tableCell: (o) => <span className="font-bold">#{o.number}</span>,
    },
    {
      // **الحالةُ عمودٌ في الجدول وشارةٌ في ترويسة البطاقة.**
      //
      // ولو عُرضت في الوضعين بالتعريف نفسِه **لَظهرت مرّتين في البطاقة**، أو
      // **غاب رأسُها في الجدول فيُقرأ العمودُ بلا اسم.**
      id: "status",
      header: m.admin.ordersPage.statusCol,
      icon: <IconStatus />,
      only: "table",
      cell: (o) => (
        <span className="inline-flex flex-wrap items-center justify-center gap-1">
          <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>
            {STATUS_LABELS[o.status]}
          </Badge>
          {o.ended_by && ENDED_BY[o.ended_by] && (
            <Badge variant="neutral">{ENDED_BY[o.ended_by]}</Badge>
          )}
        </span>
      ),
    },
    {
      id: "customer",
      header: m.admin.ordersPage.customer,
      icon: <IconUser />,
      primary: true,
      // **الاسمُ فوق ورقمُه تحته** — لا في سطرٍ واحد.
      //
      // اسمٌ ورقمٌ متجاوران يطولان فيكسران الخانةَ ويرفعان الصفّ، **ويُقرأ
      // الرقمُ امتداداً للاسم.** ومن يبحث عن رقمٍ يمسح عموداً واحداً بعينه.
      // **وأيقونةُ هاتفٍ على الرقم** — فيُعرف أنّه رقمُه لا رقمَ طلبٍ ولا
      // تاريخاً. **ورقمٌ مجرّدٌ تحت اسمٍ يُقرأ أيَّ رقم**: من نظر إلى البطاقة
      // ورأى `+963935667788` تحت `#1001` وتاريخاً **جمع ثلاثةَ أرقامٍ بلا
      // معنًى يفرّق بينها.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
      cell: (o) => (
        <span className="block">
          <span className="block">{o.customer_name || "—"}</span>
          <span className="flex items-center gap-1 text-xs text-ink-muted">
            <IconPhone size={12} className="shrink-0" />
            <span dir="ltr">{o.customer_phone}</span>
          </span>
        </span>
      ),
    },
    {
      // **التاريخُ والوقتُ في الترويسة تحت الحالة** — لا حقلاً مُعنوَناً بينها.
      //
      // **والتاريخُ معه لا الوقتُ وحدَه**: سجلُّ الطلبات يمتدّ أياماً، **و«٣:١١»
      // بلا يومٍ لا تقول شيئاً** لمن يراجع شكوى الأسبوع الماضي.
      id: "time",
      header: m.admin.ordersPage.time,
      primary: true,
      cell: (o) => (
        <span dir="ltr" className="block">
          {fmtDateTime(o.created_at)}
        </span>
      ),
    },
    {
      id: "merchant",
      header: m.admin.ordersPage.merchant,
      icon: <IconStore />,
      cell: (o) => o.merchant_name,
    },
    {
      // **السائقُ وأجرُه قبل الفاتورة** — قرارُ المالك (٢٠٢٦-٠٨-٠٤): «اسمُ
      // المتجر · أجرةُ توصيل السائق · ثمّ الفاتورة».
      id: "driver",
      header: m.admin.ordersPage.driver,
      icon: <IconDriver />,
      // **ولا يُعرض قبل أن يبلغ الطلبُ السائقين** — ولا حتى فارغاً.
      // **ومن أُسند له سائقٌ ثمّ انتهى الطلبُ يبقى اسمُه مقروءاً**: من يراجع
      // شكوى يسأل «من أوصله؟» بعد أن أُغلق.
      hide: (o: OrderRow) => !DRIVER_STATUSES.has(o.status),
      cell: (o) =>
        o.driver_name || o.driver_phone ? (
          <span>
            {o.driver_name || o.driver_phone}
            {/* **أجرُ السائق يُقال مع أصله — لا رقماً وحدَه.**

                رقمان بالليرة في بطاقةٍ واحدة (توصيلٌ ١٠٬٠٠٠ وأجرٌ ٥٬٠٠٠)
                **ولا شيءَ يقول إنّ أحدَهما من الآخر** — فيُقرأ الثاني أجرةَ
                توصيلٍ ثانيةً، **ويُظنّ أنّ في المنصة مكانين للرسم يفترقان.**
                (وقع فعلاً ٢٠٢٦-٠٨-٠٤.)

                **والمصدرُ واحدٌ فعلاً**: هذا حصّتُه من ذاك. فيُقال الأصلُ
                بجانبه، **فلا يُبحث عن عطبٍ لا وجود له.** */}
            <span className="block text-xs text-ink-muted">
              {m.admin.ordersPage.driverFee}: {fmtNum(o.driver_fee)}{" "}
              {m.common.currency}
              <span className="opacity-70">
                {" "}
                {m.admin.ordersPage.driverFeeOf.replace("{n}", fmtNum(o.delivery_fee))}
              </span>
            </span>
          </span>
        ) : o.offered_driver_name ? (
          /* **«جارٍ إسناد سائق» وحدَها لا تقول شيئاً.**

             العملياتُ ترى الطلبَ يتأخّر **ولا تعرف على من عُرض** — فلا تعرف
             من يتأخّر، **ولا تستطيع أن تتّصل بمن بيده القرارُ الآن.** */
          <span>
            {o.offered_driver_name}
            <span className="block text-xs text-warning">
              {m.admin.ordersPage.awaitingAccept}
            </span>
          </span>
        ) : (
          /* **ولا يُكرَّر التوصيلُ هنا** — صار في ذيل الفاتورة حيث يُجمع.
           **ورقمٌ يظهر مرّتين يُقرأ مرّتين**، فيُظنّ أنّ ثمّة أجرين. */
          <span className="text-ink-muted">
            {m.admin.ordersPage.noDriverYet}
          </span>
        ),
    },
    {
      // **الفاتورة — أصنافٌ بأسعارها ثمّ إجماليٌّ يشمل التوصيل.**
      //
      // قرارُ المالك (٢٠٢٦-٠٨-٠٤): «الفاتورة، تحتها الأصناف كلُّ صنفٍ بسطر
      // وتحته إضافاتُه، والسعرُ على اليسار مفصَّلاً، **ثمّ السعرُ الإجماليّ
      // للفاتورة مع سعر التوصيل**».
      //
      // **و«قيمةُ الطلب» حُذفت** — «لا فائدةَ منها أساساً»: رقمٌ بين السطور
      // والإجمالي **لا يُسأل عنه أحد**، ومجموعُ السطور يقوله لمن أراده.
      id: "items",
      header: m.admin.ordersPage.invoice,
      icon: <IconOrder />,
      // بعرض البطاقة: قائمةٌ تُقرأ سطراً سطراً لا تُحشَر في خانةٍ ضيّقة
      block: true,
      // **وفي الجدول زرٌّ لا قائمة.**
      //
      // فاتورةٌ من عشرة سطورٍ تُفسد صفَّ جدول: **ترتفع الصفوفُ وتتباين
      // أطوالُها فيُقرأ الجدولُ عشوائياً.** والزرُّ يفتحها حين تُطلب.
      tableCell: (o) => <InvoiceButton order={o} />,
      cell: (o) => <InvoiceList o={o} />,
    },
    {
      // **السائق وأجرُه — أو أجرةُ التوصيل قبل أن يُسنَد أحد.**
      // الطلبُ يولد بلا سائق، والخانةُ الفارغة لا تقول شيئاً: فيُعرض ما يُدفع
      // عن التوصيل حتى يُعرف من سيقبضه.
      // **إثباتُ التسليم — وهو موضعُ الحسم في كلّ نزاعٍ على «لم يصلني».**
      //
      // **والمسافةُ أهمُّ من الصورة**: صورةُ بابٍ قد تكون لأيّ باب، **وصورةٌ
      // على بُعد أمتارٍ من عنوان الزبون بيّنة.** ولذلك تُعرض بالرقم لا بحكم.
      //
      // **ولا يُخفى تعذّرُ الصورة**: كلمةُ السائق تُقرأ يومَ النزاع، **ومن
      // تخطّى عشراً يُقرأ ذلك في صفّه.**
      // **ولا يُعرض قبل أن يكون له معنى.**
      //
      // إثباتُ التسليم لا يوجد إلّا بعد أن يسلّم السائقُ أو يتعذّر عليه.
      // **وسطرٌ فارغٌ في طلبٍ لم يُقبل بعد يُسأل عنه ولا جواب** — «كيف والطلبُ
      // لسّا ما وافقنا عليه أساساً؟» (المالك ٢٠٢٦-٠٨-٠٤).
      //
      // **وملاحظةُ الزبون تبقى** — كُتبت لحظةَ الطلب وتُقرأ من أوّله.
      id: "proof",
      header: m.admin.ordersPage.proof,
      icon: <IconCamera />,
      block: true,
      hide: (o: OrderRow) => !PROOF_STATUSES.has(o.status),
      cell: (o) => {
        if (o.proof_skip_reason) {
          return (
            <span className="text-xs text-warning">
              {m.admin.ordersPage.proofSkipped} {o.proof_skip_reason}
            </span>
          );
        }
        if (!o.proof_url)
          return <span className="text-xs text-ink-muted">—</span>;
        const noGps = (o.proof_meters ?? -1) < 0;
        const away = Math.round(o.proof_meters ?? 0);
        return (
          <span className="flex flex-wrap items-center gap-2">
            <a href={o.proof_url} target="_blank" rel="noreferrer">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={o.proof_url}
                alt=""
                className="h-16 w-16 rounded-control border border-line object-cover"
              />
            </a>
            <span className="flex flex-col gap-1">
              {noGps ? (
                <Badge variant="neutral">{m.admin.ordersPage.proofNoGps}</Badge>
              ) : (
                <Badge variant={away <= 150 ? "success" : "warning"}>
                  {m.admin.ordersPage.proofMeters.replace("{n}", fmtNum(away))}
                </Badge>
              )}
              {o.proof_taken_at && (
                <span className="text-2xs text-ink-muted" dir="ltr">
                  {fmtDateTime(o.proof_taken_at)}
                </span>
              )}
            </span>
          </span>
        );
      },
    },
    {
      id: "note",
      header: m.admin.ordersPage.customerNote,
      icon: <IconNote />,
      block: true,
      cell: (o) =>
        o.notes ? (
          <span className="text-xs text-accent-dark">{o.notes}</span>
        ) : (
          <span className="text-ink-muted">—</span>
        ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconOrder className="text-primary" />
          {live ? m.admin.ordersPage.title : m.admin.ordersPage.historyTitle}
          <span
            className={`flex items-center gap-1.5 rounded-badge px-2.5 py-1 text-xs font-medium ${
              liveConnected
                ? "bg-success/10 text-success"
                : "bg-danger/10 text-danger"
            }`}
          >
            <span
              className={`h-2 w-2 rounded-badge ${liveConnected ? "animate-pulse bg-success" : "bg-danger"}`}
            />
            {liveConnected
              ? m.admin.ordersPage.live
              : m.admin.ordersPage.liveOff}
          </span>
        </h1>
      </div>

      {/* تنبيهات التصعيد */}
      {alerts.length > 0 && (
        <div className="mb-4 rounded-card border-2 border-danger/50 bg-danger/5 p-4">
          <p className="mb-2 flex items-center gap-2 font-bold text-danger">
            <span className="h-2.5 w-2.5 animate-pulse rounded-badge bg-danger" />
            {m.admin.ordersPage.alertsTitle} ({alerts.length})
          </p>
          <ul className="space-y-1.5">
            {alerts.map((a) => (
              <li
                key={a.order_id + a.reason}
                className="flex flex-wrap items-center gap-2 text-sm"
              >
                {/* الإنذارُ يجلب طلبَه إلى القائمة بدل أن يفتح نافذة:
                    **البطاقة نفسها صارت تحمل كل ما يُقرَّر به** — والنافذة
                    كانت تُخفي بقيّة الطلبات وهي مفتوحة. */}
                <button
                  onClick={() => {
                    // **والبحثُ بالرقم يعبر الشاشتين** — فلا يحتاج إلى إطفاء
                    // مُرشِّحٍ لم يعد له وجود.
                    setQuery(String(a.number));
                    setPage(1);
                  }}
                  className="font-bold text-danger underline-offset-2 hover:underline"
                >
                  #{a.number}
                </button>
                <Badge variant="danger">
                  {m.admin.ordersPage.alertReasons[a.reason]}
                </Badge>
                <span>{a.merchant_name}</span>
                <span dir="ltr" className="text-xs text-ink-muted">
                  {a.customer_phone}
                </span>
                <span className="text-xs text-ink-muted">
                  {m.admin.ordersPage.sinceMinutes.replace(
                    "{m}",
                    String(a.minutes),
                  )}
                </span>
                <Badge variant="warning">{STATUS_LABELS[a.status]}</Badge>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.ordersPage.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        {/* **وقائمةُ الحالة في السجلّ وحدَه.**

            في شاشة العمل لا معنى لها: الحالاتُ الجاريةُ كلُّها معروضةٌ أصلاً،
            **واختيارُ «مُسلَّم» فيها يُخرج فراغاً** — وهو التناقضُ الذي كان
            يقع بين المربّع والقائمة. **وشاشةٌ بغرضٍ واحدٍ لا تحتاج مُرشِّحاً
            يناقضها.** */}
        {!live && (
          <div className="w-44">
            <Select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{m.admin.ordersPage.allStatuses}</option>
              {[...CLOSED_STATUSES].map((k) => (
                <option key={k} value={k}>
                  {STATUS_LABELS[k] ?? k}
                </option>
              ))}
            </Select>
          </div>
        )}
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">
          {error}
        </p>
      )}

      <DataView
        items={data?.orders ?? []}
        getKey={(o) => o.id}
        columns={columns}
        view={view}
        empty={m.admin.ordersPage.empty}
        actions={(o) => (
          <OrderActions
            o={o}
            onChanged={load}
            selfManage={selfManage !== false}
            isAdmin={isAdmin}
            assignAfterMin={assignAfterMin}
            onShift={onShift}
          />
        )}
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>
            {m.admin.users.totalCount.replace("{count}", String(data.total))}
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------- تفاصيل الطلب ----------

/**
 * أزرارُ الفعل على البطاقة — لا خلف «التفاصيل».
 *
 * كانت كلُّ حركةٍ تكلّف: فتحُ النافذة ← انتظارُ تحميلها ← الفعل ← الإغلاق ←
 * **البحث عن موضعك في القائمة من جديد**. وفي ساعة ذروةٍ فيها عشرون طلباً هذه
 * عشرون رحلةَ ذهابٍ وإياب — والنافذةُ تُخفي بقيّة الطلبات وهي مفتوحة، فيعمل
 * الموظّف أعمى عمّا يجري.
 *
 * **والهدّامةُ لا تُنفَّذ بضغطةٍ واحدة**: رفضٌ أو إلغاءٌ في صفٍّ مزدحم يُتلف
 * طلبَ زبونٍ بإصبعٍ زلّ. فتُطلب ضغطةٌ ثانية تؤكّد — **تأكيدٌ في مكانه أخفُّ من
 * نافذةٍ تُفتح وتُغلق**، وأصدقُ من ثقةٍ في دقّة الإصبع.
 *
 * **وإسنادُ السائق يبقى في التفاصيل**: يحتاج قائمةَ اختيارٍ لا زرّاً.
 */
function OrderActions({
  o,
  onChanged,
  selfManage,
  isAdmin,
  assignAfterMin,
  onShift,
}: {
  o: OrderRow;
  onChanged: () => void;
  /** حين تكون `false` تُدير المنصةُ الطلبات وتُرسلها للمتجر على واتساب */
  selfManage: boolean;
  /** الأدمن فوق القاعدة — تجاوزُ المالك، وهو مُسجَّل */
  isAdmin: boolean;
  /** كم دقيقةً ينتظر الطابورُ قبل أن يظهر الإسنادُ اليدويّ */
  assignAfterMin: number;
  /** كم سائقاً في الدوام — **و`null` مجهولٌ لا صفر** */
  onShift: number | null;
}) {
  const [busy, setBusy] = useState("");
  /** خبرٌ يُقال بعد فعلٍ نجح — **وكم صُحِّح** في إعادة الحساب. */
  const [notice, setNotice] = useState("");
  /** الفعلُ الهدّام المفتوح الآن — يُطلب سببُه قبل تنفيذه */
  const [asking, setAsking] = useState("");
  /** نافذةُ التحويل إلى متجرٍ آخر — القاعدةُ الاحتياطية. */
  const [transferring, setTransferring] = useState(false);
  /** تنبيهُ «لا سائقَ في الدوام» — يُقال قبل التحويل لا بعده. */
  const [noDriverWarn, setNoDriverWarn] = useState(false);
  const [stores, setStores] = useState<{ id: string; name: string }[]>([]);
  const [target, setTarget] = useState("");
  const [unmatched, setUnmatched] = useState<string[]>([]);
  /** قائمةُ السائقين مفتوحةٌ للإسناد اليدوي */
  const [assigning, setAssigning] = useState(false);
  const [drivers, setDrivers] = useState<DriverRow[]>([]);
  /** تفصيلُ توزيع المال — يُجلب عند الطلب لا مع كل بطاقة */
  const [split, setSplit] = useState<Breakdown | null>(null);
  /** نموذجُ تعويض السائق عن طلبٍ فشل */
  const [compensating, setCompensating] = useState(false);
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");

  // **ما تملكه العملياتُ بعد حساب الوضع** — لا الخريطةُ الخام.
  const next = opsNext(o.status, selfManage, o.driver_name !== null, isAdmin);

  /**
   * أمضت المهلةُ في الطابور بلا التقاط؟
   *
   * **ويُقاس من `dispatched_at` لا من `created_at`**: طلبٌ قُبل بعد ربع ساعةٍ
   * من إنشائه لم ينتظر سائقاً تلك الربعَ — **وقياسٌ من أوّل الطلب يُظهر
   * الزرَّ قبل أن يبدأ الانتظار أصلاً.**
   */
  // **والمهلةُ تسري على المالك كما تسري على موظّفه.**
  //
  // كان `isAdmin` يتخطّاها — **وزرٌّ متاحٌ دائماً يُستعمل دائماً**، فيصير
  // الإسنادُ اليدويُّ هو الأصلَ وترتيبُ السائقين زينة. وهي علّةُ `G-09` نفسُها
  // التي أُصلحت للعمليات **وبقيت للمالك**، وهو أكثرُ من يفتح اللوحة.
  //
  // **والاحتياطُ يبقى**: المهلةُ في الإعدادات — من أرادها دقيقةً جعلها دقيقة.
  const assignReady =
    !!o.dispatched_at &&
    Date.now() - new Date(o.dispatched_at).getTime() >= assignAfterMin * 60_000;

  // **الإسنادُ اليدوي مخرجٌ لا طريق.** السائقون يلتقطون من الطابور بأنفسهم
  // (تطبيق :3005)، وهذا لمن لم يلتقطه أحد. ولذلك يُجلب السائقون **عند فتح
  // القائمة** لا مع كل بطاقة: عشرون بطاقةً تعني عشرين نداءً لقائمةٍ واحدة.
  //
  // ولا يُعرض إلا **من هو على الدوام**: إسنادُ طلبٍ إلى منصرفٍ يُخفيه عن
  // الطابور ولا يوصله أحد.
  const canAssign = o.status === "preparing" || o.status === "dispatching";

  async function openAssign() {
    setAssigning(true);
    try {
      const res = await api<{ drivers: DriverRow[] } | DriverRow[]>(
        "/api/v1/admin/drivers",
      );
      const list = Array.isArray(res) ? res : res.drivers;
      setDrivers(list.filter((x) => x.on_shift && x.status === "active"));
    } catch {
      setDrivers([]);
    }
  }

  // **ما يُرسَل باسم المنصة يُقرأ قبل أن يُرسَل.** والنصُّ والرابطُ من الخادم
  // لا من هنا: لو رُكّبا في الواجهة لأمكن أن يفترقا عمّا يصل المتجر، ولا
  // يكتشفه أحدٌ حتى يشتكي متجر.
  /**
   * **التحويلُ إلى المتجر — ضغطةٌ واحدة.**
   *
   * كانت ضغطتين: معاينةٌ ثم فتحُ واتساب. **وحجّةُ المعاينة أن ما يُرسَل باسم
   * المنصة يُقرأ قبل أن يُرسَل — وواتسابُ يعرضه في صندوق الكتابة قبل الإرسال.**
   * فكانت شاشتُنا **ثالثةَ موضعٍ يعرض النصَّ نفسه**، وخطوةً تُضغط بلا خبرٍ جديد.
   *
   * **والنافذةُ تُفتح فارغةً قبل الشبكة ثم تُوجَّه.**
   *
   * المتصفّحُ يمنع فتحَ نافذةٍ لا تنشأ عن ضغطةٍ مباشرة، و`await` قبل الفتح
   * يقطع هذا النسب فتُحجب. فتُفتح فارغةً في اللحظة نفسها — **وهي ابنةُ
   * الضغطة** — ثم يُنقل عنوانُها حين يصل الرابط.
   */
  /**
   * **ولا يُحوَّل طلبٌ ولا سائقَ في الدوام قبل أن يُقال ذلك.**
   *
   * التحويلُ يُبلّغ المطعمَ **فيبدأ الطبخ**، ثمّ ينزل الطلبُ إلى من يحمله.
   * فإن لم يكن أحدٌ **طُبخ طعامٌ لا حاملَ له** — ويبرد بينما تنتظر العملياتُ
   * سائقاً لا يأتي، **ويدفع المتجرُ والزبونُ ثمنَ خبرٍ لم يُقل.**
   *
   * **ولا يُمنع**: قد يفتح سائقٌ دوامَه بعد دقيقة، **والمنعُ يقرّر عن المالك ما
   * لا يعرفه.** يُقال له الرقمُ ويقرّر هو. (قرارُ المالك ٢٠٢٦-٠٨-٠٣)
   */
  function askThenForward() {
    if (onShift === 0) {
      setNoDriverWarn(true);
      return;
    }
    void forwardToMerchant();
  }

  async function forwardToMerchant() {
    setNoDriverWarn(false);
    setErr("");
    const win = window.open("", "_blank");
    setBusy("wa");
    try {
      const msg = await api<MerchantMessage>(
        `/api/v1/admin/orders/${o.id}/message`,
      );
      if (!msg.wa_link) {
        win?.close();
        setErr(m.admin.ordersPage.noWhatsApp);
        return;
      }
      if (win) win.location.href = msg.wa_link;
      // **الوسمُ يقع ولو حُجبت النافذة**: الموظّفُ يفتحها بنفسه، **والطلبُ
      // لا يبقى معلّقاً لأن متصفّحاً تشدّد.**
      await api(`/api/v1/admin/orders/${o.id}/whatsapp`, {
        method: "POST",
        body: JSON.stringify({ channel: "whatsapp" }),
      });
      if (!win) setErr(m.admin.ordersPage.popupBlocked);
      onChanged();
    } catch (e) {
      win?.close();
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  /**
   * إعادةُ حساب التسوية — **وتقول كم صُحِّح.**
   *
   * **ولا تقول «تمّ» عن نداءٍ لم يُغيّر شيئاً**: من ضغط ورأى «تمّ» ظنّ أنّ خللاً
   * صُلح وهو لم يكن. **وصفرٌ يُقال صراحةً أصدقُ من نجاحٍ مبهم.**
   */
  async function recompute() {
    setBusy("recompute");
    setErr("");
    try {
      const res = await api<{ delta: number }>(
        `/api/v1/admin/orders/${o.id}/recompute`,
        {
          method: "POST",
        },
      );
      setNotice(
        res.delta === 0
          ? m.admin.ordersPage.recomputeNone
          : m.admin.ordersPage.recomputeDone.replace("{n}", fmtNum(res.delta)),
      );
      onChanged();
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  async function openSplit() {
    setErr("");
    try {
      setSplit(await api<Breakdown>(`/api/v1/admin/orders/${o.id}/breakdown`));
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    }
  }

  // **مصيرُ البضاعة** — من يحمل ثمنَ طعامٍ طُبخ ولم يُسلَّم.
  async function settleGoods(to: "merchant" | "platform") {
    setBusy("goods");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/settle-goods`, {
        method: "POST",
        body: JSON.stringify({ to }),
      });
      onChanged();
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  // **تعويضُ السائق — بمبلغٍ يقدّره إنسان.**
  //
  // أجرُ التوصيل مقابل تسليمٍ تمّ، وما وقع رحلةٌ لا تسليم. وتقديرُ الرحلة
  // يختلف: مشوارٌ إلى الحيّ المجاور ليس كمشوارٍ عبر المدينة، **ورقمٌ آليٌّ
  // واحد يظلم أحدهما.**
  async function compensate() {
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0 || reason.trim() === "") return;
    setBusy("compensate");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/compensate-driver`, {
        method: "POST",
        body: JSON.stringify({
          amount: Math.round(value),
          note: reason.trim(),
        }),
      });
      setCompensating(false);
      setAmount("");
      setReason("");
      onChanged();
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  async function assign(driverID: string) {
    setBusy("assign");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/assign`, {
        method: "POST",
        body: JSON.stringify({ driver_id: driverID, note: "" }),
      });
      setAssigning(false);
      onChanged();
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  /**
   * **التحويلُ إلى متجرٍ آخر** — حين يعتذر المتجرُ على الواتساب.
   *
   * **وسعرُ الزبون لا يُمسّ**: الفرقُ على المنصة ولو كان الجديدُ أغلى. والزبونُ
   * **لا يعلم أنّ مصدراً تبدّل** — وفاتورةٌ تتغيّر بعد الطلب تنقض ذلك في سطر.
   *
   * **وصنفٌ لا يُطابق يُوقف التحويلَ ويُسمّى** — لا يُخمَّن «الأقرب».
   */
  async function transfer() {
    if (!target || !reason.trim()) return;
    setBusy("transfer");
    setErr("");
    setUnmatched([]);
    try {
      await api(`/api/v1/admin/orders/${o.id}/transfer`, {
        method: "POST",
        body: JSON.stringify({ merchant_id: target, note: reason.trim() }),
      });
      setTransferring(false);
      setReason("");
      setTarget("");
      onChanged();
    } catch (e) {
      if (e instanceof ApiError) {
        const items = (e.body as { items?: string[] }).items;
        if (items?.length) setUnmatched(items);
        else setErr(translateKey(e.body.message_key));
      } else setErr(m.errors.internal);
    } finally {
      setBusy("");
    }
  }

  async function go(to: string, note: string, manualOverride = false) {
    setBusy(to);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to, note, manual_override: manualOverride }),
      });
      setAsking("");
      setReason("");
      onChanged();
    } catch (e) {
      setErr(
        e instanceof ApiError
          ? translateKey(e.body.message_key)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  // **السببُ بدل الضغطتين العمياوين.**
  //
  // كانت الضغطةُ الثانية تحرس من الإصبع الزالّ وحده. والسببُ يحرس منه **ويُبقي
  // أثراً**: هو ما يُقال للزبون، وما يُقاس به متجرٌ يُكثر الرفض أو موظّفٌ يُكثر
  // الإلغاء. **وطلبٌ يُلغى بلا كلمة يترك الجميع يخمّنون.**
  if (split !== null) {
    const OP = m.admin.ordersPage;
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">{OP.splitTitle}</p>
        <div className="space-y-1 rounded-control bg-page p-3 text-xs">
          <Row label={OP.splitPaid} value={split.total} strong />
          {split.lines
            .filter((l) => l.party !== "customer")
            .map((l, i) => (
              <Row
                key={i}
                label={`${OP.party[l.party]} — ${l.name}`}
                value={l.amount}
              />
            ))}
          {/* **الفارقُ يُعرض ولا يُخفى.** لو ظهر رقمٌ هنا فمالٌ تحرّك بلا طرفٍ
              معروف — وهو أوّلُ ما يُسأل عنه، لا آخرُ ما يُكتشف. */}
          {split.total - split.to_parties - split.platform !== 0 && (
            <Row
              label={OP.splitUnaccounted}
              value={split.total - split.to_parties - split.platform}
              danger
            />
          )}
        </div>
        {err && <p className="text-xs text-danger">{err}</p>}
        <Button variant="secondary" onClick={() => setSplit(null)}>
          {m.common.back}
        </Button>
      </div>
    );
  }

  if (compensating) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">
          {m.admin.ordersPage.compensateTitle}
        </p>
        <Input
          type="number"
          inputMode="numeric"
          placeholder={m.admin.ordersPage.compensateAmount}
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input
          placeholder={m.admin.ordersPage.compensateReason}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex gap-2">
          <Button disabled={busy !== ""} onClick={() => void compensate()}>
            {m.common.confirm}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setCompensating(false);
              setErr("");
            }}
          >
            {m.common.cancel}
          </Button>
        </div>
      </div>
    );
  }

  if (assigning) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">{m.admin.ordersPage.chooseDriver}</p>
        {drivers.length === 0 ? (
          <p className="text-xs text-ink-muted">
            {m.admin.ordersPage.noDriversOnShift}
          </p>
        ) : (
          <div className="flex flex-col gap-1.5">
            {drivers.map((dv) => (
              <Button
                key={dv.id}
                variant="secondary"
                disabled={busy !== ""}
                onClick={() => void assign(dv.id)}
              >
                {dv.full_name || dv.phone}
                {dv.open_orders > 0 && ` (${fmtNum(dv.open_orders)})`}
              </Button>
            ))}
          </div>
        )}
        {err && <p className="text-xs text-danger">{err}</p>}
        <Button variant="secondary" onClick={() => setAssigning(false)}>
          {m.common.cancel}
        </Button>
      </div>
    );
  }

  if (transferring) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium text-primary-dark">
          {m.admin.ordersPage.transferTitle}
        </p>
        <p className="text-xs text-ink-muted">
          {m.admin.ordersPage.transferHint}
        </p>
        <Select
          id={`t-${o.id}`}
          value={target}
          onChange={(e) => {
            setTarget(e.target.value);
            setUnmatched([]);
          }}
        >
          <option value="">{m.admin.ordersPage.transferPick}</option>
          {stores.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </Select>
        <Input
          id={`tr-${o.id}`}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={m.admin.ordersPage.transferReason}
        />
        {/* **وصنفٌ لا يُطابق يُسمّى** — «لا يُطابق» وحدَها تترك الموظّفَ
            يفتح قائمتين ويقارن بعينه. */}
        {unmatched.length > 0 && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-xs text-danger">
            {m.admin.ordersPage.transferUnmatched} {unmatched.join(" · ")}
          </p>
        )}
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex gap-2">
          <Button
            disabled={!target || !reason.trim() || busy !== ""}
            onClick={() => void transfer()}
          >
            {m.admin.ordersPage.transferConfirm}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setTransferring(false);
              setReason("");
              setTarget("");
              setUnmatched([]);
              setErr("");
            }}
          >
            {m.common.cancel}
          </Button>
        </div>
      </div>
    );
  }

  if (asking) {
    // **والنافذةُ نفسُها تخدم التوقيع** — سبباً إلزامياً في الحالين.
    //
    // **ونصُّها يختلف**: الإلغاءُ يسأل «لماذا ألغيت»، والتدخّلُ يقول **«تُسجَّل
    // باسمك أنّ المنصة أعلنتها — لا أنّ السائق قالها»**. ونصٌّ واحدٌ لمعنيين
    // يجعل من يوقّع لا يعرف ما وقّع عليه.
    const manual = asking.startsWith("manual:");
    const target = manual ? asking.slice(7) : asking;
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p
          className={`text-xs font-medium ${manual ? "text-warning" : "text-danger"}`}
        >
          {manual
            ? m.admin.ordersPage.manualTitle
            : m.admin.ordersPage.reasonTitle.replace(
                "{action}",
                ACTION_LABELS[asking] ?? asking,
              )}
        </p>
        <Input
          id={`reason-${o.id}`}
          autoFocus
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setErr("");
          }}
          placeholder={m.admin.ordersPage.reasonPlaceholder}
        />
        <p className="text-xs text-ink-muted">
          {manual
            ? m.admin.ordersPage.manualHint
            : m.admin.ordersPage.reasonHint}
        </p>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex gap-2">
          <Button
            variant={manual ? "secondary" : "danger"}
            disabled={!reason.trim() || busy !== ""}
            onClick={() => void go(target, reason.trim(), manual)}
          >
            {manual
              ? m.admin.ordersPage.manualConfirm
              : m.admin.ordersPage.confirm}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setAsking("");
              setReason("");
              setErr("");
            }}
          >
            {m.common.cancel}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <>
      {next.map((to) => {
        const destructive = DESTRUCTIVE.has(to);
        return (
          <Button
            key={to}
            variant={destructive ? "danger" : "primary"}
            disabled={busy !== ""}
            onClick={() => (destructive ? setAsking(to) : void go(to, ""))}
          >
            {ACTION_LABELS[to] ?? to}
          </Button>
        );
      })}
      {/* **توزيعُ المال — لمن أُغلق أمرُه.**

          لا يُعرض قبل الإغلاق: طلبٌ في الطريق لم تُقيَّد أنصبتُه بعد، **وشاشةٌ
          تعرض أصفاراً تُقرأ خطأً لا نقصاً.** */}
      {(o.status === "delivered" ||
        o.status === "failed" ||
        o.status === "refunded") && (
        <Button
          variant="secondary"
          disabled={busy !== ""}
          onClick={() => void openSplit()}
        >
          {m.admin.ordersPage.splitButton}
        </Button>
      )}
      {/* **وتصحيحُ تسويةٍ قديمة — بقيدٍ مقابلٍ لا بتصفير.**

          حسبةُ الخزينة تُصحّح نفسَها بمجرّد أن تُنادى، **ولا شيءَ ينادِيها على
          طلبٍ أُغلق.** فإن كُشف خللٌ في المعادلة — كما وقع في `#1003` — بقي
          القيدُ الخاطئ ولو أُصلح الكود، **ولم يبقَ إلّا تصفيرُ البيانات كلِّها.**

          **والأدمنُ وحدَه**: قيدٌ ماليٌّ يُنشأ بيد. */}
      {isAdmin && SETTLED_STATUSES.has(o.status) && (
        <Button
          variant="ghost"
          disabled={busy !== ""}
          title={m.admin.ordersPage.recomputeHint}
          onClick={() => void recompute()}
        >
          {m.admin.ordersPage.recompute}
        </Button>
      )}
      {notice && (
        <span className="rounded-control bg-success/10 px-2.5 py-1 text-xs text-success">
          {notice}
        </span>
      )}

      {/* **ما بعد الفشل — سؤالان لا يُجيبهما النظام وحده.**

          طلبٌ فشل يترك طعاماً مطبوخاً ورحلةً مقطوعة. والقيدُ المالي لا يقع
          تلقائياً لأنّ الجواب ليس في القاعدة: **أاستردّ المتجرُ بضاعته؟ وكم
          يستحقّ السائقُ عن مشوارٍ لم يُثمر؟** يجيبهما من رأى، لا من حسب. */}
      {o.status === "failed" && (
        <>
          {o.goods_settled_to === null ? (
            <>
              <Button
                variant="secondary"
                disabled={busy !== ""}
                onClick={() => void settleGoods("merchant")}
              >
                {m.admin.ordersPage.goodsToMerchant}
              </Button>
              <Button
                variant="secondary"
                disabled={busy !== ""}
                onClick={() => void settleGoods("platform")}
              >
                {m.admin.ordersPage.goodsToPlatform}
              </Button>
            </>
          ) : (
            <span className="text-xs text-ink-muted">
              {o.goods_settled_to === "merchant"
                ? m.admin.ordersPage.goodsSettledMerchant
                : m.admin.ordersPage.goodsSettledPlatform}
            </span>
          )}
          {o.driver_name && (
            <Button
              variant="secondary"
              disabled={busy !== ""}
              onClick={() => {
                setAmount("");
                setReason("");
                setCompensating(true);
              }}
            >
              {m.admin.ordersPage.compensateDriver}
            </Button>
          )}
        </>
      )}

      {/* **التحويلُ إلى المتجر — في الوضعين لا في وضعٍ واحد.**

          في «المنصة تدير» هو الفعلُ الأساسيّ: تقبل العملياتُ نيابةً عن المتجر
          ثم تُحوّل إليه الطلب، ولا تعلن بدء تحضيرٍ لم تبدأه.

          وفي «المتجر يدير» يبقى بيدها: **المطعمُ لا يردّ على بوّابته أحياناً**
          — والطلبُ الذي في الشاشة لا يُطبخ حتى يراه أحد. فتُحوّله على واتساب
          حيث يعمل صاحبُه أصلاً، وهو ثانويٌّ هنا لا أساسيّ.

          والنافذةُ حتى `preparing`: بعد بدء الطبخ لم يعد المطبخُ يحتاج خبراً. */}
      {(o.status === "accepted" || o.status === "preparing") && (
        <Button
          variant={
            o.sent_to_merchant_at || selfManage ? "secondary" : "primary"
          }
          disabled={busy !== ""}
          onClick={askThenForward}
        >
          {/* **والأيقونةُ تقول القناة قبل الكلمة.**

              «تحويلٌ إلى المتجر» لا تقول كيف — **ومن ضغطها أوّلَ مرّةٍ فوجئ
              بنافذة واتساب تُفتح.** والرمزُ يقولها في نظرة. */}
          <span className="flex items-center gap-1.5">
            <IconWhatsApp size={15} />
            {o.sent_to_merchant_at
              ? m.admin.ordersPage.sentWhatsApp
              : m.admin.ordersPage.sendWhatsApp}
          </span>
        </Button>
      )}

      <Modal
        open={noDriverWarn}
        onClose={() => setNoDriverWarn(false)}
        title={m.admin.ordersPage.noDriverTitle}
      >
        <p className="mb-4 text-sm text-ink-muted">
          {m.admin.ordersPage.noDriverBody}
        </p>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setNoDriverWarn(false)}>
            {m.admin.ordersPage.noDriverWait}
          </Button>
          {/* **والتحويلُ يبقى متاحاً** — المنعُ يقرّر عن المالك ما لا يعرفه. */}
          <Button variant="primary" onClick={() => void forwardToMerchant()}>
            {m.admin.ordersPage.noDriverAnyway}
          </Button>
        </div>
      </Modal>
      {/* **الإسنادُ اليدويّ احتياطٌ لا أصل.**

          **وزرٌّ متاحٌ دائماً يُستعمل دائماً** — فيصير هو الطريقَ ويصير ترتيبُ
          السائقين زينة. فلا يظهر إلّا حين **يعجز الطابورُ**: مضت المهلةُ ولم
          يلتقطه أحد.

          والأدمنُ يراه دائماً — تجاوزُ المالك. */}
      {/* **ولماذا لا يلتقطه أحد — يُقال هنا لا في تنبيهٍ بعد عشر دقائق.**

          طلبٌ فوق سقف النقد لا يظهر لسائقٍ أبداً، **والعملياتُ ترى «جارٍ إسناد
          سائق» وتنتظر من لن يأتي.** والصمتُ أسوأُ من الرفض: الرفضُ يُقرأ
          ويُعالَج، **والصمتُ يُنتظَر.** */}
      {o.blocked_reason && (
        <p className="w-full rounded-control bg-warning/10 px-3 py-2 text-xs font-medium text-warning">
          {o.blocked_reason}
        </p>
      )}

      {/* **التدخّلُ اليدويّ — منفصلٌ عن الأزرار العادية.**

          مراحلُ الطريق بيد السائق، **والمنصةُ لا تعلم أنّه وصل.** لكنّ هاتفاً
          نفدت بطاريتُه يترك الطلبَ عالقاً بلا من يُكمله — **وقد يكون المالكُ
          وحدَه من يدير.**

          **والمشكلةُ لم تكن «من ضغط» بل «ماذا يقول السجلّ»**: بالتوقيع يصدق
          السجلُّ فيقول «أعلنتها المنصة». **ويُفصل عن بقيّة الأزرار كي لا
          يُضغط سهواً** كما وقع في `#1004`. */}
      {isAdmin && MANUAL_STEP_FROM.has(o.status) && (
        <button
          type="button"
          disabled={busy !== ""}
          onClick={() => setAsking("manual:" + (NEXT_AFTER[o.status] ?? ""))}
          className="w-full rounded-control border border-dashed border-warning/60 px-3 py-1.5 text-xs text-warning hover:bg-warning/5"
        >
          {m.admin.ordersPage.manualStep.replace(
            "{s}",
            m.orders.status[
              (NEXT_AFTER[o.status] ?? "") as keyof typeof m.orders.status
            ] ?? "",
          )}
        </button>
      )}

      {/* **التحويلُ إلى متجرٍ آخر — قاعدةٌ احتياطية.**

          سياسةُ الإخفاء تمنع الحاجةَ إليه: صنفٌ غيرُ متاحٍ لا يُعرض. **لكنّ
          الطارئَ يبقى** — والمتجرُ قد يعتذر على الواتساب.

          **ولا يظهر بعد خروج البضاعة**: المتجرُ الأوّلُ قبض عند الاستلام،
          فتحويلٌ بعدها **طلبان لا واحد.** */}
      {/* **ولا يُعرض في وضع «المنصة تدير»** — لا رفضَ من متجرٍ فيه أصلاً،
          **فالزرُّ يعرض علاجاً لمرضٍ لا وجودَ له.** ويعود حين يملك المتجرُ أن
          يرفض: عندها يعود الطلبُ إلى المنصة **ولها أن تُنقذه بمتجرٍ ثانٍ.** */}
      {selfManage && TRANSFERABLE.has(o.status) && (
        <Button
          variant="ghost"
          disabled={busy !== ""}
          onClick={() => {
            setTransferring(true);
            if (stores.length === 0) {
              void api<{ id: string; name: string }[]>(
                "/api/v1/admin/merchants",
              )
                .then((r) =>
                  setStores(
                    (Array.isArray(r) ? r : []).filter(
                      (x) => x.id !== o.merchant_id,
                    ),
                  ),
                )
                .catch(() => setStores([]));
            }
          }}
        >
          {/* **وسهمان متبادلان يقولان «تبديل» قبل الكلمة** — وسهمٌ واحدٌ
              يُقرأ «إرسالاً» لا «استبدالاً». */}
          <span className="flex items-center gap-1.5">
            <IconSwap size={15} />
            {m.admin.ordersPage.transfer}
          </span>
        </Button>
      )}

      {canAssign && assignReady && (
        <Button
          variant="secondary"
          disabled={busy !== ""}
          onClick={() => void openAssign()}
        >
          {m.admin.ordersPage.assignHere}
        </Button>
      )}
      {err && <p className="w-full text-xs text-danger">{err}</p>}
    </>
  );
}

/**
 * **الفاتورة — مكوّنٌ واحدٌ للبطاقة وللنافذة.**
 *
 * تُعرض كاملةً في البطاقة، **وفي الجدول خلف زرّ**: فاتورةٌ من عشرة سطورٍ تُفسد
 * صفَّ جدول — **ترتفع الصفوفُ وتتباين أطوالُها فيُقرأ الجدولُ عشوائياً.**
 *
 * **ونسختان تفترقان يوماً** — فتقول البطاقةُ رقماً وتقول النافذةُ غيرَه.
 */
function InvoiceList({ o }: { o: OrderRow }) {
  return (
    <ul className="space-y-1.5">
      {(o.items ?? []).map((it) => {
        /** **الخياراتُ صنفان**: ما لا سعرَ له يُلحق بالاسم، وما له سعرٌ
                يُفرد سطراً. **و«عادي» ليس بنداً في الفاتورة** — هو وصفٌ للصنف،
                **و«جبنة» بند** لأنّها زادت الحساب. */
        const free = (it.options ?? []).filter((x) => !x.price_delta);
        const paid = (it.options ?? []).filter((x) => x.price_delta > 0);
        return (
          <li key={it.id}>
            <span className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-badge bg-primary" />
              <span className="min-w-0 flex-1">
                <span className="font-medium">{it.name}</span>
                {free.length > 0 && (
                  <span className="text-ink-muted">
                    {" "}
                    {free.map((x) => x.name).join(m.common.listSeparator)}
                  </span>
                )}
                <span className="font-bold text-primary-dark">
                  {" "}
                  ×{fmtNum(it.qty)}
                </span>
                {it.note && (
                  <span className="block text-xs text-accent-dark">
                    <IconEdit size={11} className="inline align-[-1px]" />{" "}
                    {it.note}
                  </span>
                )}
              </span>
              {/* **سعرُ الصنف عارياً × الكمّية** — والإضافاتُ تحته بأسعارها،
               **فمجموعُ السطور يبلغ قيمةَ الطلب بلا نقصٍ ولا فائض.** */}
              <span dir="ltr" className="shrink-0 tabular-nums">
                {fmtNum(basePrice(it) * it.qty)}
              </span>
            </span>

            {/* **وكلُّ إضافةٍ بسطرها وسعرها** — قرارُ المالك (٢٠٢٦-٠٨-٠٤). */}
            {paid.map((x, i) => (
              <span key={i} className="flex items-center gap-2 ps-4 text-sm">
                <span className="min-w-0 flex-1 text-ink-muted">
                  + {x.name}
                </span>
                <span
                  dir="ltr"
                  className="shrink-0 tabular-nums text-accent-dark"
                >
                  {fmtNum(x.price_delta * it.qty)}
                </span>
              </span>
            ))}
          </li>
        );
      })}
      {(o.items ?? []).length === 0 && <li className="text-ink-muted">—</li>}

      {/* **وذيلُ الفاتورة: التوصيلُ ثمّ الإجمالي.**

              **والخصمُ يُقال حين يقع** — وسكوتُه يجعل الإجماليَّ لا يساوي ما
              فوقه، **فيُظنّ خطأً في الحساب.** */}
      {/* **والحسبةُ تُقرأ صاعدة**: أصنافٌ ← توصيلٌ ← إجمالي.

              قرارُ المالك (٢٠٢٦-٠٨-٠٤): «التوصيلَ اتركه تحت الفاتورة ليكون
              الإجماليُّ صحيحاً بصرياً — الشخصُ يعرف قيمةَ الطلب ويعرف قيمةَ
              التوصيل وكم أصبح الإجمالي».

              **ورقمٌ لا يُرى ما جُمع فيه يُصدَّق أو يُشكّ فيه بلا سبيل.** */}
      <li className="mt-2 flex items-center justify-between border-t border-line pt-2 text-sm">
        <span className="text-ink-muted">{m.admin.ordersPage.goodsValue}</span>
        <span dir="ltr" className="tabular-nums">
          {fmtNum(o.subtotal)}
        </span>
      </li>
      <li className="flex items-center justify-between text-sm">
        <span className="text-ink-muted">{m.admin.ordersPage.deliveryFee}</span>
        <span dir="ltr" className="tabular-nums">
          {fmtNum(o.delivery_fee)}
        </span>
      </li>
      {o.discount > 0 && (
        <li className="flex items-center justify-between text-sm text-success">
          <span>{m.admin.ordersPage.discount}</span>
          <span dir="ltr" className="tabular-nums">
            −{fmtNum(o.discount)}
          </span>
        </li>
      )}
      <li className="flex items-center justify-between border-t border-line pt-2">
        <span className="font-medium">{m.admin.ordersPage.total}</span>
        <span
          dir="ltr"
          className="text-lg font-bold tabular-nums text-primary-dark"
        >
          {fmtNum(o.total)}{" "}
          <span className="text-xs font-normal text-ink-muted">
            {PAYMENT_LABELS[o.payment_method]}
          </span>
        </span>
      </li>
    </ul>
  );
}

/** زرُّ الفاتورة في الجدول — **يفتحها حين تُطلب ولا يُثقل الصفّ.** */
function InvoiceButton({ order }: { order: OrderRow }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        variant="secondary"
        onClick={(e) => {
          e.stopPropagation();
          setOpen(true);
        }}
      >
        {m.admin.ordersPage.invoice}
      </Button>
      {open && (
        <span onClick={(e) => e.stopPropagation()}>
          <Modal
            open
            onClose={() => setOpen(false)}
            title={`${m.admin.ordersPage.invoice} · #${order.number}`}
          >
            <InvoiceList o={order} />
          </Modal>
        </span>
      )}
    </>
  );
}
