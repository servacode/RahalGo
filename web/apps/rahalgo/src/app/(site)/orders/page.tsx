"use client";

import Link from "next/link";

/** طلباتي: السجل الكامل مع حالة كل طلب. */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime, fmtClock } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Card,
  Modal,
  Invoice,
  OrderTrack,
  ChatArchive,
  Tabs,
  Select,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveRefresh,
  IconOrder,
  IconStar,
  Stars,
  IconPrint,
  IconMoto,
  IconLocation,
  IconSupport,
  IconChat,
  IconCheck,
  BrandMark,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import RatingModal from "@/components/RatingModal";
import ComplaintModal from "@/components/ComplaintModal";
import { etaText, hasEta } from "@/lib/eta";

const m = getMessages(defaultLocale);
const STATUS_LABELS: Record<string, string> = m.orders.status;

/** رسالةُ الخطأ من مفتاح الخادم — **لا نصَّ إنجليزيّ يصل الزبون.** */
function errText(e: unknown): string {
  const key = e instanceof ApiError ? ((e.body.message_key ?? "").split(".").pop() ?? "") : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/**
 * **نبرةُ الحالة — ولكلّ طورٍ لونُه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الحالاتُ لازم تكون بشكلٍ أوضح».)
 *
 * **كانت ستُّ حالاتٍ مسمّاةً وثمانٍ تسقط على الافتراضيّ** — فالطلبُ من
 * لحظة القبول إلى باب البيت بلونٍ واحد. **ومن فتح شاشتَه مرّتين في ساعةٍ
 * رأى اللونَ نفسَه فظنّ أنّ شيئاً لم يقع.**
 *
 * **وثلاثةُ أطوارٍ تُقرأ بلمحة**: انتظارٌ عند المتجر (تنبيه) · تجهيزٌ
 * وطريقٌ (خبر) · وانتهاء (نجاحٌ أو خطر).
 */
const VARIANT: Record<string, "warning" | "info" | "primary" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  accepted: "info",
  preparing: "info",
  dispatching: "primary",
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

interface OrderLineOption {
  id?: string;
  group: string;
  name: string;
  price_delta: number;
}

interface Order {
  id: string;
  number: number;
  // **ولا حقلَ متجرٍ هنا** — الخادمُ يمسح الاسمَ والمعرّفَ والشعار قبل الإرسال
  // (`customer_privacy.go`). **وحقلٌ في النوع يُغري بعرضه يوماً.**
  items_count: number;
  items_preview: string;
  items?: {
    menu_item_id: string | null;
    name: string;
    unit_price: number;
    qty: number;
    note: string;
    options: OrderLineOption[];
  }[];
  status: string;
  /**
   * ══════════════════════════════════════════════════════════════════
   * **مرحلتُه على المسار — يحسبها المحرّك**
   * ══════════════════════════════════════════════════════════════════
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الأفضل يكون بشكلٍ مركزيّ، مو كلّ صفحةٍ
   *  تاخذ من مكانٍ مختلف».)
   *
   * **وأربعَ عشرةَ حالاً تُطوى في ستّ مراحل** — والطيُّ هو ما يفترق بين
   * الشاشات إن كُتب في كلٍّ منها: **يُضاف حالٌ في المحرّك فيُدرَج في
   * شاشتين ويُنسى في الثالثة**، فيقرأ الزبونُ «في الطريق» وتقرأ الإدارةُ
   * «وصل» — ولا أحدَ يعرف أيُّهما الصواب.
   *
   * **والأسماءُ تبقى في المعجم** — المحرّكُ يرسل مفاتيحَ لا كلمات.
   */
  stages?: string[];
  /** موضعُه على المسار — **و`-1` لما انتهى قبل أن يصل.** */
  stage_at?: number;
  /** **أُسند سائق؟** — وبه وحدَه تُفتح قناةُ الحديث. */
  driver_id?: string | null;
  /** **نوعُ الطلب** — `custom` طلبٌ خاصٌّ يصفه صاحبُه بلفظه. */
  kind?: string;
  custom_request?: string;
  /** ما اتّفق عليه مع السائق — يراه كما يراه هو. */
  custom_goods_amount?: number | null;
  custom_fee?: number | null;
  total: number;
  /**
   * **تفصيلُ الإجمالي — يرسله الخادمُ وكان النوعُ يتجاهله.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «بالكرت ما هو مذكورٌ التوصيل، وهذا غلط —
   *  الزبون لازم يعرف قدّ إيش أجرة التوصيل وليش صار المبلغ هيك».)
   */
  subtotal: number;
  delivery_fee: number;
  discount: number;
  created_at: string;
  /** حقولُ الوقت المتوقَّع — **يرسلها الخادمُ أصلاً وكان النوعُ يتجاهلها.** */
  accepted_at?: string | null;
  ready_at?: string | null;
  prep_minutes?: number | null;
  delivery_estimate_min?: number;
  closed_at?: string | null;
  /** ما بقي من مهلة الإلغاء بالثواني — و`-1` تعني «بلا مهلة» (قبل القبول). */
  cancel_seconds_left?: number;
  /** ما يحتاجه **الطلبُ السريع**: العنوانُ نفسُه وطريقةُ الدفع نفسُها. */
  address_text?: string;
  lat?: number;
  lng?: number;
  payment_method?: string;
  /**
   * **لماذا انتهى قبل أن يصل.**
   *
   * الخادمُ يُلزم بالسبب في الرفض والإلغاء والفشل والاسترجاع
   * (`requiresReason` في `admin_orders_handlers.go`) **ويحفظه ويرسله** —
   * **وكانت البطاقةُ وحدَها لا تقرؤه.** فيقرأ الزبونُ «مرفوض» بلا كلمة،
   * ويبقى السببُ مكتوباً في قاعدةٍ لا يراها.
   *
   * (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٣: «لا يوجد سبب واضح للرفض».)
   *
   * **وحقلٌ واحدٌ يكفي للأربع**: `transitions.go` يكتب التعليلَ في
   * `cancel_reason` عند كلّ نهايةٍ غير التسليم — رفضاً وإلغاءً وفشلاً
   * واسترجاعاً.
   */
  cancel_reason?: string;
  /**
   * **ورمزُ التعذّر يسدّ ما يتركه النصّ.**
   *
   * صار التفصيلُ الحرُّ اختيارياً للسائق — «القائمةُ تُصنّف والنصُّ يشرح»
   * (`failreasons.go`) — **فطلبٌ تعذّر بلا تفصيلٍ يبقى `cancel_reason` فيه
   * فارغاً**، فيقرأ الزبونُ «تعذّر التسليم» بلا كلمة وقد عاد بلا طعامه.
   *
   * **والرمزُ مصنَّفٌ فيُترجَم** — ولا يُعرض خاماً.
   */
  fail_reason?: string;
}

interface RateInfo {
  order_id: string;
  number: number;
  items_preview: string;
  has_driver: boolean;
  rated: boolean;
  platform_stars: number;
  driver_stars: number | null;
  comment: string;
}

/**
 * إعادة الطلب: تُبنى السلّة من أصناف طلبٍ سابق.
 *
 * وتُستثنى الأصناف التي **حُذفت من القائمة** (`menu_item_id = null`) أو التي
 * تحمل خياراً بلا معرّف — وهي طلباتٌ سُجّلت قبل أن نحفظ معرّفات الخيارات. إضافتها
 * ناقصةً تُنتج طلباً يُرفض عند الإنشاء بـ«أصناف غير صالحة»، وهو أسوأ من إخبار
 * الزبون أن صنفاً لم يعد متاحاً.
 */
function reorderLines(o: Order) {
  const lines = [];
  let skipped = 0;
  for (const it of o.items ?? []) {
    const opts = it.options ?? [];
    if (!it.menu_item_id || opts.some((x) => !x.id)) {
      skipped++;
      continue;
    }
    lines.push({
      menu_item_id: it.menu_item_id,
      name: it.name,
      price: it.unit_price,
      qty: it.qty,
      note: it.note ?? "",
      option_ids: opts.map((x) => x.id!),
      option_names: opts.map((x) => x.name),
      options_delta: opts.reduce((a, x) => a + x.price_delta, 0),
    });
  }
  return { lines, skipped };
}

export default function MyOrdersPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [rateMap, setRateMap] = useState<Record<string, RateInfo>>({});
  const [rating, setRating] = useState<RateInfo | null>(null);
  /** الطلبُ الذي تُعرض فاتورتُه — **نافذةٌ لا صفحة**. */
  const [invoice, setInvoice] = useState<Order | null>(null);
  const [notice, setNotice] = useState("");
  /** الطلبُ الذي يُسأل عن تكراره — **سؤالٌ واحدٌ لا خطوات.** */
  const [again, setAgain] = useState<Order | null>(null);
  /**
   * **طريقةُ الدفع في الطلب المعاد — تُسأل ولا تُورَّث.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «عند تكرار طلب الزبون مرّة ثانية يجب أن نذكر
   *  طريقة الدفع — ممكن أوّل مرّة دفع نقديّ والثانية بدّو بالكرت أو العكس».)
   *
   * **وكانت تُورَّث بصمت** (`o.payment_method ?? "cash"`): من دفع بالمحفظة
   * مرّةً يُنشأ له طلبُ محفظةٍ **حتّى لو كان رصيدُه لا يكفي** — فيُرفض ولا
   * يفهم لماذا.
   */
  const [againPay, setAgainPay] = useState("cash");
  const [busy, setBusy] = useState(false);
  /** التبويبُ المفتوح — **والجاري افتراضاً**: من يفتح طلباتِه يتتبّع طلبَه. */
  const [tab, setTab] = useState<"live" | "past" | "cancelled" | "chats">("live");

  /**
   * **ثلاثُ سلالٍ — والملغاةُ لا تُخلَط بالمنتهية.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «طلبات جارية، طلبات سابقة، طلبات ملغاة».)
   *
   * **وهما نهايتان مختلفتان**: واحدةٌ وصلت، وأخرى لم تقع — **ومن يبحث عن
   * طلبٍ سُلّم لا يمرّ على كلّ ما أُلغي.**
   */
  const groups = {
    live: (orders ?? []).filter((o) => !o.closed_at),
    past: (orders ?? []).filter((o) => o.closed_at && o.status === "delivered"),
    cancelled: (orders ?? []).filter((o) => o.closed_at && o.status !== "delivered"),
  };

  /**
   * **إعادةُ الطلب: طلبٌ مباشرٌ لا سلّة.**
   *
   * كانت تملأ السلّةَ وتنقل الزبونَ إلى `/cart` — **فيُعيد الخطواتِ التي أراد
   * أن يتخطّاها**: يراجع، ويختار عنواناً، ويختار دفعاً، ويضغط. **وأربعُ ضغطاتٍ
   * لطلبٍ سبق أن طلبه.**
   *
   * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «ليس المقصود ع سلّة، يجب أن يقوم بطلب سريع
   * مباشر وليس إعادة الخطوات».
   *
   * **والعنوانُ والدفعُ من الطلب السابق** — هما ما اختاره حين طلبه أوّلَ مرّة.
   *
   * **ويُسأل مرّةً واحدة قبل الإنشاء.** لا لأنّها خطوة، **بل لأنّ ضغطةً
   * خاطئةً في قائمةٍ تُنشئ طلباً حقيقياً يُطبخ ويُوصَّل** — والزبونُ يدفع نقداً
   * عند الباب. **وسؤالٌ ثمنُه ثانية، وخطؤه ثمنُه طلب.**
   */
  async function orderAgain(o: Order) {
    const { lines, skipped } = reorderLines(o);
    if (lines.length === 0) {
      setNotice(m.site.orders.reorderNone);
      setAgain(null);
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const created = await api<{ number: number }>("/api/v1/orders", {
        method: "POST",
        body: JSON.stringify({
          items: lines.map((l) => ({
            menu_item_id: l.menu_item_id,
            qty: l.qty,
            note: l.note,
            option_ids: l.option_ids,
          })),
          address_text: o.address_text ?? "",
          lat: o.lat ?? 0,
          lng: o.lng ?? 0,
          payment_method: againPay,
        }),
      });
      setAgain(null);
      setNotice(
        (skipped > 0 ? m.site.orders.reorderPartial.replace("{n}", fmtNum(skipped)) + " " : "") +
          m.site.orders.againDone.replace("{n}", fmtNum(created.number)),
      );
      load();
    } catch (e) {
      // **وسببُ الرفض يُقال**: متجرٌ أغلق، أو صنفٌ نفد، أو رصيدٌ لا يكفي —
      // **و«تعذّر» وحدَها تترك الزبونَ يعيد المحاولة على ما لن ينجح.**
      setNotice(errText(e));
      setAgain(null);
    } finally {
      setBusy(false);
    }
  }

  /* **ولا تُفتح من هنا تلقائيّاً** — لها حارسٌ في التخطيط يسأل في أيّ
     صفحةٍ كان الزبون. **ونافذتان تُفتحان معاً تُقرآن عطباً.**
     (قرارُ المالك ٢٠٢٦-٠٨-٠٩.) وزرُّ البطاقة يبقى لمن أراد أن يقيّم قديماً. */
  const loadRatings = useCallback(() => {
    // **والردُّ صار كائناً بصفحته** — **ومصفوفةٌ تُقرأ من كائنٍ تردّ فراغاً
    // بهدوء**، فتختفي نجومُ التقييم عن كلّ بطاقة بلا خطأ.
    api<{ ratings: RateInfo[] }>("/api/v1/my/ratings")
      .then((res) =>
        setRateMap(Object.fromEntries((res?.ratings ?? []).map((r) => [r.order_id, r]))),
      )
      .catch(() => undefined);
  }, []);

  /**
   * **وفشلُ الجلب لا يُعرض «لا طلبات».**
   *
   * كان `.catch(() => setOrders([]))` — **فيرى الزبونُ «لا طلباتِ لك»** حين
   * تنقطع الشبكة، **وله طلبٌ في الطريق الآن.** فيظنّ أنّه ضاع أو أنّ المنصةَ
   * ألغته، **ويتّصل بالمكتب أو يطلب من جديد.**
   *
   * **وهي العائلةُ نفسُها** التي أُصلحت في صفحات الخادم — وهذه في أكثر ما
   * يُفتح قلقاً.
   */
  const [failed, setFailed] = useState(false);
  const load = useCallback(() => {
    setFailed(false);
    api<{ orders: Order[] }>("/api/v1/my/orders?per_page=50")
      .then((d) => setOrders(d.orders))
      .catch(() => setFailed(true));
    loadRatings();
  }, [loadRatings]);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/orders");
      return;
    }
    load();
  }, [user, loading, router, load]);

  useLiveRefresh(["order", "rating"], load);

  if (failed) {
    return (
      <PageContainer>
        <PageHeader
        icon={IconOrder}
        title={m.terms.orders}
        actions={
          /* **وطلبٌ خاصٌّ من هنا** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩).

             **وموضعُه صفحةُ الطلبات لا الشريطُ السفليّ**: فيه خمسةُ أقسامٍ
             والإبهامُ لا يتّسع لسادس. **ومن يريد أن يطلب شيئاً يفتح طلباتِه.** */
          <Link href="/custom">
            <Button variant="secondary">{m.site.custom.button}</Button>
          </Link>
        }
      />
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
        <Button variant="secondary" onClick={load}>
          {m.common.retry}
        </Button>
      </PageContainer>
    );
  }
  if (!orders) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader
        icon={IconOrder}
        title={m.terms.orders}
        actions={
          /* **وطلبٌ خاصٌّ من هنا** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩).

             **وموضعُه صفحةُ الطلبات لا الشريطُ السفليّ**: فيه خمسةُ أقسامٍ
             والإبهامُ لا يتّسع لسادس. **ومن يريد أن يطلب شيئاً يفتح طلباتِه.** */
          <Link href="/custom">
            <Button variant="secondary">{m.site.custom.button}</Button>
          </Link>
        }
      />
      {notice && (
        <Alert tone="warning" className="mb-3">{notice}</Alert>
      )}

      {/* ══════════════════════════════════════════════════════════════
          **أربعةُ تبويبات — لا أقسامٌ فوق بعضها**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١٠: «يجب أن يكون هناك أزرارُ تبويب: الجاري
           والسابقة والملغاة ودردشاتي السابقة».)

          **كانت ثلاثةَ أقسامٍ مكدَّسةً بعناوين** — تُخفى الفارغةُ منها.
          **فلا يعرف صاحبُ الشاشة ما لا يراه**: من لا طلبَ ملغًى له لا يرى
          للملغاة أثراً، **فيظنّها غيرَ موجودة** ويسأل أين ذهب طلبُه الذي
          ألغاه.

          **والتبويبُ يقول «هذه موجودةٌ وفارغة»** — وهو خبرٌ، وأن تختفي ليس
          خبراً.

          **ولا عددَ بجانب الاسم** (قرارُ المالك ٢٠٢٦-٠٨-١٠: «ليش عم تحطّ
          الأرقام؟ لا داعيَ لها»). **والعددُ يفيد حيث يُنتظَر عملٌ** — شكوى
          لم تُقرأ، طلبٌ ينتظر قبولاً. **وطلباتي ليست عملاً ينتظرني**، وثلاثةُ
          أصفارٍ فوق شاشةٍ فارغةٍ تُقرأ ضجيجاً لا خبراً.

          **ودردشاتي السابقة معها**: هي وجهُ الطلب الآخر — **من يبحث عن
          حجّةٍ يبحث عنها حيث الطلب**، لا في قائمةٍ جانبيّة. */}
      <Tabs
        items={[
          { key: "live", label: m.site.orders.tabActive },
          { key: "past", label: m.site.orders.tabPast },
          { key: "cancelled", label: m.site.orders.tabCancelled },
          { key: "chats", label: m.chat.archiveTitle, icon: IconChat },
        ]}
        value={tab}
        onChange={setTab}
        className="mb-4"
      />

      {tab === "chats" ? (
        <ChatArchive api={api} bare />
      ) : groups[tab].length === 0 ? (
        /* **وفراغُ التبويب يُقال بلفظه** — **ونصٌّ واحدٌ للثلاثة يكذب في
           اثنين**: «جرّب أوّل طلبٍ لك» تحت «الملغاة» تُقال لمن له عشرون
           طلباً. **ولكلّ تبويبٍ فراغُه.** */
        <EmptyState
          icon={IconOrder}
          title={
            orders.length === 0
              ? m.site.orders.empty
              : tab === "live"
                ? m.site.orders.emptyLive
                : tab === "past"
                  ? m.site.orders.emptyPast
                  : m.site.orders.emptyCancelled
          }
          tone={tab === "cancelled" ? "success" : "muted"}
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {groups[tab].map((o) => (
            <OrderCard
              key={o.id}
              o={o}
              rate={rateMap[o.id]}
              onReorder={() => {
                setAgainPay(o.payment_method ?? "cash");
                setAgain(o);
              }}
              onRate={() => setRating(rateMap[o.id] ?? null)}
              onInvoice={() => setInvoice(o)}
              onChanged={load}
            />
          ))}
        </div>
      )}

      {again && (
        <Modal open title={m.site.orders.againTitle} onClose={() => setAgain(null)}>
          <p className="mb-3 text-sm text-ink-muted">{m.site.orders.againBody}</p>
          <ul className="mb-3 divide-y divide-line rounded-control bg-field">
            {(again.items ?? []).map((it, i) => (
              <li key={i} className="flex items-center justify-between px-3 py-2 text-sm">
                <span className="min-w-0 flex-1">
                  {it.name}
                  {it.qty > 1 && <span className="text-ink-muted" dir="ltr"> ×{fmtNum(it.qty)}</span>}
                </span>
                <span dir="ltr" className="shrink-0 tabular-nums text-ink-muted">
                  {fmtNum(it.unit_price * it.qty)}
                </span>
              </li>
            ))}
          </ul>
          {/* **العنوانُ يُعرض لا يُفترض** — من طلب إلى بيته أمسِ قد يكون اليومَ
              في عمله، **وطلبٌ يصل إلى عنوانٍ خاطئٍ خسارةٌ للجميع.** */}
          {again.address_text && (
            <p className="mb-4 flex items-start gap-2 rounded-control bg-field px-3 py-2 text-sm">
              <IconLocation size={15} className="mt-0.5 shrink-0 text-ink-muted" />
              <span className="min-w-0 flex-1">{again.address_text}</span>
            </p>
          )}
          {/* **وطريقةُ الدفع تُسأل** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩).

              **وتبدأ بطريقة الطلب السابق اقتراحاً لا فرضاً**: أكثرُ الناس
              يعيدون كما دفعوا، **ومن أراد غيرَها يبدّلها بضغطة.** */}
          <Select
            id="again-payment"
            label={m.site.cart.payment}
            className="mb-4"
            value={againPay}
            onChange={(e) => setAgainPay(e.target.value)}
          >
            <option value="cash">{m.orders.payment.cash}</option>
            <option value="wallet">{m.orders.payment.wallet}</option>
          </Select>

          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setAgain(null)}>
              {m.common.cancel}
            </Button>
            <Button disabled={busy} onClick={() => void orderAgain(again)}>
              {busy ? m.common.loading : m.site.orders.againConfirm}
            </Button>
          </div>
        </Modal>
      )}

      {invoice && (
        <Modal open title={m.site.orders.invoice} onClose={() => setInvoice(null)} size="lg">
          <Invoice order={invoice as never} />
        </Modal>
      )}

      {rating && (
        <RatingModal
          order={rating}
          onClose={() => setRating(null)}
          onRated={() => {
            setRating(null);
            loadRatings();
          }}
        />
      )}
    </PageContainer>
  );
}

