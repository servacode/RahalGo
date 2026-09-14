"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دوامُ المنصّة وإيقافُها المؤقّت** (`PH`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ لوحٌ لا صفوفُ إعدادات
 *
 * **وجدولُ أسبوعٍ بفتراتٍ متعدّدةٍ في اليوم ليس مفتاحاً وقيمة** —
 * **ومحرّرُ الإعدادات العامُّ يعرض حقلاً واحداً لكلّ مفتاح.** **ومن
 * حشره نصّاً واحداً طلب من صاحب المنصّة أن يكتب بنيةً في مربّع نصّ**،
 * **وهو ما هربنا منه يومَ نُزع محرّرُ JSON الخام.**
 *
 * # والسريانُ فوقَ الجدول لا في تبويبٍ آخر
 *
 * **ومن كتب جدولاً كاملاً ثمّ لم يجد أين يُفعّله ظنّه ساريا** — **فوُضع
 * المفتاحُ حيث يُكتب الجدول، في أوّل ما تقع عليه العين.**
 *
 * # ولا حكمَ في هذه الشاشة
 *
 * **وكلُّ ما هنا عرضٌ وضبط** — **والقرارُ في المحرّك**: `Validate`
 * تردّ التداخلَ والفترةَ الفارغة، **ونداءٌ مباشرٌ يتجاوز هذه الشاشةَ
 * ولا يتجاوز تلك.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Card,
  Input,
  Switch,
  Textarea,
  Badge,
  IconAdd,
  IconDelete,
  LoadingState,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const P = m.admin.platformHours;

interface Win {
  day_of_week: number;
  start: string;
  end: string;
}

interface Closure {
  active: boolean;
  message?: string;
  ends_at?: string;
}

/** **أتعبر الفترةُ منتصفَ الليل؟** — الاصطلاحُ عينُه الذي في المحرّك. */
function crosses(w: Win): boolean {
  return w.end <= w.start;
}

