"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«أين تريد التوصيل؟» — عناوينُه ودبّوسٌ على الخريطة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مرآةُ ورقة العنوان في أندرويد**: قائمةُ المحفوظ، ثمّ «إضافة عنوان جديد»
 * بخريطةٍ ودبّوس، **ثمّ أجزاءٌ لا سطر**: المنطقةُ والمبنى (إلزاميّ)،
 * والشارعُ والطابق. **والسطرُ المركّب يصنعه المحرّك** (`addressLine`) —
 * فلا يصير للعنوان مصدران يفترقان.
 *
 * **والخريطةُ محرّكُ اللوحة نفسُه** (`@rahalgo/ui/map` ← `mapengine` و
 * `mapconfig`) — **ومصدرٌ واحدٌ للبلاطات في الويب كلِّه.**
 */

import { lazy, Suspense, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  EmptyState,
  FormActions,
  LoadingState,
  Chips,
  Sheet,
  Switch,
  Textarea,
  IconLocation,
  IconCheck,
  IconDelete,
  IconEdit,
  IconAdd,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCustomer } from "@/lib/customer/context";
import { liveAutoOn, setLiveAuto } from "@/lib/customer/live";
import type { Address } from "@/lib/customer/types";
import { useShopToast } from "./toast";

const m = getMessages(defaultLocale);
const A = m.shop.address;

const CenterPinMap = lazy(() => import("@rahalgo/ui/map").then((mod) => ({ default: mod.CenterPinMap })));

const KINDS = [
  ["home", A.kindHome],
  ["work", A.kindWork],
  ["other", A.kindOther],
] as const;

/**
 * **صفحةُ «عنوان التوصيل» — للجديد وللتعديل** (قرارُ المالك ٢٠٢٦-١٠-٣٠، مرآةُ
 * `AddressPage.kt`): العنوانُ التفصيليُّ (الإلزاميُّ الوحيد) · خريطةٌ بدبّوسٍ ثابتٍ تحدّد
 * موقعَه وحدَها عند الفتح · «حدّد موقعي بدقة» · النوع · «شارك موقعي المباشر» · الحفظ.
 *
 * **والمحرّكُ يريد `area_building` ونقطةً** — فالنصُّ يُرسَل منطقةً والشارعُ والطابقُ
 * فارغان، **والنقطةُ تُرسَل دائماً.** **والتعديلُ `PATCH`** فلا يضيع كونُه الافتراضيّ.
 */
function initialText(a?: Address | null): string {
  if (!a) return "";
  if (!a.street?.trim() && !a.floor?.trim()) return a.area_building ?? "";
  return a.address_text || a.area_building || "";
}

function AddressForm({
  initial,
  onDone,
  onCancel,
}: {
  initial?: Address | null;
  onDone: (id: string) => void;
  onCancel: () => void;
}) {
  const [pin, setPin] = useState<{ lat: number; lng: number } | null>(
    initial ? { lat: initial.lat, lng: initial.lng } : null,
  );
  const [text, setText] = useState(initialText(initial));
  const [kind, setKind] = useState<string>(initial?.kind || "home");
  const [live, setLive] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => setLive(liveAutoOn()), []);

  const save = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!text.trim()) return setError(A.needDetail);
    if (!pin) return setError(A.needPin);
    setBusy(true);
    setError("");
    try {
      const body = JSON.stringify({
        area_building: text.trim().slice(0, 150),
        street: "",
        floor: "",
        kind,
        lat: pin.lat,
        lng: pin.lng,
      });
      if (initial) {
        await api(`/api/v1/my/addresses/${initial.id}`, { method: "PATCH", body });
        onDone(initial.id);
      } else {
        const r = await api<{ id: string }>("/api/v1/my/addresses", { method: "POST", body });
        onDone(r.id);
      }
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={save} className="space-y-4">
      <Textarea
        id="addr-detail"
        label={A.detail}
        placeholder={A.detailHint}
        required
        autoGrow
        rows={2}
        maxLength={150}
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <Suspense fallback={<div className="h-64 w-full animate-pulse rounded-card bg-field" />}>
        <CenterPinMap
          lat={pin?.lat ?? null}
          lng={pin?.lng ?? null}
          autoLocate={!initial}
          onMove={(la: number, ln: number) => {
            setPin({ lat: la, lng: ln });
            setError("");
          }}
        />
      </Suspense>
      <Chips
        wrap
        items={KINDS.map(([k, label]) => ({ id: k, label }))}
        value={kind as (typeof KINDS)[number][0]}
        onChange={(k) => setKind(k)}
      />
      <Switch
        checked={live}
        label={A.liveAuto}
        onChange={(on) => {
          setLive(on);
          setLiveAuto(on);
        }}
      />
      {error && <Alert>{error}</Alert>}
      <FormActions submit saveLabel={A.save} onCancel={onCancel} busy={busy} />
    </form>
  );
}

