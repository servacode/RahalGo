"use client";

/**
 * **تبويبُ الأصناف — أصنافُ كلّ المتاجر، الأحدثُ أوّلاً.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «لازم أعرف الأصناف المضافة حديثاً» — تُعرض أوّلاً
 * بعلامة «جديد» واسمِ المتجر ووقتِ الإضافة.)
 *
 * **وصنفان بالاسم نفسِه من متجرين ليسا تكراراً** — كلٌّ في صفّه باسم متجره.
 *
 * **والبحثُ والفلاترُ في المحرّك** — ترشيحٌ فوق صفحةٍ وعدٌ بترشيح.
 * **والإجراءُ الجماعيُّ**: متوفّر · غير متوفّر · نقلٌ إلى قسم — **ولا حذفَ
 * جماعيّاً**: صنفُ المتجر عملُه.
 */

import { useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Select,
  Checkbox,
  Pagination,
  EmptyState,
  LoadingState,
  Money,
  useLiveData,
  useToast,
  IconStatus,
  IconEdit,
  IconCheck,
  IconBlock,
  IconAdd,
  IconCamera,
  Modal,
  MenuItemEditor,
  type MenuItem,
  type MenuPaths,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import ImageUpload, { MediaThumb, useImagePicker } from "@/components/admin/ImageUpload";
import type { Section } from "./SectionsTab";
import AddItemModal from "./AddItemModal";
import StateBadge from "./StateBadge";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const I = m.admin.market.items;

/** **مساراتُ الإدارة لتحرير الصنف** — نفسُها التي في صفحة المتجر. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/admin/merchants/${id}/menu`,
  items: (id) => `/api/v1/admin/merchants/${id}/menu/items`,
  item: (id) => `/api/v1/admin/menu/items/${id}`,
  platformSections: () => `/api/v1/admin/sections`,
  merchant: (id) => `/api/v1/admin/merchants/${id}`,
};

interface MarketItem {
  id: string;
  name: string;
  merchant_price: number;
  available: boolean;
  merchant_id: string;
  merchant_name: string;
  merchant_status: string;
  section_id: string;
  section_name: string;
  thumb_url: string | null;
  created_at: string;
  is_new: boolean;
  /** **سعرُ الزبون والعمولةُ الفعليّةُ وصافي المتجر** — من المحرّك بقاعدة التسعير نفسِها. */
  price: number;
  commission_percent: number;
  store_net: number;
}

interface ItemsPage {
  items: MarketItem[];
  total: number;
  page: number;
  per_page: number;
  new_hours: number;
}

function stateOf(it: MarketItem): { label: string; variant: "success" | "warning" | "danger" } {
  if (it.merchant_status !== "active") return { label: S.itemStoreOff, variant: "danger" };
  if (!it.available) return { label: S.itemOut, variant: "warning" };
  return { label: S.itemLive, variant: "success" };
}

