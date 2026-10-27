"use client";

/**
 * **تبويبُ «خيارات الأصناف»** — مجموعاتُ خياراتٍ جاهزة (الحجم · الوزن · المقاس…) لكلّ تصنيف.
 * (قرارُ المالك ٢٠٢٦-١٠-٢٧: كانت ثلاثاً ثابتةً لكلّ متجر.)
 *
 * **كلُّ مجموعةٍ تُربط بتصنيفات**، ومتجرُ التصنيف يراها في محرّر الصنف افتراضاً — **ويختار
 * صاحبُه غيرَها من «خياراتي».** **و«وحدة لازم» ١/١، و«قد ما بدّه» ٠/٩٩.**
 *
 * **والحذفُ لا يمسّ الأصناف** — المجموعةُ تُنسخ إلى الصنف عند اختيارها، فلا رابطَ يبقى.
 * **والموقوفةُ تختفي من المتاجر** وتبقى هنا.
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Checkbox,
  Input,
  Modal,
  Radio,
  LoadingState,
  useLiveData,
  useToast,
  FormActions,
  Confirm,
  IconEdit,
  IconDelete,
  IconCheck,
  IconBlock,
  IconAdd,
  IconCheckAll,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { LoadError, StateBox } from "./MarketParts";

const m = getMessages(defaultLocale);
const P = m.admin.market.presets;

interface PresetOption {
  name: string;
  price_delta: number;
}
interface Preset {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: PresetOption[];
  sort_order: number;
  active: boolean;
  category_ids: string[];
}
interface Category {
  id: string;
  name: string;
  sort_order: number;
  active: boolean;
}
interface Data {
  presets: Preset[];
  categories: Category[];
}

const required = (p: { min_select: number }) => p.min_select > 0;

/** **جسمُ الحفظ كاملاً** — الـ`PATCH` يأخذ الجسمَ نفسَه، و`category_ids` يستبدل الروابط. */
function bodyOf(p: Omit<Preset, "id">) {
  return JSON.stringify({
    name: p.name,
    min_select: p.min_select,
    max_select: p.max_select,
    options: p.options,
    sort_order: p.sort_order,
    active: p.active,
    category_ids: p.category_ids,
  });
}

