"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ الصنف — إضافةٌ وتعديلٌ وحذف، كما في تطبيق المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `ItemEditor.kt` و`ItemDraft.kt` في `:ui`، و`MenuViewModel.saveItem`
 *  في تطبيق المتجر.)
 *
 * **الحقولُ بترتيبها**: الصورة · الاسم (بعدّادٍ وحدٍّ من المحرّك) · الوصف ·
 * العملة (ليرة / دولار) · السعر بعملته · قسمُ السوق · الإضافاتُ والخيارات ·
 * حفظ / إلغاء · «احذف الصنف» بسؤال.
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

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Checkbox,
  Confirm,
  FormActions,
  IconAdd,
  IconDelete,
  ImageUpload,
  Input,
  Radio,
  Textarea,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type {
  MenuItem,
  ModifierGroup,
  SectionRef,
  StorePresets,
} from "@/lib/store/types";
import { parseUsd, usdInput, usdText, usdToSyp } from "@/lib/store/usd";
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

interface DraftOption {
  id?: string;
  name: string;
  price_delta: number;
  usd: string;
  /** **خيارٌ خلصان** والصنفُ متوفّر — يُرسَل `available` مع كلّ خيار. */
  available: boolean;
}
interface DraftGroup {
  id?: string;
  name: string;
  min_select: number;
  max_select: number;
  options: DraftOption[];
}

/**
 * **مجموعاتٌ جاهزةٌ بمربّع صح** — كانت ثلاثاً ثابتةً لكلّ متجر (٢٠٢٦-١٠-١٩)، **وصارت
 * من اللوحة لكلّ تصنيف** (قرارُ المالك ٢٠٢٦-١٠-٢٧): `GET /stores/{id}/option-presets`
 * وما `chosen` منها يظهر هنا. **والمربّعُ ينسخ المجموعةَ بخياراتها إلى الصنف**،
 * **وتُعرف المجموعةُ باسمها.** **وإن تعذّر الجلبُ رجعت الثلاثُ القديمة** — لا محرّرَ بلا مجموعات.
 * والمشروباتُ ليست هنا — تنضاف من قائمتها (`DrinksCard`).
 */
interface PresetDef {
  key: string;
  name: string;
  min_select: number;
  max_select: number;
  options: { name: string; price_delta: number }[];
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
    },
    {
      key: "weight",
      name: I.presetWeight,
      min_select: 1,
      max_select: 1,
      options: opts(I.optHalfKg, I.optKg),
    },
    {
      key: "extras",
      name: I.extrasName,
      min_select: 0,
      max_select: 99,
      options: [],
    },
  ];
}
const sameName = (a: string, b: string) => a.trim() === b.trim();
function presetGroup(p: PresetDef): DraftGroup {
  return {
    name: p.name,
    min_select: p.min_select,
    max_select: p.max_select,
    options: p.options.map((o) => ({
      name: o.name,
      price_delta: o.price_delta,
      usd: "",
      available: true,
    })),
  };
}

