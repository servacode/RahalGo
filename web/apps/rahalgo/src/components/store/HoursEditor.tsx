"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ساعاتُ الدوام — المخطّطُ نفسُه الذي في المعالج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `HoursEditor` في `StoreParts.kt`، ملاحظةُ المالك ٧، ٢٠٢٦-١٠-٢٨.)
 *
 * **والمسوّدةُ محلّيّةٌ حتّى يُضغط «حفظ»**، **والمحرّكُ يريد الأيّامَ السبعةَ معاً.**
 * **وفتحٌ بعد منتصف الليل يُسأل عنه قبل الحفظ** (`nightOpen`).
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Confirm, LoadingState, ReloadState } from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";
import type { DayHours } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { clock12, HoursPlanner, nightOpen, putHours, sameHours, useHoursPlan, useStoreHours } from "./SetupParts";

export { clock12, is24h } from "./SetupParts";

const m = getMessages(defaultLocale);
const H = m.store.hours;

function Planner({ saved, onSaved }: { saved: DayHours[]; onSaved: (days: DayHours[]) => void }) {
  const { store, patch } = useStore();
  const toast = useShopToast();
  const plan = useHoursPlan(saved, !!store?.has_hours);
  const [busy, setBusy] = useState(false);
  const [night, setNight] = useState<DayHours[] | null>(null);
  if (!store) return null;
  const suspended = store.status === "suspended";
  const days = plan.days();
  const dirty = plan.chosen && (!store.has_hours || !sameHours(days, saved));

  async function put(d: DayHours[]) {
    if (!store) return;
    setBusy(true);
    try {
      await putHours(store.id, d);
      patch({ has_hours: true });
      toast(H.saved, "success");
      onSaved(d);
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }
  const nightDay = night?.find(nightOpen);

  return (
    <div className="flex flex-col gap-4">
      <HoursPlanner plan={plan} disabled={suspended || busy} />
      {dirty && !suspended && (
        <Button
          size="lg"
          className="w-full font-bold"
          disabled={busy}
          data-testid="hours-save"
          onClick={() => (days.some(nightOpen) ? setNight(days) : void put(days))}
        >
          {m.common.save}
        </Button>
      )}
      <Confirm
        open={!!night}
        title={H.nightTitle}
        body={nightDay ? fill(H.nightText, { open: clock12(nightDay.open_time), close: clock12(nightDay.close_time) }) : ""}
        confirmLabel={H.nightYes}
        tone="primary"
        busy={busy}
        onConfirm={() => {
          const d = night;
          setNight(null);
          if (d) void put(d);
        }}
        onCancel={() => setNight(null)}
      />
    </div>
  );
}

export default function HoursEditor() {
  const { store } = useStore();
  const { saved, setSaved, error, reload } = useStoreHours(store?.id ?? "");
  const [stamp, setStamp] = useState(0);
  if (!store) return null;
  if (saved == null) {
    return error ? <ReloadState label={errorText(error)} onRetry={() => void reload()} /> : <LoadingState variant="list" rows={4} />;
  }
  return (
    <section aria-label={H.title} data-testid="store-hours">
      <Planner
        key={`${store.id}-${stamp}`}
        saved={saved}
        onSaved={(d) => {
          setSaved(d);
          setStamp((n) => n + 1);
        }}
      />
    </section>
  );
}