export default function OptionPresetsTab() {
  const toast = useToast();
  const { data, error, reload } = useLiveData<Data>(async () => {
    const [p, cats] = await Promise.all([
      api<{ presets: Preset[] }>("/api/v1/admin/option-presets"),
      api<Category[]>("/api/v1/admin/categories"),
    ]);
    return { presets: p.presets ?? [], categories: cats ?? [] };
  }, ["catalog"]);
  const [editing, setEditing] = useState<Preset | null | "new">(null);
  const [removing, setRemoving] = useState<Preset | null>(null);
  /** **ما ينتظر ردّ المحرّك** — معرّفُ المجموعة، أو «remove» للحذف. */
  const [busy, setBusy] = useState<string | null>(null);

  async function remove(p: Preset) {
    setBusy("remove");
    try {
      await api(`/api/v1/admin/option-presets/${p.id}`, { method: "DELETE" });
      setRemoving(null);
      reload();
    } catch (e) {
      setRemoving(null);
      toast.push(errorText(e), "error");
    } finally {
      setBusy(null);
    }
  }

  async function toggle(p: Preset) {
    if (busy) return;
    setBusy(p.id);
    try {
      await api(`/api/v1/admin/option-presets/${p.id}`, {
        method: "PATCH",
        body: bodyOf({ ...p, active: !p.active }),
      });
      reload();
    } catch (e) {
      toast.push(errorText(e), "error");
    } finally {
      setBusy(null);
    }
  }

  if (error && !data) return <LoadError onRetry={reload} />;
  if (!data) return <LoadingState />;
  const presets = [...data.presets].sort((a, b) => a.sort_order - b.sort_order);
  const cats = [...data.categories].sort((a, b) => a.sort_order - b.sort_order);
  const catName = new Map(cats.map((c) => [c.id, c.name]));

  return (
    <div className="space-y-3">
      <section className="surface flex flex-wrap items-center justify-between gap-2 p-3">
        <p className="min-w-0 flex-1 text-sm text-ink-muted">{P.hint}</p>
        <Button onClick={() => setEditing("new")} className="inline-flex shrink-0 items-center gap-1.5">
          <IconAdd size={16} />
          {P.add}
        </Button>
      </section>
      {presets.length === 0 ? (
        <StateBox
          icon={IconCheckAll}
          tone="accent"
          title={P.empty}
          hint={P.emptyHint}
          action={<Button onClick={() => setEditing("new")}>{P.add}</Button>}
        />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {presets.map((p) => (
            <div
              key={p.id}
              className={`surface flex flex-col gap-2 p-4 ${p.active ? "" : "opacity-70"}`}
              data-testid="option-preset"
            >
              <div className="flex items-start gap-2">
                <span className="min-w-0 flex-1 font-bold">{p.name}</span>
                <Badge variant={p.active ? "success" : "warning"}>{p.active ? P.active : P.stopped}</Badge>
              </div>
              <Badge variant="neutral" className="w-fit">
                {required(p) ? P.ruleOne : P.ruleAny}
              </Badge>
              <div className="flex flex-wrap gap-1">
                {p.options.length === 0 ? (
                  <span className="text-xs text-ink-muted">{P.noOptions}</span>
                ) : (
                  p.options.map((o, i) => (
                    <span key={`${o.name}-${i}`} className="rounded-control bg-field px-2 py-0.5 text-xs">
                      {o.name}
                      {o.price_delta > 0 && <span className="text-ink-muted">{` +${fmtMoney(o.price_delta)}`}</span>}
                    </span>
                  ))
                )}
              </div>
              <div className="flex flex-wrap gap-1">
                {p.category_ids.length === 0 ? (
                  <span className="text-xs text-warning">{P.noCategories}</span>
                ) : (
                  p.category_ids.map((id) => (
                    <Badge key={id} variant="accent">
                      {catName.get(id) ?? id}
                    </Badge>
                  ))
                )}
              </div>
              <div className="mt-auto flex items-center justify-end gap-1 pt-1">
                <IconBtn
                  title={p.active ? P.stop : P.activate}
                  className={p.active ? "!text-success" : "!text-warning"}
                  disabled={busy !== null}
                  onClick={() => void toggle(p)}
                >
                  {p.active ? <IconCheck size={18} /> : <IconBlock size={18} />}
                </IconBtn>
                <IconBtn title={P.edit} onClick={() => setEditing(p)}>
                  <IconEdit size={17} />
                </IconBtn>
                <IconBtn title={P.remove} className="!text-danger" onClick={() => setRemoving(p)}>
                  <IconDelete size={17} />
                </IconBtn>
              </div>
            </div>
          ))}
        </div>
      )}
      {removing && (
        <Confirm
          open
          title={P.removeTitle}
          body={P.removeBody.replace("{name}", removing.name)}
          confirmLabel={P.remove}
          busy={busy === "remove"}
          onConfirm={() => void remove(removing)}
          onCancel={() => setRemoving(null)}
        />
      )}
      {editing && (
        <PresetModal
          preset={editing === "new" ? null : editing}
          categories={cats}
          nextOrder={(presets.at(-1)?.sort_order ?? 0) + 1}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            toast.push(P.saved);
            reload();
          }}
        />
      )}
    </div>
  );
}

