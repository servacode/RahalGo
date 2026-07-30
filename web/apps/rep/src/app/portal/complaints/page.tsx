"use client";

/** الشكاوى والبلاغات — البلاغات المقدّمة على طلبات متاجر المندوب. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, IconSupport } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const R = m.rep.reputation;
const fmt = new Intl.NumberFormat("ar-SY");

interface Complaint {
  number: number;
  order_number: number | null;
  subject: string;
  status: "open" | "in_progress" | "resolved";
  created_at: string;
}
interface Reputation {
  complaints: Complaint[];
}

const VARIANT: Record<Complaint["status"], "warning" | "primary" | "success"> = {
  open: "warning",
  in_progress: "primary",
  resolved: "success",
};

export default function ComplaintsPage() {
  const [data, setData] = useState<Reputation | null>(null);

  useEffect(() => {
    api<Reputation>("/api/v1/me/reputation").then(setData).catch(() => undefined);
  }, []);

  if (!data) return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="flex items-center gap-2 text-lg font-bold">
          <IconSupport size={20} className="text-primary" />
          {R.complaintsTitle}
        </h1>
        <p className="mt-1 text-sm text-ink-muted">{R.complaintsHint}</p>
      </div>

      {data.complaints.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-success">
          {R.complaintsEmpty}
        </p>
      ) : (
        <ul className="space-y-2">
          {data.complaints.map((c) => (
            <li
              key={c.number}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4"
            >
              <span className="font-bold">#{fmt.format(c.number)}</span>
              <span className="min-w-0 flex-1 text-sm">{c.subject}</span>
              {c.order_number != null && (
                <span className="text-xs text-ink-muted">
                  {R.order} #{fmt.format(c.order_number)}
                </span>
              )}
              <Badge variant={VARIANT[c.status]}>{R.ticketStatus[c.status]}</Badge>
              <span className="text-xs text-ink-muted" dir="ltr">
                {new Date(c.created_at).toLocaleDateString("ar-SY")}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
