"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ الطوارئ الأحمر — أعلى كلّ صفحةٍ في اللوحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «تنبيهٌ لا يفوت: صوتٌ متكرّرٌ حتّى يضغط موظّفٌ
 *  استلمتها · شريطٌ أحمرُ أعلى كلّ صفحة».)
 *
 * **يعرض ما لم يُستلَم**: طارئَ سائق · إغلاقاً طارئاً لمتجر · وتوقّفَ المنصّة
 * (حالٌ تبقى ما دامت). **و«استلمتها» لا تُغلق الطارئ** — تقول «أنا عليه»،
 * والحلُّ في صفحة الطوارئ.
 *
 * **ولمن يملك الطوارئ وحدَه** (`support.manage`) — والمحرّكُ يحرس البابَ.
 */

import { useCallback, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtSpan, fmtRef, errorText } from "@rahalgo/i18n";
import { Button, useLiveData, useRepeatingChime, IconWarning } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const B = m.admin.emergencyBanner;

interface Banner {
  drivers: { id: string; driver_name: string; order_number: number | null; created_at: string }[];
  stores: { id: string; name: string; closed_at: string | null }[];
  platform_halted: boolean;
  halt_message: string;
}

const since = (iso: string | null) =>
  iso ? B.since.replace("{t}", fmtSpan(Math.max(0, (Date.now() - Date.parse(iso)) / 1000))) : "";

export function EmergencyBanner() {
  const banner = useLiveData<Banner>(
    () => api<Banner>("/api/v1/admin/emergencies/banner"),
    ["emergency", "order", "merchant", "settings"],
  );
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const b = banner.data;
  const reload = banner.reload;
  const pending = !!b && (b.drivers.length > 0 || b.stores.length > 0);
  // **والصوتُ يتكرّر ما بقي ما لم يُستلَم** — وتوقّفُ المنصّة لا يرنّ: حالٌ يعرفها صاحبُها.
  useRepeatingChime(pending);

  const ack = useCallback(
    async (path: string, key: string) => {
      setBusy(key);
      setErr("");
      try {
        await api(path, { method: "POST", body: "{}" });
        reload();
      } catch (e) {
        setErr(errorText(e) || B.ackFailed);
      } finally {
        setBusy("");
      }
    },
    [reload],
  );

  if (!b || (!pending && !b.platform_halted)) return null;
  return (
    <div
      role="alert"
      className="mb-4 rounded-card border border-danger-edge bg-danger-tint p-3 text-sm text-danger"
    >
      <p className="flex items-center gap-2 font-bold">
        <IconWarning size={18} className="shrink-0" />
        {B.title}
      </p>
      <ul className="mt-2 space-y-2">
        {b.drivers.map((d) => (
          <li key={d.id} className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 flex-1 text-ink">
              {B.driver.replace("{name}", d.driver_name)}
              {d.order_number != null &&
                `${m.common.listSeparator}${B.order.replace("{n}", fmtRef(d.order_number))}`}
              {`${m.common.listSeparator}${since(d.created_at)}`}
            </span>
            <Link href={`/dashboard/emergencies/${d.id}`} className="text-xs underline">
              {B.open}
            </Link>
            <Button
              variant="danger"
              disabled={busy === d.id}
              onClick={() => ack(`/api/v1/admin/emergencies/${d.id}/ack`, d.id)}
            >
              {B.ack}
            </Button>
          </li>
        ))}
        {b.stores.map((s) => (
          <li key={s.id} className="flex flex-wrap items-center gap-2">
            <span className="min-w-0 flex-1 text-ink">
              {B.store.replace("{name}", s.name)}
              {s.closed_at && `${m.common.listSeparator}${since(s.closed_at)}`}
            </span>
            <Link href={`/dashboard/merchants/${s.id}`} className="text-xs underline">
              {B.open}
            </Link>
            <Button
              variant="danger"
              disabled={busy === s.id}
              onClick={() => ack(`/api/v1/admin/emergencies/stores/${s.id}/ack`, s.id)}
            >
              {B.ack}
            </Button>
          </li>
        ))}
        {b.platform_halted && (
          <li className="text-ink">
            {B.halted}
            {b.halt_message && `${m.common.listSeparator}${b.halt_message}`}
          </li>
        )}
      </ul>
      {err && <p className="mt-2 text-xs">{err}</p>}
    </div>
  );
}
