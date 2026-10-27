"use client";

/**
 * **تبويبُ الأصناف — أصنافُ كلّ المتاجر، الأحدثُ أوّلاً.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «لازم أعرف الأصناف المضافة حديثاً» — تُعرض أوّلاً
 * بعلامة «جديد» واسمِ المتجر ووقتِ الإضافة.)
 *
 * **وصنفان بالاسم نفسِه من متجرين ليسا تكراراً** — كلٌّ في صفّه باسم متجره.
 *
 * **والبحثُ والفلاترُ في المحرّك** — ترشيحٌ فوق صفحةٍ وعدٌ بترشيح. **وكلُّها في بطاقةٍ واحدة**
 * (تدقيقُ السوق ٢٠٢٦-١٠-٢٧): الحالُ والجديدُ وبلا صورة أزرارُ حبوبٍ بعددها، والقسمُ قائمة، والمتجرُ
 * بالبحث (و`?merchant=` من خريطة العمليات يفتحه مرشَّحاً).
 *
 * **والإجراءُ الجماعيُّ**: متوفّر · غير متوفّر · نقلٌ إلى قسم — **ولا حذفَ
 * جماعيّاً**: صنفُ المتجر عملُه.
 */

import { useRef, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Chips,
  Input,
  Select,
  Checkbox,
  Pagination,
  LoadingState,
  useLiveData,
  useToast,
  IconEdit,
  IconCheck,
  IconBlock,
  IconAdd,
  IconCamera,
  IconOrder,
  IconSearch,
  IconStore,
  IconClose,
  MenuItemEditor,
  type MenuItem,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { useImagePicker } from "@/components/admin/ImageUpload";
import type { Section } from "./SectionsTab";
import AddItemModal, { MARKET_MEDIA, marketItemImage, marketMenuPaths } from "./AddItemModal";
import StateBadge from "./StateBadge";
import { LoadError, StateBox, StorePickerModal, type PickedStore } from "./MarketParts";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const I = m.admin.market.items;
const U = m.admin.market.ui;

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
  state_counts?: { all: number; live: number; out: number; store_off: number };
}

function stateOf(it: MarketItem): { label: string; variant: "success" | "warning" | "danger" } {
  if (it.merchant_status !== "active") return { label: S.itemStoreOff, variant: "danger" };
  if (!it.available) return { label: S.itemOut, variant: "warning" };
  return { label: S.itemLive, variant: "success" };
}

