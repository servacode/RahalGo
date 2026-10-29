"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الصنف — إضافةٌ وتعديلٌ وحذف، كما في تطبيق المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `ItemEditor.kt` و`ItemDraft.kt` في `:ui`، و`MenuViewModel.saveItem`
 *  في تطبيق المتجر.)
 *
 * # الشكلُ (ملاحظاتُ المالك ٢٠٢٦-١٠-٢٩)
 *
 * - صورةٌ مربّعةٌ كبيرةٌ يميناً، وعن يسارها الاسمُ والوصفُ فوق بعض.
 * - السعرُ وجنبه دائرتا العملة «ل.س» و«$».
 * - قسمُ السوق حقلٌ يفتح نافذةَ مربّعاتٍ بصور الأقسام وبحثٍ و✓ على المختار.
 * - «أضف مشروباتي» مفتاحٌ لكلّ صنف (`store_drinks`، هجرة 0483).
 * - المجموعاتُ الجاهزة مربّعاتٌ متساويةٌ فوق بعض، وكلُّ خيارٍ سطرٌ واحد:
 *   اسمٌ وسعرٌ ومفتاحُ «متوفر / غير متوفر» وحذفٌ أحمر.
 * - كلُّ فعلٍ زرٌّ بأيقونة — والحذفُ بلون الخطر. **ولا سطرَ شرح.**
 *
 * # والسعرُ بعملة الصنف
 *
 * **صنفٌ بالدولار يُرسَل بدولاره** (`price_currency = USD` و`price_usd`)
 * **والليرةُ يحسبها المحرّك** بسعر صرف المتجر مدوَّرةً لأقرب خمس — **وتُعرض
 * هنا قبل الحفظ بالقاعدة نفسِها.** **وخياراتُ صنفٍ دولاريٍّ بالدولار أيضاً**
 * (`price_delta_usd`)، **وخياراتُ صنفٍ ليريٍّ لا تحمل دولاراً** — وإلّا طلب
 * المحرّكُ سعرَ صرفٍ لصنفٍ لا يحتاجه.
 *
 * # والإضافاتُ تُستبدل كلُّها
 *
 * **المحرّكُ يمحو مجموعاتِ الصنف ويكتب المُرسَلة** (`UpdateItem`) — فتُرسَل
 * كلُّها في كلّ حفظ. **و«إلزامي» حدٌّ أدنى واحد، و«يختار أكثر من واحد»
 * حدٌّ أعلى ٩٩** — مفتاحان كالتطبيق لا رقمان يُكتبان.
 *
 * # والإنشاءُ مرّةً مهما أُعيد
 *
 * **مفتاحُ المحاولة** (`Idempotency-Key`، `Attempt.MERCHANT_ITEM` في التطبيق)
 * يبقى حتّى يُحسم الطلب — **فانقطاعٌ بعد الضغط لا يصنع صنفين.**
 */

import { useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  FormActions,
  IconAdd,
  IconCheck,
  IconChevronDown,
  IconClose,
  IconCopy,
  IconDelete,
  IconSearch,
  ImageUpload,
  Input,
  ITEM_CROP_FRAMES,
  MediaThumb,
  Modal,
  Switch,
  Textarea,
} from "@rahalgo/ui";
import { ColumnLabel, CurrencyCircles, RowField, StickyBar, TagChip, ToolHeader } from "./ToolParts";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type {
  MenuItem,
  ModifierGroup,
  SectionRef,
  StorePresets,
} from "@/lib/store/types";
import { parseUsd, usdInput, usdText, usdToSyp } from "@/lib/store/usd";
import {
  DEFAULT_BASE_GRAMS,
  presetBaseGrams,
  QUICK_GRAMS,
  priceModeOf,
  weightName,
  weightPrice,
  type PriceMode,
} from "@/lib/store/optionModes";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const I = m.store.item;

const SLOT = "merchant_item";
/** **مرآةُ `MAX_EDGE` في `ImagePick.kt`** — الصورةُ تُصغَّر قبل الرفع كما في التطبيق. */
const IMAGE_EDGE = 1600;
/** **حدودُ الكتابة كالتطبيق** — والمحرّكُ يحرس فوقها (`catalog.MaxItemDescription`). */
const NAME_TYPE_MAX = 120;
const DESC_MAX = 500;
/** **العدّادُ يظهر حين يبقى هذا أو أقلّ** — لا قبله. */
const NAME_NEAR = 10;

interface DraftOption {
  id?: string;
  name: string;
  price_delta: number;
  usd: string;
  /** **خيارٌ خلصان** والصنفُ متوفّر — يُرسَل `available` مع كلّ خيار. */
  available: boolean;
  /** **وزنُ الخيار بالغرام** لمجموعة «حسب الوزن». */
  grams?: number | null;
}
interface DraftGroup {
  id?: string;
  name: string;
  min_select: number;
  max_select: number;
  options: DraftOption[];
  /** **طريقةُ السعر** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — مرآةُ `ItemEditor.kt`. */
  price_mode: PriceMode;
  /** الوزنُ الذي يخصّه سعرُ الصنف — لـ«weight» وحدَها، وفراغُه ١٠٠ غ. */
  base_grams: number | null;
}

