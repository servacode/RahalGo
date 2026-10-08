"use client";

/**
 * **تبويبُ إعداد المتاجر — مَن ضبط ومَن لم يضبط** (طلبُ المالك ٢٠٢٦-١٠-٠٨).
 *
 * الدوامُ والعنوانُ والموقعُ، **وتأكيدُ طريقة المستحقّات وقبول الاسترداد** —
 * ومن اختار (صاحبُ المتجر من تطبيقه أو الموظّفُ من صفحة المتجر).
 *
 * **والتذكيرُ بيد الموظّف لا بالبوت**: زرُّ الواتساب يفتح المحادثة على جهازه
 * والرسالةُ جاهزة، **فلا يصير البوتُ مزعجاً فيُحظر الرقم.** وزرُّ الإشعار
 * يصل صندوقَ صاحب المتجر ويرنّ في تطبيقه.
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import { Badge, Button, Card, LoadingState, useLiveData, useToast } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const U = m.admin.market.setup;

interface SetupRow {
  id: string;
  name: string;
  status: string;
  phone: string;
  has_hours: boolean;
  has_address: boolean;
  has_location: boolean;
  settlement_method: "cash" | "wallet";
  settlement_confirmed_by: "merchant" | "admin" | null;
  accepts_returns: boolean;
  returns_confirmed_by: "merchant" | "admin" | null;
  missing: string[];
  message: string;
  whatsapp_url: string;
}

function Mark({ ok, label }: { ok: boolean; label: string }) {
  return <Badge variant={ok ? "success" : "danger"}>{ok ? label : U.missing.replace("{x}", label)}</Badge>;
}

function Choice({ label, value, by }: { label: string; value: string; by: SetupRow["returns_confirmed_by"] }) {
  if (!by) return <Badge variant="warning">{label + ": " + U.notAsked}</Badge>;
  return (
    <Badge variant="success">
      {label + ": " + value + " · " + (by === "merchant" ? U.byMerchant : U.byStaff)}
    </Badge>
  );
}

export default function SetupTab() {
  const { data, reload } = useLiveData<{ stores: SetupRow[] }>(
    () => api("/api/v1/admin/merchants-setup"),
    ["merchant", "catalog"],
  );
  const [onlyMissing, setOnlyMissing] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const { push, Toaster } = useToast();

  if (!data) return <LoadingState />;

  const all = data.stores;
  const incomplete = all.filter((s) => s.missing.length > 0);
  const rows = onlyMissing ? incomplete : all;
  const count = (k: string) => all.filter((s) => s.missing.includes(k)).length;

  async function remind(s: SetupRow) {
    setBusy(s.id);
    try {
      const res = await api<{ sent: boolean }>(`/api/v1/admin/merchants/${s.id}/setup-reminder`, {
        method: "POST",
      });
      push(res.sent ? U.sent : U.notSent, res.sent ? "success" : "error");
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-4">
      <Card title={U.title}>
        <p className="mb-3 text-xs text-ink-muted">{U.hint}</p>
        <div className="flex flex-wrap gap-2 text-sm">
          <Badge variant="neutral">{U.total.replace("{n}", fmtNum(all.length))}</Badge>
          <Badge variant={incomplete.length ? "warning" : "success"}>
            {U.incomplete.replace("{n}", fmtNum(incomplete.length))}
          </Badge>
          <Badge variant="neutral">{U.hours + ": " + fmtNum(count("hours"))}</Badge>
          <Badge variant="neutral">{U.address + ": " + fmtNum(count("address"))}</Badge>
          <Badge variant="neutral">{U.location + ": " + fmtNum(count("location"))}</Badge>
          <Badge variant="neutral">{U.settlement + ": " + fmtNum(count("settlement"))}</Badge>
          <Badge variant="neutral">{U.returns + ": " + fmtNum(count("returns"))}</Badge>
        </div>
        <div className="mt-3 flex gap-2">
          <Button variant={onlyMissing ? "primary" : "secondary"} onClick={() => setOnlyMissing(true)}>
            {U.onlyMissing}
          </Button>
          <Button variant={onlyMissing ? "secondary" : "primary"} onClick={() => setOnlyMissing(false)}>
            {U.showAll}
          </Button>
        </div>
      </Card>

      {rows.length === 0 ? (
        <Card>
          <p className="text-sm text-ink-muted">{U.allDone}</p>
        </Card>
      ) : (
        <ul className="space-y-3">
          {rows.map((s) => (
            <li key={s.id}>
              <Card>
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <Link className="font-bold text-primary hover:underline" href={`/dashboard/merchants/${s.id}`}>
                    {s.name}
                  </Link>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {s.phone}
                  </span>
                </div>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  <Mark ok={s.has_hours} label={U.hours} />
                  <Mark ok={s.has_address} label={U.address} />
                  <Mark ok={s.has_location} label={U.location} />
                  <Choice
                    label={U.settlement}
                    value={s.settlement_method === "wallet" ? U.wallet : U.cash}
                    by={s.settlement_confirmed_by}
                  />
                  <Choice
                    label={U.returns}
                    value={s.accepts_returns ? U.returnsYes : U.returnsNo}
                    by={s.returns_confirmed_by}
                  />
                </div>
                {s.missing.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {s.whatsapp_url ? (
                      <a
                        className="inline-flex items-center rounded-control bg-success px-3 py-1.5 text-sm font-bold text-on-bright hover:opacity-90"
                        href={s.whatsapp_url}
                        target="_blank"
                        rel="noreferrer"
                      >
                        {U.whatsapp}
                      </a>
                    ) : (
                      <Badge variant="neutral">{U.noPhone}</Badge>
                    )}
                    <Button variant="secondary" disabled={busy === s.id} onClick={() => void remind(s)}>
                      {U.appNotify}
                    </Button>
                  </div>
                )}
              </Card>
            </li>
          ))}
        </ul>
      )}
      <Toaster />
    </div>
  );
}