export function AddressSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { addresses, addressesError, reloadAddresses, address, pickAddress } = useCustomer();
  const toast = useShopToast();
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<Address | null>(null);
  const [del, setDel] = useState<Address | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  if (!open) return null;

  const close = () => {
    setAdding(false);
    setEditing(null);
    setError("");
    onClose();
  };

  const makeDefault = async (a: Address) => {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/my/addresses/${a.id}/default`, { method: "POST", body: "{}" });
      await reloadAddresses();
      toast(A.defaultDone, "success");
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Sheet open onClose={close} title={editing || adding ? A.pageTitle : A.title}>
        {editing ? (
          <AddressForm
            key={editing.id}
            initial={editing}
            onCancel={() => setEditing(null)}
            onDone={async () => {
              await reloadAddresses();
              setEditing(null);
              toast(A.saved, "success");
            }}
          />
        ) : adding ? (
          <AddressForm
            onCancel={() => setAdding(false)}
            onDone={async (id) => {
              await reloadAddresses();
              pickAddress(id);
              setAdding(false);
              close();
            }}
          />
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-ink-muted">{A.hint}</p>
            {addresses === null ? (
              <LoadingState variant="list" rows={2} />
            ) : addressesError ? (
              <Alert>
                {errorText(addressesError)}
                <button type="button" onClick={() => void reloadAddresses()} className="ms-2 underline">
                  {m.common.retry}
                </button>
              </Alert>
            ) : addresses.length === 0 ? (
              <EmptyState icon={IconLocation} title={A.empty} />
            ) : (
              <ul className="space-y-2">
                {addresses.map((a) => {
                  const on = address?.id === a.id;
                  return (
                    <li key={a.id}>
                      <div
                        className={`flex items-center gap-3 rounded-card border p-3 ${
                          on ? "border-primary bg-primary-tint" : "border-line-soft"
                        }`}
                      >
                        <button
                          type="button"
                          className="flex min-w-0 flex-1 items-center gap-3 text-start"
                          onClick={() => {
                            pickAddress(a.id);
                            close();
                          }}
                        >
                          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-field">
                            {on ? (
                              <IconCheck size={17} strokeWidth={3} className="text-primary-dark" />
                            ) : (
                              <IconLocation size={17} className="text-ink-muted" />
                            )}
                          </span>
                          <span className="min-w-0">
                            <span className="flex items-center gap-2 text-sm font-bold">
                              {KINDS.find(([k]) => k === a.kind)?.[1] ?? A.kindOther}
                              {a.is_default && <Badge>{A.default}</Badge>}
                            </span>
                            <span className="line-clamp-2 text-xs text-ink-muted">{a.address_text}</span>
                          </span>
                        </button>
                        <button
                          type="button"
                          aria-label={A.edit}
                          onClick={() => setEditing(a)}
                          className="taparea flex h-9 w-9 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover"
                        >
                          <IconEdit size={16} />
                        </button>
                        <button
                          type="button"
                          aria-label={A.delete}
                          onClick={() => setDel(a)}
                          className="taparea flex h-9 w-9 shrink-0 items-center justify-center rounded-control text-danger hover:bg-row-hover"
                        >
                          <IconDelete size={16} />
                        </button>
                      </div>
                      {!a.is_default && (
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() => void makeDefault(a)}
                          className="mt-1 ps-3 text-xs text-accent-text hover:underline disabled:opacity-60"
                        >
                          {A.makeDefault}
                        </button>
                      )}
                    </li>
                  );
                })}
              </ul>
            )}
            {error && <Alert>{error}</Alert>}
            <Button variant="secondary" className="flex w-full items-center justify-center gap-1.5" onClick={() => setAdding(true)}>
              <IconAdd size={16} />
              {A.add}
            </Button>
          </div>
        )}
      </Sheet>
      <Confirm
        open={!!del}
        title={A.deleteTitle}
        body={A.deleteBody}
        confirmLabel={A.delete}
        busy={busy}
        onCancel={() => setDel(null)}
        onConfirm={async () => {
          if (!del) return;
          setBusy(true);
          setError("");
          try {
            await api(`/api/v1/my/addresses/${del.id}`, { method: "DELETE" });
            await reloadAddresses();
          } catch (err) {
            setError(errorText(err));
          } finally {
            setBusy(false);
            setDel(null);
          }
        }}
      />
    </>
  );
}

/** **بطاقةُ العنوان المختار** — تُضغط فتفتح الورقة. */
export function AddressChip({ onOpen }: { onOpen: () => void }) {
  const { address } = useCustomer();
  return (
    <button
      type="button"
      onClick={onOpen}
      className="surface flex w-full min-w-0 items-center gap-3 p-3 text-start"
    >
      <IconLocation size={20} className="shrink-0 text-accent-text" />
      <span className="min-w-0 flex-1">
        <span className="block text-xs text-ink-muted">{m.shop.home.deliverTo}</span>
        <span className="block truncate text-sm font-bold">{address ? address.address_text : A.none}</span>
      </span>
      <span className="shrink-0 text-xs text-accent-text">{A.change}</span>
    </button>
  );
}
