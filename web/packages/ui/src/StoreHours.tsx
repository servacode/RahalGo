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
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Checkbox } from "./components";
import { Alert } from "./feedback";
import { IconPrev } from "./icons";
import { LoadingState } from "./layout";

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
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      {emergency && (
        <Checkbox
          id="hours-emergency"
          checked={closed}
          onChange={(e) => {
            setClosed(e.target.checked);
            setSaved(false);
          }}
          /* **والتنبيهُ سطران**: عنوانٌ أحمرُ وشرحٌ تحته — لا نصٌّ واحد. */
          label={
            <span className="block">
              <span className="block text-sm font-medium text-danger">{H.emergencyClose}</span>
              <span className="text-xs text-ink-muted">{H.emergencyHint}</span>
            </span>
          }
          className="mb-4 rounded-control border border-danger-edge bg-danger-tint px-3 py-2.5"
        />
      )}

      {!days ? (
        <LoadingState variant="inline" />
      ) : (
        <div className="space-y-2">
          {days.map((d, i) => (
            /* ══════════════════════════════════════════════════════════
               **واليومُ صندوقٌ على الجوّال لا صفٌّ يلتفّ**
               ══════════════════════════════════════════════════════════

               (شهده المالك ٢٠٢٦-٠٨-١١: «أوقاتُ الدوام أيضاً غيرُ مفهومةٍ
                بهذا الشكل».)

               **كان `flex-wrap` يقطع اليومَ حيث ضاق لا حيث يُفهم**: اسمُ
               اليومِ و«مغلق» وساعةُ الفتح في سطر، **ثمّ سهمٌ وساعةُ إغلاقٍ
               في سطرٍ وحدَهما بلا اسمِ يومٍ فوقهما.** فتُقرأ سبعةُ أيّامٍ
               أربعةَ عشرَ سطراً متشابهاً، **ولا يُعرف أيُّ ساعةٍ لأيّ يوم.**

               **والالتفافُ لا يُصلَح بترتيبٍ آخر** — يقطع حيث لا يتّسع،
               **وحيث لا يتّسع ليس حيث ينتهي المعنى.**

               **فيُبنى صندوقاً**: سطرٌ فيه اليومُ و«مغلق»، **وتحته
               الساعتان جنباً إلى جنبٍ تملآن العرض.**

               **والساعتان تُخفَيان في اليوم المغلق لا تُعتَّمان**: حقلٌ
               باهتٌ يبقى يُسأل عنه، **ويومٌ مغلقٌ لا ساعاتِ له أصلاً.**

               **ويعود صفّاً واحداً حيث يتّسع** — الحاسوبُ كان سليماً. */
            <div
              key={d.day_of_week}
              className="rounded-control border border-line-soft p-2.5 text-sm sm:flex sm:items-center sm:gap-3 sm:border-0 sm:p-0"
            >
              <div className="flex items-center justify-between gap-3 sm:flex-none sm:justify-start">
                <span className="w-16 shrink-0 font-medium">{H.days[i]}</span>
                <Checkbox
                  id={`hours-closed-${d.day_of_week}`}
                  checked={d.closed}
                  onChange={(e) => updateDay(i, { closed: e.target.checked })}
                  label={H.closedDay}
                  className="gap-1.5 text-ink-muted"
                />
              </div>
              {!d.closed && (
                <div className="mt-2 flex items-center gap-2 sm:mt-0 sm:flex-none">
                  <input
                    type="time"
                    value={d.open_time}
                    onChange={(e) => updateDay(i, { open_time: e.target.value })}
                    className="surface-inset min-w-0 flex-1 px-2 py-1 sm:flex-none"
                  />
                  <IconPrev size={14} className="shrink-0 text-ink-muted" />
                  <input
                    type="time"
                    value={d.close_time}
                    onChange={(e) => updateDay(i, { close_time: e.target.value })}
                    className="surface-inset min-w-0 flex-1 px-2 py-1 sm:flex-none"
                  />
                </div>
              )}
              {/* **دوامٌ يعبر منتصفَ الليل مقبولٌ ومُعلَن.**

                  ساعةُ إغلاقٍ أصغرُ من ساعة الفتح تعني «إلى ما بعد منتصف
                  الليل» — **ومطاعمُ الشاورما في الرقّة تعمل هكذا.** وبلا هذه
                  الكلمة يظنّها من يضبطها خطأً فيتراجع. */}
              {!d.closed && d.close_time <= d.open_time && (
                <span className="mt-1 block text-2xs text-ink-muted sm:mt-0 sm:inline">
                  {H.overnight}
                </span>
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
