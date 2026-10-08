"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحُ «قبول تلقائي» أعلى الصفحة** — قرارُ المالك ٢٠٢٦-١٠-٠٨
 * ══════════════════════════════════════════════════════════════════════
 *
 * **إعدادٌ منطقيٌّ يُقلب من صفحة عمله** لا من صفحة الإعدادات: «طلبات الانضمام»
 * (`leads.auto_approve`) و«الطلبات» (`orders.auto_transfer`).
 *
 * **والحَكَمُ الخادم**: يُقرأ من `GET /admin/settings` بحقل `editable` — فلا
 * يظهر المفتاحُ لمن لا يملك كتابتَه — ويُكتب بـ`PUT /admin/settings/{key}`،
 * **وخطوةُ التحقّق إن لزمت يتولّاها `StepUpGate` مركزيّاً.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Card, Switch, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const A = m.admin.autoMode;

interface SettingRow {
  key: string;
  value: unknown;
  editable: boolean;
}

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
  /**
   * **مفتاحُ دقائقَ يقبل وإن كان هذا مطفياً** (طلبُ المالك ٢٠٢٦-١٠-٠٩) — يُذكر تحته
   * صراحةً، **فلا يرى «مطفي» ثمّ يُفاجأ بطلبٍ قُبل لحاله.**
   */
  minutesKey?: string;
}) {
  const canCall = useCanCall();
  const canRead = canCall("GET", "/settings");
  const [row, setRow] = useState<{ on: boolean; editable: boolean } | null>(null);
  const [minutes, setMinutes] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      const res = await api<{ settings?: SettingRow[] } | SettingRow[]>("/api/v1/admin/settings");
      const rows: SettingRow[] = Array.isArray(res) ? res : (res.settings ?? []);
      const r = rows.find((x) => x.key === settingKey);
      setRow(r ? { on: r.value === true, editable: r.editable } : null);
      if (minutesKey) {
        const mv = rows.find((x) => x.key === minutesKey)?.value;
        setMinutes(typeof mv === "number" ? mv : null);
      }
    } catch {
      // **مفتاحٌ لا يُقرأ لا يُعرض** — والصفحةُ تعمل بلاه.
      setRow(null);
    }
  }, [canRead, settingKey, minutesKey]);

  useEffect(() => {
    void load();
  }, [load]);
  useLiveRefresh(["settings"], () => void load());

  async function toggle(next: boolean) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/settings/${settingKey}`, {
        method: "PUT",
        body: JSON.stringify({ value: next }),
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
