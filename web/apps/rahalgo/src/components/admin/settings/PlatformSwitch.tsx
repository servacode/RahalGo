"use client";

/**
 * **زرُّ الدوام** (طلبُ المالك ٢٠٢٦-١٠-١٨: «زر واحد مثل المتجر والسائق بالعمل
 * وخارج العمل») — فاتحين أو مسكّرين بضغطة، **ويسبق جدولَ الساعات** بلا أن يمحوه.
 *
 * **والحالُ من المحرّك** (`/admin/platform/hours` ← `switch`) **ويُكتب في
 * `hours.platform_switch`** — والمحرّكُ وحدَه يحكم بها على الطلب.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Card, Confirm } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const S = m.admin.platformHours.switch;

type Mode = "open" | "closed" | "schedule";

export default function PlatformSwitch({ compact = false }: { compact?: boolean }) {
  const { can } = useAuth();
  const may = can("settings.general.manage");
  const [mode, setMode] = useState<Mode | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // **التسكيرُ يُسأل قبله** (تدقيقُ الرئيسيّة ٢٠٢٦-١٠-٢٧) — ضغطةٌ غلطٌ في رأس الصفحة تمنع كلَّ الزبائن يطلبوا.
  const [asking, setAsking] = useState(false);

  const load = useCallback(() => {
    api<{ switch?: string }>("/api/v1/admin/platform/hours")
      .then((h) => setMode(h.switch === "open" || h.switch === "closed" ? h.switch : "schedule"))
      .catch((e) => setError(errorText(e, m)));
  }, []);

  useEffect(load, [load]);

  async function set(v: Mode) {
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/settings/hours.platform_switch", {
        method: "PUT",
        body: JSON.stringify({ value: v }),
      });
      setMode(v);
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setBusy(false);
    }
  }

  /** **الفتحُ والجدولُ فوراً، والتسكيرُ بعد تأكيد.** */
  const choose = (v: Mode) => {
    if (v === mode) return;
    if (v === "closed") setAsking(true);
    else void set(v);
  };

  const nowLine = mode === "open" ? S.nowOpen : mode === "closed" ? S.nowClosed : S.nowSchedule;

  const confirm = (
    <Confirm
      open={asking}
      title={S.confirmTitle}
      body={S.confirmBody}
      confirmLabel={S.closeNow}
      busy={busy}
      onCancel={() => setAsking(false)}
      onConfirm={() => {
        setAsking(false);
        void set("closed");
      }}
    />
  );

  if (!may || mode === null) return error ? <Alert>{error}</Alert> : null;

  // **مفتاحٌ بثلاث حالاتٍ في رأس الصفحة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تصميم احترافي») — مفتوح · حسب
  // الجدول · مسكّر، **والمختارُ بلونه**، **وتحته سطرٌ يقول الحالَ هلق** — الشرحُ في `title` لا يراه الجوّال.
  if (compact) {
    const opts: { v: Mode; label: string; on: string }[] = [
      { v: "open", label: S.open, on: "bg-success-solid text-on-solid" },
      { v: "schedule", label: S.schedule, on: "bg-accent text-on-bright" },
      { v: "closed", label: S.closed, on: "bg-danger-solid text-on-solid" },
    ];
    return (
      <div className="flex flex-col items-end gap-1" title={S.hint}>
        <div role="radiogroup" aria-label={S.title} className="flex overflow-hidden rounded-full border border-line text-sm">
          {opts.map((o) => (
            <button
              key={o.v}
              type="button"
              role="radio"
              aria-checked={mode === o.v}
              disabled={busy}
              onClick={() => choose(o.v)}
              className={`px-3 py-1.5 font-bold transition-colors ${mode === o.v ? o.on : "text-ink-muted hover:bg-row-hover"}`}
            >
              {o.label}
            </button>
          ))}
        </div>
        <p className={`text-2xs ${mode === "closed" ? "font-bold text-danger" : "text-ink-muted"}`}>{nowLine}</p>
        {error && <p className="text-2xs text-danger">{error}</p>}
        {confirm}
      </div>
    );
  }

  return (
    <Card tone={mode === "closed" ? "danger" : "default"}>
      <div className="flex flex-wrap items-center gap-3 p-4">
        <h3 className="font-bold">{S.title}</h3>
        <Badge variant={mode === "open" ? "success" : mode === "closed" ? "danger" : "neutral"}>
          {S[mode]}
        </Badge>
        <div className="ms-auto flex flex-wrap gap-2">
          {mode !== "open" && (
            <Button onClick={() => set("open")} disabled={busy}>
              {S.openNow}
            </Button>
          )}
          {mode !== "closed" && (
            <Button variant="danger" onClick={() => choose("closed")} disabled={busy}>
              {S.closeNow}
            </Button>
          )}
          {mode !== "schedule" && (
            <Button variant="ghost" onClick={() => set("schedule")} disabled={busy}>
              {S.backToSchedule}
            </Button>
          )}
        </div>
        <p className="w-full text-xs font-bold">{nowLine}</p>
        <p className="w-full text-xs text-ink-muted">{S.hint}</p>
        {error && <Alert>{error}</Alert>}
        {confirm}
      </div>
    </Card>
  );
}