function toDraft(groups: ModifierGroup[] | undefined): DraftGroup[] {
  return (groups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    min_select: g.min_select,
    max_select: g.max_select,
    options: g.options.map((o) => ({
      id: o.id,
      name: o.name,
      price_delta: o.price_delta,
      usd: usdText(o.price_delta_usd ?? null),
      available: o.available !== false,
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
  const [price, setPrice] = useState(
    item && item.price > 0 ? String(item.price) : "",
  );
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
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [askDelete, setAskDelete] = useState(false);
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

  const edit = (fn: (g: DraftGroup[]) => DraftGroup[]) => {
    setGroups(fn);
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
    const syp = usd ? null : Number(price.trim());
    if (
      (usd && (usdValue == null || usdValue <= 0)) ||
      (!usd && !(syp != null && Number.isInteger(syp) && syp > 0))
    ) {
      return setError(I.vPrice);
    }
    if (!section) return setError(I.vSection);
    if (groups.some((g) => !g.name.trim())) return setError(I.vGroupName);
    const modifiers = groups.map((g) => ({
      ...(g.id ? { id: g.id } : {}),
      name: g.name.trim(),
      min_select: g.min_select,
      max_select: g.max_select,
      options: g.options.map((o) => ({
        ...(o.id ? { id: o.id } : {}),
        name: o.name.trim(),
        price_delta: o.price_delta,
        available: o.available,
        ...(usd ? { price_delta_usd: parseUsd(o.usd) ?? 0 } : {}),
      })),
    }));
    const body = {
      name: name.trim(),
      description: description.trim(),
      platform_section_id: section,
      modifiers,
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

  return (
    <form
      className="flex flex-col gap-4"
      data-testid="item-editor"
      onSubmit={(e) => {
        e.preventDefault();
        void save();
      }}
    >
      <h2 className="heading-page">{item ? I.edit : I.new}</h2>

      <ImageUpload
        kind="menu_item"
        label={I.image}
        initialUrl={item?.image_thumb_url ?? item?.image_url}
        onChange={(id) => setImageId(id)}
        api={api}
        mediaUrl={mediaUrl}
        errorText={errorText}
        path="/api/v1/merchant/media"
        maxEdge={IMAGE_EDGE}
        crop
      />

      <div>
        <Input
          id="item-name"
          label={I.name}
          value={name}
          onChange={(e) => {
            setName(e.target.value.slice(0, NAME_TYPE_MAX));
            setError("");
          }}
          error={
            nameOver
              ? fill(I.nameOver, { max: itemNameMax, n: nameLen })
              : undefined
          }
        />
        {!nameOver && (
          <p
            className="mt-1 text-xs tabular-nums text-ink-muted"
            dir="ltr"
          >{`${nameLen} / ${itemNameMax}`}</p>
        )}
      </div>

      <Textarea
        id="item-desc"
        label={I.desc}
        rows={2}
        value={description}
        onChange={(e) => setDescription(e.target.value.slice(0, DESC_MAX))}
      />

      <div className="flex flex-col gap-2">
        <div
          role="radiogroup"
          aria-label={I.currency}
          className="inline-flex w-fit overflow-hidden rounded-control border border-line"
        >
          {(
            [
              ["SYP", I.currencySyp],
              ["USD", I.currencyUsd],
            ] as const
          ).map(([code, label]) => (
            <button
              key={code}
              type="button"
              role="radio"
              aria-checked={currency === code}
              disabled={busy}
              onClick={() => setCurrency(code)}
              className={`px-5 py-2 text-sm font-bold transition-colors ${
                currency === code
                  ? "bg-accent text-on-bright"
                  : "bg-field text-ink hover:bg-row-hover"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
        {usd ? (
          <>
            <Input
              id="item-price-usd"
              label={I.priceUsd}
              inputMode="decimal"
              dir="ltr"
              value={priceUsd}
              onChange={(e) => {
                setPriceUsd(usdInput(e.target.value));
                setError("");
              }}
            />
            <p
              className={`text-xs ${rate != null && rate > 0 ? "text-ink-muted" : "text-danger"}`}
            >
              {rate == null || rate <= 0
                ? I.usdNeedRate
                : usdValue == null
                  ? fill(I.usdRateIs, { rate: usdText(rate) })
                  : fill(I.usdEquals, {
                      syp: fmtMoney(usdToSyp(usdValue, rate)),
                      rate: usdText(rate),
                    })}
            </p>
          </>
        ) : (
          <Input
            id="item-price"
            label={I.price}
            inputMode="numeric"
            dir="ltr"
            value={price}
            onChange={(e) => {
              setPrice(e.target.value.replace(/\D/g, "").slice(0, 12));
              setError("");
            }}
          />
        )}
      </div>

      <fieldset className="flex flex-col gap-1">
        <legend className="mb-1 text-sm font-bold">{I.section}</legend>
        <div className="surface flex flex-col gap-2 rounded-card p-3">
          {sections.map((s) => (
            <Radio
              key={s.id}
              id={`item-sec-${s.id}`}
              name="item-section"
              checked={section === s.id}
              onChange={() => {
                setSection(s.id);
                setError("");
              }}
              label={s.name}
            />
          ))}
        </div>
      </fieldset>

      <section className="flex flex-col gap-2" aria-label={I.modTitle}>
        <h3 className="heading-card">{I.modTitle}</h3>
        <p className="text-xs text-ink-muted">{I.modHint}</p>
        {/* **مربّعُ صحٍّ لكلّ مجموعةٍ من «خياراتي»** — ينسخها بخياراتها أو يشيلها. */}
        {presetsFailed && (
          <p className="text-xs text-warning">{I.presetsFailed}</p>
        )}
        <div className="flex flex-wrap gap-4" data-testid="item-presets">
          {(presets ?? []).map((p) => (
            <Checkbox
              key={p.key}
              id={`preset-${p.key}`}
              checked={groups.some((g) => sameName(g.name, p.name))}
              onChange={(e) =>
                edit((gs) =>
                  e.target.checked
                    ? [...gs, presetGroup(p)]
                    : gs.filter((g) => !sameName(g.name, p.name)),
                )
              }
              label={p.name}
            />
          ))}
          <Button
            type="button"
            variant="secondary"
            className="flex w-fit items-center gap-1"
            data-testid="item-custom-group"
            onClick={() =>
              edit((gs) => [
                ...gs,
                { name: "", min_select: 0, max_select: 99, options: [] },
              ])
            }
          >
            <IconAdd size={13} />
            {I.customGroup}
          </Button>
        </div>
        {groups.map((g, gi) => (
          <div
            key={gi}
            className="surface flex flex-col gap-2 rounded-card p-3"
            data-testid="item-group"
          >
            {isPreset(g) ? (
              <div className="flex items-center justify-between gap-2">
                <h4 className="font-bold">{g.name}</h4>
                <Button
                  type="button"
                  variant="ghost"
                  aria-label={I.remove}
                  onClick={() => edit((gs) => gs.filter((_, i) => i !== gi))}
                >
                  <IconDelete size={14} className="text-danger" />
                </Button>
              </div>
            ) : (
              <>
                <div className="flex items-end gap-2">
                  <Input
                    id={`g-name-${gi}`}
                    label={I.groupName}
                    value={g.name}
                    onChange={(e) => setGroup(gi, { name: e.target.value })}
                    wrapperClassName="min-w-0 flex-1"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    aria-label={I.remove}
                    onClick={() => edit((gs) => gs.filter((_, i) => i !== gi))}
                  >
                    <IconDelete size={14} className="text-danger" />
                  </Button>
                </div>
                <div className="flex flex-wrap gap-4">
                  <Checkbox
                    id={`g-req-${gi}`}
                    checked={g.min_select > 0}
                    onChange={(e) =>
                      setGroup(gi, { min_select: e.target.checked ? 1 : 0 })
                    }
                    label={I.required}
                  />
                  <Checkbox
                    id={`g-multi-${gi}`}
                    checked={g.max_select > 1}
                    onChange={(e) =>
                      setGroup(gi, { max_select: e.target.checked ? 99 : 1 })
                    }
                    label={I.multi}
                  />
                </div>
              </>
            )}
            <p className="text-xs text-ink-muted">
              {g.min_select > 0 ? I.sizeRule : I.extrasRule}
            </p>
            {g.options.map((o, oi) => (
              <div
                key={oi}
                className="flex flex-col gap-1"
                data-testid="item-option"
              >
                <div className="flex items-end gap-2">
                  <Input
                    id={`o-name-${gi}-${oi}`}
                    label={I.optionName}
                    value={o.name}
                    onChange={(e) =>
                      setOption(gi, oi, { name: e.target.value })
                    }
                    wrapperClassName="min-w-0 flex-1"
                  />
                  <Input
                    id={`o-price-${gi}-${oi}`}
                    label={usd ? I.optPriceUsd : I.optPrice}
                    inputMode={usd ? "decimal" : "numeric"}
                    dir="ltr"
                    value={
                      usd
                        ? o.usd
                        : o.price_delta === 0
                          ? ""
                          : String(o.price_delta)
                    }
                    onChange={(e) =>
                      usd
                        ? setOption(gi, oi, {
                            usd: usdInput(e.target.value, 8),
                          })
                        : setOption(gi, oi, {
                            price_delta:
                              Number(
                                e.target.value.replace(/\D/g, "").slice(0, 9),
                              ) || 0,
                          })
                    }
                    wrapperClassName="w-28 shrink-0"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    aria-label={I.remove}
                    onClick={() =>
                      setGroup(gi, {
                        options: g.options.filter((_, j) => j !== oi),
                      })
                    }
                  >
                    <IconDelete size={14} className="text-danger" />
                  </Button>
                </div>
                {/* **خيارٌ خلصان والصنفُ متوفّر** — الزبونُ يشوفه رمادي. */}
                <Checkbox
                  id={`o-avail-${gi}-${oi}`}
                  checked={o.available}
                  onChange={(e) =>
                    setOption(gi, oi, { available: e.target.checked })
                  }
                  label={I.optAvailable}
                  className={o.available ? "" : "text-ink-muted"}
                />
              </div>
            ))}
            <Button
              type="button"
              variant="ghost"
              className="flex w-fit items-center gap-1"
              onClick={() =>
                setGroup(gi, {
                  options: [
                    ...g.options,
                    { name: "", price_delta: 0, usd: "", available: true },
                  ],
                })
              }
            >
              <IconAdd size={13} />
              {I.addOption}
            </Button>
          </div>
        ))}
      </section>

      {error && <Alert>{error}</Alert>}
      <FormActions
        submit
        onCancel={() => router.push("/store/menu")}
        busy={busy || nameOver}
      />

      {item && (
        <>
          <Button
            type="button"
            variant="ghost"
            className="w-fit text-danger"
            disabled={busy}
            onClick={() => setAskDelete(true)}
            data-testid="item-delete"
          >
            {I.delete}
          </Button>
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
    </form>
  );
}
