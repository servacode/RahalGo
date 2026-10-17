"use client";

/**
 * **عنوانُ المتجر ونقطتُه** — مرآةُ `AddressEditor` في `StoreScreen.kt`.
 *
 * **والخريطةُ لاقطُ عنوان الزبون نفسُه** (`PickMap` من `@rahalgo/ui/map`،
 * كما في `AddressSheet`) — لاقطٌ واحدٌ في المنصّة (قرارُ ٢٠٢٦-١٠-٠١).
 *
 * **و«حفظ» يظهر حين يتغيّر النصُّ أو النقطة** — ولا يُرسَل إلّا ما تغيّر.
 */

import { lazy, Suspense, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, IconLocation, Modal, FormActions, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const A = m.store.address;

const PickMap = lazy(() => import("@rahalgo/ui/map").then((mod) => ({ default: mod.PickMap })));

export default function AddressEditor() {
  const { store, patch, reload } = useStore();
  const toast = useShopToast();
  const [text, setText] = useState("");
  const [point, setPoint] = useState<{ lat: number; lng: number } | null>(null);
  const [picking, setPicking] = useState<{ lat: number; lng: number } | null>(null);
  const [mapOpen, setMapOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => setText(store?.address_text ?? ""), [store?.id, store?.address_text]);
  if (!store) return null;
  const suspended = store.status === "suspended";
  const lat = point?.lat ?? store.lat;
  const lng = point?.lng ?? store.lng;
  const dirty = text.trim() !== store.address_text || point != null;

  async function save() {
    if (!store) return;
    setBusy(true);
    const t = text.trim();
    try {
      await api(`/api/v1/merchant/stores/${store.id}/settings`, {
        method: "PATCH",
        body: JSON.stringify({
          ...(t ? { address_text: t } : {}),
          ...(point ? { lat: point.lat, lng: point.lng } : {}),
        }),
      });
      patch({
        address_text: t || store.address_text,
        ...(point ? { lat: point.lat, lng: point.lng } : {}),
      });
      setPoint(null);
      toast(A.saved, "success");
      void reload();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="flex flex-col gap-2" aria-label={A.title} data-testid="store-address">
      <header className="flex items-center gap-2">
        <h3 className="heading-card min-w-0 flex-1">{A.title}</h3>
        {dirty && !suspended && (
          <Button onClick={() => void save()} disabled={busy}>
            {m.common.save}
          </Button>
        )}
      </header>
      <Textarea
        id="store-address-text"
        aria-label={A.title}
        placeholder={A.hint}
        rows={2}
        value={text}
        disabled={suspended}
        onChange={(e) => setText(e.target.value)}
      />
      <div className="surface flex items-center gap-3 rounded-card p-3">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-bold">{A.point}</p>
          <p className="text-xs text-ink-muted" dir={lat != null && lng != null ? "ltr" : undefined}>
            {lat == null || lng == null ? A.pointNone : `${lat.toFixed(5)}, ${lng.toFixed(5)}`}
          </p>
        </div>
        <Button
          variant="secondary"
          disabled={suspended}
          className="flex shrink-0 items-center gap-1"
          onClick={() => {
            setPicking(lat != null && lng != null ? { lat, lng } : null);
            setMapOpen(true);
          }}
        >
          <IconLocation size={16} className="text-accent-text" />
          {A.pointPick}
        </Button>
      </div>
      <Modal open={mapOpen} onClose={() => setMapOpen(false)} title={A.point} size="lg">
        <div className="flex flex-col gap-3">
          <Suspense fallback={<div className="h-72 w-full animate-pulse rounded-card bg-field" />}>
            <PickMap
              lat={picking?.lat ?? null}
              lng={picking?.lng ?? null}
              height="h-72"
              api={api}
              onPick={(la: number, ln: number) => setPicking({ lat: la, lng: ln })}
              onLocated={(la: number, ln: number) => setPicking({ lat: la, lng: ln })}
            />
          </Suspense>
          <FormActions
            onSave={() => {
              if (picking) setPoint(picking);
              setMapOpen(false);
            }}
            onCancel={() => setMapOpen(false)}
          />
        </div>
      </Modal>
    </section>
  );
}