export default function ItemsTab() {
  // **تحريرُ الصنف من بطاقته** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «الأدمن يقدر يعدّل الصنف ويغيّر الصورة والسعر»).
  const canEdit = useCanCall()("PATCH", "/menu/items/{itemID}");
  const [editing, setEditing] = useState<{ merchantID: string; item: MenuItem } | null>(null);
  const [opening, setOpening] = useState("");
  const [adding, setAdding] = useState(false);
  // **تبديلُ الصورة من البطاقة مباشرةً** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — الضغطةُ تفتح ملفّات الجهاز فوراً.
  const imageTarget = useRef<string | null>(null);
  const picker = useImagePicker({
    kind: "menu_item",
    path: "/api/v1/admin/market/media",
    onUploaded: async (mediaID) => {
      const id = imageTarget.current;
      if (!id) return;
      await api(`/api/v1/admin/market/items/${id}`, { method: "PATCH", body: JSON.stringify({ image_media_id: mediaID }) });
      reload();
    },
    onError: (msg) => push(msg, "error"),
  });
  async function openEdit(it: MarketItem) {
    setOpening(it.id);
    try {
      const res = await api<{ sections?: { items: MenuItem[] }[] } | { items: MenuItem[] }[]>(PATHS.menu(it.merchant_id));
      const secs = Array.isArray(res) ? res : (res.sections ?? []);
      const full = secs.flatMap((x) => x.items).find((x) => x.id === it.id);
      if (full) setEditing({ merchantID: it.merchant_id, item: full });
    } catch {
      // @empty-ok — **تعذّرُ الجلب يُبقي البطاقة كما هي**، وتُعاد الضغطة.
    } finally {
      setOpening("");
    }
  }
  const [q, setQ] = useState("");
  const [section, setSection] = useState("");
  const [state, setState] = useState("");
  const [onlyNew, setOnlyNew] = useState(false);
  const [noImage, setNoImage] = useState(false);
  const [page, setPage] = useState(1);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [moveTo, setMoveTo] = useState("");
  const [busy, setBusy] = useState(false);
  const { push, Toaster } = useToast();

  const query = new URLSearchParams({
    q,
    section,
    state,
    page: String(page),
    ...(onlyNew ? { new: "1" } : {}),
    ...(noImage ? { no_image: "1" } : {}),
  }).toString();

  const { data, reload } = useLiveData<ItemsPage>(
    () => api(`/api/v1/admin/market/items?${query}`),
    ["menu", "catalog"],
    [query],
  );
  const { data: secs } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const sections = secs?.sections ?? [];

  function filter<T>(set: (v: T) => void) {
    return (v: T) => {
      set(v);
      setPage(1);
      setPicked(new Set());
    };
  }

  function toggle(id: string) {
    setPicked((p) => {
      const n = new Set(p);
      if (n.has(id)) n.delete(id);
      else n.add(id);
      return n;
    });
  }

  async function bulk(action: "available" | "unavailable" | "move") {
    if (action === "move" && !moveTo) return;
    setBusy(true);
    try {
      const res = await api<{ affected: number }>("/api/v1/admin/market/items/bulk", {
        method: "POST",
        body: JSON.stringify({ ids: [...picked], action, section_id: moveTo }),
      });
      push(I.done.replace("{n}", fmtNum(res.affected)));
      setPicked(new Set());
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setBusy(false);
    }
  }

  async function setOne(id: string, action: "available" | "unavailable") {
    setBusy(true);
    try {
      await api("/api/v1/admin/market/items/bulk", { method: "POST", body: JSON.stringify({ ids: [id], action }) });
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setBusy(false);
    }
  }

  const rows = data?.items ?? [];
  const allPicked = rows.length > 0 && rows.every((r) => picked.has(r.id));

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <Input
          placeholder={I.search}
          value={q}
          onChange={(e) => filter(setQ)(e.target.value)}
        />
        <Select value={section} onChange={(e) => filter(setSection)(e.target.value)}>
          <option value="">{I.allSections}</option>
          {sections.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
        <Select value={state} onChange={(e) => filter(setState)(e.target.value)}>
          <option value="">{I.allStates}</option>
          <option value="live">{S.itemLive}</option>
          <option value="out">{S.itemOut}</option>
          <option value="store_off">{S.itemStoreOff}</option>
        </Select>
        <div className="flex flex-wrap items-center gap-4">
          <Checkbox
            id="mkt-new"
            label={I.onlyNew}
            checked={onlyNew}
            onChange={(e) => filter(setOnlyNew)(e.target.checked)}
          />
          <Checkbox
            id="mkt-noimg"
            label={I.noImage}
            checked={noImage}
            onChange={(e) => filter(setNoImage)(e.target.checked)}
          />
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2">
        {data && <p className="text-xs text-ink-muted">{I.newHint.replace("{h}", fmtNum(data.new_hours))}</p>}
        <Button onClick={() => setAdding(true)} className="inline-flex items-center gap-1.5">
          <IconAdd size={16} />
          {I.addItem}
        </Button>
      </div>

      {/* **شريطُ الإجراء الجماعيّ** — يظهر حين يُحدَّد شيء. */}
      {picked.size > 0 && (
        <div className="surface flex flex-wrap items-center gap-2 p-3">
          <span className="text-sm font-bold">
            {I.selected.replace("{n}", fmtNum(picked.size))}
          </span>
          <Button variant="secondary" disabled={busy} onClick={() => void bulk("available")}>
            {I.makeAvailable}
          </Button>
          <Button variant="secondary" disabled={busy} onClick={() => void bulk("unavailable")}>
            {I.makeUnavailable}
          </Button>
          <div className="flex items-center gap-2">
            <Select value={moveTo} onChange={(e) => setMoveTo(e.target.value)}>
              <option value="">{I.moveTo}</option>
              {sections.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </Select>
            <Button disabled={busy || !moveTo} onClick={() => void bulk("move")}>
              {I.move}
            </Button>
          </div>
          <Button variant="ghost" onClick={() => setPicked(new Set())}>
            {I.clear}
          </Button>
        </div>
      )}

      {!data ? (
        <LoadingState />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconStatus} title={I.empty} />
      ) : (
        // **بطاقاتٌ لا قائمة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — والتعديلُ (الاسم والسعر والصورة) من البطاقة.
        <div className="space-y-3">
          <Checkbox
            id="mkt-all"
            label={I.selectAll}
            checked={allPicked}
            onChange={() => setPicked(allPicked ? new Set() : new Set(rows.map((r) => r.id)))}
          />
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {rows.map((it) => {
              const st = stateOf(it);
              return (
                <div key={it.id} className="surface flex flex-col overflow-hidden">
                  <div className="relative aspect-square bg-field">
                    <button
                      type="button"
                      title={I.changeImage}
                      aria-label={I.changeImage}
                      disabled={!canEdit || picker.busy}
                      onClick={() => {
                        imageTarget.current = it.id;
                        picker.open();
                      }}
                      className="group absolute inset-0 flex items-center justify-center"
                    >
                    {it.thumb_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={mediaUrl(it.thumb_url) ?? ""} alt={it.name} className="h-full w-full object-cover" />
                    ) : (
                      <span className="flex h-full items-center justify-center text-xs text-ink-muted">{S.noImage}</span>
                    )}
                      {canEdit && (
                        <span className="absolute bottom-2 start-2 inline-flex h-8 w-8 items-center justify-center rounded-full bg-surface text-ink elev-1 opacity-80 group-hover:opacity-100">
                          <IconCamera size={16} />
                        </span>
                      )}
                    </button>
                    <span className="absolute start-2 top-2 rounded-control bg-surface p-0.5">
                      <Checkbox
                        id={`mkt-${it.id}`}
                        label={<span className="sr-only">{it.name}</span>}
                        checked={picked.has(it.id)}
                        onChange={() => toggle(it.id)}
                      />
                    </span>
                    <span className="absolute end-2 top-2 flex flex-col items-end gap-1">
                      <StateBadge label={st.label} variant={st.variant} />
                      {it.is_new && <Badge variant="accent">{I.new}</Badge>}
                    </span>
                  </div>
                  <div className="flex flex-1 flex-col gap-1 p-2.5">
                    <p className="line-clamp-2 text-sm font-bold">{it.name}</p>
                    <Link className="truncate text-xs text-ink-muted hover:text-ink" href={`/dashboard/merchants/${it.merchant_id}`}>
                      {it.merchant_name}
                    </Link>
                    <p className="truncate text-2xs text-ink-muted">{it.section_name}</p>
                    {/* **السعرُ مفصّلاً** (قرارُ المالك ٢٠٢٦-١٠-٢٥): سعرُ الصنف، وعمولةُ المتجر، وما يبقى له. */}
                    <dl className="mt-1 space-y-0.5 border-t border-line-soft pt-1.5 text-xs tabular-nums">
                      <PriceRow label={I.price} value={<Money value={it.merchant_price} />} />
                      <PriceRow
                        muted
                        label={I.commission.replace("{n}", fmtNum(it.commission_percent))}
                        value={<><span dir="ltr">−</span><Money value={it.merchant_price - it.store_net} /></>}
                      />
                      {/* **الرقمُ الحقيقيُّ للمعلومة** (قرارُ المالك ٢٠٢٦-١٠-٢٦): العمولةُ بليراتٍ كاملةٍ مقرّبةً
                          لأقرب ليرة (النصفُ يصعد) — ويُقال الرقمُ الدقيقُ هنا. */}
                      {(() => {
                        const exact = Math.round(it.merchant_price * it.commission_percent) / 100;
                        return exact !== Math.floor(exact) ? (
                          <p className="text-2xs text-ink-muted">{I.commissionExact.replace("{x}", fmtNum(exact))}</p>
                        ) : null;
                      })()}
                      <PriceRow strong label={I.storeNet} value={<Money value={it.store_net} />} />
                      {it.price !== it.merchant_price && (
                        <PriceRow muted label={I.salePrice} value={<Money value={it.price} />} />
                      )}
                    </dl>
                    <div className="mt-auto flex items-center gap-2 pt-1.5">
                      <span className="flex w-full items-center justify-between gap-1">
                      {/* **متوفر / غير متوفر بضغطة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بباب الإجراء الجماعيّ نفسِه. */}
                      {/* **زرٌّ ذكيٌّ للتوفّر** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — يقول الحالَ بلونه ويقلبها بضغطة. */}
                      <button
                        type="button"
                        role="switch"
                        aria-checked={it.available}
                        title={it.available ? I.makeUnavailable : I.makeAvailable}
                        disabled={busy}
                        onClick={() => void setOne(it.id, it.available ? "unavailable" : "available")}
                        className={`inline-flex items-center gap-1 rounded-full px-3 py-1.5 text-xs font-bold text-on-solid transition-colors disabled:opacity-60 ${
                          it.available ? "bg-success-solid" : "bg-warning-solid"
                        }`}
                      >
                        {it.available ? <IconCheck size={14} /> : <IconBlock size={14} />}
                        {it.available ? S.itemLive : S.itemOut}
                      </button>
                      {canEdit && (
                        <Button
                          variant="ghost"
                          title={I.edit}
                          aria-label={I.edit}
                          disabled={opening === it.id}
                          onClick={() => void openEdit(it)}
                          className="!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink"
                        >
                          <IconEdit size={17} />
                        </Button>
                      )}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {data && (
        <Pagination
          page={page}
          total={data.total}
          perPage={data.per_page}
          onChange={(p) => {
            setPage(p);
            setPicked(new Set());
          }}
        />
      )}
      {picker.input}
      {adding && (
        <AddItemModal
          sections={sections}
          onClose={() => setAdding(false)}
          onSaved={() => {
            setAdding(false);
            reload();
          }}
        />
      )}
      {editing && (
        <MenuItemEditor
          api={api}
          paths={PATHS}
          merchantID={editing.merchantID}
          platformSections={sections}
          editing={{ item: editing.item, sectionId: editing.item.platform_section_id ?? "" }}
          imageUpload={(initialUrl, onChange) => (
            <ImageUpload kind="menu_item" label={m.shared.menuEditor.itemImage} initialUrl={initialUrl} onChange={onChange} />
          )}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
      <Toaster />
    </div>
  );
}

/** سطرُ حسبة — **اللفظُ يميناً والرقمُ يساراً**. */
function PriceRow({ label, value, strong, muted }: { label: string; value: ReactNode; strong?: boolean; muted?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className={muted ? "text-ink-muted" : ""}>{label}</dt>
      <dd className={strong ? "font-bold" : muted ? "text-ink-muted" : ""}>
        {value}
      </dd>
    </div>
  );
}
