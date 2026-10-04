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
  Confirm,
  Input,
  Switch,
  Textarea,
  LoadingState,
} from "@rahalgo/ui";
import WeekHours, { type Win } from "./WeekHours";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const P = m.admin.platformHours;

interface Closure {
  active: boolean;
  message?: string;
  ends_at?: string;
}

/**
 * **لحظةٌ بصيغة الحقل بتوقيت دمشق دائماً** — و`""` إن لم تكن.
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات البند ١١: «موعدُ العودة بتوقيت دمشق
 * دائماً».) **كان الحقلُ يقرأ ساعةَ المتصفّح** — فموظّفٌ جهازُه على توقيتٍ آخر
 * يُزحلق الموعدَ ساعات. **ودمشقُ على +03:00 ثابتةً** (لا توقيتَ صيفيّ منذ ٢٠٢٢).
 */
const DAMASCUS_OFFSET_MS = 3 * 60 * 60 * 1000;
function toDamascusInput(iso?: string): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  // **الساعةُ بإزاحةِ دمشق الثابتة ثمّ تُقرأ أجزاؤها بالعالميّ** — فلا تدخل ساعةُ الجهاز.
  const d = new Date(t + DAMASCUS_OFFSET_MS);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

/** **حقلُ دمشق ← لحظةٌ مطلقة.** */
function fromDamascusInput(v: string): string {
  if (!v) return "";
  const d = new Date(`${v}:00+03:00`);
  return Number.isNaN(d.getTime()) ? "" : d.toISOString();
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
  /** **التأكيدُ قبل الإيقاف أو الإعادة** — `stop` أو `resume` أو لا شيء. */
  const [ask, setAsk] = useState<"stop" | "resume" | null>(null);
  const [busy, setBusy] = useState(false);

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
      setEndsLocal(toDamascusInput(c.ends_at));
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

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

  /**
   * **حفظُ الإيقاف بحاله** — `active` صريحٌ في النداء لا من مفتاحٍ يُلمس.
   *
   * (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات البند ١١: زرٌّ أحمر «أوقف المنصّة الآن»
   * وتأكيدٌ وشريطٌ أحمر ودخولُ غرفة الطوارئ.) **كان مفتاحاً مع «احفظ» بلا
   * تأكيد** — ولمسةٌ خاطئةٌ توقف الطلباتِ في البلد كلِّه.
   */
  async function saveClosure(active: boolean) {
    setError("");
    setBusy(true);
    try {
      await api("/api/v1/admin/platform/closure", {
        method: "PUT",
        body: JSON.stringify({
          active,
          message: closure.message ?? "",
          ends_at: fromDamascusInput(endsLocal),
        }),
      });
      setSavedClosure(true);
      setTimeout(() => setSavedClosure(false), 2000);
      setAsk(null);
      await load();
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setBusy(false);
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

          {/* **والمحرّرُ مشتركٌ مع أوقات المناطق** — **ونسختان تفترقان
              يوماً، فتقبل شاشةٌ ما تردّه الأخرى.** */}
          <WeekHours windows={wins} onChange={setWins} disabled={!may} />

          {may && (
            <div className="flex items-center gap-3">
              <Button onClick={saveHours}>{P.save}</Button>
              {savedHours && <span className="text-sm text-ink-muted">{P.saved}</span>}
            </div>
          )}
        </div>
      </Card>

      <Card tone={closure.active ? "danger" : "default"}>
        <div className="space-y-3 p-4">
          <h3 className="font-medium">{P.closure.title}</h3>
          <p className="text-sm text-ink-muted">{P.closure.hint}</p>

          {closure.active && <Alert tone="error">{P.closure.banner}</Alert>}

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
              <span className="mt-1 block text-xs text-ink-muted">
                {P.closure.endsDamascus} · {P.closure.endsHint}
              </span>
            </label>
          </div>

          {may && (
            <div className="flex flex-wrap items-center gap-3">
              {closure.active ? (
                <>
                  <Button onClick={() => void saveClosure(true)} disabled={busy} variant="secondary">
                    {P.closure.save}
                  </Button>
                  <Button onClick={() => setAsk("resume")} disabled={busy}>
                    {P.closure.resume}
                  </Button>
                </>
              ) : (
                <Button variant="danger" onClick={() => setAsk("stop")} disabled={busy}>
                  {P.closure.stopNow}
                </Button>
              )}
              {savedClosure && (
                <span className="text-sm text-ink-muted">{P.closure.saved}</span>
              )}
            </div>
          )}
        </div>
      </Card>

      <Confirm
        open={ask !== null}
        title={ask === "stop" ? P.closure.confirmStopTitle : P.closure.confirmResumeTitle}
        body={ask === "stop" ? P.closure.confirmStopBody : P.closure.confirmResumeBody}
        confirmLabel={ask === "stop" ? P.closure.stopNow : P.closure.resume}
        tone={ask === "stop" ? "danger" : "primary"}
        busy={busy}
        onConfirm={() => void saveClosure(ask === "stop")}
        onCancel={() => setAsk(null)}
      />
    </div>
  );
}
