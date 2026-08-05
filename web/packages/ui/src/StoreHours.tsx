"use client";

/**
 * **أوقاتُ دوام المتجر — محرَّرٌ واحدٌ للوحتين.**
 *
 * # لماذا هنا
 *
 * كان محرَّرُ الساعات نافذةً داخل قسم المتاجر في لوحة الإدارة، **ولا وجودَ له
 * في لوحة المتجر إطلاقاً** — والنقطةُ `PUT /merchant/stores/{id}/hours` مبنيّةٌ
 * منذ البداية **ولا يناديها أحد.**
 *
 * **فصاحبُ المطعم لا يستطيع أن يقول متى يفتح.** يتّصل بالمنصة ليُغيَّر له، أو
 * لا يفعل — **فيبقى متجرُه مفتوحاً في النظام أربعاً وعشرين ساعة**، وتصله طلباتٌ
 * في الثالثة فجراً فيُلغيها فتُحسب عليه مخالفة. **وقاعدةُ «متجر مغلق» لا
 * تُختبَر أصلاً لأنّ الجدولَ فارغ.**
 *
 * # ونسخةٌ واحدةٌ لا نسختان
 *
 * وكان أسهلَ أن يُنسخ المحرِّرُ إلى لوحة المتجر — **ونسختان تفترقان يوماً**:
 * يُصلَح شرطُ «حتى بعد منتصف الليل» في إحداهما، فيبقى الآخرُ يقول للمتجر إنّ
 * جدولَه خطأ. **وهي عائلةُ الخلل التي تكرّرت في هذه الجولة ستَّ مرّات.**
 *
 * **والمسارُ معاملٌ لا ثابت** — كما في `MenuManager`: الحارسُ في الخادم يختلف
 * (ملكيةٌ للمتجر، ودورٌ للإدارة) **والشكلُ واحد.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "./components";
import { Alert } from "./feedback";
import { IconPrev } from "./icons";

const m = getMessages(defaultLocale);
const H = m.admin.hours;

export interface DayHours {
  day_of_week: number;
  closed: boolean;
  open_time: string;
  close_time: string;
}

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

export function StoreHours({
  api,
  /** مسارُ القراءة والكتابة — واحدٌ في الحالين (`GET` ثمّ `PUT`). */
  path,
  /** الإغلاقُ الطارئ — اختياريّ: تعرضه الإدارةُ والمتجر، ولا يعرضه من لا يملكه. */
  emergency,
  onEmergencyChange,
  onSaved,
}: {
  api: ApiFn;
  path: string;
  emergency?: { value: boolean; save: (v: boolean) => Promise<void> };
  onEmergencyChange?: (v: boolean) => void;
  onSaved?: () => void;
}) {
  const [days, setDays] = useState<DayHours[] | null>(null);
  const [closed, setClosed] = useState(emergency?.value ?? false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    api<DayHours[]>(path)
      .then(setDays)
      .catch(() => setError(m.errors.internal));
  }, [api, path]);

  useEffect(() => {
    setClosed(emergency?.value ?? false);
  }, [emergency?.value]);

  function updateDay(i: number, patch: Partial<DayHours>) {
    setDays((ds) => ds && ds.map((d, di) => (di === i ? { ...d, ...patch } : d)));
    setSaved(false);
  }

  async function save() {
    if (!days) return;
    setBusy(true);
    setError("");
    try {
      await api(path, { method: "PUT", body: JSON.stringify({ days }) });
      if (emergency && closed !== emergency.value) {
        await emergency.save(closed);
        onEmergencyChange?.(closed);
      }
      setSaved(true);
      onSaved?.();
    } catch {
      setError(m.errors.internal);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      {emergency && (
        <label className="mb-4 flex cursor-pointer items-center justify-between rounded-control border border-danger/40 bg-danger/5 px-3 py-2.5">
          <span>
            <span className="block text-sm font-medium text-danger">{H.emergencyClose}</span>
            <span className="text-xs text-ink-muted">{H.emergencyHint}</span>
          </span>
          <input
            type="checkbox"
            checked={closed}
            onChange={(e) => {
              setClosed(e.target.checked);
              setSaved(false);
            }}
            className="h-5 w-5 accent-danger"
          />
        </label>
      )}

      {!days ? (
        <p className="p-4 text-center text-ink-muted">{m.common.loading}</p>
      ) : (
        <div className="space-y-2">
          {days.map((d, i) => (
            <div key={d.day_of_week} className="flex flex-wrap items-center gap-3 text-sm">
              <span className="w-16 shrink-0 font-medium">{H.days[i]}</span>
              <label className="flex cursor-pointer items-center gap-1.5 text-ink-muted">
                <input
                  type="checkbox"
                  checked={d.closed}
                  onChange={(e) => updateDay(i, { closed: e.target.checked })}
                  className="h-4 w-4 accent-danger"
                />
                {H.closedDay}
              </label>
              <input
                type="time"
                disabled={d.closed}
                value={d.open_time}
                onChange={(e) => updateDay(i, { open_time: e.target.value })}
                className="rounded-control border border-line bg-surface px-2 py-1 disabled:opacity-40"
              />
              <IconPrev size={14} className="text-ink-muted" />
              <input
                type="time"
                disabled={d.closed}
                value={d.close_time}
                onChange={(e) => updateDay(i, { close_time: e.target.value })}
                className="rounded-control border border-line bg-surface px-2 py-1 disabled:opacity-40"
              />
              {/* **دوامٌ يعبر منتصفَ الليل مقبولٌ ومُعلَن.**

                  ساعةُ إغلاقٍ أصغرُ من ساعة الفتح تعني «إلى ما بعد منتصف
                  الليل» — **ومطاعمُ الشاورما في الرقّة تعمل هكذا.** وبلا هذه
                  الكلمة يظنّها من يضبطها خطأً فيتراجع. */}
              {!d.closed && d.close_time <= d.open_time && (
                <span className="text-2xs text-ink-muted">{H.overnight}</span>
              )}
            </div>
          ))}
        </div>
      )}

      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
      {saved && (
        <Alert tone="success" className="mt-3">
          {H.saved}
        </Alert>
      )}

      <div className="mt-4 flex justify-end">
        <Button disabled={busy || !days} onClick={() => void save()}>
          {m.common.save}
        </Button>
      </div>
    </div>
  );
}