/**
 * **مراحلُ الطلب كما يراها الزبون** — لا كما تُسمّى في المحرّك.
 *
 * المحرّكُ يعرف `assigned` و`at_pickup` و`picked_up` و`on_the_way` و
 * `at_dropoff` — **وهي شؤونٌ داخلية لا تخصّ من ينتظر طعامه.** فتُطوى في مرحلةٍ
 * واحدة: «في الطريق».
 *
 * **وخمسُ عُقَدٍ لا تسع**: بطاقةٌ في قائمة، وأسماءٌ تحت العُقَد. **وأربعٌ تُقرأ
 * بلمحة.**
 */
const TRACK = ["pending", "accepted", "preparing", "onway", "delivered"] as const;

/**
 * **ومسارُ الطلب الخاصّ غيرُه — لا متجرَ فيه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «بالطلب الخاصّ ما في شيءٌ اسمه قيد التحضير…
 *  تمّ الشراء برأيي» · «بس تطلع بعدما يضغط السائق اشتريتُ الطلب».)
 *
 * **«قُبل» تعني قَبِله المتجر و«قيد التحضير» تعني يطبخه** — **ولا مطبخَ
 * هنا.** وكان الطلبُ في طابور السائقين تُضيء له «قيد التحضير» **فيطمئنّ
 * صاحبُه أنّ أحداً يعمل، ولا أحدَ أخذه بعد.** **وخبرٌ كاذبٌ أسوأُ من لا
 * خبر**: من ينتظر ولا يعلم يسأل، ومن اطمأنّ كاذباً ينتظر.
 *
 * **و«تمّ الشراء» موضعُها بعد ضغطة السائق «اشتريتُ الطلب»** — لا قبلها:
 * **هي إقرارٌ بأنّ مالَه خرج من جيبه**، وعلامةٌ تسبقه تَعِد الزبونَ بما لم
 * يقع.
 */
