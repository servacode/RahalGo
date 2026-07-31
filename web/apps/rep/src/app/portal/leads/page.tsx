"use client";

/** طلبات الانضمام عبر رابط المندوب — سجل المتاجر التي سجّلت عبره. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, IconOrder } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface Lead {
  id: string;
  store_name: string;
  owner_name: string;
  phone: string;
  area: string;
  category_name: string | null;
  category_icon: string | null;
  status: "new" | "converted" | "rejected";
  created_at: string;
}

const STATUS_VARIANT: Record<Lead["status"], "warning" | "success" | "danger"> = {
  new: "warning",
  converted: "success",
  rejected: "danger",
};

export default function LeadsPage() {
  const [leads, setLeads] = useState<Lead[] | null>(null);

  useEffect(() => {
    api<Lead[]>("/api/v1/rep/leads")
      .then(setLeads)
      .catch(() => setLeads([]));
  }, []);

  if (!leads) {
    return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;
  }

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconOrder size={20} className="text-primary" />
        {m.terms.leads}
      </h1>

      {leads.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
          {m.rep.leadsEmpty}
        </p>
      ) : (
        <ul className="space-y-2">
          {leads.map((l) => (
            <li key={l.id} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    {l.category_icon && <span>{l.category_icon}</span>}
                    <p className="truncate font-medium">{l.store_name}</p>
                    <Badge variant={STATUS_VARIANT[l.status]}>{m.rep.leadStatus[l.status]}</Badge>
                  </div>
                  <p className="mt-1 text-sm text-ink-muted">
                    {l.owner_name && <span>{l.owner_name} — </span>}
                    <span dir="ltr">{l.phone}</span>
                    {l.area && <span> — {l.area}</span>}
                  </p>
                </div>
                <span className="shrink-0 text-xs text-ink-muted" dir="ltr">
                  {new Date(l.created_at).toLocaleDateString("ar-SY")}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
