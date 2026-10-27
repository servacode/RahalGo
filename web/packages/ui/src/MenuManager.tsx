"use client";

/**
 * محرّر الأصناف — **نسخة واحدة مركزية** للإدارة وبوابة المتجر.
 *
 * كان المحرّر الكامل في لوحة الإدارة (521 سطراً)، وبوابة المتجر تملك **الإتاحة
 * وحدها**. فبناء محرّرٍ ثانٍ للمتجر كان سيكرّر المنطق كلّه — وقد رأينا مرّتين في
 * هذا المشروع ما ينتج عن تكرار التركيب: نسختان تنحرفان بلا أن يقصد أحد
 * (R-35 المحفظة، R-44 الشريط العلوي).
 *
 * فما يختلف بين البوابتين **مسارات لا نسخ**: الإدارة تعمل تحت `/admin` والمتجر
 * تحت `/merchant`، والردّ واحد (`catalog.GetMenu`) والحارس في الخادم مختلف.
 *
 * ونسمّيها **الأصناف** لا «القائمة» ولا «المينو»: «القائمة» ملتبسة (list/menu)،
 * و«المينو» تخصّ المطاعم وحدها — ورحّال فيها بقالة وصيدليات وخضار. والوحدة
 * تُسمّى «صنف» في كل المشروع، فجمعُها هو اللفظ المتّسق.
 */

import { FormActions } from "./FormActions";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Money } from "./money";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import { Button, Input, Select, Badge, Modal, Checkbox } from "./components";
import { Confirm, Alert } from "./feedback";
import { EmptyState } from "./layout";
import { IconAdd, IconEdit, IconDelete, IconStore, IconPrev } from "./icons";

const m = getMessages(defaultLocale);
const L = m.shared.menuEditor;
const IT = m.store.item;

/** **حدُّ اسم الصنف** — مثلُ `catalog.item_name_max` في المحرّك (٢٠٢٦-١٠-١٤). */
const ITEM_NAME_MAX = 30;

function fill(t: string, v: Record<string, string | number>): string {
  return t.replace(/\{(\w+)\}/g, (_, k) => String(v[k] ?? ""));
}

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

export interface ModifierOption {
  id?: string;
  name: string;
  price_delta: number;
  /** **فرقُ الخيار بالدولار** لصنفٍ دولاريّ — والليرةُ يحسبها الخادمُ بسعر الصرف. */
  price_delta_usd?: number | null;
  /** نصُّ الدولار وهو يُكتب — للمحرّر وحدَه، لا يُرسَل. */
  _usd?: string;
}
export interface ModifierGroup {
  name: string;
  min_select: number;
  max_select: number;
  options: ModifierOption[];
}
/**
 * **مجموعاتٌ جاهزةٌ بمربّع صح — كتطبيق المتجر حرفاً** (قرارُ المالك ٢٠٢٦-١٠-١٩، واللوحةُ ٢٠٢٦-١٠-٢٦:
 * «المجموعات بتطبيق المتجر نفسهن لازم يكونو باللوحة»). الحجمُ والوزنُ خيارٌ واحدٌ إلزاميّ، والإضافاتُ
 * اختياريّةٌ بلا حدّ، **وتُعرف المجموعةُ باسمها** كما في `ItemEditor.kt`. والمشروباتُ من قائمتها.
 */
type Preset = "size" | "weight" | "extras";
function presetOf(g: ModifierGroup): Preset {
  const IT = getMessages(defaultLocale).store.item;
  const n = g.name.trim();
  if (n === IT.presetSize) return "size";
  if (n === IT.presetWeight) return "weight";
  return "extras";
}
function presetGroup(p: Preset): ModifierGroup {
  const IT = getMessages(defaultLocale).store.item;
  const opt = (name: string): ModifierOption => ({ name, price_delta: 0 });
  if (p === "size") return { name: IT.presetSize, min_select: 1, max_select: 1, options: [opt(IT.optNormal), opt(IT.optDouble)] };
  if (p === "weight") return { name: IT.presetWeight, min_select: 1, max_select: 1, options: [opt(IT.optHalfKg), opt(IT.optKg)] };
  return { name: IT.extrasName, min_select: 0, max_select: 99, options: [] };
}

export interface MenuItem {
  id: string;
  section_id: string;
  name: string;
  description: string;
  /** **سعرُ البيع** — ما يدفعه الزبون. تحسبه المنصةُ ولا يُكتب هنا. */
  price: number;
  /** **سعرُ الشراء** — ما يضعه المتجر وهو ما يقبضه. */
  merchant_price: number;
  /** قسمُ المنصة — **وفراغُه يعني أن الصنفَ لا يظهر في التصفّح.** */
  platform_section_id: string | null;
  platform_section_name: string;
  image_url: string | null;
  image_thumb_url: string | null;
  available: boolean;
  modifiers: ModifierGroup[];
  /** **عملةُ الصنف** «SYP» أو «USD» — وبالدولار سعرُه `merchant_price_usd`. */
  price_currency?: string;
  merchant_price_usd?: number | null;
}
/** قسمُ منصةٍ كما تراه شاشةُ التحرير — الاسمُ وحدَه يلزم. */
export interface PlatformSection {
  id: string;
  name: string;
}

export interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
  /** **وجهُ القسم** — وسوقٌ يُتصفَّح بالصور لا بالأسماء. (٢٠٢٦-٠٨-٠٧.) */
  image_url?: string | null;
  image_thumb_url?: string | null;
}

/** مسارات النقاط — تختلف بالبوابة والعملية واحدة. */
export interface MenuPaths {
  /** قراءة القائمة: `/api/v1/admin/merchants/{id}/menu` أو `/api/v1/merchant/stores/{id}/menu` */
  menu: (merchantID: string) => string;
  /* ══════════════════════════════════════════════════════════════════
     **ولا مسارَ لأقسام المتجر**
     ══════════════════════════════════════════════════════════════════

     **كانا معلَنين ومعرَّفين في أربع شاشات ولا ينادِيهما هذا المكوّن
     قطّ** — لأنّ الأقسامَ لا تُنشأ: `GetMenu` تردّ أقسامَ السوق التي فيها
     أصناف. (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الأدمنُ هو من يزرع الأقسام»،
     وأكّده ٢٠٢٦-٠٨-١٨.)

     **وعقدٌ يطلب ما لا يُستعمل يُملأ بلا فهم** — تُكتب أربعُ نسخٍ منه
     ثمّ يُقرأ يوماً على أنّ ثمّة ميزةً فيُبنى عليها. */
  items: (merchantID: string) => string;
  item: (itemID: string) => string;
  /**
   * أقسامُ السوق — **وكلُّ بوابةٍ تقرؤها من بابها.**
   *
   * كان العنوانُ مكتوباً في المكوّن: `/api/v1/admin/sections`. **فيُردُّ صاحبُ
   * المتجر ٤٠٣ فيُخفى الحقلُ كلُّه** — فكلُّ ما يضيفه يبقى خارج السوق حتى
   * يصنّفه الأدمنُ بيده.
   *
   * **ومكوّنٌ مشتركٌ لا يعرف بوابتَه**: يأخذها كما يأخذ بقيّة مساراته.
   */
  platformSections: () => string;
  /** **بياناتُ المتجر لسعر صرفه** — اختياريّ؛ بلاه يُحفظ الدولارُ ولا يُعرض ما يساويه بالليرة. */
  merchant?: (merchantID: string) => string;
}

/* ══════════════════════════════════════════════════════════════════
   **الدولارُ كتطبيق المتجر حرفاً** — مرآةُ `lib/store/usd.ts` و`ItemDraft.kt`.
   والخادمُ هو الحَكَم (`catalog.USDToSYP`): ما يُعرض هنا معاينة.
   ══════════════════════════════════════════════════════════════════ */
function usdToSyp(usd: number, rate: number): number {
  return Math.round((usd * 100 * rate) / 100 / 5) * 5;
}
function parseUsd(text: string): number | null {
  const t = text.trim();
  if (!t) return null;
  const v = Number(t.replace("\u066B", ".").replace(",", "."));
  return Number.isFinite(v) && v >= 0 ? v : null;
}
function usdText(v: number | null | undefined): string {
  if (v == null) return "";
  return String(Math.round(v * 100) / 100);
}
function usdInput(v: string, max = 10): string {
  return v.replace(/[^\d.,\u066B]/g, "").slice(0, max);
}