/**
 * **لحظةٌ بصيغة الحقل المحلّيّ** — و`""` إن لم تكن.
 *
 * **و`datetime-local` لا يقبل منطقةً** — **فيُحوَّل ما جاء من الخادم
 * إلى ساعة المتصفّح ثمّ يُعاد إلى لحظةٍ مطلقةٍ عند الحفظ.**
 */
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export default function HoursPanel() {
  const { can } = useAuth();
  // **والقدرةُ عينُها التي تحكم المناطق** — **ولا قدرةَ جديدةٌ تُخترَع.**
  const may = can("settings.general.manage");

  const [wins, setWins] = useState<Win[]>([]);
  const [enforced, setEnforced] = useState(false);
  const [closure, setClosure] = useState<Closure>({ active: false });
  const [endsLocal, setEndsLocal] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [savedHours, setSavedHours] = useState(false);
  const [savedClosure, setSavedClosure] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const h = await api<{ windows: Win[]; enforced: boolean }>(
        "/api/v1/admin/platform/hours",
      );
      setWins(h.windows ?? []);
      setEnforced(Boolean(h.enforced));
      const c = await api<Closure>("/api/v1/admin/platform/closure");
      setClosure(c);
      setEndsLocal(toLocalInput(c.ends_at));
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function addWindow(day: number) {
    setWins((prev) => [...prev, { day_of_week: day, start: "09:00", end: "17:00" }]);
  }

  function removeWindow(idx: number) {
    setWins((prev) => prev.filter((_, i) => i !== idx));
  }

  function editWindow(idx: number, patch: Partial<Win>) {
    setWins((prev) => prev.map((w, i) => (i === idx ? { ...w, ...patch } : w)));
  }

  async function saveHours() {
    setError("");
    try {
      await api("/api/v1/admin/platform/hours", {
        method: "PUT",
        body: JSON.stringify({ windows: wins }),
      });
      /* **والسريانُ مفتاحُ إعدادٍ يُحفَظ ببابه** — **ولا بابَ ثانٍ
         يُخترَع لقيمةٍ يعرفها محرّرُ الإعدادات.** */
      await api("/api/v1/admin/settings/hours.platform_enforced", {
        method: "PUT",
        body: JSON.stringify({ value: enforced }),
      });
      setSavedHours(true);
      setTimeout(() => setSavedHours(false), 2000);
      await load();
    } catch (e) {
      setError(errorText(e, m));
    }
  }

  async function saveClosure() {
    setError("");
    try {
      await api("/api/v1/admin/platform/closure", {
        method: "PUT",
        body: JSON.stringify({
          active: closure.active,
          message: closure.message ?? "",
          ends_at: endsLocal ? new Date(endsLocal).toISOString() : "",
        }),
      });
      setSavedClosure(true);
      setTimeout(() => setSavedClosure(false), 2000);
      await load();
    } catch (e) {
      setError(errorText(e, m));
    }
  }

  if (loading) return <LoadingState />;

  return (
    <div className="space-y-4">
      {error && <Alert>{error}</Alert>}

      <Card>
        <div className="space-y-3 p-4">
          <h3 className="font-medium">{P.title}</h3>
          <p className="text-sm text-ink-muted">{P.hint}</p>
          <p className="text-sm text-ink-muted">{P.timezone}</p>

          {/* **والسريانُ أوّلُ ما يُرى** — **وجدولٌ مكتوبٌ لا يسري
              يُظَنّ سارياً.** */}
          <Switch
            checked={enforced}
            onChange={(v: boolean) => setEnforced(v)}
            label={P.enforced}
            hint={P.enforcedHint}
            disabled={!may}
          />

          <p className="text-xs text-ink-muted">{P.boundary}</p>

          <div className="grid grid-cols-1 gap-3">
            {P.days.map((label: string, day: number) => {
              const rows = wins
                .map((w, i) => ({ w, i }))
                .filter((x) => x.w.day_of_week === day);
              return (
                <div key={day} className="rounded border border-line p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-sm font-medium">{label}</span>
                    {rows.length === 0 && <Badge>{P.closedDay}</Badge>}
                  </div>
                  <div className="space-y-2">
                    {rows.map(({ w, i }) => (
                      <div key={i} className="flex flex-wrap items-center gap-2">
                        <span className="text-xs text-ink-muted">{P.from}</span>
                        <Input
                          type="time"
                          value={w.start}
                          disabled={!may}
                          onChange={(e) => editWindow(i, { start: e.target.value })}
                        />
                        <span className="text-xs text-ink-muted">{P.to}</span>
                        <Input
                          type="time"
                          value={w.end}
                          disabled={!may}
                          onChange={(e) => editWindow(i, { end: e.target.value })}
                        />
                        {crosses(w) && <Badge>{P.crosses}</Badge>}
                        {may && (
                          <Button
                            variant="ghost"
                            onClick={() => removeWindow(i)}
                            aria-label={P.closedDay}
                          >
                            <IconDelete />
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                  {may && (
                    <Button variant="ghost" onClick={() => addWindow(day)}>
                      <IconAdd />
                      {P.addWindow}
                    </Button>
                  )}
                </div>
              );
            })}
          </div>

          {may && (
            <div className="flex items-center gap-3">
              <Button onClick={saveHours}>{P.save}</Button>
              {savedHours && <span className="text-sm text-ink-muted">{P.saved}</span>}
            </div>
          )}
        </div>
      </Card>

      <Card>
        <div className="space-y-3 p-4">
          <h3 className="font-medium">{P.closure.title}</h3>
          <p className="text-sm text-ink-muted">{P.closure.hint}</p>

          <Switch
            checked={closure.active}
            onChange={(v: boolean) => setClosure((c) => ({ ...c, active: v }))}
            label={P.closure.active}
            disabled={!may}
          />

          <div className="grid grid-cols-1 gap-3">
            <label className="block text-sm">
              <span className="mb-1 block">{P.closure.message}</span>
              <Textarea
                value={closure.message ?? ""}
                placeholder={P.closure.messagePlaceholder}
                disabled={!may}
                onChange={(e) => setClosure((c) => ({ ...c, message: e.target.value }))}
              />
            </label>
            <label className="block text-sm">
              <span className="mb-1 block">{P.closure.endsAt}</span>
              <Input
                type="datetime-local"
                value={endsLocal}
                disabled={!may}
                onChange={(e) => setEndsLocal(e.target.value)}
              />
              <span className="mt-1 block text-xs text-ink-muted">{P.closure.endsHint}</span>
            </label>
          </div>

          {may && (
            <div className="flex items-center gap-3">
              <Button onClick={saveClosure}>{P.closure.save}</Button>
              {savedClosure && (
                <span className="text-sm text-ink-muted">{P.closure.saved}</span>
              )}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}
