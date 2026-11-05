"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«من متاجرنا» — اختيارُ متجرٍ لطلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * بحثٌ باسم المتجر في متاجرنا الفعّالة ذات الموقع (`GET /admin/whatsapp-orders/stores`)،
 * **والمسكّرُ الآن يُقال ولا يُمنع** — الموظّفُ يعرف أكان صاحبُه سيفتح.
 *
 * **والمتجرُ يُنسخ على الطلب ولا يُربط** (`orders/whatsapp_order.go`) — فلا يرى الطلبَ في
 * تطبيقه ولا يُحاسَب عليه.
 */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Input, IconSearch, IconStore } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;

/** **متجرٌ من متاجرنا كما يردّه البحث.** */
export interface WaStore {
  id: string;
  name: string;
  address_text: string;
  phone: string;
  lat: number;
  lng: number;
  open_now: boolean;
}

export function StorePicker({
  value,
  onPick,
  disabled,
}: {
  value: WaStore | null;
  onPick: (s: WaStore | null) => void;
  disabled?: boolean;
}) {
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<WaStore[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const seq = useRef(0);

  // **بحثٌ بعد توقّف الكتابة** — حرفان على الأقلّ، والأحدثُ يغلب.
  useEffect(() => {
    const term = q.trim();
    const my = ++seq.current;
    if (term.length < 2) {
      setHits(null);
      setBusy(false);
      return;
    }
    setBusy(true);
    const t = setTimeout(async () => {
      try {
        const r = await api<{ stores: WaStore[] }>(
          `/api/v1/admin/whatsapp-orders/stores?q=${encodeURIComponent(term)}`,
        );
        if (my !== seq.current) return;
        setHits(r.stores ?? []);
        setErr("");
      } catch (e) {
        if (my === seq.current) setErr(errorText(e));
      } finally {
        if (my === seq.current) setBusy(false);
      }
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  if (value) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line bg-field p-3">
        <div className="flex min-w-0 items-start gap-2">
          <IconStore size={18} className="mt-0.5 shrink-0 text-ink-muted" />
          <div className="min-w-0">
            <p className="flex flex-wrap items-center gap-2 font-bold">
              {value.name}
              {!value.open_now && <Badge variant="warning">{W.closedNow}</Badge>}
            </p>
            {value.address_text && <p className="text-sm text-ink-muted">{value.address_text}</p>}
          </div>
        </div>
        <Button size="sm" variant="secondary" disabled={disabled} onClick={() => onPick(null)}>
          {W.changeStore}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Input
        label={W.searchStore}
        value={q}
        disabled={disabled}
        icon={<IconSearch size={16} />}
        onChange={(e) => setQ(e.target.value)}
      />
      {err && <Alert tone="error">{err}</Alert>}
      {busy && <p className="text-sm text-ink-muted">{W.searching}</p>}
      {!busy && hits && hits.length === 0 && <p className="text-sm text-ink-muted">{W.noStores}</p>}
      {hits && hits.length > 0 && (
        <ul className="divide-y divide-line-soft overflow-hidden rounded-control border border-line">
          {hits.map((s) => (
            <li key={s.id}>
              <Button
                variant="ghost"
                className="flex w-full items-center justify-between gap-2 rounded-none text-start"
                onClick={() => {
                  onPick(s);
                  setQ("");
                  setHits(null);
                }}
              >
                <span className="min-w-0">
                  <span className="block truncate font-bold text-ink">{s.name}</span>
                  {s.address_text && <span className="block truncate text-sm">{s.address_text}</span>}
                </span>
                {!s.open_now && <Badge variant="warning">{W.closedNow}</Badge>}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