/**
 * **مجموعاتٌ جاهزةٌ بمربّع صح** — كانت ثلاثاً ثابتةً لكلّ متجر (٢٠٢٦-١٠-١٩)، **وصارت
 * من اللوحة لكلّ تصنيف** (قرارُ المالك ٢٠٢٦-١٠-٢٧): `GET /stores/{id}/option-presets`
 * وما `chosen` منها يظهر هنا. **والمربّعُ ينسخ المجموعةَ بخياراتها إلى الصنف**،
 * **وتُعرف المجموعةُ باسمها.** **وإن تعذّر الجلبُ رجعت الثلاثُ القديمة** — لا محرّرَ بلا مجموعات.
 * والمشروباتُ ليست هنا — تنضاف من قائمتها (`DrinksEditor`)، **ويُطفئها مفتاحُ «أضف مشروباتي»**.
 */
interface PresetDef {
  key: string;
  name: string;
  min_select: number;
  max_select: number;
  options: { name: string; price_delta: number; grams?: number | null }[];
  price_mode: PriceMode;
}
function fallbackPresets(): PresetDef[] {
  const opts = (...names: string[]) =>
    names.map((name) => ({ name, price_delta: 0 }));
  return [
    {
      key: "size",
      name: I.presetSize,
      min_select: 1,
      max_select: 1,
      options: opts(I.optNormal, I.optDouble),
      price_mode: "priced",
    },
    {
      key: "weight",
      name: I.presetWeight,
      min_select: 1,
      max_select: 1,
      options: opts(I.optHalfKg, I.optKg),
      price_mode: "priced",
    },
    {
      key: "extras",
      name: I.extrasName,
      min_select: 0,
      max_select: 99,
      options: [],
      price_mode: "priced",
    },
  ];
}
const sameName = (a: string, b: string) => a.trim() === b.trim();
/**
 * **ليرةٌ ⇒ دولارٌ نصّاً لخيار** — مرآةُ `forItemCurrency` في `ItemDraft.kt`
 * (تدقيقُ ٢٠٢٦-١٠-٢٨). **وبلا سعر صرفٍ يبقى فارغاً و`lost`** — لا صفرَ بصمت.
 */
function sypToUsdText(
  delta: number,
  rate: number | null,
): { text: string; lost: boolean } {
  if (delta === 0) return { text: "", lost: false };
  if (rate != null && rate > 0) return { text: usdText(delta / rate), lost: false };
  return { text: "", lost: true };
}
function presetGroup(
  p: PresetDef,
  usd: boolean,
  rate: number | null,
): { group: DraftGroup; lost: boolean } {
  let lost = false;
  const mode = p.price_mode;
  const group: DraftGroup = {
    name: p.name,
    min_select: p.min_select,
    max_select: p.max_select,
    price_mode: mode,
    base_grams: mode === "weight" ? presetBaseGrams(p.options.flatMap((o) => (o.grams ? [o.grams] : []))) : null,
    options: p.options.map((o) => {
      if (mode !== "priced") {
        const g = mode === "weight" ? (o.grams ?? null) : null;
        return {
          name: g ? weightName(g) : o.name,
          price_delta: 0,
          usd: "",
          available: true,
          grams: g,
        };
      }
      const c = usd ? sypToUsdText(o.price_delta, rate) : { text: "", lost: false };
      if (c.lost) lost = true;
      return {
        name: o.name,
        price_delta: o.price_delta,
        usd: c.text,
        available: true,
      };
    }),
  };
  return { group, lost };
}

/**
 * **تبديلُ عملة الصنف يحوّل فروقَ خياراته** — مرآةُ `withCurrency` في `ItemDraft.kt`
 * (تدقيقُ ٢٠٢٦-١٠-٢٨). كانت تصير صفراً بصمت.
 */
function convertGroups(
  groups: DraftGroup[],
  to: "SYP" | "USD",
  rate: number | null,
): { groups: DraftGroup[]; lost: boolean } {
  let lost = false;
  // **ولا فرقَ يُحوَّل في «بدون سعر» و«حسب الوزن»** — الخادمُ يحسبهما.
  const out = groups.map((g) => g.price_mode !== "priced" ? g : ({
    ...g,
    options: g.options.map((o) => {
      if (to === "USD") {
        const c = sypToUsdText(o.price_delta, rate);
        if (c.lost) lost = true;
        return { ...o, usd: c.text };
      }
      const u = parseUsd(o.usd);
      if (u == null || u === 0) return { ...o, price_delta: 0 };
      if (rate != null && rate > 0) return { ...o, price_delta: usdToSyp(u, rate) };
      // **وفرقُه بالليرة من المحرّك يبقى** إن وُجد — وإلّا يُقال.
      if (o.price_delta <= 0) lost = true;
      return o;
    }),
  }));
  return { groups: out, lost };
}

