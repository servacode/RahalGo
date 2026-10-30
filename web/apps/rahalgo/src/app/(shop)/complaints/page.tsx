"use client";

/**
 * **الشكاوى والبلاغات — شكاواه وأين وصلت** (`/my/tickets`).
 *
 * **ومن اشتكى ولم يرَ جواباً ظنّ شكواه ضاعت** — فتُعرض بحالها وردّ المنصّة.
 * **وتُفتح الشكوى من تفاصيل الطلب** بسببٍ من قائمة المحرّك. **وكلُّ شكوى
 * تُفتح على خيطها** (`/complaints/[id]`) — حالُها وردودُها والردُّ عليها.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { ComplaintCard, ComplaintGrid, EmptyState, LoadingState, ReloadState, IconSupport, type ComplaintTicket } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const Q = m.shop.complaint;

export default function ComplaintsPage() {
  const { user, loading } = useAuth();
  const [rows, setRows] = useState<ComplaintTicket[] | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    if (!user) return;
    setError("");
    api<{ tickets: ComplaintTicket[] }>("/api/v1/my/tickets")
      .then((r) => setRows(r.tickets ?? []))
      .catch((err) => setError(errorText(err)));
  }, [user]);

  useEffect(load, [load]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  return (
    <div className="flex flex-col gap-4">
      <ShopPageHeader title={Q.listTitle} />
      {error && !rows ? (
        <ReloadState label={error} onRetry={load} />
      ) : rows === null ? (
        <LoadingState variant="list" rows={3} />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconSupport} title={Q.none} />
      ) : (
        <ComplaintGrid>
          {rows.map((t) => (
            <ComplaintCard
              key={t.id}
              ticket={t}
              note={Q.mineNote}
              action={
                <Link href={`/complaints/${t.id}`} className="text-sm font-medium text-accent-text">
                  {Q.details}
                </Link>
              }
            />
          ))}
        </ComplaintGrid>
      )}
    </div>
  );
}
