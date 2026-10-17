"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ساعاتُ العمل — محرّرُ التطبيق نفسُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `HoursEditor` و`HoursRow` في `StoreScreen.kt`.)
 *
 * **بطاقةُ «نفس الدوام لكل الأيام»** فوق: من · لـ · «طبّق على كل الأيام»
 * — **تفتح الأيامَ السبعةَ بالوقتين وتحفظ فوراً.**
 *
 * **وتحتها يومٌ يومٌ**: مربّعُ «مفتوح» ثمّ وقتا الفتح والإغلاق، **و«حفظ»
 * يظهر حين يتغيّر شيء.**
 *
 * **وفتحٌ بين منتصف الليل والسادسة يُسأل عنه** (`hours_night_*`) — من كتب
 * «٩:٠٠» وقصد المساء اختار «ص» بلا انتباه، **فيبين متجرُه مغلقاً طولَ
 * النهار.**
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Checkbox, Confirm, Input, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { DayHours } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const H = m.store.hours;

/** **مرآةُ `clock12`** — «٩:٠٠ م» لا «21:00». */
export function clock12(hhmm: string): string {
  const h = Number(hhmm.slice(0, 2));
  if (!Number.isFinite(h)) return hhmm;
  const mm = hhmm.slice(3, 5) || "00";
  const h12 = h === 0 ? 12 : h > 12 ? h - 12 : h;
  return `${h12}:${mm} ${h < 12 ? H.am : H.pm}`;
}

/** **مرآةُ `nightOpen`** — يفتح بين ٠ و٥ صباحاً، **إلّا الدوامَ الكاملَ** (٠٠:٠٠ → ٢٣:٥٩). */
function nightOpen(d: DayHours): boolean {
  const open = Number(d.open_time.slice(0, 2));
  const close = Number(d.close_time.slice(0, 2));
  const o = Number.isFinite(open) ? open : 9;
  const c = Number.isFinite(close) ? close : 23;
  return !d.closed && o >= 0 && o <= 5 && !(o === 0 && c >= 23);
}

const hhmm = (t: string) => t.slice(0, 5);

function TimeField({ id, label, value, onChange, disabled }: { id: string; label: string; value: string; onChange: (v: string) => void; disabled?: boolean }) {
  return (
    <Input
      id={id}
      type="time"
      aria-label={label}
      dir="ltr"
      step={60}
      value={hhmm(value) || "00:00"}
      disabled={disabled}
      onChange={(e) => e.target.value && onChange(e.target.value)}
      wrapperClassName="w-28 shrink-0"
    />
  );
}