const TRACK_CUSTOM = ["pending", "driver", "bought", "onway", "delivered"] as const;

/**
 * **مسارُه وموضعُه — من المحرّك لا من جدولٍ هنا.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٢.)
 *
 * **وكان الطيُّ مكتوباً هنا وفي تطبيق السائق** — جدولان لشيءٍ واحد.
 * **والمحرّكُ يرسلهما الآن** (`stages` و`stage_at`)، فما يُضاف فيه يظهر
 * في الشاشات كلِّها في نشرةٍ واحدة.
 *
 * **والقديمُ يبقى احتياطاً لنسخةِ خادمٍ لم تُحدَّث بعد** — شاشةٌ بلا
 * مسارٍ أسوأُ من مسارٍ قديم.
 */
function trackOf(o: Order) {
  const keys = o.stages;
  if (keys && keys.length > 0) {
    return {
      stages: keys.map((id) => ({
        id,
        label: (m.site.orders.stage as Record<string, string>)[id] ?? id,
      })),
      at: o.stage_at ?? -1,
    };
  }
  if (o.kind === "custom") {
    return {
      stages: TRACK_CUSTOM.map((id) => ({ id, label: m.site.orders.trackCustom[id] })),
      at: trackIndexCustom(o.status),
    };
  }
  return {
    stages: TRACK.map((id) => ({ id, label: m.site.orders.track[id] })),
    at: trackIndex(o.status),
  };
}

