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

import { useState } from "react";
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
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import type { Section } from "./SectionsTab";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const I = m.admin.market.items;

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

      {data && (
        <p className="text-xs text-ink-muted">
          {I.newHint.replace("{h}", fmtNum(data.new_hours))}
        </p>
      )}

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
        <div className="surface divide-y divide-line-soft">
          <div className="flex items-center gap-3 px-3 py-2">
            <Checkbox
              id="mkt-all"
              label={I.selectAll}
              checked={allPicked}
              onChange={() =>
                setPicked(allPicked ? new Set() : new Set(rows.map((r) => r.id)))
              }
            />
          </div>
          {rows.map((it) => {
            const st = stateOf(it);
            return (
              <div key={it.id} className="flex items-center gap-3 px-3 py-2.5">
                <Checkbox
                  id={`mkt-${it.id}`}
                  label={<span className="sr-only">{it.name}</span>}
                  className="shrink-0"
                  checked={picked.has(it.id)}
                  onChange={() => toggle(it.id)}
                />
                <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-control bg-field">
                  {it.thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(it.thumb_url) ?? ""}
                      alt={it.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <span className="text-2xs text-ink-muted">{S.noImage}</span>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 truncate font-bold">
                    <span className="truncate">{it.name}</span>
                    {it.is_new && <Badge variant="accent">{I.new}</Badge>}
                  </p>
                  <p className="truncate text-xs text-ink-muted">
                    <Link className="hover:text-ink" href={`/dashboard/merchants/${it.merchant_id}`}>
                      {it.merchant_name}
                    </Link>
                    {" · "}
                    {it.section_name}
                    {" · "}
                    {I.addedAt.replace("{t}", fmtDateTime(it.created_at))}
                  </p>
                </div>
                <span className="shrink-0 text-sm">
                  <Money value={it.merchant_price} />
                </span>
                <span className="shrink-0">
                  <Badge variant={st.variant}>{st.label}</Badge>
                </span>
              </div>
            );
          })}
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
      <Toaster />
    </div>
  );
}
