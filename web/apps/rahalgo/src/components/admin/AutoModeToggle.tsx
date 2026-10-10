"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحُ «قبول تلقائي» أعلى الصفحة** — قرارُ المالك ٢٠٢٦-١٠-٠٨
 * ══════════════════════════════════════════════════════════════════════
 *
 * **إعدادٌ منطقيٌّ يُقلب من صفحة عمله** لا من صفحة الإعدادات: «طلبات الانضمام»
 * (`leads.auto_approve`) و«الطلبات» (`orders.auto_transfer`).
 *
 * **والحَكَمُ الخادم**: يُقرأ من `GET /admin/auto-mode/{orders|leads}` بحقل
 * `editable` — فلا يظهر لمن لا يملك تبديلَه — ويُكتب بـ`PUT` البابِ نفسِه.
 *
 * **ولماذا بابٌ لا لوحُ الإعدادات** (ملاحظةُ المالك ٢٠٢٦-١٠-١٠): **كان لا يظهر
 * لحساب العمليات** — يقرأ اللوحَ ولا يملك كتابتَه. **والبابُ يُفتح بقدرةِ
 * الشاشة** (`orders.intervene` · `merchants.verify`) لا بقدرة الإعدادات.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Card, Switch, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const A = m.admin.autoMode;

const KIND: Record<string, "orders" | "leads"> = {
  "orders.auto_transfer": "orders",
  "leads.auto_approve": "leads",
};

export function AutoModeToggle({
  settingKey,
  label,
  hint,
  className = "mb-4",
  minutesKey,
}: {
  settingKey: string;
  label: string;
  hint: string;
  className?: string;
  minutesKey?: string;
}) {
  const kind = KIND[settingKey];
  const canCall = useCanCall();
  const canRead = !!kind && canCall("GET", `/auto-mode/${kind}`);
  const [row, setRow] = useState<{ on: boolean; editable: boolean } | null>(null);
  const [minutes, setMinutes] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      const res = await api<{ on: boolean; editable: boolean; minutes?: number }>(
        `/api/v1/admin/auto-mode/${kind}`,
      );
      setRow({ on: res.on === true, editable: res.editable === true });
      setMinutes(minutesKey && typeof res.minutes === "number" ? res.minutes : null);
    } catch {
      setRow(null);
    }
  }, [canRead, kind, minutesKey]);

  useEffect(() => {
    void load();
  }, [load]);
  useLiveRefresh(["settings"], () => void load());

  async function toggle(next: boolean) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/auto-mode/${kind}`, {
        method: "PUT",
        body: JSON.stringify({ on: next }),
      });
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!row || !row.editable) return null;

  return (
    <Card className={`space-y-2 p-4 ${className}`}>
      <p className="font-bold" aria-live="polite">
        {A.title}: <span className={row.on ? "text-success" : "text-ink-muted"}>{row.on ? A.on : A.off}</span>
      </p>
      <Switch
        checked={row.on}
        onChange={(v: boolean) => void toggle(v)}
        label={label}
        hint={hint}
        disabled={busy}
      />
      {minutesKey && minutes !== null && (
        <p className="text-xs text-ink-muted">
          {minutes > 0 ? A.unattended.replace("{n}", String(minutes)) : A.unattendedOff}
        </p>
      )}
      {error && <Alert onDismiss={() => setError("")}>{error}</Alert>}
    </Card>
  );
}