/**
 * **الحالةُ الخام ← موضعُها في مسار الطلب الخاصّ.**
 *
 * **ولا `at_pickup` فيه** — لا بابَ متجرٍ يقف عنده. (`customTransitions`
 * في المحرّك: أُسند ← اشترى.)
 */
function trackIndexCustom(status: string): number {
  switch (status) {
    case "pending":
      return 0;
    // **والطابورُ والإسنادُ عقدةٌ واحدة** — كلاهما «يُبحث له عن سائق»،
    // **والزبونُ يقرأ حالتَه بالتفصيل في الشارة فوق البطاقة.**
    case "dispatching":
    case "assigned":
      return 1;
    case "picked_up":
      return 2;
    case "on_the_way":
    case "at_dropoff":
      return 3;
    case "delivered":
      return 4;
    default:
      return -1; // ملغى · مرفوض · مُخفق — **لا مسارَ لهم**
  }
}

/** الحالةُ الخام ← موضعُها على المسار. */
/**
 * **لماذا انتهى — بأوّل ما يُقال، لا بأدقّه.**
 *
 * التفصيلُ الحرُّ أدلُّ حين يُكتب («الباب مغلق ولا أحد يردّ»)، **لكنّه اختياريّ
 * للسائق**. فإن غاب بقي الرمزُ المصنَّف — **وهو يقول شيئاً ولو أقلّ**، وسطرٌ
 * ناقصٌ خيرٌ من صمتٍ أمام زبونٍ عاد بلا طعامه.
 *
 * **والرمزُ يُترجَم ولا يُعرض خاماً** — و`customer_absent` في شاشةِ زبونٍ
 * عطبٌ يُقرأ، لا معلومة.
 */
