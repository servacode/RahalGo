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
    <Modal open onClose={onClose} title={`${m.admin.hours.title}: ${merchant.name}`}>
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
          {days.map((d, i) => (
            <div key={d.day_of_week} className="flex items-center gap-3 text-sm">
              <span className="w-16 shrink-0 font-medium">{m.admin.hours.days[i]}</span>
              <Checkbox
                id={`mr-hours-closed-${d.day_of_week}`}
                checked={d.closed}
                onChange={(e) => updateDay(i, { closed: e.target.checked })}
                label={m.admin.hours.closedDay}
                className="gap-1.5 text-ink-muted"
              />
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
