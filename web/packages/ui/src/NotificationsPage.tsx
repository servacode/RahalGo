"use client";

/**
 * صفحة الإشعارات الكاملة — **مكوّن واحد مركزي** ترثه كل اللوحات وموقع الزبون.
 *
 * الجرس يعرض آخر 30 في قائمة منسدلة ضيّقة: يكفي للحظة، ولا يكفي لمن غاب يومين
 * ويريد أن يعرف ما فاته. هنا الأرشيف كاملاً بترشيح بالنوع وتجميع بالتاريخ.
 */

import { useCallback, useMemo, useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, fmtLongDate } from "@rahalgo/i18n";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { Button } from "./components";
import { useLiveData, emitLocal, READ_EVENT, type AppNotification } from "./Notifications";
import {
  IconBell,
  IconOrder,
  IconSupport,
  IconWallet,
  IconStar,
  IconLink,
  IconUser,
  IconPromos,
} from "./icons";

const m = getMessages(defaultLocale);
const N = m.shared.notifications;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type LinkType = ComponentType<{
  href: string;
  className?: string;
  children: ReactNode;
  onClick?: () => void;
}>;

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
  // **والعرضُ له وجهُه** — إشعارٌ بلا أيقونةٍ خاصّةٍ يسقط على الافتراضيّ
  // فيختلط بما ليس منه في قائمةٍ تُمسح بالعين.
  offer: { icon: IconPromos, tone: "text-danger bg-danger/10" },
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

export function NotificationsPage({
  api,
  Link,
  // اللوحات تحصر العرض لأن سايدبارها يقتطع جانباً؛ وموقع الزبون بلا سايدبار
  // فيأخذ الصفحة كاملة — الفرق في الهيكل لا في المكوّن، فصار مُعامِلاً.
  width = "wide",
}: {
  api: ApiFn;
  Link: LinkType;
  width?: "wide" | "full";
}) {
  const [kind, setKind] = useState("");

  const { data, loading, reload } = useLiveData<Feed>(
    () => api(`/api/v1/me/notifications?limit=200${kind ? `&kind=${kind}` : ""}`),
    ["order", "ticket", "wallet", "rating", "lead", "account", READ_EVENT],
    // **والنوعُ يُعيد الجلب** — بدونه تُضيء الشريحةُ ولا تُنادى الشبكة.
    [kind],
  );

  const markAll = useCallback(async () => {
    await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id: "" }) });
    emitLocal(READ_EVENT); // الجرس يلتقط الصفر فوراً بلا تحديث صفحة
    reload();
  }, [api, reload]);

  const markOne = useCallback(
    async (id: string) => {
      await api("/api/v1/me/notifications/read", { method: "POST", body: JSON.stringify({ id }) });
      emitLocal(READ_EVENT);
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
    <PageContainer width={width}>
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
        /* **شريطٌ مقسّم لا أزرارٌ متناثرة**: المرشّحاتُ خياراتُ شيءٍ واحد،
           وحدٌّ يجمعها يقول ذلك قبل أن تُقرأ. */
        <div className="inline-flex flex-wrap gap-1 rounded-card border border-line bg-surface p-1">
          {filters.map((f) => (
            <button
              key={f.id}
              type="button"
              onClick={() => setKind(f.id)}
              className={`flex items-center gap-1.5 rounded-control px-3 py-1.5 text-sm transition-colors ${
                kind === f.id
                  ? "bg-primary font-bold text-on-solid elev-1"
                  : "text-ink-muted hover:bg-page hover:text-ink"
              }`}
            >
              {f.label}
              <span
                className={`rounded-badge px-1.5 text-2xs tabular-nums ${
                  kind === f.id ? "bg-on-solid/20" : "bg-page"
                }`}
              >
                {fmtNum(f.n)}
              </span>
            </button>
          ))}
        </div>
      )}

      {items.length === 0 ? (
        <EmptyState icon={IconBell} title={N.empty} />
      ) : (
        <div className="space-y-6">
          {groups.map((g) => (
            <section key={g.day}>
              {/* **عنوانُ اليوم يلتصق عند التمرير.**

                  أرشيفٌ من مئتي سطرٍ يفقد صاحبَه: يمرّر فينسى أيَّ يومٍ يقرأ.
                  **والعنوانُ الذي يهرب مع التمرير عنوانٌ لا يُقرأ إلّا مرّة.** */}
              <div className="sticky top-0 z-10 -mx-1 mb-2 bg-page/85 px-1 py-1.5 backdrop-blur">
                <h2 className="flex items-center gap-2 text-xs font-bold text-ink-muted">
                  <span className="h-px flex-1 bg-line" />
                  <span className="shrink-0">{g.day}</span>
                  <span className="h-px flex-1 bg-line" />
                </h2>
              </div>
              <ul className="overflow-hidden rounded-card border border-line bg-surface">
                {g.rows.map((n) => {
                  const meta = KINDS[n.kind] ?? KINDS.account!;
                  const Icon = meta.icon;
                  const inner = (
                    /* **غيرُ المقروء بعمودٍ جانبيّ لا بغسلةِ لون.**

                       كانت خلفيةٌ زرقاء تغمر السطرَ كلَّه — فيبهت النصُّ فيها
                       ويصير الأحدثُ أصعبَ قراءةً من الأقدم. **وما يُميَّز
                       بإضعافه لم يُميَّز.** والعمودُ يقول الشيءَ نفسه بحرفٍ
                       واحد ولا يمسّ النصّ. */
                    <div
                      className={`flex items-start gap-3 border-s-[3px] py-3 pe-4 ps-3.5 ${
                        n.read ? "border-transparent" : "border-primary bg-primary-light/40"
                      }`}
                    >
                      <span
                        className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-control ${meta.tone}`}
                      >
                        <Icon size={17} />
                      </span>
                      <div className="min-w-0 flex-1">
                        <p
                          className={`leading-snug ${n.read ? "text-ink" : "font-bold text-ink"}`}
                        >
                          {n.title}
                        </p>
                        {/* **الجسدُ سطران لا سطرٌ مقتطع**: «تعويض عن طلبٍ فشل —
                            المطعم مغلق» يُقصّ عند «طلبٍ» فيبقى السؤال. */}
                        {n.body && (
                          <p className="mt-0.5 line-clamp-2 text-sm leading-relaxed text-ink-muted">
                            {n.body}
                          </p>
                        )}
                      </div>
                      {/* **التاريخُ مع الساعة — لا الساعةُ وحدَها.**

                          العناوينُ تجمع بالأيام («اليوم» و«أمس»)، **وسطرٌ
                          يقول «٣:٤٠ م» وحدَه يُقرأ خارجَ عنوانه**: يُنسخ في
                          رسالةٍ أو يُذكر في اتّصال فلا يُعرف أيُّ يومٍ هو.
                          (قرارُ المالك ٢٠٢٦-٠٨-٠٥.) */}
                      <span
                        className="shrink-0 pt-0.5 text-xs tabular-nums text-ink-muted"
                        dir="ltr"
                      >
                        {fmtDateTime(n.created_at)}
                      </span>
                    </div>
                  );
                  return (
                    <li key={n.id} className="border-b border-line last:border-0">
                      {n.href ? (
                        <Link
                          href={n.href}
                          onClick={() => !n.read && void markOne(n.id)}
                          className="block transition-colors hover:bg-page"
                        >
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