function endedReason(o: Order): string {
  const free = o.cancel_reason?.trim();
  if (free) return free;
  const code = o.fail_reason?.trim();
  if (!code) return "";
  return m.common.failReasons[code as keyof typeof m.common.failReasons] ?? "";
}

function trackIndex(status: string): number {
  switch (status) {
    case "pending":
      return 0;
    case "accepted":
      return 1;
    case "preparing":
    case "dispatching":
      return 2;
    case "assigned":
    case "at_pickup":
    case "picked_up":
    case "on_the_way":
    case "at_dropoff":
      return 3;
    case "delivered":
      return 4;
    default:
      return -1; // ملغى · مرفوض · مُخفق · مُسترجَع — **لا مسارَ لهم**
  }
}

/**
 * **بطاقةُ الطلب — قائمةٌ بذاتها.**
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «لا يوجد داعي لزرّ تفاصيل الطلب · حالة الطلب
 * تكون بنفس الكرت · بالكرت كلّ صنف وسعره في حقلٍ خاصّ مرتّب · الفاتورة تصير
 * أيقونة».
 *
 * **وكلُّ ما كان خلف ضغطةٍ صار في وجه البطاقة**: ماذا طلبتُ، وبكم، وأين هو
 * الآن. **والانتقالُ إلى صفحةٍ لقراءة سطرين ضريبةٌ تُدفع في كلّ مرّة.**
 */