function toDraft(groups: ModifierGroup[] | undefined): DraftGroup[] {
  return (groups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    min_select: g.min_select,
    max_select: g.max_select,
    price_mode: priceModeOf(g.price_mode),
    base_grams: g.base_grams ?? null,
    options: g.options.map((o) => ({
      id: o.id,
      name: o.name,
      price_delta: o.price_delta,
      usd: usdText(o.price_delta_usd ?? null),
      available: o.available !== false,
      grams: o.grams ?? null,
    })),
  }));
}

export default function ItemEditor({
  item,
  sections,
  initialSection = "",
}: {
  item: MenuItem | null;
  sections: SectionRef[];
  initialSection?: string;
}) {
  const { store, itemNameMax } = useStore();
  const router = useRouter();
  const toast = useShopToast();
  const rate = store?.exchange_rate ?? null;

  const [name, setName] = useState(item?.name ?? "");
  const [description, setDescription] = useState(item?.description ?? "");
  const [currency, setCurrency] = useState<"SYP" | "USD">(
    item?.price_currency === "USD" ? "USD" : "SYP",
  );
  // **والصفرُ يُعرض صفراً** (قرارُ المالك ٢٠٢٦-١٠-٢٨: «مسموح للكل»).
  const [price, setPrice] = useState(
    item && item.price >= 0 ? String(item.price) : "",
  );
  /** **فرقُ خيارٍ لم يُحوَّل لغياب سعر الصرف** — يُقال ولا يُصفَّر بصمت. */
  const [rateLost, setRateLost] = useState(false);
  const [priceUsd, setPriceUsd] = useState(
    usdText(item?.merchant_price_usd ?? null),
  );
  const [section, setSection] = useState(
    item?.platform_section_id ?? initialSection,
  );
  /** `null` لم تُلمس (لا تُرسَل) · `""` إزالة · معرّفٌ صورةٌ جديدة. */
  const [imageId, setImageId] = useState<string | null>(null);
  const [groups, setGroups] = useState<DraftGroup[]>(() =>
    toDraft(item?.modifiers),
  );
  /** **«أضف مشروباتي»** — وافتراضُه نعم كما كان. */
  const [storeDrinks, setStoreDrinks] = useState(item?.store_drinks !== false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [askDelete, setAskDelete] = useState(false);
  /** **في المسوّدة ما لم يُحفظ؟** — فيُطفأ النسخُ حتّى يُحفظ. */
  const [touched, setTouched] = useState(false);
  /** `null` تُجلب بعد · وتعذّرُها يرجّع الثلاثَ القديمة ويقول ذلك. */
  const [presets, setPresets] = useState<PresetDef[] | null>(null);
  const [presetsFailed, setPresetsFailed] = useState(false);
  const storeId = store?.id ?? "";

  useEffect(() => {
    if (!storeId) return;
    let live = true;
    api<StorePresets>(`/api/v1/merchant/stores/${storeId}/option-presets`)
      .then((r) => {
        if (!live) return;
        setPresets(
          (r.presets ?? [])
            .filter((p) => p.chosen)
            .map((p) => ({
              key: p.id,
              name: p.name,
              min_select: p.min_select,
              max_select: p.max_select,
              options: p.options ?? [],
              price_mode: priceModeOf(p.price_mode),
            })),
        );
      })
      .catch(() => {
        if (!live) return;
        setPresets(fallbackPresets());
        setPresetsFailed(true);
      });
    return () => {
      live = false;
    };
  }, [storeId]);

  const isPreset = (g: DraftGroup) =>
    (presets ?? []).some((p) => sameName(p.name, g.name));

  const usd = currency === "USD";
  const nameLen = name.trim().length;
  const nameOver = nameLen > itemNameMax;
  const usdValue = parseUsd(priceUsd);
  /** **سعرُ الصنف بالليرة** — لمعاينة أسعار الأوزان؛ والخادمُ يحسبها ويحفظها. */
  const itemSyp: number | null = usd
    ? usdValue != null && rate != null && rate > 0
      ? usdToSyp(usdValue, rate)
      : null
    : price.trim() !== "" && Number.isFinite(Number(price))
      ? Number(price)
      : null;

  const edit = (fn: (g: DraftGroup[]) => DraftGroup[]) => {
    setGroups(fn);
    setError("");
    setTouched(true);
  };
  /** **كلُّ تعديلٍ يُطفئ النسخ ويمسح الخطأ.** */
  const dirty = () => {
    setTouched(true);
    setError("");
  };
  const setGroup = (gi: number, p: Partial<DraftGroup>) =>
    edit((gs) => gs.map((g, i) => (i === gi ? { ...g, ...p } : g)));
  const setOption = (gi: number, oi: number, p: Partial<DraftOption>) =>
    edit((gs) =>
      gs.map((g, i) =>
        i === gi
          ? {
              ...g,
              options: g.options.map((o, j) => (j === oi ? { ...o, ...p } : o)),
            }
          : g,
      ),
    );

  async function save() {
    if (!store) return;
    if (!name.trim()) return setError(I.vName);
    if (nameOver) return;
    // **والصفرُ مسموح** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — والفارغُ لا.
    const syp = usd || price.trim() === "" ? null : Number(price.trim());
    if (
      (usd && usdValue == null) ||
      (!usd && !(syp != null && Number.isInteger(syp) && syp >= 0))
    ) {
      return setError(I.vPrice);
    }
    if (!section) return setError(I.vSection);
    if (groups.some((g) => !g.name.trim())) return setError(I.vGroupName);
    // **والخيارُ بلا اسمٍ يُسقط** — مرآةُ `modifiersForSave` (تدقيقُ ٢٠٢٦-١٠-٢٨).
    const named = groups.map((g) => ({
      ...g,
      options: g.options.filter((o) =>
        g.price_mode === "weight" ? (o.grams ?? 0) > 0 : o.name.trim() !== "",
      ),
    }));
    // **وإلزاميّةٌ بلا خيارٍ متوفّر** — الزبونُ ما بيقدر يطلب الصنف أصلاً.
    const stuck = named.find(
      (g) => g.min_select > 0 && !g.options.some((o) => o.available),
    );
    if (stuck) return setError(fill(I.vRequiredEmpty, { name: stuck.name.trim() }));
    const modifiers = named.map((g) => ({
      ...(g.id ? { id: g.id } : {}),
      name: g.name.trim(),
      min_select: g.min_select,
      max_select: g.max_select,
      price_mode: g.price_mode,
      ...(g.price_mode === "weight"
        ? { base_grams: g.base_grams ?? DEFAULT_BASE_GRAMS }
        : {}),
      options: g.options.map((o) =>
        g.price_mode === "weight"
          ? {
              ...(o.id ? { id: o.id } : {}),
              name: weightName(o.grams ?? 0),
              grams: o.grams,
              price_delta: 0,
              available: o.available,
            }
          : {
              ...(o.id ? { id: o.id } : {}),
              name: o.name.trim(),
              price_delta: g.price_mode === "none" ? 0 : o.price_delta,
              available: o.available,
              ...(usd && g.price_mode === "priced"
                ? { price_delta_usd: parseUsd(o.usd) ?? 0 }
                : {}),
            },
      ),
    }));
    const body = {
      name: name.trim(),
      description: description.trim(),
      platform_section_id: section,
      modifiers,
      store_drinks: storeDrinks,
      price_currency: currency,
      ...(usd ? { price_usd: usdValue } : { price: syp }),
      ...(imageId !== null ? { image_media_id: imageId } : {}),
    };
    setBusy(true);
    setError("");
    try {
      if (item) {
        await api(`/api/v1/merchant/menu/items/${item.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
      } else {
        await api(`/api/v1/merchant/stores/${store.id}/menu/items`, {
          method: "POST",
          body: JSON.stringify(body),
          headers: { "Idempotency-Key": attemptKey(SLOT) },
        });
        clearAttempt(SLOT);
      }
      toast(I.saved, "success");
      router.push("/store/menu");
    } catch (e) {
      if (!item && isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!item) return;
    setBusy(true);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}`, { method: "DELETE" });
      toast(I.deleted, "success");
      router.push("/store/menu");
    } catch (e) {
      setError(errorText(e));
      setAskDelete(false);
    } finally {
      setBusy(false);
    }
  }

  /** **«نسخ الصنف»** — صنفٌ بالصورة والقسم والخيارات، ويُفتح للتعديل (٢٠٢٦-١٠-٢٧). */
  async function duplicate() {
    if (!item || busy) return;
    setBusy(true);
    try {
      const r = await api<{ id: string }>(`/api/v1/merchant/menu/items/${item.id}/duplicate`, { method: "POST" });
      toast(m.store.menu.duplicated, "success");
      router.push(`/store/menu/${r.id}`);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const readOnly = store?.status === "suspended";
  const current = sections.find((s) => s.id === section) ?? null;
  const removeGroup = (gi: number) => edit((gs) => gs.filter((_, i) => i !== gi));
  const removeOption = (gi: number, oi: number) =>
    setGroup(gi, { options: (groups[gi]?.options ?? []).filter((_, j) => j !== oi) });

  return (
    <form
      className="mx-auto flex w-full max-w-2xl flex-col gap-4"
      data-testid="item-editor"
      onSubmit={(e) => {
        e.preventDefault();
        void save();
      }}
    >
      <ToolHeader title={item ? I.edit : I.new} href="/store/menu" />

      {/* ── ١ · الصورةُ يميناً، والاسمُ والوصفُ عن يسارها ── */}
      <div className="flex items-start gap-3">
        <ImageUpload
          kind="menu_item"
          label={I.image}
          initialUrl={item?.image_thumb_url ?? item?.image_url}
          onChange={(id) => {
            setImageId(id);
            dirty();
          }}
          api={api}
          mediaUrl={mediaUrl}
          errorText={errorText}
          path="/api/v1/merchant/media"
          maxEdge={IMAGE_EDGE}
          crop
          cropFrames={ITEM_CROP_FRAMES}
          fit
          tile={{ pick: I.pickImage, change: I.photoChange, clear: I.clearImage }}
        />
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <div>
            <Input
              id="item-name"
              label={I.name}
              value={name}
              onChange={(e) => {
                setName(e.target.value.slice(0, NAME_TYPE_MAX));
                dirty();
              }}
              error={nameOver ? fill(I.nameOver, { max: itemNameMax, n: nameLen }) : undefined}
            />
            {/* **العدّادُ قربَ الحدّ وحدَه** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — لا سطرٌ دائم. */}
            {!nameOver && nameLen >= itemNameMax - NAME_NEAR && (
              <p className="mt-1 text-xs tabular-nums text-ink-muted" dir="ltr">{`${nameLen} / ${itemNameMax}`}</p>
            )}
          </div>
          <Textarea
            id="item-desc"
            label={I.desc}
            rows={3}
            value={description}
            onChange={(e) => {
              setDescription(e.target.value.slice(0, DESC_MAX));
              dirty();
            }}
          />
        </div>
      </div>

      {/* ── ٢ · السعرُ وجنبه دائرتا العملة ── */}
      <div className="flex flex-col gap-1">
        <div className="flex items-end gap-3">
          {usd ? (
            <Input
              id="item-price-usd"
              label={I.priceUsd}
              inputMode="decimal"
              dir="ltr"
              value={priceUsd}
              onChange={(e) => {
                setPriceUsd(usdInput(e.target.value));
                dirty();
              }}
              wrapperClassName="min-w-0 flex-1"
            />
          ) : (
            <Input
              id="item-price"
              label={I.price}
              inputMode="numeric"
              dir="ltr"
              value={price}
              onChange={(e) => {
                setPrice(e.target.value.replace(/\D/g, "").slice(0, 12));
                dirty();
              }}
              wrapperClassName="min-w-0 flex-1"
            />
          )}
          <CurrencyCircles
            label={I.currency}
            value={currency}
            syp={I.currencySyp}
            usd={I.currencyUsdShort}
            disabled={busy}
            onChange={(code) => {
              // **وفروقُ الخيارات تُحوَّل معه** — مرآةُ `withCurrency`.
              const c = convertGroups(groups, code, rate);
              if (c.lost) setRateLost(true);
              setGroups(c.groups);
              setCurrency(code);
              dirty();
            }}
          />
        </div>
        {usd && (
          <p className={`text-xs ${rate != null && rate > 0 ? "text-ink-muted" : "text-danger"}`}>
            {rate == null || rate <= 0
              ? I.usdNeedRate
              : usdValue == null
                ? fill(I.usdRateIs, { rate: usdText(rate) })
                : fill(I.usdEquals, { syp: fmtMoney(usdToSyp(usdValue, rate)), rate: usdText(rate) })}
          </p>
        )}
      </div>

      {/* ── ٣ · قسمُ السوق — مربّعاتٌ بصورها في الصفحة نفسِها، ثلاثةٌ بكلّ سطر (طلبُ المالك ٢٠٢٦-١٠-٢٩) ── */}
      <div className="flex flex-col gap-2" data-testid="item-section">
        <span className="text-sm font-bold">{I.section}</span>
        <SectionPicker
          sections={sections}
          selected={section}
          disabled={busy}
          onPick={(id) => {
            setSection(id);
            dirty();
          }}
        />
      </div>

      {/* ── ٤ · الإضافاتُ والخيارات ── */}
      <section className="flex flex-col gap-3" aria-label={I.modTitle}>
        <h3 className="heading-card">{I.modTitle}</h3>
        {/* **«أضف مشروباتي»** (هجرة 0483) — أتظهر قائمةُ المشروبات مع هذا الصنف. */}
        <div className="rounded-control bg-accent-tint px-3 py-2" data-testid="item-drinks">
          <Switch
            checked={storeDrinks}
            disabled={busy}
            onChange={(v) => {
              setStoreDrinks(v);
              dirty();
            }}
            label={I.drinksSwitch}
          />
        </div>
        {presetsFailed && <p className="text-xs text-warning">{I.presetsFailed}</p>}
        {/* **مجموعاتٌ جاهزةٌ — مربّعاتٌ متساويةٌ فوق بعض.** */}
        <div className="flex flex-col gap-2" data-testid="item-presets">
          {(presets ?? []).map((p) => {
            const on = groups.some((g) => sameName(g.name, p.name));
            const values = p.options.map((o) => o.name).join(" · ");
            return (
              <button
                key={p.key}
                type="button"
                role="checkbox"
                aria-checked={on}
                disabled={busy}
                data-testid="item-preset"
                onClick={() => {
                  if (!on) {
                    const r = presetGroup(p, usd, rate);
                    if (r.lost) setRateLost(true);
                    edit((gs) => [...gs, r.group]);
                  } else {
                    edit((gs) => gs.filter((g) => !sameName(g.name, p.name)));
                  }
                }}
                className={`flex h-[3.75rem] w-full items-center gap-3 rounded-control border px-3 text-start transition-colors disabled:opacity-60 ${
                  on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
                }`}
              >
                <span
                  className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${
                    on ? "bg-accent text-on-bright" : "bg-accent-tint text-accent-text"
                  }`}
                  aria-hidden
                >
                  {on ? <IconCheck size={16} /> : <IconAdd size={16} />}
                </span>
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-bold text-ink">{p.name}</span>
                  {values && <span className="truncate text-xs text-ink-muted">{values}</span>}
                </span>
              </button>
            );
          })}
        </div>
        <Button
          type="button"
          variant="secondary"
          className="inline-flex w-fit items-center gap-1.5 font-bold"
          data-testid="item-custom-group"
          disabled={busy}
          onClick={() =>
            edit((gs) => [
              ...gs,
              { name: I.extrasName, min_select: 0, max_select: 99, options: [], price_mode: "priced", base_grams: null },
            ])
          }
        >
          <IconAdd size={18} />
          {I.customGroup}
        </Button>

        {groups.map((g, gi) => (
          <div key={gi} className="surface flex flex-col gap-2 rounded-card border border-line p-3" data-testid="item-group">
            {/* ── الرأس: اسمٌ ورقاقتان وحذفٌ أحمر ── */}
            <div className="flex items-start gap-2">
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <span className="truncate font-bold">{g.name.trim() || I.customUntitled}</span>
                <span className="flex flex-wrap gap-1">
                  {g.price_mode === "none" ? (
                    <TagChip text={I.chipNone} tone="muted" />
                  ) : g.price_mode === "weight" ? (
                    <TagChip text={I.chipWeight} tone="warning" />
                  ) : (
                    <TagChip text={I.chipPriced} tone="accent" />
                  )}
                  <TagChip text={g.min_select > 0 ? I.chipOne : I.chipAny} tone="muted" />
                </span>
              </div>
              <button
                type="button"
                aria-label={I.removeGroup}
                title={I.removeGroup}
                disabled={busy}
                onClick={() => removeGroup(gi)}
                data-testid="item-group-remove"
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control border border-line text-ink-muted hover:bg-row-hover hover:text-ink disabled:opacity-50"
              >
                <IconDelete size={18} />
              </button>
            </div>
            {/* ── المجموعةُ الخاصّة: اسمُها وقاعدتُها بيده ── */}
            {!isPreset(g) && (
              <>
                <Input
                  id={`g-name-${gi}`}
                  label={I.groupName}
                  value={g.name}
                  onChange={(e) => setGroup(gi, { name: e.target.value })}
                />
                <Switch
                  checked={g.min_select > 0}
                  disabled={busy}
                  onChange={(one) => setGroup(gi, one ? { min_select: 1, max_select: 1 } : { min_select: 0, max_select: 99 })}
                  label={I.ruleSwitch}
                />
              </>
            )}

            {g.price_mode === "weight" ? (
              <>
                <div className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 text-sm">{I.weightBaseLabel}</span>
                  <RowField
                    id={`g-base-${gi}`}
                    value={g.base_grams ? String(g.base_grams) : ""}
                    onChange={(v) => setGroup(gi, { base_grams: Number(v.replace(/\D/g, "").slice(0, 6)) || null })}
                    placeholder={String(DEFAULT_BASE_GRAMS)}
                    label={I.weightBase}
                    numeric
                    suffix={I.weightUnit}
                    disabled={busy}
                    className="w-24 shrink-0"
                  />
                </div>
                {g.options.map((o, oi) => (
                  <OptionRow
                    key={oi}
                    available={o.available}
                    busy={busy}
                    onAvailable={(v) => setOption(gi, oi, { available: v })}
                    onDelete={() => removeOption(gi, oi)}
                  >
                    <RowField
                      id={`o-grams-${gi}-${oi}`}
                      value={o.grams ? String(o.grams) : ""}
                      onChange={(v) => {
                        const n = Number(v.replace(/\D/g, "").slice(0, 7)) || 0;
                        setOption(gi, oi, { grams: n || null, name: n ? weightName(n) : "" });
                      }}
                      placeholder="250"
                      label={I.weightGrams}
                      numeric
                      suffix={I.weightUnit}
                      disabled={busy}
                      className="w-24 shrink-0"
                    />
                    <span className="flex min-w-0 flex-1 flex-col text-sm">
                      {o.grams ? (
                        <>
                          <span className="truncate font-bold">{weightName(o.grams)}</span>
                          {itemSyp != null && (
                            <span className="truncate text-xs text-accent-text">
                              {fill(I.weightPrice, { price: fmtMoney(weightPrice(itemSyp, o.grams, g.base_grams)) })}
                            </span>
                          )}
                        </>
                      ) : null}
                    </span>
                  </OptionRow>
                ))}
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-ink-muted">{I.weightQuick}</span>
                  {QUICK_GRAMS.map((q) => {
                    const has = g.options.some((o) => o.grams === q);
                    return (
                      <button
                        key={q}
                        type="button"
                        aria-pressed={has}
                        disabled={busy}
                        onClick={() =>
                          has
                            ? undefined
                            : setGroup(gi, {
                                options: [...g.options, { name: weightName(q), grams: q, price_delta: 0, usd: "", available: true }],
                              })
                        }
                        className={`inline-flex min-h-10 items-center gap-1 rounded-full border px-3 text-sm ${
                          has ? "border-accent bg-accent-tint font-bold text-accent-text" : "border-line bg-surface text-ink hover:bg-row-hover"
                        }`}
                      >
                        {has ? <IconCheck size={14} /> : <IconAdd size={14} />}
                        {weightName(q)}
                      </button>
                    );
                  })}
                </div>
              </>
            ) : (
              <>
                {g.price_mode === "priced" && g.options.length > 0 && (
                  <div className="flex items-center gap-1.5">
                    <ColumnLabel text={I.optionName} className="min-w-0 flex-1" />
                    <ColumnLabel text={usd ? I.optPriceUsd : I.optPrice} className="w-24 shrink-0" />
                    <span className="w-[6.5rem] shrink-0" />
                  </div>
                )}
                {g.options.map((o, oi) => (
                  <OptionRow
                    key={oi}
                    available={o.available}
                    busy={busy}
                    onAvailable={(v) => setOption(gi, oi, { available: v })}
                    onDelete={() => removeOption(gi, oi)}
                  >
                    <RowField
                      id={`o-name-${gi}-${oi}`}
                      value={o.name}
                      onChange={(v) => setOption(gi, oi, { name: v })}
                      placeholder={I.optionName}
                      label={I.optionName}
                      disabled={busy}
                      className="min-w-0 flex-1"
                    />
                    {g.price_mode === "priced" && (
                      <RowField
                        id={`o-price-${gi}-${oi}`}
                        value={usd ? o.usd : o.price_delta === 0 ? "" : String(o.price_delta)}
                        onChange={(v) =>
                          usd
                            ? setOption(gi, oi, { usd: usdInput(v, 8) })
                            : setOption(gi, oi, { price_delta: Number(v.replace(/\D/g, "").slice(0, 9)) || 0 })
                        }
                        placeholder="0"
                        label={usd ? I.optPriceUsd : I.optPrice}
                        numeric={!usd}
                        decimal={usd}
                        disabled={busy}
                        className="w-24 shrink-0"
                      />
                    )}
                  </OptionRow>
                ))}
              </>
            )}
            <Button
              type="button"
              variant="secondary"
              className="inline-flex w-fit items-center gap-1.5"
              disabled={busy}
              onClick={() => setGroup(gi, { options: [...g.options, { name: "", price_delta: 0, usd: "", available: true }] })}
            >
              <IconAdd size={16} />
              {g.price_mode === "weight" ? I.weightAdd : I.addOption}
            </Button>
          </div>
        ))}
        {rateLost && <p className="text-xs text-danger">{I.usdNoRate}</p>}
      </section>

      {/* ── ٥ · النسخُ والحذف في آخر الصفحة — والحفظُ في شريطٍ ثابتٍ تحتها ── */}
      {item && !readOnly && (
        <>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="secondary"
              className="inline-flex flex-1 items-center justify-center gap-1.5 font-bold"
              disabled={busy || touched}
              onClick={() => void duplicate()}
              data-testid="item-duplicate"
            >
              <IconCopy size={18} />
              {I.duplicate}
            </Button>
            <Button
              type="button"
              variant="secondary"
              className="inline-flex flex-1 items-center justify-center gap-1.5 border-danger-edge font-bold text-danger"
              disabled={busy}
              onClick={() => setAskDelete(true)}
              data-testid="item-delete"
            >
              <IconDelete size={18} />
              {I.delete}
            </Button>
          </div>
          <Confirm
            open={askDelete}
            title={I.delete}
            body={I.deleteAsk}
            confirmLabel={I.delete}
            busy={busy}
            onConfirm={() => void remove()}
            onCancel={() => setAskDelete(false)}
          />
        </>
      )}

      {/* **«حفظ / إلغاء» ثابتان أسفلَ الشاشة** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — أكثرُ
          ما يُعدَّل السعرُ في الأعلى، **فلا يُنزَل إلى آخر الصفحة ليُحفظ.** */}
      {(error || !readOnly) && (
        <StickyBar>
          {error && <Alert>{error}</Alert>}
          {!readOnly && (
            <FormActions
              submit
              onCancel={() => router.push("/store/menu")}
              busy={busy || nameOver}
              saveIcon={<IconCheck size={18} />}
              cancelIcon={<IconClose size={18} />}
            />
          )}
        </StickyBar>
      )}
    </form>
  );
}

/**
 * **سطرُ خيارٍ واحد** — حقولُه ثمّ مفتاحُ التوفّر ثمّ حذفٌ رماديّ.
 * (**والأحمرُ لحذف الصنف وحدَه**، و«غير متوفر» رماديٌّ لا أحمر — ليس غلطاً.)
 *
 * **«نمرة ٤٢ خلصت» — يطفئها وحدَها** والصنفُ يبقى يُباع، وعند الزبون تطلع
 * رماديّةً لا تُختار.
 */
function OptionRow({
  available,
  busy,
  onAvailable,
  onDelete,
  children,
}: {
  available: boolean;
  busy: boolean;
  onAvailable: (v: boolean) => void;
  onDelete: () => void;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center gap-1.5" data-testid="item-option">
      {children}
      <span className="flex w-16 shrink-0 flex-col items-center gap-0.5">
        <button
          type="button"
          role="switch"
          aria-checked={available}
          aria-label={available ? I.optAvailable : I.unavailable}
          disabled={busy}
          onClick={() => onAvailable(!available)}
          className={`relative inline-flex h-5 w-9 items-center rounded-badge border transition-colors disabled:opacity-50 ${
            available ? "border-success-edge bg-success-fill" : "border-line bg-field"
          }`}
        >
          <span
            className={`absolute h-3.5 w-3.5 rounded-badge transition-[inset-inline-start] duration-200 ${
              available ? "start-[1.15rem] bg-success" : "start-0.5 bg-ink-muted"
            }`}
          />
        </button>
        <span className={`text-2xs font-bold ${available ? "text-success" : "text-ink-muted"}`}>
          {available ? I.optAvailable : I.unavailable}
        </span>
      </span>
      <button
        type="button"
        aria-label={I.optDelete}
        title={I.optDelete}
        disabled={busy}
        onClick={onDelete}
        className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover hover:text-ink disabled:opacity-50"
      >
        <IconDelete size={18} />
      </button>
    </div>
  );
}

/**
 * **أقسامُ السوق في الصفحة نفسِها** — مربّعاتٌ بصورة كلّ قسمٍ واسمه، وبحث، **و✓ على المختار.**
 * (مرآةُ `SectionPickerDialog` في `ItemEditor.kt`.)
 */
function SectionPicker({
  sections,
  selected,
  disabled,
  onPick,
}: {
  sections: SectionRef[];
  selected: string;
  disabled: boolean;
  onPick: (id: string) => void;
}) {
  const [q, setQ] = useState("");
  const needle = q.trim();
  const shown = needle ? sections.filter((s) => s.name.includes(needle)) : sections;
  return (
    <div className="flex flex-col gap-3" data-testid="section-picker">
        {sections.length > 6 && (
          <Input
            type="search"
            aria-label={I.sectionSearch}
            placeholder={I.sectionSearch}
            icon={<IconSearch size={16} />}
            value={q}
            onChange={(e) => setQ(e.target.value.slice(0, 40))}
          />
        )}
        {sections.length === 0 ? (
          <p className="text-sm text-ink-muted">{I.sectionEmpty}</p>
        ) : shown.length === 0 ? (
          <p className="text-sm text-ink-muted">{I.sectionNoMatch}</p>
        ) : (
          <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4">
            {shown.map((s) => {
              const on = s.id === selected;
              const img = mediaUrl(s.image_thumb_url ?? s.image_url);
              return (
                <li key={s.id}>
                  <button
                    type="button"
                    aria-pressed={on}
                    disabled={disabled}
                    onClick={() => onPick(s.id)}
                    data-testid="section-tile"
                    className={`flex w-full flex-col items-center gap-1 rounded-control border p-1 ${
                      on ? "border-2 border-accent bg-accent-tint" : "border-line hover:bg-row-hover"
                    }`}
                  >
                    <span className="relative block aspect-square w-full overflow-hidden rounded-control bg-field">
                      {img ? (
                        <img src={img} alt="" loading="lazy" className="absolute inset-0 h-full w-full object-cover" />
                      ) : (
                        <span className="absolute inset-0 flex items-center justify-center text-2xl font-medium text-ink-muted" aria-hidden>
                          {s.name.trim().charAt(0)}
                        </span>
                      )}
                      {on && (
                        <span className="absolute end-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-accent text-on-bright" aria-hidden>
                          <IconCheck size={15} />
                        </span>
                      )}
                    </span>
                    <span className={`line-clamp-2 text-center text-xs ${on ? "font-bold text-accent-text" : "text-ink"}`}>{s.name}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
    </div>
  );
}
