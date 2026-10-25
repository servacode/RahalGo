"use client";

/**
 * **دوامُ المتجر — نافذةٌ تُقرأ من موضعين.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «خياراتُ المتجر والأزرار يجب أن تنتقل لنحذف
 *  تبويبَ المتاجر لاحقاً».)
 *
 * **وكانت داخلَ شاشة المتاجر** — فلا تُنادى من ملفّ صاحبه إلّا بنسخها.
 * **ونسختان تفترقان يوماً**: يُضاف يومٌ في إحداهما ويُنسى في الأخرى.
 *
 * **والشرطُ قبل حذف التبويب**: أن يستوعب الملفُّ كلَّ فعلٍ كان فيه —
 * **ولا نريد خسارةَ أيّ ميزة** (قرارُ المالك ٢٠٢٦-٠٨-٠٣).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText} from "@rahalgo/i18n";
import { Button, Modal, Checkbox, Alert, LoadingState, IconPrev, FormActions} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **ساعاتُ يومٍ واحد** — سبعةٌ لكلّ متجر. */
export interface DayHours {
  day_of_week: number;
  closed: boolean;
  open_time: string;
  close_time: string;
  /** **«مفتوح ٢٤ ساعة»** — ٠٠:٠٠ ← ٠٠:٠٠ (قرارُ المالك ٢٠٢٦-١٠-١٨). */
  open_24h?: boolean;
}

/** **أهو ٢٤ ساعة؟** — الرايةُ، أو ٠٠:٠٠ ← ٠٠:٠٠ من خادمٍ أقدم. */
function is24h(d: DayHours): boolean {
  return d.open_24h ?? (d.open_time.slice(0, 5) === "00:00" && d.close_time.slice(0, 5) === "00:00");
}

/** **وما يلزم النافذةَ من المتجر** — اسمُه ومعرّفُه لا صفُّه كلُّه. */
export interface HoursTarget {
  id: string;
  name: string;
  /** **أمُغلَقٌ طارئاً؟** — تُعرض في النافذة ولا تُبدَّل منها. */
  emergency_closed: boolean;
}