export default function ItemsTab({ initialMerchant = "" }: { initialMerchant?: string }) {
  // **تحريرُ الصنف من بطاقته** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — من باب السوق (`PATCH /market/items/{id}`)،
  // **لا من باب إدارة المتاجر**: موظّفُ السوق كان يرى الزرَّ يُخفى أو يُردّ ٤٠٣ وهو صاحبُه.
  const canCall = useCanCall();
  const canEdit = canCall("PATCH", "/market/items/{itemID}");
  const canAdd = canCall("POST", "/market/stores/{id}/items");
  const canReadMenu = canCall("GET", "/merchants/{id}/menu");
  const canReadStore = canCall("GET", "/merchants/{id}");
  const [editing, setEditing] = useState<{ merchantID: string; merchantName: string; item: MenuItem } | null>(null);
  const [opening, setOpening] = useState("");
  const [adding, setAdding] = useState(false);
  const [pickingStore, setPickingStore] = useState(false);
  const { push, Toaster } = useToast();
  // **تبديلُ الصورة من البطاقة مباشرةً** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — الضغطةُ تفتح ملفّات الجهاز فوراً.
  const imageTarget = useRef<string | null>(null);
  const picker = useImagePicker({
    kind: "menu_item",
    path: MARKET_MEDIA,
    onUploaded: async (mediaID) => {
      const id = imageTarget.current;
      if (!id) return;
      await api(`/api/v1/admin/market/items/${id}`, { method: "PATCH", body: JSON.stringify({ image_media_id: mediaID }) });
      reload();
    },
    onError: (msg) => push(msg, "error"),
  });

  /** **الصنفُ كاملاً لنافذة التعديل** — وتعثّرُه يُقال، لا زرٌّ يُكبس ولا يصير شي. */
  async function openEdit(it: MarketItem) {
    setOpening(it.id);
    try {
      const res = await api<{ sections?: { items: MenuItem[] }[] } | { items: MenuItem[] }[]>(
        `/api/v1/admin/merchants/${it.merchant_id}/menu`,
      );
      const secs = Array.isArray(res) ? res : (res.sections ?? []);
      const full = secs.flatMap((x) => x.items).find((x) => x.id === it.id);
      if (full) setEditing({ merchantID: it.merchant_id, merchantName: it.merchant_name, item: full });
      else push(U.openItemFailed, "error");
    } catch (err) {
      push(`${U.openItemFailed} — ${errorText(err)}`, "error");
    } finally {
      setOpening("");
    }
  }

  const [q, setQ] = useState("");
  const [section, setSection] = useState("");
  const [state, setState] = useState("");
  const [onlyNew, setOnlyNew] = useState(false);
  const [noImage, setNoImage] = useState(false);
  const [store, setStore] = useState<PickedStore | null>(
    initialMerchant ? { id: initialMerchant, name: "", number: 0 } : null,
  );
  const [page, setPage] = useState(1);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [moveTo, setMoveTo] = useState("");
  const [busy, setBusy] = useState(false);

  const query = new URLSearchParams({
    q,
    section,
    state,
    page: String(page),
    ...(store ? { merchant: store.id } : {}),
    ...(onlyNew ? { new: "1" } : {}),
    ...(noImage ? { no_image: "1" } : {}),
  }).toString();

  const { data, error, reload } = useLiveData<ItemsPage>(
    () => api(`/api/v1/admin/market/items?${query}`),
    ["menu", "catalog"],
    [query],
  );
  const { data: secs } = useLiveData<{ sections: Section[] }>(() => api("/api/v1/admin/sections"), ["catalog"]);
  const sections = secs?.sections ?? [];

  function filter<T>(set: (v: T) => void) {
    return (v: T) => {
      set(v);
      setPage(1);
      setPicked(new Set());
    };
  }

  function clearFilters() {
    setQ("");
    setSection("");
    setState("");
    setOnlyNew(false);
    setNoImage(false);
    setStore(null);
    setPage(1);
    setPicked(new Set());
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
  const sc = data?.state_counts;
  const filtered = q !== "" || section !== "" || state !== "" || onlyNew || noImage || store !== null;
  // **اسمُ المتجر المرشَّح من الرابط** — يُقرأ من أوّل صنفٍ له.
  const storeName = store?.name || rows.find((r) => r.merchant_id === store?.id)?.merchant_name || U.oneStore;

  return (
    <div className="space-y-3">
      {/* ── الفلاترُ كلُّها في بطاقةٍ واحدة ── */}
      <section className="surface space-y-3 p-3">
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <Input
            id="mkt-items-q"
            icon={<IconSearch />}
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
          {store ? (
            <span className="flex min-h-[2.5rem] items-center justify-between gap-2 rounded-control border border-accent bg-row-hover px-3">
              <span className="flex min-w-0 items-center gap-1.5 text-sm">
                <IconStore size={16} className="shrink-0" />
                <b className="truncate">{storeName}</b>
              </span>
              <Button
                variant="ghost"
                title={U.allStores}
                aria-label={U.allStores}
                onClick={() => filter(setStore)(null)}
                className="!h-8 !w-8 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full"
              >
                <IconClose size={16} />
              </Button>
            </span>
          ) : (
            <Button variant="secondary" onClick={() => setPickingStore(true)} className="inline-flex items-center justify-center gap-1.5">
              <IconStore size={16} />
              {U.pickStoreFilter}
            </Button>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <Chips
            wrap
            items={[
              { id: "", label: U.stateAll, count: sc?.all },
              { id: "live", label: S.itemLive, count: sc?.live },
              { id: "out", label: S.itemOut, count: sc?.out },
              { id: "store_off", label: S.itemStoreOff, count: sc?.store_off },
            ]}
            value={state}
            onChange={(k) => filter(setState)(k)}
          />
          <Chips
            wrap
            items={[
              { id: "new", label: U.pillNew },
              { id: "noimg", label: U.pillNoImage },
            ]}
            value={[...(onlyNew ? ["new"] : []), ...(noImage ? ["noimg"] : [])]}
            onChange={(k) => (k === "new" ? filter(setOnlyNew)(!onlyNew) : filter(setNoImage)(!noImage))}
          />
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-soft pt-3">
          <p className="text-xs text-ink-muted">
            {data && I.newHint.replace("{h}", fmtNum(data.new_hours))}
            {filtered && (
              <button type="button" onClick={clearFilters} className="ms-2 font-bold text-accent hover:underline">
                {U.clearFilters}
              </button>
            )}
          </p>
          {canAdd && (
            <Button onClick={() => setAdding(true)} className="inline-flex items-center gap-1.5">
              <IconAdd size={16} />
              {I.addItem}
            </Button>
          )}
        </div>
      </section>

      {/* **شريطُ الإجراء الجماعيّ** — يظهر حين يُحدَّد شيء، **ويلزق تحت الشاشة على الجوّال**. */}
      {picked.size > 0 && (
        <div className="surface sticky bottom-2 z-10 flex flex-wrap items-center gap-2 p-3 elev-2">
          <span className="text-sm font-bold">{I.selected.replace("{n}", fmtNum(picked.size))}</span>
          <Button variant="secondary" disabled={busy} onClick={() => void bulk("available")}>
            {I.makeAvailable}
          </Button>
          <Button variant="secondary" disabled={busy} onClick={() => void bulk("unavailable")}>
            {I.makeUnavailable}
          </Button>
          <div className="flex items-center gap-2">
            <Select value={moveTo} onChange={(e) => setMoveTo(e.target.value)}>
              <option value="">{I.moveTo}</option>
              {sections
                .filter((s) => s.active)
                .map((s) => (
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

      {error && !data ? (
        <LoadError onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : rows.length === 0 ? (
        <StateBox
          icon={IconOrder}
          tone={filtered ? "muted" : "accent"}
          title={filtered ? I.empty : U.itemsEmptyTitle}
          hint={filtered ? U.itemsEmptyFilteredHint : U.itemsEmptyHint}
          action={
            filtered ? (
              <Button variant="secondary" onClick={clearFilters}>
                {U.clearFilters}
              </Button>
            ) : undefined
          }
        />
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
              const commission = it.merchant_price - it.store_net;
              const exact = Math.round(it.merchant_price * it.commission_percent) / 100;
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
                        <span className="flex flex-col items-center gap-1 text-xs text-ink-muted">
                          <IconCamera size={18} />
                          {S.noImage}
                        </span>
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
                    <p className="truncate text-2xs text-ink-muted">
                      {it.section_name} · {I.addedAt.replace("{t}", fmtDateTime(it.created_at))}
                    </p>
                    {/* **الرقمُ الكبيرُ ما يبقى للمتجر** — والسعرُ والعمولةُ سطرٌ صغيرٌ تحته (تدقيقُ السوق ٢٠٢٦-١٠-٢٧). */}
                    <div className="mt-1 rounded-control bg-field px-2 py-1.5">
                      <span className="block text-2xs text-ink-muted">{I.storeNet}</span>
                      <span className="figure block leading-tight">{fmtMoney(it.store_net)}</span>
                      <span className="block text-2xs text-ink-muted tabular-nums">
                        {U.priceLine
                          .replace("{p}", fmtMoney(it.merchant_price))
                          .replace("{n}", fmtNum(it.commission_percent))
                          .replace("{c}", fmtMoney(commission))}
                      </span>
                      {exact !== Math.floor(exact) && (
                        <span className="block text-2xs text-ink-muted">{I.commissionExact.replace("{x}", fmtNum(exact))}</span>
                      )}
                      {it.price !== it.merchant_price && (
                        <span className="block text-2xs text-ink-muted">
                          {I.salePrice}: {fmtMoney(it.price)}
                        </span>
                      )}
                    </div>
                    <div className="mt-auto flex items-center justify-between gap-1 pt-1.5">
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
                      {canEdit && canReadMenu && (
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
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {data && data.total > data.per_page && (
        <div className="flex justify-center">
          <Pagination
            page={page}
            total={data.total}
            perPage={data.per_page}
            onChange={(p) => {
              setPage(p);
              setPicked(new Set());
            }}
          />
        </div>
      )}
      {picker.input}
      {pickingStore && (
        <StorePickerModal
          title={U.pickStoreFilter}
          onPick={(s) => {
            setPickingStore(false);
            filter(setStore)(s);
          }}
          onClose={() => setPickingStore(false)}
        />
      )}
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
          paths={marketMenuPaths(canReadStore)}
          merchantID={editing.merchantID}
          platformSections={sections}
          editing={{ item: editing.item, sectionId: editing.item.platform_section_id ?? "" }}
          imageUpload={marketItemImage}
          note={
            <span>
              <span className="text-ink-muted">{U.addForStore} </span>
              <b>{editing.merchantName}</b>
            </span>
          }
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