function OrderCard({
  o,
  rate,
  onReorder,
  onRate,
  onInvoice,
  onChanged,
}: {
  o: Order;
  rate?: RateInfo;
  onReorder: () => void;
  onRate: () => void;
  onInvoice: () => void;
  /** يُعاد التحميلُ بعد إلغاءٍ ناجح — **البطاقةُ تُظهر النتيجة بلا تحديث.** */
  onChanged: () => void;
}) {
  /** نافذةُ الشكوى — **في البطاقة لا في صفحةٍ أخرى.** */
  const [complaining, setComplaining] = useState(false);
  /** نافذةُ تأكيد الإلغاء — **والإلغاءُ لا يُتراجع عنه.** */
  const [confirming, setConfirming] = useState(false);
  const [cancelBusy, setCancelBusy] = useState(false);
  const [cancelErr, setCancelErr] = useState("");
  /**
   * **العدّادُ يعدّ فعلاً.**
   *
   * الرقمُ من الخادم نسبيٌّ عند وصوله، **فيُثبَّت مرساه بساعة الجهاز نفسِه**
   * ويُطرح منه ما مضى — **فلا تدخل ساعةُ الخادم في الحساب**، وفارقُ الساعتين
   * لا يجعل زرّاً حيّاً يبدو منقضياً.
   */
  const [left, setLeft] = useState(0);
  useEffect(() => {
    const n = o.cancel_seconds_left ?? 0;
    if (n <= 0) {
      setLeft(0);
      return;
    }
    const anchor = Date.now();
    const tick = () => setLeft(Math.max(0, n - Math.floor((Date.now() - anchor) / 1000)));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [o.cancel_seconds_left]);
  const [ticketNo, setTicketNo] = useState(0);
  const { stages: trackStages, at } = trackOf(o);
  const closed = at < 0;
  const live = !closed && o.status !== "delivered";
  const canRate = o.status === "delivered" && rate && !rate.rated;
  /** **انتهى أمرُه** — سُلّم أو أُغلق. وعندها وحدَها تُفتح الشكوى. */
  const finished = closed || o.status === "delivered";
  const items = o.items ?? [];

  return (
    <Card className="flex flex-col gap-4">
      {/* ══════════════════════════════════════════════════════════
          **الترويسة: علامةٌ وتاريخُها · ورقمٌ وحالتُه**
          ══════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «احذف اسمَ المنصة · جِب اللوغو من المركز
           بشكلٍ مركزيّ · التاريخُ لازم يكون تحت اللوغو».)

          **ولا اسمَ للمنصة هنا** — كان سطراً عريضاً يقول ما تقوله العلامةُ
          فوقه. (قاعدةُ المالك: «لا تكتب اسمَ المنصة بأيّ مكان».)

          **والعلامةُ كما تأتي من المركز** — كان حولها غلافٌ يقصّها وشريطٌ
          ملوّنٌ أسفلَها، **زخرفةٌ محلّيّةٌ لا يعرفها باقي المواضع.**

          **والتاريخُ تحتها**: عمودٌ واحدٌ يُقرأ من فوق إلى تحت — ما هذا،
          ومتى. **وكان بجانبها فيزاحم الرقمَ على العرض نفسِه.** */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex shrink-0 flex-col items-start gap-1.5">
          <BrandMark size={72} rounded="card" />
          <span className="text-2xs tabular-nums text-ink-muted" dir="ltr">
            {fmtDateTime(o.created_at)}
          </span>
        </div>

        <div className="flex min-w-0 flex-col items-end gap-1.5">
          {/* **الرقمُ بالنبرة** — هو ما يُقال في الهاتف حين يُسأل عن طلب. */}
          <span
            dir="ltr"
            className="rounded-control bg-accent px-2.5 py-1 text-sm font-bold tabular-nums text-on-bright"
          >
            #{fmtRef(o.number)}
          </span>
          <Badge variant={VARIANT[o.status] ?? "primary"}>
            {STATUS_LABELS[o.status] ?? o.status}
          </Badge>
        </div>
      </div>

      {/* ── الأصناف: كلٌّ في سطره وسعرُه أمامه ────────────────────────── */}
      {items.length > 0 ? (
        <ul className="divide-y divide-line rounded-control bg-field">
          {items.map((it, i) => (
            <li key={i} className="flex items-start gap-3 px-3 py-2 text-sm">
              <span className="min-w-0 flex-1">
                {it.name}
                {it.qty > 1 && (
                  <span className="text-ink-muted" dir="ltr">
                    {" "}
                    ×{fmtNum(it.qty)}
                  </span>
                )}
                {/* الخياراتُ تحت الاسم — **هي ما يُميّز طلباً عن طلب.** */}
                {it.options?.length > 0 && (
                  <span className="block text-xs text-ink-muted">
                    {it.options.map((x) => x.name).join(m.common.listSep)}
                  </span>
                )}
              </span>
              <span dir="ltr" className="shrink-0 tabular-nums text-ink-muted">
                {fmtNum(it.unit_price * it.qty)}
              </span>
            </li>
          ))}
        </ul>
      ) : o.kind === "custom" ? (
        /* **والطلبُ الخاصُّ يُعرَض بلفظه لا «بلا أصنافٍ مسجّلة».**

           (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «الطلب نفسه يظهر بدل أصناف غير مسجّلة».)

           **ولا أصنافَ له تُسجَّل** — هو سطرٌ كتبه صاحبُه. **وجملةُ «بلا
           أصناف» تُقرأ نقصاً في الطلب** لا وصفاً لنوعه. */
        <p className="whitespace-pre-wrap rounded-control bg-field px-3 py-2 text-sm">
          {o.custom_request || "—"}
        </p>
      ) : (
        <p className="text-sm text-ink-muted">{o.items_preview || m.site.orders.noItems}</p>
      )}

      {/* **وما اتُّفق عليه يراه الزبونُ كما يراه السائق.**

          (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «وبكرت الزبون لا يظهر السعر وأجرة
           التوصيل كما اتّفقنا».)

          **وهو طرفُ الاتّفاق** — قيل له في المحادثة، **ويبقى مكتوباً حيث
          يراه**: فمن نسي رجع إليه، ومن خولف احتجّ به. */}
      {o.kind === "custom" ? (
        o.custom_goods_amount == null ? (
          /* **ولا مبلغَ قبل أن يُتّفق عليه** — (قرارُ المالك ٢٠٢٦-٠٨-١٠
             ضمناً: «الإجمالي صفر ورسمُ التوصيل مجّانيّ» على طلبٍ لم يُسعَّر).

             **وكانت البطاقةُ تعرض حسابَ المنصّة**: بضاعةٌ صفرٌ ورسمٌ
             «مجّانيّ» وإجماليٌّ صفر. **وثلاثتُها كذبٌ في الطلب الخاصّ**:
             أعمدةُ المحاسبة أصفارٌ **بقصد** — لأنّ المنصّةَ لا تحاسب على
             هذا الطلب — **والزبونُ يقرؤها ثمناً.**

             **و«مجّانيّ» أخطرُها**: يقرأ الزبونُ أنّه لا يدفع أجرة، **ثمّ
             يطلبها السائقُ عند الباب** فيقع الخلافُ الذي بُني التوثيقُ
             كلُّه ليمنعه. */
          <p className="mt-2 rounded-control bg-field px-3 py-2 text-xs text-ink-muted">
            {m.site.custom.pending}
          </p>
        ) : (
        <div className="mt-2 space-y-1 rounded-control bg-field px-3 py-2 text-sm">
          <p className="flex items-center justify-between gap-2">
            <span className="text-ink-muted">{m.site.custom.goods}</span>
            <span className="tabular-nums" dir="ltr">{fmtNum(o.custom_goods_amount)}</span>
          </p>
          <p className="flex items-center justify-between gap-2">
            <span className="text-ink-muted">{m.site.custom.fee}</span>
            <span className="tabular-nums" dir="ltr">{fmtNum(o.custom_fee ?? 0)}</span>
          </p>
          <p className="flex items-center justify-between gap-2 border-t border-line-soft pt-1 font-bold">
            <span>{m.site.custom.total}</span>
            <span className="tabular-nums" dir="ltr">
              {fmtNum((o.custom_goods_amount ?? 0) + (o.custom_fee ?? 0))} {m.common.currency}
            </span>
          </p>
        </div>
        )
      ) : (
      /* ══════════════════════════════════════════════════════════════
          **ومن أين جاء الإجمالي — لا الإجمالي وحدَه**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «بالكرت ما هو مذكورٌ التوصيل، وهذا غلط
           — الزبون لازم يعرف قدّ إيش أجرة التوصيل وليش صار المبلغ هيك».)

          **كانت البطاقةُ تعرض صنفاً بثلاثين ألفاً ثمّ إجمالياً بأربعين** —
          وبينهما عشرةٌ لا يقول أحدٌ من أين. **ومن لا يفهم فرقاً في حسابه
          يظنّه زيادةً مسروقة**، ولا يسأل: يسكت ولا يعود.

          **ورسمُ التوصيل ليس زيادةً تُخفى** — هو خدمةٌ تُؤدّى ويُدفع أجرُها،
          **وذِكرُه صراحةً يبني الثقةَ لا يهدمها.**

          **والسطرُ يظهر ولو كان صفراً** إن كان هناك أصنافٌ تُحسب: توصيلٌ
          مجّانيٌّ خبرٌ سارٌّ لا يُكتم. **والخصمُ لا يظهر إلّا إن وقع.** */
      <div className="space-y-1.5 border-t border-line-soft pt-3">
        <div className="flex items-center justify-between text-sm">
          <span className="text-ink-muted">{m.shared.invoice.subtotal}</span>
          <span dir="ltr" className="tabular-nums text-ink-muted">
            {fmtNum(o.subtotal ?? 0)}
          </span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-ink-muted">{m.shared.invoice.deliveryFee}</span>
          {/* **وصفرُ الأجرة تُقال «مجاني»** — والصفرُ يُقرأ حساباً لم يكتمل.
              (قرارُ المالك ٢٠٢٦-٠٨-٠٩.) */}
          {(o.delivery_fee ?? 0) === 0 ? (
            <span className="font-medium text-success">
              {m.shared.invoice.deliveryFree}
            </span>
          ) : (
            <span dir="ltr" className="tabular-nums text-ink-muted">
              {fmtNum(o.delivery_fee)}
            </span>
          )}
        </div>
        {o.discount > 0 && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-ink-muted">{m.shared.invoice.discount}</span>
            <span dir="ltr" className="tabular-nums text-success">
              {fmtNum(-o.discount)}
            </span>
          </div>
        )}
        <div className="flex items-center justify-between border-t border-line-soft pt-2">
          <span className="text-sm text-ink-muted">{m.site.orders.statTotal}</span>
          <span dir="ltr" className="figure">
            {fmtNum(o.total)}{" "}
            <span className="text-xs font-normal text-ink-muted">{m.common.currency}</span>
          </span>
        </div>
      </div>
      )}

      {/* ── أين هو الآن ───────────────────────────────────────────────── */}
      {closed ? (
        /* **ولا مسارَ لما انتهى قبل أن يصل.** شريطٌ يقف في منتصفه يُقرأ
           «عالق» لا «انتهى» — فيُقال بالحرف. */
        <div className="space-y-1.5 rounded-control bg-field px-3 py-2 text-center">
          <p className="text-sm text-ink-muted">
            {m.site.orders.trackClosed.replace("{s}", STATUS_LABELS[o.status] ?? o.status)}
          </p>
          {/* **والسببُ يُقال لصاحبه.**

              الخادمُ يمنع الرفضَ والإلغاءَ والفشلَ بلا تعليل (`requiresReason`)،
              **ثمّ كانت البطاقةُ تبتلع ما كُتب**: يقرأ الزبونُ «مرفوض» ولا يعرف
              أنفدت الأصنافُ أم أُغلق المتجر أم أخطأ عنوانُه — **فيعيد الطلبَ
              نفسَه فيُرفض مرّةً ثانية.**

              وهو ظاهرٌ للإدارة وللمندوب منذ البداية — **وللزبون وحدَه لا.**
              (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٣: «لا يوجد سبب واضح للرفض».) */}
          {endedReason(o) && (
            <p className="text-sm">
              <span className="text-ink-muted">{m.site.orders.endedReason}: </span>
              <span className="font-medium text-danger">{endedReason(o)}</span>
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          <OrderTrack
            stages={trackStages}
            current={at}
            vehicle={IconMoto}
            live={live}
          />
          {/* **الوقتُ المتوقَّع تحت المسار مباشرةً.**

              «قيد التحضير» وحدَها لا تقول عشرَ دقائقَ أم ساعة. **والمسارُ يقول
              أين، والوقتُ يقول متى** — ولا يُقرأ أحدُهما بلا الآخر.

              وحسابُه مشتركٌ مع صفحة التتبّع (`lib/eta.ts`) — **ولو نُسخ لَافترقا
              يوماً، فتقول البطاقةُ عشرين وتقول الصفحةُ خمساً.** */}
          {hasEta(o, o.status, closed) && (
            <p className="flex items-center justify-center gap-1.5 text-xs font-medium text-primary-dark">
              <IconCheck size={13} strokeWidth={3} />
              {m.site.orders.eta}: {etaText(o)}
            </p>
          )}
        </div>
      )}

      {/* ── ولا حديثَ في البطاقة ─────────────────────────────────────
          (قرارُ المالك ٢٠٢٦-٠٨-١٠: «ما بدّي محادثة داخل الكرت أبداً —
           المحادثة بزرّ الدردشة العائم فقط».)

          **كانت شاشةَ حديثٍ كاملةً داخل كلّ بطاقة**: عنوانٌ وسجلُّ رسائلَ
          وحقلُ كتابةٍ وعدّادُ حروف — **فتطول البطاقةُ ضِعفَها** ويُدفع ما
          تحتها خارجَ الشاشة.

          **وعشرةُ طلباتٍ تعني عشرَ شاشاتِ حديثٍ في صفحةٍ واحدة** — كلٌّ
          تنادي الخادمَ وتُحدّث نفسَها.

          **والفقّاعةُ العائمةُ تفعلها مرّةً واحدةً لكلّ المفتوح** بتبويبٍ
          بالأرقام: **موضعٌ واحدٌ يُعرف** بدل أن يُبحث عن الحديث في بطاقةٍ
          بين بطاقات. **والبطاقةُ تبقى للطلب: حالتُه ومبلغُه وطريقُه.** */}

      {/* ── الإلغاء: نافذةٌ تُرى وهي تنقضي ───────────────────────────── */}
      {/* **ومتى يُعرض بابُ الإلغاء** — (شهده المالك ٢٠٢٦-٠٨-١٠: «الإلغاءُ
          لم يُطبَّق على الطلبات الخاصّة بنفس الأسلوب»).

          **العاديُّ**: قبل قبول المتجر متى شاء، وبعده ضمن مهلة.

          **والخاصُّ إلى أن يُشترى** — ولا مهلةَ فيه: **حدُّه حدثٌ لا ساعة**،
          خروجُ المال من جيب السائق. **ولا يبلغ `accepted` أصلاً**، فكان
          الشرطُ لا يصدق عليه بعد موافقة الإدارة أبداً.

          **والشاشةُ تعرض ما يقبله المحرّك** — `customTransitions` هي الحكم،
          **وشرطان يفترقان يجعلان زرّاً يُضغط فيُردّ.** */}
      {(o.kind === "custom"
        ? ["pending", "dispatching", "assigned"].includes(o.status)
        : o.status === "pending" || (o.status === "accepted" && left > 0)) && (
        <div>
          <Button
            variant="danger"
            disabled={cancelBusy}
            onClick={() => setConfirming(true)}
            className="w-full !py-2"
          >
            {m.site.orders.cancel}
          </Button>
          {o.status === "accepted" && (
            <p className="mt-1 text-center text-xs text-ink-muted">
              {m.site.orders.cancelWindow.replace("{t}", fmtClock(left))}
            </p>
          )}
          {/* **ويُقال متى يُقفل البابُ** — **زرٌّ يختفي بلا سابق إنذارٍ
              يُقرأ عطباً**، ومن عرف الحدَّ قرّر قبل أن يبلغه. */}
          {o.kind === "custom" && (
            <p className="mt-1 text-center text-xs text-ink-muted">
              {m.site.orders.cancelUntilBought}
            </p>
          )}
          {cancelErr && <p className="mt-1 text-sm text-danger">{cancelErr}</p>}

          <Modal
            open={confirming}
            onClose={() => setConfirming(false)}
            title={m.site.orders.cancelConfirmTitle}
          >
            <p className="mb-4 text-sm text-ink-muted">{m.site.orders.cancelConfirmBody}</p>
            <div className="flex gap-2">
              {/* **والتراجعُ أوّلاً** — من فتح النافذةَ بالخطأ يجد المخرجَ في
                  موضع الزرّ الذي اعتاده. */}
              <Button variant="secondary" onClick={() => setConfirming(false)}>
                {m.site.orders.cancelConfirmNo}
              </Button>
              <Button
                variant="danger"
                disabled={cancelBusy}
                onClick={async () => {
                  setCancelBusy(true);
                  setCancelErr("");
                  try {
                    await api(`/api/v1/orders/${o.id}/cancel`, {
                      method: "POST",
                      body: JSON.stringify({ note: "" }),
                    });
                    setConfirming(false);
                    onChanged();
                  } catch (e) {
                    setConfirming(false);
                    setCancelErr(errText(e));
                  } finally {
                    setCancelBusy(false);
                  }
                }}
              >
                {m.site.orders.cancelConfirmYes}
              </Button>
            </div>
          </Modal>
        </div>
      )}

      {/* ── الأفعال ───────────────────────────────────────────────────── */}
      <div className="flex flex-wrap items-center gap-2">
        {items.length > 0 && (
          /* **الفعلُ الأكثرُ تكراراً يلبس لونَ العلامة** — ومن طلب مرّةً
             يطلب ثانية، **وزرٌّ باهتٌ لأكثر ما يُضغط يُبطئ ما يجب أن يسرع.** */
          <Button
            onClick={onReorder}
            className="flex flex-1 items-center justify-center gap-1.5 !py-1.5"
          >
            <IconOrder size={15} />
            {m.site.orders.reorder}
          </Button>
        )}
        {canRate && (
          <Button onClick={onRate} className="flex flex-1 items-center justify-center gap-1.5 !py-1.5">
            <IconStar size={15} />
            {m.site.rating.rateOrder}
          </Button>
        )}
        {o.status === "delivered" && rate?.rated && (
          <span className="flex flex-1 items-center justify-center gap-2 rounded-control bg-field px-3 py-1.5">
            <span className="text-xs text-ink-muted">{m.site.rating.merchant}</span>
            <Stars value={rate.platform_stars} size="sm" />
          </span>
        )}
        {/* **الفاتورةُ أيقونة** — يحتاجها من يطبع، ولا يحتاجها الباقون.
            **وزرٌّ بعرض الثلث لفعلٍ نادرٍ يزاحم فعلاً يوميّاً.**

            ══════════════════════════════════════════════════════════════
            **ولا فاتورةَ قبل التسليم**
            ══════════════════════════════════════════════════════════════

            (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الفاتورةُ أصولاً تظهر بعد التسليم،
             وليس قبل التسليم».)

            **كانت تُطبع لطلبٍ في الطابور** — ورقةٌ بمبالغَ لم تُقبض بعد،
            **وقد يُلغى الطلبُ أو يتعذّر** فتبقى في يد صاحبها ورقةٌ تقول
            إنّه دفع.

            **والفاتورةُ إقرارُ ما وقع لا وعدٌ بما سيقع** — وهي في الطلب
            الخاصّ أوضح: **مبلغُها لا يُعرف حتّى يتّفقا.** */}
        {o.status === "delivered" && (
        <button
          type="button"
          onClick={onInvoice}
          aria-label={m.site.orders.invoice}
          title={m.site.orders.invoice}
          /* **الطباعةُ خضراء والشكوى حمراء** — فعلان متجاوران ومعناهما متضادّ:
             أحدُهما يحفظ والآخر يشتكي. **ولونٌ واحدٌ لهما يجعل الضغطةَ قرعةً**،
             والعينُ تفرّق بالألوان قبل أن تقرأ الأيقونات.
             (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-success-tint text-success transition-opacity hover:opacity-90"
        >
          <IconPrint size={16} />
        </button>
        )}

        {/* **وبابُ الشكوى في البطاقة.**

            كان خلف صفحة التفاصيل — **وقد أُغلق بابُها.** ومن وقع له خطأٌ في
            طلبه لا يبحث عن مكانِ الشكوى، **ومن لم يجدها في ثانيتين يتّصل أو
            يسكت** — والسكوتُ أسوأ: نخسر الزبونَ ولا نعرف لماذا.

            **ولا يظهر إلّا بعد أن ينتهي الطلب**: شكوى على طلبٍ في الطريق
            شكوى على ما لم يقع بعد. */}
        {finished &&
          (ticketNo > 0 ? (
            <span
              title={m.site.complaint.opened.replace("{n}", fmtNum(ticketNo))}
              className="flex h-9 shrink-0 items-center rounded-control bg-field px-2 text-xs text-ink-muted"
              dir="ltr"
            >
              #{fmtRef(ticketNo)}
            </span>
          ) : (
            <button
              type="button"
              onClick={() => setComplaining(true)}
              aria-label={m.site.complaint.open}
              title={m.site.complaint.open}
              /* **أيقونةُ الشكوى هي أيقونتُها في اللوحة كلِّها** — طوقُ
                 النجاة: قسمُ التذاكر، ورأسُ صفحتها، وبطاقاتُها.

                 وكانت مثلّثَ تحذير — **وهو يُقرأ «في هذه البطاقة خطأ» لا
                 «اشتكِ من هذا الطلب»**، فيقلق من لا شكوى له.

                 **وحدودٌ حمراءُ وخلفيةٌ خفيفةٌ تجعله يُرى** بجانب أيقونة
                 الطباعة الرمادية: الفعلان مختلفان، **فلا يُلبسان لباساً
                 واحداً.** */
              className="flex h-9 shrink-0 items-center gap-1.5 rounded-control bg-danger-solid px-2.5 text-xs font-medium text-on-solid transition-opacity hover:opacity-90"
            >
              <IconSupport size={16} />
              {m.site.complaint.short}
            </button>
          ))}
      </div>

      {complaining && (
        <ComplaintModal
          orderId={o.id}
          orderNumber={o.number}
          onClose={() => setComplaining(false)}
          onOpened={(n) => {
            setTicketNo(n);
            setComplaining(false);
          }}
        />
      )}
    </Card>
  );
}
