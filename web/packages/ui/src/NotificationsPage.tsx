"use client";

/**
 * صفحة الإشعارات الكاملة — **مكوّن واحد مركزي** ترثه كل اللوحات وموقع الزبون.
 *
 * الجرس يعرض آخر 30 في قائمة منسدلة ضيّقة: يكفي للحظة، ولا يكفي لمن غاب يومين
 * ويريد أن يعرف ما فاته. هنا الأرشيف كاملاً بترشيح بالنوع وتجميع بالتاريخ.
 */

import { useCallback, useMemo, useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtTime, fmtLongDate } from "@rahalgo/i18n";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { Button } from "./components";
import { useLiveData, type AppNotification } from "./Notifications";
import {
  IconBell,
  IconOrder,
  IconSupport,
  IconWallet,
  IconStar,
  IconLink,
  IconUser,
} from "./icons";

const m = getMessages(defaultLocale);
const N = m.shared.notifications;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type LinkType = ComponentType<{ href: string; className?: string; children: ReactNode }>;

interface Feed {
  items: AppNotification[];
  unread: number;
  counts: Record<string, number>;
}

/** أيقونة ولون لكل نوع — مصدر واحد يخدم الصفحة والجرس. */
const KINDS: Record<string, { icon: ComponentType<{ size?: number; className?: string }>; tone: string }> = {
  order: { icon: IconOrder, tone: "text-primary bg-primary-light" },
  ticket: { icon: IconSupport, tone: "text-danger bg-danger/10" },
  wallet: { icon: IconWallet, tone: "text-success bg-success/10" },
  rating: { icon: IconStar, tone: "text-accent-dark bg-accent/10" },
  lead: { icon: IconLink, tone: "text-info bg-info/10" },
  account: { icon: IconUser, tone: "text-ink-muted bg-page" },
};

/** يوم الإشعار بصيغة قابلة للقراءة — "اليوم" و"أمس" أوضح من تاريخ كامل. */
function dayLabel(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const diff = Math.floor(
    (new Date(today.toDateString()).getTime() - new Date(d.toDateString()).getTime()) / 86_400_000,
  );
  if (diff <= 0) return N.today;
  if (diff === 1) return N.yesterday;
  return fmtLongDate(d);
}

export function NotificationsPage({ api, Link }: { api: ApiFn; Link: LinkType }) {
  const [kind, setKind] = useState("");

  const { data, loading, reload } = useLiveData<Feed>(
    () => api(`/api/v1/me/notifications?limit=200${kind ? `&kind=${kind}` : ""}`),
    ["order", "ticket", "wallet", "rating", "lead", "account"],
  );

  const markAll = useCallback(async () => {
    await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id: "" }) });
    reload();
  }, [api, reload]);

  const markOne = useCallback(
    async (id: string) => {
      await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id }) });
      reload();
    },
    [api, reload],
  );

  const items = useMemo(() => data?.items ?? [], [data]);

  // تجميع بالأيام — بلا هذا يصير الأرشيف جداراً من الأسطر
  const groups = useMemo(() => {
    const out: { day: string; rows: AppNotification[] }[] = [];
    for (const n of items) {
      const day = dayLabel(n.created_at);
      const last = out[out.length - 1];
      if (last && last.day === day) last.rows.push(n);
      else out.push({ day, rows: [n] });
    }
    return out;
  }, [items]);

  if (loading) return <LoadingState />;

  const counts = data?.counts ?? {};
  const filters = [
    { id: "", label: N.all, n: Object.values(counts).reduce((a, b) => a + b, 0) },
    ...Object.keys(KINDS)
      .filter((k) => counts[k])
      .map((k) => ({ id: k, label: N.kinds[k as keyof typeof N.kinds] ?? k, n: counts[k] ?? 0 })),
  ];

  return (
    <PageContainer width="wide">
      <PageHeader
        icon={IconBell}
        title={N.title}
        subtitle={data && data.unread > 0 ? N.unreadCount.replace("{n}", fmtNum(data.unread)) : undefined}
        actions={
          data && data.unread > 0 ? (
            <Button variant="secondary" onClick={markAll}>
              {N.markAllRead}
            </Button>
          ) : undefined
        }
      />

      {filters.length > 1 && (
        <div className="flex flex-wrap gap-2">
          {filters.map((f) => (
            <button
              key={f.id}
              type="button"
              onClick={() => setKind(f.id)}
              className={`rounded-badge px-3 py-1.5 text-sm transition-colors ${
                kind === f.id
                  ? "bg-primary font-medium text-white"
                  : "border border-line text-ink-muted hover:text-ink"
              }`}
            >
              {f.label}
              <span className="ms-1.5 opacity-70">{fmtNum(f.n)}</span>
            </button>
          ))}
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState icon={IconBell} title={N.empty} />
      ) : (
        <div className="space-y-5">
          {groups.map((g) => (
            <section key={g.day}>
              <h2 className="mb-2 text-xs font-medium text-ink-muted">{g.day}</h2>
              <ul className="overflow-hidden rounded-card border border-line bg-surface">
                {g.rows.map((n) => {
                  const meta = KINDS[n.kind] ?? KINDS.account!;
                  const Icon = meta.icon;
                  const inner = (
                    <div className="flex items-start gap-3 px-4 py-3">
                      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${meta.tone}`}>
                        <Icon size={17} />
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className={`truncate ${n.read ? "text-ink" : "font-bold text-ink"}`}>
                          {n.title}
                        </p>
                        {n.body && <p className="truncate text-sm text-ink-muted">{n.body}</p>}
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <span className="text-xs text-ink-muted" dir="ltr">
                          {fmtTime(n.created_at)}
                        </span>
                        {!n.read && <span className="h-2 w-2 rounded-badge bg-primary" />}
                      </div>
                    </div>
                  );
                  return (
                    <li
                      key={n.id}
                      className={`border-b border-line last:border-0 ${n.read ? "" : "bg-primary-light/25"}`}
                    >
                      {n.href ? (
                        <Link href={n.href} className="block transition-colors hover:bg-page">
                          {inner}
                        </Link>
                      ) : (
                        <button
                          type="button"
                          onClick={() => markOne(n.id)}
                          className="block w-full text-start transition-colors hover:bg-page"
                        >
                          {inner}
                        </button>
                      )}
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