export function HoursModal({
  merchant,
  onClose,
  onChanged,
}: {
  merchant: HoursTarget;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [days, setDays] = useState<DayHours[] | null>(null);
  const [emergency, setEmergency] = useState(merchant.emergency_closed);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api<DayHours[]>(`/api/v1/admin/merchants/${merchant.id}/hours`)
      .then(setDays)
      .catch((err) => setError(errorText(err)));
  }, [merchant.id]);

  // **دوامٌ واحدٌ لكلّ الأيام، أو ٢٤ ساعة للكلّ** (قرارُ المالك ٢٠٢٦-١٠-٢٥).
  const [allOpen, setAllOpen] = useState("10:00");
  const [allClose, setAllClose] = useState("23:00");
  function applyAll(h24: boolean) {
    setDays(
      (ds) =>
        ds &&
        ds.map((d) =>
          h24
            ? { ...d, closed: false, open_24h: true, open_time: "00:00", close_time: "00:00" }
            : { ...d, closed: false, open_24h: false, open_time: allOpen, close_time: allClose },
        ),
    );
  }

  function updateDay(i: number, patch: Partial<DayHours>) {
    setDays((ds) => ds && ds.map((d, di) => (di === i ? { ...d, ...patch } : d)));
  }

  async function save() {
    if (!days) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchant.id}/hours`, {
        method: "PUT",
        body: JSON.stringify({ days }),
      });
      if (emergency !== merchant.emergency_closed) {
        await api(`/api/v1/admin/merchants/${merchant.id}`, {
          method: "PATCH",
          body: JSON.stringify({ emergency_closed: emergency }),
        });
      }
      await onChanged();
      onClose();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.hours.title}: ${merchant.name}`} size="lg">
      <Checkbox
        id="mr-hours-emergency"
        checked={emergency}
        onChange={(e) => setEmergency(e.target.checked)}
        label={
          <span className="block">
            <span className="block text-sm font-medium text-danger">
              {m.admin.hours.emergencyClose}
            </span>
            <span className="text-xs text-ink-muted">{m.admin.hours.emergencyHint}</span>
          </span>
        }
        className="mb-4 rounded-control border border-danger-edge bg-danger-tint px-3 py-2.5"
      />

      {!days ? (
        <LoadingState variant="inline" />
      ) : (
        <div className="space-y-2">
          <div className="mb-3 flex flex-wrap items-center gap-2 rounded-control border border-primary-edge bg-primary-tint p-2.5 text-sm">
            <span className="font-medium">{m.admin.hours.allDays}</span>
            <input
              type="time"
              aria-label={m.admin.hours.allOpen}
              value={allOpen}
              onChange={(e) => setAllOpen(e.target.value)}
              className="rounded-control border border-line px-2 py-1"
            />
            <IconPrev size={14} className="text-ink-muted" />
            <input
              type="time"
              aria-label={m.admin.hours.allClose}
              value={allClose}
              onChange={(e) => setAllClose(e.target.value)}
              className="rounded-control border border-line px-2 py-1"
            />
            <Button variant="secondary" onClick={() => applyAll(false)} disabled={!allOpen || !allClose}>
              {m.admin.hours.applyAll}
            </Button>
            <Button variant="secondary" onClick={() => applyAll(true)}>
              {m.admin.hours.all24}
            </Button>
          </div>
          {days.map((d, i) => (
            <div key={d.day_of_week} className="flex flex-wrap items-center gap-3 text-sm">
              <span className="w-16 shrink-0 font-medium">{m.admin.hours.days[i]}</span>
              <Checkbox
                id={`mr-hours-closed-${d.day_of_week}`}
                checked={d.closed}
                onChange={(e) => updateDay(i, { closed: e.target.checked })}
                label={m.admin.hours.closedDay}
                className="gap-1.5 text-ink-muted"
              />
              {/* **«مفتوح ٢٤ ساعة»** — يُخفي الوقتين، **وإطفاؤه يعيد ٩–٢٣**: ٠٠:٠٠ ← ٠٠:٠٠
                  تُقرأ ٢٤ ساعة ثانيةً. */}
              <Checkbox
                id={`mr-hours-24h-${d.day_of_week}`}
                checked={!d.closed && is24h(d)}
                disabled={d.closed}
                onChange={(e) =>
                  updateDay(
                    i,
                    e.target.checked
                      ? { open_24h: true, open_time: "00:00", close_time: "00:00" }
                      : { open_24h: false, open_time: "09:00", close_time: "23:00" },
                  )
                }
                label={m.admin.hours.open24Short}
                className="gap-1.5 text-ink-muted"
              />
              {!d.closed && is24h(d) ? (
                <span className="font-medium text-accent-text">{m.admin.hours.open24}</span>
              ) : (
              <>
              <input
                type="time"
                disabled={d.closed}
                value={d.open_time}
                onChange={(e) => updateDay(i, { open_time: e.target.value })}
                className="rounded-control border border-line px-2 py-1 disabled:opacity-40"
              />
              <IconPrev size={14} className="text-ink-muted" />
              <input
                type="time"
                disabled={d.closed}
                value={d.close_time}
                onChange={(e) => updateDay(i, { close_time: e.target.value })}
                className="rounded-control border border-line px-2 py-1 disabled:opacity-40"
              />
              {/* **دوامٌ يعبر منتصفَ الليل مقبولٌ ومُعلَن.**

                  ساعةُ إغلاقٍ أصغرُ من ساعة الفتح تعني «إلى ما بعد منتصف
                  الليل» — **ومطاعمُ الشاورما في الرقّة تعمل هكذا.** وبلا
                  هذه الكلمة يظنّها من يضبطها خطأً فيتراجع، **أو يضبطها
                  ولا يثق أنّها فُهمت.** */}
              {!d.closed && d.close_time <= d.open_time && (
                <span className="text-2xs text-ink-muted">{m.admin.hours.overnight}</span>
              )}
              </>
              )}
            </div>
          ))}
        </div>
      )}

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
      <FormActions onSave={save} onCancel={onClose} />
    </Modal>
  );
}
