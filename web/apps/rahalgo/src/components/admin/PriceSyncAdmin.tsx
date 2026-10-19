"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مزامنةُ الأمين — القسمُ الفنيّ لفريق المنصّة** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **برنامجُ المتجر يبعث جداولَ الأمين وحدَه** — فيُقرأ هنا أين السعرُ والكميّة، **ويُكتب
 * سطرُ القراءة من بعيد** بلا زيارةٍ لكمبيوتر المول. والسطرُ قراءةٌ فقط (المحرّكُ يردّ غيرَها،
 * والبرنامجُ يردّها ثانيةً).
 */

import { useCallback, useEffect, useState } from "react";
import {
  getMessages,
  defaultLocale,
  errorText,
  fmtDateTime,
  fmtNum,
} from "@rahalgo/i18n";
import { Alert, Button, Card, Input, Select, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const P = m.admin.priceSync;

interface Table {
  name: string;
  rows: number;
  columns: string[];
}
interface Config {
  sql_query: string | null;
  sql_connection: string | null;
  probe_table: string;
  folder: string | null;
  currency: "SYP" | "USD";
  want_schema: boolean;
  agent_seen_at: string | null;
  agent_version: string | null;
  agent_error: string | null;
  agent_status: { database?: string; mode?: string } | null;
  schema: { database?: string; tables?: Table[] } | null;
  schema_at: string | null;
}

export default function PriceSyncAdmin({ merchantId }: { merchantId: string }) {
  const [c, setC] = useState<Config | null>(null);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState<string | null>(null);

  const load = useCallback(() => {
    api<Config>(`/api/v1/admin/merchants/${merchantId}/price-sync`)
      .then(setC)
      .catch((e) => setError(errorText(e)));
  }, [merchantId]);
  useEffect(load, [load]);

  async function save(extra: Partial<Config> = {}, msg = P.saved) {
    if (!c) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchantId}/price-sync`, {
        method: "PUT",
        body: JSON.stringify({
          sql_query: c.sql_query ?? "",
          sql_connection: c.sql_connection ?? "",
          probe_table: c.probe_table,
          folder: c.folder ?? "",
          currency: c.currency,
          ...extra,
        }),
      });
      setNote(msg);
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!c) return error ? <Alert>{error}</Alert> : null;
  const tables = c.schema?.tables ?? [];

  return (
    <Card title={P.title}>
      <div className="flex flex-col gap-3">
        <p className="text-xs text-ink-muted">{P.hint}</p>
        <div className="text-sm">
          <p className="font-bold">{P.status}</p>
          {c.agent_seen_at ? (
            <p>
              {fill(P.seen, {
                at: fmtDateTime(c.agent_seen_at),
                v: c.agent_version ?? "—",
              })}
            </p>
          ) : (
            <p className="text-ink-muted">{P.never}</p>
          )}
          {c.agent_status?.database && (
            <p>{fill(P.found, { db: c.agent_status.database })}</p>
          )}
          {c.agent_error && (
            <p className="text-danger">{fill(P.error, { e: c.agent_error })}</p>
          )}
        </div>
        <Textarea
          id="ps-query"
          label={P.query}
          dir="ltr"
          rows={4}
          value={c.sql_query ?? ""}
          onChange={(e) => setC({ ...c, sql_query: e.target.value })}
        />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input
            id="ps-conn"
            label={P.connection}
            dir="ltr"
            value={c.sql_connection ?? ""}
            onChange={(e) => setC({ ...c, sql_connection: e.target.value })}
          />
          <Input
            id="ps-probe"
            label={P.probe}
            dir="ltr"
            value={c.probe_table}
            onChange={(e) => setC({ ...c, probe_table: e.target.value })}
          />
          <Input
            id="ps-folder"
            label={P.folder}
            dir="ltr"
            value={c.folder ?? ""}
            onChange={(e) => setC({ ...c, folder: e.target.value })}
          />
          <Select
            id="ps-currency"
            label={P.currency}
            value={c.currency}
            onChange={(e) =>
              setC({ ...c, currency: e.target.value as "SYP" | "USD" })
            }
          >
            <option value="SYP">{P.syp}</option>
            <option value="USD">{P.usd}</option>
          </Select>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => void save()} disabled={busy}>
            {P.save}
          </Button>
          <Button
            variant="secondary"
            onClick={() => void save({ want_schema: true }, P.schemaAsked)}
            disabled={busy}
          >
            {P.askSchema}
          </Button>
        </div>
        {note && <p className="text-sm text-success">{note}</p>}
        {error && <Alert>{error}</Alert>}

        {tables.length > 0 ? (
          <div className="flex flex-col gap-1">
            <p className="text-sm font-bold">
              {fill(P.schema, {
                n: fmtNum(tables.length),
                at: c.schema_at ? fmtDateTime(c.schema_at) : "",
              })}
            </p>
            <ul className="max-h-96 overflow-y-auto text-xs" dir="ltr">
              {tables.map((t) => (
                <li key={t.name} className="border-b border-line-soft py-1">
                  <button
                    type="button"
                    className="w-full text-start"
                    onClick={() => setOpen(open === t.name ? null : t.name)}
                  >
                    <span className="font-bold">{t.name}</span> ·{" "}
                    {fill(P.rows, { n: fmtNum(t.rows) })}
                  </button>
                  {open === t.name && (
                    <p className="mt-1 break-words text-ink-muted">
                      {t.columns.join(" · ")}
                    </p>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <p className="text-sm text-ink-muted">{P.noSchema}</p>
        )}
      </div>
    </Card>
  );
}