export default function HoursEditor({ onSaved }: { onSaved?: () => void }) {
  const { store, patch } = useStore();
  const id = store?.id ?? "";
  const toast = useShopToast();
  const [saved, setSaved] = useState<DayHours[] | null>(null);
  const [draft, setDraft] = useState<DayHours[]>([]);
  const [allOpen, setAllOpen] = useState("09:00");
  const [allClose, setAllClose] = useState("23:00");
  const [busy, setBusy] = useState(false);
  const [night, setNight] = useState<DayHours[] | null>(null);
  const suspended = store?.status === "suspended";

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const rows = await api<DayHours[]>(`/api/v1/merchant/stores/${id}/hours`);
      const sorted = [...(rows ?? [])]
        .map((d) => ({ ...d, open_time: hhmm(d.open_time), close_time: hhmm(d.close_time) }))
        .sort((a, b) => a.day_of_week - b.day_of_week);
      setSaved(sorted);
      setDraft(sorted);
      const first = sorted.find((d) => !d.closed);
      if (first) {
        setAllOpen(first.open_time);
        setAllClose(first.close_time);
      }
    } catch {
      setSaved([]);
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);

  const dirty = useMemo(() => JSON.stringify(draft) !== JSON.stringify(saved ?? []), [draft, saved]);

  async function put(days: DayHours[]) {
    if (!id) return;
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${id}/hours`, { method: "PUT", body: JSON.stringify({ days }) });
      setSaved(days);
      setDraft(days);
      patch({ has_hours: true });
      toast(H.saved, "success");
      onSaved?.();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }
  const save = (days: DayHours[]) => (days.some(nightOpen) ? setNight(days) : void put(days));

  if (saved == null) return <LoadingState variant="inline" />;
  if (saved.length === 0) return null;
  const nightDay = night?.find(nightOpen);

  return (
    <section className="flex flex-col gap-3" aria-label={H.title} data-testid="store-hours">
      <header className="flex items-center gap-2">
        <h3 className="heading-card min-w-0 flex-1">{H.title}</h3>
        {dirty && !suspended && (
          <Button onClick={() => save(draft)} disabled={busy} data-testid="hours-save">
            {m.common.save}
          </Button>
        )}
      </header>
      <div className="surface flex flex-col gap-2 rounded-card p-4">
        <p className="text-sm font-bold">{H.allDays}</p>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm">{H.from}</span>
          <TimeField id="hours-all-open" label={H.from} value={allOpen} onChange={setAllOpen} disabled={suspended} />
          <span className="text-sm">{H.to}</span>
          <TimeField id="hours-all-close" label={H.to} value={allClose} onChange={setAllClose} disabled={suspended} />
        </div>
        <p className="text-xs text-ink-muted">{fill(H.summary, { from: clock12(allOpen), to: clock12(allClose) })}</p>
        <Button
          variant="secondary"
          className="w-full"
          data-testid="hours-apply-all"
          disabled={busy || suspended || !allOpen || !allClose}
          onClick={() => {
            const days = draft.map((d) => ({ ...d, closed: false, open_time: allOpen, close_time: allClose }));
            setDraft(days);
            save(days);
          }}
        >
          {H.applyAll}
        </Button>
      </div>
      <ul className="surface flex flex-col divide-y divide-line rounded-card px-4">
        {draft.map((d) => (
          <li key={d.day_of_week} className="flex flex-wrap items-center gap-2 py-2" data-testid="hours-day">
            <Checkbox
              id={`hours-open-${d.day_of_week}`}
              checked={!d.closed}
              disabled={suspended}
              onChange={(e) =>
                setDraft((all) => all.map((x) => (x.day_of_week === d.day_of_week ? { ...x, closed: !e.target.checked } : x)))
              }
              label={H.days[d.day_of_week] ?? ""}
              className="min-w-24 flex-1"
            />
            {d.closed ? (
              <span className="text-xs text-ink-muted">{H.dayClosed}</span>
            ) : (
              <span className="flex items-center gap-1">
                <TimeField
                  id={`hours-o-${d.day_of_week}`}
                  label={`${H.days[d.day_of_week] ?? ""} ${H.from}`}
                  value={d.open_time}
                  disabled={suspended}
                  onChange={(v) => setDraft((all) => all.map((x) => (x.day_of_week === d.day_of_week ? { ...x, open_time: v } : x)))}
                />
                <span className="text-xs text-ink-muted">—</span>
                <TimeField
                  id={`hours-c-${d.day_of_week}`}
                  label={`${H.days[d.day_of_week] ?? ""} ${H.to}`}
                  value={d.close_time}
                  disabled={suspended}
                  onChange={(v) => setDraft((all) => all.map((x) => (x.day_of_week === d.day_of_week ? { ...x, close_time: v } : x)))}
                />
              </span>
            )}
          </li>
        ))}
      </ul>
      <Confirm
        open={!!night}
        title={H.nightTitle}
        body={nightDay ? fill(H.nightText, { open: clock12(nightDay.open_time), close: clock12(nightDay.close_time) }) : ""}
        confirmLabel={H.nightYes}
        tone="primary"
        busy={busy}
        onConfirm={() => {
          const days = night;
          setNight(null);
          if (days) void put(days);
        }}
        onCancel={() => setNight(null)}
      />
    </section>
  );
}
