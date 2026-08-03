"use client";

/** إعدادات العمولات — نِسب العمولة لكل دور في مكان واضح واحد. */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  PageHeader, Button, Input, FormSection, IconBalance, IconStore, IconUser,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const C = m.admin.commissions;

interface Setting {
  key: string;
  value: unknown;
}

const REP_KEY = "sales.commission_percent";
const MERCHANT_KEY = "merchants.default_commission_percent";

function asPercent(v: unknown): string {
  if (typeof v === "number") return String(v);
  if (typeof v === "string") return v;
  return "";
}

export default function CommissionsPanel() {
  const { user } = useAuth();
  const isAdmin = !!user?.roles.includes("admin");
  const [repPct, setRepPct] = useState("");
  const [merchantPct, setMerchantPct] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const all = await api<Setting[]>("/api/v1/admin/settings");
      setRepPct(asPercent(all.find((s) => s.key === REP_KEY)?.value));
      setMerchantPct(asPercent(all.find((s) => s.key === MERCHANT_KEY)?.value));
    } catch (err) {
      setError(err instanceof ApiError ? m.errors.internal : m.errors.internal);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function save(key: string, value: string) {
    const n = parseInt(value, 10);
    if (Number.isNaN(n) || n < 0 || n > 100) {
      setError(C.invalid);
      return;
    }
    setBusy(key);
    setError("");
    setMsg("");
    try {
      await api(`/api/v1/admin/settings/${key}`, {
        method: "PUT",
        body: JSON.stringify({ value: n }),
      });
      setMsg(C.saved);
      await load();
    } catch {
      setError(m.errors.internal);
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader icon={IconBalance} title={C.title} />
      <p className="mb-6 text-sm text-ink-muted">{C.subtitle}</p>

      {/* شرح تدفّق العمولة */}
      <div className="mb-6 rounded-card border border-line bg-primary-light/40 p-4 text-sm leading-relaxed text-ink">
        {C.flow}
      </div>

      <div className="space-y-4">
        <FormSection title={C.merchantRate} icon={<IconStore />}>
          <p className="mb-3 text-sm text-ink-muted">{C.merchantHint}</p>
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Input
                id="merchant-pct"
                label={C.percentLabel}
                type="number"
                min={0}
                max={100}
                dir="ltr"
                value={merchantPct}
                onChange={(e) => setMerchantPct(e.target.value)}
                disabled={!isAdmin}
              />
            </div>
            {isAdmin && (
              <Button
                onClick={() => save(MERCHANT_KEY, merchantPct)}
                disabled={busy === MERCHANT_KEY}
              >
                {busy === MERCHANT_KEY ? m.common.loading : m.common.save}
              </Button>
            )}
          </div>
        </FormSection>

        <FormSection title={C.repRate} icon={<IconUser />}>
          <p className="mb-3 text-sm text-ink-muted">{C.repHint}</p>
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Input
                id="rep-pct"
                label={C.percentLabel}
                type="number"
                min={0}
                max={100}
                dir="ltr"
                value={repPct}
                onChange={(e) => setRepPct(e.target.value)}
                disabled={!isAdmin}
              />
            </div>
            {isAdmin && (
              <Button onClick={() => save(REP_KEY, repPct)} disabled={busy === REP_KEY}>
                {busy === REP_KEY ? m.common.loading : m.common.save}
              </Button>
            )}
          </div>
        </FormSection>
      </div>

      {msg && (
        <p className="mt-4 rounded-control bg-success/10 px-3 py-2 text-sm text-success">{msg}</p>
      )}
      {error && (
        <p className="mt-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}
    </div>
  );
}
