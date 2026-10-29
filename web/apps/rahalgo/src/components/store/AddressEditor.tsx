"use client";

/**
 * **العنوانُ والموقع** — مرآةُ `AddressEditor` في `StoreParts.kt`: حقلُ العنوان،
 * **وتحته الخريطةُ ظاهرةً** كما في المعالج، والحفظُ تحتهما معاً.
 *
 * **والنقطةُ المختارةُ لا تُحفظ حتّى يُضغط «حفظ»** — فلا تبدو محفوظةً وهي لم تصل.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MerchantStore } from "@/lib/store/types";
import { useShopToast } from "@/components/customer/toast";
import { StorePointMap } from "./SetupParts";

const m = getMessages(defaultLocale);
const A = m.store.address;

/** **حفظُ العنوان والنقطة** — `PATCH /settings` بما تغيّر وحدَه. */
export async function saveAddress(
  store: MerchantStore,
  text: string,
  point: { lat: number; lng: number } | null,
): Promise<Partial<MerchantStore>> {
  const t = text.trim();
  await api(`/api/v1/merchant/stores/${store.id}/settings`, {
    method: "PATCH",
    body: JSON.stringify({
      ...(t ? { address_text: t } : {}),
      ...(point ? { lat: point.lat, lng: point.lng } : {}),
    }),
  });
  return { address_text: t || store.address_text, ...(point ? { lat: point.lat, lng: point.lng } : {}) };
}

export default function AddressEditor() {
  const { store, patch, reload } = useStore();
  const toast = useShopToast();
  const [text, setText] = useState("");
  const [point, setPoint] = useState<{ lat: number; lng: number } | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => setText(store?.address_text ?? ""), [store?.id, store?.address_text]);
  if (!store) return null;
  const suspended = store.status === "suspended";
  const dirty = text.trim() !== store.address_text || point != null;

  async function save() {
    if (!store) return;
    setBusy(true);
    try {
      patch(await saveAddress(store, text, point));
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
    <section className="flex flex-col gap-3" aria-label={A.title} data-testid="store-address">
      <Textarea
        id="store-address-text"
        label={A.title}
        placeholder={A.hint}
        rows={2}
        value={text}
        disabled={suspended}
        onChange={(e) => setText(e.target.value)}
      />
      <StorePointMap
        lat={point?.lat ?? store.lat}
        lng={point?.lng ?? store.lng}
        disabled={suspended}
        onPick={(lat, lng) => setPoint({ lat, lng })}
      />
      {dirty && !suspended && (
        <Button size="lg" className="w-full font-bold" onClick={() => void save()} disabled={busy}>
          {m.common.save}
        </Button>
      )}
    </section>
  );
}