function PresetModal({
  preset,
  categories,
  nextOrder,
  onClose,
  onSaved,
}: {
  preset: Preset | null;
  categories: Category[];
  nextOrder: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(preset?.name ?? "");
  const [one, setOne] = useState(preset ? required(preset) : true);
  const [options, setOptions] = useState<PresetOption[]>(preset?.options ?? [{ name: "", price_delta: 0 }]);
  const [active, setActive] = useState(preset?.active ?? true);
  const [catIds, setCatIds] = useState<Set<string>>(new Set(preset?.category_ids ?? []));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const setOption = (i: number, p: Partial<PresetOption>) => {
    setOptions((os) => os.map((o, j) => (j === i ? { ...o, ...p } : o)));
    setError("");
  };

  async function submit() {
    if (!name.trim()) return setError(P.nameRequired);
    if (options.some((o) => !o.name.trim())) return setError(P.optionNameRequired);
    setBusy(true);
    setError("");
    try {
      const body = bodyOf({
        name: name.trim(),
        min_select: one ? 1 : 0,
        max_select: one ? 1 : 99,
        options: options.map((o) => ({ name: o.name.trim(), price_delta: o.price_delta })),
        sort_order: preset?.sort_order ?? nextOrder,
        active,
        category_ids: categories.filter((c) => catIds.has(c.id)).map((c) => c.id),
      });
      if (preset) {
        await api(`/api/v1/admin/option-presets/${preset.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/admin/option-presets", { method: "POST", body });
      }
      onSaved();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open title={preset ? P.edit : P.add} onClose={onClose}>
      <div className="space-y-4">
        <Input
          label={P.name}
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setError("");
          }}
        />
        <fieldset className="space-y-2">
          <legend className="mb-1 text-sm font-bold">{P.rule}</legend>
          <div className="flex flex-wrap gap-4">
            <Radio id="preset-rule-one" name="preset-rule" checked={one} onChange={() => setOne(true)} label={P.ruleOne} />
            <Radio id="preset-rule-any" name="preset-rule" checked={!one} onChange={() => setOne(false)} label={P.ruleAny} />
          </div>
        </fieldset>
        <fieldset className="space-y-2">
          <legend className="mb-1 text-sm font-bold">{P.options}</legend>
          {options.map((o, i) => (
            <div key={i} className="flex items-end gap-2">
              <Input
                id={`preset-opt-${i}`}
                label={P.optionName}
                value={o.name}
                onChange={(e) => setOption(i, { name: e.target.value })}
                wrapperClassName="min-w-0 flex-1"
              />
              <Input
                id={`preset-price-${i}`}
                label={P.optionPrice}
                inputMode="numeric"
                dir="ltr"
                value={o.price_delta === 0 ? "" : String(o.price_delta)}
                onChange={(e) => setOption(i, { price_delta: Number(e.target.value.replace(/\D/g, "").slice(0, 9)) || 0 })}
                wrapperClassName="w-28 shrink-0"
              />
              <IconBtn
                title={P.removeOption}
                className="!text-danger"
                onClick={() => setOptions((os) => os.filter((_, j) => j !== i))}
              >
                <IconDelete size={16} />
              </IconBtn>
            </div>
          ))}
          <Button
            variant="ghost"
            className="inline-flex w-fit items-center gap-1"
            onClick={() => setOptions((os) => [...os, { name: "", price_delta: 0 }])}
          >
            <IconAdd size={14} />
            {P.addOption}
          </Button>
        </fieldset>
        <fieldset className="space-y-2">
          <legend className="mb-1 text-sm font-bold">{P.categories}</legend>
          <p className="text-xs text-ink-muted">{P.categoriesHint}</p>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {categories.map((c) => (
              <Checkbox
                key={c.id}
                id={`preset-cat-${c.id}`}
                checked={catIds.has(c.id)}
                onChange={(e) =>
                  setCatIds((s) => {
                    const next = new Set(s);
                    if (e.target.checked) next.add(c.id);
                    else next.delete(c.id);
                    return next;
                  })
                }
                label={c.name}
                className={c.active ? "" : "text-ink-muted"}
              />
            ))}
          </div>
        </fieldset>
        <Checkbox id="preset-active" checked={active} onChange={(e) => setActive(e.target.checked)} label={P.active} />
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions onSave={submit} onCancel={onClose} busy={busy} />
      </div>
    </Modal>
  );
}

/** **زرُّ أيقونةٍ دائريّ** — كزرّ تبويب التصنيفات. */
function IconBtn({
  title,
  className = "",
  disabled = false,
  onClick,
  children,
}: {
  title: string;
  className?: string;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      variant="ghost"
      title={title}
      aria-label={title}
      disabled={disabled}
      onClick={onClick}
      className={`!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink ${className}`}
    >
      {children}
    </Button>
  );
}