export function MenuManager({
  api,
  paths,
  merchantID,
  title,
  imageUpload,
  thumb,
  mediaUrl,
  /**
   * **إظهارُ سعر البيع في المنصّة بجانب سعر المتجر.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٠.)
   *
   * **للأدمن وحدَه** — وهو من يضع الهامش. **وصاحبُ المتجر يضع سعرَه ويقبض
   * عليه، وما تبيع به المنصّةُ ليس شأنَه.**
   *
   * **وافتراضُه الإخفاء لا الإظهار**: من نسي تمريرَه في شاشةٍ جديدة يُخفي —
   * **وخطأُ الإخفاء يُكتشف بسؤالٍ من موظّف، وخطأُ الإظهار لا يُكتشف أبداً**:
   * يمضي في كلّ متجرٍ قبل أن ينتبه أحد. (وهي القاعدةُ نفسُها في `showSource`
   * بالفاتورة.)
   */
  showSalePrice = false,
  /**
   * **عرضٌ بلا تعديل** — لمن يقرأ القائمةَ ولا يملك تعديلَها (فحصُ المتصفّح
   * ٢٠٢٦-١٠-٠٥: العمليات ترى «صنف جديد» والمحرّكُ يردّ حفظَه ٤٠٣).
   */
  readOnly = false,
}: {
  api: ApiFn;
  paths: MenuPaths;
  merchantID: string;
  showSalePrice?: boolean;
  readOnly?: boolean;
  title?: ReactNode;
  /** رافع الصور — يبقى محقوناً لأنه يعتمد على عميل الرفع الخاص بكل تطبيق */
  imageUpload?: (initialUrl: string | null | undefined, onChange: (id: string | null) => void) => ReactNode;
    thumb?: (url: string | null, alt: string) => ReactNode;
  /**
   * **محوّلُ مسار الوسائط إلى رابطٍ كامل** — محقونٌ كأخواته.
   *
   * (كشفه المالك ٢٠٢٦-٠٨-٠٨: صورةُ القسم مكسورةٌ في القائمة.)
   *
   * **كان مسارُ القسم يُوضع كما جاء** — `/media/…` — **فيطلبه المتصفّح من
   * منفذ البوّابة (3002) لا من المحرّك (8080)** فيردّ 404. **وصورةُ الصنف
   * تمرّ على `thumb` فتسلم، والقسمُ لا يمرّ بشيء.**
   */
  mediaUrl?: (p: string | null | undefined) => string | null;
}) {
  const [sections, setSections] = useState<MenuSection[]>([]);
  /**
   * أقسامُ المنصة — **ما يجعل الصنفَ مرئياً في التصفّح.**
   *
   * **وتُقرأ هنا مرّةً لا في كلّ نافذة**: تُفتح نافذةُ الصنف عشرَ مرّاتٍ في
   * الجلسة، **وعشرةُ نداءاتٍ لقائمةٍ لا تتغيّر حملٌ بلا حاجة.**
   */
  const [platformSections, setPlatformSections] = useState<PlatformSection[]>([]);
  const [error, setError] = useState("");
  /** الصنفُ الذي يُسأل عن حذفه — **وفارغُه يعني لا سؤال.** */
  const [pendingDelete, setPendingDelete] = useState<MenuItem | null>(null);
  const [busy, setBusy] = useState(false);
  /** **القسمُ المفتوح** — وفارغٌ يعني شبكةَ الأقسام. (٢٠٢٦-٠٨-٠٧.) */
  const [openID, setOpenID] = useState<string | null>(null);
      const [editing, setEditing] = useState<{ item: MenuItem | null; sectionId: string } | null>(null);

  useEffect(() => {
    // **وأقسامُ السوق لنافذة التحرير وحدَها** — والعرضُ بلا تعديلٍ لا يفتحها،
    // **ومن لا يملك السوقَ يُردّ نداؤه ٤٠٣.**
    if (readOnly) return;
    api<{ sections: PlatformSection[] }>(paths.platformSections())
      .then((r) => setPlatformSections(r.sections ?? []))
      // **وتعذّرُها لا يمنع تحرير القائمة** — يبقى الحقلُ مخفيّاً والباقي يعمل.
      .catch(() => undefined);
  }, [api, paths, readOnly]);

  const load = useCallback(async () => {
    try {
      /* **وشكلٌ غيرُ متوقَّعٍ لا يُبيّض الشاشة.**

         (كُشف ٢٠٢٦-٠٨-٠٧ بمسبار صورة الصنف: ردٌّ بجسمٍ بدل مصفوفةٍ رمى
         `sections.map is not a function` **فسقطت الصفحةُ كلُّها.**)

         **وهي عائلةُ العطب التي أسقطت بوّابةَ المتجر صباحاً** — قراءةٌ بلا
         حارسٍ في مكوّنٍ يعلو الشجرة. **وقائمةٌ فارغةٌ تُقرأ «لا أصناف»
         ويُعاد التحميل، وشاشةٌ بيضاءُ لا تُقرأ شيئاً.**

         @empty-ok — الفراغُ هنا قرارٌ لا صمت. */
      const res = await api<MenuSection[] | { sections?: MenuSection[] }>(paths.menu(merchantID));
      setSections(Array.isArray(res) ? res : (res?.sections ?? []));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [api, paths, merchantID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggleAvailable(item: MenuItem) {
    try {
      await api(paths.item(item.id), {
        method: "PATCH",
        body: JSON.stringify({ available: !item.available }),
      });
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  /**
   * deleteItem حذفُ صنفٍ — **بتأكيدٍ من المنصة لا من النظام.**
   *
   * كان `confirm()` الأصليّ: **نافذةُ نظامٍ بخطّه ولغته**، تخرج من المنصة
   * كلَّ خروج، **وأزرارُها «موافق/إلغاء» لا تقول ماذا سيقع.**
   *
   * **وهي تُوقف الصفحةَ كلَّها** (`blocking`) — فلا يُرى ما تحتها، ولا
   * تُغلق بـ`Esc` في كلّ متصفّح، **وبعضُ المتصفّحات يعرض «امنع هذه الصفحة
   * من فتح نوافذ» فيُطفئها المستخدمُ فلا يعود يُسأل أصلاً** — فيُحذف الصنفُ
   * بضغطةٍ واحدةٍ بلا سؤال.
   */
  async function deleteItem(item: MenuItem) {
    setBusy(true);
    try {
      await api(paths.item(item.id), { method: "DELETE" });
      setPendingDelete(null);
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  /**
   * **والمفتوحُ يُشتقّ لا يُخزَّن** — قسمٌ خرج آخرُ أصنافه يختفي وحدَه،
   * **ونسخةٌ محفوظةٌ منه تبقى تعرض أصنافاً لم تعد فيه.**
   *
   * **والاسمُ `openSec` لا `open`**: الأخيرةُ دالّةٌ عامّةٌ في المتصفّح
   * (`window.open`)، **فمتغيّرٌ باسمها يُقرأ هي حين يُخطئ الاستدلال.**
   */
  const openSec = openID ? (sections.find((x) => x.id === openID) ?? null) : null;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">{title}</div>

      {error && (
        <Alert>{error}</Alert>
      )}

      {/* ══════════════════════════════════════════════════════════════
          **بابان لا قائمةٌ واحدة**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ صنفٍ تفوت عليه تضيف المنتجاتِ
           بداخله… وتكون نظامَ كروتٍ مثل ما هي بالسوق بلوحة الأدمن».)

          **والأقسامُ من المنصة لا من المتجر**: الأدمنُ يزرعها باسمها وصورتها،
          **وصاحبُ المطعم يجدها جاهزة** فيفتح ما يخصّه ويضيف فيه.

          **ولا يُصنع قسمٌ هنا ولا يُحذف** — وإلّا كتب كلُّ متجرٍ اسماً لِما
          له اسمٌ عند المنصة، **فصار للسوق عشرون اسماً لقسمٍ واحد.**

          **والقسمُ يظهر لأنّ فيه صنفاً** — لا لأنّ أحداً اختاره: فمطعمٌ لا
          يبيع بقالةً لا يرى «بقالة» بلا أن يقول ذلك لأحد. */}
      {!openSec ? (
        sections.length === 0 ? (
          /* ══════════════════════════════════════════════════════════
             **وأوّلُ صنفٍ يُضاف من الفراغ نفسِه**
             ══════════════════════════════════════════════════════════

             (كشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨ على متجرٍ جديدٍ في قاعدةٍ نظيفة.)

             **القسمُ يظهر لأنّ فيه صنفاً، وزرُّ «أضف صنفاً» داخل القسم.**
             فمتجرٌ بلا أصناف لا يرى قسماً، ولا يرى زرّاً، **ولا يستطيع أن
             يضيف أوّلَ صنفٍ أبداً** — والشاشةُ تقول له «أضف قسماً ثم
             أصنافاً» ولا تعطيه باباً.

             **ولا مخرجَ من اللوحتين**: الإدارةُ ترى الشاشةَ نفسَها بالمكوّن
             نفسِه. **فمتجرٌ جديدٌ لا يُباع منه شيءٌ إلى الأبد.**

             **و`section_id` لا يكتبه الخادمُ أصلاً** (`CreateItem`: «ولا
             `section_id` يُكتب») — الشرطُ قسمُ السوق، ويُختار في النافذة.
             فالفراغُ هنا صحيحٌ لا حيلة. */
          <EmptyState
            icon={IconStore}
            title={L.empty}
            action={
              readOnly ? undefined : (
                <Button onClick={() => setEditing({ item: null, sectionId: "" })} className="flex items-center gap-1.5">
                  <IconAdd size={15} />
                  {L.addItem}
                </Button>
              )
            }
          />
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {sections.map((sec) => (
              <button
                key={sec.id}
                type="button"
                onClick={() => setOpenID(sec.id)}
                className="flex flex-col overflow-hidden surface text-start transition-shadow hover:elev-2"
              >
                {/* **الصورةُ أوّلاً — وهي هويّةُ القسم لا زينتُه.**
                    والشبكةُ نفسُها التي في سوق الإدارة، **فمن رآها هناك
                    يعرفها هنا.** */}
                <span className="relative flex aspect-[4/3] items-center justify-center bg-field">
                  {sec.image_url || sec.image_thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={
                        (mediaUrl
                          ? mediaUrl(sec.image_url ?? sec.image_thumb_url)
                          : (sec.image_url ?? sec.image_thumb_url)) as string
                      }
                      alt={sec.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <IconStore size={22} className="text-ink-muted" />
                  )}
                </span>
                <span className="flex min-w-0 flex-1 items-center justify-between gap-2 px-3 py-2.5">
                  <span className="min-w-0 truncate text-sm font-bold">{sec.name}</span>
                  {/* **والعددُ يقول أفيه شيءٌ أم لا** — قبل أن يُفتح. */}
                  <Badge variant="neutral">{fmtNum(sec.items.length)}</Badge>
                </span>
              </button>
            ))}
          </div>
        )
      ) : (
        <section className="surface">
          {/* **ورأسُ القسم بابُ الرجوع** — ومن دخل باباً يخرج منه. */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line-soft px-4 py-3">
            <h2 className="flex min-w-0 items-center gap-2.5 font-bold">
              <Button variant="ghost" onClick={() => setOpenID(null)} aria-label={m.common.back}>
                <IconPrev size={16} />
              </Button>
              {thumb?.(openSec.image_thumb_url ?? null, openSec.name)}
              <span className="truncate">{openSec.name}</span>
            </h2>
            {!readOnly && (
              <Button
                variant="secondary"
                onClick={() => setEditing({ item: null, sectionId: openSec.id })}
                className="flex items-center gap-1.5"
              >
                <IconAdd size={15} />
                {L.addItem}
              </Button>
            )}
          </div>

              {openSec.items.length === 0 ? (
                <p className="p-6 text-center text-sm text-ink-muted">{L.noItems}</p>
              ) : (
                <ul className="divide-y divide-line">
                  {openSec.items.map((item) => (
                    <li key={item.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                      {thumb?.(item.image_thumb_url, item.name)}
                      <div className="min-w-48 flex-1">
                        <p
                          className={`font-medium ${
                            item.available ? "" : "text-ink-muted line-through"
                          }`}
                        >
                          {item.name}
                        </p>
                        {item.description && (
                          <p className="text-sm text-ink-muted">{item.description}</p>
                        )}
                        {item.modifiers.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">
                            {item.modifiers.map((g, i) => (
                              <Badge key={i} variant="neutral">
                                {g.name} ({fmtNum(g.options.length)})
                              </Badge>
                            ))}
                          </div>
                        )}
                      </div>
                      <span className="font-bold text-primary-dark" dir="ltr">
                        {/* ══════════════════════════════════════════════
                            **وسعرُ المنصّة لا يُعرض لصاحب المتجر**
                            ══════════════════════════════════════════════

                            (قرارُ المالك ٢٠٢٦-٠٨-١٠: «هون غلط تُذكر السعر
                             شقد بالمنصّة — خلص، الشخصُ يكتب السعرَ الذي
                             يبيع به المطعمُ بشكلٍ عاديّ».)

                            **كان التعليقُ يفترض أنّ المتجر يرى سعرَه وحدَه**
                            لأنّ `merchant_price === price` حين لا هامش.
                            **والهامشُ الثابتُ يكسر الافتراض**: صار يرى
                            «٩٬٠٠٠ ← ١٩٬٠٠٠» في كلّ سطر.

                            **وما يبيع به المنصّةُ ليس شأنَه**: هو يضع سعرَه
                            ويقبض عليه، **ورقمٌ ثانٍ بجانبه يدعوه إلى مساءلةٍ
                            لا تخصّ اتّفاقَه** — أو إلى رفع سعره ليقتسم
                            الفرق.

                            **والأدمن يراهما** — وهو من يضع الهامش، **ومن
                            يضع رقماً لا يرى أثرَه يضعه أعمى.** */}
                        <Money value={item.merchant_price || item.price} />
                        {showSalePrice && item.merchant_price > 0 && item.price > item.merchant_price && (
                          <span className="ms-1.5 text-2xs text-success">
                            ← {fmtNum(item.price)}
                          </span>
                        )}
                      </span>
                      {/* **ولا «بانتظار المراجعة»** — رُفعت المراجعة (قرارُ المالك
                          ٢٠٢٦-١٠-٠٤): ما يُضاف يظهر فوراً. */}
                      <Badge variant={item.available ? "success" : "warning"}>
                        {item.available ? L.available : L.unavailable}
                      </Badge>
                      {!readOnly && <div className="flex gap-1.5">
                        <Button variant="secondary" onClick={() => toggleAvailable(item)}>
                          {item.available ? L.markUnavailable : L.markAvailable}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => setEditing({ item, sectionId: openSec.id })}
                        >
                          <IconEdit size={15} />
                        </Button>
                        <Button variant="ghost" onClick={() => setPendingDelete(item)}>
                          <IconDelete size={15} className="text-danger" />
                        </Button>
                      </div>}
                    </li>
                  ))}
                </ul>
              )}
        </section>
      )}

      {editing && (
        <ItemModal
          api={api}
          paths={paths}
          merchantID={merchantID}
          platformSections={platformSections}
          editing={editing}
          imageUpload={imageUpload}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
    {/* **وتأكيدُ ما لا يُستدرَك** — بزرٍّ يقول فعلَه لا «موافق»،
        **وتركيزٍ يبدأ على الإلغاء** فمن ضغط `Enter` بلا قراءةٍ لا يحذف. */}
      <Confirm
        open={!!pendingDelete}
        title={L.confirmDeleteItem}
        body={pendingDelete?.name}
        confirmLabel={m.common.delete}
        busy={busy}
        onConfirm={() => pendingDelete && void deleteItem(pendingDelete)}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}

/** **محرّرُ الصنف وحدَه** — يُفتح من خارج القائمة أيضاً (أصنافُ السوق، ٢٠٢٦-١٠-٢٥). */
export function ItemModal({
  api,
  paths,
  merchantID,
  platformSections,
  editing,
  imageUpload,
  note,
  onClose,
  onSaved,
}: {
  api: ApiFn;
  paths: MenuPaths;
  merchantID: string;
  platformSections: PlatformSection[];
  editing: { item: MenuItem | null; sectionId: string };
  /** **سطرٌ فوق النموذج** — «الصنف لمتجر …» حين تُفتح النافذةُ من خارج صفحة المتجر (السوق، ٢٠٢٦-١٠-٢٧). */
  note?: ReactNode;
  imageUpload?: (
    initialUrl: string | null | undefined,
    onChange: (id: string | null) => void,
  ) => ReactNode;
  onClose: () => void;
  onSaved: () => void;
}) {
  const item = editing.item;
  const [name, setName] = useState(item?.name ?? "");
  const [description, setDescription] = useState(item?.description ?? "");
  // **المُحرَّرُ سعرُ الشراء لا سعرُ البيع.**
  //
  // **ولو حُرِّر سعرُ البيع لَضاع الهامشُ في أوّل تعديل**: يفتح المتجرُ الصنفَ
  // فيرى ١٢٬٠٠٠ (وسعرُه ١٠٬٠٠٠)، **فيحفظ فيصير سعرُ شرائه اثني عشر** ويُضاف
  // عليه الهامشُ من جديد — **ورقمٌ يرتفع بكلّ فتحةٍ للنافذة.**
  const [price, setPrice] = useState(item ? String(item.merchant_price || item.price) : "");
  // **وعملةُ الصنف كتطبيق المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «لازم يكون كلّ شي مثل المتجر بالضبط»).
  const [currency, setCurrency] = useState<"SYP" | "USD">(item?.price_currency === "USD" ? "USD" : "SYP");
  const usd = currency === "USD";
  const [priceUsd, setPriceUsd] = useState(usdText(item?.merchant_price_usd ?? null));
  const usdValue = parseUsd(priceUsd);
  const [rate, setRate] = useState<number | null>(null);
  useEffect(() => {
    if (!paths.merchant) return;
    api<{ exchange_rate?: number | null }>(paths.merchant(merchantID))
      .then((r) => setRate(r.exchange_rate ?? null))
      .catch(() => setRate(null)); // @empty-ok — بلا سعرٍ يُحفظ الدولارُ ويقول الخادمُ إن لزم.
  }, [api, paths, merchantID]);
  /* ══════════════════════════════════════════════════════════════════
     **قسمٌ واحدٌ لا اثنان — والثاني كان يردّ ٤٠٣**
     ══════════════════════════════════════════════════════════════════

     (كشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨: أُضيف صنفٌ أوّلُ فنجح، **وثانٍ من داخل
      القسم فرُدّ** — `403 POST /menu/items`.)

     هجرةُ ٠٠٨٤ جعلت `sections` أقسامَ السوق، **فصار حقلا «القسم» و«قسم
     المنصة» يعرضان القائمةَ نفسَها.** والأوّلُ يُرسَل في `section_id`،
     **والخادمُ يفحص ملكيّتَه في جدول أقسام المتاجر** (`ownsSection`) —
     ومعرّفُ قسمِ سوقٍ ليس فيه، **فيُردّ الصنفُ كلُّه.**

     **ولا يقع إلّا من داخل قسم**: الإضافةُ من الفراغ تمرّر فراغاً فتنجح.
     فمتجرٌ يضيف أوّلَ صنفٍ ثمّ **لا يستطيع أن يضيف ثانياً أبداً.**

     **و`section_id` لا يكتبه الخادمُ أصلاً** — فحذفُ الحقل حذفٌ لما لا
     أثرَ له، **والقسمُ الذي فُتحت منه النافذةُ يصير اختياراً مسبقاً لقسم
     السوق** فلا يعيد المتجرُ اختيارَ ما هو فيه. */
  const [psID, setPsID] = useState(item?.platform_section_id ?? editing.sectionId ?? "");
  const [groups, setGroups] = useState<ModifierGroup[]>(
    item?.modifiers.map((g) => ({ ...g, options: g.options.map((o) => ({ ...o, _usd: usdText(o.price_delta_usd ?? null) })) })) ?? [],
  );
  // null = لم تُلمس (لا تُرسل)، "" = إزالة، معرف = صورة جديدة
  const [imageID, setImageID] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function updateGroup(i: number, patch: Partial<ModifierGroup>) {
    setGroups((gs) => gs.map((g, gi) => (gi === i ? { ...g, ...patch } : g)));
  }
  function updateOption(gi: number, oi: number, patch: Partial<ModifierOption>) {
    setGroups((gs) =>
      gs.map((g, i) =>
        i === gi
          ? { ...g, options: g.options.map((o, j) => (j === oi ? { ...o, ...patch } : o)) }
          : g,
      ),
    );
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // **وما تجاوز الحدَّ لا يُرسَل** — الخطأُ ظاهرٌ تحت الحقل.
    if (name.trim().length > ITEM_NAME_MAX) return;
    // **والسعرُ صفرٌ مسموح** (قرارُ المالك ٢٠٢٦-١٠-٢٧، كتطبيق المندوب والخادم) — والفارغُ وحدَه
    // يُردّ: حقلُ الليرة `required`، وفي الدولار يُفحص هنا.
    if (usd && usdValue == null) return setError(IT.vPrice);
    setBusy(true);
    setError("");
    const body = {
      name,
      description,
      price_currency: currency,
      ...(usd ? { price_usd: usdValue } : { price: Number(price) || 0 }),
      // **والفراغُ الصريح يرفع التصنيف** — يقرؤه الخادمُ «ارفع» لا «بلا تغيير».
      platform_section_id: psID,
      modifiers: groups.map((g) => ({
        ...g,
        options: g.options.map(({ _usd, price_delta_usd: _old, ...rest }) => {
          void _old;
          return usd ? { ...rest, price_delta_usd: parseUsd(_usd ?? "") ?? 0 } : rest;
        }),
      })),
      ...(imageID !== null ? { image_media_id: imageID } : {}),
    };
    try {
      if (item) {
        await api(paths.item(item.id), { method: "PATCH", body: JSON.stringify(body) });
      } else {
        await api(paths.items(merchantID), { method: "POST", body: JSON.stringify(body) });
      }
      onSaved();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    /* ══════════════════════════════════════════════════════════════════
       **ونموذجٌ عريضٌ لا طويلٌ بسكرول**
       ══════════════════════════════════════════════════════════════════

       (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الفورم يجب أن يكون عريضاً بدون سكرول».
        وهي السابقةُ نفسُها التي حُسمت في نموذج المتجر ٢٠٢٦-٠٨-٠٨: «فورم طول
        بسكرول مزعج».)

       **الطولُ يُسكرَل والعرضُ لا** — فالشاشةُ عريضةٌ وكان النموذجُ يضيّق
       نفسَه إلى ٤٤٨ بكسلاً ثمّ يمتدّ طولاً حتّى يخرج زرُّ الحفظ.

       **ومن سكرل ليصل إلى «حفظ» لا يرى ما كتبه** — يحفظ على غير يقين. */
    <Modal open onClose={onClose} title={item ? L.editItem : L.addItem} size="lg">
      <form onSubmit={submit} className="space-y-4 p-0.5">
        {note && <div className="rounded-control bg-field px-3 py-2 text-sm">{note}</div>}
        {/* **والاسمُ والسعرُ والقسمُ في صفٍّ واحد** — ثلاثةُ حقولٍ قصيرةٍ
            تملأ العرضَ بدل أن تُطيل النموذج. **وتنهار إلى عمودٍ على
            الجوّال**: صفٌّ من ثلاثةٍ في شاشةٍ ضيّقةٍ يُقصّ لا يُقرأ. */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          {/* **واسمُ الصنف بحدٍّ يُعدّ وهو يُكتب** (قرارُ المالك ٢٠٢٦-١٠-١٤: ٣٠ حرفاً)
              — بطاقةُ السوق سطران، وما زاد يطلع عنها. والمحرّكُ يرفضه أيضاً. */}
          <div>
            <Input
              id="i-name"
              label={L.itemName}
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              error={name.trim().length > ITEM_NAME_MAX ? fill(L.nameOver, { max: ITEM_NAME_MAX, n: name.trim().length }) : undefined}
            />
            {name.trim().length <= ITEM_NAME_MAX && (
              <p className="mt-1 text-xs text-ink-muted tabular-nums" dir="ltr">{`${name.trim().length} / ${ITEM_NAME_MAX}`}</p>
            )}
          </div>
          <div>
          {/* **اللفظُ يقول أيَّ سعرٍ هو.**

              «السعر» وحدَها تحتمل الاثنين، **وصاحبُ المتجر يقرؤها سعرَ البيع**
              فيضع فيها ما يريد أن يدفعه الزبون — **فيُضاف عليه هامشُنا فيصير
              الصنفُ أغلى ممّا قصد**، ويشكو من رقمٍ لم يضعه. */}
          {/* **ليرة أو دولار — كتطبيق المتجر** (٢٠٢٦-١٠-٢٦). */}
          <div role="radiogroup" aria-label={IT.currency} className="mb-1 inline-flex w-fit overflow-hidden rounded-control border border-line">
            {(
              [
                ["SYP", IT.currencySyp],
                ["USD", IT.currencyUsd],
              ] as const
            ).map(([code, label]) => (
              <button
                key={code}
                type="button"
                role="radio"
                aria-checked={currency === code}
                onClick={() => setCurrency(code)}
                className={`px-4 py-1 text-xs font-bold ${currency === code ? "bg-accent text-on-bright" : "bg-field text-ink"}`}
              >
                {label}
              </button>
            ))}
          </div>
          {usd ? (
            <>
              <Input
                id="i-price-usd"
                label={IT.priceUsd}
                inputMode="decimal"
                dir="ltr"
                required
                value={priceUsd}
                onChange={(e) => setPriceUsd(usdInput(e.target.value))}
              />
              <p className={`mt-1 text-2xs ${rate != null && rate > 0 ? "text-ink-muted" : "text-danger"}`}>
                {rate == null || rate <= 0
                  ? IT.usdNeedRate
                  : usdValue == null
                    ? fill(IT.usdRateIs, { rate: usdText(rate) })
                    : fill(IT.usdEquals, { syp: fmtNum(usdToSyp(usdValue, rate)), rate: usdText(rate) })}
              </p>
            </>
          ) : (
            <Input
              id="i-price"
              label={`${L.price} (${m.common.currency})`}
              type="number"
              min="0"
              required
              value={price}
              onChange={(e) => setPrice(e.target.value)}
            />
          )}
          </div>

        {/* **وبلا قسمِ منصةٍ لا يظهر الصنفُ في التصفّح.**

            الزبونُ يتصفّح أقساماً — شاورما، بيتزا — **وصنفٌ بلا قسمٍ صنفٌ لا
            يراه أحد**: موجودٌ في القاعدة، قابلٌ للطلب برابطه، **ولا طريقَ
            إليه.**

            **والتنبيهُ عند الحقل لا في صفحةٍ أخرى**: من يضيف صنفاً يقرؤه في
            اللحظة التي يملك فيها أن يصحّح. */}
        {platformSections.length > 0 && (
          <div>
            <Select
              id="i-psection"
              label={L.platformSection}
              value={psID}
              onChange={(e) => setPsID(e.target.value)}
            >
              <option value="">{L.noPlatformSection}</option>
              {platformSections.map((ps) => (
                <option key={ps.id} value={ps.id}>
                  {ps.name}
                </option>
              ))}
            </Select>
            {!psID && (
              <p className="mt-1 text-2xs text-warning">{L.noPlatformSectionHint}</p>
            )}
          </div>
        )}
        </div>
        <Input
          id="i-desc"
          label={L.itemDescription}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        {imageUpload?.(item?.image_thumb_url, setImageID)}

        <div>
          <span className="mb-1 block text-sm font-medium">{L.modifiers}</span>
          <p className="mb-2 text-xs text-ink-muted">{IT.modHint}</p>
          {/* **مربّعُ صحٍّ لكلّ مجموعةٍ جاهزة** — يفتحها بخياراتها أو يشيلها، كتطبيق المتجر. */}
          <div className="mb-3 flex flex-wrap gap-4">
            {(["size", "weight", "extras"] as const).map((p) => (
              <Checkbox
                key={p}
                id={`preset-${p}`}
                checked={groups.some((g) => presetOf(g) === p)}
                onChange={(e) =>
                  setGroups((gs) =>
                    e.target.checked ? [...gs, presetGroup(p)] : gs.filter((g) => presetOf(g) !== p),
                  )
                }
                label={p === "size" ? IT.presetSize : p === "weight" ? IT.presetWeight : IT.presetExtras}
              />
            ))}
          </div>

          <div className="space-y-3">
            {groups.map((g, gi) => (
              <div key={gi} className="rounded-control border border-line bg-field p-3">
                <div className="mb-2 flex items-end gap-2">
                  <div className="flex-1">
                    {presetOf(g) === "extras" ? (
                      <Input
                        id={`g-name-${gi}`}
                        label={L.groupName}
                        required
                        value={g.name}
                        onChange={(e) => updateGroup(gi, { name: e.target.value })}
                      />
                    ) : (
                      <h4 className="font-bold">{g.name}</h4>
                    )}
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    title={m.common.delete}
                    aria-label={m.common.delete}
                    onClick={() => setGroups((gs) => gs.filter((_, i) => i !== gi))}
                    className="!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full"
                  >
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
                {/* «إلزامية» تُشتق من الحدّ الأدنى لا تُكتب: رقمٌ واحد لا حقلان يتناقضان */}
                <p className="text-xs text-ink-muted">{g.min_select > 0 ? IT.sizeRule : IT.extrasRule}</p>

                <div className="mt-2 space-y-2">
                  {g.options.map((o, oi) => (
                    <div key={oi} className="flex items-center gap-2">
                      <Input
                        id={`o-name-${gi}-${oi}`}
                        placeholder={L.optionName}
                        required
                        value={o.name}
                        onChange={(e) => updateOption(gi, oi, { name: e.target.value })}
                      />
                      <div className="w-32">
                        <Input
                          id={`o-delta-${gi}-${oi}`}
                          placeholder={usd ? IT.optPriceUsd : L.priceDelta}
                          inputMode={usd ? "decimal" : "numeric"}
                          dir="ltr"
                          value={usd ? (o._usd ?? "") : o.price_delta}
                          onChange={(e) =>
                            usd
                              ? updateOption(gi, oi, { _usd: usdInput(e.target.value, 8) })
                              : updateOption(gi, oi, { price_delta: Number(e.target.value) || 0 })
                          }
                        />
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() =>
                          updateGroup(gi, { options: g.options.filter((_, j) => j !== oi) })
                        }
                      >
                        <IconDelete size={14} className="text-danger" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() =>
                      updateGroup(gi, { options: [...g.options, { name: "", price_delta: 0 }] })
                    }
                    className="flex items-center gap-1"
                  >
                    <IconAdd size={13} />
                    {L.addOption}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {error && (
          <Alert>{error}</Alert>
        )}
        <FormActions submit onCancel={onClose} busy={busy} />
      </form>
    </Modal>
  );
}

